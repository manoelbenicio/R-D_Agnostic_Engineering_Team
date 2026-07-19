package observability

import (
	"fmt"
	"time"
)

type PanelSpec struct {
	ID          string
	Title       string
	Metrics     []string
	Aggregation string
	Purpose     string
}

type DashboardSpec struct {
	ID       string
	Title    string
	Audience string
	Panels   []PanelSpec
}

type AlertSeverity string

const (
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

type AlertSpec struct {
	ID           string
	Severity     AlertSeverity
	Signal       string
	Condition    string
	ThresholdRef string
	For          time.Duration
	Scope        string
	Runbook      string
	Evidence     string
}

type DashboardAlertCatalog struct {
	EvidenceID string
	Dashboards []DashboardSpec
	Alerts     []AlertSpec
}

func DefaultDashboardAlertCatalog() DashboardAlertCatalog {
	return DashboardAlertCatalog{
		EvidenceID: "EV-G2D-04",
		Dashboards: []DashboardSpec{
			{
				ID: "gateway-routing", Title: "Gateway readiness, routing, and eligibility", Audience: "OmniRoute operator and Agent Brain integrator",
				Panels: []PanelSpec{
					{"readiness", "Authenticated route readiness", []string{"omniroute_gateway_readiness", "omniroute_eligible_slots"}, "current and change by route/protocol/reason", "distinguish process liveness from usable inference"},
					{"selection", "Selection and affinity", []string{"omniroute_route_selection_total", "omniroute_affinity_total"}, "rate and outcome by safe reason", "explain strict rotation exceptions without account identity"},
					{"quota-circuit", "Quota and circuit state", []string{"omniroute_quota_state", "omniroute_circuit_state"}, "current state and transition count", "show why a route slot is ineligible"},
				},
			},
			{
				ID: "auth-errors-recovery", Title: "Authentication, errors, retry, and recovery", Audience: "Incident commander and OmniRoute operator",
				Panels: []PanelSpec{
					{"refresh", "Credential refresh outcomes", []string{"omniroute_refresh_total"}, "success/failure ratio by route and reason", "detect systemic refresh failure without identity"},
					{"upstream-errors", "401/403/429/5xx classes", []string{"omniroute_upstream_errors_total"}, "rate by status class and safe reason", "separate auth, entitlement, throttling, and upstream failure"},
					{"retry-fallback", "Retry and fallback", []string{"omniroute_retry_total", "omniroute_fallback_total", "omniroute_cancellations_total"}, "rate, outcome, and amplification", "confirm bounded recovery and cancellation"},
				},
			},
			{
				ID: "capacity-slo", Title: "Capacity, latency, errors, and resources", Audience: "Operations and product owner",
				Panels: []PanelSpec{
					{"tasks-queue", "Tasks, in-flight, queue, and overload", []string{"agent_brain_active_tasks", "omniroute_in_flight", "omniroute_queue_depth", "omniroute_overload_total"}, "current/max and rejection rate by tier", "prove bounded admission and identify saturation"},
					{"latency", "Selection, queue, first output, and total latency", []string{"omniroute_selection_seconds", "omniroute_queue_wait_seconds", "omniroute_time_to_first_token_seconds", "omniroute_request_duration_seconds"}, "p50/p95/p99 by route/protocol", "evaluate approved SLO profiles"},
					{"resources", "CPU, memory, and sockets", []string{"omniroute_process_cpu_ratio", "omniroute_process_memory_bytes", "omniroute_open_sockets"}, "current, peak, and recovery", "detect resource pressure and leaks"},
					{"usage", "Normalized usage", []string{"omniroute_usage_tokens_total"}, "rate and total by route/protocol/token class", "reconcile usage without account identity or content"},
				},
			},
		},
		Alerts: []AlertSpec{
			{"no-eligible-accounts", AlertCritical, "omniroute_eligible_slots", "minimum for an enabled route equals zero", "hard.zero_eligible_enabled_route", time.Minute, "route", "RB-INCIDENT", "EV-G4-07"},
			{"auth-refresh-failure", AlertCritical, "omniroute_refresh_total", "failure ratio exceeds approved profile", "slo.auth_refresh_failure_ratio", 2 * time.Minute, "route", "RB-INCIDENT", "EV-G4-05"},
			{"upstream-401-spike", AlertCritical, "omniroute_upstream_errors_total", "401 rate exceeds approved baseline", "slo.upstream_401_rate", 2 * time.Minute, "route", "RB-INCIDENT", "EV-G4-05"},
			{"upstream-403-spike", AlertCritical, "omniroute_upstream_errors_total", "403 rate exceeds approved baseline", "slo.upstream_403_rate", 2 * time.Minute, "route", "RB-INCIDENT", "EV-G4-05"},
			{"account-429-spike", AlertWarning, "omniroute_upstream_errors_total", "account-scoped 429 rate exceeds approved profile", "slo.account_429_rate", 5 * time.Minute, "route", "RB-INCIDENT", "EV-G4-05"},
			{"provider-global-429", AlertCritical, "omniroute_upstream_errors_total", "provider-global 429 is sustained", "slo.provider_global_429_rate", time.Minute, "provider-scope", "RB-INCIDENT", "EV-G4-05"},
			{"upstream-5xx-spike", AlertCritical, "omniroute_upstream_errors_total", "5xx rate exceeds approved profile", "slo.upstream_5xx_rate", 5 * time.Minute, "route", "RB-INCIDENT", "EV-G4-05"},
			{"circuit-open-ratio", AlertCritical, "omniroute_circuit_state", "open circuit ratio exceeds approved profile", "slo.circuit_open_ratio", 2 * time.Minute, "route", "RB-INCIDENT", "EV-G4-05"},
			{"queue-growth", AlertWarning, "omniroute_queue_depth", "queue remains above approved utilization", "slo.queue_utilization", 5 * time.Minute, "route", "RB-INCIDENT", "EV-G4-CAP"},
			{"overload-rejections", AlertCritical, "omniroute_overload_total", "retryable overload rate exceeds approved profile", "slo.overload_rejection_rate", 5 * time.Minute, "tier", "RB-INCIDENT", "EV-G4-CAP"},
			{"cpu-pressure", AlertWarning, "omniroute_process_cpu_ratio", "CPU exceeds approved sustained level", "slo.cpu_ratio", 10 * time.Minute, "instance", "RB-INCIDENT", "EV-G4-CAP"},
			{"memory-pressure", AlertCritical, "omniroute_process_memory_bytes", "memory exceeds approved level or fails to recover", "slo.memory_bytes", 5 * time.Minute, "instance", "RB-INCIDENT", "EV-G4-CAP"},
			{"socket-pressure", AlertCritical, "omniroute_open_sockets", "socket count exceeds approved level or leaks after recovery", "slo.open_sockets", 5 * time.Minute, "instance", "RB-INCIDENT", "EV-G4-CAP"},
			{"selection-latency", AlertWarning, "omniroute_selection_seconds", "p95 exceeds approved profile", "slo.selection_p95", 10 * time.Minute, "route", "RB-INCIDENT", "EV-G4-CAP"},
			{"first-output-latency", AlertWarning, "omniroute_time_to_first_token_seconds", "p95 exceeds approved route profile", "slo.ttft_p95", 10 * time.Minute, "route", "RB-INCIDENT", "EV-G4-CAP"},
			{"request-latency", AlertWarning, "omniroute_request_duration_seconds", "p95 exceeds approved route profile", "slo.request_p95", 10 * time.Minute, "route", "RB-INCIDENT", "EV-G4-CAP"},
			{"error-slo-burn", AlertCritical, "omniroute_upstream_errors_total", "multi-window error-budget burn exceeds approved policy", "slo.error_budget_burn", 5 * time.Minute, "route/protocol", "RB-ROLLBACK", "EV-G4-CAP"},
		},
	}
}

func (c DashboardAlertCatalog) Validate(schema TelemetrySchema) error {
	if c.EvidenceID != "EV-G2D-04" {
		return fmt.Errorf("unexpected dashboard evidence id")
	}
	metrics := map[string]bool{}
	for _, metric := range schema.MetricCatalog {
		metrics[metric.Name] = true
	}
	seenDashboards := map[string]bool{}
	for _, dashboard := range c.Dashboards {
		if dashboard.ID == "" || seenDashboards[dashboard.ID] || dashboard.Title == "" || dashboard.Audience == "" || len(dashboard.Panels) == 0 {
			return fmt.Errorf("invalid dashboard")
		}
		seenDashboards[dashboard.ID] = true
		for _, panel := range dashboard.Panels {
			if panel.ID == "" || panel.Title == "" || panel.Aggregation == "" || panel.Purpose == "" || len(panel.Metrics) == 0 {
				return fmt.Errorf("incomplete dashboard panel")
			}
			for _, metric := range panel.Metrics {
				if !metrics[metric] {
					return fmt.Errorf("dashboard references unknown metric %q", metric)
				}
			}
		}
	}
	seenAlerts := map[string]bool{}
	for _, alert := range c.Alerts {
		if alert.ID == "" || seenAlerts[alert.ID] || !metrics[alert.Signal] || alert.Condition == "" || alert.ThresholdRef == "" || alert.For <= 0 || alert.Runbook == "" || alert.Evidence == "" {
			return fmt.Errorf("invalid alert specification")
		}
		seenAlerts[alert.ID] = true
	}
	return nil
}
