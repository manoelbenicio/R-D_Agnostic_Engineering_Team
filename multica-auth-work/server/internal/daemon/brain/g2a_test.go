package brain

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestStaticRuntimeRegistryUsesCLIKindOnly(t *testing.T) {
	registry, err := NewStaticRuntimeRegistry(
		RuntimeDescriptor{CLIKind: CLIClaudeCode, Path: "claude"},
		RuntimeDescriptor{CLIKind: CLICodex, Path: "codex"},
	)
	if err != nil {
		t.Fatalf("NewStaticRuntimeRegistry: %v", err)
	}
	got, err := registry.ResolveRuntime(context.Background(), CLIClaudeCode)
	if err != nil {
		t.Fatalf("ResolveRuntime: %v", err)
	}
	if got.CLIKind != CLIClaudeCode || got.Path != "claude" {
		t.Fatalf("unexpected runtime: %+v", got)
	}
	if _, err := registry.ResolveRuntime(context.Background(), CLIKimi); !errors.Is(err, ErrRuntimeNotFound) {
		t.Fatalf("missing runtime error = %v, want ErrRuntimeNotFound", err)
	}
	if _, err := NewStaticRuntimeRegistry(
		RuntimeDescriptor{CLIKind: CLICodex, Path: "codex-a"},
		RuntimeDescriptor{CLIKind: CLICodex, Path: "codex-b"},
	); err == nil {
		t.Fatal("duplicate CLIKind runtime was accepted")
	}
}

func TestLifecycleTaskExecutorDelegatesPreservedLifecycle(t *testing.T) {
	registry, err := NewStaticRuntimeRegistry(RuntimeDescriptor{CLIKind: CLICodex, Path: "codex"})
	if err != nil {
		t.Fatalf("NewStaticRuntimeRegistry: %v", err)
	}
	lifecycle := &fakePreservedLifecycle{result: TaskResult{Status: TaskStatusCompleted}}
	executor, err := NewLifecycleTaskExecutor(registry, lifecycle)
	if err != nil {
		t.Fatalf("NewLifecycleTaskExecutor: %v", err)
	}
	task := validTask(true)
	task.Request.CLIKind = CLICodex
	result, err := executor.ExecuteTask(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != TaskStatusCompleted || lifecycle.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, lifecycle.calls)
	}
	if lifecycle.request.Runtime.CLIKind != CLICodex || lifecycle.request.Task.Request.RouteModel != task.Request.RouteModel {
		t.Fatalf("lifecycle request lost neutral fields: %+v", lifecycle.request)
	}
	if lifecycle.request.Task.Lifecycle.WorktreeRef != task.Lifecycle.WorktreeRef || lifecycle.request.Task.Lifecycle.StreamPolicyRef != task.Lifecycle.StreamPolicyRef {
		t.Fatalf("lifecycle request lost cold-plane references: %+v", lifecycle.request)
	}
}

func TestGatewayAdmissionFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		snapshot   ReadinessSnapshot
		wantState  AdmissionState
		wantStatus TaskStatus
	}{
		{
			name:       "gateway unavailable",
			snapshot:   ReadinessSnapshot{},
			wantState:  AdmissionGatewayUnavailable,
			wantStatus: TaskStatusGatewayUnavailable,
		},
		{
			name:       "authentication failed",
			snapshot:   ReadinessSnapshot{Live: true},
			wantState:  AdmissionGatewayAuthFailed,
			wantStatus: TaskStatusGatewayAuthFailed,
		},
		{
			name:       "model registry unavailable",
			snapshot:   ReadinessSnapshot{Live: true, Authenticated: true},
			wantState:  AdmissionCapabilityRejected,
			wantStatus: TaskStatusCapabilityRejected,
		},
		{
			name:       "model unavailable",
			snapshot:   ReadinessSnapshot{Live: true, Authenticated: true, ModelRegistryReady: true},
			wantState:  AdmissionCapabilityRejected,
			wantStatus: TaskStatusCapabilityRejected,
		},
		{
			name: "protocol unavailable",
			snapshot: ReadinessSnapshot{
				Live: true, Authenticated: true, ModelRegistryReady: true, SelectedModelReady: true,
			},
			wantState:  AdmissionCapabilityRejected,
			wantStatus: TaskStatusCapabilityRejected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := mustAdmissionController(t, &fakeReadinessChecker{snapshot: tt.snapshot})
			decision, err := controller.Admit(context.Background(), validTask(true))
			if err != nil {
				t.Fatalf("Admit: %v", err)
			}
			if decision.State != tt.wantState || decision.TaskStatus != tt.wantStatus || decision.Admitted() {
				t.Fatalf("unexpected decision: %+v", decision)
			}
		})
	}
}

func TestGatewayAdmissionReadyAndLegacyBypass(t *testing.T) {
	checker := &fakeReadinessChecker{snapshot: ReadinessSnapshot{
		Live: true, Authenticated: true, ModelRegistryReady: true,
		SelectedModelReady: true, SelectedProtocolReady: true,
	}}
	controller := mustAdmissionController(t, checker)
	decision, err := controller.Admit(context.Background(), validTask(true))
	if err != nil || !decision.Admitted() || decision.ReadinessState != GatewayReadinessReady {
		t.Fatalf("gateway admission decision=%+v err=%v", decision, err)
	}

	legacy := validTask(false)
	legacy.RoutePolicy.Approved = false
	decision, err = controller.Admit(context.Background(), legacy)
	if err != nil || !decision.Admitted() || decision.ReadinessState != GatewayReadinessNotRequired {
		t.Fatalf("legacy admission decision=%+v err=%v", decision, err)
	}
	if checker.calls != 1 {
		t.Fatalf("readiness calls=%d, want 1", checker.calls)
	}
}

func TestGatewayAdmissionRejectsUnapprovedPolicyWithoutProbe(t *testing.T) {
	checker := &fakeReadinessChecker{}
	controller := mustAdmissionController(t, checker)
	task := validTask(true)
	task.RoutePolicy.Approved = false
	decision, err := controller.Admit(context.Background(), task)
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if decision.State != AdmissionRoutePolicyRejected || decision.TaskStatus != TaskStatusCapabilityRejected {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if checker.calls != 0 {
		t.Fatalf("readiness called %d times for rejected policy", checker.calls)
	}
}

func TestCoordinatorRejectsBeforeExecutionAndPublishesOnce(t *testing.T) {
	executor := &fakeTaskExecutor{}
	sink := &fakeResultSink{}
	coordinator, err := NewCoordinator(
		admissionFunc(func(context.Context, Task) (AdmissionDecision, error) {
			return unavailableDecision(), nil
		}),
		executor,
		sink,
	)
	if err != nil {
		t.Fatalf("NewCoordinator: %v", err)
	}
	result, err := coordinator.Run(context.Background(), validTask(true))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != TaskStatusGatewayUnavailable || !result.Retryable {
		t.Fatalf("unexpected result: %+v", result)
	}
	if executor.calls != 0 || sink.calls != 1 {
		t.Fatalf("executor calls=%d sink calls=%d", executor.calls, sink.calls)
	}
}

func TestCoordinatorPreservesCancellationAndTerminalResult(t *testing.T) {
	executor := &fakeTaskExecutor{err: context.Canceled}
	sink := &fakeResultSink{}
	coordinator, err := NewCoordinator(
		admissionFunc(func(context.Context, Task) (AdmissionDecision, error) {
			return AdmissionDecision{State: AdmissionAdmitted}, nil
		}),
		executor,
		sink,
	)
	if err != nil {
		t.Fatalf("NewCoordinator: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := coordinator.Run(ctx, validTask(true))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error=%v, want context.Canceled", err)
	}
	if result.Status != TaskStatusCancelled || sink.calls != 1 || sink.contextErr != nil {
		t.Fatalf("result=%+v sink calls=%d sink context err=%v", result, sink.calls, sink.contextErr)
	}
}

func TestCompatibilityTranslatorEmitsBoundedMeasurements(t *testing.T) {
	recorder := NewMemoryLegacyUseRecorder()
	translator, err := NewCompatibilityTranslator(recorder)
	if err != nil {
		t.Fatalf("NewCompatibilityTranslator: %v", err)
	}
	policy := validTask(true).RoutePolicy
	_, err = translator.TranslateTask(context.Background(), LegacyTaskInput{
		Provider: "claude", Model: "agy/claude-opus-4-6-thinking", AuthToken: "mat_test_only",
	}, Correlation{TaskID: "task", SessionID: "session", RequestID: "request"}, policy, validTask(true).Lifecycle, true)
	if err != nil {
		t.Fatalf("TranslateTask: %v", err)
	}
	measurements := recorder.Snapshot()
	if len(measurements) != 2 {
		t.Fatalf("measurements=%+v, want 2", measurements)
	}
	for _, measurement := range measurements {
		if measurement.Count != 1 || measurement.Outcome != LegacyUseTranslated {
			t.Fatalf("unexpected measurement: %+v", measurement)
		}
		if fmt.Sprint(measurement) == "mat_test_only" {
			t.Fatal("measurement retained a task token")
		}
	}
}

func TestCompatibilityConfigTranslationMeasuresShadowAndRejectsUnsafeAlias(t *testing.T) {
	recorder := NewMemoryLegacyUseRecorder()
	translator, err := NewCompatibilityTranslator(recorder)
	if err != nil {
		t.Fatalf("NewCompatibilityTranslator: %v", err)
	}
	resolved, err := translator.ResolveConfig(context.Background(),
		ConfigCandidate{Name: EnvControlURL, Value: "neutral", Source: SourceNeutralEnv, Set: true},
		ConfigCandidate{Name: "MULTICA_SERVER_URL", Value: "legacy", Source: SourceLegacyEnv, Set: true},
	)
	if err != nil || resolved.Name != EnvControlURL {
		t.Fatalf("ResolveConfig=%+v err=%v", resolved, err)
	}
	if _, err := translator.ResolveConfig(context.Background(),
		ConfigCandidate{Name: "MULTICA_L2_BASE_URL", Value: "legacy", Source: SourceLegacyEnv, Set: true},
	); err == nil {
		t.Fatal("semantically incompatible legacy alias was accepted")
	}
	measurements := recorder.Snapshot()
	if len(measurements) != 2 || measurements[0].Count != 1 || measurements[1].Count != 1 {
		t.Fatalf("unexpected measurements: %+v", measurements)
	}
}

func validTask(gatewayRequired bool) Task {
	owner := RouterOwnerOmniRoute
	if !gatewayRequired {
		owner = RouterOwnerLegacyNativeCLI
	}
	return Task{
		Request: TaskRequest{
			Version:         ContractVersion,
			Correlation:     Correlation{TaskID: "task", SessionID: "session", RequestID: "request"},
			CLIKind:         CLIClaudeCode,
			RouteModel:      RouteModel("agy/claude-opus-4-6-thinking"),
			RouterOwner:     owner,
			RoutePolicyID:   "canary-20",
			GatewayRequired: gatewayRequired,
		},
		RoutePolicy: ApprovedRoutePolicy{
			ID: "canary-20", Revision: "v1", Protocol: ProtocolAnthropicMessages, Approved: true,
		},
		Lifecycle: LifecycleBindings{
			WorkspaceRef: "workspace", WorktreeRef: "worktree", ContextRef: "context",
			SkillRefs: []string{"skills"}, RecoveryRef: "recovery", WatchdogPolicyRef: "watchdog",
			StreamPolicyRef: "stream", TerminalPolicyRef: "terminal",
		},
	}
}

func mustAdmissionController(t *testing.T, checker GatewayReadinessChecker) *GatewayAdmissionController {
	t.Helper()
	controller, err := NewGatewayAdmissionController(checker, StrictReadinessPolicy())
	if err != nil {
		t.Fatalf("NewGatewayAdmissionController: %v", err)
	}
	return controller
}

type fakeReadinessChecker struct {
	snapshot ReadinessSnapshot
	err      error
	calls    int
}

func (f *fakeReadinessChecker) CheckGatewayReadiness(context.Context, ReadinessRequest) (ReadinessSnapshot, error) {
	f.calls++
	return f.snapshot, f.err
}

type fakePreservedLifecycle struct {
	request LifecycleRequest
	result  TaskResult
	err     error
	calls   int
}

func (f *fakePreservedLifecycle) ExecuteLifecycle(_ context.Context, request LifecycleRequest) (TaskResult, error) {
	f.calls++
	f.request = request
	return f.result, f.err
}

type fakeTaskExecutor struct {
	result TaskResult
	err    error
	calls  int
}

func (f *fakeTaskExecutor) ExecuteTask(context.Context, Task) (TaskResult, error) {
	f.calls++
	return f.result, f.err
}

type fakeResultSink struct {
	result     TaskResult
	contextErr error
	err        error
	calls      int
}

func (f *fakeResultSink) PublishResult(ctx context.Context, result TaskResult) error {
	f.calls++
	f.result = result
	f.contextErr = ctx.Err()
	return f.err
}

type admissionFunc func(context.Context, Task) (AdmissionDecision, error)

func (f admissionFunc) Admit(ctx context.Context, task Task) (AdmissionDecision, error) {
	return f(ctx, task)
}
