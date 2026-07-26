# T20-harness-plan — deterministic 20-task submission design

- agent: `Codex56#A` · lane: `LANE2` · task: `T20-HARNESS-PLAN` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/T20-harness-plan.md`
- documents: `/tmp/t20_workload.py` — sha256 `6d266266ffc3719129c494709d218ac40776ff872f3cde2ad06e537b0d2d2c00`, 46 lines, 2115 bytes.
- MODE: **DOCUMENTATION ONLY**. Not executed here — running the harness submits real chat/tasks (a
  gated **live run**; `live_runs=false`, tier-20/`6.3` externally gated). No inference, no deploy, no secret.

> The harness offers a fixed **burst of 20 concurrent tasks**, each a *direct-target* chat session +
> one *read-only* message, keyed by a deterministic index `#i (1..20)` embedded in the prompt, with a
> receipt sorted by `i` and persisted to JSON. The offered-load *shape* is fully reproducible; the
> server-assigned session/task UUIDs and timings are inherently per-run.

## 1. Target + fixed parameters (deterministic constants)

| Param | Value | Note |
|---|---|---|
| Backend base `B` | `http://127.0.0.1:18080` | local loopback backend under test (distinct from OmniRoute `:20128`) |
| Workspace `WS` | `20fce817-895d-447b-965a-49f5e279314a` | must pre-exist |
| Agent `AGENT` | `8101fcf3-2e58-4a15-9583-70fc8749ee63` | "Direct-Acceptance, online runtime" — must be online |
| `N` | `20` | tier-20 offered load (matches architect §7.3 first tier) |
| HTTP timeout | `12s` per call | in `call()` |
| Receipt file | `/tmp/t20_receipt.json` | `{submitted, agent, n}` |

## 2. Per-task unit of work (2 HTTP calls each)

`submit(i)` for `i` in `1..20`:
1. Capture `submit_ts = HH:MM:SSZ` (UTC, `time.gmtime`).
2. **POST** `/api/chat/sessions?workspace_id=WS` body `{"agent_id": AGENT}` → **direct-target** session
   create (explicit `agent_id` = the direct-to-agent escape hatch; cf. the C3 optional-agent contract —
   here agent_id is supplied → routes directly, no squad TL, no delegation). Extract `session_id`.
   - Missing `session_id` → receipt `{i, err:"no-session", resp}` (task counts as not-submitted).
3. **POST** `/api/chat/sessions/{session_id}/messages?workspace_id=WS` body
   `{"content": "[cap-tier20 #{i}] Report your current working directory. Read-only; do not modify anything; do not delegate."}`
   → returns `task_id`, HTTP status `send_http`.
4. Receipt: `{i, submit_ts, session_id, task_id, send_http}`.

## 3. Offered load, concurrency, ordering

- **Offered load:** 20 logical tasks = 40 HTTP requests total (1 session-create + 1 message per task).
- **Concurrency:** `ThreadPoolExecutor(max_workers=20)` + `ex.map(submit, range(1,21))` → **all 20 submit
  in parallel** — a synchronized burst (peak concurrency 20), not a rate-limited trickle. This is the
  intended tier-20 saturation shape.
- **Per-task ids:** the deterministic key is the **index `i`**; the prompt carries the stable marker
  `[cap-tier20 #i]` for correlation. `session_id`/`task_id` are **server-assigned** (fresh per run).
- **Ordering:** submission wall-clock order is non-deterministic (concurrent). The **receipt is sorted
  by `i`** (`receipts.sort(key=lambda x: x["i"])`) so the printed/persisted output is deterministic and
  diffable. Server-side *start* order is intentionally unconstrained (concurrency/fairness under load).

## 4. Success accounting + output

- `ok = [r for r in receipts if r.send_http == 201 and r.task_id]` → **HTTP 201 + a task_id** is the
  submit-success criterion (message accepted, task enqueued).
- Prints `=== WORKLOAD START RECEIPT: {len(ok)}/{N} bounded tasks submitted in {elapsed:.1f}s ===` then
  one line per task (`#i ts= session=<8-char> task=<id> http=<code> <err>`), and writes
  `/tmp/t20_receipt.json`. This is a **submission** receipt (start-of-work), not a completion/acceptance record.

## 5. Reproducibility

**Deterministic (identical every run):** target `B/WS/AGENT`, `N=20`, concurrency=20, the prompt
template + `#i` markers, the read-only/no-modify/no-delegate instruction, the receipt sort-by-`i`, the
success predicate (201 + task_id), and the receipt file path. Pin the script by sha256 (§0) to fix the
design.

**Inherently per-run (does NOT affect offered-load shape):** server-assigned `session_id`/`task_id`
UUIDs, `submit_ts` wall-clock, HTTP timing/elapsed, and the actual task start/completion order and
outcomes (depend on live daemon/agent/OmniRoute state).

**Not idempotent in state:** each run creates **20 new sessions + 20 new tasks** (additive). Repeated
runs accumulate real work; there is no dedupe key. For clean comparison, run against a known workspace
state and archive `/tmp/t20_receipt.json` per run (e.g. timestamped copy).

**Preconditions to reproduce:** backend reachable at `:18080`; `WS` exists; `AGENT` exists and its
runtime is **online**; the daemon + OmniRoute path ready (else sessions/messages fail and `ok < 20`).

**Safety of the workload:** the prompt is a benign read-only probe ("report cwd; do not modify; do not
delegate") — no mutation, bounded output, no fan-out. It exercises the *submission + capacity* path,
not destructive behavior.

## 6. Execution gate (why this is documentation, not a run)

Running `/tmp/t20_workload.py` **submits real chat sessions + messages → enqueues 20 real agent tasks**
= a **live run**. Under current gates that is **not authorized**: `live_runs.*=false`, and tier-20
capacity acceptance (`6.3`) is an external gate requiring Principal/operator authorization, a ready
deployed stack, and the reserved single-run token per family. This artifact freezes the deterministic
design + reproducibility so an authorized operator can execute it later and attach the resulting
`/tmp/t20_receipt.json` (submission receipt) plus downstream completion/terminal evidence.

- **Owner of execution:** Principal Orchestrator (`w5:p9`) + operator, once tier-20 is authorized and
  the stack (backend `:18080`, agent online, OmniRoute ready) is confirmed.
- **Not this lane:** I did not run it, did not probe `:18080`, and captured no live receipt.

## 7. Non-claims / limitations
- Documentation only — no execution, no live submission, no `:18080` probe, no inference/secret/deploy; no source edits.
- The harness is a *submission* load generator; it records start receipts, not capacity/fairness metrics
  (p50/p95/p99, queue depth, CPU/mem) — those require a separate measured run per the evidence contract.
- Agent/workspace IDs are environment-specific to the harness author's setup; reproduction requires the
  same (or equivalent online) IDs.
- Backend `:18080` is the harness target as written; confirm it is the intended tier-20 backend before any authorized run.
