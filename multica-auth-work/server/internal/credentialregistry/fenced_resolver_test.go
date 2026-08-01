package credentialregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type testStore struct {
	reserve func(context.Context, Request, CandidateValidator) (Candidate, error)
	release func(context.Context, ReleaseRequest) error
}

func (s *testStore) ReserveAssignment(ctx context.Context, request Request, validate CandidateValidator) (Candidate, error) {
	return s.reserve(ctx, request, validate)
}

func (s *testStore) ReleaseAssignment(ctx context.Context, request ReleaseRequest) error {
	if s.release == nil {
		return nil
	}
	return s.release(ctx, request)
}

func approvedCandidate() Candidate {
	return Candidate{
		TaskID:                      "00000000-0000-0000-0000-000000000101",
		TaskStatus:                  "queued",
		BindingID:                   "00000000-0000-0000-0000-000000000102",
		AgentID:                     "00000000-0000-0000-0000-000000000103",
		WorkspaceID:                 "00000000-0000-0000-0000-000000000104",
		RuntimeID:                   "00000000-0000-0000-0000-000000000105",
		RuntimeSessionID:            "00000000-0000-0000-0000-000000000106",
		StandardVersionID:           "00000000-0000-0000-0000-000000000107",
		ConfigurationVersionID:      "00000000-0000-0000-0000-000000000108",
		ConfigurationDigest:         "sha256:configuration-digest",
		CapabilityDigest:            "sha256:capability-digest",
		Provider:                    "antigravity",
		SessionProvider:             "antigravity",
		HomeRef:                     "home_01JABCDEFGHJKMNPQRSTVWXYZ",
		BindingGeneration:           11,
		AssignmentCatalogGeneration: 17,
		CatalogGeneration:           17,
		Approval:                    ApprovalApproved,
		Status:                      StatusAvailable,
		BindingState:                "active",
		TransportBinding:            "native_credential_home",
		AssignmentState:             "active",
		CatalogState:                "available",
		CatalogEntryState:           "healthy",
		HealthFresh:                 true,
		AssignmentOwners:            1,
		BindingAssignments:          1,
		ActiveTasks:                 2,
		TaskConcurrencyLimit:        4,
	}
}

func approvedRequest() Request {
	return Request{
		TaskID:                    "00000000-0000-0000-0000-000000000101",
		BindingID:                 "00000000-0000-0000-0000-000000000102",
		AgentID:                   "00000000-0000-0000-0000-000000000103",
		WorkspaceID:               "00000000-0000-0000-0000-000000000104",
		Provider:                  "agy",
		ExpectedBindingGeneration: 11,
		ExpectedCatalogGeneration: 17,
	}
}

func resolvingStore(candidate Candidate, storeErr error) *testStore {
	return &testStore{reserve: func(_ context.Context, _ Request, validate CandidateValidator) (Candidate, error) {
		if storeErr != nil {
			return Candidate{}, storeErr
		}
		if err := validate(candidate); err != nil {
			return Candidate{}, err
		}
		candidate.ActiveTasks++
		return candidate, nil
	}}
}

func TestValidHomeRef(t *testing.T) {
	for _, ref := range []HomeRef{
		"home_01JABCDEFGHJKMNPQRSTVWXYZ", "0194f956-5e6c-7f6b-9a76-1b2759d2bf31", "catalog.home~opaque_0001",
	} {
		if !ValidHomeRef(ref) {
			t.Fatalf("ValidHomeRef(%q)=false, want true", ref)
		}
	}
	for _, ref := range []HomeRef{
		"", "short", "/var/lib/credential/home", `C:\\credential-home`, "https://catalog/home",
		"home ref with spaces", "../opaque-home-reference", "home_opaque\nreference",
	} {
		if ValidHomeRef(ref) {
			t.Fatalf("ValidHomeRef(%q)=true, want false", ref)
		}
	}
}

func TestResolveValidatesBeforeStoreMutation(t *testing.T) {
	candidate := approvedCandidate()
	mutations := 0
	store := &testStore{reserve: func(_ context.Context, request Request, validate CandidateValidator) (Candidate, error) {
		if request.Provider != "antigravity" {
			t.Fatalf("provider was not canonicalized: %q", request.Provider)
		}
		if err := validate(candidate); err != nil {
			return Candidate{}, err
		}
		mutations++
		candidate.ActiveTasks++
		return candidate, nil
	}}

	got, err := NewFencedResolver(store).Resolve(context.Background(), approvedRequest())
	if err != nil {
		t.Fatalf("Resolve() error=%v", err)
	}
	if mutations != 1 || got.BindingGeneration != 11 || got.CatalogGeneration != 17 {
		t.Fatalf("mutations=%d assignment=%+v", mutations, got)
	}
}

func TestResolveRejectedCandidateCannotLeakCapacity(t *testing.T) {
	candidate := approvedCandidate()
	candidate.Approval = ApprovalRevoked
	mutations := 0
	store := &testStore{reserve: func(_ context.Context, _ Request, validate CandidateValidator) (Candidate, error) {
		if err := validate(candidate); err != nil {
			return Candidate{}, err
		}
		mutations++
		return candidate, nil
	}}

	_, err := NewFencedResolver(store).Resolve(context.Background(), approvedRequest())
	if !errors.Is(err, ErrNoApprovedAssignment) {
		t.Fatalf("Resolve() error=%v, want revoked failure", err)
	}
	if mutations != 0 {
		t.Fatalf("rejected validation leaked %d mutations", mutations)
	}
}

func TestResolveFailsClosed(t *testing.T) {
	storeFailure := errors.New("database unavailable at /private/path")
	tests := []struct {
		name      string
		mutate    func(*Candidate)
		storeErr  error
		wantError error
	}{
		{name: "missing row", storeErr: ErrNoApprovedAssignment, wantError: ErrNoApprovedAssignment},
		{name: "store task conflict", storeErr: ErrTaskConflict, wantError: ErrTaskConflict},
		{name: "store generation conflict", storeErr: ErrGenerationConflict, wantError: ErrGenerationConflict},
		{name: "store failure bounded", storeErr: storeFailure, wantError: ErrRegistryUnavailable},
		{name: "task mismatch", mutate: func(c *Candidate) { c.TaskID = "other-task" }, wantError: ErrNoApprovedAssignment},
		{name: "task already claimed", mutate: func(c *Candidate) { c.TaskStatus = "dispatched" }, wantError: ErrTaskConflict},
		{name: "binding mismatch", mutate: func(c *Candidate) { c.BindingID = "other-binding" }, wantError: ErrNoApprovedAssignment},
		{name: "agent mismatch", mutate: func(c *Candidate) { c.AgentID = "other-agent" }, wantError: ErrNoApprovedAssignment},
		{name: "workspace mismatch", mutate: func(c *Candidate) { c.WorkspaceID = "other-workspace" }, wantError: ErrNoApprovedAssignment},
		{name: "approval pending", mutate: func(c *Candidate) { c.Approval = ApprovalPending }, wantError: ErrNoApprovedAssignment},
		{name: "approval revoked", mutate: func(c *Candidate) { c.Approval = ApprovalRevoked }, wantError: ErrNoApprovedAssignment},
		{name: "provider mismatch", mutate: func(c *Candidate) { c.Provider = "kiro" }, wantError: ErrProviderMismatch},
		{name: "stale binding generation", mutate: func(c *Candidate) { c.BindingGeneration++ }, wantError: ErrGenerationConflict},
		{name: "stale assignment generation", mutate: func(c *Candidate) { c.AssignmentCatalogGeneration-- }, wantError: ErrGenerationConflict},
		{name: "stale catalog generation", mutate: func(c *Candidate) { c.CatalogGeneration++ }, wantError: ErrGenerationConflict},
		{name: "no owner", mutate: func(c *Candidate) { c.AssignmentOwners = 0 }, wantError: ErrNoApprovedAssignment},
		{name: "multiple owners", mutate: func(c *Candidate) { c.AssignmentOwners = 2 }, wantError: ErrExclusiveAssignmentConflict},
		{name: "ambiguous binding homes", mutate: func(c *Candidate) { c.BindingAssignments = 2 }, wantError: ErrNoApprovedAssignment},
		{name: "unavailable status", mutate: func(c *Candidate) { c.Status = "cooldown" }, wantError: ErrAccountUnavailable},
		{name: "session provider mismatch", mutate: func(c *Candidate) { c.SessionProvider = "kiro" }, wantError: ErrProviderMismatch},
		{name: "draining binding", mutate: func(c *Candidate) { c.BindingState = "draining" }, wantError: ErrAccountUnavailable},
		{name: "stale health", mutate: func(c *Candidate) { c.HealthFresh = false }, wantError: ErrAccountUnavailable},
		{name: "path home ref", mutate: func(c *Candidate) { c.HomeRef = "/private/credential/home" }, wantError: ErrInvalidMetadata},
		{name: "missing runtime", mutate: func(c *Candidate) { c.RuntimeID = "" }, wantError: ErrInvalidMetadata},
		{name: "missing session", mutate: func(c *Candidate) { c.RuntimeSessionID = "" }, wantError: ErrInvalidMetadata},
		{name: "missing standard version", mutate: func(c *Candidate) { c.StandardVersionID = "" }, wantError: ErrInvalidMetadata},
		{name: "missing config version", mutate: func(c *Candidate) { c.ConfigurationVersionID = "" }, wantError: ErrInvalidMetadata},
		{name: "missing config digest", mutate: func(c *Candidate) { c.ConfigurationDigest = "" }, wantError: ErrInvalidMetadata},
		{name: "missing capability digest", mutate: func(c *Candidate) { c.CapabilityDigest = "" }, wantError: ErrInvalidMetadata},
		{name: "negative active tasks", mutate: func(c *Candidate) { c.ActiveTasks = -1 }, wantError: ErrInvalidMetadata},
		{name: "missing concurrency policy", mutate: func(c *Candidate) { c.TaskConcurrencyLimit = 0 }, wantError: ErrInvalidMetadata},
		{name: "capacity exhausted", mutate: func(c *Candidate) { c.ActiveTasks = c.TaskConcurrencyLimit }, wantError: ErrCapacityExhausted},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := approvedCandidate()
			if test.mutate != nil {
				test.mutate(&candidate)
			}
			got, err := NewFencedResolver(resolvingStore(candidate, test.storeErr)).Resolve(context.Background(), approvedRequest())
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Resolve() error=%v, want %v", err, test.wantError)
			}
			if got != (FencedAssignment{}) {
				t.Fatalf("Resolve() on failure returned %+v", got)
			}
			if test.storeErr == storeFailure && strings.Contains(err.Error(), "/private/path") {
				t.Fatalf("bounded error leaked store detail: %v", err)
			}
		})
	}
}

func TestResolveRejectsIncompleteRequestBeforeStore(t *testing.T) {
	requests := []Request{
		{},
		{TaskID: "task", BindingID: "binding", AgentID: "agent", WorkspaceID: "workspace", Provider: "claude", ExpectedBindingGeneration: 1, ExpectedCatalogGeneration: 1},
		{TaskID: "task", BindingID: "binding", AgentID: "agent", WorkspaceID: "workspace", Provider: "agy", ExpectedCatalogGeneration: 1},
		{TaskID: "task", BindingID: "binding", AgentID: "agent", WorkspaceID: "workspace", Provider: "agy", ExpectedBindingGeneration: 1},
	}
	for i, request := range requests {
		called := false
		store := &testStore{reserve: func(context.Context, Request, CandidateValidator) (Candidate, error) {
			called = true
			return Candidate{}, nil
		}}
		_, err := NewFencedResolver(store).Resolve(context.Background(), request)
		if err == nil || called {
			t.Fatalf("case %d: err=%v called=%v", i, err, called)
		}
	}
}

func TestAssignmentSurfaceIsPathless(t *testing.T) {
	candidateType := reflect.TypeOf(Candidate{})
	for _, forbidden := range []string{"AccountID", "HomeDir", "ConfigDir", "Path", "SecretRef", "Credential"} {
		if _, ok := candidateType.FieldByName(forbidden); ok {
			t.Fatalf("Candidate exposes forbidden field %s", forbidden)
		}
	}
	assignment, err := NewFencedResolver(resolvingStore(approvedCandidate(), nil)).Resolve(context.Background(), approvedRequest())
	if err != nil {
		t.Fatalf("Resolve() error=%v", err)
	}
	encoded, err := json.Marshal(assignment)
	if err != nil {
		t.Fatalf("Marshal() error=%v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"00000000-", "account", "home_dir", "config_dir", "/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("assignment JSON %s contains forbidden %q", text, forbidden)
		}
	}
}

type atomicStore struct {
	mu        sync.Mutex
	candidate Candidate
	active    map[string]ReleaseRequest
	released  map[string]bool
}

func newAtomicStore(limit int) *atomicStore {
	candidate := approvedCandidate()
	candidate.ActiveTasks = 0
	candidate.TaskConcurrencyLimit = limit
	return &atomicStore{candidate: candidate, active: make(map[string]ReleaseRequest), released: make(map[string]bool)}
}

func (s *atomicStore) ReserveAssignment(_ context.Context, request Request, validate CandidateValidator) (Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.active[request.TaskID]; exists || s.released[request.TaskID] {
		return Candidate{}, ErrTaskConflict
	}
	candidate := s.candidate
	candidate.TaskID = request.TaskID
	if err := validate(candidate); err != nil {
		return Candidate{}, err
	}
	s.candidate.ActiveTasks++
	s.active[request.TaskID] = ReleaseRequest{
		TaskID: request.TaskID, BindingID: request.BindingID,
		BindingGeneration: request.ExpectedBindingGeneration, CatalogGeneration: request.ExpectedCatalogGeneration,
	}
	candidate.ActiveTasks = s.candidate.ActiveTasks
	return candidate, nil
}

func (s *atomicStore) ReleaseAssignment(_ context.Context, request ReleaseRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.released[request.TaskID] {
		return nil
	}
	want, exists := s.active[request.TaskID]
	if !exists || want != request {
		return ErrNoApprovedAssignment
	}
	delete(s.active, request.TaskID)
	s.released[request.TaskID] = true
	s.candidate.ActiveTasks--
	return nil
}

func (s *atomicStore) snapshot() Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.candidate
}

func TestResolveConcurrentClaimsReserveOnlyConfiguredCapacity(t *testing.T) {
	const workers, limit = 128, 7
	store := newAtomicStore(limit)
	resolver := NewFencedResolver(store)
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			request := approvedRequest()
			request.TaskID = fmt.Sprintf("00000000-0000-0000-0001-%012d", i)
			_, err := resolver.Resolve(context.Background(), request)
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	succeeded, capacityFailures := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrCapacityExhausted):
			capacityFailures++
		default:
			t.Errorf("unexpected concurrent result: %v", err)
		}
	}
	if succeeded != limit || capacityFailures != workers-limit || store.snapshot().ActiveTasks != limit {
		t.Fatalf("success=%d capacity_failures=%d active=%d", succeeded, capacityFailures, store.snapshot().ActiveTasks)
	}
}

func TestReleaseIsFencedIdempotentAndRestoresCapacity(t *testing.T) {
	store := newAtomicStore(1)
	resolver := NewFencedResolver(store)
	request := approvedRequest()
	if _, err := resolver.Resolve(context.Background(), request); err != nil {
		t.Fatalf("Resolve() error=%v", err)
	}
	release := ReleaseRequest{
		TaskID: request.TaskID, BindingID: request.BindingID,
		BindingGeneration: request.ExpectedBindingGeneration, CatalogGeneration: request.ExpectedCatalogGeneration,
	}
	stale := release
	stale.CatalogGeneration++
	if err := resolver.Release(context.Background(), stale); !errors.Is(err, ErrNoApprovedAssignment) {
		t.Fatalf("stale Release() error=%v", err)
	}
	if got := store.snapshot().ActiveTasks; got != 1 {
		t.Fatalf("stale release changed capacity to %d", got)
	}
	if err := resolver.Release(context.Background(), release); err != nil {
		t.Fatalf("Release() error=%v", err)
	}
	if err := resolver.Release(context.Background(), release); err != nil {
		t.Fatalf("idempotent Release() error=%v", err)
	}
	if got := store.snapshot().ActiveTasks; got != 0 {
		t.Fatalf("released active tasks=%d", got)
	}
}
