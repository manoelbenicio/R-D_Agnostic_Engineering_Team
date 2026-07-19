# EV-CREDISO-4.4 — fresh independent reviewer reproduction (alerting/recording slice)

Fresh independent reproduction of agent-credential-isolation task 4.4
("Registrar/alertar a troca"). This artifact is produced by a **distinct
reviewer** (GLM52-auth-QA), assigned by the TL (Kiro/Opus-4.8) in
AGENT_LEDGER row `credio-4.4-remediation-decision` after the prior review
(`evidence/credential-isolation-reassignment-alerting-review.md`,
`EV-CREDISO-4.4-REVIEW`) was TECH PASS / CONTRACT REJECT for four evidence-
contract gaps. This artifact closes those four gaps and is submitted for TL
adjudication. It is **not** self-acceptance: task 4.4 remains OPEN until Kiro
adjudicates.

## Distinct reviewer identity (contract gap 4 closed)

- **Reviewer:** GLM52-auth-QA (Herdr pane `w4:p3`, workspace `w4`).
- **Producer (implementation):** Codex/root (per ledger row `cred-iso-4.4-alerting`,
  check-in `2026-07-18T20:05:36Z`).
- **Prior reviewer:** Codex/root — REJECTED for non-distinct identity (same actor
  as producer) plus three other contract gaps.
- **Adjudicator:** Kiro/Opus-4.8 (TL) — owns EVIDENCE_INDEX acceptance and the
  task checkbox; does not double as reviewer.
- **Distinctness basis:** GLM52-auth-QA is a different agent identity from the
  producer (Codex/root) and the prior reviewer (Codex/root). The TL explicitly
  assigned GLM52-auth-QA because Codex56#D (the only other candidate) was
  usage-limited and producer-adjacent, and the TL must stay separate from
  adjudication. No historical identity was relabeled.

## Provenance (contract gap 3 closed)

- **Host:** `manoelneto-laptop` (WSL2, Linux amd64).
- **Toolchain:** `/home/dataops-lab/go-sdk/bin/go` →
  `go version go1.26.4 linux/amd64`; `GOTOOLCHAIN=local`.
- **Repository commit (HEAD):** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
  ("fix(runtimes): stabilize CLI update detection for dev builds",
  2026-07-17 10:53:28 -0300). Matches the prior review's cited commit.
- **Review execution window:** 2026-07-18T20:35:00Z through 2026-07-18T20:55:00Z
  UTC (start/end timestamps bracket the command runs below).
- **Offline environment:** `GOPROXY=off GOSUMDB=off`; for the full daemon
  package run only, `APP_ENV=test DATABASE_URL=://offline-invalid` (to force
  DB-gated tests to skip). The focused alert tests pass with plain
  `GOPROXY=off GOSUMDB=off` (verified); the extra env is only needed for the
  full package. No proxy contact; module cache slot-isolated.
- **No credential, auth home, session file, token, environment secret,
  database, network, live provider/daemon/CLI, or multi-node state was read
  or used.**

## AB-REQ and acceptance mapping (contract gap 2 closed)

- **OpenSpec spec:** `agent-credential-isolation/specs/agent-credential-isolation/spec.md:72-84`,
  requirement "Rotação automática ao esgotar conta (Fase 2)".
- **OpenSpec acceptance scenario:** `spec.md:82-84` "Sem conta disponível" —
  "o sistema sinaliza o esgotamento (alerta) sem sobrescrever credenciais".
  This is the **alert condition** task 4.4 implements.
- **OpenSpec task:** `agent-credential-isolation/tasks.md:28` "4.4
  Registrar/alertar a troca (aproveitar useSessionMonitor/isExpiringSoon)" —
  unchecked `[ ]`.
- **AB-REQ mapping (REQUIREMENTS.md):**
  - **Primary — AB-REQ-12** ("Credential+quota lifecycle: refresh proativo,
    single-flight, classify 401/403, quarantine, quota/reset"; ORR; spec
    scenario "Selected account token has expired"; tasks 4.x, 8.5; owner
    Codex2; evidence EV-G4-05). The "Sem conta disponível" alert is the
    quota-lifecycle exhaustion signal; task 4.x subsumes 4.4.
  - **Secondary — AB-REQ-21** ("Secret-safe evidence: redige
    secrets/cookies/prompts/tool payloads/repo content/reasoning"; CLE;
    spec scenario "Upstream error includes an authorization value"; tasks
    6.3, 8.3; owner Codex4/3; evidence EV-G4-03). The alert path's
    no-credential-leak property (whitelist attrs + bounded error class +
    `redact.SanitizeSlogAttr`) directly satisfies this.
  - **Tertiary — AB-REQ-38** ("Operational handover: owners nomeados,
    dashboards/alerts, backup/restore, rotation, upgrade/rollback, escala";
    BCO; spec scenario "Provider-wide throttling occurs"; tasks 6.3, 6.4,
    6.6, 9.7; owner Codex4; evidence EV-G4-07). The operator-visible
    WARN/ERROR alerts are the operational handover signal.
- **Honest gap:** no single AB-REQ is dedicated solely to "alert/record the
  rotation switch"; task 4.4 is an implementation facet of the spec's "Sem
  conta disponível" scenario. AB-REQ-12 is the closest lifecycle owner; the
  alert and no-leak properties cross-cover AB-REQ-21/38. This is a mapping,
  not a perfect 1:1 fit; recorded for TL adjudication.

## Proposed EVIDENCE_INDEX entry (contract gap 1 — PROPOSED for TL)

The TL owns `EVIDENCE_INDEX.md`; this review does not edit it. The proposed
entry for Kiro to add (mirroring the sibling `EV-CREDISO-4.1/4.2/5.2` row
format) is:

```text
| EV-CREDISO-4.4 | agent-credential-isolation task 4.4 (alerting/recording slice) | PENDING (TL adjudication) | `evidence/credential-isolation-4.4-fresh-review.md` | Operator-visible success/no-account/failure/no-op alert path; ErrNoAccountAvailable → WARN `alert=no_account_available`; no credential/raw-error leak. Fresh distinct-reviewer (GLM52-auth-QA) reproduction: focused 6 tests ×20 = 120 PASS/0 FAIL, race exit 0, vet exit 0, full daemon offline exit 0, 8-file SHA-256 manifest revalidates. Maps to spec "Sem conta disponível" (spec.md:82-84), AB-REQ-12 (+21/38). Artifact SHA-256: see the final SHA reported by the reviewer to the TL (self-referential loop prevents embedding the final SHA inside this proposed entry). Task 4.4 stays OPEN until TL adjudication. |
```

## Alert condition verification

The acceptance scenario "Sem conta disponível" (spec.md:82-84) requires the
system to "sinaliza o esgotamento (alerta) sem sobrescrever credenciais". Both
clauses are verified:

- **Alerta (signal):** `wakeup.go:345-347` —
  `if errors.Is(err, rotation.ErrNoAccountAvailable) {
  d.logger.Warn("rotation: automatic credential account reassignment
  unavailable", attrs...) }` with `attrs` containing
  `alert=no_account_available` (via `credentialSessionReassignmentErrorClass`
  at `wakeup.go:364-376`). The test
  `TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors/no_account`
  (`credential_session_alert_test.go:75`) asserts `level=WARN`,
  `alert=no_account_available`, `provider=codex`, `tenant_id=workspace-1`.
- **Sem sobrescrever credenciais (no overwrite):** `service.go:97-122`
  `onExhaustionLocked` selects a replacement **before** logout; on
  `ErrNoAccountAvailable` it returns without calling `Logout` or `Login`, so
  the current assignment and authenticated session are undisturbed. The
  sibling 5.3 test
  `TestCredentialIsolationTask53AutomaticRotation/no_same-provider_same-tenant_candidate_fails_closed`
  (`credential_rotation_task53_test.go:59-98`) asserts `assignment == "current"`
  (preserved) and `len(auth calls) == 0` (no logout/login). No credential
  overwrite.

## Behavior coverage (all four alert outcomes)

| Outcome | Source anchor | Test anchor | Verified log content |
| --- | --- | --- | --- |
| Success (reassignment completed) | `wakeup.go:357-361` WARN "completed" | `alert_test.go:17` (ReportsReassignmentWithoutCredentials) | agent/provider/tenant/previous/next account + `reason=quota_exhausted_reactive`; no sentinel |
| No account available | `wakeup.go:345-347` WARN "unavailable" | `alert_test.go:75` (no_account subtest) | `alert=no_account_available`; no sentinel |
| Other failure (auth error) | `wakeup.go:349` ERROR "failed" | `alert_test.go:76` (auth_error subtest) | `alert=reassignment_failed`; no sentinel |
| No-op (stale/duplicate/future) | `wakeup.go:352-354` DEBUG | `alert_test.go:109` (NoopIsDebugOnly) | `level=DEBUG` + "produced no reassignment"; no WARN/ERROR |

Plus the task-4.3 bridge tests (unchanged) at `monitor_test.go:14/57/79`
(forwarding, error preservation, malformed/unavailable/unrelated rejection).

## Record-before-success verification

`service.go:150` `Assign` → `:156` `RecordRotation` → `:157` return on record
failure → `:159` return success. The daemon bridge sets `Reassigned` and
`NextAccountID` only after `ReassignDiscoverySession` returns without error
(`monitor.go:88-107`). The WARN completion alert (`wakeup.go:361`) therefore
cannot be reached before `RecordRotation` succeeds. This is static
control-flow evidence plus the executed synthetic `TestServiceOnExhaustionRotatesHappyPath`
(`service_test.go:10`); the focused alert tests do not inject a `RecordRotation`
failure end-to-end, so this is **not** a DB-backed persistence claim.

## Exact reproduction commands and results

Working directory: `multica-auth-work/server`. Toolchain:
`/home/dataops-lab/go-sdk/bin/go`. Every command reproduced exactly as the
producer artifact specifies.

### Named execution proof (verbose single run)

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 -v ./internal/daemon -run 'TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

Result: exit 0; `ok github.com/multica-ai/multica/server/internal/daemon 0.031s`.
Transcript reproduces the producer artifact's `:49-72` exactly — 6 top-level
tests + 5 subtests PASS, including `...AlertsWithoutLeakingErrors/no_account`
and `.../auth_error`.

### Focused ×20 (non-zero count)

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=20 ./internal/daemon -run 'TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

Result: exit 0; `ok ... 0.059s`. Verbose re-run: **120 `--- PASS` / 0 `--- FAIL`**
(6 top-level tests × 20), 220 `=== RUN` lines (includes 5 subtests × 20 = 100).
The `no_account` and `auth_error` alert subtests are present in every
iteration. **Non-zero count confirmed.**

### Focused race

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -race -count=1 ./internal/daemon -run 'TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

Result: exit 0; `ok ... 1.083s` (race clean).

### Vet

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go vet -tags=offline ./internal/daemon
```

Result: exit 0, no diagnostics.

### Full offline daemon package

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 ./internal/daemon
```

Result: exit 0; `ok ... 23.437s` (consistent with the producer's 19.957s /
prior review's 20.397s; the small variance is machine load, not a defect).

### Formatting

```sh
/home/dataops-lab/go-sdk/bin/gofmt -d internal/daemon/wakeup.go internal/daemon/credential_session_monitor.go internal/daemon/credential_session_monitor_test.go internal/daemon/credential_session_alert_test.go
```

Result: exit 0, no diff (all 4 task-4.4 files gofmt-clean).

## Source SHA-256 manifest (8 files — full alert + record + boundary surface)

Hashes computed only over the repository paths below. No external home,
credential, auth, session, token, environment-secret, or live service path
was enumerated, read, copied, or hashed. The first 4 (task-4.4 daemon files)
match the producer artifact's 4-file manifest exactly; the last 4 (rotation
record + boundary) match the prior review's 8-file manifest exactly.

```text
71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730  multica-auth-work/server/internal/daemon/wakeup.go
936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2  multica-auth-work/server/internal/daemon/credential_session_monitor.go
5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2  multica-auth-work/server/internal/daemon/credential_session_monitor_test.go
8e369510e814ff2f5743bf60dc76ee3074110a98da3daa172301aae1fda12bea  multica-auth-work/server/internal/daemon/credential_session_alert_test.go
f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0  multica-auth-work/server/internal/rotation/service.go
989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1  multica-auth-work/server/internal/rotation/service_test.go
c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832  multica-auth-work/server/internal/rotation/discovery_reassignment.go
d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2  multica-auth-work/server/internal/rotation/discovery_reassignment_test.go
```

The manifest was revalidated with `sha256sum -c` against these repository
paths after the artifact was written. The 4 rotation hashes match the
GLM52-auth-QA task-4.3 and task-5.3 review manifests (consistent baseline
across the three credential-isolation lanes); `wakeup.go` and
`credential_session_monitor.go` carry the 4.4 alerting changes (distinct
from the 4.3 artifact's pre-alerting hashes), consistent with the 4.4
implementation timeline.

## No-secret / no-DB / no-live verification

- **No-secret in test files:** `credential_session_alert_test.go` has zero
  `os.Getenv`/`os.ReadFile`/`os.UserHomeDir`/`DATABASE_URL`/real-home/
  `CODEX_HOME`/`XDG_*`/`live`/`network`/`t.Setenv`/`t.TempDir` references
  (precise word-boundary grep). All stores/authenticators/emitters are
  in-process synthetic (`syntheticDiscoveryReassigner`,
  `credentialSessionAlertTestLogger` into a `bytes.Buffer`). The two
  synthetic sentinels (`synthetic-credential-sentinel`,
  `synthetic-error-credential-sentinel`) are deliberately placed in
  `Account.HomeDir`/`ConfigDir`/`LastError` and a provider-style error, then
  asserted **absent** from the captured log (`alert_test.go:62-64`,
  `:102-104`).
- **No-DB:** the 6 alert tests are all synthetic (none `t.Skip` on
  `DATABASE_URL`; none match a DB-gated pattern). DB-gated tests in the
  daemon/rotation packages (`rotation_e2e_test.go`, `store_pg_test.go`,
  several `*_test.go` in daemon) skip without `DATABASE_URL`; the full
  package run uses `DATABASE_URL=://offline-invalid` to force-skip. No
  Postgres, Redis, or SQLite contact.
- **No-live:** no real provider, daemon, CLI, WebSocket, or agent process
  was started. `dispatchAndReportCredentialSessionDiscoveryEvent` is
  invoked directly with a synthetic `protocol.Message`.

## Explicit non-claims

This reproduction does **not** claim:
- live WebSocket delivery or live vendor login/logout behavior;
- PostgreSQL or any database-backed persistence execution (the
  record-before-success proof is static control-flow + synthetic happy-path);
- transactionality between `Assign` and `RecordRotation`;
- multi-node CAS, distributed locking, or cross-process deduplication;
- frontend `useSessionMonitor` UI delivery (the spec task mentions
  `useSessionMonitor/isExpiringSoon`, but the offline tests cover the
  daemon-side alert/record path only, not the frontend);
- external notification delivery (Slack/email/PagerDuty etc.) beyond the
  daemon structured log;
- production deployment, cutover, tier acceptance, or full task 4.3/4.4
  completion;
- that the proposed EVIDENCE_INDEX entry has been added (it is PROPOSED for
  the TL, who owns that file);
- TL adjudication/acceptance — this is a reviewer reproduction submitted
  for adjudication, not a self-acceptance.

## Verdict

**ACCEPT (fresh distinct-reviewer reproduction; contract-complete; TL
adjudication pending).** All four prior evidence-contract gaps are closed:
(1) proposed EV-index entry for the TL to add; (2) AB-REQ-12 (+21/38) and
spec "Sem conta disponível" acceptance mapping; (3) full provenance (host,
go1.26.4, commit `b6571299`, UTC window, offline env); (4) distinct reviewer
identity GLM52-auth-QA (≠ producer Codex/root, ≠ prior reviewer Codex/root).
The alert condition is verified: `ErrNoAccountAvailable` → WARN
`alert=no_account_available` with no credential/raw-error leak and no
credential overwrite. All 5 reproduction commands exit 0 (focused ×20 =
120 PASS/0 FAIL, race, vet, full daemon, gofmt). 8-file SHA-256 manifest
revalidates. Task 4.4 stays `[ ]` (OPEN) until Kiro adjudicates and adds the
EVIDENCE_INDEX entry. No product/test/STATE/EVIDENCE_INDEX/OpenSpec edit by
this reviewer; no staging/commit/push.
