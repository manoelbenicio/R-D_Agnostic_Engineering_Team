# Checklist Independente para Re-Review do Plano de Ondas V2 — ORQ-41 (READ-ONLY)

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T14:20Z  
**governança**: Kanban Issue `ORQ-41` ("Decouple Kanban metadata from paid task execution")  
**objetivo**: Preparar a checklist independente de re-review para a próxima versão (V2) do Plano de Ondas da ORQ-41.  
**modo**: READ-ONLY / PREPARAÇÃO — NENHUMA revisão executada até a entrega da V2 do plano. Zero alterações em código, banco, API ou quadros.  

---

## 1. Instrução de Pré-Execução

> ⚠️ **REGRA DE PARADA**: Este documento é uma checklist de prontidão. **NENHUMA REVISÃO DEVE SER INICIADA** até que o Plano de Ondas V2 da ORQ-41 seja formalmente entregue pelo responsável e disponibilizado para auditoria.

---

## 2. Checklist dos 8 Bloqueadores Principais (GTL-41 WAVES)

A futura revisão do Plano de Ondas V2 deverá verificar obrigatoriamente os seguintes 8 itens de bloqueio:

### 1. Ownership Único de `generated/sqlc` (Evitar Conflitos de Fusão)
- [ ] **Critério**: O plano especifica uma única lane/agente responsável por executar `sqlc generate` e commitar o código gerado em `server/pkg/db/generated/`.
- [ ] **Verificação**: Nenhuma outra wave paralela executa a geração de código SQL concomitantemente.

### 2. Reserva de Migrations no Sequenciamento Z01 (Sem Números Prematuros)
- [ ] **Critério**: Todas as migrations DDL utilizam estritamente placeholders `NEXT_CANONICAL_a` (`task_execution_request`), `NEXT_CANONICAL_b` (índice único ampliado), `NEXT_CANONICAL_c` (`task_trigger_audit`) e `NEXT_CANONICAL_d` (`issue_workflow_authorization`).
- [ ] **Verificação**: Nenhum número fixo (ex: 127 ou 128) é clamado prematuramente sem a confirmação da lane de sequenciamento Z01.

### 3. Integração dos Arquivos do Ledger & População da Secret HMAC
- [ ] **Critério**: O plano detalha os arquivos do `commitledger` (`server/internal/service/task.go`, `service/autopilot.go`) e a verificação do `CommitLedgerHMACSecret`.
- [ ] **Verificação**: O rollout do Autopilot em fail-closed está condicionado à presença da secret HMAC em produção, impedindo paralisação acidental do Autopilot.

### 4. Remoção do Efeito Colateral Silencioso no `CancelTasksForIssue`
- [ ] **Critério**: A reatribuição de issues (`assign` / `reassign`) é definida como **metadado puro** e NÃO chama `CancelTasksForIssue` por padrão.
- [ ] **Verificação**: O cancelamento de tarefas em andamento exige a flag explícita `cancel_active_task=true` ou chamada dedicada via API com RBAC e auditoria.

### 5. Existência e Fases da Feature Flag (`MULTICA_EXECUTION_TRIGGER_DECOUPLED`)
- [ ] **Critério**: A feature flag está mapeada em `server/internal/config/config.go` com suporte a 4 fases (`0: Off`, `1: Warn`, `2: On + Deprecated`, `3: On Strict`).
- [ ] **Verificação**: O desligamento da flag desativa com segurança as restrições sem quebrar os endpoints legados.

### 6. Plano de Testes (32 Testes Exigidos) e Validação de `TestMain`
- [ ] **Critério**: O plano contempla a suíte completa de 32 testes automatizados (metadata zero-cost, regressoes de child-done, autopilot fail-closed, runs explícitos e cancelamento).
- [ ] **Verificação**: Os testes de banco de dados possuem o guard de inicialização `TestMain` prevenindo **falsos-verdes** quando o Postgres estiver inacessível.

### 7. Segurança de Rollback do Índice Único 037 (`NEXT_CANONICAL_b`)
- [ ] **Critério**: A ampliação do índice único em `agent_task_queue (issue_id, agent_id)` cobrindo os 4 estados ativos (`queued`, `dispatched`, `running`, `waiting_local_directory`) possui um procedimento de rollback isolado.
- [ ] **Verificação**: O rollback recria com segurança o índice antigo de 2 estados sem corromper as tarefas em execução.

### 8. Harness Estrutural Executável (Allowlist Test)
- [ ] **Critério**: O teste `TestNoUnauthorizedEnqueueCallSites` é definido como o gate de encerramento.
- [ ] **Verificação**: O teste inspeciona a AST/grep do Go garantindo que `EnqueueTaskFor*` só exista na lista branca autorizada de 21 call sites (C1-C17 e I1-I4) e esteja sempre precedido pelo gate de validação.

---

## 3. Matriz de Mapeamento das 6 Waves Recomendadas

| Wave | Componentes do Plano V2 | Requisitos Obrigatórios de Entrega | Gate de Saída |
|---|---|---|---|
| **Wave 0** | Migrations Placeholders & Tabela de Auditoria Passiva | Placeholders Z01 (`_a`, `_c`, `_d`), `task_trigger_audit` passiva com flag `Off`. | Ephemeral DB Up/Down limpo |
| **Wave 1** | Endpoint Canônico `POST /runs` & Metadados Fail-Closed | Endpoint `POST /api/issues/{id}/runs` + desativação de enfileiramento em assign/reassign/comentários. | 9 Testes Metadata Zero-Cost PASS |
| **Wave 2** | Desacoplamento do `issue_child_done.go` | Tabela `issue_workflow_authorization` + chave determinística `(parent, child, child_done_at)`. | 5 Testes Child-Done PASS |
| **Wave 3** | Expansão do Índice Único Ativo (`NEXT_CANONICAL_b`) | Índice de 4 estados em `(issue_id, agent_id)` sem quebrar paralelismo de squads. | Testes de Paralelismo Squad PASS |
| **Wave 4** | Autopilot Fail-Closed & Secret HMAC | Integração com `commitledger.CheckOrAllow` + secret HMAC populada. | 5 Testes Autopilot Replay PASS |
| **Wave 5** | UI/CLI Alignment & Allowlist Harness | Diálogo na UI, opt-in de menção com `cost_ack`, e `TestNoUnauthorizedEnqueueCallSites`. | 32 Testes PASS + AST Gate |

---

## 4. Check-out Citing ORQ-41

- **Governança**: `ORQ-41`
- **Artefato Gerado**: `.deploy-control/p0/evidence/orq41-v2-waves-re-review-checklist.md`
- **Status**: **PREPARAÇÃO CONCLUÍDA — AGUARDANDO V2 DO PLANO** ⏳
- **Status de Mutação**: READ-ONLY. Zero alterações executadas.
