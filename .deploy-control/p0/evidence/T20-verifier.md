# T20 — instrumented harness update + 20×8-hop/9-ID trace verifier (DESIGN; not run)

- agent: **Opus48#C** · lane **tier20-workload** · task **T20-VERIFIER** · pane `w8:p1`
- as-of (UTC): `2026-07-23T23:49Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock (sole mutable): `.deploy-control/p0/evidence/T20-verifier.md`
- check-in: `.deploy-control/p0/checkins/Opus48-C__T20-VERIFIER__20260723T234904Z.json`
- companions: `T20-workload.md` (submission), `T20-barrier-harness.md` (deterministic hold).
- **No product source written; nothing executed** (`live_runs.*=false`; no export files exist yet). The verifier below is delivered verbatim — it must live in the server module (imports the frozen `e2e` package), so a future authorized writer drops it in; it is inert (skips) until export files are present.

## 1. Instrumented tier-20 harness — update

The 20-task deterministic-hold run (`T20-workload.md` §3 submission + `T20-barrier-harness.md` §2 dev-gated hold) is **instrumented**: every hop owner emits its redacted `agent-brain.e2e.v1` span (metadata-only, `secrets_present=false`) and the run **exports** the spans to two JSON files (arrays of `e2e.Span`):

- **backend export** `T20_BACKEND_SPANS` (service side): hops **ingress**(1), **queue**(2), **persist**(6), **delivery**(7).
- **daemon export** `T20_DAEMON_SPANS` (daemon side): hops **admission**(3), **cli**(4)*, **route**(5).
  - *under the deterministic barrier, the "cli" hop is the control-plane hold (no LLM): it still emits a hop-4 span (`launch_id`/`proc_id`, `exit_code_class`), so the 7 emitting hops are complete without inference.

Merged, each of the 20 tasks yields 7 emitting-hop spans; the verifier's `Assemble` synthesizes the 8th hop (`HopTrace`) → **8 hops end-to-end per task**. Export is append-only JSON; no bodies/prompts/results/secrets (per `EVIDENCE_CONTRACT` + the frozen contract's closed label/counter/argv vocabularies).

## 2. Verifier — `AssembleFromLogs` (full source; place as `internal/daemon/observability/e2e/assemble_from_logs_verify_test.go`, package `e2e`)

```go
package e2e

import (
	"encoding/json"
	"os"
	"testing"
)

// AssembleFromLogs loads redacted e2e.Span arrays from the given export files
// (backend + daemon), merges them, and assembles per-task end-to-end traces
// using the frozen agent-brain.e2e.v1 assembler. It is verification-only: it
// adds no product behavior and reuses the public Assemble contract.
func AssembleFromLogs(paths ...string) (AssemblyReport, []Span, error) {
	var all []Span
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			return AssemblyReport{}, nil, err
		}
		var spans []Span
		if err := json.Unmarshal(raw, &spans); err != nil {
			return AssemblyReport{}, nil, err
		}
		all = append(all, spans...)
	}
	return Assemble(all), all, nil
}

// allCorrelationIDs is the frozen 9-ID set every complete trace must carry.
var allCorrelationIDs = []IDField{
	IDRequest, IDQueueMsg, IDTask, IDSession, IDLaunch, IDProc, IDOmniReq, IDResult, IDDelivery,
}

// TestT20AssembleFromLogs asserts the tier-20 acceptance shape:
// 20 traces, each with all 7 emitting hops present (+ the assembled HopTrace = 8
// hops end-to-end) and all 9 correlation IDs, AllContinuous, dropped=0, and no
// orphans/anomalies. It is inert (skips) until the export files exist, so it can
// live in-tree without gating CI before a run.
func TestT20AssembleFromLogs(t *testing.T) {
	backend := os.Getenv("T20_BACKEND_SPANS")
	daemon := os.Getenv("T20_DAEMON_SPANS")
	if backend == "" || daemon == "" {
		t.Skip("T20_BACKEND_SPANS / T20_DAEMON_SPANS unset; run the instrumented tier-20 harness first")
	}

	report, allSpans, err := AssembleFromLogs(backend, daemon)
	if err != nil {
		t.Fatalf("AssembleFromLogs: %v", err)
	}

	// 20 traces.
	if len(report.Traces) != 20 {
		t.Fatalf("traces = %d, want 20", len(report.Traces))
	}
	// AllContinuous + no orphans + no anomalies (dropped/lost spans == 0).
	if !report.AllContinuous {
		t.Fatalf("AllContinuous = false; missing/gapped traces present")
	}
	if len(report.Orphans) != 0 {
		t.Fatalf("orphans = %d, want 0: %+v", len(report.Orphans), report.Orphans)
	}
	if len(report.Anomalies) != 0 {
		t.Fatalf("anomalies = %d, want 0: %+v", len(report.Anomalies), report.Anomalies)
	}

	emitting := EmittingHops() // 7
	for _, tr := range report.Traces {
		// 8 hops end-to-end = 7 emitting hops present + the assembled HopTrace.
		if !tr.Continuous || len(tr.Present) != len(emitting) {
			t.Fatalf("task %s: present=%v missing=%v (want all %d emitting hops)", tr.TaskID, tr.Present, tr.Missing, len(emitting))
		}
		// 9 IDs: union across the trace's spans must cover every correlation field.
		seen := map[IDField]bool{}
		var dropped int64
		for hop, span := range tr.Hops {
			for _, id := range allCorrelationIDs {
				if span.Correlation.Get(id) != "" {
					seen[id] = true
				}
			}
			if hop == HopDelivery {
				dropped += span.Counters["drop_count"]
			}
		}
		for _, id := range allCorrelationIDs {
			if !seen[id] {
				t.Fatalf("task %s: missing correlation id %s (want all 9)", tr.TaskID, id)
			}
		}
		// dropped == 0 (no delivered-payload drops on the delivery hop).
		if dropped != 0 {
			t.Fatalf("task %s: delivery drop_count = %d, want 0", tr.TaskID, dropped)
		}
	}

	// Metadata-only leak gate over every exported span (defense in depth).
	if scan := ScanSpans(allSpans); !scan.Clean {
		t.Fatalf("leak scan not clean: %+v", scan.Findings)
	}
}
```

### Why these assertions map to the ask
- **20 traces** → `len(report.Traces)==20`.
- **8 hops each** → `tr.Continuous && len(tr.Present)==len(EmittingHops())` (7 emitting present) + `Assemble` synthesizes `HopTrace` (the 8th, end-to-end join hop) → 8.
- **9 IDs each** → union of `Correlation.Get(id)` over the trace's spans covers all of `{request_id, queue_msg_id, task_id, session_id, launch_id, proc_id, omni_request_id, result_id, delivery_id}`.
- **AllContinuous** → `report.AllContinuous`.
- **dropped=0** → delivery-hop `drop_count` counter sums to 0 per trace (no delivered-payload drops) AND `report.Orphans==0` (no spans dropped from any trace during join).
- **no orphans** → `len(report.Orphans)==0` (and `Anomalies==0` guards duplicate/conflicting joins).
- Bonus: `ScanSpans(...).Clean` enforces metadata-only over the raw export.

## 3. Run (only when authorized; not now)
```text
# after an authorized instrumented tier-20 run has written the two export files:
cd multica-auth-work/server
T20_BACKEND_SPANS=/path/backend-spans.json T20_DAEMON_SPANS=/path/daemon-spans.json \
  /home/ec2-user/goroot/go/bin/go test ./internal/daemon/observability/e2e/ -run TestT20AssembleFromLogs -count=1 -v
# exit 0 + PASS => 20x8-hop/9-ID/AllContinuous/dropped=0/no-orphans verified; non-zero => the exact failing task/field.
```
Inert today: the test `t.Skip`s when the env files are unset, so dropping it in-tree does not fail CI before a run.

## 4. Scope / non-claims
- **No product source written**: the verifier is provided verbatim here; it is NOT added to the module tree by this lane (it belongs under the module because it imports the internal `e2e` package). No product `.go` edited; only this evidence file created.
- **Not run**: no instrumented harness executed, no export files produced, no inference, no live call (`live_runs.*=false`).
- Reuses the FROZEN `agent-brain.e2e.v1` public `Assemble`/`ScanSpans` — introduces no new product contract, no `AssembleFromLogs` in the product package (it is a verifier-local wrapper in the test file).
- Verifies concurrency/trace/persistence-accounting shape only; does NOT prove LLM/OmniRoute protocol behavior (separate gated live-run family), and asserts no acceptance of a model route.
- Does not authorize tier activation (9.2), higher tiers, cutover, or production.

## 5. Status
- STATUS: DONE (harness-instrumentation update + verifier delivered).
- DELIVERED: §1 instrumented export design (backend+daemon span files), §2 full `AssembleFromLogs` verifier (20×8-hop/9-ID/AllContinuous/dropped=0/no-orphans + leak gate), §3 run recipe. Not implemented in-tree, not run.
