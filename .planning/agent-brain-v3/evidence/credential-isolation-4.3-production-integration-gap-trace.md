# READ-ONLY trace — agent-credential-isolation 4.3 production integration gap

- Author: independent trace (Kiro/Opus-4.8). **Read-only.** No product/test/spec/task/shared-ledger/git/index change.
- Task 4.3: "Reatribuir o agente à nova conta sem intervenção manual." (auto-reassignment)
- Kiro TL adjudicates; owner architecture gates remain (daemon owner Codex1; rotation owner W-PGSTORE).

## Golden Rule check-IN / check-OUT

- **Check-IN** 2026-07-18T21:20:00Z — claimed: read-only call-graph trace + this single artifact.
- Excluded (honored): no product/test/spec/`tasks.md`/shared-ledger edit; no git/index op; no DB/network/live
  provider/credential/env-value access. Only this file written.
- **Check-OUT** 2026-07-18T21:38:00Z — DONE; trace + disjoint plan below; nothing else modified.

## Provenance — current source hashes (SHA-256, read-only)

| File | SHA-256 |
|---|---|
| `internal/daemon/daemon.go` | `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07` |
| `internal/daemon/wakeup.go` | `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` |
| `internal/daemon/credential_session_monitor.go` | `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` |
| `internal/daemon/credential_session_discovery_producer.go` | `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` |
| `internal/rotation/service.go` | `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` |
| `internal/rotation/discovery_reassignment.go` | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` |
| `internal/rotation/auth_authenticator.go` | `caaa23505bb66dbaf637c71ae8f987c88a642f04fdc647bed3b9ccb1a3d1c22a` |
| `internal/rotation/store_pg.go` | `e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8` |

## Actual call graph (traced)

### Bootstrap (production-wired)
`NewDaemon → initRotationService` (daemon.go:~308): unless `AgentBrain.DevelopmentEnabled &&
Neutral.Gateway.Required`, builds — **only when `RotationDatabaseURL` is non-empty** — a real
`rotation.NewPGStore(pool)` + `rotation.NewCredentialAuthenticator()` + `rotation.NewService(...)` and assigns
`d.rotationStore`/`d.rotationService`. Empty DB URL ⇒ WARN "rotation DISABLED", services nil.

### Path A — reactive/proactive rotation during a running task (PRODUCTION-WIRED)
Task-run loop → `rotateTaskOnExhaustion` (result text) / `maybeProactiveRotateOnText` (warning/usage banner) /
`maybeProactiveRotateFromLedger` (quota ledger) → `rotateTaskWithReason` → `d.rotationService.OnExhaustion(...)`
(service.go:`onExhaustionLocked`). All four are gated by `legacyGoRotationAllowed` → suppressed when the task's
`runtime_router_owner == rust_l2` (single-router invariant). On success returns `(account, true)`; on
`ErrNoAccountAvailable`/error returns `(Account{}, false)` and **"preserves current failure behavior"** — the
in-flight task still fails; the new assignment benefits the **next** claim only (no live task re-dispatch).

### Path B — discovery-driven reassignment (CONSUMER wired + tested; PRODUCER is a test-only bounded slice)
`wakeup.go readTaskWakeupMessages` (:275): on WS frame `daemon:credential_session_discovery` →
`dispatchAndReportCredentialSessionDiscoveryEvent` → `dispatchCredentialSessionDiscoveryEventWithOutcome`
(credential_session_monitor.go) → `(*rotation.Service).ReassignDiscoverySession` → `onExhaustionLocked`.
Alerts (WARN/ERROR/DEBUG) are emitted without secrets (covered by EV-CREDISO-5.4 + EV-CREDISO-4.4-RECORDFAIL).

## Existing bounded slice vs production-required behavior

| Concern | Existing (in repo) | Production-required for 4.3 | Gap |
|---|---|---|---|
| Reactive/proactive reassign | Path A fully wired + gated by single-router | keep | **none** (works when a task is running and Go owns routing) |
| Discovery-driven reassign consumer | Path B consumer + monitor + alerts wired & tested | keep | **none** |
| Discovery **producer** | `CredentialSessionDiscoveryProducer.Produce` called **only in tests**; `CredentialSessionDiscoveryEmitter` has **no production impl**; **no non-test code emits** `daemon:credential_session_discovery` | a production source must classify session-discovery observations and emit the event | **PRIMARY GAP** — no producer/emitter/source wired |
| Tenant/provider/session boundary | enforced in consumer (`errDiscoveryAssignmentBoundary`, `expectedAccountID` stale-compare, canonical provider) | reuse as-is | none (boundary logic complete) |
| Cross-process concurrency/lock | `agentLock` = **in-process** `sync.Mutex` only | serialize per agent across processes | **GAP** — no cross-process lock; safe only if a single daemon owns the agent's runtime |
| Atomicity Assign+RecordRotation | **non-atomic**: `Assign` persists before `RecordRotation`; no tx, no rollback (service.go `onExhaustionLocked`; documented by EV-CREDISO-4.4-RECORDFAIL) | atomic or idempotent at-least-once | **GAP** — owner architecture decision |
| Result propagation | Path A → next-claim only; Path B → alert log only | acceptable for "no manual intervention"; live in-flight switch is extra | partial (live switch out of smallest scope) |
| Logout semantics | `CredentialAuthenticator.Logout` = `os.RemoveAll` of per-account cred path in home root; runs **before** Login(next); failed Login ⇒ current already removed, account marked `degraded` | ensure per-account home isolation + no concurrent same-account use across processes | **GAP/HAZARD** — destructive logout on shared home if isolation not guaranteed |

## Smallest safe production integration (for the discovery-driven auto-reassign, Path B producer)

Reuse the fully-wired consumer; add only a producer + emitter fed by the existing session-discovery source.

- **Call site**: instantiate `NewCredentialSessionDiscoveryProducer(emitter)` and drive `Produce(ctx, obs, now)`
  from the session-discovery source (server-side `/auth/sessions` discovery / `useSessionMonitor` equivalent, or a
  daemon-side discovery loop if the daemon already holds observations).
- **Emitter (the one missing production impl)**:
  - Option 1 (server→daemon): implement `CredentialSessionDiscoveryEmitter` that publishes the existing
    `daemon:credential_session_discovery` WS frame to the daemon owning the runtime (consumer already handles it).
  - Option 2 (daemon-local loopback): emitter calls `d.dispatchCredentialSessionDiscoveryEvent` directly — mirrors
    the test `producerLoopbackEmitter`; avoids the WS hop when discovery already runs in-daemon.
- **Boundaries**: `WorkspaceID`=tenant, `Provider`, `AgentID`, `AccountID`=expected/active account — already the
  `CredentialSessionDiscoveryObservation`/payload contract; no new secret fields.
- **Concurrency**: route each agent's discovery events to the single daemon that owns its runtime (natural
  single-writer), OR add a DB advisory-lock / conditional `Assign` for cross-process safety.
- **Atomicity**: prefer a new rotation-owner `Store` method that performs Assign+RecordRotation in one tx; else
  accept documented at-least-once with idempotent `RecordRotation`. Owner architecture gate.
- **Logout**: keep Logout(current)→Login(next) ordering only under guaranteed per-account home isolation (Phase-1
  mechanism); never run against a home shared by a concurrent process.

## Exact files / symbols / tests

- Reuse (no edit): `credential_session_discovery_producer.go` (`Produce`, `CredentialSessionDiscoveryEmitter`,
  `CredentialSessionDiscoveryObservation`); `credential_session_monitor.go` (`dispatchCredentialSessionDiscoveryEvent`,
  `dispatchCredentialSessionDiscoveryEventWithOutcome`); `wakeup.go` (WS consumer :275);
  `rotation/service.go` (`OnExhaustion`, `ReassignDiscoverySession`, `onExhaustionLocked`, `agentLock`);
  `rotation/auth_authenticator.go` (`Login/Logout/WaitAuthenticated`); `rotation/store_pg.go` (`Assign`,
  `RecordRotation`); `daemon.go` (`initRotationService`, `legacyGoRotationAllowed`).

### Disjoint implementation plan (NEW files only; owner adds the single wiring line)
1. NEW `internal/daemon/credential_session_discovery_source.go` — production `CredentialSessionDiscoveryEmitter`
   + a `Run(ctx)` loop that pulls session-discovery observations and calls `Produce`. **Wiring into
   `initRotationService`/bootstrap is ONE line owned by the daemon owner (Codex1) — an owner gate, not this plan.**
2. NEW rotation-owner store method (e.g. `AssignAndRecordRotation`) for atomic Assign+RecordRotation — owner-gated
   (rotation/W-PGSTORE + migration owner); `onExhaustionLocked` would call it (service.go edit is owner-gated).
3. NEW tests (disjoint files): production-emitter loopback happy-path; cross-process single-writer (synthetic,
   two producers/one agent → one reassignment); atomicity/idempotency; logout-before-login failure isolation;
   no-secret-in-logs (reuse redact hook). All offline synthetic, no DB/network/credentials.

## Owner architecture gates (must clear before implementation)
1. **Daemon owner (Codex1)**: the single bootstrap wiring line for the discovery source; `legacyGoRotationAllowed`
   interaction (should discovery-driven reassign also be suppressed under `rust_l2`? design decision).
2. **Rotation owner (W-PGSTORE)**: atomic Assign+RecordRotation store method + any `onExhaustionLocked` edit.
3. **Cross-process concurrency policy**: confirm single-daemon-per-agent ownership vs DB-level lock.
4. **Kiro TL**: adjudicates this trace and the smallest-safe scope. No self-acceptance.

## Non-claims
- Read-only trace. No code/test/spec/task/ledger/git/index change; no DB/network/live-provider/credential/env
  access. Findings are from static source reading at the hashes above; runtime behavior not executed here.
