# Independent design critique — credential isolation task 4.3 production integration gap

## Golden Rule check-in / check-out

- **Check-IN:** 2026-07-18T21:25:21Z — Codex-root claimed an independent, static design review of `credential-isolation-4.3-production-integration-gap-trace.md`, current product source, and the current OpenSpec task/spec. The sole writable file is this artifact.
- **Constraints honored:** no shared planning document, product, test, spec, task, git index, credential path/value, environment value, database, network, provider, or service was accessed or changed. No test or runtime command was executed.
- **Check-OUT:** 2026-07-18T21:28:10Z — DONE. Kiro TL remains the sole adjudicator. This is a design grade only; it neither accepts nor rejects OpenSpec task 4.3 and changes no checkbox.

## Verdict

**Design grade: REJECT as an implementation-ready design.**

The trace correctly identifies the primary production integration gap: the repository contains a bounded producer abstraction and a discovery-event consumer, but no production observation source, no concrete non-test emitter, and no non-test caller of `Produce`. That diagnosis is **ACCEPT**. The proposed “smallest safe production integration,” however, is not safe or complete: it assumes an existing discovery source that is absent, assumes transport/routing support that does not exist, leaves discovery rotation outside the established router-owner gate, offers process-local locking as if single-daemon ownership could be assumed, and does not define recoverable semantics across destructive filesystem authentication and the database transaction. Those are correctness boundaries, not optional hardening.

Subgrades:

| Design field | Grade | Finding |
|---|---|---|
| Missing production producer/emitter diagnosis | **ACCEPT** | Static reference closure confirms no non-test constructor call, `Produce` call, or concrete emitter method. |
| Production call-graph description | **PARTIAL** | Bootstrap and both consumers are substantially correct, but the trace incorrectly says Path A only benefits the next claim: reactive exhaustion retries the current task via `goto runAttempt`. |
| Task-run/router-owner gating | **REJECT** | Path A is gated; discovery Path B is not, and its payload lacks task/runtime ownership identity needed to apply the gate safely. |
| Process/cross-process concurrency | **REJECT** | Both dedup and agent locking are process-local; PostgreSQL assignment is unconditional last-writer-wins and the WebSocket hub permits multiple clients per runtime. |
| Assignment/audit atomicity | **REJECT** | `Assign` precedes `RecordRotation`; a record failure leaves the new assignment durable, and stale-event suppression prevents a normal retry from repairing the missing audit row. |
| Authentication rollback/logout safety | **REJECT** | Current account credential paths are deleted before next-account authentication and before durable commit, with no restore of the current account on downstream failure. Multiple agents may share one account. |
| Proposed new-file/one-line integration scope | **REJECT** | It requires authoritative source, transport routing, owner gating, transactional store/service changes, and rollback semantics across existing ownership boundaries. |

## Current SHA-256 provenance

Hashes were computed over repository files only.

| File | Current SHA-256 |
|---|---|
| `.planning/agent-brain-v3/evidence/credential-isolation-4.3-production-integration-gap-trace.md` | `d793712503932cee5eb9757f18d252e6a89da8c2ff63a4cdd6874ba0790454b4` |
| `openspec/changes/agent-credential-isolation/tasks.md` | `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3` |
| `openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md` | `02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b` |
| `multica-auth-work/server/internal/daemon/daemon.go` | `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07` |
| `multica-auth-work/server/internal/daemon/wakeup.go` | `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` |
| `multica-auth-work/server/internal/daemon/credential_session_monitor.go` | `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` |
| `multica-auth-work/server/internal/daemon/credential_session_discovery_producer.go` | `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` |
| `multica-auth-work/server/internal/daemonws/hub.go` | `20eec4cb8754125f46c4199479403f43404d2027a2d544dc2089972be125385a` |
| `multica-auth-work/server/internal/rotation/contract.go` | `eef92d45127137f5f339a430490c7b188f5704c9ebf55b545a54fc0c3c85fd4e` |
| `multica-auth-work/server/internal/rotation/pool.go` | `0c4c453c730f5b9d49c605e2206e1501b778f32f40b20b142d86e70cdb8fb3fc` |
| `multica-auth-work/server/internal/rotation/service.go` | `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` |
| `multica-auth-work/server/internal/rotation/discovery_reassignment.go` | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` |
| `multica-auth-work/server/internal/rotation/auth_authenticator.go` | `caaa23505bb66dbaf637c71ae8f987c88a642f04fdc647bed3b9ccb1a3d1c22a` |
| `multica-auth-work/server/internal/rotation/store_pg.go` | `e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8` |
| `multica-auth-work/server/internal/daemon/l2_runtime.go` | `a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de` |
| `multica-auth-work/server/migrations/123_rotation.up.sql` | `df26a5debe8657914faa91b8c22d14e423b82802d7e877054000c3c405cebfdc` |
| `multica-auth-work/server/internal/daemon/credential_session_record_failure_alert_test.go` | `5b4d82caba027d4dd6b2650d9cd5a2ad78ebdd07504060fd84a415d40be739f5` |

## Production call graph validation

### Bootstrap and task-run path

`daemon.New` calls `initRotationService` only outside gateway-required Agent Brain mode (`daemon.go:300-303`). `initRotationService` (`daemon.go:310`) leaves `rotationStore` and `rotationService` nil when `RotationDatabaseURL` is empty and otherwise constructs `PGStore`, `CredentialAuthenticator`, and `rotation.Service` (`daemon.go:310-327`). The trace is correct on this bootstrap gate.

Path A is production-wired and router-owner gated. `legacyGoRotationAllowed` is at `daemon.go:4152`; reactive detection enters at `daemon.go:4248` and calls the gated rotation path at `daemon.go:4267`. There is also an outer `retry_after_rotation` gate. Contrary to the trace’s blanket “next claim only” statement, successful reactive rotation updates the account home, clears prior session/workdir, and retries the same in-flight task (`daemon.go:3827-3838`, `goto runAttempt`). This is a new execution attempt, not an in-place provider-session switch, but it does satisfy more of the spec’s “execução continua” clause than the trace credits. Proactive rotation timing should be described separately rather than grouped into the same next-claim claim.

### Discovery path and reachability

The consumer call graph exists:

`readTaskWakeupMessages` (`wakeup.go:262`, event branch `:275`) → `dispatchAndReportCredentialSessionDiscoveryEvent` (`wakeup.go:330`) → `dispatchCredentialSessionDiscoveryEventWithOutcome` (`credential_session_monitor.go:66`) → `rotation.Service.ReassignDiscoverySession` (`discovery_reassignment.go:23`) → `onExhaustionLocked` (`service.go:97`).

The producer reachability claim is confirmed more strongly than the trace states:

- A non-test reference search finds only the interface, producer implementation, and consumer constants/functions. There is no non-test call to `NewCredentialSessionDiscoveryProducer` or `Produce`.
- The only concrete `EmitCredentialSessionDiscovery` methods are test emitters in `credential_session_discovery_producer_test.go`; there is no non-test implementation.
- The current tree contains no production `/auth/sessions` endpoint, `useSessionMonitor`, or `isExpiringSoon` implementation. Tasks 0.3 and 2.1 remain open. Therefore “fed by the existing session-discovery source” is unsupported in this repository.
- The server-side daemon WebSocket relay is not ready for this event. `daemonws.Hub.DeliverDaemonRuntime` (`hub.go:286`) recognizes only task-available and runtime-profile-changed server→daemon frames (`:298`, `:311`); discovery events fall into the default miss. A server emitter therefore needs relay/protocol routing work, not merely a new emitter file.

The consumer is consequently present but production-unreachable from a real discovery observation at these hashes.

## Router ownership and fallback gating

The trace correctly identifies the question but understates it as an owner choice. It is a required safety decision before integration.

Path B calls the rotation service directly from `credential_session_monitor.go:66-104`; it never calls `legacyGoRotationAllowed` or `legacyGoRotationBlockError`. Its payload carries agent/account/provider/workspace/status/expiry, but no task ID, runtime ID, router owner, or ownership epoch (`credential_session_monitor.go:20-27`). A daemon-local emitter would therefore rotate even when Rust L2 or OmniRoute owns routing. A server-side emitter cannot safely infer “the daemon owning the runtime” from the current observation contract either.

Adding a boolean gate at the consumer is insufficient unless its value is resolved from authoritative current ownership. `runtimeRouterOwnerForTask` is task-oriented (`l2_runtime.go:624-633`); the discovery event is agent-oriented and may arrive outside an active task. The design must define one of:

1. a single authoritative rotation coordinator that owns assignment and credential transition for the agent; or
2. a routed command containing a server-resolved runtime identity plus an ownership generation, revalidated by the target daemon before mutation.

An asserted router owner in the event payload alone is not authoritative and can be stale.

## Process-local locking and cross-process correctness

`CredentialSessionDiscoveryProducer` protects a bounded, TTL dedup map with one process-local mutex (`credential_session_discovery_producer.go:62-72,176-239`). `rotation.Service.agentLock` returns a process-local mutex from a process-lifetime map (`service.go:168-177`). Neither serializes another daemon/server process, and the agent-lock map is never pruned.

The database layer does not close this gap:

- `PGStore.Assign` is an unconditional `INSERT ... ON CONFLICT (agent_id) DO UPDATE` (`store_pg.go:127-139`), with no expected-current comparison, row lock, or ownership epoch.
- Selection reads a provider/tenant account list and chooses a selectable row (`pool.go:33-58`) without reserving it transactionally.
- `assignments.account_id` has only a non-unique index (`123_rotation.up.sql:36-42`), so multiple agents may share one account. That may be a product choice, but it is incompatible with destructive logout unless per-account concurrent use is explicitly coordinated.
- The WebSocket hub stores a set of clients for each runtime and sends a frame to every connected client (`daemonws/hub.go:330-349`); current transport topology does not prove a single daemon per runtime.

“Route to the single daemon” is therefore an invariant the proposed design assumes but current code does not enforce. A DB advisory lock alone is also incomplete unless selection, expected-assignment compare, assignment update, status changes, and audit insertion occur under the same transaction/lock.

## Assign-before-RecordRotation atomicity

The trace’s atomicity finding is correct, but its idempotent-at-least-once fallback is not sufficient.

`onExhaustionLocked` calls `Assign` at `service.go:150`, then `RecordRotation` at `:156`. The focused failure test explicitly proves that a record failure leaves the next assignment persisted (`credential_session_record_failure_alert_test.go:39-42,124-128`). On retry, discovery reassignment reads the new current account and compares it to the old `expectedAccountID`; the mismatch returns a no-op at `discovery_reassignment.go:54-63`. Thus an idempotent `RecordRotation` method is never reached by the ordinary retry. The audit hole becomes durable unless a separate reconciliation mechanism exists.

The minimum durable operation must be a transactional compare-and-swap, conceptually:

`RotateAssignment(expected account, next account, provider, tenant, event id, reason, time)` → lock agent assignment → confirm expected account/ownership generation → validate/reserve next account → atomically update status/assignment and insert one uniquely identified rotation event → commit.

It must return “stale/already applied” distinctly from failure. A unique event/idempotency key is needed if at-least-once delivery is retained. Merely combining current `Assign` and `RecordRotation` without expected-state validation does not prevent two processes from committing competing rotations.

## Destructive logout and rollback semantics

The destructive logout risk is confirmed and is broader than “ensure per-account home isolation.”

`onExhaustionLocked` selects a next account and then logs out the current account before authenticating the next (`service.go:103-134`). `CredentialAuthenticator.Logout` recursively removes each current credential destination (`auth_authenticator.go:94-106`). Default paths use `HomeRoot=acc.HomeDir` and `ConfigDir` as the optional source (`auth_authenticator.go:142-163`). If source and destination are the same, restore is a no-op (`:166-173`), so logout can delete the only local copy. If another agent/process uses the same account home, it is interrupted. The schema permits that sharing.

No current failure branch restores the old credential/session:

- next login/wait failure marks the next account degraded and tries another, while current remains logged out (`service.go:134-147`);
- assignment failure logs out next but does not restore current (`:150-154`);
- record failure leaves assignment and next credentials active while returning an error (`:156-158`);
- status update and authentication filesystem effects are outside any database transaction.

A safe design should prepare and validate the next credential without destroying current state, then atomically commit assignment+audit, and only afterward retire the old credential when exclusive ownership is proven. If a provider forces logout-first behavior, the design needs an explicit recoverable snapshot/re-login operation and a per-account cross-process lease. On pre-commit failure, next state must be cleaned and current state must remain usable; on post-commit cleanup failure, the durable assignment should remain next and cleanup should be retried/alerted without claiming rollback.

## Required design corrections before implementation

1. **Name the authoritative observation source.** Define its API/store contract and ownership; do not cite absent `/auth/sessions` or frontend hooks as existing production inputs.
2. **Choose the coordinator and route exactly once.** Server-to-daemon requires a supported hub event, runtime/daemon lookup, authorization, delivery identity, and an ownership generation. Workspace broadcast and daemon-local loopback without owner proof are unsafe.
3. **Apply the router-owner gate before any status, filesystem, or assignment mutation.** Define behavior for Rust L2 and OmniRoute, including events arriving without an active task.
4. **Replace process-local correctness with transactional correctness.** Use a per-agent DB transaction/advisory or row lock plus expected-current CAS, transactional next-account reservation, status transition, assignment, and audit event.
5. **Define event idempotency and reconciliation.** A durable event ID and unique constraint must make duplicate delivery safe and allow repair after ambiguous outcomes.
6. **Redesign authentication ordering.** Prepare/verify next first; commit assignment+audit; retire current only under exclusive account ownership. Define cleanup and rollback for every failure boundary.
7. **Decide whether one account may serve multiple agents.** If yes, logout cannot delete an account-global home. If no, enforce exclusivity in Postgres rather than assuming it.
8. **Correct continuation semantics.** Preserve and test the existing same-task reactive retry; separately specify what discovery-triggered reassignment does to an already running task.

This necessarily crosses existing daemon, daemon-WebSocket/protocol, rotation store/service, and possibly migration ownership. Calling it a “new files only plus one wiring line” plan obscures the required atomic boundary and should not be used for implementation assignment.

## Acceptance tests required by the corrected design

All can use pure fakes except the separately owned PostgreSQL transaction contract, which needs an offline database-backed lane when authorized; no such test was run here.

- production construction/reachability test proving a real observation reaches exactly one authorized coordinator (not a test-only loopback);
- Rust L2 and OmniRoute owner negatives proving zero status/auth/assignment/audit mutation;
- two independent service instances racing the same expected assignment: exactly one CAS commit and one audit event;
- duplicate event ID replay: no second logout/login/assignment/event;
- `RecordRotation`/transaction failure: old assignment and usable current credential remain; no success alert;
- assignment commit ambiguity/retry: reconciliation returns the single committed outcome;
- next login/wait/status/assign failure at each boundary: current remains usable and next cleanup is bounded;
- two agents sharing one account, or a schema constraint prohibiting it: destructive logout cannot interrupt the other agent;
- reactive task-run rotation still retries the current task exactly once; discovery rotation has explicitly tested continuation behavior;
- no secrets, credential paths, raw provider errors, or tokens in events/logs.

## Non-claims

This review does not select the product’s server-versus-daemon coordinator, authorize a migration, assert that database-backed behavior ran, accept task 4.3, accept any evidence package, or grant push eligibility. It is based on static current-source inspection at the hashes above. Kiro TL adjudicates all next actions.
