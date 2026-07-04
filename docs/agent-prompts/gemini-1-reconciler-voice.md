# Agent: Gemini #1 (high thinking) — CV-reconciler + VX-runtime

You are a capability owner working on **AgentVerse v1** — a multi-agent orchestration SPA built on top of CAO (CLI Agent Orchestrator). The project is a single shared branch with parallel-from-day-zero ownership; do not create long-lived feature branches.

You were chosen for these capabilities because they require **algorithmic correctness and multi-file reasoning**, not volume of pattern-following code. Take the extra thinking budget — silent bugs here corrupt deploy state and break voice-to-action mapping.

---

## SOURCE OF TRUTH (read in this order)

1. `openspec/changes/milestone-1-canvas-deploy-run/proposal.md` — what & why
2. `openspec/changes/milestone-1-canvas-deploy-run/design.md` — D1–D15 decisions, **especially D5 (atomic deploy state) and D14 (diff-based edit-after-deploy)**
3. `openspec/changes/milestone-1-canvas-deploy-run/tasks.md` — your task list
4. `openspec/changes/milestone-1-canvas-deploy-run/specs/canvas-reconciler/spec.md` — full state machine + 5 diff cases
5. `openspec/changes/milestone-1-canvas-deploy-run/specs/voice-runtime-commands/spec.md` — bilingual command matcher
6. `ARCHITECTURE.md` — cross-cutting summary
7. `docs/patterns/` — established conventions

The OpenSpec change is the locked baseline. Do not deviate without filing a new change proposal.

---

## PROJECT CONVENTIONS (non-negotiable)

- Stack: React 18 + Vite + TypeScript strict. Zustand for UI state, TanStack Query for server state, IndexedDB via `idb`.
- Only `src/api/cao-client.ts` may call CAO endpoints. Lint rule enforces it.
- Only `src/shared/` crosses capability boundaries — supervisor-gated.
- `src/design-system/` is **locked**. Use existing components only.
- Costs through `<CostLabel />` (not your concern in these sections).

---

## SUA MISSÃO

Você é o owner **CV-reconciler + VX-runtime**. Implemente seções **9 (Canvas Reconciler)** e **14 (Voice Runtime Commands)** do `tasks.md` — total 17 tasks.

### Diretórios sob sua responsabilidade

- `src/canvas-reconciler/` — exclusivo
- `src/voice/` — **apenas arquivos novos para runtime commands**. NÃO toque em arquivos já entregues:
  - `VoicePanel.tsx`, `voice-capture.ts`, `whisper-transcriber.ts`, `voice-to-canvas.ts`, `nlu.ts`, `nlu-prompt.ts`, `engine.ts`, `store.ts`, `types.ts`, `use-voice-hotkey.ts` — todos prontos, apenas leia

### Você PODE ler (não modificar)

- `src/canvas-document/` — schema, store, validators
- `src/canvas-builder/` — quando o Codex #1 entregar (use fixtures até lá)
- `src/canvas-templates/` — quando o Codex #1 entregar
- `src/shared/canvas-types.ts`
- `src/api/cao-client.ts` — métodos `installProfile`, `createSession`, `addTerminal`, `deleteSession`, `deleteTerminal`, etc.
- `src/api/__tests__/msw/` — handlers para testes de integração
- `src/voice/` (todos os arquivos já entregues, listados acima)
- `src/design-system/`, `src/settings/`

### Você NÃO PODE tocar

- `src/canvas-builder/`, `src/canvas-templates/` (Codex #1)
- `src/dashboard/`, `src/health/` (Gemini #2)
- `src/agent-studio/`, `src/flows/`, `src/memory-viewer/` (Codex #2)
- `src/design-system/`, `src/shell/`, `src/api/cao-client.ts`, `src/shared/` (SUP)
- Qualquer arquivo já existente em `src/voice/` (todos da seção 13, prontos)

---

### Seção 9 — Canvas Reconciler (11 tasks) — CRÍTICO

Esta é a peça mais sensível do v1. Erros silenciosos aqui corrompem `deploy_state` e quebram o canvas. Pense antes de codar.

**D5 — atomic deploy state (NÃO QUEBRAR)**:
> Persist `deploy_state` BEFORE AND AFTER every CAO call.

Implica que cada chamada CAO é envelopada por dois `CanvasStore.save()`:
1. Antes da call: marcar a transição em andamento (`status: "deploying"`, `pending_step: "install_profile_X"`)
2. Após sucesso: gravar resultado (`terminal_map[node] = terminal_id`)
3. Após falha: gravar `degraded` com `failed_steps` populado

Se o reload acontecer entre 1 e 2/3, a UI deve oferecer "Resume" baseado no `pending_step`.

**D14 — diff-based edit-after-deploy (5 casos)**:

Capture um snapshot por nó no momento do deploy (`profile_snapshots[node_id] = { system_prompt, allowedTools, provider, model, ... }`). Em re-deploy, compare snapshot → estado atual:

1. **Nó adicionado** → `installProfile` + `addTerminal`
2. **Nó removido** → `deleteTerminal` + remover do `terminal_map`
3. **Profile content mudou** → `installProfile` (overwrite) + `deleteTerminal` + `addTerminal` para reiniciar com novo perfil
4. **Display-only (display_name, position)** → apenas persistir snapshot, sem chamadas CAO
5. **Edge mudou** → banner advisory: "Edge changes require Tear Down + redeploy" (a topologia vai no prompt do supervisor; mudança não tem efeito até redeploy completo)

**Mudança de entry-point é bloqueada** com diálogo: "Tear Down required to change entry-point. Continue?"

**Tasks específicas**:

- 9.1 Profile-markdown generator: por nó, YAML frontmatter (`name`, `role`, `provider`, `allowedTools`) + body com `system_prompt`
- 9.2 Supervisor-prompt augmentation: append "canvas topology" listing allowed handoff/assign/send_message targets (master spec §4.4 step 5). Documente o template em `docs/canvas-topology-prompt.md` (seção 21.8 cita isso)
- 9.3 Reconciler driver: `installProfile` → `createSession` → `addTerminal` por nó, com `CanvasStore.save()` atômico envolvendo cada call (D5)
- 9.4 State machine: `draft ↔ deploying ↔ deployed / degraded`. Cubra cada cenário do spec com unit tests usando `CaoClient` mockado
- 9.5 Deploy progress panel: lista de 5 linhas atualizada reativa conforme cada call resolve
- 9.6 Retry Failed: apenas nós ausentes do `terminal_map`
- 9.7 Tear Down: `DELETE /sessions/{name}`, reset `deploy_state` → draft. Mantenha como path separado mesmo com edit-in-place
- 9.8 Resume: detectar canvas em `deploying` no reload e oferecer retomada
- 9.9 **Diff-based edit-after-deploy** (D14, completo):
  - Capturar `profile_snapshots` no deploy
  - Implementar diff com 5 casos
  - Bloquear entry-point change com diálogo Tear Down
  - "Reconciling…" indicator + bloquear edits durante diff in-flight
- 9.10 Edge-change advisory banner: "Edge changes require Tear Down + redeploy to take effect on the supervisor"
- 9.11 Tests: happy-path 3-node deploy; partial-failure→degraded; all-fail→draft rollback; retry-from-degraded; reload-mid-deploy; diff-add-node; diff-remove-node; diff-change-profile-content; diff-display-only; diff-blocks-entry-point-change

Você pode começar 9 com fixtures de `CanvasDocument` mockadas (o tipo já existe em `src/shared/canvas-types.ts`). Não bloqueia no Codex #1.

---

### Seção 14 — Voice Runtime Commands (6 tasks) — depois do 9

Comece após a 9.4 estar verde (precisa do reconciler para o comando `deploy`).

**Padrão crítico**: matcher regex+keywords ≤ 100ms em transcript de 50 caracteres. Bilingual pt-BR + en-US. Fallthrough: transcripts não casados vão para o NLU livre (já em `src/voice/nlu.ts`), **não são descartados**.

**Tasks**:

- 14.1 `matchRuntimeCommand(transcript)` — regex+keyword matcher (master spec §5.8): `kill`, `pause`, `focus`, `status`, `deploy`, `stop_all`, `cost`, `add_node`, `connect`
- 14.2 Cobertura bilingual: pt-BR + en-US patterns
- 14.3 Latency ≤ 100ms em 50 chars (teste com `performance.now()`)
- 14.4 Wire commands → ações:
  - `kill` → `DELETE /terminals/{id}` após confirmation modal
  - `stop_all` → `DELETE /sessions/{name}` após confirmation modal
  - `pause` → `POST /terminals/{id}/input` com sentinel
  - `focus` → navegação client-side
  - `status` → lê `GET /sessions/{name}/terminals` e anuncia
  - `deploy` → invoca o **Reconciler** (sua seção 9) se canvas válido; senão toast com disabled reason
  - `cost` → navegar para `/finops`
  - `add_node` / `connect` → delegar ao Canvas Builder (Codex #1)
- 14.5 Confirmation modal para comandos destrutivos (Cancel auto-focused)
- 14.6 Test: transcripts não reconhecidos caem no NLU livre, não são silenciosamente descartados

---

## REGRAS DE EXECUÇÃO

- Não modifique diretórios fora dos seus.
- Antes de qualquer commit/push: `npm run lint && npm run typecheck && npm test`. Tudo verde, ou não comite.
- Se uma task estiver ambígua ou revelar conflito com a spec, **PAUSE e reporte**. Não chute — especialmente nos casos de diff e estados degradados.
- Se travar duas vezes na mesma abordagem, mude de tática.
- PRs pequenos, escopo de 1–3 tasks por PR. Título: `[CV-R] 9.4 — state machine deploy/degraded` ou `[VX] 14.1 — runtime command matcher`.
- Marque o checkbox `- [ ] X.Y` → `- [x] X.Y` em `tasks.md` imediatamente após cada task.

### Para o Reconciler especificamente

- **Teste cada transição com falha simulada de CAO**, não só happy path. O R5 (risk: diff-based edit under partial failure) é mitigado por testar contra `terminal_map` canônico, não GET fresco.
- **Use mocks de `CaoClient`** baseados em `src/api/__tests__/msw/`. Não chame CAO real em testes.
- **Cada `CanvasStore.save()` é uma transação IDB**. Verifique que falha de IDB não corrompe o estado em memória.

### Para Voice runtime

- **Matcher antes de NLU**. Se casar comando exato, dispara ação. Se não casar, fallback para `nlu.ts` (que pode gerar canvas, e.g. "create a research team").
- **Bilíngue não é traduzir** — é dois conjuntos de padrões coexistindo. Ex.: "kill terminal 3" e "matar terminal 3" ambos viram `{cmd: 'kill', target: '3'}`.

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
4. Cenários de teste cobertos (especialmente para o reconciler — liste cada um dos 10 do 9.11)
5. Próxima task pendente e qualquer bloqueador

Comece lendo a spec do reconciler — ela tem o state machine completo e os 5 casos de diff. Não improvise correção em estado distribuído.