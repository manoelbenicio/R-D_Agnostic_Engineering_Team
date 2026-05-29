# v1 Decisions Log

> Open questions raised during planning of `milestone-1-canvas-deploy-run`
> and the decisions that closed them. v4.2 spec is the locked baseline;
> deltas surfaced post-baseline live as separate change proposals under
> `openspec/changes/`.

## How to read this file

- **Question** — original ambiguity or fork in the road.
- **Decision** — what we shipped in v1.
- **Rationale** — why that path, in 1–3 sentences.
- **Locked by** — file/section where the decision is now authoritative.
- **Follow-up** — if any, the change proposal that owns the residual work.

## Decisions

### D-001: Stack choice — React 18 + Vite + TypeScript

- **Question:** The master spec v4.1 conditioned the stack on a "Principal Architect verdict." Should the team block until that verdict arrives, or proceed with the working assumption?
- **Decision:** React 18 + Vite + TypeScript is final per v4.2 §12. The team proceeded without further gating. ~30% rework risk accepted — framework-agnostic layers (CAO client, schema, IndexedDB stores, reconciler) survive a future stack swap.
- **Rationale:** Blocking 6 agents on an external verdict defeats the parallel-from-day-zero mandate. The v4.2 override closed the question permanently.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D1, lines 33–39)
- **Follow-up:** none

### D-002: Canvas graph library

- **Question:** Build the canvas editor from scratch (SVG/Canvas), or adopt a library? If a library, which one?
- **Decision:** `@xyflow/react` (formerly React Flow). Its `nodes`/`edges` model maps directly onto the Canvas Document schema.
- **Rationale:** Battle-tested, actively maintained, supports custom node renderers and edge types. Building from scratch was estimated at 4–6 weeks of unnecessary work.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D2, lines 41–47)
- **Follow-up:** none

### D-003: State management pattern

- **Question:** Zustand-only, Redux+RTK Query, Jotai atoms, or split store?
- **Decision:** Split: Zustand for cross-component UI state, TanStack Query for all CAO REST endpoints, `useState` for component-local state.
- **Rationale:** Zustand alone forces hand-rolled polling and cache invalidation across 6 ownership directories. TanStack Query solves CAO's polling matrix declaratively. Co-existence is standard.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D3, lines 49–59)
- **Follow-up:** none

### D-004: Local persistence — IndexedDB plaintext keys

- **Question:** How to store provider API keys for local dev? Encrypted? OS keychain? Plaintext IndexedDB? What about cloud?
- **Decision:** Plaintext IndexedDB via the `idb` library behind typed interfaces (`KeyStore`, `CanvasStore`, `SettingsStore`). Cloud persistence (Firestore with encrypted fields) is post-launch.
- **Rationale:** Zero-setup local dev. The v1 threat model is "developer's own laptop, only their key." Interface boundaries allow swapping the backend to Firestore later without touching consumers.
- **Locked by:** `src/api/key-store/index.ts` (KeyStore interface) and `docs/key-storage-v1.md` (threat model)
- **Follow-up:** none (cloud encryption tracked under `openspec/changes/cloud-runtime-deployment/`)

### D-005: Atomic deploy-state persistence

- **Question:** Should the Reconciler persist state only after CAO calls complete, or also before?
- **Decision:** Two writes per CAO call — persist `deploy_state` before AND after. A page reload mid-call lands the canvas in `deploying` with a Resume affordance, never in a silently inconsistent state.
- **Rationale:** Without before-write persistence, a reload during an in-flight `POST /sessions` leaves the canvas in `draft` while CAO has a live session — a silent resource leak.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D5, lines 68–71) and `src/canvas-reconciler/reconciler.ts`
- **Follow-up:** none

### D-006: No Validation Proxy in v1 — supervisor obedience is prompt-only

- **Question:** Should v1 enforce the canvas topology graph at runtime, or trust the supervisor LLM to follow prompt instructions?
- **Decision:** Prompt-only enforcement. The Reconciler generates a `canvas-topology` block in the supervisor's system prompt listing allowed handoff/assign/send_message targets. No server-side proxy intercepts or blocks violations.
- **Rationale:** A Validation Proxy requires a server-side component that v1 does not have (v1 is a pure SPA). Prompt-level topology is sufficient for short, well-scoped tasks in the v1 use case.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D6, lines 73–79; R3, lines 165–166)
- **Follow-up:** `openspec/changes/validation-proxy/`

### D-007: WebGL mandatory — no Canvas2D fallback in production

- **Question:** Should xterm.js fall back to Canvas2D when WebGL is unavailable, or refuse to render?
- **Decision:** Production refuses Canvas2D. The terminal shows a "WebGL is required" error. Dev builds allow Canvas2D only with `VITE_ALLOW_CANVAS2D=true`.
- **Rationale:** The "zero visible lag" promise depends on GPU rendering. A silent fallback would degrade performance without the user knowing why.
- **Locked by:** `src/terminal/use-terminal-stream.ts` (WebglAddon initialization + fallback gate) and `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D7, lines 82–85)
- **Follow-up:** none

### D-008: Default model selection for templates

- **Question:** Should templates pre-select a recommended model, or leave the model field empty for the user to choose?
- **Decision:** No default, no recommendation. Templates inherit the user's `provider_default` setting and the user explicitly selects the model. The model dropdown shows all available models for the validated provider.
- **Rationale:** Resolved in v4.2 §8.10. Recommending a specific model would create a support burden when that model becomes deprecated or rate-limited.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (Open Questions §2, line 205) and task 7.7 in `tasks.md`
- **Follow-up:** none

### D-009: First-run wizard scope — browsable without keys

- **Question:** Should the app block entirely until a provider key is configured, or allow browsing with deploys gated?
- **Decision:** Users can browse the full UI without keys (read-only canvas, settings, templates). Deploy is the only gate — disabled until at least one provider is validated. The wizard is offered on first visit but always skippable.
- **Rationale:** Blocking the entire UI on first visit prevents exploration and discourages adoption. Gating only Deploy aligns with the BYOK cost model — no key, no inference, no deploy.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (Open Questions §3, line 207) and `src/health/FirstRunWizard.tsx`
- **Follow-up:** none

### D-010: Cron schedule editor UX

- **Question:** Should the Flows editor use only a raw cron input, or add visual presets?
- **Decision:** Quick-pick presets (every-N-minutes, hourly, daily-at-time, weekdays-at-time, weekly) plus a raw cron input for power users, with `cronstrue` for human-readable explanations.
- **Rationale:** Raw cron alone creates a high-friction experience for non-DevOps users. Presets cover 90% of use cases; the raw input covers the rest.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (Open Questions §4, line 209) and task 19.2 in `tasks.md`
- **Follow-up:** none

### D-011: Diff-based edit-after-deploy from day one

- **Question:** Should users be forced to Tear Down + Redeploy when editing a deployed canvas, or can the Reconciler apply a delta?
- **Decision:** The Reconciler computes the diff between the current Canvas Document and the recorded actual state, then applies only the delta. Five diff cases are supported: node added, node removed, profile changed, display-only change, edge change. Entry-point changes are blocked — they require Tear Down.
- **Rationale:** Full redeploy on every edit is hostile to iterative workflows. The user-approved override in v4.2 §12 mandated diff-based edit despite the added reconciler complexity.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D14, lines 137–149; R5, lines 171–172) and `src/canvas-reconciler/reconciler.ts`
- **Follow-up:** none

### D-012: Voice NLU provider selection

- **Question:** Which LLM runs the NLU structured-extraction call? User-chosen or auto-selected?
- **Decision:** Auto-selected: cheapest validated key wins. Preference order: Gemini Flash → GPT-4o-mini → Haiku. If no LLM key is validated, voice input is disabled with an inline message directing the user to Settings.
- **Rationale:** AgentVerse never pays for inference (BYOK). The cheapest model is sufficient for the structured-JSON extraction task. Users don't need to care which model runs NLU.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (D15, lines 151–155) and `src/voice/nlu.ts`
- **Follow-up:** none

### D-013: WebSocket fan-out — single connection per terminal ID

- **Question:** When multiple UI consumers display the same terminal (focused tab, grid cell, dashboard preview), should each open its own WebSocket or share one?
- **Decision:** Single WebSocket per terminal ID, shared via a fan-out helper. Multiple consumers register callbacks against the same connection.
- **Rationale:** Opening N WebSocket connections for the same terminal multiplies server load and risks duplicate data processing. The fan-out helper (`terminal-socket-fanout.ts`) solves this at the infra layer.
- **Locked by:** `src/api/terminal-socket-fanout.ts` and task 4.6 in `tasks.md`
- **Follow-up:** none

### D-014: FinOps Tier 1 — wall-clock only, mandatory ⚠️ labels

- **Question:** How accurate should v1 cost estimates be? Should token-level accuracy be attempted?
- **Decision:** Wall-clock × `PROVIDER_COST_PER_HOUR` only. Every cost surface in the UI (dashboard KPI, FinOps page, templates picker) renders the ⚠️ glyph and a "rough estimate" disclaimer. Tier 2 (token parsing) and Tier 3 (billing APIs) are explicitly post-launch.
- **Rationale:** Token-level parsing requires per-provider response parsing and is a significant implementation effort. Wall-clock estimates with honest labeling give users directional awareness without false precision.
- **Locked by:** `src/finops/cost-estimate.ts` (computation) and `openspec/changes/milestone-1-canvas-deploy-run/design.md` (lines 27, 38)
- **Follow-up:** `openspec/changes/finops-tier2-token-parsing/`

### D-015: `SCHEMA_VERSION` location — accepted D9 violation

- **Question:** Should `SCHEMA_VERSION` live in `src/shared/` (per D9's cross-capability constant rule) or in `src/canvas-document/`?
- **Decision:** Kept in `src/canvas-document/schema.ts` to avoid import cycles during the parallel build phase. The D9 violation is accepted and documented as tech debt.
- **Rationale:** Moving it to `shared/` during the parallel sprint would have required a supervisor-gated merge for a single constant, blocking canvas work. The constant is only read in two places (schema migration and reconciler version check).
- **Locked by:** `src/canvas-document/schema.ts` (SCHEMA_VERSION constant)
- **Follow-up:** `openspec/changes/tech-debt-schema-version-shared/`

### D-016: VoicePanel cross-capability imports — accepted D9 violation

- **Question:** Should VoicePanel use an event bus or shared interface to communicate with canvas-builder and canvas-reconciler, or import directly?
- **Decision:** Direct sideways imports with `eslint-disable` comments. VoicePanel imports from `@/canvas-builder/deploy-validation`, `@/canvas-reconciler/reconciler`, and `@/canvas-builder/provider-options`.
- **Rationale:** Introducing an event bus mid-sprint risks destabilizing both voice and canvas modules. The lint suppressions explicitly document the coupling for post-v1 cleanup.
- **Locked by:** `src/voice/VoicePanel.tsx` (eslint-disable comments)
- **Follow-up:** `openspec/changes/tech-debt-voice-event-bus/`

### D-017: Smoke spec voice flow — mock-based in headless

- **Question:** Should the Playwright smoke spec exercise the real STT engine, or mock the voice state for headless CI?
- **Decision:** Mock-based. The test injects voice state directly via `useVoiceStore.setState()` and uses `{ force: true }` on the animated stop button to bypass Playwright's stability checks.
- **Rationale:** Headless Chromium does not support the Web Speech API. A real STT flow would require a polyfill or a Chromium build with SpeechRecognition support — non-trivial to set up and maintain.
- **Locked by:** `tests/e2e/smoke.spec.ts` (voice command segment)
- **Follow-up:** `openspec/changes/tech-debt-smoke-voice-real-flow/`

### D-018: Activity feed retention — unlimited, browser-managed

- **Question:** Should the Dashboard activity feed impose an application-level cap (e.g. last 500 events) or retain everything?
- **Decision:** Unlimited retention. The browser's memory limits are the natural ceiling. A manual "Clear" affordance is provided.
- **Rationale:** Resolved in v4.2 §12. Application-level caps add complexity and hide recent events for active sessions. Browser memory is sufficient for v1's local-first use case.
- **Locked by:** `openspec/changes/milestone-1-canvas-deploy-run/design.md` (Open Questions §5, line 211) and task 15.4 in `tasks.md`
- **Follow-up:** none

### D-019: Performance target — 12+ terminals at ≥55 FPS (≥45 CI)

- **Question:** What is the frame-rate threshold for concurrent terminal streaming, and how is it measured?
- **Decision:** Production target: ≥55 FPS with 12 concurrent WebSocket streams. CI headless threshold: ≥45 FPS (headless Chromium uses software rasterization). Measured via `requestAnimationFrame` counter over a 10-second window.
- **Rationale:** Headless Chromium lacks GPU compositing and caps at ~50 FPS even idle. The CI threshold avoids false failures while the production target documents the real-browser expectation.
- **Locked by:** `tests/e2e/perf-12-terminals.spec.ts` (constants and test assertions)
- **Follow-up:** none

### D-020: `appFetch` — v1 pass-through, documented for Firebase JWT

- **Question:** Should the app use raw `fetch` or a wrapper? If a wrapper, what does v1 include?
- **Decision:** All AgentVerse-managed HTTP goes through `appFetch` in `src/shell/app-fetch.ts`. In v1, it is a pass-through. The wrapper is documented as the future attachment point for Firebase JWT headers when cloud auth arrives.
- **Rationale:** A lint rule bans direct `fetch` for CAO routes outside `cao-client.ts`. The wrapper ensures a single point of control for auth headers, error normalization, and telemetry — even if v1 doesn't use those features yet.
- **Locked by:** `src/shell/app-fetch.ts` and `src/api/cao-client.ts`
- **Follow-up:** `openspec/changes/cloud-runtime-deployment/`

## Index by capability

- **SUP:** D-001, D-003, D-015, D-019, D-020
- **IF:** D-004, D-013, D-020
- **CV:** D-002, D-005, D-008, D-011, D-015
- **TM:** D-007, D-013, D-019
- **DB:** D-009, D-014, D-018
- **ST:** D-010
- **VX:** D-012, D-016, D-017
