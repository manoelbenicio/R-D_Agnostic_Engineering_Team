# FIX build/test — baseline + standby

- agent: Codex56#B · pane: w7:p4 · lane: build-test · task: FIX-BUILD-TEST
- repo HEAD: `a6d5098` (working tree intentionally dirty; unrelated work preserved)
- toolchain: `/home/ec2-user/goroot/go/bin/go` = `go version go1.26.1 linux/amd64`
- caches (root disk tight): `GOCACHE=/tmp/base-gocache GOTMPDIR=/tmp/base-gotmp GOMODCACHE=/tmp/f4-gomodcache`; /tmp 2.4G free after run
- constraints: no edits / deploy / inference / secrets / commit; `live_runs=false`

## BASELINE (pre-fix) — recorded 2026-07-23T00:09Z — currently GREEN

Exact commands (run from `multica-auth-work/server`) and results:

| Command | Result | Exit |
|---|---|---|
| `go build ./internal/service/... ./internal/handler/... ./pkg/db/... ./internal/daemon/...` | compiles clean | 0 |
| `go vet ./internal/service/ ./internal/daemon/` | clean | 0 |
| `go test ./internal/service/ -count=1 -timeout 120s` | `ok ... 0.022s` | 0 |
| `go test ./internal/daemon/ -count=1 -timeout 180s` | `ok ... 15.048s` | 0 |
| `go test ./internal/handler/ -count=1 -timeout 150s` | `ok ... 0.065s` | 0 |
| `go test ./pkg/db/... -count=1 -timeout 90s` | `? generated [no test files]` (no tests) | 0 |

Note: these package suites passed without a live DB (no Postgres/Redis running); DB-integration paths are
either self-contained, mocked, or skipped in this environment. `pkg/db/generated` has no test files (not a
PASS, just no tests to run).

## STANDBY — authoritative post-fix run (BLOCKED until fix lands)

- **Blocker:** the authoritative artifact build + focused `internal/service` and `internal/daemon` (and
  affected handler/db) re-run is deferred until the fix lands/integrates on the orq2 target tree. The fix
  has not been signaled as landed to this lane.
- **Owner:** manager (`w5:p1`) / the fix owner.
- **Next action (on "fix landed" signal):** re-run, from `multica-auth-work/server` with the same /tmp
  caches:
  - `go build ./...` (authoritative artifact)
  - `go test ./internal/service/ ./internal/daemon/ -count=1` (focused)
  - plus affected handler/db packages if the fix touches them
  Then append post-fix results here (exact command + exit code + any per-package failure vs this baseline)
  and check out with evidence.

## Non-claims

- Baseline only; the authoritative post-fix build/test has NOT been run (fix not landed).
- No product code modified; no deploy/inference/secret; `pkg/db/generated` has no tests.

## POST-FIX AUTHORITATIVE VALIDATION — 2026-07-23T00:18Z — result: PASS

### Artifact / source provenance (race fix = working tree at HEAD `a6d5098`, uncommitted)

The race fix is in the working tree (HEAD unchanged `a6d5098`; last commits are pre-shutdown autosaves).
`git diff --stat` of the race-fix source:
- `internal/service/task.go` — +88 (task-complete finalization guard)
- `internal/service/task_complete_race_test.go` — +58 (new)
- `internal/daemon/daemon.go` — 582 changed (−500/+82; runTask/workdir ordering + legacy removal)
- `internal/daemon/workdir_race_test.go` — ±8

sha256 (source provenance of the exact artifact validated):
```
dc184cb0f8d2122c2d199249bc1f3c7c7dfeb7df56dfaad1c13614d7e3656305  internal/service/task.go
41b574c33934b5421f58306181733fbe1e8485581691e15ab591d587a75da0f0  internal/service/task_complete_race_test.go
981830478f8a40e8f955366b67a8d93ae444091139833375f4184ddcbce21e0e  internal/daemon/daemon.go
903bf0ca287d7ecc0bc0621b7136d2e7a6688a67d078f2186b0187d819ed8af7  internal/daemon/workdir_race_test.go
```

### Commands + results (from `multica-auth-work/server`, /tmp caches)

| Command | Result | Exit |
|---|---|---|
| `go build ./...` (authoritative server build) | builds clean | 0 |
| `go test ./internal/service/ -run '^(TestCompleteTask_AlreadyFinalized\|TestFailTask_AlreadyFinalized\|TestTaskFailureClassifiers\|TestEmitPersistSpanAfterPersistenceMetadataOnly\|TestEmitPersistSpanOrderingConsumesPersistedRow)$' -count=1 -v` | 5/5 PASS | 0 |
| `go test ./internal/daemon/ -run '^(TestHandleTask_DoesNotCallStartTaskItself\|TestRunTask_StartTaskCalledAfterWorkdirOnDisk\|TestHandleTask_KeepsEnvRootActiveAcrossCompletion)$' -count=1 -v` | 3/3 PASS | 0 |

**PASS** — authoritative server build green; all 8 targeted race-fix tests (5 service task-complete +
3 daemon workdir-ordering) pass against the exact deployed artifact. No baseline suites re-run (targeted
only — no duplicate baseline testing).

### Limitation / non-claim (post-fix)

- **`-race` data-race detector could NOT be run:** `go env CGO_ENABLED=0` and no C compiler (`gcc`/`cc`)
  is available on this host, so the detector build is unavailable. The targeted tests validate the fix's
  **functional ordering invariants** (finalize-once, StartTask-after-workdir-on-disk) but do NOT constitute
  a `-race` detector pass. Owner/next action for a detector-level gate: run
  `CGO_ENABLED=1 go test -race ./internal/service/ ./internal/daemon/` on a host with a C toolchain.
- No product code modified by this validation; no deploy/inference/secret.

