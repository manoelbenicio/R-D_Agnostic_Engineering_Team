# Independent Design Review — credential-isolation 4.3 production emitter + atomicity/logout

- reviewer: Kiro / Opus-4.8, wave w8:p1 (independent; distinct pane from design author w8:p2)
- date: 2026-07-18T22:30:00Z
- mode: READ-ONLY except this artifact. No source/tests/shared-planning/spec/tasks/git/index/refs edit; no credentials/env/network/DB/services. No implementation authorization.

## Check-in / check-out
- CHECK-IN 2026-07-18T22:23:00Z — Kiro/Opus-4.8 w8:p1 — stream CREDISO-4.3-EMITTER-ATOMICITY-DESIGN-INDEPENDENT-REVIEW — READ-ONLY.
- CHECK-OUT 2026-07-18T22:30:00Z — DONE. Verdicts below. Owner policy + Kiro TL adjudication required. Not self-accepted.

Reviewed: `credential-isolation-4.3-production-emitter-atomicity-design.md` — SHA-256 `a06b8b5a7a81e16fc9edec71fc7d1d7c2fd69e6579ebffcc28cd7239c2fc0472` (matches asserted `a06b8b5a…`; stable). Task **4.3 `[ ]` and 4.4 `[ ]` remain OPEN** (verified; preserved).

## VERDICTS (separate)

- **Source-claim accuracy: PASS.** All five "verified-from-source" claims independently confirmed.
- **Alternatives A/B/C/D: sound and correctly ranked for reversibility.**
- **Implementation readiness: CONDITIONAL-READY** — the additive NEW-file skeleton is implementation-ready and reversible; concrete variant selection + owner-gated edits are **BLOCKED on 4 owner policies** (kept strictly separate from readiness below).

## Source-claim verification (independent, at the design's pinned hashes)

| Design claim | Verdict | Evidence |
|---|---|---|
| Producer complete; **only tests call `Produce`** (no production emitter/source) | ✅ | `.Produce(` and `NewCredentialSessionDiscoveryProducer` occur **only** in `_test.go` (`credential_session_discovery_producer_test.go`, `credential_rotation_task53_test.go`); the production `EmitCredentialSessionDiscovery` interface (producer.go:46) has no production implementer/caller. |
| Consumer complete: `wakeup → ReassignDiscoverySession → onExhaustionLocked`, stale-compare `expectedAccountID` | ✅ | `discovery_reassignment.go` serializes on `agentLock`, no-ops when `currentAccountID != expectedAccountID`, validates provider/tenant, sets `StatusExhausted`, then `onExhaustionLocked`. |
| **Assign + RecordRotation non-atomic; RecordRotation failure leaves assignment written, no rollback** | ✅ | `service.go:97-166` `onExhaustionLocked`: `store.Assign(...)` then `store.RecordRotation(...)`; on RecordRotation error → `return Account{}, err` with assignment already committed. `PGStore.Assign` (store_pg.go:127-139) is a single `pool.Exec` upsert `ON CONFLICT (agent_id)`, separate call from RecordRotation. |
| **Destructive logout before login retry** | ✅ | `onExhaustionLocked`: `Logout(current)` runs **before** the `for attempts` `Login(next)` loop; on total login failure → `return Account{}, lastLoginErr` with current's HOME already `RemoveAll`-ed and no restore. |
| Concurrency = **in-process `sync.Mutex` only** | ✅ | `service.go:168-177` `agentLock` returns `*sync.Mutex` from `agentLocks map[string]*sync.Mutex`; no DB/cross-process lock. |

Input-manifest corroboration: the design's `credential_session_monitor.go` hash `936b3e40…` matches current disk (the same "drift" blob the push-candidate matrix flagged) — confirming the design was written against **current** bytes, not stale ones.

## Alternatives A/B/C/D — assessment

- **A (transport/source): T1 loopback + S1 daemon-loop behind a default-OFF flag** — sound; mirrors the test `producerLoopbackEmitter`; flag-off = exact current behavior; observation contract already non-secret. Correct reversible pick; T2/S2 correctly reserved for cross-process.
- **B (locking): CPL3 ownership + CPL1 CAS net; CPL2 only if multi-writer** — sound. CPL1 (`WHERE account_id = $expectedFrom`) reuses the `expectedAccountID` already carried by `ReassignDiscoverySession` and needs no schema change.
- **C (atomicity): AT1 optional interface with type-assert + fallback** — correct and **test-preserving**: I count ≥5 synthetic stores implementing `Assign`/`RecordRotation` (producer_test, daemon_test, discovery_reassignment_test, service_test, record_failure_alert_test). **AT2 (extend `Store`) would break all of them** — correctly rejected. AT1 is the reversible choice.
- **D (logout): L1 reorder if per-account HOME isolation guaranteed, else L2 compensation; L3 rejected** — sound and correctly conditioned on `next.HomeRoot/RelPath ≠ current.HomeRoot/RelPath` (shared `HomeDir`+vendor would collide).

## Hidden coupling / migration risks (surfaced; several the design flags, some it does not)

1. **Discovery-observation data source may not exist (S1 prerequisite).** S1 presumes the daemon can produce `session status + expires_at` observations in-process. Per my prior `credential-isolation-session-api` review, there is **no Go `/auth/sessions` discovery endpoint**; the in-daemon observation source is itself unbuilt. The design's stop-condition #3 flags infeasibility but treats the source as assumed-available — **this is a real upstream dependency, not just a transport choice.**
2. **AT1 fallback preserves the bug.** The `else { Assign; RecordRotation }` path keeps the non-atomic behavior for any store not implementing `AtomicRotationStore`. Therefore the proposed **rollback test (#3) must exercise the atomic implementation, not a fallback synthetic store** — otherwise the atomicity assertion is vacuous. (Design does not call this out explicitly.)
3. **CPL1 ↔ AT1 coupling.** CPL1's CAS safety exists **only inside** the AT1 transaction; adopting CPL1 without AT1 yields no atomic guarantee. They must ship together.
4. **A third non-atomic write is outside AT1's scope.** `ReassignDiscoverySession` calls `UpdateAccountStatus(current, StatusExhausted)` **before** `onExhaustionLocked`, un-transacted. If the subsequent reassign fails, the account is already marked exhausted. AT1 wraps only Assign+RecordRotation — so **three** state mutations exist and only two are atomized. Design scope gap worth an owner note.
5. **`rust_l2` / `legacyGoRotationAllowed` cross-lane coupling.** Whether discovery-driven Go reassignment should run at all under Rust-L2 routing (design stop-condition #4) is entangled with the **Prodex-vs-OmniRoute runtime-authority posture** I audited separately (persist-prodex supersession). This is a cross-lane governance coupling, not just a daemon-owner wiring decision.
6. **Flag-off ⇒ 4.3 stays behaviorally OPEN.** Shipping the additive code with the default-OFF flag does not close 4.3 functionally until the owner enables the flag and wires the source — correct for reversibility, but the task cannot be marked done on code merge alone.

## Implementation readiness (separate from owner policy)

- **Additive skeleton = READY.** The two NEW files (`internal/rotation/atomic_store.go`, `internal/daemon/credential_session_discovery_source.go`), the optional `AtomicRotationStore` interface, CPL1 predicate, default-OFF flag, and offline synthetic tests are implementation-ready, reversible, test-store-preserving, and require **no schema migration**. NEW-only scoping avoids owner hotspots.
- **Concrete variant + owner-gated edits = BLOCKED (policy, not readiness).** The exact code shape depends on four unresolved owner policies: (a) per-account HOME-isolation guarantee → L1 vs L2; (b) multi-writer-per-agent reality → CPL3 vs CPL2; (c) in-daemon discovery-observation availability → S1/T1 vs S2/T2; (d) `rust_l2` interaction. The owner-gated edits (`service.go onExhaustionLocked`, `daemon.go initRotationService`) touch hotspots and must not be performed under this design alone.
- **Net:** design is technically build-ready as an additive slice; which variant is built and whether the hotspot edits proceed await owner adjudication. **These are distinct axes** — readiness is high; authorization is absent.

## Explicit non-claims
- Created only this file. No source/test/spec/tasks/shared-planning/git/index/ref edit; no credentials/env/network/DB/services. No task checkbox changed; **4.3 and 4.4 remain OPEN.**
- Verified behavior statically from source at the design's pinned hashes; ran no build/test.
- I authorize no implementation and select no A/B/C/D variant or owner policy. Owner + Kiro TL adjudicate.
