# L1 — Central Main Brain integration convergence (Wave A)

- agent: `Opus48#A`  ·  lane: `L1`  ·  task: `P0-L1-INTEGRATION`  ·  pane: `w6:p1`
- window: P0 six-hour · plan `P0_SIX_HOUR_EXECUTION_PLAN.md` · prompt `#L1`
- toolchain: `/home/ec2-user/goroot/go/bin/{go,gofmt}` (go1.26.1), node v22.23.1
- git HEAD at check-in: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- scope: owned central files only (see check-in lock). No out-of-lock edits. No deploy/inference/secret/commit.

## 1. Defects found & fixed (owned files, minimal)

1. **`internal/daemon/wakeup.go` — over-deleted tail helpers (build breaker).** The working-tree
   credential-removal dropped 4 **non-credential** core helpers still called by the retained
   `taskWakeupLoop`/`readTaskWakeupMessages`: `handleRuntimeProfilesChanged`, `signalTaskWakeup`,
   `taskWakeupURL`, `sleepWithContextOrRuntimeChange`. Restored all four **verbatim from HEAD**
   (`multica-auth-work/server/internal/daemon/wakeup.go` at HEAD, lines 379–end) plus the imports they
   use (`fmt`, `net/url`, `sort`; `strings` retained). The intentionally-removed credential-rotation
   code (`dispatchAndReportCredentialSessionDiscoveryEvent`, the `rotation` import, the
   credential-session-discovery handler) was **kept removed**. Verified deps still exist
   (`recoveryContext` daemon.go:455, `refreshWorkspaceRuntimeProfiles` daemon.go:1406,
   `RuntimeProfilesChangedPayload`).
2. **`internal/daemon/daemon.go:~3404` — orphan `d.observeCredentialPrepare` call.** Method never
   defined at HEAD nor working tree (surviving observability is `observeAgentBrainAdmission`). Removed
   the orphaned call and its now-unused `prepareStarted := time.Now()` timer; `execenv.Prepare` and
   its error handling preserved unchanged.
3. **`internal/daemon/brain/admission.go` — missing fail-closed gateway-bypass rejection.** The
   working tree removed the old "admit non-gateway-required task" block and rewrote the test
   (`g2a_test.go`: `TestGatewayAdmissionReadyAndLegacyBypass` → `...RejectsGatewayBypass`) to require
   bypass tasks to **error**, but never added the replacement guard. Added, right after
   `task.Request.Validate()` and before `validateFor`/readiness probe:
   `if !task.Request.GatewayRequired { return AdmissionDecision{}, fmt.Errorf(... ) }`. This makes a
   non-gateway task error before any route-policy decision or readiness call (preserves
   `checker.calls==1`); gateway-required tasks are unaffected.
4. **gofmt (task 5.1)** on owned files that were format-dirty: `config.go`, `brain/config.go`,
   `brain/recovery.go`, `brain/recovery_test.go`, `execenv/codex_sandbox.go`, `execenv/git.go`
   (+ the files edited above). `gofmt -l` over all owned paths is now clean.

> Note: `git diff HEAD` shows large deletions in these files; those are **pre-existing D-work** in the
> dirty tree, not my edits. My session edits are the minimal changes in items 1–4.

## 2. Validation (exact commands, from `multica-auth-work/server`, go1.26.1)

| Command | Result |
|---|---|
| `gofmt -l <all owned files/dirs>` | clean (empty) — PASS |
| `go build ./internal/daemon/ ./internal/daemon/brain/ ./internal/daemon/execenv/ ./pkg/agent/ ./cmd/multica/` | exit 0 — PASS |
| `go test ./internal/daemon/brain/ ./internal/daemon/execenv/ -count=1` | `ok` both — PASS (incl. `TestGatewayAdmissionReadyAndRejectsGatewayBypass`) |
| `go vet ./internal/daemon/brain/ ./internal/daemon/execenv/` | exit 0 — PASS |
| `git diff --check` (edited owned files) | exit 0 — PASS |
| `go vet ./internal/daemon/` (whole pkg incl. tests) | exit 1 — **BLOCKED by out-of-lock `daemon_test.go`** (see §3) |

## 3. FINDING / ESCALATION to manager (out of my exact lock)

- **`internal/daemon/daemon_test.go` — 742-line partial deletion; `newTestDaemon` undefined.** Defined
  at HEAD (`daemon_test.go:1265`), deleted in the working tree while ~10+ call sites (lines
  1128,1184,1313,1350,1378,1399,1423,1494,1536,1576,…) remain → the **daemon-package test build
  fails**, blocking targeted/server-wide daemon tests (tasks 5.2/5.3 for package `daemon`).
- This file is **not in any lane's exact lock** (L1's frozen lock is the 6 named non-test daemon files
  + `brain/**` + `execenv/**` + `cmd_daemon.go` + `models.go` + `go.mod`). Per the six-hour contract
  ("stop and escalate if a fix requires a file outside scope; one owner per file"), I did **not** edit
  it.
- **Requested manager action:** assign an owner / expand a lock for `daemon_test.go` (same
  restore-the-over-deleted-tail pattern as wakeup.go — likely restore `newTestDaemon` + companion test
  helpers from HEAD, keeping intentional test deletions). I can converge it immediately if my lock is
  expanded to include it.

## 4. Non-claims

- daemon-package **test** build/run not green (blocked by §3, out-of-lock). Owned **source** compiles
  (`go build` exit 0) and owned package tests (`brain`, `execenv`) pass.
- No server-wide `go build ./...` run in this checkout (Wave A owned-scope only; L8 owns full build).
- No deploy, inference, live run, secret access, commit/push, or reset/stash/revert/clean.
