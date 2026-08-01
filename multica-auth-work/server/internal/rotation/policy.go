package rotation

import "fmt"

// PolicyType selects the routing strategy for a rotation policy.
type PolicyType string

const (
	PolicyTypeFallback      PolicyType = "FALLBACK"
	PolicyTypeLoadBalancing PolicyType = "LOAD_BALANCING"
	PolicyTypeLatency       PolicyType = "LATENCY"
)

// WorkType classifies the task shape used to resolve a named policy.
type WorkType string

const (
	WorkTypeGeneral WorkType = "GENERAL"
	WorkTypeHeavy   WorkType = "HEAVY"
	WorkTypeCheap   WorkType = "CHEAP"
	WorkTypeReview  WorkType = "REVIEW"
)

// PolicyItem describes one account/vendor candidate in a policy chain.
type PolicyItem struct {
	Vendor        string
	AccountRef    string
	Retries       int
	Weight        int
	CredentialSrc string
}

// RotationPolicy is the named policy model used by the rotation router.
type RotationPolicy struct {
	Name     string
	Type     PolicyType
	WorkType WorkType
	Items    []PolicyItem
}

const policyRetryMax = 10

var defaultPolicies = map[string]RotationPolicy{
	"general": {
		Name:     "general",
		Type:     PolicyTypeFallback,
		WorkType: WorkTypeGeneral,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "kiro", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "antigravity", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
		},
	},
	"heavy": {
		Name:     "heavy",
		Type:     PolicyTypeFallback,
		WorkType: WorkTypeHeavy,
		Items: []PolicyItem{
			{Vendor: "kiro", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "codex", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "antigravity", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
		},
	},
	"cheap": {
		Name:     "cheap",
		Type:     PolicyTypeFallback,
		WorkType: WorkTypeCheap,
		Items: []PolicyItem{
			{Vendor: "cline", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "opencode", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "codex", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
		},
	},
	"review": {
		Name:     "review",
		Type:     PolicyTypeFallback,
		WorkType: WorkTypeReview,
		Items: []PolicyItem{
			{Vendor: "codex", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "kiro", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
			{Vendor: "antigravity", AccountRef: "any-of-vendor", Retries: 1, CredentialSrc: "registry"},
		},
	},
}

// ResolvePolicy returns a copy of a named default policy.
func ResolvePolicy(name string) (RotationPolicy, error) {
	p, ok := defaultPolicies[name]
	if !ok {
		return RotationPolicy{}, fmt.Errorf("rotation policy %q: unknown policy", name)
	}
	p.Items = clonePolicyItems(p.Items)
	if err := p.Validate(); err != nil {
		return RotationPolicy{}, err
	}
	return p, nil
}

// Ordered returns policy items in fallback priority order.
func (p RotationPolicy) Ordered() []PolicyItem {
	items := clonePolicyItems(p.Items)
	normalizePolicyItems(items)
	return items
}

// Validate verifies the policy and normalizes zero retries to the default of 1.
func (p RotationPolicy) Validate() error {
	if !validPolicyType(p.Type) {
		return fmt.Errorf("rotation policy %q: invalid type %q", p.Name, p.Type)
	}
	if !validWorkType(p.WorkType) {
		return fmt.Errorf("rotation policy %q: invalid work type %q", p.Name, p.WorkType)
	}
	if len(p.Items) == 0 {
		return fmt.Errorf("rotation policy %q: items must be non-empty", p.Name)
	}
	for i := range p.Items {
		if p.Items[i].Retries == 0 {
			p.Items[i].Retries = 1
		}
		if p.Items[i].Retries < 0 || p.Items[i].Retries > policyRetryMax {
			return fmt.Errorf("rotation policy %q item %d: retries must be between 0 and 10", p.Name, i)
		}
		if p.Items[i].Weight < 0 {
			return fmt.Errorf("rotation policy %q item %d: weight must be non-negative", p.Name, i)
		}
		if p.Type != PolicyTypeLoadBalancing && p.Items[i].Weight != 0 {
			return fmt.Errorf("rotation policy %q item %d: weight is only meaningful for LOAD_BALANCING", p.Name, i)
		}
	}
	return nil
}

func validPolicyType(t PolicyType) bool {
	switch t {
	case PolicyTypeFallback, PolicyTypeLoadBalancing, PolicyTypeLatency:
		return true
	default:
		return false
	}
}

func validWorkType(t WorkType) bool {
	switch t {
	case WorkTypeGeneral, WorkTypeHeavy, WorkTypeCheap, WorkTypeReview:
		return true
	default:
		return false
	}
}

func clonePolicyItems(items []PolicyItem) []PolicyItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]PolicyItem, len(items))
	copy(out, items)
	return out
}

func normalizePolicyItems(items []PolicyItem) {
	for i := range items {
		if items[i].Retries == 0 {
			items[i].Retries = 1
		}
	}
}
