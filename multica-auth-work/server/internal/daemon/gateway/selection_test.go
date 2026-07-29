package gateway

import (
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestSelectorFirstContinuationCapableRequestSelectsIndependentlyThenBindsOwner(t *testing.T) {
	// Defect 2: a continuation-capable request that does not yet reference prior
	// state is independent; ownership is established only by Bind after success,
	// not by rotation at admission.
	cases := []struct {
		name    string
		handle  ContinuationRefs
		refName SelectionReason
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
			// The first turn is independent (creates state); it must not pre-bind.
			first, err := selector.Select(ContinuationRefs{})
			if err != nil {
				t.Fatal(err)
			}
			if first.Reason != SelectionIndependentRotation {
				t.Fatalf("first turn reason=%s want independent", first.Reason)
			}
			if selector.BindingCount() != 0 {
				t.Fatalf("independent selection created a binding: count=%d", selector.BindingCount())
			}
			// The owner is bound explicitly to the account that actually served it.
			if err := selector.Bind(tc.handle, first.Account); err != nil {
				t.Fatalf("Bind: %v", err)
			}
			// Interleaved independent requests must not steal the affinity owner.
			independent, err := selector.Select(ContinuationRefs{})
			if err != nil {
				t.Fatal(err)
			}
			if independent.Account == first.Account {
				t.Fatalf("independent request reused affinity owner %s", first.Account)
			}
			// A continuation referencing the produced handle pins to the owner.
			continuation, err := selector.Select(tc.handle)
			if err != nil {
				t.Fatal(err)
			}
			if continuation.Account != first.Account {
				t.Fatalf("continuation not pinned to owner: got %s want %s", continuation.Account, first.Account)
			}
			if continuation.Reason != tc.refName {
				t.Fatalf("continuation reason=%s want %s", continuation.Reason, tc.refName)
			}
		})
	}
}

func TestSelectorReferencingUnknownContinuationFailsClosedUnderOriginAffinity(t *testing.T) {
	// Defect 2/3: a reference to state with no known owner must fail closed
	// under origin-account affinity, not silently rotate to an arbitrary account.
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b")
	if err != nil {
		t.Fatal(err)
	}
	_, err = selector.Select(ContinuationRefs{PreviousResponseID: "unknown-resp"})
	if !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("expected fail-closed continuation-unavailable, got %v", err)
	}
}

func TestSelectorIneligiblePinnedOwnerFailsClosedUnderOriginAffinity(t *testing.T) {
	// Defect 3: an ineligible pinned owner must NOT be silently replaced for a
	// stateful continuation under origin-account affinity.
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a", "acct-b", "acct-c")
	if err != nil {
		t.Fatal(err)
	}
	handle := ContinuationRefs{PreviousResponseID: "resp-pin"}
	origin, err := selector.Select(ContinuationRefs{})
	if err != nil {
		t.Fatal(err)
	}
	if err := selector.Bind(handle, origin.Account); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	// Owner leaves rotation: continuation must fail closed, not reroute.
	selector.SetStatus(origin.Account, AccountQuarantined)
	if _, err := selector.Select(handle); !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("ineligible owner was silently replaced: got %v", err)
	}
	// When the owner recovers, the continuation is honored again on it.
	selector.SetStatus(origin.Account, AccountEligible)
	recovered, err := selector.Select(handle)
	if err != nil {
		t.Fatalf("recovered owner not honored: %v", err)
	}
	if recovered.Account != origin.Account {
		t.Fatalf("recovered continuation routed to %s want owner %s", recovered.Account, origin.Account)
	}
}

func TestSelectorStatelessMaterializeRebindsFromIneligibleOwner(t *testing.T) {
	// Defect 3 (other path): only a route that explicitly declares stateless
	// materialization may rebind an ineligible/unknown continuation owner.
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation: RotationStrictIndependentRequest,
		Affinity: AffinityStatelessMaterialize,
		Accounts: []string{"acct-a", "acct-b", "acct-c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	handle := ContinuationRefs{PreviousResponseID: "resp-materialize"}
	// Unknown reference materializes a fresh owner instead of failing closed.
	origin, err := selector.Select(handle)
	if err != nil {
		t.Fatalf("stateless materialize should not fail closed on unknown ref: %v", err)
	}
	// Repeated references stay pinned while the owner is eligible.
	again, err := selector.Select(handle)
	if err != nil || again.Account != origin.Account {
		t.Fatalf("stateless materialize did not pin: got %v err=%v", again.Account, err)
	}
	// Owner leaves rotation: rebinds to a new eligible account, and does not
	// revert once the original recovers.
	selector.SetStatus(origin.Account, AccountQuarantined)
	rebind, err := selector.Select(handle)
	if err != nil || rebind.Account == origin.Account {
		t.Fatalf("stateless materialize did not rebind off ineligible owner: got %v err=%v", rebind.Account, err)
	}
	selector.SetStatus(origin.Account, AccountEligible)
	continued, err := selector.Select(handle)
	if err != nil || continued.Account != rebind.Account {
		t.Fatalf("stale owner reclaimed materialized continuation: got %v want %s", continued.Account, rebind.Account)
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
	if _, err := NewSelectorFromConfig(SelectorConfig{Rotation: RotationStrictIndependentRequest, Affinity: AffinityMode("bogus"), Accounts: []string{"acct-a"}}); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for bad affinity, got %v", err)
	}
	if _, err := NewSelectorFromConfig(SelectorConfig{Rotation: RotationStrictIndependentRequest, Affinity: AffinityOriginAccount, Accounts: []string{"acct-a"}, MaxBindings: -1}); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config for negative max bindings, got %v", err)
	}
}

func TestSelectorBindingsAreBoundedAndReturnToZero(t *testing.T) {
	// Defect 4 (selector): binding state must be bounded by explicit release and
	// TTL expiry, must return to zero after completion, and must fail closed
	// rather than evict a live continuation when full.
	clock := time.Unix(1_700_000_000, 0)
	nowFn := func() time.Time { return clock }
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation:    RotationStrictIndependentRequest,
		Affinity:    AffinityOriginAccount,
		Accounts:    []string{"acct-a", "acct-b"},
		BindingTTL:  time.Minute,
		MaxBindings: 2,
		now:         nowFn,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "r1"}, "acct-a"); err != nil {
		t.Fatal(err)
	}
	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "r2"}, "acct-b"); err != nil {
		t.Fatal(err)
	}
	if selector.BindingCount() != 2 {
		t.Fatalf("binding count=%d want 2", selector.BindingCount())
	}
	// Table full of live entries: a new distinct binding fails closed rather
	// than evicting an active continuation.
	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "r3"}, "acct-a"); !IsErrorClass(err, ErrorContinuationCapacity) {
		t.Fatalf("expected capacity fail-closed, got %v", err)
	}
	// Explicit release returns state toward zero and frees capacity.
	selector.Unbind(ContinuationRefs{PreviousResponseID: "r1"})
	if selector.BindingCount() != 1 {
		t.Fatalf("binding count after unbind=%d want 1", selector.BindingCount())
	}
	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "r3"}, "acct-a"); err != nil {
		t.Fatalf("bind after release should succeed: %v", err)
	}
	// TTL expiry reclaims all live entries once the clock advances past TTL.
	clock = clock.Add(2 * time.Minute)
	if selector.BindingCount() != 0 {
		t.Fatalf("expired bindings not reclaimed: count=%d", selector.BindingCount())
	}
	// An expired binding is treated as unknown and fails closed under origin affinity.
	if _, err := selector.Select(ContinuationRefs{PreviousResponseID: "r2"}); !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("expired binding still honored: %v", err)
	}
}

func TestSelectorBindRejectsUnknownAccountAndBlankHandle(t *testing.T) {
	selector, err := NewSelector(RotationStrictIndependentRequest, "acct-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "r1"}, "acct-unknown"); !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("expected invalid request for unknown account, got %v", err)
	}
	if err := selector.Bind(ContinuationRefs{}, "acct-a"); !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("expected invalid request for blank handle, got %v", err)
	}
}

// --- W2 second-pass regression tests (F3 AffinityNone, F4 TTL refresh + bounded accounts) ---

func TestSelectorAffinityNoneFailsClosedForStatefulRefs(t *testing.T) {
	// F3: a route with no continuation affinity must fail closed for any
	// stateful reference (both Select and Bind), never rotate it to an
	// arbitrary account. Independent requests still rotate.
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation: RotationStrictIndependentRequest,
		Affinity: AffinityNone,
		Accounts: []string{"acct-a", "acct-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Independent request rotates.
	if _, err := selector.Select(ContinuationRefs{}); err != nil {
		t.Fatalf("independent request should rotate under AffinityNone: %v", err)
	}
	// Stateful reference fails closed.
	for _, refs := range []ContinuationRefs{
		{PreviousResponseID: "resp-x"},
		{PromptCacheID: "cache-x"},
		{ToolTurnID: "tool-x"},
	} {
		if _, err := selector.Select(refs); !IsErrorClass(err, ErrorContinuationUnavailable) {
			t.Fatalf("AffinityNone stateful select should fail closed, got %v", err)
		}
	}
	// Bind fails closed too.
	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "resp-x"}, "acct-a"); !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("AffinityNone bind should fail closed, got %v", err)
	}
}

func TestSelectorAffinityHitRefreshesBindingLifetime(t *testing.T) {
	// F4: an actively-used continuation must never be reclaimed by the expiry
	// sweep; each affinity hit refreshes the binding's lifetime.
	clock := time.Unix(1_700_000_000, 0)
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation:   RotationStrictIndependentRequest,
		Affinity:   AffinityOriginAccount,
		Accounts:   []string{"acct-a", "acct-b"},
		BindingTTL: time.Minute,
		now:        func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	handle := ContinuationRefs{PreviousResponseID: "resp-live"}
	if err := selector.Bind(handle, "acct-a"); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	// Two hits 50s apart: total 100s > 60s TTL. Without refresh the second hit
	// would fail closed after expiry; with refresh it stays pinned.
	clock = clock.Add(50 * time.Second)
	if sel, err := selector.Select(handle); err != nil || sel.Account != "acct-a" {
		t.Fatalf("first hit should refresh and pin: sel=%v err=%v", sel.Account, err)
	}
	clock = clock.Add(50 * time.Second)
	if sel, err := selector.Select(handle); err != nil || sel.Account != "acct-a" {
		t.Fatalf("active continuation was reclaimed despite use: sel=%v err=%v", sel.Account, err)
	}
	// Once genuinely idle past TTL, it expires and fails closed.
	clock = clock.Add(2 * time.Minute)
	if selector.BindingCount() != 0 {
		t.Fatalf("idle binding not reclaimed: count=%d", selector.BindingCount())
	}
	if _, err := selector.Select(handle); !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("expired binding still honored: %v", err)
	}
}

func TestSelectorAccountLifecycleIsBounded(t *testing.T) {
	// F4: account order/status growth is bounded; Add fails closed at the cap,
	// and Remove hard-prunes order/status and the removed account's bindings.
	selector, err := NewSelectorFromConfig(SelectorConfig{
		Rotation:    RotationStrictIndependentRequest,
		Affinity:    AffinityOriginAccount,
		Accounts:    []string{"acct-a", "acct-b"},
		MaxAccounts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := selector.Add("acct-c"); err != nil {
		t.Fatalf("Add within bound: %v", err)
	}
	if err := selector.Add("acct-d"); !IsErrorClass(err, ErrorAccountCapacity) {
		t.Fatalf("Add beyond bound should fail closed, got %v", err)
	}
	if selector.AccountCount() != 3 {
		t.Fatalf("account count=%d want 3", selector.AccountCount())
	}
	// Re-adding a known account is idempotent (no growth).
	if err := selector.Add("acct-a"); err != nil || selector.AccountCount() != 3 {
		t.Fatalf("idempotent re-add changed count: err=%v count=%d", err, selector.AccountCount())
	}
	// Bind to acct-c, then hard-remove it: order/status and its bindings prune,
	// freeing capacity and failing the continuation closed.
	if err := selector.Bind(ContinuationRefs{PreviousResponseID: "resp-c"}, "acct-c"); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	selector.Remove("acct-c")
	if selector.AccountCount() != 2 {
		t.Fatalf("Remove did not prune account: count=%d", selector.AccountCount())
	}
	if selector.BindingCount() != 0 {
		t.Fatalf("Remove did not prune owned binding: count=%d", selector.BindingCount())
	}
	if _, err := selector.Select(ContinuationRefs{PreviousResponseID: "resp-c"}); !IsErrorClass(err, ErrorContinuationUnavailable) {
		t.Fatalf("continuation on removed owner should fail closed, got %v", err)
	}
	// Capacity freed by Remove allows a fresh add.
	if err := selector.Add("acct-d"); err != nil {
		t.Fatalf("Add after Remove freed capacity should succeed: %v", err)
	}
	// Removed account still routes independent requests only to live accounts.
	for i := 0; i < 6; i++ {
		sel, err := selector.Select(ContinuationRefs{})
		if err != nil {
			t.Fatal(err)
		}
		if sel.Account == "acct-c" {
			t.Fatal("removed account was selected")
		}
	}
}

func TestNewSelectorRejectsAccountsExceedingBound(t *testing.T) {
	if _, err := NewSelectorFromConfig(SelectorConfig{
		Rotation:    RotationStrictIndependentRequest,
		Affinity:    AffinityOriginAccount,
		Accounts:    []string{"acct-a", "acct-b", "acct-c"},
		MaxAccounts: 2,
	}); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("expected invalid config when initial accounts exceed bound, got %v", err)
	}
}
