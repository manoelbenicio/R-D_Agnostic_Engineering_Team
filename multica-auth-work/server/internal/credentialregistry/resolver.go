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
	ErrNoApprovedAssignment = errors.New("credential registry: no approved assignment")
	ErrProviderMismatch     = errors.New("credential registry: provider mismatch")
	ErrAccountAlreadyUsed   = errors.New("credential registry: home assigned to multiple agents")
	ErrAccountUnavailable   = errors.New("credential registry: account unavailable")
	ErrInvalidMetadata      = errors.New("credential registry: invalid assignment metadata")
	ErrCapacityExhausted    = errors.New("credential registry: task concurrency exhausted")
	ErrRegistryUnavailable  = errors.New("credential registry: assignment store unavailable")
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

// Candidate is the metadata-only snapshot returned by an assignment store.
// Internal admission predicates are excluded from JSON. The type intentionally
// has no account ID, source path, config path, credential reference, or secret.
type Candidate struct {
	AgentID              string        `json:"-"`
	WorkspaceID          string        `json:"-"`
	Provider             string        `json:"-"`
	HomeRef              HomeRef       `json:"home_ref"`
	CatalogGeneration    uint64        `json:"catalog_generation"`
	Approval             ApprovalState `json:"-"`
	Status               AccountStatus `json:"-"`
	WorktypeScope        string        `json:"-"`
	AssignmentOwners     int           `json:"-"`
	ActiveTasks          int           `json:"-"`
	TaskConcurrencyLimit int           `json:"-"`
}

// Store returns one atomic assignment snapshot for an existing logical agent
// and workspace. Implementations must perform no task-time rotation and must
// return ErrNoApprovedAssignment when no unambiguous snapshot exists.
type Store interface {
	LookupAssignment(ctx context.Context, agentID, workspaceID string) (Candidate, error)
}

// Request is the complete identity supplied to the resolver at admission.
type Request struct {
	AgentID     string
	WorkspaceID string
	Provider    string
}

// Assignment is the bounded result that may cross into shared admission code.
// It exposes only an opaque home reference and safe catalog/admission metadata.
type Assignment struct {
	HomeRef              HomeRef       `json:"home_ref"`
	CatalogGeneration    uint64        `json:"catalog_generation"`
	Provider             string        `json:"provider"`
	Status               AccountStatus `json:"status"`
	TaskConcurrencyLimit int           `json:"task_concurrency_limit"`
}

// Resolver validates a store snapshot without caching or mutating it. This
// keeps concurrent admission deterministic and leaves assignment CAS and
// catalog publication with their durable owners.
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

// Resolve returns one approved, exclusive, usable assignment or fails closed.
// Concurrency is validated from explicit policy; it is never inferred from the
// number of catalog homes.
func (r *Resolver) Resolve(ctx context.Context, request Request) (Assignment, error) {
	if r == nil || r.store == nil || strings.TrimSpace(request.AgentID) == "" ||
		strings.TrimSpace(request.WorkspaceID) == "" || !RequiresApprovedAssignment(request.Provider) {
		return Assignment{}, ErrNoApprovedAssignment
	}

	candidate, err := r.store.LookupAssignment(ctx, request.AgentID, request.WorkspaceID)
	if err != nil {
		if errors.Is(err, ErrNoApprovedAssignment) {
			return Assignment{}, ErrNoApprovedAssignment
		}
		return Assignment{}, ErrRegistryUnavailable
	}

	// Identity and approval checks intentionally collapse to one error so a
	// caller cannot use this resolver to enumerate cross-workspace or revoked
	// assignments.
	if candidate.AgentID != request.AgentID || candidate.WorkspaceID != request.WorkspaceID ||
		candidate.Approval != ApprovalApproved {
		return Assignment{}, ErrNoApprovedAssignment
	}

	provider := CanonicalProvider(request.Provider)
	// Providers are canonicalized at write time. A noncanonical stored alias is
	// drift and must be rejected rather than repaired independently at read time.
	if candidate.Provider != provider {
		return Assignment{}, ErrProviderMismatch
	}
	if candidate.AssignmentOwners > 1 {
		return Assignment{}, ErrAccountAlreadyUsed
	}
	if candidate.AssignmentOwners != 1 {
		return Assignment{}, ErrNoApprovedAssignment
	}
	if candidate.Status != StatusAvailable && candidate.Status != StatusLeased {
		return Assignment{}, ErrAccountUnavailable
	}
	if candidate.WorktypeScope != "GENERAL" || !ValidHomeRef(candidate.HomeRef) ||
		candidate.CatalogGeneration == 0 || candidate.ActiveTasks < 0 ||
		candidate.TaskConcurrencyLimit <= 0 {
		return Assignment{}, ErrInvalidMetadata
	}
	if candidate.ActiveTasks >= candidate.TaskConcurrencyLimit {
		return Assignment{}, ErrCapacityExhausted
	}

	return Assignment{
		HomeRef:              candidate.HomeRef,
		CatalogGeneration:    candidate.CatalogGeneration,
		Provider:             provider,
		Status:               candidate.Status,
		TaskConcurrencyLimit: candidate.TaskConcurrencyLimit,
	}, nil
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
