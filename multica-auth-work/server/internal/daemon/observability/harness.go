package observability

import (
	"fmt"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type ProtocolFamily string

const (
	ProtocolAnthropicMessages ProtocolFamily = "anthropic-messages"
	ProtocolOpenAIResponses   ProtocolFamily = "openai-responses"
	ProtocolOpenAIChat        ProtocolFamily = "openai-chat"
	ProtocolAntigravityDirect ProtocolFamily = "antigravity-direct"
)

type ProtocolWeight struct {
	Protocol ProtocolFamily
	Percent  int
}

type PayloadClass struct {
	ID                      string
	WeightPercent           int
	PromptTokens            int
	OutputTokens            int
	PromptContextFraction   float64
	OutputContextFraction   float64
	RequiresRegistryContext bool
}

type TrafficShape struct {
	ProtocolMix             []ProtocolWeight
	PayloadMix              []PayloadClass
	StreamingPercent        int
	ToolRequestPercent      int
	ParallelToolPercent     int
	CancellationPercent     int
	IndependentRequestShare int
	ContinuationShare       int
}

type FailureCase struct {
	ID             string
	Injection      string
	ExpectedSafety string
	Evidence       []string
}

type DistributionSpec struct {
	IdentityPolicy  string
	ExpectedModel   string
	Exclusions      []string
	RequiredOutputs []string
}

type TierProfile struct {
	Tier                 brain.CapacityTier
	TaskCount            int
	Warmup               time.Duration
	Sustained            time.Duration
	Recovery             time.Duration
	RunnableNow          bool
	AuthorizationGate    string
	RequiredThresholdRef []string
}

type AcceptanceHarnessSpec struct {
	EvidenceID           string
	SyntheticOnly        bool
	ContentPolicy        string
	CredentialPolicy     string
	Profiles             []TierProfile
	Traffic              TrafficShape
	Failures             []FailureCase
	Distribution         DistributionSpec
	RequiredMeasurements []string
}

func DefaultAcceptanceHarnessSpec() AcceptanceHarnessSpec {
	thresholds := []string{
		"accepted_completion_ratio", "error_ratio", "selection_p95", "ttft_p95",
		"request_p95", "retry_ratio", "fallback_ratio", "peak_queue",
		"fairness_deviation", "cpu_peak", "memory_peak", "socket_peak",
		"cancellation_release_deadline", "steady_state_recovery_deadline",
	}
	return AcceptanceHarnessSpec{
		EvidenceID:       "EV-G2D-05",
		SyntheticOnly:    true,
		ContentPolicy:    "generated non-repository prompts and inert tool schemas/results only; no raw production content or opaque reasoning",
		CredentialPolicy: "test harness receives only the approved scoped OmniRoute reference through the service; reports never contain authorization data or fingerprints",
		Profiles: []TierProfile{
			{brain.CapacityTier20, 20, 5 * time.Minute, 30 * time.Minute, 10 * time.Minute, false, "Wave 3 after G3 wiring and all selected protocol/security gates; tier report stored as EV-G4-CAP", thresholds},
			{brain.CapacityTier50, 50, 10 * time.Minute, 45 * time.Minute, 15 * time.Minute, false, "new authorization plus accepted tier 20; report stored as EV-G7-50", thresholds},
			{brain.CapacityTier100, 100, 10 * time.Minute, 60 * time.Minute, 20 * time.Minute, false, "new authorization, accepted tier 50, and resolved shared-state decision; report stored as EV-G7-100", thresholds},
		},
		Traffic: TrafficShape{
			ProtocolMix: []ProtocolWeight{
				{ProtocolAnthropicMessages, 30},
				{ProtocolOpenAIResponses, 30},
				{ProtocolOpenAIChat, 30},
				{ProtocolAntigravityDirect, 10},
			},
			PayloadMix: []PayloadClass{
				{"small", 40, 4096, 1024, 0, 0, false},
				{"medium", 40, 16384, 4096, 0, 0, false},
				{"large-context-relative", 20, 0, 0, 0.70, 0.10, true},
			},
			StreamingPercent:        70,
			ToolRequestPercent:      40,
			ParallelToolPercent:     15,
			CancellationPercent:     10,
			IndependentRequestShare: 80,
			ContinuationShare:       20,
		},
		Failures: []FailureCase{
			{"FAIL-ACCOUNT-DISABLE", "disable one pseudonymous slot during active load", "new independent requests skip it; in-flight policy is honored", []string{"selection distribution", "terminal outcomes"}},
			{"FAIL-ACCESS-EXPIRY", "expire one synthetic access authorization", "single-flight refresh succeeds once or slot is quarantined", []string{"refresh count", "safe slot state"}},
			{"FAIL-REFRESH-REVOKE", "revoke one synthetic refresh authorization", "only the affected slot leaves eligibility", []string{"eligibility transition", "fallback outcome"}},
			{"FAIL-QUOTA", "exhaust one synthetic quota", "traffic advances and re-entry waits for reset/probe", []string{"quota state", "selection reason"}},
			{"FAIL-429-ACCOUNT", "inject repeated account-scoped 429", "scoped circuit opens and half-open recovery is bounded", []string{"circuit timeline", "other-slot completions"}},
			{"FAIL-429-GLOBAL", "inject provider-global 429", "no account thrash; actionable bounded retry or approved fallback", []string{"selection attempts", "safe retry time"}},
			{"FAIL-UPSTREAM-MATRIX", "inject 401, 403, timeout, reset, malformed response, and 500/502/503", "each follows its documented bounded class", []string{"reason codes", "attempt/deadline counts"}},
			{"FAIL-SSE-PREPOST", "break SSE before first output and after partial output", "only pre-output case may replay; partial output is surfaced", []string{"commit marker", "retry count", "terminal status"}},
			{"FAIL-CANCEL", "cancel queued and active streaming requests", "all capacity is released exactly once and upstream stops", []string{"slot counters", "terminal cancellation"}},
			{"FAIL-HOT-ACCOUNT", "add and remove a synthetic slot with 20 or more requests", "pool update is atomic and selection remains safe", []string{"policy revision", "distribution"}},
			{"FAIL-CONTINUATION", "exercise Responses continuation, prompt cache, and tool turn", "dependent requests retain ownership while independent requests rotate", []string{"affinity reason", "selection sequence"}},
			{"FAIL-RESTART", "restart or roll OmniRoute under load", "readiness closes admission and recovery/drain behavior matches policy", []string{"readiness timeline", "affected terminal outcomes", "recovery time"}},
		},
		Distribution: DistributionSpec{
			IdentityPolicy:  "use ephemeral slot labels only; never provider account identity, email, subscription name, or credential-derived fingerprint",
			ExpectedModel:   "distribution is proportional to eligible time and configured per-slot capacity for independent requests",
			Exclusions:      []string{"dependent continuation affinity", "quota/cooldown/circuit ineligibility", "safe concurrency limit", "approved provider-global hold"},
			RequiredOutputs: []string{"per-slot independent-request count", "eligible-duration denominator", "expected count", "observed deviation", "documented exclusion intervals"},
		},
		RequiredMeasurements: []string{
			"offered, admitted, queued, rejected, started, completed, failed, and cancelled counts",
			"selection, queue, first-output, and end-to-end p50/p95/p99",
			"401, 403, 429, 5xx, timeout, retry, fallback, and partial-stream rates",
			"peak queue, active tasks, in-flight requests, and exactly-once capacity reconciliation",
			"CPU, memory, sockets, log volume, and post-failure steady-state recovery",
			"route/model/protocol mix, prompt/output class, streaming/tool/cancel ratios, duration, and upstream limits",
			"pseudonymous slot fairness with eligibility and affinity explanations",
		},
	}
}

func (s AcceptanceHarnessSpec) Validate() error {
	if s.EvidenceID != "EV-G2D-05" || !s.SyntheticOnly || s.ContentPolicy == "" || s.CredentialPolicy == "" {
		return fmt.Errorf("harness must be synthetic, content-off, and reference-only")
	}
	if len(s.Profiles) != 3 {
		return fmt.Errorf("20/50/100 profile specifications are required")
	}
	wantTiers := []brain.CapacityTier{brain.CapacityTier20, brain.CapacityTier50, brain.CapacityTier100}
	for index, profile := range s.Profiles {
		if profile.Tier != wantTiers[index] || profile.TaskCount != int(profile.Tier) || profile.RunnableNow ||
			profile.Warmup <= 0 || profile.Sustained <= 0 || profile.Recovery <= 0 || profile.AuthorizationGate == "" || len(profile.RequiredThresholdRef) == 0 {
			return fmt.Errorf("invalid or prematurely runnable tier profile")
		}
	}
	protocolTotal := 0
	seenProtocols := map[ProtocolFamily]bool{}
	for _, weight := range s.Traffic.ProtocolMix {
		if weight.Percent <= 0 || seenProtocols[weight.Protocol] {
			return fmt.Errorf("invalid protocol mix")
		}
		seenProtocols[weight.Protocol] = true
		protocolTotal += weight.Percent
	}
	if protocolTotal != 100 || len(seenProtocols) != 4 {
		return fmt.Errorf("protocol mix must cover four families and total 100")
	}
	payloadTotal := 0
	for _, payload := range s.Traffic.PayloadMix {
		payloadTotal += payload.WeightPercent
		if payload.ID == "" || payload.WeightPercent <= 0 {
			return fmt.Errorf("invalid payload class")
		}
		if payload.RequiresRegistryContext {
			if payload.PromptContextFraction <= 0 || payload.PromptContextFraction+payload.OutputContextFraction >= 1 {
				return fmt.Errorf("invalid context-relative payload")
			}
		} else if payload.PromptTokens <= 0 || payload.OutputTokens <= 0 {
			return fmt.Errorf("fixed payload must declare token sizes")
		}
	}
	if payloadTotal != 100 || s.Traffic.IndependentRequestShare+s.Traffic.ContinuationShare != 100 {
		return fmt.Errorf("traffic shares must total 100")
	}
	for _, percent := range []int{s.Traffic.StreamingPercent, s.Traffic.ToolRequestPercent, s.Traffic.ParallelToolPercent, s.Traffic.CancellationPercent} {
		if percent < 0 || percent > 100 {
			return fmt.Errorf("traffic percentage is out of range")
		}
	}
	if len(s.Failures) != 12 || len(s.Distribution.RequiredOutputs) == 0 || len(s.RequiredMeasurements) == 0 {
		return fmt.Errorf("failure, distribution, and measurement specifications are incomplete")
	}
	seenFailures := map[string]bool{}
	for _, failure := range s.Failures {
		if failure.ID == "" || seenFailures[failure.ID] || failure.Injection == "" || failure.ExpectedSafety == "" || len(failure.Evidence) == 0 {
			return fmt.Errorf("invalid failure case")
		}
		seenFailures[failure.ID] = true
	}
	return nil
}
