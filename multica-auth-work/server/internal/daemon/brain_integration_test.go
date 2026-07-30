package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
	"github.com/multica-ai/multica/server/internal/daemon/runtimeenv"
)

const syntheticReferenceSecret = "synthetic-reference-only"

type syntheticCredentialSource struct{}

func (syntheticCredentialSource) WithCredential(ctx context.Context, ref brain.SecretFileRef, use func(string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ref.Path != "/synthetic/omniroute/reference" || use == nil {
		return &agentBrainAdmissionError{class: "synthetic_reference_invalid"}
	}
	return use(syntheticReferenceSecret)
}

type countingSyntheticCredentialSource struct {
	calls int
}

func (source *countingSyntheticCredentialSource) WithCredential(ctx context.Context, ref brain.SecretFileRef, use func(string) error) error {
	source.calls++
	if err := ctx.Err(); err != nil {
		return err
	}
	return use(syntheticReferenceSecret)
}

func TestAgentBrainDevelopmentIsolationSmoke(t *testing.T) {
	gatewayServer := newSyntheticGateway(t, true)
	defer gatewayServer.Close()

	config := syntheticAgentBrainConfig(t, gatewayServer.URL)
	runtime, err := newAgentBrainRuntime(config, AgentBrainDependencies{
		CredentialSource: syntheticCredentialSource{},
		HTTPClient:       gatewayServer.Client(),
		InheritedEnvironment: func() []string {
			return []string{
				"PATH=" + os.Getenv("PATH"),
				"HOME=/synthetic/provider-home",
				"OPENAI_API_KEY=synthetic-provider-value",
				"OPENAI_BASE_URL=https://direct-provider.invalid/v1",
				"NVIDIA_API_KEY=synthetic-provider-value",
			}
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	task := syntheticGatewayTask()
	plan, err := runtime.admitTask(context.Background(), task, "claude", string(config.RouteModel))
	if err != nil {
		t.Fatalf("admitTask: %v", err)
	}
	if plan.Task.Request.RouterOwner != brain.RouterOwnerOmniRoute || plan.Profile.ID != gateway.ProfileAnthropicMessages {
		t.Fatalf("unexpected router/profile: owner=%q profile=%q", plan.Task.Request.RouterOwner, plan.Profile.ID)
	}

	envRoot := t.TempDir()
	workDir := filepath.Join(envRoot, "workdir")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("create workdir: %v", err)
	}
	prepared := &execenv.Environment{RootDir: envRoot, WorkDir: workDir}
	launch, err := runtime.buildLaunch(context.Background(), plan, prepared, map[string]string{
		"MULTICA_TOKEN":       "mat_synthetic_task_scope",
		"MULTICA_TASK_ID":     "legacy-task-reference",
		"G3_SYNTHETIC_CHILD":  "1",
		"G3_EXPECTED_GATEWAY": gatewayServer.URL,
	}, map[string]string{"SAFE_CUSTOM_SETTING": "synthetic"})
	if err != nil {
		t.Fatalf("buildLaunch: %v", err)
	}
	keys := launch.Environment.Keys()
	for _, forbidden := range []string{"OPENAI_API_KEY", "OPENAI_BASE_URL", "NVIDIA_API_KEY", "CODEX_HOME"} {
		if containsString(keys, forbidden) {
			t.Fatalf("forbidden child key present: %s", forbidden)
		}
	}
	for _, required := range []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "MULTICA_SESSION_ID", "MULTICA_REQUEST_ID", "MULTICA_ROUTER_OWNER"} {
		if !containsString(keys, required) {
			t.Fatalf("required child key missing: %s", required)
		}
	}

	command := exec.Command(os.Args[0], "-test.run=TestAgentBrainSyntheticChild")
	command.Env = launch.Environment.Exec()
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		t.Fatalf("synthetic isolation child: %v", err)
	}
}

func TestAgentBrainSyntheticChild(t *testing.T) {
	if os.Getenv("G3_SYNTHETIC_CHILD") != "1" {
		return
	}
	for _, forbidden := range []string{
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "NVIDIA_API_KEY", "NIM_BASE_URL", "CODEX_HOME",
	} {
		if _, present := os.LookupEnv(forbidden); present {
			os.Exit(41)
		}
	}
	if value, present := os.LookupEnv("ANTHROPIC_BASE_URL"); !present || value != os.Getenv("G3_EXPECTED_GATEWAY") {
		os.Exit(42)
	}
	if _, present := os.LookupEnv("ANTHROPIC_AUTH_TOKEN"); !present {
		os.Exit(43)
	}
	home := os.Getenv("HOME")
	if home == "" {
		os.Exit(44)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); !os.IsNotExist(err) {
		os.Exit(45)
	}
	os.Exit(0)
}

func TestAgentBrainRejectsDualRouterBeforeGatewayAccess(t *testing.T) {
	gatewayServer := newSyntheticGateway(t, true)
	defer gatewayServer.Close()
	runtime, err := newAgentBrainRuntime(syntheticAgentBrainConfig(t, gatewayServer.URL), AgentBrainDependencies{
		CredentialSource: syntheticCredentialSource{}, HTTPClient: gatewayServer.Client(),
		InheritedEnvironment: func() []string { return []string{"PATH=/synthetic/bin"} },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	task := syntheticGatewayTask()
	task.RuntimeRouterOwner = "alternate_router"
	if _, err := runtime.admitTask(context.Background(), task, "claude", string(runtime.config.RouteModel)); err == nil {
		t.Fatal("dual router task was admitted")
	}
}

func TestAgentBrainFailsClosedWhenGatewayNotReady(t *testing.T) {
	gatewayServer := newSyntheticGateway(t, false)
	defer gatewayServer.Close()
	runtime, err := newAgentBrainRuntime(syntheticAgentBrainConfig(t, gatewayServer.URL), AgentBrainDependencies{
		CredentialSource: syntheticCredentialSource{}, HTTPClient: gatewayServer.Client(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	if _, err := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel)); err == nil {
		t.Fatal("unready gateway task was admitted")
	}
	if snapshot := runtime.snapshot(); snapshot.Readiness == brain.GatewayReadinessReady {
		t.Fatalf("readiness=%q, want fail-closed state", snapshot.Readiness)
	}
}

func TestAgentBrainUsesInstalledOmniRouteHealthContract(t *testing.T) {
	type recordedRequest struct {
		path          string
		authenticated bool
	}
	var requests []recordedRequest
	available := true
	enabled := true
	gatewayServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests = append(requests, recordedRequest{
			path: request.URL.Path, authenticated: request.Header.Get("Authorization") != "",
		})
		switch request.URL.Path {
		case "/api/health/ping":
			if request.Header.Get("Authorization") != "" {
				response.WriteHeader(http.StatusBadRequest)
				return
			}
			response.WriteHeader(http.StatusOK)
		case "/v1/models":
			if request.Header.Get("Authorization") != "Bearer "+syntheticReferenceSecret {
				response.WriteHeader(http.StatusUnauthorized)
				return
			}
			writeSyntheticModels(response, available, enabled)
		case "/api/monitoring/health":
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte(`{"status":"degraded"}`))
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer gatewayServer.Close()

	config := syntheticAgentBrainConfig(t, gatewayServer.URL)
	runtime, err := newAgentBrainRuntime(config, AgentBrainDependencies{
		CredentialSource: syntheticCredentialSource{}, HTTPClient: gatewayServer.Client(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	plan, err := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(config.RouteModel))
	if err != nil || plan == nil {
		t.Fatal("installed OmniRoute health contract did not admit the synthetic task")
	}
	if runtime.config.Neutral.Gateway.BaseURL != gatewayServer.URL {
		t.Fatal("gateway root BaseURL changed while constructing health endpoints")
	}
	want := []recordedRequest{
		{path: "/api/health/ping", authenticated: false},
		{path: "/v1/models", authenticated: true},
		{path: "/v1/models", authenticated: true},
	}
	if len(requests) != len(want) {
		t.Fatalf("gateway request count=%d, want %d", len(requests), len(want))
	}
	for index := range want {
		if requests[index] != want[index] {
			t.Fatalf("gateway request %d did not match the bounded health contract", index)
		}
	}
}

func TestAgentBrainDegradedPingFailsClosedBeforeAuthenticatedReadiness(t *testing.T) {
	// Bounded resilient admission may retry a transient 5xx liveness a few
	// times; keep the wait short so the test stays fast while still exercising
	// the fail-closed path.
	t.Setenv("AGENT_BRAIN_READINESS_ADMISSION_WAIT_MS", "150")
	var paths []string
	gatewayServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		switch request.URL.Path {
		case "/api/health/ping":
			response.WriteHeader(http.StatusServiceUnavailable)
		case "/api/monitoring/health":
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write([]byte(`{"status":"degraded"}`))
		case "/v1/models":
			response.WriteHeader(http.StatusOK)
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer gatewayServer.Close()

	credential := &countingSyntheticCredentialSource{}
	runtime, err := newAgentBrainRuntime(syntheticAgentBrainConfig(t, gatewayServer.URL), AgentBrainDependencies{
		CredentialSource: credential, HTTPClient: gatewayServer.Client(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	if _, err := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel)); err == nil {
		t.Fatal("degraded OmniRoute liveness was admitted")
	}
	if credential.calls != 0 {
		t.Fatal("degraded liveness reached the authenticated readiness credential source")
	}
	// Security invariant: degraded liveness must never advance past the public
	// ping endpoint to authenticated readiness/models (retries of the ping
	// itself are acceptable under bounded resilient admission).
	if len(paths) == 0 {
		t.Fatal("liveness ping was never attempted")
	}
	for _, p := range paths {
		if p != "/api/health/ping" {
			t.Fatalf("degraded liveness advanced past ping to %q", p)
		}
	}
	if snapshot := runtime.snapshot(); snapshot.Readiness == brain.GatewayReadinessReady {
		t.Fatal("degraded liveness reported ready")
	}
}

func TestAgentBrainCentralCapacityReconcilesOverloadAndCancellation(t *testing.T) {
	gatewayServer := newSyntheticGateway(t, true)
	defer gatewayServer.Close()
	credential := &countingSyntheticCredentialSource{}
	runtime, err := newAgentBrainRuntime(syntheticAgentBrainConfig(t, gatewayServer.URL), AgentBrainDependencies{
		CredentialSource: credential, HTTPClient: gatewayServer.Client(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}

	first, err := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel))
	if err != nil {
		t.Fatalf("first admitTask: %v", err)
	}
	runtime.recordLaunch(first)
	credentialCalls := credential.calls

	_, err = runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel))
	var admissionErr *agentBrainAdmissionError
	if !errors.As(err, &admissionErr) || admissionErr.class != "local_capacity_overloaded" || !admissionErr.retryable {
		t.Fatalf("overload error=%v, want retryable local_capacity_overloaded", err)
	}
	if credential.calls != credentialCalls {
		t.Fatal("overload reached the credential callback before failing closed")
	}

	runtime.recordTerminal(context.Background(), first, "completed", nil)
	second, err := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel))
	if err != nil {
		t.Fatalf("post-release admitTask: %v", err)
	}
	runtime.recordLaunch(second)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	runtime.recordTerminal(cancelled, second, "cancelled", context.Canceled)
	runtime.recordTerminal(cancelled, second, "cancelled", context.Canceled)

	counters := runtime.snapshot().Capacity
	if err := counters.Reconcile(); err != nil {
		t.Fatalf("central counters do not reconcile: %v; counters=%+v", err, counters)
	}
	if counters.Offered != 3 || counters.Admitted != 2 || counters.Rejected != 1 || counters.Overloaded != 1 {
		t.Fatalf("admission counters=%+v", counters)
	}
	if counters.Started != 2 || counters.Completed != 1 || counters.Cancelled != 1 || counters.CancelledAfterStart != 1 {
		t.Fatalf("terminal counters=%+v", counters)
	}
	if counters.InUse != 0 || counters.CapacityAcquired != counters.CapacityReleased || counters.PeakInUse != 1 {
		t.Fatalf("release counters=%+v", counters)
	}
}

func TestAgentBrainConcurrentDuplicateStartAndCancellationFinishEmitOnce(t *testing.T) {
	runtime, plan, recorder := newAgentBrainLifecycleDiagnosticTest(t)

	runAgentBrainConcurrentCalls(64, func(int) {
		runtime.recordLaunch(plan)
	})
	started := runtime.capacity.Snapshot()
	if started.Started != 1 || started.Active != 1 || started.PendingStart != 0 {
		t.Fatalf("duplicate launch counter transition=%+v", started)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	runAgentBrainConcurrentCalls(64, func(int) {
		runtime.recordTerminal(cancelled, plan, "cancelled", context.Canceled)
	})

	diagnostics := recorder.snapshot()
	if diagnostics.launchLogs != 1 || diagnostics.routeSelectionEvents != 1 {
		t.Fatalf("launch diagnostics=%+v, want one launch log and one route-selection event", diagnostics)
	}
	if diagnostics.terminalLogs != 1 || diagnostics.cancellationEvents != 1 || diagnostics.terminalOutcomes["cancelled"] != 1 {
		t.Fatalf("terminal diagnostics=%+v, want one cancellation event and one cancelled terminal log", diagnostics)
	}
	if len(diagnostics.terminalOutcomes) != 1 {
		t.Fatalf("contradictory terminal outcomes emitted: %+v", diagnostics.terminalOutcomes)
	}

	counters := runtime.capacity.Snapshot()
	if err := counters.Reconcile(); err != nil {
		t.Fatalf("concurrent duplicate lifecycle does not reconcile: %v; counters=%+v", err, counters)
	}
	if counters.Offered != 1 || counters.Admitted != 1 || counters.Started != 1 || counters.Cancelled != 1 || counters.CancelledAfterStart != 1 {
		t.Fatalf("duplicate lifecycle counter transition=%+v", counters)
	}
	if counters.Completed != 0 || counters.Failed != 0 || counters.Active != 0 || counters.InUse != 0 || counters.CapacityAcquired != 1 || counters.CapacityReleased != 1 {
		t.Fatalf("duplicate lifecycle release=%+v", counters)
	}
}

func TestAgentBrainConcurrentContradictoryTerminalCallsEmitWinnerOnly(t *testing.T) {
	runtime, plan, recorder := newAgentBrainLifecycleDiagnosticTest(t)
	runtime.recordLaunch(plan)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	runAgentBrainConcurrentCalls(96, func(index int) {
		switch index % 3 {
		case 0:
			runtime.recordTerminal(cancelled, plan, "cancelled", context.Canceled)
		case 1:
			runtime.recordTerminal(context.Background(), plan, "completed", nil)
		default:
			runtime.recordTerminal(context.Background(), plan, "failed", errors.New("synthetic execution failure"))
		}
	})

	diagnostics := recorder.snapshot()
	if diagnostics.terminalLogs != 1 || terminalDiagnosticCount(diagnostics.terminalOutcomes) != 1 || len(diagnostics.terminalOutcomes) != 1 {
		t.Fatalf("racing terminal diagnostics were contradictory: %+v", diagnostics)
	}
	winner := ""
	for outcome := range diagnostics.terminalOutcomes {
		winner = outcome
	}
	wantCancellationEvents := 0
	if winner == "cancelled" {
		wantCancellationEvents = 1
	}
	if diagnostics.cancellationEvents != wantCancellationEvents {
		t.Fatalf("cancellation events=%d, winner=%q", diagnostics.cancellationEvents, winner)
	}

	counters := runtime.capacity.Snapshot()
	if err := counters.Reconcile(); err != nil {
		t.Fatalf("racing terminal lifecycle does not reconcile: %v; counters=%+v", err, counters)
	}
	if counters.Completed+counters.Failed+counters.Cancelled != 1 || counters.CapacityReleased != 1 || counters.InUse != 0 {
		t.Fatalf("racing terminal counter/release transition=%+v", counters)
	}
}

func TestAgentBrainTier20SchemaRemainsFailClosedAtDevelopmentLimit(t *testing.T) {
	config := syntheticAgentBrainConfig(t, "http://127.0.0.1:20128")
	if config.Neutral.CapacityTier != brain.CapacityTier20 {
		t.Fatalf("capacity tier=%d, want schema tier 20", config.Neutral.CapacityTier)
	}
	if agentBrainDevelopmentMaxTasks != 1 {
		t.Fatalf("development limit=%d, want 1", agentBrainDevelopmentMaxTasks)
	}
	if got := effectiveTaskAdmissionLimit(config, 20); got != agentBrainDevelopmentMaxTasks {
		t.Fatalf("effective admission limit=%d, want fail-closed development limit 1", got)
	}
	config.DevelopmentEnabled = false
	if got := effectiveTaskAdmissionLimit(config, 20); got != 20 {
		t.Fatalf("disabled slice changed legacy admission limit: got %d", got)
	}
}

func TestAgentBrainCustomEnvironmentCannotOverrideTrustedValues(t *testing.T) {
	err := validateAgentBrainCustomEnvironment(
		map[string]string{"MULTICA_REQUEST_ID": "shadow"},
		map[string]string{"MULTICA_REQUEST_ID": "trusted"},
	)
	if err == nil {
		t.Fatal("trusted correlation override was accepted")
	}
	if err := runtimeenv.ValidateCustomEnvironment(map[string]string{"OPENAI_BASE_URL": "https://direct-provider.invalid"}); err == nil {
		t.Fatal("direct provider endpoint override was accepted")
	}
}

func TestAgentBrainRejectsAllCustomArgsBeforeCredentialOrLaunch(t *testing.T) {
	tests := []struct {
		name       string
		taskArgs   []string
		daemonArgs []string
	}{
		{name: "task config override", taskArgs: []string{"-c", `model_provider="direct"`}},
		{name: "task config path override", taskArgs: []string{"--config", "/synthetic/untrusted/config.toml"}},
		{name: "task model override", taskArgs: []string{"--model", "direct-model"}},
		{name: "task base URL override", taskArgs: []string{"--base-url", "https://direct.invalid"}},
		{name: "daemon config override", daemonArgs: []string{"-c", `model="direct-model"`}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			daemon, credential, launchMarker := newAgentBrainSecurityTestDaemon(t, false)
			daemon.cfg.ClaudeArgs = append([]string(nil), test.daemonArgs...)
			task := syntheticGatewayTask()
			task.Agent.CustomArgs = append([]string(nil), test.taskArgs...)

			_, err := daemon.runTask(context.Background(), task, "claude", 0, daemon.logger)
			assertAgentBrainAdmissionClass(t, err, "custom_args_not_allowed")
			if credential.calls != 0 {
				t.Fatalf("credential source called %d times before custom-argument rejection", credential.calls)
			}
			if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
				t.Fatal("synthetic launch marker exists after custom-argument rejection")
			}
		})
	}
}

func TestAgentBrainRejectsCustomRuntimeBeforeCredentialOrLaunch(t *testing.T) {
	daemon, credential, launchMarker := newAgentBrainSecurityTestDaemon(t, true)

	_, err := daemon.runTask(context.Background(), syntheticGatewayTask(), "claude", 0, daemon.logger)
	assertAgentBrainAdmissionClass(t, err, "custom_runtime_not_allowed")
	if credential.calls != 0 {
		t.Fatalf("credential source called %d times before custom-runtime rejection", credential.calls)
	}
	if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
		t.Fatal("synthetic custom executable ran after custom-runtime rejection")
	}
}

func TestAgentBrainSuppressesWorkspaceRuntimeProfiles(t *testing.T) {
	daemon, _, _ := newAgentBrainSecurityTestDaemon(t, false)
	runtimes := []map[string]string{{"type": "claude"}}
	signature := daemon.appendProfileRuntimes(context.Background(), "synthetic-workspace", &runtimes)
	if len(runtimes) != 1 || signature != profileSetSignature(nil) {
		t.Fatal("gateway-required registration did not suppress workspace runtime profiles")
	}
	if err := daemon.refreshWorkspaceRuntimeProfiles(context.Background(), "synthetic-workspace"); err != nil {
		t.Fatalf("gateway-required profile refresh was not suppressed: %v", err)
	}
}

func TestAgentBrainBuiltInResolutionIgnoresCommandPathOverride(t *testing.T) {
	binDir := t.TempDir()
	builtInPath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(builtInPath, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write synthetic built-in: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("MULTICA_CLAUDE_PATH", "/synthetic/untrusted/custom-claude")

	provider, entry, err := resolveAgentBrainBuiltInEntry(brain.CLIClaudeCode)
	if err != nil {
		t.Fatalf("resolveAgentBrainBuiltInEntry: %v", err)
	}
	if provider != "claude" || entry.Path != builtInPath {
		t.Fatalf("gateway built-in resolution used an untrusted mapping: provider=%q", provider)
	}
}

func TestCredentiallessCodexPrepareDoesNotCreateAuthState(t *testing.T) {
	root := t.TempDir()
	environment, err := execenv.Prepare(execenv.PrepareParams{
		WorkspacesRoot: root, WorkspaceID: "workspace", TaskID: "task", Provider: "codex",
		CredentiallessGateway: true,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if environment.CodexHome == "" {
		t.Fatal("credentialless Codex home missing")
	}
	if _, err := os.Stat(filepath.Join(environment.CodexHome, "auth.json")); !os.IsNotExist(err) {
		t.Fatal("credentialless Codex home contains auth.json")
	}
}

func TestAgentBrainDisabledNilPlanLaunchesNativeBackend(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := syntheticAgentBrainConfig(t, "http://127.0.0.1:1")
	config.DevelopmentEnabled = false
	config.Neutral.Gateway.Required = false
	runtime, err := newAgentBrainRuntime(config, AgentBrainDependencies{}, logger)
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	if runtime.enabled() {
		t.Fatal("synthetic Agent Brain runtime unexpectedly enabled")
	}

	marker := filepath.Join(t.TempDir(), "native-launched")
	fakeClaude := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\n: > \"$ORQ75_NATIVE_LAUNCH_MARKER\"\nexit 1\n"), 0o700); err != nil {
		t.Fatalf("write fake native backend: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	daemon := &Daemon{
		cfg: Config{
			WorkspacesRoot: t.TempDir(), ServerBaseURL: server.URL,
			Agents: map[string]AgentEntry{"claude": {Path: fakeClaude}},
		},
		client:         NewClient(server.URL),
		agentBrain:     runtime,
		logger:         logger,
		runtimeIndex:   map[string]Runtime{"native-runtime": {ID: "native-runtime", Provider: "claude"}},
		activeEnvRoots: make(map[string]int),
	}
	task := Task{
		ID: "native-task", AgentID: "native-agent", RuntimeID: "native-runtime",
		IssueID: "native-issue", WorkspaceID: "native-workspace", AuthToken: "mat_synthetic_task_scope",
		Agent: &AgentData{
			ID: "native-agent", Name: "native", CustomEnv: map[string]string{"ORQ75_NATIVE_LAUNCH_MARKER": marker},
			McpConfig: json.RawMessage(`{"mcpServers":{"synthetic":{"command":"printf"}}}`),
		},
	}

	_, runErr := daemon.runTask(context.Background(), task, "claude", 0, logger)
	var admissionErr *agentBrainAdmissionError
	if errors.As(runErr, &admissionErr) && (admissionErr.class == "launch_plan_unavailable" || admissionErr.class == "managed_mcp_not_accepted_in_g3_slice") {
		t.Fatalf("disabled native task was rejected by gateway-only launch policy: %v", runErr)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("native backend was not launched: %v (run error: %v)", err, runErr)
	}
}

func TestRunTaskGatewayRequiredRejectsUnavailablePlan(t *testing.T) {
	daemon, credential, launchMarker := newAgentBrainSecurityTestDaemon(t, false)
	disabledConfig := daemon.cfg.AgentBrain
	disabledConfig.DevelopmentEnabled = false
	disabledConfig.Neutral.Gateway.Required = false
	disabledRuntime, err := newAgentBrainRuntime(disabledConfig, AgentBrainDependencies{}, daemon.logger)
	if err != nil {
		t.Fatalf("new disabled Agent Brain runtime: %v", err)
	}
	daemon.agentBrain = disabledRuntime

	_, err = daemon.runTask(context.Background(), syntheticGatewayTask(), "claude", 0, daemon.logger)
	assertAgentBrainAdmissionClass(t, err, "gateway_required")
	if credential.calls != 0 {
		t.Fatalf("credential source called %d times for unavailable plan", credential.calls)
	}
	if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
		t.Fatal("synthetic executable ran without an admitted gateway plan")
	}
}

func TestRunTaskGatewayPlanRejectsManagedMCPBeforeLaunch(t *testing.T) {
	gatewayServer := newSyntheticGateway(t, true)
	defer gatewayServer.Close()
	apiServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	config := syntheticAgentBrainConfig(t, gatewayServer.URL)
	credential := &countingSyntheticCredentialSource{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runtime, err := newAgentBrainRuntime(config, AgentBrainDependencies{
		CredentialSource: credential,
		HTTPClient:       gatewayServer.Client(),
	}, logger)
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	launchMarker := filepath.Join(t.TempDir(), "gateway-launched")
	fakeClaude := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(fakeClaude, []byte("#!/bin/sh\n: > \""+launchMarker+"\"\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write fake gateway backend: %v", err)
	}
	daemon := &Daemon{
		cfg: Config{
			AgentBrain: config, WorkspacesRoot: t.TempDir(), ServerBaseURL: apiServer.URL,
			Agents: map[string]AgentEntry{"claude": {Path: fakeClaude}},
		},
		client:         NewClient(apiServer.URL),
		agentBrain:     runtime,
		logger:         logger,
		runtimeIndex:   map[string]Runtime{"synthetic-runtime": {ID: "synthetic-runtime", Provider: "claude"}},
		activeEnvRoots: make(map[string]int),
	}
	task := syntheticGatewayTask()
	task.Agent.McpConfig = json.RawMessage(`{"mcpServers":{"synthetic":{"command":"printf"}}}`)

	_, err = daemon.runTask(context.Background(), task, "claude", 0, logger)
	assertAgentBrainAdmissionClass(t, err, "managed_mcp_not_accepted_in_g3_slice")
	if credential.calls == 0 {
		t.Fatal("gateway admission did not reach authenticated readiness")
	}
	if _, statErr := os.Stat(launchMarker); !os.IsNotExist(statErr) {
		t.Fatal("synthetic executable ran after managed-MCP gateway rejection")
	}
}

func TestAgentBrainEnabledNilPlanStillFailsClosed(t *testing.T) {
	runtime, err := newAgentBrainRuntime(
		syntheticAgentBrainConfig(t, "http://127.0.0.1:1"),
		AgentBrainDependencies{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	_, err = runtime.buildLaunch(context.Background(), nil, &execenv.Environment{RootDir: t.TempDir()}, nil, nil)
	assertAgentBrainAdmissionClass(t, err, "launch_plan_unavailable")
}

func newAgentBrainSecurityTestDaemon(t *testing.T, customRuntime bool) (*Daemon, *countingSyntheticCredentialSource, string) {
	t.Helper()
	config := syntheticAgentBrainConfig(t, "http://127.0.0.1:1")
	credential := &countingSyntheticCredentialSource{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runtime, err := newAgentBrainRuntime(config, AgentBrainDependencies{CredentialSource: credential}, logger)
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	launchMarker := filepath.Join(t.TempDir(), "launched")
	customPath := filepath.Join(t.TempDir(), "custom-claude")
	script := "#!/bin/sh\n: > \"" + launchMarker + "\"\nexit 0\n"
	if err := os.WriteFile(customPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write synthetic custom executable: %v", err)
	}
	profileID := ""
	if customRuntime {
		profileID = "synthetic-custom-profile"
	}
	daemon := &Daemon{
		cfg: Config{
			AgentBrain: config,
			Agents:     map[string]AgentEntry{"claude": {Path: customPath}},
		},
		agentBrain: runtime,
		logger:     logger,
		runtimeIndex: map[string]Runtime{
			"synthetic-runtime": {ID: "synthetic-runtime", Provider: "claude", ProfileID: profileID},
		},
		profileCommandPaths: map[string]string{profileID: customPath},
	}
	return daemon, credential, launchMarker
}

func assertAgentBrainAdmissionClass(t *testing.T, err error, want string) {
	t.Helper()
	var admissionErr *agentBrainAdmissionError
	if !errors.As(err, &admissionErr) || admissionErr.class != want {
		t.Fatalf("admission error class=%v, want %q", err, want)
	}
}

func syntheticAgentBrainConfig(t *testing.T, baseURL string) AgentBrainIntegrationConfig {
	t.Helper()
	secretRef, err := brain.NewSecretFileRef("/synthetic/omniroute/reference")
	if err != nil {
		t.Fatalf("NewSecretFileRef: %v", err)
	}
	return AgentBrainIntegrationConfig{
		DevelopmentEnabled: true,
		Neutral: brain.Config{
			ControlURL: "ws://synthetic-control.invalid/ws",
			Gateway: brain.GatewayConfig{
				Required: true, BaseURL: baseURL, SecretFile: secretRef, Readiness: brain.StrictReadinessPolicy(),
			},
			CapacityTier: brain.CapacityTier20,
		},
		CLIKind: brain.CLIClaudeCode, RouteModel: brain.RouteModel("agy/claude-opus-4-6-thinking"),
	}
}

func syntheticGatewayTask() Task {
	return Task{
		ID: "synthetic-task", AgentID: "synthetic-agent", RuntimeID: "synthetic-runtime",
		IssueID: "synthetic-issue", WorkspaceID: "synthetic-workspace",
		AuthToken: "mat_synthetic_task_scope",
		Agent:     &AgentData{ID: "synthetic-agent", Name: "synthetic", Model: "agy/claude-opus-4-6-thinking"},
	}
}

func newSyntheticGateway(t *testing.T, ready bool) *httptest.Server {
	t.Helper()
	available := true
	enabled := true
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/health/ping":
			if !ready {
				response.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			response.WriteHeader(http.StatusOK)
			return
		case "/v1/models":
			if request.Header.Get("Authorization") != "Bearer "+syntheticReferenceSecret {
				response.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !ready {
				response.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		default:
			response.WriteHeader(http.StatusNotFound)
			return
		}
		writeSyntheticModels(response, available, enabled)
	}))
	return server
}

func writeSyntheticModels(response http.ResponseWriter, available, enabled bool) {
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(gateway.ModelsDocument{
		Object: "list", RegistryVersion: "synthetic-v1",
		Models: []gateway.ModelDocument{{
			ID: "agy/claude-opus-4-6-thinking", Protocol: string(brain.ProtocolAnthropicMessages),
			Streaming: &enabled, Tools: &enabled, Reasoning: &enabled, StructuredOutput: &enabled,
			ContextLimit: 1000, AccountPool: "synthetic-pool", Rotation: string(gateway.RotationStrictIndependentRequest),
			Affinity: string(gateway.AffinityOriginAccount), Available: &available,
		}},
	})
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

type agentBrainDiagnosticSnapshot struct {
	launchLogs           int
	terminalLogs         int
	routeSelectionEvents int
	cancellationEvents   int
	terminalOutcomes     map[string]int
}

type agentBrainDiagnosticRecorder struct {
	mu                   sync.Mutex
	launchLogs           int
	terminalLogs         int
	routeSelectionEvents int
	cancellationEvents   int
	terminalOutcomes     map[string]int
}

func (r *agentBrainDiagnosticRecorder) Enabled(context.Context, slog.Level) bool {
	return true
}

func (r *agentBrainDiagnosticRecorder) Handle(_ context.Context, record slog.Record) error {
	kind := ""
	outcome := ""
	record.Attrs(func(attribute slog.Attr) bool {
		switch attribute.Key {
		case "kind":
			kind = attribute.Value.String()
		case "outcome":
			outcome = attribute.Value.String()
		}
		return true
	})

	r.mu.Lock()
	defer r.mu.Unlock()
	switch record.Message {
	case "agent brain launch":
		r.launchLogs++
	case "agent brain terminal":
		r.terminalLogs++
		r.terminalOutcomes[outcome]++
	case "agent brain event":
		switch kind {
		case "route.selection":
			r.routeSelectionEvents++
		case "request.cancellation":
			r.cancellationEvents++
		}
	}
	return nil
}

func (r *agentBrainDiagnosticRecorder) WithAttrs([]slog.Attr) slog.Handler {
	return r
}

func (r *agentBrainDiagnosticRecorder) WithGroup(string) slog.Handler {
	return r
}

func (r *agentBrainDiagnosticRecorder) snapshot() agentBrainDiagnosticSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	outcomes := make(map[string]int, len(r.terminalOutcomes))
	for outcome, count := range r.terminalOutcomes {
		outcomes[outcome] = count
	}
	return agentBrainDiagnosticSnapshot{
		launchLogs: r.launchLogs, terminalLogs: r.terminalLogs,
		routeSelectionEvents: r.routeSelectionEvents, cancellationEvents: r.cancellationEvents,
		terminalOutcomes: outcomes,
	}
}

func newAgentBrainLifecycleDiagnosticTest(t *testing.T) (*agentBrainRuntime, *agentBrainTaskPlan, *agentBrainDiagnosticRecorder) {
	t.Helper()
	capacity, err := brain.NewLifecycleCapacity(1)
	if err != nil {
		t.Fatalf("NewLifecycleCapacity: %v", err)
	}
	attempt, decision := capacity.TryBegin()
	if attempt == nil || !decision.Admitted() {
		t.Fatalf("TryBegin: attempt=%v decision=%+v", attempt, decision)
	}
	lease := attempt.Admit()
	if lease == nil {
		t.Fatal("Admit returned a nil capacity lease")
	}
	recorder := &agentBrainDiagnosticRecorder{terminalOutcomes: make(map[string]int)}
	runtime := &agentBrainRuntime{
		config: AgentBrainIntegrationConfig{
			CLIKind:    brain.CLIClaudeCode,
			RouteModel: brain.RouteModel("agy/claude-opus-4-6-thinking"),
		},
		logger:   slog.New(recorder),
		capacity: capacity,
	}
	plan := &agentBrainTaskPlan{
		Task: brain.Task{Request: brain.TaskRequest{
			Correlation: brain.Correlation{TaskID: "synthetic-task", SessionID: "synthetic-session", RequestID: "synthetic-request"},
			CLIKind:     brain.CLIClaudeCode, RouteModel: runtime.config.RouteModel, RouterOwner: brain.RouterOwnerOmniRoute,
		}},
		Capacity: lease,
	}
	return runtime, plan, recorder
}

func runAgentBrainConcurrentCalls(count int, call func(int)) {
	ready := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(count)
	for index := 0; index < count; index++ {
		go func(index int) {
			defer workers.Done()
			<-ready
			call(index)
		}(index)
	}
	close(ready)
	workers.Wait()
}

func terminalDiagnosticCount(outcomes map[string]int) int {
	total := 0
	for _, count := range outcomes {
		total += count
	}
	return total
}

// --- D6: buildLaunch OpenAI-compatible (Cline) credentialless launch ---
// Cline (CLIOpenAICompatible) rides the shared OpenAI Chat Completions gateway
// contract. buildLaunch must materialize the non-secret providers.json carrier
// in a controlled in-root CLINE_DATA_DIR and inject the stable OmniRoute secret
// only through the environment. The existing Claude/Codex cases are untouched.

func syntheticClineAgentBrainRuntime(t *testing.T) *agentBrainRuntime {
	t.Helper()
	config := syntheticAgentBrainConfig(t, "http://127.0.0.1:20128")
	config.CLIKind = brain.CLIOpenAICompatible
	config.RouteModel = brain.RouteModel("cp/cline-pass/glm-5.2")
	runtime, err := newAgentBrainRuntime(config, AgentBrainDependencies{
		CredentialSource: syntheticCredentialSource{},
		InheritedEnvironment: func() []string {
			return []string{
				"PATH=" + os.Getenv("PATH"),
				"HOME=/synthetic/provider-home",
				"OPENAI_API_KEY=synthetic-provider-value",
				"OPENAI_BASE_URL=https://direct-provider.invalid/v1",
				"NVIDIA_API_KEY=synthetic-provider-value",
			}
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("newAgentBrainRuntime (cline): %v", err)
	}
	return runtime
}

func syntheticClinePlan(route string) *agentBrainTaskPlan {
	return &agentBrainTaskPlan{Task: brain.Task{Request: brain.TaskRequest{
		CLIKind:     brain.CLIOpenAICompatible,
		RouteModel:  brain.RouteModel(route),
		RouterOwner: brain.RouterOwnerOmniRoute,
		Correlation: brain.Correlation{TaskID: "task-cline", SessionID: "session-cline", RequestID: "request-cline"},
	}}}
}

func TestBuildLaunchOpenAICompatibleWiresClineDataDirAndSecret(t *testing.T) {
	runtime := syntheticClineAgentBrainRuntime(t)
	plan := syntheticClinePlan("cp/cline-pass/glm-5.2")

	envRoot := t.TempDir()
	workDir := filepath.Join(envRoot, "workdir")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("create workdir: %v", err)
	}
	prepared := &execenv.Environment{RootDir: envRoot, WorkDir: workDir}

	launch, err := runtime.buildLaunch(context.Background(), plan, prepared, map[string]string{
		"G3_SYNTHETIC_CHILD_CLINE": "1",
	}, map[string]string{"SAFE_CUSTOM_SETTING": "synthetic"})
	if err != nil {
		t.Fatalf("buildLaunch (cline): %v", err)
	}

	keys := launch.Environment.Keys()
	for _, required := range []string{"CLINE_DATA_DIR", "CLINE_OMNIROUTE_API_KEY", "MULTICA_SESSION_ID", "MULTICA_REQUEST_ID", "MULTICA_ROUTER_OWNER"} {
		if !containsString(keys, required) {
			t.Fatalf("required child key missing: %s", required)
		}
	}
	for _, forbidden := range []string{"OPENAI_API_KEY", "OPENAI_BASE_URL", "NVIDIA_API_KEY", "CODEX_HOME", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_BASE_URL"} {
		if containsString(keys, forbidden) {
			t.Fatalf("forbidden child key present: %s", forbidden)
		}
	}

	// The controlled Cline data dir is under the execution root; CLINE_DATA_DIR
	// is pinned to it.
	clineDataDir := filepath.Join(envRoot, "cline-data")
	if !containsString(launch.Environment.Exec(), "CLINE_DATA_DIR="+clineDataDir) {
		t.Fatalf("CLINE_DATA_DIR not pinned to the controlled in-root dir %q", clineDataDir)
	}

	// The non-secret providers.json carrier is materialized with the reference
	// sentinel and never the resolved secret value.
	carrier := filepath.Join(clineDataDir, "settings", "providers.json")
	raw, err := os.ReadFile(carrier)
	if err != nil {
		t.Fatalf("read cline carrier: %v", err)
	}
	if !strings.Contains(string(raw), runtimeenv.ClineSecretReferenceSentinel) {
		t.Fatal("cline carrier missing the reference sentinel")
	}
	if strings.Contains(string(raw), syntheticReferenceSecret) {
		t.Fatal("cline carrier embedded the stable OmniRoute secret value")
	}

	// The child process actually receives the controlled environment.
	command := exec.Command(os.Args[0], "-test.run=TestAgentBrainSyntheticChildCline")
	command.Env = launch.Environment.Exec()
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		t.Fatalf("synthetic cline isolation child: %v", err)
	}
}

func TestAgentBrainSyntheticChildCline(t *testing.T) {
	if os.Getenv("G3_SYNTHETIC_CHILD_CLINE") != "1" {
		return
	}
	for _, forbidden := range []string{
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "NVIDIA_API_KEY", "NIM_BASE_URL",
		"CODEX_HOME", "ANTHROPIC_AUTH_TOKEN",
	} {
		if _, present := os.LookupEnv(forbidden); present {
			os.Exit(51)
		}
	}
	dataDir, present := os.LookupEnv("CLINE_DATA_DIR")
	if !present || dataDir == "" {
		os.Exit(52)
	}
	if _, present := os.LookupEnv("CLINE_OMNIROUTE_API_KEY"); !present {
		os.Exit(53)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "settings", "providers.json")); err != nil {
		os.Exit(54)
	}
	if os.Getenv("HOME") == "" {
		os.Exit(55)
	}
	os.Exit(0)
}

func TestBuildLaunchOpenAICompatibleRejectsNVIDIAOwnedRoute(t *testing.T) {
	runtime := syntheticClineAgentBrainRuntime(t)
	// NVIDIA is an OmniRoute-owned fallback the Brain must never select for a
	// Cline task; the launch stage must fail closed and write no carrier.
	plan := syntheticClinePlan("nvidia/z-ai/glm-5.2")

	envRoot := t.TempDir()
	workDir := filepath.Join(envRoot, "workdir")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("create workdir: %v", err)
	}
	prepared := &execenv.Environment{RootDir: envRoot, WorkDir: workDir}

	if _, err := runtime.buildLaunch(context.Background(), plan, prepared, map[string]string{}, nil); err == nil {
		t.Fatal("buildLaunch admitted an OmniRoute-owned NVIDIA route for a Cline task")
	}
	if _, statErr := os.Stat(filepath.Join(envRoot, "cline-data", "settings", "providers.json")); !os.IsNotExist(statErr) {
		t.Fatal("cline carrier written for a rejected NVIDIA route")
	}
}
