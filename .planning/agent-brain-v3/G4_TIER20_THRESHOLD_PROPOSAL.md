# G4_TIER20_THRESHOLD_PROPOSAL — Agent Brain v3 (task 9.1)

> **STATUS: NON-AUTHORITATIVE PROPOSAL. NOT EVIDENCE. NOT APPROVAL. NOT ACTIVATION.**
> Every number is **PROPOSED / PENDING OWNER APPROVAL** — a conservative starting point from the
> synthetic development contracts, not a measured result. This file does not run task 9.1, enable
> tier-20 (9.2), change any OpenSpec checkbox, or touch code/credentials/live systems. Revised to
> resolve the independent (pB) review defects. 46/85 unchanged; PD-01/PD-08 absolute.

## Scope and derivation basis

- Profile: the **synthetic, single-node, development** 20-task model (`evidence/g4-synthetic-capacity-phase1.md`) + central capacity ledger (`evidence/g4-tier20-central-prereqs.md`). Development-validation only; not production SLOs; single-node caveat (D-V3-08) applies.

## Measurement conventions (PROPOSED)

- **Phase windows (explicit, phase-tagged):** each run has three named windows — **warm-up = first 5 s**, **steady/measurement = next 30 s**, **cool-down = final 10 s** (durations PROPOSED/PENDING). Every sample and every task/request outcome is **phase-tagged**. Only **steady-tagged** populations feed thresholds; warm-up/cool-down populations are recorded but excluded from pass/fail. Missing/ambiguous phase tags ⇒ STOP (never inferred).
- **Percentiles:** **nearest-rank** method (no interpolation); report **p50 / p95 / p99** for each latency class over the steady population, with a declared **minimum sample count** (PROPOSED ≥ 100); below the minimum ⇒ STOP (no noisy percentile).
- **No-first-output rule:** a request that never emits first output is a **no-first-output FAILURE** — counted against completion/error and never dropped from populations to make a latency percentile look good.
- **Resource sampling (cgroup):** CPU and RSS are read from the **run's cgroup**, sampled ≥ 1 Hz; **peak** is the primary statistic (steady median also recorded). RSS reported in **MiB** (bytes ÷ 2²⁰). CPU reported as % of a **declared CPU budget** (owner-provided cores; PENDING).
- **Frozen integer outcome model (20 tasks):** `admitted, completed, failedReal, failedInjected, cancelled, rejected, overloaded` are **integer counts**; invariant `offered = admitted + rejected + overloaded`. **Injected failure cases are a separately-required set** with their own required integer count (`failedInjected ≥ requiredInjected`, PROPOSED) and are **excluded** from the error/completion bases (reported separately). No fractional/derived denominators.

## Proposed thresholds (14) — all PROPOSED / PENDING OWNER APPROVAL

| # | Threshold | PROPOSED value | Unit | Denominator / population (frozen, steady-phase) | Pass if |
|---|---|---|---|---|---|
| 1 | Completion rate | ≥ 99.0 | % | `completed ÷ (admitted − failedInjected)` (integers) | measured ≥ threshold |
| 2 | Error rate | ≤ 1.0 | % | `failedReal ÷ (admitted − failedInjected)`; injected counted separately | measured ≤ threshold AND `failedInjected ≥ requiredInjected` all correctly classified |
| 3 | Selection latency | p50 ≤ 10 / p95 ≤ 50 / p99 ≤ 100 | ms | steady selection samples, nearest-rank | all three percentiles ≤ their PROPOSED values |
| 4 | Time-to-first-token | p50 ≤ 500 / p95 ≤ 2000 / p99 ≤ 4000 | ms (synthetic mock) | steady first-output samples; no-first-output = failure | all three ≤ AND no-first-output rate ≤ error bound |
| 5 | End-to-end latency (per task) | p50 ≤ 8000 / p95 ≤ 30000 / p99 ≤ 45000 | ms | steady per-task durations, nearest-rank | all three ≤ |
| 6 | Retry rate | ≤ 5.0 | % | steady requests with ≥1 bounded retry ÷ steady requests | measured ≤ |
| 7 | Fallback rate | ≤ 5.0 (same-model); cross-model = 0 | % | steady requests with fallback ÷ steady requests | measured ≤ AND cross-model = 0 unless approved policy AND fallback-cycle validation passed |
| 8 | Queue peak / wait / growth | peak ≤ 40 (int); wait p50 ≤ 1000 / p95 ≤ 5000; growth slope ≤ 0 | count / ms / slope | steady queue-depth series + queue-wait samples | peak ≤ bound AND waits ≤ AND steady linear-fit depth slope ≤ 0 (numeric) |
| 9 | Fairness deviation (capacity-time) | ≤ 10 | % | `max_slot |selectionShare − capacityTimeShare|`, where `capacityTimeShare = slotEligibleCapacityTime ÷ Σ` | measured ≤ threshold — **no post-hoc waiver**; ineligible/affinity/cooldown time is already folded into `capacityTimeShare` deterministically |
| 10 | CPU (peak, cgroup) | ≤ 80 | % of declared CPU budget (cores) | run cgroup peak CPU | measured ≤ threshold; **budget missing or ≤0, or duration ≤0 ⇒ STOP (never clamp to pass)** |
| 11 | RSS (peak, cgroup) | ≤ 512; steady slope ≤ 0 | MiB | run cgroup peak RSS | peak ≤ threshold AND steady RSS linear-fit slope ≤ 0 |
| 12 | Sockets / fds | peak ≤ 2×tasks + 16 (int); post-run leak = 0 | count | run cgroup/process-tree fds+sockets | peak ≤ bound AND post-run run-attributable open sockets/fds = 0 |
| 13 | Cancellation | `cancellationsRequested == releases == requiredCancels`; release p95 ≤ 500 | count / ms | cancellation events | exact-equality of counts AND each releases **process-tree + queue slot + lease exactly once** AND release p95 ≤ |
| 14 | Recovery to steady state | ≤ 60 per event; `recoveryEvents == injectedDisruptions` | s / count | recovery spans (bounded) | each span ≤ threshold AND count matches disruptions AND steady-state = circuits closed ∧ accounts re-entered ∧ resources→baseline; **span exceeding max timeout ⇒ FAIL** |

## Fail-closed comparison rule (revised)

A 9.1 run **PASSES only if every threshold is (a) owner-approved and (b) satisfied by a steady-phase-tagged measurement**. It **FAILS closed** if any of:
- a threshold lacks an owner-approved number; a PROPOSED value alone is insufficient;
- **any required input is missing** — CPU budget, phase tags, minimum sample count, `requiredInjected` count, fallback-cycle validation — **these STOP; they are never clamped/inferred into acceptance** (e.g., zero/absent CPU budget or zero duration must NOT resolve to 0% pass);
- any measured value is worse than its approved threshold (p50/p95/p99 each);
- a request had no first output (counts as failure);
- fairness deviation exceeds bound (no post-hoc waiver);
- cross-model fallback outside approved policy, or a fallback cycle is detected;
- queue/RSS growth slope > 0, or a recovery span exceeds its max timeout;
- counts do not reconcile (cancellation/recovery/lease/process-tree/queue-slot not released exactly once), a launch happened on failed readiness, a duplicate terminal occurred;
- any real credential/secret, live provider, second router, or Multica-daemon dispatch was required.

> **Gate semantics (PD-08 / 8.1 vs 9.1):** PD-08 is an **absolute STOP invariant** and does **not
> categorically prohibit** a contained, offline, synthetic 9.1 attempt; PD-08 compliance for such an
> attempt is **accepted only after every containment, prerequisite, named-evidence-owner authorization
> and STOP condition in the acceptance checklist has been verified** (not by construction). **Task 9.1
> is presently STOPPED / not ready.** Legacy-credential remediation remains **mandatory before any
> live-auth or cutover path**. Task **8.1** (authenticated/live path) is **OPEN** — blocked by PD-08's
> invariant + no-live-provider. Task **9.1** (offline synthetic) is scoped to development validation
> (D-V3-14) and is **subject to every prerequisite and STOP condition in the acceptance checklist**,
> with thresholds PENDING owner and independent pB acceptance required. This does not weaken PD-08.

Passing 9.1 is **necessary but not sufficient** for 9.2 (tier-20 enable), which additionally needs the §7.4 gates (PD-02 pinned digest, PD-03 Smart Context, affinity-`preserve` proof, single-flight refresh, readiness fail-closed) and stays development-only.

## Exact remaining owner decisions (numbers to approve or replace)

1. **CPU budget** (cores) for #10 denominator — no default invented.
2. **Phase-window durations** (warm-up 5 / steady 30 / cool-down 10 — confirm units and values).
3. **`requiredInjected`** count of injected failure cases (#2) and the **`requiredCancels`**/`injectedDisruptions` counts (#13/#14).
4. **Minimum sample count** for percentiles (proposed ≥ 100).
5. Each threshold number in the table (completion/error/latency p50/p95/p99/retry/fallback/queue bound+waits+slope/fairness/RSS/sockets/cancellation/recovery).
6. Whether any **approved cross-model fallback policy** exists (else #7 requires cross-model = 0).

## What this file is NOT

Not evidence, not approval, not a 9.1 authorization, not tier enablement. No checkbox changed; 46/85 unchanged; PD-01/PD-08 absolute; no code/credential/live action.
