package handler

// SPE-6 C3: runtime standard API.
//
// A Runtime Standard is a named, versioned configuration template. Its
// versions are append-only and immutable; activation moves a pointer and never
// edits history. This file owns the standard-scoped endpoints and reuses the
// DTOs, admission engine, issue mapping and request helpers from
// runtime_configuration.go — nothing here restates a validation rule.
//
// Standard scope has no binding generation, so its compare-and-swap fence is
// the expected active version id alone, which is exactly what the accepted
// client contract sends for this scope.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
)

// ---------------------------------------------------------------------------
// Wire DTOs
// ---------------------------------------------------------------------------

// runtimeStandardSummary mirrors RuntimeStandardSummary.
type runtimeStandardSummary struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         *string   `json:"description"`
	ActiveVersionID     *string   `json:"active_version_id"`
	ActiveVersionNumber *int64    `json:"active_version_number,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// runtimeStandardDetail mirrors RuntimeStandardDetail.
type runtimeStandardDetail struct {
	runtimeStandardSummary
	ActiveVersion *runtimeConfigurationVersionResponse                    `json:"active_version"`
	Versions      runtimeManagerPage[runtimeConfigurationVersionResponse] `json:"versions"`
}

// createRuntimeStandardRequest mirrors CreateRuntimeStandardRequest.
type createRuntimeStandardRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// ---------------------------------------------------------------------------
// Plug-in interfaces
// ---------------------------------------------------------------------------

// RuntimeStandardRecord is the durable standard row this API needs.
type RuntimeStandardRecord struct {
	ID                  string
	Name                string
	Description         *string
	ActiveVersionID     *string
	ActiveVersionNumber *int64
	UpdatedAt           time.Time
}

// CreateStandardParams creates a standard with no active version. A standard is
// deliberately born empty: a version has to be written and admitted before
// anything can be activated.
type CreateStandardParams struct {
	Name        string
	Description *string
	RequestID   string
}

// CreateStandardVersionParams is an append-only version insert.
type CreateStandardVersionParams struct {
	StandardID       string
	Document         []byte
	Digest           string
	ApplyClass       string
	Reason           string
	RequestID        string
	CapabilityDigest string
}

// ActivateStandardVersionParams is a compare-and-swap activation fenced on the
// expected active version id.
type ActivateStandardVersionParams struct {
	StandardID              string
	VersionID               string
	ExpectedActiveVersionID *string
	Reason                  string
	RequestID               string
	EffectiveDigest         string
	CapabilityDigest        string
	ApplyClass              string
}

// StandardActivationRecord is the committed activation.
type StandardActivationRecord struct {
	ActiveVersionID   string
	PreviousVersionID *string
}

// RuntimeStandardStore is the durable standard state C2 owns. Implementations
// return the sentinel errors declared in runtime_configuration.go so status
// mapping stays in one place.
type RuntimeStandardStore interface {
	ListStandards(ctx context.Context, limit int, cursor string) ([]RuntimeStandardRecord, string, error)
	CreateStandard(ctx context.Context, params CreateStandardParams) (RuntimeStandardRecord, error)
	GetStandard(ctx context.Context, standardID string) (RuntimeStandardRecord, error)
	ListStandardVersions(ctx context.Context, standardID string, limit int, cursor string) ([]RuntimeConfigurationVersionRecord, string, error)
	GetStandardVersion(ctx context.Context, standardID, versionID string) (RuntimeConfigurationVersionRecord, error)
	CreateStandardVersion(ctx context.Context, params CreateStandardVersionParams) (RuntimeConfigurationVersionRecord, error)
	ActivateStandardVersion(ctx context.Context, params ActivateStandardVersionParams) (StandardActivationRecord, error)
}

// ---------------------------------------------------------------------------
// API type
// ---------------------------------------------------------------------------

// RuntimeStandardAPI serves the standard-scoped endpoints.
type RuntimeStandardAPI struct {
	Store        RuntimeStandardStore
	Capabilities CapabilityCatalog
	Platform     PlatformLayerSource
}

// NewRuntimeStandardAPI returns an API bound to its dependencies.
func NewRuntimeStandardAPI(store RuntimeStandardStore, capabilities CapabilityCatalog, platform PlatformLayerSource) *RuntimeStandardAPI {
	return &RuntimeStandardAPI{Store: store, Capabilities: capabilities, Platform: platform}
}

func (a *RuntimeStandardAPI) engine() runtimeConfigurationEngine {
	return runtimeConfigurationEngine{Capabilities: a.Capabilities, Platform: a.Platform}
}

// Routes returns the standard-scoped surface.
func (a *RuntimeStandardAPI) Routes() []RuntimeManagerRoute {
	const base = "/api/runtime-standards"
	return []RuntimeManagerRoute{
		{http.MethodGet, base, a.List},
		{http.MethodPost, base, a.Create},
		{http.MethodGet, base + "/{standardId}", a.Get},
		{http.MethodPost, base + "/{standardId}/versions", a.CreateVersion},
		{http.MethodPost, base + "/{standardId}/versions/{versionId}/validate", a.ValidateVersion},
		{http.MethodPost, base + "/{standardId}/versions/{versionId}/activate", a.ActivateVersion},
		{http.MethodPost, base + "/{standardId}/rollback", a.Rollback},
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// List returns a page of standards.
func (a *RuntimeStandardAPI) List(w http.ResponseWriter, r *http.Request) {
	limit, cursor, ok := pageParams(w, r)
	if !ok {
		return
	}
	records, next, err := a.Store.ListStandards(r.Context(), limit, cursor)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	items := make([]runtimeStandardSummary, 0, len(records))
	for _, record := range records {
		items = append(items, standardSummary(record))
	}
	writeJSON(w, http.StatusOK, runtimeManagerPage[runtimeStandardSummary]{Items: items, NextCursor: optionalCursor(next)})
}

// Create creates an empty standard.
func (a *RuntimeStandardAPI) Create(w http.ResponseWriter, r *http.Request) {
	requestID, ok := idempotencyKey(w, r)
	if !ok {
		return
	}
	var request createRuntimeStandardRequest
	if !decodeRuntimeManagerRequest(w, r, &request) {
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 200 {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "name is too long")
		return
	}
	if request.Description != nil && len(*request.Description) > 2000 {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "description is too long")
		return
	}

	record, err := a.Store.CreateStandard(r.Context(), CreateStandardParams{
		Name:        name,
		Description: request.Description,
		RequestID:   requestID,
	})
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, runtimeStandardDetail{
		runtimeStandardSummary: standardSummary(record),
		Versions: runtimeManagerPage[runtimeConfigurationVersionResponse]{
			Items: []runtimeConfigurationVersionResponse{}, NextCursor: nil,
		},
	})
}

// Get returns a standard with its version history and active version.
func (a *RuntimeStandardAPI) Get(w http.ResponseWriter, r *http.Request) {
	standardID := chi.URLParam(r, "standardId")
	if standardID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "standard id is required")
		return
	}
	limit, cursor, ok := pageParams(w, r)
	if !ok {
		return
	}
	record, err := a.Store.GetStandard(r.Context(), standardID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	versions, next, err := a.Store.ListStandardVersions(r.Context(), standardID, limit, cursor)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	detail := runtimeStandardDetail{
		runtimeStandardSummary: standardSummary(record),
		Versions: runtimeManagerPage[runtimeConfigurationVersionResponse]{
			Items:      make([]runtimeConfigurationVersionResponse, 0, len(versions)),
			NextCursor: optionalCursor(next),
		},
	}
	for _, version := range versions {
		response, err := versionResponse(version)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		detail.Versions.Items = append(detail.Versions.Items, response)
		if record.ActiveVersionID != nil && *record.ActiveVersionID == version.ID {
			active := response
			detail.ActiveVersion = &active
		}
	}
	if record.ActiveVersionID != nil && detail.ActiveVersion == nil {
		activeRecord, err := a.Store.GetStandardVersion(r.Context(), standardID, *record.ActiveVersionID)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		active, err := versionResponse(activeRecord)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		detail.ActiveVersion = &active
	}
	writeJSON(w, http.StatusOK, detail)
}

// CreateVersion admits a candidate document and appends it as a new immutable
// version. The document is validated before the insert, so storage never holds
// a version that could not be activated.
func (a *RuntimeStandardAPI) CreateVersion(w http.ResponseWriter, r *http.Request) {
	standardID := chi.URLParam(r, "standardId")
	if standardID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "standard id is required")
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

	if _, err := a.Store.GetStandard(r.Context(), standardID); err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	// A standard carries no provider of its own, so the document must declare
	// the provider it is admitted against.
	outcome := a.engine().resolveDocument(r.Context(), "", request.Configuration)
	if !outcome.Valid {
		writeRuntimeManagerOutcomeError(w, r, outcome)
		return
	}

	document, err := canonicalDocumentBytes(outcome.Effective)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	record, err := a.Store.CreateStandardVersion(r.Context(), CreateStandardVersionParams{
		StandardID:       standardID,
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

// ValidateVersion re-admits a stored version without activating it. The
// capability declaration can change after a version is written, so a version
// that was valid at creation is not assumed to still be valid now.
func (a *RuntimeStandardAPI) ValidateVersion(w http.ResponseWriter, r *http.Request) {
	standardID := chi.URLParam(r, "standardId")
	versionID := chi.URLParam(r, "versionId")
	if standardID == "" || versionID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "standard id and version id are required")
		return
	}
	record, err := a.Store.GetStandardVersion(r.Context(), standardID, versionID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	document, _, err := materializeStoredVersion(record)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a.engine().resolveDocument(r.Context(), "", document).validation())
}

// ActivateVersion activates a stored version under an expected-active-version
// compare-and-swap.
func (a *RuntimeStandardAPI) ActivateVersion(w http.ResponseWriter, r *http.Request) {
	var request activationRequest
	if !decodeRuntimeManagerRequest(w, r, &request) {
		return
	}
	a.transition(w, r, standardTransition{
		targetVersionID:         chi.URLParam(r, "versionId"),
		expectedActiveVersionID: request.ExpectedActiveVersionID,
		reason:                  request.Reason,
	})
}

// Rollback activates any prior immutable version of the standard. Like the
// binding rollback it goes through the activation path rather than the typed
// contract's single-step Rollback, because the accepted contract names an
// arbitrary target and storage keeps the full append-only history.
func (a *RuntimeStandardAPI) Rollback(w http.ResponseWriter, r *http.Request) {
	var request rollbackRequest
	if !decodeRuntimeManagerRequest(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.TargetVersionID) == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "target_version_id is required")
		return
	}
	a.transition(w, r, standardTransition{
		targetVersionID:         request.TargetVersionID,
		expectedActiveVersionID: request.ExpectedActiveVersionID,
		reason:                  request.Reason,
	})
}

// standardTransition is the shared shape of a standard activation and
// rollback. They must agree on validation and commit, differing only in where
// the target comes from.
type standardTransition struct {
	targetVersionID         string
	expectedActiveVersionID runtimeNullableString
	reason                  string
}

func (a *RuntimeStandardAPI) transition(w http.ResponseWriter, r *http.Request, transition standardTransition) {
	standardID := chi.URLParam(r, "standardId")
	if standardID == "" {
		writeRuntimeManagerAPIError(w, r, http.StatusBadRequest, "standard id is required")
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

	record, err := a.Store.GetStandard(r.Context(), standardID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	if !activeVersionMatches(record.ActiveVersionID, transition.expectedActiveVersionID.Value) {
		writeRuntimeManagerAPIError(w, r, http.StatusConflict, "expected_active_version_id does not match the current active version")
		return
	}

	target, err := a.Store.GetStandardVersion(r.Context(), standardID, transition.targetVersionID)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	targetDocument, _, err := materializeStoredVersion(target)
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}
	outcome := a.engine().resolveDocument(r.Context(), "", targetDocument)
	if !outcome.Valid {
		writeRuntimeManagerOutcomeError(w, r, outcome)
		return
	}

	// Rebuild the current activation from the active version's own immutable
	// document so Activate revalidates it. A tampered active generation is
	// rejected rather than used as the base of a new transition.
	var current *runtimeconfig.Activation
	if record.ActiveVersionID != nil {
		activeRecord, err := a.Store.GetStandardVersion(r.Context(), standardID, *record.ActiveVersionID)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		_, activeEffective, err := materializeStoredVersion(activeRecord)
		if err != nil {
			writeRuntimeManagerError(w, r, err)
			return
		}
		generation := uint64(1)
		if activeRecord.VersionNumber > 0 {
			generation = uint64(activeRecord.VersionNumber)
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
		if record.ActiveVersionID != nil {
			activeVersionID = *record.ActiveVersionID
		}
		writeJSON(w, http.StatusOK, runtimeActivationResult{
			ActiveVersionID: activeVersionID, PreviousVersionID: nil,
			ApplyClass: string(runtimeconfig.ReloadHot), PendingAcknowledgement: false,
			CapabilityDigest: outcome.CapabilityDigest,
		})
		return
	}

	activation, err := a.Store.ActivateStandardVersion(r.Context(), ActivateStandardVersionParams{
		StandardID:              standardID,
		VersionID:               target.ID,
		ExpectedActiveVersionID: transition.expectedActiveVersionID.Value,
		Reason:                  transition.reason,
		RequestID:               requestID,
		EffectiveDigest:         outcome.Effective.Digest,
		CapabilityDigest:        outcome.CapabilityDigest,
		ApplyClass:              string(plan.ApplyClass()),
	})
	if err != nil {
		writeRuntimeManagerError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, runtimeActivationResult{
		ActiveVersionID:        activation.ActiveVersionID,
		PreviousVersionID:      activation.PreviousVersionID,
		ApplyClass:             string(plan.ApplyClass()),
		PendingAcknowledgement: !plan.IsEmpty(),
		CapabilityDigest:       outcome.CapabilityDigest,
	})
}

func standardSummary(record RuntimeStandardRecord) runtimeStandardSummary {
	return runtimeStandardSummary{
		ID:                  record.ID,
		Name:                record.Name,
		Description:         record.Description,
		ActiveVersionID:     record.ActiveVersionID,
		ActiveVersionNumber: record.ActiveVersionNumber,
		UpdatedAt:           record.UpdatedAt,
	}
}
