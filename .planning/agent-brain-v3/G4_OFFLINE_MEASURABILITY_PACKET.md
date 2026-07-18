# G4_OFFLINE_MEASURABILITY_PACKET — Agent Brain v3

> **PREPARATION ONLY.** No threshold approval, no task-9.1 run, no OpenSpec checkbox closure,
> no code/doc edits from this file, no credentials/network/live providers. Synthetic/offline only,
> Linux disposable container. Revised to resolve the independent (pB) review defects. PD-01/PD-08
> absolute; 46/85 unchanged. **I1/I2 are now implemented (see below) but are NOT independently
> accepted until pB completes its review.**

## Scope — the 8 GAP metrics

| # | Metric | Gap type |
|---|---|---|
| 1 | Completion rate | source split (ledger→evidence bridge) — I1 implemented |
| 2 | Error rate (non-injected) | injected-vs-non-injected tag — I2 implemented |
| 5 | End-to-end latency (per task) | task-level source + nearest-rank p50/p95/p99 aggregator |
| 6 | Retry rate | gateway telemetry (I3) |
| 7 | Fallback rate | gateway telemetry (I3) + fallback-cycle validation |
| 8 | Queue peak / wait / growth | external queue must emit depth (I4) |
| 10 | CPU % | owner-declared budget denominator (guarded) |
| 14 | Recovery to steady state | bounded steady-state detection (I5) |

READY (already measurable): #3 selection, #4 TTFT, #9 fairness (capacity-time), #11 RSS, #12 sockets, #13 cancellation-release.

## Interface direction — acyclic by construction (corrected)

**Actual imports:** `gateway/**` imports the frozen `brain` contract (`agent-brain.v1`); `observability/**` also imports `brain`. Neither imports the other. To keep it that way, **all cross-stream interface types (I1–I5) are defined in the frozen `brain` contract package (Codex1-owned, content-free)**; producers/consumers depend only on `brain`. This makes every edge point **into `brain`** — no `gateway↔observability` cycle.

- **I1 `LedgerCounters` (defined in brain; Codex1 produces → Codex4 reads):** `{offered, admitted, rejected, overloaded, started, completed, failedReal, failedInjected, cancelled}` integer snapshot. **IMPLEMENTED** in `brain/capacity.go` (`LedgerCounters` + `failedInjected`); NOT independently accepted until pB.
- **I2 `InjectedFailureMarker` (brain contract field; Codex2 sets → Codex1 ledger honors):** boolean tagging a synthetic-injected failure so the ledger increments `failedInjected` (never `failedReal`). **IMPLEMENTED** on the neutral contract; NOT independently accepted until pB.
- **I3 `GatewaySyntheticTelemetry` (type in brain; Codex2 populates → Codex4 reads):** `{retryCount, fallbackSameModel, fallbackCrossModel}`; `fallbackCrossModel` MUST be 0 unless an approved ordered policy exists. Content-off. **Integration gated on fallback-cycle validation (below).**
- **I4 `QueueDepthAccessor` (type in brain; Codex2 optional producer → Codex4 reads):** read-only current-depth getter if a gateway request queue exists; else Codex4's synthetic queue is authoritative and feeds `ObserveQueueDepth`.
- **I5 `SteadyStatePredicate` (type in brain; Codex2 producer → Codex4 reads):** content-free "circuits closed AND accounts re-entered" that **bounds** the `BeginRecovery/End` span (see recovery bound below).

No I* type is defined inside `gateway/**` or `observability/**`; if implementing one would require gateway to import observability (or vice-versa), that is a STOP (cycle).

## Exact file ownership (disjoint)

- **Codex1** — `internal/daemon/brain/**` (sole editor): I1–I5 **type definitions** (content-free), `capacity.go` ledger + `failedInjected`. Must not edit gateway/observability/non-owned central.
- **Codex2** — `internal/daemon/gateway/telemetry.go`, `gateway/policy.go` (+ tests): populate I3, set I2 marker, optional I4 accessor, I5 predicate, and the fallback-cycle validator. Must not edit brain type defs, observability, or central entrypoints.
- **Codex4** — `internal/daemon/observability/realtime.go`, `harness.go`, `synthetic.go`, `evidence.go`, new `observability/aggregate.go` (+ tests): consume I1/I3/I4/I5 read-only via brain; nearest-rank p50/p95/p99 aggregator; CPU/duration guards; bounded recovery. Must not edit brain/gateway.
- Central hotspots remain **Codex1-only**.

## Three-wave dependency order

1. **Wave 1 — DONE/contract:** Codex1 I1/I2 **implemented** (`brain/capacity.go`, `failedInjected`); Codex1 freezes I3/I4/I5 **type definitions in brain**. Codex4 builds self-contained #5 (E2E per-task = `ProcessMeasurement.Duration`) and the **nearest-rank p50/p95/p99** aggregator (`aggregate.go`) with declared minimum sample count. *(Acceptance of I1/I2 remains pending pB.)*
2. **Wave 2:** Codex2 populates I3 (retry/fallback), sets I2 marker, adds I4 accessor + I5 predicate, and implements the **fallback-cycle validator**. **I3 integration is blocked until fallback-cycle validation passes** (no A→B→A / unbounded fallback chains).
3. **Wave 3 — Codex4 integrate:** #1,#2 (from I1, excluding `failedInjected`), #6,#7 (from I3, only after cycle validation), #8 (I4 + synthetic queue + queue-latency + numeric growth slope), #14 (I5-bounded recovery with max-timeout), and #10 CPU% = `CPUTime ÷ (Duration × budget)` **with guards (below)**.

## Guards (fail-closed, never mask missing inputs)

- **CPU (#10):** if the declared CPU budget is **missing or ≤ 0**, or measured **Duration ≤ 0**, the computation **STOPs and the run FAILS** — the value is **never clamped to 0% or any passing number**. Missing input ≠ acceptance.
- **Recovery (#14):** every `BeginRecovery` span has a **bounded max timeout**; if steady-state (I5) is not reached within the bound, recovery **FAILS** (no open-ended spans).
- **Fallback (#7/I3):** before I3 is integrated into aggregation, the **fallback-cycle validator must pass**; a detected cycle or missing validation ⇒ STOP.
- **Percentiles:** below the declared minimum sample count ⇒ STOP (no noisy p50/p95/p99).

## Tests per stream (synthetic, offline)

- **Codex1:** ledger reconciles (`offered = admitted + rejected + overloaded`; `failedInjected ≤ failed`); injected marker increments only `failedInjected`; exactly-once lease release; I1–I5 type defs are content-free; no secret.
- **Codex2:** deterministic retry/fallback counts; `fallbackCrossModel = 0` without approved policy; **fallback-cycle validator rejects A→B→A and unbounded chains**; I5 predicate flips only when circuits closed + accounts re-entered; queue accessor bounded/non-negative; content/credential scans zero.
- **Codex4:** nearest-rank p50/p95/p99 correctness on known sets + min-count STOP; E2E per-task; completion/error from I1 excluding injected; retry/fallback from I3 (post cycle-validation); queue peak/wait/growth-slope; **CPU guard STOPs on zero/absent budget or duration (no clamp)**; recovery bounded by max timeout; all `ContentCapture=false`; one synthetic end-to-end reconciliation proving all 8 gaps yield field+equation.

## STOP conditions (any → halt and report)

- Any real credential/secret read, live provider, network call, or Multica-daemon dispatch.
- Introducing a `gateway↔observability` import cycle (define I* in `brain` instead).
- CPU budget or duration missing/≤0 clamped into a passing value (must STOP).
- Fallback-cycle detected, or I3 integrated before cycle validation passes.
- Recovery span exceeding its max timeout treated as pass.
- Cross-model fallback without approved policy.
- Ledger counter mismatch, leaked lease, duplicate terminal, non-reconciling snapshot.
- Content/secret/account-identity entering any telemetry/measurement field.
- A stream editing another stream's files, or a non-Codex1 edit of a central hotspot / brain type defs.
- Any PD-01 baseline reset/revert/discard or PD-08 credential mutation.
- Claiming independent acceptance of I1/I2 (or any item) before pB completes.

## Measurement-validity note

Latency/throughput/resource numbers here are **synthetic-shape/plumbing validation only** — NOT
performance or capacity acceptance; real numbers require the separately-gated live path. cgroup/
per-process resource sampling bounds the measured run, not unrelated daemon behavior. These caveats
accompany any evidence produced under this packet.
