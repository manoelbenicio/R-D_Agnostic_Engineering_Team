# ÍNDICE E FILA DO REGISTRADOR DE WORKLOGS KANBAN (POLICY DO OWNER)

- **Política de Governança:** Diretiva Permanente do Owner para o Registrador Kanban.
- **Card Real ORQ-26 (Regressão dos Botões do Painel de Chat — UUID: `e966922d-c6a5-4812-87bb-8b9576ccbc60`):**
  - **Relatório Gravado:** `.deploy-control/p0/evidence/orq26-chat-panel-buttons-diagnosis-report.md` (READ-ONLY estrito).
  - **Agente Executor Único:** `Agy-P0-A8`
  - **FILES_LOCKED:** `multica-auth-work/packages/views/chat/components/chat-window.tsx`
  - **Causa Raiz Medida:** Nas linhas 694-749 de `chat-window.tsx`, os ícones (`<Plus />`, `<Minimize2 />`/`<Maximize2 />`, `<Minus />`) foram passados como filhos de `<TooltipTrigger>` fora de `render={<Button />}`. No Base UI, o `TooltipTrigger` ignora filhos externos quando `render` é utilizado, renderizando botões vazios.
  - **Plano de Correção:** Mover os ícones para dentro do elemento `<Button>` da prop `render`.
- **Checklist P1 Follow-Up do Owner (Seletor Gemini + AGY):**
  - Relatório: `.deploy-control/p0/evidence/gemini-selector-p1-followup-owner-checklist.md` (Backend 200 OK; UI manual pendente).
- **Retratação da Alegação 'Codex-A/B Duplicados' (PREMISSA FALSA):**
  - Status: RETRACTED (`agent_workspace_name_unique` aplicada na migração 046; instâncias em workspaces diferentes). Bug real `UpdateAgent` 23505 -> 409 sendo corrigido por Codex56-A.
- **Auditoria Mecânica do Lifecycle Diário ORQ2 (Pós-RCA AGY):**
  - Relatório: `.deploy-control/p0/evidence/orq2-daily-lifecycle-audit-post-rca.md` (READ-ONLY). `slot150` materializado e ativo.
- **Status dos Bundles Kanban:**
  - **Bundle V1 (`4d922b25...`)**: **RETRACTED (STALE)**
  - **Bundle V2 (`fa016cf4...`)**: **SUPERSEDED BY V3**
  - **Bundle V3 (`44c72dc4...`)**: **CORRETO E ATIVO** (`.deploy-control/p0/worklogs-draft/KIRO_KANBAN_PAYLOAD_BUNDLE_V3.json`)
- **Sinalização do Owner (Kiro Signal Dependency):** Rascunhos mantidos estritamente em `DRAFT_NOT_POSTED`. Não postar V2 nem V3 antes do login autenticado da aplicação; zero mutação direta em banco (`psql`). Não bloquear Kiro.
- **Fast Gate 0 Preflight (Kanban Status Sync):** PASS (< 2s em 2026-07-27T17:46:54Z; Go 1.26.1, gofmt, git 2.50.1, jq 1.8.1, sha256, cfn-lint 1.46.0).
- **RULING DO OWNER — Separação Coordenação / Execução & Incidente Duplicate `go build`:**
  - O General-TL (`Codex56-TL`) atua exclusivamente em orientação (steering), aceite de Gate 0 e monitoramento. O agente designado é o ÚNICO escritor e executor do seu alvo.
- **RULING DO OWNER — `NO_COST_BLOCK`:** Custo/tokens **NÃO SÃO BLOQUEADOR**. Medir custo **DEPOIS**; nunca bloquear entrega por custo.
- **Papéis de Liderança:** AWS Lead: `Kiro-Opus5` | General-TL: `Codex56-TL`.

---

## Classificação de Pistas de Execução Operacional (Lanes Framework)

### 1. FAST LANE (Operações Limitadas, Reversíveis e de Baixo Risco)
- **Escopo:** Leitura/diagnóstico, testes herméticos em worktrees isolados, documentação local, correções de sintaxe/gofmt em arquivos próprios.
- **Teto de Preflight:** Máximo de 5 minutos de análise prévia (após aprovação do Gate 0).
- **Sequência Obrigatória (5 Passos):** Target identity -> Current state -> Exact command -> Execute -> Post-validate.

### 2. STRICT LANE (Operações de Alto Risco / Irreversíveis)
- **Escopo:** Rotação/acesso a segredos, alterações de autenticação, migrações de esquema/banco, fusões (merges) no branch principal, operações destrutivas ou sem rollback.
- **Requisitos:** Preflight formal exaustivo, peer review adversarial aprovado (PASS), autorização explícita do owner/General-TL e plano de rollback testado antes da aplicação.

---

- **Estado Global de Worklogs:** `DRAFT_NOT_POSTED` (Retido localmente até sinalização de autenticação normal por Kiro).
- **Proibição Absoluta:** ZERO acesso direto a banco de dados (`psql`), bypass de auth ou mutações não autenticadas no board.
- **Plano de Projeto de Operações:** `.deploy-control/p0/worklogs-draft/KANBAN_OPERATIONS_PROJECT_PLAN.md`
- **Plano de Preparação de Sincronização:** `.deploy-control/p0/worklogs-draft/PROD_KANBAN_SYNC_PREPARATION.md`
- **Relatório de Gate 0:** `.deploy-control/p0/evidence/tool-permission-preflight-report.md`
- **Diagnóstico ORQ-26 Botões Chat:** `.deploy-control/p0/evidence/orq26-chat-panel-buttons-diagnosis-report.md`
- **Checklist Follow-Up P1 Gemini:** `.deploy-control/p0/evidence/gemini-selector-p1-followup-owner-checklist.md`
- **Pacote de Payloads V3 Kiro:** `.deploy-control/p0/worklogs-draft/KIRO_KANBAN_PAYLOAD_BUNDLE_V3.json`
- **Data UTC:** 2026-07-28T16:09:45Z
- **Agente Registrar:** Agy-P0-A8 (wB:p2)

---

## Fila Sequencial de Rascunhos Registrados (10 Cards - Bundle V3)

| Ordem | Card | UUID do Card | Status Atual V3 | Arquivo de Rascunho Local | Idempotency Marker V3 |
|---|---|---|---|---|---|
| 1 | **ORQ-13** | `2aefcb3d-97b3-4e70-a4b0-ae3729b7981d` | CODE_PASS / EPHEMERAL_DB_RUNNING | `ORQ-13.md` | `WORKLOG-V3:ORQ-13:code-pass-ephemeral-db-running` |
| 2 | **ORQ-17** | `7d873133-16d5-42c6-8595-629d6fb16251` | LIVE / PASS (HTTPS + login ok) | `ORQ-17.md` | `WORKLOG-V3:ORQ-17:live-pass-https-login` |
| 3 | **ORQ-26** | `e966922d-c6a5-4812-87bb-8b9576ccbc60` | DIAGNOSED (FILES_LOCKED confirmed) | `ORQ-26.md` | `WORKLOG-V3:ORQ-26:chat-panel-buttons-diagnosed` |
| 4 | **ORQ-32** | `b01925fe-e914-422a-812e-f63cada274dc` | CLOSED (não-defeito) | `ORQ-32.md` | `WORKLOG-V3:ORQ-32:closed-non-defect` |
| 5 | **ORQ-38** | `fd5c4d55-8ce5-412f-9f15-db666831bfdb` | PASS (SHA `0ecc6f4e...abd2`) | `ORQ-38.md` | `WORKLOG-V3:ORQ-38:closeout-pass-0ecc6f4e` |
| 6 | **ORQ-39** | `c03941bc-3bde-4de1-ab19-1ba93de0ad51` | ACTIVE (correção em andamento) | `ORQ-39.md` | `WORKLOG-V3:ORQ-39:active-remediation-v3` |
| 7 | **ORQ-41** | `666f1ead-7fe9-4051-bab9-5d0a936c4701` | CODE_PASS / FLAG_OFF (`c047c0b`) | `ORQ-41.md` | `WORKLOG-V3:ORQ-41:w4-pass-flag-off-c047c0b` |
| 8 | **ORQ-42** | `64bfcae0-b867-4812-ad33-ae03ef7f25ae` | V5_DESIGN_PASS (janela owner) | `ORQ-42.md` | `df598420703cfaeb5fb80d8bdfb7db18dc7dd944ec39fe705b6c8230dede3913` |
| 9 | **ORQ-43** | `d1149dd3-9da8-4678-a4b7-d98f3eddca14` | ACTIVE (ORQ-43A hmac) | `ORQ-43.md` | `WORKLOG-V3:ORQ-43:orq43a-implementation-active` |
| 10 | **ORQ-44** | `abd12d6a-16a5-439b-bc52-74ec4bb6b231` | DESIGN_PASS (OmniRoute key) | `ORQ-44.md` | `WORKLOG-V3:ORQ-44:design-pass-omniroute` |

---

## Garantias de Não-Incorrência
- Todos os rascunhos possuem prefixo `/note ` garantindo que nenhuma tarefa paga seja disparada por comentários.
- Nenhuma alteração em atribuição (`assignee`), status, título, descrição ou tarefas associadas.
- Nenhuma chave/segredo ou corpo bruto em logs/argv.
