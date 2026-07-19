# EV-CREDISO-4.4-RECORDFAIL — check-in + failure-path test evidence (agent-credential-isolation 4.4)

- Producer: implementation agent (Kiro/Opus-4.8). **Producer does NOT self-accept** — distinct review + Kiro TL adjudication required.
- Task 4.4: "Registrar/alertar a troca (aproveitar useSessionMonitor/isExpiringSoon)" — failure-path alert coverage.
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; offline.

## Golden Rule check-IN (before edit) — 2026-07-18T21:05:00Z

- Owned scope (disjoint, additive): create ONE new test file
  `multica-auth-work/server/internal/daemon/credential_session_record_failure_alert_test.go` + this evidence artifact.
- Will NOT edit any existing product or test file. Will NOT touch `internal/rotation/service.go` or any owned hotspot.
- Preflight conflict scan:
  - `git status`: target file `credential_session_record_failure_alert_test.go` is **absent** from the working
    tree / index → no conflict on my file. `internal/rotation/service.go` shows `M` (owned/in-flight by another
    stream) → I only READ it; I do not edit it.
  - `FILE_OWNERSHIP.md`: Codex1-only hotspots are `daemon.go/config.go/health.go/prodex*.go/l2_runtime.go/brain/**`
    + `execenv/{execenv,codex_home}.go` + `pkg/agent/models.go`. My new **test** file is not a listed hotspot and
    is disjoint/additive. Flagged for TL: the file lives in the `daemon` package (Codex1 integration area) but is a
    new test-only file per this assignment's explicit grant.
  - No OpenSpec checkbox/index edit; no git stage/commit/push; no DB/network/live provider/credential/env values.

## What the test exercises (real service, not a stub)

Builds a **real** `rotation.NewService(store, producerNoopDetector{}, producerSyntheticAuthenticator{})` where
`store` is the existing `producerSyntheticStore` wrapped so **only** `RecordRotation` returns a synthetic sentinel
error. A `daemon:credential_session_discovery` event (Status `exhausted`) is dispatched through
`dispatchAndReportCredentialSessionDiscoveryEvent`.

Service path executed: `CurrentAssignment → UpdateAccountStatus(exhausted) → selectNext(account-next) →
Logout(current) → Login(account-next) → WaitAuthenticated → Assign(agent-1, account-next) → RecordRotation (FAILS
with sentinel)`. The service returns `(Account{}, false, err)`.

### Assertions
- Operator log at `level=ERROR` with `alert=reassignment_failed` and message "automatic credential account
  reassignment failed"; carries `agent_id`, `provider=codex`, `tenant_id=workspace-1`, `previous_account_id=account-current`.
- **No** success alert "automatic credential account reassignment completed".
- **No** `next_account_id=` success metadata.
- Synthetic record-rotation sentinel, credential home/config sentinels, and raw error text are **absent** from logs.
- Auth fixture recorded `login:account-next` → proves the real service ran the rotation up to the record step.

### Documented (unchanged) non-atomic behavior
`onExhaustionLocked` persists `Assign` **before** `RecordRotation` (service.go). When `RecordRotation` fails the
service returns an error, yet the assignment has already been written and is **not rolled back**. The test records
this by asserting `store.assignment("agent-1") == "account-next"` after the failed record — documenting the current
non-atomic assignment-before-record behavior **without changing it and without claiming any rollback**.

## Check-OUT / results — 2026-07-18T21:12:00Z

New file (only artifact created besides this evidence):
`multica-auth-work/server/internal/daemon/credential_session_record_failure_alert_test.go`
- SHA-256 `5b4d82caba027d4dd6b2650d9cd5a2ad78ebdd07504060fd84a415d40be739f5`

Pinned offline results (`/home/dataops-lab/go-sdk/bin/go`, `go1.26.4`, `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`):
- `gofmt -l` (GOROOT `/home/dataops-lab/go-sdk`): empty → formatted.
- `go vet ./internal/daemon/`: exit 0.
- `go test ./internal/daemon/ -run TestDispatchCredentialSessionRecordRotationFailureAlertsWithoutSuccessOrLeak -v -count=20`:
  **20/20 PASS**, `ok` — named test genuinely executes (20 `RUN` lines).
- Same test `-race -v -count=20`: **20/20 PASS**, no data races, `ok` (1.214s).

Scope compliance: only the new test file + this evidence were written. No existing product/test file edited;
`internal/rotation/service.go` and all owned hotspots untouched; no OpenSpec checkbox/index change; no
git stage/commit/push; no DB/network/live-provider/credential/env-value access.

Disposition: record as `EV-CREDISO-4.4-RECORDFAIL` (producer evidence). **Not self-accepted.** Requires a
distinct independent reviewer + Kiro TL adjudication before task 4.4 is considered covered.
