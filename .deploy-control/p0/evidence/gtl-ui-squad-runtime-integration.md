# Consolidação READ-ONLY GTL-08: UI Squad, Project Lead & Delete Runtime

- **Autor:** Agy-P0-A8 (wB:p2)
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T11:27:25Z
- **Modo:** SOMENTE LEITURA / INTEGRATED DESIGN (Nenhum build ou edição executada)

---

## 1. Separação Funcional das 3 Frentes

### 1.1 Squad como Issue Assignee
- **Status do Schema DB:** **Pronto**. Migration `084_squad.up.sql:32` alterou `issue_assignee_type_check` para `CHECK (assignee_type IN ('member', 'agent', 'squad'))`.
- **Contrato API Existente:** `PATCH /api/issues/{id}` com payload `{"assignee_type": "squad", "assignee_id": "<UUID>"}`.
- **Estado do Frontend Web:** **Pronto**. `packages/views/issues/components/pickers/assignee-picker.tsx` (L215-234) possui a seção `<PickerSection label="Squads">`.
- **Lacuna Mobile:** `apps/mobile/components/issue/pickers/assignee-picker-body.tsx` precisa adicionar o grupo de squads.

### 1.2 Squad como Project Lead
- **Status do Schema DB:** **Necessita Migration**. Migration `034_projects.up.sql:10` restringe `lead_type TEXT CHECK (lead_type IN ('member', 'agent'))`.
- **Migration Necessária (`105_project_squad_lead.up.sql`):**
  ```sql
  ALTER TABLE project DROP CONSTRAINT IF EXISTS project_lead_type_check;
  ALTER TABLE project ADD CONSTRAINT project_lead_type_check
      CHECK (lead_type IN ('member', 'agent', 'squad'));
  ```
- **Contrato API a Ajustar:** `POST /api/projects` e `PATCH /api/projects/{id}` em `server/internal/handler/project.go:473` (permitir `"squad"` no mapa de validação).
- **Frontend Core:** `packages/core/types/project.ts:13` (`lead_type: "member" | "agent" | "squad" | null`).

### 1.3 Delete Runtime
- **Contrato API Existente:** `DELETE /api/runtimes/{id}`. Devolve HTTP 200 em caso de sucesso, ou HTTP 409 Conflict se houver vínculo FK ativo com agents/tasks em execução.
- **Estado do Frontend Web:** `packages/views/runtimes/components/runtime-list.tsx` (L473, L582) e `runtime-detail.tsx` (L114, L485).
- **Lacuna de UI:** O guard `canDelete` oculta o menu quando o usuário não é `owner` ou `admin`. O botão deve ser exibido desabilitado com tooltip ou liberado com tratamento de erro.

---

## 2. Mapa Mínimo de Patches (Patch Map para Codex56-TL)

| Arquivo Alvo | Operação | Descrição Mínima da Alteração |
| :--- | :--- | :--- |
| `server/migrations/105_project_squad_lead.up.sql` | **Criar Migration** | Drop/Add constraint `project_lead_type_check` incluindo `'squad'`. |
| `server/internal/handler/project.go` | **Modificar Backend** | Atualizar validação da chave `lead_type` (Linha 473) para aceitar `"squad"`. |
| `packages/core/types/project.ts` | **Modificar Type** | Atualizar definição de tipo para `lead_type: "member" \| "agent" \| "squad" \| null`. |
| `packages/views/runtimes/components/runtime-list.tsx` | **Modificar UI** | Ajustar `canDelete` (Linha 582) e desabilitar botão com tooltip caso haja conflito em vez de ocultar. |
| `packages/views/runtimes/components/delete-runtime-dialog.tsx` | **Modificar UI/Error** | Adicionar tratamento do status HTTP 409 (FK Conflict / Active Tasks) com mensagem amigável. |

---

## 3. Cobertura de Testes Exigida

1. **Confirm Dialog Test:** Validar abertura do diálogo de confirmação `DeleteRuntimeDialog` ao clicar na ação.
2. **Success Toast Test:** Verificar exibição do toast de sucesso `toast.success("Runtime deleted")` após resposta HTTP 200.
3. **Error & FK 409 Conflict Test:** Simular erro HTTP 409 da API `DELETE /api/runtimes/{id}` e garantir exibição do alerta de tarefas/agentes vinculados sem crash da UI.
4. **Cache Refetch Test:** Confirmar invalidação do cache React Query `queryClient.invalidateQueries({ queryKey: runtimeKeys.all(wsId) })` recarregando a lista.
