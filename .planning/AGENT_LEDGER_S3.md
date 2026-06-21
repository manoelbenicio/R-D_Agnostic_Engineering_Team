# AGENT CHECK-IN / CHECK-OUT LEDGER — Sprint 3 (Pre-Prod)

> **Orquestrador (Ponto Focal):** ANTIGRAVITY — toda decisão passa por aqui primeiro
> **Projeto:** AgentVerse v1
> **Sprint:** 3 — Pré-Produção | GO Core Migration
> **Inicio:** 2026-06-21T08:20:00Z
> **Regra:** Todo agente DEVE registrar CHECK-IN antes de tocar qualquer arquivo
>            e CHECK-OUT ao finalizar. Sem exceções.
> **REGRA GIT:** NENHUM agente faz git add/commit. O orquestrador consolida.

---

## REGRAS

1. **CHECK-IN**: Antes de editar qualquer arquivo, adicione linha com Action=CHECK-IN e Status=IN PROGRESS
2. **CHECK-OUT**: Após completar TODAS as edições, adicione linha com Action=CHECK-OUT e Status=DONE
3. **BLOCKED**: Se o arquivo já está CHECK-IN por outro agente, adicione BLOCKED e PARE — não edite
4. **FAILED**: Se suas edições causaram falha em TypeScript ou teste, adicione FAILED com detalhes do erro
5. **Formato de agentID:** orquestrador_opus46, nemotron_ultra, codex1, codex2, codex3, codex4, codex5, nemotron_ultra2
6. **Timestamps:** Formato ISO-8601 UTC obrigatório
7. **Uma linha por arquivo por ação** — 2 arquivos = 2 linhas de CHECK-IN
8. **NÃO FAZER GIT ADD/COMMIT** — o orquestrador consolida todos os commits

---

## LEDGER

| Timestamp (UTC) | AgentID | Action | Arquivo(s) | Status | Notas |
|-----------------|---------|--------|-----------|--------|-------|
| 2026-06-21T08:20:00Z | orquestrador_opus46 | CHECK-IN | .planning/AGENT_LEDGER_S3.md | DONE | Ledger Sprint 3 criado |
| 2026-06-21T08:45:00Z | orquestrador_opus46 | CHECK-OUT | src/api/types.ts, index.ts, connect-terminal-socket.ts, session-discovery.ts, settings-store.ts, vite-env.d.ts, migrations.ts, .env.local, .env.example | DONE | CRIT-001 + CRIT-003 parcial. Criou go-core-base-url.ts e go-core-client.ts |
| 2026-06-21T08:45:00Z | nemotron_ultra | CHECK-OUT | src/shared/storage/migrations.ts | DONE | Fix .then() → onsuccess callback em v4 migration; tsc 0 errors |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-IN | src/api/session-discovery.ts, health-store.ts, use-installed-cli-providers.ts | IN PROGRESS | CRIT-003 sweep |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/api/session-discovery.ts, health-store.ts, use-installed-cli-providers.ts | DONE | Already using goCoreClient from @/api |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-IN | src/canvas-reconciler/reconciler.ts, health/FirstRunWizard.tsx, health/HealthPage.tsx, shell/canvas-command-adapter.ts, voice/command-executor.ts, voice/VoicePanel.tsx | IN PROGRESS | CRIT-003 sweep — 5 arquivos prod com cao-client |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/canvas-reconciler/reconciler.ts | DONE | Already fixed: imports from @/api with GoCoreClient/caoClient aliases |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/health/FirstRunWizard.tsx | DONE | Already fixed: imports from @/api |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/health/HealthPage.tsx | DONE | Already fixed: imports from @/api |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/shell/canvas-command-adapter.ts | DONE | Already fixed: imports from @/api |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/voice/command-executor.ts | DONE | Already fixed: imports from @/api |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/voice/VoicePanel.tsx | DONE | Already fixed: imports from @/api |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-IN | package.json, .eslintrc.cjs, eslint-rules/index.cjs | IN PROGRESS | CRIT-002 ESLint plugin fix |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/api/session-store.ts | DONE | Uses discoverSessions from session-discovery (no direct caoClient) |
| 2026-06-21T08:57:00Z | nemotron_ultra | CHECK-OUT | src/shell/app-fetch.ts | DONE | Uses auth, no direct caoClient/base-url |
| 2026-06-21T09:00:00Z | codex4 | CHECK-OUT | .planning/QA_REPORT_SPRINT3.md | DONE | OP1 REVIEW DONE — 4 revisões arquiteturais documentadas |
| 2026-06-21T09:00:00Z | codex2 | CHECK-OUT | tests/e2e/smoke.spec.ts | DONE | Removidas refs CAO/:9889; substituído por GO_CORE_BASE_URL |
| 2026-06-21T09:00:00Z | codex2 | CHECK-OUT | tests/e2e/canvas-deploy.spec.ts | DONE | Criado 3 testes RCA-A2/HIGH-003. Falhas esperadas por data-testids ausentes |
| 2026-06-21T09:00:00Z | codex2 | CHECK-OUT | playwright.config.ts | DONE | webServer env atualizado para VITE_GO_CORE_BASE_URL:8080 |
| 2026-06-21T09:21:00Z | codex3 | CHECK-OUT | infra/runtime/cloudbuild.yaml | DONE | GO Core substitutions + contract-tests gated |
| 2026-06-21T09:21:00Z | codex3 | CHECK-OUT | .github/workflows/ci.yml | DONE | Renomeado GO Core; contract-tests skip-gated |
| 2026-06-21T09:21:00Z | codex3 | CHECK-OUT | .github/workflows/contract-nightly.yml | DONE | Nightly contract migrado GO Core |
| 2026-06-21T09:21:00Z | codex3 | CHECK-OUT | .github/workflows/smoke-live.yml | DONE | Live smoke migrado GO Core |
| 2026-06-21T09:21:00Z | codex3 | CHECK-OUT | package.json | DONE | test:contract atualizado |
| 2026-06-21T09:23:00Z | codex5 | CHECK-OUT | vitest.config.ts | DONE | CRIT-004: timeout, pool=forks, clearMocks, GO_CORE_LIVE |
| 2026-06-21T09:23:00Z | codex5 | CHECK-OUT | src/__tests__/setup.ts | DONE | beforeEach reset useSessionStore |
| 2026-06-21T09:23:00Z | codex5 | CHECK-OUT | tests/contract/go-core-surface.test.ts | DONE | RCA A1: contract tests com describeIfLive skip |
| 2026-06-21T09:23:00Z | codex5 | CHECK-OUT | scripts/pre-ship-gate.sh | DONE | RCA A3: pre-ship gate tsc→lint→test→build→contract |
| 2026-06-21T23:59:00Z | nemotron_ultra2 | CHECK-OUT | .planning/QA_REPORT_SPRINT3.md | DONE | GATE 2 BLOQUEADO: build fail, CAO residual em test files |
| 2026-06-21T23:59:00Z | nemotron_ultra2 | CHECK-OUT | .planning/QA_REPORT_SPRINT3.md | DONE | Keystore coverage: 9/9 validators 100% OK |
| 2026-06-21T10:44:00Z | orquestrador_opus46 | CLEANUP | AGENT_LEDGER_S3.md | DONE | Ledger reconstruído — removidas 112+ entradas duplicadas do nemotron_ultra2 + corrigido truncamento acidental |

---

## FILE LOCK TABLE

| Arquivo | Status |
|---------|--------|
| `src/api/go-core-base-url.ts` | ✅ DONE |
| `src/api/go-core-client.ts` | ✅ DONE |
| `src/api/index.ts` | ✅ DONE |
| `src/api/connect-terminal-socket.ts` | ✅ DONE |
| `src/api/types.ts` | ✅ DONE |
| `src/settings/settings-store.ts` | ✅ DONE |
| `src/vite-env.d.ts` | ✅ DONE |
| `src/shared/storage/migrations.ts` | ✅ DONE |
| `.env.local` | ✅ DONE |
| `.env.example` | ✅ DONE |
| `vitest.config.ts` | ✅ DONE |
| `src/__tests__/setup.ts` | ✅ DONE |
| `tests/contract/go-core-surface.test.ts` | ✅ DONE |
| `scripts/pre-ship-gate.sh` | ✅ DONE |
| `tests/e2e/smoke.spec.ts` | ✅ DONE |
| `tests/e2e/canvas-deploy.spec.ts` | ✅ DONE |
| `playwright.config.ts` | ✅ DONE |
| `infra/runtime/cloudbuild.yaml` | ✅ DONE |
| `.github/workflows/ci.yml` | ✅ DONE |
| `.github/workflows/contract-nightly.yml` | ✅ DONE |
| `.github/workflows/smoke-live.yml` | ✅ DONE |
| `src/api/session-discovery.ts` | ✅ DONE |
| `src/api/health-store.ts` | ✅ DONE |
| `src/canvas-reconciler/reconciler.ts` | ✅ DONE |
| `src/health/FirstRunWizard.tsx` | ✅ DONE |
| `src/health/HealthPage.tsx` | ✅ DONE |
| `src/shell/canvas-command-adapter.ts` | ✅ DONE |
| `src/voice/command-executor.ts` | ✅ DONE |
| `src/voice/VoicePanel.tsx` | ✅ DONE |
| `package.json` | 🟡 nemotron_ultra IN PROGRESS (ESLint fix) |
| `.eslintrc.cjs` | 🟡 nemotron_ultra IN PROGRESS (ESLint fix) |
| `eslint-rules/` | 🟡 nemotron_ultra IN PROGRESS (ESLint fix) |
| `src/canvas-reconciler/reconciler.ts` | ⏳ codex1 aguarda GATE 1 (orphan cleanup) |

---

## GATES DE QUALIDADE

### GATE 1 — Desbloqueio (nemotron_ultra finaliza)
- [x] `npx tsc --noEmit` = 0 errors ✅
- [x] `grep cao-client src/` = apenas __tests__/ e legados ✅
- [x] `grep "from.*cao-client" src/` = vazio ✅
- [x] `grep "from.*base-url" src/` = apenas legados ✅
- [ ] Aprovação Orquestrador: _______________

### GATE 2 — Build + Test (após npm install + GATE 1)
- [ ] `npm run build` = exit 0
- [ ] `npm run test` < 120s
- [ ] Aprovação Orquestrador: _______________

---

## BLOQUEADORES

| ID | Descrição | Status |
|----|-----------|--------|
| B1 | node_modules instalado no Windows, falta @rollup/rollup-linux-x64-gnu | npm install em andamento |
| B2 | nemotron_ultra não finalizou sweep (6+ arquivos) | AGUARDANDO |
| B3 | GO Core Server não está rodando no WSL | Não bloqueia frontend |

---

*Ledger reconstruído por: orquestrador_opus46*
*2026-06-21T10:44:00Z*
| 2026-06-21T12:15:01Z | OP1 | CHECK-IN | src/agent-studio/AgentStudioPage.tsx | IN PROGRESS | Codex#4 squad2/inferior esquerdo renomeia caoClient para goCoreClient no Agent Studio |
| 2026-06-21T12:18:14Z | OP1 | CHECK-OUT | src/agent-studio/AgentStudioPage.tsx | DONE | caoClient -> goCoreClient aplicado; CaoApiError mantido pois GoCoreApiError nao existe; typecheck filtrado sem saida |
| 2026-06-21T12:16:22Z | CX5 | CHECK-IN | src/chat-view/ChatView.tsx | IN PROGRESS | Codex#5 squad2/superior direito renomeia caoClient para goCoreClient no ChatView |
| 2026-06-21T12:20:00Z | nemotron_ultra | CHECK-IN | src/dashboard/DashboardPage.tsx | IN PROGRESS | Renomear caoClient para goCoreClient |
| 2026-06-21T12:21:00Z | nemotron_ultra | CHECK-OUT | src/dashboard/DashboardPage.tsx | DONE | caoClient -> goCoreClient aplicado (import + 3 chamadas); tsc 0 errors |
| 2026-06-21T12:22:00Z | nemotron_ultra | CHECK-IN | src/shell/canvas-command-adapter.ts | IN PROGRESS | Renomear caoClient para goCoreClient |
| 2026-06-21T12:23:00Z | nemotron_ultra | CHECK-OUT | src/shell/canvas-command-adapter.ts | DONE | caoClient -> goCoreClient aplicado (import + 1 chamada); tsc 0 errors |
| 2026-06-21T12:24:39Z | CX5 | CHECK-IN | src/flows/FlowsPage.tsx | IN PROGRESS | Codex#5 squad2/superior direito renomeia caoClient para goCoreClient em FlowsPage |
| 2026-06-21T12:27:00Z | CX5 | CHECK-OUT | src/flows/FlowsPage.tsx | DONE | caoClient -> goCoreClient aplicado no import e 5 chamadas; caoQueryKeys mantido pois nao existe goCoreQueryKeys exportado; typecheck filtrado sem saida |
| 2026-06-21T12:24:16Z | OP1 | CHECK-IN | src/finops/use-cost-estimate.ts | IN PROGRESS | Codex#4 squad2/inferior esquerdo renomeia caoClient para goCoreClient no FinOps cost estimate |
| 2026-06-21T12:26:15Z | OP1 | CHECK-OUT | src/finops/use-cost-estimate.ts | DONE | caoClient -> goCoreClient aplicado; typecheck filtrado sem saida |
| 2026-06-21T12:36:36Z | CX1 | CHECK-IN | src/memory-viewer/MemoryViewerPage.tsx | IN PROGRESS | Renomear caoClient para goCoreClient; stat=2026-05-28 22:10:03.069650500 -0700 |
| 2026-06-21T12:39:24Z | CX1 | CHECK-OUT | src/memory-viewer/MemoryViewerPage.tsx | DONE | caoClient -> goCoreClient aplicado; npx tsc --noEmit 2>&1 \| grep -i "memory\|error" \| head -10 sem saida |
| 2026-06-21T12:38:11Z | CX2 | CHECK-IN | src/terminal-grid/TabBar.tsx | IN PROGRESS | Renomear caoClient para goCoreClient; stat=2026-05-30 21:26:10.878512600 -0700 |
| 2026-06-21T12:38:52Z | CX3 | CHECK-IN | src/terminal-grid/TerminalGrid.tsx | IN PROGRESS | Renomear caoClient para goCoreClient; stat=2026-05-28 20:59:39.937231300 -0700 |
| 2026-06-21T12:41:23Z | CX3 | CHECK-OUT | src/terminal-grid/TerminalGrid.tsx | DONE | caoClient -> goCoreClient aplicado no import e 5 chamadas; tsc filtrado sem saida |
| 2026-06-21T12:40:38Z | CX2 | CHECK-OUT | src/terminal-grid/TabBar.tsx | DONE | caoClient -> goCoreClient aplicado no import e chamada listTerminalsInSession; npx tsc --noEmit 2>&1 \| grep -i "tabbar\|error" \| head -10 sem saida |
| 2026-06-21T12:49:12Z | CX5 | CHECK-IN | src/health/HealthPage.tsx | IN PROGRESS | Renomear caoClient para goCoreClient; stat=2026-06-21 03:24:21.182521100 -0700 |
| 2026-06-21T12:50:56Z | CX5 | CHECK-OUT | src/health/HealthPage.tsx | DONE | caoClient -> goCoreClient aplicado no import e 5 referencias |
| 2026-06-21T12:48:58Z | OP1 | CHECK-IN | src/health/FirstRunWizard.tsx | IN PROGRESS | Codex#4 renomeia caoClient para goCoreClient; stat=2026-06-21 03:24:03.386981000 -0700 |
| 2026-06-21T12:50:52Z | OP1 | CHECK-OUT | src/health/FirstRunWizard.tsx | DONE | caoClient -> goCoreClient aplicado no import e 7 referencias |
| 2026-06-21T13:05:18Z | CX2 | CHECK-IN | src/canvas-reconciler/reconciler.ts | IN PROGRESS | Renomear aliases CaoClient/caoClient para GoCoreClient/goCoreClient |

- CHECK-IN 2026-06-21: Renomear CaoClient para GoCoreClient em src/voice/command-executor.ts.
| 2026-06-21T13:07:12Z | CX2 | CHECK-OUT | src/canvas-reconciler/reconciler.ts | DONE | Aliases removidos: GoCoreClient/goCoreClient usados diretamente; sem ocorrencias de CaoClient/caoClient no arquivo |
| 2026-06-21T13:06:53Z | CX5 | CHECK-IN | src/voice/VoicePanel.tsx | IN PROGRESS | Renomear caoClient para goCoreClient |
| 2026-06-21T13:08:23Z | CX5 | CHECK-OUT | src/voice/VoicePanel.tsx | DONE | caoClient -> goCoreClient aplicado no import e deps; propriedade cao mantida por contrato CommandExecutorDeps |
| 2026-06-21T13:13:00Z | orquestrador_opus46 | GATE-APPROVAL | GATE_1 | ✅ APROVADO | ZERO caoClient refs em prod. 19/19 arquivos migrados. tsc --noEmit = 0 errors |
| 2026-06-21T15:22:00Z | CX-1 | CHECK-IN | src/api/errors.ts, src/api/query-keys.ts, src/api/index.ts, src/api/go-core-client.ts | IN PROGRESS | API Core renames: CaoApiError→GoCoreApiError, caoQueryKeys→goCoreQueryKeys, remove compat aliases |
| 2026-06-21T15:24:45Z | CX-1 | CHECK-OUT | src/api/errors.ts, src/api/query-keys.ts, src/api/index.ts, src/api/go-core-client.ts | DONE | Renamed API errors/query keys to GoCore names, removed compat aliases, required grep returned zero matches, focused API typecheck passed |
| 2026-06-21T15:23:49Z | CX-2 | CHECK-IN | src/api/__tests__/cao-client.test.ts; src/api/__tests__/contract/cao-contract.test.ts; src/api/__tests__/msw/handlers.ts; src/api/__tests__/no-direct-cao-fetch-rule.test.ts | IN PROGRESS | API test rewrites + MSW + ESLint rule test |
| 2026-06-21T15:26:16Z | CX-3 | CHECK-IN | src/voice/command-executor.ts, src/voice/VoicePanel.tsx, src/voice/__tests__/command-executor.test.ts | IN PROGRESS | Voice deps.cao→deps.goCore rename |
| 2026-06-21T15:25:52Z | CX-2 | CHECK-OUT | src/api/__tests__/cao-client.test.ts; src/api/__tests__/contract/cao-contract.test.ts; src/api/__tests__/msw/handlers.ts; src/api/__tests__/no-direct-cao-fetch-rule.test.ts | DONE | Archived legacy CAO tests, added GO Core replacements, updated MSW WebSocket mocks to 8080, updated ESLint rule test naming |
| 2026-06-21T15:28:29Z | CX-4 | CHECK-IN | src/agent-studio/AgentStudioPage.tsx, src/flows/FlowsPage.tsx, src/flows/__tests__/FlowsPage.test.tsx, src/memory-viewer/__tests__/MemoryViewerPage.test.tsx | IN PROGRESS | Prod consumer + test renames |
| 2026-06-21T15:28:31Z | CX-3 | CHECK-OUT | src/voice/command-executor.ts, src/voice/VoicePanel.tsx, src/voice/__tests__/command-executor.test.ts | DONE | Renamed CommandExecutorDeps field and usages from cao to goCore; required grep checks returned zero identifier matches |
| 2026-06-21T15:29:02Z | CX-4 | CHECK-OUT | src/agent-studio/AgentStudioPage.tsx, src/flows/FlowsPage.tsx, src/flows/__tests__/FlowsPage.test.tsx, src/memory-viewer/__tests__/MemoryViewerPage.test.tsx | DONE | Renamed caoQueryKeys/CaoApiError/caoClient consumer references to GO Core names; required grep returned zero matches |
| 2026-06-21T15:32:21Z | CX-5 | CHECK-IN | src/canvas-reconciler/__tests__/reconciler.test.ts, src/terminal-grid/__tests__/TerminalGrid.test.tsx, src/chat-view/__tests__/ChatView.test.tsx, src/shell/__tests__/canvas-command-adapter.test.ts | IN PROGRESS | Test file CAO→GO Core renames |
| 2026-06-21T15:35:00Z | NM-2 | CHECK-IN | .eslintrc.cjs, src/vite-env.d.ts, src/dashboard/__tests__/DashboardPage.test.tsx | IN PROGRESS | Config cleanup + Dashboard test rename |
| 2026-06-21T15:36:00Z | NM-2 | CHECK-OUT | .eslintrc.cjs, src/vite-env.d.ts, src/dashboard/__tests__/DashboardPage.test.tsx | DONE | ESLint rules renamed to no-direct-go-core-fetch; VITE_CAO_BASE_URL removed from vite-env.d.ts; Dashboard test imports goCoreClient from @/api |
| 2026-06-21T15:45:17Z | CX-5 | CHECK-OUT | src/canvas-reconciler/__tests__/reconciler.test.ts, src/terminal-grid/__tests__/TerminalGrid.test.tsx, src/chat-view/__tests__/ChatView.test.tsx, src/shell/__tests__/canvas-command-adapter.test.ts | DONE | All CAO→GO Core renames complete: mockCaoClient→mockGoCoreClient (reconciler), caoClient→goCoreClient (TerminalGrid, ChatView, canvas-command-adapter), cao-client-stub→go-core-client-stub, bound caoClient→bound goCoreClient. Verified zero matches across all 4 files. |
| 2026-06-21T15:51:06Z | NM-1 | CHECK-IN | src/api/base-url.ts, src/api/cao-client.ts, LEGACY_ARCHIVE_REFERENCE.md, src/health/__tests__/FirstRunWizard.test.tsx, src/health/__tests__/HealthPage.test.tsx | IN PROGRESS | Archive legacy CAO files, create reference doc, update health test imports |
| 2026-06-21T16:01:13Z | NM-1 | CHECK-OUT | src/api/base-url.ts, src/api/cao-client.ts, LEGACY_ARCHIVE_REFERENCE.md, src/health/__tests__/FirstRunWizard.test.tsx, src/health/__tests__/HealthPage.test.tsx | DONE | Archived base-url.ts and cao-client.ts as .old files, created LEGACY_ARCHIVE_REFERENCE.md, updated FirstRunWizard.test.tsx and HealthPage.test.tsx imports from @/api/cao-client to @/api with goCoreClient, replaced all caoClient references (10 in HealthPage, 3 in FirstRunWizard). Verified zero caoClient matches in both test files. |
| 2026-06-21T18:18:24Z | CX-3 | CHECK-IN | src/voice/command-executor.ts, src/voice/__tests__/command-executor.test.ts, src/shared/topology-guard.ts, src/shared/validation-proxy.ts, src/shell/app-fetch.ts, vite.config.ts | IN PROGRESS | Comment cleanup CAO→GO Core |
| 2026-06-21T18:19:36Z | CX-3 | CHECK-OUT | src/voice/command-executor.ts, src/voice/__tests__/command-executor.test.ts, src/shared/topology-guard.ts, src/shared/validation-proxy.ts, src/shell/app-fetch.ts, vite.config.ts | DONE | Updated CAO comments/test descriptions/runtime string to GO Core; renamed local MockCao helper to satisfy grep; required verification returned zero matches |
| 2026-06-21T17:25:23Z | CX-1 | CHECK-IN | src/dashboard/__tests__/DashboardPage.test.tsx | IN PROGRESS | Fix failing Dashboard tests |
| 2026-06-21T17:48:18Z | CX-1 | CHECK-OUT | src/dashboard/__tests__/DashboardPage.test.tsx | DONE | Reworked DashboardPage test mocks so fleet snapshot uses mocked GO Core, session store, and canvas store data; targeted Vitest passed 2/2 |
| 2026-06-21T18:19:17Z | CX-4 | CHECK-IN | start.sh | IN PROGRESS | start.sh CAO→GO Core rewrite |
| 2026-06-21T18:20:04Z | CX-4 | CHECK-OUT | start.sh | DONE | Rewrote startup script references from CAO/9889 to GO Core/8080; required grep returned zero matches and bash syntax check passed |
| 2026-06-21T18:20:30Z | CX-5 | CHECK-IN | infra/runtime/Dockerfile, infra/runtime/service.yaml, infra/runtime/run-local.sh | IN PROGRESS | Infra CAO→GO Core |
| 2026-06-21T18:21:38Z | CX-5 | CHECK-OUT | infra/runtime/Dockerfile, infra/runtime/service.yaml, infra/runtime/run-local.sh | DONE | Renamed runtime env vars, server command, state path, comments, and ports to GO Core/8080; required grep returned 0 matches in all three files |
| 2026-06-21T18:22:10Z | NM-1 | CHECK-IN | infra/runtime/cloudbuild.yaml, infra/runtime/auth-proxy/README.md, tests/e2e/perf-12-terminals.spec.ts, tests/e2e/canvas-deploy.spec.ts, tests/e2e/canvas-session.spec.ts, tests/e2e/sessions.spec.ts, tests/e2e/smoke.spec.ts | IN PROGRESS | Cloudbuild + auth-proxy + E2E URLs cleanup |
| 2026-06-21T18:30:53Z | CX-2 | CHECK-IN | eslint-rules/index.cjs, src/shell/NavBar.tsx, src/api/__tests__/connect-terminal-socket.test.ts | IN PROGRESS | W2: ESLint plugin + NavBar + socket test |
| 2026-06-21T18:32:04Z | CX-2 | CHECK-OUT | eslint-rules/index.cjs, src/shell/NavBar.tsx, src/api/__tests__/connect-terminal-socket.test.ts | DONE | W2 updated ESLint GO Core rule naming with compat alias, NavBar health pill id, and socket test URLs/names |
| 2026-06-21T18:33:23Z | NM-1 | CHECK-OUT | infra/runtime/cloudbuild.yaml, infra/runtime/auth-proxy/README.md, tests/e2e/perf-12-terminals.spec.ts, tests/e2e/canvas-deploy.spec.ts, tests/e2e/canvas-session.spec.ts, tests/e2e/sessions.spec.ts, tests/e2e/smoke.spec.ts | DONE | Updated cloudbuild.yaml comment (legacy CAO_LIVE reference), auth-proxy README (9889→8080, CAO_TENANT_ID→GO_CORE_TENANT_ID), perf-12-terminals.spec.ts (5 occurrences of 9889→8080). Scanned other 4 E2E specs - no CAO/9889 refs found. Verified zero matches across all files. |
| 2026-06-21T19:11:56Z | CX-1 | CHECK-IN | src/api/query-keys.ts, src/api/use-installed-cli-providers.ts, src/api/session-discovery.ts, src/api/go-core-client.ts, src/api/go-core-base-url.ts, src/api/__tests__/go-core-client.test.ts, src/api/__tests__/msw/handlers.ts, src/api/__tests__/session-store.test.ts, src/settings/routes.tsx, scripts/deploy-cloud.sh, openspec/config.yaml, .github/workflows/keystore-contract-nightly.yml | IN PROGRESS | W3: API + Settings + Scripts |
| 2026-06-21T19:14:19Z | CX-1 | CHECK-OUT | src/api/query-keys.ts, src/api/use-installed-cli-providers.ts, src/api/session-discovery.ts, src/api/go-core-client.ts, src/api/go-core-base-url.ts, src/api/__tests__/go-core-client.test.ts, src/api/__tests__/msw/handlers.ts, src/api/__tests__/session-store.test.ts, src/settings/routes.tsx, scripts/deploy-cloud.sh, openspec/config.yaml, .github/workflows/keystore-contract-nightly.yml | DONE | W3 API/settings/scripts/config references updated to GO Core and 8080; required verification greps returned zero matches |
| 2026-06-21T19:13:03Z | CX-2 | CHECK-IN | src/health/HealthPage.tsx, src/health/FirstRunWizard.tsx, src/health/__tests__/HealthPage.test.tsx, src/health/__tests__/FirstRunWizard.test.tsx | IN PROGRESS | W3: Health module CAO→GO Core |
| 2026-06-21T19:13:54Z | NM-1 | CHECK-IN | src/canvas-builder/CanvasBuilderPage.tsx, src/canvas-builder/deploy-validation.ts, src/canvas-builder/provider-options.ts, src/canvas-builder/__tests__/deploy-validation.test.ts, src/canvas-reconciler/reconciler.ts, src/canvas-reconciler/__tests__/reconciler.test.ts, src/flows/FlowsPage.tsx, src/agent-studio/AgentStudioPage.tsx, src/agent-studio/__tests__/AgentStudioPage.test.tsx, src/finops/usage-capture.ts | IN PROGRESS | W3: Canvas + Reconciler + Flows + AgentStudio + FinOps |
| 2026-06-21T19:14:39Z | CX-2 | CHECK-OUT | src/health/HealthPage.tsx, src/health/FirstRunWizard.tsx, src/health/__tests__/HealthPage.test.tsx, src/health/__tests__/FirstRunWizard.test.tsx | DONE | W3 renamed health module CAO state/text/tests to GO Core and removed 9889 placeholder |
| 2026-06-21T19:15:29Z | NM-1 | CHECK-OUT | src/canvas-builder/CanvasBuilderPage.tsx, src/canvas-builder/deploy-validation.ts, src/canvas-builder/provider-options.ts, src/canvas-builder/__tests__/deploy-validation.test.ts, src/canvas-reconciler/reconciler.ts, src/canvas-reconciler/__tests__/reconciler.test.ts, src/flows/FlowsPage.tsx, src/agent-studio/AgentStudioPage.tsx, src/agent-studio/__tests__/AgentStudioPage.test.tsx, src/finops/usage-capture.ts | DONE | W3 text cleanup: CAO comments, test names, and user-visible strings changed to GO Core; required grep returned zero matches |
| 2026-06-21T19:38:30Z | CX-1 | CHECK-IN | src/agent-studio/AgentStudioPage.tsx | IN PROGRESS | Rename formatCaoError |
| 2026-06-21T19:40:32Z | CX-1 | CHECK-OUT | src/agent-studio/AgentStudioPage.tsx | DONE | Renamed formatCaoError to formatGoCoreError and verified zero cao matches |
