# Credential-isolation 4.3 production emitter/atomicity design — Codex independent review

## Golden Rule CHECK-IN — 2026-07-18T22:16:18Z

- Reviewer: **Codex56#B** (cross-family, independent of design author Kiro/Opus-4.8 `w8:p2` and adjudicator Kiro TL).
- Reviewed artifact: `credential-isolation-4.3-production-emitter-atomicity-design.md`, requested SHA prefix `a06b8b5a...`; actual SHA-256 **`a06b8b5a7a81e16fc9edec71fc7d1d7c2fd69e6579ebffcc28cd7239c2fc0472` — PASS**.
- Scope: static source/design/OpenSpec/Kiro-record review only. Sole write is this artifact.
- Exclusions honored: no source/test/spec/task/shared-planning/git/index/ref edit; no auth/token/environment-value access; no DB, network, provider, or service access; no command executed against credentials or live infrastructure.
- This review grants no implementation authority and preserves credential-isolation tasks **4.3 and 4.4 OPEN**.

## Executive verdict

| Dimension | Verdict | Reason |
|---|---|---|
| Current-source fact integrity | **PASS** | All ten input hashes match; the missing production source/emitter, separate PG writes, process-local lock, and destructive logout ordering are source-correct. |
| Gap diagnosis | **PASS** | A/B/C/D are the correct high-level concern classes and Kiro is right that each contains an owner policy gate. |
| Technical completeness / “implementation-ready” claim | **REJECT** | The proposed interfaces and state machine do not yet close cross-process auth side effects, CAS result semantics, optional-store fail-open behavior, source lifecycle, T2 delivery, or post-commit logout failure. |
| Owner-policy readiness | **PARTIAL** | The owner can choose broad deployment posture, but A/B/C/D need the decision refinements below before a choice is safe and implementable. |
| Overall design | **PARTIAL** | Strong diagnosis and useful option framing; not yet a complete implementation contract. |

Kiro’s gate remains controlling: **no A/B/C/D option is selected here; implementation remains forbidden pending owner policy and revised technical closure.**

## Hash and source-fact verification

The design’s manifest reproduces exactly at the current checkout (actual paths are under `multica-auth-work/server/`):

| Current source | SHA-256 | Source fact |
|---|---|---|
| `internal/daemon/credential_session_discovery_producer.go` | `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` | bounded process-local dedup; only emits through injected interface (`:30-47`, `:62-72`, `:97-147`) |
| producer test | `818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a` | loopback emitter is test-only |
| `credential_session_monitor.go` | `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` | payload bridge uses `workspace_id` as rotation `tenantID` (`:20-27`, `:79-100`) |
| `wakeup.go` | `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` | daemon reader consumes the event asynchronously (`:263-288`) and reports bounded metadata (`:325-361`) |
| `daemon.go` | `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07` | PG service constructed in `initRotationService` (`:310-325`); daemon root context exists only after `Run` begins (`:776-783`) |
| `rotation/service.go` | `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` | logout→login→Assign→RecordRotation ordering (`:114-159`); lock map is per process (`:168-176`) |
| `discovery_reassignment.go` | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` | stale compare and provider/tenant checks precede status update and rotation (`:50-80`) |
| `store_pg.go` | `e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8` | unconditional upsert Assign (`:127-139`) and event insert (`:157-165`) use separate pool calls |
| `auth_authenticator.go` | `caaa23505bb66dbaf637c71ae8f987c88a642f04fdc647bed3b9ccb1a3d1c22a` | Login restores from source (`:63-91`); Logout removes destination recursively (`:94-107`); ConfigDir falls back to HomeDir (`:142-162`) |
| prior trace | `d793712503932cee5eb9757f18d252e6a89da8c2ff63a4cdd6874ba0790454b4` | matches design manifest |

OpenSpec hashes also reproduce: tasks `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3`; spec `02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b`. Task 4.3 remains unchecked at `tasks.md:27`, task 4.4 remains unchecked at `:28`, and the automatic-reassignment requirement remains normative at `spec.md:72-84`.

Current Kiro records were read at `AGENT_LEDGER.md:412-416` (file SHA during review `9e1f0113dd7720fca5ce3442bb54665d4e90093b930334188054df5c2de0e8f3`): A topology, B concurrency, C atomicity, and D logout/rollback are all owner gates; none is selected; 4.3 and 4.4 stay open. This review agrees with the hold but not with the record’s unqualified “implementation-ready” characterization.

## A — emitter/source topology: PARTIAL

### Verified positives

- T1 local dispatch is compatible with the existing daemon consumer and avoids inventing a WebSocket round trip.
- T2 is the necessary family of solutions if observations originate server-side or a different process owns discovery.
- Default-off enablement is the right migration posture.

### Blocking incompleteness

1. **S1 has no named real source.** A repository search finds no non-test session-discovery feed producing the required `status`/`expires_at` plus agent/account/provider/workspace tuple. `Source.Run` is a proposed shell, not a demonstrated adapter. The owner cannot safely select S1 until the concrete source API, cadence/event trigger, freshness model, and assignment binding are named.
2. **T2 is not currently deliverable.** The daemon reader recognizes `daemon:credential_session_discovery`, but server `daemonws.Hub.DeliverDaemonRuntime` routes only task-available and runtime-profiles-changed frames; every other type is a delivery miss (`internal/daemonws/hub.go:286-327`). The discovery payload also has no runtime/daemon routing key. “The consumer already handles it” is only the destination half of T2.
3. **Router ownership is absent from the event.** Existing Go rotation suppresses tasks owned by Rust-L2/OmniRoute (`daemon.go:4142-4169`), but discovery payload contains no task/session/runtime-owner identity. Enabling T1/T2 can therefore bypass the existing single-router rule or double-rotate an externally owned session.
4. **Lifecycle wiring is placed at the wrong phase.** `initRotationService()` is called during daemon construction and has no run context. The cancellable daemon root context is installed only in `Run` (`daemon.go:776-783`). A `go src.Run(ctx)` “one-line” constructor edit would either lack `ctx` or risk an uncancellable/background goroutine. Start/stop ownership and duplicate-start prevention must be defined in the run lifecycle.
5. **Workspace/tenant equivalence is hidden.** The bridge passes `payload.WorkspaceID` as `tenantID`. That may be the intended current mapping, but T2 must authenticate and validate it at the routing boundary rather than trust an emitter-provided string.

Correct policy question A is not merely T1 versus T2: it is **where the authoritative observation originates, which daemon/runtime owner is allowed to act, how the event is routed and authenticated, and where its lifecycle is owned**. T1 is admissible only with a proven single owning daemon and an in-daemon observation source; T2 requires protocol/hub routing work and ownership identity.

## B — multi-process concurrency: REJECT as currently specified

The design correctly notes that `sync.Mutex` and producer dedup are process-local. Its proposed “CPL3 primary + CPL1 safety net” does not yet provide end-to-end safety:

- CAS at final assignment prevents two DB assignment commits, but both processes can already select, mark status, restore/login credential homes, and mutate authenticator state before CAS. The losing process therefore is not a harmless no-op.
- The pool selection path reads and sorts accounts without reservation, `FOR UPDATE`, or durable ownership (`pool.go:35-57`). Two processes may prepare the same next account concurrently.
- An advisory **transaction** lock acquired only inside `AssignAndRecordRotation` is too late to serialize selection/auth filesystem side effects. Holding a DB transaction across bounded vendor authentication would serialize more, but creates a long-lived-transaction failure mode and is not designed here.
- CPL3 is safe only if single-writer routing is enforced, observable, and fail-closed—not merely a deployment assumption. The present WS hub can have routing ambiguity, and the discovery message lacks a daemon/runtime owner key.

The owner must choose an end-to-end ownership model. Either enforce one authoritative daemon writer per agent before any auth mutation, or introduce a durable claim/reservation/fencing token that spans prepare/commit/cleanup. CAS remains valuable as a final fence, but is not sufficient by itself.

## C — transactional Assign+RecordRotation: PARTIAL

Atomic treatment of the assignment and event is mandatory and a PG transaction is appropriate. The proposed optional interface preserves compilation of existing synthetic stores, but its contract is incomplete:

1. `AssignAndRecordRotation(...) error` cannot distinguish **CAS lost/stale no-op** from transaction failure. Yet Concern B requires zero affected rows to return `reassigned=false`. The interface needs an explicit result or a typed stale sentinel that `ReassignDiscoverySession` maps to a no-op with no success alert.
2. The conditional SQL cannot be the current upsert plus a `WHERE` alone. Discovery already requires an existing assignment; the transaction needs a conditional `UPDATE ... WHERE agent_id=$1 AND account_id=$expected`, must require exactly one affected row, then insert the event and commit.
3. The current-account `UpdateAccountStatus(StatusExhausted)` happens before `onExhaustionLocked` (`discovery_reassignment.go:72-76`) and is outside the proposed transaction. On CAS loss, commit failure, or auth failure, account status and assignment/event state may diverge. The design must explicitly decide whether status is an idempotent observation that remains exhausted or belongs in the atomic state transition.
4. **Optional fallback is compatibility-safe but production-unsafe.** Falling back to the existing `Assign; RecordRotation` sequence silently retains the known non-atomic defect. Since the contract says production persistence is Postgres-only, discovery-driven production should fail closed if its store lacks the atomic capability. Legacy/synthetic tests can use a separate adapter or exercise the old reactive path; optionality must not become a production downgrade.
5. Commit failure, rollback failure, context cancellation, and affected-row-count behavior are not defined. Transaction code must preserve the primary error and never report reassignment before commit.

Thus AT1 is a reasonable compatibility technique only after it has explicit CAS semantics and a fail-closed production boundary. It is not yet an implementation-ready interface.

## D — logout ordering and rollback: PARTIAL

L1 is directionally safer than logout-first when distinct destination homes are a validated invariant. L2 is a defensible contingency for shared paths. Neither is fully specified:

- **Pre-commit failure after successful next login:** if CAS loses, event insert fails, or commit fails, the prepared next credentials/session must be cleaned up. The design currently discusses assignment rollback but not authenticator/filesystem rollback.
- **Post-commit old-logout failure:** L1 has already durably assigned and recorded next. Rolling that DB state back is no longer possible; returning a generic failure could cause a retry and duplicate behavior. The design needs a committed-with-cleanup-failure outcome, operator alert, and bounded retry/reconciliation policy.
- **L2 compensation can fail.** `Login(current)` may find no source, partially restore, or fail after another filesystem error. Required behavior is error aggregation, operator-visible degraded state, and no false success—not an assertion that compensation is bounded.
- **Path isolation must be checked structurally before mutation.** Comparing account IDs is insufficient. `defaultCredentialPaths` can fall back from empty ConfigDir to HomeDir, and accounts sharing `HomeDir` plus vendor RelPath collide (`auth_authenticator.go:142-162`). L1 must fail closed unless every current/next destination is disjoint and every restore source is approved.
- The current account was already marked exhausted. The design must state whether preserving/restoring its credentials is only rollback safety (not making it selectable again) and how status remains consistent.

L3 remains correctly unacceptable.

## Optional interface, feature flag, and hidden coupling

| Area | Grade | Required clarification before implementation |
|---|---|---|
| Optional `AtomicRotationStore` compatibility | **PARTIAL** | Keeps fake stores compiling, but production discovery must require it; define typed CAS outcome and cleanup semantics. |
| Feature flag | **PARTIAL** | Default OFF is safe. Add a typed config field/parser, invalid-value behavior, startup/run lifecycle, and fail-closed enablement when source, atomic store, router ownership, or path isolation is unavailable. Do not read the environment ad hoc inside the source. |
| Legacy Go/Rust-L2/OmniRoute coupling | **REJECT unresolved** | Discovery must carry or resolve authoritative task/session/runtime ownership and reuse the same suppression policy before any rotation side effect. |
| WS transport coupling | **REJECT unresolved for T2** | Extend authenticated hub routing and include an owning runtime/daemon key; current hub drops the event type. |
| Workspace/tenant boundary | **PARTIAL** | Current equality is implicit. Validate authoritative agent→workspace/tenant assignment at the consumer/store boundary. |
| Daemon lifecycle | **REJECT unresolved** | Start source from `Run` after prerequisites, cancel it on shutdown, prevent duplicate starts, and surface fatal/nonfatal source errors. |
| Status/selection coupling | **PARTIAL** | Define status transaction semantics and durable reservation/fencing around selection/auth preparation. |

Flag-off does preserve today’s discovery-producer inactivity, but it does not by itself make flag-on safe. Enablement must be all-or-nothing and observable; partial prerequisites must not fall back to the non-atomic path.

## Proposed-test assessment

The five proposed synthetic tests are useful but **insufficient**. In particular, “two producers” in one process cannot prove multi-process DB correctness, and a fake transactional store can prove service behavior but not the actual PG SQL/row-count contract.

Minimum acceptance matrix before any completion claim:

| ID | Required proof | Offline-safe portion |
|---|---|---|
| A1 | flag absent/false produces no source, goroutine, event, or behavior change | pure config/lifecycle fake |
| A2 | flag true with missing source, non-atomic production store, unknown router owner, or shared path fails closed before mutation | pure fakes |
| A3 | source cancellation, duplicate-start prevention, bounded error behavior | fake source + cancellable context |
| A4 | T1 exact owner happy path plus provider/tenant/stale negatives | pure loopback |
| A5 | T2 hub accepts only authenticated owning-runtime route; wrong workspace/runtime/type is rejected | pure hub/protocol fakes; no socket required |
| B1 | two independent service instances race; exactly one fenced winner; loser performs no durable success and cleans any prepared next session | concurrency-safe shared fake with barriers |
| B2 | actual conditional SQL reports zero rows as typed stale/no-op and never inserts event | SQL/transaction adapter test; real PG remains a separately gated integration proof |
| C1 | record insert, commit, and context failures leave assignment/event unchanged; rollback error is joined without masking primary error | fake transaction executor |
| C2 | production discovery rejects a store without atomic capability; legacy test-store compatibility remains explicitly scoped | pure fakes |
| C3 | account-status outcome is asserted for stale, auth failure, commit failure, and success | pure state-machine fake |
| D1 | L1 distinct-path success; shared/overlapping path rejected before Login/Logout | `t.TempDir`, synthetic files only |
| D2 | CAS loss/commit failure after next login cleans next and preserves current | fake auth + fake transaction |
| D3 | old Logout failure after commit yields committed-with-cleanup-alert, no false rollback/success ambiguity, bounded retry | pure fake |
| D4 | L2 restore succeeds and restore failure is surfaced without secret/error-body leakage or false success | `t.TempDir` + synthetic sentinels |
| E1 | no raw error, credential path, config path, session ID, token, or credential content in every new failure log | synthetic sentinels through captured logger |
| E2 | router-owner suppression prevents discovery rotation for Rust-L2/OmniRoute | pure daemon/service fake |

No DB-backed test is run or claimed by this review. A later owner-approved implementation may need a separately gated PG integration test for transaction/locking semantics; pure fakes cannot establish actual cross-process Postgres behavior.

## Corrected owner decision gate

Kiro’s A/B/C/D classification is valid, but safe owner choices require these sharper questions:

- **A — authority/topology:** identify the real observation source and authoritative owning daemon/runtime; choose T1 only with enforced colocation, otherwise design T2 routing/authentication first.
- **B — concurrency:** choose an enforced single writer or a durable reservation/fencing protocol spanning auth side effects; retain CAS as final protection, not the sole cross-process mechanism.
- **C — atomicity:** require atomic PG assignment+event with explicit stale/no-op result and decide account-status transaction semantics; prohibit production fallback to split writes.
- **D — credential state machine:** choose L1 only with structural path isolation and define pre-commit cleanup/post-commit cleanup-failure behavior; otherwise specify L2 including compensation failure.

These are owner-policy selections with daemon-owner, rotation/PGStore owner, and Kiro TL adjudication. The detailed interfaces, lifecycle, SQL, and tests remain assignable engineering/QA work after policy selection. **No option is selected here.**

## Non-claims and disposition

- **4.3 remains OPEN. 4.4 remains OPEN behind 4.3 production reachability.**
- Source hashes and static facts are verified; runtime, DB, network, credential, provider, and test behavior is not executed or claimed.
- No implementation, checkbox, integration, push, policy selection, or owner authorization is granted.
- The reviewed design should be revised before being treated as implementation-ready; its diagnostic content remains useful and should not be discarded.

## Golden Rule CHECK-OUT — 2026-07-18T22:19:00Z

Independent design review complete: source-fact integrity **PASS**, technical completeness **REJECT as implementation-ready**, owner-policy readiness **PARTIAL**, overall design **PARTIAL**. Only this artifact was created.
