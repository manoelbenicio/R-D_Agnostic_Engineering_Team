// Package credentialregistry resolves metadata-only, owner-approved credential
// home assignments for native task admission. It never handles credential
// values or raw host paths.
package credentialregistry

import (
	"context"
	"errors"
	"strings"
	"unicode"
)

var (
	ErrNoApprovedAssignment        = errors.New("credential registry: no approved assignment")
	ErrProviderMismatch            = errors.New("credential registry: provider mismatch")
	ErrExclusiveAssignmentConflict = errors.New("credential registry: exclusive assignment conflict")
	ErrAccountAlreadyUsed          = ErrExclusiveAssignmentConflict
	ErrAccountUnavailable          = errors.New("credential registry: account unavailable")
	ErrInvalidMetadata             = errors.New("credential registry: invalid assignment metadata")
	ErrGenerationConflict          = errors.New("credential registry: generation conflict")
	ErrTaskConflict                = errors.New("credential registry: task claim conflict")
	ErrCapacityExhausted           = errors.New("credential registry: task concurrency exhausted")
	ErrRegistryUnavailable         = errors.New("credential registry: assignment store unavailable")
)

// HomeRef is an opaque catalog identifier. It is deliberately not a path or
// an account identity. Only the daemon-local catalog may resolve it to a host
// location.
type HomeRef string

// ApprovalState represents the durable owner decision for an assignment.
type ApprovalState string

const (
	ApprovalPending  ApprovalState = "pending"
	ApprovalApproved ApprovalState = "approved"
	ApprovalRevoked  ApprovalState = "revoked"
)

// AccountStatus is the persisted availability state of the assigned account.
type AccountStatus string

const (
	StatusAvailable AccountStatus = "available"
	StatusLeased    AccountStatus = "leased"
)

// Candidate is the locked metadata-only snapshot validated before an atomic
// reservation and returned after commit. Internal admission predicates are
// excluded from JSON. The type intentionally has no account ID, source path,
// credential reference, or secret.
type Candidate struct {
	TaskID                      string        `json:"-"`
	TaskStatus                  string        `json:"-"`
	BindingID                   string        `json:"-"`
	AgentID                     string        `json:"-"`
	WorkspaceID                 string        `json:"-"`
	RuntimeID                   string        `json:"-"`
	RuntimeSessionID            string        `json:"-"`
	StandardVersionID           string        `json:"-"`
	ConfigurationVersionID      string        `json:"-"`
	ConfigurationDigest         string        `json:"-"`
	CapabilityDigest            string        `json:"-"`
	Provider                    string        `json:"-"`
	SessionProvider             string        `json:"-"`
	HomeRef                     HomeRef       `json:"home_ref"`
	BindingGeneration           uint64        `json:"binding_generation"`
	AssignmentCatalogGeneration uint64        `json:"-"`
	CatalogGeneration           uint64        `json:"catalog_generation"`
	Approval                    ApprovalState `json:"-"`
	Status                      AccountStatus `json:"-"`
	BindingState                string        `json:"-"`
	TransportBinding            string        `json:"-"`
	AssignmentState             string        `json:"-"`
	CatalogState                string        `json:"-"`
	CatalogEntryState           string        `json:"-"`
	HealthFresh                 bool          `json:"-"`
	AssignmentOwners            int           `json:"-"`
	BindingAssignments          int           `json:"-"`
	ActiveTasks                 int           `json:"-"`
	TaskConcurrencyLimit        int           `json:"-"`
	ExistingSnapshot            bool          `json:"-"`
	SnapshotMatches             bool          `json:"-"`
}

// Request is the complete fenced identity supplied at admission. Both expected
// generations are mandatory: accepting an unfenced claim would permit a revoked
// or replaced assignment to be silently remapped.
type Request struct {
	TaskID                    string
	BindingID                 string
	AgentID                   string
	WorkspaceID               string
	Provider                  string
	ExpectedBindingGeneration uint64
	ExpectedCatalogGeneration uint64
}

// CandidateValidator is the sole admission authority. A Store must invoke it
// after locking durable state and before mutating the task, reservation, or
// snapshot. This prevents a validation failure from leaking capacity.
type CandidateValidator func(Candidate) error

// Store owns durable atomic mutations. ReserveAssignment must lock the exact
// task, binding, assignment, and catalog state; invoke validate; and only then
// atomically claim the task and persist its immutable snapshot. ReleaseAssignment
// ends exactly that task's active reference. Neither operation may rotate or
// remap a binding/home.
type Store interface {
	ReserveAssignment(ctx context.Context, request Request, validate CandidateValidator) (Candidate, error)
	ReleaseAssignment(ctx context.Context, request ReleaseRequest) error
}

// ReleaseRequest identifies one previously persisted reservation. Generations
// fence completion so it cannot decrement another task or replacement binding.
type ReleaseRequest struct {
	TaskID            string
	BindingID         string
	BindingGeneration uint64
	CatalogGeneration uint64
}

// Assignment is the bounded claim snapshot that may cross into shared
// admission code. It exposes only an opaque home reference and safe metadata.
type Assignment struct {
	HomeRef              HomeRef       `json:"home_ref"`
	BindingGeneration    uint64        `json:"binding_generation"`
	CatalogGeneration    uint64        `json:"catalog_generation"`
	Provider             string        `json:"provider"`
	Status               AccountStatus `json:"status"`
	TaskConcurrencyLimit int           `json:"task_concurrency_limit"`
}

// Resolver performs no lookup/retry cycle. Each Resolve call attempts exactly
// one fenced reservation so a conflict can never fall through to another home.
type Resolver struct {
	store Store
}

func NewResolver(store Store) *Resolver {
	return &Resolver{store: store}
}

// CanonicalProvider normalizes only established aliases. Unknown providers
// are returned unchanged so they cannot silently inherit another provider's
// approved home.
func CanonicalProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "agy":
		return "antigravity"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

// RequiresApprovedAssignment identifies native credential-bearing providers
// covered by the reviewed R3 contract. Discovery alone does not enroll another
// provider.
func RequiresApprovedAssignment(provider string) bool {
	switch CanonicalProvider(provider) {
	case "antigravity", "codex", "kiro":
		return true
	default:
		return false
	}
}

// Resolve atomically reserves and returns one approved, exclusive, usable
// assignment or fails closed. Concurrency comes from explicit policy and is
// never inferred from the number of catalog homes.
func (r *Resolver) Resolve(ctx context.Context, request Request) (Assignment, error) {
	if r == nil || r.store == nil || strings.TrimSpace(request.TaskID) == "" ||
		strings.TrimSpace(request.BindingID) == "" || strings.TrimSpace(request.AgentID) == "" ||
		strings.TrimSpace(request.WorkspaceID) == "" || !RequiresApprovedAssignment(request.Provider) {
		return Assignment{}, ErrNoApprovedAssignment
	}
	if request.ExpectedBindingGeneration == 0 || request.ExpectedCatalogGeneration == 0 {
		return Assignment{}, ErrGenerationConflict
	}

	provider := CanonicalProvider(request.Provider)
	request.Provider = provider
	candidate, err := r.store.ReserveAssignment(ctx, request, func(candidate Candidate) error {
		return validateCandidate(request, candidate)
	})
	if err != nil {
		return Assignment{}, boundedStoreError(err)
	}

	return Assignment{
		HomeRef:              candidate.HomeRef,
		BindingGeneration:    candidate.BindingGeneration,
		CatalogGeneration:    candidate.CatalogGeneration,
		Provider:             provider,
		Status:               candidate.Status,
		TaskConcurrencyLimit: candidate.TaskConcurrencyLimit,
	}, nil
}

// Release ends one active task reference. The durable store is responsible for
// idempotency when the same task-completion signal is delivered more than once.
func (r *Resolver) Release(ctx context.Context, request ReleaseRequest) error {
	if r == nil || r.store == nil || strings.TrimSpace(request.TaskID) == "" ||
		strings.TrimSpace(request.BindingID) == "" || request.BindingGeneration == 0 ||
		request.CatalogGeneration == 0 {
		return ErrGenerationConflict
	}
	if err := r.store.ReleaseAssignment(ctx, request); err != nil {
		return boundedStoreError(err)
	}
	return nil
}

func validateCandidate(request Request, candidate Candidate) error {
	// Identity and approval failures intentionally collapse to one error so a
	// caller cannot enumerate cross-workspace or revoked assignments.
	if candidate.TaskID != request.TaskID || candidate.BindingID != request.BindingID ||
		candidate.AgentID != request.AgentID || candidate.WorkspaceID != request.WorkspaceID ||
		candidate.Approval != ApprovalApproved {
		return ErrNoApprovedAssignment
	}
	if candidate.Provider != request.Provider || candidate.SessionProvider != request.Provider {
		return ErrProviderMismatch
	}
	if candidate.ExistingSnapshot {
		if !candidate.SnapshotMatches || (candidate.TaskStatus != "dispatched" &&
			candidate.TaskStatus != "running" && candidate.TaskStatus != "waiting_local_directory") {
			return ErrTaskConflict
		}
	} else if candidate.TaskStatus != "queued" {
		return ErrTaskConflict
	}
	if candidate.BindingGeneration != request.ExpectedBindingGeneration ||
		candidate.CatalogGeneration != request.ExpectedCatalogGeneration ||
		candidate.AssignmentCatalogGeneration != candidate.CatalogGeneration {
		return ErrGenerationConflict
	}
	if candidate.AssignmentOwners > 1 {
		return ErrExclusiveAssignmentConflict
	}
	if candidate.AssignmentOwners != 1 || candidate.BindingAssignments != 1 {
		return ErrNoApprovedAssignment
	}
	if candidate.BindingState != "active" || candidate.TransportBinding != "native_credential_home" ||
		candidate.AssignmentState != "active" || candidate.CatalogState != "available" ||
		candidate.CatalogEntryState != "healthy" || !candidate.HealthFresh ||
		(candidate.Status != StatusAvailable && candidate.Status != StatusLeased) {
		return ErrAccountUnavailable
	}
	if !ValidHomeRef(candidate.HomeRef) ||
		candidate.BindingGeneration == 0 || candidate.CatalogGeneration == 0 ||
		candidate.ActiveTasks < 0 || candidate.TaskConcurrencyLimit <= 0 ||
		strings.TrimSpace(candidate.RuntimeID) == "" || strings.TrimSpace(candidate.RuntimeSessionID) == "" ||
		strings.TrimSpace(candidate.StandardVersionID) == "" || strings.TrimSpace(candidate.ConfigurationVersionID) == "" ||
		strings.TrimSpace(candidate.ConfigurationDigest) == "" || strings.TrimSpace(candidate.CapabilityDigest) == "" {
		return ErrInvalidMetadata
	}
	if !candidate.ExistingSnapshot && candidate.ActiveTasks >= candidate.TaskConcurrencyLimit {
		return ErrCapacityExhausted
	}
	return nil
}

func boundedStoreError(err error) error {
	for _, bounded := range []error{
		ErrNoApprovedAssignment,
		ErrProviderMismatch,
		ErrExclusiveAssignmentConflict,
		ErrAccountUnavailable,
		ErrInvalidMetadata,
		ErrGenerationConflict,
		ErrTaskConflict,
		ErrCapacityExhausted,
	} {
		if errors.Is(err, bounded) {
			return bounded
		}
	}
	return ErrRegistryUnavailable
}

// ValidHomeRef accepts an opaque, non-path catalog identifier. It rejects
// whitespace, control characters, URI/path separators, drive delimiters, and
// path traversal tokens. Catalog issuance remains responsible for uniqueness
// and non-reuse.
func ValidHomeRef(ref HomeRef) bool {
	value := string(ref)
	if len(value) < 16 || len(value) > 128 || strings.TrimSpace(value) != value ||
		value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
		switch r {
		case '/', '\\', ':':
			return false
		}
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') && r != '-' && r != '_' && r != '.' && r != '~' {
			return false
		}
	}
	return true
}
