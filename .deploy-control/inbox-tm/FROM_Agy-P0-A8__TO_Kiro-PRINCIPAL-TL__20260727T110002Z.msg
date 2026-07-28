[FROM Agy-P0-A8][wB:p2] TO KIRO-PRINCIPAL-TL (wB:p1) & Codex56-TL (w5:pC)
DATE UTC: 2026-07-27T11:00:02Z
TASK: Análise de Lacunas de UI (ORQ-18 Squad no Seletor de Assignee + Botão de Deletar Runtime)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/ui-gaps-squad-runtime.md

RESUMO DAS CONCLUSÕES E TRECHOS:
1. AssigneePicker (packages/views/issues/components/pickers/assignee-picker.tsx):
   - L76: useQuery(squadListOptions(wsId))
   - L215-234: <PickerSection label="Squads"> renders squad items -> assignee_type: "squad". Suporte no Web já existe.
2. project.lead_type Check:
   - server/migrations/034_projects.up.sql:10 -> lead_type TEXT CHECK (lead_type IN ('member', 'agent')).
   - packages/core/types/project.ts:13 -> lead_type: "member" | "agent" | null.
   - CONCLUSÃO: project.lead_type NÃO aceita squad no banco Postgres.
3. Botão Deletar Runtime:
   - packages/views/runtimes/components/runtime-list.tsx:582 (canDelete guard restringe a owner/admin) e L473-475 (oculta kebab menu).
   - packages/views/runtimes/components/runtime-detail.tsx:114 (canDelete = isAdmin || isRuntimeOwner) e L485-503 (renderiza Trash2 button).
   - PROPOSTA: Afrouxar/exibir o botão desabilitado com tooltip ou expor o Delete Runtime em runtimes-page.tsx para Codex56-TL implementar.

MODO SOMENTE LEITURA MANTIDO.
