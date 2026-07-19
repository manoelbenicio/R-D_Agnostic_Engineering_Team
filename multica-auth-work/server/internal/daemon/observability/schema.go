package observability

import (
	"fmt"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const EventSchemaVersion = "agent-brain.observability.v1"

type EventKind string

const (
	EventAdmissionDecision EventKind = "admission.decision"
	EventGatewayReadiness  EventKind = "gateway.readiness"
	EventRouteSelection    EventKind = "route.selection"
	EventRouteAffinity     EventKind = "route.affinity"
	EventCredentialRefresh EventKind = "credential.refresh"
	EventQuotaState        EventKind = "quota.state"
	EventUpstream401       EventKind = "upstream.401"
	EventUpstream403       EventKind = "upstream.403"
	EventUpstream429       EventKind = "upstream.429"
	EventUpstream5xx       EventKind = "upstream.5xx"
	EventCircuitState      EventKind = "circuit.state"
	EventRequestRetry      EventKind = "request.retry"
	EventRequestFallback   EventKind = "request.fallback"
	EventCancellation      EventKind = "request.cancellation"
	EventUsage             EventKind = "usage.report"
	EventOverload          EventKind = "overload.rejection"
)

type UsageCounters struct {
	InputTokens     int64 `json:"input_tokens,omitempty"`
	OutputTokens    int64 `json:"output_tokens,omitempty"`
	CacheReadTokens int64 `json:"cache_read_tokens,omitempty"`
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"`
}

// SafeEvent has no content, authorization, cookie, account identity, or free-
// form payload field. ConnectionSlot must be an ephemeral pseudonymous routing
// slot, never an upstream account identifier or personal label.
type SafeEvent struct {
	SchemaVersion    string           `json:"schema_version"`
	Kind             EventKind        `json:"kind"`
	At               time.Time        `json:"at"`
	TaskID           string           `json:"task_id,omitempty"`
	SessionID        string           `json:"session_id,omitempty"`
	RequestID        string           `json:"request_id"`
	GatewayRequestID string           `json:"gateway_request_id,omitempty"`
	RouteModel       brain.RouteModel `json:"route_model,omitempty"`
	Protocol         string           `json:"protocol,omitempty"`
	ConnectionSlot   string           `json:"connection_slot,omitempty"`
	Outcome          string           `json:"outcome"`
	ReasonCode       string           `json:"reason_code,omitempty"`
	HTTPStatus       int              `json:"http_status,omitempty"`
	RetryCount       int              `json:"retry_count,omitempty"`
	FallbackCount    int              `json:"fallback_count,omitempty"`
	CapacityTier     int              `json:"capacity_tier,omitempty"`
	QueueDepth       int              `json:"queue_depth,omitempty"`
	CircuitState     string           `json:"circuit_state,omitempty"`
	QuotaState       string           `json:"quota_state,omitempty"`
	AffinityReason   string           `json:"affinity_reason,omitempty"`
	Usage            UsageCounters    `json:"usage,omitempty"`
}

func RequiredEventKinds() []EventKind {
	return []EventKind{
		EventAdmissionDecision,
		EventGatewayReadiness,
		EventRouteSelection,
		EventRouteAffinity,
		EventCredentialRefresh,
		EventQuotaState,
		EventUpstream401,
		EventUpstream403,
		EventUpstream429,
		EventUpstream5xx,
		EventCircuitState,
		EventRequestRetry,
		EventRequestFallback,
		EventCancellation,
		EventUsage,
		EventOverload,
	}
}

func (e SafeEvent) Validate() error {
	if e.SchemaVersion != EventSchemaVersion {
		return fmt.Errorf("unsupported event schema version")
	}
	if !isRequiredEventKind(e.Kind) {
		return fmt.Errorf("unsupported event kind")
	}
	if e.At.IsZero() || !safeID(e.RequestID, 128) {
		return fmt.Errorf("event timestamp and safe request correlation are required")
	}
	if e.TaskID != "" && !safeID(e.TaskID, 128) {
		return fmt.Errorf("invalid task correlation")
	}
	if e.SessionID != "" && !safeID(e.SessionID, 128) {
		return fmt.Errorf("invalid session correlation")
	}
	if e.GatewayRequestID != "" && !safeID(e.GatewayRequestID, 128) {
		return fmt.Errorf("invalid gateway correlation")
	}
	if e.ConnectionSlot != "" && (!strings.HasPrefix(e.ConnectionSlot, "slot-") || !safeID(e.ConnectionSlot, 96)) {
		return fmt.Errorf("connection slot must be ephemeral and pseudonymous")
	}
	if !safeCode(e.Outcome, 64) || (e.ReasonCode != "" && !safeCode(e.ReasonCode, 96)) {
		return fmt.Errorf("outcome and reason must be bounded safe codes")
	}
	if e.RetryCount < 0 || e.FallbackCount < 0 || e.QueueDepth < 0 {
		return fmt.Errorf("event counters must not be negative")
	}
	if e.CapacityTier != 0 && e.CapacityTier != 20 && e.CapacityTier != 50 && e.CapacityTier != 100 {
		return fmt.Errorf("unsupported capacity tier")
	}
	if e.HTTPStatus < 0 || e.HTTPStatus > 599 {
		return fmt.Errorf("invalid HTTP status")
	}
	if e.Usage.InputTokens < 0 || e.Usage.OutputTokens < 0 || e.Usage.CacheReadTokens < 0 || e.Usage.ReasoningTokens < 0 {
		return fmt.Errorf("usage counters must not be negative")
	}
	return nil
}

func isRequiredEventKind(kind EventKind) bool {
	for _, candidate := range RequiredEventKinds() {
		if candidate == kind {
			return true
		}
	}
	return false
}

func safeID(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func safeCode(value string, max int) bool {
	return safeID(value, max)
}

type MetricType string

const (
	MetricCounter   MetricType = "counter"
	MetricGauge     MetricType = "gauge"
	MetricHistogram MetricType = "histogram"
)

type MetricSpec struct {
	Name        string
	Type        MetricType
	Unit        string
	Description string
	Labels      []string
}

type TelemetrySchema struct {
	EvidenceID       string
	Version          string
	ContentCapture   bool
	RequiredKinds    []EventKind
	MetricCatalog    []MetricSpec
	ProhibitedFields []string
}

func DefaultTelemetrySchema() TelemetrySchema {
	return TelemetrySchema{
		EvidenceID:     "EV-G2D-03",
		Version:        EventSchemaVersion,
		ContentCapture: false,
		RequiredKinds:  RequiredEventKinds(),
		MetricCatalog: []MetricSpec{
			{"agent_brain_admission_total", MetricCounter, "decisions", "Task admission decisions", []string{"outcome", "reason_code", "capacity_tier", "cohort"}},
			{"omniroute_gateway_readiness", MetricGauge, "state", "Authenticated selected-route readiness", []string{"route_model", "protocol", "outcome", "reason_code"}},
			{"omniroute_route_selection_total", MetricCounter, "selections", "Route selections by safe reason", []string{"route_model", "protocol", "outcome", "selection_reason"}},
			{"omniroute_affinity_total", MetricCounter, "decisions", "Continuation affinity decisions", []string{"route_model", "protocol", "outcome", "affinity_reason"}},
			{"omniroute_refresh_total", MetricCounter, "attempts", "Credential refresh outcomes without identity", []string{"route_model", "outcome", "reason_code"}},
			{"omniroute_quota_state", MetricGauge, "state", "Route-level quota eligibility state", []string{"route_model", "quota_state"}},
			{"omniroute_eligible_slots", MetricGauge, "slots", "Eligible pseudonymous route slots", []string{"route_model"}},
			{"omniroute_upstream_errors_total", MetricCounter, "errors", "Classified upstream errors", []string{"route_model", "protocol", "status_class", "reason_code"}},
			{"omniroute_circuit_state", MetricGauge, "state", "Scoped circuit state", []string{"route_model", "circuit_state"}},
			{"omniroute_retry_total", MetricCounter, "attempts", "Bounded retry outcomes", []string{"route_model", "protocol", "outcome", "reason_code"}},
			{"omniroute_fallback_total", MetricCounter, "attempts", "Fallback outcomes", []string{"route_model", "protocol", "outcome", "reason_code"}},
			{"omniroute_cancellations_total", MetricCounter, "cancellations", "Cancellation terminal outcomes", []string{"route_model", "protocol", "outcome"}},
			{"omniroute_usage_tokens_total", MetricCounter, "tokens", "Normalized safe usage", []string{"route_model", "protocol", "token_class"}},
			{"omniroute_overload_total", MetricCounter, "rejections", "Bounded overload decisions", []string{"route_model", "reason_code", "capacity_tier"}},
			{"agent_brain_active_tasks", MetricGauge, "tasks", "Active Agent Brain tasks", []string{"capacity_tier", "cohort"}},
			{"omniroute_in_flight", MetricGauge, "requests", "Active inference requests", []string{"route_model", "protocol"}},
			{"omniroute_queue_depth", MetricGauge, "requests", "Bounded inference queue depth", []string{"route_model"}},
			{"omniroute_queue_wait_seconds", MetricHistogram, "seconds", "Inference queue wait", []string{"route_model", "protocol"}},
			{"omniroute_selection_seconds", MetricHistogram, "seconds", "Account selection latency", []string{"route_model", "protocol"}},
			{"omniroute_time_to_first_token_seconds", MetricHistogram, "seconds", "Time to first output", []string{"route_model", "protocol"}},
			{"omniroute_request_duration_seconds", MetricHistogram, "seconds", "End-to-end request duration", []string{"route_model", "protocol", "outcome"}},
			{"omniroute_process_cpu_ratio", MetricGauge, "ratio", "OmniRoute process CPU pressure", nil},
			{"omniroute_process_memory_bytes", MetricGauge, "bytes", "OmniRoute process memory", nil},
			{"omniroute_open_sockets", MetricGauge, "sockets", "OmniRoute open sockets", nil},
		},
		ProhibitedFields: []string{
			"authorization", "credential", "secret", "cookie", "account_id", "account_email",
			"prompt", "completion", "message_content", "tool_arguments", "tool_result",
			"repository_content", "reasoning_content", "raw_request", "raw_response",
		},
	}
}

func (s TelemetrySchema) Validate() error {
	if s.EvidenceID != "EV-G2D-03" || s.Version != EventSchemaVersion || s.ContentCapture {
		return fmt.Errorf("telemetry schema must be versioned and content-off")
	}
	wantKinds := RequiredEventKinds()
	if len(s.RequiredKinds) != len(wantKinds) {
		return fmt.Errorf("telemetry schema is missing required event kinds")
	}
	seenKinds := map[EventKind]bool{}
	for _, kind := range s.RequiredKinds {
		if !isRequiredEventKind(kind) || seenKinds[kind] {
			return fmt.Errorf("invalid or duplicate event kind")
		}
		seenKinds[kind] = true
	}
	allowedLabels := map[string]bool{
		"route_model": true, "protocol": true, "outcome": true, "reason_code": true,
		"capacity_tier": true, "cohort": true, "selection_reason": true,
		"affinity_reason": true, "quota_state": true, "status_class": true,
		"circuit_state": true, "token_class": true,
	}
	seenMetrics := map[string]bool{}
	for _, metric := range s.MetricCatalog {
		if metric.Name == "" || seenMetrics[metric.Name] || metric.Description == "" {
			return fmt.Errorf("invalid metric specification")
		}
		seenMetrics[metric.Name] = true
		for _, label := range metric.Labels {
			if !allowedLabels[label] {
				return fmt.Errorf("metric %q uses unsafe or high-cardinality label %q", metric.Name, label)
			}
		}
	}
	if len(s.ProhibitedFields) == 0 {
		return fmt.Errorf("telemetry prohibited-field list is required")
	}
	return nil
}
