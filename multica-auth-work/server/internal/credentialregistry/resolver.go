// Package credentialregistry resolves metadata-only, owner-approved account
// assignments for daemon task claims. It never queries credential values.
package credentialregistry

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNoApprovedAssignment = errors.New("credential registry: no approved assignment")
	ErrProviderMismatch     = errors.New("credential registry: provider mismatch")
	ErrAccountUnavailable   = errors.New("credential registry: account unavailable")
	ErrInvalidMetadata      = errors.New("credential registry: invalid account metadata")
)

// QueryRower is the read-only database surface used by Resolver.
type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Assignment contains only routing metadata. It deliberately has no secret
// reference, credential format, token, cookie, or provider credential value.
type Assignment struct {
	AccountID     string
	TenantID      string
	Vendor        string
	HomeDir       string
	ConfigDir     string
	Status        string
	WorktypeScope *string
}

// Resolver reads the persisted agent -> account assignment and requires a
// matching owner approval in the same tenant.
type Resolver struct {
	db QueryRower
}

func NewResolver(db QueryRower) *Resolver {
	return &Resolver{db: db}
}

// CanonicalProvider normalizes only established aliases. Unknown providers
// are returned unchanged so they cannot silently inherit another vendor's
// approved account.
func CanonicalProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "agy":
		return "antigravity"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

// RequiresApprovedAssignment identifies credential-bearing providers covered
// by the ORQ-21 contract. Other runtimes keep their existing execution path.
func RequiresApprovedAssignment(provider string) bool {
	switch CanonicalProvider(provider) {
	case "antigravity", "codex", "kiro":
		return true
	default:
		return false
	}
}

// Resolve returns one approved assignment or a fail-closed error. The query
// intentionally does not join credentials: secret_ref and credential values
// are outside this package's data surface.
func (r *Resolver) Resolve(ctx context.Context, agentID, provider string) (Assignment, error) {
	if r == nil || r.db == nil {
		return Assignment{}, ErrNoApprovedAssignment
	}

	var assignment Assignment
	err := r.db.QueryRow(ctx, `
		SELECT a.account_id::text,
		       a.tenant_id::text,
		       a.vendor,
		       a.home_dir,
		       a.config_dir,
		       a.status,
		       aa.worktype_scope
		  FROM assignments AS ass
		  JOIN agent AS ag
		    ON ag.id = ass.agent_id
		  JOIN accounts AS a
		    ON a.account_id = ass.account_id
		  JOIN approved_accounts AS aa
		    ON aa.account_id = a.account_id
		   AND aa.tenant_id = a.tenant_id
		   AND aa.allowed = true
		 WHERE ass.agent_id = $1::uuid
		   AND ag.workspace_id = a.tenant_id
	`, agentID).Scan(
		&assignment.AccountID,
		&assignment.TenantID,
		&assignment.Vendor,
		&assignment.HomeDir,
		&assignment.ConfigDir,
		&assignment.Status,
		&assignment.WorktypeScope,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Assignment{}, ErrNoApprovedAssignment
		}
		return Assignment{}, fmt.Errorf("credential registry: resolve metadata: %w", err)
	}

	if CanonicalProvider(assignment.Vendor) != CanonicalProvider(provider) {
		return Assignment{}, ErrProviderMismatch
	}
	if assignment.Status != "available" && assignment.Status != "leased" {
		return Assignment{}, ErrAccountUnavailable
	}
	if !validAbsoluteMetadataPath(assignment.HomeDir) {
		return Assignment{}, ErrInvalidMetadata
	}
	if assignment.ConfigDir != "" && !validAbsoluteMetadataPath(assignment.ConfigDir) {
		return Assignment{}, ErrInvalidMetadata
	}
	if assignment.WorktypeScope == nil {
		return Assignment{}, ErrInvalidMetadata
	}
	return assignment, nil
}

func validAbsoluteMetadataPath(path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(path)
	return clean != string(filepath.Separator) && clean == path
}
