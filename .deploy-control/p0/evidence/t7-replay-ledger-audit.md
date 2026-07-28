# Auditoria e Arquitetura Segura do Commit Ledger — T7 Replay Ledger Audit

## 1. Estado Atual
- ReplayGateHook (server/internal/daemon/commitledger/replay_gate.go L152) e LedgerRegistry mantêm o histórico de intenções de efeito colateral em memória no processo do servidor/daemon.
- Em server/internal/service/task.go L1666, MaybeRetryFailedTask consulta commitledger.CheckOrAllow(s.ReplayGateHook, parentTaskID).
- Se o servidor reiniciar ou se o ReplayGateHook for nil, o sistema falha fechado (replay_gate.go L201), bloqueando o retry. A telemetria (tool_result) é apenas leitura/observabilidade e não constitui prova durável de commit.

## 2. Menor Schema / API Segura
Tabela SQL em Postgres (commit_ledger_events):
```sql
CREATE TABLE commit_ledger_events (
    id BIGSERIAL PRIMARY KEY,
    correlation_id TEXT NOT NULL,
    task_id UUID NOT NULL,
    event_kind TEXT NOT NULL, -- "tool_intent", "tool_commit", "tool_ambiguous"
    tool_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_commit_ledger_correlation ON commit_ledger_events(correlation_id, created_at);
```

API Go Durável:
```go
type ReplayGateStore interface {
    GetReplaySummary(ctx context.Context, correlationID string) (ReplaySummary, error)
    RecordEvent(ctx context.Context, event CommitLedgerEvent) error
}
```

## 3. Invariantes Fail-Closed
1. **Replay Permitido**: Apenas quando o histórico durável em Postgres atestar explicitamente NoSideEffects ou DefiniteRollback.
2. **Replay Bloqueado (Fail-Closed)**: Qualquer presença de tool_intent / tool_ambiguous sem commit confirmado, ou a ausência/corrupção de registro no DB, bloqueia o retry (ErrReplayBlocked).
3. **Telemetria Isolada**: Spans de telemetria em logs NUNCA são usados como autorização para permitir replay.

## 4. Persistência pelo Daemon (Pré e Pós Tool Use)
- **Pré-Execução (Antes)**: O daemon envia requisição HTTP/DB síncrona (ReportToolIntent) registrando o evento tool_intent no Postgres ANTES de acionar a ferramenta com efeito colateral.
- **Pós-Execução (Depois)**: O daemon grava tool_commit se o resultado foi confirmado de ponta a ponta, ou a entrada permanece como tool_ambiguous se o processo cair mid-flight.

## 5. Regra de Retenção (TTL 24h)
- Registros em commit_ledger_events são limpos automaticamente por pg_cron ou worker de expurgo para entradas com idade superior a 24 horas (created_at < NOW() - INTERVAL 24 hours), evitando acúmulo de churn no Postgres.

## 6. Impacto no MaybeRetryFailedTask
- Em task.go L1666, MaybeRetryFailedTask continuará consultando CheckOrAllow.
- Com o estado durável em Postgres, tarefas sem efeito colateral que falharem durante quedas do servidor poderão ser reexecutadas com segurança após o restart, enquanto tarefas com efeitos ambíguos serão bloqueadas com 100% de garantia contra efeitos duplicados.
