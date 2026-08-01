package handler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestValidateCredentialRegistryOpaqueReference(t *testing.T) {
	valid := []string{
		"home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		"550e8400-e29b-41d4-a716-446655440000",
		"operation-ABC_123",
	}
	for _, ref := range valid {
		t.Run("valid_"+ref, func(t *testing.T) {
			if err := ValidateCredentialRegistryOpaqueReference(ref); err != nil {
				t.Fatalf("ValidateCredentialRegistryOpaqueReference(%q) = %v", ref, err)
			}
		})
	}

	invalid := []string{
		"",
		".",
		"..",
		"/home/user/.config/provider",
		`C:\Users\owner\credentials`,
		"home/ref",
		"home.ref",
		"home ref",
		"home%2Fref",
		"https://catalog.invalid/ref",
		"opaque\nreference",
		"opaque\x00reference",
		"référence",
		strings.Repeat("a", credentialRegistryOpaqueReferenceMaxLength+1),
	}
	for _, ref := range invalid {
		t.Run("invalid", func(t *testing.T) {
			err := ValidateCredentialRegistryOpaqueReference(ref)
			if err == nil {
				t.Fatalf("ValidateCredentialRegistryOpaqueReference(%q) unexpectedly succeeded", ref)
			}
			if strings.Contains(err.Error(), ref) && ref != "" {
				t.Fatalf("validation error reflected rejected reference %q: %v", ref, err)
			}
		})
	}
}

func TestRedactCredentialRegistryOpaqueReferenceIsConstant(t *testing.T) {
	refs := []string{
		"home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		"/sensitive/source/path",
		"provider-account@example.invalid",
	}
	for _, ref := range refs {
		got := RedactCredentialRegistryOpaqueReference(ref)
		if got != credentialRegistryRedactedReference {
			t.Fatalf("redaction = %q, want constant marker %q", got, credentialRegistryRedactedReference)
		}
		if strings.Contains(got, ref) {
			t.Fatalf("redaction leaked input %q", ref)
		}
	}

	request := CredentialHomeAssignmentRequest{
		HomeRef:                   refs[0],
		ExpectedBindingGeneration: 7,
		ExpectedCatalogGeneration: 11,
	}
	redacted := request.Redacted()
	if redacted.HomeRef != credentialRegistryRedactedReference {
		t.Fatalf("Redacted().HomeRef = %q", redacted.HomeRef)
	}
	if request.HomeRef != refs[0] {
		t.Fatal("Redacted mutated the source request")
	}
}

func TestCredentialRegistryDTOsArePathless(t *testing.T) {
	cases := []struct {
		name string
		v    any
		keys []string
	}{
		{
			name: "home summary",
			v: CredentialHomeSummary{
				HomeRef:           "home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
				CatalogGeneration: 12,
				Provider:          "codex",
				State:             "healthy",
				Health:            "fresh",
			},
			keys: []string{"catalog_generation", "health", "home_ref", "provider", "state"},
		},
		{
			name: "assignment request",
			v: CredentialHomeAssignmentRequest{
				HomeRef:                   "home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
				ExpectedBindingGeneration: 7,
				ExpectedCatalogGeneration: 12,
			},
			keys: []string{"expected_binding_generation", "expected_catalog_generation", "home_ref"},
		},
		{
			name: "release request",
			v: CredentialHomeReleaseRequest{
				ExpectedBindingGeneration: 7,
				Drain:                     true,
			},
			keys: []string{"drain", "expected_binding_generation"},
		},
		{
			name: "daemon reconciliation report",
			v: CredentialCatalogReconciliationReportRequest{
				DaemonID:           "daemon_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
				PreviousGeneration: 11,
				Generation:         12,
				ScanKind:           "full",
				Counters:           map[string]int64{"healthy": 3},
				Digest:             strings.Repeat("a", 64),
			},
			keys: []string{"counters", "daemon_id", "digest", "generation", "previous_generation", "scan_kind"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.v)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			got := make([]string, 0, len(fields))
			for key := range fields {
				got = append(got, key)
				lower := strings.ToLower(key)
				for _, forbidden := range []string{"path", "filesystem", "inode", "device", "account", "credential", "environment", "argv", "prompt", "response_body"} {
					if strings.Contains(lower, forbidden) {
						t.Fatalf("DTO field %q contains forbidden disclosure category %q", key, forbidden)
					}
				}
			}
			slicesSort(got)
			slicesSort(tc.keys)
			if !reflect.DeepEqual(got, tc.keys) {
				t.Fatalf("JSON fields = %v, want %v; body=%s", got, tc.keys, body)
			}
		})
	}
}

func TestNewCredentialRegistryErrorCanonicalMappings(t *testing.T) {
	cases := []struct {
		code   CredentialRegistryErrorCode
		status int
	}{
		{CredentialRegistryInvalidArgument, 400},
		{CredentialRegistryUnauthenticated, 401},
		{CredentialRegistryForbidden, 403},
		{CredentialRegistryNotFound, 404},
		{CredentialRegistryVersionConflict, 409},
		{CredentialRegistryGenerationConflict, 409},
		{CredentialRegistryExclusiveAssignmentConflict, 409},
		{CredentialRegistryActiveReference, 409},
		{CredentialRegistryCapabilityUnsupported, 412},
		{CredentialRegistryHealthStale, 412},
		{CredentialRegistryDriftDetected, 412},
		{CredentialRegistryInvalidConfiguration, 422},
		{CredentialRegistryCapacityExhausted, 429},
		{CredentialRegistryCatalogUnavailable, 503},
		{CredentialRegistryDaemonNotReady, 503},
	}

	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			status, envelope, ok := NewCredentialRegistryError(tc.code, "home_ref", "request_ABC-123", true)
			if !ok {
				t.Fatal("canonical code was rejected")
			}
			if status != tc.status {
				t.Fatalf("status = %d, want %d", status, tc.status)
			}
			if envelope.Error.Code != tc.code || envelope.Error.Field != "home_ref" || envelope.Error.RequestID != "request_ABC-123" || !envelope.Error.Retryable {
				t.Fatalf("unexpected envelope: %+v", envelope)
			}
			if envelope.Error.Message == "" {
				t.Fatal("canonical message is empty")
			}
		})
	}
}

func TestNewCredentialRegistryErrorNeverReflectsUnsafeMetadata(t *testing.T) {
	status, envelope, ok := NewCredentialRegistryError(
		CredentialRegistryInvalidArgument,
		"/source/home/path",
		"request/../../credential",
		false,
	)
	if !ok || status != 400 {
		t.Fatalf("status=%d ok=%v", status, ok)
	}
	if envelope.Error.Field != "" {
		t.Fatalf("unsafe field was reflected: %q", envelope.Error.Field)
	}
	if envelope.Error.RequestID != credentialRegistryUnavailableRequestID {
		t.Fatalf("unsafe request ID was reflected: %q", envelope.Error.RequestID)
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"/source/home/path", "request/../../credential"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("error envelope leaked %q: %s", forbidden, body)
		}
	}

	if _, _, ok := NewCredentialRegistryError("invented_code", "", "request_ABC-123", false); ok {
		t.Fatal("unknown error code must fail closed")
	}
}

func TestCredentialRegistryDTOValidation(t *testing.T) {
	actor := CredentialRegistryActor{Type: "human", ID: "actor_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"}
	if err := actor.Validate(); err != nil {
		t.Fatalf("valid actor: %v", err)
	}
	actor.ID = "/local/actor"
	if err := actor.Validate(); err == nil || strings.Contains(err.Error(), actor.ID) {
		t.Fatalf("invalid actor error must be safe, got %v", err)
	}

	validSummary := CredentialHomeSummary{
		HomeRef:           "home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		CatalogGeneration: 9,
		Provider:          "codex",
		State:             "healthy",
		Health:            "fresh",
	}
	if err := validSummary.Validate(); err != nil {
		t.Fatalf("valid summary: %v", err)
	}
	validList := CredentialHomeListResponse{
		Items:      []CredentialHomeSummary{validSummary},
		NextCursor: "cursor_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
	}
	if err := validList.Validate(); err != nil {
		t.Fatalf("valid list: %v", err)
	}
	invalidList := validList
	invalidList.NextCursor = "/catalog/page/2"
	if err := invalidList.Validate(); err == nil || strings.Contains(err.Error(), invalidList.NextCursor) {
		t.Fatalf("invalid list error must be safe, got %v", err)
	}
	invalidList = validList
	invalidList.Items = append([]CredentialHomeSummary(nil), validList.Items...)
	invalidList.Items[0].Provider = "provider/account"
	if err := invalidList.Validate(); err == nil || strings.Contains(err.Error(), invalidList.Items[0].Provider) {
		t.Fatalf("invalid summary error must be safe, got %v", err)
	}

	validAssignment := CredentialHomeAssignmentRequest{
		HomeRef:                   "home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		ExpectedBindingGeneration: 4,
		ExpectedCatalogGeneration: 9,
	}
	if err := validAssignment.Validate(); err != nil {
		t.Fatalf("valid assignment: %v", err)
	}

	invalidAssignment := validAssignment
	invalidAssignment.HomeRef = "/daemon/local/home"
	if err := invalidAssignment.Validate(); err == nil || strings.Contains(err.Error(), invalidAssignment.HomeRef) {
		t.Fatalf("invalid assignment error must be safe, got %v", err)
	}

	validReport := CredentialCatalogReconciliationReportRequest{
		DaemonID:           "daemon_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		PreviousGeneration: 8,
		Generation:         9,
		ScanKind:           "full",
		Counters:           map[string]int64{"candidate": 2, "healthy": 7},
		Digest:             strings.Repeat("b", 64),
	}
	if err := validReport.Validate(); err != nil {
		t.Fatalf("valid report: %v", err)
	}

	invalidReports := []CredentialCatalogReconciliationReportRequest{
		func() CredentialCatalogReconciliationReportRequest {
			r := validReport
			r.DaemonID = "../daemon"
			return r
		}(),
		func() CredentialCatalogReconciliationReportRequest { r := validReport; r.Generation = 8; return r }(),
		func() CredentialCatalogReconciliationReportRequest {
			r := validReport
			r.ScanKind = "full/scan"
			return r
		}(),
		func() CredentialCatalogReconciliationReportRequest {
			r := validReport
			r.Counters = map[string]int64{"/path": 1}
			return r
		}(),
		func() CredentialCatalogReconciliationReportRequest {
			r := validReport
			r.Counters = map[string]int64{"healthy": -1}
			return r
		}(),
		func() CredentialCatalogReconciliationReportRequest { r := validReport; r.Digest = " "; return r }(),
		func() CredentialCatalogReconciliationReportRequest {
			r := validReport
			r.Digest = strings.Repeat("g", 64)
			return r
		}(),
	}
	for i, report := range invalidReports {
		if err := report.Validate(); err == nil {
			t.Fatalf("invalid report %d unexpectedly validated", i)
		}
	}
}

type credentialRegistryGuardStub struct{}

func (credentialRegistryGuardStub) RequireOwner(context.Context, CredentialRegistryActor) error {
	return nil
}

func (credentialRegistryGuardStub) RequireOwnerOrWorkspaceAdmin(context.Context, CredentialRegistryActor, string) error {
	return nil
}

func (credentialRegistryGuardStub) RequireWorkspace(context.Context, CredentialRegistryActor, string) error {
	return nil
}

func (credentialRegistryGuardStub) RequireDaemonWorkspace(context.Context, CredentialRegistryActor, string, string) error {
	return errors.New("denied")
}

func TestCredentialRegistryGuardInterfacesRemainDependencySafe(t *testing.T) {
	var actorGuard CredentialRegistryActorGuard = credentialRegistryGuardStub{}
	var workspaceGuard CredentialRegistryWorkspaceGuard = credentialRegistryGuardStub{}
	actor := CredentialRegistryActor{Type: "human", ID: "actor_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"}

	if err := actorGuard.RequireOwner(context.Background(), actor); err != nil {
		t.Fatal(err)
	}
	if err := actorGuard.RequireOwnerOrWorkspaceAdmin(context.Background(), actor, "workspace_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceGuard.RequireWorkspace(context.Background(), actor, "workspace_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceGuard.RequireDaemonWorkspace(context.Background(), actor, "daemon_01JQ9S7K7Z2J9Q8VB4DMCN5F6A", "workspace_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"); err == nil {
		t.Fatal("stub should prove guard denials propagate without handler or DB dependencies")
	}
}

func slicesSort(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func TestCredentialRegistryDigestRequiresRawLowercase64Hex(t *testing.T) {
	valid := CredentialCatalogReconciliationReportRequest{
		DaemonID:           "daemon_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		PreviousGeneration: 1,
		Generation:         2,
		ScanKind:           "full",
		Counters:           map[string]int64{"healthy": 1},
		Digest:             strings.Repeat("a", 64),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("raw lowercase digest rejected: %v", err)
	}

	invalid := []string{
		"sha256:" + strings.Repeat("a", 64),
		strings.Repeat("A", 64),
		strings.Repeat("a", 63),
		strings.Repeat("a", 65),
		strings.Repeat("g", 64),
		strings.Repeat("a", 63) + " ",
	}
	for _, digest := range invalid {
		report := valid
		report.Digest = digest
		if err := report.Validate(); err == nil {
			t.Fatalf("non-canonical digest of length %d unexpectedly validated", len(digest))
		} else if strings.Contains(err.Error(), digest) {
			t.Fatalf("digest validation reflected rejected input: %v", err)
		}
	}
}

func TestCredentialRegistryGenerationsMustBeNonzero(t *testing.T) {
	validSummary := CredentialHomeSummary{
		HomeRef:           "home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		CatalogGeneration: 1,
		Provider:          "codex",
		State:             "healthy",
		Health:            "fresh",
	}
	validAssignment := CredentialHomeAssignmentRequest{
		HomeRef:                   validSummary.HomeRef,
		ExpectedBindingGeneration: 1,
		ExpectedCatalogGeneration: 1,
	}
	validReport := CredentialCatalogReconciliationReportRequest{
		DaemonID:           "daemon_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		PreviousGeneration: 1,
		Generation:         2,
		ScanKind:           "full",
		Counters:           map[string]int64{"healthy": 1},
		Digest:             strings.Repeat("b", 64),
	}

	cases := []struct {
		name     string
		validate func() error
	}{
		{name: "summary catalog", validate: func() error { r := validSummary; r.CatalogGeneration = 0; return r.Validate() }},
		{name: "assignment binding", validate: func() error { r := validAssignment; r.ExpectedBindingGeneration = 0; return r.Validate() }},
		{name: "assignment catalog", validate: func() error { r := validAssignment; r.ExpectedCatalogGeneration = 0; return r.Validate() }},
		{name: "release binding", validate: func() error { return (CredentialHomeReleaseRequest{}).Validate() }},
		{name: "reconciliation", validate: func() error {
			return (CredentialHomeReconciliationResponse{OperationID: "operation_1", Counters: map[string]int64{"healthy": 1}}).Validate()
		}},
		{name: "report previous", validate: func() error { r := validReport; r.PreviousGeneration = 0; return r.Validate() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.validate(); err == nil {
				t.Fatal("zero generation unexpectedly validated")
			}
		})
	}
}

func TestCredentialRegistryPaginationAndCardinalityBounds(t *testing.T) {
	omitted := CredentialHomeListRequest{}
	if err := omitted.Validate(); err != nil {
		t.Fatalf("omitted limit: %v", err)
	}
	if got := omitted.EffectiveLimit(); got != credentialRegistryDefaultPageLimit {
		t.Fatalf("default effective limit = %d, want %d", got, credentialRegistryDefaultPageLimit)
	}
	maximum := CredentialHomeListRequest{Limit: credentialRegistryMaxPageLimit}
	if err := maximum.Validate(); err != nil {
		t.Fatalf("maximum limit: %v", err)
	}
	if got := maximum.EffectiveLimit(); got != credentialRegistryMaxPageLimit {
		t.Fatalf("maximum effective limit = %d", got)
	}
	for _, limit := range []int{-1, credentialRegistryMaxPageLimit + 1} {
		if err := (CredentialHomeListRequest{Limit: limit}).Validate(); err == nil {
			t.Fatalf("out-of-range limit %d unexpectedly validated", limit)
		}
	}

	item := CredentialHomeSummary{
		HomeRef:           "home_01JQ9S7K7Z2J9Q8VB4DMCN5F6A",
		CatalogGeneration: 1,
		Provider:          "codex",
		State:             "healthy",
		Health:            "fresh",
	}
	atLimit := CredentialHomeListResponse{Items: make([]CredentialHomeSummary, credentialRegistryMaxPageLimit)}
	for i := range atLimit.Items {
		atLimit.Items[i] = item
	}
	if err := atLimit.Validate(); err != nil {
		t.Fatalf("response at item limit: %v", err)
	}
	overLimit := atLimit
	overLimit.Items = append(append([]CredentialHomeSummary(nil), atLimit.Items...), item)
	if err := overLimit.Validate(); err == nil {
		t.Fatal("over-cardinality response unexpectedly validated")
	}

	atCounterLimit := make(map[string]int64, credentialRegistryMaxCounterCardinality)
	for i := 0; i < credentialRegistryMaxCounterCardinality; i++ {
		atCounterLimit[credentialRegistryCounterName(i)] = int64(i)
	}
	if err := validateCredentialRegistryCounters(atCounterLimit); err != nil {
		t.Fatalf("counters at cardinality limit: %v", err)
	}
	overCounterLimit := make(map[string]int64, credentialRegistryMaxCounterCardinality+1)
	for key, value := range atCounterLimit {
		overCounterLimit[key] = value
	}
	overCounterLimit[credentialRegistryCounterName(credentialRegistryMaxCounterCardinality)] = 1
	if err := validateCredentialRegistryCounters(overCounterLimit); err == nil {
		t.Fatal("over-cardinality counters unexpectedly validated")
	}
	if err := validateCredentialRegistryCounters(map[string]int64{}); err == nil {
		t.Fatal("empty counters unexpectedly validated")
	}
}

func credentialRegistryCounterName(index int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	return "counter_" + string([]byte{alphabet[(index/len(alphabet))%len(alphabet)], alphabet[index%len(alphabet)]})
}

type credentialRegistryBoundaryGuard struct {
	actorErr          error
	workspaceErr      error
	daemonErr         error
	actorCalls        int
	workspaceCalls    int
	daemonCalls       int
	lastWorkspaceID   string
	lastDaemonID      string
	lastDaemonActorID string
}

func (g *credentialRegistryBoundaryGuard) RequireOwner(context.Context, CredentialRegistryActor) error {
	return g.actorErr
}

func (g *credentialRegistryBoundaryGuard) RequireOwnerOrWorkspaceAdmin(_ context.Context, _ CredentialRegistryActor, workspaceID string) error {
	g.actorCalls++
	g.lastWorkspaceID = workspaceID
	return g.actorErr
}

func (g *credentialRegistryBoundaryGuard) RequireWorkspace(_ context.Context, _ CredentialRegistryActor, workspaceID string) error {
	g.workspaceCalls++
	g.lastWorkspaceID = workspaceID
	return g.workspaceErr
}

func (g *credentialRegistryBoundaryGuard) RequireDaemonWorkspace(_ context.Context, actor CredentialRegistryActor, daemonID, workspaceID string) error {
	g.daemonCalls++
	g.lastDaemonID = daemonID
	g.lastDaemonActorID = actor.ID
	g.lastWorkspaceID = workspaceID
	return g.daemonErr
}

// assertCredentialRegistryDenialIsOpaque proves a guard denial was collapsed to
// the constant forbidden sentinel before leaving the boundary.
//
// The assertion compares against the sentinel text itself instead of hand-picked
// substrings. Hand-picked substrings are unsound here: the sentinel legitimately
// reads "credential registry access denied", so probing for a word like
// "credential" flags correctly redacted output as a leak. Requiring the message
// to equal the sentinel verbatim is strictly stronger, since any guard detail
// that survived would change the message. Guard-detail tokens are then checked
// individually, skipping tokens the sentinel itself contains.
func assertCredentialRegistryDenialIsOpaque(t *testing.T, err error, guardErr error) {
	t.Helper()
	if !errors.Is(err, ErrCredentialRegistryForbidden) {
		t.Fatalf("guard denial error = %v, want %v", err, ErrCredentialRegistryForbidden)
	}
	sentinel := ErrCredentialRegistryForbidden.Error()
	if err.Error() != sentinel {
		t.Fatalf("guard denial message = %q, want constant sentinel %q", err.Error(), sentinel)
	}
	for _, token := range strings.Fields(guardErr.Error()) {
		if strings.Contains(sentinel, token) {
			continue
		}
		if strings.Contains(err.Error(), token) {
			t.Fatalf("guard denial leaked detail token %q: %v", token, err)
		}
	}
}

func TestCredentialRegistryBoundaryEnforcesHumanAndTenantConfinement(t *testing.T) {
	if _, err := NewCredentialRegistryBoundary(nil, &credentialRegistryBoundaryGuard{}); err == nil {
		t.Fatal("nil actor guard unexpectedly accepted")
	}
	if _, err := NewCredentialRegistryBoundary(&credentialRegistryBoundaryGuard{}, nil); err == nil {
		t.Fatal("nil workspace guard unexpectedly accepted")
	}

	guard := &credentialRegistryBoundaryGuard{}
	boundary, err := NewCredentialRegistryBoundary(guard, guard)
	if err != nil {
		t.Fatal(err)
	}
	actor := CredentialRegistryActor{Type: "human", ID: "actor_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"}
	workspaceID := "workspace_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"
	if err := boundary.AuthorizeWorkspace(context.Background(), actor, workspaceID); err != nil {
		t.Fatalf("authorized human rejected: %v", err)
	}
	if guard.actorCalls != 1 || guard.workspaceCalls != 1 || guard.lastWorkspaceID != workspaceID {
		t.Fatalf("guards not both applied to the same workspace: %+v", guard)
	}

	guard.actorCalls, guard.workspaceCalls = 0, 0
	task := CredentialRegistryActor{Type: "task", ID: "task_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"}
	if err := boundary.AuthorizeWorkspace(context.Background(), task, workspaceID); !errors.Is(err, ErrCredentialRegistryForbidden) {
		t.Fatalf("task actor error = %v", err)
	}
	if guard.actorCalls != 0 || guard.workspaceCalls != 0 {
		t.Fatal("task actor reached authorization guards")
	}

	guard.actorErr = errors.New("provider account /secret/owner path")
	assertCredentialRegistryDenialIsOpaque(t, boundary.AuthorizeWorkspace(context.Background(), actor, workspaceID), guard.actorErr)
	if guard.workspaceCalls != 0 {
		t.Fatal("workspace guard ran after actor denial")
	}

	guard.actorErr = nil
	guard.workspaceErr = errors.New("cross-workspace /tenant/private")
	assertCredentialRegistryDenialIsOpaque(t, boundary.AuthorizeWorkspace(context.Background(), actor, workspaceID), guard.workspaceErr)

	malformed := CredentialRegistryActor{Type: "human", ID: "/local/identity"}
	if err := boundary.AuthorizeWorkspace(context.Background(), malformed, workspaceID); !errors.Is(err, ErrCredentialRegistryUnauthenticated) {
		t.Fatalf("malformed actor error = %v", err)
	}
	if err := boundary.AuthorizeWorkspace(nil, actor, workspaceID); !errors.Is(err, ErrCredentialRegistryUnauthenticated) {
		t.Fatalf("nil context error = %v", err)
	}
}

func TestCredentialRegistryBoundaryConfinesDaemonReports(t *testing.T) {
	guard := &credentialRegistryBoundaryGuard{}
	boundary, err := NewCredentialRegistryBoundary(guard, guard)
	if err != nil {
		t.Fatal(err)
	}
	actor := CredentialRegistryActor{Type: "daemon", ID: "daemon_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"}
	workspaceID := "workspace_01JQ9S7K7Z2J9Q8VB4DMCN5F6A"
	if err := boundary.AuthorizeDaemonReport(context.Background(), actor, actor.ID, workspaceID); err != nil {
		t.Fatalf("authorized daemon rejected: %v", err)
	}
	if guard.daemonCalls != 1 || guard.lastDaemonID != actor.ID || guard.lastDaemonActorID != actor.ID || guard.lastWorkspaceID != workspaceID {
		t.Fatalf("daemon guard did not receive confined identity: %+v", guard)
	}

	guard.daemonCalls = 0
	if err := boundary.AuthorizeDaemonReport(context.Background(), actor, "daemon_other", workspaceID); !errors.Is(err, ErrCredentialRegistryForbidden) {
		t.Fatalf("daemon identity mismatch error = %v", err)
	}
	if guard.daemonCalls != 0 {
		t.Fatal("mismatched daemon identity reached workspace guard")
	}

	guard.daemonErr = errors.New("credential path /home/daemon/.config")
	assertCredentialRegistryDenialIsOpaque(t, boundary.AuthorizeDaemonReport(context.Background(), actor, actor.ID, workspaceID), guard.daemonErr)

	human := CredentialRegistryActor{Type: "human", ID: actor.ID}
	if err := boundary.AuthorizeDaemonReport(context.Background(), human, human.ID, workspaceID); !errors.Is(err, ErrCredentialRegistryForbidden) {
		t.Fatalf("human daemon-report error = %v", err)
	}
}
