package e2e

import "fmt"

// LeakFinding locates a structural leak and states why it was rejected. Reasons
// never echo the offending value.
type LeakFinding struct {
	Location string `json:"location"`
	Reason   string `json:"reason"`
}

// ScanReport is the OBS-10 result. Clean is true only when zero findings exist.
type ScanReport struct {
	Clean    bool          `json:"clean"`
	Scanned  int           `json:"scanned"`
	Findings []LeakFinding `json:"findings,omitempty"`
}

// scanSpan performs the STRUCTURAL (not pattern-only) leak sweep over a single
// span. It enforces the metadata-only shape of every field: correlation
// identifiers, outcome/reason codes, closed label-key set with key-aware value
// charset, numeric-only counters, structural argv shape, and the
// secrets_present invariant. It returns findings; empty means clean.
func scanSpan(s Span) []LeakFinding {
	var f []LeakFinding
	loc := func(field string) string { return fmt.Sprintf("span[%s].%s", s.Hop, field) }

	if s.ContractVersion != ContractVersion {
		f = append(f, LeakFinding{loc("contract_version"), "unsupported contract version"})
	}
	if s.SecretsPresent {
		f = append(f, LeakFinding{loc("secrets_present"), "secrets_present invariant violated"})
	}

	// Correlation identifiers must be safe tokens.
	ids := []struct {
		field IDField
		val   string
	}{
		{IDRequest, s.Correlation.RequestID}, {IDQueueMsg, s.Correlation.QueueMsgID},
		{IDTask, s.Correlation.TaskID}, {IDSession, s.Correlation.SessionID},
		{IDLaunch, s.Correlation.LaunchID}, {IDProc, s.Correlation.ProcID},
		{IDOmniReq, s.Correlation.OmniRequestID}, {IDResult, s.Correlation.ResultID},
		{IDDelivery, s.Correlation.DeliveryID},
	}
	for _, id := range ids {
		if id.val == "" {
			continue
		}
		if !safeID(id.val, maxIDLen) {
			f = append(f, LeakFinding{loc("correlation." + string(id.field)), "identifier outside safe charset"})
		}
	}

	// Outcome / reason codes.
	if s.Outcome != "" && !safeCode(s.Outcome, maxCodeLen) {
		f = append(f, LeakFinding{loc("outcome"), "outcome is not a bounded safe code"})
	}
	if s.ReasonCode != "" && !safeCode(s.ReasonCode, maxCodeLen) {
		f = append(f, LeakFinding{loc("reason_code"), "reason_code is not a bounded safe code"})
	}

	// Labels: closed key set + key-aware value structural check.
	for k, v := range s.Labels {
		kind, ok := allowedLabelKeys[k]
		if !ok {
			f = append(f, LeakFinding{loc("labels." + k), "label key not in approved metadata set"})
			continue
		}
		if leak, reason := detectInlineSecret(v, kind); leak {
			f = append(f, LeakFinding{loc("labels." + k), reason})
		}
	}

	// Counters: numeric only, safe keys.
	for k, v := range s.Counters {
		if !safeCode(k, maxCodeLen) {
			f = append(f, LeakFinding{loc("counters." + k), "counter key is not a safe code"})
		}
		if v < 0 {
			f = append(f, LeakFinding{loc("counters." + k), "counter is negative"})
		}
	}

	// Argv shape: closed vocabulary only.
	for i, tok := range s.ArgvShape {
		if _, ok := allowedArgvShapeTokens[tok]; !ok {
			f = append(f, LeakFinding{loc(fmt.Sprintf("argv_shape[%d]", i)), "argv token is not a redacted shape"})
		}
	}
	return f
}

// ScanSpans runs the OBS-10 structural leak scan across all spans and returns an
// auditable report. It fails closed: any single finding makes the whole report
// not-clean. This is the co-owned (W5+W4) leak gate for G4-OBS.
func ScanSpans(spans []Span) ScanReport {
	report := ScanReport{Scanned: len(spans), Clean: true}
	for _, s := range spans {
		if findings := scanSpan(s); len(findings) > 0 {
			report.Clean = false
			report.Findings = append(report.Findings, findings...)
		}
	}
	return report
}

// ScanLogLines applies the marker/structure leak check to free-form log or
// alert-annotation strings captured elsewhere. Log lines may legitimately
// contain spaces and 'key=value' fragments, so the identifier charset is NOT
// enforced; instead each line is checked for secret markers (URLs, emails,
// bearer/JWT/API-key/connection-string shapes) and control characters. Any line
// that could carry a secret fails closed. Reasons never echo the value.
func ScanLogLines(lines []string) ScanReport {
	report := ScanReport{Scanned: len(lines), Clean: true}
	for i, line := range lines {
		if line == "" {
			continue
		}
		if leak, reason := detectSecretMarkers(line, 4096); leak {
			report.Clean = false
			report.Findings = append(report.Findings, LeakFinding{
				Location: fmt.Sprintf("log[%d]", i), Reason: reason,
			})
		}
	}
	return report
}

// ScanFromSink is a convenience wrapper over a MemorySink.
func ScanFromSink(sink *MemorySink) ScanReport {
	if sink == nil {
		return ScanReport{Clean: true}
	}
	return ScanSpans(sink.Spans())
}
