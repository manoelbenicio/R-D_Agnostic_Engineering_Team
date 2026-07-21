package commitledger

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// ReplayDecision is the gate verdict for whether an automatic retry is safe.
type ReplayDecision uint8

const (
	// ReplayAllowed means the ledger shows NO tool activity ever occurred —
	// replaying is safe because there's no risk of duplicate side effects.
	ReplayAllowed ReplayDecision = iota
	// ReplayBlocked means tool activity occurred (definite or ambiguous) or
	// the state is unknown/corrupt — replaying could cause duplicates.
	ReplayBlocked
)

func (d ReplayDecision) String() string {
	switch d {
	case ReplayAllowed:
		return "allowed"
	case ReplayBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// ReplayGateResult contains the full decision context.
// No raw task IDs — use pseudonymous correlation only.
type ReplayGateResult struct {
	Decision   ReplayDecision
	Reason     string
	FailClosed bool
	Summary    DurableSummary
}

// ReplayGate evaluates whether an automatic retry/replay is safe for a task.
//
// Decision logic (frozen contract):
//   - nil ledger → blocked (fail closed)
//   - fail-closed ledger → blocked
//   - summary.EverHadToolUse → blocked (any tool activity blocks)
//   - summary.EverDefinite → blocked (committed side effects)
//   - summary.EverAmbiguous → blocked (uncertain side effects)
//   - summary.EverSaturated → blocked (overflow, can't trust state)
//   - otherwise (no tool activity ever) → allowed
//
// Manual RerunIssue (explicit user action) is exempt from this gate.
func ReplayGate(ledger *Ledger) ReplayGateResult {
	if ledger == nil {
		return ReplayGateResult{
			Decision: ReplayBlocked,
			Reason:   "no ledger available; fail closed",
		}
	}

	if ledger.IsFailClosed() {
		return ReplayGateResult{
			Decision:   ReplayBlocked,
			Reason:     "ledger is fail-closed (schema mismatch, corrupt, or saturated)",
			FailClosed: true,
			Summary:    ledger.Summary(),
		}
	}

	summary := ledger.Summary()

	if summary.BlocksReplay() {
		reason := "tool activity occurred"
		if summary.EverAmbiguous {
			reason = "ambiguous tool state — outcome unknown"
		} else if summary.EverDefinite {
			reason = "definite tool commits — side effects occurred"
		} else if summary.EverSaturated {
			reason = "ledger saturated — cannot trust state"
		} else if summary.EverHadToolUse {
			reason = "tool_use recorded — side effects may have occurred"
		}
		return ReplayGateResult{
			Decision: ReplayBlocked,
			Reason:   reason,
			Summary:  summary,
		}
	}

	// No tool activity ever — safe to replay
	return ReplayGateResult{
		Decision: ReplayAllowed,
		Reason:   "no tool activity; replay is safe",
		Summary:  summary,
	}
}

// ErrReplayBlocked is returned when an automatic retry is blocked.
var ErrReplayBlocked = errors.New("commitledger: replay blocked")

// LedgerRegistry is a thread-safe registry of active task ledgers.
// This is the in-process cache; durable state lives server-side.
type LedgerRegistry struct {
	mu      sync.RWMutex
	ledgers map[string]*Ledger // keyed by pseudonymous task correlation
}

// NewLedgerRegistry creates an empty registry.
func NewLedgerRegistry() *LedgerRegistry {
	return &LedgerRegistry{
		ledgers: make(map[string]*Ledger),
	}
}

// Register adds a ledger. Replaces any existing entry.
func (r *LedgerRegistry) Register(correlationID string, ledger *Ledger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ledgers[correlationID] = ledger
}

// Unregister removes a ledger.
func (r *LedgerRegistry) Unregister(correlationID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.ledgers, correlationID)
}

// Get retrieves a ledger. Returns nil if not found.
func (r *LedgerRegistry) Get(correlationID string) *Ledger {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ledgers[correlationID]
}

// CheckReplayAllowed looks up the ledger and runs the replay gate.
// Returns nil if allowed, ErrReplayBlocked if blocked.
// Missing ledger → fail closed.
func (r *LedgerRegistry) CheckReplayAllowed(correlationID string) error {
	ledger := r.Get(correlationID)
	result := ReplayGate(ledger)
	if result.Decision == ReplayBlocked {
		return fmt.Errorf("%w: %s", ErrReplayBlocked, result.Reason)
	}
	return nil
}

// ReplayGateHook is the interface that TaskService uses to consult the
// commit ledger before allowing an automatic retry. Nil-safe: if nil,
// the check fails closed (blocks replay — durable state unavailable).
type ReplayGateHook struct {
	Registry *LedgerRegistry
	Logger   *slog.Logger
}

// NewReplayGateHook creates a production replay gate hook.
func NewReplayGateHook(registry *LedgerRegistry, logger *slog.Logger) *ReplayGateHook {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReplayGateHook{
		Registry: registry,
		Logger:   logger,
	}
}

// Check evaluates whether an automatic retry is allowed for the given parent.
// Uses pseudonymous correlation ID (not raw task ID).
// Returns nil if allowed, non-nil if blocked.
//
// Any nil/missing state fails closed (blocks replay).
func (h *ReplayGateHook) Check(correlationID string) error {
	if h == nil || h.Registry == nil {
		return fmt.Errorf("%w: no ledger registry available; fail closed", ErrReplayBlocked)
	}

	ledger := h.Registry.Get(correlationID)
	result := ReplayGate(ledger)

	if result.Decision == ReplayBlocked {
		h.Logger.Warn("replay gate blocked automatic retry",
			"reason", result.Reason,
			"ever_tool_use", result.Summary.EverHadToolUse,
			"ever_definite", result.Summary.EverDefinite,
			"ever_ambiguous", result.Summary.EverAmbiguous,
			"ever_saturated", result.Summary.EverSaturated,
			"fail_closed", result.FailClosed,
		)
		return fmt.Errorf("%w: %s", ErrReplayBlocked, result.Reason)
	}

	h.Logger.Debug("replay gate allowed automatic retry")
	return nil
}

// CheckOrAllow is like Check but when hook is nil, BLOCKS (fail closed).
// This is the safe default: if the ledger system isn't wired, we cannot
// verify safety, so we must block.
func CheckOrAllow(hook *ReplayGateHook, correlationID string) error {
	if hook == nil {
		return fmt.Errorf("%w: replay gate hook not configured; fail closed", ErrReplayBlocked)
	}
	return hook.Check(correlationID)
}
