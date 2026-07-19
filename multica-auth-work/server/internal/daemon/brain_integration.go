package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/deploy"
	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
	"github.com/multica-ai/multica/server/internal/daemon/observability"
	"github.com/multica-ai/multica/server/internal/daemon/runtimeenv"
)

const (
	agentBrainDevelopmentMaxTasks = 1
	agentBrainRouteRevision       = "g3-development-v1"
)

// AgentBrainDependencies are injected only by the default-off development
// slice. No file-backed credential reader is provided while PD-08 remains in
// force; normal command startup therefore fails closed if the slice is
// enabled without an explicitly authorized source.
type AgentBrainDependencies struct {
	CredentialSource     gateway.CredentialSource
	HTTPClient           *http.Client
	InheritedEnvironment func() []string
}

type agentBrainRuntime struct {
	config         AgentBrainIntegrationConfig
	dependencies   AgentBrainDependencies
	logger         *slog.Logger
	legacy         *brain.CompatibilityTranslator
	legacyRecorder *brain.MemoryLegacyUseRecorder
	capacity       *brain.LifecycleCapacity
	requestSeq     atomic.Uint64

	diagnosticsMu sync.RWMutex
	diagnostics   agentBrainDiagnostics
}

type agentBrainDiagnostics struct {
	State          string
	Readiness      brain.GatewayReadinessState
	RouterOwner    brain.RouterOwner
	CLIKind        brain.CLIKind
	RouteModel     brain.RouteModel
	Protocol       brain.ProtocolFamily
	Profile        gateway.ProfileID
	LastOutcome    string
	LegacyUseCount uint64
	Capacity       brain.CapacityCounters
}

type agentBrainTaskPlan struct {
	Task       brain.Task
	Profile    gateway.RuntimeProfile
	Capability brain.ModelCapability
	Capacity   *brain.CapacityLease
}

type agentBrainLaunch struct {
	Environment runtimeenv.ChildEnvironment
	CodexConfig *runtimeenv.CodexConfigContract
}

type agentBrainAdmissionError struct {
	class     string
	retryable bool
}

func (e *agentBrainAdmissionError) Error() string {
	if e == nil || e.class == "" {
		return "agent brain admission failed closed"
	}
	return "agent brain admission failed closed: " + e.class
}

func newAgentBrainRuntime(config AgentBrainIntegrationConfig, dependencies AgentBrainDependencies, logger *slog.Logger) (*agentBrainRuntime, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	capacity, err := brain.NewLifecycleCapacity(agentBrainDevelopmentMaxTasks)
	if err != nil {
		return nil, err
	}
	recorder := brain.NewMemoryLegacyUseRecorder()
	translator, err := brain.NewCompatibilityTranslator(recorder)
	if err != nil {
		return nil, err
	}
	runtime := &agentBrainRuntime{
		config: config, dependencies: dependencies, logger: logger, legacy: translator, legacyRecorder: recorder,
		capacity:    capacity,
		diagnostics: agentBrainDiagnostics{State: "disabled", Readiness: brain.GatewayReadinessNotRequired},
	}
	if !config.DevelopmentEnabled {
		return runtime, nil
	}
	if config.Neutral.LegacyExecution {
		runtime.diagnostics = agentBrainDiagnostics{
			State: "legacy-migration", Readiness: brain.GatewayReadinessNotRequired,
			RouterOwner: brain.RouterOwnerLegacyNativeCLI,
		}
		return runtime, nil
	}
	if err := deploy.DefaultRolloutPlan().Validate(); err != nil {
		return nil, fmt.Errorf("agent brain rollout contract: %w", err)
	}
	if err := observability.DefaultTelemetrySchema().Validate(); err != nil {
		return nil, fmt.Errorf("agent brain telemetry contract: %w", err)
	}
	policy := gateway.FrozenTier20CanaryPolicy()
	if err := policy.Validate(config.RouteModel); err != nil {
		return nil, fmt.Errorf("agent brain route policy: %w", err)
	}
	adapter, err := runtimeenv.CredentiallessAdapterContract(config.CLIKind)
	if err != nil {
		return nil, err
	}
	profile, err := gateway.LookupRuntimeProfile(adapter.Protocol, config.CLIKind)
	if err != nil {
		return nil, err
	}
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	runtime.diagnostics = agentBrainDiagnostics{
		State: "configured-default-off", Readiness: brain.GatewayReadinessUnavailable,
		RouterOwner: brain.RouterOwnerOmniRoute, CLIKind: config.CLIKind, RouteModel: config.RouteModel,
		Protocol: adapter.Protocol, Profile: profile.ID,
	}
	return runtime, nil
}

func (r *agentBrainRuntime) enabled() bool {
	return r != nil && r.config.DevelopmentEnabled && r.config.Neutral.Gateway.Required && !r.config.Neutral.LegacyExecution
}

func (r *agentBrainRuntime) admitTask(ctx context.Context, task Task, provider, legacyModel string) (*agentBrainTaskPlan, error) {
	if !r.enabled() {
		return nil, nil
	}
	correlation := r.newCorrelation(task)
	attempt, capacityDecision := r.capacity.TryBegin()
	if !capacityDecision.Admitted() {
		r.recordAdmission(correlation, capacityDecision.State, brain.GatewayReadinessNotRequired, capacityDecision.ErrorClass)
		return nil, &agentBrainAdmissionError{class: capacityDecision.ErrorClass, retryable: capacityDecision.Retryable}
	}
	committed := false
	defer func() {
		if !committed {
			attempt.Reject()
		}
	}()
	policy := gateway.FrozenTier20CanaryPolicy()
	adapter, err := runtimeenv.CredentiallessAdapterContract(r.config.CLIKind)
	if err != nil {
		r.recordAdmission(correlation, brain.AdmissionCapabilityRejected, brain.GatewayReadinessSelectedProtocol, "adapter_fail_closed")
		return nil, &agentBrainAdmissionError{class: "adapter_fail_closed"}
	}
	profile, err := gateway.LookupRuntimeProfile(adapter.Protocol, r.config.CLIKind)
	if err != nil {
		r.recordAdmission(correlation, brain.AdmissionCapabilityRejected, brain.GatewayReadinessSelectedProtocol, "trusted_profile_unavailable")
		return nil, &agentBrainAdmissionError{class: "trusted_profile_unavailable"}
	}
	model := r.config.RouteModel
	if strings.TrimSpace(legacyModel) != "" {
		parsed, parseErr := brain.ParseRouteModel(legacyModel)
		if parseErr != nil || parsed != model {
			r.recordAdmission(correlation, brain.AdmissionCapabilityRejected, brain.GatewayReadinessSelectedModel, "route_model_not_approved")
			return nil, &agentBrainAdmissionError{class: "route_model_not_approved"}
		}
	}
	approvedPolicy := brain.ApprovedRoutePolicy{
		ID: policy.ID, Revision: agentBrainRouteRevision, Protocol: adapter.Protocol, Approved: true,
	}
	translation, err := r.legacy.TranslateTask(ctx, brain.LegacyTaskInput{
		Provider: provider, Model: string(model), RuntimeRouterOwner: task.RuntimeRouterOwner, AuthToken: task.AuthToken,
	}, correlation, approvedPolicy, brain.LifecycleBindings{
		WorkspaceRef: task.WorkspaceID, WorktreeRef: task.ID, ContextRef: task.IssueID,
		RecoveryRef: task.PriorSessionID, WatchdogPolicyRef: "daemon-watchdogs",
		StreamPolicyRef: "daemon-stream-batching", TerminalPolicyRef: "daemon-terminal-result",
	}, true)
	if err != nil || translation.Task.Request.CLIKind != r.config.CLIKind {
		r.recordAdmission(correlation, brain.AdmissionRoutePolicyRejected, brain.GatewayReadinessNotRequired, "legacy_contract_rejected")
		return nil, &agentBrainAdmissionError{class: "legacy_contract_rejected"}
	}
	if err := policy.Validate(model); err != nil {
		return nil, &agentBrainAdmissionError{class: "route_policy_rejected"}
	}
	if r.dependencies.CredentialSource == nil {
		r.recordAdmission(correlation, brain.AdmissionGatewayAuthFailed, brain.GatewayReadinessAuthentication, "credential_source_unavailable")
		return nil, &agentBrainAdmissionError{class: "credential_source_unavailable"}
	}
	client, err := gateway.NewClient(gateway.ClientOptions{
		Gateway:    r.config.Neutral.Gateway,
		Endpoints:  gateway.EndpointSet{Liveness: "/api/health/ping", Readiness: "/v1/models"},
		Credential: r.dependencies.CredentialSource, HTTPClient: r.dependencies.HTTPClient,
	})
	if err != nil {
		r.recordAdmission(correlation, brain.AdmissionGatewayUnavailable, brain.GatewayReadinessUnavailable, "gateway_client_invalid")
		return nil, &agentBrainAdmissionError{class: "gateway_client_invalid"}
	}
	registry, err := gateway.NewRegistry(gateway.ModelsFetchFunc(func(fetchCtx context.Context) (gateway.ModelsDocument, error) {
		return client.FetchModels(fetchCtx, correlation)
	}), time.Second)
	if err != nil {
		return nil, &agentBrainAdmissionError{class: "model_registry_invalid"}
	}
	checker, err := gateway.NewReadinessChecker(client, registry, r.config.Neutral.Gateway.Readiness, func() (brain.Correlation, error) {
		return correlation, nil
	})
	if err != nil {
		return nil, &agentBrainAdmissionError{class: "readiness_checker_invalid"}
	}
	admission, err := brain.NewGatewayAdmissionController(checker, r.config.Neutral.Gateway.Readiness)
	if err != nil {
		return nil, &agentBrainAdmissionError{class: "admission_controller_invalid"}
	}
	decision, err := admission.Admit(ctx, translation.Task)
	if err != nil {
		r.recordAdmission(correlation, brain.AdmissionGatewayUnavailable, brain.GatewayReadinessUnavailable, "readiness_cancelled")
		return nil, err
	}
	if !decision.Admitted() {
		r.recordAdmission(correlation, decision.State, decision.ReadinessState, decision.ErrorClass)
		return nil, &agentBrainAdmissionError{class: decision.ErrorClass, retryable: decision.Retryable}
	}
	if err := registry.ValidateCapability(ctx, model, gateway.CapabilityRequirement{
		Protocol: adapter.Protocol, Streaming: true, Tools: true,
	}); err != nil {
		r.recordAdmission(correlation, brain.AdmissionCapabilityRejected, brain.GatewayReadinessSelectedModel, "capability_rejected")
		return nil, &agentBrainAdmissionError{class: "capability_rejected"}
	}
	capability, err := registry.LookupModelCapability(ctx, model)
	if err != nil {
		return nil, &agentBrainAdmissionError{class: "capability_unavailable"}
	}
	lease := attempt.Admit()
	if lease == nil {
		return nil, &agentBrainAdmissionError{class: "capacity_admission_closed"}
	}
	committed = true
	r.recordAdmission(correlation, brain.AdmissionAdmitted, brain.GatewayReadinessReady, "admitted")
	return &agentBrainTaskPlan{Task: translation.Task, Profile: profile, Capability: capability, Capacity: lease}, nil
}

func (r *agentBrainRuntime) buildLaunch(ctx context.Context, plan *agentBrainTaskPlan, env *execenv.Environment, local, custom map[string]string) (agentBrainLaunch, error) {
	if !r.enabled() || plan == nil || env == nil {
		return agentBrainLaunch{}, &agentBrainAdmissionError{class: "launch_plan_unavailable"}
	}
	if err := validateAgentBrainCustomEnvironment(custom, local); err != nil {
		return agentBrainLaunch{}, err
	}
	if err := runtimeenv.ValidateExecutionRoot(env.RootDir); err != nil {
		return agentBrainLaunch{}, err
	}
	taskHome := filepath.Join(env.RootDir, "agent-brain-home")
	if info, err := os.Lstat(taskHome); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return agentBrainLaunch{}, fmt.Errorf("controlled task home is not a physical directory")
		}
	} else if os.IsNotExist(err) {
		if err := os.Mkdir(taskHome, 0o700); err != nil {
			return agentBrainLaunch{}, fmt.Errorf("create controlled task home: %w", err)
		}
	} else {
		return agentBrainLaunch{}, fmt.Errorf("inspect controlled task home: %w", err)
	}
	if err := runtimeenv.ValidateExecutionRoot(taskHome); err != nil {
		return agentBrainLaunch{}, err
	}
	if err := os.Chmod(taskHome, 0o700); err != nil {
		return agentBrainLaunch{}, fmt.Errorf("restrict controlled task home: %w", err)
	}
	local = cloneStringMap(local)
	local["MULTICA_SESSION_ID"] = plan.Task.Request.Correlation.SessionID
	local["MULTICA_REQUEST_ID"] = plan.Task.Request.Correlation.RequestID
	local["MULTICA_ROUTER_OWNER"] = string(brain.RouterOwnerOmniRoute)
	inherited := os.Environ
	if r.dependencies.InheritedEnvironment != nil {
		inherited = r.dependencies.InheritedEnvironment
	}
	var launch agentBrainLaunch
	err := r.dependencies.CredentialSource.WithCredential(ctx, r.config.Neutral.Gateway.SecretFile, func(value string) error {
		secret, secretErr := runtimeenv.NewStableSecret(value)
		if secretErr != nil {
			return &agentBrainAdmissionError{class: "stable_key_invalid"}
		}
		child, _, buildErr := runtimeenv.BuildGatewayEnvironment(runtimeenv.ComposeOptions{
			Inherited: inherited(), Local: local, Custom: custom,
			Adapter: runtimeenv.AdapterEnvironment{
				CLI: plan.Task.Request.CLIKind, GatewayRoot: r.config.Neutral.Gateway.BaseURL,
				TaskHome: taskHome, CodexHome: env.CodexHome, StableSecret: secret,
			},
		})
		if buildErr != nil {
			return buildErr
		}
		var codexConfig *runtimeenv.CodexConfigContract
		manifest := []runtimeenv.HomeEntry{}
		if plan.Task.Request.CLIKind == brain.CLICodex {
			contract, configErr := runtimeenv.NewCodexConfigContract(
				r.config.Neutral.Gateway.BaseURL, plan.Task.Request.RouteModel, plan.Task.Request.Correlation,
			)
			if configErr != nil {
				return configErr
			}
			if err := execenv.WriteCredentiallessCodexConfig(env.CodexHome, contract.Bytes()); err != nil {
				return err
			}
			codexConfig = &contract
			manifest = append(manifest,
				runtimeenv.HomeEntry{RelativePath: "config.toml"},
				runtimeenv.HomeEntry{RelativePath: "sessions", Directory: true},
				runtimeenv.HomeEntry{RelativePath: "skills", Directory: true},
			)
		}
		if err := runtimeenv.AssertPreLaunch(runtimeenv.LaunchPlan{
			Environment: child, CodexConfig: codexConfig, TaskHome: manifest,
			ExecutionRoot: env.RootDir,
		}); err != nil {
			return err
		}
		launch = agentBrainLaunch{Environment: child, CodexConfig: codexConfig}
		return nil
	})
	if err != nil {
		return agentBrainLaunch{}, err
	}
	return launch, nil
}

func (r *agentBrainRuntime) validateThinking(plan *agentBrainTaskPlan, thinking string) error {
	if strings.TrimSpace(thinking) != "" {
		return runtimeenv.ErrThinkingNotApproved
	}
	policy, err := runtimeenv.NewGatewayModelPolicy([]runtimeenv.ApprovedGatewayModel{{
		Model: plan.Task.Request.RouteModel, Protocol: plan.Task.RoutePolicy.Protocol,
		CLIs: []brain.CLIKind{plan.Task.Request.CLIKind},
	}})
	if err != nil {
		return err
	}
	return policy.ValidateSelection(plan.Task.Request.CLIKind, plan.Task.Request.RouteModel, "")
}

func (r *agentBrainRuntime) newCorrelation(task Task) brain.Correlation {
	sequence := r.requestSeq.Add(1)
	return brain.Correlation{
		TaskID:    safeCorrelationID("task", task.ID),
		SessionID: safeCorrelationID("session", firstConfigured(task.ChatSessionID, task.PriorSessionID, task.ID)),
		RequestID: fmt.Sprintf("request-%s-%d", correlationDigest(task.ID), sequence),
	}
}

func safeCorrelationID(prefix, value string) string {
	return prefix + "-" + correlationDigest(value)
}

func correlationDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:8])
}

func validateAgentBrainCustomEnvironment(custom, local map[string]string) error {
	reserved := make(map[string]struct{}, len(local)+3)
	for key := range local {
		reserved[strings.ToUpper(key)] = struct{}{}
	}
	for _, key := range []string{"MULTICA_SESSION_ID", "MULTICA_REQUEST_ID", "MULTICA_ROUTER_OWNER"} {
		reserved[key] = struct{}{}
	}
	violations := make([]string, 0)
	for key := range custom {
		canonical := strings.ToUpper(key)
		if _, found := reserved[canonical]; found || isBlockedEnvKey(key) {
			violations = append(violations, key)
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("agent brain custom environment attempts to override trusted keys: %s", strings.Join(violations, ","))
	}
	return runtimeenv.ValidateCustomEnvironment(custom)
}

func childEnvironmentMap(environment runtimeenv.ChildEnvironment) (map[string]string, error) {
	result := make(map[string]string)
	for _, entry := range environment.Exec() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("controlled child environment contains a malformed entry")
		}
		result[key] = value
	}
	return result, nil
}

func cloneStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source)+3)
	for key, value := range source {
		result[key] = value
	}
	return result
}

func (r *agentBrainRuntime) recordAdmission(correlation brain.Correlation, state brain.AdmissionState, readiness brain.GatewayReadinessState, reason string) {
	if reason == "" {
		reason = "unspecified"
	}
	r.diagnosticsMu.Lock()
	r.diagnostics.State = "development"
	r.diagnostics.Readiness = readiness
	r.diagnostics.RouterOwner = brain.RouterOwnerOmniRoute
	r.diagnostics.CLIKind = r.config.CLIKind
	r.diagnostics.RouteModel = r.config.RouteModel
	r.diagnostics.LastOutcome = string(state)
	r.diagnosticsMu.Unlock()
	r.emit(observability.EventAdmissionDecision, correlation, string(state), reason)
	r.emit(observability.EventGatewayReadiness, correlation, string(readiness), reason)
}

func (r *agentBrainRuntime) recordLaunch(plan *agentBrainTaskPlan) {
	if plan == nil || !plan.Capacity.Start() {
		return
	}
	r.emit(observability.EventRouteSelection, plan.Task.Request.Correlation, "launch", "trusted_profile")
	if r.logger != nil {
		r.logger.Info("agent brain launch",
			"task_id", plan.Task.Request.Correlation.TaskID,
			"session_id", plan.Task.Request.Correlation.SessionID,
			"request_id", plan.Task.Request.Correlation.RequestID,
			"cli_kind", plan.Task.Request.CLIKind,
			"route_model", plan.Task.Request.RouteModel,
			"router_owner", brain.RouterOwnerOmniRoute,
		)
	}
}

func (r *agentBrainRuntime) recordTerminal(ctx context.Context, plan *agentBrainTaskPlan, status string, runErr error) {
	if plan == nil {
		return
	}
	outcome := "result"
	reason := "terminal_result"
	if runErr != nil {
		outcome, reason = "error", "execution_error"
	}
	if ctx.Err() != nil || status == "cancelled" {
		outcome, reason = "cancelled", "task_cancelled"
	}
	terminalStatus := brain.TaskStatusCompleted
	if outcome == "cancelled" {
		terminalStatus = brain.TaskStatusCancelled
	} else if outcome == "error" || status != "completed" {
		terminalStatus = brain.TaskStatusFailed
	}
	if !plan.Capacity.Finish(terminalStatus) {
		return
	}
	if outcome == "cancelled" {
		r.emit(observability.EventCancellation, plan.Task.Request.Correlation, outcome, reason)
	}
	if r.logger != nil {
		r.logger.Info("agent brain terminal",
			"task_id", plan.Task.Request.Correlation.TaskID,
			"session_id", plan.Task.Request.Correlation.SessionID,
			"request_id", plan.Task.Request.Correlation.RequestID,
			"outcome", outcome, "reason_code", reason,
		)
	}
}

func (r *agentBrainRuntime) emit(kind observability.EventKind, correlation brain.Correlation, outcome, reason string) {
	event := observability.SafeEvent{
		SchemaVersion: observability.EventSchemaVersion, Kind: kind, At: time.Now().UTC(),
		TaskID: correlation.TaskID, SessionID: correlation.SessionID, RequestID: correlation.RequestID,
		RouteModel: r.config.RouteModel, Outcome: outcome, ReasonCode: reason,
		CapacityTier: int(brain.CapacityTier20),
	}
	if err := event.Validate(); err != nil || r.logger == nil {
		return
	}
	r.logger.Debug("agent brain event",
		"schema", event.SchemaVersion, "kind", event.Kind,
		"task_id", event.TaskID, "session_id", event.SessionID, "request_id", event.RequestID,
		"route_model", event.RouteModel, "outcome", event.Outcome, "reason_code", event.ReasonCode,
	)
}

func (r *agentBrainRuntime) snapshot() agentBrainDiagnostics {
	if r == nil {
		return agentBrainDiagnostics{State: "unavailable"}
	}
	r.diagnosticsMu.RLock()
	defer r.diagnosticsMu.RUnlock()
	result := r.diagnostics
	for _, use := range r.config.LegacyUses {
		result.LegacyUseCount += use.Count
	}
	for _, use := range r.legacyRecorder.Snapshot() {
		result.LegacyUseCount += use.Count
	}
	result.Capacity = r.capacity.Snapshot()
	return result
}
