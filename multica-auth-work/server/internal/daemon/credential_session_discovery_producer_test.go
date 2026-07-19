package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

func TestCredentialSessionDiscoveryProducerEndToEndReassigns(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newProducerSyntheticStore([]rotation.Account{
		{AccountID: "account-current", Vendor: "codex", TenantID: "workspace-1", Priority: 10, Status: rotation.StatusLeased},
		{AccountID: "wrong-provider", Vendor: "kiro", TenantID: "workspace-1", Priority: 1, Status: rotation.StatusAvailable},
		{AccountID: "account-next", Vendor: "codex", TenantID: "workspace-1", Priority: 20, Status: rotation.StatusAvailable},
	})
	store.assignments["agent-1"] = "account-current"
	auth := &producerSyntheticAuthenticator{}
	service := rotation.NewService(store, producerNoopDetector{}, auth)
	daemon := &Daemon{rotationService: service}
	emitter := &producerLoopbackEmitter{daemon: daemon, now: now}
	producer := NewCredentialSessionDiscoveryProducer(emitter)
	observation := CredentialSessionDiscoveryObservation{
		AgentID:     "agent-1",
		AccountID:   "account-current",
		Provider:    "codex",
		WorkspaceID: "workspace-1",
		Status:      "active",
		ExpiresAt:   now.Add(-time.Second).Format(time.RFC3339),
	}

	emitted, err := producer.Produce(context.Background(), observation, now)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if !emitted {
		t.Fatal("expired discovery observation was not emitted")
	}
	if assigned := store.assignment("agent-1"); assigned != "account-next" {
		t.Fatalf("assignment = %q, want account-next", assigned)
	}
	if status := store.accountStatus("account-current"); status != rotation.StatusExhausted {
		t.Fatalf("current status = %q, want exhausted", status)
	}
	if got := auth.snapshotCalls(); fmt.Sprint(got) != fmt.Sprint([]string{"logout:account-current", "login:account-next", "wait:session-account-next"}) {
		t.Fatalf("auth calls = %v", got)
	}
	if emitter.count() != 1 {
		t.Fatalf("emitted events = %d, want 1", emitter.count())
	}

	var payload credentialSessionDiscoveryPayload
	message := emitter.lastMessage()
	if message.Type != eventDaemonCredentialSessionDiscovery {
		t.Fatalf("event type = %q", message.Type)
	}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("decode emitted payload: %v", err)
	}
	if payload != (credentialSessionDiscoveryPayload{
		AgentID: observation.AgentID, AccountID: observation.AccountID, Provider: observation.Provider,
		WorkspaceID: observation.WorkspaceID, Status: observation.Status, ExpiresAt: observation.ExpiresAt,
	}) {
		t.Fatalf("emitted payload = %+v, want exact observation", payload)
	}

	emitted, err = producer.Produce(context.Background(), observation, now.Add(time.Second))
	if err != nil || emitted {
		t.Fatalf("duplicate Produce = (%v, %v), want false/nil", emitted, err)
	}
	if emitter.count() != 1 {
		t.Fatalf("duplicate emitted event count = %d, want 1", emitter.count())
	}
}

func TestCredentialSessionDiscoveryProducerRequiresExactBoundedFields(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	base := CredentialSessionDiscoveryObservation{
		AgentID: "agent-1", AccountID: "account-1", Provider: "codex", WorkspaceID: "workspace-1", Status: "expired",
	}
	for _, tt := range []struct {
		name   string
		mutate func(*CredentialSessionDiscoveryObservation)
	}{
		{name: "missing agent", mutate: func(o *CredentialSessionDiscoveryObservation) { o.AgentID = "" }},
		{name: "padded account", mutate: func(o *CredentialSessionDiscoveryObservation) { o.AccountID = " account-1" }},
		{name: "missing provider", mutate: func(o *CredentialSessionDiscoveryObservation) { o.Provider = "" }},
		{name: "missing workspace", mutate: func(o *CredentialSessionDiscoveryObservation) { o.WorkspaceID = "" }},
		{name: "missing status and expiry", mutate: func(o *CredentialSessionDiscoveryObservation) { o.Status = "" }},
		{name: "oversized provider", mutate: func(o *CredentialSessionDiscoveryObservation) {
			o.Provider = strings.Repeat("p", maxCredentialDiscoveryProviderLength+1)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			observation := base
			tt.mutate(&observation)
			emitter := &producerRecordingEmitter{}
			producer := NewCredentialSessionDiscoveryProducer(emitter)
			emitted, err := producer.Produce(context.Background(), observation, now)
			if emitted || !errors.Is(err, errInvalidCredentialDiscoveryObservation) {
				t.Fatalf("Produce = (%v, %v), want false/invalid", emitted, err)
			}
			if emitter.count() != 0 {
				t.Fatalf("invalid observation emitted %d events", emitter.count())
			}
		})
	}
}

func TestCredentialSessionDiscoveryProducerDeduplicatesConcurrently(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	emitter := &producerRecordingEmitter{}
	producer := NewCredentialSessionDiscoveryProducer(emitter)
	observation := CredentialSessionDiscoveryObservation{
		AgentID: "agent-1", AccountID: "account-1", Provider: "codex", WorkspaceID: "workspace-1", Status: "exhausted",
	}

	const workers = 64
	start := make(chan struct{})
	results := make(chan bool, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			emitted, err := producer.Produce(context.Background(), observation, now)
			results <- emitted
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Produce: %v", err)
		}
	}
	var emittedCount int
	for emitted := range results {
		if emitted {
			emittedCount++
		}
	}
	if emittedCount != 1 || emitter.count() != 1 {
		t.Fatalf("emitted results/events = %d/%d, want 1/1", emittedCount, emitter.count())
	}
}

func TestCredentialSessionDiscoveryProducerBoundsAndResetsDedup(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	emitter := &producerRecordingEmitter{}
	producer := newCredentialSessionDiscoveryProducer(emitter, 2, time.Minute)
	observation := func(accountID string) CredentialSessionDiscoveryObservation {
		return CredentialSessionDiscoveryObservation{
			AgentID: "agent-1", AccountID: accountID, Provider: "codex", WorkspaceID: "workspace-1", Status: "expired",
		}
	}
	for _, accountID := range []string{"account-1", "account-2", "account-3"} {
		if emitted, err := producer.Produce(context.Background(), observation(accountID), now); err != nil || !emitted {
			t.Fatalf("Produce(%s) = (%v, %v)", accountID, emitted, err)
		}
	}
	producer.mu.Lock()
	seenCount := len(producer.seen)
	producer.mu.Unlock()
	if seenCount != 2 {
		t.Fatalf("dedup entries = %d, want bounded size 2", seenCount)
	}

	active := observation("account-3")
	active.Status = "active"
	active.ExpiresAt = now.Add(time.Hour).Format(time.RFC3339)
	if emitted, err := producer.Produce(context.Background(), active, now); err != nil || emitted {
		t.Fatalf("active reset Produce = (%v, %v), want false/nil", emitted, err)
	}
	if emitted, err := producer.Produce(context.Background(), observation("account-3"), now); err != nil || !emitted {
		t.Fatalf("post-reset Produce = (%v, %v), want true/nil", emitted, err)
	}
}

func TestCredentialSessionDiscoveryProducerRetriesAfterEmitterFailure(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	wantErr := errors.New("synthetic emit failure")
	emitter := &producerRecordingEmitter{failuresRemaining: 1, err: wantErr}
	producer := NewCredentialSessionDiscoveryProducer(emitter)
	observation := CredentialSessionDiscoveryObservation{
		AgentID: "agent-1", AccountID: "account-1", Provider: "codex", WorkspaceID: "workspace-1", Status: "expired",
	}
	if emitted, err := producer.Produce(context.Background(), observation, now); emitted || !errors.Is(err, wantErr) {
		t.Fatalf("first Produce = (%v, %v), want false/failure", emitted, err)
	}
	if emitted, err := producer.Produce(context.Background(), observation, now); err != nil || !emitted {
		t.Fatalf("retry Produce = (%v, %v), want true/nil", emitted, err)
	}
}

type producerLoopbackEmitter struct {
	mu       sync.Mutex
	daemon   *Daemon
	now      time.Time
	messages []protocol.Message
}

func (e *producerLoopbackEmitter) EmitCredentialSessionDiscovery(ctx context.Context, message protocol.Message) error {
	e.mu.Lock()
	e.messages = append(e.messages, protocol.Message{Type: message.Type, Payload: append([]byte(nil), message.Payload...)})
	e.mu.Unlock()
	_, err := e.daemon.dispatchCredentialSessionDiscoveryEvent(ctx, message, e.now)
	return err
}

func (e *producerLoopbackEmitter) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.messages)
}

func (e *producerLoopbackEmitter) lastMessage() protocol.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.messages[len(e.messages)-1]
}

type producerRecordingEmitter struct {
	mu                sync.Mutex
	messages          []protocol.Message
	failuresRemaining int
	err               error
}

func (e *producerRecordingEmitter) EmitCredentialSessionDiscovery(_ context.Context, message protocol.Message) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.failuresRemaining > 0 {
		e.failuresRemaining--
		return e.err
	}
	e.messages = append(e.messages, protocol.Message{Type: message.Type, Payload: append([]byte(nil), message.Payload...)})
	return nil
}

func (e *producerRecordingEmitter) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.messages)
}

type producerNoopDetector struct{}

func (producerNoopDetector) Detect(string, string, int) rotation.DetectionResult {
	return rotation.DetectionResult{}
}

type producerSyntheticStore struct {
	mu          sync.Mutex
	accounts    map[string]rotation.Account
	order       []string
	assignments map[string]string
	rotations   int
}

func newProducerSyntheticStore(accounts []rotation.Account) *producerSyntheticStore {
	store := &producerSyntheticStore{accounts: map[string]rotation.Account{}, assignments: map[string]string{}}
	for _, account := range accounts {
		store.accounts[account.AccountID] = account
		store.order = append(store.order, account.AccountID)
	}
	return store
}

func (s *producerSyntheticStore) ListAccounts(_ context.Context, vendor, tenantID string) ([]rotation.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var accounts []rotation.Account
	for _, id := range s.order {
		account := s.accounts[id]
		if account.Vendor == vendor && account.TenantID == tenantID {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

func (s *producerSyntheticStore) GetAccount(_ context.Context, accountID string) (rotation.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return rotation.Account{}, errors.New("synthetic account not found")
	}
	return account, nil
}

func (s *producerSyntheticStore) UpdateAccountStatus(_ context.Context, accountID string, status rotation.AccountStatus, cooldownUntil *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return errors.New("synthetic account not found")
	}
	account.Status = status
	account.CooldownUntil = cooldownUntil
	s.accounts[accountID] = account
	return nil
}

func (*producerSyntheticStore) RecordUsage(context.Context, string, int64, time.Time) error {
	return nil
}

func (s *producerSyntheticStore) Assign(_ context.Context, agentID, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accounts[accountID]; !ok {
		return errors.New("synthetic account not found")
	}
	s.assignments[agentID] = accountID
	return nil
}

func (s *producerSyntheticStore) CurrentAssignment(_ context.Context, agentID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assignments[agentID], nil
}

func (s *producerSyntheticStore) RecordRotation(context.Context, string, string, string, rotation.RotationReason, time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rotations++
	return nil
}

func (s *producerSyntheticStore) assignment(agentID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assignments[agentID]
}

func (s *producerSyntheticStore) accountStatus(accountID string) rotation.AccountStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accounts[accountID].Status
}

type producerSyntheticAuthenticator struct {
	mu    sync.Mutex
	calls []string
}

func (a *producerSyntheticAuthenticator) Login(_ context.Context, account rotation.Account) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "login:"+account.AccountID)
	return "session-" + account.AccountID, nil
}

func (a *producerSyntheticAuthenticator) Logout(_ context.Context, account rotation.Account) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "logout:"+account.AccountID)
	return nil
}

func (a *producerSyntheticAuthenticator) WaitAuthenticated(_ context.Context, sessionID string, _ time.Duration) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "wait:"+sessionID)
	return true, nil
}

func (a *producerSyntheticAuthenticator) snapshotCalls() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.calls...)
}
