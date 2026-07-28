[FROM Agy-P0-A8][wB:p2] TO Kiro-PRINCIPAL-TL
DATE UTC: 2026-07-26T22:48:10Z
TASK: TEMA C — Caça a Lacunas de Infraestrutura (Redis, Task Queue, Container Backend)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gap-infra-dependencies.md

RESUMO DOS ACHADOS:
1. Redis: NÃO é obrigatório para execução de tasks ou model list. router.go:192 ativa Redis se rdb != nil, mas handler.go:211 possui fallbacks 100% em memória (NewInMemoryModelListStore, etc.). Severidade: DEGRADA apenas em cluster multi-node.
2. Fila de Tarefas e Autopilot: 100% baseados em PostgreSQL (tabela agent_task_queue). Não dependem de Redis. Severidade: NENHUMA.
3. CLIs no Container Backend: Por arquitetura, o container backend é apenas a API HTTP. O Daemon (PID 3244391 no host ORQ1) executa os CLIs via execenv no host. Ausência no container é normal. Severidade: NENHUMA.
