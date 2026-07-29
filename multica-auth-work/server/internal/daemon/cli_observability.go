package daemon

// cli_observability.go — HopCLI metadata-only span helper for the end-to-end
// correlation contract (OpenSpec 6.2, AB-REQ-39/40). It is a pure, metadata-only
// builder over the FROZEN L5 contract `agent-brain.e2e.v1`
// (server/internal/daemon/observability/e2e).
//
// Inputs: launch_id, proc_id, bounded CLI kind/exit-class/latency + closed argv shape;
// NEVER prompt, raw argv, env, repo content or stderr/stdout.
// Call site signature for F1 integrator (daemon.go):
//
//	err := daemon.EmitCLI(rec, daemon.CLIObservation{ ... })

import (
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// CLIObservation is the metadata-only input for the CLI hop (hop 4).
// All fields are caller-derived metadata — raw argv, prompts, environment variables,
// repository content, and process stdout/stderr MUST NEVER be included.
type CLIObservation struct {
	LaunchID string // correlation: launch_id (required by HopCLI contract)
	ProcID   string // correlation: proc_id (required by HopCLI contract)
	TaskID   string // optional correlation: task_id if known at CLI launch/completion

	CLIKind       string // bounded classification label: cli_kind (e.g. "omnicli", "subprocess", "agent")
	ExitCodeClass string // bounded classification label: exit_code_class (e.g. "success", "error", "exit_0")

	LatencyMs int64 // counter: latency_ms (non-negative)
	CPUMs     int64 // counter: cpu_ms (non-negative, 0 if unused)
	RSSBytes  int64 // counter: rss_bytes (non-negative, 0 if unused)

	ArgvShape []string // closed argv shape tokens (e.g. ["subcommand", "flag", "arg=<redacted>"])

	Outcome    string // bounded safe code, e.g. "completed", "failed", "killed"
	ReasonCode string // optional bounded safe code, e.g. "ok", "exit_1"
}

// NewCLISpan builds the finished hop-4 (HopCLI) span. It sets only label keys,
// counter keys, and argv shape tokens permitted by the frozen contract.
// The resulting span is validated fail-closed by e2e.Recorder on Emit.
func NewCLISpan(obs CLIObservation) *e2e.Span {
	span := e2e.NewSpan(e2e.HopCLI, e2e.Correlation{
		LaunchID: obs.LaunchID,
		ProcID:   obs.ProcID,
		TaskID:   obs.TaskID,
	}).
		WithCounter("latency_ms", obs.LatencyMs).
		WithOutcome(obs.Outcome, obs.ReasonCode)

	if obs.CPUMs > 0 {
		span = span.WithCounter("cpu_ms", obs.CPUMs)
	}
	if obs.RSSBytes > 0 {
		span = span.WithCounter("rss_bytes", obs.RSSBytes)
	}
	if obs.CLIKind != "" {
		span = span.WithLabel("cli_kind", obs.CLIKind)
	}
	if obs.ExitCodeClass != "" {
		span = span.WithLabel("exit_code_class", obs.ExitCodeClass)
	}
	if len(obs.ArgvShape) > 0 {
		span = span.WithArgvShape(obs.ArgvShape)
	}

	return span.Finish()
}

// EmitCLI builds and emits the CLI hop span through rec.
// A nil recorder is an explicit no-op (instrumentation must never crash callers).
// Any invalid field, missing required ID (launch_id, proc_id), or unapproved shape token
// causes the recorder to refuse the span fail-closed and return an error.
func EmitCLI(rec *e2e.Recorder, obs CLIObservation) error {
	if rec == nil {
		return nil
	}
	return rec.Emit(NewCLISpan(obs))
}

// EmitCLIHop is an alias for EmitCLI, matching alternative call-site conventions.
func EmitCLIHop(rec *e2e.Recorder, obs CLIObservation) error {
	return EmitCLI(rec, obs)
}
