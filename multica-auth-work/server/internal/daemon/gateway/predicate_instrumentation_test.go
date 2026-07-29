package gateway

import (
	"context"
	"log/slog"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// capturingHandler records the attributes of the most recent slog record so a
// test can assert the exact fields emitted by ReadinessChecker.emitPredicates.
type capturingHandler struct {
	msg   string
	attrs map[string]slog.Value
}

func (h *capturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.msg = r.Message
	h.attrs = make(map[string]slog.Value)
	r.Attrs(func(a slog.Attr) bool {
		h.attrs[a.Key] = a.Value
		return true
	})
	return nil
}

func (h *capturingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(string) slog.Handler      { return h }

func newCaptureChecker() (*ReadinessChecker, *capturingHandler) {
	h := &capturingHandler{}
	c := &ReadinessChecker{}
	c.SetDiagnosticsLogger(slog.New(h))
	return c, h
}

func attrBool(t *testing.T, h *capturingHandler, key string, want bool) {
	t.Helper()
	v, ok := h.attrs[key]
	if !ok {
		t.Fatalf("missing attr %q", key)
	}
	if v.Kind() != slog.KindBool || v.Bool() != want {
		t.Fatalf("attr %q = %v, want bool %v", key, v, want)
	}
}

func attrStr(t *testing.T, h *capturingHandler, key, want string) {
	t.Helper()
	v, ok := h.attrs[key]
	if !ok {
		t.Fatalf("missing attr %q", key)
	}
	if v.Kind() != slog.KindString || v.String() != want {
		t.Fatalf("attr %q = %v, want string %q", key, v, want)
	}
}

func attrInt(t *testing.T, h *capturingHandler, key string, want int) {
	t.Helper()
	v, ok := h.attrs[key]
	if !ok {
		t.Fatalf("missing attr %q", key)
	}
	if v.Kind() != slog.KindInt64 || v.Int64() != int64(want) {
		t.Fatalf("attr %q = %v, want int %d", key, v, want)
	}
}

// TestEmitPredicatesCapturesAllFieldsOnSuccess proves every strict-readiness
// predicate field is emitted and that a successful check reports ok=true with
// empty failure fields.
func TestEmitPredicatesCapturesAllFieldsOnSuccess(t *testing.T) {
	c, h := newCaptureChecker()
	req := brain.ReadinessRequest{
		RouteModel: brain.RouteModel("cp/cline-pass/glm-5.2"),
		Protocol:   brain.ProtocolOpenAIChat,
	}
	snap := brain.ReadinessSnapshot{
		Live: true, Authenticated: true, ModelRegistryReady: true,
		SelectedModelReady: true, SelectedProtocolReady: true,
	}
	c.emitPredicates(req, snap, "reg-v1", nil)

	if h.msg != "strict_readiness_predicate" {
		t.Fatalf("diagnostic message = %q, want strict_readiness_predicate", h.msg)
	}
	attrStr(t, h, "route_model", "cp/cline-pass/glm-5.2")
	attrStr(t, h, "protocol", string(brain.ProtocolOpenAIChat))
	attrBool(t, h, "live", true)
	attrBool(t, h, "authenticated", true)
	attrBool(t, h, "model_registry_ready", true)
	attrBool(t, h, "selected_model_ready", true)
	attrBool(t, h, "selected_protocol_ready", true)
	attrBool(t, h, "registry_version_present", true)
	attrBool(t, h, "ok", true)
	attrStr(t, h, "fail_operation", "")
	attrStr(t, h, "fail_error_class", "")
	attrInt(t, h, "fail_status_code", 0)
}

// TestEmitPredicatesCapturesFailingSubcheckFromGatewayError proves the failing
// sub-check operation, error class, and transport status are captured from a
// *GatewayError, and that partial predicate state (authenticated=false,
// registry version absent) is reflected.
func TestEmitPredicatesCapturesFailingSubcheckFromGatewayError(t *testing.T) {
	c, h := newCaptureChecker()
	req := brain.ReadinessRequest{
		RouteModel: brain.RouteModel("cp/cline-pass/glm-5.2"),
		Protocol:   brain.ProtocolOpenAIChat,
	}
	// Liveness passed, then the authenticated readiness probe was rate-limited.
	snap := brain.ReadinessSnapshot{Live: true}
	gerr := &GatewayError{Operation: operationReadiness, Class: ErrorRateLimited, StatusCode: 429}
	c.emitPredicates(req, snap, "", gerr)

	attrBool(t, h, "ok", false)
	attrBool(t, h, "live", true)
	attrBool(t, h, "authenticated", false)
	attrBool(t, h, "registry_version_present", false)
	attrStr(t, h, "fail_operation", operationReadiness)
	attrStr(t, h, "fail_error_class", string(ErrorRateLimited))
	attrInt(t, h, "fail_status_code", 429)
}

// TestEmitPredicatesClassifiesNonGatewayError proves a non-*GatewayError is
// labelled non_gateway_error with no operation/status leakage.
func TestEmitPredicatesClassifiesNonGatewayError(t *testing.T) {
	c, h := newCaptureChecker()
	req := brain.ReadinessRequest{RouteModel: brain.RouteModel("cp/x/y"), Protocol: brain.ProtocolOpenAIChat}
	c.emitPredicates(req, brain.ReadinessSnapshot{}, "", context.Canceled)

	attrBool(t, h, "ok", false)
	attrStr(t, h, "fail_error_class", "non_gateway_error")
	attrStr(t, h, "fail_operation", "")
	attrInt(t, h, "fail_status_code", 0)
}

// TestEmitPredicatesNilDiagnosticsIsNoop proves emitPredicates is a safe no-op
// (no panic) when no diagnostics logger was configured.
func TestEmitPredicatesNilDiagnosticsIsNoop(t *testing.T) {
	c := &ReadinessChecker{} // diag == nil
	c.emitPredicates(brain.ReadinessRequest{}, brain.ReadinessSnapshot{}, "", nil)
}
