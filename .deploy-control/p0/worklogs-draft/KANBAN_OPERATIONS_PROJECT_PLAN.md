# PLANO DE PROJETO KANBAN DE OPERAÇÕES P0 PROD (POST-AUTH DISPATCH)

- **Nome do Projeto:** `Delivery Control — Production Readiness`
- **Estado Global:** DRAFT_NOT_POSTED (Aguardando Sinalização de Autenticação por Kiro)
- **Agente Registrar:** Agy-P0-A8 (wB:p2)
- **Liderança Técnica AWS:** Kiro-Opus5
- **Autoridade de Integração:** Codex56-TL
- **Data UTC:** 2026-07-27T17:24:26Z

---

## 1. Diretiva de Execução Pós-Autenticação

Assim que Kiro emitir o sinal de acesso autenticado da aplicação (via API), o registrador criará o projeto e os 10 cards de status sem atribuição (UNASSIGNED) seguindo as regras de segurança do Owner:

1. **Criar Projeto:** `Delivery Control — Production Readiness` no workspace `20fce817-895d-447b-965a-49f5e279314a`.
2. **Criar 10 Cards de Status (Um por Painel/Agente):**
   - Atribuição: **UNASSIGNED** (`assignee_id: null`).
   - Sem `@mentions` no título ou descrição.
   - Sem disparos de tarefas pagas (Preview com array `agents: []`).
   - Entradas de status inicial via notas estritas `/note ` (`is_note=true`).
   - Validação GET-back e checagem de idempotência (`WORKLOG-V1:OPS-PROJECT:<agent-id>:<sha>`).

---

## 2. Alocação Atual e Estrutura dos 10 Cards de Status

| Card # | Painel / Agente Alvo | Título do Card | Alocação Atual / Estado | Porcentagem / Duração | Evidência / Commit Alvo | Blocker Principal | Próxima Ação & ETA |
|---|---|---|---|---|---|---|---|
| 1 | **Kiro-Opus5** | Status — Kiro-Opus5 (AWS Lead) | Governança de Infraestrutura AWS e Segredos | NOT_MEASURED | `aws-secrets-manager/SKILL.md` | Janela de Manutenção Owner | Auditoria de Segredos |
| 2 | **Codex56-TL** | Status — Codex56-TL (General-TL) | Integração Única e Aceite do Gate 0 Preflight | NOT_MEASURED | `tool-permission-preflight-report.md` | Aguardando sinal de auth | Autorizar fusão e sync |
| 3 | **Codex56-A** | Status — Codex56-A | Standby / Disponível para Engenharia | 0% (Idle) | N/A | Aguardando atribuição | Escutar fila de tarefas |
| 4 | **Codex56-B** | Status — Codex56-B | Standby / Disponível para Engenharia | 0% (Idle) | N/A | Aguardando atribuição | Escutar fila de tarefas |
| 5 | **Codex56-Z** | Status — Codex56-Z | Standby / Disponível para Engenharia | 0% (Idle) | N/A | Aguardando atribuição | Escutar fila de tarefas |
| 6 | **Opus48-A** | Status — Opus48-A | Standby / Disponível para Engenharia | 0% (Idle) | N/A | Aguardando atribuição | Escutar fila de tarefas |
| 7 | **Opus48-B** | Status — Opus48-B | Standby / Disponível para Engenharia | 0% (Idle) | N/A | Aguardando atribuição | Escutar fila de tarefas |
| 8 | **Opus48-D** | Status — Opus48-D | Transferência W4 Autopilot concluída para Agy-P0-A8 | Concluído (0%) | `orq41-w4-autopilot-rescue-implementation.md` | Exaustão de Cota (Resolvido) | Standby pós-resgate |
| 9 | **Agy-P0-A7** | Status — Agy-P0-A7 | Implementação de Testes do Consumidor ORQ-26 | 100% PASS | `gtl-orq26-consumer-tests-impl.md` | Validação E2E Browser | Conclusão ORQ-26 |
| 10 | **Agy-P0-A8** | Status — Agy-P0-A8 (Registrar) | Controle de Worklogs DRAFT_NOT_POSTED & Gate 0 | Ativo (100%) | `PROD_KANBAN_SYNC_PREPARATION.md` | Aguardando Login Kiro | Disparar Sync Pós-Sinal |

---

## 3. Garantias Finais de Segurança
- Zero requisições enviadas antes do login autenticado da aplicação.
- Zero mutação direta no banco (`psql`).
- Todos os comentários utilizam prefixo `/note ` ativando `is_note=true`.
- Zero triggers de agentes em preview (`agents: []`).
