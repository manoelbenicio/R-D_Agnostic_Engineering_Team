package daemon

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestRunTaskObservesLaunchIdentityRejectionsExactlyOnce(t *testing.T) {
	tests := []struct {
		name            string
		class           string
		customRuntime   bool
		claimedProvider string
		mutate          func(*Daemon, *Task)
		wantDecision    brain.AdmissionState
		wantReadiness   brain.GatewayReadinessState
	}{
		{
			name: "custom runtime", class: "custom_runtime_not_allowed", customRuntime: true,
			claimedProvider: "claude", wantDecision: brain.AdmissionRoutePolicyRejected,
			wantReadiness: brain.GatewayReadinessNotRequired,
		},
		{
			name: "custom args", class: "custom_args_not_allowed", claimedProvider: "claude",
			mutate:       func(daemon *Daemon, task *Task) { task.Agent.CustomArgs = []string{"--synthetic"} },
			wantDecision: brain.AdmissionRoutePolicyRejected, wantReadiness: brain.GatewayReadinessNotRequired,
		},
		{
			name: "built-in mapping unavailable", class: "builtin_runtime_mapping_unavailable", claimedProvider: "claude",
			mutate:       func(daemon *Daemon, _ *Task) { daemon.cfg.AgentBrain.CLIKind = brain.CLIKind("unsupported") },
			wantDecision: brain.AdmissionCapabilityRejected, wantReadiness: brain.GatewayReadinessSelectedProtocol,
		},
		{
			name: "built-in provider mismatch", class: "builtin_runtime_provider_mismatch", claimedProvider: "codex",
			wantDecision: brain.AdmissionRoutePolicyRejected, wantReadiness: brain.GatewayReadinessNotRequired,
		},
		{
			name: "built-in unavailable", class: "builtin_runtime_unavailable", claimedProvider: "claude",
			mutate:       func(daemon *Daemon, _ *Task) { delete(daemon.cfg.Agents, "claude") },
			wantDecision: brain.AdmissionCapabilityRejected, wantReadiness: brain.GatewayReadinessSelectedProtocol,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			daemon, credential, launchMarker := newAgentBrainSecurityTestDaemon(t, test.customRuntime)
			sink := e2e.NewMemorySink()
			daemon.agentBrainOBS = brain.NewAdmissionObserver(sink)
			task := syntheticGatewayTask()
			if test.mutate != nil {
				test.mutate(daemon, &task)
			}

			_, err := daemon.runTask(context.Background(), task, test.claimedProvider, 0, daemon.logger)
			assertAgentBrainAdmissionClass(t, err, test.class)
			if credential.calls != 0 {
				t.Fatalf("credential source called %d times before launch-identity rejection", credential.calls)
			}
			if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
				t.Fatal("synthetic executable ran after launch-identity rejection")
			}
			assertSingleCleanAdmissionSpan(t, sink, test.class, test.wantDecision, test.wantReadiness)
		})
	}
}

func TestRunTaskObservesMissingWorkspaceRejectionExactlyOnce(t *testing.T) {
	daemon, credential, launchMarker := newAgentBrainSecurityTestDaemon(t, false)
	sink := e2e.NewMemorySink()
	daemon.agentBrainOBS = brain.NewAdmissionObserver(sink)
	task := syntheticGatewayTask()
	task.WorkspaceID = ""

	_, err := daemon.runTask(context.Background(), task, "claude", 0, daemon.logger)
	if err == nil || err.Error() != "refusing to spawn agent: task has no workspace_id (task_id=synthetic-task)" {
		t.Fatalf("missing-workspace error=%v", err)
	}
	if credential.calls != 0 {
		t.Fatalf("credential source called %d times before missing-workspace rejection", credential.calls)
	}
	if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
		t.Fatal("synthetic executable ran after missing-workspace rejection")
	}
	assertSingleCleanAdmissionSpan(t, sink, "workspace_required", brain.AdmissionRoutePolicyRejected, brain.GatewayReadinessNotRequired)
}

func TestRunTaskDoesNotDuplicateLaterAdmissionRejectionSpan(t *testing.T) {
	daemon, credential, launchMarker := newAgentBrainSecurityTestDaemon(t, false)
	sink := e2e.NewMemorySink()
	daemon.agentBrainOBS = brain.NewAdmissionObserver(sink)
	daemon.agentBrainInitErr = errors.New("synthetic initialization failure")

	_, err := daemon.runTask(context.Background(), syntheticGatewayTask(), "claude", 0, daemon.logger)
	assertAgentBrainAdmissionClass(t, err, "integration_initialization_failed")
	if credential.calls != 0 {
		t.Fatalf("credential source called %d times before initialization rejection", credential.calls)
	}
	if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
		t.Fatal("synthetic executable ran after initialization rejection")
	}
	assertSingleCleanAdmissionSpan(t, sink, "integration_initialization_failed", brain.AdmissionGatewayUnavailable, brain.GatewayReadinessUnavailable)
}

func assertSingleCleanAdmissionSpan(
	t *testing.T,
	sink *e2e.MemorySink,
	wantClass string,
	wantDecision brain.AdmissionState,
	wantReadiness brain.GatewayReadinessState,
) {
	t.Helper()
	if got := sink.Len(); got != 1 {
		t.Fatalf("admission span count=%d, want 1", got)
	}
	span := sink.Spans()[0]
	if got := span.Labels["fail_closed_class"]; got != wantClass {
		t.Fatalf("fail_closed_class=%q, want %q", got, wantClass)
	}
	if got := span.Labels["admission_decision"]; got != string(wantDecision) {
		t.Fatalf("admission_decision=%q, want %q", got, wantDecision)
	}
	if got := span.Labels["readiness_result"]; got != string(wantReadiness) {
		t.Fatalf("readiness_result=%q, want %q", got, wantReadiness)
	}
	if report := e2e.ScanFromSink(sink); !report.Clean {
		t.Fatalf("admission span failed structural scan: findings=%d", len(report.Findings))
	}
}
