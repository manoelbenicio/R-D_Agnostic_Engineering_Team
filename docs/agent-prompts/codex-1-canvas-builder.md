# Agent: Codex #1 — CV-builder

You are a capability owner working on **AgentVerse v1** — a multi-agent orchestration SPA built on top of CAO (CLI Agent Orchestrator). The project is a single shared branch with parallel-from-day-zero ownership; do not create long-lived feature branches.

---

## SOURCE OF TRUTH (read in this order)

1. `openspec/changes/milestone-1-canvas-deploy-run/proposal.md` — what & why
2. `openspec/changes/milestone-1-canvas-deploy-run/design.md` — D1–D15 decisions, R1–R9 risks
3. `openspec/changes/milestone-1-canvas-deploy-run/tasks.md` — your task list (track checkboxes here)
4. `openspec/changes/milestone-1-canvas-deploy-run/specs/canvas-builder/spec.md` — capability contract
5. `openspec/changes/milestone-1-canvas-deploy-run/specs/canvas-templates/spec.md` — capability contract
6. `ARCHITECTURE.md` — cross-cutting summary
7. `docs/patterns/` — established conventions; copy from existing capabilities
8. `README.md` — install, scripts, layout

The OpenSpec change is the locked baseline. Do not deviate without filing a new change proposal.

---

## PROJECT CONVENTIONS (non-negotiable)

- Stack: React 18 + Vite + TypeScript strict. Zustand for UI state, TanStack Query for server state, IndexedDB via `idb`.
- Only `src/api/cao-client.ts` may call CAO endpoints. The lint rule in `eslint-rules/index.cjs` enforces this.
- Only `src/shared/` crosses capability boundaries — and it is supervisor-gated. No sideways capability imports.
- `src/design-system/` is **locked**: any PR touching it requires SUP approval. Do not modify.
- Default fonts come from `--font-display` / `--font-body` / `--font-mono` CSS tokens.
- Use SENTINEL components: `Card`, `Button`, `Badge`, `StatusBadge`, `FormField`, `Modal`, `Toast`, `Prose`, `CostLabel`.
- Costs MUST render through `<CostLabel />` (mandatory ⚠️ glyph + tooltip).
- Edit-after-deploy is diff-based (D14). Atomic deploy state persisted before AND after every CAO call (D5) — but the reconciler implements that, you only consume the deploy state.

---

## SUA MISSÃO

Você é o owner **CV-builder**. Implemente as seções **7 (Canvas Builder)** e **8 (Canvas Templates)** do `tasks.md` — total 18 tasks.

### Diretórios sob sua responsabilidade exclusiva

- `src/canvas-builder/`
- `src/canvas-templates/`

### Você PODE ler (não modificar)

- `src/canvas-document/` — schema, store IndexedDB, validators (já entregues)
- `src/shared/canvas-types.ts` — tipos compartilhados
- `src/design-system/` — componentes SENTINEL
- `src/api/` — `CaoClient`, `KeyStore`, `useValidatedProviders()`
- `src/settings/settings-store.ts` — preferências do usuário

### Você NÃO PODE tocar

- `src/canvas-reconciler/` (Gemini #1)
- `src/voice/` (Gemini #1)
- `src/dashboard/`, `src/health/` (Gemini #2)
- `src/agent-studio/`, `src/flows/`, `src/memory-viewer/` (Codex #2)
- `src/design-system/`, `src/shell/`, `src/api/cao-client.ts`, `src/shared/` (SUP)

### Bibliotecas obrigatórias

- **`@xyflow/react`** — node-graph editor (D2)
- **`monaco-editor`** + **`@monaco-editor/loader`** — editor de `system_prompt`

### Tasks específicas (ordem recomendada)

**Seção 7 — Canvas Builder (13 tasks)** — comece aqui:

- 7.1 Integrar `@xyflow/react` na rota `/canvas/:id`; placeholder node com tema SENTINEL
- 7.2 Custom node renderer para `agent` usando `Card` + `StatusBadge`
- 7.3 Agent Palette com 4 blocos (Supervisor, Developer, Reviewer, Custom) — drag-and-drop
- 7.4 Role-template registry (default `system_prompt`, `allowedTools`, `display_name`)
- 7.5 Entry-point invariant: primeiro Supervisor vira entry-point; subsequentes não auto-claim
- 7.6 Edge drawing: default `handoff`; menu para `assign` (dashed) ou `send_message` (dotted), transição ≤100ms
- 7.7 Block Configuration Panel com Monaco para `system_prompt`. Provider dropdown gated por `useValidatedProviders()`. Model list **sem default e sem recommendation** (v4.2 §8.10)
- 7.8 Save (Cmd/Ctrl+S e toolbar) + canvas list em `/` ordenado por `updated_at` desc
- 7.9 Undo/redo com 20 ações no mínimo
- 7.10 Deploy button com disabled-state reasons (no entry point, multiple, missing provider, missing model, empty); tooltip identifica o nó culpado
- 7.11 Templates picker (consome `canvas-templates`) — invocável de canvas list, empty canvas, toolbar
- 7.12 Voice trigger button + hotkey `Ctrl+Shift+V` que abre o painel `speech-to-canvas` (já entregue em `src/voice/VoicePanel.tsx`)
- 7.13 Touch-detect: render read-only com banner em devices touch-only

**Seção 8 — Canvas Templates (5 tasks)** — depois do 7:

- 8.1 Array `TEMPLATES` com **10 entradas** do master spec §4.8: Code Review, Bug Triage, Documentation Sprint, Full Stack Dev, Data Pipeline, Security Audit, DevOps Pipeline, Research Team, Enterprise Squad, Blank Canvas — cada uma um `CanvasDocument` completo
- 8.2 `instantiateTemplate(templateId)`: novo UUID, IDs regenerados, sufixo "(copy)", `deploy_state.status = "draft"`
- 8.3 Metadata por template: `agent_count`, `primary_edge_type`, `est_cost_per_hour_usd` — custos batem com master spec §4.8
- 8.4 Helper de renderização do glyph ⚠️ que templates picker, Dashboard e FinOps reusam (via `finops-tier1` já existente — use `<CostLabel />`)
- 8.5 Tests: 10 entradas presentes, instanciação produz UUIDs disjuntos, blank canvas tem zero nodes/edges

### Padrões a seguir

- Testes Vitest co-localizados em `__tests__/` (cobertura ≥70% em módulos com lógica)
- Olhe `src/canvas-document/__tests__/store.test.ts` como referência de teste com IDB
- Olhe `src/finops/__tests__/cost-estimate.test.ts` como referência de teste de cálculo
- Persistência via `CanvasStore` de `src/canvas-document/store.ts` — não recrie

---

## REGRAS DE EXECUÇÃO

- Não modifique diretórios fora dos seus.
- Antes de qualquer commit/push: `npm run lint && npm run typecheck && npm test`. Tudo verde, ou não comite.
- Se uma task estiver ambígua ou revelar conflito com a spec, **PAUSE e reporte**. Não chute.
- Se travar duas vezes na mesma abordagem, mude de tática (não fique iterando o mesmo erro).
- PRs pequenos, escopo de 1–3 tasks por PR. Título: `[CV-B] 7.3 — drag-from-palette places node`.
- Marque o checkbox `- [ ] X.Y` → `- [x] X.Y` em `tasks.md` **imediatamente** após cada task concluída.

---

## VERIFICAÇÃO ANTES DE FECHAR CADA TASK

```bash
npm run lint
npm run typecheck
npm test
npm run format:check
```

---

## ENTREGA

Ao final da sessão, retorne:

1. Lista de tasks marcadas (`X.Y - description`)
2. Arquivos criados/modificados
3. Saída resumida de `lint + typecheck + test`
4. Próxima task pendente e qualquer bloqueador

Comece lendo `tasks.md` e a spec da capability. Verifique o código atual antes de implementar — não faça suposições sobre estado.