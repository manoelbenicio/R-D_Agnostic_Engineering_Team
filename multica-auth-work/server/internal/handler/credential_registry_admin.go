package handler

import (
	"context"
	"errors"
	"fmt"
)

const (
	credentialRegistryOpaqueReferenceMaxLength = 128
	credentialRegistryDefaultPageLimit         = 50
	credentialRegistryMaxPageLimit             = 100
	credentialRegistryMaxCounterCardinality    = 64
	credentialRegistryRedactedReference        = "[redacted]"
	credentialRegistryUnavailableRequestID     = "unavailable"
)

var (
	// ErrCredentialRegistryUnauthenticated is deliberately constant so malformed
	// authentication material can never be reflected by a handler.
	ErrCredentialRegistryUnauthenticated = errors.New("credential registry authentication failed")
	// ErrCredentialRegistryForbidden is deliberately constant so authorization
	// and tenant-guard internals can never disclose identities or paths.
	ErrCredentialRegistryForbidden = errors.New("credential registry access denied")
)

// CredentialRegistryActor identifies an already-authenticated caller without
// coupling the registry boundary to an authentication middleware or database
// representation. ID is always an opaque identifier and must never contain a
// path, account identity, or credential material.
type CredentialRegistryActor struct {
	Type string
	ID   string
}

// CredentialRegistryActorGuard owns the global and human-actor checks for
// registry administration. Implementations must deny task actors and other
// machine credentials on both methods.
type CredentialRegistryActorGuard interface {
	RequireOwner(ctx context.Context, actor CredentialRegistryActor) error
	RequireOwnerOrWorkspaceAdmin(ctx context.Context, actor CredentialRegistryActor, workspaceID string) error
}

// CredentialRegistryWorkspaceGuard owns tenant confinement. Implementations
// must fail closed for missing, stale, or cross-workspace bindings.
type CredentialRegistryWorkspaceGuard interface {
	RequireWorkspace(ctx context.Context, actor CredentialRegistryActor, workspaceID string) error
	RequireDaemonWorkspace(ctx context.Context, actor CredentialRegistryActor, daemonID, workspaceID string) error
}

// CredentialRegistryBoundary applies the authentication and tenant checks that
// every pathless registry handler must complete before invoking registry core.
// It intentionally has no router, database, or registry-core dependency.
type CredentialRegistryBoundary struct {
	actorGuard     CredentialRegistryActorGuard
	workspaceGuard CredentialRegistryWorkspaceGuard
}

func NewCredentialRegistryBoundary(actorGuard CredentialRegistryActorGuard, workspaceGuard CredentialRegistryWorkspaceGuard) (*CredentialRegistryBoundary, error) {
	if actorGuard == nil || workspaceGuard == nil {
		return nil, errors.New("credential registry guards are required")
	}
	return &CredentialRegistryBoundary{actorGuard: actorGuard, workspaceGuard: workspaceGuard}, nil
}

// AuthorizeWorkspace confines a human owner/admin operation to one validated
// workspace. Task and machine actors are denied before any tenant lookup. Guard
// errors are collapsed to a constant sentinel to prevent disclosure.
func (b *CredentialRegistryBoundary) AuthorizeWorkspace(ctx context.Context, actor CredentialRegistryActor, workspaceID string) error {
	if ctx == nil || actor.Validate() != nil {
		return ErrCredentialRegistryUnauthenticated
	}
	if actor.Type != "human" {
		return ErrCredentialRegistryForbidden
	}
	if validateCredentialRegistryOpaqueReference(workspaceID) != nil {
		return ErrCredentialRegistryForbidden
	}
	if b == nil || b.actorGuard == nil || b.workspaceGuard == nil {
		return ErrCredentialRegistryForbidden
	}
	if err := b.actorGuard.RequireOwnerOrWorkspaceAdmin(ctx, actor, workspaceID); err != nil {
		return ErrCredentialRegistryForbidden
	}
	if err := b.workspaceGuard.RequireWorkspace(ctx, actor, workspaceID); err != nil {
		return ErrCredentialRegistryForbidden
	}
	return nil
}

// AuthorizeDaemonReport confines a daemon report to the authenticated daemon
// identity and its explicitly authorized workspace. A caller cannot ask the
// workspace guard to authorize a different daemon ID.
func (b *CredentialRegistryBoundary) AuthorizeDaemonReport(ctx context.Context, actor CredentialRegistryActor, daemonID, workspaceID string) error {
	if ctx == nil || actor.Validate() != nil {
		return ErrCredentialRegistryUnauthenticated
	}
	if actor.Type != "daemon" || actor.ID != daemonID {
		return ErrCredentialRegistryForbidden
	}
	if validateCredentialRegistryOpaqueReference(daemonID) != nil || validateCredentialRegistryOpaqueReference(workspaceID) != nil {
		return ErrCredentialRegistryForbidden
	}
	if b == nil || b.workspaceGuard == nil {
		return ErrCredentialRegistryForbidden
	}
	if err := b.workspaceGuard.RequireDaemonWorkspace(ctx, actor, daemonID, workspaceID); err != nil {
		return ErrCredentialRegistryForbidden
	}
	return nil
}

// CredentialHomeListRequest is the query DTO for the pathless catalog
// projection. Cursor is opaque; it is never a filesystem cursor or path.
type CredentialHomeListRequest struct {
	Cursor string
	Limit  int
}

// EffectiveLimit returns the bounded default used when limit is omitted.
func (r CredentialHomeListRequest) EffectiveLimit() int {
	if r.Limit == 0 {
		return credentialRegistryDefaultPageLimit
	}
	return r.Limit
}

// CredentialHomeSummary is the complete public catalog projection. Its shape
// deliberately excludes raw paths, filesystem identities, account identities,
// credential names/data, environment values, and provider response bodies.
type CredentialHomeSummary struct {
	HomeRef           string `json:"home_ref"`
	CatalogGeneration int64  `json:"catalog_generation"`
	Provider          string `json:"provider"`
	State             string `json:"state"`
	Health            string `json:"health"`
}

// CredentialHomeListResponse is the canonical cursor-paginated response.
type CredentialHomeListResponse struct {
	Items      []CredentialHomeSummary `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

// CredentialHomeAssignmentRequest is the exact body frozen for assigning one
// opaque home to an existing binding. Binding identity remains in the route,
// not in this body.
type CredentialHomeAssignmentRequest struct {
	HomeRef                   string `json:"home_ref"`
	ExpectedBindingGeneration int64  `json:"expected_binding_generation"`
	ExpectedCatalogGeneration int64  `json:"expected_catalog_generation"`
}

// CredentialHomeReleaseRequest is the exact body frozen for draining or
// releasing an existing assignment.
type CredentialHomeReleaseRequest struct {
	ExpectedBindingGeneration int64 `json:"expected_binding_generation"`
	Drain                     bool  `json:"drain"`
}

// CredentialHomeReconcileResponse is returned when a full reconciliation has
// been accepted. OperationID is opaque and carries no daemon-local location.
type CredentialHomeReconcileResponse struct {
	OperationID string `json:"operation_id"`
}

// CredentialHomeReconciliationResponse exposes only generation and aggregate
// counters. Counter names are validated as non-path tokens before exposure.
type CredentialHomeReconciliationResponse struct {
	OperationID string           `json:"operation_id"`
	Generation  int64            `json:"generation"`
	Counters    map[string]int64 `json:"counters"`
}

// CredentialCatalogReconciliationReportRequest is the exact daemon report
// body frozen by the canonical OpenSpec. It contains no paths or discovered
// child identifiers.
type CredentialCatalogReconciliationReportRequest struct {
	DaemonID           string           `json:"daemon_id"`
	PreviousGeneration int64            `json:"previous_generation"`
	Generation         int64            `json:"generation"`
	ScanKind           string           `json:"scan_kind"`
	Counters           map[string]int64 `json:"counters"`
	Digest             string           `json:"digest"`
}

// CredentialRegistryErrorCode is restricted to the canonical registry error
// vocabulary. HTTP status is derived by NewCredentialRegistryError rather than
// supplied by callers, preventing status/code drift.
type CredentialRegistryErrorCode string

const (
	CredentialRegistryInvalidArgument             CredentialRegistryErrorCode = "invalid_argument"
	CredentialRegistryUnauthenticated             CredentialRegistryErrorCode = "unauthenticated"
	CredentialRegistryForbidden                   CredentialRegistryErrorCode = "forbidden"
	CredentialRegistryNotFound                    CredentialRegistryErrorCode = "not_found"
	CredentialRegistryVersionConflict             CredentialRegistryErrorCode = "version_conflict"
	CredentialRegistryGenerationConflict          CredentialRegistryErrorCode = "generation_conflict"
	CredentialRegistryExclusiveAssignmentConflict CredentialRegistryErrorCode = "exclusive_assignment_conflict"
	CredentialRegistryActiveReference             CredentialRegistryErrorCode = "active_reference"
	CredentialRegistryCapabilityUnsupported       CredentialRegistryErrorCode = "capability_unsupported"
	CredentialRegistryHealthStale                 CredentialRegistryErrorCode = "health_stale"
	CredentialRegistryDriftDetected               CredentialRegistryErrorCode = "drift_detected"
	CredentialRegistryInvalidConfiguration        CredentialRegistryErrorCode = "invalid_configuration"
	CredentialRegistryCapacityExhausted           CredentialRegistryErrorCode = "capacity_exhausted"
	CredentialRegistryCatalogUnavailable          CredentialRegistryErrorCode = "catalog_unavailable"
	CredentialRegistryDaemonNotReady              CredentialRegistryErrorCode = "daemon_not_ready"
)

// CredentialRegistryErrorEnvelope is the frozen error wire shape.
type CredentialRegistryErrorEnvelope struct {
	Error CredentialRegistryError `json:"error"`
}

// CredentialRegistryError contains only a fixed safe message and validated
// metadata. Callers cannot pass provider or filesystem error text through it.
type CredentialRegistryError struct {
	Code      CredentialRegistryErrorCode `json:"code"`
	Message   string                      `json:"message"`
	Field     string                      `json:"field,omitempty"`
	RequestID string                      `json:"request_id"`
	Retryable bool                        `json:"retryable"`
}

type credentialRegistryErrorSpec struct {
	status  int
	message string
}

var credentialRegistryErrorSpecs = map[CredentialRegistryErrorCode]credentialRegistryErrorSpec{
	CredentialRegistryInvalidArgument:             {status: 400, message: "request contains an invalid argument"},
	CredentialRegistryUnauthenticated:             {status: 401, message: "authentication is required"},
	CredentialRegistryForbidden:                   {status: 403, message: "access is forbidden"},
	CredentialRegistryNotFound:                    {status: 404, message: "resource was not found"},
	CredentialRegistryVersionConflict:             {status: 409, message: "the expected version does not match"},
	CredentialRegistryGenerationConflict:          {status: 409, message: "the expected generation does not match"},
	CredentialRegistryExclusiveAssignmentConflict: {status: 409, message: "the home is already assigned"},
	CredentialRegistryActiveReference:             {status: 409, message: "the resource has an active reference"},
	CredentialRegistryCapabilityUnsupported:       {status: 412, message: "the requested capability is unsupported"},
	CredentialRegistryHealthStale:                 {status: 412, message: "health information is stale"},
	CredentialRegistryDriftDetected:               {status: 412, message: "runtime configuration drift was detected"},
	CredentialRegistryInvalidConfiguration:        {status: 422, message: "runtime configuration is invalid"},
	CredentialRegistryCapacityExhausted:           {status: 429, message: "registry capacity is exhausted"},
	CredentialRegistryCatalogUnavailable:          {status: 503, message: "credential catalog is unavailable"},
	CredentialRegistryDaemonNotReady:              {status: 503, message: "daemon is not ready"},
}

var credentialRegistryErrorFields = map[string]struct{}{
	"agent_id":                    {},
	"configuration":               {},
	"counters":                    {},
	"cursor":                      {},
	"daemon_id":                   {},
	"digest":                      {},
	"drain":                       {},
	"expected_active_version_id":  {},
	"expected_binding_generation": {},
	"expected_catalog_generation": {},
	"generation":                  {},
	"home_ref":                    {},
	"limit":                       {},
	"previous_generation":         {},
	"reason":                      {},
	"runtime_id":                  {},
	"scan_kind":                   {},
	"standard_id":                 {},
	"target_version_id":           {},
}

// NewCredentialRegistryError constructs a canonical error and derives its HTTP
// status. Unknown codes are rejected instead of being silently remapped.
// Message text is fixed by code, and an unrecognized field is omitted so raw
// input can never be reflected into the response.
func NewCredentialRegistryError(code CredentialRegistryErrorCode, field, requestID string, retryable bool) (int, CredentialRegistryErrorEnvelope, bool) {
	spec, ok := credentialRegistryErrorSpecs[code]
	if !ok {
		return 0, CredentialRegistryErrorEnvelope{}, false
	}
	if _, allowed := credentialRegistryErrorFields[field]; !allowed {
		field = ""
	}
	if validateCredentialRegistryOpaqueReference(requestID) != nil {
		requestID = credentialRegistryUnavailableRequestID
	}
	return spec.status, CredentialRegistryErrorEnvelope{Error: CredentialRegistryError{
		Code:      code,
		Message:   spec.message,
		Field:     field,
		RequestID: requestID,
		Retryable: retryable,
	}}, true
}

// ValidateCredentialRegistryOpaqueReference accepts only bounded ASCII tokens.
// Path separators, dot segments, whitespace, control bytes, URI punctuation,
// percent encoding, and Unicode are all rejected. The returned error never
// includes the rejected value.
func ValidateCredentialRegistryOpaqueReference(ref string) error {
	return validateCredentialRegistryOpaqueReference(ref)
}

func validateCredentialRegistryOpaqueReference(ref string) error {
	if ref == "" {
		return errors.New("opaque reference is required")
	}
	if len(ref) > credentialRegistryOpaqueReferenceMaxLength {
		return fmt.Errorf("opaque reference exceeds %d bytes", credentialRegistryOpaqueReferenceMaxLength)
	}
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return errors.New("opaque reference contains a forbidden character")
	}
	return nil
}

// RedactCredentialRegistryOpaqueReference returns a constant marker and never
// preserves prefixes, suffixes, lengths, or other identifying material.
func RedactCredentialRegistryOpaqueReference(_ string) string {
	return credentialRegistryRedactedReference
}

// Redacted returns an audit-safe copy of an assignment request.
func (r CredentialHomeAssignmentRequest) Redacted() CredentialHomeAssignmentRequest {
	r.HomeRef = RedactCredentialRegistryOpaqueReference(r.HomeRef)
	return r
}

func (a CredentialRegistryActor) Validate() error {
	if !isCredentialRegistryToken(a.Type) {
		return errors.New("actor type must be a non-path token")
	}
	if err := validateCredentialRegistryOpaqueReference(a.ID); err != nil {
		return errors.New("actor ID must be an opaque non-path reference")
	}
	return nil
}

func (r CredentialHomeListRequest) Validate() error {
	if r.Cursor != "" {
		if err := validateCredentialRegistryOpaqueReference(r.Cursor); err != nil {
			return errors.New("cursor must be an opaque non-path reference")
		}
	}
	if r.Limit < 0 || r.Limit > credentialRegistryMaxPageLimit {
		return fmt.Errorf("limit must be between 0 and %d", credentialRegistryMaxPageLimit)
	}
	return nil
}

func (r CredentialHomeSummary) Validate() error {
	if err := validateCredentialRegistryOpaqueReference(r.HomeRef); err != nil {
		return errors.New("home_ref must be an opaque non-path reference")
	}
	if r.CatalogGeneration <= 0 {
		return errors.New("catalog_generation must be greater than zero")
	}
	if !isCredentialRegistryToken(r.Provider) {
		return errors.New("provider must be a non-path token")
	}
	if !isCredentialRegistryToken(r.State) {
		return errors.New("state must be a non-path token")
	}
	if !isCredentialRegistryToken(r.Health) {
		return errors.New("health must be a non-path token")
	}
	return nil
}

func (r CredentialHomeListResponse) Validate() error {
	if len(r.Items) > credentialRegistryMaxPageLimit {
		return fmt.Errorf("items must contain at most %d entries", credentialRegistryMaxPageLimit)
	}
	for i := range r.Items {
		if err := r.Items[i].Validate(); err != nil {
			return fmt.Errorf("items[%d]: %w", i, err)
		}
	}
	if r.NextCursor != "" {
		if err := validateCredentialRegistryOpaqueReference(r.NextCursor); err != nil {
			return errors.New("next_cursor must be an opaque non-path reference")
		}
	}
	return nil
}

func (r CredentialHomeAssignmentRequest) Validate() error {
	if err := validateCredentialRegistryOpaqueReference(r.HomeRef); err != nil {
		return errors.New("home_ref must be an opaque non-path reference")
	}
	if r.ExpectedBindingGeneration <= 0 {
		return errors.New("expected_binding_generation must be greater than zero")
	}
	if r.ExpectedCatalogGeneration <= 0 {
		return errors.New("expected_catalog_generation must be greater than zero")
	}
	return nil
}

func (r CredentialHomeReleaseRequest) Validate() error {
	if r.ExpectedBindingGeneration <= 0 {
		return errors.New("expected_binding_generation must be greater than zero")
	}
	return nil
}

func (r CredentialHomeReconcileResponse) Validate() error {
	if err := validateCredentialRegistryOpaqueReference(r.OperationID); err != nil {
		return errors.New("operation_id must be an opaque non-path reference")
	}
	return nil
}

func (r CredentialHomeReconciliationResponse) Validate() error {
	if err := validateCredentialRegistryOpaqueReference(r.OperationID); err != nil {
		return errors.New("operation_id must be an opaque non-path reference")
	}
	if r.Generation <= 0 {
		return errors.New("generation must be greater than zero")
	}
	return validateCredentialRegistryCounters(r.Counters)
}

func (r CredentialCatalogReconciliationReportRequest) Validate() error {
	if err := validateCredentialRegistryOpaqueReference(r.DaemonID); err != nil {
		return errors.New("daemon_id must be an opaque non-path reference")
	}
	if r.PreviousGeneration <= 0 {
		return errors.New("previous_generation must be greater than zero")
	}
	if r.Generation <= r.PreviousGeneration {
		return errors.New("generation must be greater than previous_generation")
	}
	if !isCredentialRegistryToken(r.ScanKind) {
		return errors.New("scan_kind must be a non-path token")
	}
	if err := validateCredentialRegistryCounters(r.Counters); err != nil {
		return err
	}
	if !isCredentialRegistrySHA256(r.Digest) {
		return errors.New("digest must be a raw lowercase 64-hex SHA-256 value")
	}
	return nil
}

func validateCredentialRegistryCounters(counters map[string]int64) error {
	if counters == nil {
		return errors.New("counters are required")
	}
	if len(counters) == 0 {
		return errors.New("counters must not be empty")
	}
	if len(counters) > credentialRegistryMaxCounterCardinality {
		return fmt.Errorf("counters must contain at most %d entries", credentialRegistryMaxCounterCardinality)
	}
	for name, value := range counters {
		if !isCredentialRegistryToken(name) {
			return errors.New("counter name must be a non-path token")
		}
		if value < 0 {
			return errors.New("counter value must not be negative")
		}
	}
	return nil
}

func isCredentialRegistryToken(value string) bool {
	return validateCredentialRegistryOpaqueReference(value) == nil
}

func isCredentialRegistrySHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}
