package rotation

import "testing"

func TestPolicyResolveDefaultSet(t *testing.T) {
	cases := []struct {
		name     string
		wantType PolicyType
		wantWork WorkType
	}{
		{"general", PolicyTypeFallback, WorkTypeGeneral},
		{"heavy", PolicyTypeFallback, WorkTypeHeavy},
		{"cheap", PolicyTypeFallback, WorkTypeCheap},
		{"review", PolicyTypeFallback, WorkTypeReview},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolvePolicy(tc.name)
			if err != nil {
				t.Fatalf("ResolvePolicy(%q) returned error: %v", tc.name, err)
			}
			if got.Name != tc.name {
				t.Fatalf("ResolvePolicy(%q).Name = %q, want %q", tc.name, got.Name, tc.name)
			}
			if got.Type != tc.wantType {
				t.Fatalf("ResolvePolicy(%q).Type = %q, want %q", tc.name, got.Type, tc.wantType)
			}
			if got.WorkType != tc.wantWork {
				t.Fatalf("ResolvePolicy(%q).WorkType = %q, want %q", tc.name, got.WorkType, tc.wantWork)
			}
			if len(got.Items) == 0 {
				t.Fatalf("ResolvePolicy(%q).Items is empty", tc.name)
			}
		})
	}
}

func TestPolicyResolveUnknown(t *testing.T) {
	if _, err := ResolvePolicy("nope"); err == nil {
		t.Fatal("ResolvePolicy(\"nope\") returned nil error, want error")
	}
}

func TestPolicyReviewOrderedByStrength(t *testing.T) {
	got, err := ResolvePolicy("review")
	if err != nil {
		t.Fatalf("ResolvePolicy(\"review\") returned error: %v", err)
	}

	items := got.Ordered()
	if len(items) != 3 {
		t.Fatalf("len(review.Ordered()) = %d, want 3", len(items))
	}

	wantVendors := []string{"codex", "kiro", "antigravity"}
	for i, want := range wantVendors {
		if items[i].Vendor != want {
			t.Fatalf("review.Ordered()[%d].Vendor = %q, want %q", i, items[i].Vendor, want)
		}
	}

	items[0].Vendor = "changed"
	again := got.Ordered()
	if again[0].Vendor != "codex" {
		t.Fatalf("Ordered returned mutable backing storage, got first vendor %q", again[0].Vendor)
	}
}

func TestPolicyValidateNormalizesRetries(t *testing.T) {
	p := RotationPolicy{
		Name:     "test",
		Type:     PolicyTypeFallback,
		WorkType: WorkTypeGeneral,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "acct", Retries: 0, CredentialSrc: "registry"},
		},
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if p.Items[0].Retries != 1 {
		t.Fatalf("Validate normalized retries to %d, want 1", p.Items[0].Retries)
	}
}

func TestPolicyValidate(t *testing.T) {
	base := RotationPolicy{
		Name:     "test",
		Type:     PolicyTypeFallback,
		WorkType: WorkTypeGeneral,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "acct", Retries: 1, CredentialSrc: "registry"},
		},
	}

	cases := []struct {
		name string
		edit func(*RotationPolicy)
	}{
		{
			name: "invalid_type",
			edit: func(p *RotationPolicy) {
				p.Type = PolicyType("ROUND_ROBIN")
			},
		},
		{
			name: "invalid_work_type",
			edit: func(p *RotationPolicy) {
				p.WorkType = WorkType("FAST")
			},
		},
		{
			name: "empty_items",
			edit: func(p *RotationPolicy) {
				p.Items = nil
			},
		},
		{
			name: "negative_retries",
			edit: func(p *RotationPolicy) {
				p.Items[0].Retries = -1
			},
		},
		{
			name: "retries_too_high",
			edit: func(p *RotationPolicy) {
				p.Items[0].Retries = 11
			},
		},
		{
			name: "fallback_weight",
			edit: func(p *RotationPolicy) {
				p.Items[0].Weight = 1
			},
		},
		{
			name: "negative_weight",
			edit: func(p *RotationPolicy) {
				p.Type = PolicyTypeLoadBalancing
				p.Items[0].Weight = -1
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			p.Items = clonePolicyItems(base.Items)
			tc.edit(&p)
			if err := p.Validate(); err == nil {
				t.Fatal("Validate returned nil error, want error")
			}
		})
	}
}

func TestPolicyValidateLoadBalancingWeight(t *testing.T) {
	p := RotationPolicy{
		Name:     "lb",
		Type:     PolicyTypeLoadBalancing,
		WorkType: WorkTypeGeneral,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "acct", Retries: 1, Weight: 10, CredentialSrc: "registry"},
		},
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("Validate LOAD_BALANCING policy returned error: %v", err)
	}
}
