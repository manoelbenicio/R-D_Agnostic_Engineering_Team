package deploy

import "fmt"

type RunbookID string

const (
	RunbookBackupRestore RunbookID = "RB-BACKUP-RESTORE"
	RunbookAccountChange RunbookID = "RB-ACCOUNT-HOT-CHANGE"
	RunbookRouteChange   RunbookID = "RB-ROUTE-HOT-CHANGE"
	RunbookKeyRotation   RunbookID = "RB-KEY-ROTATION"
	RunbookUpgrade       RunbookID = "RB-UPGRADE"
	RunbookRollback      RunbookID = "RB-ROLLBACK"
	RunbookIncident      RunbookID = "RB-INCIDENT"
	RunbookEscalation    RunbookID = "RB-ESCALATION"
)

type Runbook struct {
	ID            RunbookID
	Title         string
	OwnerRole     string
	Preconditions []string
	Actions       []string
	Verification  []string
	Rollback      []string
	Evidence      []string
	Prohibited    []string
}

type IncidentSeverity string

const (
	SeverityZero  IncidentSeverity = "SEV0"
	SeverityOne   IncidentSeverity = "SEV1"
	SeverityTwo   IncidentSeverity = "SEV2"
	SeverityThree IncidentSeverity = "SEV3"
)

type IncidentClass struct {
	Code            string
	DefaultSeverity IncidentSeverity
	Condition       string
	ImmediateAction string
	EscalateTo      []string
}

type OperationsCatalog struct {
	EvidenceID string
	Runbooks   []Runbook
	Incidents  []IncidentClass
}

func DefaultOperationsCatalog() OperationsCatalog {
	return OperationsCatalog{
		EvidenceID: "EV-G2D-06",
		Runbooks: []Runbook{
			{
				ID: RunbookBackupRestore, Title: "Single-node state backup and restore", OwnerRole: "OmniRoute operator",
				Preconditions: []string{"hold new admissions", "identify pinned image/config revision", "verify encrypted destination and retention policy"},
				Actions:       []string{"quiesce or checkpoint the supported state backend", "create an encrypted atomic backup through the supported mechanism", "restore into an isolated validation target before production selection"},
				Verification:  []string{"state schema/version matches", "route/account counts reconcile without identities", "authenticated readiness and a synthetic request pass"},
				Rollback:      []string{"retain the untouched prior state", "return to the last accepted image/config/state generation"},
				Evidence:      []string{"backup generation", "schema version", "duration", "restore validation result"},
				Prohibited:    []string{"copy plaintext credentials into general backups", "log database rows", "resume admission before readiness"},
			},
			{
				ID: RunbookAccountChange, Title: "Account add, disable, quarantine, remove, and re-entry", OwnerRole: "OmniRoute operator",
				Preconditions: []string{"approved change ticket", "healthy alternate capacity", "redacted baseline eligibility snapshot"},
				Actions:       []string{"apply one atomic account-state change", "preserve in-flight policy", "observe selection and cooldown state before the next change"},
				Verification:  []string{"new independent requests use only eligible pseudonymous slots", "continuations remain affinitized", "distribution changes are explainable"},
				Rollback:      []string{"restore prior eligibility generation", "hold admissions if safe capacity is unavailable"},
				Evidence:      []string{"change revision", "eligible slot counts", "selection outcomes", "re-entry reason"},
				Prohibited:    []string{"expose account identity", "change multiple independent controls without an intermediate check"},
			},
			{
				ID: RunbookRouteChange, Title: "Atomic route/model/pool policy change", OwnerRole: "OmniRoute architect/operator",
				Preconditions: []string{"versioned validated policy", "exact model capability row", "accepted rollback revision"},
				Actions:       []string{"stage policy without activation", "validate routes, pools, limits, affinity, fallback, and kill switches", "atomically activate one revision"},
				Verification:  []string{"registry revision matches", "unknown or unsupported models fail closed", "synthetic protocol and route selection pass"},
				Rollback:      []string{"atomically select previous accepted revision", "hold affected admissions until readiness recovers"},
				Evidence:      []string{"old/new revision", "validation result", "activation time", "actual route from synthetic request"},
				Prohibited:    []string{"partial file overwrite", "unapproved cross-model fallback", "provider-direct fallback"},
			},
			{
				ID: RunbookKeyRotation, Title: "Scoped OmniRoute inference credential rotation", OwnerRole: "Security operator",
				Preconditions: []string{"approved rotation window", "restricted reference metadata valid", "revocation and rollback authority available"},
				Actions:       []string{"provision the new value outside repository/image/logging surfaces", "atomically replace the restricted reference target", "reload or restart through the approved service procedure", "revoke the prior generation after readiness"},
				Verification:  []string{"authenticated readiness succeeds", "prior generation is rejected", "no provider-native credential is introduced"},
				Rollback:      []string{"atomically restore the operator-controlled prior reference generation if still authorized", "hold admissions while authentication is unavailable"},
				Evidence:      []string{"rotation generation", "metadata validation", "readiness transition", "revocation outcome code"},
				Prohibited:    []string{"print, hash, or fingerprint the value in general telemetry", "put the value on a command line", "copy it into task homes"},
			},
			{
				ID: RunbookUpgrade, Title: "Pinned OmniRoute upgrade", OwnerRole: "OmniRoute operator",
				Preconditions: []string{"immutable candidate digest", "release/config/state compatibility statement", "backup and rollback target validated"},
				Actions:       []string{"hold or drain admissions", "checkpoint state", "deploy candidate to synthetic cohort", "run readiness/protocol/failure gates", "expand only through approved cohort stages"},
				Verification:  []string{"schema migration reconciles", "readiness and selected model capability pass", "error/latency/security triggers remain within approved bounds"},
				Rollback:      []string{"invoke RB-ROLLBACK on any mandatory trigger"},
				Evidence:      []string{"old/new digest", "config revision", "gate results", "cohort timeline"},
				Prohibited:    []string{"use a mutable tag as release identity", "enable tier 50/100", "remove Prodex during this phase"},
			},
			{
				ID: RunbookRollback, Title: "Safe release/config rollback", OwnerRole: "Incident commander",
				Preconditions: []string{"rollback trigger recorded", "last accepted digest/config/state compatibility known"},
				Actions:       []string{"stop cohort expansion", "hold new affected admissions", "drain or cancel according to commit safety", "select prior accepted release/config", "repeat authenticated readiness"},
				Verification:  []string{"no dual router owner", "no direct-provider credential path", "counters and terminal results reconcile"},
				Rollback:      []string{"if the prior release is not ready, keep admissions closed and escalate; do not reactivate legacy routing"},
				Evidence:      []string{"trigger", "decision time", "selected revision", "recovery duration", "affected task counts"},
				Prohibited:    []string{"restore provider credentials", "start Prodex/L2", "silently replay partial output or tool actions"},
			},
			{
				ID: RunbookIncident, Title: "Incident classification and containment", OwnerRole: "Incident commander",
				Preconditions: []string{"safe correlation IDs and redacted symptoms available"},
				Actions:       []string{"classify severity and scope", "apply the narrowest safe kill switch", "preserve redacted evidence", "hold or drain affected admissions", "invoke specialist runbook"},
				Verification:  []string{"blast radius stops growing", "no credential/content exposure in evidence", "owner and next update time assigned"},
				Rollback:      []string{"remove containment only after exit criteria and readiness pass"},
				Evidence:      []string{"incident code", "severity", "timeline", "correlation IDs", "safe counters", "decision log"},
				Prohibited:    []string{"inspect raw prompts or tool payloads", "publish account identity", "bypass fail-closed readiness"},
			},
			{
				ID: RunbookEscalation, Title: "Operational escalation", OwnerRole: "Incident commander",
				Preconditions: []string{"incident class and current containment recorded"},
				Actions:       []string{"page OmniRoute operator for state/routing", "page Agent Brain integrator for admission/lifecycle", "page Security for any auth/content boundary", "page product owner for route/capacity/waiver decisions"},
				Verification:  []string{"each role acknowledges", "one decision owner and update cadence are recorded"},
				Rollback:      []string{"retain containment until the designated owner approves exit"},
				Evidence:      []string{"role acknowledgements", "escalation timestamps", "decision owner"},
				Prohibited:    []string{"share credentials or raw content in paging systems"},
			},
		},
		Incidents: []IncidentClass{
			{"SEC-AUTH-BOUNDARY", SeverityZero, "credential exposure, authentication bypass, or management/inference authorization crossover", "revoke or isolate affected access and hold admissions", []string{"Security", "OmniRoute operator", "Agent Brain integrator"}},
			{"ROUTER-DUAL-OWNER", SeverityZero, "provider-direct, Prodex, or legacy router participates in a gateway-required request", "stop affected admissions and apply gateway kill switch", []string{"Agent Brain integrator", "Security", "Product owner"}},
			{"PROTOCOL-COMMIT", SeverityOne, "message/tool/continuation corruption or replay after commit", "disable exact route/protocol cohort", []string{"OmniRoute architect", "Agent Brain integrator"}},
			{"STATE-RECOVERY", SeverityOne, "state corruption, failed restore, or unsafe rotation cursor/affinity", "hold admissions and select last accepted state/release", []string{"OmniRoute operator", "Incident commander"}},
			{"AUTH-REFRESH", SeverityOne, "systemic refresh, 401, or 403 failure", "disable affected route/provider and preserve scoped evidence", []string{"OmniRoute operator", "Security"}},
			{"PROVIDER-THROTTLE", SeverityTwo, "provider-global 429 or sustained circuit-open ratio", "hold affected route and publish safe retry guidance", []string{"OmniRoute operator", "Product owner"}},
			{"CAPACITY-OVERLOAD", SeverityTwo, "bounded queue saturation, latency breach, or resource pressure", "cap admission at the accepted tier and shed retryable work", []string{"Agent Brain integrator", "OmniRoute operator"}},
			{"OBS-EVIDENCE-GAP", SeverityThree, "missing correlation, metric, or required acceptance evidence without active safety impact", "stop gate promotion and repair evidence", []string{"Operations/evidence owner"}},
		},
	}
}

func (c OperationsCatalog) Validate() error {
	if c.EvidenceID != "EV-G2D-06" {
		return fmt.Errorf("unexpected operations evidence id")
	}
	required := map[RunbookID]bool{
		RunbookBackupRestore: false,
		RunbookAccountChange: false,
		RunbookRouteChange:   false,
		RunbookKeyRotation:   false,
		RunbookUpgrade:       false,
		RunbookRollback:      false,
		RunbookIncident:      false,
		RunbookEscalation:    false,
	}
	for _, runbook := range c.Runbooks {
		if _, ok := required[runbook.ID]; !ok {
			return fmt.Errorf("unknown runbook %q", runbook.ID)
		}
		if required[runbook.ID] {
			return fmt.Errorf("duplicate runbook %q", runbook.ID)
		}
		if runbook.Title == "" || runbook.OwnerRole == "" || len(runbook.Actions) == 0 ||
			len(runbook.Verification) == 0 || len(runbook.Rollback) == 0 || len(runbook.Evidence) == 0 || len(runbook.Prohibited) == 0 {
			return fmt.Errorf("runbook %q is incomplete", runbook.ID)
		}
		required[runbook.ID] = true
	}
	for id, present := range required {
		if !present {
			return fmt.Errorf("missing runbook %q", id)
		}
	}
	if len(c.Incidents) == 0 {
		return fmt.Errorf("incident classification is required")
	}
	seen := map[string]bool{}
	for _, incident := range c.Incidents {
		if incident.Code == "" || seen[incident.Code] || incident.Condition == "" || incident.ImmediateAction == "" || len(incident.EscalateTo) == 0 {
			return fmt.Errorf("invalid incident class")
		}
		seen[incident.Code] = true
	}
	return nil
}
