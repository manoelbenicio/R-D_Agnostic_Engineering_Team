# Observability — easy deploy (stream W-OBS)

Stack **Prometheus + Grafana + Alertmanager + Postgres exporter** que monitora todos
os componentes do fluxo de credencial / cota / rotação, conforme
`docs/project/05-observability.md`. Deploy de um comando.

> Stream **W-OBS** — 100% independente do código Go. Não edita nenhum `.go`.

## Pré-requisitos
- Docker Engine 20.10+ (suporte a `host-gateway`) + Docker Compose v2.

## Subir o stack
```bash
cd deploy/observability

# 1) Segredos (uma vez) — NUNCA commite os reais (gitignored).
cp secrets/grafana_admin_password.example secrets/grafana_admin_password
cp secrets/pg_user.example                 secrets/pg_user
cp secrets/pg_pass.example                 secrets/pg_pass
# Edite os 3 arquivos com valores reais (sem newline final).

# 2) (opcional) ajustar alvos não-secretos:
cp .env.example .env        # PG_HOST / PG_DB / PROM_RETENTION

# 3) Subir o stack completo já provisionado:
docker compose up -d
```

## Portas
| Serviço           | Porta | URL                                              |
|-------------------|-------|--------------------------------------------------|
| Prometheus        | 9090  | http://localhost:9090                            |
| Grafana           | 3000  | http://localhost:3000  (admin / senha do secret) |
| Alertmanager      | 9093  | http://localhost:9093                            |
| Postgres exporter | 9187  | http://localhost:9187/metrics                    |

O Grafana sobe **já provisionado**: datasource Prometheus + 4 dashboards na pasta
"Credential Isolation" (sem clique manual).

## Segredos
Via **docker secrets** (arquivos em `./secrets/`, gitignored). Nenhum segredo em
config / env / métrica / label / log (RNF-01/02).
- `grafana_admin_password` → `GF_SECURITY_ADMIN_PASSWORD__FILE` (nativo do Grafana).
- `pg_user` / `pg_pass` → lidos pelo wrapper `pg-exporter-entrypoint.sh`, que monta o
  `DATA_SOURCE_NAME` em runtime (a imagem do postgres-exporter **não** tem `*_FILE`
  nativo). O wrapper remove CR/LF e limpa as vars do ambiente após montar o DSN.

## Alvos de scrape (`prometheus.yml`)
- **`credential-service`** → `host.docker.internal:8081` — **PLACEHOLDER**.
  As métricas do doc 05 §3 ainda **não** estão instrumentadas no Go (streams
  W-VENDORS / W-INT). O backend do produto **já** expõe `/metrics` num endereço
  separado via `METRICS_ADDR` (ver `.env.example` do produto). Ajuste o target para
  onde o `/metrics` estiver ouvindo (`host.docker.internal:<porta>` ou `backend:<porta>`
  na rede do produto).
- **`postgres`** → `postgres-exporter:9187`.  **`prometheus`** → `localhost:9090`.

### Atingir o Postgres do produto
O `postgres-exporter` aponta por padrão para `host.docker.internal:5432` (Postgres
publicado no host). Se o Postgres roda no compose do produto (`name: multica`,
serviço `postgres`), use a rede compartilhada — em `.env`: `PG_HOST=postgres:5432` —
e adicione a rede `multica_default` como `external` no `docker-compose.yml`.

## Alertas (`alerts.yml` — doc 05 §4)
| Alerta                     | Severidade |
|----------------------------|------------|
| CredentialRestoreFailing   | warning    |
| EnvInjectionFailing        | warning    |
| AllAccountsExhausted       | critical   |
| RotationFailing            | critical   |
| NoAvailableAccounts        | warning    |
| PostgresDown               | critical   |
| SecretInLogSuspected (opc.)| critical   |

Roteados pelo `alertmanager.yml` para um **webhook local placeholder**
(`http://host.docker.internal:5001/alertmanager`). Substitua por Slack / email /
PagerDuty em produção.

## Dashboards (`grafana/dashboards/`)
1. **Credential Health** — injeções OK/erro por vendor; restore success rate; tempo de preparação de home.
2. **Accounts & Quota** — status por conta; tokens usados vs janela; contas disponíveis; ETA de cooldown.
3. **Rotation (Fase 2)** — rotações por motivo; duração; timeline; "todas esgotadas" + wake ETA; false positives.
4. **Platform Health** — Postgres; build info; HTTP; goroutines; memória; agentes (up / heartbeat age).

## Cobertura obrigatória (doc 05 §3)
| Componente               | Painel           | Alerta                                         |
|--------------------------|------------------|------------------------------------------------|
| Daemon / execenv         | Credential Health| EnvInjectionFailing, CredentialRestoreFailing  |
| Store de credencial (PG) | Platform Health  | PostgresDown                                   |
| Contas / cota            | Accounts & Quota | NoAvailableAccounts                            |
| Rotação (Fase 2)         | Rotation         | RotationFailing, AllAccountsExhausted          |
| Detecção de esgotamento  | Rotation         | AllAccountsExhausted, SecretInLogSuspected     |
| Agentes                  | Platform Health  | (heartbeat age)                                |
| Host / containers        | Platform Health (`go_*`, `process_*`) | —                          |

## Validar (antes de marcar DONE)
```bash
docker compose config >/dev/null && echo COMPOSE_OK
docker run --rm --entrypoint promtool -v "$PWD":/w -w /w prom/prometheus:latest check config prometheus.yml
docker run --rm --entrypoint promtool -v "$PWD":/w -w /w prom/prometheus:latest check rules alerts.yml
docker run --rm --entrypoint amtool   -v "$PWD":/w -w /w prom/alertmanager:latest check-config alertmanager.yml
# Nota: as imagens oficiais tem ENTRYPOINT=prometheus/alertmanager; promtool e
# amtool sao binarios separados, por isso --entrypoint (sem ele: "unexpected promtool").
for f in grafana/dashboards/*.json; do python3 -m json.tool "$f" >/dev/null && echo "JSON_OK $f"; done
```

## Notas
- As métricas de credencial/cota/rotação (doc §3) são **PLACEHOLDER** até a
  instrumentação Go (outra stream); painéis e alertas já estão prontos para
  recebê-las (mostram "no data" até então). As métricas de runtime do backend
  (`multica_*`, `go_*`, `process_*`) já são reais e aparecem no **Platform Health**.
- Postgres-only (RNF-04/05). Nenhum SQLite próprio.
