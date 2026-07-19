package dashboards

import (
	"time"
)

// G4OBSAcceptanceBundle declares PASS only when OBS-1..OBS-10 are each independently accepted,
// OBS-9 shows a continuous trace per synthetic task, and OBS-10 is clean.
type G4OBSAcceptanceBundle struct {
	OBS1_CorrelationSchema bool
	OBS2_IngressSpan       bool
	OBS3_QueueSpan         bool
	OBS4_AdmissionSpan     bool
	OBS5_CLISpan           bool
	OBS6_GatewaySpan       bool
	OBS7_PersistenceSpan   bool
	OBS8_WSSpan            bool
	OBS9_TraceContinuous   bool // shows a continuous trace per synthetic task
	OBS10_CleanLeakScan    bool // clean structural leak scan
}

// Pass returns true only when all prerequisites are met
func (b G4OBSAcceptanceBundle) Pass() bool {
	return b.OBS1_CorrelationSchema && b.OBS2_IngressSpan && b.OBS3_QueueSpan &&
		b.OBS4_AdmissionSpan && b.OBS5_CLISpan && b.OBS6_GatewaySpan &&
		b.OBS7_PersistenceSpan && b.OBS8_WSSpan && b.OBS9_TraceContinuous &&
		b.OBS10_CleanLeakScan
}

type OBS11Panel struct {
	Title       string
	Metrics     []string
	Aggregation string
	Purpose     string
}

type OBS11Dashboard struct {
	ID     string
	Title  string
	Panels []OBS11Panel
}

type OBS11Alert struct {
	ID        string
	Severity  string
	Metric    string
	Condition string
	For       time.Duration
}

// GetOBS11Dashboards returns per-hop latency/error/drop/gap dashboards using pseudonymous identifiers only.
func GetOBS11Dashboards() []OBS11Dashboard {
	return []OBS11Dashboard{
		{
			ID:    "obs11-spans",
			Title: "Per-hop Span Latency, Errors, Drops, and Gaps",
			Panels: []OBS11Panel{
				{"Hop Latency", []string{"obs_hop_latency_seconds"}, "p50/p95/p99 by hop", "monitor individual hop overhead (R30)"},
				{"Hop Errors", []string{"obs_hop_errors_total"}, "rate by hop and reason", "monitor errors at each observability hop"},
				{"Hop Drops", []string{"obs_hop_drops_total"}, "rate by hop", "monitor dropped spans at each hop"},
				{"Trace Gaps", []string{"obs_trace_gaps_total"}, "rate by task", "monitor disconnected or orphaned traces"},
			},
		},
	}
}

// GetOBS11Alerts returns alerts for hop latency, errors, drops, and gaps.
func GetOBS11Alerts() []OBS11Alert {
	return []OBS11Alert{
		{"hop-latency-spike", "warning", "obs_hop_latency_seconds", "p95 exceeds approved overhead", 5 * time.Minute},
		{"hop-error-rate", "critical", "obs_hop_errors_total", "error rate exceeds approved profile", 2 * time.Minute},
		{"hop-drops", "critical", "obs_hop_drops_total", "spans are being dropped", 2 * time.Minute},
		{"trace-gaps", "critical", "obs_trace_gaps_total", "traces are broken or orphaned", 2 * time.Minute},
	}
}
