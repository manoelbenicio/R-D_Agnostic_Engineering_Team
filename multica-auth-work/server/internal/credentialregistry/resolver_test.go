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

type reserveFunc func(context.Context, Request) (Candidate, error)

func (f reserveFunc) ReserveAssignment(ctx context.Context, request Request) (Candidate, error) {
	return f(ctx, request)
}

func approvedCandidate() Candidate {
	return Candidate{
		AgentID:              "agent-a",
		WorkspaceID:          "workspace-a",
		Provider:             "antigravity",
		HomeRef:              "home_01JABCDEFGHJKMNPQRSTVWXYZ",
		BindingGeneration:    11,
		CatalogGeneration:    17,
		Approval:             ApprovalApproved,
		Status:               StatusAvailable,
		WorktypeScope:        "GENERAL",
		AssignmentOwners:     1,
		ActiveTasks:          3,
		TaskConcurrencyLimit: 4,
	}
}

func approvedRequest() Request {
	return Request{
		AgentID:                   "agent-a",
		WorkspaceID:               "workspace-a",
		Provider:                  "agy",
		ExpectedBindingGeneration: 11,
		ExpectedCatalogGeneration: 17,
	}
}

func TestCanonicalProvider(t *testing.T) {
	for input, want := range map[string]string{
		"agy":         "antigravity",
		" AGY ":       "antigravity",
		"ANTIGRAVITY": "antigravity",
		" kiro ":      "kiro",
		"codex":       "codex",
		"openclaw":    "openclaw",
	} {
		if got := CanonicalProvider(input); got != want {
			t.Fatalf("CanonicalProvider(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestRequiresApprovedAssignment(t *testing.T) {
	for _, provider := range []string{"agy", "antigravity", "codex", "kiro"} {
		if !RequiresApprovedAssignment(provider) {
			t.Fatalf("%s must require an approved assignment", provider)
		}
	}
	for _, provider := range []string{"claude", "openclaw", "custom", ""} {
		if RequiresApprovedAssignment(provider) {
			t.Fatalf("%s must not be enrolled implicitly", provider)
		}
	}
}

func TestValidHomeRef(t *testing.T) {
	for _, ref := range []HomeRef{
		"home_01JABCDEFGHJKMNPQRSTVWXYZ",
		"0194f956-5e6c-7f6b-9a76-1b2759d2bf31",
		"catalog.home~opaque_0001",
	} {
		if !ValidHomeRef(ref) {
			t.Fatalf("ValidHomeRef(%q)=false, want true", ref)
		}
	}
	for _, ref := range []HomeRef{
		"", "short", "/var/lib/credential/home", `C:\\credential-home`,
		"https://catalog/home", "home ref with spaces", "../opaque-home-reference", "home_opaque\nreference",
	} {
		if ValidHomeRef(ref) {
			t.Fatalf("ValidHomeRef(%q)=true, want false", ref)
		}
	}
}

func TestResolveAtomicallyReservesApprovedAssignment(t *testing.T) {
	for _, status := range []AccountStatus{StatusAvailable, StatusLeased} {
		t.Run(string(status), func(t *testing.T) {
			candidate := approvedCandidate()
			candidate.Status = status
			calls := 0
			resolver := NewResolver(reserveFunc(func(_ context.Context, request Request) (Candidate, error) {
				calls++
				if request != (Request{
					AgentID:                   "agent-a",
					WorkspaceID:               "workspace-a",
					Provider:                  "antigravity",
					ExpectedBindingGeneration: 11,
					ExpectedCatalogGeneration: 17,
				}) {
					t.Fatalf("reservation request=%+v", request)
				}
				return candidate, nil
			}))

			got, err := resolver.Resolve(context.Background(), approvedRequest())
			if err != nil {
				t.Fatalf("Resolve() error=%v", err)
			}
			if calls != 1 {
				t.Fatalf("atomic reservation calls=%d, want 1", calls)
			}
			if got.HomeRef != candidate.HomeRef || got.BindingGeneration != 11 ||
				got.CatalogGeneration != 17 || got.Provider != "antigravity" ||
				got.Status != status || got.TaskConcurrencyLimit != 4 {
				t.Fatalf("Resolve()=%+v", got)
			}
		})
	}
}

func TestResolveFailsClosed(t *testing.T) {
	storeFailure := errors.New("database unavailable at /private/path")
	tests := []struct {
		name      string
		mutate    func(*Candidate)
		request   Request
		storeErr  error
		wantError error
	}{
		{name: "nil resolver", request: Request{}, wantError: ErrNoApprovedAssignment},
		{name: "missing agent", request: Request{WorkspaceID: "workspace-a", Provider: "kiro", ExpectedBindingGeneration: 11, ExpectedCatalogGeneration: 17}, wantError: ErrNoApprovedAssignment},
		{name: "missing workspace", request: Request{AgentID: "agent-a", Provider: "kiro", ExpectedBindingGeneration: 11, ExpectedCatalogGeneration: 17}, wantError: ErrNoApprovedAssignment},
		{name: "uncovered provider", request: Request{AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "claude", ExpectedBindingGeneration: 11, ExpectedCatalogGeneration: 17}, wantError: ErrNoApprovedAssignment},
		{name: "missing binding fence", mutate: nil, request: Request{AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "agy", ExpectedCatalogGeneration: 17}, wantError: ErrGenerationConflict},
		{name: "missing catalog fence", mutate: nil, request: Request{AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "agy", ExpectedBindingGeneration: 11}, wantError: ErrGenerationConflict},
		{name: "missing row", storeErr: ErrNoApprovedAssignment, wantError: ErrNoApprovedAssignment},
		{name: "store generation conflict", storeErr: ErrGenerationConflict, wantError: ErrGenerationConflict},
		{name: "store exclusive conflict", storeErr: ErrExclusiveAssignmentConflict, wantError: ErrExclusiveAssignmentConflict},
		{name: "store capacity exhausted", storeErr: ErrCapacityExhausted, wantError: ErrCapacityExhausted},
		{name: "store failure is bounded", storeErr: storeFailure, wantError: ErrRegistryUnavailable},
		{name: "agent mismatch", mutate: func(c *Candidate) { c.AgentID = "agent-b" }, wantError: ErrNoApprovedAssignment},
		{name: "workspace mismatch", mutate: func(c *Candidate) { c.WorkspaceID = "workspace-b" }, wantError: ErrNoApprovedAssignment},
		{name: "approval pending", mutate: func(c *Candidate) { c.Approval = ApprovalPending }, wantError: ErrNoApprovedAssignment},
		{name: "approval revoked", mutate: func(c *Candidate) { c.Approval = ApprovalRevoked }, wantError: ErrNoApprovedAssignment},
		{name: "provider mismatch", mutate: func(c *Candidate) { c.Provider = "kiro" }, wantError: ErrProviderMismatch},
		{name: "stored provider alias drift", mutate: func(c *Candidate) { c.Provider = "agy" }, wantError: ErrProviderMismatch},
		{name: "stale binding generation", mutate: func(c *Candidate) { c.BindingGeneration++ }, wantError: ErrGenerationConflict},
		{name: "stale catalog generation", mutate: func(c *Candidate) { c.CatalogGeneration++ }, wantError: ErrGenerationConflict},
		{name: "no owner", mutate: func(c *Candidate) { c.AssignmentOwners = 0 }, wantError: ErrNoApprovedAssignment},
		{name: "multiple owners", mutate: func(c *Candidate) { c.AssignmentOwners = 2 }, wantError: ErrExclusiveAssignmentConflict},
		{name: "unavailable status", mutate: func(c *Candidate) { c.Status = "cooldown" }, wantError: ErrAccountUnavailable},
		{name: "unsupported scope", mutate: func(c *Candidate) { c.WorktypeScope = "HEAVY" }, wantError: ErrInvalidMetadata},
		{name: "missing home ref", mutate: func(c *Candidate) { c.HomeRef = "" }, wantError: ErrInvalidMetadata},
		{name: "path home ref", mutate: func(c *Candidate) { c.HomeRef = "/private/credential/home" }, wantError: ErrInvalidMetadata},
		{name: "missing binding generation", mutate: func(c *Candidate) { c.BindingGeneration = 0 }, wantError: ErrGenerationConflict},
		{name: "missing catalog generation", mutate: func(c *Candidate) { c.CatalogGeneration = 0 }, wantError: ErrGenerationConflict},
		{name: "reservation not recorded", mutate: func(c *Candidate) { c.ActiveTasks = 0 }, wantError: ErrInvalidMetadata},
		{name: "negative active tasks", mutate: func(c *Candidate) { c.ActiveTasks = -1 }, wantError: ErrInvalidMetadata},
		{name: "missing concurrency policy", mutate: func(c *Candidate) { c.TaskConcurrencyLimit = 0 }, wantError: ErrInvalidMetadata},
		{name: "capacity exceeded", mutate: func(c *Candidate) { c.ActiveTasks = c.TaskConcurrencyLimit + 1 }, wantError: ErrCapacityExhausted},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var resolver *Resolver
			calls := 0
			if test.name != "nil resolver" {
				resolver = NewResolver(reserveFunc(func(context.Context, Request) (Candidate, error) {
					calls++
					candidate := approvedCandidate()
					if test.mutate != nil {
						test.mutate(&candidate)
					}
					return candidate, test.storeErr
				}))
			}
			request := test.request
			if request == (Request{}) && test.name != "nil resolver" {
				request = approvedRequest()
			}
			got, err := resolver.Resolve(context.Background(), request)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Resolve() error=%v, want %v", err, test.wantError)
			}
			if got != (Assignment{}) {
				t.Fatalf("Resolve() on failure returned %+v", got)
			}
			if (test.name == "missing binding fence" || test.name == "missing catalog fence") && calls != 0 {
				t.Fatalf("unfenced request reached store")
			}
			if test.storeErr == storeFailure && strings.Contains(err.Error(), "/private/path") {
				t.Fatalf("bounded error leaked store detail: %v", err)
			}
		})
	}
}

func TestAssignmentSurfaceIsPathless(t *testing.T) {
	candidateType := reflect.TypeOf(Candidate{})
	for _, forbidden := range []string{"AccountID", "HomeDir", "ConfigDir", "Path", "SecretRef", "Credential"} {
		if _, ok := candidateType.FieldByName(forbidden); ok {
			t.Fatalf("Candidate exposes forbidden field %s", forbidden)
		}
	}

	resolver := NewResolver(reserveFunc(func(context.Context, Request) (Candidate, error) {
		return approvedCandidate(), nil
	}))
	assignment, err := resolver.Resolve(context.Background(), approvedRequest())
	if err != nil {
		t.Fatalf("Resolve() error=%v", err)
	}
	encoded, err := json.Marshal(assignment)
	if err != nil {
		t.Fatalf("Marshal() error=%v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"agent-a", "workspace-a", "account", "home_dir", "config_dir", "/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("assignment JSON %s contains forbidden %q", text, forbidden)
		}
	}
}

type atomicStore struct {
	mu        sync.Mutex
	candidate Candidate
}

func newAtomicStore(activeTasks, limit int) *atomicStore {
	candidate := approvedCandidate()
	candidate.ActiveTasks = activeTasks
	candidate.TaskConcurrencyLimit = limit
	return &atomicStore{candidate: candidate}
}

func (s *atomicStore) ReserveAssignment(_ context.Context, request Request) (Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate := s.candidate
	if candidate.AgentID != request.AgentID || candidate.WorkspaceID != request.WorkspaceID ||
		candidate.Approval != ApprovalApproved {
		return Candidate{}, ErrNoApprovedAssignment
	}
	if candidate.Provider != request.Provider {
		return Candidate{}, ErrProviderMismatch
	}
	if candidate.BindingGeneration != request.ExpectedBindingGeneration ||
		candidate.CatalogGeneration != request.ExpectedCatalogGeneration {
		return Candidate{}, ErrGenerationConflict
	}
	if candidate.AssignmentOwners != 1 {
		return Candidate{}, ErrExclusiveAssignmentConflict
	}
	if candidate.Status != StatusAvailable && candidate.Status != StatusLeased {
		return Candidate{}, ErrAccountUnavailable
	}
	if candidate.ActiveTasks >= candidate.TaskConcurrencyLimit {
		return Candidate{}, ErrCapacityExhausted
	}
	s.candidate.ActiveTasks++
	return s.candidate, nil
}

func (s *atomicStore) snapshot() Candidate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.candidate
}

func (s *atomicStore) revoke() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.candidate.Approval = ApprovalRevoked
	s.candidate.BindingGeneration++
}

func (s *atomicStore) publishBindingGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.candidate.BindingGeneration++
}

func (s *atomicStore) publishCatalogGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.candidate.CatalogGeneration++
}

func TestResolveConcurrentClaimsReserveOnlyConfiguredCapacity(t *testing.T) {
	const (
		workers = 128
		limit   = 7
	)
	store := newAtomicStore(0, limit)
	resolver := NewResolver(store)
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := resolver.Resolve(context.Background(), approvedRequest())
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	succeeded := 0
	capacityFailures := 0
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
	if succeeded != limit || capacityFailures != workers-limit {
		t.Fatalf("success=%d capacity_failures=%d, want %d/%d", succeeded, capacityFailures, limit, workers-limit)
	}
	if got := store.snapshot().ActiveTasks; got != limit {
		t.Fatalf("active tasks=%d, want %d", got, limit)
	}
}

func TestResolveRevokedAndStaleGenerationsDoNotReserve(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*atomicStore)
		wantError error
	}{
		{name: "revoked binding", change: (*atomicStore).revoke, wantError: ErrNoApprovedAssignment},
		{name: "stale binding generation", change: (*atomicStore).publishBindingGeneration, wantError: ErrGenerationConflict},
		{name: "stale catalog generation", change: (*atomicStore).publishCatalogGeneration, wantError: ErrGenerationConflict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newAtomicStore(0, 1)
			test.change(store)
			_, err := NewResolver(store).Resolve(context.Background(), approvedRequest())
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Resolve() error=%v, want %v", err, test.wantError)
			}
			if got := store.snapshot().ActiveTasks; got != 0 {
				t.Fatalf("failed fenced claim reserved %d tasks", got)
			}
		})
	}
}

func TestResolveNeverRetriesOrRemapsConflict(t *testing.T) {
	calls := 0
	resolver := NewResolver(reserveFunc(func(context.Context, Request) (Candidate, error) {
		calls++
		return Candidate{}, fmt.Errorf("changed during claim: %w", ErrGenerationConflict)
	}))
	_, err := resolver.Resolve(context.Background(), approvedRequest())
	if !errors.Is(err, ErrGenerationConflict) {
		t.Fatalf("Resolve() error=%v, want generation conflict", err)
	}
	if calls != 1 {
		t.Fatalf("reservation calls=%d, want exactly one", calls)
	}
}
