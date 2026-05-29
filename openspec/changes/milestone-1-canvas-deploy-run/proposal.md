## Why

AgentVerse is a 18-page master specification (v4.1) with no executable contracts. The team — a multi-agent squad of 2+ — needs a single source of truth pinned into capability contracts and a parallel-executable task list, so work begins immediately rather than re-litigating decisions.

Per master spec §12 (user-approved), AgentVerse v1 ships as **a single delivery — all 15 feature areas at once, no milestones, no phasing**. All agents start in parallel from day one (§14, user-approved override of any sequential-bootstrap pattern). This change captures every capability needed for that delivery as a coordinated set of contracts.

The single-delivery / parallel-from-day-one decisions trade execution risk for speed. The Risks section of the design document records that trade explicitly so the team enters with eyes open.

## What Changes

This change introduces the entire AgentVerse v1 frontend on top of CAO. Concretely:

- **Application shell** — Vite + TypeScript + React 18 SPA (confirmed in master spec v4.2 §12; rework risk acknowledged), top-level routing, navigation, layout, auth-aware fetch boundary, global error boundary, toasts.
- **SENTINEL design system** — full token set per master spec §3, base component library, JetBrains Mono as the default typeface family throughout (per §14.7 override of §3.3) with user-configurable fonts via Settings → Appearance.
- **CAO HTTP & WebSocket integration** — typed client over the full CAO REST surface needed for v1 (sessions, terminals, profiles, providers, flows, memory, skills, settings) plus the PTY WebSocket helper.
- **API key management (BYOK) for all 8 providers** — OpenAI, Anthropic, Google, AWS (Q + Kiro), Azure, Moonshot, plus the remaining CAO-mapped providers (`copilot_cli`, `opencode_cli`). Live validation per provider, plaintext-in-IndexedDB storage for local dev (encrypted Firestore field is post-launch / cloud).
- **Canvas Document schema and persistence** — node/edge/config/deploy_state schema, IndexedDB persistence with per-save versioning.
- **Canvas Builder** — drag-drop editor with agent palette, edge mode selection (handoff/assign/send_message), block configuration panel, save/load, undo/redo, Deploy entry point, Templates picker, voice trigger.
- **Canvas Templates** — 10 built-in starter canvases per master spec §4.8 (Code Review, Bug Triage, Documentation Sprint, Full Stack Dev, Data Pipeline, Security Audit, DevOps Pipeline, Research Team, Enterprise Squad, Blank Canvas) with displayed cost-per-hour estimates.
- **Canvas Reconciler with diff-based edit-after-deploy** — translates Canvas Document into CAO calls; tracks `draft → deploying → deployed → degraded`; supports Retry Failed; supports edit-after-deploy by diffing desired vs. actual and applying only the delta (per §12 user override).
- **Terminal streaming** — xterm.js + WebGL + binary WebSocket frames per master spec §6.3 zero-lag profile.
- **Terminal Grid** — tab bar + 2×3/3×4 grid view + mini-terminals + full-screen mode per master spec §6.6.
- **Chat View** — parsed agent output rendered as chat bubbles for mobile-first and simple flows.
- **Speech-to-Canvas** — Web Speech API (Tier 1) + Whisper API (Tier 2) fallback, NLU intent extraction via the user's BYOK LLM key, canvas generation from voice, intent-preview confirm step.
- **Voice runtime commands** — pt-BR + en-US runtime command vocabulary (kill, pause, focus, status, deploy, stop_all, cost) with regex+keyword matcher per master spec §5.8.
- **Dashboard** — Central de Comando with Fleet Status / Cost MTD / Budget Util / Errors KPIs, cost-by-provider chart, fleet status donut, activity feed, terminal preview.
- **FinOps Tier 1** — wall-clock × `PROVIDER_COST_PER_HOUR` cost estimation with mandatory ⚠️ "rough estimate" labeling; cost-by-provider breakdown; budget utilization. Tier 2/3 are explicitly post-launch.
- **Agent Studio** — profile list + provider availability + profile detail viewer + profile editor (markdown) + install from store/file/URL.
- **Flows** — list, create, delete, enable/disable, run; cron schedule editor (visual + raw); show next/last run, status; conditional gating script support.
- **Memory viewer** — list (global + project scoped) + detail view + search/filter + manual creation + retention info.
- **Health page** — CAO status + per-provider validation + browser capability checks (WebGL, IndexedDB, microphone permission) — first-run wizard included.
- **First-run UX** — health wizard + template picker on first visit.
- **Polish** — keyboard shortcuts (Ctrl+Shift+V voice; Ctrl+F terminal search; Cmd/Ctrl+S save), copy/paste handling, responsive layout for desktop/tablet/mobile, performance verified at 12+ concurrent terminals.

**Out of scope (per master spec §13, post-launch):**
- Validation Proxy (server-side edge enforcement middleware).
- Cloud Runtime architectural deep-dive (Cloud Run vs. GKE vs. user-hosted CAO economics).
- FinOps Tier 2 (token parsing) and Tier 3 (provider billing APIs).
- Autonomous Copilot (persistent autonomous meta-agent).

## Capabilities

### New Capabilities

**Foundations**
- `agentverse-shell`: SPA bootstrap, routing, layout, NavBar, error boundary, toasts, auth-aware fetch boundary.
- `design-system-sentinel`: SENTINEL CSS token system + base components + JetBrains Mono default with user-configurable fonts.
- `cao-integration`: Typed CAO REST client over the full v1 surface plus the PTY WebSocket helper and CORS-config docs.
- `api-key-management`: BYOK provider settings for all 8 providers with live validation and gated provider selection.
- `canvas-document`: Canvas Document JSON schema and IndexedDB persistence with per-save versioning.

**Canvas authoring**
- `canvas-builder`: Drag-drop editor with palette, edges, block config, save/load, undo/redo, Deploy.
- `canvas-templates`: 10 built-in starter canvases with cost-estimate badges and a templates picker UI.
- `canvas-reconciler`: Canvas → CAO translation, deploy state machine, Retry Failed, **diff-based edit-after-deploy** (per §12 override), Tear Down.

**Voice authoring & runtime control**
- `speech-to-canvas`: Capture (getUserMedia/MediaRecorder) → STT (Web Speech API + Whisper fallback) → NLU (user's BYOK LLM) → canvas generation with intent-preview confirm.
- `voice-runtime-commands`: Voice command matcher for runtime control (kill, pause, focus, status, deploy, stop_all, cost) in pt-BR and en-US.

**Terminal surfaces**
- `terminal-streaming`: Single-terminal PTY rendering via xterm.js + WebGL + binary WebSocket; the primitive consumed by every higher-level terminal surface.
- `terminal-grid`: Tab bar, 2×3/3×4 grid of mini-terminals, full-screen mode, per-terminal kill/inbox/input controls.
- `chat-view`: Parsed agent output rendered as chat bubbles for mobile and simple flows.

**Operations surfaces**
- `dashboard`: Central de Comando KPIs, cost chart, fleet donut, activity feed, terminal preview.
- `finops-tier1`: Wall-clock cost estimation with mandatory ⚠️ labels, budget utilization, per-provider breakdown.
- `agent-studio`: Profile list/detail/editor, provider availability, install from store/file/URL.
- `flows`: Cron-scheduled flows CRUD with visual and raw editor.
- `memory-viewer`: Memory list/detail/search across global and project scopes.
- `health-and-onboarding`: Health page (CAO + providers + browser capabilities) plus first-run wizard and template picker.

### Modified Capabilities

_None._ All capabilities introduced in this change are net-new — `openspec/specs/` is currently empty.

## Impact

**Code**
- Full `src/` tree per master spec §14.2 with directory-level ownership:
  - `src/design-system/` — Supervisor (locked after design-system stabilization)
  - `src/canvas-builder/`, `src/canvas-document/`, `src/canvas-reconciler/`, `src/canvas-templates/` — Canvas Dev
  - `src/terminal/`, `src/terminal-grid/`, `src/chat-view/` — Terminal Dev
  - `src/dashboard/`, `src/finops/`, `src/health/` — Dashboard Dev
  - `src/agent-studio/`, `src/flows/`, `src/memory-viewer/` — Studio Dev
  - `src/voice/` — Voice Dev (covers `speech-to-canvas` and `voice-runtime-commands`)
  - `src/api/`, `src/settings/` — Infra Dev
  - `src/shell/` — Supervisor (routing, layout, app shell)
  - `src/shared/` — Supervisor-gated cross-capability types and utilities
- New top-level config: `vite.config.ts`, `tsconfig.json`, `package.json`, `.env.example`.

**Dependencies (new, all pinned)**
- Runtime (working assumption pending Principal Architect): `react`, `react-dom`, `react-router`, `zustand`, `@tanstack/react-query`, `@xterm/xterm`, `@xterm/addon-fit`, `@xterm/addon-webgl`, `@xterm/addon-web-links`, `@xterm/addon-search`, `@xterm/addon-unicode11`, `@xyflow/react`, `idb`, `zod`, `@monaco-editor/react`, `cronstrue` (cron parser/explainer for the Flows editor), `recharts` (KPI charts).
- Dev: `vite`, `typescript`, `vitest`, `@testing-library/react`, `msw` (mock service worker for integration tests), `@playwright/test`, `eslint`, `prettier`, `axe-core`.
- The Principal Architect may revise the framework choice; if so, every decision recorded in design.md against React/Vite is contingent.

**External systems**
- Requires a reachable CAO server. Local dev: `localhost:9889`. Cloud: deferred per §13.
- Requires the user to have at least one valid provider API key. Anthropic is the recommended default for templates; all 8 providers are validated and selectable.

**Operational**
- v1 is structured to run from a static SPA bundle plus an external CAO. Cloud hosting decisions are explicitly post-launch.

**Risk**
- **Single-delivery, all-at-once execution risk**: 19 capabilities in parallel by 2+ agents on a single branch is a high-coordination workload. The Multi-Agent Build Strategy (master spec §14) assigns directory ownership and supervisor review on every PR; design.md captures the residual risk and the mitigations.
- **No sequential bootstrap**: Patterns (state, fetch, components, testing) are established alongside feature work rather than before. Expect early-cycle PR thrash as patterns settle. The supervisor agent owns this thrash through review.
- **Stack rework risk acknowledged**: React 18 + Vite + TypeScript is confirmed in master spec v4.2 §12. The team accepts that a future architectural revisit could force ~30% UI rework; the framework-agnostic layers (cao-integration client, schema, persistence, reconciler) survive.
- **CAO API drift**: AgentVerse depends on the CAO REST contract. The HTTP client encapsulates the surface area; contract tests (gated behind `CAO_LIVE=1`) detect drift.
- **Diff-based edit-after-deploy from day one** introduces complexity in the Reconciler (state diffing under partial-failure). Spec captures the safe-update invariants.
