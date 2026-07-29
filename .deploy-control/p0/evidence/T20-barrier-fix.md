# T20-barrier-fix — deterministic ~120s read-only hold + timeout/cancel cleanup (design only)

- agent: `Codex56#A` · lane: `T20-WAVE` · task: `T20-BARRIER-FIX` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/T20-barrier-fix.md`
- extends: `T20-harness-plan.md` (/tmp/t20_workload.py). MODE: **DESIGN ONLY** — not executed; running it is a gated live run (`live_runs=false`).

> GOAL: keep each of the 20 tasks reliably ACTIVE ~120s so all 20 overlap (a measurement barrier),
> using a **read-only** hold whose duration is fixed by a **shell wall-clock bound**, not by the model's
> sense of time — plus an explicit timeout/cancel cleanup so nothing hangs past the bound.

## 1. Root cause — why prose "sleep" is ignored (agents finish in ~10s)

The current prompt asks a question ("report cwd") and adds a prose "sleep". An LLM agent has **no
reliable internal timer**: it answers the question and returns in one turn. Prose like "wait 120s" is
not an executable action, so the task completes in ~10s. **Determinism must come from a real
wall-clock-bounded shell command the agent is required to run — never from the model's timing.**

## 2. The hold — mandatory single read-only bounded-blocking tool call

Replace the answerable prompt with a **single-step, imperative** contract whose *only* deliverable is
running one exact read-only command that blocks for a fixed wall-clock duration.

**Prompt (per task `#i`):**
```
[cap-tier20 #{i}] Run EXACTLY this one shell command and then report only its exit code.
Do not answer any question, do not run any other command, do not read/modify files, do not delegate.
Command:
  timeout 115 sh -c 'end=$((SECONDS+110)); while [ "$SECONDS" -lt "$end" ]; do pwd >/dev/null 2>&1; sleep 5; done; echo T20_HELD_#{i}'
```

Why this command is the most reliable **read-only** hold:
- **Wall-clock deterministic:** the inner `while [ $SECONDS -lt end ]` bound (110s) is fixed by the
  shell clock, not the model. It runs ~110s regardless of what the agent "thinks".
- **Read-only:** `pwd` + `sleep` + `echo` only — no writes, no network, no delegation, bounded output.
- **Stays observably active:** the 5s `pwd` heartbeat loop produces periodic activity so any
  activity/heartbeat-based liveness sees the task as busy (a bare `sleep 120` can look idle and is more
  likely to be short-circuited).
- **Self-terminating + hard-capped:** the outer `timeout 115` guarantees the command exits by ~115s
  even if the loop misbehaves (exit 124 on timeout), so a task can never hang unbounded from the hold.
- **Unambiguous single action:** "run EXACTLY this one command, report only its exit code, do nothing
  else" removes the model's freedom to answer-and-finish — the hold *is* the task.

**Fallbacks (if the runtime shell lacks a feature), same contract, ranked:**
1. `timeout 115 tail -f /dev/null` — blocks exactly ~115s (exit 124), read-only, minimal.
2. `sleep 110` (plain) — simplest; use only if `timeout`/`SECONDS` unavailable; less observable.
The loop form (§2 primary) is preferred for observable liveness + hard cap.

## 3. Duration choice vs platform timeouts (avoid premature kill)

- Pick hold **< the daemon's task running-timeout** so the platform does not kill the hold early and
  mis-record it as a failure. **BLOCKER/verify:** confirm the effective per-task running timeout on the
  tier-20 backend. The frontend notes a **60s** running timeout for *model discovery* (`packages/core/runtimes/models.ts`);
  the **agent-task** running timeout must be confirmed by the daemon/service owner. If the task timeout
  is ~60s, set the hold to ~50s (and the barrier target to that); if it is ≥120s, 110–115s holds.
  **Do not exceed the platform timeout** — the design targets "reliably active up to the platform
  bound," and the harness cancel (§4) enforces the upper edge deterministically.
- Recommended: parameterize `HOLD_SECONDS` (default 110) and `TIMEOUT_SECONDS = HOLD_SECONDS + 5`, both
  ≤ the confirmed platform task timeout.

## 4. Explicit timeout / cancel cleanup (two layers, both deterministic)

1. **In-command hard timeout (self-cleanup):** `timeout 115 …` → the agent's command self-terminates;
   exit `124` (timeout) or `0` (loop finished) are **benign expected** terminals for the barrier, not errors.
2. **Harness cancel sweep (authoritative upper bound):** after the barrier window, the harness issues an
   **explicit cancel** to every still-running task so none linger:
   - At `T0 + CANCEL_AT` (e.g. 120s), for each receipt with a `task_id`, call the cancel API
     (`POST /api/daemon|chat cancel` — the same path `CancelTaskResponse`/RerunIssue cancel uses) and
     record a cancel receipt `{i, task_id, cancel_http, ts}`.
   - Cancel is idempotent-safe: a task that already self-terminated returns a benign "already terminal"
     (treat non-2xx "not cancellable/terminal" as benign, mirroring the superseded-skip classification
     from `FIX-race-design.md`).
3. **Classification:** a task is "barrier-held-OK" if it was ACTIVE across the overlap window and reached
   a terminal via `timeout` self-exit OR harness cancel. **Neither counts as a task failure** for the
   barrier; record `held_ok / cancelled / errored` separately.

## 5. Barrier determinism (all 20 overlapping)

- Submission stays the T20 burst (ThreadPoolExecutor max_workers=20). With a ~110s hold and a
  few-second submit+startup spread, there is a solid window where **all 20 are simultaneously active** —
  the measurement barrier.
- Optional tightening: after submit, poll each task's status until `running` (bounded, e.g. ≤20s) before
  starting the cancel timer, so `CANCEL_AT` is measured from "all-running" rather than "submitted",
  making the overlap window deterministic. Record the all-running timestamp.
- Deterministic knobs: `N=20`, `HOLD_SECONDS`, `TIMEOUT_SECONDS`, `CANCEL_AT`, fixed prompt+`#i`,
  sorted receipts — same offered-load/hold shape every run (pin script by sha256).

## 6. Harness integration (exact edits to /tmp/t20_workload.py — for the authorized operator)

- Replace the message `content` in `submit(i)` with the §2 hold prompt (parameterized by `HOLD_SECONDS`).
- Add constants `HOLD_SECONDS=110`, `TIMEOUT_SECONDS=115`, `CANCEL_AT=120` (all ≤ confirmed platform task timeout).
- After the submit map: optional bounded "wait until running" poll; then at `CANCEL_AT` run a cancel
  sweep over all `task_id`s and append `cancelled` receipts to `/tmp/t20_receipt.json`.
- Keep read-only guarantees and the receipt sort-by-`i`.

## 7. Non-claims / limitations
- Design only — not executed, no live submission, no `:18080` probe, no inference/secret/deploy, no source edits.
- **Depends on the confirmed platform per-task running timeout** (§3) — owner: daemon/service owner; the
  hold must be set below it. If the platform hard-caps tasks below 120s, "~120s per task" is infeasible
  and the achievable barrier is that platform bound (report, do not weaken timeouts).
- Reliability rests on the agent executing the single mandated command; the imperative single-action
  contract + observable loop maximize this, but a runtime that refuses shell tools would need a
  runtime-level hold (out of prompt scope) — flag if observed.
- Execution remains a gated live run (`live_runs=false`, tier-20/`6.3` external); owner: Principal (`w5:p9`) + operator.

---

## 8. VALIDATION (2026-07-23) — mechanism proven locally + platform-fit resolved

### 8.1 Hold-mechanism determinism — proven locally (safe: local shell, read-only, no product/inference)

Scaled reproduction of the §2 command (`timeout <cap> sh -c 'end=$((SECONDS+H)); while [ "$SECONDS" -lt "$end" ]; do pwd >/dev/null; sleep 1; done; echo …'`):

| Case | Params | Expected | Observed | Verdict |
|---|---|---|---|---|
| A — loop finishes first | HOLD=6, TIMEOUT=8 | ~6s, exit 0, prints marker | `elapsed=6s exit=0`, printed `T20_HELD_A` | PASS |
| B — outer timeout fires | HOLD=20, TIMEOUT=5 | ~5s, exit 124, no marker | `elapsed=5s exit=124`, marker NOT printed | PASS |
| Shell features | — | `timeout` + `SECONDS` present | `/usr/bin/timeout`; `sh -c 'echo $SECONDS'` → `SECONDS=0` | PASS |

Confirms: the hold duration is **wall-clock deterministic** (independent of any model), the outer
`timeout` **hard-caps** and yields a clean **exit 124**, the normal path yields **exit 0** with the
marker, and the construct is **read-only + portable** (POSIX `sh`, coreutils `timeout`). Extrapolates
linearly to HOLD=110 / TIMEOUT=115.

### 8.2 Platform-fit — §3 dependency RESOLVED (read from source)

- `config.go:45` `DefaultAgentIdleWatchdog = 30 * time.Minute` (fires only on inactivity).
- `config.go:55` `DefaultAgentToolWatchdog = 2 * time.Hour` (force-stops a *single silent tool call*
  only after 2h; `config.go:103`: "backstop for hung tools now that there is **no wall-clock cap**").
- `config.go:27-28`: max agent run "0 = no cap: bounded only by the inactivity watchdogs."
- The "60s running timeout" (`daemon.go:2134`) is an operation-window reference, **not** a cap on how
  long a running task may execute.

⇒ A ~110s hold — **even fully silent** — is far under both watchdogs (110s ≪ 30 min ≪ 2h) and there is
**no wall-clock task cap**, so the hold will **not** be prematurely killed. **"~120s per task" is
feasible**; the earlier §3 blocker is resolved. (Periodic stdout heartbeat remains a nice-to-have for
observability but is not required for watchdog survival at ~110s; keeping `sleep 5` cadence is fine.)

### 8.3 Barrier overlap — deterministic

With submission as a 20-wide burst (few-second spread) and a ~110s hold >> spread+startup, all 20 tasks
are simultaneously active for a solid window; the harness cancel sweep at `CANCEL_AT≈120s` (§4)
deterministically closes it. Optional "wait until all `running`" tightens the window start.

### 8.4 Scope of this validation / residual

- **VALIDATED (read-only, offline):** hold-command determinism (timing/exit/read-only) and platform-fit
  (watchdogs/no-cap) → the approach reliably holds each task ~110–115s without premature kill.
- **NOT validated here (gated live run):** end-to-end behavior on the deployed stack — that the agent
  reliably executes the single mandated command and that all 20 overlap in practice. Requires tier-20
  authorization (`live_runs=false`, `6.3` external; owner Principal `w5:p9` + operator). Residual risk:
  a runtime that refuses shell tools would need a runtime-level hold (flag if observed).
- Commands run locally in this pane only (scaled shell timing); no product, no `:18080`, no inference,
  no secret, no deploy, no source edits.
