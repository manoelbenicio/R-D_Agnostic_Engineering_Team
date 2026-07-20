package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestProjectOmniRouteModelsProducesRegistryAcceptedEnrichedSnapshot(t *testing.T) {
	// Native OmniRoute (OpenAI-basic) shape with the approved route plus
	// non-approved rows — mirrors the real /v1/models catalog.
	native := OmniRouteNativeModels{
		Object: "list",
		Data: []OmniRouteNativeModel{
			{ID: "gpt-4o", Object: "model", OwnedBy: "openai"},
			{ID: "claude_code_kimi_2.7_Code", Object: "model", OwnedBy: "anthropic"},
			{ID: "gemini-2.5-pro", Object: "model", OwnedBy: "google"},
		},
	}
	doc := ProjectOmniRouteModels(native, "omni-native-v1")

	// The whole point: the enriched projection must be accepted by the registry
	// parser that previously fail-closed-rejected the raw native shape.
	snap, err := buildSnapshot(doc, time.Now())
	if err != nil {
		t.Fatalf("buildSnapshot rejected the projected document: %v", err)
	}
	if snap.Version != "omni-native-v1" {
		t.Fatalf("registry version mismatch: %q", snap.Version)
	}
	if len(snap.Models) != 3 {
		t.Fatalf("expected 3 projected rows, got %d", len(snap.Models))
	}

	// Approved row: available + frozen Anthropic/Claude profile (not inferred).
	approved := snap.Models[brain.RouteModel(approvedProjectionRouteModel)]
	if !approved.Available {
		t.Fatal("approved route must be available")
	}
	if approved.Capability.Protocol != brain.ProtocolAnthropicMessages {
		t.Fatalf("approved protocol=%q want anthropic-messages", approved.Capability.Protocol)
	}
	if approved.Capability.ContextLimit != approvedProjectionContextLimit ||
		!approved.Capability.Streaming || !approved.Capability.Tools ||
		approved.Capability.Reasoning || approved.Capability.StructuredOutput {
		t.Fatalf("approved capability defaults mismatch: %+v", approved.Capability)
	}
	if approved.Rotation != RotationStrictIndependentRequest || approved.Affinity != AffinityOriginAccount {
		t.Fatalf("approved rotation/affinity mismatch: %q/%q", approved.Rotation, approved.Affinity)
	}
	if approved.AccountPool != projectionAccountPool {
		t.Fatalf("approved account_pool mismatch: %q", approved.AccountPool)
	}

	// Non-approved rows: present (no whole-snapshot rejection) but NOT available.
	for _, id := range []string{"gpt-4o", "gemini-2.5-pro"} {
		row := snap.Models[brain.RouteModel(id)]
		if row.Available {
			t.Fatalf("non-approved row %q must be available=false", id)
		}
	}
}

func TestProjectOmniRouteModelsPreservesFailClosedSelection(t *testing.T) {
	native := OmniRouteNativeModels{Data: []OmniRouteNativeModel{
		{ID: "claude_code_kimi_2.7_Code"},
		{ID: "gpt-4o"},
	}}
	doc := ProjectOmniRouteModels(native, "v1")

	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		return doc, nil
	}), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// Approved route resolves.
	if _, err := registry.LookupModel(context.Background(), brain.RouteModel(approvedProjectionRouteModel)); err != nil {
		t.Fatalf("approved route should resolve: %v", err)
	}
	// Non-approved route is present-but-unavailable -> fail closed.
	if _, err := registry.LookupModel(context.Background(), brain.RouteModel("gpt-4o")); !IsErrorClass(err, ErrorUnknownModel) {
		t.Fatalf("non-approved route must fail closed with unknown_model, got %v", err)
	}
	// Capability validation for the approved route succeeds on the frozen protocol.
	if err := registry.ValidateCapability(context.Background(), brain.RouteModel(approvedProjectionRouteModel), CapabilityRequirement{
		Protocol: brain.ProtocolAnthropicMessages, Streaming: true, Tools: true,
	}); err != nil {
		t.Fatalf("approved capability validation failed: %v", err)
	}
}

func TestProjectOmniRouteModelsSkipsInvalidNativeRows(t *testing.T) {
	// Missing/blank id and an unparseable (whitespace) id must be skipped, not
	// fail-closed-reject the whole snapshot.
	native := OmniRouteNativeModels{Data: []OmniRouteNativeModel{
		{ID: ""},
		{ID: "  "},
		{ID: "bad id with spaces"},
		{ID: "claude_code_kimi_2.7_Code"},
		{ID: "claude_code_kimi_2.7_Code"}, // duplicate -> deduped
	}}
	doc := ProjectOmniRouteModels(native, "")
	if doc.RegistryVersion != projectionRegistryVersionDefault {
		t.Fatalf("empty version should default, got %q", doc.RegistryVersion)
	}
	snap, err := buildSnapshot(doc, time.Now())
	if err != nil {
		t.Fatalf("buildSnapshot rejected doc with skipped rows: %v", err)
	}
	if len(snap.Models) != 1 {
		t.Fatalf("expected exactly 1 valid deduped row, got %d", len(snap.Models))
	}
	if !snap.Models[brain.RouteModel(approvedProjectionRouteModel)].Available {
		t.Fatal("approved route missing after skipping invalid rows")
	}
}

func TestProjectOmniRouteModelsWithoutApprovedRowIsFullyFailClosed(t *testing.T) {
	// If the approved route is absent from the catalog, nothing is selectable.
	native := OmniRouteNativeModels{Data: []OmniRouteNativeModel{
		{ID: "gpt-4o"}, {ID: "gemini-2.5-pro"},
	}}
	snap, err := buildSnapshot(ProjectOmniRouteModels(native, "v1"), time.Now())
	if err != nil {
		t.Fatalf("buildSnapshot rejected: %v", err)
	}
	for id, spec := range snap.Models {
		if spec.Available {
			t.Fatalf("no row should be available without the approved route; %q was", id)
		}
	}
}
