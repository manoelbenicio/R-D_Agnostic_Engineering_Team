package gateway

import (
	"sync"
	"testing"
)

// collectConcurrentSelections drives n independent selections concurrently and
// returns them ordered by the Selector's atomic sequence. It asserts the
// sequence is exactly 1..n with no gaps or duplicates, proving selection is
// atomic under concurrency regardless of goroutine scheduling.
func collectConcurrentSelections(t *testing.T, selector *Selector, n int) []Selection {
	t.Helper()
	results := make(chan Selection, n)
	var workers sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			selection, err := selector.Select(ContinuationRefs{})
			if err != nil {
				results <- Selection{}
				return
			}
			results <- selection
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	ordered := make([]Selection, 0, n)
	for selection := range results {
		if selection.Sequence == 0 {
			t.Fatal("eligible selector rejected a request under load")
		}
		ordered = append(ordered, selection)
	}
	if len(ordered) != n {
		t.Fatalf("collected %d selections, want %d", len(ordered), n)
	}
	seqSort(ordered)
	base := ordered[0].Sequence
	if base == 0 {
		t.Fatal("selection sequence must be positive")
	}
	for i, selection := range ordered {
		// The Selector's sequence is monotonic per Selector across calls, so a
		// batch is contiguous from its own base rather than restarting at 1.
		if selection.Sequence != base+uint64(i) {
			t.Fatalf("non-atomic sequence at %d: got %d want %d", i, selection.Sequence, base+uint64(i))
		}
	}
	return ordered
}

func seqSort(selections []Selection) {
	for i := 1; i < len(selections); i++ {
		for j := i; j > 0 && selections[j-1].Sequence > selections[j].Sequence; j-- {
			selections[j-1], selections[j] = selections[j], selections[j-1]
		}
	}
}

// assertBalancedStrictRotation proves the selections form a strict round-robin
// over exactly the eligible set: only eligible accounts appear, the order has
// period k (selections[i] == selections[i+k]), adjacent selections differ, and
// each eligible account is selected exactly n/k times. n MUST be a multiple of
// k = len(eligible).
func assertBalancedStrictRotation(t *testing.T, selections []Selection, eligible []string) {
	t.Helper()
	eligibleSet := make(map[string]struct{}, len(eligible))
	for _, account := range eligible {
		eligibleSet[account] = struct{}{}
	}
	k := len(eligible)
	if k == 0 || len(selections)%k != 0 {
		t.Fatalf("selection count %d not a multiple of eligible size %d", len(selections), k)
	}
	counts := make(map[string]int, k)
	for i, selection := range selections {
		if _, ok := eligibleSet[selection.Account]; !ok {
			t.Fatalf("ineligible account %q selected at %d", selection.Account, i)
		}
		if selection.Reason != SelectionIndependentRotation {
			t.Fatalf("independent request reason mismatch at %d: %s", i, selection.Reason)
		}
		counts[selection.Account]++
		if k > 1 && i+1 < len(selections) && selection.Account == selections[i+1].Account {
			t.Fatalf("adjacent repeat at %d: %q (rotation not strict)", i, selection.Account)
		}
		if i+k < len(selections) && selection.Account != selections[i+k].Account {
			t.Fatalf("rotation period mismatch at %d: %q vs %q", i, selection.Account, selections[i+k].Account)
		}
	}
	for account := range eligibleSet {
		if counts[account] != len(selections)/k {
			t.Fatalf("imbalanced rotation: %q selected %d times, want %d", account, counts[account], len(selections)/k)
		}
	}
}

// TestSelectorAccountLifecycleAndReentryUnderConcurrentLoad proves the
// gateway-side 8.7 contract against the production Selector: as OmniRoute drives
// account quarantine, cooldown, removal, re-entry and additions, the gateway's
// strict round-robin only ever serves eligible accounts, stays atomic under
// concurrency, and returns to balanced steady-state rotation after re-entry.
// The authoritative lifecycle *decision* remains OmniRoute-owned; this exercises
// only the gateway rotation's reflection of those transitions.
func TestSelectorAccountLifecycleAndReentryUnderConcurrentLoad(t *testing.T) {
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b", "acct-c")
	if err != nil {
		t.Fatal(err)
	}

	// Quarantine removes an account from rotation without disturbing the rest.
	selector.SetStatus("acct-b", AccountQuarantined)
	assertBalancedStrictRotation(t, collectConcurrentSelections(t, selector, 96), []string{"acct-a", "acct-c"})

	// Cooldown is likewise ineligible for selection.
	selector.SetStatus("acct-b", AccountCooldown)
	assertBalancedStrictRotation(t, collectConcurrentSelections(t, selector, 96), []string{"acct-a", "acct-c"})

	// Removal keeps it out of rotation.
	selector.SetStatus("acct-b", AccountRemoved)
	assertBalancedStrictRotation(t, collectConcurrentSelections(t, selector, 96), []string{"acct-a", "acct-c"})

	// Re-entry returns the account to a balanced steady-state rotation.
	selector.SetStatus("acct-b", AccountEligible)
	assertBalancedStrictRotation(t, collectConcurrentSelections(t, selector, 96), []string{"acct-a", "acct-b", "acct-c"})

	// A newly added account joins the rotation without breaking atomicity.
	if err := selector.Add("acct-d"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	assertBalancedStrictRotation(t, collectConcurrentSelections(t, selector, 96), []string{"acct-a", "acct-b", "acct-c", "acct-d"})
}

// TestSelectorReAddedRemovedAccountBecomesEligibleAgain proves an account that
// OmniRoute removes and later re-admits (add/re-enable) re-enters the gateway
// rotation, mirroring the account add/remove/re-entry lifecycle of 8.7.
func TestSelectorReAddedRemovedAccountBecomesEligibleAgain(t *testing.T) {
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b")
	if err != nil {
		t.Fatal(err)
	}
	selector.SetStatus("acct-b", AccountRemoved)
	if got := selector.EligibleAccounts(); len(got) != 1 || got[0] != "acct-a" {
		t.Fatalf("removed account still eligible: %v", got)
	}
	// Re-enable via status and add a brand-new account.
	selector.SetStatus("acct-b", AccountEligible)
	if err := selector.Add("acct-c"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := selector.EligibleAccounts()
	want := map[string]bool{"acct-a": true, "acct-b": true, "acct-c": true}
	if len(got) != len(want) {
		t.Fatalf("eligible set mismatch: %v", got)
	}
	for _, account := range got {
		if !want[account] {
			t.Fatalf("unexpected eligible account %q", account)
		}
	}
	// Adding an already-known account is idempotent (no duplicate rotation slot).
	if err := selector.Add("acct-a"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got := selector.EligibleAccounts(); len(got) != 3 {
		t.Fatalf("re-adding known account changed rotation size: %v", got)
	}
}
