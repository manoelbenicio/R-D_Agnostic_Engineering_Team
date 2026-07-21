package gateway

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Task 8.4 acceptance — "Demonstrate strict concurrent round-robin for
// independent requests and correct affinity for Responses continuation, prompt
// cache and tool turns." Deterministic, in-process proofs against the
// production Selector/Executor. Account pools are realistic, injected named
// sets (never hard-coded ["default"]); the eligible set is supplied by the
// caller (central wiring resolves it from the route's ModelDocument.AccountPool
// membership and passes it to NewSelectorFromConfig).

func newAcceptanceSelector(t *testing.T, accounts ...string) *Selector {
	t.Helper()
	s, err := NewSelectorFromConfig(SelectorConfig{
		Rotation: RotationStrictIndependentRequest,
		Affinity: AffinityOriginAccount,
		Accounts: accounts, // injected eligible pool (not hard-coded)
	})
	if err != nil {
		t.Fatalf("NewSelectorFromConfig: %v", err)
	}
	return s
}

// Blocker 1: strict round-robin for independent requests — sequential exact
// order AND concurrent overlapping order (goroutines parked at a barrier so
// they contend simultaneously; the atomic sequence proves one strict order).
func TestStrictRoundRobinIndependent(t *testing.T) {
	accounts := []string{"acct-blue", "acct-green", "acct-red", "acct-amber"}
	selector := newAcceptanceSelector(t, accounts...)

	// Sequential: exact expected order, no skipping/randomization.
	for i := 0; i < 3*len(accounts); i++ {
		sel, err := selector.Select(ContinuationRefs{})
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if sel.Account != accounts[i%len(accounts)] {
			t.Fatalf("request %d strict order violated: got %q want %q", i, sel.Account, accounts[i%len(accounts)])
		}
		if sel.Reason != SelectionIndependentRotation || sel.Sequence != uint64(i+1) {
			t.Fatalf("request %d: reason=%s seq=%d", i, sel.Reason, sel.Sequence)
		}
	}

	// Concurrent overlapping: all goroutines park at the barrier before release.
	concurrentSelector := newAcceptanceSelector(t, accounts...)
	const n = 120
	var parked atomic.Int64
	release := make(chan struct{})
	results := make(chan Selection, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parked.Add(1)
			<-release // ensure temporal overlap: everyone waits here
			sel, err := concurrentSelector.Select(ContinuationRefs{})
			if err != nil {
				results <- Selection{}
				return
			}
			results <- sel
		}()
	}
	for parked.Load() < n { // spin until all N are parked at the barrier
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	close(results)

	ordered := make([]Selection, 0, n)
	for sel := range results {
		if sel.Sequence == 0 {
			t.Fatal("overlapping request failed under strict round-robin")
		}
		ordered = append(ordered, sel)
	}
	if len(ordered) != n {
		t.Fatalf("got %d selections want %d", len(ordered), n)
	}
	sort.Slice(ordered, func(a, b int) bool { return ordered[a].Sequence < ordered[b].Sequence })
	for i, sel := range ordered {
		if sel.Sequence != uint64(i+1) {
			t.Fatalf("overlapping sequence gap at %d: got %d", i, sel.Sequence)
		}
		if sel.Account != accounts[i%len(accounts)] {
			t.Fatalf("overlapping strict order violated at seq %d: got %q want %q", sel.Sequence, sel.Account, accounts[i%len(accounts)])
		}
	}
}

// Blocker 2: affinity for Responses continuation / prompt cache / tool turns —
// bound (via Bind) continuation requests return the SAME account.
func TestAffinityContinuation(t *testing.T) {
	cases := []struct {
		name   string
		handle ContinuationRefs
		reason SelectionReason
	}{
		{"responses_previous_response_id", ContinuationRefs{PreviousResponseID: "resp-xyz"}, SelectionContinuation},
		{"prompt_cache", ContinuationRefs{PromptCacheID: "cache-xyz"}, SelectionPromptCache},
		{"tool_turn", ContinuationRefs{ToolTurnID: "tool-xyz"}, SelectionToolTurn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selector := newAcceptanceSelector(t, "acct-one", "acct-two", "acct-three")
			first, err := selector.Select(ContinuationRefs{})
			if err != nil {
				t.Fatal(err)
			}
			if err := selector.Bind(tc.handle, first.Account); err != nil {
				t.Fatalf("Bind: %v", err)
			}
			for k := 0; k < 6; k++ {
				cont, err := selector.Select(tc.handle)
				if err != nil {
					t.Fatalf("continuation %d: %v", k, err)
				}
				if cont.Account != first.Account {
					t.Fatalf("continuation %d not pinned: got %q want %q", k, cont.Account, first.Account)
				}
				if cont.Reason != tc.reason {
					t.Fatalf("continuation %d reason=%s want %s", k, cont.Reason, tc.reason)
				}
			}
		})
	}
}

// Blocker 3: affinity hits must NOT advance the independent round-robin cursor.
// Interleave continuation hits between independent requests and prove the
// independent subsequence is strict rotation, unaffected by the affinity hits.
func TestAffinityDoesNotAdvanceIndependentCursor(t *testing.T) {
	accounts := []string{"acct-alpha", "acct-beta", "acct-gamma"}
	selector := newAcceptanceSelector(t, accounts...)

	first, err := selector.Select(ContinuationRefs{}) // acct-alpha; cursor now at beta
	if err != nil {
		t.Fatal(err)
	}
	if first.Account != "acct-alpha" {
		t.Fatalf("first independent = %q want acct-alpha", first.Account)
	}
	handle := ContinuationRefs{PreviousResponseID: "resp-pin"}
	if err := selector.Bind(handle, first.Account); err != nil {
		t.Fatal(err)
	}

	// Each iteration: one affinity hit (must be pinned + must NOT advance the
	// cursor) then one independent request (must continue strict rotation).
	independent := make([]string, 0, 6)
	for k := 0; k < 6; k++ {
		hit, err := selector.Select(handle)
		if err != nil {
			t.Fatal(err)
		}
		if hit.Account != first.Account {
			t.Fatalf("affinity hit %d not pinned: %q", k, hit.Account)
		}
		indep, err := selector.Select(ContinuationRefs{})
		if err != nil {
			t.Fatal(err)
		}
		independent = append(independent, indep.Account)
	}

	// With no affinity influence, independents after acct-alpha rotate:
	// beta, gamma, alpha, beta, gamma, alpha.
	want := []string{"acct-beta", "acct-gamma", "acct-alpha", "acct-beta", "acct-gamma", "acct-alpha"}
	for i := range want {
		if independent[i] != want[i] {
			t.Fatalf("affinity advanced the independent cursor at %d: got %v want %v", i, independent, want)
		}
	}
}

// Blocker 4: Executor emits a pseudonymous selection record (account ALIAS, not
// the raw id) with request correlation, on a realistic injected pool.
func TestExecutorEmitsPseudonymousSelectionRecord(t *testing.T) {
	accounts := []string{"acct-one", "acct-two", "acct-three"}
	selector := newAcceptanceSelector(t, accounts...)
	coordinator, err := NewCoordinator(RetryPolicy{MaxAttempts: 2, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	executor, err := NewExecutor(selector, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var records []SelectionRecord
	executor.SetSelectionRecorder(func(r SelectionRecord) {
		mu.Lock()
		records = append(records, r)
		mu.Unlock()
	})

	rawAccounts := map[string]struct{}{"acct-one": {}, "acct-two": {}, "acct-three": {}}
	for i := 0; i < len(accounts); i++ {
		reqID := fmt.Sprintf("corr-%d", i)
		out, err := executor.Execute(context.Background(), reqID, ContinuationRefs{}, func(context.Context, string, int) ProviderOutcome {
			return ProviderOutcome{OutputCommitted: true}
		})
		if err != nil {
			t.Fatalf("execute %d: %v", i, err)
		}
		// Executor drives strict rotation across the injected pool.
		if out.Account != accounts[i%len(accounts)] {
			t.Fatalf("executor strict order at %d: got %q want %q", i, out.Account, accounts[i%len(accounts)])
		}
		// Alias is pseudonymous: present and NOT the raw id.
		if out.AccountAlias == "" || out.AccountAlias == out.Account {
			t.Fatalf("outcome alias not pseudonymous: alias=%q account=%q", out.AccountAlias, out.Account)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(records) != len(accounts) {
		t.Fatalf("got %d selection records want %d", len(records), len(accounts))
	}
	for i, r := range records {
		if r.RequestID != fmt.Sprintf("corr-%d", i) {
			t.Fatalf("record %d correlation mismatch: %q", i, r.RequestID)
		}
		if r.AccountAlias == "" {
			t.Fatalf("record %d missing pseudonymous alias", i)
		}
		if _, isRaw := rawAccounts[r.AccountAlias]; isRaw {
			t.Fatalf("record %d leaked a raw account id in the alias: %q", i, r.AccountAlias)
		}
		if r.Sequence != uint64(i+1) {
			t.Fatalf("record %d sequence=%d want %d", i, r.Sequence, i+1)
		}
		if r.Reason != SelectionIndependentRotation {
			t.Fatalf("record %d reason=%s", i, r.Reason)
		}
	}
}
