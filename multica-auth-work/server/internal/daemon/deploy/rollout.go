package deploy

import "fmt"

type GateKind string

const (
	GateProtocol GateKind = "protocol"
	GateProvider GateKind = "provider"
	GateCapacity GateKind = "capacity"
	GateSecurity GateKind = "security"
)

type FeatureGate struct {
	ID             string
	Kind           GateKind
	DefaultEnabled bool
	Prerequisites  []string
	Evidence       []string
	RollbackFlag   string
}

type CanaryCohort struct {
	ID             string
	Order          int
	Scope          string
	MaxTaskTier    int
	EntryEvidence  []string
	ExitEvidence   []string
	ExpansionBlock []string
}

type RollbackTrigger struct {
	ID             string
	Scope          string
	Condition      string
	Action         string
	Runbook        RunbookID
	EvidenceTarget string
}

type RolloutPlan struct {
	EvidenceID string
	Gates      []FeatureGate
	Cohorts    []CanaryCohort
	Triggers   []RollbackTrigger
}

func DefaultRolloutPlan() RolloutPlan {
	commonProtocolEvidence := []string{"EV-G4-01", "EV-G4-02", "EV-G4-04", "EV-G4-06"}
	return RolloutPlan{
		EvidenceID: "EV-G2D-07",
		Gates: []FeatureGate{
			{"gateway-required", GateSecurity, false, []string{"EV-G3-04", "EV-G3-05", "EV-G4-03"}, []string{"EV-G3-WIRE", "EV-G4-03"}, "disable-gateway-admission"},
			{"protocol-anthropic-messages", GateProtocol, false, []string{"exact-model registry", "synthetic stream/non-stream fixtures"}, commonProtocolEvidence, "disable-anthropic-route"},
			{"protocol-openai-responses", GateProtocol, false, []string{"exact-model registry", "continuation affinity proof", "synthetic stream/non-stream fixtures"}, commonProtocolEvidence, "disable-responses-route"},
			{"protocol-openai-chat", GateProtocol, false, []string{"exact-model registry", "synthetic stream/non-stream fixtures"}, commonProtocolEvidence, "disable-chat-route"},
			{"protocol-antigravity-direct", GateProtocol, false, []string{"exact native schema/auth contract", "native endpoint override proof"}, []string{"EV-G4-01", "EV-G4-AGY"}, "disable-antigravity-direct"},
			{"provider-claude", GateProvider, false, []string{"Anthropic route accepted", "no provider credential in child"}, []string{"EV-G4-02", "EV-G4-03", "EV-G4-ADP"}, "disable-provider-claude"},
			{"provider-codex-openai", GateProvider, false, []string{"Responses route accepted", "controlled provider config"}, []string{"EV-G4-02", "EV-G4-COD"}, "disable-provider-codex"},
			{"provider-kimi", GateProvider, false, []string{"native registry or approved compatible frontend", "exact model capability"}, []string{"EV-G4-02", "EV-G4-ADP"}, "disable-provider-kimi"},
			{"provider-glm", GateProvider, false, []string{"Chat route accepted", "exact model capability"}, []string{"EV-G4-02", "EV-G4-ADP"}, "disable-provider-glm"},
			{"provider-nvidia", GateProvider, false, []string{"generic gateway adapter", "no direct NIM credential path"}, []string{"EV-G4-02", "EV-G4-NIM"}, "disable-provider-nvidia"},
			{"provider-antigravity", GateProvider, false, []string{"accepted direct or compatible frontend path"}, []string{"EV-G4-02", "EV-G4-AGY"}, "disable-provider-antigravity"},
			{"capacity-tier-20", GateCapacity, false, []string{"all selected protocol/provider gates", "tier-20 load/failure report"}, []string{"EV-G4-CAP"}, "cap-admission-below-20-or-hold"},
			{"capacity-tier-50", GateCapacity, false, []string{"new authorization", "accepted tier-20", "tier-50 report"}, []string{"EV-G7-50"}, "cap-admission-at-20"},
			{"capacity-tier-100", GateCapacity, false, []string{"new authorization", "accepted tier-50", "state-topology decision", "tier-100 report"}, []string{"EV-G7-100", "EV-G7-state"}, "cap-admission-at-highest-accepted-tier"},
			{"smart-context", GateSecurity, false, []string{"SC01-SC10 accepted or signed waiver"}, []string{"EV-G5-SC"}, "disable-smart-context"},
			{"cross-model-fallback", GateSecurity, false, []string{"signed ordered policy", "capability equivalence proof"}, []string{"EV-G5-PAR"}, "disable-cross-model-fallback"},
		},
		Cohorts: []CanaryCohort{
			{"synthetic-contract", 1, "synthetic content and credentials only; no production admission", 0, []string{"EV-G2B-07", "EV-G2D-05"}, []string{"all package tests and fixture validation pass"}, []string{"any secret/content policy violation", "schema or protocol mismatch"}},
			{"single-route-internal", 2, "frozen Claude Code plus exact Agy canary route", 20, []string{"EV-G3-07", "protocol and provider gates accepted"}, []string{"stream/non-stream/tools/reasoning/cancel evidence", "no credential/direct-route evidence"}, []string{"any commit replay", "any affinity loss", "any unknown actual model"}},
			{"multi-protocol-internal", 3, "accepted Anthropic, Responses, and Chat routes", 20, []string{"EV-G4-01", "EV-G4-02"}, []string{"failure matrix and correlation complete"}, []string{"any route lacks exact capability evidence", "systemic auth or circuit incident"}},
			{"tier-20-acceptance", 4, "authorized 20-task mixed profile", 20, []string{"EV-G4-04", "EV-G4-05", "EV-G4-06", "EV-G4-07"}, []string{"EV-G4-CAP accepted by product/operations/security"}, []string{"capacity counter leak", "unbounded queue/resource growth", "SLO or fairness gate failure"}},
		},
		Triggers: []RollbackTrigger{
			{"RBK-SECURITY", "global", "any credential/content exposure, auth bypass, or dual router owner", "disable affected admission immediately and revoke/isolate as applicable", RunbookIncident, "EV-G4-03"},
			{"RBK-READINESS", "gateway", "authenticated readiness fails or selected model/protocol becomes unavailable", "hold new model-dependent admissions", RunbookRollback, "EV-G3-04"},
			{"RBK-PROTOCOL", "route", "event ordering, tool correlation, reasoning, structure, usage, or error contract fails", "disable exact protocol/route cohort", RunbookRollback, "EV-G4-01"},
			{"RBK-AFFINITY", "route", "dependent continuation loses ownership or unrelated traffic becomes sticky", "disable exact route and hold affected continuations", RunbookIncident, "EV-G4-04"},
			{"RBK-REPLAY", "route", "partial output or non-idempotent tool action is replayed", "stop route immediately and preserve correlation evidence", RunbookIncident, "EV-G4-06"},
			{"RBK-AUTH", "provider", "refresh, 401, or 403 exceeds the approved gate", "disable provider cohort and quarantine scoped slots", RunbookIncident, "EV-G4-05"},
			{"RBK-RATE", "provider-or-route", "provider-global 429, circuit-open ratio, or retry amplification exceeds the approved gate", "stop expansion and apply scoped cooldown/hold", RunbookIncident, "EV-G4-05"},
			{"RBK-CAPACITY", "capacity", "queue, latency, error, CPU, memory, sockets, or cancellation reconciliation breaches the approved profile", "cap admission at the last accepted tier or hold", RunbookRollback, "EV-G4-CAP"},
			{"RBK-STATE", "global", "state consistency, backup/restore, or restart recovery fails", "hold admissions and restore last accepted revision", RunbookBackupRestore, "EV-G4-07"},
		},
	}
}

func (p RolloutPlan) Validate() error {
	if p.EvidenceID != "EV-G2D-07" {
		return fmt.Errorf("unexpected rollout evidence id")
	}
	requiredGates := map[string]bool{
		"gateway-required": false, "protocol-anthropic-messages": false,
		"protocol-openai-responses": false, "protocol-openai-chat": false,
		"protocol-antigravity-direct": false, "provider-claude": false,
		"provider-codex-openai": false, "provider-kimi": false,
		"provider-glm": false, "provider-nvidia": false,
		"provider-antigravity": false, "capacity-tier-20": false,
		"capacity-tier-50": false, "capacity-tier-100": false,
		"smart-context": false, "cross-model-fallback": false,
	}
	for _, gate := range p.Gates {
		if _, ok := requiredGates[gate.ID]; !ok || requiredGates[gate.ID] {
			return fmt.Errorf("unknown or duplicate rollout gate %q", gate.ID)
		}
		if gate.DefaultEnabled || len(gate.Prerequisites) == 0 || len(gate.Evidence) == 0 || gate.RollbackFlag == "" {
			return fmt.Errorf("rollout gate %q must be default-off and complete", gate.ID)
		}
		requiredGates[gate.ID] = true
	}
	for id, present := range requiredGates {
		if !present {
			return fmt.Errorf("missing rollout gate %q", id)
		}
	}
	lastOrder := 0
	for _, cohort := range p.Cohorts {
		if cohort.Order <= lastOrder || cohort.MaxTaskTier > 20 || len(cohort.EntryEvidence) == 0 || len(cohort.ExitEvidence) == 0 || len(cohort.ExpansionBlock) == 0 {
			return fmt.Errorf("invalid or unauthorized canary cohort %q", cohort.ID)
		}
		lastOrder = cohort.Order
	}
	if len(p.Triggers) == 0 {
		return fmt.Errorf("rollback triggers are required")
	}
	for _, trigger := range p.Triggers {
		if trigger.ID == "" || trigger.Condition == "" || trigger.Action == "" || trigger.Runbook == "" || trigger.EvidenceTarget == "" {
			return fmt.Errorf("incomplete rollback trigger")
		}
	}
	return nil
}
