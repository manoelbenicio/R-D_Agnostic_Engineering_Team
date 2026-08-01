package rotation

import "testing"

func TestPickWeightedEmptyPolicy(t *testing.T) {
	if got := PickWeighted(RotationPolicy{}, "seed"); got != (PolicyItem{}) {
		t.Fatalf("PickWeighted(empty) = %#v, want zero PolicyItem", got)
	}
}

func TestPickWeightedStable(t *testing.T) {
	p := loadBalancePolicy()

	first := PickWeighted(p, "task-42")
	for i := 0; i < 20; i++ {
		if got := PickWeighted(p, "task-42"); got != first {
			t.Fatalf("PickWeighted stability run %d = %#v, want %#v", i, got, first)
		}
	}
}

func TestPickWeightedDistributionFollowsWeights(t *testing.T) {
	p := loadBalancePolicy()

	counts := map[string]int{}
	const samples = 12000
	for i := 0; i < samples; i++ {
		item := PickWeighted(p, string(rune(i))+"-weighted-seed")
		counts[item.AccountRef]++
	}

	assertWithin(t, "acct-a", counts["acct-a"], 1000, 250)
	assertWithin(t, "acct-b", counts["acct-b"], 3000, 450)
	assertWithin(t, "acct-c", counts["acct-c"], 8000, 650)
}

func TestPickWeightedDefaultsNonPositiveWeights(t *testing.T) {
	p := RotationPolicy{
		Name:     "unweighted",
		Type:     PolicyTypeLoadBalancing,
		WorkType: WorkTypeGeneral,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "acct-a", Weight: 0},
			{Vendor: "kiro", AccountRef: "acct-b", Weight: -3},
			{Vendor: "cline", AccountRef: "acct-c", Weight: 0},
		},
	}

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		item := PickWeighted(p, string(rune(i))+"-default-weight")
		seen[item.AccountRef] = true
	}

	for _, want := range []string{"acct-a", "acct-b", "acct-c"} {
		if !seen[want] {
			t.Fatalf("PickWeighted with default weights never selected %q; seen=%v", want, seen)
		}
	}
}

func TestPickConsistentEmptyPolicy(t *testing.T) {
	if got := PickConsistent(RotationPolicy{}, "trace"); got != (PolicyItem{}) {
		t.Fatalf("PickConsistent(empty) = %#v, want zero PolicyItem", got)
	}
}

func TestPickConsistentStable(t *testing.T) {
	p := loadBalancePolicy()

	first := PickConsistent(p, "task-42")
	for i := 0; i < 20; i++ {
		if got := PickConsistent(p, "task-42"); got != first {
			t.Fatalf("PickConsistent stability run %d = %#v, want %#v", i, got, first)
		}
	}
}

func TestPickConsistentCanSelectEveryItem(t *testing.T) {
	p := loadBalancePolicy()

	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		item := PickConsistent(p, string(rune(i))+"-consistent-seed")
		seen[item.AccountRef] = true
	}

	for _, want := range []string{"acct-a", "acct-b", "acct-c"} {
		if !seen[want] {
			t.Fatalf("PickConsistent never selected %q; seen=%v", want, seen)
		}
	}
}

func TestPickByWindowHealthEmptyItems(t *testing.T) {
	if got := PickByWindowHealth(nil, map[string]float64{"acct-a": 1}); got != (PolicyItem{}) {
		t.Fatalf("PickByWindowHealth(empty) = %#v, want zero PolicyItem", got)
	}
}

func TestPickByWindowHealthPrefersMostRemaining(t *testing.T) {
	items := []PolicyItem{
		{Vendor: "codex", AccountRef: "acct-a"},
		{Vendor: "kiro", AccountRef: "acct-b"},
		{Vendor: "cline", AccountRef: "acct-c"},
	}
	health := map[string]float64{
		"acct-a": 0.30,
		"acct-b": 0.85,
		"acct-c": 0.60,
	}

	got := PickByWindowHealth(items, health)
	if got.AccountRef != "acct-b" {
		t.Fatalf("PickByWindowHealth selected %q, want acct-b", got.AccountRef)
	}
}

func TestPickByWindowHealthTieKeepsPriorityOrder(t *testing.T) {
	items := []PolicyItem{
		{Vendor: "codex", AccountRef: "acct-a"},
		{Vendor: "kiro", AccountRef: "acct-b"},
	}
	health := map[string]float64{
		"acct-a": 0.50,
		"acct-b": 0.50,
	}

	got := PickByWindowHealth(items, health)
	if got.AccountRef != "acct-a" {
		t.Fatalf("PickByWindowHealth tie selected %q, want acct-a", got.AccountRef)
	}
}

func loadBalancePolicy() RotationPolicy {
	return RotationPolicy{
		Name:     "lb",
		Type:     PolicyTypeLoadBalancing,
		WorkType: WorkTypeGeneral,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "acct-a", Weight: 1},
			{Vendor: "kiro", AccountRef: "acct-b", Weight: 3},
			{Vendor: "cline", AccountRef: "acct-c", Weight: 8},
		},
	}
}

func assertWithin(t *testing.T, name string, got, want, tolerance int) {
	t.Helper()
	if got < want-tolerance || got > want+tolerance {
		t.Fatalf("%s count = %d, want %d ± %d", name, got, want, tolerance)
	}
}
