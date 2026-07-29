package gateway

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// Proves the gateway captures OmniRoute's own selection telemetry from a live
// response — the primitive for 8.4 production evidence (OmniRoute-owned
// round-robin/affinity), with no raw account identity exposed.
func TestParseSelectionTelemetryCapturesOmniRouteSelectionHeaders(t *testing.T) {
	cases := []struct {
		name       string
		reasonHdr  string
		wantReason SelectionReason
	}{
		{"independent_round_robin", string(SelectionIndependentRotation), SelectionIndependentRotation},
		{"continuation_affinity", string(SelectionContinuation), SelectionContinuation},
		{"prompt_cache_affinity", string(SelectionPromptCache), SelectionPromptCache},
		{"tool_turn_affinity", string(SelectionToolTurn), SelectionToolTurn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := make(http.Header)
			h.Set(HeaderOmniRouteRequestID, "omni-req-1")
			h.Set(HeaderActualModel, "claude_code_kimi_2.7_Code")
			h.Set(HeaderActualRoute, "route-anthropic-primary")
			h.Set(HeaderAccountID, "acct-secret-raw-123")
			h.Set(HeaderConnectionID, "conn-secret-raw-456")
			h.Set(HeaderSelectionReason, tc.reasonHdr)
			h.Set(HeaderRetryCount, "0")
			h.Set(HeaderFallbackUsed, "false")
			h.Set(HeaderUsageInput, "10")
			h.Set(HeaderUsageOutput, "20")
			resp := &http.Response{Header: h}

			tel, err := ParseSelectionTelemetry(resp)
			if err != nil {
				t.Fatalf("ParseSelectionTelemetry: %v", err)
			}
			if tel.ActualRoute != "route-anthropic-primary" {
				t.Fatalf("route=%q", tel.ActualRoute)
			}
			if tel.ActualModel != brain.RouteModel("claude_code_kimi_2.7_Code") {
				t.Fatalf("model=%q", tel.ActualModel)
			}
			if tel.SelectionReason != tc.wantReason {
				t.Fatalf("selection_reason=%q want %q", tel.SelectionReason, tc.wantReason)
			}
			// Account/connection must be pseudonymous — present but NOT the raw ids.
			if tel.PseudonymousAccount == "" || tel.PseudonymousAccount == "acct-secret-raw-123" {
				t.Fatalf("account not pseudonymized: %q", tel.PseudonymousAccount)
			}
			if tel.PseudonymousConnection == "" || tel.PseudonymousConnection == "conn-secret-raw-456" {
				t.Fatalf("connection not pseudonymized: %q", tel.PseudonymousConnection)
			}
			// No raw identity anywhere in the record's string form.
			if s := fmt.Sprintf("%+v", tel); strings.Contains(s, "acct-secret-raw-123") || strings.Contains(s, "conn-secret-raw-456") {
				t.Fatalf("raw identity leaked: %s", s)
			}
		})
	}
}

func TestParseSelectionTelemetryRejectsNilResponse(t *testing.T) {
	if _, err := ParseSelectionTelemetry(nil); !IsErrorClass(err, ErrorInvalidRequest) {
		t.Fatalf("nil response should be invalid_request, got %v", err)
	}
}
