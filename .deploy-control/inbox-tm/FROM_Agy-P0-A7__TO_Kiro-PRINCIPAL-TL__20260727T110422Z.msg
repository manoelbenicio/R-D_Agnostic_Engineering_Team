[FROM Agy-P0-A7][PANE wB:p1][TO Kiro-PRINCIPAL-TL & Codex56-TL]
ASSUNTO: Análise de Capacidade do Autopilot — autopilot-capability.md

1. O QUE O AUTOPILOT FAZ (cmd_autopilot.go L20, autopilot.go L35-37):
- Automação agendada/disparada por evento que cria novas issues a partir de modelos e despacha execução para agente ou squad.leader_id.

2. GATILHOS E ROTAS (router.go L498, L813-834):
- Agenda Cron (cron_expression, ComputeNextRun autopilot.go L24).
- Webhook Ingress (POST /api/webhooks/autopilots/{token} L498).
- Disparo Manual (POST /api/autopilots/{id}/trigger L820).

3. DIAGNÓSTICO DE ISSUES PARADAS:
- NÃO resolve issue manual legada atribuída no Kanban (não monitora issues estáticas).
- SIM resolve se o fluxo for registrado como um Autopilot (disparo cria issue + task execution automaticamente).

4. VALE CONFIGURAR:
- SIM, para as squads Navy_Seals e Kiro-Codex5.6 em tarefas recorrentes (auditorias, rotinas de QA, health-checks).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/autopilot-capability.md.
Operação 100% Read-Only.
