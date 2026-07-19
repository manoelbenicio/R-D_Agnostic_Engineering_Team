package gateway

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const (
	HeaderOmniRouteRequestID = "X-OmniRoute-Request-Id"
	HeaderActualModel        = "X-OmniRoute-Actual-Model"
	HeaderActualRoute        = "X-OmniRoute-Route"
	HeaderAccountID          = "X-OmniRoute-Account-Id"
	HeaderConnectionID       = "X-OmniRoute-Connection-Id"
	HeaderSelectionReason    = "X-OmniRoute-Selection-Reason"
	HeaderRetryCount         = "X-OmniRoute-Retry-Count"
	HeaderFallbackUsed       = "X-OmniRoute-Fallback"
	HeaderQuotaState         = "X-OmniRoute-Quota-State"
	HeaderCircuitState       = "X-OmniRoute-Circuit-State"
	HeaderUsageInput         = "X-OmniRoute-Usage-Input"
	HeaderUsageOutput        = "X-OmniRoute-Usage-Output"
	HeaderUsageCache         = "X-OmniRoute-Usage-Cache"
	HeaderUsageReasoning     = "X-OmniRoute-Usage-Reasoning"
	HeaderUsageTotal         = "X-OmniRoute-Usage-Total"
)

type QuotaState string

const (
	QuotaUnknown   QuotaState = "unknown"
	QuotaAvailable QuotaState = "available"
	QuotaLimited   QuotaState = "limited"
	QuotaExhausted QuotaState = "exhausted"
)

type CircuitState string

const (
	CircuitUnknown  CircuitState = "unknown"
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

type SelectionReason string

const (
	SelectionUnknown             SelectionReason = "unknown"
	SelectionIndependentRotation SelectionReason = "independent-round-robin"
	SelectionContinuation        SelectionReason = "continuation-affinity"
	SelectionPromptCache         SelectionReason = "prompt-cache-affinity"
	SelectionToolTurn            SelectionReason = "tool-turn-affinity"
	SelectionRetry               SelectionReason = "retry"
	SelectionSameModelFallback   SelectionReason = "same-model-fallback"
	SelectionCrossModelFallback  SelectionReason = "cross-model-fallback"
	SelectionHalfOpenProbe       SelectionReason = "half-open-probe"
)

type Usage struct {
	Input     int64 `json:"input"`
	Output    int64 `json:"output"`
	Cache     int64 `json:"cache"`
	Reasoning int64 `json:"reasoning"`
	Total     int64 `json:"total"`
}

type Telemetry struct {
	RequestID              string           `json:"request_id,omitempty"`
	ActualModel            brain.RouteModel `json:"actual_model,omitempty"`
	ActualRoute            string           `json:"actual_route,omitempty"`
	PseudonymousAccount    string           `json:"pseudonymous_account,omitempty"`
	PseudonymousConnection string           `json:"pseudonymous_connection,omitempty"`
	SelectionReason        SelectionReason  `json:"selection_reason"`
	RetryCount             int              `json:"retry_count"`
	FallbackUsed           bool             `json:"fallback_used"`
	Quota                  QuotaState       `json:"quota_state"`
	Circuit                CircuitState     `json:"circuit_state"`
	Usage                  Usage            `json:"usage"`
}

type telemetryDocument struct {
	RequestID    string `json:"request_id"`
	ActualModel  string `json:"actual_model"`
	ActualRoute  string `json:"actual_route"`
	AccountID    string `json:"account_id"`
	ConnectionID string `json:"connection_id"`
	Selection    string `json:"selection_reason"`
	RetryCount   int    `json:"retry_count"`
	FallbackUsed bool   `json:"fallback_used"`
	QuotaState   string `json:"quota_state"`
	CircuitState string `json:"circuit_state"`
	Usage        Usage  `json:"usage"`
}

func ParseTelemetryHeaders(header http.Header) (Telemetry, error) {
	retryCount, err := parseBoundedInt(header.Get(HeaderRetryCount), 0, 100)
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	fallback, err := parseOptionalBool(header.Get(HeaderFallbackUsed))
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	usage, err := parseUsageHeaders(header)
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	document := telemetryDocument{
		RequestID:    header.Get(HeaderOmniRouteRequestID),
		ActualModel:  header.Get(HeaderActualModel),
		ActualRoute:  header.Get(HeaderActualRoute),
		AccountID:    header.Get(HeaderAccountID),
		ConnectionID: header.Get(HeaderConnectionID),
		Selection:    header.Get(HeaderSelectionReason),
		RetryCount:   retryCount,
		FallbackUsed: fallback,
		QuotaState:   header.Get(HeaderQuotaState),
		CircuitState: header.Get(HeaderCircuitState),
		Usage:        usage,
	}
	return sanitizeTelemetry(document)
}

func ParseTelemetryEvent(reader io.Reader, maxBytes int64) (Telemetry, error) {
	if maxBytes < 256 || maxBytes > 64<<10 {
		return Telemetry{}, &GatewayError{Operation: "telemetry.event", Class: ErrorInvalidConfiguration}
	}
	body, err := readBounded(reader, maxBytes)
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var document telemetryDocument
	if err := decoder.Decode(&document); err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	return sanitizeTelemetry(document)
}

func sanitizeTelemetry(document telemetryDocument) (Telemetry, error) {
	requestID := safeIdentifier(document.RequestID)
	actualRoute := safeIdentifier(document.ActualRoute)
	if strings.TrimSpace(document.RequestID) != "" && requestID == "" || strings.TrimSpace(document.ActualRoute) != "" && actualRoute == "" {
		return Telemetry{}, telemetryProtocolError()
	}
	var actualModel brain.RouteModel
	if strings.TrimSpace(document.ActualModel) != "" {
		model, err := brain.ParseRouteModel(document.ActualModel)
		if err != nil {
			return Telemetry{}, telemetryProtocolError()
		}
		actualModel = model
	}
	if document.RetryCount < 0 || document.RetryCount > 100 || validateUsage(document.Usage) != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	quota, err := parseQuota(document.QuotaState)
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	circuit, err := parseCircuit(document.CircuitState)
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	selection, err := parseSelectionReason(document.Selection)
	if err != nil {
		return Telemetry{}, telemetryProtocolError()
	}
	return Telemetry{
		RequestID:              requestID,
		ActualModel:            actualModel,
		ActualRoute:            actualRoute,
		PseudonymousAccount:    pseudonymizeIdentifier("acct_", document.AccountID),
		PseudonymousConnection: pseudonymizeIdentifier("conn_", document.ConnectionID),
		SelectionReason:        selection,
		RetryCount:             document.RetryCount,
		FallbackUsed:           document.FallbackUsed,
		Quota:                  quota,
		Circuit:                circuit,
		Usage:                  document.Usage,
	}, nil
}

func safeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return ""
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("._:-", character) {
			continue
		}
		return ""
	}
	return value
}

func pseudonymizeIdentifier(prefix, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 512 {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return prefix + hex.EncodeToString(digest[:8])
}

func parseQuota(value string) (QuotaState, error) {
	if strings.TrimSpace(value) == "" {
		return QuotaUnknown, nil
	}
	state := QuotaState(value)
	switch state {
	case QuotaUnknown, QuotaAvailable, QuotaLimited, QuotaExhausted:
		return state, nil
	default:
		return "", errors.New("invalid quota state")
	}
}

func parseCircuit(value string) (CircuitState, error) {
	if strings.TrimSpace(value) == "" {
		return CircuitUnknown, nil
	}
	state := CircuitState(value)
	switch state {
	case CircuitUnknown, CircuitClosed, CircuitOpen, CircuitHalfOpen:
		return state, nil
	default:
		return "", errors.New("invalid circuit state")
	}
}

func parseSelectionReason(value string) (SelectionReason, error) {
	if strings.TrimSpace(value) == "" {
		return SelectionUnknown, nil
	}
	reason := SelectionReason(value)
	switch reason {
	case SelectionUnknown,
		SelectionIndependentRotation,
		SelectionContinuation,
		SelectionPromptCache,
		SelectionToolTurn,
		SelectionRetry,
		SelectionSameModelFallback,
		SelectionCrossModelFallback,
		SelectionHalfOpenProbe:
		return reason, nil
	default:
		return "", errors.New("invalid selection reason")
	}
}

func parseUsageHeaders(header http.Header) (Usage, error) {
	values := []*int64{}
	usage := Usage{}
	values = append(values, &usage.Input, &usage.Output, &usage.Cache, &usage.Reasoning, &usage.Total)
	headers := []string{HeaderUsageInput, HeaderUsageOutput, HeaderUsageCache, HeaderUsageReasoning, HeaderUsageTotal}
	for index, name := range headers {
		value, err := parseBoundedInt64(header.Get(name), 0, 1<<50)
		if err != nil {
			return Usage{}, err
		}
		*values[index] = value
	}
	return usage, validateUsage(usage)
}

func validateUsage(usage Usage) error {
	values := []int64{usage.Input, usage.Output, usage.Cache, usage.Reasoning, usage.Total}
	for _, value := range values {
		if value < 0 || value > 1<<50 {
			return errors.New("invalid usage")
		}
	}
	return nil
}

func parseBoundedInt(raw string, minimum, maximum int) (int, error) {
	value, err := parseBoundedInt64(raw, int64(minimum), int64(maximum))
	return int(value), err
}

func parseBoundedInt64(raw string, minimum, maximum int64) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < minimum || value > maximum {
		return 0, errors.New("invalid numeric telemetry")
	}
	return value, nil
}

func parseOptionalBool(raw string) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func telemetryProtocolError() error {
	return &GatewayError{Operation: "telemetry.parse", Class: ErrorProtocol}
}
