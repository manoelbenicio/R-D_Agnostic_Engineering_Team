package e2e

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

// Collector reconstructs end-to-end spans from the DURABLE cross-process export
// path — the JSONL span files written by JSONLSink in each process (backend +
// daemon) — and feeds them to Assemble. The log is the drop-free source of
// truth: unlike the in-memory BoundedSink ring (operational only, subject to
// eviction), JSONLSink writes every validated span exactly once as one JSON
// object per line, so a complete trace can be reassembled across processes.
//
// Each line is parsed by ParseSpanLine, which decodes the full closed Span
// schema, checks the contract version, and re-runs Span.Validate (schema +
// structural leak scan) fail-closed — so a line that is not metadata-only,
// declares secrets_present, or is malformed is dropped and counted, never handed
// to the assembler. Point the collector at the DEDICATED JSONLSink export files
// (one per process), not the general application log.

// maxSpanLogLineBytes bounds a single scanned line so a corrupt/huge line can
// neither exhaust memory nor stall the collector.
const maxSpanLogLineBytes = 4 << 20 // 4 MiB

// CollectStats is a metadata-only accounting of a collection pass. It carries no
// span contents or identifier values — only counts and bounded drop reasons.
type CollectStats struct {
	Files         []string       `json:"files"`
	LinesTotal    int            `json:"lines_total"`
	SpanLines     int            `json:"span_lines"` // non-empty JSON-object lines considered
	Reconstructed int            `json:"reconstructed"`
	Dropped       int            `json:"dropped"`
	DropReasons   map[string]int `json:"drop_reasons"`
}

// CollectorReport pairs the collection stats with the assembled OBS-9 report.
type CollectorReport struct {
	Stats    CollectStats   `json:"stats"`
	Assembly AssemblyReport `json:"assembly"`
}

// Dropped reports how many span-candidate lines failed reconstruction/validation.
// Acceptance for a durable-source merge is Dropped()==0: JSONLSink records every
// validated span, so a healthy collection over the dedicated export files loses none.
func (r CollectorReport) Dropped() int { return r.Stats.Dropped }

// classifyDrop maps a ParseSpanLine error to a bounded, value-free reason code so
// DropReasons stays low-cardinality and never echoes identifier/secret values.
func classifyDrop(err error) string {
	m := err.Error()
	switch {
	case strings.Contains(m, "secrets_present"):
		return "secrets_present"
	case strings.Contains(m, "unsupported contract version"):
		return "unsupported_contract"
	case strings.Contains(m, "missing required correlation"):
		return "missing_required_id"
	case strings.Contains(m, "not an emitting hop"):
		return "bad_hop"
	case strings.Contains(m, "not a safe identifier"):
		return "unsafe_identifier"
	case strings.Contains(m, "decode"):
		return "malformed_json"
	default:
		return "invalid_span"
	}
}

// CollectSpans reads the given JSONL span files, reconstructs and validates each
// span line via ParseSpanLine, and returns the valid spans plus a metadata-only
// accounting. Empty lines and non-JSON-object lines are ignored (not counted as
// drops). A span-candidate line ('{'-prefixed) that fails to parse/validate
// increments Dropped with a bounded reason. A file open/scan error is returned.
func CollectSpans(paths ...string) ([]Span, CollectStats, error) {
	stats := CollectStats{DropReasons: map[string]int{}}
	var spans []Span
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, stats, fmt.Errorf("open span log: %w", err)
		}
		stats.Files = append(stats.Files, path)
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), maxSpanLogLineBytes)
		for scanner.Scan() {
			stats.LinesTotal++
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 || line[0] != '{' {
				continue // blank line or non-JSON banner — not a span candidate
			}
			stats.SpanLines++
			span, err := ParseSpanLine(line)
			if err != nil {
				stats.Dropped++
				stats.DropReasons[classifyDrop(err)]++
				continue
			}
			spans = append(spans, span)
			stats.Reconstructed++
		}
		closeErr := f.Close()
		if err := scanner.Err(); err != nil {
			return nil, stats, fmt.Errorf("scan span log: %w", err)
		}
		if closeErr != nil {
			return nil, stats, fmt.Errorf("close span log: %w", closeErr)
		}
	}
	return spans, stats, nil
}

// AssembleFromLogs is the end-to-end entry point: read the backend and daemon
// JSONL span export files, reconstruct spans, and assemble them into per-task
// traces. Pass every process's export path (order does not matter — assembly
// joins on identifiers). Acceptance for a complete run: report.Dropped()==0 and
// report.Assembly.AllContinuous.
func AssembleFromLogs(paths ...string) (CollectorReport, error) {
	spans, stats, err := CollectSpans(paths...)
	if err != nil {
		return CollectorReport{}, err
	}
	return CollectorReport{Stats: stats, Assembly: Assemble(spans)}, nil
}
