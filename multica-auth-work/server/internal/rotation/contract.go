// Package rotation defines the automatic account-rotation domain for
// credential exhaustion (5h/quota) handling.
//
// CONTRACT FILE (published by Opus 4.8, orchestrator). This file intentionally
// contains ONLY the shared types and interfaces (the "mold") so the detector,
// service, and Postgres store can be implemented IN PARALLEL by different
// agents without depending on each other. Do NOT add logic here. Owners of the
// other files program against these signatures.
//
// File ownership (no collisions):
//   - detector.go        (W-DETECT)  — implements ExhaustionDetector
//   - service.go/pool.go (W-ROTATE)  — implements RotationService, uses Store + AccountAuthenticator
//   - store_pg.go         (W-PGSTORE) — implements Store (Postgres)
//   - auth_*.go           (later)     — implements AccountAuthenticator (e.g. Indra adapter)
package rotation

import (
	"context"
	"errors"
	"time"
)

// ErrNoAccountAvailable is returned by SelectNext when the vendor pool has no
// selectable account (all leased/exhausted/cooldown/degraded).
var ErrNoAccountAvailable = errors.New("rotation: no account available")

// AccountStatus is the lifecycle state of a vendor account in the pool.
type AccountStatus string

const (
	StatusAvailable AccountStatus = "available" // has quota, free to use
	StatusLeased    AccountStatus = "leased"    // currently serving one or more agents
	StatusExhausted AccountStatus = "exhausted" // quota hit; waiting for cooldown
	StatusCooldown  AccountStatus = "cooldown"  // timed cooldown window
	StatusDegraded  AccountStatus = "degraded"  // login/credential problem; skip
)

// RotationReason explains why a rotation was triggered.
type RotationReason string

const (
	ReasonQuotaReactive  RotationReason = "quota_exhausted_reactive"  // on-screen / 429
	ReasonQuotaProactive RotationReason = "quota_forecast_proactive"  // ledger near cap
	ReasonLoginFailed    RotationReason = "login_failed"
	ReasonManual         RotationReason = "manual"
)

// ExhaustionSignal identifies how exhaustion was detected.
type ExhaustionSignal string

const (
	SignalScreen  ExhaustionSignal = "screen"
	SignalHTTP429 ExhaustionSignal = "http429"
	SignalLedger  ExhaustionSignal = "ledger"
)

// Account is one vendor subscription that can serve agents. Its credential is
// isolated per account (home_dir/config_dir) per the Phase-1 mechanism.
type Account struct {
	AccountID     string
	Vendor        string
	TenantID      string
	Priority      int // lower = higher priority (expertise order)
	HomeDir       string
	ConfigDir     string
	Status        AccountStatus
	TokensPerWin  int64
	TokensUsed    int64
	WindowStart   *time.Time
	CooldownUntil *time.Time
	LastError     string
}

// DetectionResult is what the detector returns for a single observation.
type DetectionResult struct {
	Exhausted bool
	Signal    ExhaustionSignal
	ResetAt   *time.Time // parsed vendor reset time when present
}

// ExhaustionDetector inspects a vendor observation (screen text and/or HTTP
// status) and reports whether the current account is exhausted. Implemented by
// detector.go (W-DETECT). MUST distinguish transient 503/"high traffic" (NOT
// exhaustion) from real quota limits.
type ExhaustionDetector interface {
	Detect(vendor, screenText string, httpStatus int) DetectionResult
}

// AccountAuthenticator is the port for switching credentials on a vendor
// account. Implemented later by a concrete adapter (e.g. Indra). Device-login /
// OAuth only — no passwords on disk.
type AccountAuthenticator interface {
	Login(ctx context.Context, acc Account) (sessionID string, err error)
	Logout(ctx context.Context, acc Account) error
	WaitAuthenticated(ctx context.Context, sessionID string, timeout time.Duration) (bool, error)
}

// Store is the persistence port. Implemented by store_pg.go (W-PGSTORE) on
// Postgres ONLY. Credential material is stored by reference (KMS/secret ref),
// never as plaintext, and never logged.
type Store interface {
	ListAccounts(ctx context.Context, vendor, tenantID string) ([]Account, error)
	GetAccount(ctx context.Context, accountID string) (Account, error)
	UpdateAccountStatus(ctx context.Context, accountID string, status AccountStatus, cooldownUntil *time.Time) error
	RecordUsage(ctx context.Context, accountID string, tokensUsed int64, windowStart time.Time) error
	Assign(ctx context.Context, agentID, accountID string) error
	CurrentAssignment(ctx context.Context, agentID string) (accountID string, err error)
	RecordRotation(ctx context.Context, agentID, fromAccountID, toAccountID string, reason RotationReason, at time.Time) error
}

// RotationService orchestrates the exhaustion→switch→resume loop. Implemented
// by service.go (W-ROTATE), composing a Store, an ExhaustionDetector, and an
// AccountAuthenticator. Selection follows expertise priority.
type RotationService interface {
	// SelectNext returns the next selectable account for the vendor by
	// priority, skipping leased-only-when-required / exhausted / cooldown /
	// degraded per policy. Returns ErrNoAccountAvailable when none.
	SelectNext(ctx context.Context, vendor, tenantID string, now time.Time) (Account, error)
	// OnExhaustion runs the full rotation for an agent whose current account
	// is exhausted, returning the account it rotated to.
	OnExhaustion(ctx context.Context, agentID, vendor, tenantID string, reason RotationReason, now time.Time) (Account, error)
}
