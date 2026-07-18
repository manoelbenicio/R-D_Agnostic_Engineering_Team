package observability

import (
	"testing"
	"time"
)

func TestDefaultTelemetrySchema(t *testing.T) {
	schema := DefaultTelemetrySchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("telemetry schema: %v", err)
	}
	if schema.ContentCapture {
		t.Fatal("telemetry schema enables content capture")
	}
}

func TestSafeEventValidation(t *testing.T) {
	event := SafeEvent{
		SchemaVersion:  EventSchemaVersion,
		Kind:           EventRouteSelection,
		At:             time.Unix(1, 0).UTC(),
		RequestID:      "request-1",
		ConnectionSlot: "slot-3",
		Outcome:        "selected",
		ReasonCode:     "strict-round-robin",
		CapacityTier:   20,
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("safe event: %v", err)
	}
	event.ConnectionSlot = "personal account"
	if err := event.Validate(); err == nil {
		t.Fatal("unsafe connection identity was accepted")
	}
}

func TestMetricLabelsAreAllowlisted(t *testing.T) {
	schema := DefaultTelemetrySchema()
	schema.MetricCatalog[0].Labels = append(schema.MetricCatalog[0].Labels, "account_id")
	if err := schema.Validate(); err == nil {
		t.Fatal("unsafe metric label was accepted")
	}
}

func TestDashboardAlertCatalog(t *testing.T) {
	schema := DefaultTelemetrySchema()
	catalog := DefaultDashboardAlertCatalog()
	if err := catalog.Validate(schema); err != nil {
		t.Fatalf("dashboard and alert catalog: %v", err)
	}
}

func TestAcceptanceHarnessIsSpecificationOnly(t *testing.T) {
	spec := DefaultAcceptanceHarnessSpec()
	if err := spec.Validate(); err != nil {
		t.Fatalf("acceptance harness: %v", err)
	}
	for _, profile := range spec.Profiles {
		if profile.RunnableNow {
			t.Fatalf("tier %d is runnable during G2D", profile.Tier)
		}
	}
}
