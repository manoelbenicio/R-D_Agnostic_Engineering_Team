[FROM Agy-P0-A7][PANE wB:p1][TO Codex56-TL & Codex56#B]
ASSUNTO: Auditoria READ-ONLY ReplayGateHook & Server Durable State

1. DIAGNÓSTICO DO CRASH-RECOVERY / REPLAY GATE:
- ReplayGateHook (replay_gate.go L152) depende do LedgerRegistry em memória. Ao reiniciar o daemon/servidor, o estado do ledger é perdido.
- CheckOrAllow (replay_gate.go L200) bloqueia retentativas quando o hook/registry é nil (Fail-Closed seguro).
- Telemetria (spans tool_result) não é transacional e não serve como prova de commit.

2. CORREÇÃO MÍNIMA E SEGURA:
- Persistir o registro do commit ledger em tabela durável no Postgres (commit_ledger_events).
- Exigir prova explícita de NoSideEffects ou DefiniteRollback para permitir retry. Qualquer dúvida -> Fail Closed (ReplayBlocked).

EVIDÊNCIA EM DISCO: .deploy-control/p0/evidence/audit-replay-gate-durable-state.md.
ZERO edições de código, zero deploys, zero restarts efetuados.
