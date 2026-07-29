# G4_OWNER_DECISIONS_9_1 — remaining task-9.1 blockers (owner input required)

> **READ-ONLY OWNER-DECISION PACKET. No values invented.** Every field below is **PENDING OWNER**;
> where a *non-authoritative PROPOSED* value exists it is referenced (from
> `G4_TIER20_THRESHOLD_PROPOSAL.md`) as a proposal only, **not a decision**. This file changes no
> OpenSpec checkbox, accepts nothing, and touches no product/credential/network/live system.
> Task **9.1 STOPPED/not ready**; **8.1 OPEN/live**; 46/85; PD-01/PD-08 preserved. The 9.1 run stays
> STOPPED until the owner supplies these and the code gates (pA SteadyStateFacts/aggregate, p9 catalog,
> provenance reconciliation) are independently accepted by pB.

## A. Baseline / build identity
| ID | Decision | Value |
|---|---|---|
| A1 | OmniRoute image **digest** pin (PD-02; replace `:latest`) | PENDING OWNER |
| A2 | Harness toolchain/**cache digest** pin (Go 1.26 container + module cache) | PENDING OWNER |

## B. Resource limits for the 20-task run (cgroup enforcement)
| ID | Decision | Value |
|---|---|---|
| B1 | **CPU** limit/quota (cores or cgroup cpu.max) | PENDING OWNER |
| B2 | **Memory** limit (cgroup memory.max) | PENDING OWNER |
| B3 | **PID** limit (pids.max) | PENDING OWNER |
| B4 | **nofile / FD** limit | PENDING OWNER |
| B5 | **Socket** cap | PENDING OWNER |

## C. Containment policy
| ID | Decision | Value |
|---|---|---|
| C1 | **cgroup** controller/hierarchy (v2 path, delegation) | PENDING OWNER |
| C2 | **Process-tree** containment + cleanup policy (grouping, kill-on-exit, orphan reaping) | PENDING OWNER |

## D. 20-task workload policy
| ID | Decision | Value |
|---|---|---|
| D1 | Model mix / route mix | PENDING OWNER |
| D2 | Streaming ratio | PENDING OWNER |
| D3 | Tool-call ratio | PENDING OWNER |
| D4 | Prompt/output sizes | PENDING OWNER |
| D5 | Request rate + run duration | PENDING OWNER |
| D6 | Account-pool size + per-account/route/global concurrency limits | PENDING OWNER |

## E. Measurement parameters
| ID | Decision | Value (proposal reference — not a decision) |
|---|---|---|
| E1 | **Phase durations** warm-up / steady / cool-down | PENDING OWNER (proposed 5/30/10 s) |
| E2 | **Minimum sample count** for percentiles | PENDING OWNER (proposed ≥ 100) |
| E3 | **CPU budget** (cores) — #10 denominator; absent ⇒ STOP (never clamp) | PENDING OWNER |
| E4 | **14 thresholds** (completion, error, selection p50/p95/p99, TTFT p50/p95/p99, E2E p50/p95/p99, retry, fallback, queue peak+waits+slope, fairness %, CPU %, RSS MiB, sockets bound, cancellation release p95, recovery s) | PENDING OWNER (see PROPOSED table; approve/replace each) |
| E5 | **Cross-model fallback policy** — approved ordered chain? else cross-model = 0 | PENDING OWNER |

## F. Evidence governance
| ID | Decision | Value |
|---|---|---|
| F1 | Named **evidence owner** authorizing the contained offline 9.1 attempt | PENDING OWNER |
| F2 | **Output location** for 9.1 evidence artifacts | PENDING OWNER |
| F3 | **Retention** policy (logs/metrics/artifacts; bounded) | PENDING OWNER |

## G. Cross-cutting gates already tracked (9.2 / cutover — not 9.1 run inputs, listed for completeness)
| ID | Decision | Value |
|---|---|---|
| G1 | PD-03 Smart Context SC01–SC10 waiver-or-impl | PENDING OWNER |
| G2 | Single-flight refresh + readiness fail-closed (§7.4) | PENDING (evidence) |
| G3 | PD-08 legacy-credential remediation (live-auth/cutover only) | PENDING OWNER |

## Fail-closed note
Any missing required input (A–F) leaves 9.1 **STOPPED**; no value is inferred/clamped, and a PROPOSED
value alone never becomes the comparison basis. Owner approval + independent pB acceptance of the code
gates are both required before a separately authorized 9.1 run.
