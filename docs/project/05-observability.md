# 05 — Observabilidade (Grafana / Prometheus — easy deploy)

**Data:** 2026-07-01
**Meta:** cada componente do fluxo de credencial/cota/rotação 100% monitorado,
com deploy fácil via docker-compose.

---

## 1. Princípios

- **Todo componente expõe `/metrics`** (formato Prometheus).
- **Deploy de um comando** (`docker compose up -d`) sobe o stack completo.
- **Sem segredo em métrica/label** (RNF-01/02): usar `account_id`/`vendor`, nunca token.
- **Três pilares**: métricas (Prometheus), dashboards (Grafana), alertas (Alertmanager).
- Logs estruturados (JSON) e, opcionalmente, traces (OpenTelemetry → Tempo).

## 2. Stack (easy deploy)

| Componente | Porta | Função |
|------------|-------|--------|
| Prometheus | 9090 | scrape de métricas + regras de alerta |
| Grafana | 3000 | dashboards |
| Alertmanager | 9093 | roteamento de alertas (webhook/email/slack) |
| node/cadvisor (opc.) | 9100/8080 | host/containers |
| Postgres exporter | 9187 | saúde do banco (RNF-04) |

## 3. Cobertura por componente (nada fica sem monitoração)

| Componente | Métricas-chave |
|------------|----------------|
| Daemon / execenv | `cred_env_injection_total{vendor,result}`, `codex_home_prepare_seconds`, `credential_restore_total{vendor,result}` |
| Store de credencial (Postgres) | `credential_store_ops_total{op,result}`, `pg_up`, `pg_stat_activity_count`, latência de query |
| Contas / cota | `account_status{vendor,account_id,status}` (available/leased/exhausted/cooldown), `account_tokens_used`, `account_window_seconds_remaining` |
| Rotação (Fase 2) | `rotation_total{vendor,reason,result}`, `rotation_duration_seconds`, `accounts_available{vendor}`, `all_accounts_exhausted{vendor}` |
| Detecção de esgotamento | `exhaustion_detected_total{vendor,signal}` (screen/429/ledger), `false_positive_total` |
| Agentes | `agent_state{agent_id,state}`, `agent_up`, heartbeat age |
| Host/containers | CPU/mem/disk, container restarts |

## 4. Alertas (regras)

| Alerta | Condição | Severidade |
|--------|----------|-----------|
| `CredentialRestoreFailing` | `rate(credential_restore_total{result="error"}[5m]) > 0` | warning |
| `EnvInjectionFailing` | `rate(cred_env_injection_total{result="error"}[5m]) > 0` | warning |
| `AllAccountsExhausted` | `all_accounts_exhausted == 1` por 1m | critical |
| `RotationFailing` | `rate(rotation_total{result="error"}[10m]) > 0` | critical |
| `NoAvailableAccounts` | `accounts_available < 1` por 5m | warning |
| `PostgresDown` | `pg_up == 0` | critical |
| `SecretInLogSuspected` | log matcher (Loki) para padrões de token | critical |

## 5. Dashboards Grafana (painéis)

1. **Credential Health** — injeções OK/erro por vendor; restore success rate;
   tempo de preparação de home.
2. **Accounts & Quota** — status de cada conta (rings), tokens usados vs janela,
   contas disponíveis por vendor, ETA de cooldown.
3. **Rotation (Fase 2)** — rotações por motivo, duração, timeline de eventos,
   "todas esgotadas" com wake ETA.
4. **Platform Health** — Postgres, host, containers, agentes (up/estado).

## 6. Deploy (docker-compose)

`deploy/observability/docker-compose.yml` (esboço):

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./alerts.yml:/etc/prometheus/alerts.yml:ro
    ports: ["9090:9090"]
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD__FILE=/run/secrets/grafana_admin
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
    ports: ["3000:3000"]
  alertmanager:
    image: prom/alertmanager:latest
    volumes: ["./alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro"]
    ports: ["9093:9093"]
  postgres-exporter:
    image: quay.io/prometheuscommunity/postgres-exporter:latest
    environment:
      - DATA_SOURCE_URI=postgres:5432/aop?sslmode=disable
      - DATA_SOURCE_USER__FILE=/run/secrets/pg_user
      - DATA_SOURCE_PASS__FILE=/run/secrets/pg_pass
    ports: ["9187:9187"]
```

`prometheus.yml` (scrape) — inclui o próprio serviço de credencial/daemon:

```yaml
global: { scrape_interval: 15s }
rule_files: [ "alerts.yml" ]
alerting:
  alertmanagers: [ { static_configs: [ { targets: ["alertmanager:9093"] } ] } ]
scrape_configs:
  - job_name: credential-service
    static_configs: [ { targets: ["host.docker.internal:8081"] } ]  # /metrics do daemon/serviço
  - job_name: postgres
    static_configs: [ { targets: ["postgres-exporter:9187"] } ]
  - job_name: prometheus
    static_configs: [ { targets: ["localhost:9090"] } ]
```

> Grafana com **provisioning** (datasource + dashboards versionados em git) → sobe
> já configurado, sem clique manual. Segredos via docker secrets / *_FILE (nunca
> hardcoded — corrige a falha do JWT secret encontrada na auditoria).

## 7. Critérios de "monitorado de verdade"

- [ ] Todo componente do fluxo expõe `/metrics`.
- [ ] Existe painel Grafana para cada componente da §3.
- [ ] Existe alerta para cada falha crítica da §4.
- [ ] `docker compose up -d` sobe o stack completo já provisionado.
- [ ] Nenhum segredo aparece em métrica, label ou log.
