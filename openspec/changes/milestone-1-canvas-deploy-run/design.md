## Context

Per master spec v4.1 §12 (user-approved), AgentVerse v1 ships as a single delivery: all 15 capabilities at once, executed in parallel by a multi-agent squad of 2+ agents starting simultaneously, on a single shared branch (master spec §14.6). This document records the cross-cutting technical decisions that span those capabilities so each agent has a consistent baseline rather than re-litigating choices in their own PR.

The starting state is a clean repository. `openspec/specs/` is empty; `src/` does not yet exist. CAO is treated as an external system reachable on `127.0.0.1:9889` (local dev) and is not modified by this change.

The user has explicitly overridden three earlier-recommended risk mitigations:

1. **No sequential bootstrap.** All agents start at once.
2. **No phased delivery.** Single ship of the entire v1 surface.
3. **Stack final decision pending Principal Architect.** Working assumption is React 18 + Vite + TypeScript; this document is contingent on that.

The Risks section is honest about what these overrides cost and what mitigations remain available within them.

## Goals / Non-Goals

**Goals:**

- Ship the complete AgentVerse v1 frontend per master spec v4.1 §12.
- Establish opinionated patterns (state, fetch, file structure, testing, design tokens) early enough that PRs converge on consistent shapes despite parallel execution.
- Make every cross-capability boundary explicit in TypeScript (shared types) so that surface contracts are checkable at compile time, not at code review time.
- Keep failure modes legible end-to-end: every CAO interaction has an observable state transition, and partial failures land in `degraded` rather than nowhere.

**Non-Goals:**

- Cloud hosting, multi-tenant auth, hosted CAO. Master spec §13 post-launch.
- Validation Proxy (server-side enforcement). Master spec §13 post-launch.
- FinOps Tier 2 (token parsing) and Tier 3 (provider billing APIs). Master spec §13 post-launch.
- Autonomous Copilot. Master spec §13 explicitly cut.

## Decisions

### D1. Stack: React 18 + Vite + TypeScript (confirmed in v4.2)

**Choice:** React 18 + Vite + TypeScript per master spec v4.2 §12. The team accepts the rework risk noted in master spec §12 ("accept rework risk if Principal Architect changes") and proceeds without further gating.

**Rationale:** Decided per user-approved override at the v4.2 mark. Earlier draft language conditioned all dependency-specific decisions on a Principal Architect verdict; that gate is now closed and the working assumption is the final answer.

**Trade-off accepted:** If a future architect revisits the stack and chooses differently, ~30% of v1 code (UI components and React-specific hooks) needs porting. The cao-integration client, Canvas Document schema, IndexedDB stores, and reconciler logic are framework-agnostic and survive.

### D2. `@xyflow/react` for the canvas graph

**Choice:** Use `@xyflow/react` (formerly React Flow) as the canvas-editor primitive.

**Rationale:** Battle-tested for node-graph editors, supports custom node renderers (we'll render our SENTINEL agent blocks), supports edge types and labels, has good touch fallbacks, is actively maintained. The Canvas Document schema maps directly onto its `nodes`/`edges` model.

**Alternatives considered:** From-scratch SVG/Canvas (4–6 weeks of avoidable work), Rete.js (less polished UX for non-engine cases), Drawflow (less actively maintained). If D1 changes to a non-React framework, the equivalent library in that ecosystem is acceptable.

### D3. State management: Zustand for UI state + TanStack Query for server state

**Choice:** Two complementary stores per master spec §14.4:

- **Zustand** for cross-component application state (current canvas, deploy progress, validated providers, voice transcript, toasts).
- **TanStack Query (React Query)** for all CAO REST endpoints (caching, polling, invalidation). Every `GET` against CAO flows through React Query with capability-owned query keys.
- **`useState`/`useReducer`** for component-local state (form inputs, UI toggles).

**Rationale:** Zustand alone would force capabilities to hand-roll polling and cache invalidation, duplicating effort across 6 ownership directories. TanStack Query already solves CAO's polling matrix (master spec §9) declaratively. Co-existence is the standard pattern. Each store has a single owner per master spec §14.4 to avoid cross-capability mutation contention.

**Alternatives considered:** Zustand-only (rejected: re-implements caching), Redux Toolkit + RTK Query (rejected: too much ceremony for parallel-agent throughput), Jotai (rejected: atom-everywhere is harder to reason about across 6 owners).

### D4. IndexedDB for local persistence behind typed interfaces

**Choice:** All local persistence (canvases, canvas_versions, provider_keys, settings) goes through IndexedDB via the `idb` library, exposed through small typed interfaces (`CanvasStore`, `KeyStore`, `SettingsStore`). Cloud persistence (Firestore) is post-launch — but the interfaces are designed so swapping the implementation is mechanical.

**Rationale:** Zero-setup local dev. The interface boundary lets the cloud implementation arrive later without touching consumers. Plaintext storage of provider keys is acceptable for v1 because the threat model is "developer's own laptop, only their key." This limitation is documented in `docs/key-storage-v1.md`.

### D5. Atomic deploy-state persistence on every transition

**Choice:** The Reconciler persists `deploy_state` to IndexedDB *before* and *after* every CAO call. Two writes per call.

**Rationale:** A page reload must never lose information about whether a CAO call was attempted. Without before-write persistence, a reload during the in-flight window of a `POST /sessions` would leave the canvas in `draft` while CAO has a session — silent leak. With before-write persistence, a reload mid-call leaves the canvas in `deploying` with a `Resume` affordance.

### D6. No Validation Proxy in v1 — supervisor obedience is best-effort

**Choice:** Do not implement the Validation Proxy in v1 (master spec §13 post-launch). Generate supervisor system prompts that *describe* the canvas topology and instruct handoff-only-to-listed-targets, but do not enforce.

**Rationale:** The Validation Proxy requires a server-side AgentVerse component, which v1 does not have. AgentVerse v1 trusts the canvas topology to flow through prompt-level instructions to the supervisor, which is sufficient for short, well-scoped tasks. Full enforcement is a customer-trust feature for cloud deployments that arrives in a later release.

**Risk captured in Risks (R3).**

### D7. WebGL is mandatory in production; no Canvas2D fallback ships

**Choice:** xterm.js loads `WebglAddon` and refuses to start the Terminal View if WebGL initialization fails. Only the `npm run dev` development build allows a Canvas2D fallback (gated behind `VITE_ALLOW_CANVAS2D=true`) for environments without GPU access.

**Rationale:** The "zero visible lag" promise depends on GPU rendering. Surfacing a "WebGL required" error is more honest than a degraded silent fallback at production scale. Master spec §6.5 lists WebGL as MANDATORY.

### D8. All agents start in parallel, on a single shared branch (user override)

**Choice:** Per master spec §14 user-approved override, the multi-agent squad starts all directory-owned work concurrently from day zero, on a single shared `main` branch. Supervisor review is required on every PR.

**Rationale:** This is a user decision. Earlier draft design recommended a Week 1 sequential vertical slice to establish patterns before parallelization; that recommendation is overridden. The single-branch / parallel-from-day-one configuration trades pattern-coherence-up-front for velocity. Mitigations live in D9 (gated `shared/`), D10 (testing), and the supervisor's review duty.

**Risk captured in Risks (R2).**

### D9. `src/shared/` is gated by the supervisor; bounded contexts otherwise

**Choice:** Cross-cutting types live in `src/shared/`, but only the supervisor agent (or a human equivalent) may merge changes there. Each capability owns its directory and imports outward only via `src/shared/` — never sideways into another capability's internals. CI flags any PR that adds a sideways import.

**Rationale:** Multi-agent merges into shared types are the main source of conflicts. Centralizing those merges through a single review gate trades latency for coherence. Inside each capability, agents have full autonomy.

### D10. Testing strategy per master spec §14.5

**Choice:**

- **Unit:** Vitest. Each agent owns tests in its capability directory (`src/<capability>/__tests__/`). Coverage target: 70% per capability for logic-heavy modules (reconciler, NLU parser, cost calculator, schema validators).
- **Component:** Vitest + `@testing-library/react`. Each agent for its own components.
- **Integration:** Vitest + MSW (Mock Service Worker). Infra Dev maintains MSW handlers covering the CAO surface; capability owners write integration tests that consume those handlers.
- **E2E:** Playwright. Supervisor authors and maintains the smoke suite that exercises critical paths (configure provider → create canvas → deploy → see terminal output → speak voice command).
- **Live contract:** A `CAO_LIVE=1`-gated test suite runs the full CAO client surface against a live CAO instance and asserts response shape; runs nightly in CI when CAO is reachable.

**Definition of done per task** (master spec §14.5):
- Code compiles with zero TypeScript errors.
- Unit tests pass for new logic.
- Component renders without errors in tests.
- PR reviewed and approved by supervisor.

### D11. CAO client exposes the full v1 surface

**Choice:** The `cao-integration` HTTP client exposes the full CAO REST surface needed by v1 (sessions, terminals, profiles, providers, flows, memory-context, skills, settings) plus the PTY WebSocket helper. Contract tests gate the surface against drift.

**Rationale:** Per §12 user override, all features ship together. The earlier "M1 needs only a subset" boundary is no longer applicable.

### D12. Same-branch coordination model

**Choice:** All agents push to a single shared branch (master spec §14.6). PRs target that branch, supervisor reviews and merges. No long-lived feature branches.

**Rationale:** With 6+ ownership directories, long-lived branches diverge faster than they can be reconciled. Same-branch is acceptable when (a) ownership is directory-bounded so cross-cutting conflicts are rare, and (b) every PR is reviewed before merge so failed merges are caught at PR time.

**Trade-off:** A bad merge from one agent can transiently break another. Mitigation: CI runs typecheck + unit tests on every PR; broken main triggers immediate revert at supervisor discretion.

### D13. JetBrains Mono everywhere as default; user-configurable in Settings

**Choice:** Per master spec §14.7 (user-approved override of §3.3), JetBrains Mono is the default font for body, headings, and terminal/code. Settings → Appearance exposes a font picker that lets the user change UI body, headings, and terminal fonts independently. Inter remains a built-in option.

**Rationale:** Mainframe/tmux aesthetic by default; user choice for those who want softer typography.

### D14. Diff-based edit-after-deploy in v1 (per master spec §12 user override)

**Choice:** When a user edits a canvas that is `deployed` or `degraded`, the Reconciler computes the delta between the current `CanvasDocument` and the recorded actual state, and applies only the differences (create new terminals, kill removed ones, update changed profiles). No full redeploy.

**Diff strategy:**
- Node added → install profile, add terminal.
- Node removed → kill terminal, snapshot saved.
- Node profile content changed (system_prompt, allowedTools, model, provider) → install new profile, kill old terminal, add new terminal.
- Node display-only change (display_name, position) → no CAO action; persist canvas.
- Edge added/removed → no CAO action in v1 (edges affect supervisor prompt regeneration only on full redeploy of supervisor; v1 documents this limitation; user-visible warning when an edge change requires a supervisor restart).
- Entry-point changed → blocked: requires Tear Down + redeploy. The diff path does not move the supervisor.

**Risk captured in Risks (R5).**

### D15. Voice NLU runs on the user's BYOK LLM key

**Choice:** Per master spec §16.1, the NLU layer that converts a voice transcript into a `CreateCanvasIntent` issues a structured-extraction call against the user's own LLM key. The system selects the cheapest available validated key in this preference order: Gemini Flash → GPT-4o-mini → Haiku. If no LLM key is validated, voice input is disabled with an inline message directing the user to Settings.

**Rationale:** AgentVerse never pays for inference (BYOK is the cost model). The cheapest available extraction model is sufficient for the structured-JSON task.

## Risks / Trade-offs

- **[R1] Stack rework risk acknowledged.** The v4.2 stack decision (React 18 + Vite + TypeScript) is final. If a future architect revisits the choice, ~30% of v1 code (UI components and React-specific hooks) needs porting; the framework-agnostic layers (cao-integration client, Canvas Document schema, IndexedDB stores, reconciler logic) survive. The team accepts this risk per master spec §12.
  → *Mitigation:* Capability boundaries (D9) localize the blast radius. The cao-integration client deliberately uses plain TypeScript and `fetch` (not React-specific hooks) so it ports cleanly.

- **[R2] Parallel-from-day-zero increases pattern-coherence risk.** Without a sequential bootstrap, six capabilities will land their first PRs simultaneously. State patterns (D3), fetch patterns, component patterns, and test patterns will diverge if not actively reconciled.
  → *Mitigation:* The supervisor reviews every PR (master spec §14.6). Early-cycle PRs are explicitly tagged "patterns" and rejected if they invent unnecessary novelty. A `docs/patterns/` directory captures decisions as they crystallize. Some thrash in week 1 is expected and accepted as the cost of velocity.

- **[R3] Supervisor LLM disregards canvas topology because no Validation Proxy enforces it (D6).** A misbehaving supervisor calls `handoff("anything")` and CAO obeys.
  → *Mitigation:* v1 templates are designed for short, well-scoped tasks where supervisor drift is unlikely. Demo scripts do not depend on the supervisor staying inside the canvas if the prompt isn't precise. The Validation Proxy remains tracked as the long-term fix (post-launch).

- **[R4] Same-branch coordination breaks when a PR introduces a typecheck failure that survives to main.** Other agents are blocked until the regression is fixed.
  → *Mitigation:* CI gates merge on typecheck + unit tests + lint. Supervisor must revert promptly if main breaks. A "main is green" rule is binding.

- **[R5] Diff-based edit-after-deploy under partial failure.** A user edits a canvas in `degraded` state, the diff sees the partial actual-state, and the apply-delta logic produces a sequence that double-creates or fails to clean up.
  → *Mitigation:* Reconciler computes diffs against the canonical `terminal_map`, not against a fresh `GET /sessions/...`, so partial state is reflected accurately. Edit-from-degraded is allowed only after the user has reviewed and acknowledged the partial state. Edge cases are covered by spec scenarios in `canvas-reconciler/spec.md`.

- **[R6] Single delivery means no incremental customer feedback.** Without a phased rollout, the team cannot validate any one piece against real users until everything is ready.
  → *Mitigation:* The Playwright smoke suite serves as a synthetic end-to-end customer. It runs against a recorded CAO container in CI and against a live CAO weekly. Internal demos at every supervisor sync (recommended weekly) catch usability regressions before delivery.

- **[R7] WebGL hardware diversity may surprise the team late.** Some developer machines, headless CI, and entry-level Chromebooks lack WebGL2 capabilities.
  → *Mitigation:* Production refuses Canvas2D fallback (D7). CI runs xterm tests in headed Chrome with WebGL flags forced on. The error message in production tells users exactly why the terminal won't start.

- **[R8] CAO API contract drift breaks v1 silently.** A non-AgentVerse change to CAO's REST shapes lands and the client misreads responses without throwing.
  → *Mitigation:* The `CAO_LIVE=1` contract test suite (D10) runs nightly when CAO is reachable and posts a diff if any endpoint shape changes. The Infra Dev role owns this suite.

- **[R9] BYOK plaintext IndexedDB key storage leaks via shared dev-tools demos.** A developer recording a demo accidentally exposes a key.
  → *Mitigation:* Documented threat model in `docs/key-storage-v1.md`. Settings page redacts keys to `sk-…XXXX`. Encrypted Firestore storage is on the post-launch list. Dev-tools use is at the user's risk; the documentation flags it.

## Migration Plan

There is no existing system to migrate. The deployment plan is:

1. Provision the repository, CI, and ownership boundaries (master spec §14.2 directories).
2. All squad members open initial PRs against the same shared `main` branch — application shell, design system seeds, CAO client skeleton, IndexedDB infra — supervisor reviews and merges.
3. Capability work proceeds in parallel across the directory-owned partitions.
4. Continuous green main: every supervisor merge keeps typecheck + unit tests passing.
5. At v1 done, run Playwright smoke against a fresh checkout + a fresh CAO and capture screenshots/recordings as proof artifacts.
6. Tag `v1.0.0`. Begin post-launch work (Validation Proxy, FinOps Tier 2, Cloud Runtime).

Rollback is trivial: v1 ships nothing to production. If the change is abandoned mid-flight, the only consequence is unused code in the repository.

## Open Questions

All five open questions raised in v4.1 have been resolved by master spec v4.2. They are recorded here as resolved decisions for traceability.

1. ~~**Principal Architect stack verdict.**~~ **Resolved (v4.2 §12):** React 18 + Vite + TypeScript is the final stack. See D1.

2. ~~**Default model for templates.**~~ **Resolved (v4.2 §8.10):** "Model dropdown shows all available models for the provider — no default, no recommendation." Templates inherit the user's `provider_default` and the user explicitly selects the model. No special template defaults are applied.

3. ~~**First-run wizard scope.**~~ **Resolved (v4.2 §12):** Users can browse the full UI without keys (read-only). Deploy is the only gate; it is disabled until at least one provider is validated. The wizard is offered but never required. See `health-and-onboarding/spec.md` and `canvas-builder/spec.md`.

4. ~~**Cron schedule editor visual UX.**~~ **Resolved (v4.2 §12):** Presets + manual override. The `flows` capability ships with quick-pick presets (every-N-minutes, hourly, daily-at-time, weekdays-at-time, weekly) and a raw cron input for power users, with `cronstrue` for human-readable explanations.

5. ~~**Activity feed ordering and retention.**~~ **Resolved (v4.2 §12):** Unlimited retention, browser-managed. The Dashboard activity feed does not impose an application-level cap. The browser's memory limits are the natural ceiling. See `dashboard/spec.md`.
