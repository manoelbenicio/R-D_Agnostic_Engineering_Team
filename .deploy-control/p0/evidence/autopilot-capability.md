# Análise de Capacidade do Autopilot — Multica (autopilot-capability.md)

## 1. O que o Autopilot Faz
- O Autopilot é um mecanismo de automação agendada/disparada por evento (\`cmd_autopilot.go\` L20: *"scheduled/triggered agent automations"*).
- Ele gera e executa tarefas automaticamente instanciando um \`autopilot_run\`, criando uma nova issue a partir do modelo (\`issue_title_template\`) e despachando a execução para um agente específico ou liderança de squad (\`autopilot.go\` L35-37: \`AssigneeType: "squad"\` resolve em \`squad.leader_id\` no momento do disparo).

## 2. Formas de Disparo (Gatilhos vs Agenda)
- **Agenda (Cron)**: Disparo recorrente via expressão Cron (\`cron_expression\`, \`timezone\`) calculado em \`service.ComputeNextRun\` (\`autopilot.go\` L24).
- **Webhook Externo**: Ingress público de webhooks em \`POST /api/webhooks/autopilots/{token}\` (\`cmd/server/router.go\` L498).
- **Disparo Manual (API)**: Endpoint de disparo sob demanda em \`POST /api/autopilots/{id}/trigger\` (\`cmd/server/router.go\` L820).

## 3. Resolveria o Problema de Issue Atribuída sem Execução?
- **NÃO para Issues Manuais Existentes**: O Autopilot não monitora nem executa issues legadas estáticas atribuídas no Kanban por um usuário.
- **SIM para Fluxos Recorrentes / Disparados**: Se o fluxo for configurado como um Autopilot, o disparo cria a issue E aciona a execução do agente/squad automaticamente.

## 4. Principais Rotas da API (\`cmd/server/router.go\` L498, L813-834)
- **Criação e Gestão**: \`POST /api/autopilots\`, \`GET /api/autopilots\`, \`PATCH /api/autopilots/{id}\`
- **Gatilhos**: \`POST /api/autopilots/{id}/triggers\` (adiciona cron ou webhook), \`POST /api/autopilots/{id}/trigger\` (manual)
- **Ingress Webhook**: \`POST /api/webhooks/autopilots/{token}\`
- **Histórico e Deliveries**: \`GET /api/autopilots/{id}/runs\`, \`GET /api/autopilots/{id}/deliveries\`

## 5. Vale Configurar para as Squads do Owner? (Recomendação)
- **SIM, Vale Configurar**: Recomendado para tarefas periódicas de auditoria, testes contínuos e varredura de bugs nas duas squads reais (\`Navy_Seals\` com 10 membros e \`Kiro-Codex5.6\` com 7 membros).
- **Como Configurar**:
  1. Criar um Autopilot via \`POST /api/autopilots\` definindo \`assignee_type: "squad"\` e \`assignee_id: <ID_DA_SQUAD>\`.
  2. Adicionar gatilho Cron via \`POST /api/autopilots/{id}/triggers\` (\`kind: "schedule"\`, \`cron_expression: "0 * * * *"\`).
  3. Para eventos externos (ex: GitHub push/issue), adicionar gatilho Webhook (\`kind: "webhook"\`) e apontar a URL.
