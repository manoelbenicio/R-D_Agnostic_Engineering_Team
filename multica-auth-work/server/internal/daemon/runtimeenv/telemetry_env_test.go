package runtimeenv

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestTrustedTelemetryEnvExactOfficialValues(t *testing.T) {
	got, err := trustedTelemetryEnv(AdapterEnvironment{
		TelemetryOTLPLogsEndpoint: "http://127.0.0.1:4318/v1/logs",
		TelemetryTaskID:           "task-canonical-1",
		TelemetryRequestID:        "req-canonical-1",
	})
	if err != nil {
		t.Fatalf("valid telemetry config errored: %v", err)
	}
	want := map[string]string{
		"CLAUDE_CODE_ENABLE_TELEMETRY":     "1",
		"OTEL_LOGS_EXPORTER":               "otlp",
		"OTEL_METRICS_EXPORTER":            "none",
		"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL": "http/json",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://127.0.0.1:4318/v1/logs",
		"OTEL_RESOURCE_ATTRIBUTES":         "agent_brain.task_id=task-canonical-1,agent_brain.request_id=req-canonical-1",
		"OTEL_LOG_USER_PROMPTS":            "0",
		"OTEL_LOG_ASSISTANT_RESPONSES":     "0",
		"OTEL_LOG_TOOL_DETAILS":            "0",
		"OTEL_LOG_TOOL_CONTENT":            "0",
	}
	if len(got) != len(want) {
		t.Fatalf("key count=%d want %d (got=%v)", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("key %s=%q want %q", k, got[k], v)
		}
	}
	if _, ok := got["OTEL_LOG_RAW_API_BODIES"]; ok {
		t.Fatal("OTEL_LOG_RAW_API_BODIES must never be injected")
	}
	// Telemetry off ONLY when endpoint entirely absent.
	off, err := trustedTelemetryEnv(AdapterEnvironment{})
	if off != nil || err != nil {
		t.Fatalf("absent endpoint must be (nil,nil); got %v err %v", off, err)
	}
}

func TestTrustedTelemetryEnvFailsClosedOnMalformedConfig(t *testing.T) {
	base := func(ep, task, req string) AdapterEnvironment {
		return AdapterEnvironment{TelemetryOTLPLogsEndpoint: ep, TelemetryTaskID: task, TelemetryRequestID: req}
	}
	cases := map[string]AdapterEnvironment{
		"external_host":  base("http://10.0.0.5:4318/v1/logs", "t", "r"),
		"localhost_dns":  base("http://localhost:4318/v1/logs", "t", "r"),
		"https_scheme":   base("https://127.0.0.1:4318/v1/logs", "t", "r"),
		"wrong_path":     base("http://127.0.0.1:4318/v1/traces", "t", "r"),
		"missing_port":   base("http://127.0.0.1/v1/logs", "t", "r"),
		"has_query":      base("http://127.0.0.1:4318/v1/logs?x=1", "t", "r"),
		"has_userinfo":   base("http://u:p@127.0.0.1:4318/v1/logs", "t", "r"),
		"empty_task":     base("http://127.0.0.1:4318/v1/logs", "", "r"),
		"empty_request":  base("http://127.0.0.1:4318/v1/logs", "t", ""),
		"unsafe_task":    base("http://127.0.0.1:4318/v1/logs", "t,x=1", "r"),
		"unsafe_request": base("http://127.0.0.1:4318/v1/logs", "t", "r=evil,y=z"),
		// Partial config: endpoint absent but a correlation id set must error
		// (only entirely-absent telemetry is "off").
		"partial_task_only":    base("", "task-1", ""),
		"partial_request_only": base("", "", "req-1"),
	}
	for name, cfg := range cases {
		got, err := trustedTelemetryEnv(cfg)
		if err == nil || got != nil {
			t.Fatalf("%s: expected fail-closed (nil,err); got %v err %v", name, got, err)
		}
	}
}

func TestTelemetryKeysDeniedForNonTrustedOrigins(t *testing.T) {
	denied := []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY", "OTEL_LOGS_EXPORTER", "OTEL_METRICS_EXPORTER",
		"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_RESOURCE_ATTRIBUTES", "OTEL_LOG_USER_PROMPTS", "OTEL_LOG_ASSISTANT_RESPONSES",
		"OTEL_LOG_TOOL_DETAILS", "OTEL_LOG_TOOL_CONTENT", "OTEL_LOG_RAW_API_BODIES",
		"OTEL_EXPORTER_OTLP_HEADERS", "OTEL_EXPORTER_OTLP_LOGS_HEADERS", "OTEL_EXPORTER_OTLP_CERTIFICATE",
	}
	for _, k := range denied {
		c := ClassifyEnvironmentKey(k)
		if !c.Denied {
			t.Fatalf("key %s must be denied for user/local/inherited origins", k)
		}
	}
}

func TestTelemetryKeysAllowedOnlyAsTrusted(t *testing.T) {
	keys := []string{
		"CLAUDE_CODE_ENABLE_TELEMETRY", "OTEL_LOGS_EXPORTER", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_RESOURCE_ATTRIBUTES", "OTEL_LOG_USER_PROMPTS", "OTEL_LOG_TOOL_CONTENT",
	}
	for _, k := range keys {
		if !trustedEntryAllowed(brain.CLIClaudeCode, k, originTrustedLocal) {
			t.Fatalf("%s must be allowed as trusted-local", k)
		}
		if trustedEntryAllowed(brain.CLIClaudeCode, k, originLocal) {
			t.Fatalf("%s must NOT be allowed as local origin (override attempt)", k)
		}
		if trustedEntryAllowed(brain.CLIClaudeCode, k, originInherited) {
			t.Fatalf("%s must NOT be allowed as inherited origin (override attempt)", k)
		}
	}
}
