# validation-proxy — Implementation Tasks

> Owner: **SUP** (`src/shared/`) for the topology model + guard; **IF** for the
> eventual CAO-side interception. R3 runtime defense-in-depth on top of the
> v1 prompt-level mitigation (`docs/canvas-topology-prompt.md`).

## 1. Topology model + guard (SUP) — DONE

- [x] 1.1 `src/shared/topology-guard.ts`: compile a `CanvasDocument` into a
      queryable `CanvasTopology` (`buildCanvasTopology`).
- [x] 1.2 Resolve agent identities through an alias table (node id,
      `profile_name`, generated `<profile>_<id>`, `display_name`) so calls
      using any identifier CAO/the prompt emit are recognised.
- [x] 1.3 `validateOrchestrationCall(topology, { action, source, target })`
      allows a call only when an edge of the matching `OrchestrationType`
      exists source→target; otherwise blocks with an explicit, auditable
      reason + machine code (`unknown-source` / `unknown-target` /
      `edge-not-allowed`).
- [x] 1.4 `validateAgainstCanvas(canvas, call)` convenience wrapper.
- [x] 1.5 Pure + dependency-free (only `@/shared` types) so the same module
      runs SPA-side now and lifts into a CAO-side proxy later unchanged.
- [x] 1.6 Unit tests `src/shared/__tests__/topology-guard.test.ts` covering
      valid edges, alias resolution, wrong-action, no-edge, unknown
      source/target, degraded edges, and audit-reason content.

## 2. Interception seam + CAO-side install (IF) — PARTIAL (SPA-side core DONE)

Blocked on the cloud runtime: agent→agent `handoff`/`assign`/`send_message`
happen *inside* CAO, not through SPA HTTP calls (the SPA's
`sendInboxMessage`/`sendTerminalInput` are user→terminal). Enforcement that
actually *prevents* a violation must sit in front of the CAO orchestration
endpoints.

- [ ] 2.1 Decide install point: CAO middleware vs. sidecar proxy (folds into
      `cloud-runtime-deployment` auth-proxy decision). **BLOCKED — human
      decision.** Options documented inline at the top of
      `src/shared/validation-proxy.ts`: (a) CAO middleware [strongest, blocked
      on CAO source — not in this repo], (b) sidecar proxy [needs cloud-runtime
      auth-proxy], (c) SPA-side observe-only [cannot prevent]. Do not write CAO
      server code until chosen.
- [x] 2.2 Compile the deployed canvas topology and resolve identities via
      `deploy_state.terminal_map` (node_id→terminal_id). Done in
      `createValidationProxy(canvas)` — inverts the map so CAO calls addressed
      by terminal id resolve back to node ids before the guard runs.
- [x] 2.3 (core) Install-point-agnostic decision core:
      `guardOrchestration(call)` → `{ allowed, reason, code }`, delegating to
      `validateOrchestrationCall` and emitting an auditable `ViolationRecord`
      via the injectable `onViolation` sink on every denial. The HTTP 4xx
      wiring remains blocked on §2.1.
- [ ] 2.4 Surface violations to the SPA (toast + health/log panel) and add
      contract + integration tests against valid / invalid / degraded canvases.
      **Pending** — unit tests for the core land in
      `src/shared/__tests__/validation-proxy.test.ts` (allow, deny by missing
      edge, deny by wrong type, terminal_map resolution, onViolation firing);
      UI surfacing + contract/integration tests await the install point.

## Out of scope

- Removing the v1 prompt-level `canvas-topology` block — it stays as
  defense-in-depth.
- Mutating canvas edges from the proxy (read-only enforcement only).
