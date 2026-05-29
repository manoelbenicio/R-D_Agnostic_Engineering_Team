# Changelog

All notable changes to AgentVerse are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] — 2026-05-29

First general-availability release. Implements the AgentVerse Master Specification
v4.2 (`openspec/changes/milestone-1-canvas-deploy-run/`) — the full multi-agent
orchestration SPA on top of CAO, plus six post-v4.2 tech-debt remediations.

### Highlights

- Visual canvas builder with drag-drop agent blocks, three edge types
  (`handoff` / `assign` / `send_message`), 10 starter templates, voice authoring,
  and diff-based edit-after-deploy.
- Real-time terminal streaming over WebSocket binary frames at 12+ concurrent
  terminals, with grid view, focused tab view, and chat-bubble view.
- Bring-your-own-key (BYOK) provider integration for OpenAI, Anthropic, Google,
  AWS (Q + Kiro), Azure, Moonshot, GitHub Copilot, and OpenCode CLI — each with
  live validation against real provider endpoints.
- SENTINEL design system (military / Bloomberg / NASA Mission Control aesthetic)
  with JetBrains Mono default, user-configurable typography, and ambient effects
  guarded behind `prefers-reduced-motion`.
- FinOps Tier 1 cost estimation with mandatory ⚠️ "rough estimate" labeling on
  every cost surface.

### Added

#### Application shell
- React 18 + Vite + TypeScript SPA bootstrapped on Node ≥ 20.10 / npm ≥ 10.2.
- Top-level router with the v1 route table (`/`, `/dashboard`, `/canvas/:id`,
  `/canvas/:id/terminal/:terminalId`, `/agent-studio`, `/flows`, `/finops`,
  `/memory`, `/settings/{providers,appearance,general}`, `/health`, 404).
- Global `<NavBar>` with CAO health pill, `<AppLayout>`, `<ErrorBoundary>` with
  SENTINEL fallback, bottom-right toast region.
- `appFetch` wrapper in `src/shell/app-fetch.ts` (v1 pass-through, ready for
  Firebase JWT attachment in M2).
- IndexedDB infrastructure (`src/shared/storage/idb.ts`) with object stores
  `canvases`, `canvas_versions`, `provider_keys`, `settings`, `app_state` and a
  migration helper.

#### Design system — SENTINEL
- Full token set (`--void`, `--panel`, `--card`, `--cyan`, `--amber`, `--threat`,
  `--ops`, `--text-*`) per master spec §3.
- Base component library: `Card`, `Button` (primary / secondary / ghost), `Badge`,
  `NavBar`, `StatusBadge` (5 statuses → SENTINEL colors), `FormField`, `Modal`,
  `Toast`, `Prose` (markdown viewer).
- Opt-in ambient effects (`scanlines`, `scan-sweep`, `kpi-glow`) gated behind
  `prefers-reduced-motion: reduce`.
- Font-override mechanism: settings store overrides `--font-display`,
  `--font-body`, `--font-mono` at `:root`.
- Locked-files CI policy — `src/design-system/` edits require a supervisor
  approval label.

#### CAO integration
- `CaoClient` covering the full v1 REST surface (32 endpoints across Health,
  Profiles, Sessions, Terminals, Inbox, Flows, Settings, Skills) plus the PTY
  WebSocket helper.
- Typed error classes `CaoApiError` and `CaoNetworkError`.
- WebSocket fan-out: focused tab + mini-terminal in grid + Dashboard preview
  share one socket per terminal id.
- `useHealthStore()` (Zustand) — 10 s polling of `GET /health`, paused on
  `document.hidden`, resumed on visibility return.
- TanStack Query keys with capability-owned namespacing.
- MSW handlers covering every CAO endpoint for integration tests.
- `CAO_LIVE=1`-gated contract test suite + nightly CI workflow.
- Lint rule `agentverse/no-direct-cao-fetch` blocking direct `fetch()` to CAO
  routes outside `src/api/cao-client.ts`.

#### API key management (BYOK)
- Provider registry with key field shapes for all 8 v1 providers.
- `KeyStore` backed by IndexedDB (plaintext per the documented v1 threat model
  in `docs/key-storage-v1.md`; encrypted Firestore field is post-launch).
- Key masking helper (`sk-…XXXX` for any UI display).
- Provider-specific validators with live model-list fetches:
  OpenAI, Anthropic, Google, AWS STS GetCallerIdentity, Azure (configurable
  endpoint), Moonshot, GitHub Copilot CLI, OpenCode CLI.
- Settings → Providers / General / Appearance pages.
- `useValidatedProviders()` selector consumed by Canvas Builder, Agent Studio,
  Voice.

#### Canvas authoring
- `CanvasDocument` schema (Zod) with self-loop, dangling-edge, and
  zero/multiple-entry-point validators.
- `CanvasStore` with `list()`, `get()`, `save()`, `delete()`, `listVersions()`
  + per-save versioning.
- React Flow canvas with custom `agent` node renderer, agent palette
  (Supervisor / Developer / Reviewer / Custom), drag-from-palette placement,
  role-template registry, and edge mode menu (handoff / assign / send_message)
  with ≤100 ms style transitions.
- Block Configuration Panel with Monaco editor for `system_prompt`.
- Save (Cmd/Ctrl+S), undo/redo (≥20 actions), canvas list sorted by
  `updated_at` desc.
- Deploy button with disabled-state reasons surfaced in tooltip.
- Templates picker invokable from canvas list, empty canvas, and toolbar.
- Voice trigger button + `Ctrl+Shift+V` hotkey.
- Touch-detect read-only mode with banner.

#### Canvas templates
- 10 built-in `CanvasDocument` definitions per master spec §4.8 (Code Review,
  Bug Triage, Documentation Sprint, Full Stack Dev, Data Pipeline,
  Security Audit, DevOps Pipeline, Research Team, Enterprise Squad,
  Blank Canvas).
- `instantiateTemplate(templateId)` with fresh UUIDs, "(copy)" suffix,
  `deploy_state.status: 'draft'`.
- Per-template metadata (`agent_count`, `primary_edge_type`,
  `est_cost_per_hour_usd`).
- Shared ⚠️-glyph rendering helper used by templates picker + Dashboard +
  FinOps.

#### Canvas reconciler
- Profile-markdown generator (YAML frontmatter + body).
- Supervisor-prompt augmentation appending the canvas topology block per
  master spec §4.4 step 5.
- Reconciler driver: profile installs → session creation → terminal additions,
  with atomic `deploy_state` persistence before/after each CAO call (design D5).
- Deploy state machine: `draft` ↔ `deploying` ↔ `deployed` / `degraded`.
- Deploy progress panel (5-row reactive list).
- Retry Failed (only nodes absent from `terminal_map`).
- Tear Down (`DELETE /sessions/{name}` + state reset).
- Resume affordance for canvases found in `deploying` on reload.
- **Diff-based edit-after-deploy** (master spec §12 override / D14):
  per-node profile snapshots, 5-case diff (add / remove / change profile /
  display-only / edge change), entry-point change blocked behind Tear Down,
  edge-change advisory banner.

#### Terminal streaming
- `<TerminalView>` with master-spec §6.3 zero-lag config
  (`smoothScrollDuration: 0`, `scrollback: 10000`, fonts from CSS tokens).
- Addons in order: WebGL, Fit, WebLinks, Search, Unicode11.
- Production fails loudly without WebGL; dev allows Canvas2D fallback when
  `VITE_ALLOW_CANVAS2D=true`.
- Binary-frame handler writes `Uint8Array` directly to `terminal.write` with no
  string conversion (lint-enforced).
- Input handler emits JSON `{type:'input',data:'…'}`; resize via
  `ResizeObserver` debounced 100 ms.
- Connection-state pill: `connecting` / `connected` / `reconnecting` /
  `terminated`.
- Reconnect logic: 4003/4004 permanent; otherwise exponential backoff
  500 ms → 30 s with ±20 % jitter.
- SENTINEL theme applied to xterm.

#### Terminal grid
- `<TabBar>` with `StatusBadge` per terminal + close affordance + trailing `+`.
- TanStack Query polling at 3 s.
- Grid View (responsive 2×3 or 3×4) with mini-terminal cells (40×15, read-only).
- Cell-click expansion to focused tab view.
- Full-Screen mode toggle (Escape exits).
- Per-terminal controls: working dir display, send-message input, inbox viewer,
  kill button (with confirmation modal).

#### Chat view
- `<ChatView>` with output parser (strips ANSI/VT100, groups by agent prompt
  and tool-call markers, attributes by terminal id).
- SENTINEL Card-styled bubbles with display_name, provider badge, timestamp,
  partial-message buffering.
- Inline send-message composer (Enter sends, Shift+Enter newlines).
- Terminal Grid ↔ Chat View toggle (per-canvas preference).
- Viewport-based default: ≤768 px = Chat View.
- Touch affordances: swipe-left actions, `100dvh` sizing, composer above
  on-screen keyboard.

#### Voice — speech-to-canvas
- `VoiceCapture` wrapping Web Speech API (continuous, interim results, pt-BR
  default, auto-restart on `no-speech`).
- `WhisperTranscriber` fallback (`MediaRecorder` 16 kHz mono webm/opus, 3 s
  chunks, OpenAI Whisper API with the user's BYOK key).
- Engine selection: Web Speech API default → Whisper fallback → user-forced
  via Settings.
- NLU intent extraction with the cheapest validated provider (Gemini Flash
  → GPT-4o-mini → Haiku per design D15); 3 s latency budget.
- Bilingual NLU prompt template (pt-BR + en-US) with provider/edge-type
  mappings.
- `voiceToCanvas(intent)` per master spec §5.7 (auto-layout, default supervisor
  as entry-point, role-template defaults).
- 5-state UI: `idle` / `listening` / `processing` / `confirming` / `error`.
- Confirming state shows parsed intent summary, confidence, and three actions
  (Cancel / Edit Before Deploy / Generate).
- `Ctrl+Shift+V` / `Cmd+Shift+V` global hotkey.
- Disabled state when no LLM key validated, with link to `/settings/providers`.
- Privacy: transcripts are not persisted; mic permission released between
  activations.

#### Voice — runtime commands
- `matchRuntimeCommand` regex+keyword matcher (kill, pause, focus, status,
  deploy, stop_all, cost, add_node, connect) with bilingual coverage and
  ≤100 ms match latency.
- Wired actions: kill (`DELETE /terminals/{id}`), stop_all
  (`DELETE /sessions/{name}`), pause (input sentinel), focus (client-side
  navigation), status (announces via `GET /sessions/{name}/terminals`), deploy,
  cost (navigate), add_node / connect (delegate to Canvas Builder).
- Confirmation modal for destructive commands (Cancel auto-focused).
- Unrecognized transcripts fall through to NLU rather than being silently
  dropped.

#### Dashboard
- KPI Row: Fleet Status, Cost / MTD, Budget Util, Threats.
- Cost-by-Provider bar chart (recharts).
- Fleet Status donut (active / error / offline).
- Activity Feed: inbox messages + session lifecycle events, unlimited
  retention per v4.2 §12, manual Clear affordance.
- Terminal Preview Card (read-only mini-terminal, click-to-navigate).

#### FinOps Tier 1
- `PROVIDER_COST_PER_HOUR` constant matching master spec §8.7.
- `computeCostEstimate(window, terminals)` — wall-clock × per-hour rate.
- `useCostEstimate()` hook with TanStack Query selectors.
- `<CostLabel>` rendering ⚠️ glyph + disclaimer tooltip on every cost surface
  (mandatory).
- FinOps page: MTD cost, budget utilization gauge, cost-by-provider table,
  top-10 cost-by-canvas table, budget configuration affordance.

#### Health & onboarding
- Health page with three sections: Server Health, Provider Validations,
  Browser Capabilities.
- Browser-capability checks: WebGL2, IndexedDB, microphone permission via
  `navigator.permissions.query({ name: 'microphone' })`.
- Test Microphone affordance using `getUserMedia`.
- Fix affordances per failed-check type.
- First-Run Wizard: Verify CAO → Configure provider (mini Settings) → Pick
  starting point (Templates picker / Start Blank).
- Wizard skip remembered via `app_state` IndexedDB store.

#### Agent Studio
- Profile list (`GET /agents/profiles`) with search + filter by name / role
  / provider.
- Provider availability panel (`GET /agents/providers`).
- Profile detail viewer (parsed markdown body in SENTINEL prose, frontmatter
  as key-value list).
- Profile editor with Monaco markdown body, Save → `POST
  /agents/profiles/install`.
- Install From Source: built-in store, local file picker (`.md`), URL fetch
  with preview-before-install confirm.

#### Flows
- Flow list with `cronstrue` schedules; 15 s refresh.
- Quick-pick schedule UI (every-N-minutes, hourly, daily-at-time,
  weekdays-at-time, weekly) + raw cron with `cronstrue` validation.
- Create / edit form with all `Flow` fields (name, schedule, agent_profile,
  provider, prompt_template Monaco, enabled).
- Run Now (`POST /flows/{name}/run`).
- Enable / Disable optimistic toggle with revert on failure.
- Conditional badge for gated flows.

#### Memory viewer
- Memory list via per-terminal context API + agent-dirs setting.
- Scope filter (global / project / session / agent), type filter (project /
  user / feedback / reference), tag filter.
- Detail viewer with parsed markdown body, scope/type metadata, tags,
  retention info, location path.
- Full-text search across content + tags.
- Manual memory creation form.
- Retention notice for session-scoped memories.

#### Cross-cutting quality gates
- Playwright smoke spec covering the v1 critical path.
- Bundle size budget: `dist/` ≤ 1.5 MB gzipped (`scripts/check-bundle-size.mjs`).
- Performance test: 12+ concurrent terminals streaming without dropped frames.
- Accessibility audit (axe-core) on all v1 routes; criticals + serious resolved.
- Nightly `CAO_LIVE=1` contract suite against a live CAO; alerts on shape drift.

#### Tech-debt remediations (post-v4.2)
- **`tech-debt-keystore-validator-coverage`** — added 8 `KEYSTORE_LIVE=1`-gated
  contract tests (one per provider) under
  `src/api/key-store/__tests__/contract/`, the `npm run test:keystore-contract`
  script, the `keystore-contract-nightly` GitHub workflow, and
  `docs/keystore-contract-tests.md`. MSW unit tests stay as the deterministic
  CI gate.
- **`tech-debt-schema-version-shared`** — moved the canonical
  `SCHEMA_VERSION` + `isCompatible()` to `src/shared/schema-version.ts` per
  design D9; left a backwards-compatible re-export shim at
  `src/canvas-document/schema-version.ts`.
- **`tech-debt-react-refresh-cleanups`** — collapsed the
  `src/canvas-builder/CanvasBuilder.tsx` and `CanvasList.tsx` barrels into
  `src/canvas-builder/index.ts`; converted `CostWarning` to a function
  declaration so the `react-refresh` plugin recognizes its boundary.
- **`tech-debt-voice-coverage-gap`** — extracted
  `src/voice/command-executor.ts` (517 lines, pure async function with all
  side-effects routed through injected deps); added unit tests for all 9
  runtime command actions; raised `src/voice/` statement coverage from
  ~40.78 % to **87.80 %**. `VoicePanel.tsx` shrunk from 803 to 586 lines.
- **`tech-debt-voice-event-bus`** — introduced `CanvasCommandBus` in
  `src/shared/canvas-command-bus.ts` and the concrete
  `canvasCommandBus` adapter in `src/shell/canvas-command-adapter.ts`. Removed
  all three `eslint-disable agentverse/no-sideways-capability-imports`
  directives from voice modules; the adapter is now the only module allowed to
  import from canvas-builder + canvas-reconciler simultaneously.
- **`tech-debt-smoke-voice-real-flow`** — replaced the Playwright
  `{ force: true }` workaround with an animation-disabling stylesheet helper
  (`tests/e2e/helpers/disable-animations.ts`) and replaced the direct
  `useVoiceStore.setState()` workaround with a real `SpeechRecognition`
  polyfill (`tests/e2e/helpers/speech-recognition-mock.ts`). The smoke test
  now drives the full STT → NLU → command-executor pipeline in headless
  Chromium.

### Documentation
- `ARCHITECTURE.md` summarizing decisions D1–D15 and risks R1–R9.
- `docs/cao-cors.md` — verbatim CAO env vars to allow this SPA.
- `docs/key-storage-v1.md` — v1 BYOK threat model.
- `docs/canvas-topology-prompt.md` — supervisor-prompt augmentation pattern.
- `docs/v1-decisions.md` — open questions resolved during implementation.
- `docs/keystore-contract-tests.md` — env var matrix + acquisition tips for the
  keystore contract suite.
- `docs/patterns/testing.md` — `installSpeechRecognitionMock` and
  animation-disable pattern conventions for E2E specs.

### Out of scope (post-launch follow-ups)
The following are tracked as separate change proposals under
`openspec/changes/`:
- `validation-proxy` — server-side edge enforcement middleware.
- `cloud-runtime-deployment` — Cloud Run / GKE / user-hosted CAO economics.
- `finops-tier2-token-parsing` — per-token cost attribution.
- Autonomous Copilot (persistent autonomous meta-agent).

### Quality

- `npm run lint` — 0 errors.
- `npm run typecheck` — clean.
- `npm test` — 50 test files, 367 passed, 8 skipped (the skipped 8 are the new
  `KEYSTORE_LIVE=1`-gated contract tests).
- `npm run test:smoke` — green against MSW-mocked CAO.
- `src/voice/` statement coverage: **87.80 %** (target ≥ 70 %).
- Bundle size: within the 1.5 MB gzipped budget (`scripts/check-bundle-size.mjs`).

[Unreleased]: https://github.com/your-org/agentverse/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/your-org/agentverse/releases/tag/v1.0.0
