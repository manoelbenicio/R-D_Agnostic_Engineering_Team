package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/deploy"
	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/daemon/gateway"
	"github.com/multica-ai/multica/server/internal/daemon/observability"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
	"github.com/multica-ai/multica/server/internal/daemon/runtimeenv"
	"golang.org/x/sync/singleflight"
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

	// admitGroup coalesces concurrent gateway-readiness admissions into a
	// single in-flight evaluation (no /v1/models stampede) while allowing the
	// admitted verdict to fan out concurrently to all waiters. It does NOT
	// serialize admissions — required for bounded tier concurrency.
	admitGroup singleflight.Group
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
	capacity, err := brain.NewLifecycleCapacity(effectiveAgentBrainCapacity(config))
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
	return r != nil && r.config.DevelopmentEnabled && r.config.Neutral.Gateway.Required
}

// readinessAdmissionWait bounds how long admitTask will keep retrying for a
// FRESH successful readiness result before failing closed. Configurable safe
// cadence via env; default 20s. A value of 0 disables retry (single attempt).
func readinessAdmissionWait() time.Duration {
	if v := strings.TrimSpace(os.Getenv("AGENT_BRAIN_READINESS_ADMISSION_WAIT_MS")); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return 20 * time.Second
}

const (
	readinessBackoffBase = 500 * time.Millisecond
	readinessBackoffMax  = 5 * time.Second
)

// transientReadinessRetry reports whether an admission failure is a transient
// gateway condition worth a bounded retry for a FRESH ready result, and any
// server-advised Retry-After. Deterministic rejections (auth, invalid request,
// selected-protocol mismatch, capability) are NOT retried — they fail closed
// immediately. This never weakens StrictReadinessPolicy: a persistent transient
// failure still exhausts the bounded wait and fails closed.
func transientReadinessRetry(err error, decision brain.AdmissionDecision) (bool, time.Duration) {
	var ge *gateway.GatewayError
	if errors.As(err, &ge) {
		switch ge.Class {
		case gateway.ErrorRateLimited, gateway.ErrorTimeout, gateway.ErrorOverloaded,
			gateway.ErrorUpstream, gateway.ErrorTransport:
			return true, ge.RetryAfter
		}
		if ge.Retryable {
			return true, ge.RetryAfter
		}
		return false, 0
	}
	if err == nil && !decision.Admitted() && decision.Retryable {
		switch decision.ReadinessState {
		case brain.GatewayReadinessUnavailable, brain.GatewayReadinessModelRegistry:
			return true, 0
		}
	}
	return false, 0
}

func (r *agentBrainRuntime) admitWithReadinessResilience(ctx context.Context, admission *brain.GatewayAdmissionController, task brain.Task) (brain.AdmissionDecision, error) {
	// Coalesce concurrent readiness admissions into ONE in-flight gateway
	// readiness evaluation so N concurrent tasks trigger a single /v1/models
	// check (no stampede) and all share the fresh verdict — then each task
	// independently takes its capacity lease. This preserves single-flight
	// safety without serializing admissions, which is required to reach the
	// bounded tier concurrency limit.
	type admitResult struct {
		decision brain.AdmissionDecision
		err      error
	}
	v, _, _ := r.admitGroup.Do("gateway-readiness", func() (interface{}, error) {
		// Readiness is a global gateway property. Evaluate it on a fresh bounded
		// context detached from any single caller's cancellation so one task's
		// cancel cannot poison the shared verdict.
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), readinessAdmissionWait()+5*time.Second)
		defer cancel()
		d, e := admitRetryLoop(rctx, readinessAdmissionWait(), func(c context.Context) (brain.AdmissionDecision, error) {
			return admission.Admit(c, task)
		})
		return admitResult{decision: d, err: e}, nil
	})
	res := v.(admitResult)
	return res.decision, res.err
}

// admitRetryLoop retries admitFn with Retry-After honoring + bounded
// exponential backoff+jitter for transient readiness-fetch failures, returning
// only a fresh ADMITTED decision produced within this call (never stale), and
// failing closed once the bounded wait is exhausted or on deterministic
// rejection. Extracted for deterministic testing.
func admitRetryLoop(ctx context.Context, wait time.Duration, admitFn func(context.Context) (brain.AdmissionDecision, error)) (brain.AdmissionDecision, error) {
	deadline := time.Now().Add(wait)
	var lastDecision brain.AdmissionDecision
	var lastErr error
	for attempt := 0; ; attempt++ {
		decision, err := admitFn(ctx)
		if err == nil && decision.Admitted() {
			return decision, nil // fresh ready
		}
		lastDecision, lastErr = decision, err
		retry, retryAfter := transientReadinessRetry(err, decision)
		if !retry {
			return decision, err // deterministic rejection -> fail closed now
		}
		sleep := retryAfter
		if sleep <= 0 {
			backoff := readinessBackoffBase << uint(attempt)
			if backoff <= 0 || backoff > readinessBackoffMax {
				backoff = readinessBackoffMax
			}
			jitter := time.Duration(rand.Int63n(int64(backoff/2)+1)) - backoff/4
			sleep = backoff + jitter
		}
		if sleep < 0 {
			sleep = 0
		}
		if time.Now().Add(sleep).After(deadline) {
			return lastDecision, lastErr // bounded wait exhausted -> fail closed
		}
		select {
		case <-ctx.Done():
			return lastDecision, ctx.Err()
		case <-time.After(sleep):
		}
	}
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
		doc, fetchErr := client.FetchModels(fetchCtx, correlation)
		if fetchErr != nil {
			return gateway.ModelsDocument{}, fetchErr
		}
		// DEV compatibility: project raw OmniRoute /v1/models into the
		// enriched schema the registry requires. Gated on the explicit
		// OMNIROUTE_DEV_MODELS_COMPAT=1 flag; when unset, the enriched
		// response passes through unchanged (for future OmniRoute versions
		// that serve the full schema natively).
		if os.Getenv("OMNIROUTE_DEV_MODELS_COMPAT") == "1" {
			native := gateway.OmniRouteNativeModels{Object: doc.Object}
			for _, m := range doc.Models {
				native.Data = append(native.Data, gateway.OmniRouteNativeModel{ID: m.ID})
			}
			return gateway.ProjectOmniRouteModels(native, doc.RegistryVersion), nil
		}
		return doc, nil
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
	checker.SetDiagnosticsLogger(r.logger)
	admission, err := brain.NewGatewayAdmissionController(checker, r.config.Neutral.Gateway.Readiness)
	if err != nil {
		return nil, &agentBrainAdmissionError{class: "admission_controller_invalid"}
	}
	decision, err := r.admitWithReadinessResilience(ctx, admission, translation.Task)
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
	clineDataDir := ""
	if plan.Task.Request.CLIKind == brain.CLIOpenAICompatible {
		clineDataDir = filepath.Join(env.RootDir, "cline-data")
		if err := os.Mkdir(clineDataDir, 0o700); err != nil && !os.IsExist(err) {
			return agentBrainLaunch{}, fmt.Errorf("create controlled cline data dir: %w", err)
		}
		if err := os.Chmod(clineDataDir, 0o700); err != nil {
			return agentBrainLaunch{}, fmt.Errorf("restrict controlled cline data dir: %w", err)
		}
		if err := runtimeenv.ValidateExecutionRoot(clineDataDir); err != nil {
			return agentBrainLaunch{}, err
		}
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
		telEndpoint, telTask, telReq := "", "", ""
		if plan.Task.Request.CLIKind == brain.CLIClaudeCode {
			// Route-hop telemetry: point Claude at the fixed loopback OTLP
			// receiver and carry the canonical correlation as resource attrs.
			telEndpoint = daemonOTLPLogsEndpoint
			telTask = plan.Task.Request.Correlation.TaskID
			telReq = plan.Task.Request.Correlation.RequestID
		}
		child, _, buildErr := runtimeenv.BuildGatewayEnvironment(runtimeenv.ComposeOptions{
			Inherited: inherited(), Local: local, Custom: custom,
			Adapter: runtimeenv.AdapterEnvironment{
				CLI: plan.Task.Request.CLIKind, GatewayRoot: r.config.Neutral.Gateway.BaseURL,
				TaskHome: taskHome, CodexHome: env.CodexHome, ClineDataDir: clineDataDir, StableSecret: secret,
				TelemetryOTLPLogsEndpoint: telEndpoint,
				TelemetryTaskID:           telTask,
				TelemetryRequestID:        telReq,
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
		} else if plan.Task.Request.CLIKind == brain.CLIOpenAICompatible {
			contract, configErr := runtimeenv.NewClineConfigContract(
				r.config.Neutral.Gateway.BaseURL, plan.Task.Request.RouteModel, time.Now().UTC().Format(time.RFC3339),
			)
			if configErr != nil {
				return configErr
			}
			if err := execenv.WriteCredentiallessClineConfig(clineDataDir, contract.Bytes()); err != nil {
				return err
			}
			// The Cline providers.json carrier lives in CLINE_DATA_DIR and is
			// deliberately excluded from the task-home manifest: home.go's
			// ValidateTaskHomeManifest forbids providers.json/.cline in the task
			// home. The carrier is byte-validated by NewClineConfigContract; the
			// pre-launch gate validates only env/roots (no config-byte checks).
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

// gatewayApprovedThinkingLevels is the gateway-side allowlist of reasoning
// levels, keyed by the built-in provider that agentBrainBuiltInCLIFor resolves
// for an accepted CLIKind. Only the three accepted gateway frontends appear
// here; every other CLI (Antigravity, Kimi, NIM, anything unmapped) has no
// entry and therefore fails closed.
//
// The lists mirror pkg/agent's provider enums, which remain the authority for
// what a level means. TestGatewayThinkingLevelsMatchProviderEnums compares the
// two in BOTH directions over a closed token universe, so a level added to or
// removed from pkg/agent's enum for these providers breaks the build. The
// comparison is not exhaustive: a token nobody listed in that universe would
// escape it. Closing that hole requires pkg/agent to export its enum, which is
// another package's file and is requested as a formal handoff rather than taken
// silently. Antigravity is deliberately absent from both: `agy` exposes no
// effort flag, so reasoning there is model-embedded only.
var gatewayApprovedThinkingLevels = map[string][]string{
	"claude": {"low", "medium", "high", "xhigh", "max"},
	"codex":  {"none", "minimal", "low", "medium", "high", "xhigh"},
	"cline":  {"none", "low", "medium", "high", "xhigh"},
}

// gatewayThinkingLevelsFor resolves the approved level list for a CLIKind. An
// unmapped CLI, or a CLI whose built-in provider has no reasoning contract, is
// a fail-closed condition rather than an empty allowlist.
func gatewayThinkingLevelsFor(kind brain.CLIKind) ([]string, error) {
	builtIn, err := agentBrainBuiltInCLIFor(kind)
	if err != nil {
		return nil, runtimeenv.ErrThinkingNotApproved
	}
	levels, ok := gatewayApprovedThinkingLevels[builtIn.Provider]
	if !ok || len(levels) == 0 {
		return nil, runtimeenv.ErrThinkingNotApproved
	}
	return levels, nil
}

// gatewayReasoningCapabilityAuthoritative reports whether
// brain.ModelCapability.Reasoning carries an observed value.
//
// It does not, in the deployment that runs today. With
// OMNIROUTE_DEV_MODELS_COMPAT=1 the registry is fed by
// gateway.ProjectOmniRouteModels, whose rows hardcode Reasoning=false for every
// model (gateway/model_projection.go:130 for the approved row, :146 for the
// unavailable one) because raw OmniRoute /v1/models carries only ids. In that
// mode `false` means "not advertised", not "cannot reason", so treating it as a
// veto would refuse every configured level in production — the exact bug this
// change exists to remove. Assuming `true` instead would be equally wrong, so
// the flag is read and the model-level gate is declared unavailable, leaving the
// per-provider allowlist as the operative check.
//
// When the flag is unset the enriched schema passes through untouched, the bit
// is an observation, and reasoning becomes a hard requirement.
func gatewayReasoningCapabilityAuthoritative() bool {
	return os.Getenv("OMNIROUTE_DEV_MODELS_COMPAT") != "1"
}

func (r *agentBrainRuntime) validateThinking(plan *agentBrainTaskPlan, thinking string) error {
	// Native execution has no Agent Brain launch plan. Its provider-specific
	// backend validates the persisted thinking level, so the gateway allowlist
	// must not run (or dereference a nil plan) on this path.
	if plan == nil {
		return nil
	}

	// Only the empty string means "runtime default". A whitespace-only value is
	// a misconfiguration and is validated like any other token, so it fails
	// closed: daemon.go hands the persisted string to the child unchanged, and
	// " " is not an effort level for any provider.
	requested := thinking
	var approved []string
	if requested != "" {
		// The model-level gate applies only when the capability bit is an
		// observation (see gatewayReasoningCapabilityAuthoritative). When it is
		// a projection placeholder the check is skipped rather than inverted,
		// and admission still depends on the provider allowlist below.
		if gatewayReasoningCapabilityAuthoritative() && !plan.Capability.Reasoning {
			return runtimeenv.ErrThinkingNotApproved
		}
		levels, err := gatewayThinkingLevelsFor(plan.Task.Request.CLIKind)
		if err != nil {
			return err
		}
		approved = levels
	}

	policy, err := runtimeenv.NewGatewayModelPolicy([]runtimeenv.ApprovedGatewayModel{{
		Model: plan.Task.Request.RouteModel, Protocol: plan.Task.RoutePolicy.Protocol,
		CLIs: []brain.CLIKind{plan.Task.Request.CLIKind}, ThinkingLevels: approved,
	}})
	if err != nil {
		return err
	}
	// The persisted level is what the child process will actually receive, so it
	// is what gets validated. Passing "" here (the previous behaviour) left the
	// allowlist unreachable and forced an unconditional rejection of every
	// configured level.
	return policy.ValidateSelection(plan.Task.Request.CLIKind, plan.Task.Request.RouteModel, requested)
}

func (r *agentBrainRuntime) newCorrelation(task Task) brain.Correlation {
	// Canonical, deterministic correlation shared with the server hops: raw
	// task_id; request_id and session_id derived from task_id (and chat session)
	// so ingress, route, admission, and delivery independently compute the
	// IDENTICAL join keys without cross-process propagation.
	return brain.Correlation{
		TaskID:    e2e.CanonicalTaskID(task.ID),
		SessionID: e2e.CanonicalSessionID(task.ChatSessionID, task.ID),
		RequestID: e2e.CanonicalRequestID(task.ID),
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
