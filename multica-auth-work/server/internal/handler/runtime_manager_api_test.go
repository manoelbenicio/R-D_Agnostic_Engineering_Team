package handler

// SPE-6 C3: runtime manager API tests.
//
// These tests exercise the handlers through a real chi router built from each
// API's own Routes() table, so the route patterns and the URL parameter names
// the handlers read are covered too — a mismatch between them would otherwise
// only surface at integration time.
//
// Persistence and capability discovery are faked because C1 and C2 are
// blocked. The fakes record what they were asked to commit, which is what lets
// these tests assert the compare-and-swap fences and the exact digest and
// apply-class values that reach storage.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type rmFakeCatalog struct {
	capabilities runtimeconfig.ProviderCapabilities
	err          error
	askedFor     []string
}

func (c *rmFakeCatalog) Capabilities(_ context.Context, provider string) (runtimeconfig.ProviderCapabilities, error) {
	c.askedFor = append(c.askedFor, provider)
	if c.err != nil {
		return runtimeconfig.ProviderCapabilities{}, c.err
	}
	return c.capabilities, nil
}

type rmFakeConfigStore struct {
	binding  RuntimeBindingRecord
	versions map[string]RuntimeConfigurationVersionRecord
	list     []RuntimeConfigurationVersionRecord
	nextPage string

	createErr   error
	activateErr error

	created   []CreateBindingVersionParams
	activated []ActivateBindingVersionParams
}

func (s *rmFakeConfigStore) GetBinding(_ context.Context, _, _ string) (RuntimeBindingRecord, error) {
	return s.binding, nil
}

func (s *rmFakeConfigStore) GetBindingVersion(_ context.Context, _, _, versionID string) (RuntimeConfigurationVersionRecord, error) {
	record, ok := s.versions[versionID]
	if !ok {
		return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
	}
	return record, nil
}

func (s *rmFakeConfigStore) ListBindingVersions(_ context.Context, _, _ string, _ int, _ string) ([]RuntimeConfigurationVersionRecord, string, error) {
	return s.list, s.nextPage, nil
}

func (s *rmFakeConfigStore) CreateBindingVersion(_ context.Context, params CreateBindingVersionParams) (RuntimeConfigurationVersionRecord, error) {
	s.created = append(s.created, params)
	if s.createErr != nil {
		return RuntimeConfigurationVersionRecord{}, s.createErr
	}
	return RuntimeConfigurationVersionRecord{
		ID:            "version-new",
		VersionNumber: 3,
		Document:      params.Document,
		Digest:        params.Digest,
		ApplyClass:    params.ApplyClass,
		Reason:        params.Reason,
		State:         "draft",
		CreatedAt:     time.Unix(0, 0).UTC(),
	}, nil
}

func (s *rmFakeConfigStore) ActivateBindingVersion(_ context.Context, params ActivateBindingVersionParams) (BindingActivationRecord, error) {
	s.activated = append(s.activated, params)
	if s.activateErr != nil {
		return BindingActivationRecord{}, s.activateErr
	}
	return BindingActivationRecord{
		ActiveVersionID:   params.VersionID,
		PreviousVersionID: params.ExpectedActiveVersionID,
		BindingGeneration: params.ExpectedBindingGeneration + 1,
	}, nil
}

type rmFakeStandardStore struct {
	standard   RuntimeStandardRecord
	versions   map[string]RuntimeConfigurationVersionRecord
	list       []RuntimeConfigurationVersionRecord
	nextPage   string
	listLimit  int
	listCursor string

	created   []CreateStandardVersionParams
	activated []ActivateStandardVersionParams
}

func (s *rmFakeStandardStore) ListStandards(_ context.Context, _ int, _ string) ([]RuntimeStandardRecord, string, error) {
	return []RuntimeStandardRecord{s.standard}, "", nil
}

func (s *rmFakeStandardStore) CreateStandard(_ context.Context, params CreateStandardParams) (RuntimeStandardRecord, error) {
	return RuntimeStandardRecord{ID: "standard-new", Name: params.Name, Description: params.Description, UpdatedAt: time.Unix(0, 0).UTC()}, nil
}

func (s *rmFakeStandardStore) GetStandard(_ context.Context, _ string) (RuntimeStandardRecord, error) {
	return s.standard, nil
}

func (s *rmFakeStandardStore) ListStandardVersions(_ context.Context, _ string, limit int, cursor string) ([]RuntimeConfigurationVersionRecord, string, error) {
	s.listLimit, s.listCursor = limit, cursor
	return s.list, s.nextPage, nil
}

func (s *rmFakeStandardStore) GetStandardVersion(_ context.Context, _, versionID string) (RuntimeConfigurationVersionRecord, error) {
	record, ok := s.versions[versionID]
	if !ok {
		return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
	}
	return record, nil
}

func (s *rmFakeStandardStore) CreateStandardVersion(_ context.Context, params CreateStandardVersionParams) (RuntimeConfigurationVersionRecord, error) {
	s.created = append(s.created, params)
	return RuntimeConfigurationVersionRecord{
		ID:            "standard-version-new",
		VersionNumber: 2,
		Document:      params.Document,
		Digest:        params.Digest,
		ApplyClass:    params.ApplyClass,
		Reason:        params.Reason,
		CreatedAt:     time.Unix(0, 0).UTC(),
	}, nil
}

func (s *rmFakeStandardStore) ActivateStandardVersion(_ context.Context, params ActivateStandardVersionParams) (StandardActivationRecord, error) {
	s.activated = append(s.activated, params)
	return StandardActivationRecord{ActiveVersionID: params.VersionID, PreviousVersionID: params.ExpectedActiveVersionID}, nil
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func rmCapabilities() runtimeconfig.ProviderCapabilities {
	return runtimeconfig.ProviderCapabilities{
		Version:  runtimeconfig.VersionV1,
		Provider: runtimeconfig.ProviderID("provider-a"),
		Models: map[runtimeconfig.ModelID]runtimeconfig.ModelCapabilities{
			runtimeconfig.ModelID("model-a"): {
				ReasoningEfforts:    []runtimeconfig.ReasoningEffort{"low", "medium", "high"},
				MaxInputTokens:      8_000,
				MaxOutputTokens:     4_000,
				ContextWindowTokens: 10_000,
				MaxToolCalls:        20,
			},
			runtimeconfig.ModelID("model-b"): {
				ReasoningEfforts:    []runtimeconfig.ReasoningEffort{"low", "high"},
				MaxInputTokens:      16_000,
				MaxOutputTokens:     8_000,
				ContextWindowTokens: 20_000,
				MaxToolCalls:        40,
			},
		},
	}
}

// rmValidDocument is a whole configuration: every required field is present, so
// it resolves rather than failing on a missing field.
func rmValidDocument(model string) runtimeConfigurationDocument {
	return runtimeConfigurationDocument{
		Version: "v1",
		Values: runtimeConfigurationValues{
			TransportBinding: ptrTo("omniroute"),
			CLIKind:          ptrTo("codex"),
			Provider:         ptrTo("provider-a"),
			Model:            ptrTo(model),
			ReasoningEffort:  ptrTo("low"),
		},
		Delegability: map[string]bool{"model": true},
	}
}

func rmStoredRecord(t *testing.T, id string, versionNumber int64, model string) RuntimeConfigurationVersionRecord {
	t.Helper()
	config, issues := documentToConfig(rmValidDocument(model))
	if len(issues) != 0 {
		t.Fatalf("convert stored document: %+v", issues)
	}
	effective, err := runtimeconfig.Materialize(config)
	if err != nil {
		t.Fatalf("materialize stored document: %v", err)
	}
	document, err := canonicalDocumentBytes(effective)
	if err != nil {
		t.Fatalf("encode stored document: %v", err)
	}
	return RuntimeConfigurationVersionRecord{ID: id, VersionNumber: versionNumber, Document: document, Digest: effective.Digest}
}

// rmConfigAPI wires a binding-scoped API onto a router built from its own
// route table.
func rmConfigAPI(store RuntimeConfigurationStore, catalog CapabilityCatalog) http.Handler {
	api := NewRuntimeConfigurationAPI(store, catalog, nil)
	router := chi.NewRouter()
	for _, route := range api.Routes() {
		router.Method(route.Method, route.Pattern, route.Handler)
	}
	return router
}

func rmStandardAPI(store RuntimeStandardStore, catalog CapabilityCatalog) http.Handler {
	api := NewRuntimeStandardAPI(store, catalog, nil)
	router := chi.NewRouter()
	for _, route := range api.Routes() {
		router.Method(route.Method, route.Pattern, route.Handler)
	}
	return router
}

func rmRequest(t *testing.T, router http.Handler, method, path string, body any, withKey bool) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	switch typed := body.(type) {
	case nil:
		reader = strings.NewReader("{}")
	case string:
		reader = strings.NewReader(typed)
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = strings.NewReader(string(raw))
	}
	request := httptest.NewRequest(method, path, reader)
	if withKey {
		request.Header.Set("Idempotency-Key", "key-1")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func rmDecodeValidation(t *testing.T, recorder *httptest.ResponseRecorder) runtimeConfigurationValidation {
	t.Helper()
	var validation runtimeConfigurationValidation
	if err := json.Unmarshal(recorder.Body.Bytes(), &validation); err != nil {
		t.Fatalf("decode validation: %v (body %s)", err, recorder.Body.String())
	}
	return validation
}

func rmAssertRaw64(t *testing.T, digest, label string) {
	t.Helper()
	if !runtimeconfig.ValidDigest(digest) {
		t.Fatalf("%s is not a raw lowercase 64-hex digest: %q", label, digest)
	}
}

func rmAssertApplyClass(t *testing.T, applyClass, label string) {
	t.Helper()
	if applyClass != "hot" && applyClass != "restart" {
		t.Fatalf("%s = %q, outside the accepted apply_class domain", label, applyClass)
	}
}

const rmBindingVersionsPath = "/api/workspaces/workspace-1/runtime-bindings/binding-1/configuration-versions"
const rmBindingRollbackPath = "/api/workspaces/workspace-1/runtime-bindings/binding-1/rollback"

func rmBinding(activeVersionID *string, generation int64) RuntimeBindingRecord {
	return RuntimeBindingRecord{
		ID:                           "binding-1",
		WorkspaceID:                  "workspace-1",
		Provider:                     "provider-a",
		TransportBinding:             "omniroute",
		BindingGeneration:            generation,
		ActiveConfigurationVersionID: activeVersionID,
		State:                        "active",
	}
}

// ---------------------------------------------------------------------------
// Explicit DTOs and fail-closed decoding
// ---------------------------------------------------------------------------

func TestRuntimeConfigurationRejectsUnrecognisedRequestShape(t *testing.T) {
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	bodies := map[string]string{
		"unknown top level field": `{"configuration":{"version":"v1","values":{}},"reason":"r","surprise":1}`,
		"unknown document field":  `{"configuration":{"version":"v1","values":{},"extra":1},"reason":"r"}`,
		"unknown value field":     `{"configuration":{"version":"v1","values":{"endpoint":"http://x"}},"reason":"r"}`,
		"unknown limits field":    `{"configuration":{"version":"v1","values":{"limits":{"max_files":1}}},"reason":"r"}`,
		"trailing content":        `{"configuration":{"version":"v1","values":{}},"reason":"r"}{"another":true}`,
		"not an object":           `[]`,
		"wrong type for reason":   `{"configuration":{"version":"v1","values":{}},"reason":5}`,
	}

	for name, body := range bodies {
		recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, body, true)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %s)", name, recorder.Code, recorder.Body.String())
		}
	}

	// Nothing may reach storage from a rejected request.
	if len(store.created) != 0 {
		t.Fatalf("rejected requests reached the store: %d writes", len(store.created))
	}
}

func TestRuntimeConfigurationRequiresIdempotencyKeyAndReason(t *testing.T) {
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	body := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "initial"}
	if recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, body, false); recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing Idempotency-Key: status = %d, want 400", recorder.Code)
	}

	blank := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "   "}
	if recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, blank, true); recorder.Code != http.StatusBadRequest {
		t.Fatalf("blank reason: status = %d, want 400", recorder.Code)
	}

	if len(store.created) != 0 {
		t.Fatalf("rejected requests reached the store: %d writes", len(store.created))
	}
}

// ---------------------------------------------------------------------------
// capability_digest, apply_class, secret-free responses
// ---------------------------------------------------------------------------

func TestRuntimeConfigurationCreateVersionEmitsRaw64AndStorableApplyClass(t *testing.T) {
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	catalog := &rmFakeCatalog{capabilities: rmCapabilities()}
	router := rmConfigAPI(store, catalog)

	body := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "initial"}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, body, true)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", recorder.Code, recorder.Body.String())
	}

	var response runtimeConfigurationVersionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	rmAssertRaw64(t, response.ConfigurationDigest, "configuration_digest")
	rmAssertApplyClass(t, response.ApplyClass, "apply_class")
	if response.Validation == nil {
		t.Fatal("created version carries no validation result")
	}
	rmAssertRaw64(t, response.Validation.CapabilityDigest, "validation capability_digest")
	if !response.Validation.Valid {
		t.Fatalf("created version reported invalid: %+v", response.Validation.Issues)
	}

	// A whole first configuration sets restart-class fields, so it is restart.
	if response.ApplyClass != "restart" {
		t.Fatalf("apply_class = %q, want restart for a first whole configuration", response.ApplyClass)
	}

	// The values that reached storage must carry the same raw-64 digests and a
	// storable apply class.
	if len(store.created) != 1 {
		t.Fatalf("store writes = %d, want 1", len(store.created))
	}
	written := store.created[0]
	rmAssertRaw64(t, written.Digest, "stored configuration digest")
	rmAssertRaw64(t, written.CapabilityDigest, "stored capability digest")
	rmAssertApplyClass(t, written.ApplyClass, "stored apply_class")
	if written.RequestID != "key-1" {
		t.Fatalf("stored request id = %q, want the idempotency key", written.RequestID)
	}
}

func TestRuntimeConfigurationResponsesAreSecretFreeAndPathless(t *testing.T) {
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	body := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "initial"}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, body, true)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}

	// The response is re-encoded from the typed contract, so the document may
	// only contain contract keys. Decoding it strictly is the assertion.
	var envelope struct {
		Configuration json.RawMessage `json:"configuration"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(envelope.Configuration)))
	decoder.DisallowUnknownFields()
	var document runtimeConfigurationDocument
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("response document carries a field outside the contract: %v", err)
	}

	// No response field may look like a filesystem path or a bearer secret.
	for _, forbidden := range []string{"/home/", "auth.json", "Bearer ", "home_path", "credential"} {
		if strings.Contains(recorder.Body.String(), forbidden) {
			t.Fatalf("response body contains %q", forbidden)
		}
	}
}

func TestRuntimeConfigurationStoredDocumentWithExtraFieldFailsClosed(t *testing.T) {
	// A row written by an unexpected producer must not be partially
	// interpreted; the version is unreadable rather than half-applied.
	store := &rmFakeConfigStore{
		binding: rmBinding(nil, 1),
		versions: map[string]RuntimeConfigurationVersionRecord{
			"version-1": {ID: "version-1", VersionNumber: 1, Document: []byte(`{"version":"v1","values":{},"home_path":"/home/x"}`)},
		},
	}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath+"/version-1/validate", nil, true)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for an undecodable stored document", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "/home/") {
		t.Fatalf("error body leaked stored content: %s", recorder.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Retryable issue mapping
// ---------------------------------------------------------------------------

func TestRuntimeConfigurationRetryableIssueMapping(t *testing.T) {
	unavailable := &rmFakeCatalog{err: ErrRuntimeManagerCapabilitiesUnavailable}
	router := rmConfigAPI(&rmFakeConfigStore{binding: rmBinding(nil, 1)}, unavailable)
	body := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "initial"}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, body, true)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body %s)", recorder.Code, recorder.Body.String())
	}
	var envelope runtimeManagerErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope.Error.Code != "catalog_unavailable" || !envelope.Error.Retryable || envelope.Error.RequestID == "" {
		t.Fatalf("error = %+v, want retryable catalog_unavailable with request id", envelope.Error)
	}

	available := &rmFakeCatalog{capabilities: rmCapabilities()}
	router = rmConfigAPI(&rmFakeConfigStore{binding: rmBinding(nil, 1)}, available)
	unsupported := createConfigurationVersionRequest{Configuration: rmValidDocument("model-z"), Reason: "initial"}
	recorder = rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, unsupported, true)
	if recorder.Code != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want 412", recorder.Code)
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope.Error.Code != "capability_unsupported" || envelope.Error.Retryable {
		t.Fatalf("error = %+v, want non-retryable capability_unsupported", envelope.Error)
	}
}

func TestRuntimeConfigurationMissingProviderFailsClosedWithoutCatalogCall(t *testing.T) {
	catalog := &rmFakeCatalog{capabilities: rmCapabilities()}
	router := rmConfigAPI(&rmFakeConfigStore{binding: RuntimeBindingRecord{ID: "binding-1", WorkspaceID: "workspace-1"}}, catalog)

	document := rmValidDocument("model-a")
	document.Values.Provider = nil
	body := createConfigurationVersionRequest{Configuration: document, Reason: "initial"}

	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, body, true)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
	var envelope runtimeManagerErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope.Error.Code != "invalid_configuration" || envelope.Error.Field != "provider" {
		t.Fatalf("error = %+v, want invalid_configuration on provider", envelope.Error)
	}
	if len(catalog.askedFor) != 0 {
		t.Fatalf("catalog was queried for an absent provider: %v", catalog.askedFor)
	}
}

// ---------------------------------------------------------------------------
// Compare-and-swap fences
// ---------------------------------------------------------------------------

func TestRuntimeConfigurationActivateEnforcesGenerationCAS(t *testing.T) {
	active := "version-1"
	store := &rmFakeConfigStore{
		binding: rmBinding(&active, 7),
		versions: map[string]RuntimeConfigurationVersionRecord{
			"version-1": rmStoredRecord(t, "version-1", 1, "model-a"),
			"version-2": rmStoredRecord(t, "version-2", 2, "model-b"),
		},
	}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	stale := bindingActivationRequest{
		ExpectedActiveVersionID:   expectedActive(&active),
		Reason:                    "stale_generation",
		ExpectedBindingGeneration: 6,
	}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath+"/version-2/activate", stale, true)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("stale generation: status = %d, want 409", recorder.Code)
	}

	wrongActive := "version-9"
	mismatched := bindingActivationRequest{
		ExpectedActiveVersionID:   expectedActive(&wrongActive),
		Reason:                    "wrong_active_version",
		ExpectedBindingGeneration: 7,
	}
	recorder = rmRequest(t, router, http.MethodPost, rmBindingVersionsPath+"/version-2/activate", mismatched, true)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("wrong active version: status = %d, want 409", recorder.Code)
	}

	nonPositive := bindingActivationRequest{
		ExpectedActiveVersionID:   expectedActive(&active),
		Reason:                    "bad_generation",
		ExpectedBindingGeneration: 0,
	}
	recorder = rmRequest(t, router, http.MethodPost, rmBindingVersionsPath+"/version-2/activate", nonPositive, true)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("non-positive generation: status = %d, want 400", recorder.Code)
	}

	if len(store.activated) != 0 {
		t.Fatalf("a refused transition reached the store: %d activations", len(store.activated))
	}
}

func TestRuntimeConfigurationActivateCommitsExpectedFences(t *testing.T) {
	active := "version-1"
	store := &rmFakeConfigStore{
		binding: rmBinding(&active, 7),
		versions: map[string]RuntimeConfigurationVersionRecord{
			"version-1": rmStoredRecord(t, "version-1", 1, "model-a"),
			"version-2": rmStoredRecord(t, "version-2", 2, "model-b"),
		},
	}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	request := bindingActivationRequest{
		ExpectedActiveVersionID:   expectedActive(&active),
		Reason:                    "promote_model-b",
		ExpectedBindingGeneration: 7,
	}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath+"/version-2/activate", request, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var result runtimeActivationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ActiveVersionID != "version-2" {
		t.Fatalf("active_version_id = %q, want version-2", result.ActiveVersionID)
	}
	if result.PreviousVersionID == nil || *result.PreviousVersionID != "version-1" {
		t.Fatalf("previous_version_id = %v, want version-1", result.PreviousVersionID)
	}
	rmAssertApplyClass(t, result.ApplyClass, "apply_class")
	rmAssertRaw64(t, result.CapabilityDigest, "capability_digest")

	// model is a hot field, so promoting model-b alone must not demand a
	// restart, and the change is still observable so it needs acknowledgement.
	if result.ApplyClass != "hot" {
		t.Fatalf("apply_class = %q, want hot for a model-only change", result.ApplyClass)
	}
	if !result.PendingAcknowledgement {
		t.Fatal("an observable change reported no pending acknowledgement")
	}

	if len(store.activated) != 1 {
		t.Fatalf("activations = %d, want 1", len(store.activated))
	}
	committed := store.activated[0]
	if committed.ExpectedBindingGeneration != 7 {
		t.Fatalf("committed expected generation = %d, want 7", committed.ExpectedBindingGeneration)
	}
	if committed.ExpectedActiveVersionID == nil || *committed.ExpectedActiveVersionID != "version-1" {
		t.Fatalf("committed expected active version = %v, want version-1", committed.ExpectedActiveVersionID)
	}
	rmAssertRaw64(t, committed.EffectiveDigest, "committed effective digest")
	rmAssertRaw64(t, committed.CapabilityDigest, "committed capability digest")
	rmAssertApplyClass(t, committed.ApplyClass, "committed apply_class")
}

func TestRuntimeConfigurationStoreConflictBecomesConflictStatus(t *testing.T) {
	active := "version-1"
	store := &rmFakeConfigStore{
		binding: rmBinding(&active, 7),
		versions: map[string]RuntimeConfigurationVersionRecord{
			"version-1": rmStoredRecord(t, "version-1", 1, "model-a"),
			"version-2": rmStoredRecord(t, "version-2", 2, "model-b"),
		},
		activateErr: ErrRuntimeManagerConflict,
	}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	request := bindingActivationRequest{ExpectedActiveVersionID: expectedActive(&active), Reason: "race", ExpectedBindingGeneration: 7}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath+"/version-2/activate", request, true)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", recorder.Code)
	}
}

// ---------------------------------------------------------------------------
// Arbitrary target rollback
// ---------------------------------------------------------------------------

func TestRuntimeConfigurationRollbackTargetsAnyImmutableVersion(t *testing.T) {
	// Three versions exist and the third is active. Rolling back to the first
	// is the case the typed contract's single retained previous generation
	// cannot express, which is why rollback goes through the activation path.
	active := "version-3"
	store := &rmFakeConfigStore{
		binding: rmBinding(&active, 9),
		versions: map[string]RuntimeConfigurationVersionRecord{
			"version-1": rmStoredRecord(t, "version-1", 1, "model-a"),
			"version-2": rmStoredRecord(t, "version-2", 2, "model-b"),
			"version-3": rmStoredRecord(t, "version-3", 3, "model-b"),
		},
	}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	request := bindingRollbackRequest{
		ExpectedActiveVersionID:   expectedActive(&active),
		Reason:                    "revert_to_first_version",
		TargetVersionID:           "version-1",
		ExpectedBindingGeneration: 9,
	}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingRollbackPath, request, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}

	var result runtimeActivationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ActiveVersionID != "version-1" {
		t.Fatalf("active_version_id = %q, want version-1", result.ActiveVersionID)
	}
	if result.PreviousVersionID == nil || *result.PreviousVersionID != "version-3" {
		t.Fatalf("previous_version_id = %v, want version-3", result.PreviousVersionID)
	}
	rmAssertApplyClass(t, result.ApplyClass, "rollback apply_class")
	rmAssertRaw64(t, result.CapabilityDigest, "rollback capability_digest")

	if len(store.activated) != 1 {
		t.Fatalf("activations = %d, want 1", len(store.activated))
	}
	committed := store.activated[0]
	if committed.VersionID != "version-1" {
		t.Fatalf("committed version = %q, want version-1", committed.VersionID)
	}
	if committed.ExpectedBindingGeneration != 9 {
		t.Fatalf("committed expected generation = %d, want 9", committed.ExpectedBindingGeneration)
	}
	rmAssertApplyClass(t, committed.ApplyClass, "committed rollback apply_class")

	// A rollback still needs a target and a reason.
	missing := bindingRollbackRequest{ExpectedActiveVersionID: expectedActive(&active), Reason: "no_target", ExpectedBindingGeneration: 9}
	if recorder := rmRequest(t, router, http.MethodPost, rmBindingRollbackPath, missing, true); recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing target: status = %d, want 400", recorder.Code)
	}
}

func TestRuntimeConfigurationRollbackToUnknownVersionIsNotFound(t *testing.T) {
	active := "version-1"
	store := &rmFakeConfigStore{
		binding: rmBinding(&active, 1),
		versions: map[string]RuntimeConfigurationVersionRecord{
			"version-1": rmStoredRecord(t, "version-1", 1, "model-a"),
		},
	}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	request := bindingRollbackRequest{
		ExpectedActiveVersionID:   expectedActive(&active),
		Reason:                    "revert",
		TargetVersionID:           "version-absent",
		ExpectedBindingGeneration: 1,
	}
	recorder := rmRequest(t, router, http.MethodPost, rmBindingRollbackPath, request, true)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if len(store.activated) != 0 {
		t.Fatalf("an unresolvable rollback reached the store: %d activations", len(store.activated))
	}
}

// ---------------------------------------------------------------------------
// Standard scope
// ---------------------------------------------------------------------------

func TestRuntimeStandardVersionLifecycle(t *testing.T) {
	store := &rmFakeStandardStore{
		standard: RuntimeStandardRecord{ID: "standard-1", Name: "baseline", UpdatedAt: time.Unix(0, 0).UTC()},
		versions: map[string]RuntimeConfigurationVersionRecord{},
	}
	router := rmStandardAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	body := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "baseline"}
	recorder := rmRequest(t, router, http.MethodPost, "/api/runtime-standards/standard-1/versions", body, true)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create version: status = %d, want 201 (body %s)", recorder.Code, recorder.Body.String())
	}
	var version runtimeConfigurationVersionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &version); err != nil {
		t.Fatalf("decode version: %v", err)
	}
	rmAssertRaw64(t, version.ConfigurationDigest, "standard configuration_digest")
	rmAssertApplyClass(t, version.ApplyClass, "standard apply_class")
	if version.Validation == nil || !runtimeconfig.ValidDigest(version.Validation.CapabilityDigest) {
		t.Fatalf("standard version validation = %+v, want a raw-64 capability digest", version.Validation)
	}

	if len(store.created) != 1 {
		t.Fatalf("standard version writes = %d, want 1", len(store.created))
	}
	rmAssertRaw64(t, store.created[0].CapabilityDigest, "stored standard capability digest")
}

func TestRuntimeStandardActivationAndRollbackFenceOnActiveVersion(t *testing.T) {
	active := "standard-version-2"
	store := &rmFakeStandardStore{
		standard: RuntimeStandardRecord{ID: "standard-1", Name: "baseline", ActiveVersionID: &active, UpdatedAt: time.Unix(0, 0).UTC()},
		versions: map[string]RuntimeConfigurationVersionRecord{
			"standard-version-1": rmStoredRecord(t, "standard-version-1", 1, "model-a"),
			"standard-version-2": rmStoredRecord(t, "standard-version-2", 2, "model-b"),
		},
	}
	router := rmStandardAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	// A stale expectation is refused before any commit.
	wrong := "standard-version-1"
	stale := rollbackRequest{ExpectedActiveVersionID: expectedActive(&wrong), Reason: "stale", TargetVersionID: "standard-version-1"}
	if recorder := rmRequest(t, router, http.MethodPost, "/api/runtime-standards/standard-1/rollback", stale, true); recorder.Code != http.StatusConflict {
		t.Fatalf("stale expectation: status = %d, want 409", recorder.Code)
	}
	if len(store.activated) != 0 {
		t.Fatalf("a refused rollback reached the store: %d activations", len(store.activated))
	}

	// The correct expectation rolls back to an arbitrary earlier version.
	request := rollbackRequest{ExpectedActiveVersionID: expectedActive(&active), Reason: "revert", TargetVersionID: "standard-version-1"}
	recorder := rmRequest(t, router, http.MethodPost, "/api/runtime-standards/standard-1/rollback", request, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("rollback: status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
	var result runtimeActivationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ActiveVersionID != "standard-version-1" {
		t.Fatalf("active_version_id = %q, want standard-version-1", result.ActiveVersionID)
	}
	rmAssertApplyClass(t, result.ApplyClass, "standard rollback apply_class")
	rmAssertRaw64(t, result.CapabilityDigest, "standard rollback capability_digest")

	if len(store.activated) != 1 {
		t.Fatalf("activations = %d, want 1", len(store.activated))
	}
	if store.activated[0].VersionID != "standard-version-1" {
		t.Fatalf("committed version = %q, want standard-version-1", store.activated[0].VersionID)
	}
}

func TestRuntimeStandardCreateValidatesName(t *testing.T) {
	store := &rmFakeStandardStore{versions: map[string]RuntimeConfigurationVersionRecord{}}
	router := rmStandardAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})

	if recorder := rmRequest(t, router, http.MethodPost, "/api/runtime-standards", createRuntimeStandardRequest{Name: "  "}, true); recorder.Code != http.StatusBadRequest {
		t.Fatalf("blank name: status = %d, want 400", recorder.Code)
	}
	if recorder := rmRequest(t, router, http.MethodPost, "/api/runtime-standards", createRuntimeStandardRequest{Name: "baseline"}, false); recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency key: status = %d, want 400", recorder.Code)
	}

	recorder := rmRequest(t, router, http.MethodPost, "/api/runtime-standards", createRuntimeStandardRequest{Name: "baseline"}, true)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201 (body %s)", recorder.Code, recorder.Body.String())
	}
	var detail runtimeStandardDetail
	if err := json.Unmarshal(recorder.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.ActiveVersion != nil {
		t.Fatal("a new standard must have no active version")
	}
	if detail.Versions.Items == nil {
		t.Fatal("versions.items must be an empty array, not null")
	}
}

// ---------------------------------------------------------------------------
// Route table and error mapping
// ---------------------------------------------------------------------------

func TestRuntimeManagerRouteTablesAreWellFormed(t *testing.T) {
	configuration := NewRuntimeConfigurationAPI(&rmFakeConfigStore{}, &rmFakeCatalog{}, nil).Routes()
	standard := NewRuntimeStandardAPI(&rmFakeStandardStore{}, &rmFakeCatalog{}, nil).Routes()

	if len(configuration) == 0 || len(standard) == 0 {
		t.Fatal("a route table is empty")
	}
	seen := map[string]struct{}{}
	for _, route := range append(append([]RuntimeManagerRoute{}, configuration...), standard...) {
		if route.Handler == nil {
			t.Fatalf("route %s %s has no handler", route.Method, route.Pattern)
		}
		if !strings.HasPrefix(route.Pattern, "/api/") {
			t.Fatalf("route %s %s is not under /api/", route.Method, route.Pattern)
		}
		key := route.Method + " " + route.Pattern
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate route %s", key)
		}
		seen[key] = struct{}{}
	}
}

func TestRuntimeManagerErrorMappingHidesInternalDetail(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeRuntimeManagerError(recorder, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("connection string postgres://user:secret@host/db"))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "secret") || strings.Contains(recorder.Body.String(), "postgres://") {
		t.Fatalf("error body leaked internal detail: %s", recorder.Body.String())
	}

	for err, want := range map[error]int{
		ErrRuntimeManagerNotFound:                http.StatusNotFound,
		ErrRuntimeManagerConflict:                http.StatusConflict,
		ErrRuntimeManagerCapabilitiesUnavailable: http.StatusServiceUnavailable,
	} {
		recorder := httptest.NewRecorder()
		writeRuntimeManagerError(recorder, httptest.NewRequest(http.MethodGet, "/", nil), err)
		if recorder.Code != want {
			t.Fatalf("%v mapped to %d, want %d", err, recorder.Code, want)
		}
	}
}

func TestRuntimeManagerCASPresenceAndExplicitNull(t *testing.T) {
	target := rmStoredRecord(t, "version-1", 1, "model-a")
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1), versions: map[string]RuntimeConfigurationVersionRecord{"version-1": target}}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})
	path := rmBindingVersionsPath + "/version-1/activate"

	omitted := `{"reason":"initial","expected_binding_generation":1}`
	if recorder := rmRequest(t, router, http.MethodPost, path, omitted, true); recorder.Code != http.StatusBadRequest {
		t.Fatalf("omitted CAS status = %d, want 400", recorder.Code)
	}
	request := bindingActivationRequest{ExpectedActiveVersionID: expectedActive(nil), Reason: "initial", ExpectedBindingGeneration: 1}
	if recorder := rmRequest(t, router, http.MethodPost, path, request, true); recorder.Code != http.StatusOK {
		t.Fatalf("explicit null CAS status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
}

func TestRuntimeManagerStoredDigestIntegrityAndTrailingPayload(t *testing.T) {
	valid := rmStoredRecord(t, "version-1", 1, "model-a")
	for name, mutate := range map[string]func(*RuntimeConfigurationVersionRecord){
		"digest mismatch": func(record *RuntimeConfigurationVersionRecord) {
			record.Digest = strings.Repeat("0", runtimeconfig.DigestLength)
		},
		"trailing payload": func(record *RuntimeConfigurationVersionRecord) {
			record.Document = append(record.Document, []byte(`{}`)...)
		},
	} {
		record := valid
		mutate(&record)
		store := &rmFakeConfigStore{list: []RuntimeConfigurationVersionRecord{record}}
		recorder := rmRequest(t, rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()}), http.MethodGet, rmBindingVersionsPath, nil, false)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("%s status = %d, want 500", name, recorder.Code)
		}
		if strings.Contains(recorder.Body.String(), "model-a") {
			t.Fatalf("%s leaked document: %s", name, recorder.Body.String())
		}
	}
}

func TestRuntimeManagerSameDigestIsNoStoreNoOp(t *testing.T) {
	activeID := "version-1"
	active := rmStoredRecord(t, activeID, 1, "model-a")
	target := rmStoredRecord(t, "version-2", 2, "model-a")
	store := &rmFakeConfigStore{binding: rmBinding(&activeID, 4), versions: map[string]RuntimeConfigurationVersionRecord{active.ID: active, target.ID: target}}
	request := bindingActivationRequest{ExpectedActiveVersionID: expectedActive(&activeID), Reason: "same_digest", ExpectedBindingGeneration: 4}
	recorder := rmRequest(t, rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()}), http.MethodPost, rmBindingVersionsPath+"/version-2/activate", request, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
	if len(store.activated) != 0 {
		t.Fatalf("same digest caused %d store writes", len(store.activated))
	}
	var result runtimeActivationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ActiveVersionID != activeID || result.PreviousVersionID != nil || result.ApplyClass != "hot" || result.PendingAcknowledgement {
		t.Fatalf("no-op result = %+v", result)
	}
}

func TestRuntimeManagerValidateIsPureAndChecksIntegrity(t *testing.T) {
	record := rmStoredRecord(t, "version-1", 1, "model-a")
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1), versions: map[string]RuntimeConfigurationVersionRecord{record.ID: record}}
	recorder := rmRequest(t, rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()}), http.MethodPost, rmBindingVersionsPath+"/version-1/validate", nil, false)
	if recorder.Code != http.StatusOK {
		t.Fatalf("validation without idempotency key status = %d, want 200", recorder.Code)
	}
	validation := rmDecodeValidation(t, recorder)
	if !validation.Valid {
		t.Fatalf("validation = %+v, want valid", validation)
	}
}

func TestRuntimeManagerRejectsOversizedBodyAndUnsafeReason(t *testing.T) {
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	router := rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()})
	oversized := strings.Repeat(" ", maxRuntimeConfigurationBody+1)
	if recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, oversized, true); recorder.Code != http.StatusBadRequest {
		t.Fatalf("oversized body status = %d, want 400", recorder.Code)
	}
	unsafe := createConfigurationVersionRequest{Configuration: rmValidDocument("model-a"), Reason: "contains path/token"}
	if recorder := rmRequest(t, router, http.MethodPost, rmBindingVersionsPath, unsafe, true); recorder.Code != http.StatusBadRequest {
		t.Fatalf("unsafe reason status = %d, want 400", recorder.Code)
	}
	if len(store.created) != 0 {
		t.Fatalf("rejected request caused %d writes", len(store.created))
	}
}

func TestRuntimeManagerRecoversFromCapabilityDrift(t *testing.T) {
	activeID := "version-old"
	active := rmStoredRecord(t, activeID, 1, "model-b")
	target := rmStoredRecord(t, "version-new", 2, "model-a")
	capabilities := rmCapabilities()
	delete(capabilities.Models, runtimeconfig.ModelID("model-b"))
	store := &rmFakeConfigStore{binding: rmBinding(&activeID, 6), versions: map[string]RuntimeConfigurationVersionRecord{active.ID: active, target.ID: target}}
	request := bindingActivationRequest{ExpectedActiveVersionID: expectedActive(&activeID), Reason: "recover_drift", ExpectedBindingGeneration: 6}
	recorder := rmRequest(t, rmConfigAPI(store, &rmFakeCatalog{capabilities: capabilities}), http.MethodPost, rmBindingVersionsPath+"/version-new/activate", request, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
	if len(store.activated) != 1 {
		t.Fatalf("activations = %d, want 1", len(store.activated))
	}
}

func TestRuntimeStandardDetailPaginatesAndFetchesActiveOutsidePage(t *testing.T) {
	activeID := "version-active"
	pageRecord := rmStoredRecord(t, "version-page", 3, "model-a")
	activeRecord := rmStoredRecord(t, activeID, 1, "model-b")
	store := &rmFakeStandardStore{
		standard: RuntimeStandardRecord{ID: "standard-1", ActiveVersionID: &activeID},
		versions: map[string]RuntimeConfigurationVersionRecord{activeID: activeRecord},
		list:     []RuntimeConfigurationVersionRecord{pageRecord}, nextPage: "next-2",
	}
	recorder := rmRequest(t, rmStandardAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()}), http.MethodGet, "/api/runtime-standards/standard-1?limit=1&cursor=start", nil, false)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
	var detail runtimeStandardDetail
	if err := json.Unmarshal(recorder.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if store.listLimit != 1 || store.listCursor != "start" {
		t.Fatalf("page params = %d %q", store.listLimit, store.listCursor)
	}
	if len(detail.Versions.Items) != 1 || detail.Versions.NextCursor == nil || *detail.Versions.NextCursor != "next-2" {
		t.Fatalf("page = %+v", detail.Versions)
	}
	if detail.ActiveVersion == nil || detail.ActiveVersion.ID != activeID {
		t.Fatalf("active version = %+v", detail.ActiveVersion)
	}
}

func TestRuntimeManagerExpandedDTORoundTrip(t *testing.T) {
	capabilityDigest, err := runtimeconfig.CapabilityDigest(rmCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	document := rmValidDocument("model-a")
	document.Values.ProviderCatalogVersion = ptrTo("v1")
	document.Values.CapabilityDigest = &capabilityDigest
	document.Values.ReasoningMode = ptrTo("deliberate")
	document.Values.ReasoningBudget = ptrTo(int64(100))
	document.Values.Limits = &runtimeConfigurationLimits{MaxContextTokens: ptrTo(int64(10_000)), MaxTotalTokens: ptrTo(int64(8_000)), WallTimeoutMS: ptrTo(int64(1000))}
	document.Values.Concurrency = &runtimeconfig.ConcurrencyPolicy{MaxTasks: ptrTo(int64(2))}
	document.Values.Retry = &runtimeconfig.RetryPolicy{MaxAttempts: ptrTo(int64(2)), RetryableClasses: []string{"catalog_unavailable"}}
	document.Values.Flags = &runtimeconfig.FlagPolicy{Ordered: []string{"--quiet"}}
	document.Values.Environment = &runtimeconfig.EnvironmentPolicy{Entries: []runtimeconfig.EnvironmentEntry{{Key: "MODE", Value: ptrTo("safe")}}}
	document.Values.Permissions = &runtimeconfig.PermissionPolicy{ToolGrants: []string{"tool-a"}}
	document.Values.Fallback = &runtimeconfig.FallbackPolicy{Routes: []runtimeconfig.FallbackRoute{{RouteID: "route-b"}}}
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	body := createConfigurationVersionRequest{Configuration: document, Reason: "expanded_contract"}
	recorder := rmRequest(t, rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()}), http.MethodPost, rmBindingVersionsPath, body, true)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", recorder.Code, recorder.Body.String())
	}
	if len(store.created) != 1 {
		t.Fatalf("created = %d, want 1", len(store.created))
	}
	stored, _, err := materializeStoredVersion(RuntimeConfigurationVersionRecord{Document: store.created[0].Document, Digest: store.created[0].Digest})
	if err != nil {
		t.Fatalf("materialize created record: %v", err)
	}
	if stored.Values.Concurrency == nil || stored.Values.Environment == nil || stored.Values.Fallback == nil || stored.Values.CapabilityDigest == nil {
		t.Fatalf("expanded fields lost: %+v", stored.Values)
	}
}

func TestRuntimeConfigurationInvalidVersionIssueNamesWireField(t *testing.T) {
	store := &rmFakeConfigStore{binding: rmBinding(nil, 1)}
	document := rmValidDocument("model-a")
	document.Version = ""
	body := createConfigurationVersionRequest{Configuration: document, Reason: "invalid_version"}
	recorder := rmRequest(t, rmConfigAPI(store, &rmFakeCatalog{capabilities: rmCapabilities()}), http.MethodPost, rmBindingVersionsPath, body, true)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body %s)", recorder.Code, recorder.Body.String())
	}
	var envelope runtimeManagerErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != "invalid_configuration" || envelope.Error.Field != "version" {
		t.Fatalf("error = %+v, want invalid_configuration on version", envelope.Error)
	}
	if len(store.created) != 0 {
		t.Fatalf("invalid version caused %d writes", len(store.created))
	}
}
