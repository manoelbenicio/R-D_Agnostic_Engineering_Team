[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:27:25Z
TASK: GTL-08 Consolidação READ-ONLY UI Squad / Delete Runtime

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-ui-squad-runtime-integration.md

RESUMO GTL-08:
1. Issue Assignee: Schema DB pronto (084_squad.up.sql:32), Web UI pronto (assignee-picker.tsx:215), falta mobile.
2. Project Lead: Requer migration 105_project_squad_lead.up.sql alterando CHECK em project.lead_type (034_projects.up.sql:10) e ajuste em server/internal/handler/project.go:473.
3. Delete Runtime: API DELETE /api/runtimes/{id} existente. Exige ajuste no guard canDelete (runtime-list.tsx:582) e tratamento do erro HTTP 409 (FK conflict).
4. Suíte de Testes: confirm dialog, success toast, FK 409 error banner e React Query cache refetch (runtimeKeys.all).

MODO SOMENTE LEITURA MANTIDO.
