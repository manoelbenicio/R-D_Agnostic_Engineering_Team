package brain

type RouteApproval string

const (
	RouteApprovalCanaryOnly       RouteApproval = "canary-only"
	RouteApprovalEvidenceRequired RouteApproval = "evidence-required"
)

type InitialRoute struct {
	Model    RouteModel
	CLI      CLIKind
	Protocol ProtocolFamily
	Approval RouteApproval
	Fallback FallbackPolicy
}

type FallbackPolicy struct {
	SameModelAccountFallback bool
	CrossModelFallback       []RouteModel
	PreCommitOnly            bool
}

// InitialModelSet freezes the one route allowed for contract/build work and
// later guarded canary evaluation. It is not currently admissible: protocol
// capability evidence is still required. Cross-model fallback remains empty
// until capability equivalence is proven and approved.
func InitialModelSet() []InitialRoute {
	return []InitialRoute{
		{
			Model:    RouteModel("agy/claude-opus-4-6-thinking"),
			CLI:      CLIClaudeCode,
			Protocol: ProtocolAnthropicMessages,
			Approval: RouteApprovalEvidenceRequired,
			Fallback: FallbackPolicy{
				SameModelAccountFallback: true,
				CrossModelFallback:       nil,
				PreCommitOnly:            true,
			},
		},
	}
}

type StableKeyScope struct {
	InferenceOnly        bool
	ApprovedRoutesOnly   bool
	ManagementDenied     bool
	ProviderNativeDenied bool
}

func FrozenStableKeyScope() StableKeyScope {
	return StableKeyScope{
		InferenceOnly:        true,
		ApprovedRoutesOnly:   true,
		ManagementDenied:     true,
		ProviderNativeDenied: true,
	}
}

type CutoverBlocker string

const (
	BlockerProtocolConformance  CutoverBlocker = "protocol-model-conformance"
	BlockerFailureInjection     CutoverBlocker = "failure-injection"
	BlockerSingleFlightRefresh  CutoverBlocker = "single-flight-refresh"
	BlockerReadinessFailClosed  CutoverBlocker = "readiness-fail-closed"
	BlockerSmartContext         CutoverBlocker = "smart-context-parity-or-waiver"
	BlockerContinuationAffinity CutoverBlocker = "continuation-affinity"
	BlockerStateTopology        CutoverBlocker = "state-topology"
	BlockerImageDigest          CutoverBlocker = "image-digest"
	BlockerFormalSignoff        CutoverBlocker = "formal-signoff"
)

func FrozenCutoverBlockers() []CutoverBlocker {
	return []CutoverBlocker{
		BlockerProtocolConformance,
		BlockerFailureInjection,
		BlockerSingleFlightRefresh,
		BlockerReadinessFailClosed,
		BlockerSmartContext,
		BlockerContinuationAffinity,
		BlockerStateTopology,
		BlockerImageDigest,
		BlockerFormalSignoff,
	}
}
