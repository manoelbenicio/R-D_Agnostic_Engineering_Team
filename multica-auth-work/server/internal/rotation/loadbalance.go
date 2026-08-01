package rotation

import "github.com/cespare/xxhash/v2"

// PickWeighted chooses a policy item by normalized weight using a deterministic
// hash of seed. Non-positive weights are treated as 1 so unweighted policies
// still distribute across all items.
func PickWeighted(p RotationPolicy, seed string) PolicyItem {
	items := p.Ordered()
	if len(items) == 0 {
		return PolicyItem{}
	}

	total := uint64(0)
	for _, item := range items {
		total += uint64(effectiveLoadBalanceWeight(item))
	}
	if total == 0 {
		return PolicyItem{}
	}

	bucket := xxhash.Sum64String(seed) % total
	running := uint64(0)
	for _, item := range items {
		running += uint64(effectiveLoadBalanceWeight(item))
		if bucket < running {
			return item
		}
	}
	return items[len(items)-1]
}

// PickConsistent chooses the same policy item for the same traceID, preserving
// task/session affinity for context and cache reuse.
func PickConsistent(p RotationPolicy, traceID string) PolicyItem {
	items := p.Ordered()
	if len(items) == 0 {
		return PolicyItem{}
	}
	return items[xxhash.Sum64String(traceID)%uint64(len(items))]
}

// PickByWindowHealth chooses the item with the most quota-window remaining.
// The router sends new load toward the healthiest account so subscription
// windows drain together and aggregate throughput is maximized.
func PickByWindowHealth(items []PolicyItem, health map[string]float64) PolicyItem {
	if len(items) == 0 {
		return PolicyItem{}
	}

	best := items[0]
	bestHealth := healthValue(best, health)
	for _, item := range items[1:] {
		itemHealth := healthValue(item, health)
		if itemHealth > bestHealth {
			best = item
			bestHealth = itemHealth
		}
	}
	return best
}

func effectiveLoadBalanceWeight(item PolicyItem) int {
	if item.Weight <= 0 {
		return 1
	}
	return item.Weight
}

func healthValue(item PolicyItem, health map[string]float64) float64 {
	if health == nil {
		return 0
	}
	return health[item.AccountRef]
}
