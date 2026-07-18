package rotation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestReassignDiscoverySessionExpiredReassignsSameProvider(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newSyntheticReassignmentStore([]Account{
		{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusLeased},
		{AccountID: "wrong-provider", Vendor: "kiro", TenantID: "tenant-1", Priority: 1, Status: StatusAvailable},
		{AccountID: "next", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: StatusAvailable},
	})
	store.assignments["agent-1"] = "current"
	auth := &syntheticReassignmentAuthenticator{}
	service := NewService(store, fakeDetector{}, auth)

	got, reassigned, err := service.ReassignDiscoverySession(
		context.Background(),
		"agent-1",
		"current",
		"codex",
		"tenant-1",
		DiscoverySession{Provider: "codex", Status: "active", ExpiresAt: now.Add(-time.Second).Format(time.RFC3339)},
		now,
	)
	if err != nil {
		t.Fatalf("ReassignDiscoverySession: %v", err)
	}
	if !reassigned || got.AccountID != "next" {
		t.Fatalf("result = (%+v, %v), want next/true", got, reassigned)
	}
	if assigned := store.assignment("agent-1"); assigned != "next" {
		t.Fatalf("assignment = %q, want next", assigned)
	}
	if status := store.accountStatus("current"); status != StatusExhausted {
		t.Fatalf("current status = %q, want exhausted", status)
	}
	assertSyntheticCalls(t, auth.snapshotCalls(), []string{"logout:current", "login:next", "wait:session-next"})
}

func TestReassignDiscoverySessionFutureOrCrossProviderDoesNothing(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name     string
		provider string
		session  DiscoverySession
	}{
		{
			name:     "future expiry",
			provider: "codex",
			session:  DiscoverySession{Provider: "codex", Status: "active", ExpiresAt: now.Add(time.Second).Format(time.RFC3339)},
		},
		{
			name:     "cross provider",
			provider: "codex",
			session:  DiscoverySession{Provider: "kiro", Status: "expired"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := newSyntheticReassignmentStore([]Account{
				{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Status: StatusLeased},
				{AccountID: "next", Vendor: "codex", TenantID: "tenant-1", Status: StatusAvailable},
			})
			store.assignments["agent-1"] = "current"
			auth := &syntheticReassignmentAuthenticator{}
			service := NewService(store, fakeDetector{}, auth)

			_, reassigned, err := service.ReassignDiscoverySession(
				context.Background(), "agent-1", "current", tt.provider, "tenant-1", tt.session, now,
			)
			if err != nil {
				t.Fatalf("ReassignDiscoverySession: %v", err)
			}
			if reassigned {
				t.Fatal("non-exhausted/provider-mismatched observation reassigned the agent")
			}
			if assigned := store.assignment("agent-1"); assigned != "current" {
				t.Fatalf("assignment = %q, want current", assigned)
			}
			if calls := auth.snapshotCalls(); len(calls) != 0 {
				t.Fatalf("auth calls = %v, want none", calls)
			}
		})
	}
}

func TestReassignDiscoverySessionNoNextFailsClosedBeforeLogout(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newSyntheticReassignmentStore([]Account{
		{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Status: StatusLeased},
	})
	store.assignments["agent-1"] = "current"
	auth := &syntheticReassignmentAuthenticator{}
	service := NewService(store, fakeDetector{}, auth)

	_, reassigned, err := service.ReassignDiscoverySession(
		context.Background(), "agent-1", "current", "codex", "tenant-1",
		DiscoverySession{Provider: "codex", Status: "expired"}, now,
	)
	if !errors.Is(err, ErrNoAccountAvailable) {
		t.Fatalf("error = %v, want ErrNoAccountAvailable", err)
	}
	if reassigned {
		t.Fatal("exhausted pool reported reassignment")
	}
	if assigned := store.assignment("agent-1"); assigned != "current" {
		t.Fatalf("assignment = %q, want current", assigned)
	}
	if calls := auth.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("auth calls = %v, want none before replacement selection", calls)
	}
}

func TestReassignDiscoverySessionRejectsAssignmentBoundary(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name          string
		currentVendor string
		currentTenant string
	}{
		{name: "different provider", currentVendor: "kiro", currentTenant: "tenant-1"},
		{name: "different tenant", currentVendor: "codex", currentTenant: "tenant-2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := newSyntheticReassignmentStore([]Account{
				{AccountID: "current", Vendor: tt.currentVendor, TenantID: tt.currentTenant, Status: StatusLeased},
				{AccountID: "next", Vendor: "codex", TenantID: "tenant-1", Status: StatusAvailable},
			})
			store.assignments["agent-1"] = "current"
			auth := &syntheticReassignmentAuthenticator{}
			service := NewService(store, fakeDetector{}, auth)

			_, reassigned, err := service.ReassignDiscoverySession(
				context.Background(), "agent-1", "current", "codex", "tenant-1",
				DiscoverySession{Provider: "codex", Status: "expired"}, now,
			)
			if !errors.Is(err, errDiscoveryAssignmentBoundary) {
				t.Fatalf("error = %v, want assignment-boundary error", err)
			}
			if reassigned {
				t.Fatal("out-of-boundary assignment was reassigned")
			}
			if assigned := store.assignment("agent-1"); assigned != "current" {
				t.Fatalf("assignment = %q, want current", assigned)
			}
			if calls := auth.snapshotCalls(); len(calls) != 0 {
				t.Fatalf("auth calls = %v, want none", calls)
			}
		})
	}
}

func TestReassignDiscoverySessionAssignFailureCleansNewSession(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	assignErr := errors.New("synthetic assign failure")
	store := newSyntheticReassignmentStore([]Account{
		{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Status: StatusLeased},
		{AccountID: "next", Vendor: "codex", TenantID: "tenant-1", Status: StatusAvailable},
	})
	store.assignments["agent-1"] = "current"
	store.assignErr = assignErr
	auth := &syntheticReassignmentAuthenticator{}
	service := NewService(store, fakeDetector{}, auth)

	_, reassigned, err := service.ReassignDiscoverySession(
		context.Background(), "agent-1", "current", "codex", "tenant-1",
		DiscoverySession{Provider: "codex", Status: "exhausted"}, now,
	)
	if !errors.Is(err, assignErr) {
		t.Fatalf("error = %v, want synthetic assign failure", err)
	}
	if reassigned {
		t.Fatal("failed assignment reported reassignment")
	}
	if assigned := store.assignment("agent-1"); assigned != "current" {
		t.Fatalf("assignment = %q, want current", assigned)
	}
	assertSyntheticCalls(t, auth.snapshotCalls(), []string{
		"logout:current",
		"login:next",
		"wait:session-next",
		"logout:next",
	})
	if got := store.rotationCount(); got != 0 {
		t.Fatalf("rotation records = %d, want 0", got)
	}
}

func TestReassignDiscoverySessionConcurrentDuplicateObservationRotatesOnce(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	store := newSyntheticReassignmentStore([]Account{
		{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Status: StatusLeased},
		{AccountID: "next", Vendor: "codex", TenantID: "tenant-1", Status: StatusAvailable},
	})
	store.assignments["agent-1"] = "current"
	auth := &syntheticReassignmentAuthenticator{}
	service := NewService(store, fakeDetector{}, auth)

	const workers = 32
	start := make(chan struct{})
	results := make(chan bool, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, reassigned, err := service.ReassignDiscoverySession(
				context.Background(), "agent-1", "current", "codex", "tenant-1",
				DiscoverySession{Provider: "codex", Status: "expired"}, now,
			)
			results <- reassigned
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent reassignment: %v", err)
		}
	}
	var reassignedCount int
	for reassigned := range results {
		if reassigned {
			reassignedCount++
		}
	}
	if reassignedCount != 1 {
		t.Fatalf("successful reassignments = %d, want 1", reassignedCount)
	}
	if assigned := store.assignment("agent-1"); assigned != "next" {
		t.Fatalf("assignment = %q, want next", assigned)
	}
	if got := store.rotationCount(); got != 1 {
		t.Fatalf("rotation records = %d, want 1", got)
	}
}

func assertSyntheticCalls(t *testing.T, got, want []string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("auth calls = %v, want %v", got, want)
	}
}

type syntheticReassignmentStore struct {
	mu          sync.Mutex
	accounts    map[string]Account
	order       []string
	assignments map[string]string
	rotations   []rotationRecord
	assignErr   error
}

func newSyntheticReassignmentStore(accounts []Account) *syntheticReassignmentStore {
	store := &syntheticReassignmentStore{
		accounts:    make(map[string]Account, len(accounts)),
		assignments: map[string]string{},
	}
	for _, account := range accounts {
		store.accounts[account.AccountID] = account
		store.order = append(store.order, account.AccountID)
	}
	return store
}

func (s *syntheticReassignmentStore) ListAccounts(_ context.Context, vendor, tenantID string) ([]Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var accounts []Account
	for _, accountID := range s.order {
		account := s.accounts[accountID]
		if account.Vendor == vendor && account.TenantID == tenantID {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

func (s *syntheticReassignmentStore) GetAccount(_ context.Context, accountID string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[accountID]
	if !ok {
		return Account{}, errors.New("synthetic account not found")
	}
	return account, nil
}

func (s *syntheticReassignmentStore) UpdateAccountStatus(_ context.Context, accountID string, status AccountStatus, cooldownUntil *time.Time) error {
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

func (s *syntheticReassignmentStore) RecordUsage(_ context.Context, _ string, _ int64, _ time.Time) error {
	return nil
}

func (s *syntheticReassignmentStore) Assign(_ context.Context, agentID, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.assignErr != nil {
		return s.assignErr
	}
	if _, ok := s.accounts[accountID]; !ok {
		return errors.New("synthetic account not found")
	}
	s.assignments[agentID] = accountID
	return nil
}

func (s *syntheticReassignmentStore) CurrentAssignment(_ context.Context, agentID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assignments[agentID], nil
}

func (s *syntheticReassignmentStore) RecordRotation(_ context.Context, agentID, fromAccountID, toAccountID string, reason RotationReason, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rotations = append(s.rotations, rotationRecord{
		agentID:       agentID,
		fromAccountID: fromAccountID,
		toAccountID:   toAccountID,
		reason:        reason,
		at:            at,
	})
	return nil
}

func (s *syntheticReassignmentStore) assignment(agentID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.assignments[agentID]
}

func (s *syntheticReassignmentStore) accountStatus(accountID string) AccountStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accounts[accountID].Status
}

func (s *syntheticReassignmentStore) rotationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.rotations)
}

type syntheticReassignmentAuthenticator struct {
	mu    sync.Mutex
	calls []string
}

func (a *syntheticReassignmentAuthenticator) Login(_ context.Context, account Account) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "login:"+account.AccountID)
	return "session-" + account.AccountID, nil
}

func (a *syntheticReassignmentAuthenticator) Logout(_ context.Context, account Account) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "logout:"+account.AccountID)
	return nil
}

func (a *syntheticReassignmentAuthenticator) WaitAuthenticated(_ context.Context, sessionID string, _ time.Duration) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "wait:"+sessionID)
	return true, nil
}

func (a *syntheticReassignmentAuthenticator) snapshotCalls() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.calls...)
}
