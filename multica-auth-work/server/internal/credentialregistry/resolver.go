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

// Candidate is the metadata-only snapshot returned after a successful atomic
// reservation. Internal admission predicates are excluded from JSON. The type
// intentionally has no account ID, source path, credential reference, or secret.
type Candidate struct {
	AgentID              string        `json:"-"`
	WorkspaceID          string        `json:"-"`
	Provider             string        `json:"-"`
	HomeRef              HomeRef       `json:"home_ref"`
	BindingGeneration    uint64        `json:"binding_generation"`
	CatalogGeneration    uint64        `json:"catalog_generation"`
	Approval             ApprovalState `json:"-"`
	Status               AccountStatus `json:"-"`
	WorktypeScope        string        `json:"-"`
	AssignmentOwners     int           `json:"-"`
	ActiveTasks          int           `json:"-"`
	TaskConcurrencyLimit int           `json:"-"`
}

// Request is the complete fenced identity supplied at admission. Both expected
// generations are mandatory: accepting an unfenced claim would permit a revoked
// or replaced assignment to be silently remapped.
type Request struct {
	AgentID                   string
	WorkspaceID               string
	Provider                  string
	ExpectedBindingGeneration uint64
	ExpectedCatalogGeneration uint64
}

// Store owns the durable atomic mutation. ReserveAssignment must, in one CAS or
// serializable transaction, verify the exact agent/workspace/provider binding,
// approval, exclusive home ownership, account usability, all Candidate metadata,
// both expected generations, and configured capacity before adding one active
// task reference. It must not rotate or remap a binding/home. A failed check
// must not reserve.
type Store interface {
	ReserveAssignment(ctx context.Context, request Request) (Candidate, error)
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
	if r == nil || r.store == nil || strings.TrimSpace(request.AgentID) == "" ||
		strings.TrimSpace(request.WorkspaceID) == "" || !RequiresApprovedAssignment(request.Provider) {
		return Assignment{}, ErrNoApprovedAssignment
	}
	if request.ExpectedBindingGeneration == 0 || request.ExpectedCatalogGeneration == 0 {
		return Assignment{}, ErrGenerationConflict
	}

	provider := CanonicalProvider(request.Provider)
	request.Provider = provider
	candidate, err := r.store.ReserveAssignment(ctx, request)
	if err != nil {
		return Assignment{}, boundedStoreError(err)
	}

	// Identity and approval failures intentionally collapse to one error so a
	// caller cannot enumerate cross-workspace or revoked assignments.
	if candidate.AgentID != request.AgentID || candidate.WorkspaceID != request.WorkspaceID ||
		candidate.Approval != ApprovalApproved {
		return Assignment{}, ErrNoApprovedAssignment
	}
	if candidate.Provider != provider {
		return Assignment{}, ErrProviderMismatch
	}
	if candidate.BindingGeneration != request.ExpectedBindingGeneration ||
		candidate.CatalogGeneration != request.ExpectedCatalogGeneration {
		return Assignment{}, ErrGenerationConflict
	}
	if candidate.AssignmentOwners > 1 {
		return Assignment{}, ErrExclusiveAssignmentConflict
	}
	if candidate.AssignmentOwners != 1 {
		return Assignment{}, ErrNoApprovedAssignment
	}
	if candidate.Status != StatusAvailable && candidate.Status != StatusLeased {
		return Assignment{}, ErrAccountUnavailable
	}
	if candidate.WorktypeScope != "GENERAL" || !ValidHomeRef(candidate.HomeRef) ||
		candidate.BindingGeneration == 0 || candidate.CatalogGeneration == 0 ||
		candidate.ActiveTasks <= 0 || candidate.TaskConcurrencyLimit <= 0 {
		return Assignment{}, ErrInvalidMetadata
	}
	if candidate.ActiveTasks > candidate.TaskConcurrencyLimit {
		return Assignment{}, ErrCapacityExhausted
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

func boundedStoreError(err error) error {
	for _, bounded := range []error{
		ErrNoApprovedAssignment,
		ErrProviderMismatch,
		ErrExclusiveAssignmentConflict,
		ErrAccountUnavailable,
		ErrInvalidMetadata,
		ErrGenerationConflict,
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
