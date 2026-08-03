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
	"fmt"
	"time"

	"github.com/google/uuid"
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
	ReasonQuotaReactive  RotationReason = "quota_exhausted_reactive" // on-screen / 429
	ReasonQuotaProactive RotationReason = "quota_forecast_proactive" // ledger near cap
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

// CredentialExpiryReader exposes the authoritative credentials.expires_at
// timestamp without expanding the mutation-oriented Store contract.
type CredentialExpiryReader interface {
	CredentialExpiresAt(ctx context.Context, accountID string) (*time.Time, error)
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

// NativeHomeIdentityV1 is the complete pathless, value-free identity pinned by
// runtime_task_home_epoch. All fields are server-derived; no account, path,
// credential reference/value, token, or provider response is representable.
type NativeHomeIdentityV1 struct {
	TaskID               string
	WorkspaceID          string
	AgentID              string
	RuntimeID            string
	RuntimeSessionID     string
	DaemonID             string
	DaemonBootID         string
	Provider             string
	RuntimeBindingID     string
	BindingGeneration    int64
	HomeAssignmentID     string
	CatalogID            string
	CatalogEntryID       string
	CatalogGeneration    int64
	HomeRef              string
	HomeEpoch            int64
	LifetimeID           string
	AcquisitionRequestID string
}

type NativeHomeFenceV1 struct {
	LifetimeState         string
	LifetimeStateVersion  int16
	ProcessIdentityDigest string
}

type NativeHomeReceiptV1 struct {
	Identity NativeHomeIdentityV1
	Fence    NativeHomeFenceV1
}

type NativeRotationRequestV1 struct {
	OperationID         string
	OperationRequestID  string
	TransitionRequestID string
	AssignedBy          string
	Current             NativeHomeReceiptV1
	Target              NativeHomeIdentityV1
	ReasonCode          string
}

type NativeRotationOutcome string

const (
	NativeDefinitelyNotCommitted     NativeRotationOutcome = "definitely_not_committed"
	NativeCommittedRetirementPending NativeRotationOutcome = "committed_retirement_pending"
	NativeCommittedRetired           NativeRotationOutcome = "committed_retired"
	NativeCommitUnknownFenced        NativeRotationOutcome = "commit_unknown_fenced"
	NativeStaleIdentityConflict      NativeRotationOutcome = "stale_identity_conflict"
)

type NativeRotationResultV1 struct {
	OperationRequestID string
	Outcome            NativeRotationOutcome
	Current            NativeHomeReceiptV1
	Target             NativeHomeReceiptV1
	ReasonCode         string
	RetryAt            *time.Time
}

type NativeRetirementRequestV1 struct {
	AttemptID           string
	RetirementRequestID string
	OperationID         string
	Retiring            NativeHomeReceiptV1
	DaemonID            string
	DaemonBootID        string
	RuntimeSessionID    string
	RuntimeBindingID    string
	BindingGeneration   int64
	AttemptNumber       int16
}

type NativeRetirementResultV1 struct {
	AttemptID            string
	RetirementRequestID  string
	State                string
	ResultCode           string
	ChannelBindingDigest string
	RequestBodyDigest    string
}

// NativeRetirementTransitionV1 binds value-free transport evidence to the
// immutable attempt identity. It contains no credential or home path.
type NativeRetirementTransitionV1 struct {
	AttemptID            string
	RetirementRequestID  string
	DaemonID             string
	DaemonBootID         string
	RuntimeSessionID     string
	RuntimeBindingID     string
	BindingGeneration    int64
	ChannelBindingDigest string
	RequestBodyDigest    string
}

var (
	ErrInvalidNativeIdentity = errors.New("rotation: invalid native identity")
	ErrNativeCommitUnknown   = errors.New("rotation: native swap commit unknown and fenced")
	ErrNativeCASConflict     = errors.New("rotation: native state changed")
)

// NativeRotationStore owns only durable database phases. Implementations must
// commit before any local/provider operation and reacquire the full lock chain
// for every later phase.
type NativeRotationStore interface {
	ReserveNativeCandidate(context.Context, NativeRotationRequestV1) (NativeHomeReceiptV1, error)
	RecordNativePreparation(context.Context, NativeRotationRequestV1, bool, string) (NativeHomeReceiptV1, error)
	CommitNativeSwap(context.Context, NativeRotationRequestV1, string) (NativeRotationResultV1, error)
	ResolveNativeCommit(context.Context, NativeRotationRequestV1) (NativeRotationResultV1, error)
}

// NativeHomePreparer is a local-only port. The receipt remains opaque and
// pathless at the server boundary. Implementations are section-5 wiring and are
// intentionally not composed by this core commit.
type NativeHomePreparer interface {
	PrepareNativeHome(context.Context, NativeHomeReceiptV1) (processIdentityDigest string, err error)
}

func (i NativeHomeIdentityV1) validate() error {
	for name, value := range map[string]string{
		"task_id": i.TaskID, "workspace_id": i.WorkspaceID,
		"agent_id": i.AgentID, "runtime_id": i.RuntimeID,
		"runtime_session_id": i.RuntimeSessionID, "daemon_boot_id": i.DaemonBootID,
		"runtime_binding_id": i.RuntimeBindingID,
		"home_assignment_id": i.HomeAssignmentID, "catalog_id": i.CatalogID,
		"catalog_entry_id": i.CatalogEntryID, "home_ref": i.HomeRef,
		"lifetime_id":            i.LifetimeID,
		"acquisition_request_id": i.AcquisitionRequestID,
	} {
		if _, err := uuid.Parse(value); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidNativeIdentity, name)
		}
	}
	if i.DaemonID == "" || len(i.DaemonID) > 128 {
		return fmt.Errorf("%w: daemon_id", ErrInvalidNativeIdentity)
	}
	if i.Provider != "antigravity" && i.Provider != "codex" && i.Provider != "kiro" {
		return fmt.Errorf("%w: provider", ErrInvalidNativeIdentity)
	}
	if i.BindingGeneration < 1 || i.CatalogGeneration < 1 || i.HomeEpoch < 1 {
		return fmt.Errorf("%w: generation", ErrInvalidNativeIdentity)
	}
	return nil
}

func (r NativeRotationRequestV1) validate() error {
	if err := r.Current.Identity.validate(); err != nil {
		return err
	}
	if err := r.Target.validate(); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"operation_id":          r.OperationID,
		"operation_request_id":  r.OperationRequestID,
		"transition_request_id": r.TransitionRequestID,
		"assigned_by":           r.AssignedBy,
	} {
		if _, err := uuid.Parse(value); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidNativeIdentity, name)
		}
	}
	if r.Current.Identity.TaskID != r.Target.TaskID ||
		r.Current.Identity.WorkspaceID != r.Target.WorkspaceID ||
		r.Current.Identity.AgentID != r.Target.AgentID ||
		r.Current.Identity.RuntimeID != r.Target.RuntimeID ||
		r.Current.Identity.RuntimeSessionID != r.Target.RuntimeSessionID ||
		r.Current.Identity.RuntimeBindingID != r.Target.RuntimeBindingID ||
		r.Current.Identity.BindingGeneration != r.Target.BindingGeneration ||
		r.Current.Identity.DaemonID != r.Target.DaemonID ||
		r.Current.Identity.DaemonBootID != r.Target.DaemonBootID ||
		r.Target.HomeEpoch != r.Current.Identity.HomeEpoch+1 ||
		r.Target.HomeRef == r.Current.Identity.HomeRef {
		return fmt.Errorf("%w: cross-linked epoch", ErrInvalidNativeIdentity)
	}
	if r.ReasonCode == "" || len(r.ReasonCode) > 64 {
		return fmt.Errorf("%w: reason_code", ErrInvalidNativeIdentity)
	}
	return nil
}
