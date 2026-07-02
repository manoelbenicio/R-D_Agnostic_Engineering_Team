package rotation

import (
	"sort"
	"time"
)

const (
	defaultProactiveThreshold = 0.95
	quotaWindowDuration       = 5 * time.Hour
)

type ProactiveDetector struct {
	threshold float64
}

func NewProactiveDetector(threshold float64) *ProactiveDetector {
	if threshold <= 0 || threshold > 1 {
		threshold = defaultProactiveThreshold
	}
	return &ProactiveDetector{threshold: threshold}
}

// ShouldRotate detects quota exhaustion before the vendor hard-stop by reading
// the account ledger. The window is fixed at five hours: if the ledger window
// has already expired, usage is treated as reset and the account is not marked
// exhausted. Otherwise, TokensUsed/TokensPerWin at or above the threshold
// (default 95%) emits SignalLedger.
func (d *ProactiveDetector) ShouldRotate(acc Account, now time.Time) DetectionResult {
	threshold := defaultProactiveThreshold
	if d != nil && d.threshold > 0 && d.threshold <= 1 {
		threshold = d.threshold
	}
	if acc.TokensPerWin <= 0 {
		return DetectionResult{}
	}

	var resetAt *time.Time
	if acc.WindowStart != nil {
		reset := acc.WindowStart.Add(quotaWindowDuration)
		if !now.Before(reset) {
			return DetectionResult{}
		}
		resetAt = &reset
	}

	if float64(acc.TokensUsed)/float64(acc.TokensPerWin) < threshold {
		return DetectionResult{}
	}
	return DetectionResult{
		Exhausted: true,
		Signal:    SignalLedger,
		ResetAt:   resetAt,
	}
}

func AccountsNeedingProactiveRotation(accounts []Account, now time.Time, detector *ProactiveDetector) []Account {
	if detector == nil {
		detector = NewProactiveDetector(0)
	}
	var out []Account
	for _, acc := range accounts {
		if detector.ShouldRotate(acc, now).Exhausted {
			out = append(out, acc)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].AccountID < out[j].AccountID
		}
		return out[i].Priority < out[j].Priority
	})
	return out
}
