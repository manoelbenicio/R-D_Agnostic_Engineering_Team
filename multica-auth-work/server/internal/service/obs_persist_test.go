package service

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func TestNewPersistSpanIsMetadataOnlyAndValid(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	sp := NewPersistSpan(PersistObservation{
		TaskID: "task-abc", ResultID: "result-1",
		TerminalStatus: "completed", PersistLatencyMs: 7, ByteCount: 2048, TokenCount: 128,
		Outcome: "persisted",
	})
	if err := sp.Validate(); err != nil {
		t.Fatalf("persist span invalid: %v", err)
	}
	if err := rec.Emit(sp); err != nil {
		t.Fatalf("emit persist: %v", err)
	}
	if sink.Len() != 1 {
		t.Fatalf("sink len = %d, want 1", sink.Len())
	}
	got := sink.Spans()[0]
	if got.Hop != e2e.HopPersist || got.SecretsPresent {
		t.Fatalf("unexpected span shape: %+v", got)
	}
	if got.Correlation.TaskID != "task-abc" || got.Correlation.ResultID != "result-1" {
		t.Fatalf("correlation not carried: %+v", got.Correlation)
	}
	for k := range got.Labels {
		if k != "terminal_status" {
			t.Fatalf("unexpected label key %q (helper must stay metadata-only)", k)
		}
	}
	for k := range got.Counters {
		switch k {
		case "persist_latency_ms", "byte_count", "token_count":
		default:
			t.Fatalf("unexpected counter key %q", k)
		}
	}
	if report := e2e.ScanSpans([]e2e.Span{got}); !report.Clean {
		t.Fatalf("leak scan not clean: %+v", report.Findings)
	}
}

func TestNewPersistSpanFailsClosedOnMissingResultID(t *testing.T) {
	sink := e2e.NewMemorySink()
	rec := e2e.NewRecorder(sink)
	sp := NewPersistSpan(PersistObservation{TaskID: "task-abc", TerminalStatus: "completed", PersistLatencyMs: 1, Outcome: "persisted"})
	if err := rec.Emit(sp); err == nil {
		t.Fatal("persist span missing required result_id was emitted; want fail-closed refusal")
	}
	if sink.Len() != 0 {
		t.Fatalf("sink len = %d, want 0 after fail-closed refusal", sink.Len())
	}
}
