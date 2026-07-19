# STATE — Agent Brain v3 (estado vivo; re-ler após qualquer reinício)

updated: 2026-07-19 (Wave A planning/governance freeze authored in isolated worktree `planning/agent-brain-observability-freeze`; D-V3-16/17/18) · planning_owner: Kiro/Opus-4.8 · operational_co_lead: Codex#56#A · milestone: Agent Brain v3
authorization_gate: OMNIROUTE_ARCHITECT_RESPONSE.md §7.1 = `AUTORIZADO` → Waves 0–3/tier 20 autorizados
current_phase: G4 IN_PROGRESS (**51/96** after evidence-backed closure of governance tasks 0.2–0.6 — 0.2 via active owner decision D-V3-13 — plus **task 8.8 documentary closure (EV-G4-08; provenance reconciled 2026-07-18T13:41:33Z)**; **task 8.2 REOPENED by independent adjudication** (5-vendor accepted-path verification not met — Claude/Codex synthetic-only, Kimi/GLM-NVIDIA/Antigravity fail-closed, no live); 0.1/0.7 OPEN; G3 ACCEPTED). Accepted: I1/I2 + I3–I5 contracts; GATEWAY CORE RUNTIME; aggregate `SteadyStateFacts`; gateway I3/I4/I5 producers; cleanup report-only/no-delete (pB + Gemini); pD L2 + AccountHome isolation (formal pB ACCEPT); Grok A–D physical-profile isolation (formal Kiro ACCEPT); MCP protected-agent command guard (independent ACCEPT; mitigates R23); **ExactEnv physical containment (ACCEPT — residual hostile same-UID TOCTOU requires OS-level isolation; R22 MITIGATED, not open)**; **gateway RouteModel projection warm-cache pre-cancel/deadline cancellation (ACCEPT)**; **p9 runtime discovery + Windows fail-closed hardening (ACCEPT — R24/R25 closed)**; **RSS Linux lifecycle fix (ACCEPT; RSS collector-set pinned in EV-G4-08)**; **R26 route preservation (independent `REVIEW-R26-ACCEPT`; exact slash RouteModel preserved only across compatible CLI mappings, incompatible changes fail closed or atomically reselect)**. Residual ExactEnv same-UID TOCTOU remains tracked for OS-level isolation. **Task 9.1 remains STOPPED/not ready despite the accepted RSS Linux lifecycle fix** (thresholds PENDING owner; independent pB acceptance still required). **pD pane QUOTA-BLOCKED — no reset redemption without owner authorization.** 9.1 STOPPED; 8.1 OPEN; 0.1/0.7 OPEN; all credential/network/live STOPs; PD-01/PD-08 preserved. Count 51/96; architecture unchanged; no invented hashes/evidence.
herdr_context: workspace `w3`; Kiro/Opus-4.8 planning/adjudication lead=`w3:p3` (`Kiro#Opus48-TL`); Codex#56#A operational co-lead/transport/verification=`w3:p1`; workers: Codex1/G3 integrator=`w3:pD`, Codex2 gateway=`w3:p8`, Codex3 runtime=`w3:p9`, Codex4 ops=`w3:pA`, reserve=`w3:pB`. Claude pane `w3:p5` was closed by the owner on 2026-07-18 and is not a dependency. Herdr is directly available from `w3:p1`; always use explicit pane IDs/`--current`, never bare `herdr`, and never close panes/server without owner direction.

## Estado por fase

| Fase | Estado | Último fato verificado | Blocker | Próxima ação autorizada |
|---|---|---|---|---|
| G0 | COMPLETE | GSD v3 criado; 85 tasks auditadas; 44 IDs de paridade mapeados; PD-01 resolvida por preservação auditável da worktree | — | — |
| G1 | COMPLETE | EV-G1-01..05 plus model matrix, adapter prep and ops prep accepted for documentary/freeze scope | Protocol/security/capacity execution acceptance remains later; PD-08 STOP remains | — |
| G2 | COMPLETE (2026-07-18) | G2A EV-G2A-01..05; G2B EV-G2B-01..07; G2C EV-G2C-01..10 with 5.6–5.8 intentionally fail-closed; G2D EV-G2D-01..07 | Native credential-bearing adapters still require future acceptance; PD-08 STOP; PD-01 preserved | — |
| G3 | ACCEPTED (restored 2026-07-18) | REVIEW-G3-02 = ACCEPT: F1 argv route/config override, F2 custom-runtime credential exposure, F3 argv-log disclosure all closed with implementation anchors + passing focused/race regression on the final launch path (`internal/daemon`, `pkg/agent`); corrections in `evidence/g3-security-corrections{,-adapters}.md` | — (security-correction hold lifted) | — |
| G4 | IN_PROGRESS | Accepted: I1/I2 + I3–I5 contracts; GATEWAY CORE RUNTIME; aggregate `SteadyStateFacts`; gateway I3/I4/I5 producers; cleanup no-delete (pB+Gemini); pD L2+AccountHome (pB ACCEPT); Grok A–D (Kiro ACCEPT); MCP guard (ACCEPT; mitigates R23); **ExactEnv physical containment (ACCEPT; residual same-UID TOCTOU needs OS isolation; R22 MITIGATED)**; **gateway projection warm-cache cancellation (ACCEPT)**; **p9 runtime discovery + Windows fail-closed (ACCEPT; R24/R25 closed)**; **RSS Linux lifecycle fix (ACCEPT; EV-G4-08 pin)**; **R26 route preservation (`REVIEW-R26-ACCEPT`)**. pD pane QUOTA-BLOCKED (no reset without owner auth). **51/96** (0.2–0.6 + 8.8 documentary closed; **8.2 REOPENED**; 0.1/0.7 OPEN; 8.1 OPEN; 9.1 STOPPED) | OS-level isolation for residual ExactEnv same-UID TOCTOU; 9.1 thresholds PENDING owner + pB acceptance (STOPPED); all credential/live STOPs | 9.1 readiness (gated; STOPPED until thresholds + pB) → provenance finalize |
| G4-OBS | AUTHORIZED (gate, not started) | Stop-gate bloqueante D-V3-17; OBS-1..OBS-11 novos/OPEN; nenhuma evidência ainda | precede capacidade/cutover; depende de G4 8.x | Wave B dispatch (W5/W6/W7 + W1/W2/W3/W4) |
| G5 | NOT_STARTED | — | G4 + G4-OBS PASS | — || G6 | NOT_STARTED | — | G5 + autorização cutover/removal | — |
| G7 | NOT_STARTED | — | G6 + relatório de capacidade | — |
| G8 | NOT_STARTED | — | G7 | — |

## Último fato verificado (evidence-backed, read-only)

- **Wave A planning/governance freeze (2026-07-19, isolated worktree `planning/agent-brain-observability-freeze` off recovery SHA `da42282`):** owner decisions D-V3-16 (Prodex retido como cold platform recovery mode default-OFF, mutuamente exclusivo, operator-gated — NÃO deletado; nunca per-request/automático; nunca hot simultâneo com OmniRoute), D-V3-17 (G4-OBS stop-gate bloqueante antes de capacidade/cutover), D-V3-18 (topologia 8 lanes com prova de zero-overlap). Alterações **planning/spec/docs somente** — nenhum código de produto, credencial ou restart de agente. OpenSpec: `build-omniroute-agent-brain` (proposal/design/tasks/3 specs + nova capability `end-to-end-observability`, tasks OBS-1..OBS-11, task 10.4 emendada para retain-as-recovery) e `persist-prodex-runtime-integration` (re-escopo cold-recovery-only, `MULTICA_PRODEX_REQUIRED` default 0) — ambos `openspec validate --strict` = VALID. GSD: DECISIONS/ROADMAP(+G4-OBS)/REQUIREMENTS(AB-REQ-39/40/41; AB-REQ-37 retention)/RISKS(R27-30)/FILE_OWNERSHIP(8 lanes)/REMOVAL_REGISTER(Prodex RETAIN-AS-RECOVERY)/EVIDENCE_INDEX(EV-OBS-01..11)/TRACEABILITY/COMPONENT/INTERFACE/DISPATCH atualizados. **Contagem atualizada (correção de Codex56-Principal-TL, 2026-07-19): build-omniroute agora 51/96** — as 11 novas tasks OBS-1..OBS-11 (D-V3-17) elevam o total de 85 para **96**; concluídas permanecem **51** (nenhuma OBS marcada; todas OPEN); reopened statuses (8.2, native 1.5/1.6) preservados; 0.1/0.7/8.1 OPEN; 9.1 STOPPED; persist 0/16; PD-01/PD-08 preservados. Nenhum mandato ativo de delete/removal de Prodex permanece.

- OpenSpec `build-omniroute-agent-brain` completo (proposal, design, architecture, 5 specs,
  tasks.md com **85** tarefas, MASTER_PLANNING, handover, profile, requirements handoff,
  architect response, acceptance checklist, parity matrix). Lido na íntegra.
- `tasks.md` = **85 tarefas**; o MASTER_PLANNING foi corrigido durante o handover.
- Current count = **51/96**: governance tasks 0.2–0.6 (0.3–0.6 = GSD docs + registers on disk; 0.2 via active owner decision D-V3-13, not superseded D-V3-09) + **task 8.8 documentary closure** (EV-G4-08; provenance reconciled 2026-07-18T13:41:33Z); **task 8.2 REOPENED** by independent adjudication (Claude/Codex synthetic-only; Kimi/GLM-NVIDIA/Antigravity fail-closed; no live E2E); **0.1/0.7 remain OPEN**, **8.1 OPEN**, **9.1 STOPPED**; earlier 46/50/51/52 superseded. Tasks 5.6–5.8 intentionally remain open.
- Sibling OpenSpec changes (disk-recount 2026-07-18): **chat-orchestration-standard 4/10** — task **1.1** (TL/Manager identity + `## Squad Operating Protocol`; clarify→openspec→plan→delegate→synthesize) and task **1.4** both independently ACCEPTED — evidence executed honestly: daemon package tests **genuinely executed**; handler package **compiled only** (its TestMain is DB-gated, so handler-layer tests did not run). **agent-credential-isolation 4/21** — task **4.1** (detect exhausted/`expired` session via discovery status + `expires_at`) independently ACCEPTED — focused **×20 / race / vet / gofmt / diff**. **native-runtimes-onboarding 9/17** — offline 1.1–1.4 + 2.1–2.3 pass (1.4 KEPT: discovery/timeout/cache/daemon-report validated; its "UI popula" clause is implementation-only, E2E-unverified); **tasks 1.5 & 1.6 REOPENED** by independent adjudication (frontend onboarding / design-parity+i18n+web-build+QA — no evidence artifact; web/UI/build/QA verification lane unperformed); **task 1.7 (auth BACKEND) ACCEPTED & CHECKED** (EV-AUTH-1.7, artifact sha256 `2a5f7368…c095`: `/auth/login` + bcrypt/argon2 credential store behind `AuthProvider`; `/auth/send-code`+`/auth/verify-code` removed; `/auth/google`+`/auth/logout` kept; reviewer ACCEPT reproduced — 17-file hashes, focused tests+race+vet, JWT fail-closed startup probe) — residuals tracked as follow-ons (CLI first-password bootstrap, mobile legacy-endpoint migration, distributed limiter, token revocation). build-omniroute-agent-brain now **51/96** (8.2 reopened). No production/cutover claim.
- **persist-prodex-runtime-integration = PROGRAM HOLD / OWNER DECISION REQUIRED** (2026-07-18; audit `persist-prodex-vs-omniroute-reconciliation-audit.md` sha `e1a65416…7504`): conflicts with OmniRoute **0.5** (blocks concurrent superseded-Prodex execution) and **7.8/AB-REQ-37** (Prodex default-off/drain/removal) vs persist 1.3/2.x making Prodex required+restart-durable. **NOT rejected; no checkbox changed (0/16 preserved).** All persist product/test implementation **FROZEN**; persist code **EXCLUDED from push scope**; audits/designs preserved. Owner options A (sanctioned transitional+sunset) / B (defer/decline) / C (minimal reversible continuity) — **pending owner; co-lead selects none**.
- **Push-scope (2026-07-18):** **no current atomic group is push-ready.** G1 **FROZEN** (backup-snapshot QA `26027e5d…` rejects the "ready-to-commit" claim; snapshot-recovery itself PASS). All **11 staged Packet B/frontend files EXCLUDED** (ownership review `1a4d58dd…`: 7 PENDING, `client.ts` PENDING+dual-lane conflict, `types/agent.*` UNKNOWN/UNOWNED; no accepted task owns the picker UI — 1.5/1.6 `[ ]` BLOCKED). Persist groups excluded under PROGRAM HOLD. **Staging ≠ acceptance; git index unmutated.**
- **chat READY-1 atom = TECHNICAL ACCEPT / push NOT authorized** (2026-07-18; clean-room review `e8d1d1ce…`, reviewer Codex56#B distinct): exact 3-file atom (`daemon/prompt_test.go`+`handler/squad_briefing.go`+`_test.go`, manifest `f7d7a2ef…`) — hashes match disk; daemon 100/100 (TL 300/300), AST 24/24, race clean; handler compile/vet-only (DB-gated TestMain, zero runtime assertions). **Push blocked pending w8:p1 governance/provenance reconciliation; no checkbox change.** Covers only READY-1, not Packet B (still excluded) or persist (HOLD).
- **native 1.6 acceptance gates (2026-07-18; diagnostic `e59c3a29…`, task OPEN):** (A) offline cold build requires **vendored OFL `.woff2` + `next/font/local`** preserving CSS variable names + `OFL.txt` licensing + CLS gate; (B) **jsdom full acceptance must run on a Linux-native filesystem via the existing harness** — `/mnt/c` (9p/DrvFs) pool/timeout tuning is NOT an acceptance fix. **Residual to clean:** untracked `apps/web/__tmp_callback_vitest.config.ts` must be removed before any 1.6 acceptance. native stays 9/17.
- **cred 5.4 = OPEN / PARTIAL** (2026-07-18; critique `84702b60…` vs audit `2b060da6…`): original PASS **overstates coverage** (non-exhaustive 703-slog proof → PARTIAL) and **raw Claude stderr redaction is pattern-dependent**; new minimal `claude.go` hardening **in progress, needs distinct review**. **Verified positives preserved:** 703 matches/82 files exact; all production `slog` handlers wired (global hook; 2 `slog.New` sites, zero unwired); OAuth body covered pattern-dependently; agent-output sinks sanitized. Post-fix gate = structural Claude-stderr redaction + distinct review + honest PARTIAL coverage grade + contract completeness. Email slice already ACCEPTED (`EV-CREDISO-5.4-EMAIL`).
  - **5.4 Claude-stderr hunk = TECH PASS** (2026-07-18; producer `69f57eaa…`, review `94d1eb5d…`, reviewer Kiro/Opus-4.8 w8:p2 **truthfully** distinct from producer w7:p1/adjudicator w3:p3): `logWriter → redact.Text` hunk PASS (TestLogWriter ×20 = 120/120, race 0, full pkg ok). **But 5.4 stays OPEN + push eligibility PARTIAL** — `claude.go` co-mingles **unrelated argv/environment hunks**; **no whole-file/whole-task acceptance**. Queue: clean-room isolated-patch evidence + residual absolute-risk audit before final 5.4 decision.
- **cred 4.3 = DESIGN REJECT (as implementation-ready) → OPEN** (2026-07-18; review `e740f715…`): missing production producer/emitter is real (ACCEPT diagnosis), but 5 owner-gate correctness boundaries block implementation — transport routing (`daemonws` relay/protocol), router-owner gating + task/runtime ownership identity, cross-process serialization (process-local locks; PG last-writer-wins; WS multi-client), **atomic `Assign`+`RecordRotation`** (Assign precedes record; durable on record-fail), and **destructive logout rollback** (credential paths deleted pre-auth/pre-commit, no restore). Reviewer independence = **truthful distinct session/pane provenance, not model-family name**. 4.4 clean path = corrected w8:p2 record + fresh assigned Codex review.
- **native 1.5 = OPEN** (2026-07-18; review `40b4cb7a…`): positives verified — 14 manifest hashes reproduce, type/static contract pass, AuthService/api.login/CLI-callback/desktop-handoff seams correct, **43 real assertions** close the producer's zero-assertion gap. **Blocked → OPEN:** (1) **provenance mismatch** — artifact reviewer "opencode" vs queue "GLM52#B" (identity not truthfully established); (2) **`callback/page.test.tsx` unexecuted** (jsdom worker-startup on `/mnt/c`). Build/e2e/UAT non-claimed (not borrowed). Remediation = truthful distinct reviewer identity + Linux-native jsdom execution of the callback test. native stays 9/17.
- **native 1.7 backend atom = TECH PASS(bounded) / TASK QUALIFIED / ATOMIC PUSH HOLD** (2026-07-18; review `1bc6ca43…`, eligibility `1937b6f…` Codex56#A HOLD): 5 packages green+race offline (reproduced); `auth.go` wholly 1.7; 16 non-env hashes match. **Push HELD:** `auth_routes_test.go` essential-but-unpinned (outside 17-file manifest), unnamed producer/reviewer attribution, `.env.example` bytes unread, CLI bootstrap incomplete, topology tests excluded, DB-gated handler runtime. **1.7 `[x]` historical/qualified — NOT push authorization; not unchecked.** Remediation: name producer+distinct reviewer (or waiver) + pin auth_routes_test.go & re-hash + env-permitted `.env.example` reconciliation + Kiro TL auth + root re-hash.
- Matriz de paridade = **44 IDs** (P01–P34 + SC01–SC10).
- Seção 7.1 = `AUTORIZADO` para Waves 0–3/tier 20; 7.4 permanece gate de evidência para cutover; 7.5 assinaturas formais continuam pendentes.
- Worktree não-commitada: `daemon.go/config.go/health.go/l2_runtime.go/prodex.go` modificados
  + novos `prodex_fs_linux.go/prodex_fs_other.go/prodex_profiles.go` + 2 CHECKIN_* deletados
  + 2 CHECKIN_* novos. Natureza do diff = change `persist-prodex-runtime-integration`
  preservado como baseline auditável pela decisão PD-01. O módulo neutro `brain` agora
  existe como contrato congelado e não está ligado ao daemon ativo; `gateway/runtimeenv/cli`
  permanecem para as próximas waves.
- GSD legado `.planning/` (RPP/Prodex v2.1) lido e preservado; decidido SUPERSEDED como plano ativo.
- Reconciliation Kiro/Opus-4.8 + Codex#56#A (2026-07-18): all four G2 panes are idle and
  their final outputs match the latest DONE ledger rows. The earlier Claude statement that
  no G2 worker had finished is superseded. `git diff --check` passed. This shell lacks `go`,
  so no new local Go rerun is claimed; test results remain attributed to the worker panes.
- Owner decision (2026-07-18): this system is not in production. Production canary/soak is
  removed from the immediate gate; equivalent development integration, security, failure,
  rollback and bounded-capacity validation remains mandatory before enabling broader paths.
- Antigravity CLI 1.1.4 IPv6-resolver eligibility failure RESOLVED (2026-07-18): external CLI
  regression (not repo); per-process `GODEBUG=netdns=cgo` workaround confirmed (pF Sonnet 4.6
  working); optional host IPv6 disable applied by owner; no auth/token/TLS/login/eligibility
  change. Evidence: `evidence/antigravity-1.1.4-ipv4-resolver.md`.
- Temporary Kanban bridge (2026-07-18, recorded, NOT dispatched): the active Multica daemon still
  discovers native CLI catalogs and merges inherited env. Safe temporary target = **gateway registry
  projection + credentialless Claude/Codex launch only, fail-closed for unsupported adapters, NO DB
  migration**. Consistent with D-V3-15 (Kanban parked; no Multica-daemon Codex dispatch before
  credentialless wiring + isolation smoke).
- R26 independently ACCEPTED (`REVIEW-R26-ACCEPT`, 2026-07-18): handler review confirmed exact
  slash RouteModel preservation only for compatible runtime/CLI mappings, fail-closed incompatible
  changes or atomic exact slash replacement, empty/native downgrade rejection, empty-prior handling,
  and runtime/model compare-and-swap. Offline Go 1.26.4: pure route test ×20, brain package ×20,
  pure route race, vet, gofmt and diff checks PASS. DB-backed runtime-switch/concurrency tests compiled under
  normal/race builds but did not execute because the existing localhost PostgreSQL rejected the
  default test login; no credentials or DB/Docker state were used. Count/gates unchanged: **51/96**;
  0.1/0.7 and 8.1 OPEN; 9.1 STOPPED; PD-01/PD-08 preserved.

## Blockers ativos

- **STOP (adjudicação de monitor, 2026-07-17):** mantém STOP em mutações de credencial / qualquer
  dispatch que toque auth. A credencial Windows legada `C:\Users\dataops-lab\.codex\auth.json`
  (criada 2026-07-15T17:12:59Z, ACL `CodexSandboxUsers` ReadAndExecute) é exposição real pré-existente
  (não overwrite, não falso-positivo). TL NÃO lê/move/apaga/reescreve. **PD-08** (pendente do dono)
  exige autorização para restringir/quarentena/remover + rotacionar a conta. Tasks de doc/contrato
  sem-secret podem terminar (já terminaram p/ Codex1/2/3). Monitor Linux de isolamento = OK.
- **PD-08 gate semantics (clarification — does NOT weaken PD-08):** PD-08 has two facets.
  (a) It is an **absolute STOP invariant** (no credential/auth/secret read/copy/rewrite/rotation/
  quarantine/mutation). PD-08 does **not categorically prohibit** a *contained, offline, synthetic*
  task-9.1 attempt; however, PD-08 compliance for such an attempt is **accepted only after every
  containment, prerequisite, named-evidence-owner authorization and STOP condition in the acceptance
  checklist has been verified** — not by construction. **Task 9.1 is presently STOPPED / not ready.**
  (b) As a **pending owner remediation** of the specific legacy `.codex/auth.json` exposure
  (restrict/quarantine/remove + rotate), it **remains mandatory before any live-auth or cutover path**.
  **8.1 vs 9.1:** task **8.1** (authenticated models/capabilities) is **OPEN / live** — it needs a real
  credential path and is blocked by PD-08's invariant + no-live-provider until separately authorized;
  task **9.1** (offline synthetic 20-task profile) is scoped to development validation (D-V3-14) and is
  **subject to every prerequisite and STOP condition in the acceptance checklist**, with thresholds
  PENDING owner and independent pB acceptance still required. PD-08 remains fully in force; no-credential
  access is absolute; count is **51/96** (0.1/0.7 OPEN, 8.1 OPEN, 9.1 STOPPED).
- **Orchestration blocker RESOLVED:** Herdr is available directly to Codex#56#A inside
  `HERDR_ENV=1`; live `pane list/get/read` reconciled G2. The former MSYS/WSL bridge limitation
  belonged to the retired Claude pane and is historical only.
- **G1 documentary evidence:** Codex1 EV-G1-01..05 is complete and Codex2-4 inputs are present.
  This does not claim live protocol, security, failure or capacity acceptance.
- A worktree permanece não-commitada, mas não é mais órfã: PD-01 determinou preservação,
  auditoria e ownership exclusivo por Codex1. Verificado: diff `persist-prodex` intacto e
  preservado (config/daemon/health/l2_runtime/prodex + prodex_fs_*/prodex_profiles). Há também
  mods em `docs/operations/RUNBOOK_ISOLAMENTO_CREDENCIAL_PANES.md` e `scripts/ops/agent-cred-isolation{.sh,-harness.sh}` — fora do escopo de freeze do Codex1 (TL nao tocou; signal ao dono).
- **Evidence granularity:** G2C/G2D implementation evidence exists as code, tests, ledger and
  index entries. Narrative summaries were added during handover for parity with G2A/G2B;
  this does not upgrade native adapters 5.6–5.8 beyond fail-closed contracts.
- Os portões da seção 7.4, assinaturas 7.5, a quiescência/ativação do cold recovery mode do Prodex (retido, não deletado — D-V3-16), cutover default e tiers 50/100
  permanecem bloqueados até suas evidências/autorizações específicas.

## Próxima ação autorizada

- G2 is closed. G3 is RESTORED ACCEPTED (REVIEW-G3-02 = ACCEPT); the security-correction hold is lifted.
- G4 remains IN_PROGRESS. The earlier `SteadyStateFacts`, aggregate, gateway-producer, runtime-catalog,
  cleanup and provenance lanes are accepted/reconciled and are no longer current actions. **Current
  sibling lanes:** native-runtimes-onboarding task **1.7 auth review**; agent-credential-isolation tasks
  **4.3** (automatic reassignment), **5.4** (log sanitization evidence) and **5.1** (frontend/backend
  build and tests); and chat-orchestration-standard task **0.1 owner decision** on the default TL/Manager
  squad name/structure. These tasks remain OPEN; no credential/live action or closure is implied.
  **GATED (not now):** 9.1 readiness (owner decisions A–F); SC01–SC10 / reset-redeem (owner/G5).
  Preserve **51/96** unless a further evidence-backed closure exists (0.2–0.6 + 8.8 closed; 8.2/1.5/1.6 reopened;
  0.1/0.7/8.1 OPEN; 9.1 STOPPED).
- Preserve Codex1 hotspot ownership; no agent may edit daemon/config/health/cmd/go.mod/execenv/models.go
  except Codex1.
- Maintain STOP on credential/auth mutations until PD-08 is resolved by the owner.
- Do not dispatch Codex through the current Multica daemon until end-to-end credentialless
  dispatch is accepted beyond the synthetic G3 smoke. Existing Kanban MUL-2..MUL-25 remains
  parked; reconcile MUL-11/12/15 with OmniRoute ownership before any bulk dispatch.
- Production admission/cutover, Prodex removal and tiers 50/100 remain blocked by their specific gates.

## Regra de reporte

Todo reporte ao dono contém: phase · task IDs · agents ativos · owners/files locked ·
progresso real · evidence IDs · decisões · blockers · riscos novos · ETA atualizado ·
próxima ação que exige autorização.
