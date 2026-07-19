# Agent credential isolation task 4.2: independent same-provider next-account selection validation

Independently inspected and reproduced on 2026-07-18 from repository source and
synthetic, temporary in-process test fixtures only. No real credential, auth
home, session home, credential file, token, environment secret, network
service, live session, live daemon/CLI/provider process, database, or
multi-node shared state was read or used. Toolchain:
`/home/dataops-lab/go-sdk/bin/go` (`go1.26.4 linux/amd64`); environment:
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`. Run inside Herdr pane
`w4:p3` (workspace `w4`), slot-isolated `GOMODCACHE`; no proxy contact.

## Verdict

**NOT SELF-ACCEPTED.** Submitted for TL independent acceptance. This artifact
fills the evidence-index gap noted in `EVIDENCE_INDEX.md` (task 4.2 "Selecionar
próxima conta disponível do mesmo provedor", KEPT CHECKED): implementation was
present but no dedicated acceptance test/artifact was indexed for the
selection step. It does not mark the task DONE; per `EVIDENCE_CONTRACT.md` rule
6 ("Distinguir reviewed · implemented · verified · accepted") and rule 7 ("TL
valida independentemente"), acceptance is reserved for the TL.

The synthetic source-only evidence supports both acceptance scenarios of the
"Rotação automática ao esgotar conta (Fase 2)" requirement
(`openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md:72-84`,
task `openspec/changes/agent-credential-isolation/tasks.md:26`, marked `[x]`):

1. **Troca automática ao esgotar** — when the active account is detected
   exhausted/`expired` and another same-provider account exists, the agent is
   reassigned to the next account of the same provider+tenant, without crossing
   provider or tenant boundaries.
2. **Sem conta disponível** — when the active account exhausts and no other
   same-provider account is available, the system signals exhaustion (alert
   gauge) and fails closed without overwriting credentials or disturbing the
   current assignment/session.

## Provider/tenant boundary (same-provider selection)

Selection is scoped by **vendor + tenantID** at every layer; it cannot return an
account from a different provider or tenant:

- `internal/rotation/pool.go:21` `Pool.SelectNext(ctx, vendor, tenantID, now)`
  delegates to `selectNext`, which calls
  `p.store.ListAccounts(ctx, vendor, tenantID)` (`pool.go:39`, `:142`).
- `internal/rotation/contract.go:102` `Store.ListAccounts(ctx, vendor, tenantID)`
  is the persistence port; every synthetic store filters on
  `account.Vendor == vendor && account.TenantID == tenantID`
  (`service_test.go:116`, `discovery_reassignment_test.go:284`,
  `credential_session_discovery_producer_test.go:289`).
- `internal/rotation/contract.go:59-72` `Account` carries `Vendor` (`:61`) and
  `TenantID` (`:62`) as first-class identity; priority order is `Priority` then
  `AccountID` (`pool.go:43-48`, `:146-151`).
- `internal/rotation/discovery_reassignment.go:69-71` `ReassignDiscoverySession`
  rejects a current assignment whose provider or tenant does not match the
  observation with `errDiscoveryAssignmentBoundary` (`:13`), before any logout
  or selection.
- `internal/rotation/detector_discovery.go:29-32` `DetectDiscoverySession`
  returns a non-exhausted result when `requestedProvider` and `session.Provider`
  do not identify the same provider family (`sameDiscoveryProvider`, `:50-54`),
  so an expired observation from one provider cannot affect another provider's
  pool. `canonicalDiscoveryProvider` (`:56-67`) folds aliases
  (`kiro_cli`→`kiro`, `gemini_cli`/`agy`→`antigravity`, `claude_code`→`claude`).
- `internal/daemon/daemon.go:4193-4194` the proactive-ledger path also enforces
  the provider boundary: `if !strings.EqualFold(account.Vendor, provider) {
  return ... false }` before any rotation.
- `internal/daemon/daemon.go:4275` the daemon calls
  `rotationService.OnExhaustion(ctx, task.AgentID, provider, task.WorkspaceID,
  reason, start)` — the tenant scope is `task.WorkspaceID`, the provider scope
  is the task's provider.

## No-candidate fail-closed behavior

When no selectable same-provider account exists, every path returns
`rotation.ErrNoAccountAvailable` (`internal/rotation/contract.go:25`) and
neither overwrites credentials nor disturbs the current session:

- `internal/rotation/pool.go:57` `selectNextPriority` returns
  `ErrNoAccountAvailable` after the priority loop finds no selectable account;
  also `:74`, `:100`, `:106`, `:117`, `:135` for the policy/fallback/
  load-balanced branches. `accountSelectable` (`:155-164`) admits only
  `StatusAvailable`/`StatusLeased`, or `StatusCooldown` whose `CooldownUntil`
  is not after `now`; `StatusExhausted` and `StatusDegraded` are never
  selectable.
- `internal/rotation/service.go:97-166` `onExhaustionLocked` selects a
  replacement **before** logging out the current account (`:103-122`), so an
  exhausted pool returns `ErrNoAccountAvailable` without ever calling `Logout`
  on or disturbing the current authenticated session. On repeated login
  failure it falls through to `ErrNoAccountAvailable` (`:165`) only after
  marking failed candidates `StatusDegraded` (`:144`).
- `internal/rotation/discovery_reassignment.go:72-79` marks the current account
  `StatusExhausted` and calls `onExhaustionLocked`; if no replacement exists,
  `ErrNoAccountAvailable` propagates and the assignment is unchanged (the
  synthetic test at `discovery_reassignment_test.go:93-118` asserts no auth
  calls occurred and the assignment stayed `current`).
- `internal/daemon/daemon.go:4277-4283` on
  `errors.Is(err, rotation.ErrNoAccountAvailable)` the daemon sets the alert
  gauge `credentialMetrics.SetAllAccountsExhausted(provider, true)`
  (`internal/metrics/credential_metrics.go:146-155`, gauge
  `all_accounts_exhausted`; the available-count gauge is `accounts_available`,
  `credential_metrics.go:48-51`), logs "rotation: no account available;
  preserving current failure behavior", and returns `(Account{}, false)` — no
  credential overwrite, current failure preserved. The selection-latency
  histogram `omniroute_selection_seconds` is declared at
  `internal/daemon/observability/schema.go:205` (route/protocol labels); its
  thresholds remain G4 evidence-gated and are not exercised here.

## Exact synthetic-test source anchors

All paths are relative to `multica-auth-work/server`. No path under test reads
a real credential, auth home, session home, token, or environment secret.

| Coverage | Exact source anchor | What the synthetic test proves |
| --- | --- | --- |
| SelectNext by priority, same provider+tenant | `internal/rotation/pool_test.go:10` | Three `codex`/`tenant-1` accounts select by `Priority` then `AccountID`; the synthetic store only returns same vendor+tenant rows. |
| Cooldown respected / recovers after cooldown | `internal/rotation/pool_test.go:26` | A `StatusCooldown` account with future `CooldownUntil` is skipped for an available lower-priority one; after the cooldown passes it becomes selectable again. |
| All exhausted/cooldown/degraded → fail-closed | `internal/rotation/pool_test.go:53` | With only `StatusExhausted`/`StatusCooldown`(future)/`StatusDegraded` rows, `SelectNext` returns `ErrNoAccountAvailable`. |
| OnExhaustion happy path, same provider+tenant | `internal/rotation/service_test.go:10` | `current`(leased)→`next`(available) for `codex`/`tenant-1`; auth sequence `logout:current,login:next,wait:session-next`; exactly one rotation record with the right reason/time. |
| Failed login → degraded → try next | `internal/rotation/service_test.go:40` | A failing `first` is marked `StatusDegraded` and selection advances to `second`; same provider+tenant throughout. |
| Same-provider reassignment skips a different-provider account | `internal/rotation/discovery_reassignment_test.go:12` | Pool has `current`(codex), `wrong-provider`(kiro, priority 1, available), `next`(codex); an expired `codex` observation reassigns to `next`, never to `wrong-provider`. |
| Future expiry & cross-provider observation are no-ops | `internal/rotation/discovery_reassignment_test.go:47` | A future `expires_at` and a `kiro` observation for a `codex` request do nothing; assignment stays `current`, no auth calls. |
| No next account → fail-closed before logout | `internal/rotation/discovery_reassignment_test.go:93` | Only `current`(leased), no next → `ErrNoAccountAvailable`, assignment unchanged, **zero** auth calls (no logout before replacement). |
| Assignment provider/tenant boundary rejection | `internal/rotation/discovery_reassignment_test.go:120` | `different provider` (current=kiro) and `different tenant` (current=tenant-2) both yield `errDiscoveryAssignmentBoundary`, no reassignment, no auth calls. |
| Assign failure cleans the new session only | `internal/rotation/discovery_reassignment_test.go:159` | A synthetic `Assign` failure logs out the newly-logged-in `next` and preserves `current`; zero rotation records. |
| Concurrent duplicate observation rotates once | `internal/rotation/discovery_reassignment_test.go:195` | 32 concurrent `ReassignDiscoverySession` calls produce exactly 1 successful reassignment and 1 rotation record (per-agent lock idempotency). |
| Discovery detector provider boundaries | `internal/rotation/detector_discovery_test.go` (`TestDetectDiscoverySessionProviderBoundaries`) | `DetectDiscoverySession` is non-exhausted when requested and session providers differ; alias canonicalization is exercised. |
| Daemon monitor forwards provider+tenant identity | `internal/daemon/credential_session_monitor_test.go:14` | `dispatchCredentialSessionDiscoveryEvent` forwards `AgentID`/`AccountID` as the stale-event compare value and `Provider`/`WorkspaceID` as the selection scope; payload carries only non-secret metadata. |
| Daemon monitor propagates `ErrNoAccountAvailable` | `internal/daemon/credential_session_monitor_test.go:57` | When the reassigner returns `ErrNoAccountAvailable` the daemon event handler returns `handled=true` with that error (no swallow, no fabricated success). |
| Daemon monitor rejects malformed/unavailable/unrelated | `internal/daemon/credential_session_monitor_test.go:79` | Malformed payload, unavailable service, and an unrelated event type are each handled correctly (no reassignment calls). |
| Producer→monitor→reassignment end-to-end, same provider | `internal/daemon/credential_session_discovery_producer_test.go:17` | Store has `account-current`(codex), `wrong-provider`(kiro, priority 1, available), `account-next`(codex); an expired `codex` observation emits exactly one event and the assignment becomes `account-next` (never `wrong-provider`); `account-current`→`StatusExhausted`; auth sequence `logout:account-current,login:account-next,wait:session-account-next`; a duplicate observation is suppressed. |
| Producer validates exact bounded fields | `internal/daemon/credential_session_discovery_producer_test.go:83` | Missing agent/account/provider/workspace, missing status+expiry, padded values, and oversized provider all return `errInvalidCredentialDiscoveryObservation` with zero emissions. |
| Producer deduplicates concurrently | `internal/daemon/credential_session_discovery_producer_test.go:117` | 64 concurrent `Produce` calls emit exactly one event. |
| Producer bounds and resets dedup | `internal/daemon/credential_session_discovery_producer_test.go:161` | Dedup is size-bounded (limit 2) and an active observation clears the key so a later exhausted observation re-emits. |
| Producer retries after emitter failure | `internal/daemon/credential_session_discovery_producer_test.go:193` | A one-shot synthetic emitter failure releases the dedup slot and a retry succeeds. |
| Daemon proactive fail-closed on no account | `internal/daemon/daemon_test.go:1346` | `maybeProactiveRotateOnText` with a fake service returning `ErrNoAccountAvailable` returns `ok=false` (current flow preserved; one call with `ReasonQuotaProactive`). |
| Daemon proactive no-trigger on normal text | `internal/daemon/daemon_test.go:1316` | Normal text produces zero rotation calls. |
| Daemon proactive nil-service preserves flow | `internal/daemon/daemon_test.go:1333` | `rotationService=nil` returns `ok=false`. |
| Daemon proactive ledger rotates before task | `internal/daemon/daemon_test.go:1364` | A `kiro` account at 95/100 tokens with an open window triggers proactive rotation to `next` (`ReasonQuotaProactive`). |
| Daemon proactive ledger below threshold does not trigger | `internal/daemon/daemon_test.go:1398` | 50/100 tokens does not trigger. |

The discovery detector gate (`DetectDiscoverySession`,
`internal/rotation/detector_discovery.go:29`) is the task-4.1 boundary that
gates the task-4.2 selection step; `TestDetectDiscoverySessionProviderBoundaries`
is included so the provider-family gate that precedes selection is exercised
in the same run. Task 4.1 detection-only evidence (`EV-CREDISO-4.1`) remains
the detection acceptance of record; this artifact does not re-grade it.

## Commands and results

Working directory for every Go command:
`multica-auth-work/server`. Toolchain:
`/home/dataops-lab/go-sdk/bin/go`; environment:
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.

### Rotation package — twelve focused synthetic tests, repeated 20 times

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./internal/rotation -count=20 -run '^(TestPoolSelectNextByPriority|TestPoolSelectNextRespectsCooldown|TestPoolSelectNextAllExhausted|TestServiceOnExhaustionRotatesHappyPath|TestServiceOnExhaustionMarksFailedLoginDegradedAndTriesNext|TestReassignDiscoverySessionExpiredReassignsSameProvider|TestReassignDiscoverySessionFutureOrCrossProviderDoesNothing|TestReassignDiscoverySessionNoNextFailsClosedBeforeLogout|TestReassignDiscoverySessionRejectsAssignmentBoundary|TestReassignDiscoverySessionAssignFailureCleansNewSession|TestReassignDiscoverySessionConcurrentDuplicateObservationRotatesOnce|TestDetectDiscoverySessionProviderBoundaries)$'
```

Result: exit 0; `ok github.com/multica-ai/multica/server/internal/rotation 0.032s`.
Verbose re-run (`-v`) recorded `240` `--- PASS` lines and `0` `--- FAIL`
lines (12 tests × 20). The `TestReassignDiscoverySessionRejectsAssignmentBoundary`
subtests `different_provider` and `different_tenant`, and the
`TestReassignDiscoverySessionFutureOrCrossProviderDoesNothing` subtests
`future_expiry` and `cross_provider`, were each observed in the run.

### Daemon package — thirteen focused synthetic tests, repeated 20 times

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./internal/daemon -count=20 -run '^(TestCredentialSessionDiscoveryProducerEndToEndReassigns|TestCredentialSessionDiscoveryProducerRequiresExactBoundedFields|TestCredentialSessionDiscoveryProducerDeduplicatesConcurrently|TestCredentialSessionDiscoveryProducerBoundsAndResetsDedup|TestCredentialSessionDiscoveryProducerRetriesAfterEmitterFailure|TestDispatchCredentialSessionDiscoveryEventForwardsExpiredObservation|TestDispatchCredentialSessionDiscoveryEventPreservesReassignmentErrors|TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable|TestProactiveRotationNoAccountAvailablePreservesCurrentFlow|TestProactiveRotationNormalTextDoesNotTrigger|TestProactiveRotationNilServicePreservesCurrentFlow|TestProactiveRotationLedgerBeforeTask|TestProactiveRotationLedgerBelowThresholdDoesNotTrigger)$'
```

Result: exit 0; `ok github.com/multica-ai/multica/server/internal/daemon 0.073s`.
Verbose re-run (`-v`) recorded `260` `--- PASS` lines and `0` `--- FAIL`
lines (13 tests × 20). The boundary/fail-closed subtests
(`missing_agent`, `missing_provider`, `missing_workspace`, `missing_status_and_expiry`,
`padded_account`, `oversized_provider`, `malformed_payload`, `service_unavailable`,
`unrelated_event`) were each observed in the run.

### Race detector

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/rotation -run '<rotation set above>'
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race ./internal/daemon -run '<daemon set above>'
```

Results: both exit 0; rotation `ok` in 1.042s and daemon `ok` in 1.081s
(race clean; the 32-worker and 64-worker concurrency cases are included).

### Vet, format, and build checks

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go vet ./internal/rotation ./internal/daemon ./internal/metrics
/home/dataops-lab/go-sdk/bin/gofmt -l <16 source/test paths listed in the manifest below>
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go build ./internal/rotation/ ./internal/daemon/ ./internal/metrics/
```

Results: `go vet` exited 0 with no output; `go build` of all three packages
exited 0. `gofmt -l` listed exactly one file, `internal/rotation/contract.go`,
as having a pre-existing comment-alignment drift
(`ReasonQuotaReactive`/`ReasonQuotaProactive` trailing-comment spacing). That
file is unmodified in the working tree (`git status --porcelain
internal/rotation/contract.go` empty) and was last touched by commit
`aa62401` on 2026-07-02 ("checkpoint: rotation platform
(isolation+rotation+observability) + prod-readiness wave"), i.e. the drift
predates this session. Per the no-edit constraint for this evidence lane, it
was not modified; the other 15 manifest paths are `gofmt`-clean.

## Source SHA-256 manifest

Hashes were computed only over the repository source/test paths listed below.
No external home, credential, auth, session, token, environment-secret, or
live service path was enumerated, read, copied, or hashed.

```text
eef92d45127137f5f339a430490c7b188f5704c9ebf55b545a54fc0c3c85fd4e  internal/rotation/contract.go
0c4c453c730f5b9d49c605e2206e1501b778f32f40b20b142d86e70cdb8fb3fc  internal/rotation/pool.go
f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0  internal/rotation/service.go
c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832  internal/rotation/discovery_reassignment.go
bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55  internal/rotation/detector_discovery.go
f5617e8112ab72528857c83316ca11bf612b8085176c3d9080a00a12a946139f  internal/rotation/proactive.go
da401ef882af6fe06bb923494f4393b685c45dca01e9b0707127bda16a87f005  internal/rotation/pool_test.go
989ae633e1865bc0f1f5a9f7eed5d2541100dbee94960c9e2c15f84346aa8ba1  internal/rotation/service_test.go
d87aa80717e33a3f3768ebb7d698dcd9ace3bd934550317ec64c57f56a6566b2  internal/rotation/discovery_reassignment_test.go
a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07  internal/daemon/daemon.go
a77d2f7052ba20c99a068b8dd46fbd688a94fd24c5a9c7e62bc1287ae140478e  internal/daemon/credential_session_monitor.go
5fd7005b4209b0c8d5c42ad9b240ad039d7568ad1f895bd6b9463b8e237613e2  internal/daemon/credential_session_monitor_test.go
4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c  internal/daemon/credential_session_discovery_producer.go
818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a  internal/daemon/credential_session_discovery_producer_test.go
fbb91116b46e25de5451dcc2fe2b6d8e46bca89e5a2665799d96d2f2f1cff624  internal/metrics/credential_metrics.go
a7fc41a69ea9f40df5c5eae18b56dcde7b345526041d86c473dafacfc835dd6c  internal/daemon/observability/schema.go
```

The manifest was revalidated with `sha256sum -c` against these repository
paths after the artifact was written. `internal/daemon/daemon.go`
(`a1d96a3c…fe07`) matches the hash recorded in the sibling task-5.2 artifact
`credential-isolation-two-account-coexistence.md`, confirming a consistent
source baseline across the two credential-isolation evidence lanes.

## Explicit non-claims

This validation used no database, network, live service, live daemon/CLI/provider
process, environment secret, or multi-node shared state. The following are
therefore **not** claimed:

- **DB-backed store/E2E.** `internal/rotation/store_pg_test.go:17-29` requires
  `DATABASE_URL` and Postgres and `t.Skip`s without it.
  `internal/rotation/rotation_e2e_test.go:1` carries `//go:build !offline` and
  `e2eRotationPool` (`:95-110`) `t.Skip`s when `DATABASE_URL` is unset. Neither
  was executed; this artifact does not claim the Postgres-backed
  `ListAccounts`/`Assign`/`RecordRotation` paths, persisted-state operation, or
  the E2E rotation-credential-restore flow ran. All store interactions here are
  in-process synthetic fakes
  (`fakeStore`, `syntheticReassignmentStore`, `producerSyntheticStore`).
- **Live provider/daemon/CLI.** No real provider, daemon, CLI, or agent process
  was started; `OnExhaustion`/`ReassignDiscoverySession` were exercised through
  synthetic authenticators (`fakeAuthenticator`,
  `syntheticReassignmentAuthenticator`, `producerSyntheticAuthenticator`) that
  never touch a real `CODEX_HOME`/`XDG_DATA_HOME`/`HOME`/credential file.
- **Multi-node / shared state.** The per-agent `agentLock`
  (`internal/rotation/service.go:168-177`) and the producer's process-local
  dedup (`credential_session_discovery_producer.go:62-92`) were exercised only
  within a single process; no cross-node coordination, shared persisted
  assignment ledger, or distributed lock behavior is claimed.
- **Policy/load-balance selection mode.** The policy-driven
  fallback/load-balanced branches (`pool.go:60-109`,
  `loadbalance.go`, `policy.go`, `fallback.go`) were not in the focused
  `-run` set; their helper tests (`TestPickWeighted*`, `TestPickConsistent*`,
  `TestPickByWindowHealth*`, `TestFallback*`, `TestPolicy*`) exist in the
  package and compile/build offline, but this artifact makes no acceptance
  claim for them — the task-4.2 acceptance scenarios above are satisfied by
  the priority/`OnExhaustion`/`ReassignDiscoverySession` paths actually run.
- **Tier/selection-latency thresholds.** `omniroute_selection_seconds`
  (`observability/schema.go:205`) thresholds and any tier-20/50/100 capacity
  acceptance remain G4 evidence-gated and are not exercised here.

## No-edit statement

No product source file, OpenSpec artifact/checklist/checkbox, STATE file,
AGENT_LEDGER row, EVIDENCE_INDEX row, or other evidence file was edited for this
validation. The only file created is this artifact,
`.planning/agent-brain-v3/evidence/credential-isolation-next-account-selection.md`.
The pre-existing `gofmt` drift in `internal/rotation/contract.go` was left
unchanged per the no-edit constraint. This artifact does not mark task 4.2
DONE or ACCEPTED; it is submitted for TL independent acceptance.
