package daemonws

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestEmitDelivery_ValidSpan(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := NewDeliveryRecorder(e2e.NewRecorder(sink))

	err := rec.EmitDelivery(DeliveryResult{
		SessionID:         "ses-abc123",
		DeliveryID:        "del-xyz789",
		Outcome:           "delivered",
		ReasonCode:        "",
		LatencyMs:         42,
		DropCount:         0,
		ReconnectCount:    0,
		BackpressureCount: 0,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Hop != e2e.HopDelivery {
		t.Errorf("hop = %q, want %q", s.Hop, e2e.HopDelivery)
	}
	if s.ContractVersion != e2e.ContractVersion {
		t.Errorf("contract_version = %q, want %q", s.ContractVersion, e2e.ContractVersion)
	}
	if s.Correlation.SessionID != "ses-abc123" {
		t.Errorf("session_id = %q, want %q", s.Correlation.SessionID, "ses-abc123")
	}
	if s.Correlation.DeliveryID != "del-xyz789" {
		t.Errorf("delivery_id = %q, want %q", s.Correlation.DeliveryID, "del-xyz789")
	}
	if s.Outcome != "delivered" {
		t.Errorf("outcome = %q, want %q", s.Outcome, "delivered")
	}
	if s.SecretsPresent {
		t.Error("secrets_present must be false")
	}
	if s.Counters["delivery_latency_ms"] != 42 {
		t.Errorf("delivery_latency_ms = %d, want 42", s.Counters["delivery_latency_ms"])
	}
	if !s.EndedAt.After(s.StartedAt) && !s.EndedAt.Equal(s.StartedAt) {
		t.Error("EndedAt should be >= StartedAt after Finish()")
	}
}

func TestEmitDelivery_Dropped(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := NewDeliveryRecorder(e2e.NewRecorder(sink))

	err := rec.EmitDelivery(DeliveryResult{
		SessionID:         "ses-drop1",
		DeliveryID:        "del-drop1",
		Outcome:           "dropped",
		ReasonCode:        "slow_consumer",
		LatencyMs:         0,
		DropCount:         3,
		ReconnectCount:    1,
		BackpressureCount: 2,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	spans := sink.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Outcome != "dropped" {
		t.Errorf("outcome = %q, want %q", s.Outcome, "dropped")
	}
	if s.ReasonCode != "slow_consumer" {
		t.Errorf("reason_code = %q, want %q", s.ReasonCode, "slow_consumer")
	}
	if s.Counters["drop_count"] != 3 {
		t.Errorf("drop_count = %d, want 3", s.Counters["drop_count"])
	}
	if s.Counters["reconnect_count"] != 1 {
		t.Errorf("reconnect_count = %d, want 1", s.Counters["reconnect_count"])
	}
	if s.Counters["backpressure_count"] != 2 {
		t.Errorf("backpressure_count = %d, want 2", s.Counters["backpressure_count"])
	}
}

func TestEmitDelivery_MissingSessionID_Rejected(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := NewDeliveryRecorder(e2e.NewRecorder(sink))

	err := rec.EmitDelivery(DeliveryResult{
		SessionID:  "",
		DeliveryID: "del-abc",
		Outcome:    "delivered",
	})
	if err == nil {
		t.Fatal("expected error for missing session_id, got nil")
	}
	if sink.Len() != 0 {
		t.Errorf("expected 0 recorded spans, got %d", sink.Len())
	}
}

func TestEmitDelivery_MissingDeliveryID_Rejected(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := NewDeliveryRecorder(e2e.NewRecorder(sink))

	err := rec.EmitDelivery(DeliveryResult{
		SessionID:  "ses-abc",
		DeliveryID: "",
		Outcome:    "delivered",
	})
	if err == nil {
		t.Fatal("expected error for missing delivery_id, got nil")
	}
	if sink.Len() != 0 {
		t.Errorf("expected 0 recorded spans, got %d", sink.Len())
	}
}

func TestEmitDelivery_NilRecorder_NoOp(t *testing.T) {
	rec := NewDeliveryRecorder(nil)
	err := rec.EmitDelivery(DeliveryResult{
		SessionID:  "ses-abc",
		DeliveryID: "del-abc",
		Outcome:    "delivered",
	})
	if err != nil {
		t.Fatalf("nil recorder should silently succeed, got: %v", err)
	}
}
