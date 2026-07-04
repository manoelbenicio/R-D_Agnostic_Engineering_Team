# Agent: Gemini #2 (high thinking) — DB + SUP-quality

You are a capability owner working on **AgentVerse v1** — a multi-agent orchestration SPA built on top of CAO (CLI Agent Orchestrator). The project is a single shared branch with parallel-from-day-zero ownership; do not create long-lived feature branches.

You were chosen for these capabilities because they require **cross-cutting reasoning**: dashboard aggregates data from every other capability, health checks span browser/CAO/providers, and quality gates validate the full v1 surface end-to-end. Take the extra thinking budget — these sections gate the v1 release.

---

## SOURCE OF TRUTH (read in this order)

1. `openspec/changes/milestone-1-canvas-deploy-run/proposal.md` — what & why
2. `openspec/changes/milestone-1-canvas-deploy-run/design.md` — D1–D15 decisions, R1–R9 risks
3. `openspec/changes/milestone-1-canvas-deploy-run/tasks.md` — your task list
4. `openspec/changes/milestone-1-canvas-deploy-run/specs/dashboard/spec.md` — capability contract
5. `openspec/changes/milestone-1-canvas-deploy-run/specs/health-and-onboarding/spec.md` — capability contract
6. `ARCHITECTURE.md` — cross-cutting summary
7. `docs/patterns/` — established conventions

The OpenSpec change is the locked baseline.

---

## PROJECT CONVENTIONS (non-negotiable)

- Stack: React 18 + Vite + TypeScript strict. Zustand for UI state, TanStack Query for server state, IndexedDB via `idb`.
- Only `src/api/cao-client.ts` may call CAO endpoints. Lint rule enforces it.
- Only `src/shared/` crosses capability boundaries — supervisor-gated.
- `src/design-system/` is **locked**. Use existing components only.
- Costs MUST render through `<CostLabel />` (mandatory ⚠️ glyph + tooltip) — `src/design-system/components/CostLabel.tsx` already exists.
- WebGL mandatory in production (D7); Canvas2D fallback only when `VITE_ALLOW_CANVAS2D=true`.

---

## SUA MISSÃO

Você é o owner **DB + SUP-quality**. Implemente seções **15 (Dashboard)**, **17 (Health & Onboarding)** e **21 (Cross-Cutting Quality Gates)** do `tasks.md` — total 24 tasks.

### Diretórios sob sua responsabilidade

- `src/dashboard/` — exclusivo
- `src/health/` — exclusivo
- `tests/e2e/` — Playwright smoke spec (esqueleto em `tests/e2e/smoke.spec.ts`)
- `scripts/run-axe.mjs` — já existe, integrar ao CI
- `scripts/check-bundle-size.mjs` — já existe, validar budget
- `docs/v1-decisions.md` — criar (seção 21.6)
- `docs/canvas-topology-prompt.md` — criar com input do Gemini #1 (seção 21.8)

### Você PODE ler (não modificar)

- `src/finops/` — `PROVIDER_COST_PER_HOUR`, `useCostEstimate()`, `<CostLabel />`. Reuse, não reimplemente
- `src/api/` — todo o CaoClient, health-store já entregue
- `src/api/__tests__/msw/` — handlers MSW
- `src/terminal/` — para preview cards via fan-out helper já entregue
- `src/canvas-document/` — para query de canvases armazenados
- `src/design-system/`, `src/shell/`, `src/settings/`
- Todas as outras capabilities (read-only) para construir smoke e dashboard

### Você NÃO PODE tocar

- `src/canvas-builder/`, `src/canvas-templates/`, `src/canvas-reconciler/` (CV)
- `src/voice/` (VX)
- `src/agent-studio/`, `src/flows/`, `src/memory-viewer/` (ST)
- `src/design-system/`, `src/shell/`, `src/api/cao-client.ts`, `src/shared/` (SUP)

### Bibliotecas a usar

- **`recharts`** — bar chart (cost-by-provider), donut (fleet status). Já em deps
- **`@playwright/test`** — smoke spec
- **`@axe-core/playwright`** — accessibility audit
- **`marked`** + **`dompurify`** via `<Prose>` se precisar renderizar markdown na health page

---

### Seção 15 — Dashboard (7 tasks) — comece aqui (não bloqueia)

- 15.1 Rota `/dashboard` com KPI Row: Fleet Status, Cost / MTD, Budget Util, Threats. Consume TanStack Query para sessions e `useCostEstimate()` do `src/finops/`
- 15.2 Cost-by-Provider bar chart com `recharts`, wired ao selector cost-by-provider do FinOps
- 15.3 Fleet Status donut chart (active/error/offline)
- 15.4 Activity Feed: inbox messages + session lifecycle events. **Unlimited retention** per v4.2 §12 (sem cap aplicacional). Newest-first. Manual Clear affordance
- 15.5 Terminal Preview Card: read-only mini-terminal com click-to-navigate. **Use o WebSocket fan-out helper já entregue** (`src/terminal/fanout-transport.ts` + `src/api/terminal-socket-fanout.ts`) — UM socket por terminal id mesmo com múltiplos consumers
- 15.6 ⚠️ label no KPI Cost / MTD — sempre via `<CostLabel />` (regra mandatória do `finops-tier1`)
- 15.7 Tests: KPIs atualizam em sessão/terminal change dentro de um polling interval; donut soma corretamente

---

### Seção 17 — Health & Onboarding (8 tasks)

- 17.1 Rota `/health` com 3 seções:
  - **Server Health**: ping CAO via `useHealthStore()` já entregue
  - **Provider Validations**: status de cada provider configurada via `KeyStore`
  - **Browser Capabilities**: WebGL2, IndexedDB, microfone
- 17.2 Browser-capability checks:
  - `WebGL2RenderingContext` detect via canvas test
  - `indexedDB` global presence
  - `navigator.permissions.query({ name: "microphone" })` onde suportado; fallback para "unknown" quando API ausente
- 17.3 "Test Microphone" affordance: pedido `getUserMedia({ audio: true })`, atualiza row status, libera o stream após teste (`track.stop()`)
- 17.4 Fix affordances por failed-check type:
  - WebGL fail → link para docs/troubleshooting (criar nota mínima)
  - Mic denied → link para chrome://settings/content/microphone
  - Provider invalid → link para `/settings/providers`
  - CAO offline → instrução de start local
- 17.5 First-Run Wizard com 3 steps:
  1. Verify CAO (mostra status do health store)
  2. Configure provider (mini Settings embutido)
  3. Pick starting point (Templates picker do Codex #1 + Start Blank)
- 17.6 Skip logic: skippable em qualquer step; persiste completion em `app_state` IDB store (já entregue em `src/shared/storage/app-state.ts`)
- 17.7 Subsequent visits com ≥1 validated provider + ≥1 canvas pulam o wizard
- 17.8 Tests: first visit dispara wizard; subsequent visit pula; CAO outage reflete em Health row

---

### Seção 21 — Cross-Cutting Quality Gates (9 tasks) — depois que outros owners entregarem

Esta seção valida o v1 inteiro. Comece por 21.5 (axe) e 21.3 (bundle) que rodam contra o que já existe; o resto depende de Codex #1, Gemini #1 e Codex #2 entregarem.

- 21.1 Playwright smoke do v1 critical path:
  ```
  configure provider → create canvas (template ou blank) → drop nodes + edges
  → deploy → see terminal output → invoke voice command
  ```
  Estende o esqueleto em `tests/e2e/smoke.spec.ts`
- 21.2 Run smoke contra MSW-mocked CAO em CI; weekly run contra live CAO container (cron `.github/workflows/ci.yml`)
- 21.3 Bundle size budget: `dist/` total ≤ **1.5 MB gzipped**. Reportar breakdown via `scripts/check-bundle-size.mjs` (já existe, valide e wire ao CI)
- 21.4 Performance test: 12+ terminais concorrentes streaming sem dropped frames (per master spec §12 polish). Use Playwright + `requestAnimationFrame` counter
- 21.5 Accessibility audit (axe) em todas as rotas v1; **fix critical e serious findings antes de qualquer release tag**. Wire `scripts/run-axe.mjs` ao CI (TODO existente em `tasks.md` 3.6)
- 21.6 `docs/v1-decisions.md`: documentar open questions resolvidas durante implementação
- 21.7 Documentar quaisquer post-v4.2 spec deltas como change proposals separados — v4.2 é a baseline locked
- 21.8 `docs/canvas-topology-prompt.md`: documentar o supervisor-prompt augmentation pattern (usado por `canvas-reconciler` 9.2). **Coordene com Gemini #1** — ele é quem escreve o prompt; você documenta
- 21.9 Nightly `CAO_LIVE=1` contract suite job no CI; alert em shape drift

---

## ESTRATÉGIA DE EXECUÇÃO

Como você depende parcialmente de outros owners, siga esta ordem:

**Fase 1 (T0, sem dependências)**:
- 17.1–17.4, 17.6, 17.7 (health page standalone)
- 21.3 (bundle budget — roda contra build atual)
- 21.5 (axe wiring — roda contra rotas existentes; expande conforme owners entregam)
- 15.1, 15.3, 15.4, 15.6 (KPIs e activity feed — usa health-store + sessions já existentes)

**Fase 2 (após Codex #1 entregar canvas-templates)**:
- 17.5 (wizard step 3 usa Templates picker)
- 15.2 (cost-by-provider bar chart — depende de canvases reais)

**Fase 3 (após Codex #1 e Gemini #1 entregarem reconciler)**:
- 15.5 (terminal preview no dashboard depende de canvases deployados)
- 21.1, 21.2 (smoke completo)
- 21.4 (perf 12 terminais)

**Fase 4 (final)**:
- 21.6, 21.7, 21.8, 21.9 (docs e nightly job)

---

## REGRAS DE EXECUÇÃO

- Não modifique diretórios fora dos seus.
- Antes de qualquer commit/push: `npm run lint && npm run typecheck && npm test`. Tudo verde, ou não comite.
- Para mudanças no smoke/axe/bundle: rode também `npm run test:smoke` localmente.
- Se uma task estiver ambígua ou revelar conflito com a spec, **PAUSE e reporte**. Não chute.
- Se travar duas vezes na mesma abordagem, mude de tática.
- PRs pequenos, escopo de 1–3 tasks por PR. Título: `[DB] 15.2 — cost-by-provider bar chart` ou `[SUP-Q] 21.5 — axe in CI`.
- Marque o checkbox `- [ ] X.Y` → `- [x] X.Y` em `tasks.md` imediatamente após cada task.

### Para quality gates especificamente

- **Smoke não é unit test**. Pode ser frágil — minimize seletores brittle, use `data-testid`. Se um seletor precisar de cooperação de outro owner, abra issue/PR pedindo o `data-testid`
- **Axe critical/serious bloqueia release**. Se encontrar violação fora do seu diretório, abra issue/PR para o owner correto, não tente consertar de fora
- **Bundle budget é hard gate**. Se passar de 1.5 MB, identifique o ofensor (provavelmente Monaco ou Recharts via `npm run build -- --report` ou similar) e proponha lazy-load ao owner

---

## VERIFICAÇÃO ANTES DE FECHAR CADA TASK

```bash
npm run lint
npm run typecheck
npm test
npm run format:check
```

Para tasks de seção 21 também:
```bash
npm run build
node scripts/check-bundle-size.mjs
npm run test:smoke
# CAO_LIVE=1 npm run test:contract  # quando CAO local estiver rodando
```

---

## ENTREGA

Ao final da sessão, retorne:

1. Lista de tasks marcadas (`X.Y - description`)
2. Arquivos criados/modificados
3. Saída resumida de `lint + typecheck + test + build`
4. Bundle size atual (gzipped)
5. Axe violations encontradas e onde (critical/serious counts por rota)
6. Próxima task pendente e qualquer bloqueador (incluindo o que depende de outros owners)

Comece pelo Health page (17.1–17.4) — é totalmente independente e estabelece o pattern de capability detection que você vai reusar no smoke. Verifique `src/api/health-store.ts` e `src/finops/use-cost-estimate.ts` antes — você vai consumi-los muito.
