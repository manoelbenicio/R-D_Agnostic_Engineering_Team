# Credential isolation task 4.4 — automatic reassignment recording and alerting evidence

Evidence ID: `EV-CREDISO-4.4`

Date: 2026-07-18. Implementer: Codex/root. This artifact is implementation evidence for independent review; it is not self-acceptance.

## Scope and shared-diff boundary

The task was claimed in `.deploy-control/Codex-root__CREDISO-4.4__20260718T200536Z.md` and `.planning/agent-brain-v3/AGENT_LEDGER.md` after verifying that no active row claimed the target files.

The existing task-4.3 shared diff in `server/internal/daemon/wakeup.go` already recognized `daemon:credential_session_discovery`, copied its payload, and dispatched it asynchronously. Task 4.4 preserved that behavior and changed only the discarded result path plus its narrow bridge result:

- `server/internal/daemon/wakeup.go:275-287` keeps the non-blocking two-minute dispatch and now calls the reporting wrapper instead of discarding `(handled, error)`.
- `server/internal/daemon/wakeup.go:325-377` emits operator-visible completion, unavailable-pool, failure, and no-op records. It logs assignment metadata and a bounded error class only; it never logs raw errors, sessions, credential homes, config paths, or credential material.
- `server/internal/daemon/credential_session_monitor.go:45-108` returns a non-secret outcome containing the existing provider/tenant request boundary and the selected account ID. The existing `(bool, error)` wrapper remains compatible.
- `server/internal/daemon/credential_session_alert_test.go:17-140` is the new focused offline test surface. The pre-existing task-4.3 bridge tests remain in `credential_session_monitor_test.go` and continue proving exact provider/tenant forwarding.

No rotation-store or database code changed. Durable recording remains `server/internal/rotation/service.go:150-159`: assignment is followed by `Store.RecordRotation`, and only then does the service return success for the daemon completion alert. A record failure is returned and becomes a failure alert rather than a false completion.

## Behavior and security properties

- Successful automatic reassignment is logged at WARN with `agent_id`, `provider`, `tenant_id`, `previous_account_id`, `next_account_id`, and the fixed reason `quota_exhausted_reactive`.
- `rotation.ErrNoAccountAvailable` becomes a WARN alert with `alert=no_account_available`.
- Other failures become an ERROR alert with a bounded class (`deadline_exceeded`, `canceled`, `service_unavailable`, or `reassignment_failed`). Raw error text is deliberately excluded.
- Stale, duplicate, future, or otherwise non-reassigning discovery is DEBUG-only and does not raise a false operator alert.
- Provider and tenant values passed to reassignment are unchanged from the signed discovery event; task 4.4 does not add selection logic or broaden the task-4.3 boundary checks.
- Tests place a synthetic credential sentinel in `Account.HomeDir`, `ConfigDir`, `LastError`, and an authenticator-style error. The captured operator output contains none of it.

## Current SHA256 manifest

```text
71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730  server/internal/daemon/wakeup.go
936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2  server/internal/daemon/credential_session_monitor.go
5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2  server/internal/daemon/credential_session_monitor_test.go
8e369510e814ff2f5743bf60dc76ee3074110a98da3daa172301aae1fda12bea  server/internal/daemon/credential_session_alert_test.go
```

## Genuine offline execution

Working directory: `multica-auth-work/server`. Toolchain: `/home/dataops-lab/go-sdk/bin/go`. Every Go command used `GOPROXY=off GOSUMDB=off`; offline package gates additionally used `APP_ENV=test DATABASE_URL=://offline-invalid`. No credential, PostgreSQL, Redis, external network, or live service was used.

### Named execution proof

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 -v ./internal/daemon -run 'TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

```text
=== RUN   TestDispatchAndReportCredentialSessionDiscoveryEventReportsReassignmentWithoutCredentials
--- PASS: TestDispatchAndReportCredentialSessionDiscoveryEventReportsReassignmentWithoutCredentials (0.00s)
=== RUN   TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors
=== RUN   TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors/no_account
=== RUN   TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors/auth_error
--- PASS: TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors (0.00s)
    --- PASS: TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors/no_account (0.00s)
    --- PASS: TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors/auth_error (0.00s)
=== RUN   TestDispatchAndReportCredentialSessionDiscoveryEventNoopIsDebugOnly
--- PASS: TestDispatchAndReportCredentialSessionDiscoveryEventNoopIsDebugOnly (0.00s)
=== RUN   TestDispatchCredentialSessionDiscoveryEventForwardsExpiredObservation
--- PASS: TestDispatchCredentialSessionDiscoveryEventForwardsExpiredObservation (0.00s)
=== RUN   TestDispatchCredentialSessionDiscoveryEventPreservesReassignmentErrors
--- PASS: TestDispatchCredentialSessionDiscoveryEventPreservesReassignmentErrors (0.00s)
=== RUN   TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable
=== RUN   TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable/malformed_payload
=== RUN   TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable/service_unavailable
=== RUN   TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable/unrelated_event
--- PASS: TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable (0.00s)
    --- PASS: TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable/malformed_payload (0.00s)
    --- PASS: TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable/service_unavailable (0.00s)
    --- PASS: TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable/unrelated_event (0.00s)
PASS
ok  github.com/multica-ai/multica/server/internal/daemon  0.046s
```

### Focused repeat, race, and vet

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=20 ./internal/daemon -run 'TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

```text
ok  github.com/multica-ai/multica/server/internal/daemon  0.041s
```

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -race -count=1 ./internal/daemon -run 'TestDispatch(AndReport)?CredentialSessionDiscoveryEvent'
```

```text
ok  github.com/multica-ai/multica/server/internal/daemon  1.083s
```

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go vet -tags=offline ./internal/daemon
```

```text
<no output; exit status 0>
```

### Complete offline daemon package

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -count=1 ./internal/daemon
```

```text
ok  github.com/multica-ai/multica/server/internal/daemon  19.957s
```

```sh
env APP_ENV=test DATABASE_URL='://offline-invalid' GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -tags=offline -race -count=1 ./internal/daemon
```

```text
ok  github.com/multica-ai/multica/server/internal/daemon  22.346s
```

### Formatting and diff hygiene

```sh
/home/dataops-lab/go-sdk/bin/gofmt -d internal/daemon/wakeup.go internal/daemon/credential_session_monitor.go internal/daemon/credential_session_monitor_test.go internal/daemon/credential_session_alert_test.go
git diff --check -- multica-auth-work/server/internal/daemon/wakeup.go multica-auth-work/server/internal/daemon/credential_session_monitor.go multica-auth-work/server/internal/daemon/credential_session_monitor_test.go multica-auth-work/server/internal/daemon/credential_session_alert_test.go
```

```text
<no output from either command; exit status 0>
```

## Review boundary

This evidence proves the offline daemon recording/alerting path and its no-secret behavior with synthetic sentinels. It does not claim frontend `useSessionMonitor` UI delivery, live provider login, live WebSocket delivery, PostgreSQL persistence execution, external notification delivery, production deployment, or independent acceptance.
