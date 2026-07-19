package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestTrustedRuntimeProfiles(t *testing.T) {
	profiles := TrustedRuntimeProfiles()
	if len(profiles) != 4 {
		t.Fatalf("expected four protocol profiles, got %d", len(profiles))
	}
	tests := []struct {
		protocol brain.ProtocolFamily
		cli      brain.CLIKind
		endpoint string
		baseForm BaseURLForm
	}{
		{brain.ProtocolAnthropicMessages, brain.CLIClaudeCode, "/v1/messages", BaseURLRoot},
		{brain.ProtocolOpenAIResponses, brain.CLICodex, "/v1/responses", BaseURLV1},
		{brain.ProtocolOpenAIChat, brain.CLIOpenAICompatible, "/v1/chat/completions", BaseURLV1},
		{brain.ProtocolAntigravity, brain.CLIAntigravity, "/v1/antigravity", BaseURLV1},
	}
	for _, test := range tests {
		profile, err := LookupRuntimeProfile(test.protocol, test.cli)
		if err != nil {
			t.Fatalf("LookupRuntimeProfile(%s): %v", test.protocol, err)
		}
		if profile.Endpoint != test.endpoint || profile.BaseURLForm != test.baseForm || !profile.EvidenceRequired {
			t.Fatalf("unexpected profile: %+v", profile)
		}
		if err := profile.Validate(); err != nil {
			t.Fatalf("Validate profile: %v", err)
		}
		baseURL, err := profile.AdapterBaseURL("http://127.0.0.1:20128")
		if err != nil {
			t.Fatalf("AdapterBaseURL: %v", err)
		}
		if test.baseForm == BaseURLV1 && !strings.HasSuffix(baseURL, "/v1") || test.baseForm == BaseURLRoot && strings.HasSuffix(baseURL, "/v1") {
			t.Fatalf("unexpected base URL %q", baseURL)
		}
	}
	if _, err := LookupRuntimeProfile(brain.ProtocolOpenAIResponses, brain.CLIClaudeCode); !IsErrorClass(err, ErrorCapability) {
		t.Fatalf("expected CLI/profile mismatch, got %v", err)
	}
}

func TestFrozenTier20CanaryPolicyIsFailClosed(t *testing.T) {
	model := brain.RouteModel("agy/claude-opus-4-6-thinking")
	policy := FrozenTier20CanaryPolicy()
	if err := policy.Validate(model); err != nil {
		t.Fatalf("frozen policy invalid: %v", err)
	}
	if len(policy.Fallback.CrossModel) != 0 || !policy.Fallback.SameModelAccounts || !policy.Retry.PreCommitOnly || policy.SmartContext.Mode != SmartContextOff {
		t.Fatalf("unsafe frozen policy: %+v", policy)
	}
	policy.RouterOwner = brain.RouterOwnerLegacyGo
	if err := policy.Validate(model); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("legacy owner accepted: %v", err)
	}
}

func TestRoutePolicyRequiresApprovedEquivalentCrossModelFallback(t *testing.T) {
	model := brain.RouteModel("synthetic/primary")
	policy := FrozenTier20CanaryPolicy()
	policy.Fallback.CrossModel = []ApprovedFallback{{Model: "synthetic/fallback"}}
	if err := policy.Validate(model); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("unapproved fallback accepted: %v", err)
	}
	policy.Fallback.CrossModel[0].ApprovalID = "approval-synthetic"
	policy.Fallback.CrossModel[0].CapabilityEquivalent = true
	if err := policy.Validate(model); err != nil {
		t.Fatalf("approved fallback rejected: %v", err)
	}
}

func TestCircuitPolicyDurationBounds(t *testing.T) {
	valid := FrozenTier20CanaryPolicy().Circuit
	tests := []struct {
		name    string
		mutate  func(*CircuitPolicy)
		wantErr bool
	}{
		{
			name: "minimum positive durations",
			mutate: func(policy *CircuitPolicy) {
				policy.ObservationWindow = time.Nanosecond
				policy.OpenDuration = time.Nanosecond
			},
		},
		{
			name: "documented maximum durations",
			mutate: func(policy *CircuitPolicy) {
				policy.ObservationWindow = MaxCircuitObservationWindow
				policy.OpenDuration = MaxCircuitOpenDuration
			},
		},
		{
			name: "observation window above maximum",
			mutate: func(policy *CircuitPolicy) {
				policy.ObservationWindow = MaxCircuitObservationWindow + time.Nanosecond
			},
			wantErr: true,
		},
		{
			name: "open duration above maximum",
			mutate: func(policy *CircuitPolicy) {
				policy.OpenDuration = MaxCircuitOpenDuration + time.Nanosecond
			},
			wantErr: true,
		},
		{
			name: "timer overflow duration",
			mutate: func(policy *CircuitPolicy) {
				policy.ObservationWindow = time.Duration(1<<63 - 1)
				policy.OpenDuration = time.Duration(1<<63 - 1)
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := valid
			test.mutate(&policy)
			err := policy.Validate()
			if test.wantErr && !IsErrorClass(err, ErrorInvalidConfiguration) {
				t.Fatalf("unsafe circuit duration accepted: %v", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("safe circuit duration rejected: %v", err)
			}
		})
	}
}

func TestTelemetryHeaderParsingAllowlistAndPseudonymization(t *testing.T) {
	header := make(http.Header)
	header.Set(HeaderOmniRouteRequestID, "omni-request-synthetic")
	header.Set(HeaderActualModel, "agy/claude-opus-4-6-thinking")
	header.Set(HeaderActualRoute, "route-synthetic")
	header.Set(HeaderConnectionID, "source-connection-identity")
	header.Set(HeaderRetryCount, "2")
	header.Set(HeaderFallbackUsed, "true")
	header.Set(HeaderQuotaState, string(QuotaLimited))
	header.Set(HeaderCircuitState, string(CircuitHalfOpen))
	header.Set(HeaderUsageInput, "10")
	header.Set(HeaderUsageOutput, "4")
	header.Set(HeaderUsageTotal, "14")
	header.Set("Authorization", "synthetic value ignored by parser")
	header.Set("X-Synthetic-Content", "must be ignored")

	telemetry, err := ParseTelemetryHeaders(header)
	if err != nil {
		t.Fatalf("ParseTelemetryHeaders: %v", err)
	}
	if telemetry.PseudonymousConnection == "source-connection-identity" || !strings.HasPrefix(telemetry.PseudonymousConnection, "conn_") {
		t.Fatalf("connection was not pseudonymized: %q", telemetry.PseudonymousConnection)
	}
	encoded, err := json.Marshal(telemetry)
	if err != nil {
		t.Fatalf("Marshal telemetry: %v", err)
	}
	if bytes.Contains(encoded, []byte("source-connection-identity")) || bytes.Contains(encoded, []byte("synthetic value ignored")) || bytes.Contains(encoded, []byte("must be ignored")) {
		t.Fatal("telemetry included disallowed fields")
	}
}

func TestTelemetryEventRejectsUnknownContentFields(t *testing.T) {
	valid := `{"request_id":"request-synthetic","actual_model":"synthetic/model","actual_route":"route-synthetic","connection_id":"connection-synthetic","retry_count":0,"fallback_used":false,"quota_state":"available","circuit_state":"closed","usage":{"input":1,"output":1,"cache":0,"reasoning":0,"total":2}}`
	telemetry, err := ParseTelemetryEvent(strings.NewReader(valid), 4096)
	if err != nil || telemetry.ActualModel != "synthetic/model" {
		t.Fatalf("valid telemetry rejected: %+v %v", telemetry, err)
	}
	unsafe := `{"request_id":"request-synthetic","content":"synthetic content must not be parsed"}`
	if _, err := ParseTelemetryEvent(strings.NewReader(unsafe), 4096); !IsErrorClass(err, ErrorProtocol) {
		t.Fatalf("unsafe telemetry field accepted: %v", err)
	}
}

func TestSyntheticProtocolFixturesMatchContracts(t *testing.T) {
	fixtures := []struct {
		protocol    brain.ProtocolFamily
		requestPath string
		streamPath  string
	}{
		{brain.ProtocolAnthropicMessages, "testdata/anthropic/messages-request.json", "testdata/anthropic/messages-stream.sse"},
		{brain.ProtocolOpenAIResponses, "testdata/responses/responses-request.json", "testdata/responses/responses-stream.sse"},
		{brain.ProtocolOpenAIChat, "testdata/chat/chat-request.json", "testdata/chat/chat-stream.sse"},
	}
	for _, fixture := range fixtures {
		requestBody, err := os.ReadFile(fixture.requestPath)
		if err != nil {
			t.Fatalf("read request fixture: %v", err)
		}
		var requestDocument map[string]any
		if err := json.Unmarshal(requestBody, &requestDocument); err != nil {
			t.Fatalf("request fixture is not JSON: %v", err)
		}
		if model, _ := requestDocument["model"].(string); !strings.HasPrefix(model, "synthetic/") {
			t.Fatalf("fixture model is not synthetic: %q", model)
		}
		stream, err := os.Open(fixture.streamPath)
		if err != nil {
			t.Fatalf("open stream fixture: %v", err)
		}
		events, parseErr := readFixtureEvents(stream, fixture.protocol)
		_ = stream.Close()
		if parseErr != nil {
			t.Fatalf("parse stream fixture: %v", parseErr)
		}
		if err := ValidateSSESequence(fixture.protocol, events); err != nil {
			t.Fatalf("SSE fixture violates contract: %v", err)
		}
	}
}

func readFixtureEvents(reader *os.File, protocol brain.ProtocolFamily) ([]string, error) {
	var events []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events = append(events, strings.TrimPrefix(line, "event: "))
			continue
		}
		if protocol == brain.ProtocolOpenAIChat && strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				events = append(events, payload)
				continue
			}
			var chunk struct {
				Object string `json:"object"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				return nil, err
			}
			events = append(events, chunk.Object)
		}
	}
	return events, scanner.Err()
}

func TestSmartContextSafetyFlags(t *testing.T) {
	unsafe := SmartContextPolicy{Mode: SmartContextCanary, KillSwitch: true}
	if err := unsafe.Validate(); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("unsafe Smart Context accepted: %v", err)
	}
	safe := SmartContextPolicy{Mode: SmartContextShadow, KillSwitch: true, StructuralValidation: true, ExactWholeRequestFallback: true}
	if err := safe.Validate(); err != nil {
		t.Fatalf("safe shadow policy rejected: %v", err)
	}
}

func TestRetryDeadlineIsBounded(t *testing.T) {
	policy := FrozenTier20CanaryPolicy()
	policy.Retry.EndToEndDeadline = 11 * time.Minute
	if err := policy.Validate("synthetic/model"); !IsErrorClass(err, ErrorInvalidConfiguration) {
		t.Fatalf("unbounded deadline accepted: %v", err)
	}
}
