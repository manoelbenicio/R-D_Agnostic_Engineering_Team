package handler

// SPE-6 C3: runtime configuration API.
//
// This file owns the configuration half of the Runtime Manager API: the wire
// DTOs, the translation between those DTOs and the typed runtimeconfig
// contract, and the binding-scoped endpoints. runtime_standard.go owns the
// standard-scoped endpoints and reuses everything below.
//
// Design constraints this file is built around:
//
//   - The typed contract in internal/service/runtimeconfig is the only
//     authority on precedence, delegability, capability admission, digests and
//     apply class. Nothing here re-implements a rule; it decodes, delegates,
//     and encodes.
//   - Requests are decoded through explicit DTOs with unknown fields rejected.
//     A configuration document is security-relevant input, so an unrecognised
//     key is a failure, not something to ignore. Ignoring it would let a
//     client believe it set a constraint that was silently dropped.
//   - Responses are re-encoded from the typed contract, never echoed from
//     stored bytes. The typed contract has no credential, token, endpoint,
//     path or prompt field, so re-encoding is what makes a response
//     structurally incapable of carrying one.
//   - Persistence and capability discovery are interfaces. C1 and C2 are
//     blocked, so this package must compile, be tested and be reviewable
//     without them; they plug in later without touching this file.
//   - Everything fails closed. An unreadable layer, an unavailable capability
//     declaration, a stale generation or an invalid document produces an error
//     response, never a partially applied configuration.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
)

// maxRuntimeConfigurationBody bounds a configuration request body. The typed
// contract is a fixed, small field set, so a large body is either a mistake or
// an attempt to exhaust the decoder.
const maxRuntimeConfigurationBody = 64 << 10

// Pagination bounds for every Runtime Manager list endpoint.
const (
	defaultRuntimeManagerPageSize = 50
	maxRuntimeManagerPageSize     = 200
)

// Store-independent sentinel errors. A store implementation maps its own
// failures onto these so the HTTP status mapping lives here, in one place,
// rather than being reinvented per backend.
var (
	// ErrRuntimeManagerNotFound reports a missing standard, binding or version.
	ErrRuntimeManagerNotFound = errors.New("runtime manager: resource not found")

	// ErrRuntimeManagerConflict reports a failed compare-and-swap: the caller's
	// expected active version or expected generation no longer matches.
	ErrRuntimeManagerConflict = errors.New("runtime manager: generation conflict")

	// ErrRuntimeManagerCapabilitiesUnavailable reports that the capability
	// declaration could not be read. It is deliberately distinct from an
	// invalid declaration: unavailable is retryable, invalid is not.
	ErrRuntimeManagerCapabilitiesUnavailable = errors.New("runtime manager: capabilities unavailable")

	// ErrRuntimeManagerIntegrity reports an immutable row whose canonical
	// document cannot be reproduced or whose digest does not match its bytes.
	ErrRuntimeManagerIntegrity = errors.New("runtime manager: stored document integrity failure")
)

// ---------------------------------------------------------------------------
// Wire DTOs
// ---------------------------------------------------------------------------

// runtimeConfigurationLimits mirrors the accepted client contract's limits
// object. Pointers distinguish an absent limit from an explicit value, so an
// omitted limit inherits and an explicit zero is rejected as invalid rather
// than silently read as "no limit".
type runtimeConfigurationLimits struct {
	MaxContextTokens *int64 `json:"max_context_tokens,omitempty"`
	MaxInputTokens   *int64 `json:"max_input_tokens,omitempty"`
	MaxOutputTokens  *int64 `json:"max_output_tokens,omitempty"`
	MaxTotalTokens   *int64 `json:"max_total_tokens,omitempty"`
	MaxToolCalls     *int64 `json:"max_tool_calls,omitempty"`
	WallTimeoutMS    *int64 `json:"wall_timeout_ms,omitempty"`
	IdleTimeoutMS    *int64 `json:"idle_timeout_ms,omitempty"`
}

// runtimeConfigurationValues is the frozen pathless wire contract. Composite
// groups use the runtimeconfig types directly so handler and resolver schemas
// cannot drift.
type runtimeConfigurationValues struct {
	TransportBinding       *string                          `json:"transport_binding,omitempty"`
	CLIKind                *string                          `json:"cli_kind,omitempty"`
	Provider               *string                          `json:"provider,omitempty"`
	SubscriptionRef        *string                          `json:"subscription_ref,omitempty"`
	ProviderCatalogVersion *string                          `json:"provider_catalog_version,omitempty"`
	CapabilityDigest       *string                          `json:"capability_digest,omitempty"`
	Model                  *string                          `json:"model,omitempty"`
	ReasoningMode          *string                          `json:"reasoning_mode,omitempty"`
	ReasoningEffort        *string                          `json:"reasoning_effort,omitempty"`
	ReasoningBudget        *int64                           `json:"reasoning_budget,omitempty"`
	Limits                 *runtimeConfigurationLimits      `json:"limits,omitempty"`
	Concurrency            *runtimeconfig.ConcurrencyPolicy `json:"concurrency,omitempty"`
	Retry                  *runtimeconfig.RetryPolicy       `json:"retry,omitempty"`
	Flags                  *runtimeconfig.FlagPolicy        `json:"flags,omitempty"`
	Environment            *runtimeconfig.EnvironmentPolicy `json:"env,omitempty"`
	Skills                 *runtimeconfig.SkillPolicy       `json:"skills,omitempty"`
	MCPTools               *runtimeconfig.MCPToolPolicy     `json:"mcp_tools,omitempty"`
	Permissions            *runtimeconfig.PermissionPolicy  `json:"permissions,omitempty"`
	Eligibility            *runtimeconfig.EligibilityPolicy `json:"eligibility,omitempty"`
	Health                 *runtimeconfig.HealthPolicy      `json:"health,omitempty"`
	Fallback               *runtimeconfig.FallbackPolicy    `json:"fallback,omitempty"`
}

// runtimeConfigurationDocument mirrors RuntimeConfigurationDocument: the
// secret-free, pathless document the frozen API accepts and returns.
type runtimeConfigurationDocument struct {
	Version      string                     `json:"version"`
	Values       runtimeConfigurationValues `json:"values"`
	Delegability map[string]bool            `json:"delegability,omitempty"`
}

// runtimeConfigurationIssue mirrors RuntimeConfigurationIssue. It carries a
// stable code and the field it concerns, never the rejected value, so an issue
// is safe to log and to return across a trust boundary.
type runtimeConfigurationIssue struct {
	Code      string `json:"code"`
	Field     string `json:"field,omitempty"`
	Retryable bool   `json:"retryable"`
}

// runtimeConfigurationValidation mirrors RuntimeConfigurationValidation.
type runtimeConfigurationValidation struct {
	Valid            bool                        `json:"valid"`
	CapabilityDigest string                      `json:"capability_digest"`
	ApplyClass       string                      `json:"apply_class,omitempty"`
	Issues           []runtimeConfigurationIssue `json:"issues"`
}

// runtimeConfigurationVersionResponse mirrors RuntimeConfigurationVersion.
type runtimeConfigurationVersionResponse struct {
	ID                  string                          `json:"id"`
	VersionNumber       int64                           `json:"version_number"`
	Configuration       runtimeConfigurationDocument    `json:"configuration"`
	ConfigurationDigest string                          `json:"configuration_digest"`
	Reason              string                          `json:"reason"`
	State               string                          `json:"state,omitempty"`
	ApplyClass          string                          `json:"apply_class,omitempty"`
	Validation          *runtimeConfigurationValidation `json:"validation,omitempty"`
	CreatedAt           time.Time                       `json:"created_at"`
}

// runtimeActivationResult mirrors RuntimeActivationResult.
type runtimeActivationResult struct {
	ActiveVersionID        string  `json:"active_version_id"`
	PreviousVersionID      *string `json:"previous_version_id"`
	ApplyClass             string  `json:"apply_class"`
	PendingAcknowledgement bool    `json:"pending_acknowledgement"`
	CapabilityDigest       string  `json:"capability_digest"`
}

// runtimeManagerPage mirrors RuntimeManagerPage.
type runtimeManagerPage[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// createConfigurationVersionRequest is the body of both version-create
// endpoints.
type createConfigurationVersionRequest struct {
	Configuration runtimeConfigurationDocument `json:"configuration"`
	Reason        string                       `json:"reason"`
}

// runtimeNullableString distinguishes an omitted field from explicit JSON null.
// The frozen CAS body requires the field even for first activation, where null
// is the explicit expectation that no active version exists.
type runtimeNullableString struct {
	Set   bool
	Value *string
}

func (v *runtimeNullableString) UnmarshalJSON(raw []byte) error {
	v.Set = true
	if string(raw) == "null" {
		v.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	v.Value = &value
	return nil
}

func (v runtimeNullableString) MarshalJSON() ([]byte, error) {
	if v.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*v.Value)
}

func expectedActive(value *string) runtimeNullableString {
	return runtimeNullableString{Set: true, Value: value}
}

// activationRequest is the shared activation body.
type activationRequest struct {
	ExpectedActiveVersionID runtimeNullableString `json:"expected_active_version_id"`
	Reason                  string                `json:"reason"`
}

// bindingActivationRequest adds the binding generation a caller believes it is
// acting on, so activation is a compare-and-swap rather than a blind write.
type bindingActivationRequest struct {
	ExpectedActiveVersionID   runtimeNullableString `json:"expected_active_version_id"`
	Reason                    string                `json:"reason"`
	ExpectedBindingGeneration int64                 `json:"expected_binding_generation"`
}

// rollbackRequest targets any immutable prior version.
type rollbackRequest struct {
	ExpectedActiveVersionID runtimeNullableString `json:"expected_active_version_id"`
	Reason                  string                `json:"reason"`
	TargetVersionID         string                `json:"target_version_id"`
}

// bindingRollbackRequest is a binding rollback with generation CAS.
type bindingRollbackRequest struct {
	ExpectedActiveVersionID   runtimeNullableString `json:"expected_active_version_id"`
	Reason                    string                `json:"reason"`
	TargetVersionID           string                `json:"target_version_id"`
	ExpectedBindingGeneration int64                 `json:"expected_binding_generation"`
}

// ---------------------------------------------------------------------------
// Plug-in interfaces
// ---------------------------------------------------------------------------

// RuntimeConfigurationVersionRecord is one immutable stored configuration
// version. Document holds the exact bytes that were accepted, so a rollback
// re-resolves the original document rather than a reconstruction of it.
type RuntimeConfigurationVersionRecord struct {
	ID            string
	VersionNumber int64
	Document      []byte
	Digest        string
	ApplyClass    string
	Reason        string
	State         string
	CreatedAt     time.Time
}

// RuntimeBindingRecord is the subset of a binding this API needs. It carries
// no path, no credential and no account: HomeRef is an opaque reference.
type RuntimeBindingRecord struct {
	ID                           string
	WorkspaceID                  string
	Provider                     string
	TransportBinding             string
	BindingGeneration            int64
	ActiveConfigurationVersionID *string
	State                        string
}

// CreateBindingVersionParams is an append-only version insert.
type CreateBindingVersionParams struct {
	WorkspaceID      string
	BindingID        string
	Document         []byte
	Digest           string
	ApplyClass       string
	Reason           string
	RequestID        string
	CapabilityDigest string
}

// ActivateBindingVersionParams is a compare-and-swap activation. Both expected
// values must still hold at commit time or the store returns
// ErrRuntimeManagerConflict.
type ActivateBindingVersionParams struct {
	WorkspaceID               string
	BindingID                 string
	VersionID                 string
	ExpectedActiveVersionID   *string
	ExpectedBindingGeneration int64
	Reason                    string
	RequestID                 string
	EffectiveDigest           string
	CapabilityDigest          string
	ApplyClass                string
}

// BindingActivationRecord is the committed activation.
type BindingActivationRecord struct {
	ActiveVersionID   string
	PreviousVersionID *string
	BindingGeneration int64
}

// RuntimeConfigurationStore is the durable configuration state C2 owns. Every
// method is expected to be transactional and to return the sentinel errors
// above so this package maps status codes in one place.
type RuntimeConfigurationStore interface {
	GetBinding(ctx context.Context, workspaceID, bindingID string) (RuntimeBindingRecord, error)
	GetBindingVersion(ctx context.Context, workspaceID, bindingID, versionID string) (RuntimeConfigurationVersionRecord, error)
	ListBindingVersions(ctx context.Context, workspaceID, bindingID string, limit int, cursor string) ([]RuntimeConfigurationVersionRecord, string, error)
	CreateBindingVersion(ctx context.Context, params CreateBindingVersionParams) (RuntimeConfigurationVersionRecord, error)
	ActivateBindingVersion(ctx context.Context, params ActivateBindingVersionParams) (BindingActivationRecord, error)
}

// CapabilityCatalog supplies the provider capability declaration a
// configuration is admitted against. A declaration that cannot be read must
// return ErrRuntimeManagerCapabilitiesUnavailable so the caller learns the
// failure is retryable; returning an empty declaration instead would look like
// a provider with no models and be reported as a permanent rejection.
type CapabilityCatalog interface {
	Capabilities(ctx context.Context, provider string) (runtimeconfig.ProviderCapabilities, error)
}

// PlatformLayerSource supplies the platform precedence layer: the floor of
// non-negotiable defaults and delegability policy. Returning nil means the
// platform declares nothing, which is legal.
type PlatformLayerSource interface {
	PlatformLayer(ctx context.Context) (*runtimeconfig.Config, error)
}

// ---------------------------------------------------------------------------
// API type
// ---------------------------------------------------------------------------

// RuntimeConfigurationAPI serves the binding-scoped configuration endpoints.
// It is a standalone type rather than a method set on the shared Handler so
// this stream owns its own dependencies and adds no field to shared wiring.
type RuntimeConfigurationAPI struct {
	Store        RuntimeConfigurationStore
	Capabilities CapabilityCatalog
	Platform     PlatformLayerSource
}

// NewRuntimeConfigurationAPI returns an API bound to its dependencies.
func NewRuntimeConfigurationAPI(store RuntimeConfigurationStore, capabilities CapabilityCatalog, platform PlatformLayerSource) *RuntimeConfigurationAPI {
	return &RuntimeConfigurationAPI{Store: store, Capabilities: capabilities, Platform: platform}
}

// RuntimeManagerRoute describes one endpoint. The API exposes its own route
// table so integration can mount it verbatim without this stream editing
// shared router wiring, and so a review can read the surface in one place.
type RuntimeManagerRoute struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

// Routes returns the binding-scoped configuration surface.
func (a *RuntimeConfigurationAPI) Routes() []RuntimeManagerRoute {
	const base = "/api/workspaces/{workspaceId}/runtime-bindings/{bindingId}"
	return []RuntimeManagerRoute{
		{http.MethodGet, base + "/configuration-versions", a.ListVersions},
		{http.MethodPost, base + "/configuration-versions", a.CreateVersion},
		{http.MethodPost, base + "/configuration-versions/{versionId}/validate", a.ValidateVersion},
		{http.MethodPost, base + "/configuration-versions/{versionId}/activate", a.ActivateVersion},
		{http.MethodPost, base + "/rollback", a.Rollback},
	}
}

// ---------------------------------------------------------------------------
// Binding-scoped handlers
// ---------------------------------------------------------------------------

// ListVersions returns the append-only version history for a binding.
func (a *RuntimeConfigurationAPI) ListVersions(w http.ResponseWriter, r *http.Request) {
	workspaceID, bindingID, ok := bindingScope(w, r)
	if !ok {
		return
	}
	limit, cursor, ok := pageParams(w, r)
	if !ok {
		return
	}

	records, next, err := a.Store.ListBindingVersions(r.Context(), workspaceID, bindingID, limit, cursor)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	items := make([]runtimeConfigurationVersionResponse, 0, len(records))
	for _, record := range records {
		response, err := versionResponse(record)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		items = append(items, response)
	}
	writeJSON(w, http.StatusOK, runtimeManagerPage[runtimeConfigurationVersionResponse]{Items: items, NextCursor: optionalCursor(next)})
}

// CreateVersion validates a candidate document and appends it as a new
// immutable version. Validation happens before the insert, so an unresolvable
// or capability-rejected document never reaches storage and never becomes a
// version a later activation could pick up.
func (a *RuntimeConfigurationAPI) CreateVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID, bindingID, ok := bindingScope(w, r)
	if !ok {
		return
	}
	requestID, ok := idempotencyKey(w, r)
	if !ok {
		return
	}

	var request createConfigurationVersionRequest
	if !decodeRuntimeManagerRequest(w, r, &request) {
		return
	}
	if !requireReason(w, r, request.Reason) {
		return
	}

	binding, err := a.Store.GetBinding(r.Context(), workspaceID, bindingID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	outcome := a.engine().resolveDocument(r.Context(), binding.Provider, request.Configuration)
	if !outcome.Valid {
		writeRuntimeManagerOutcomeError(w, r, outcome)
		return
	}

	document, err := canonicalDocumentBytes(outcome.Effective)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	record, err := a.Store.CreateBindingVersion(r.Context(), CreateBindingVersionParams{
		WorkspaceID:      workspaceID,
		BindingID:        bindingID,
		Document:         document,
		Digest:           outcome.Effective.Digest,
		ApplyClass:       string(outcome.ApplyClass),
		Reason:           request.Reason,
		RequestID:        requestID,
		CapabilityDigest: outcome.CapabilityDigest,
	})
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	response, err := versionResponse(record)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	validation := outcome.validation()
	response.Validation = &validation
	writeJSON(w, http.StatusCreated, response)
}

// ValidateVersion re-resolves a stored version without activating it. It is a
// pure non-inference check and is safe to call repeatedly: it reports whether
// the version would still be admitted against the current capability
// declaration, which can change after the version was written.
func (a *RuntimeConfigurationAPI) ValidateVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID, bindingID, ok := bindingScope(w, r)
	if !ok {
		return
	}
	versionID := chi.URLParam(r, "versionId")
	if versionID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "version id is required")
		return
	}
	binding, err := a.Store.GetBinding(r.Context(), workspaceID, bindingID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	record, err := a.Store.GetBindingVersion(r.Context(), workspaceID, bindingID, versionID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	document, _, err := materializeStoredVersion(record)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	outcome := a.engine().resolveDocument(r.Context(), binding.Provider, document)
	writeJSON(w, http.StatusOK, outcome.validation())
}

// ActivateVersion activates a stored version under a generation
// compare-and-swap. The document is re-resolved from its immutable stored form
// rather than trusted from the row, so a version cannot be activated on the
// strength of a digest that no longer matches its own document.
func (a *RuntimeConfigurationAPI) ActivateVersion(w http.ResponseWriter, r *http.Request) {
	var request bindingActivationRequest
	if !decodeRuntimeManagerRequest(w, r, &request) {
		return
	}
	a.transition(w, r, bindingTransition{
		targetVersionID:           chi.URLParam(r, "versionId"),
		expectedActiveVersionID:   request.ExpectedActiveVersionID,
		expectedBindingGeneration: request.ExpectedBindingGeneration,
		reason:                    request.Reason,
	})
}

// Rollback activates any prior immutable version. It deliberately reuses the
// activation path: the typed contract's in-memory Rollback restores only the
// single retained previous generation, which cannot express the arbitrary
// target the accepted contract exposes. Re-resolving the target document and
// activating it advances the generation exactly as a forward transition does,
// which is what keeps a rollback auditable and itself reversible.
func (a *RuntimeConfigurationAPI) Rollback(w http.ResponseWriter, r *http.Request) {
	var request bindingRollbackRequest
	if !decodeRuntimeManagerRequest(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.TargetVersionID) == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "target_version_id is required")
		return
	}
	a.transition(w, r, bindingTransition{
		targetVersionID:           request.TargetVersionID,
		expectedActiveVersionID:   request.ExpectedActiveVersionID,
		expectedBindingGeneration: request.ExpectedBindingGeneration,
		reason:                    request.Reason,
	})
}

// bindingTransition is the shared shape of an activation and a rollback. They
// differ only in where the target version id comes from, so they must not
// differ in how the transition is validated or committed.
type bindingTransition struct {
	targetVersionID           string
	expectedActiveVersionID   runtimeNullableString
	expectedBindingGeneration int64
	reason                    string
}

func (a *RuntimeConfigurationAPI) transition(w http.ResponseWriter, r *http.Request, transition bindingTransition) {
	workspaceID, bindingID, ok := bindingScope(w, r)
	if !ok {
		return
	}
	requestID, ok := idempotencyKey(w, r)
	if !ok {
		return
	}
	if transition.targetVersionID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "version id is required")
		return
	}
	if !transition.expectedActiveVersionID.Set {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "expected_active_version_id is required")
		return
	}
	if !requireReason(w, r, transition.reason) {
		return
	}
	if transition.expectedBindingGeneration <= 0 {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "expected_binding_generation must be positive")
		return
	}

	binding, err := a.Store.GetBinding(r.Context(), workspaceID, bindingID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	// Refuse before doing any work if the caller is reasoning about a
	// generation the binding has already moved past.
	if binding.BindingGeneration != transition.expectedBindingGeneration {
		writeRuntimeManagerAPIError(w, r, http.StatusConflict, "expected_binding_generation does not match the current binding generation")
		return
	}
	if !activeVersionMatches(binding.ActiveConfigurationVersionID, transition.expectedActiveVersionID.Value) {
		writeRuntimeManagerAPIError(w, r, http.StatusConflict, "expected_active_version_id does not match the current active version")
		return
	}

	target, err := a.Store.GetBindingVersion(r.Context(), workspaceID, bindingID, transition.targetVersionID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	targetDocument, _, err := materializeStoredVersion(target)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	outcome := a.engine().resolveDocument(r.Context(), binding.Provider, targetDocument)
	if !outcome.Valid {
		writeRuntimeManagerOutcomeError(w, r, outcome)
		return
	}

	// Rebuild the current activation from the currently active version's own
	// immutable document. Activate revalidates it, so a tampered active
	// generation is rejected instead of being used as a transition base.
	var current *runtimeconfig.Activation
	if binding.ActiveConfigurationVersionID != nil {
		activeRecord, err := a.Store.GetBindingVersion(r.Context(), workspaceID, bindingID, *binding.ActiveConfigurationVersionID)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		_, activeEffective, err := materializeStoredVersion(activeRecord)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		generation := uint64(0)
		if binding.BindingGeneration > 0 {
			generation = uint64(binding.BindingGeneration)
		}
		current = &runtimeconfig.Activation{Generation: generation, Active: activeEffective}
	}

	_, plan, err := runtimeconfig.Activate(current, outcome.Effective)
	if err != nil {
		writeRuntimeManagerValidationError(w, r, err, outcome.CapabilityDigest)
		return
	}

	if plan.IsEmpty() {
		activeVersionID := target.ID
		if binding.ActiveConfigurationVersionID != nil {
			activeVersionID = *binding.ActiveConfigurationVersionID
		}
		writeJSON(w, http.StatusOK, runtimeActivationResult{
			ActiveVersionID: activeVersionID, PreviousVersionID: nil,
			ApplyClass: string(runtimeconfig.ReloadHot), PendingAcknowledgement: false,
			CapabilityDigest: outcome.CapabilityDigest,
		})
		return
	}

	activation, err := a.Store.ActivateBindingVersion(r.Context(), ActivateBindingVersionParams{
		WorkspaceID:               workspaceID,
		BindingID:                 bindingID,
		VersionID:                 target.ID,
		ExpectedActiveVersionID:   transition.expectedActiveVersionID.Value,
		ExpectedBindingGeneration: transition.expectedBindingGeneration,
		Reason:                    transition.reason,
		RequestID:                 requestID,
		EffectiveDigest:           outcome.Effective.Digest,
		CapabilityDigest:          outcome.CapabilityDigest,
		ApplyClass:                string(plan.ApplyClass()),
	})
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, runtimeActivationResult{
		ActiveVersionID:   activation.ActiveVersionID,
		PreviousVersionID: activation.PreviousVersionID,
		ApplyClass:        string(plan.ApplyClass()),
		// A transition that moves nothing needs no acknowledgement; anything
		// observable does, including a delegability-only change, because a
		// running process has to confirm what it is now enforcing.
		PendingAcknowledgement: !plan.IsEmpty(),
		CapabilityDigest:       outcome.CapabilityDigest,
	})
}

// ---------------------------------------------------------------------------
// Resolution core, shared with runtime_standard.go
// ---------------------------------------------------------------------------

// resolutionOutcome is the result of admitting one document. It carries either
// a resolved effective configuration or the issues that rejected it, never
// both, so a caller cannot accidentally use a half-resolved value.
type resolutionOutcome struct {
	Valid            bool
	Effective        runtimeconfig.Effective
	ApplyClass       runtimeconfig.ReloadClass
	CapabilityDigest string
	Issues           []runtimeConfigurationIssue
}

func (o resolutionOutcome) validation() runtimeConfigurationValidation {
	validation := runtimeConfigurationValidation{
		Valid:            o.Valid,
		CapabilityDigest: o.CapabilityDigest,
		Issues:           o.Issues,
	}
	if validation.Issues == nil {
		validation.Issues = []runtimeConfigurationIssue{}
	}
	if o.Valid {
		validation.ApplyClass = string(o.ApplyClass)
	}
	return validation
}

// runtimeConfigurationEngine is the admission core shared by both scopes. It
// holds only the read-side dependencies, so the standard and binding APIs
// reuse one implementation of "is this document admissible" without either one
// reaching into the other's store.
type runtimeConfigurationEngine struct {
	Capabilities CapabilityCatalog
	Platform     PlatformLayerSource
}

func (a *RuntimeConfigurationAPI) engine() runtimeConfigurationEngine {
	return runtimeConfigurationEngine{Capabilities: a.Capabilities, Platform: a.Platform}
}

// resolveDocument admits one document against the current capability
// declaration for a provider. It never returns an error: every failure is a
// structured issue set, because the accepted contract models validation as a
// result rather than as a transport failure.
func (e runtimeConfigurationEngine) resolveDocument(ctx context.Context, provider string, document runtimeConfigurationDocument) resolutionOutcome {
	standard, issues := documentToConfig(document)
	if len(issues) != 0 {
		return resolutionOutcome{Issues: issues}
	}

	// The document's own provider wins when it declares one, so validation is
	// performed against the capabilities of the provider actually configured
	// rather than whatever the binding currently points at.
	capabilityProvider := provider
	if document.Values.Provider != nil && *document.Values.Provider != "" {
		capabilityProvider = *document.Values.Provider
	}
	// Without a provider there is no capability authority to admit against.
	// Report the missing field instead of querying the catalog for an empty
	// provider, which would surface as a confusing provider mismatch.
	if strings.TrimSpace(capabilityProvider) == "" {
		return resolutionOutcome{Issues: []runtimeConfigurationIssue{{
			Code:  string(runtimeconfig.ErrRequiredField),
			Field: "provider",
		}}}
	}

	capabilities, err := e.Capabilities.Capabilities(ctx, capabilityProvider)
	if err != nil {
		return resolutionOutcome{Issues: []runtimeConfigurationIssue{capabilityIssue(err)}}
	}

	capabilityDigest, err := runtimeconfig.CapabilityDigest(capabilities)
	if err != nil {
		// An invalid declaration is a permanent server-side fault, not
		// something the client can fix by resending.
		return resolutionOutcome{Issues: issuesFromError(err)}
	}

	var platform *runtimeconfig.Config
	if e.Platform != nil {
		platform, err = e.Platform.PlatformLayer(ctx)
		if err != nil {
			return resolutionOutcome{
				CapabilityDigest: capabilityDigest,
				Issues:           []runtimeConfigurationIssue{{Code: "platform_layer_unavailable", Retryable: true}},
			}
		}
	}

	effective, err := runtimeconfig.Resolve(runtimeconfig.ResolveInput{
		Platform:     platform,
		Standard:     standard,
		Capabilities: capabilities,
	})
	if err != nil {
		return resolutionOutcome{CapabilityDigest: capabilityDigest, Issues: issuesFromError(err)}
	}

	// Classify against nothing so the apply class reported for a candidate is
	// the cost of reaching it from a cold start. A transition-specific class is
	// computed separately at activation time.
	_, plan, err := runtimeconfig.InitialActivation(effective)
	if err != nil {
		return resolutionOutcome{CapabilityDigest: capabilityDigest, Issues: issuesFromError(err)}
	}

	return resolutionOutcome{
		Valid:            true,
		Effective:        effective,
		ApplyClass:       plan.ApplyClass(),
		CapabilityDigest: capabilityDigest,
	}
}

// documentToConfig converts a wire document into a typed layer. It rejects a
// malformed schema version and malformed enum shape here so the typed contract
// receives a well-formed layer; every semantic rule stays in the contract.
func documentToConfig(document runtimeConfigurationDocument) (*runtimeconfig.Config, []runtimeConfigurationIssue) {
	if strings.TrimSpace(document.Version) == "" {
		return nil, []runtimeConfigurationIssue{{Code: string(runtimeconfig.ErrInvalidVersion), Field: "version"}}
	}

	values := runtimeconfig.Values{}
	if document.Values.TransportBinding != nil {
		values.TransportBinding = ptrTo(runtimeconfig.TransportBinding(*document.Values.TransportBinding))
	}
	if document.Values.CLIKind != nil {
		values.CLIKind = ptrTo(runtimeconfig.CLIKind(*document.Values.CLIKind))
	}
	if document.Values.Provider != nil {
		values.Provider = ptrTo(runtimeconfig.ProviderID(*document.Values.Provider))
	}
	values.SubscriptionRef = document.Values.SubscriptionRef
	values.ProviderCatalogVersion = document.Values.ProviderCatalogVersion
	values.CapabilityDigest = document.Values.CapabilityDigest
	if document.Values.Model != nil {
		values.Model = ptrTo(runtimeconfig.ModelID(*document.Values.Model))
	}
	if document.Values.ReasoningMode != nil {
		values.ReasoningMode = ptrTo(runtimeconfig.ReasoningMode(*document.Values.ReasoningMode))
	}
	if document.Values.ReasoningEffort != nil {
		values.ReasoningEffort = ptrTo(runtimeconfig.ReasoningEffort(*document.Values.ReasoningEffort))
	}
	values.ReasoningBudget = document.Values.ReasoningBudget
	if document.Values.Limits != nil {
		values.Limits = runtimeconfig.Limits{
			MaxContextTokens: document.Values.Limits.MaxContextTokens,
			MaxInputTokens:   document.Values.Limits.MaxInputTokens,
			MaxOutputTokens:  document.Values.Limits.MaxOutputTokens,
			MaxTotalTokens:   document.Values.Limits.MaxTotalTokens,
			MaxToolCalls:     document.Values.Limits.MaxToolCalls,
			WallTimeoutMS:    document.Values.Limits.WallTimeoutMS,
			IdleTimeoutMS:    document.Values.Limits.IdleTimeoutMS,
		}
	}
	values.Concurrency = document.Values.Concurrency
	values.Retry = document.Values.Retry
	values.Flags = document.Values.Flags
	values.Environment = document.Values.Environment
	values.Skills = document.Values.Skills
	values.MCPTools = document.Values.MCPTools
	values.Permissions = document.Values.Permissions
	values.Eligibility = document.Values.Eligibility
	values.Health = document.Values.Health
	values.Fallback = document.Values.Fallback

	var delegability map[runtimeconfig.Field]bool
	if len(document.Delegability) != 0 {
		delegability = make(map[runtimeconfig.Field]bool, len(document.Delegability))
		for field, allowed := range document.Delegability {
			delegability[runtimeconfig.Field(field)] = allowed
		}
	}

	return &runtimeconfig.Config{
		Version:      runtimeconfig.Version(document.Version),
		Values:       values,
		Delegability: delegability,
	}, nil
}

// effectiveToDocument re-encodes a resolved configuration as a wire document.
// Responses go through this rather than echoing stored bytes, so a response
// can only ever contain fields the typed contract defines.
func effectiveToDocument(effective runtimeconfig.Effective) runtimeConfigurationDocument {
	document := runtimeConfigurationDocument{Version: string(effective.Version)}
	if effective.Values.TransportBinding != nil {
		document.Values.TransportBinding = ptrTo(string(*effective.Values.TransportBinding))
	}
	if effective.Values.CLIKind != nil {
		document.Values.CLIKind = ptrTo(string(*effective.Values.CLIKind))
	}
	if effective.Values.Provider != nil {
		document.Values.Provider = ptrTo(string(*effective.Values.Provider))
	}
	document.Values.SubscriptionRef = effective.Values.SubscriptionRef
	document.Values.ProviderCatalogVersion = effective.Values.ProviderCatalogVersion
	document.Values.CapabilityDigest = effective.Values.CapabilityDigest
	if effective.Values.Model != nil {
		document.Values.Model = ptrTo(string(*effective.Values.Model))
	}
	if effective.Values.ReasoningMode != nil {
		document.Values.ReasoningMode = ptrTo(string(*effective.Values.ReasoningMode))
	}
	if effective.Values.ReasoningEffort != nil {
		document.Values.ReasoningEffort = ptrTo(string(*effective.Values.ReasoningEffort))
	}
	document.Values.ReasoningBudget = effective.Values.ReasoningBudget
	limits := effective.Values.Limits
	if limits.MaxContextTokens != nil || limits.MaxInputTokens != nil || limits.MaxOutputTokens != nil || limits.MaxTotalTokens != nil || limits.MaxToolCalls != nil || limits.WallTimeoutMS != nil || limits.IdleTimeoutMS != nil {
		document.Values.Limits = &runtimeConfigurationLimits{
			MaxContextTokens: limits.MaxContextTokens, MaxInputTokens: limits.MaxInputTokens,
			MaxOutputTokens: limits.MaxOutputTokens, MaxTotalTokens: limits.MaxTotalTokens,
			MaxToolCalls: limits.MaxToolCalls, WallTimeoutMS: limits.WallTimeoutMS, IdleTimeoutMS: limits.IdleTimeoutMS,
		}
	}
	document.Values.Concurrency = effective.Values.Concurrency
	document.Values.Retry = effective.Values.Retry
	document.Values.Flags = effective.Values.Flags
	document.Values.Environment = effective.Values.Environment
	document.Values.Skills = effective.Values.Skills
	document.Values.MCPTools = effective.Values.MCPTools
	document.Values.Permissions = effective.Values.Permissions
	document.Values.Eligibility = effective.Values.Eligibility
	document.Values.Health = effective.Values.Health
	document.Values.Fallback = effective.Values.Fallback
	if len(effective.Delegable) != 0 {
		document.Delegability = make(map[string]bool, len(effective.Delegable))
		for field, allowed := range effective.Delegable {
			document.Delegability[string(field)] = allowed
		}
	}
	return document
}

// canonicalDocumentBytes serialises the full resolved document used to derive
// the stored digest, including effective delegability inherited from platform.
func canonicalDocumentBytes(effective runtimeconfig.Effective) ([]byte, error) {
	return json.Marshal(effectiveToDocument(effective))
}

// decodeStoredDocument reads a stored document back. Stored bytes are treated
// as untrusted input and decoded strictly, so a row written by an older or
// unexpected producer fails closed instead of being partially interpreted.
func decodeStoredDocument(raw []byte) (runtimeConfigurationDocument, error) {
	var document runtimeConfigurationDocument
	if len(raw) == 0 {
		return document, ErrRuntimeManagerIntegrity
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return document, ErrRuntimeManagerIntegrity
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return runtimeConfigurationDocument{}, ErrRuntimeManagerIntegrity
	}
	return document, nil
}

func materializeStoredVersion(record RuntimeConfigurationVersionRecord) (runtimeConfigurationDocument, runtimeconfig.Effective, error) {
	document, err := decodeStoredDocument(record.Document)
	if err != nil {
		return runtimeConfigurationDocument{}, runtimeconfig.Effective{}, ErrRuntimeManagerIntegrity
	}
	config, issues := documentToConfig(document)
	if len(issues) != 0 {
		return runtimeConfigurationDocument{}, runtimeconfig.Effective{}, ErrRuntimeManagerIntegrity
	}
	effective, err := runtimeconfig.Materialize(config)
	if err != nil || !runtimeconfig.ValidDigest(record.Digest) || effective.Digest != record.Digest {
		return runtimeConfigurationDocument{}, runtimeconfig.Effective{}, ErrRuntimeManagerIntegrity
	}
	return effectiveToDocument(effective), effective, nil
}

// versionResponse renders only a verified canonical stored version.
func versionResponse(record RuntimeConfigurationVersionRecord) (runtimeConfigurationVersionResponse, error) {
	document, _, err := materializeStoredVersion(record)
	if err != nil {
		return runtimeConfigurationVersionResponse{}, err
	}
	return runtimeConfigurationVersionResponse{
		ID: record.ID, VersionNumber: record.VersionNumber, Configuration: document,
		ConfigurationDigest: record.Digest, Reason: record.Reason, State: record.State,
		ApplyClass: record.ApplyClass, CreatedAt: record.CreatedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// Issue mapping
// ---------------------------------------------------------------------------

// retryableIssueCodes lists the codes a client may usefully retry. Every code
// the typed contract produces is a deterministic verdict on the submitted
// document, so none of them are retryable: resending an identical request
// yields an identical rejection. Only a failure to read a dependency is
// retryable, and those codes are minted here rather than by the contract.
var retryableIssueCodes = map[string]bool{
	"capabilities_unavailable":   true,
	"platform_layer_unavailable": true,
}

// issuesFromError converts typed validation failures into wire issues. A
// non-validation error becomes a single opaque issue: an unexpected error's
// text is not a contract and must not be leaked as one.
func issuesFromError(err error) []runtimeConfigurationIssue {
	var validationErrors runtimeconfig.ValidationErrors
	if errors.As(err, &validationErrors) && len(validationErrors) != 0 {
		issues := make([]runtimeConfigurationIssue, 0, len(validationErrors))
		for _, validationError := range validationErrors {
			issues = append(issues, runtimeConfigurationIssue{
				Code:      string(validationError.Code),
				Field:     string(validationError.Field),
				Retryable: retryableIssueCodes[string(validationError.Code)],
			})
		}
		return issues
	}
	return []runtimeConfigurationIssue{{Code: "invalid_configuration"}}
}

// capabilityIssue distinguishes an unavailable capability declaration, which
// is retryable, from an invalid one, which is not.
func capabilityIssue(err error) runtimeConfigurationIssue {
	if errors.Is(err, ErrRuntimeManagerCapabilitiesUnavailable) {
		return runtimeConfigurationIssue{Code: "capabilities_unavailable", Field: "provider", Retryable: true}
	}
	if errors.Is(err, ErrRuntimeManagerNotFound) {
		return runtimeConfigurationIssue{Code: string(runtimeconfig.ErrProviderMismatch), Field: "provider"}
	}
	return runtimeConfigurationIssue{Code: string(runtimeconfig.ErrInvalidCapabilities), Field: "provider"}
}

// ---------------------------------------------------------------------------
// Request helpers
// ---------------------------------------------------------------------------

// decodeRuntimeManagerRequest decodes a bounded body with unknown fields
// rejected and no trailing content allowed.
func decodeRuntimeManagerRequest(w http.ResponseWriter, r *http.Request, destination any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRuntimeConfigurationBody)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "invalid request body")
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "unexpected content after request body")
		return false
	}
	return true
}

// bindingScope reads and requires the workspace and binding path parameters.
func bindingScope(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	workspaceID := chi.URLParam(r, "workspaceId")
	bindingID := chi.URLParam(r, "bindingId")
	if workspaceID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "workspace id is required")
		return "", "", false
	}
	if bindingID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "binding id is required")
		return "", "", false
	}
	return workspaceID, bindingID, true
}

// idempotencyKey requires the header the accepted contract sends on every
// mutation. Without it a retried mutation cannot be deduplicated, so it is
// rejected rather than accepted as a possible duplicate.
func idempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "Idempotency-Key header is required")
		return "", false
	}
	if len(key) > 200 {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "Idempotency-Key header is too long")
		return "", false
	}
	return key, true
}

// requireReason enforces the audit reason every mutation carries.
func requireReason(w http.ResponseWriter, r *http.Request, reason string) bool {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "reason is required")
		return false
	}
	if len(trimmed) > 128 {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "reason is too long")
		return false
	}
	for _, char := range trimmed {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-' || char == '.' {
			continue
		}
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "reason must be a symbolic reason code")
		return false
	}
	return true
}

// pageParams reads bounded pagination parameters.
func pageParams(w http.ResponseWriter, r *http.Request) (int, string, bool) {
	limit := defaultRuntimeManagerPageSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "limit must be a positive integer")
			return 0, "", false
		}
		if parsed > maxRuntimeManagerPageSize {
			parsed = maxRuntimeManagerPageSize
		}
		limit = parsed
	}
	cursor := r.URL.Query().Get("cursor")
	if len(cursor) > 200 {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "cursor is invalid")
		return 0, "", false
	}
	return limit, cursor, true
}

// activeVersionMatches compares the caller's expectation with reality,
// treating nil on both sides as a match so a first activation can be asserted.
func activeVersionMatches(actual, expected *string) bool {
	if actual == nil || expected == nil {
		return actual == nil && expected == nil
	}
	return *actual == *expected
}

func optionalCursor(cursor string) *string {
	if cursor == "" {
		return nil
	}
	return &cursor
}

func ptrTo[T any](value T) *T { return &value }

type runtimeManagerErrorEnvelope struct {
	Error runtimeManagerErrorBody `json:"error"`
}

type runtimeManagerErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Field     string `json:"field,omitempty"`
	RequestID string `json:"request_id"`
	Retryable bool   `json:"retryable"`
}

func runtimeManagerRequestID(r *http.Request) string {
	if requestID := strings.TrimSpace(chimw.GetReqID(r.Context())); requestID != "" {
		return requestID
	}
	return uuid.NewString()
}

func runtimeManagerCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_argument"
	case http.StatusUnauthorized:
		return "unauthenticated"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "version_conflict"
	case http.StatusPreconditionFailed:
		return "capability_unsupported"
	case http.StatusUnprocessableEntity:
		return "invalid_configuration"
	case http.StatusTooManyRequests:
		return "capacity_exhausted"
	case http.StatusServiceUnavailable:
		return "catalog_unavailable"
	default:
		return "internal_error"
	}
}

func writeRuntimeManagerAPIError(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeRuntimeManagerCodedError(w, r, status, runtimeManagerCodeForStatus(status), message, "", status >= 500)
}

func writeRuntimeManagerCodedError(w http.ResponseWriter, r *http.Request, status int, code, message, field string, retryable bool) {
	writeJSON(w, status, runtimeManagerErrorEnvelope{Error: runtimeManagerErrorBody{
		Code: code, Message: message, Field: field, RequestID: runtimeManagerRequestID(r), Retryable: retryable,
	}})
}

// writeRuntimeManagerError maps a store or internal failure onto the frozen
// status/code vocabulary without returning backend error detail.
func writeRuntimeManagerError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrRuntimeManagerNotFound):
		writeRuntimeManagerCodedError(w, r, http.StatusNotFound, "not_found", "resource not found", "", false)
	case errors.Is(err, ErrRuntimeManagerConflict):
		writeRuntimeManagerCodedError(w, r, http.StatusConflict, "generation_conflict", "generation conflict", "", false)
	case errors.Is(err, ErrRuntimeManagerCapabilitiesUnavailable):
		writeRuntimeManagerCodedError(w, r, http.StatusServiceUnavailable, "catalog_unavailable", "capability catalog unavailable", "", true)
	default:
		writeRuntimeManagerCodedError(w, r, http.StatusInternalServerError, "internal_error", "runtime configuration request failed", "", true)
	}
}

func writeRuntimeManagerOutcomeError(w http.ResponseWriter, r *http.Request, outcome resolutionOutcome) {
	status := http.StatusUnprocessableEntity
	code := "invalid_configuration"
	message := "runtime configuration is invalid"
	field := ""
	retryable := false
	if len(outcome.Issues) != 0 {
		field = outcome.Issues[0].Field
		retryable = outcome.Issues[0].Retryable
		for _, issue := range outcome.Issues {
			switch issue.Code {
			case "capabilities_unavailable", "platform_layer_unavailable":
				status, code, message, retryable = http.StatusServiceUnavailable, "catalog_unavailable", "runtime configuration authority unavailable", true
			case string(runtimeconfig.ErrProviderMismatch), string(runtimeconfig.ErrUnsupportedModel), string(runtimeconfig.ErrUnsupportedReasoning), string(runtimeconfig.ErrLimitExceeded), string(runtimeconfig.ErrContextWindowExceeded), string(runtimeconfig.ErrInvalidCapabilities):
				if status != http.StatusServiceUnavailable {
					status, code, message = http.StatusPreconditionFailed, "capability_unsupported", "runtime configuration is unsupported"
				}
			}
		}
	}
	writeRuntimeManagerCodedError(w, r, status, code, message, field, retryable)
}

func writeRuntimeManagerValidationError(w http.ResponseWriter, r *http.Request, err error, _ string) {
	issues := issuesFromError(err)
	outcome := resolutionOutcome{Issues: issues}
	writeRuntimeManagerOutcomeError(w, r, outcome)
}
