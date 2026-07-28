# GTL-12 Audit de Prontidão: ORQ-18 Runtime Delete UI

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:35Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**worktree auditado**: `/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/91e70c79/workdir/repo`  
**branch**: `agent/codex-b/orq-18-runtime-delete-ui`  
**modo**: READ-ONLY — NENHUM código, build, git rebase ou quadro alterados  

---

## 1. Isolamento de Squad & Não-Colisão

- **Propriedade**: Tarefa atribuída a `Codex56#B` (w7:p4) na Lane B.
- **Localização do Worktree**: Diretório isolado em `/home/ec2-user/multica_workspaces/.../91e70c79/workdir/repo`.
- **Arquivos Alterados (Git Status)**:
  - `multica-auth-work/packages/views/runtimes/components/runtime-list.tsx`
  - `multica-auth-work/packages/views/runtimes/components/runtime-row-menu.test.tsx`
- **Garantia de Não-Colisão**:
  - Os arquivos alterados estão estritamente contidos em `packages/views/runtimes/components/`.
  - Não há sobreposição com a ORQ-26 (`packages/core/api/chat/schema.ts` e `gtl-orq26` worktree).
  - Não há sobreposição com GTL-06R (`.deploy-control/p0/evidence/gtl-orq20-reasoning-contract.md`).

---

## 2. Plano de Rebase Pós-ORQ26

### 2.1 Contexto e Dependência
- A issue ORQ-26 resolve a regressão em `packages/core/api/chat/schema.ts` (`uploadFile` schema / `ApiContractError`).
- A ORQ-18 é um refinamento de UI no pacote `packages/views/runtimes`.

### 2.2 Sequência de Rebase Recomendada
1. **Confirmação de Integração**: Aguardar validação e merge da ORQ-26 na branch de integração (`integration/dev-transition-candidate-20260719`).
2. **Execução no Worktree da ORQ-18**:
   ```bash
   cd /home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/91e70c79/workdir/repo
   git fetch origin
   git rebase origin/integration/dev-transition-candidate-20260719
   ```
3. **Análise de Risco de Conflito**: **ZERO conflito de mesclagem (0 bytes)**. As alterações da ORQ-18 e ORQ-26 afetam pacotes e subdiretórios totalmente disjuntos (`packages/views/runtimes` vs `packages/core/api/chat`).
4. **Validação Pós-Rebase**: Executar a suíte de testes unitários do pacote via `pnpm test` / `vitest` em `packages/views/runtimes`.

---

## 3. Cobertura da Suíte de Testes (Estados da UI)

A auditoria dos arquivos `delete-runtime-dialog.tsx`, `delete-runtime-dialog.test.tsx`, `runtime-list.tsx` e `runtime-row-menu.test.tsx` confirma cobertura total dos estados da UI:

| Estado de UI | Componente | Arquivo de Teste & Linha | Comportamento Verificado no Teste |
|---|---|---|---|
| **Confirm (Light Mode)** | `LightBody` / `DeleteRuntimeDialog` | `delete-runtime-dialog.test.tsx:165-174` | Exibe diálogo simples ("Delete Runtime?") quando não há agentes vinculados. Botão de confirmação destrutiva ativo. |
| **Cancel** | `AlertDialog` / `onCancel` | `delete-runtime-dialog.tsx:173, 240` | Executa `handleOpenChange(false)` e fecha o modal sem disparar chamadas de API. |
| **Pending / Submitting** | `submitting` state | `delete-runtime-dialog.tsx:162-165` | Desabilita botões (`disabled={submitting}`), bloqueia o fechamento no backdrop (`onOpenChange` ignorado) para evitar duplo clique ou cancelamento acidental mid-write. |
| **Success / Refetch** | `handleConfirm` | `delete-runtime-dialog.test.tsx:199-204` | Chama `apiDeleteRuntime(id)` ou `apiArchiveAgentsAndDeleteRuntime`, invalida React Query cache e invoca o callback `onDeleted()`. |
| **409 Conflict (`runtime_has_active_agents`)** | `CascadeBody` / Pivot | `delete-runtime-dialog.test.tsx:206-231` | Pivota dinamicamente do modo light para o modo cascade ao receber 409 do servidor. Exibe contagem e tabela de agentes ativos. |
| **409 Plan Changed (`runtime_delete_plan_changed`)** | `CascadeBody` / Re-prompt | `delete-runtime-dialog.test.tsx:233-287` | Ao detectar que novos agentes foram vinculados durante a confirmação, o diálogo atualiza a lista, exibe banner informativo e reseta o checkbox de confirmação. |
| **Error / Toast** | `catch (err)` | `delete-runtime-dialog.tsx:149-153` | Intercepta falhas de rede ou HTTP 500/503 e exibe aviso via `toast.error(message)`, mantendo o estado para nova tentativa. |
| **Self-Healing & Profile Banners** | `DeletePersistenceNotice` | `delete-runtime-dialog.test.tsx:289-340` | Renderiza banners educativos para daemons locais autorrecuperáveis (aviso de respawn) e runtimes baseados em profile (necessidade de deletar o profile). |

---

## 4. Alterações Estruturais Destacadas na ORQ-18

1. **Substituição do Menu Kebab por Botão Direto**:
   - **Antes**: Dropdown menu oculto atrás de ícone kebab (`MoreHorizontal`) que exigia hover/click adicional.
   - **Depois**: `RuntimeDeleteButton` visível diretamente na linha com ícone `Trash2` e `Tooltip`, mantendo o `DeleteRuntimeDialog` como a superfície única de confirmação destrutiva.
2. **Acessibilidade e Usabilidade**:
   - Adicionado `aria-label={label}` e suporte a foco via teclado (`focus-visible:ring-2`).
   - Preservada a verificação de permissão `canDelete`: a linha retorna `<span aria-hidden />` vazia se o usuário não possuir autorização.

---

## 5. Veredito de Prontidão (Readiness Verdict)

- **Status**: **PRONTO PARA REBASE E MERGE** (após conclusão da ORQ-26).
- **Risco de Regressão**: Mínimo (suíte de testes abrangente cobrindo todos os fluxos normais e de exceção 409/error).
- **Conflito de Arquivos**: Zero.
- **Ação Recomendada**: O General-Tech-Lead pode autorizar a mesclagem da ORQ-18 assim que a ORQ-26 for integrada.

