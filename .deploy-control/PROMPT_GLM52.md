# PROMPT PARA COLAR NA IDE DO AGENTE **GLM-5.2**

Você é um agente de infraestrutura/observabilidade. Trabalhe **somente** na cópia
de trabalho: `/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/`
(NUNCA o source original). Sua stream é **100% independente do código Go** — você
não toca em nenhum `.go`, então roda em paralelo total, sem colisão.

## Objetivo
Entregar o stack de observabilidade **easy-deploy** (docker-compose) que monitora
todos os componentes do fluxo de credencial/cota/rotação, conforme o documento de
observabilidade já aprovado:
`Automonous_Agentic/docs/project/05-observability.md`.

## Protocolo de check-in/out (OBRIGATÓRIO)
1. Leia `Automonous_Agentic/.deploy-control/*.md` (status IN_PROGRESS).
2. Confirme que seus arquivos não colidem (você só cria em
   `multica-auth-work/deploy/observability/**` — ninguém mais mexe lá).
3. Crie o check-in ANTES de editar:
   `.deploy-control/GLM52__W-OBS__<START_UTC>.md`:
   ```
   agent: GLM-52
   stream: W-OBS
   started_at: <UTC ISO>
   finished_at:
   status: IN_PROGRESS
   files_locked:
     - deploy/observability/**
   depends_on: []
   build_result:
   notes:
   ```
4. Check-out: `finished_at`, `status: DONE`, `build_result` (validações abaixo).

## Entregáveis (em `multica-auth-work/deploy/observability/`)
1. `docker-compose.yml` — serviços: prometheus (9090), grafana (3000),
   alertmanager (9093), postgres-exporter (9187). Segredos via docker secrets /
   variáveis `*_FILE` (NUNCA hardcoded).
2. `prometheus.yml` — scrape dos alvos: serviço de credencial/daemon (`/metrics`),
   postgres-exporter, o próprio prometheus. `rule_files: [alerts.yml]`.
3. `alerts.yml` — as regras do doc 05: CredentialRestoreFailing,
   EnvInjectionFailing, AllAccountsExhausted, RotationFailing, NoAvailableAccounts,
   PostgresDown, (opcional) SecretInLogSuspected.
4. `alertmanager.yml` — roteamento para um webhook local (placeholder).
5. `grafana/provisioning/` (datasource Prometheus + dashboards auto-load) e
   `grafana/dashboards/*.json` — 4 dashboards: Credential Health, Accounts & Quota,
   Rotation, Platform Health (painéis conforme doc 05 §5).
6. `README.md` — como subir (`docker compose up -d`) e portas.

## Cobertura obrigatória (doc 05 §3)
Cada componente deve ter painel + (onde crítico) alerta: daemon/execenv, store de
credencial (Postgres), contas/cota, rotação, detecção de esgotamento, agentes,
host/containers.

## Validações (antes do check-out DONE)
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/deploy/observability
docker compose config >/dev/null && echo COMPOSE_OK
docker run --rm -v "$PWD":/w -w /w prom/prometheus:latest \
  promtool check config prometheus.yml
docker run --rm -v "$PWD":/w -w /w prom/prometheus:latest \
  promtool check rules alerts.yml
```
JSON dos dashboards deve ser válido (`python -m json.tool` em cada `.json`).
Só marque DONE com `COMPOSE_OK` + `promtool` sem erro + JSON válido.

## Regras
- Postgres-only (o exporter aponta para o Postgres do produto).
- NENHUM segredo em métrica, label, config ou log. Use placeholders / *_FILE.
- Não edite nenhum `.go` nem toque em `execenv/`, `daemon/`, docs fora de
  `deploy/observability/`.
- Se faltar o endpoint `/metrics` do serviço (ainda não instrumentado), configure
  o scrape assim mesmo (target placeholder) e anote em `notes` — a instrumentação
  Go é de outra stream.
