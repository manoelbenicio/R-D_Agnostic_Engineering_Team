# GATE 1(a)/(b) — live chat routing functional smoke (owner-authorized)

- provenance: orq1 (Tailscale 100.118.244.61), deployed stack multica-dev-transition
  - containers (initial state, UNCHANGED): frontend/backend/postgres all RUNNING/healthy (up ~21h);
    postgres multica-dev-transition-postgres-1 pgvector/pgvector:pg17, healthy, accepting connections.
  - backend http://127.0.0.1:18080/health=200; web http://127.0.0.1:13100/=200
  - workspace orq2-dev id=20fce817-895d-447b-965a-49f5e279314a
  - default squad "Workspace Team" id=3223178f-1f78-4bee-b7ef-38eec6cd69f0 leader_id=cfc9d0cc-afb7-45cb-953c-cab62b09a948 (TL, agent claude-brain-canary)
- executed (no inference; stdlib urllib via ssh-exec on orq1; workspace_id as query param):
  - 1a POST /api/chat/sessions?workspace_id=WS body {} -> HTTP 201, session.agent_id=cfc9d0cc (leader) => ASSERT untargeted->TL PASS
  - 1b POST /api/chat/sessions?workspace_id=WS body {"agent_id":cfc9d0cc} -> HTTP 201, session.agent_id=cfc9d0cc => ASSERT explicit->direct PASS
- non-claims: only leader agent provisioned (member_count=1); no distinct Codex coder member -> "->Codex" specificity is a SETUP GAP, not a routing-code defect. No inference performed. No container start/stop/recreate/reset. Data preserved (2 test chat sessions created = additive).
- verdict: GATE 1(a)/1(b) PASS (live). GATE 1(c) Kanban->terminal (live Main Brain->CLI->OmniRoute) pending; gates 2/3 capacity pending.

## GATE 1(c) — Kanban task -> persisted terminal (live) : BLOCKED
- trigger: POST /api/issues/ORQ-7/rerun -> 202, task_id=d9079555-df82-40d0-a038-c263880debc9 (agent=TL leader cfc9d0cc), issue "report cwd" (read-only).
- observed: task stayed status=queued for full ~90s poll; backend log "task enqueued" + "issue rerun enqueued" present; internal scheduler alive (usage rollup claimed) BUT no claim/dispatch/launch of the agent task and NO Main Brain execution daemon process in backend container (ps: none).
- ROOT CAUSE: deployed stack has API+scheduler but the Main Brain task-execution daemon (claim -> launch CLI -> OmniRoute) is NOT running -> task never dispatched.
- cost: ZERO OmniRoute inference incurred (task never launched).
- BLOCKER owner/next: start+configure Main Brain execution daemon (requires OmniRoute gateway config); this is a service-start beyond safe inspection and touches gateway config -> explicitly reported per safeguard, needs owner direction. Gates 2/3 (tier-20/50/100) NOT started (sequential; 1c must pass first).
- NOT an operational finish.

## POST-DAEMON live results (owner full authorization)
- DAEMON: Main Brain host daemon on orq1 RUNNING+AUTHENTICATED+READY (login->workspace orq2-dev; health 19514=200; rotation store enabled via DATABASE_URL from existing dev.env creds; gateway.env secret-file ref sourced, values never printed). Launch via atd-detached launcher; auth via mul_ PAT (POST /api/tokens 201) -> multica --server-url login --token (0600 file, never printed).
- PROVISIONED: Kiro-TL agent bd1ebd0a on ONLINE Kiro runtime eb41a0c9 (squad leader via PUT 200); Codex agent 9d4321bb on ONLINE Codex runtime 7e8afc71 (member). member_count=3.
- TEST 1 untargeted -> Kiro-TL: HTTP 201 routed to Kiro-TL leader => PASS (routing).
- TEST 2 explicit agent_id=Codex -> HTTP 201 routed to Codex => PASS (direct escape hatch).
- TEST 3 Kanban->Main Brain->CLI->OmniRoute->terminal: PIPELINE PROVEN TO LAUNCH — daemon claims+picks task (task wakeup received -> task received -> picked task agent=Kiro-TL provider=kiro) then FAILS CLOSED: "credential isolation required for provider \"kiro\" but no account assignment exists". 
  - BLOCKER (concrete, owner input): native provider account assignment = account/credential management = OmniRoute-owned + SUPERSEDED per prior owner directives (consume-only). Options: (A) owner/OmniRoute provisions a provider account assignment; OR (B) route the agent via the OmniRoute gateway path (e.g., Cline OpenAI-compatible) instead of a native-account provider, which additionally needs the previously-identified RouteModel availability (BLK-AVAIL GLM / BLK-KIMI / BLK-OPUS48-ID) resolved by OmniRoute. Kiro will NOT create account assignments (boundary). No inference reached OmniRoute (fail-closed before launch); zero cost.

## Path B (OmniRoute CLI adapter) attempted per Principal 18:30 — same gate
- Reconfigured daemon to GATEWAY MODE: AGENT_BRAIN_CLI_KIND=claude-code + AGENT_BRAIN_ROUTE_MODEL=auto/claude-sonnet (consumed OmniRoute /v1/models via gateway secret-file ref; 269 models declared ready incl auto/claude-*, auto/coding, auto/glm; secret value never printed). Daemon authenticated+ready.
- Rerun (Kiro-TL, provider=kiro): FAILED "credential isolation required for provider \"kiro\" but no account assignment exists".
- Moved agent to OmniRoute-compatible CLINE runtime (Acceptance-Cline on runtime 9c9df48a): rerun FAILED identically "credential isolation required for provider \"cline\" but no account assignment exists".
- ROOT CAUSE (code-verified): execenv.go CredentialEnv resolves provider-native credential isolation from an agent->account assignment; taken when task.Request.GatewayRequired is false (admission.go:64). The daemon-side AGENT_BRAIN_* gateway config does NOT change the BACKEND-created task's gateway_required flag (rerun API), so every task takes the native-credential path requiring an account assignment for ALL providers (kiro AND cline).
- BLOCKER (concrete): to run consume-only, tasks must be gateway_required=true so the gateway secret satisfies credential isolation instead of a per-provider account. That derives from backend/runtime router-owner=omniroute config, which is not exposed via the available APIs and is not flipped by daemon env. Creating account assignments is FORBIDDEN (OmniRoute-owned/superseded). OmniRoute readiness is PRESENT (not the blocker).
- Requires owner/architecture action: configure the deployed backend/runtime so agent tasks are omniroute-routed (gateway_required=true), OR a deploy/code change wiring the gateway-secret credential path for these runtimes. Zero inference reached OmniRoute (fail-closed pre-launch); zero cost. Boundaries preserved (no account/credential creation, no OmniRoute-internal, data/secrets intact).

## Platform-defect remediation (Principal 21:49) + resolved layers
Authoritative repo already committed 880338b "wire production FileCredentialSource for gateway-required development mode"; orq1 deployed checkout (9a42a29) LACKED it. Built the authoritative artifact on orq2 (go1.26.1) and deployed /tmp/multica-auth to orq1 (no ad-hoc orq1 source patch). Resolved layers in sequence, each with daemon-log evidence:
1. daemon not running -> built + atd-launched (fixed self-pkill bug).
2. not authenticated -> mul_ PAT via POST /api/tokens(201) + `multica --server-url login --token` (secret never printed).
3. rotation store unavailable -> DATABASE_URL from existing dev.env creds.
4. native provider account assignment (kiro AND cline) -> enabled Agent-Brain gateway slice: AGENT_BRAIN_DEVELOPMENT_ENABLED=1 + --agent-brain-gateway-required + CLI_KIND + ROUTE_MODEL (moves off native providers).
5. claude-code CLI absent -> switched CLI_KIND=codex (installed codex-cli 0.144.6) + ROUTE_MODEL=auto/coding.
6. credential_source_unavailable -> deployed authoritative artifact wiring FileCredentialSource (the actual platform defect vs OpenSpec 2.1). NOW WIRED.
7. gateway_unavailable (CURRENT): daemon env has correct AGENT_BRAIN_GATEWAY_BASE_URL=http://100.118.244.61:20128; /api/health/ping=200 and /v1/models=200 from orq1 with the secret; but StrictReadinessPolicy requires per-model SelectedModelReady+SelectedProtocolReady+ModelRegistryReady. OmniRoute /v1/models declares model IDs only; OMNIROUTE_DEV_MODELS_COMPAT=1 projection fills IDs only (no protocol/ready flags) -> strict readiness cannot be satisfied -> fail-closed gateway_unavailable.
BLOCKER = concrete EXTERNAL OmniRoute readiness declaration absent (enriched per-model protocol/ready metadata) = the P0 BLK-AVAIL, and the exact stop condition Principal named. Owner/OmniRoute action: publish enriched /v1/models readiness rows (protocol + ready per selected model), OR relax StrictReadinessPolicy for dev (authoritative code change) to accept ID-only availability. Zero inference reached a model (fail-closed at readiness). Boundaries preserved: consume-only, no account/credential creation, secret-file value never printed.

## Option A executed — Claude Code installed + consume-only gateway admission PROVEN
- INSTALL RECEIPT: @anthropic-ai/claude-code@2.1.218 (official npm, pinned), binary /home/ec2-user/.nvm/versions/node/v22.23.1/bin/claude, `claude --version`=2.1.218 (Claude Code). No secrets.
- CONFIG: daemon gateway mode CLI_KIND=claude-code + ROUTE_MODEL=claude_code_kimi_2.7_Code (sole approved dev-compat route, Anthropic Messages); authoritative /tmp/multica-auth (credential source wired); agents=[claude]; runtime 588ebcba registered; health 200.
- PROVEN: agent-brain admission span (23:40:13.113) admission_decision=ADMITTED readiness_result=READY cli_kind=claude-code route_model=claude_code_kimi_2.7_Code fail_closed_class=none latency_ms=411 secrets_present=false. => Strict readiness PASSED end-to-end via OmniRoute consume-only. Architecture+fix+config all validated.
- REMAINING (two precisely-bounded issues):
  1. OmniRoute /v1/models readiness INTERMITTENCY: only the first per-admission readiness fetch returned ready; all subsequent fetches (371-548ms real round-trips, then 3-6ms circuit-breaker fast-fails) returned readiness_result=unavailable -> gateway_unavailable. Pattern = OmniRoute-side /v1/models availability/throttle after first fetch (EXTERNAL; forbidden to inspect/modify). Persists after 150s cooldown.
  2. TASK-START lifecycle race: the one ADMITTED task (28de22fa) failed at start: POST /api/daemon/tasks/<id>/start -> 400 "start task: no rows in result set" (task record superseded by overlapping rerun/reassign under max_concurrent_tasks=1). Backend task-lifecycle defect on concurrent rerun.
- NON-CLAIM: no task reached CLI/OmniRoute inference; zero inference; fail-closed throughout. Boundaries preserved (no OmniRoute-internal, no native accounts, secret never printed, install pinned/official per owner).

## Readiness-resilience fix — live acceptance PROVEN (2026-07-23T02:23Z)
Fix: Main-Brain-side resilient admission (single-flight admitMu + bounded exp backoff+jitter + Retry-After honoring + transient-only retry + bounded fresh-wait) in brain_integration.go admitWithReadinessResilience/admitRetryLoop/transientReadinessRetry. Never reuses stale ready, never weakens StrictReadinessPolicy, no OmniRoute-internal. Tests: TestAdmitRetryLoop_TransientThenFreshReady, _PersistentFailsClosed, _DeterministicRejectionImmediate, TestTransientReadinessRetry_Classification — ALL PASS. Built /tmp/multica-auth-fixed, deployed orq1 (pid 818615, AGENT_BRAIN_READINESS_ADMISSION_WAIT_MS=60000).
LIVE RUN: ORQ-7 rerun task-run d2264e2f -> admission_decision=admitted readiness_result=ready cli_kind=claude-code route_model=claude_code_kimi_2.7_Code fail_closed_class=none latency_ms=398 -> launch router_owner=omniroute -> completed 02:23:05Z -> persisted result "current working directory is /home/ec2-user/mul...". Full path squad->Kanban->Main Brain->CLI->OmniRoute(consume-only)->persisted terminal/status VERIFIED.
