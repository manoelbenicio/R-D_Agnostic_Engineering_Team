package gateway

import (
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

// ProviderSpanRecord is the terminal, metadata-only input for OBS-6.
// Pseudonyms are trace attributes only; they must never be promoted to metric
// label dimensions.
type ProviderSpanRecord struct {
	RequestID          string
	PrincipalPseudonym string
	Protocol           brain.ProtocolFamily
	Telemetry          Telemetry
	StartedAt          time.Time
	EndedAt            time.Time
	Outcome            string
	ReasonCode         string
	HTTPStatus         int
}

// EmitProviderSpan emits the OmniRoute/provider hop through the W5 correlation
// library. It accepts only sanitized gateway telemetry and the W5 closed
// metadata vocabulary; it has no content, credential, or free-form log field.
func EmitProviderSpan(recorder *e2e.Recorder, record ProviderSpanRecord) error {
	if recorder == nil || record.StartedAt.IsZero() || record.EndedAt.IsZero() || record.EndedAt.Before(record.StartedAt) {
		return providerSpanError(ErrorInvalidConfiguration)
	}
	if !approvedSpanProtocol(record.Protocol) || validateProviderSpanTelemetry(record.Telemetry) != nil {
		return providerSpanError(ErrorProtocol)
	}

	fallbackCount := int64(0)
	if record.Telemetry.FallbackUsed {
		fallbackCount = 1
	}
	span := e2e.NewSpan(e2e.HopRoute, e2e.Correlation{
		RequestID:     record.RequestID,
		OmniRequestID: record.Telemetry.RequestID,
	})
	span.StartedAt = record.StartedAt.UTC()
	span.EndedAt = record.EndedAt.UTC()
	span.
		WithLabel("route_model", string(record.Telemetry.ActualModel)).
		WithLabel("route", record.Telemetry.ActualRoute).
		WithLabel("protocol", string(record.Protocol)).
		WithLabel("principal_pseudonym", record.PrincipalPseudonym).
		WithLabel("account_pseudonym", record.Telemetry.PseudonymousAccount).
		WithLabel("connection_pseudonym", record.Telemetry.PseudonymousConnection).
		WithLabel("selection_reason", string(record.Telemetry.SelectionReason)).
		WithLabel("quota_state", string(record.Telemetry.Quota)).
		WithLabel("circuit_state", string(record.Telemetry.Circuit)).
		WithCounter("latency_ms", record.EndedAt.Sub(record.StartedAt).Milliseconds()).
		WithCounter("retry_count", int64(record.Telemetry.RetryCount)).
		WithCounter("fallback_count", fallbackCount).
		WithCounter("input_tokens", record.Telemetry.Usage.Input).
		WithCounter("output_tokens", record.Telemetry.Usage.Output).
		WithCounter("cache_tokens", record.Telemetry.Usage.Cache).
		WithCounter("reasoning_tokens", record.Telemetry.Usage.Reasoning).
		WithCounter("total_tokens", record.Telemetry.Usage.Total).
		WithHTTPStatus(record.HTTPStatus).
		WithOutcome(record.Outcome, record.ReasonCode)

	if err := recorder.Emit(span); err != nil {
		return providerSpanError(ErrorProtocol)
	}
	return nil
}

func approvedSpanProtocol(protocol brain.ProtocolFamily) bool {
	switch protocol {
	case brain.ProtocolAnthropicMessages,
		brain.ProtocolOpenAIResponses,
		brain.ProtocolOpenAIChat,
		brain.ProtocolAntigravity:
		return true
	default:
		return false
	}
}

func validateProviderSpanTelemetry(telemetry Telemetry) error {
	if telemetry.RequestID == "" ||
		telemetry.ActualModel == "" ||
		telemetry.ActualRoute == "" ||
		telemetry.PseudonymousAccount == "" ||
		telemetry.PseudonymousConnection == "" {
		return providerSpanError(ErrorProtocol)
	}
	if _, err := brain.ParseRouteModel(string(telemetry.ActualModel)); err != nil {
		return providerSpanError(ErrorProtocol)
	}
	if safeIdentifier(telemetry.ActualRoute) != telemetry.ActualRoute {
		return providerSpanError(ErrorProtocol)
	}
	if telemetry.SelectionReason == "" || telemetry.Quota == "" || telemetry.Circuit == "" {
		return providerSpanError(ErrorProtocol)
	}
	if _, err := parseSelectionReason(string(telemetry.SelectionReason)); err != nil {
		return providerSpanError(ErrorProtocol)
	}
	if _, err := parseQuota(string(telemetry.Quota)); err != nil {
		return providerSpanError(ErrorProtocol)
	}
	if _, err := parseCircuit(string(telemetry.Circuit)); err != nil {
		return providerSpanError(ErrorProtocol)
	}
	if telemetry.RetryCount < 0 || telemetry.RetryCount > 100 || validateUsage(telemetry.Usage) != nil {
		return providerSpanError(ErrorProtocol)
	}
	return nil
}

func providerSpanError(class ErrorClass) error {
	return &GatewayError{Operation: "observability.route_span", Class: class}
}
