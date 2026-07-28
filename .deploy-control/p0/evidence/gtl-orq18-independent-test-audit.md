# GTL-42 — Auditoria independente de teste, ORQ-18 (READ-ONLY) — VEREDITO: **BLOCK**

Auditor: Codex56#A (ORQ2, pane `w7:p3`) · UTC 2026-07-27T11:52Z
Worktree auditado: `/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/91e70c79/workdir/repo`
branch `agent/codex-b/orq-18-runtime-delete-ui`, HEAD `0cb8aeb` (mesmo HEAD da base; **zero commits**, mudanca so em working tree).
Nao editei, nao rebasei, nao commitei. Executei testes (leitura + execucao), sem alterar arquivo do worktree.

## 1. O que a mudanca e (2 arquivos, `git status --porcelain`)

```text
 M multica-auth-work/packages/views/runtimes/components/runtime-list.tsx
 M multica-auth-work/packages/views/runtimes/components/runtime-row-menu.test.tsx
 (66 +/- e 46 +/-, 48 insertions, 64 deletions)
```

Troca o kebab por botao de delete visivel: `RuntimeRowMenu` → `RuntimeDeleteButton`, `DropdownMenu*`
→ `Tooltip/TooltipTrigger`, `--rtc-kebab` → `--rtc-actions`, e no consumidor (`runtime-list.tsx`,
bloco do `RuntimeList`) o render passa a `<RuntimeDeleteButton .../>`. `delete-runtime-dialog.tsx`
**nao foi tocado**.

## 2. Primeiro achado: o harness usado importa, e invalida PASS/FAIL de relatorio

`multica-auth-work/packages/views/vitest.config.ts:7-9`:

```text
7:    environment: "jsdom",
9:    setupFiles: ["./test/setup.ts"],
```

`multica-auth-work/package.json:24` -> `"test": "turbo test --filter=!@multica/mobile"`;
`packages/views/package.json:9` -> `"test": "vitest run"`.

Rodando **da raiz** `multica-auth-work` (sem o config de `packages/views`, portanto sem `setup.ts`):

```text
worktree ORQ-18 : Test Files 2 failed (2) | Tests 16 failed | 1 passed (17)
base (main repo): Test Files 2 failed (2) | Tests 16 failed | 1 passed (17)
```

Identico nas duas arvores ⇒ essas 16 falhas sao **do harness errado**, nao da mudanca. Qualquer
PASS ou BLOCK obtido dessa forma nao prova nada. Refiz tudo com o harness sancionado
(`cd packages/views && npx vitest run <arquivo>`).

## 3. Achado que sustenta o BLOCK: o teste modificado NAO TERMINA

Harness sancionado, um arquivo por execucao, `timeout 150`:

```text
# BASE (main repo, arquivo original)
$ cd multica-auth-work/packages/views && timeout 150 npx vitest run runtimes/components/runtime-row-menu.test.tsx
BASE_EXIT=0
 ✓ runtimes/components/runtime-row-menu.test.tsx (7 tests) 111ms
 Test Files  1 passed (1)
      Tests  7 passed (7)

# WORKTREE ORQ-18 (mesmo arquivo, versao modificada)
$ cd <worktree>/multica-auth-work/packages/views && timeout 150 npx vitest run runtimes/components/runtime-row-menu.test.tsx
EXIT=124        # morto pelo timeout, ZERO linha de resultado emitida
```

Reproduzido duas vezes (isolado com `timeout 150` e no par de arquivos com `timeout 420`, tambem
`EXIT=124`). Base: **7 passed em 111 ms**. Worktree: **nao emite resultado e nao termina**.

`delete-runtime-dialog.test.tsx` (nao modificado por este worktree) passa nas duas arvores:

```text
 ✓ runtimes/components/delete-runtime-dialog.test.tsx (10 tests) 1079ms
 Test Files  1 passed (1)      Tests  10 passed (10)      EXIT=0
```

Ou seja: o unico arquivo que trava e exatamente o que a ORQ-18 alterou. Hipotese mais provavel (nao
provada, porque o processo nao emite saida): o novo caso
`runtime-row-menu.test.tsx` faz `fireEvent.click(deleteButton)` e depois
`screen.getByRole("heading", { name: "Delete Runtime?" })`, isto e, **passa a montar o
`DeleteRuntimeDialog` de verdade** dentro de um arquivo cujos mocks foram escritos para
`open=false` — o proprio comentario do arquivo dizia "The dialog never renders in these tests
(`open=false` throughout)". Os mocks presentes (`@multica/core/runtimes/mutations` com
`isPending: false`, `@multica/core/api`, `sonner`, `./shared`) cobrem inicializacao de modulo, nao um
dialog realmente aberto com portal/tooltip.

**Reconciliacao GTL-12 PASS vs Codex56#B BLOCK: o BLOCK esta correto.** O PASS nao e reproduzivel com
o harness sancionado; com o harness da raiz ele tambem nao se sustenta (16 falhas iguais na base).

## 4. Cobertura real dos 7 comportamentos pedidos (com linha)

| comportamento | coberto? | evidencia (arquivo:linha) |
|---|---|---|
| confirm | **parcial** | `delete-runtime-dialog.test.tsx:213` (cascade com destrutivo gated), `:224-232` (`confirm.disabled` true→false apos checkbox), `:409` (confirm prossegue em runtime self-healing). O caso novo do worktree que clica e abre o dialog existe mas **esta no arquivo que trava** |
| cancel | **NAO** | fonte tem `onCancel` em `delete-runtime-dialog.tsx:213,220,270,275,298,301,336,348,412,415` e guarda de fechamento em `:182-185`; **nenhum teste** exercita cancelar (grep de `Cancel|cancel` nos `*.test.tsx` de `runtimes/` retorna zero ocorrencia de asserção) |
| success | **NAO** | `toast` mockado em `delete-runtime-dialog.test.tsx:106` (`{ error: vi.fn(), success: vi.fn() }`) e **nunca asseverado**; nenhum teste verifica fechamento do dialog apos sucesso |
| 409 | **SIM** | `delete-runtime-dialog.test.tsx:248` + `ApiError(...,409,...)` em `:251` (`runtime_has_active_agents`, pivot light→cascade) e `:276` + `:278` (`runtime_delete_plan_changed`, re-prompt) |
| generic error | **NAO** | fonte emite `toast.error(message)` em `delete-runtime-dialog.tsx:175` e comenta o fallback em `:599` ("callers can fall through to the generic error toast"); nenhum teste com erro nao-409 |
| refetch | **NAO** | as invalidacoes vivem em `packages/core/runtimes/mutations.ts:12` (`runtimeKeys.all`), `:38-40` (`runtimeKeys.all`, `workspaceKeys.agents`, `agentTaskSnapshotKeys.all`) e `:58`; `packages/core/runtimes` tem apenas `cli-version.test.ts`, `derive-health.test.ts` e `models.test.tsx` — **nao existe teste de `mutations.ts`**, e nos testes de view o hook e mockado |
| pending | **NAO** | mock fixa `isPending: false` em `runtime-row-menu.test.tsx` (bloco `vi.mock("@multica/core/runtimes/mutations")`) e em `delete-runtime-dialog.test.tsx:54,59`; nenhum teste com `isPending: true` (botao desabilitado/spinner durante a escrita) |

Resumo: **1 de 7 plenamente coberto (409)**, confirm parcial, e 5 sem cobertura. O botao ficou **mais
facil de acionar** (era hover-only atras de kebab, agora e um alvo destrutivo sempre visivel) sem
nenhum teste novo para cancel, success, erro generico, pending ou refetch.

## 5. Risco ON DELETE CASCADE / historico (evidencia de schema)

```text
server/migrations/004_agent_runtime_loop.up.sql:73
        FOREIGN KEY (runtime_id) REFERENCES agent_runtime(id) ON DELETE RESTRICT;   -- tabela agent
server/migrations/004_agent_runtime_loop.up.sql:86
        FOREIGN KEY (runtime_id) REFERENCES agent_runtime(id) ON DELETE CASCADE;    -- agent_task_queue
server/migrations/013_runtime_usage.up.sql:3
    runtime_id UUID NOT NULL REFERENCES agent_runtime(id) ON DELETE CASCADE,        -- runtime_usage (dropada em 046)
server/migrations/060_chat_session_runtime_id.up.sql:2
ADD COLUMN runtime_id UUID REFERENCES agent_runtime(id) ON DELETE SET NULL;         -- chat_session
```

Leitura: `agent.runtime_id` e **RESTRICT** — e o que produz o 409 quando ha agente vinculado, e por
isso o caminho 409 e o unico bem testado. Mas `agent_task_queue.runtime_id` e **CASCADE**: apagar um
runtime remove em silencio as linhas de fila daquele runtime. `chat_session.runtime_id` vira NULL.
O dialog exibe contadores do plano (o teste usa `tasks_cancelled: 0` em `delete-runtime-dialog.test.tsx:237`),
porem **nenhum teste cobre `tasks_cancelled > 0`** nem afirma o que acontece com historico. Com a acao
agora exposta permanentemente na lista, o risco de perda de fila por clique acidental sobe **sem**
teste que fixe o comportamento.

## 6. Veredito e o que desbloquearia

**BLOCK.** Motivo unico e suficiente: `runtime-row-menu.test.tsx` do worktree **nao termina** sob o
harness sancionado (EXIT=124, zero resultado), enquanto a base passa 7/7 em 111 ms. Nao ha como
declarar PASS sobre uma suite que nao conclui.

Para desbloquear (nao executei nada disto): (a) fazer o novo caso nao montar o dialog real, ou mockar
`DeleteRuntimeDialog`/`Tooltip` neste arquivo, e provar `Test Files 1 passed` com tempo comparavel ao
da base; (b) adicionar os 5 comportamentos ausentes — cancel, success, erro generico, pending
(`isPending: true`) e refetch (teste de `core/runtimes/mutations.ts` assertando as 4 invalidacoes das
linhas 12 e 38-40); (c) um caso com `tasks_cancelled > 0` explicitando a perda em `agent_task_queue`
(CASCADE, `004:86`).

## 7. Nao-alegacoes

- Nao editei, nao rebasei, nao commitei, nao instalei dependencia e nao alterei config; so executei
  `vitest run` (leitura) e comandos git/grep read-only.
- **Nao provei a causa** do travamento: o processo nao emitiu saida antes de ser morto. O que provei e
  o fato binario (base termina, worktree nao) e a mudanca de premissa do arquivo (`open=false` deixou
  de valer).
- Nao rodei a suite inteira de `packages/views` nem o `turbo test` completo; escopo foram os dois
  arquivos do dominio de delete de runtime.
- Nao validei o backend do delete (handler Go), nem executei DELETE real contra banco; a analise de
  cascade e por schema (migrations), nao por experimento.
- Nao avaliei o relatorio GTL-12 nem o do Codex56#B como fonte: reconciliei apenas por execucao propria.
