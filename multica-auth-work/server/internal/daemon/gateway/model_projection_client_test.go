package gateway

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const nativeOmniRouteModelsBody = `{"object":"list","data":[{"id":"gpt-4o","object":"model","owned_by":"openai"},{"id":"claude_code_kimi_2.7_Code","object":"model","owned_by":"anthropic"}]}`

func nativeModelsTestClient(t *testing.T, body, registryVersion string) *Client {
	t.Helper()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := syntheticResponse(req, http.StatusOK, "application/json", body)
		if registryVersion != "" {
			resp.Header.Set(HeaderRegistryVersion, registryVersion)
		}
		return resp, nil
	})
	client, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:     &syntheticCredentialSource{},
		HTTPClient:     &http.Client{Transport: transport},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestFetchModelsProjectsNativeShapeWhenDevFlagSet(t *testing.T) {
	t.Setenv(envDevModelsCompat, "1")
	client := nativeModelsTestClient(t, nativeOmniRouteModelsBody, "omni-native-v1")

	doc, err := client.FetchModels(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if doc.RegistryVersion != "omni-native-v1" {
		t.Fatalf("registry version=%q want omni-native-v1", doc.RegistryVersion)
	}
	// The projected document must be accepted by the registry parser that
	// previously fail-closed-rejected the raw native shape.
	snap, err := buildSnapshot(doc, time.Now())
	if err != nil {
		t.Fatalf("projected native doc rejected by buildSnapshot: %v", err)
	}
	if !snap.Models[brain.RouteModel(approvedProjectionRouteModel)].Available {
		t.Fatal("approved route must be available after projection")
	}
	if snap.Models[brain.RouteModel("gpt-4o")].Available {
		t.Fatal("non-approved route must be available=false")
	}
}

func TestFetchModelsDoesNotProjectNativeShapeWhenDevFlagAbsent(t *testing.T) {
	t.Setenv(envDevModelsCompat, "") // deterministic: flag explicitly off
	// Flag absent (default): a native shape must NOT be enriched — returned
	// as-is, preserving the pre-existing enriched-schema contract (the registry
	// will then reject it downstream, which is the unchanged behavior).
	client := nativeModelsTestClient(t, nativeOmniRouteModelsBody, "")
	doc, err := client.FetchModels(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(doc.Models) != 2 {
		t.Fatalf("expected raw passthrough of 2 rows, got %d", len(doc.Models))
	}
	for _, m := range doc.Models {
		if m.Streaming != nil || m.Available != nil || m.Rotation != "" || m.AccountPool != "" {
			t.Fatalf("row %q was enriched despite DEV flag absent: %+v", m.ID, m)
		}
	}
}

func TestFetchModelsEnrichedDocumentPassthroughWhenDevFlagAbsent(t *testing.T) {
	t.Setenv(envDevModelsCompat, "") // deterministic: flag explicitly off
	enriched := `{"object":"list","registry_version":"enr-v1","data":[{"id":"claude_code_kimi_2.7_Code","protocol":"anthropic-messages","streaming":true,"tools":true,"reasoning":false,"structured_output":false,"context_limit":200000,"account_pool":"default","rotation":"strict-independent-request-round-robin","affinity":"origin-account","available":true}]}`
	client := nativeModelsTestClient(t, enriched, "")
	doc, err := client.FetchModels(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if doc.RegistryVersion != "enr-v1" || len(doc.Models) != 1 || doc.Models[0].Rotation != string(RotationStrictIndependentRequest) {
		t.Fatalf("enriched passthrough altered the document: %+v", doc)
	}
	if _, err := buildSnapshot(doc, time.Now()); err != nil {
		t.Fatalf("enriched passthrough doc rejected: %v", err)
	}
}

func TestFetchModelsProjectionFailsClosedWithoutApprovedRoute(t *testing.T) {
	t.Setenv(envDevModelsCompat, "1")
	body := `{"object":"list","data":[{"id":"gpt-4o"},{"id":"gemini-2.5-pro"}]}`
	client := nativeModelsTestClient(t, body, "v1")
	doc, err := client.FetchModels(context.Background(), testCorrelation())
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	snap, err := buildSnapshot(doc, time.Now())
	if err != nil {
		t.Fatalf("projected doc rejected: %v", err)
	}
	for id, spec := range snap.Models {
		if spec.Available {
			t.Fatalf("no row should be available without the approved route; %q was", id)
		}
	}
}
