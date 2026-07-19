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
	loc := func(field string) string { return "span." + field }

	if err := s.Validate(); err != nil {
		f = append(f, LeakFinding{loc("structure"), "span failed closed structural validation"})
	}
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
		if err := validateLabel(k, v); err != nil {
			f = append(f, LeakFinding{loc("labels"), "label failed closed structural validation"})
		}
	}

	// Counters: closed per-hop key set and non-negative numeric values.
	for k, v := range s.Counters {
		if !counterAllowed(s.Hop, k) {
			f = append(f, LeakFinding{loc("counters"), "counter key not approved for hop"})
		}
		if v < 0 {
			f = append(f, LeakFinding{loc("counters"), "counter is negative"})
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

// EventKind is the closed vocabulary for metadata-only structured events.
type EventKind string

const (
	EventHop         EventKind = "hop"
	EventTraceGap    EventKind = "trace_gap"
	EventTraceOrphan EventKind = "trace_orphan"
	EventLeakRefused EventKind = "leak_refused"
)

// Event is the only accepted log/event shape. It deliberately has no message,
// body, error text, or other free-form content field. Labels remain span/event
// metadata and MUST NOT be promoted into Prometheus dimensions.
type Event struct {
	ContractVersion string            `json:"contract_version"`
	Kind            EventKind         `json:"kind"`
	Hop             HopKind           `json:"hop"`
	Correlation     Correlation       `json:"correlation"`
	Outcome         string            `json:"outcome"`
	ReasonCode      string            `json:"reason_code,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Counters        map[string]int64  `json:"counters,omitempty"`
	SecretsPresent  bool              `json:"secrets_present"`
}

func (e Event) validate() error {
	if !SupportedContractVersion(e.ContractVersion) {
		return fmt.Errorf("unsupported event contract version")
	}
	switch e.Kind {
	case EventHop, EventTraceGap, EventTraceOrphan, EventLeakRefused:
	default:
		return fmt.Errorf("event kind is not approved")
	}
	if !isEmittingHop(e.Hop) && e.Hop != HopTrace {
		return fmt.Errorf("event hop is not approved")
	}
	if e.SecretsPresent {
		return fmt.Errorf("event secrets_present invariant violated")
	}
	if err := e.Correlation.Validate(); err != nil {
		return err
	}
	if isEmittingHop(e.Hop) {
		for _, required := range RequiredIDs(e.Hop) {
			if e.Correlation.Get(required) == "" {
				return fmt.Errorf("event missing required correlation")
			}
		}
	}
	if !safeCode(e.Outcome, maxCodeLen) {
		return fmt.Errorf("event outcome is not a safe code")
	}
	if e.ReasonCode != "" && !safeCode(e.ReasonCode, maxCodeLen) {
		return fmt.Errorf("event reason is not a safe code")
	}
	if len(e.Labels) > maxLabels || len(e.Counters) > maxCounters {
		return fmt.Errorf("event metadata budget exceeded")
	}
	for key, value := range e.Labels {
		if err := validateLabel(key, value); err != nil {
			return fmt.Errorf("event label rejected")
		}
	}
	for key, value := range e.Counters {
		if value < 0 {
			return fmt.Errorf("event counter is negative")
		}
		if e.Hop == HopTrace {
			switch key {
			case "gap_count", "orphan_count", "hop_count", "trace_count":
			default:
				return fmt.Errorf("trace event counter is not approved")
			}
		} else if !counterAllowed(e.Hop, key) {
			return fmt.Errorf("event counter is not approved for hop")
		}
	}
	return nil
}

// ScanEvents structurally validates closed metadata-only events. No
// pattern-based content acceptance is involved because Event has no free-form
// content field.
func ScanEvents(events []Event) ScanReport {
	report := ScanReport{Scanned: len(events), Clean: true}
	for index, event := range events {
		if err := event.validate(); err != nil {
			report.Clean = false
			report.Findings = append(report.Findings, LeakFinding{
				Location: fmt.Sprintf("event[%d]", index),
				Reason:   "event failed closed structural validation",
			})
		}
	}
	return report
}

// ScanLogLines rejects every non-empty free-form log line. Pattern scans cannot
// prove that arbitrary text is metadata-only; producers must emit Event values
// and use ScanEvents instead.
func ScanLogLines(lines []string) ScanReport {
	report := ScanReport{Scanned: len(lines), Clean: true}
	for i, line := range lines {
		if line == "" {
			continue
		}
		report.Clean = false
		report.Findings = append(report.Findings, LeakFinding{
			Location: fmt.Sprintf("log[%d]", i),
			Reason:   "free-form log line is not structurally metadata-only",
		})
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
