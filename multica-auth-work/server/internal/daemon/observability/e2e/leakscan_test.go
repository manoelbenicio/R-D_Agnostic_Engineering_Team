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
	clean := ScanLogLines([]string{"hop=cli", "outcome=exited", "code_0"})
	if !clean.Clean {
		t.Fatalf("expected clean log lines, got %+v", clean.Findings)
	}
	dirty := ScanLogLines([]string{"authorization: Bearer abc", "user@example.com"})
	if dirty.Clean {
		t.Fatalf("expected dirty log lines to fail closed")
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
