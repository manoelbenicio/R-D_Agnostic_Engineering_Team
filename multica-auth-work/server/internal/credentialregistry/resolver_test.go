package credentialregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type lookupFunc func(context.Context, string, string) (Candidate, error)

func (f lookupFunc) LookupAssignment(ctx context.Context, agentID, workspaceID string) (Candidate, error) {
	return f(ctx, agentID, workspaceID)
}

func approvedCandidate() Candidate {
	return Candidate{
		AgentID:              "agent-a",
		WorkspaceID:          "workspace-a",
		Provider:             "antigravity",
		HomeRef:              "home_01JABCDEFGHJKMNPQRSTVWXYZ",
		CatalogGeneration:    17,
		Approval:             ApprovalApproved,
		Status:               StatusAvailable,
		WorktypeScope:        "GENERAL",
		AssignmentOwners:     1,
		ActiveTasks:          2,
		TaskConcurrencyLimit: 4,
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

func TestResolveApprovedAssignment(t *testing.T) {
	for _, status := range []AccountStatus{StatusAvailable, StatusLeased} {
		t.Run(string(status), func(t *testing.T) {
			candidate := approvedCandidate()
			candidate.Status = status
			resolver := NewResolver(lookupFunc(func(_ context.Context, agentID, workspaceID string) (Candidate, error) {
				if agentID != "agent-a" || workspaceID != "workspace-a" {
					t.Fatalf("lookup identity=(%q,%q)", agentID, workspaceID)
				}
				return candidate, nil
			}))

			got, err := resolver.Resolve(context.Background(), Request{
				AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: " AGY ",
			})
			if err != nil {
				t.Fatalf("Resolve() error=%v", err)
			}
			if got.HomeRef != candidate.HomeRef || got.CatalogGeneration != 17 ||
				got.Provider != "antigravity" || got.Status != status || got.TaskConcurrencyLimit != 4 {
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
		{name: "missing agent", request: Request{WorkspaceID: "workspace-a", Provider: "kiro"}, wantError: ErrNoApprovedAssignment},
		{name: "missing workspace", request: Request{AgentID: "agent-a", Provider: "kiro"}, wantError: ErrNoApprovedAssignment},
		{name: "uncovered provider", request: Request{AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "claude"}, wantError: ErrNoApprovedAssignment},
		{name: "missing row", storeErr: ErrNoApprovedAssignment, wantError: ErrNoApprovedAssignment},
		{name: "store failure is bounded", storeErr: storeFailure, wantError: ErrRegistryUnavailable},
		{name: "agent mismatch", mutate: func(c *Candidate) { c.AgentID = "agent-b" }, wantError: ErrNoApprovedAssignment},
		{name: "workspace mismatch", mutate: func(c *Candidate) { c.WorkspaceID = "workspace-b" }, wantError: ErrNoApprovedAssignment},
		{name: "approval pending", mutate: func(c *Candidate) { c.Approval = ApprovalPending }, wantError: ErrNoApprovedAssignment},
		{name: "approval revoked", mutate: func(c *Candidate) { c.Approval = ApprovalRevoked }, wantError: ErrNoApprovedAssignment},
		{name: "provider mismatch", mutate: func(c *Candidate) { c.Provider = "kiro" }, wantError: ErrProviderMismatch},
		{name: "stored provider alias drift", mutate: func(c *Candidate) { c.Provider = "agy" }, wantError: ErrProviderMismatch},
		{name: "no owner", mutate: func(c *Candidate) { c.AssignmentOwners = 0 }, wantError: ErrNoApprovedAssignment},
		{name: "multiple owners", mutate: func(c *Candidate) { c.AssignmentOwners = 2 }, wantError: ErrAccountAlreadyUsed},
		{name: "unavailable status", mutate: func(c *Candidate) { c.Status = "cooldown" }, wantError: ErrAccountUnavailable},
		{name: "unsupported scope", mutate: func(c *Candidate) { c.WorktypeScope = "HEAVY" }, wantError: ErrInvalidMetadata},
		{name: "missing home ref", mutate: func(c *Candidate) { c.HomeRef = "" }, wantError: ErrInvalidMetadata},
		{name: "path home ref", mutate: func(c *Candidate) { c.HomeRef = "/private/credential/home" }, wantError: ErrInvalidMetadata},
		{name: "missing generation", mutate: func(c *Candidate) { c.CatalogGeneration = 0 }, wantError: ErrInvalidMetadata},
		{name: "negative active tasks", mutate: func(c *Candidate) { c.ActiveTasks = -1 }, wantError: ErrInvalidMetadata},
		{name: "missing concurrency policy", mutate: func(c *Candidate) { c.TaskConcurrencyLimit = 0 }, wantError: ErrInvalidMetadata},
		{name: "capacity exhausted", mutate: func(c *Candidate) { c.ActiveTasks = c.TaskConcurrencyLimit }, wantError: ErrCapacityExhausted},
		{name: "capacity exceeded", mutate: func(c *Candidate) { c.ActiveTasks = c.TaskConcurrencyLimit + 1 }, wantError: ErrCapacityExhausted},
	}

	defaultRequest := Request{AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "agy"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var resolver *Resolver
			if test.name != "nil resolver" {
				resolver = NewResolver(lookupFunc(func(context.Context, string, string) (Candidate, error) {
					candidate := approvedCandidate()
					if test.mutate != nil {
						test.mutate(&candidate)
					}
					return candidate, test.storeErr
				}))
			}
			request := test.request
			if request == (Request{}) && test.name != "nil resolver" {
				request = defaultRequest
			}
			got, err := resolver.Resolve(context.Background(), request)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Resolve() error=%v, want %v", err, test.wantError)
			}
			if got != (Assignment{}) {
				t.Fatalf("Resolve() on failure returned %+v", got)
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

	resolver := NewResolver(lookupFunc(func(context.Context, string, string) (Candidate, error) {
		return approvedCandidate(), nil
	}))
	assignment, err := resolver.Resolve(context.Background(), Request{
		AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "agy",
	})
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

func TestResolveConcurrentReadsAreDeterministic(t *testing.T) {
	const workers = 128
	var lookups atomic.Int64
	resolver := NewResolver(lookupFunc(func(context.Context, string, string) (Candidate, error) {
		lookups.Add(1)
		return approvedCandidate(), nil
	}))

	start := make(chan struct{})
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			got, err := resolver.Resolve(context.Background(), Request{
				AgentID: "agent-a", WorkspaceID: "workspace-a", Provider: "agy",
			})
			if err != nil {
				errorsCh <- err
				return
			}
			if got.HomeRef != "home_01JABCDEFGHJKMNPQRSTVWXYZ" || got.TaskConcurrencyLimit != 4 {
				errorsCh <- fmt.Errorf("unexpected assignment: %+v", got)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
	if got := lookups.Load(); got != workers {
		t.Fatalf("lookups=%d, want %d", got, workers)
	}
}
