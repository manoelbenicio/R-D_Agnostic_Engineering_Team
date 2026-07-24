package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// t20_hardened_verify_test.go — anti-fabrication hardening for the tier-20
// AssembleFromLogs verifier. These gates make synthetic /tmp fixtures, generated
// <prefix>-<task_id> IDs, cumulative/stale spans, non-run provenance, sentinel
// proc_ids, duplicated persisted outcomes, and off-cardinality hop sets fail
// closed. Test-only: no product/shared source. Fixtures are built in-memory
// (never read from /tmp), which is itself part of the hardening posture.

// t20RunManifest binds a verification to exactly one real run: a run_id, a
// time-window, the exact 20-task set, and the set of real OS pids the daemon
// recorded launching. A verification with no manifest is non-run provenance and
// is rejected.
type t20RunManifest struct {
	RunID         string
	WindowStart   time.Time
	WindowEnd     time.Time
	ExpectedTasks map[string]bool
	LaunchedProcs map[string]bool // proc_id -> launched (numeric pid strings)
}

func t20AssertManifestPresent(m t20RunManifest) (bool, string) {
	if strings.TrimSpace(m.RunID) == "" || m.WindowEnd.Before(m.WindowStart) ||
		len(m.ExpectedTasks) == 0 || len(m.LaunchedProcs) == 0 {
		return false, "missing/empty run manifest (non-run provenance)"
	}
	return true, ""
}

func t20WithinWindow(spans []Span, m t20RunManifest) (bool, string) {
	for _, s := range spans {
		if s.StartedAt.Before(m.WindowStart) || s.StartedAt.After(m.WindowEnd) {
			return false, "span started outside the run time-window (stale/cumulative)"
		}
		if !s.EndedAt.IsZero() && s.EndedAt.After(m.WindowEnd) {
			return false, "span ended outside the run time-window (stale/cumulative)"
		}
	}
	return true, ""
}

func t20ExactTaskSet(report AssemblyReport, m t20RunManifest) (bool, string) {
	seen := map[string]bool{}
	for _, tr := range report.Traces {
		if !m.ExpectedTasks[tr.TaskID] {
			return false, "unexpected task_id not in the run set"
		}
		if seen[tr.TaskID] {
			return false, "duplicate task_id"
		}
		seen[tr.TaskID] = true
	}
	if len(seen) != len(m.ExpectedTasks) {
		return false, "task set mismatch (missing or extra tasks)"
	}
	return true, ""
}

func t20Exact7HopCardinality(report AssemblyReport) (bool, string) {
	if len(report.Anomalies) != 0 {
		return false, "duplicate/conflicting hop spans (cardinality inflation)"
	}
	if len(report.Orphans) != 0 {
		return false, "orphan spans (cumulative/unjoined)"
	}
	emitting := len(EmittingHops()) // 7
	total := 0
	for _, tr := range report.Traces {
		if !tr.Continuous || len(tr.Present) != emitting || len(tr.Missing) != 0 || len(tr.Hops) != emitting {
			return false, fmt.Sprintf("task %s not exactly %d emitting hops (present=%d hops=%d)", tr.TaskID, emitting, len(tr.Present), len(tr.Hops))
		}
		total += len(tr.Hops)
	}
	if total != len(report.Traces)*emitting {
		return false, "extra/cumulative hop spans across traces"
	}
	return true, ""
}

func t20RealProcIDs(report AssemblyReport, m t20RunManifest) (bool, string) {
	for _, tr := range report.Traces {
		cli, ok := tr.Hops[HopCLI]
		if !ok {
			return false, "missing cli hop"
		}
		pid := cli.Correlation.ProcID
		n, err := strconv.Atoi(pid)
		if err != nil || n <= 0 {
			return false, "proc_id is not a real pid > 0 (sentinel/synthetic)"
		}
		if !m.LaunchedProcs[pid] {
			return false, "proc_id not tied to a daemon-launched process"
		}
	}
	return true, ""
}

// t20NoGeneratedIDs rejects the SyntheticTraceSpans-style derived IDs of the
// form "<prefix><task_id>", which a hand-built /tmp fixture would carry.
func t20NoGeneratedIDs(report AssemblyReport) (bool, string) {
	prefixes := []string{"req-", "qmsg-", "sess-", "launch-", "proc-", "omni-", "result-", "delivery-"}
	for _, tr := range report.Traces {
		for _, s := range tr.Hops {
			ids := []string{
				s.Correlation.RequestID, s.Correlation.QueueMsgID, s.Correlation.SessionID,
				s.Correlation.LaunchID, s.Correlation.ProcID, s.Correlation.OmniRequestID,
				s.Correlation.ResultID, s.Correlation.DeliveryID,
			}
			for _, id := range ids {
				if id == "" {
					continue
				}
				for _, p := range prefixes {
					if strings.HasPrefix(id, p) && strings.HasSuffix(id, tr.TaskID) {
						return false, "correlation id is a generated <prefix><task_id> synthetic value"
					}
				}
			}
		}
	}
	return true, ""
}

func t20UniquePersistedOutcomes(report AssemblyReport) (bool, string) {
	results := map[string]bool{}
	for _, tr := range report.Traces {
		persist, ok := tr.Hops[HopPersist]
		if !ok {
			return false, "missing persist hop"
		}
		rid := persist.Correlation.ResultID
		if rid == "" {
			return false, "empty persisted result_id"
		}
		if results[rid] {
			return false, "duplicate persisted result_id (dup/reused outcome)"
		}
		results[rid] = true
	}
	if len(results) != 20 {
		return false, "not exactly 20 unique persisted outcomes"
	}
	return true, ""
}

// t20Harden runs every gate and returns the first failure reason.
func t20Harden(spans []Span, report AssemblyReport, m t20RunManifest) (bool, string) {
	for _, gate := range []func() (bool, string){
		func() (bool, string) { return t20AssertManifestPresent(m) },
		func() (bool, string) { return t20WithinWindow(spans, m) },
		func() (bool, string) { return t20ExactTaskSet(report, m) },
		func() (bool, string) { return t20Exact7HopCardinality(report) },
		func() (bool, string) { return t20RealProcIDs(report, m) },
		func() (bool, string) { return t20NoGeneratedIDs(report) },
		func() (bool, string) { return t20UniquePersistedOutcomes(report) },
	} {
		if ok, reason := gate(); !ok {
			return false, reason
		}
	}
	return true, ""
}

// --- fixtures ---

// opaqueID returns a real-shaped opaque identifier (not derived from the
// task_id and not a synthetic prefix), within the safeID charset. Distinct per
// (kind, i) so no two correlation fields collide.
func opaqueID(kind string, i int) string {
	h := 0
	for _, c := range kind {
		h = h*131 + int(c)
	}
	return fmt.Sprintf("id%04x%08x", h&0xffff, (i*2654435761+h)&0x7fffffff)
}

func realSpan(hop HopKind, corr Correlation, at time.Time) Span {
	return Span{
		ContractVersion: ContractVersion,
		Hop:             hop,
		Correlation:     corr,
		StartedAt:       at,
		EndedAt:         at.Add(time.Second),
		Outcome:         "ok",
		SecretsPresent:  false,
	}
}

// buildTaskSpans builds the 7 emitting-hop spans for one task at index i, with
// opaque IDs, a real numeric proc_id, and a unique result_id. Returns the spans
// plus the task_id and proc_id so the manifest can register them.
func buildTaskSpans(i int, at time.Time) (spans []Span, task, proc string) {
	task = fmt.Sprintf("t%08xk", (i*2246822519+5)&0x7fffffff)
	req := opaqueID("request", i)
	qmsg := opaqueID("queue", i)
	sess := opaqueID("session", i)
	launch := opaqueID("launch", i)
	proc = strconv.Itoa(4000 + i) // real pid string > 0
	omni := opaqueID("omni", i)
	result := opaqueID("result", i)
	delivery := opaqueID("delivery", i)
	spans = []Span{
		realSpan(HopIngress, Correlation{RequestID: req, TaskID: task}, at),
		realSpan(HopQueue, Correlation{QueueMsgID: qmsg, TaskID: task}, at),
		realSpan(HopAdmission, Correlation{TaskID: task, SessionID: sess, LaunchID: launch}, at),
		realSpan(HopCLI, Correlation{LaunchID: launch, ProcID: proc}, at),
		realSpan(HopRoute, Correlation{RequestID: req, OmniRequestID: omni}, at),
		realSpan(HopPersist, Correlation{TaskID: task, ResultID: result}, at),
		realSpan(HopDelivery, Correlation{SessionID: sess, DeliveryID: delivery}, at),
	}
	return spans, task, proc
}

// buildRealRun produces n well-formed tasks (7 emitting hops each) with opaque
// IDs, real numeric proc_ids, and unique result_ids, all inside the window.
func buildRealRun(n int) ([]Span, t20RunManifest) {
	start := time.Unix(1_700_000_000, 0).UTC()
	m := t20RunManifest{
		RunID:         "run-2026-07-24T00Z-abc123",
		WindowStart:   start,
		WindowEnd:     start.Add(time.Hour),
		ExpectedTasks: map[string]bool{},
		LaunchedProcs: map[string]bool{},
	}
	at := start.Add(time.Minute)
	var spans []Span
	for i := 0; i < n; i++ {
		taskSpans, task, proc := buildTaskSpans(i, at)
		m.ExpectedTasks[task] = true
		m.LaunchedProcs[proc] = true
		spans = append(spans, taskSpans...)
	}
	return spans, m
}

// TestT20HardenedVerifierAcceptsRealRun proves a well-formed real-run fixture
// passes assembly AND every hardening gate.
func TestT20HardenedVerifierAcceptsRealRun(t *testing.T) {
	spans, manifest := buildRealRun(20)
	report := Assemble(spans)
	if len(report.Traces) != 20 || !report.AllContinuous {
		t.Fatalf("assembly: traces=%d allContinuous=%v", len(report.Traces), report.AllContinuous)
	}
	if ok, reason := t20Harden(spans, report, manifest); !ok {
		t.Fatalf("hardening rejected a valid real run: %s", reason)
	}
}

// TestT20HardenedVerifierRejectsFabrication proves each fabrication vector is
// caught by the corresponding gate.
func TestT20HardenedVerifierRejectsFabrication(t *testing.T) {
	t.Run("synthetic_generated_ids", func(t *testing.T) {
		// SyntheticTraceSpans uses "req-"+task etc. and "proc-"+task.
		var spans []Span
		for i := 0; i < 20; i++ {
			spans = append(spans, SyntheticTraceSpans(fmt.Sprintf("task-%d", i))...)
		}
		report := Assemble(spans)
		// non-run provenance too (no manifest) — assert it cannot pass.
		if ok, _ := t20Harden(spans, report, t20RunManifest{}); ok {
			t.Fatal("synthetic fixture with generated IDs and no manifest passed hardening")
		}
		// With a manifest it still fails on generated IDs and sentinel proc_id.
		if ok, reason := t20NoGeneratedIDs(report); ok {
			t.Fatalf("generated <prefix><task> IDs not rejected: reason=%q", reason)
		}
	})

	t.Run("non_run_provenance_missing_manifest", func(t *testing.T) {
		spans, _ := buildRealRun(20)
		report := Assemble(spans)
		if ok, _ := t20AssertManifestPresent(t20RunManifest{}); ok {
			t.Fatal("empty manifest accepted (non-run provenance)")
		}
		if ok, _ := t20Harden(spans, report, t20RunManifest{}); ok {
			t.Fatal("run without manifest passed hardening")
		}
	})

	t.Run("stale_cumulative_span_outside_window", func(t *testing.T) {
		spans, manifest := buildRealRun(20)
		spans[0].StartedAt = manifest.WindowStart.Add(-time.Minute) // stale
		if ok, _ := t20WithinWindow(spans, manifest); ok {
			t.Fatal("stale span outside window not rejected")
		}
	})

	t.Run("sentinel_proc_id", func(t *testing.T) {
		spans, manifest := buildRealRun(20)
		// Corrupt one cli proc_id to a non-numeric sentinel.
		for i := range spans {
			if spans[i].Hop == HopCLI {
				spans[i].Correlation.ProcID = "proc-sentinel"
				break
			}
		}
		if ok, _ := t20RealProcIDs(Assemble(spans), manifest); ok {
			t.Fatal("sentinel non-numeric proc_id not rejected")
		}
		// A zero pid is also rejected.
		for i := range spans {
			if spans[i].Hop == HopCLI {
				spans[i].Correlation.ProcID = "0"
				break
			}
		}
		if ok, _ := t20RealProcIDs(Assemble(spans), manifest); ok {
			t.Fatal("proc_id 0 not rejected")
		}
	})

	t.Run("proc_id_not_launched", func(t *testing.T) {
		spans, manifest := buildRealRun(20)
		delete(manifest.LaunchedProcs, "4000") // pid present in span but not launched
		if ok, _ := t20RealProcIDs(Assemble(spans), manifest); ok {
			t.Fatal("proc_id not tied to a launched process was accepted")
		}
	})

	t.Run("duplicate_persisted_outcome", func(t *testing.T) {
		spans, _ := buildRealRun(20)
		// Force two tasks to share a result_id (dup/lost outcome).
		var firstResult string
		seen := 0
		for i := range spans {
			if spans[i].Hop == HopPersist {
				if seen == 0 {
					firstResult = spans[i].Correlation.ResultID
				} else if seen == 1 {
					spans[i].Correlation.ResultID = firstResult
					break
				}
				seen++
			}
		}
		if ok, _ := t20UniquePersistedOutcomes(Assemble(spans)); ok {
			t.Fatal("duplicate persisted result_id not rejected")
		}
	})

	t.Run("extra_task_not_in_set", func(t *testing.T) {
		spans, manifest := buildRealRun(20)
		start := manifest.WindowStart.Add(time.Minute)
		extra, _, _ := buildTaskSpans(999, start) // a 21st DISTINCT task not in the manifest set
		report := Assemble(append(spans, extra...))
		if ok, _ := t20ExactTaskSet(report, manifest); ok {
			t.Fatal("extra task outside the run set was accepted")
		}
	})

	t.Run("cardinality_inflation_duplicate_hop", func(t *testing.T) {
		spans, _ := buildRealRun(20)
		// Duplicate one task's ingress hop -> assembler flags a duplicate anomaly.
		for i := range spans {
			if spans[i].Hop == HopIngress {
				dup := spans[i]
				spans = append(spans, dup)
				break
			}
		}
		if ok, reason := t20Exact7HopCardinality(Assemble(spans)); ok {
			t.Fatalf("duplicate hop span (cardinality inflation) not rejected: %q", reason)
		}
	})

	t.Run("missing_hop_under_cardinality", func(t *testing.T) {
		spans, _ := buildRealRun(20)
		// Drop one task's delivery hop -> that trace is not exactly 7 hops.
		out := spans[:0]
		dropped := false
		for _, s := range spans {
			if !dropped && s.Hop == HopDelivery {
				dropped = true
				continue
			}
			out = append(out, s)
		}
		if ok, _ := t20Exact7HopCardinality(Assemble(out)); ok {
			t.Fatal("trace missing a hop (under-cardinality) not rejected")
		}
	})
}
