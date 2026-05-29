# Agent Prompts — AgentVerse v1 Handoff

Este diretório contém os prompts completos para os 4 agentes paralelos do v1. Você (humano) é o **SUP** (supervisor) e coordena merges.

## Quem faz o quê

| Agente | Modelo | Arquivo | Owner | Seções tasks.md | Tasks | Diretórios |
|---|---|---|---|---|---|---|
| Codex #1 | Codex | [`codex-1-canvas-builder.md`](./codex-1-canvas-builder.md) | CV-builder | 7 + 8 | 18 | `src/canvas-builder/`, `src/canvas-templates/` |
| Gemini #1 | Gemini 3.5 high thinking | [`gemini-1-reconciler-voice.md`](./gemini-1-reconciler-voice.md) | CV-reconciler + VX | 9 + 14 | 17 | `src/canvas-reconciler/`, `src/voice/` (novos arquivos) |
| Codex #2 | Codex | [`codex-2-studio.md`](./codex-2-studio.md) | ST | 18 + 19 + 20 | 22 | `src/agent-studio/`, `src/flows/`, `src/memory-viewer/` |
| Gemini #2 | Gemini 3.5 high thinking | [`gemini-2-dashboard-quality.md`](./gemini-2-dashboard-quality.md) | DB + SUP-quality | 15 + 17 + 21 | 24 | `src/dashboard/`, `src/health/`, `tests/e2e/`, `scripts/` |
| **Você (SUP)** | humano | — | merge + close | 22 | 6 | revisão de PRs |

**Total**: 81 tasks distribuídas + 6 de fechamento.

## Por que essa distribuição

- **Codex** é mais forte em volume de UI plumbing e pattern-following (xyflow, monaco, recharts, formulários CRUD). Recebe Builder/Templates e Studio/Flows/Memory.
- **Gemini high thinking** é mais forte em raciocínio multi-arquivo e correção algorítmica. Recebe:
  - Reconciler (state machine atômico D5, diff de 5 casos D14)
  - Voice runtime commands (matcher bilíngue + fallthrough para NLU)
  - Cross-cutting quality (smoke, axe, perf, bundle — valida o v1 inteiro)

## Como entregar para cada agente

1. Abra a sessão do agente (Codex ou Gemini)
2. Cole o **conteúdo completo** do arquivo correspondente como prompt inicial
3. O agente vai ler `tasks.md`, as specs e começar pela primeira task da lista dele
4. Cada agente opera em diretórios isolados — não há conflito de merge entre eles

## Ordem temporal sugerida

```
T0 ────────────────────────────────────────────────────────►  T+5d
Codex #1   [7 Builder ───────────────────►][8 Templates ──►]
Gemini #1  [9 Reconciler (mocks) ─►][9 integrado ────►][14 Voice cmds ─►]
Codex #2   [18 Studio ──►][19 Flows ──►][20 Memory ──►]
Gemini #2  [15 Dashboard ──►][17 Health ──►][21 Quality ────────►]
SUP (você) [revisão contínua ─────────────────────────────────►][22 close]
```

### Dependências reais entre agentes

- **Gemini #1 (Reconciler)** começa com fixtures de `CanvasDocument` mockadas. Não bloqueia no Codex #1.
- **Voice 14** (dentro de Gemini #1) depende parcialmente do Reconciler para o comando `deploy`. Por isso ambos no mesmo agente — sem context-switch.
- **Codex #2** e parte de **Gemini #2** (15.1, 15.3, 15.4, 15.6, 17.x exceto 17.5) são totalmente independentes — rodam do T0 ao fim.
- **Smoke/perf (21.1, 21.4)** dependem de Builder + Reconciler entregues.
- **Wizard step 3 (17.5)** depende de Templates (Codex #1 seção 8).

## Sua responsabilidade como SUP

### Durante a execução

1. **Revisar cada PR** antes do merge:
   - Lint, typecheck, test passando (CI bloqueia merge)
   - Mudanças apenas em diretórios permitidos do owner
   - Sem novos imports cruzando capability boundaries (lint rule pega, mas confirme)
   - Checkbox em `tasks.md` marcado para a task entregue
2. **Resolver conflitos previsíveis** (3 arquivos):
   - `tasks.md` — 4 agentes marcam checkbox; merge é trivial
   - `package.json` — se algum precisar de dep nova, valide e merge primeiro
   - `src/shell/router.tsx` — rotas adicionadas; ordem sugerida: Codex #1 → Codex #2 → Gemini #2 → Gemini #1
3. **Daily check**: `git log --oneline main` e validar que cada agente está só nos diretórios permitidos.
4. **Locked-files policy**: PR tocando `src/design-system/`, `src/shell/` ou `src/api/cao-client.ts` requer sua aprovação explícita (label `sup-approved`).

### Seção 22 — v1 close (apenas você)

- 22.1 Verificar coverage ≥70% em módulos com lógica
- 22.2 MSW integration tests verdes cross-capability
- 22.3 Playwright smoke verde
- 22.4 Demo manual: 3-node canvas (Supervisor → Developer → Reviewer com handoff edges) deploya limpo; voice gera o mesmo canvas; edit-after-deploy adiciona Reviewer; Tear Down limpa. **Gravar vídeo**.
- 22.5 Tag `v1.0.0`, archive a change via `/opsx:archive`
- 22.6 Abrir follow-up changes: `validation-proxy`, `cloud-runtime-deployment`, `finops-tier2-token-parsing`

## Padrão de PR (todos os agentes)

```
Title: [<OWNER>] <section.task> — <short summary>
Body:
  - Tasks: 7.1, 7.2 (links para tasks.md)
  - Tests added: src/canvas-builder/__tests__/...
  - Verification: lint ✓ typecheck ✓ test ✓
  - Notes: <qualquer observação para o reviewer>
```

Owner tags:
- `[CV-B]` → Codex #1 (canvas builder/templates)
- `[CV-R]` → Gemini #1 (reconciler)
- `[VX]` → Gemini #1 (voice runtime)
- `[ST]` → Codex #2 (studio/flows/memory)
- `[DB]` → Gemini #2 (dashboard/health)
- `[SUP-Q]` → Gemini #2 (quality gates)
- `[SUP]` → você

## Comandos úteis para você (SUP)

```bash
# Ver status atual de tasks
grep -c '\[x\]' openspec/changes/milestone-1-canvas-deploy-run/tasks.md
grep -c '\[ \]' openspec/changes/milestone-1-canvas-deploy-run/tasks.md

# Validar diretórios tocados em um PR
git diff --name-only main..<branch> | sort -u

# Ver quem tocou o que recentemente
git log --since="1 day ago" --pretty=format:"%h %an %s" --name-only

# Rodar a suite completa antes de merge
npm run lint && npm run typecheck && npm test && npm run build && node scripts/check-bundle-size.mjs

# Smoke (lento, ~2min)
npm run test:smoke
```

## Bloqueios esperados e como resolver

| Sintoma | Causa provável | Resolução |
|---|---|---|
| Gemini #1 reconciler test falha em diff edge case | Spec ambígua | Pause, abra issue, decida com SUP |
| Codex #1 não consegue salvar canvas | Schema mismatch com canvas-document | Codex #1 reporta ao SUP; SUP atualiza `src/shared/canvas-types.ts` (gated) |
| Gemini #2 smoke trava em deploy | Builder ou Reconciler incompleto | Gemini #2 mocka via MSW até CV entregar |
| Codex #2 precisa de novo MSW handler | Endpoint não tem mock | Codex #2 adiciona em `src/api/__tests__/msw/` |
| Bundle estoura 1.5MB | Monaco eager-loaded | Lazy-load Monaco em todas as páginas que usam (Builder, Studio, Flows) |

## Arquivos de referência

- `openspec/changes/milestone-1-canvas-deploy-run/tasks.md` — fonte da verdade das tasks
- `ARCHITECTURE.md` — D1–D15 + R1–R9 resumidos
- `README.md` — setup do projeto
- `docs/cao-cors.md` — config CAO necessária
- `docs/key-storage-v1.md` — threat model do BYOK
- `eslint-rules/index.cjs` — regras de capability boundary
