# FIX-race-design — superseded-task StartTask race (design only)

- agent: `Codex56#A` · lane: `race-design` · task: `FIX-RACE-DESIGN` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/FIX-race-design.md`
- MODE: **DESIGN ONLY** (read-only; no source edits, no deploy). Constraint: no fail-closed weakening.

> DELIVERABLE: root-cause + a minimal fix with EXACT diff points. Primary fix = Option A (daemon
> treats a *fenced/superseded* StartTask as a benign skip, not a real failure → no circuit/failure
> trip). Complementary hardening = Option B (RerunIssue cancels in-flight tasks + enqueue atomically in
> one txn). Neither weakens fail-closed behavior: only a row that provably exists in a
> terminal/cancelled state is treated as benign; genuinely-missing/unknown tasks and all other errors
> still fail as before.

## 1. Root cause (verified by reading)

- `StartAgentTask` (SQL, `pkg/db/generated/agent.sql.go:~2556`, `:one`):
  `UPDATE agent_task_queue SET status='running' … WHERE id=$1 AND status IN ('dispatched','waiting_local_directory') RETURNING …`.
  If the row was fenced (e.g. `RerunIssue` cancelled it → status `cancelled`), the UPDATE matches **0
  rows** → `QueryRow.Scan` returns **`pgx.ErrNoRows`**.
- `TaskService.StartTask` (`internal/service/task.go:1228`): wraps **every** error generically —
  `return nil, fmt.Errorf("start task: %w", err)` — so a benign supersede is indistinguishable from a real failure.
- Handler `StartTask` (`internal/handler/daemon.go:1871`): on any service error →
  `writeError(w, http.StatusBadRequest, …)` (**400**). (Note: the row still exists, so the existing
  `requireDaemonTaskAccessWithWorkspace` 404 "task gone" idiom at `daemon.go:116` does NOT catch this —
  the task is present but in a terminal state.)
- daemon client `StartTask` (`internal/daemon/client.go:166`): `postJSON` → non-2xx → returns error.
- daemon caller (`internal/daemon/daemon.go:3456`):
  `if err := d.client.StartTask(ctx, task.ID); err != nil { return TaskResult{}, fmt.Errorf("start task failed: %w", err) }`
  → `handleTask` runs the FailTask + `taskfailure.Classify` path (`daemon.go:2891`/`3092`), recording a
  **real failure** and feeding the failure/health/circuit taxonomy — for a task that was legitimately superseded.

**Race window:** `RerunIssue` (`service/task.go:1727`) calls `CancelAgentTasksByIssueAndAgent`
(`agent.sql.go:399`, `:many`) then `enqueueRerunTask` in **separate** DB calls (no txn). If the daemon
has already *claimed* the prior task (status `dispatched`) and is between claim and `StartTask` when the
cancel lands, the cancel correctly fences the row, but the daemon's subsequent `StartTask` then
false-fails as above.

## 2. Option A — classify benign superseded/not-found (PRIMARY; exact diff points)

Mirror the established "task gone" benign idiom (`handler/daemon.go:71,116`), but for the terminal-state case.

**A1 — sentinel error.** `internal/service/task.go` (near other task sentinels): add
```go
// ErrTaskNotStartable means the task row exists but is no longer in a startable
// state (cancelled/superseded/terminal) — a benign supersede, not a failure.
var ErrTaskNotStartable = errors.New("task not startable (superseded or terminal)")
```

**A2 — classify in `TaskService.StartTask`** (`internal/service/task.go:1228-1231`):
- OLD:
  ```go
  task, err := s.Queries.StartAgentTask(ctx, taskID)
  if err != nil {
      return nil, fmt.Errorf("start task: %w", err)
  }
  ```
- NEW intent (uses the already-imported `errors`/`pgx` — see existing `errors.Is(err, pgx.ErrNoRows)` at
  task.go:942/988/1061/1140/1321):
  ```go
  task, err := s.Queries.StartAgentTask(ctx, taskID)
  if err != nil {
      if errors.Is(err, pgx.ErrNoRows) {
          // UPDATE matched 0 rows: either the row is gone or it exists in a
          // non-startable (cancelled/superseded/terminal) state. Look it up to
          // distinguish a benign supersede from a genuine error. Fail closed on
          // any lookup error.
          if cur, getErr := s.Queries.GetAgentTask(ctx, taskID); getErr == nil &&
              cur.Status != "dispatched" && cur.Status != "waiting_local_directory" {
              return nil, ErrTaskNotStartable
          }
          return nil, fmt.Errorf("start task: %w", err) // truly gone / unknown → unchanged
      }
      return nil, fmt.Errorf("start task: %w", err)     // any other error → unchanged
  }
  ```
  (No fail-closed weakening: only a *present* row in a non-startable state returns the benign sentinel.)

**A3 — map to a distinct HTTP status in handler `StartTask`** (`internal/handler/daemon.go:1880-1884`):
- OLD: `if err != nil { slog.Warn(...); writeError(w, http.StatusBadRequest, err.Error()); return }`
- NEW intent:
  ```go
  if err != nil {
      if errors.Is(err, service.ErrTaskNotStartable) {
          // Benign supersede: the row was fenced (e.g. RerunIssue). Tell the
          // daemon to skip without recording a failure. 409 = distinct, not 400/500.
          writeError(w, http.StatusConflict, "task superseded")
          return
      }
      slog.Warn("start task failed", "task_id", taskID, "error", err)
      writeError(w, http.StatusBadRequest, err.Error())
      return
  }
  ```

**A4 — daemon client recognizes benign status** (`internal/daemon/client.go:166-168`): return a typed
benign error on 409 so the caller can branch. Add exported sentinel `var ErrTaskSuperseded = errors.New("task superseded")`
and, in `StartTask`, map a `409` postJSON result to `ErrTaskSuperseded` (mirror how other daemon calls
already special-case status codes; the postJSON error must expose the HTTP status — reuse the existing
status-aware error type the client uses for 404 "gone").

**A5 — benign skip at the caller** (`internal/daemon/daemon.go:3456`):
- OLD: `if err := d.client.StartTask(ctx, task.ID); err != nil { return TaskResult{}, fmt.Errorf("start task failed: %w", err) }`
- NEW intent:
  ```go
  if err := d.client.StartTask(ctx, task.ID); err != nil {
      if errors.Is(err, ErrTaskSuperseded) {
          d.logger.Info("start task skipped: superseded", "task_id", task.ID)
          return TaskResult{Skipped: true}, nil // benign: no FailTask, no Classify, no failure/circuit
      }
      return TaskResult{}, fmt.Errorf("start task failed: %w", err) // unchanged real-error path
  }
  ```
  `handleTask` must treat `TaskResult{Skipped:true}` as a no-op terminal (release the slot; do NOT call
  FailTask/`taskfailure.Classify`). If a `Skipped` field does not already exist on `TaskResult`, add a
  minimal boolean; the alternative is a dedicated sentinel returned as the error and branched in
  `handleTask` before the FailTask path (`daemon.go:2891`).

## 3. Option B — atomic cancel-before-enqueue in RerunIssue (COMPLEMENTARY; exact diff points)

Goal: a single authoritative supersede so no window enqueues the new task before the old is cancelled,
and in-flight (claimed) rows are fenced under the same snapshot.

**B1 — wrap cancel+enqueue in one txn.** `internal/service/task.go` `RerunIssue`
(`:1771` cancel … `:1789` enqueue): begin a transaction, run `qtx := s.Queries.WithTx(tx)` (idiom used
across the codebase, e.g. `handler/daemon.go:2235`, `service/autopilot.go:157`), perform
`CancelAgentTasksByIssueAndAgent` and `enqueueRerunTask` on `qtx`, then commit. This removes the
cancel→enqueue gap and makes the supersede atomic.

**B2 — cancel query already fences in-flight states (CONFIRMED).** `CancelAgentTasksByIssueAndAgent`
(`agent.sql.go:399`) is `… SET status='cancelled' … WHERE issue_id=$1 AND agent_id=$2 AND status IN
('queued','dispatched','running','waiting_local_directory')` — it **already covers `dispatched` and
`waiting_local_directory`** (claimed-but-not-started). So the cancel correctly fences an in-flight
claimed task; **no B2 change is needed**. This also *confirms the root cause*: the fencing is correct,
and the daemon's subsequent `StartAgentTask` legitimately matches 0 rows → the false failure that
Option A fixes. Thus **only B1 (atomic txn) is a real Option B change**; B2 is a verified no-op.

**Interaction:** Option B reduces the frequency of the race and prevents double-runs, but the daemon can
still observe a fenced row between its own claim and StartTask; therefore **Option A is required** to
eliminate the false failure, and B is defense-in-depth. Recommend shipping A first.

## 4. Focused tests (author with the owning edit)

- `internal/service/task_test.go`: `StartTask` returns `ErrTaskNotStartable` when the row exists in
  `cancelled` state; returns wrapped `pgx.ErrNoRows` when the row is truly absent; unchanged success path.
- `internal/handler/daemon_test.go`: `StartTask` handler → **409** "task superseded" for
  `ErrTaskNotStartable`; still **400** for other errors; **200** on success (extend existing
  `TestStartTask_*`).
- `internal/daemon/workdir_race_test.go` (existing race test at `:88` `TestRunTask_StartTaskCalledAfterWorkdirOnDisk`):
  add a case where `client.StartTask` returns 409 → caller returns `Skipped` with **no** FailTask/Classify
  call and no failure/circuit increment. Preserve the existing ordering invariant (StartTask after workdir on disk).
- `internal/service/…` RerunIssue: assert cancel+enqueue occur in one txn and prior `dispatched` rows are cancelled.

Run focused only: `go test ./internal/service ./internal/handler ./internal/daemon -count=1` + `go vet` (via /tmp caches).

## 5. Deploy / rollback checklist (Kiro deploy-prep; NO deploy here)

> Per steering: authoritative artifact deploy/rollback checklist. **No deploy until Kiro provides the
> fixed artifact.** The fix is Go server + daemon code (no schema/migration in Option A; Option B is
> code-only if `CancelAgentTasksByIssueAndAgent` already covers `dispatched`).

Pre-deploy (build/verify, an owning env with deps):
1. Apply A1–A5 (+ optional B1/B2) on a branch off the fixed HEAD Kiro designates.
2. `gofmt -l` clean on changed files; `go vet ./internal/service ./internal/handler ./internal/daemon`.
3. Focused tests (§4) PASS `-count=1`; then server-wide `go build ./...` and `go test ./... -count=1`.
4. Record artifact provenance: exact built binary **sha256**, source **commit SHA**, `go version`
   (go1.26.1), build flags, UTC, builder identity. Store as signed metadata (no secrets).
5. Confirm **no DB migration** required (Option A) — if Option B needs a query change, it is
   compile-time generated SQL, still no runtime migration.

Deploy (operator only; not this lane):
6. Snapshot the currently-running ORQ1 artifact (binary sha256 + commit) for rollback (see §6).
7. Deploy the new artifact via the standard runbook; health-gate on readiness (fail-closed).
8. Post-deploy verification: trigger the race scenario (RerunIssue while a prior task is claimed) →
   assert the daemon logs `start task skipped: superseded`, the task is NOT recorded failed, and no
   circuit/failure counter increments; the fresh rerun task starts normally.

Rollback (fast, reversible):
9. Redeploy the previously-snapshotted artifact (step 6) by its recorded sha256/commit; health-gate.
10. Verify prior behavior restored; capture logs. No DB rollback needed (no migration).
11. Record rollback provenance + reason.

## 6. ORQ1 artifact provenance verification (current)

Attempted/available (read-only, no host creds):
- Repo HEAD in this pane: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b` (build baseline candidate).
- **I have no access to the ORQ1 host filesystem, deploy manifest, or the running server/daemon binary**,
  and must not read secrets or deploy. Therefore the *currently deployed* ORQ1 artifact's provenance
  (which commit/binary sha256 it was built from, build time, builder) **cannot be verified from this
  pane** — it is a concrete external dependency.
- BLOCKER/OWNER: OmniRoute/ORQ1 operator + Principal (`w5:p9`) must publish the current ORQ1 artifact
  provenance (server + daemon binary sha256, source commit, build metadata) as signed non-secret
  metadata, so the rollback snapshot (step 6) has a verified baseline before any deploy.
- NEXT ACTION: obtain that provenance → record it here → then, once Kiro supplies the *fixed* artifact
  with its own provenance, the deploy/rollback checklist (§5) is executable by the operator.

## 7. Non-claims / limitations
- Design only — no source edited, no tests run, no deploy, no secret, no OmniRoute-internal change.
- `TaskResult.Skipped` may need a small additive field; if absent, use the sentinel-branch variant in `handleTask`.
- Option B2 verified: `CancelAgentTasksByIssueAndAgent` already cancels `queued/dispatched/running/waiting_local_directory` → only B1 (atomic txn) is a real change.
- Current ORQ1 artifact provenance is UNVERIFIED from this pane (no host access) → §6 blocker.
- The gateway "circuit" (OmniRoute) is unrelated to this daemon-side failure/skip path and is untouched.
