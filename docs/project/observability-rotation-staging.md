# Observability Rotation Staging

Data: 2026-07-02 UTC.

Escopo: observabilidade da rotação de contas em staging. Este documento registra somente evidências capturadas por comandos e citações de código. Não houve alteração de código de produção.

## Métricas Reais

| Métrica | Tipo | Labels | Significado | Emissor |
|---|---|---|---|---|
| `rotation_total` | CounterVec | `vendor`, `reason`, `result` | Total de tentativas de rotação por vendor, motivo e resultado. | Definida em `server/internal/metrics/credential_metrics.go:56`; labels em `server/internal/metrics/credential_metrics.go:59`; incrementada por `ObserveRotation` em `server/internal/metrics/credential_metrics.go:157`; chamada em erro por `server/internal/daemon/daemon.go:3962` e em sucesso por `server/internal/daemon/daemon.go:3969`. |
| `rotation_duration_seconds` | HistogramVec | `vendor` | Duração da rotação de conta em segundos por vendor. | Definida em `server/internal/metrics/credential_metrics.go:60`; labels em `server/internal/metrics/credential_metrics.go:64`; observada por `ObserveRotation` em `server/internal/metrics/credential_metrics.go:163`; chamada em erro por `server/internal/daemon/daemon.go:3962` e em sucesso por `server/internal/daemon/daemon.go:3969`. |
| `all_accounts_exhausted` | GaugeVec | `vendor` | Indica se todos os accounts do vendor estão esgotados. | Definida em `server/internal/metrics/credential_metrics.go:52`; labels em `server/internal/metrics/credential_metrics.go:55`; setada por `SetAllAccountsExhausted` em `server/internal/metrics/credential_metrics.go:146`; chamada com `true` por `server/internal/daemon/daemon.go:3956` e com `false` por `server/internal/daemon/daemon.go:3968`. |
| `exhaustion_detected_total` | CounterVec | `vendor`, `signal` | Total de detecções de exaustão por vendor e sinal. | Definida em `server/internal/metrics/credential_metrics.go:65`; labels em `server/internal/metrics/credential_metrics.go:68`; incrementada por `ObserveExhaustionDetected` em `server/internal/metrics/credential_metrics.go:167`; chamada no caminho reativo por `server/internal/daemon/daemon.go:3940`. |
| `accounts_available` | GaugeVec | `vendor` | Quantidade atual de accounts disponíveis por vendor. | Definida em `server/internal/metrics/credential_metrics.go:48`; labels em `server/internal/metrics/credential_metrics.go:51`; setada por `SetAccountsAvailable` em `server/internal/metrics/credential_metrics.go:139`. |
| `account_status` | GaugeVec | `vendor`, `account_id`, `status` | Marcador de status atual por account. | Definida em `server/internal/metrics/credential_metrics.go:36`; labels em `server/internal/metrics/credential_metrics.go:39`; setada por `SetAccountStatus` em `server/internal/metrics/credential_metrics.go:118`. |
| `account_tokens_used` | GaugeVec | `vendor`, `account_id` | Uso atual de tokens por account. | Definida em `server/internal/metrics/credential_metrics.go:40`; labels em `server/internal/metrics/credential_metrics.go:43`; setada por `SetAccountTokensUsed` em `server/internal/metrics/credential_metrics.go:125`. |
| `account_window_seconds_remaining` | GaugeVec | `vendor`, `account_id` | Segundos restantes da janela de quota por account. | Definida em `server/internal/metrics/credential_metrics.go:44`; labels em `server/internal/metrics/credential_metrics.go:47`; setada por `SetAccountWindowSecondsRemaining` em `server/internal/metrics/credential_metrics.go:132`. |
| `credential_restore_total` | CounterVec | `vendor`, `result` | Total de tentativas de restore de credencial por vendor e resultado. | Definida em `server/internal/metrics/credential_metrics.go:23`; labels em `server/internal/metrics/credential_metrics.go:26`; incrementada por `ObserveRestore` em `server/internal/metrics/credential_metrics.go:97`. |
| `cred_env_injection_total` | CounterVec | `vendor`, `result` | Total de tentativas de injeção de ambiente de credencial por vendor e resultado. | Definida em `server/internal/metrics/credential_metrics.go:27`; labels em `server/internal/metrics/credential_metrics.go:30`; incrementada por `ObserveEnvInjection` em `server/internal/metrics/credential_metrics.go:104`. |
| `credential_prepare_seconds` | HistogramVec | `vendor` | Duração de preparação de credencial em segundos por vendor. | Definida em `server/internal/metrics/credential_metrics.go:31`; labels em `server/internal/metrics/credential_metrics.go:35`; observada por `ObservePrepare` em `server/internal/metrics/credential_metrics.go:111`. |

Motivo da rotação proativa: `quota_forecast_proactive`, definido em `server/internal/rotation/contract.go:43`. O log estruturado do sinal proativo é emitido por `rotateTaskProactively` em `server/internal/daemon/daemon.go:3923`.

## Comandos Operacionais

Gerar rotação staging pela rede compose:

```sh
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm --network multica_default -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://multica:multica@postgres:5432/multica?sslmode=disable" \
  golang:1.26-alpine sh -c "apk add --no-cache git >/dev/null 2>&1; adduser -D t >/dev/null; \
    mkdir -p /tmp/gc /tmp/gm; chown -R t /tmp/gc /tmp/gm; \
    su t -c 'GOCACHE=/tmp/gc GOMODCACHE=/tmp/gm go test -tags staging ./internal/daemon/ -run StagingRotation -v'" 2>&1 | tail -8
```

Provar rotação no banco:

```sh
docker exec -i multica-postgres-1 psql -U multica -d multica -c \
  "SELECT reason, count(*) FROM rotation_events WHERE reason='quota_forecast_proactive' GROUP BY reason;"
```

Capturar logs de rotação no backend:

```sh
docker logs multica-backend-1 2>&1 | grep -i "rotation:" | tail -5
```

Verificar se endpoint Prometheus está ligado:

```sh
docker exec multica-backend-1 sh -c 'echo METRICS_ADDR=$METRICS_ADDR'
```

Se `METRICS_ADDR` estiver ligado, coletar métricas de rotação:

```sh
docker exec multica-backend-1 sh -c 'wget -qO- 127.0.0.1:9090/metrics | grep -E "rotation_total|all_accounts_exhausted|exhaustion_detected_total"'
```

## Evidências Capturadas

### Passo 1 - gerar uma rotação real

Comando:

```sh
docker run --rm --network multica_default -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://multica:multica@postgres:5432/multica?sslmode=disable" \
  golang:1.26-alpine sh -c "apk add --no-cache git >/dev/null 2>&1; adduser -D t >/dev/null; \
    mkdir -p /tmp/gc /tmp/gm; chown -R t /tmp/gc /tmp/gm; \
    su t -c 'GOCACHE=/tmp/gc GOMODCACHE=/tmp/gm go test -tags staging ./internal/daemon/ -run StagingRotation -v'" 2>&1 | tail -8
```

Saída:

```text
go: downloading go.uber.org/atomic v1.11.0
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7
go: downloading github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21
go: downloading golang.org/x/sys v0.35.0
=== RUN   TestStagingRotationProactiveBannerRotatesOnce
--- PASS: TestStagingRotationProactiveBannerRotatesOnce (0.09s)
PASS
ok  	github.com/multica-ai/multica/server/internal/daemon	0.129s
```

Interpretação: o teste staging passou e, pelo seu contrato de teste, executou o caminho `maybeProactiveRotateOnText` com banner Codex real e validou rotação proativa.

### Passo 2 - provar rotação no banco

Comando:

```sh
docker exec -i multica-postgres-1 psql -U multica -d multica -c \
  "SELECT reason, count(*) FROM rotation_events WHERE reason='quota_forecast_proactive' GROUP BY reason;"
```

Saída:

```text
 reason | count 
--------+-------
(0 rows)
```

Interpretação: `rotation_events` está vazio para `quota_forecast_proactive` depois do teste porque o teste staging executa cleanup ao final. A linha proativa é validada dentro do teste antes do cleanup, mas a consulta externa posterior não retém a linha.

### Passo 3 - log real de sinal proativo

Comando:

```sh
docker logs multica-backend-1 2>&1 | grep -i "rotation:" | tail -5
```

Saída:

```text
NÃO CAPTURADO
```

Interpretação: o backend `multica-backend-1` não emitiu linha `rotation:` para essa execução. O teste staging roda o daemon em processo próprio no container Go, não no backend. A saída `-v` do Passo 1 também não contém a linha `rotation: proactive quota signal detected`.

### Passo 4 - métricas

Comando:

```sh
docker exec multica-backend-1 sh -c 'echo METRICS_ADDR=$METRICS_ADDR'
```

Saída:

```text
METRICS_ADDR=
```

Estado atual: `/metrics` está OFF por padrão no backend atual porque `METRICS_ADDR` está vazio.

Como ligar em staging: configurar `METRICS_ADDR=127.0.0.1:9090` no ambiente do backend e reiniciar o backend. Esta tarefa não alterou produção nem reiniciou containers.

Coleta Prometheus:

```text
NÃO CAPTURADO
```

Motivo: `METRICS_ADDR` estava vazio.
