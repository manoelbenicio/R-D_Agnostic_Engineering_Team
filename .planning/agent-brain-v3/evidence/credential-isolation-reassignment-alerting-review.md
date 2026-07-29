# EV-CREDISO-4.4 independent QA review

Review target: agent-credential-isolation task 4.4 and
`.planning/agent-brain-v3/evidence/credential-isolation-reassignment-alerting.md`.

Review check-in:
`.deploy-control/Codex-root__CREDISO-4.4-QA__20260718T201551Z_START.md` at
2026-07-18T20:15:51Z, before QA test execution.

Reviewer provenance: Codex/root; host `manoelneto-laptop`; repository commit
`b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`; Go
`go1.26.4 linux/amd64`; review execution window 2026-07-18T20:15:51Z through
2026-07-18T20:19:46Z UTC. All Go commands used
`/home/dataops-lab/go-sdk/bin/go`, `GOTOOLCHAIN=local`, `GOPROXY=off`,
`GOSUMDB=off`, `APP_ENV=test`, `DATABASE_URL=://offline-invalid`, and
`-tags=offline`.

## Verdict

> [!CAUTION]
> **REJECT at the evidence-contract hard gate; technical implementation QA PASS.**

The alerting implementation and its synthetic tests reproduce successfully, including
the complete daemon package and complete daemon race suite. It is not accepted because
the submitted evidence chain does not satisfy the binding `EVIDENCE_CONTRACT.md`:

1. `EV-CREDISO-4.4` has no entry in the current immutable
   `.planning/agent-brain-v3/EVIDENCE_INDEX.md`.
2. The submitted artifact does not map task 4.4 to an `AB-REQ` and acceptance ID,
   as evidence-contract rule 3 requires.
3. The submitted artifact records a date and implementer but omits exact host,
   Go version, repository commit, and exact UTC execution timestamps required by
   evidence-contract rule 1.
4. The implementation check-in records `started_at=2026-07-18T20:05:36Z`, before
   source mtimes around 20:08Z, but the only current check-in file is its final DONE
   form with a 20:12Z mtime. The current checkout therefore cannot independently
   establish that a distinct immutable START snapshot existed before execution.
5. Both the submitted artifact and this review identify the actor as `Codex/root`.
   The fresh rerun is independently executed in time, but the stored provenance cannot
   establish a distinct reviewer identity. Kiro TL must perform or attribute the
   independent adjudication.

Per the review assignment, no index, checkbox, STATE, ledger, OpenSpec, product, or
test file was edited. Kiro owns remediation and adjudication. This review does not
self-accept task 4.4.

## Submitted artifact integrity

The claimed abbreviated hash `45a0bf08…15a6` reproduces exactly:

```text
45a0bf0820aa66e8504ecfacc8afe8644dca0a618b043232c3b7daa5cfa015a6  .planning/agent-brain-v3/evidence/credential-isolation-reassignment-alerting.md
```

Its four-file manifest also reproduces against the current source. The implementation
check-in exists with SHA-256:

```text
2f151c9a75f9be1a89ac3bde6c22b09e5afc1eb99d7b3bc2fd61f13220e62462  .deploy-control/Codex-root__CREDISO-4.4__20260718T200536Z.md
```

## Current source SHA-256 manifest

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

## Behavior inspection

### Success

`dispatchAndReportCredentialSessionDiscoveryEvent` builds a whitelist of assignment
metadata at `server/internal/daemon/wakeup.go:336`. On a successful reassignment it
adds only `next_account_id` and fixed reason `quota_exhausted_reactive`, then emits
WARN at `server/internal/daemon/wakeup.go:357-361`.

The outcome contains only handled/reassigned flags plus agent, previous/next account,
provider, and tenant identifiers at
`server/internal/daemon/credential_session_monitor.go:45-56`. Returned account homes,
config directories, last errors, and session IDs cannot enter this outcome.

### No account available

`rotation.ErrNoAccountAvailable` maps to `alert=no_account_available` and WARN at
`server/internal/daemon/wakeup.go:342-347`. It does not append the raw error.

### Other errors

All other errors emit ERROR at `server/internal/daemon/wakeup.go:349`. Error values
are converted to one of `deadline_exceeded`, `canceled`, `service_unavailable`, or
`reassignment_failed` at `server/internal/daemon/wakeup.go:364-376`; no raw error
string is passed to the logger.

### No-op

A handled event with `Reassigned=false` emits DEBUG only at
`server/internal/daemon/wakeup.go:352-354`; it does not raise WARN or ERROR.

### No credential or raw-error logging

The successful alert test places one synthetic sentinel in `Account.HomeDir`,
`ConfigDir`, and `LastError` at
`server/internal/daemon/credential_session_alert_test.go:17-28`, then proves the
captured log omits it at lines 48-64. The error test embeds a second synthetic
sentinel in a provider-style error at lines 67-76 and proves it is absent at lines
96-104. This is backed by the static outcome and attribute whitelists above; logger
sanitization at lines 135-139 is supplemental rather than the only protection.

### Record before success alert

The rotation service calls `Store.Assign` at `server/internal/rotation/service.go:150`,
then `Store.RecordRotation` at line 156, and returns success only at line 159. A record
failure returns at line 157. The daemon bridge sets `Reassigned` and `NextAccountID`
only after `ReassignDiscoverySession` returns without error at
`server/internal/daemon/credential_session_monitor.go:88-107`. The WARN completion
alert therefore cannot be reached before `RecordRotation` succeeds.

The focused alert tests do not inject a `RecordRotation` failure end to end; this
ordering proof is static control-flow evidence plus the executed synthetic service
happy-path test, not a DB-backed persistence claim.

### Provider and tenant boundaries

The bridge copies provider and workspace/tenant directly from the signed event at
`server/internal/daemon/credential_session_monitor.go:83-97`. The task-4.3 service
continues to reject a current assignment outside the canonical provider or tenant at
`server/internal/rotation/discovery_reassignment.go:65-70`. The forwarding and both
boundary-rejection subtests were executed successfully.

## Named test existence and genuine execution

Source declarations exist at:

- `server/internal/daemon/credential_session_alert_test.go:17`
- `server/internal/daemon/credential_session_alert_test.go:67`
- `server/internal/daemon/credential_session_alert_test.go:109`
- `server/internal/daemon/credential_session_monitor_test.go:14`
- `server/internal/daemon/credential_session_monitor_test.go:57`
- `server/internal/daemon/credential_session_monitor_test.go:79`

Command:

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 -v ./internal/daemon -run '^TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

Result: exit 0, package time 0.032s. The verbose transcript genuinely executed:

```text
TestDispatchAndReportCredentialSessionDiscoveryEventReportsReassignmentWithoutCredentials PASS
TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors PASS
  no_account PASS
  auth_error PASS
TestDispatchAndReportCredentialSessionDiscoveryEventNoopIsDebugOnly PASS
TestDispatchCredentialSessionDiscoveryEventForwardsExpiredObservation PASS
TestDispatchCredentialSessionDiscoveryEventPreservesReassignmentErrors PASS
TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable PASS
  malformed_payload PASS
  service_unavailable PASS
  unrelated_event PASS
```

## Focused repeated, race, and vet results

Focused ×20:

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=20 ./internal/daemon -run '^TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
ok  github.com/multica-ai/multica/server/internal/daemon  0.048s
```

Focused uncached race:

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -race -count=1 ./internal/daemon -run '^TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
ok  github.com/multica-ai/multica/server/internal/daemon  1.119s
```

Vet:

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go vet -tags=offline ./internal/daemon
```

Result: exit 0 with no diagnostics.

## Full daemon results

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 ./internal/daemon
ok  github.com/multica-ai/multica/server/internal/daemon  20.397s
```

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -race -count=1 ./internal/daemon
ok  github.com/multica-ai/multica/server/internal/daemon  22.351s
```

## Supporting rotation execution

After one invocation from the module parent failed before test execution with
`cannot find main module`, the exact command was rerun from
`multica-auth-work/server`:

```text
env APP_ENV=test DATABASE_URL='://offline-invalid' GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 -v ./internal/rotation -run '^(TestReassignDiscoverySessionRejectsAssignmentBoundary|TestServiceOnExhaustionRotatesHappyPath)$'
```

Result: exit 0, package time 0.013s. Executed names:

```text
TestReassignDiscoverySessionRejectsAssignmentBoundary PASS
  different_provider PASS
  different_tenant PASS
TestServiceOnExhaustionRotatesHappyPath PASS
```

## Formatting and diff hygiene

`gofmt -d` over the four task-4.4 daemon files and `git diff --check` over the
task-4.3/4.4 shared sources both exited 0 with no output.

## Scope and non-claims

All executed tests were offline and synthetic. No credential, authentication home,
session file, token, environment secret, database, external network, provider, or live
daemon was inspected or used.

This review does not claim live WebSocket delivery, live vendor behavior, PostgreSQL
execution, transactionality, multi-node CAS, distributed deduplication, frontend
`useSessionMonitor` delivery, external notification delivery, production deployment,
full task 4.3 completion, or Kiro/TL acceptance. The technical slice passes; the
evidence-contract gaps above block an ACCEPT verdict.
