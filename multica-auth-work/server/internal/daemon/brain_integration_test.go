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
		"MULTICA_PRODEX_ENABLED", "MULTICA_L2_ENABLED",
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
	task.RuntimeRouterOwner = string(brain.RouterOwnerLegacyRustL2)
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

func TestAgentBrainDevelopmentSkipsLegacyStartup(t *testing.T) {
	config := Config{
		ServerBaseURL: "ws://127.0.0.1:1/ws", WorkspacesRoot: t.TempDir(),
		AgentBrain:          syntheticAgentBrainConfig(t, "http://127.0.0.1:20128"),
		RotationDatabaseURL: "synthetic-invalid-database-reference",
		Prodex:              ProdexConfig{Enabled: true, Required: true},
		L2Runtime:           L2RuntimeConfig{Enabled: true},
	}
	daemon := NewWithAgentBrainDependencies(config, slog.New(slog.NewTextHandler(io.Discard, nil)), AgentBrainDependencies{})
	if daemon.rotationStore != nil || daemon.rotationService != nil || daemon.l2Client != nil || daemon.l2Sidecar != nil {
		t.Fatal("legacy rotation or Prodex/L2 startup initialized in Agent Brain development mode")
	}
}

func TestAgentBrainOwnerSuppressesLegacyGoRotation(t *testing.T) {
	daemon := &Daemon{}
	task := Task{ID: "synthetic-task", RuntimeRouterOwner: string(brain.RouterOwnerOmniRoute)}
	if daemon.legacyGoRotationAllowed(task, nil, "synthetic") {
		t.Fatal("legacy Go rotation remained enabled for an OmniRoute-owned task")
	}
}

func TestAgentBrainRejectsUnsafeLegacyGatewayAlias(t *testing.T) {
	t.Setenv("MULTICA_L2_BASE_URL", "http://legacy-router.invalid")
	enabled := true
	required := true
	_, err := loadAgentBrainIntegrationConfig(Overrides{
		AgentBrainDevelopment:  &enabled,
		AgentBrainGateway:      &required,
		AgentBrainControlURL:   "ws://synthetic-control.invalid/ws",
		AgentBrainSecretFile:   "/synthetic/omniroute/reference",
		AgentBrainCLIKind:      string(brain.CLIClaudeCode),
		AgentBrainRouteModel:   "agy/claude-opus-4-6-thinking",
		AgentBrainCapacityTier: 20,
	}, "ws://legacy-control.invalid/ws")
	if err == nil {
		t.Fatal("semantically unsafe legacy gateway alias was accepted")
	}
}

func TestAgentBrainNeutralGatewayAliasWinsAndIsMeasured(t *testing.T) {
	t.Setenv("MULTICA_L2_BASE_URL", "http://legacy-router.invalid")
	enabled := true
	required := true
	config, err := loadAgentBrainIntegrationConfig(Overrides{
		AgentBrainDevelopment:  &enabled,
		AgentBrainGateway:      &required,
		AgentBrainControlURL:   "ws://synthetic-control.invalid/ws",
		AgentBrainGatewayURL:   "http://127.0.0.1:20128",
		AgentBrainSecretFile:   "/synthetic/omniroute/reference",
		AgentBrainCLIKind:      string(brain.CLIClaudeCode),
		AgentBrainRouteModel:   "agy/claude-opus-4-6-thinking",
		AgentBrainCapacityTier: 20,
	}, "ws://legacy-control.invalid/ws")
	if err != nil {
		t.Fatalf("loadAgentBrainIntegrationConfig: %v", err)
	}
	if config.Neutral.Gateway.BaseURL != "http://127.0.0.1:20128" || len(config.LegacyUses) == 0 {
		t.Fatalf("neutral precedence or measurable legacy use missing")
	}
}

func TestAgentBrainLegacyMigrationFlagIsExplicitAndMutuallyExclusive(t *testing.T) {
	legacy := AgentBrainIntegrationConfig{
		DevelopmentEnabled: true,
		Neutral: brain.Config{
			ControlURL:   "ws://synthetic-control.invalid/ws",
			Gateway:      brain.GatewayConfig{BaseURL: brain.DefaultHostGatewayURL, Readiness: brain.StrictReadinessPolicy()},
			CapacityTier: brain.CapacityTier20, LegacyExecution: true,
		},
	}
	if err := legacy.Validate(); err != nil {
		t.Fatalf("explicit legacy migration mode rejected: %v", err)
	}
	legacy.Neutral.Gateway.Required = true
	secretRef, err := brain.NewSecretFileRef("/synthetic/omniroute/reference")
	if err != nil {
		t.Fatalf("NewSecretFileRef: %v", err)
	}
	legacy.Neutral.Gateway.SecretFile = secretRef
	if err := legacy.Validate(); err == nil {
		t.Fatal("legacy and OmniRoute router modes were enabled together")
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
		case "/health/live":
			response.WriteHeader(http.StatusNoContent)
			return
		case "/health/ready", "/v1/models":
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
		if request.URL.Path == "/health/ready" {
			response.WriteHeader(http.StatusNoContent)
			return
		}
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
	}))
	return server
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
