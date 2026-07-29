package e2e

import "testing"

func TestScanCleanSyntheticTrace(t *testing.T) {
	report := ScanSpans(SyntheticTraceSpans("t1"))
	if !report.Clean {
		t.Fatalf("expected clean scan, got findings %+v", report.Findings)
	}
	if report.Scanned != 7 {
		t.Fatalf("expected 7 scanned, got %d", report.Scanned)
	}
}

func TestScanDetectsSecretsPresent(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	spans[0].SecretsPresent = true
	report := ScanSpans(spans)
	if report.Clean {
		t.Fatalf("expected leak from secrets_present")
	}
}

func TestScanDetectsBadLabelKey(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	if spans[0].Labels == nil {
		spans[0].Labels = map[string]string{}
	}
	spans[0].Labels["prompt"] = "safe" // disallowed key
	report := ScanSpans(spans)
	if report.Clean {
		t.Fatalf("expected leak from disallowed label key")
	}
}

func TestScanDetectsInlineSecretValues(t *testing.T) {
	values := []string{
		"user@example.com",
		"Bearer sk-abc",
		"https://evil/x",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signaturepart",
		"postgres://u:p@h:5432/db",
		"has space",
	}
	for _, v := range values {
		spans := SyntheticTraceSpans("t1")
		spans[4].Labels["route_model"] = v // route hop
		report := ScanSpans(spans)
		if report.Clean {
			t.Fatalf("expected leak for value %q", v)
		}
	}
}

func TestScanDetectsRawArgv(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	spans[3].ArgvShape = []string{"--api-key=sk-live"} // cli hop
	report := ScanSpans(spans)
	if report.Clean {
		t.Fatalf("expected leak from raw argv value")
	}
}

func TestScanLogLines(t *testing.T) {
	if report := ScanLogLines([]string{"hop=cli outcome=exited code_0"}); report.Clean {
		t.Fatal("free-form log line must fail closed even without a known secret marker")
	}
	if report := ScanLogLines([]string{""}); !report.Clean {
		t.Fatalf("empty log entry carries no content: %+v", report.Findings)
	}
}

func TestScanEventsUsesClosedStructuralShape(t *testing.T) {
	valid := Event{
		ContractVersion: ContractVersion,
		Kind:            EventHop,
		Hop:             HopRoute,
		Correlation:     Correlation{RequestID: "req-1", OmniRequestID: "omni-1"},
		Outcome:         "ok",
		Labels: map[string]string{
			"route_model":          "agy/claude-opus-4-6-thinking",
			"account_pseudonym":    "acct_0123456789abcdef",
			"connection_pseudonym": "conn_fedcba9876543210",
		},
		Counters:       map[string]int64{"retry_count": 1},
		SecretsPresent: false,
	}
	if report := ScanEvents([]Event{valid}); !report.Clean {
		t.Fatalf("valid event rejected: %+v", report.Findings)
	}

	tests := []struct {
		name   string
		mutate func(*Event)
	}{
		{name: "unknown event kind", mutate: func(event *Event) { event.Kind = "free_form" }},
		{name: "unsupported version", mutate: func(event *Event) { event.ContractVersion = "v999" }},
		{name: "unknown label", mutate: func(event *Event) { event.Labels["message"] = "looks-safe" }},
		{name: "raw account", mutate: func(event *Event) { event.Labels["account_pseudonym"] = "raw-account" }},
		{name: "wrong-hop counter", mutate: func(event *Event) { event.Counters["queue_depth"] = 1 }},
		{name: "secret invariant", mutate: func(event *Event) { event.SecretsPresent = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := valid
			event.Labels = make(map[string]string, len(valid.Labels))
			for key, value := range valid.Labels {
				event.Labels[key] = value
			}
			event.Counters = make(map[string]int64, len(valid.Counters))
			for key, value := range valid.Counters {
				event.Counters[key] = value
			}
			test.mutate(&event)
			if report := ScanEvents([]Event{event}); report.Clean {
				t.Fatal("structurally unsafe event accepted")
			}
		})
	}
}

func TestScanFromSink(t *testing.T) {
	sink := NewMemorySink()
	rec := NewRecorder(sink)
	if err := EmitSyntheticTask(rec, "t1"); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if !ScanFromSink(sink).Clean {
		t.Fatalf("expected clean sink scan")
	}
	if !ScanFromSink(nil).Clean {
		t.Fatalf("nil sink scan should be clean")
	}
}

func TestScanNegativeCounter(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	spans[0].Counters["latency_ms"] = -1
	if ScanSpans(spans).Clean {
		t.Fatalf("expected negative counter to fail")
	}
}

func TestScanRejectsCounterOutsideHopContract(t *testing.T) {
	spans := SyntheticTraceSpans("t1")
	spans[4].Counters["queue_depth"] = 1
	if ScanSpans(spans).Clean {
		t.Fatal("route span accepted queue-only counter")
	}
}
