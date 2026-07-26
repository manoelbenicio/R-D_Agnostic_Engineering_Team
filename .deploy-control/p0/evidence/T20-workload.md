# T20 — bounded 20-concurrent workload generator (PREPARE-ONLY; DO NOT RUN)

- agent: **Opus48#C** · lane **tier20-workload** · task **T20-WORKLOAD-PREP** · pane `w8:p1`
- as-of (UTC): `2026-07-23T21:09Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock (sole mutable): `.deploy-control/p0/evidence/T20-workload.md`
- check-in: `.deploy-control/p0/checkins/Opus48-C__T20-WORKLOAD-PREP__20260723T210941Z.json`

## 0. Hard gate — NOT RUN

**This is a preparation artifact only. Nothing is executed; no task is enqueued; no inference is issued.** Per operator instruction ("Do NOT run until I give go") and `control.json`: **`live_runs.{cline_glm,cline_kimi,opus48,antigravity}.authorized = false`** — a live run is forbidden. Running requires ALL of §5 to be GREEN. This prepares the tier-20 canary evidence-attempt (OpenSpec 9.1); it does NOT authorize tier activation (9.2) or cutover.

## 1. Objective

Submit **exactly 20 concurrent, safe, read-only** chat/issue tasks against the operator-approved route, capture each task's admission → queue → start → complete lifecycle with correlation IDs, compute bounded metrics, and **abort automatically** on any instability/error/persistence/trace threshold — one run, no duplicate acceptance.

## 2. Preconditions consumed (facts)

- Tier-20 schema: `brain.CapacityTier20 = 20`, `MaxActiveTasks: 20`, `TierStateCanaryAuthorized` (`brain/config.go`); daemon dev mode is authorized **only** for tier-20 (`daemon/config.go:194`); `CapacityGateEnabled` honored only with `CapacityTier20`; `DefaultMaxConcurrentTasks = 20`. → **bounded concurrency = 20; never exceed.**
- Submission API (existing, real): `TaskService.EnqueueQuickCreateTask(workspaceID, requesterID, agentID/squadID, prompt, projectID, parentIssueID, attachmentIDs)`, `EnqueueTaskForIssue(issue, triggerCommentID?)`, `EnqueueChatTask(chatSession, initiatorUserID)`. The generator uses these through the backend API (the same Kanban/chat path a user drives) — no bespoke enqueue.
- Lifecycle hooks (real): admission `agentBrainRuntime.admitTask` (router_owner=omniroute, readiness gates); queue enqueue/claim in `agent_task_queue`; `StartTask` (dispatched→running); `CompleteTask` (terminal persist, idempotent). Correlation is the FROZEN `agent-brain.e2e.v1` contract (9 IDs / 8 hops; `secrets_present=false`).
- Approved route: the operator/Principal designates `<APPROVED_ROUTE_FAMILY>` + exact `<APPROVED_ROUTE_MODEL>` from the frozen route matrix (`authoritative-route-matrix-D-V3-27.md`); the generator NEVER invents a RouteModel and only targets the family whose `live_runs.<family>.authorized=true`.

## 3. Workload definition (safe, read-only)

- **20 tasks**, each a chat or issue task whose prompt is strictly **read-only / non-mutating**: e.g. "summarize the following provided text", "answer this question", "list N safe facts" — no file writes, no shell/tool side effects, no repo mutation, no external calls. Prompts are fixed, synthetic, content-free of secrets.
- Distinct `workspace/project/issue|chat_session` per task (or a shared read-only fixture) so tasks are independent (no cross-task ordering dependency), enabling honest strict-concurrency measurement.
- Submission concurrency limited by a **semaphore of 20** (matches tier cap); the generator submits all 20 and lets the daemon admit up to `MaxActiveTasks=20`.

## 4. Per-task capture schema (redacted, metadata-only)

For each task capture a record keyed by `task_id`, populated from the redacted OBS spans (`agent-brain.e2e.v1`) and the `agent_task_queue` row transitions — **IDs and bounded codes only; never bodies/prompts/results/secrets**:

| Hop | Fields captured |
|---|---|
| ingress/submit | `request_id`, submit_ts, `route_model`, `cli_kind`, `principal_class` |
| queue | `queue_msg_id`, `task_id`, enqueue_ts, dequeue_ts, `wait_ms`, `queue_depth` |
| admission | `task_id`, `session_id`, `launch_id`, `admission_decision`, `readiness_result`, `router_owner`(=omniroute), `fail_closed_class?` |
| start | `launch_id`, `proc_id`, started_ts |
| route | `request_id`↔`omni_request_id`, `status_class`, retries/fallback counts (from safe telemetry) |
| persist | `task_id`, `result_id`, `terminal_status`, persisted_ts, `byte_count`/`token_count` |
| delivery | `session_id`, `delivery_id`, delivered_ts |

Derived per task: `admission_ms`, `queue_wait_ms`, `start_ms`, `run_ms`, `total_ms`, `terminal_status ∈ {completed,failed,timeout,cancelled,aborted}`.
Aggregate: p50/p95/p99 of total_ms; observed max concurrency; error rate; queue depth over time; per-task trace continuity.

## 5. GO / NO-GO preconditions (ALL must be GREEN before running)

1. **Operator "go"** explicitly given (this doc's gate).
2. `control.json` `live_runs.<APPROVED_ROUTE_FAMILY>.authorized == true` with exact `RouteModel` + integrated commit/build/config provenance and **no prior accepted run** for the same family/build/scenario.
3. Tier authorized: `Neutral.CapacityTier == CapacityTier20` AND `CapacityGateEnabled == true` (canary tier-20 only).
4. **D-V3-25(B) security stop CLEARED** (Owner confirms UI key invalidation/revocation) — live-provider tests are SECURITY-STOPPED otherwise.
5. Readiness sustained **Ready** at start: liveness + authentication + model-registry + selected-model + selected-protocol (strict readiness).
Any precondition RED ⇒ do not run.

## 6. ABORT thresholds (continuous watcher; abort = stop submitting, cancel in-flight, record reason)

| # | Threshold | Precise trigger | Signal source |
|---|---|---|---|
| A1 | **readiness instability** | gateway readiness transitions Ready→not-Ready at any point during the run, OR any post-start admission returns `readiness_result != ready` / auth-fail after initial Ready | `agentBrainRuntime.snapshot().Readiness`, admission records |
| A2 | **error > 10%** | `failed_or_nonterminal_count / 20 > 0.10` (i.e. ≥3 of 20 in {failed,timeout,aborted,execution_error}); also abort immediately on any `execution_error`/gateway-trip class | per-task `terminal_status`, admission `fail_closed_class` |
| A3 | **dup / lost persistence** | any `task_id` with >1 persisted terminal `result_id`/persist span (dup), OR a `completed` task with no persisted terminal row (lost) | `CompleteTask` idempotency + `EmitPersist` span + e2e assembler `AnomalyDuplicateSpan` |
| A4 | **trace breakage** | `e2e.Assemble(spans).AllContinuous == false` — any task missing an emitting hop, orphan span, or conflicting join | FROZEN `agent-brain.e2e.v1` assembler (`observability/e2e`) |
| A5 | **tier breach** (guard) | observed concurrent running tasks > 20 | `agent_task_queue` running count / capacity ledger |

On any trigger: the watcher halts new submissions, cancels in-flight tasks (graceful cancel → slot cleanup), and writes an ABORT record with the exact triggering metric + affected `task_id`s. No retry, no second run.

## 7. Generator shape (pseudocode — NOT executed)

```text
preflight: assert §5 all GREEN, else EXIT("no-go", reason)     # currently EXITs: live_runs all false
sem := semaphore(20); records := map[task_id]Record
start abortWatcher(records, thresholds A1..A5) -> abortCh
for i in 1..20:
    acquire(sem)
    go:
        rec := submit(readOnlyTask_i, <APPROVED_ROUTE_MODEL>)   # EnqueueQuickCreateTask/EnqueueChatTask
        capture admission/queue/start/complete + ids into records[rec.task_id]  (redacted)
        release(sem)
    if abortCh fired: break (stop submitting)
wait(in-flight or abort)
emit summary + per-task table + GO/ABORT verdict -> .deploy-control/p0/evidence/T20-run-<ts>.md
```
Correctness/timing is measured, never sleep-gated; concurrency is hard-capped at 20; all captured fields are redacted IDs/codes (`secrets_present=false`).

## 8. Output artifact (what a future authorized run WOULD produce)

`.deploy-control/p0/evidence/T20-run-<UTC>.md`: run provenance (commit/build/config/RouteModel/tier + who ran), the §4 per-task table, aggregate metrics (p50/p95/p99, error rate, max concurrency, queue depth), the §6 abort verdict (GREEN or the exact triggered threshold), and the single-run acceptance note. That artifact — not this prep doc — is the tier-20 (9.1) evidence.

## 9. Scope / non-claims
- **NOT RUN**: no task enqueued, no inference, no live call; `live_runs.*=false` ⇒ preflight would EXIT no-go now.
- No product code edited; no OmniRoute internals invoked/probed; no secret read/printed; no RouteModel invented (operator supplies the approved one).
- Does NOT authorize tier-20 activation (9.2), 50/100 tiers, cutover, or production; a single authorized run closes only its own overlapping 5.x/8.x scope — no duplicate acceptance.
- Evidence capture is metadata-only per `EVIDENCE_CONTRACT.md` + the frozen `agent-brain.e2e.v1` `secrets_present=false` invariant.

## 10. Status
- STATUS: DONE (prepared; awaiting operator "go" + §5 GREEN).
- DELIVERED: workload definition (§3), per-task capture schema (§4), GO/NO-GO gates (§5), ABORT thresholds (§6), generator shape (§7), output schema (§8). Ready to implement/run under authorization; not executed.
