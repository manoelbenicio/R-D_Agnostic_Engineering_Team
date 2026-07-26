# T20 — verifier hardening (anti-fabrication gates) — RAN, green

- agent: **Opus48#C** · lane **tier20-workload** · task **T20-VERIFIER-HARDENING** · pane `w8:p1`
- as-of (UTC): `2026-07-24T00:09Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock: `internal/daemon/observability/e2e/t20_hardened_verify_test.go` (NEW test) + this evidence.
- check-in: `.deploy-control/p0/checkins/Opus48-C__T20-VERIFIER-HARDENING__20260724T000811Z.json`
- go: `/home/ec2-user/goroot/go/bin/go` (go1.26.1); GOCACHE/GOTMPDIR under `/tmp`. **Test + evidence files only; no product/shared source; no inference/deploy.**

## 0. Result

**DONE — hardening gates written AND run green.** The tier-20 verifier now cannot be satisfied by fabricated evidence: synthetic /tmp fixtures, generated `<prefix><task_id>` IDs, cumulative/stale spans, non-run provenance, sentinel proc_ids, duplicated persisted outcomes, and off-cardinality hop sets each fail closed. A well-formed real-run fixture passes all gates; every fabrication vector is rejected by its gate. All in `package e2e` over the FROZEN `agent-brain.e2e.v1` public `Assemble` — no product/shared source added (only a `_test.go`).

## 1. Hardening gates (test-only functions)

| Gate | Rejects | Rule |
|---|---|---|
| `t20AssertManifestPresent` | **non-run provenance** | requires a run manifest: non-empty `RunID`, valid time-window, non-empty 20-task set, non-empty launched-proc set |
| `t20WithinWindow` | **cumulative/stale spans** | every span `StartedAt`/`EndedAt` ∈ `[WindowStart, WindowEnd]`; a span from another/prior run fails |
| `t20ExactTaskSet` | **fabricated/extra tasks** | trace task_id set == manifest's exact 20 (no missing, no extra, no dup) |
| `t20Exact7HopCardinality` | **cumulative/inflated or missing hops** | zero anomalies, zero orphans, and every trace has EXACTLY `len(EmittingHops())==7` hops present (`Continuous`, `Missing==0`, `len(Hops)==7`); total emitting spans == `20×7` |
| `t20RealProcIDs` | **sentinel/synthetic proc_id** | each cli-hop `proc_id` parses as int `> 0` AND ∈ manifest `LaunchedProcs` (a real daemon-launched pid); `proc-…`, `0`, non-numeric, or not-launched all fail |
| `t20NoGeneratedIDs` | **generated IDs** | rejects any correlation id of the `SyntheticTraceSpans` shape `<prefix><task_id>` (`req-/qmsg-/sess-/launch-/proc-/omni-/result-/delivery-` + task_id) |
| `t20UniquePersistedOutcomes` | **dup/lost outcomes** | exactly **20 unique** persist-hop `result_id`s (no reuse, none empty) |

`t20Harden` runs all gates and returns the first failure — the composite verifier the run evidence must pass.

## 2. Tests (RAN)

`go test ./internal/daemon/observability/e2e/ -run 'TestT20Hardened' -count=1 -v` → **PASS**:
- `TestT20HardenedVerifierAcceptsRealRun` — a 20-task fixture with opaque IDs, real numeric proc_ids, unique result_ids, exactly 7 hops each, in-window → assembles to 20 continuous traces AND passes every gate.
- `TestT20HardenedVerifierRejectsFabrication` (9 subtests, all PASS — each vector rejected by its gate):
  `synthetic_generated_ids`, `non_run_provenance_missing_manifest`, `stale_cumulative_span_outside_window`, `sentinel_proc_id` (`proc-…` and `0`), `proc_id_not_launched`, `duplicate_persisted_outcome`, `extra_task_not_in_set`, `cardinality_inflation_duplicate_hop`, `missing_hop_under_cardinality`.

Notably the `SyntheticTraceSpans` helper (used by OBS-9 synthetic acceptance) is explicitly rejected — its `req-…/proc-…` derived IDs and non-numeric `proc-<task>` fail `t20NoGeneratedIDs` + `t20RealProcIDs` — so a synthetic fixture cannot masquerade as a real tier-20 run. This also encodes that the **deterministic barrier** run (sentinel proc_id, no launched process) is NOT real acceptance: it can prove concurrency/trace shape but fails `t20RealProcIDs`, which real acceptance requires.

## 3. Evidence — exact commands + exit codes

Run from `multica-auth-work/server`, `GOROOT=/home/ec2-user/goroot/go`, `GOCACHE=/tmp/l5-gocache`, `GOTMPDIR=/tmp`:
```text
gofmt -l internal/daemon/observability/e2e/t20_hardened_verify_test.go   -> (empty; clean) ; exit 0
go vet ./internal/daemon/observability/e2e/                              -> clean ; exit 0
go test ./internal/daemon/observability/e2e/ -run 'TestT20Hardened' -count=1 -v
    -> PASS (accept-real + 9 reject-fabrication subtests) ; ok 0.005s ; exit 0
go test ./internal/daemon/observability/e2e/ -count=1  (full package)    -> ok 0.012s ; exit 0 (no regression)
git diff --check -- .../t20_hardened_verify_test.go                      -> PASS ; exit 0
```

## 4. Scope / non-claims
- **Test + evidence files only**: added one `_test.go` (`package e2e`) + this evidence; **no product/shared source edited** (other new `e2e/*.go` in `git status` — `export_sink.go`/`collector.go`/`bounded_sink.go`/etc. — are OTHER lanes' work, NOT mine; my only addition is the test file).
- Fixtures are built **in-memory**, never read from `/tmp` — the hardening posture (a real run supplies a manifest + real launched procs; a hand-crafted file cannot).
- Reuses the frozen `agent-brain.e2e.v1` public `Assemble`/`EmittingHops`/`Correlation.Get`; introduces no new product contract; the manifest + gates are verifier-local (test) types.
- `-race` not run (cgo/gcc absent); gates are deterministic.
- Verifies anti-fabrication of the trace/concurrency/persistence-accounting evidence only; does NOT prove LLM/OmniRoute protocol behavior and asserts no model-route acceptance; no inference/deploy; does not authorize tier-9.2/cutover.

## 5. Status
- STATUS: DONE. DELIVERED: 7 anti-fabrication gates (§1) + composite `t20Harden`, and RAN tests proving accept-real + reject each of 9 fabrication vectors (§2/§3). FILE: created the `_test.go` + this evidence; no product/shared source.
