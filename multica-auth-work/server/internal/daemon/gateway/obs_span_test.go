package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestEmitProviderSpanUsesSafeTelemetryAndClosedContract(t *testing.T) {
	const (
		rawAccount    = "source-account-identity"
		rawConnection = "source-connection-identity"
	)
	header := make(http.Header)
	header.Set(HeaderOmniRouteRequestID, "omni-request-synthetic")
	header.Set(HeaderActualModel, "agy/claude-opus-4-6-thinking")
	header.Set(HeaderActualRoute, "route-synthetic")
	header.Set(HeaderAccountID, rawAccount)
	header.Set(HeaderConnectionID, rawConnection)
	header.Set(HeaderSelectionReason, string(SelectionIndependentRotation))
	header.Set(HeaderRetryCount, "2")
	header.Set(HeaderFallbackUsed, "true")
	header.Set(HeaderQuotaState, string(QuotaLimited))
	header.Set(HeaderCircuitState, string(CircuitHalfOpen))
	header.Set(HeaderUsageInput, "10")
	header.Set(HeaderUsageOutput, "4")
	header.Set(HeaderUsageCache, "2")
	header.Set(HeaderUsageReasoning, "3")
	header.Set(HeaderUsageTotal, "19")
	telemetry, err := ParseTelemetryHeaders(header)
	if err != nil {
		t.Fatalf("parse telemetry: %v", err)
	}

	sink := e2e.NewMemorySink()
	started := time.Unix(100, 0).UTC()
	record := ProviderSpanRecord{
		RequestID:          "platform-request-synthetic",
		PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol:           brain.ProtocolAnthropicMessages,
		Telemetry:          telemetry,
		StartedAt:          started,
		EndedAt:            started.Add(125 * time.Millisecond),
		Outcome:            "ok",
		ReasonCode:         "completed",
		HTTPStatus:         http.StatusOK,
	}
	if err := EmitProviderSpan(e2e.NewRecorder(sink), record); err != nil {
		t.Fatalf("emit provider span: %v", err)
	}

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("recorded spans=%d want=1", len(spans))
	}
	span := spans[0]
	if span.Hop != e2e.HopRoute ||
		span.Correlation.RequestID != record.RequestID ||
		span.Correlation.OmniRequestID != telemetry.RequestID {
		t.Fatalf("unexpected route correlation: %+v", span.Correlation)
	}
	wantLabels := map[string]string{
		"route_model":          "agy/claude-opus-4-6-thinking",
		"route":                "route-synthetic",
		"protocol":             string(brain.ProtocolAnthropicMessages),
		"principal_pseudonym":  record.PrincipalPseudonym,
		"account_pseudonym":    telemetry.PseudonymousAccount,
		"connection_pseudonym": telemetry.PseudonymousConnection,
		"selection_reason":     string(SelectionIndependentRotation),
		"quota_state":          string(QuotaLimited),
		"circuit_state":        string(CircuitHalfOpen),
	}
	for key, want := range wantLabels {
		if got := span.Labels[key]; got != want {
			t.Fatalf("label %q=%q want=%q", key, got, want)
		}
	}
	wantCounters := map[string]int64{
		"latency_ms":       125,
		"retry_count":      2,
		"fallback_count":   1,
		"input_tokens":     10,
		"output_tokens":    4,
		"cache_tokens":     2,
		"reasoning_tokens": 3,
		"total_tokens":     19,
	}
	if len(span.Counters) != len(wantCounters) {
		t.Fatalf("counter count=%d want=%d", len(span.Counters), len(wantCounters))
	}
	for key, want := range wantCounters {
		if got := span.Counters[key]; got != want {
			t.Fatalf("counter %q=%d want=%d", key, got, want)
		}
	}
	if report := e2e.ScanSpans(spans); !report.Clean {
		t.Fatalf("provider span failed structural scan: %+v", report.Findings)
	}
	encoded, err := json.Marshal(span)
	if err != nil {
		t.Fatalf("marshal span: %v", err)
	}
	if strings.Contains(string(encoded), rawAccount) || strings.Contains(string(encoded), rawConnection) {
		t.Fatal("raw identity escaped into provider span")
	}
}

func TestEmitProviderSpanFailsClosed(t *testing.T) {
	started := time.Unix(100, 0).UTC()
	valid := ProviderSpanRecord{
		RequestID:          "platform-request-synthetic",
		PrincipalPseudonym: "principal_0123456789abcdef",
		Protocol:           brain.ProtocolOpenAIResponses,
		Telemetry: Telemetry{
			RequestID:              "omni-request-synthetic",
			ActualModel:            "openai/gpt-5",
			ActualRoute:            "route-synthetic",
			PseudonymousAccount:    "acct_0123456789abcdef",
			PseudonymousConnection: "conn_fedcba9876543210",
			SelectionReason:        SelectionContinuation,
			RetryCount:             1,
			Quota:                  QuotaAvailable,
			Circuit:                CircuitClosed,
			Usage:                  Usage{Input: 1, Output: 1, Total: 2},
		},
		StartedAt:  started,
		EndedAt:    started.Add(time.Second),
		Outcome:    "ok",
		ReasonCode: "completed",
		HTTPStatus: http.StatusOK,
	}

	tests := []struct {
		name   string
		mutate func(*ProviderSpanRecord)
	}{
		{name: "nil recorder"},
		{name: "missing platform request", mutate: func(record *ProviderSpanRecord) { record.RequestID = "" }},
		{name: "missing omni request", mutate: func(record *ProviderSpanRecord) { record.Telemetry.RequestID = "" }},
		{name: "raw principal", mutate: func(record *ProviderSpanRecord) { record.PrincipalPseudonym = "raw-principal" }},
		{name: "raw account", mutate: func(record *ProviderSpanRecord) { record.Telemetry.PseudonymousAccount = "raw-account" }},
		{name: "unsafe model", mutate: func(record *ProviderSpanRecord) { record.Telemetry.ActualModel = "https://provider.invalid/model" }},
		{name: "unknown protocol", mutate: func(record *ProviderSpanRecord) { record.Protocol = "unknown-protocol" }},
		{name: "negative usage", mutate: func(record *ProviderSpanRecord) { record.Telemetry.Usage.Total = -1 }},
		{name: "invalid outcome", mutate: func(record *ProviderSpanRecord) { record.Outcome = "free form content" }},
		{name: "reversed timestamps", mutate: func(record *ProviderSpanRecord) { record.EndedAt = record.StartedAt.Add(-time.Second) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := valid
			if test.mutate != nil {
				test.mutate(&record)
			}
			sink := e2e.NewMemorySink()
			var recorder *e2e.Recorder
			if test.name != "nil recorder" {
				recorder = e2e.NewRecorder(sink)
			}
			if err := EmitProviderSpan(recorder, record); err == nil {
				t.Fatal("unsafe provider span accepted")
			}
			if sink.Len() != 0 {
				t.Fatal("unsafe provider span was recorded")
			}
		})
	}
}
