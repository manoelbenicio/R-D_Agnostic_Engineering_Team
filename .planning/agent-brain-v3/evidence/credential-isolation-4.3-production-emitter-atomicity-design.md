# Credential-isolation 4.3 — production emitter + atomicity/logout design (ADVISORY, implementation-ready)

Design resolving the task-4.3 production gaps: producer/emitter construction & wiring, transport boundary,
cross-process locking, Assign+RecordRotation atomicity/rollback, and destructive-logout failure behavior.
**Advisory only — no implementation.** Owner policy + Kiro TL adjudication required before any coding.

- Author: **Kiro/Opus-4.8, session `w8:p2`** (read-only design). Owner gates: daemon-owner (Codex1),
  rotation-owner (W-PGSTORE), Kiro TL. HEAD `b6571299`.
- Task 4.3 (`openspec/changes/agent-credential-isolation/tasks.md:27`, file SHA
  `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3`):
  "Reatribuir o agente à nova conta sem intervenção manual."

## CHECK-IN 2026-07-18T22:00:00Z
Mode: READ-ONLY design. Sole writable deliverable = this file. Excluded (honored): no source/test/spec/
tasks/shared-planning/git/index/refs edit; no credentials/env values; no DB/network/live services.

## Input manifest (SHA-256, hashed this session)
| File | SHA-256 |
|---|---|
| `internal/daemon/credential_session_discovery_producer.go` | `4e240c09af1653fd2ecdbcd0763f8c05d3eb3f57ee8b9c339045edfa32a9ce6c` |
| `internal/daemon/credential_session_discovery_producer_test.go` | `818e69d4fe04057503b776716ae0704a6b27b7fc7b2b875e8f507b9a7b99491a` |
| `internal/daemon/credential_session_monitor.go` | `936b3e40f19c699078e994740c9ae63fdce5c172516b678d289f05c3cc38d1e2` |
| `internal/daemon/wakeup.go` | `71b89980c9b8cb7c03a81a502da5b9f4bdab4c0c522c117d2147b670e050f730` |
| `internal/daemon/daemon.go` | `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07` |
| `internal/rotation/service.go` | `f20951a36765d2b6b8576232871f1991697de56d8577377b0226ed5e30130cf0` |
| `internal/rotation/discovery_reassignment.go` | `c655c9c4f94716d2c93109b1cc4d3c14e62d6fe8dea615dddec3ec93769b2832` |
| `internal/rotation/store_pg.go` | `e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8` |
| `internal/rotation/auth_authenticator.go` | `caaa23505bb66dbaf637c71ae8f987c88a642f04fdc647bed3b9ccb1a3d1c22a` |
| prior trace `credential-isolation-4.3-production-integration-gap-trace.md` | `d793712503932cee5eb9757f18d252e6a89da8c2ff63a4cdd6874ba0790454b4` |

## Current state (verified from source)
- **Producer COMPLETE:** `CredentialSessionDiscoveryProducer.Produce` validates the non-secret observation,
  dedups (process-local, TTL, size-bounded, in-flight-safe), and calls
  `emitter.EmitCredentialSessionDiscovery(ctx, protocol.Message)`. **Missing = a production emitter + a
  source that calls `Produce`** (only tests do today via `producerLoopbackEmitter`).
- **Consumer COMPLETE:** `wakeup.go:~275` → `dispatchCredentialSessionDiscoveryEventWithOutcome` →
  `(*rotation.Service).ReassignDiscoverySession` (validates tenant/provider/`expectedAccountID` stale
  compare) → `onExhaustionLocked`. Alerts redacted (EV-CREDISO-5.4 / -4.4-RECORDFAIL).
- **Store:** `PGStore.Assign` (single upsert `ON CONFLICT (agent_id)`) and `RecordRotation` (single insert)
  are **separate `pool.Exec` calls — no shared tx**. `onExhaustionLocked` does `Assign` then `RecordRotation`;
  a `RecordRotation` failure leaves the assignment written, **no rollback**.
- **Authenticator:** `Logout(acc)` = `os.RemoveAll(HomeRoot/RelPath)` (destroys the per-account HOME copy;
  the SOURCE `ConfigDir` copy persists). `onExhaustionLocked` calls `Logout(current)` **before** the
  `Login(next)` retry loop → on **total login failure** current's home copy is destroyed with no restore.
- **Concurrency:** `agentLock` is an **in-process `sync.Mutex`** only (safe single-daemon-per-agent).

---

## Concern A — producer construction/wiring, emitter, transport boundary
**Transport alternatives**
- **T1 daemon-local loopback** — emitter calls `d.dispatchCredentialSessionDiscoveryEvent(ctx, msg, now)`
  directly (mirrors the test `producerLoopbackEmitter`). No WS hop. Requires the discovery source to run in
  the **same daemon that owns the agent's runtime**.
- **T2 WS publish (server→daemon)** — emitter marshals the existing `daemon:credential_session_discovery`
  frame and publishes to the owning daemon; the `wakeup.go` consumer already handles it. Needed when
  discovery runs server-side / cross-process.

**Source alternatives**
- **S1 daemon-side loop** — `Source.Run(ctx)` collects `CredentialSessionDiscoveryObservation`s from the
  daemon's own session state and calls `Produce`.
- **S2 server-side** — `/auth/sessions`/`useSessionMonitor`-equivalent feeds observations, emits via T2.

**Recommended (reversible):** **S1 + T1 behind a feature flag**, under the single-daemon-per-agent
invariant; **T2** is the cross-process generalization. Flag-off = today's behavior exactly (producer stays
test-only). No secret fields added — the observation contract is already non-secret.

## Concern B — cross-process locking
- **CPL1 optimistic CAS** — conditional `Assign` (`WHERE assignments.account_id = $expectedFrom`) inside the
  atomic tx; `ReassignDiscoverySession` already carries `expectedAccountID`. If another process rotated
  first, 0 rows affected → return `reassigned=false` (no-op). Lightest; no new infra; no schema change.
- **CPL2 pg advisory lock** — `pg_advisory_xact_lock(hashtext(agent_id))` at tx start → hard cross-process
  serialization. Stronger; one extra call.
- **CPL3 single-writer routing** — route each agent's discovery events to the one daemon owning its runtime;
  keep the in-proc mutex. Zero DB change; relies on the ownership invariant.

**Recommended (reversible):** **CPL3 (ownership invariant) as primary + CPL1 (conditional Assign) as the
in-tx safety net.** Add **CPL2** only if multi-writer per agent is unavoidable. CPL1 is purely additive.

## Concern C — Assign+RecordRotation atomicity/rollback
- **AT1 optional atomic interface (RECOMMENDED)** — add
  `type AtomicRotationStore interface { AssignAndRecordRotation(ctx, agentID, fromID, toID string, reason RotationReason, at time.Time, expectedFrom string) error }`.
  `PGStore` implements it via `pool.Begin(ctx)`: conditional `Assign` (CPL1) + `INSERT rotation_events`,
  then `Commit`; any error → `Rollback` (assignment **not** written). `onExhaustionLocked` does
  `if a, ok := s.store.(AtomicRotationStore); ok { a.AssignAndRecordRotation(...) } else { Assign; RecordRotation }`.
  **No `Store` interface break** → synthetic test stores keep working via the fallback.
- **AT2 extend `Store`** — add the method to `Store` directly; **breaks every synthetic store** (must add
  the method). Rejected as non-reversible-friendly.
- **AT3 idempotent at-least-once** — keep two writes; make `RecordRotation` idempotent (unique key) + retry;
  accept documented non-atomicity. Weakest.

**Recommended (reversible):** **AT1.** Fixes the EV-CREDISO-4.4-RECORDFAIL non-atomic observation (rollback
on record failure) while preserving all existing test stores.

## Concern D — destructive logout failure behavior
- **L1 reorder "prepare-next, commit, logout-old-last"** — `Login(next)`+`WaitAuthenticated`+atomic
  `Assign+Record` FIRST, then `Logout(current)` LAST. On any pre-commit failure, **current is untouched**.
  **Requires per-account HOME isolation** (`next.HomeRoot/RelPath ≠ current.HomeRoot/RelPath`), else
  `Login(next)` would overwrite the shared path current still uses. (`defaultCredentialPaths` keys on
  `acc.HomeDir`+vendor `RelPath`; two accounts sharing `HomeDir`+vendor **collide** → not isolated.)
- **L2 compensating rollback** — keep logout-first, but on **total** login failure re-restore current
  (`Login(current)` from its persistent `ConfigDir` source) to undo the destructive logout. Works even with
  a shared home (sequential); bounded.
- **L3 status quo** — current destroyed on total failure. Not acceptable for production.

**Recommended (reversible):** **L1 when per-account HomeDir isolation is a guaranteed invariant** (Phase-1
credential-isolation distinct homes) — cleanest, no destruction; **else L2 compensation.** Owner confirms
which isolation guarantee holds.

---

## Exact proposed files / interfaces / tests (NEW-only; owner-gated edits flagged)
**NEW files (additive, no owner hotspot):**
- `internal/daemon/credential_session_discovery_source.go` — production `CredentialSessionDiscoveryEmitter`
  (T1 loopback) + `type CredentialSessionDiscoverySource struct{…}` with `Run(ctx)` (S1); flag-gated.
- `internal/rotation/atomic_store.go` — `AtomicRotationStore` interface + `PGStore.AssignAndRecordRotation`
  (tx: conditional Assign + rotation_events insert). *(Placing the PGStore method here keeps `store_pg.go`
  untouched; rotation-owner may prefer it in `store_pg.go` — owner call.)*

**OWNER-GATED edits (identify, do not perform):**
- `internal/rotation/service.go onExhaustionLocked` — type-assert `AtomicRotationStore`; apply L1 reorder or
  L2 compensation. (Rotation owner.)
- `internal/daemon/daemon.go initRotationService` — **ONE** flag-gated wiring line constructing the source +
  `go src.Run(ctx)`; decide the `legacyGoRotationAllowed`/`rust_l2` interaction (should discovery-driven
  reassign also be suppressed under Rust-L2 routing?). (Daemon owner Codex1.)

**NEW tests (disjoint, offline synthetic — no DB/network/creds):**
1. producer→T1-loopback happy path (exhausted → one reassignment).
2. CPL1 CAS race: two producers, one agent → exactly one reassignment; the loser no-ops (`reassigned=false`).
3. AT1 rollback: `RecordRotation` fails inside tx → assignment **not** written (via a synthetic store that
   fails the record leg; asserts CurrentAssignment unchanged).
4. D-logout isolation: total login failure → current home preserved (L1) or restored (L2).
5. no-secret-in-logs on the new source/emitter path (reuse `redact` hook; assert redacted alerts).

## Migration / compatibility
- **AT1 is optional** → synthetic stores unaffected (fallback). **No `Store` interface break.**
- **CPL1** uses existing `assignments`/`rotation_events` columns → **no schema migration**; it only adds a
  `WHERE account_id = $expectedFrom` predicate and wraps both writes in one tx.
- **Feature flag** (e.g. `MULTICA_DISCOVERY_REASSIGN_ENABLED`, default **OFF**) → zero behavior change until
  the owner enables it; fully reversible by flag-off.
- **L1/L2** change timing/compensation only, no schema; active only on the discovery-driven reassign path.
- The DB-gated `!offline` E2E tests remain untouched; all new tests are `offline`/synthetic.

## Stop conditions (halt + escalate to owner/TL)
1. **Home isolation unverified** and L1 chosen → would corrupt a shared home; use L2 instead. STOP.
2. **Real multi-writer per agent** and CPL3 ownership not guaranteeable → CAS alone is insufficient; require
   CPL2 advisory lock. STOP for owner concurrency policy.
3. **Discovery observations not available in-daemon** → S1/T1 infeasible; must use S2 server-side + T2 WS.
4. **`legacyGoRotationAllowed`/`rust_l2` interaction undecided** → daemon owner must rule whether
   discovery-driven reassign is suppressed under Rust-L2 routing before wiring.
5. Any edit would touch a shared hotspot (`daemon.go`, `service.go`, `store_pg.go`) without authorized
   owner scope → STOP; this design only authors NEW files and *proposes* the owner-gated edits.

## Non-claims
- Advisory design only; **implements nothing**; no source/test/spec/tasks/shared-planning/git/index/ref
  change; no credentials/env values; no DB/network/live services. Behavior described from static source at
  the hashes above (not executed). Owner policy + Kiro TL adjudication required.

## CHECK-OUT 2026-07-18T22:05:00Z — DONE
Only this file created. Everything else unchanged.
