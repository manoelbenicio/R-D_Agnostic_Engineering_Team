package gateway

import (
	"sort"
	"sync"
	"sync/atomic"
	"testing"
)

func TestSelectorStrictIndependentRoundRobinIsAtomicUnderConcurrency(t *testing.T) {
	accounts := []string{"acct-a", "acct-b", "acct-c"}
	selector, err := NewSelector(RotationStrictIndependentRequest, accounts...)
	if err != nil {
		t.Fatalf("NewSelector: %v", err)
	}
	const requestCount = 120
	results := make(chan Selection, requestCount)
	release := make(chan struct{})
	var workers sync.WaitGroup
	var active, peak atomic.Int64
	for i := 0; i < requestCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			selection, err := selector.Select(ContinuationRefs{})
			if err != nil {
				results <- Selection{}
				return
			}
			current := active.Add(1)
			for observed := peak.Load(); current > observed && !peak.CompareAndSwap(observed, current); observed = peak.Load() {
			}
			results <- selection
			<-release
			active.Add(-1)
		}()
	}
	ordered := make([]Selection, 0, requestCount)
	for i := 0; i < requestCount; i++ {
		ordered = append(ordered, <-results)
	}
	close(release)
	workers.Wait()

	sort.Slice(ordered, func(l, r int) bool { return ordered[l].Sequence < ordered[r].Sequence })
	seen := make(map[uint64]bool, requestCount)
	for i, selection := range ordered {
		if selection.Sequence != uint64(i+1) {
			t.Fatalf("non-atomic sequence at %d: got %d", i, selection.Sequence)
		}
		if seen[selection.Sequence] {
			t.Fatalf("duplicate sequence %d", selection.Sequence)
		}
		seen[selection.Sequence] = true
		if selection.Account != accounts[i%len(accounts)] {
			t.Fatalf("round-robin mismatch at %d: got %s want %s", i, selection.Account, accounts[i%len(accounts)])
		}
		if selection.Reason != SelectionIndependentRotation {
			t.Fatalf("independent request reason mismatch: %s", selection.Reason)
		}
	}
	if peak.Load() < 2 {
		t.Fatalf("expected concurrent execution, peak=%d", peak.Load())
	}
	if active.Load() != 0 {
		t.Fatalf("active slots leaked: %d", active.Load())
	}
}

func TestSelectorContinuationAffinityPinsAccountAndPreservesIndependentRotation(t *testing.T) {
	cases := []struct {
		name   string
		refs   ContinuationRefs
		reason SelectionReason
	}{
		{"previous_response_id", ContinuationRefs{PreviousResponseID: "resp-1"}, SelectionContinuation},
		{"prompt_cache", ContinuationRefs{PromptCacheID: "cache-1"}, SelectionPromptCache},
		{"tool_turn", ContinuationRefs{ToolTurnID: "tool-1"}, SelectionToolTurn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b", "acct-c")
			if err != nil {
				t.Fatal(err)
			}
			origin, err := selector.Select(tc.refs)
			if err != nil {
				t.Fatal(err)
			}
			if origin.Reason != tc.reason {
				t.Fatalf("origin reason=%s want %s", origin.Reason, tc.reason)
			}
			// An interleaved independent request must not be captured by affinity.
			independent, err := selector.Select(ContinuationRefs{})
			if err != nil {
				t.Fatal(err)
			}
			continuation, err := selector.Select(tc.refs)
			if err != nil {
				t.Fatal(err)
			}
			if continuation.Account != origin.Account {
				t.Fatalf("affinity broke: origin=%s continuation=%s", origin.Account, continuation.Account)
			}
			if independent.Account == origin.Account {
				t.Fatalf("independent request stole the affinity owner %s", origin.Account)
			}
			if continuation.Reason != tc.reason {
				t.Fatalf("continuation reason=%s want %s", continuation.Reason, tc.reason)
			}
		})
	}
}

func TestSelectorAffinityRebindsFromIneligibleOwnerAndDoesNotRevert(t *testing.T) {
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b", "acct-c")
	if err != nil {
		t.Fatal(err)
	}
	refs := ContinuationRefs{PreviousResponseID: "resp-rebind"}
	origin, err := selector.Select(refs)
	if err != nil {
		t.Fatal(err)
	}
	selector.SetStatus(origin.Account, AccountQuarantined)
	rebind, err := selector.Select(refs)
	if err != nil {
		t.Fatal(err)
	}
	if rebind.Account == origin.Account {
		t.Fatalf("continuation stayed on ineligible owner %s", origin.Account)
	}
	// Original owner recovers, but the continuation must stay with the replacement.
	selector.SetStatus(origin.Account, AccountEligible)
	continued, err := selector.Select(refs)
	if err != nil {
		t.Fatal(err)
	}
	if continued.Account != rebind.Account {
		t.Fatalf("stale owner reclaimed continuation: got %s want %s", continued.Account, rebind.Account)
	}
}

func TestSelectorFailsClosedWhenNoEligibleAccount(t *testing.T) {
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b")
	if err != nil {
		t.Fatal(err)
	}
	selector.SetStatus("acct-a", AccountRemoved)
	selector.SetStatus("acct-b", AccountCooldown)
	_, err = selector.Select(ContinuationRefs{})
	if !IsErrorClass(err, ErrorNoEligibleAccount) {
		t.Fatalf("expected fail-closed no-eligible-account, got %v", err)
	}
}

func TestSelectorSkipsIneligibleAccountsInStrictRotation(t *testing.T) {
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b", "acct-c")
	if err != nil {
		t.Fatal(err)
	}
	selector.SetStatus("acct-b", AccountQuarantined)
	got := map[string]int{}
	for i := 0; i < 12; i++ {
		selection, err := selector.Select(ContinuationRefs{})
		if err != nil {
			t.Fatal(err)
		}
		if selection.Account == "acct-b" {
			t.Fatal("selected quarantined account")
		}
		got[selection.Account]++
	}
	if got["acct-a"] != 6 || got["acct-c"] != 6 {
		t.Fatalf("imbalanced rotation over eligible pool: %v", got)
	}
}

func TestSelectorFailureOnlyRotationIsStickyUntilIneligible(t *testing.T) {
	selector, err := NewSelector(RotationFailureOnly, "acct-a", "acct-b", "acct-c")
	if err != nil {
		t.Fatal(err)
	}
	first, err := selector.Select(ContinuationRefs{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		next, err := selector.Select(ContinuationRefs{})
		if err != nil {
			t.Fatal(err)
		}
		if next.Account != first.Account {
			t.Fatalf("failure-only rotation was not sticky: %s then %s", first.Account, next.Account)
		}
	}
	selector.SetStatus(first.Account, AccountQuarantined)
	moved, err := selector.Select(ContinuationRefs{})
	if err != nil {
		t.Fatal(err)
	}
	if moved.Account == first.Account {
		t.Fatal("failure-only rotation did not advance past an ineligible account")
	}
}

func TestNewSelectorRejectsInvalidPools(t *testing.T) {
	if _, err := NewSelector(RotationMode("bogus"), "acct-a"); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for bad rotation, got %v", err)
	}
	if _, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-a"); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for duplicate account, got %v", err)
	}
	if _, err := NewSelector(RotationStrictIndependentRequest, ""); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for blank account, got %v", err)
	}
}
