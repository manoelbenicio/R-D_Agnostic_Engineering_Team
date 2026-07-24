package daemon

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestNewCLISpan_Valid(t *testing.T) {
	obs := CLIObservation{
		LaunchID:      "launch-123",
		ProcID:        "proc-456",
		TaskID:        "task-789",
		CLIKind:       "omnicli",
		ExitCodeClass: "success",
		LatencyMs:     150,
		CPUMs:         45,
		RSSBytes:      1048576,
		ArgvShape:     []string{"subcommand", "flag", "arg=<redacted>"},
		Outcome:       "completed",
		ReasonCode:    "ok",
	}

	span := NewCLISpan(obs)
	if span == nil {
		t.Fatal("expected non-nil span")
	}

	if err := span.Validate(); err != nil {
		t.Fatalf("span validation failed: %v", err)
	}

	if span.Hop != e2e.HopCLI {
		t.Errorf("got hop %q, want %q", span.Hop, e2e.HopCLI)
	}
	if span.Correlation.LaunchID != "launch-123" {
		t.Errorf("got launch_id %q, want launch-123", span.Correlation.LaunchID)
	}
	if span.Correlation.ProcID != "proc-456" {
		t.Errorf("got proc_id %q, want proc-456", span.Correlation.ProcID)
	}
	if span.Correlation.TaskID != "task-789" {
		t.Errorf("got task_id %q, want task-789", span.Correlation.TaskID)
	}
	if span.Labels["cli_kind"] != "omnicli" {
		t.Errorf("got cli_kind %q, want omnicli", span.Labels["cli_kind"])
	}
	if span.Labels["exit_code_class"] != "success" {
		t.Errorf("got exit_code_class %q, want success", span.Labels["exit_code_class"])
	}
	if span.Counters["latency_ms"] != 150 {
		t.Errorf("got latency_ms %d, want 150", span.Counters["latency_ms"])
	}
	if span.Counters["cpu_ms"] != 45 {
		t.Errorf("got cpu_ms %d, want 45", span.Counters["cpu_ms"])
	}
	if span.Counters["rss_bytes"] != 1048576 {
		t.Errorf("got rss_bytes %d, want 1048576", span.Counters["rss_bytes"])
	}
	if span.Outcome != "completed" {
		t.Errorf("got outcome %q, want completed", span.Outcome)
	}
	if span.ReasonCode != "ok" {
		t.Errorf("got reason %q, want ok", span.ReasonCode)
	}
	if span.SecretsPresent {
		t.Error("secrets_present must be false")
	}
}

func TestEmitCLI_SuccessAndLeakScan(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	obs := CLIObservation{
		LaunchID:      "launch-abc",
		ProcID:        "proc-def",
		CLIKind:       "agent",
		ExitCodeClass: "exit_0",
		LatencyMs:     200,
		ArgvShape:     []string{"subcommand", "flag=<redacted>"},
		Outcome:       "completed",
		ReasonCode:    "ok",
	}

	err := EmitCLI(rec, obs)
	if err != nil {
		t.Fatalf("EmitCLI failed: %v", err)
	}

	if sink.Len() != 1 {
		t.Fatalf("expected 1 recorded span, got %d", sink.Len())
	}

	spans := sink.Spans()
	report := e2e.ScanSpans(spans)
	if !report.Clean {
		t.Fatalf("leak scan failed with findings: %v", report.Findings)
	}
}

func TestEmitCLI_NilRecorder(t *testing.T) {
	obs := CLIObservation{
		LaunchID:  "launch-xyz",
		ProcID:    "proc-xyz",
		LatencyMs: 10,
		Outcome:   "completed",
	}

	if err := EmitCLI(nil, obs); err != nil {
		t.Errorf("expected nil error on nil recorder, got %v", err)
	}

	if err := EmitCLIHop(nil, obs); err != nil {
		t.Errorf("expected nil error on nil recorder (alias), got %v", err)
	}
}

func TestEmitCLI_MissingRequiredIDs(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	// Missing LaunchID
	obsMissingLaunch := CLIObservation{
		ProcID:    "proc-123",
		LatencyMs: 50,
		Outcome:   "completed",
	}
	if err := EmitCLI(rec, obsMissingLaunch); err == nil {
		t.Error("expected error when LaunchID is missing, got nil")
	}
	if sink.Len() != 0 {
		t.Errorf("expected 0 spans recorded on missing LaunchID, got %d", sink.Len())
	}

	// Missing ProcID
	obsMissingProc := CLIObservation{
		LaunchID:  "launch-123",
		LatencyMs: 50,
		Outcome:   "completed",
	}
	if err := EmitCLI(rec, obsMissingProc); err == nil {
		t.Error("expected error when ProcID is missing, got nil")
	}
	if sink.Len() != 0 {
		t.Errorf("expected 0 spans recorded on missing ProcID, got %d", sink.Len())
	}
}

func TestEmitCLI_InvalidArgvShape(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	obs := CLIObservation{
		LaunchID:  "launch-123",
		ProcID:    "proc-456",
		LatencyMs: 50,
		// Raw unapproved token (contains raw argument value / secret)
		ArgvShape: []string{"secret_password_value"},
		Outcome:   "completed",
	}

	err := EmitCLI(rec, obs)
	if err == nil {
		t.Error("expected error when ArgvShape contains unapproved token, got nil")
	}
	if sink.Len() != 0 {
		t.Errorf("expected 0 spans recorded on invalid ArgvShape, got %d", sink.Len())
	}
}

func TestEmitCLI_NegativeLatencyRefused(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	obs := CLIObservation{
		LaunchID:  "launch-123",
		ProcID:    "proc-456",
		LatencyMs: -100, // Invalid negative counter
		Outcome:   "completed",
	}

	err := EmitCLI(rec, obs)
	if err == nil {
		t.Error("expected error for negative latency, got nil")
	}
	if sink.Len() != 0 {
		t.Errorf("expected 0 spans recorded on negative latency, got %d", sink.Len())
	}
}

func TestEmitCLIHop_Alias(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)

	obs := CLIObservation{
		LaunchID:      "launch-alias",
		ProcID:        "proc-alias",
		CLIKind:       "subprocess",
		ExitCodeClass: "success",
		LatencyMs:     80,
		Outcome:       "completed",
	}

	err := EmitCLIHop(rec, obs)
	if err != nil {
		t.Fatalf("EmitCLIHop failed: %v", err)
	}

	if sink.Len() != 1 {
		t.Fatalf("expected 1 recorded span via alias, got %d", sink.Len())
	}
}
