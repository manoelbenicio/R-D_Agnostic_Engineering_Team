# OmniRoute `docs/architecture` — Full Study, Read-Evidence & Roadmap Synthesis (2026-07-24)

Source: `github.com/diegosouzapw/OmniRoute` @ `release/v3.8.49`, path `docs/architecture/`.
All **13** files read in full (read-only, no changes to OmniRoute). Evidence ledger + roadmap synthesis below.

## Files covered (13/13)
ARCHITECTURE.md · AUTHZ_GUIDE.md · CODEBASE_DOCUMENTATION.md · REPOSITORY_MAP.md · RESILIENCE_GUIDE.md ·
QUALITY_GATES.md · ROUTER_BACKENDS.md (ADR) · cluster-decisions.md (ADR) · persistence-backend-boundary.md (ADR) ·
sqlite-coupling-inventory.md · DESIGN_SYSTEM.md · MONITORING_SECTIONS.md · meta.json.
(meta.json declares only 6 nav "pages"; the other 7 are ADR/reference docs in the same folder.)

## Proof-of-read: one distinctive verbatim detail per file
- **meta.json** — pages = [ARCHITECTURE, AUTHZ_GUIDE, CODEBASE_DOCUMENTATION, REPOSITORY_MAP, RESILIENCE_GUIDE, QUALITY_GATES].
- **ARCHITECTURE.md** — "271 providers, 86 executors"; OpenAI hub format; bypass handler only when UA contains `claude-cli`; DATA_DIR→XDG→~/.omniroute; Auto-Combo called "9-factor" here.
- **AUTHZ_GUIDE.md** — PUBLIC/CLIENT_API/MANAGEMENT; unclassifiable→MANAGEMENT (fail-closed); `x-omniroute-auth-*` stripped; AUTH_001; `/api/mcp/*` manage-scope bypass (v3.8.2) but `/api/cli-tools/runtime/*` NOT bypassable; hasManageScope = manage|admin.
- **CODEBASE_DOCUMENTATION.md** — Node `>=22.22.2 <23 || >=24.0.0 <27`; TS strict:false; port 20128; "84 executors" (§4.2); coverage ≥60%; localDb.ts re-export only.
- **REPOSITORY_MAP.md** — 236 providers; 17 strategies; 67 executors; MCP "94 tools/3 transports/30 scopes"; baselines→config/quality in v3.8.26.
- **RESILIENCE_GUIDE.md** — breaker trip codes [408,500,502,503,504]; OAuth 5/8/60s, API-key 7/12/30s, Local -/2/15s; model lockout OFF by default, success-decay halves failureCount; queue maxWaitMs 120s→15s (v3.8.49); session-affinity #7274 generalized from codex-only.
- **QUALITY_GATES.md** — ~50 gates; check:cycles(blocking) vs check:circular-deps dpdm "91 cycles"(advisory); Playwright retries:1 / Vitest none / node:test never; typecheck:noimplicit advisory.
- **ROUTER_BACKENDS.md** — two axes lifecycle{in-process,supervised,external,disabled} × relay{ts,bifrost,auto}; cliproxy :8317 (/v1/models), 9router :20130 (/api/health); bifrost external-only until #5817; forced bifrost fail → hard 502.
- **cluster-decisions.md** — opt-in profiles memory(Qdrant v1.12.4 :6333)/bifrost(bifrost:1.5.21 :8080); default 3×omniroute+Caddy+Redis+CliproxyAPI; DROP Dragonfly/NATS/PostgreSQL/Neo4j/MinIO/pgvector/HAProxy/Envoy w/ reasons.
- **persistence-backend-boundary.md** — ADR proposed (#8075); domain repo contracts + internal async backend; SQLite default; Postgres first external, MySQL peer; portable repo forbids prepare/get/all/run,PRAGMA,VACUUM,lastInsertRowid,FTS5/sqlite-vec; Redis rejected as durable authority.
- **sqlite-coupling-inventory.md** — rev 9a3b605…, corpus sha256 72334620…; 3,830 src files + 129 migrations; getDbInstance() 45 files/150 occ outside src/lib/db; .prepare() 1,219/163; datetime('now') 171; fts5 43; vec0 7.
- **DESIGN_SYSTEM.md** — grid 32px (from 46px); --primary #e54d5e / --accent #6366f1 / --grad-brand red→violet; body::before z-index:-1; radius 14/9; D8 shell max-w-7xl→max-w-[3840px].
- **MONITORING_SECTIONS.md** — Costs = new top-level (Group B/plan 16); Activity(/dashboard/activity,level=high) vs Audit(/dashboard/audit,level=all); /dashboard/logs/activity 308 redirect; source locales pt-BR+en, 39 fallback.

## Cross-doc integrity note
Counts drift across docs (own `check:docs-counts` ratchet acknowledges lag): MCP tools 94/99/104/31/34; providers 236/237/268/271/290; strategies 17/18/19; migrations 52/55/106/110; version headers `v3.8.40` while branch is `v3.8.49`.

## Architecture understanding (roadmap-relevant)
1. **Shape:** single-process Next.js 16 app (UI + `/v1` API + libs + domain + server) + `open-sse` streaming workspace (handlers → translator[OpenAI hub] → 84 executors) + Electron + CLI. SQLite (`better-sqlite3`, WAL) is the durable store; Redis for rate-limit/cache; Caddy LB/TLS.
2. **Routing:** combos + Auto-Combo (9–12-factor scoring depending on doc) + 17–19 strategies incl. fusion/pipeline; 4-tier fallback; per-request steer headers; self-healing.
3. **Resilience:** 3 independent layers (provider breaker / connection cooldown / model lockout) + quota-share serialization + request-queue admission. Fail-OPEN for routing continuity.
4. **AuthZ:** deterministic fail-closed 3-class pipeline; LOCAL_ONLY spawn routes (services, cli-tools runtime, MCP); MCP manage-scope carve-out.
5. **Scaling posture (ADRs):** scale-out is **opt-in sidecars** (Qdrant vector memory, Bifrost Tier-1 router), NOT a DB migration. PostgreSQL/MySQL are only a **proposed** persistence-boundary ADR (#8075) gated behind SQLite-first conformance; SQLite remains the zero-config default. Postgres/Dragonfly/NATS/Neo4j/MinIO explicitly rejected for current workload.
6. **Router-backend abstraction** (ts/bifrost/auto) is emerging; per-request backend selection in-progress (#5869/#5870).

## Implications for OUR product roadmap (Main Brain / Multica)
- Consuming OmniRoute as the sole router + credential owner is architecturally correct; Main Brain stays credentialless and just hits `/v1` on the gateway endpoint.
- If we self-host/scale OmniRoute: default = 1 process + Redis + Caddy (+3 replicas); vector memory (Qdrant) and Bifrost are opt-in; do NOT assume Postgres — SQLite is the default and their Postgres path is unapproved/proposed.
- OmniRoute's **MCP** is a management/control surface (health/quota/cost/explain-route/combos) reachable via a `manage`/`mcp:connect`-scoped key (LOCAL_ONLY otherwise) — a candidate for read-only ops telemetry, separate from the `/v1` inference path.
- Their resilience is fail-open; keep Main Brain's strict fail-closed readiness (it already is).
- Codex/Chat 2.2: their model = Codex is OAuth `openai-responses`; our combo set has no Codex route (verified via MCP), so Chat 2.2 needs OmniRoute to publish an approved Codex route.


---

# Part 2 — Full docs-tree word-by-word pass (release/v3.8.49), 2026-07-24

Principal API & Solutions Architect read-through of the remaining OmniRoute doc directories, per owner instruction ("read all, no exception"). Source = `raw.githubusercontent.com/diegosouzapw/OmniRoute/release/v3.8.49/docs/...`. Per-file proof-of-read = one distinctive detail only present in that file.

## docs/reference (11 — complete)
- meta.json — pages [API_REFERENCE, CLI-TOOLS, ENVIRONMENT, FEATURE_FLAGS, FREE_TIERS, FREE_PROXIES_API, PROVIDER_REFERENCE].
- API_REFERENCE.md — `X-OmniRoute-Decision: strategy=…;provider=…;latency_ms=…`; `Idempotency-Key` 5s window; cache-hit cost header `0.0000000000`; `x-omniroute-no-memory`.
- CLI-TOOLS.md — 3 pages (CLI Code 21/25, CLI Agents 6/8, ACP); `baseUrlSupport: none` (cursor/antigravity/hermes/kiro) → MITM backlog.
- ENVIRONMENT.md — required secrets JWT_SECRET/API_KEY_SECRET/INITIAL_PASSWORD/OMNIROUTE_WS_BRIDGE_SECRET; PORT=20128; DATA_DIR=~/.omniroute; emergency fallback nvidia/gpt-oss-120b.
- FEATURE_FLAGS.md — 38 flags / 6 categories; resolution DB>env>default; OMNIROUTE_MCP_ENFORCE_SCOPES default true.
- FREE_TIERS.md — headline ~1.53B free tokens/mo (43 pools, deduped); mistral 1.00B biggest; rejects ~10B RPM×24/7 inflation; CI-gated by computeFreeModelTotals().
- FREE_PROXIES_API.md — free_proxies table synced from 1proxy/proxifly/iplocate/webshare; syncErrors keyed by source.
- PROVIDER_PLUGIN_MANIFEST.md — GET /api/v1/provider-plugin-manifest, Cache-Control max-age=60+ETag; excludes OAuth secrets.
- PROVIDER_REFERENCE.md — auto-generated, Total providers **290**; OAuth 23, Web cookie 31; tool-calling native/emulated/none.
- RELAY_BACKEND_STRATEGY.md — baseline OMNIROUTE_RELAY_BACKEND=auto + BIFROST_ENABLED=1; forced bifrost = hard 502 no fallback.
- RELAY_TROUBLESHOOTING.md — relay auth in proxy `notes` as relayAuthEnc(AES)/relayAuth; POST /api/settings/proxies/[id]/repair-relay → {repaired,mode}.

## docs/providers (5 — complete)
- meta.json — [AGENTROUTER, CLAUDE_WEB, ALIBABA-QWEN-PROVIDER-FAMILIES, ZED-DOCKER].
- AGENTROUTER.md — native `agentrouter` provider ships Claude Code wire image (UA `claude-cli/2.1.207`, anthropic-beta claude-code-20250219…); advanced path needs ENABLE_CC_COMPATIBLE_PROVIDER=true.
- ALIBABA-QWEN-PROVIDER-FAMILIES.md — issue #7854; 4 families (alibaba, bailian-coding-plan[Anthropic wire], qwen-cloud, qwen-cloud-token-plan); region=connection data (global-sg/china-beijing); alibaba-cn compat alias.
- CLAUDE_WEB.md — claude-web executor over authenticated claude.ai session; 7 static models incl claude-opus-4-8/sonnet-5/fable-5; conversation cache 30min/5000-cap; TLS "Chrome 146"; WEB_COOKIE_USE_BROWSER Playwright fallback; 16 MiB body cap.
- ZED-DOCKER.md — keychain import fails in Docker (fs+IPC isolation); detects /.dockerenv or docker in /proc/1/cgroup → HTTP 422 zedDockerEnvironment:true → POST /api/providers/zed/manual-import.

## docs/guides (20 — complete)
- SETUP_GUIDE.md — npm/pnpm(--allow-build)/AUR/Docker/Electron/headless; `setup-*` per-CLI writers; tokenized `/api/v1/vscode/KEY/` base for header-less clients.
- USER_GUIDE.md — pricing tiers table (SUBSCRIPTION/API-KEY/CHEAP/FREE); use-case combos maximize-claude/free-forever/always-on/openclaw-free; Haiku rejects `max` effort → downgraded to high.
- FEATURES.md — dashboard gallery; v3.8 highlights (auto-combo, reasoning replay cache, per-session sticky routing, model cooldowns dashboard); Settings has 7 tabs; 17 combo strategies incl fusion.
- CLAUDE-CODE-CONFIGURATION.md — (read in Part 1) `ANTHROPIC_BASE_URL`/`ANTHROPIC_AUTH_TOKEN`; per-model profile dirs.
- CODEX-CLI-CONFIGURATION.md — wire_api="responses" is required+default (chat crashes v0.138+); OmniRoute Responses↔Chat transformer; auto_compact ≤90% window; cx/ prefix; effort none/low/medium/high/xhigh.
- CLI-INTEGRATIONS.md — master table of setup-codex/claude/opencode/cline/kilo/continue/cursor/roo/crush/goose/aider/qwen + launchers launch/launch-codex (env-inject, no config).
- REMOTE-MODE.md — contexts in ~/.omniroute/config.json chmod600; oma_live_… scoped tokens read/write/admin; process-spawn routes stay loopback-only even w/ admin; Antigravity remote-login blob omniroute-cred-v1.…
- USAGE_QUOTA_GUIDE.md — usage record fields; per-key quotaLimit/quotaWindow → 429; quotaSnapshots table; ~4 chars/token estimate for web-cookie providers.
- COST_TRACKING.md — "cost = savings tracker, not a bill"; pricing precedence user-override > synced LiteLLM > hardcoded; PRICING_SYNC_ENABLED default false; SpendBatchWriter 60s/1000-entry.
- DOCKER_GUIDE.md — 4 compose profiles (base/cli/host/cliproxyapi); Redis sidecar always-on (rate limiter); Dockerfile targets builder/runner-base/runner-cli; OMNIROUTE_MEMORY_MB heap override.
- TROUBLESHOOTING.md — quick-ref error table; Avast/Kaspersky false-positive explanations (unsigned NSIS PDM:Trojan); macOS better-sqlite3 rebuild; Docker IPv6 -p 127.0.0.1 fix.
- TIERS.md — 3 economic tiers (Subscription/Cheap<$1-1M/Free); tierDefaults.json; combo patterns pure-free & subscription-first.
- FREE_PROVIDER_RANKINGS.md — Arena ELO (LMArena-style) sync from api.wulong.dev; taskFit=0.4+0.58·norm; entries expire 7d; ARENA_ELO_SYNC_ENABLED default true.
- KIRO_SETUP.md — v3.8.0 per-connection registerClient() OIDC isolation (fixes multi-account token revocation); refresh token starts `aorAAAAAG`; OIDC client ~90d expiry; API-key path via ListAvailableProfiles→profileArn.
- ELECTRON_GUIDE.md — Electron 41 + electron-builder 26.10; spawns Next standalone as child; contextIsolation IPC whitelist; zero-config secret bootstrap (JWT/STORAGE_ENCRYPTION_KEY/API_KEY_SECRET) → server.env.
- I18N.md — 43 languages; hash-based incremental translator via cx/gpt-5.4-mini; .i18n-state.json SHA-256 drift gate (npm run i18n:check); RTL ar/he.
- PWA_GUIDE.md — installable PWA; sw.js caching strategies per asset type; API routes bypass cache; no push (Electron handles that); Termux server+PWA-client on one phone.
- TERMUX_GUIDE.md — headless Android; Node >=22.22.2 required (nodejs-lts 20 unsupported); Termux:Boot autostart; no Electron/tray.
- UNINSTALL.md — npm run uninstall (keep data) vs uninstall:full (erase ~/.omniroute); storage.sqlite + call_logs/ + backups/.
- meta.json — guides nav index.

## docs/frameworks (24 — complete)
- OPEN_SSE_ARCHITECTURE.md — @omniroute/open-sse workspace ~900 files; 5-stage pipeline ROUTE→TRANSLATE→EXECUTE→STREAM→RECORD; chatCore.ts 5977 LOC; combo.ts 4456 LOC; 17-strategy table.
- MCP-SERVER.md — **104 unique tools** via countUniqueMcpTools() (42 canonical + memory3+skills4+github3+pool6+gamification8+plugins8+notion6+obsidian22+RTK2); 3 transports; /api/mcp/* LOCAL_ONLY + manage-scope bypass since v3.8.2.
- A2A-SERVER.md — JSON-RPC 2.0 POST /a2a + REST /api/a2a/*; 6 skills (smart-routing/quota/discovery/cost/health/list-capabilities); A2ATaskManager default 5-min TTL; Agent Card version auto-synced.
- AGENT_PROTOCOLS_GUIDE.md — decision tree A2A vs ACP vs Cloud Agents; A2A=in-OmniRoute compute, Cloud=external repo-aware.
- ACP.md — CLI-as-backend transport; spawns CLI as child proc (13 built-in agents); stdio; 2s idle-complete; spawn allowlist [claude,codex,gemini,qwen].
- CLOUD_AGENT.md — 4 agents jules/devin/codex-cloud/cursor-cloud (Codex/Cursor no approval gate); lazy status sync (no bg poller); cloud_agent_tasks column-whitelist; management auth (commit 588a0333).
- EVALS.md — built-in suites (golden-set, coding-proficiency, safety-guardrails…); rubrics exact/contains/regex/custom(builtin-only); no LLM-judge yet; management-auth only.
- GAMIFICATION.md — local-first XP/levels/badges/streaks/leaderboards; fire-and-forget off hot path; xp_for_level(n)=floor(100·n^1.5); server-authoritative anti-cheat.
- MEMORY.md — **OFF by default (v3.8.30+)**; per-API-key scope; 3-tier retrieval FTS5→sqlite-vec→Qdrant; embedding auto(remote→static→transformers); hybrid RRF k=60; x-omniroute-no-memory disables memory+skills.
- NOTION_CONTEXT.md — token in SQLite key_value ns notion (no env); Notion-Version 2026-03-11; 6 MCP tools (read:notion/write:notion); no public /v1 proxy.
- OBSIDIAN_CONTEXT.md — Local REST API plugin HTTP 27123 (NOT 27124); token AES-256-GCM at rest; 22 MCP tools; optional WebDAV vault sync (27781); per-API-key context source override.
- OPENCODE.md — two paths (CLI generator vs @omniroute/opencode-provider npm); emits @ai-sdk/openai-compatible provider; /v1 dedup (fixes /v1/v1 breakage).
- PLUGIN_MARKETPLACE.md — WordPress-style; /api/plugins LOCAL_ONLY; seed registry (request-logger/rate-limiter/cost-tracker/theme-manager); one-click marketplace install "coming soon".
- PLUGINS.md — CLI plugin system omniroute-cmd-* (like gh extension); register(program,ctx); ctx.apiFetch/emit/t/withSpinner.
- PLUGIN_SDK.md — definePlugin() onRequest/onResponse/onError; blockRequest/modifyBody/addMetadata; permission model network/file-read/file-write/env/exec; VM sandbox.
- SKILLS.md — Omni Skills (LLM tool exec) vs Agent Skills (SKILL.md catalog); builtins file_read/write/http/web_search/eval_code/execute_command/browser; Docker sandbox --network none --cap-drop ALL; AUTO scoring min 3 / max 5.
- AGENT-SKILLS.md — 42 canonical (22 API + 20 CLI) SKILL.md; generator preserves CUSTOM blocks; REST /api/agent-skills/* + 3 MCP tools + A2A list-capabilities.
- AGENTBRIDGE.md — MITM proxy intercepting 9 IDE agents; per-SNI cert signed by persisted root CA (#6684); /etc/hosts DNS redirect → server.cjs:443; secret masking before Traffic Inspector.
- TRAFFIC_INSPECTOR.md — LLM/agent-aware HTTPS debugger; 5 capture modes (agent-bridge/custom-host/http-proxy/system-proxy/tproxy); TrafficBuffer ring 1000; SSE stream merger.
- EMBEDDED-SERVICES.md — 4 sidecars 9Router(20130)/CLIProxyAPI/Mux(8322)/Bifrost(8080); ServiceSupervisor spawn+health+ring buffer; all /api/services/* LOCAL_ONLY; key AES-256-GCM.
- PLAYGROUND_STUDIO.md — 4 tabs Chat/Compare(4 models ‖)/API(Monaco 10 endpoints)/Build(tools+structured); improve-prompt route; presets SQLite.
- SEARCH_TOOLS_STUDIO.md — 3 tabs Search/Scrape(/v1/web/fetch, 256KB cap)/Compare(4 providers); ProviderCatalog kind search|fetch + status; shared ExportCodeModal.
- meta.json — nav lists 16 pages (NOTE: omits EMBEDDED-SERVICES, PLAYGROUND_STUDIO, SEARCH_TOOLS_STUDIO, AGENT-SKILLS, TRAFFIC_INSPECTOR, AGENTBRIDGE, OPEN_SSE_ARCHITECTURE — nav/file drift).

## docs/diagrams (README + 9 Mermaid sources; SVGs are rendered images)
- README.md — canonical .mmd→exported/.svg pairs + hand-authored animated SVGs (SMIL-only, GitHub sandbox-safe); render via `npm run docs:render-diagrams`.
- request-pipeline.mmd — Client→Route→CORS→Zod→AuthZ→Policy→Guard→ChatCore→Cache→Rate→Combo(18 strategies)→Translate→getExecutor(31)→Upstream(177)→responsesTransformer.
- resilience-3layers.mmd — L1 Provider breaker (CLOSED/OPEN/HALF_OPEN, trips on 408/5xx) → L2 Connection cooldown (401/403) → L3 Model lockout (429 quota).
- authz-pipeline.mmd — strip trusted internal headers → classifyRoute PUBLIC/CLIENT_API/MANAGEMENT → policy → stamp x-omniroute-auth-*; management allows Bearer w/ manage scope.
- auto-combo-12factor.mmd — **12-factor** scoring weights sum=1.0 (health .20/quota .15/costInv .15/latencyInv .12/taskFit .08/… resetWindowAffinity .00) — supersedes "9-factor" wording elsewhere.
- db-schema-overview.mmd — erDiagram of core tables (api_keys, provider_connections, domain_circuit_breakers, combos/targets/executions, memory_documents/chunks, agent_tasks/events, audit_log).
- mcp-tools-104.mmd — derivation of the 104 count by category (documents why other docs say 87/94/95/104).
- cloud-agent-flow.mmd — sequence: createTask→Zod+mgmt-auth→registry→agent→DB; poll getStatus; approvePlan; async upstream (agents codex-cloud/devin/jules).
- i18n-flow.mmd — source MDs→sha256→.i18n-state.json→per-locale(39) LLM cx/gpt-5.4-mini→docs/i18n/<locale>→CI drift gate.

## docs/comparison/OMNIROUTE_VS_ALTERNATIVES.md
- Self-audited positioning vs LiteLLM/OpenRouter/Portkey: 237+ providers, 90+ free-tier, 15+ OAuth, 17 strategies + Fusion + Tier1/2/3, 10-engine compression, MCP(95)+A2A(6), JA3/JA4 stealth, MITM, MIT license. Capabilities reliable; numbers approximate.

---

## Cross-doc count-drift catalog (for roadmap accuracy — treat exact numbers as approximate)
| Metric | Values seen across docs |
|---|---|
| Providers | 177 (request-pipeline.mmd) · 207+ (TIERS) · 226+/237+ (AgentBridge/comparison) · 268/271/290 (readme-hero/PROVIDER_REFERENCE) |
| Free-tier pools/providers | 39→43 pools (FREE_TIERS) · 90+/160+/180+ (rankings/A2A) |
| Routing strategies | 17 (FEATURES/OPEN_SSE) · 18 (request-pipeline.mmd/strategies-grid) |
| Auto-combo factors | 9-factor (FEATURES/USER_GUIDE) · **12-factor** (auto-combo-12factor.mmd, weights sum 1.0) |
| Executors | 31 (request-pipeline.mmd) · 67/68 (OPEN_SSE_ARCHITECTURE) |
| MCP tools | 87 (SETUP/agent-skills) · 94/95/99 (comparison/live) · **104** (MCP-SERVER + mcp-tools-104.mmd, canonical formula) |
| Agent skills | 42 canonical (AGENT-SKILLS/A2A) |
Root cause: docs authored at different versions; only FREE_TIERS totals and MCP-tool count are CI-gated. **For roadmap decisions, rely on capabilities + CI-gated numbers, not headline counts.**

## Roadmap synthesis (net-new from Part 2)
1. **Programmatic control surface is rich and supported**: MCP (104 tools, manage-scope remote bypass), A2A (6 skills incl smart-routing), REST management. If Main Brain ever needs to drive OmniRoute beyond inference (combo switch, budget guard, resilience profile, simulate/explain route), the read-only tools we already exercised extend to write tools — all gated by scopes + LOCAL_ONLY, consistent with our consume-only posture.
2. **Resilience is exactly the 3-layer fail-open model** (provider breaker → connection cooldown → model lockout) with explicit HTTP-code triggers — matches what we verified live for Topic-1 (MB 6.3). Instrumented-run evidence should assert on these three layers.
3. **Cost = savings tracker** (not a bill) and **memory OFF by default** — important for our credentialless product framing: no surprise billing, no injected context unless opted-in; `x-omniroute-no-memory` is available per request.
4. **Codex reality confirmed** (CODEX-CLI-CONFIGURATION): OmniRoute exposes Codex via `cx/` OAuth models + Responses↔Chat transformer, but this is the **client-config** path; there is still **no `claude_code_*`-style Codex ROUTE** in the live combo set — consistent with Chat 2.2 (@codex) being externally blocked.
5. **Scaling stays SQLite-first**: EMBEDDED-SERVICES + DOCKER_GUIDE confirm horizontal features are opt-in sidecars (Qdrant/Bifrost/9Router/Redis), not a DB migration — reinforces the earlier ADR-8075 (Postgres only proposed) finding.
6. **AgentBridge/Traffic Inspector/TPROXY** are powerful but MITM/root-scoped and LOCAL_ONLY — out of scope for our consume-only integration; noted for security awareness (a trusted MITM CA is high-privilege).

All items remain OPEN; nothing here authorizes an OpenSpec close or an orq1/deploy/git action. No topic closes without OmniRoute per-topic agreement + owner sign-off.
