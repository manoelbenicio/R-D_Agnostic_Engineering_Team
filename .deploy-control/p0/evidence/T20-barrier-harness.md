# T20 — deterministic control-plane barrier harness (DESIGN; not run)

- agent: **Opus48#C** · lane **tier20-workload** · task **T20-BARRIER-HARNESS** · pane `w8:p1`
- as-of (UTC): `2026-07-23T22:20Z` · HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- lock (sole mutable): `.deploy-control/p0/evidence/T20-barrier-harness.md`
- check-in: `.deploy-control/p0/checkins/Opus48-C__T20-BARRIER-HARNESS__20260723T222029Z.json`
- companion to `T20-workload.md`. **Design only — no source edit, nothing executed; `live_runs.*=false`.**

## 0. Purpose

Give the tier-20 workload a **deterministic, LLM-independent** way to hold N concurrent lifecycle **leases** (capacity slots) open simultaneously — so strict-concurrency=20, capacity-ledger reconciliation, cancel/cleanup, and trace continuity can be proven **without any inference** (which is gated by `live_runs.*=false` and is nondeterministic). The barrier replaces the model call with a control-plane hold that releases on an explicit signal or a bounded timeout.

## 1. The lease lifecycle it targets (facts)

The capacity lease is acquired/held/released along the real path:
- **acquire**: `agentBrainRuntime.admitTask` → `r.capacity.TryBegin()` (`brain_integration.go:273`; `brain.LifecycleCapacity`, `Limit()`, `Snapshot() CapacityCounters`).
- **promote (running)**: `recordLaunch(plan)` (`brain_integration.go:598`; called `daemon.go:3728`) — lease is now an active running slot.
- **held across**: `executeAndDrainForTask(...)` (`daemon.go:3730`) — **this is where the real agent/LLM runs**; the lease is held for its whole duration.
- **release**: `recordTerminal(...)` → `capacity.Finish` (`brain_integration.go:615`; `daemon.go:3295`) — slot returns to the pool.
- **gate**: the whole Agent-Brain slice is default-off, guarded by `AGENT_BRAIN_DEVELOPMENT_ENABLED` (`config.go:696`) + gateway-required + `CapacityTier20`.

**Insight:** the lease is held precisely across `executeAndDrainForTask`. Replacing that single call with a deterministic hold (when dev-gated) keeps the lease open for a controlled duration **with no model involved**.

## 2. The dev-gated task-hold hook (for a future writer; not implemented here)

A minimal, fail-closed, dev-only hook at the execution seam:

- **Env gate (all required, else no-op / normal execution):**
  - `AGENT_BRAIN_DEVELOPMENT_ENABLED=true` (existing dev slice) AND gateway-required AND `Neutral.CapacityTier==CapacityTier20`.
  - `AGENT_BRAIN_T20_BARRIER_HOLD=1` — activates the hold.
  - `AGENT_BRAIN_T20_BARRIER_TIMEOUT_MS=<bounded>` — **mandatory** upper bound (e.g. ≤ 120000ms); absent/invalid ⇒ hold disabled (fail closed).
  - Optional `AGENT_BRAIN_T20_BARRIER_RELEASE_DIR=<path>` — control-plane release channel (see §3).
- **Placement:** in `runTask`, when the hold is active, **do NOT construct/spawn the agent backend** (`agent.New` / `executeAndDrainForTask`). Instead call `barrierHold(ctx, taskID, ...)` **after** `recordLaunch` (so the lease is promoted and counted) and **before** `recordTerminal`. This holds the lease with zero inference.
- **`barrierHold` semantics (deterministic):**
  ```text
  select {
    case <-releaseSignal(taskID):   status = "completed"          // explicit release
    case <-time.After(timeout):     status = "completed"/"timeout" // bounded auto-release
    case <-ctx.Done():              status = "aborted"/"cancelled" // cancel/watchdog
  }
  return syntheticTerminal(status)   // empty, redacted, clearly marked BARRIER_HOLD; NOT a model result
  ```
- **Terminal:** returns a deterministic synthetic `agent.Result` (status set, output empty or `"[barrier-hold]"`, `Usage={}`), flowing into the normal `recordTerminal`→`capacity.Finish` so the slot releases exactly once (reuses the real terminal/idempotency path).
- **Safety invariants:** production-off (all gates false ⇒ hook is dead code path); never spawns a CLI; never calls OmniRoute; never reads a secret; the synthetic terminal is explicitly labeled a barrier hold so it can NEVER be mistaken for or promoted to real acceptance evidence.

## 3. Release mechanism (control-plane, LLM-independent)

Three interchangeable release channels (writer picks one; all deterministic):
1. **Filesystem marker (recommended, ops-friendly):** `barrierHold` watches `AGENT_BRAIN_T20_BARRIER_RELEASE_DIR/<task_id>.release` (and `all.release`); the coordinator writes the marker to release. Bounded poll (e.g. 25ms) — deterministic, no network.
2. **Control endpoint / CLI:** `multica daemon barrier release --task <id> | --all` toggling an in-process registry the hold selects on.
3. **In-process channel (unit tests):** a `map[taskID]chan struct{}` the test closes.

Timeout is always the backstop so a lease can never hang (ties into the existing cancellation/watchdog).

## 4. Barrier coordinator (harness side; drives T20-workload)

```text
1. preflight §5 GREEN (else EXIT no-go)               # currently EXITs: live_runs.*=false
2. submit 20 read-only tasks (per T20-workload.md) with the dev-hold env active
3. WAIT until capacity Snapshot().Active == 20 (all leases held) OR abort threshold fires
   - assert Active never exceeds Limit()==20 (strict tier bound)
   - assert exactly 20 distinct leases held simultaneously (the BARRIER point)
4. HOLD-VERIFY: sample the capacity ledger + e2e spans while barriered
   - LedgerSnapshot reconciles (admitted==started==20, finished==0 mid-barrier)
   - e2e trace: each task has ingress→queue→admission→start spans (hop through the hold)
5. RELEASE all (write all.release / endpoint)
6. WAIT until Active == 0; assert each lease Finished exactly once (no dup/lost)
7. assemble e2e traces: AllContinuous across admission→...→terminal for all 20
8. emit T20-barrier-run-<ts>.md verdict (GREEN or the tripped abort threshold)
```

The barrier makes the 20-way concurrency **deterministic and simultaneous** (all held at step 3), which a real-LLM run cannot guarantee (responses complete at varying times, and inference is forbidden now).

## 5. GO / NO-GO (same gate family as T20-workload; ALL GREEN before running)

1. Operator "go".
2. Dev slice: `AGENT_BRAIN_DEVELOPMENT_ENABLED=true`, gateway-required, `CapacityTier20` + `CapacityGateEnabled`.
3. Barrier env set with a **bounded** timeout.
4. Readiness sustained Ready (strict) — admission still gates on readiness even for held tasks.
5. Note: because the barrier is inference-free, it MAY be authorized **independent of** a `live_runs` token (no model call) — but only under the dev gate and operator go; it still must NOT run without explicit authorization. `live_runs.*=false` blocks the *inference* dimension, which the barrier deliberately does not exercise.

## 6. What it proves / does NOT prove

- **Proves (deterministically, no LLM):** strict concurrency == 20; capacity ledger reconciliation (Active peaks at 20, never exceeds `Limit()`); cancel/cleanup (release/timeout → `capacity.Finish` → slot 0, exactly once); no dup/lost lease accounting; E2E trace continuity across the admission→start→hold→terminal hops (`agent-brain.e2e.v1` `Assemble.AllContinuous`).
- **Does NOT prove (out of scope):** any LLM/OmniRoute protocol/tools/reasoning/usage behavior — that is the live-run route family (gated, separate). The synthetic barrier terminal is never acceptance of a model route.

## 7. Abort thresholds (reuse T20-workload §6, barrier-specialized)

- **A1 readiness instability** — readiness flaps during the barrier → abort.
- **A2 error>10%** — >2/20 leases fail to reach the barrier or fail on release → abort.
- **A3 dup/lost lease** — any lease Finished ≠ exactly once, or Active miscount vs LedgerSnapshot → abort.
- **A4 trace breakage** — `e2e.Assemble(...).AllContinuous==false` (missing hop / orphan / conflicting join) → abort.
- **A5 tier breach** — `Snapshot().Active > 20` at any instant → abort immediately.

## 8. Scope / non-claims
- **DESIGN ONLY**: no source edited, no hook implemented, nothing executed, no task enqueued, no inference, no secret; `git diff --check` clean on this evidence file.
- The dev-gated hook is fail-closed and production-off; the synthetic barrier terminal is explicitly non-acceptance.
- Does not authorize tier activation (9.2), higher tiers, cutover, or production. A future writer implements §2/§3 and, under §5 authorization, runs the §4 coordinator once → `T20-barrier-run-<ts>.md`.
- Evidence is metadata-only (`EVIDENCE_CONTRACT` + `agent-brain.e2e.v1` `secrets_present=false`).

## 9. Status
- STATUS: DONE (design delivered; awaiting operator go + §5 + a writer).
- DELIVERED: lease-lifecycle grounding (§1), dev-gated task-hold hook spec (§2), release mechanisms (§3), barrier coordinator (§4), go/no-go (§5), proves/not-proves (§6), abort thresholds (§7). Not implemented, not run.
