# AgentVerse v1 — Implementation Tasks

> Owner tags follow master spec §14.1:
> **SUP** = Supervisor · **CV** = Canvas Dev · **TM** = Terminal Dev ·
> **DB** = Dashboard Dev · **ST** = Studio Dev · **VX** = Voice Dev · **IF** = Infra Dev
> Per master spec §14 user-approved override, all owners start in parallel on a single shared branch.
> Tasks within a section are ordered by dependency where it matters; otherwise concurrent within an owner.

## 1. Repo Foundation (SUP)

- [x] 1.1 [SUP] Initialize repo with the confirmed v4.2 stack: Vite + React 18 + TypeScript; strict `tsconfig.json`; dev server on port 5173 to match documented CAO CORS allow-list
- [x] 1.2 [SUP] Add pinned dependencies per proposal Impact section (runtime + dev), commit `package-lock.json`, document install with `npm ci`
- [x] 1.3 [SUP] Configure ESLint + Prettier with shared config; add lint rule that forbids cross-capability sideways imports (only `src/shared/` is allowed across capability boundaries)
- [x] 1.4 [SUP] Configure Vitest + `@testing-library/react` for unit/component tests, MSW for integration mocks, Playwright for E2E smoke; add `npm run test`, `npm run test:smoke`, `npm run test:contract` (gated on `CAO_LIVE=1`), and `npm run typecheck`
- [x] 1.5 [SUP] Set up CI: lint → typecheck → unit tests → MSW integration tests on every PR; nightly job runs the `CAO_LIVE=1` contract suite when CAO is reachable; broken main triggers immediate revert
- [x] 1.6 [SUP] Create `src/` directory structure with directory-level CODEOWNERS per master spec §14.2: `design-system/` (SUP), `canvas-builder/` `canvas-document/` `canvas-reconciler/` `canvas-templates/` (CV), `terminal/` `terminal-grid/` `chat-view/` (TM), `dashboard/` `finops/` `health/` (DB), `agent-studio/` `flows/` `memory-viewer/` (ST), `voice/` (VX), `api/` `settings/` (IF), `shell/` (SUP), `shared/` (SUP-gated)
- [x] 1.7 [SUP] Create `docs/patterns/` skeleton — first PRs from each owner write their established patterns here (state, fetch, components, testing) so subsequent PRs can copy
- [x] 1.8 [SUP] Author `ARCHITECTURE.md` summarizing design.md decisions D1–D15 and risks R1–R9

## 2. Application Shell (SUP) — `agentverse-shell`

- [x] 2.1 [SUP] Implement React Router with the v1 route table from `agentverse-shell/spec.md`: `/`, `/dashboard`, `/canvas/:id`, `/canvas/:id/terminal/:terminalId`, `/agent-studio`, `/flows`, `/finops`, `/memory`, `/settings/providers`, `/settings/appearance`, `/settings/general`, `/health`, `*`
- [x] 2.2 [SUP] Implement top-level `<NavBar>` with wordmark, primary navigation, and a CAO health pill driven by the `useHealthStore()` (clicking pill navigates to `/health`)
- [x] 2.3 [SUP] Implement `<AppLayout>` with NavBar, main content region, and bottom-right toast region; responsive at 1024×768 minimum
- [x] 2.4 [SUP] Implement global `<ErrorBoundary>` wrapping the routed view with a SENTINEL-styled fallback offering Reload and Report
- [x] 2.5 [SUP] Implement `appFetch` wrapper in `src/shell/app-fetch.ts` (v1 pass-through; documented for future Firebase JWT attachment); add lint rule banning direct `fetch` for AgentVerse-managed routes
- [x] 2.6 [SUP] Implement Zustand `toastsStore` and `useToast()` hook; smoke test info/error toast lifecycle
- [x] 2.7 [SUP] Establish IndexedDB infrastructure in `src/shared/storage/idb.ts` opening the AgentVerse DB with object stores `canvases`, `canvas_versions`, `provider_keys`, `settings`, `app_state`. Migration helper supports `schema_version` bumping

## 3. Design System (SUP) — `design-system-sentinel`

- [x] 3.1 [SUP] Implement SENTINEL CSS tokens in `src/design-system/tokens.css` per master spec §3.2 colors and §3.4 component patterns; defaults for typography use JetBrains Mono per §14.7 user override
- [x] 3.2 [SUP] Implement font-override mechanism: settings store can override `--font-display`, `--font-body`, `--font-mono` at `:root` based on user selections from Settings → Appearance
- [x] 3.3 [SUP] Implement core components in `src/design-system/components/`: `Card`, `Button` (primary/secondary/ghost), `Badge`, `NavBar`, `StatusBadge` (mapping all 5 statuses to SENTINEL colors), `FormField`, `Modal`, `Toast`
- [x] 3.4 [SUP] Implement SENTINEL ambient effects as opt-in classes: `scanlines`, `scan-sweep`, `kpi-glow`, all guarded by `prefers-reduced-motion: reduce`
- [x] 3.5 [SUP] Add `prose` component for parsed markdown bodies (used by Agent Studio detail viewer, Memory viewer, voice intent preview)
- [x] 3.6 [SUP] Add accessibility audit script (axe-core) to CI; fix critical and serious findings before any release tag <!-- TODO: Wires scripts/run-axe.mjs into CI (SUP-quality owns, task 21.5) -->
- [x] 3.7 [SUP] Add CI check that fails any PR touching `src/design-system/` without a supervisor approval label (locked-files policy from `design-system-sentinel/spec.md`)

## 4. CAO Integration (IF) — `cao-integration`

- [x] 4.1 [IF] Implement `CaoClient` class in `src/api/cao-client.ts` exposing the full v1 method surface from `cao-integration/spec.md` (32 endpoints across Health, Profiles, Sessions, Terminals, Inbox, Flows, Settings, Skills + the WebSocket helper)
- [x] 4.2 [IF] Define typed error classes `CaoApiError` and `CaoNetworkError`; every method translates failures into one of these
- [x] 4.3 [IF] Implement `VITE_CAO_BASE_URL` reading with default `http://127.0.0.1:9889`; bake the base URL at SPA bootstrap, not per-request
- [x] 4.4 [IF] Implement `connectTerminalSocket(id, handlers)` helper; verify URL construction (http→ws, https→wss) and `binaryType="arraybuffer"` set before opening
- [x] 4.5 [IF] Implement typed close-code surfacing: 4003 → `IpNotAllowed`, 4004 → `TerminalNotFound`
- [x] 4.6 [IF] Implement WebSocket fan-out helper: multiple xterm consumers (focused tab + mini-terminal in grid + Dashboard preview) share one connection per terminal id
- [x] 4.7 [IF] Implement TanStack Query keys for all `GET` endpoints with capability-owned namespacing (`['cao', 'sessions']`, `['cao', 'terminal', id]`, etc.)
- [x] 4.8 [IF] Implement `useHealthStore()` (Zustand) that wraps `GET /health` polling at 10 s, pauses on `document.hidden`, resumes on visibility return
- [x] 4.9 [IF] Wire NavBar health pill to the health store
- [x] 4.10 [IF] Author `docs/cao-cors.md` with verbatim `CAO_CORS_ORIGINS`, `CAO_ALLOWED_HOSTS`, `CAO_WS_ALLOWED_CLIENTS` values for local dev
- [x] 4.11 [IF] Add MSW handlers under `src/api/__tests__/msw/` covering every CAO endpoint with realistic responses; capability owners consume these in their integration tests
- [x] 4.12 [IF] Add `CAO_LIVE=1`-gated contract tests under `src/api/__tests__/contract/` asserting response shape on every endpoint
- [x] 4.13 [IF] Add lint rule: any `fetch(` to a CAO endpoint outside `src/api/cao-client.ts` fails the build

## 5. API Key Management (IF) — `api-key-management`

- [x] 5.1 [IF] Define `ProviderDefinition` registry shape; add entries for all 8 v1 providers (OpenAI, Anthropic, Google, AWS Q+Kiro combined, Azure, Moonshot, Copilot, OpenCode) with their key-field names, validation endpoints, and model-list parsers
- [x] 5.2 [IF] Implement `KeyStore` interface in `src/api/key-store/index.ts` backed by IndexedDB `provider_keys`; document v1 plaintext threat model in `docs/key-storage-v1.md`
- [x] 5.3 [IF] Implement key masking helper: returns `sk-…XXXX` for any UI display
- [x] 5.4 [IF] Implement Settings page at `/settings/providers` rendering one card per registered provider; status (`set`/`unset`/`invalid`)
- [x] 5.5 [IF] Implement key-add flow: input → live validation request → on success persist + transition card → on failure surface error verbatim with key value redacted from any log
- [x] 5.6 [IF] Implement provider-specific validators per spec scenarios:
  - OpenAI: `GET /v1/models`
  - Anthropic: `GET /v1/models`
  - Google: `GET /v1beta/models?key=...`
  - AWS: signed STS GetCallerIdentity (combined access key + secret)
  - Azure: configurable endpoint health check
  - Moonshot: models endpoint
  - Copilot CLI / OpenCode CLI: documented validation endpoints
- [x] 5.7 [IF] Implement key-remove flow: purges from `KeyStore`, transitions card to `unset`, marks any canvas referencing the now-unconfigured provider
- [x] 5.8 [IF] Implement Settings → General page (CAO base URL, default provider, default working directory)
- [x] 5.9 [IF] Implement Settings → Appearance page (font picker for body/headings/terminal, custom font-family input with safe fallback, theme — SENTINEL only for v1)
- [x] 5.10 [IF] Wire `useValidatedProviders()` selector consumed by Canvas Builder, Agent Studio, Voice
- [x] 5.11 [IF] Implement non-blocking inline notice "No providers configured — Deploy is disabled until you validate at least one in Settings" with link to `/settings/providers`; consumed by Canvas Builder when validated-provider set is empty (canvas remains interactive per v4.2 §12)

## 6. Canvas Document Layer (CV) — `canvas-document`

- [x] 6.1 [CV] Define TypeScript types for `CanvasDocument`, `CanvasNode`, `CanvasEdge`, `ProviderType`, `OrchestrationType` in `src/shared/canvas-types.ts` (SUP-gated review)
- [x] 6.2 [CV] Implement Zod schemas mirroring the TypeScript types; `parseCanvasDocument(unknown): CanvasDocument` validates and rejects malformed inputs
- [x] 6.3 [CV] Implement self-loop and dangling-edge validators with unit tests
- [x] 6.4 [CV] Implement entry-point invariant validators (zero/multiple) with unit tests
- [x] 6.5 [CV] Implement `CanvasStore` interface in `src/canvas-document/store.ts` with `list()`, `get(id)`, `save(doc)`, `delete(id)`, `listVersions(id)` backed by IndexedDB
- [x] 6.6 [CV] Add `schema_version` constant; implement future-version refusal logic with unit test
- [x] 6.7 [CV] Add per-save versioning: every save inserts a row in `canvas_versions` with the snapshot

## 7. Canvas Builder (CV) — `canvas-builder`

- [x] 7.1 [CV] Add `@xyflow/react` and integrate at the canvas builder route; verify a placeholder node renders in SENTINEL theming
- [x] 7.2 [CV] Implement custom node renderer for `agent` nodes using SENTINEL `Card` + `StatusBadge`
- [x] 7.3 [CV] Implement Agent Palette with the 4 starter blocks (Supervisor, Developer, Reviewer, Custom); drag from palette places a node at drop coordinates
- [x] 7.4 [CV] Implement role-template registry providing default `system_prompt`, `allowedTools`, `display_name`
- [x] 7.5 [CV] Implement entry-point invariant: first Supervisor added becomes entry-point; subsequent Supervisors do not auto-claim
- [x] 7.6 [CV] Implement edge drawing (default `handoff`); edge label menu allowing change to `assign` (dashed) or `send_message` (dotted) with style transitions ≤100 ms
- [x] 7.7 [CV] Implement Block Configuration Panel: `display_name`, `role`, `provider` (gated dropdown), `model` (lists all provider models with NO default and NO recommendation per v4.2 §8.10), `allowedTools`, `system_prompt` in Monaco; wire edits to live update the canvas
- [x] 7.8 [CV] Implement Save (Cmd/Ctrl+S and toolbar) and canvas list at `/` (sorted by `updated_at` desc; "New Canvas" creates draft)
- [x] 7.9 [CV] Implement undo/redo for at least 20 actions
- [x] 7.10 [CV] Implement Deploy button with disabled-state reasons (no entry point, multiple entry points, missing provider config, missing model selection, empty canvas); hover tooltip identifies the offending node where applicable
- [x] 7.11 [CV] Implement Templates picker (consumes `canvas-templates`) — invokable from canvas list, empty canvas, and toolbar
- [x] 7.12 [CV] Implement voice trigger button + `Ctrl+Shift+V` hotkey wiring to the `speech-to-canvas` panel
- [x] 7.13 [CV] Touch-detect: render canvas as read-only with banner on touch-only devices

## 8. Canvas Templates (CV) — `canvas-templates`

- [x] 8.1 [CV] Define `TEMPLATES` array with all 10 master-spec §4.8 entries (Code Review, Bug Triage, Documentation Sprint, Full Stack Dev, Data Pipeline, Security Audit, DevOps Pipeline, Research Team, Enterprise Squad, Blank Canvas) — each a fully-formed `CanvasDocument` definition
- [x] 8.2 [CV] Implement `instantiateTemplate(templateId): CanvasDocument` — fresh UUID, regenerated node/edge IDs, name "(copy)" suffix, `deploy_state.status: "draft"`
- [x] 8.3 [CV] Add metadata to each template (`agent_count`, `primary_edge_type`, `est_cost_per_hour_usd`); verify costs match master spec §4.8
- [x] 8.4 [CV] Add ⚠️-glyph rendering helper that templates picker, Dashboard, FinOps all reuse (consumed via `finops-tier1`)
- [x] 8.5 [CV] Tests: 10 entries present, each instantiation produces disjoint UUIDs, blank canvas yields zero nodes/edges

## 9. Canvas Reconciler (CV) — `canvas-reconciler`

- [x] 9.1 [CV] Implement profile-markdown generator: per node produces YAML frontmatter (`name`, `role`, `provider`, `allowedTools`) + `system_prompt` body
- [x] 9.2 [CV] Implement supervisor-prompt augmentation: append a "canvas topology" block listing allowed handoff/assign/send_message targets per master spec §4.4 step 5
- [x] 9.3 [CV] Implement Reconciler driver: profile installs → session creation → terminal additions, with atomic `deploy_state` persistence before AND after each CAO call (design D5)
- [x] 9.4 [CV] Implement deploy state machine transitions (`draft` ↔ `deploying` ↔ `deployed` / `degraded`); cover each scenario from `canvas-reconciler/spec.md` with unit tests using mocked `CaoClient`
- [x] 9.5 [CV] Implement deploy progress panel: 5-row list updated reactively as each call resolves
- [x] 9.6 [CV] Implement Retry Failed (only nodes absent from `terminal_map`)
- [x] 9.7 [CV] Implement Tear Down (`DELETE /sessions/{name}`, reset `deploy_state` to draft); preserve as a separate path even with edit-in-place support
- [x] 9.8 [CV] Implement Resume affordance for canvases found in `deploying` on reload
- [x] 9.9 [CV] **Implement diff-based edit-after-deploy** (per master spec §12 user override / design D14):
  - Capture per-node profile snapshots at deploy time so diffs can compare against canonical state
  - Implement diff algorithm with the 5 cases from `canvas-reconciler/spec.md` (added node, removed node, changed profile content, display-only change, edge change)
  - Block entry-point change with the dialog requiring Tear Down
  - Show "Reconciling…" indicator and block edits during in-flight diff
- [x] 9.10 [CV] Implement edge-change advisory banner: "Edge changes require Tear Down + redeploy to take effect on the supervisor"
- [x] 9.11 [CV] Tests: full happy-path 3-node deploy; partial-failure→degraded; all-fail→draft rollback; retry-from-degraded; reload-mid-deploy; diff-add-node; diff-remove-node; diff-change-profile-content; diff-display-only; diff-blocks-entry-point-change

## 10. Terminal Streaming (TM) — `terminal-streaming`

- [x] 10.1 [TM] Add xterm.js packages and CSS imports
- [x] 10.2 [TM] Implement `<TerminalView terminalId={...} themeOverride? readOnly? />` component
- [x] 10.3 [TM] Apply master-spec §6.3 zero-lag config exactly (`smoothScrollDuration:0`, `scrollback:10000`, fonts from CSS tokens, sizes, colors)
- [x] 10.4 [TM] Load WebGL, Fit, WebLinks, Search, Unicode11 addons in order; assert via test
- [x] 10.5 [TM] WebGL initialization check: production fails loudly; dev allows Canvas2D fallback only when `VITE_ALLOW_CANVAS2D=true`
- [x] 10.6 [TM] Implement binary-frame handler: `Uint8Array` directly to `terminal.write` with no string conversion (lint check verifies)
- [x] 10.7 [TM] Implement input handler: keystroke → JSON text frame `{type:"input",data:"..."}` (suppressed when `readOnly`)
- [x] 10.8 [TM] Implement `ResizeObserver` resize with 100 ms debounce → `{type:"resize",rows,cols}`
- [x] 10.9 [TM] Set initial dimensions to 220×50
- [x] 10.10 [TM] Implement connection-state pill (`connecting`/`connected`/`reconnecting`/`terminated`)
- [x] 10.11 [TM] Implement reconnect logic: 4003/4004 = permanent; other = exponential backoff 500 ms→30 s with ±20% jitter
- [x] 10.12 [TM] Apply SENTINEL theme to xterm
- [x] 10.13 [TM] Verify two side-by-side `<TerminalView />` instances do not share state (test for `terminal-grid` use)

## 11. Terminal Grid (TM) — `terminal-grid`

- [x] 11.1 [TM] Implement `<TabBar>` rendering one tab per terminal in the current session with `StatusBadge` + close affordance + trailing `+` tab
- [x] 11.2 [TM] Wire tab-bar polling to TanStack Query at 3 s interval (per master spec §9 terminal-status cadence)
- [x] 11.3 [TM] Implement Grid View (responsive 2×3 or 3×4) with mini-terminal cells (40×15 read-only), each consuming the WebSocket fan-out helper from `cao-integration`
- [x] 11.4 [TM] Implement cell-click expansion to focused tab view
- [x] 11.5 [TM] Implement Full-Screen mode toggle; Escape exits
- [x] 11.6 [TM] Implement per-terminal controls (working dir display, send-message input, inbox viewer, kill button) on focused/full-screen views; collapse to menu on grid cells
- [x] 11.7 [TM] Implement kill confirmation modal
- [x] 11.8 [TM] Add unit tests verifying one WebSocket per terminal id even when multiple consumers (focused tab + grid cell) display the same terminal

## 12. Chat View (TM) — `chat-view`

- [x] 12.1 [TM] Implement `<ChatView sessionName={...} />` component
- [x] 12.2 [TM] Implement output parser: strip ANSI/VT100 escapes, group lines by agent-prompt boundaries and tool-call markers, attribute chunks to terminal id
- [x] 12.3 [TM] Implement bubble rendering: SENTINEL Card-styled bubble per message with display_name, provider badge, timestamp, content; partial-message buffering shows typing state
- [x] 12.4 [TM] Implement inline send-message composer (Enter to send, Shift+Enter newline) → `POST /terminals/{id}/input`
- [x] 12.5 [TM] Implement Terminal Grid ↔ Chat View toggle in Orchestrator toolbar; persist choice per canvas in settings store
- [x] 12.6 [TM] Implement viewport-based default: ≤768 px = Chat View by default
- [x] 12.7 [TM] Implement touch affordances: swipe-left for per-message actions, `100dvh` sizing, composer above on-screen keyboard
- [x] 12.8 [TM] Test ANSI stripping: `\x1b[31mERROR\x1b[0m` renders as red "ERROR" without escapes

## 13. Speech-to-Canvas (VX) — `speech-to-canvas`

- [x] 13.1 [VX] Implement `VoiceCapture` class wrapping Web Speech API per master spec §5.4 (continuous, interimResults, pt-BR default, auto-restart on `no-speech`)
- [x] 13.2 [VX] Implement `WhisperTranscriber` fallback per master spec §5.5 (`MediaRecorder` 16 kHz mono webm/opus, 3-second chunks, OpenAI Whisper API call using user's BYOK key)
- [x] 13.3 [VX] Implement engine selection logic: Web Speech API default; fall back to Whisper when unavailable; user-forced selection from Settings → STT Engine
- [x] 13.4 [VX] Implement NLU intent extraction: structured-extraction call against the cheapest validated provider (Gemini Flash → GPT-4o-mini → Haiku per design D15); 3-second latency budget
- [x] 13.5 [VX] Implement bilingual NLU prompt template per master spec §5.6 with provider/edge-type mappings
- [x] 13.6 [VX] Implement `voiceToCanvas(intent)` per master spec §5.7 (auto-layout, default supervisor as entry-point, role-template defaults)
- [x] 13.7 [VX] Implement Voice UI panel with 5 states (`idle`/`listening`/`processing`/`confirming`/`error`); confirming step shows parsed intent summary with confidence and three actions (Cancel, Edit Before Deploy, Generate)
- [x] 13.8 [VX] Wire `Ctrl+Shift+V` / `Cmd+Shift+V` global hotkey to toggle voice input
- [x] 13.9 [VX] Implement disabled state when no LLM key is validated, with link to `/settings/providers`
- [x] 13.10 [VX] Tests: pt-BR canonical example yields the 4-node 3-edge canvas from spec; mixed-language utterance is parsed; mic permission released between activations; transcripts are not persisted

## 14. Voice Runtime Commands (VX) — `voice-runtime-commands`

- [x] 14.1 [VX] Implement `matchRuntimeCommand(transcript)` regex+keyword matcher per master spec §5.8 (kill, pause, focus, status, deploy, stop_all, cost, add_node, connect)
- [x] 14.2 [VX] Add bilingual pattern coverage for pt-BR + en-US
- [x] 14.3 [VX] Verify matcher latency ≤100 ms on 50-character transcript
- [x] 14.4 [VX] Wire commands to actions:
  - `kill` → `DELETE /terminals/{id}` after confirmation modal
  - `stop_all` → `DELETE /sessions/{name}` after confirmation modal
  - `pause` → `POST /terminals/{id}/input` with pause sentinel
  - `focus` → client-side navigation
  - `status` → reads `GET /sessions/{name}/terminals` and announces
  - `deploy` → invokes Reconciler if canvas valid; otherwise toast with disabled reason
  - `cost` → navigate to `/finops`
  - `add_node` / `connect` → delegate to Canvas Builder
- [x] 14.5 [VX] Implement confirmation modal for destructive commands (Cancel auto-focused)
- [x] 14.6 [VX] Test: unrecognized transcripts fall through to NLU (`speech-to-canvas`) rather than being silently dropped

## 15. Dashboard (DB) — `dashboard`

- [x] 15.1 [DB] Implement `/dashboard` route with KPI Row (Fleet Status, Cost / MTD, Budget Util, Threats) consuming TanStack Query for sessions and `finops-tier1` for cost
- [x] 15.2 [DB] Implement Cost-by-Provider bar chart using `recharts`, wired to `finops-tier1` cost-by-provider selector
- [x] 15.3 [DB] Implement Fleet Status donut chart (active/error/offline)
- [x] 15.4 [DB] Implement Activity Feed: inbox messages + session lifecycle events; unlimited retention per v4.2 §12 (no application-level cap), newest-first, with manual Clear affordance
- [x] 15.5 [DB] Implement Terminal Preview Card: read-only mini-terminal with click-to-navigate to full terminal view; consumes WebSocket fan-out helper
- [x] 15.6 [DB] Apply ⚠️ label to Cost / MTD KPI per `finops-tier1` mandatory rule
- [x] 15.7 [DB] Tests: KPIs update on session/terminal change within one polling interval; donut sums correctly

## 16. FinOps (DB) — `finops-tier1`

- [x] 16.1 [DB] Define `PROVIDER_COST_PER_HOUR` constant matching master spec §8.7 exactly; export from `src/finops/`
- [x] 16.2 [DB] Implement `computeCostEstimate(window, terminals): { total, byProvider, byCanvas }` using wall-clock × per-hour rate
- [x] 16.3 [DB] Implement `useCostEstimate()` hook wrapping the computation with TanStack Query for derived selectors
- [x] 16.4 [DB] Implement `<CostLabel value={...} />` component that ALWAYS renders the ⚠️ glyph and disclaimer tooltip
- [x] 16.5 [DB] Implement FinOps page at `/finops`: MTD cost, budget utilization gauge, cost-by-provider table, top-10 cost-by-canvas table, budget configuration affordance (writes to `settings` store)
- [x] 16.6 [DB] Add page footer noting Tier 2/3 are post-launch
- [x] 16.7 [DB] Tests: mixed-provider cost computation; ⚠️ label present on every cost surface (Dashboard KPI, FinOps tables, Templates picker); top-10 sorting

## 17. Health & Onboarding (DB) — `health-and-onboarding`

- [x] 17.1 [DB] Implement Health page at `/health` with three sections: Server Health, Provider Validations, Browser Capabilities
- [x] 17.2 [DB] Implement browser-capability checks: WebGL2 detect, IndexedDB detect, microphone permission status (using `navigator.permissions.query({ name: "microphone" })` where supported)
- [x] 17.3 [DB] Implement "Test Microphone" affordance that requests `getUserMedia` and updates the row status
- [x] 17.4 [DB] Implement Fix affordances per failed-check type (link to Settings or external docs)
- [x] 17.5 [DB] Implement First-Run Wizard with three steps: Verify CAO → Configure provider (embedded mini Settings) → Pick starting point (Templates picker + Start Blank)
- [x] 17.6 [DB] Wizard skip logic: skippable at any step; remembers completion via `app_state` IndexedDB store
- [x] 17.7 [DB] Subsequent visits with at least one validated provider + one canvas skip the wizard
- [x] 17.8 [DB] Tests: first visit triggers wizard; subsequent visit skips; CAO outage reflected in Health row

## 18. Agent Studio (ST) — `agent-studio`

- [x] 18.1 [ST] Implement profile list at `/agent-studio` from `GET /agents/profiles` with name/role/provider/description; searchable + filterable
- [x] 18.2 [ST] Implement provider availability panel from `GET /agents/providers` (CAO-managed install state)
- [x] 18.3 [ST] Flag profiles whose provider reports `installed: false`
- [x] 18.4 [ST] Implement profile detail viewer: parsed markdown body (SENTINEL prose), YAML frontmatter as key-value list, metadata
- [x] 18.5 [ST] Implement Profile Editor: form for frontmatter (provider dropdown gated by validated set), Monaco-style markdown body editor, Save invokes `POST /agents/profiles/install`
- [x] 18.6 [ST] Implement Install From Source: built-in store (curated profiles shipped with AgentVerse), local file picker (`.md` upload), URL fetch with preview-before-install confirm
- [x] 18.7 [ST] Surface CAO validation errors verbatim
- [x] 18.8 [ST] Tests: install flow end-to-end via MSW; provider gating; markdown rendering

## 19. Flows (ST) — `flows`

- [x] 19.1 [ST] Implement Flow list at `/flows` from `GET /flows` with `cronstrue` for human-readable schedule; refresh on 15 s poll
- [x] 19.2 [ST] Implement quick-pick schedule UI (every-N-minutes, hourly, daily-at-time, weekdays-at-time, weekly) + raw cron input with `cronstrue` validation
- [x] 19.3 [ST] Implement create/edit form with all `Flow` fields (name, schedule, agent_profile selector, provider selector gated by validated set, prompt_template Monaco editor, enabled toggle)
- [x] 19.4 [ST] Implement Run Now action issuing `POST /flows/{name}/run` with toast confirmation
- [x] 19.5 [ST] Implement Enable/Disable toggle (optimistic update with revert on failure)
- [x] 19.6 [ST] Implement gating-script display: "Conditional" badge + hover text when present
- [x] 19.7 [ST] Tests: invalid cron rejected before submit; quick-pick fills cron correctly; toggle persists after refresh

## 20. Memory Viewer (ST) — `memory-viewer`

- [x] 20.1 [ST] Implement memory list at `/memory` reading via per-terminal context API plus agent-dirs setting; clear empty state where direct listing isn't supported in v1
- [x] 20.2 [ST] Implement scope filter (global / project / session / agent) and type filter (project / user / feedback / reference) and tag filter
- [x] 20.3 [ST] Implement detail viewer with parsed markdown body (SENTINEL prose), scope/type metadata, tags, retention info, location path
- [x] 20.4 [ST] Implement full-text search across content + tags
- [x] 20.5 [ST] Implement manual memory creation form (validation, scope, type, tags, content)
- [x] 20.6 [ST] Implement retention notice for session-scoped memories ("Persists until session `<name>` ends")
- [x] 20.7 [ST] Tests: filters narrow correctly; search matches case-insensitively; empty-state copy is present where applicable

## 21. Cross-Cutting Quality Gates (SUP)

- [x] 21.1 [SUP] Add Playwright smoke spec `smoke.spec.ts` covering the v1 critical path: configure provider → create canvas (template or blank) → drop nodes + edges → deploy → see terminal output → invoke voice command
- [x] 21.2 [SUP] Run smoke against MSW-mocked CAO in CI; weekly run against live CAO container
- [x] 21.3 [SUP] Bundle size budget: `dist/` total ≤ 1.5 MB gzipped (looser than M1's 500 KB given full v1 surface); report breakdown
- [x] 21.4 [SUP] Performance test: 12+ concurrent terminals streaming simultaneously without dropped frames (per master spec §12 polish)
- [x] 21.5 [SUP] Accessibility audit (axe) on all v1 routes; fix critical and serious findings
- [x] 21.6 [SUP] Document open questions resolved during implementation in `docs/v1-decisions.md`
- [x] 21.7 [SUP] Document any post-v4.2 spec deltas as separate change proposals — v4.2 itself is treated as the locked baseline
- [x] 21.8 [SUP] Document supervisor-prompt augmentation pattern in `docs/canvas-topology-prompt.md` (used by `canvas-reconciler` 9.2)
- [x] 21.9 [SUP] Run nightly `CAO_LIVE=1` contract suite against a live CAO; alert on shape drift

## 22. v1 Close

- [x] 22.1 [SUP] All capability unit tests passing; per-capability coverage ≥70% for logic-heavy modules

- [x] 22.2 [SUP] MSW integration tests passing across capability boundaries
- [x] 22.3 [SUP] Playwright smoke green
- [ ] 22.4 [SUP] Manual demo: 3-node canvas (Supervisor → Developer → Reviewer with handoff edges) deploys cleanly; voice generates the same canvas; edit-after-deploy adds a Reviewer; Tear Down cleans up. Record demo video.
- [ ] 22.5 [SUP] Tag `v1.0.0`; archive this change via `/opsx:archive`
- [x] 22.6 [SUP] Open follow-up change proposals: `validation-proxy`, `cloud-runtime-deployment`, `finops-tier2-token-parsing` (post-launch capabilities tracked in `openspec/changes/`)
