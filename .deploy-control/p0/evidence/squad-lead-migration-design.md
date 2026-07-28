# Desenho de Migration & Arquitetura: Squad Lead & Múltiplos Líderes (`project.lead_type`)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-27T11:02:59Z
- **Escritor Único de Código:** Codex56-TL (w5:pC)
- **Modo:** SOMENTE LEITURA / ANÁLISE & DESIGN (Nenhuma alteração em banco ou código aplicada)

---

## 1. Migration Mínima no Banco (`project.lead_type`)

### Estado Atual
- **Migration `034_projects.up.sql` (Linha 10):**
  ```sql
  lead_type TEXT CHECK (lead_type IN ('member', 'agent'))
  ```
- **Limitação:** `project.lead_type` rejeita o valor `'squad'`, impedindo que uma squad seja atribuída diretamente como líder de um projeto.

### Script da Migration Mínima (`105_project_squad_lead.up.sql`)
```sql
-- Estender project.lead_type CHECK para incluir 'squad'
ALTER TABLE project DROP CONSTRAINT IF EXISTS project_lead_type_check;
ALTER TABLE project ADD CONSTRAINT project_lead_type_check
    CHECK (lead_type IN ('member', 'agent', 'squad'));
```

### Impacto da Migration
- **Backend Go:**
  - `server/internal/handler/project.go` (Linha 473): Atualizar validação de `lead_type` para aceitar `"squad"`.
  - `server/cmd/multica/cmd_issue.go` (Linha 1755): Atualizar helper `memberOrAgentKinds`.
- **Frontend TypeScript:**
  - `packages/core/types/project.ts` (Linha 13): Atualizar o tipo `lead_type: "member" | "agent" | "squad" | null;`.
- **Custo & Risco:** **Baixo**. Segue exatamente o mesmo padrão executado na migration `084_squad.up.sql` para `issue.assignee_type`.

---

## 2. Análise de Arquitetura: Múltiplos Líderes via `squad_member.role`

### Contexto Atual do Código
- **Tabela `squad` (`084_squad.up.sql`, Linha 7):**
  ```sql
  leader_id UUID NOT NULL REFERENCES agent(id) ON DELETE RESTRICT
  ```
- **Tabela `squad_member` (`084_squad.up.sql`, Linha 22):**
  Possui a coluna `role TEXT NOT NULL DEFAULT ''`. Hoje o valor `'leader'` em `squad_member` é consumido apenas para verificações de permissões e privacidade (`memberAllowedForPrivateAgent` e `canViewAgentSecrets`), e NÃO pela lógica de roteamento de tarefas.

---

## 3. Comparativo de Opções, Custos e Riscos

| Opção | Arquitetura de Liderança | Custo de Implementação | Risco Operacional / Runtime |
| :--- | :--- | :--- | :--- |
| **Opção A: Líder Escalar Único (`squad.leader_id`)** | Mantém `leader_id` escalar. Atribuir projeto à squad roteia 100% dos eventos para o agente líder primário. | **ZERO em Schema** (apenas aplicar a migration do `project.lead_type`). | **BAIXO**. Padrão determinístico já testado e em uso. Não suporta múltiplos líderes simultâneos. |
| **Opção B: Múltiplos Líderes via `squad_member.role = 'leader'`** | Torna `squad.leader_id` opcional ou deprecado. A rota busca todos os membros da squad com `role = 'leader'`. | **MÉDIO/ALTO**. Exige alterar queries SQL em `squad.sql`, `issue.sql`, `handler/squad.go` e no Autopilot. | **ALTO**. Ambiguidade de Roteamento: se 2 agentes forem líderes, acionar o projeto pode gerar corridas de tarefa (*race condition*), duplicar chamadas LLM e inflar custos. |

---

## 4. Recomendação Técnica
Para atender ao owner com segurança:
1. **Passo 1 (Imediato):** Aplicar a migration `105_project_squad_lead.up.sql` estendendo `project.lead_type` para `'squad'`. O roteamento do projeto continuará determinístico através do líder primário da squad.
2. **Passo 2 (Múltiplos Líderes):** Se múltiplos líderes forem estritamente necessários, definir regra de concorrência explícita (ex: Round-Robin vs Fallback Sequencial) antes de alterar a query de roteamento em `squad_member`.
