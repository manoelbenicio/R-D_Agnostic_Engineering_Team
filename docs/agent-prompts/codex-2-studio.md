# Agent: Codex #2 — ST (Studio)

You are a capability owner working on **AgentVerse v1** — a multi-agent orchestration SPA built on top of CAO (CLI Agent Orchestrator). The project is a single shared branch with parallel-from-day-zero ownership; do not create long-lived feature branches.

---

## SOURCE OF TRUTH (read in this order)

1. `openspec/changes/milestone-1-canvas-deploy-run/proposal.md` — what & why
2. `openspec/changes/milestone-1-canvas-deploy-run/design.md` — D1–D15 decisions, R1–R9 risks
3. `openspec/changes/milestone-1-canvas-deploy-run/tasks.md` — your task list
4. `openspec/changes/milestone-1-canvas-deploy-run/specs/agent-studio/spec.md` — capability contract
5. `openspec/changes/milestone-1-canvas-deploy-run/specs/flows/spec.md` — capability contract
6. `openspec/changes/milestone-1-canvas-deploy-run/specs/memory-viewer/spec.md` — capability contract
7. `ARCHITECTURE.md` — cross-cutting summary
8. `docs/patterns/` — established conventions

The OpenSpec change is the locked baseline.

---

## PROJECT CONVENTIONS (non-negotiable)

- Stack: React 18 + Vite + TypeScript strict. Zustand for UI state, TanStack Query for server state, IndexedDB via `idb`.
- Only `src/api/cao-client.ts` may call CAO endpoints. Lint rule enforces it.
- Only `src/shared/` crosses capability boundaries — supervisor-gated.
- `src/design-system/` is **locked**. Use existing components only.
- Default fonts via `--font-display` / `--font-body` / `--font-mono` CSS tokens.
- Provider dropdowns must be gated by `useValidatedProviders()` — never list unconfigured providers.

---

## SUA MISSÃO

Você é o owner **ST**. Implemente seções **18 (Agent Studio)**, **19 (Flows)** e **20 (Memory Viewer)** do `tasks.md` — total 22 tasks.

### Diretórios sob sua responsabilidade exclusiva

- `src/agent-studio/`
- `src/flows/`
- `src/memory-viewer/`

### Você PODE ler (não modificar)

- `src/design-system/` — `Card`, `FormField`, `Modal`, `Prose`, `Button`, `Badge`, `StatusBadge`, `Toast`
- `src/api/cao-client.ts` — métodos `listProfiles`, `installProfile`, `getProviders`, `listFlows`, `runFlow`, `getMemories`, etc.
- `src/api/key-store/use-validated-providers.ts` — gating de provider dropdowns
- `src/api/__tests__/msw/` — handlers MSW para testes
- `src/settings/`
- `src/shell/`

### Você NÃO PODE tocar

- `src/canvas-*` (Codex #1 e Gemini #1)
- `src/voice/` (Gemini #1)
- `src/dashboard/`, `src/health/` (Gemini #2)
- `src/design-system/`, `src/shell/`, `src/api/cao-client.ts`, `src/shared/` (SUP)

### Bibliotecas a usar

- **`monaco-editor`** + **`@monaco-editor/loader`** — markdown body editor para profiles e prompt template
- **`cronstrue`** — human-readable cron strings
- **`marked`** + **`dompurify`** — markdown rendering (já em deps); usar via `<Prose>` do design system
- **`@tanstack/react-query`** — listas e mutations

---

### Seção 18 — Agent Studio (8 tasks) — comece aqui

- 18.1 Lista de profiles em `/agent-studio` via `GET /agents/profiles`. Renderizar nome/role/provider/description. Search + filtros
- 18.2 Painel de provider availability via `GET /agents/providers` (CAO-managed install state)
- 18.3 Flag profiles cujo provider tem `installed: false` (badge ou dim)
- 18.4 Profile detail viewer: parsed markdown body via `<Prose>`, YAML frontmatter como key-value list, metadata
- 18.5 Profile Editor: form para frontmatter (provider dropdown gated por `useValidatedProviders()`), Monaco para markdown body, Save invoca `POST /agents/profiles/install`
- 18.6 Install From Source com 3 caminhos:
  - **Built-in store**: profiles curados shipados com AgentVerse (defina array em `src/agent-studio/built-in-profiles.ts`)
  - **Local file**: file picker para `.md` upload
  - **URL fetch**: preview-before-install com confirmação
- 18.7 Surface CAO validation errors verbatim (não traduza/parafraseia)
- 18.8 Tests: install flow end-to-end via MSW; provider gating; markdown rendering

---

### Seção 19 — Flows (7 tasks)

- 19.1 Lista em `/flows` via `GET /flows`. Use `cronstrue` para human-readable. Refresh em poll de 15s (TanStack Query `refetchInterval`)
- 19.2 Quick-pick schedule UI:
  - every-N-minutes
  - hourly
  - daily-at-time
  - weekdays-at-time
  - weekly
  
  Plus raw cron input com validação live via `cronstrue` (try/catch → mostrar erro inline)
- 19.3 Create/edit form com todos os campos `Flow`:
  - `name`
  - `schedule` (quick-pick + raw)
  - `agent_profile` (selector consumindo lista da seção 18)
  - `provider` (gated por `useValidatedProviders()`)
  - `prompt_template` (Monaco)
  - `enabled` toggle
- 19.4 Run Now: `POST /flows/{name}/run` com toast confirmação
- 19.5 Enable/Disable toggle: optimistic update com revert se falhar (use `onMutate` + `onError` do TanStack Query)
- 19.6 Gating-script display: badge "Conditional" + hover text quando `gating_script` presente no flow
- 19.7 Tests:
  - cron inválido rejeitado antes do submit
  - quick-pick preenche cron corretamente (e.g. "every 5 minutes" → `*/5 * * * *`)
  - toggle persiste após refresh

---

### Seção 20 — Memory Viewer (7 tasks)

- 20.1 Lista em `/memory` via per-terminal context API + agent-dirs setting. Empty-state claro onde direct listing não é suportado em v1
- 20.2 Filtros:
  - Scope: global / project / session / agent
  - Type: project / user / feedback / reference
  - Tags
- 20.3 Detail viewer: markdown body via `<Prose>`, scope/type metadata, tags, retention info, location path
- 20.4 Full-text search em content + tags (case-insensitive, client-side é OK em v1)
- 20.5 Manual memory creation form: validation, scope, type, tags, content
- 20.6 Retention notice para session-scoped: "Persists until session `<name>` ends"
- 20.7 Tests: filtros narrow corretamente; search case-insensitive; empty-state copy presente

---

## PADRÕES A SEGUIR

- Testes Vitest co-localizados em `__tests__/` (cobertura ≥70% em módulos com lógica)
- Use MSW handlers em `src/api/__tests__/msw/` — adicione novos handlers se faltarem
- Forms: use o pattern de `src/settings/routes.tsx` (controlled inputs, validation inline, FormField do design system)
- Listas com filtros: pattern de TanStack Query com `select` para derivar sub-listas
- Monaco lazy-load: importe via `@monaco-editor/loader` para não engordar o bundle inicial

---

## REGRAS DE EXECUÇÃO

- Não modifique diretórios fora dos seus.
- Antes de qualquer commit/push: `npm run lint && npm run typecheck && npm test`. Tudo verde, ou não comite.
- Se uma task estiver ambígua ou revelar conflito com a spec, **PAUSE e reporte**. Não chute.
- Se travar duas vezes na mesma abordagem, mude de tática.
- PRs pequenos, escopo de 1–3 tasks por PR. Título: `[ST] 18.5 — profile editor with monaco` ou `[ST] 19.2 — quick-pick schedule UI`.
- Marque o checkbox `- [ ] X.Y` → `- [x] X.Y` em `tasks.md` imediatamente após cada task.

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

Comece pela seção 18 (Agent Studio) — Flows e Memory dependem do pattern de lista+filtro+detail que você vai estabelecer ali. Verifique o código existente em `src/settings/` e `src/finops/` antes de implementar — replique o padrão.