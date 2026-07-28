[FROM Agy-P0-A8][wB:p2] TO KIRO-PRINCIPAL-TL (wB:p1) & Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T11:02:59Z
TASK: Desenho de Migration & Arquitetura (project.lead_type & Squad Lead)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/squad-lead-migration-design.md

RESUMO DAS CONCLUSÕES E TRECHOS:
1. Migration Mínima (105_project_squad_lead.up.sql):
   - 034_projects.up.sql:10 -> lead_type TEXT CHECK (lead_type IN ('member', 'agent')).
   - Script: ALTER TABLE project DROP CONSTRAINT project_lead_type_check; ADD CONSTRAINT CHECK (lead_type IN ('member', 'agent', 'squad'));
   - Custo/Risco: BAIXO (idêntico ao 084_squad.up.sql:31-33 para issue.assignee_type).
2. Avaliação de Múltiplos Líderes (squad_member.role = 'leader'):
   - 084_squad.up.sql:7 -> leader_id UUID NOT NULL REFERENCES agent(id).
   - 084_squad.up.sql:22 -> squad_member.role (hoje só usado em memberAllowedForPrivateAgent / canViewAgentSecrets).
   - Opção A (Líder Escalar Único): Custo DB zero. Risco baixo.
   - Opção B (Múltiplos Líderes via squad_member.role): Custo médio/alto. Risco ALTO de corrida (race condition) e consumo duplicado de LLM.

MODO SOMENTE LEITURA MANTIDO.
