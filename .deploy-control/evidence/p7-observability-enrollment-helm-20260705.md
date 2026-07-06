# P7 Evidence - Observability, Enrollment, Helm/Selfhost Runbook

Timestamp: 2026-07-05
Change: `rotation-parity-polyglot`
Tasks: 7.4a, 7.4e, 7.6
Secrets present: false

## Observability Stack

Static validation:

- `docker compose config` passed for `multica-auth-work/deploy/observability/docker-compose.yml`.
- `promtool check config prometheus.yml` passed.
- `promtool check rules alerts.yml` passed with 10 rules.
- `amtool check-config alertmanager.yml` passed.
- 6 Grafana dashboard JSON files parsed: `accounts-quota.json`, `credential-health.json`, `platform-health.json`, `rotation-overview.generated.json`, `rotation.json`, `savings.generated.json`.

Runtime validation:

- Docker Engine responded with version `29.6.0`.
- `multica-prometheus` running, published `0.0.0.0:9090->9090/tcp`.
- `multica-alertmanager` running, published `0.0.0.0:9093->9093/tcp`.
- `multica-postgres-exporter` running, published `0.0.0.0:9187->9187/tcp`.
- `multica-grafana` running, container port `3000`, published locally as `127.0.0.1:13000->3000/tcp` because host port `3000` was already occupied.
- Prometheus readiness returned `Prometheus Server is Ready`.
- Alertmanager readiness returned `OK`.
- postgres-exporter `/metrics` returned Go/process metric families.
- Grafana `/api/health` returned `database: ok`.

Port note:

- Required production port for Grafana remains `3000`.
- This workstation had `wslrelay.exe` PID 5688 listening on `localhost:3000` and serving an existing app that redirects to `/login`.
- No existing process was stopped. For production, free `3000` and run the compose file without the local override.

Prometheus targets at validation time:

- `postgres`: up
- `prometheus`: up
- `credential-service`: down because the product backend was not running in this isolated observability validation.

## Enrollment Runbook

Created `docs/deploy/prodex-account-enrollment-runbook.md`.

Coverage:

- owner and filesystem hard gates;
- POSIX filesystem and `0600`/`0700` permission checks;
- supported vendor credential layouts;
- idempotent `scripts/staging/enroll_account.sh` invocation;
- scrubbed Postgres validation queries;
- failure handling for partial enrollment;
- evidence fields with `secrets_present=false`.

## Helm/Selfhost/Observability/Migrations Runbook

Updated `docs/deploy/prod-rollout-runbook.md`.

Coverage:

- Helm path: `multica-auth-work/deploy/helm/multica`;
- self-host path: `multica-auth-work/docker-compose.selfhost.yml`;
- observability path: `multica-auth-work/deploy/observability/docker-compose.yml`;
- expected observability ports: Prometheus `9090`, Grafana `3000`, Alertmanager `9093`, postgres-exporter `9187`;
- reversible migration gate using `*.up.sql` and `*.down.sql` stem parity;
- prodex account enrollment runbook reference.

Migration parity:

- `*.up.sql`: 161
- `*.down.sql`: 161
- unmatched stems: none

## Remaining Risks

- Production deploy is still owner-gated by `docs/deploy/prod-rollout-runbook.md`.
- Grafana host port `3000` must be free for production observability deploy exactly as documented.
- `credential-service` target requires the product backend metrics endpoint to be running and reachable.
