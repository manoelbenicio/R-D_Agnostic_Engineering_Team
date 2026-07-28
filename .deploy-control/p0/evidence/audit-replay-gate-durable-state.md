# Auditoria READ-ONLY: ReplayGateHook & Server Durable State

## 1. Achados da Auditoria
- **In-Memory Registry Vulnerável a Reinício**: \`ReplayGateHook\` (\`server/internal/daemon/commitledger/replay_gate.go\` L152) consulta um \`LedgerRegistry\` em memória. Reinícios do servidor/daemon descartam o histórico de intenções de efeito colateral.
- **Fail-Closed Sem Estado**: \`CheckOrAllow\` (\`replay_gate.go\` L200) e \`Check\` (\`replay_gate.go\` L173) falham fechados (\`ErrReplayBlocked\`) quando o estado do ledger está ausente. Se o estado em memória sumiu, o sistema bloqueia retentativas válidas ou, se contornado, corre risco de reexecução duplicada.
- **Telemetria NÃO é Prova de Commit**: Spans de telemetria (\`tool_result\`) capturados no CloudWatch/logs são diagnósticos não-transacionais e não constituem prova de commit no banco ou efeito final idempotente.

## 2. Correção Mínima e Segura Proposta (Sem Mudar Produção Agora)
1. **Persistência Durável em Postgres**: Gravar eventos do ledger de commit em tabela durável (\`commit_ledger_events\`) sincronizada antes e depois da execução de ferramentas com efeito colateral.
2. **Regra Estrita de Prova de Commit**: Uma task só é elegível para retry automático se o banco durável registrar explicitamente \`NoSideEffects\` ou \`DefiniteRollback\`. Qualquer estado de dúvida ou ausência de registro deve **FALHAR FECHADO** (\`ReplayBlocked\`).
3. **Telemetria Apenas para Observabilidade**: Tratar \`tool_result\` estritamente como dado de telemetria/leitura, nunca como permissão ou prova para desbloqueio de replay.

## 3. Garantias
- Zero edições de código em produção efetuadas nesta auditoria.
- Zero restarts, deploys ou reruns executados.
