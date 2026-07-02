# Observability Runbook — Rotation & Credential Alerting

**Owner:** GLM-5.2 (PR-OBS-ALERTS stream)
**Date:** 2026-07-02
**Scope:** the `deploy/observability` stack (Prometheus + Alertmanager + Grafana +
postgres-exporter) and the alert rules in `deploy/observability/alerts.yml`.

This runbook tells you how to run the stack, what each alert means and how to
respond, and — critically — which alerts are live today vs. which await the
daemon-metrics stream (H2).

---

## 1. Stack overview (URLs / ports)

| Component           | Container                 | URL / port            | Purpose                                  |
|---------------------|---------------------------|-----------------------|------------------------------------------|
| Prometheus          | `multica-prometheus`      | http://localhost:9090 | scrape metrics + evaluate alert rules    |
| Alertmanager        | `multica-alertmanager`    | http://localhost:9093 | route firing alerts (webhook/email/slack) |
| Grafana             | `multica-grafana`         | http://localhost:3000 | dashboards (provisioned)                 |
| postgres-exporter   | `multica-postgres-exporter`| http://localhost:9187 | Postgres health metrics (`pg_up`, …)     |
| credential-service  | `multica-backend-1`       | scraped at `backend:9090` (job `credential-service`) | product backend `/metrics` |

Key Prometheus endpoints:
- Rules loaded: `GET http://localhost:9090/api/v1/rules`
- Alert states: `GET http://localhost:9090/api/v1/alerts`
- Targets health: `GET http://localhost:9090/api/v1/targets`
- Reload rules/config: `POST http://localhost:9090/-/reload` (requires `--web.enable-lifecycle`, which is set)

---

## 2. Bring the stack up / down

The stack is defined in `deploy/observability/docker-compose.yml` (compose project
`multica-observability`). It depends on docker secrets in `./secrets/`
(`grafana_admin_password`, `pg_user`, `pg_pass`) — see `secrets/*.example`.

```bash
cd deploy/observability
# one-time: create the docker secrets from the example templates
#   cp secrets/grafana_admin_password.example secrets/grafana_admin_password  # then edit
#   cp secrets/pg_user.example secrets/pg_user ; cp secrets/pg_pass.example secrets/pg_pass
docker compose up -d          # start the full stack
docker compose ps             # verify all services are Up
```

Tear down (keeps volumes):
```bash
cd deploy/observability && docker compose down
```

The product backend (`multica-backend-1`) and Postgres (`multica-postgres-1`) are
started by the product's own compose, not this one. The postgres-exporter connects
to the product Postgres via `host.docker.internal:5432` (see `PG_HOST` in the
compose env). If `pg_up == 0`, the exporter is up but cannot reach the DB — check
the DB container and the `pg_user`/`pg_pass` secrets first.

---

## 3. Reloading alert rules

`alerts.yml` is bind-mounted into Prometheus at `/etc/prometheus/alerts.yml:ro` and
referenced by `prometheus.yml` (`rule_files: [alerts.yml]`). After editing the host
file, validate then hot-reload (no container restart needed):

```bash
# 1. syntax check (catches bad PromQL / YAML before reload)
docker exec multica-prometheus promtool check rules /etc/prometheus/alerts.yml
# 2. hot-reload
curl -s -X POST http://localhost:9090/-/reload
# 3. confirm rules loaded
curl -s http://localhost:9090/api/v1/rules | \
  python3 -c "import sys,json;d=json.load(sys.stdin);print([r['name'] for g in d['data']['groups'] for r in g['rules']])"
```

> **Do not edit `prometheus.yml`** — it is owned by the Opus stream. If a scrape
> target change is required, stop and note it; do not edit it here.

---

## 4. Alert catalog

All rules live in `deploy/observability/alerts.yml` (3 groups:
`credential-rotation`, `credential-isolation`, `platform`). **Every metric name is
real** — defined in `server/internal/metrics/credential_metrics.go`, plus the
standard `up` and `pg_up`. The removed `SecretInLogSuspected` rule referenced a
non-existent metric (`secret_in_log_suspected_total`) and was deleted.

### 4.1 Rotation domain (await daemon metrics — see §6)

#### AllAccountsExhausted — **critical**
- **Condition:** `all_accounts_exhausted == 1` for `1m`
- **Meaning:** every account for a vendor is exhausted or in cooldown. No account
  can serve that vendor.
- **First response:** park the affected agent; schedule a wake at
  `min(cooldown_until)` from the `accounts` table; consider enrolling a spare
  account (`scripts/staging/enroll_account.sh`).

#### NoAccountsAvailable — **warning**
- **Condition:** `accounts_available == 0` for `1m`
- **Meaning:** no account is currently leasable for a vendor (may be transient).
- **First response:** check `accounts` for status mix (available vs cooldown vs
  exhausted); if persistent, treat like `AllAccountsExhausted`.

#### RotationErrorsSpiking — **warning**
- **Condition:** `sum by (vendor) (increase(rotation_total{result="error"}[5m])) > 0`
- **Meaning:** at least one rotation failed in the last 5m.
- **First response:** query `rotation_events` (DB) for recent rows; check
  `accounts.last_error` for the involved accounts.

#### ExhaustionDetectedSpiking — **warning**
- **Condition:** `sum by (vendor) (increase(exhaustion_detected_total[5m])) > 0`
- **Meaning:** exhaustion signals (screen / 429 / ledger) detected in the last 5m.
- **First response:** correlate with `rotation_events`; confirm the signal source
  label and whether proactive rotation kicked in.

### 4.2 Credential isolation domain (await daemon metrics — see §6)

#### CredentialRestoreFailing — **warning**
- **Condition:** `sum by (vendor) (increase(credential_restore_total{result="error"}[5m])) > 0`
- **Meaning:** restoring the per-account credential from its isolated path failed.
- **First response:** verify the account `home_dir` exists and the credential file
  is present with mode `0600` (`ls -l …/creds/<alias>/auth.json`); re-run
  `enroll_account.sh` to re-copy if needed.

#### EnvInjectionFailing — **warning**
- **Condition:** `sum by (vendor) (increase(cred_env_injection_total{result="error"}[5m])) > 0`
- **Meaning:** injecting the vendor-native env var into the task env failed.
- **First response:** check execenv logs; confirm the vendor CLI is installed.

#### CredentialPrepareSlow — **warning**
- **Condition:** `histogram_quantile(0.95, sum by (vendor, le) (rate(credential_prepare_seconds_bucket[5m]))) > 5` for `5m`
- **Meaning:** p95 credential/home preparation exceeds 5s.
- **First response:** check `home_dir` disk latency (esp. on `/mnt/c` DrvFs mounts);
  consider relocating creds to ext4 (see `enroll_account.sh` ext4-backed symlink).

### 4.3 Platform domain (live and drivable today)

#### PostgresDown — **critical**
- **Condition:** `pg_up == 0` for `1m`
- **Meaning:** postgres-exporter is up as a scrape target but reports it cannot
  reach the Postgres DB. The credential store is effectively unobservable / down.
- **First response:** check `multica-postgres-1` health; verify the `pg_user` /
  `pg_pass` docker secrets and `PG_HOST` match the DB. (As of this writing this
  alert was firing because `pg_up == 0` — a real condition to investigate.)
- **Drivable?** Yes — restart/fix the DB or exporter credentials and `pg_up`
  returns to 1.

#### CredentialServiceDown — **critical**
- **Condition:** `up{job="credential-service"} == 0` for `1m`
- **Meaning:** Prometheus cannot scrape the credential-service `/metrics` target
  (`backend:9090` per `prometheus.yml`). The backend/daemon is down or its metrics
  endpoint moved.
- **First response:** check `multica-backend-1` is running; confirm `METRICS_ADDR`
  matches the scraped port; check the `credential-service` target lastError at
  `http://localhost:9090/api/v1/targets`.
- **Drivable?** Yes — stop/start the backend container; `up` flips 1↔0.

#### PostgresExporterDown — **warning**
- **Condition:** `up{job="postgres"} == 0` for `1m`
- **Meaning:** the postgres-exporter itself is not being scraped (the sidecar is
  down), so `pg_*` metrics go stale. Note: while this is firing, `PostgresDown`
  may *resolve* simply because `pg_up` is no longer published — do not read that
  as “DB recovered”.
- **First response:** check `multica-postgres-exporter` container; `docker start
  multica-postgres-exporter`.
- **Drivable?** Yes — proven in this stream (see §5).

---

## 5. Proving alerts fire (drivable signals)

The rotation-metric alerts (§4.1, §4.2) cannot be driven today because the daemon
 does not export those series yet (§6). To prove the alerting pipeline end-to-end,
 use a genuinely drivable target-down signal. **Never fabricate a rotation metric.**

Proven in this stream (`PostgresExporterDown`, identical `up{job}==0` mechanism to
`CredentialServiceDown`):

```bash
# baseline: all targets up
curl -s --data-urlencode 'query=up' http://localhost:9090/api/v1/query
# drive: stop the monitoring sidecar (zero product impact)
docker stop multica-postgres-exporter
# after ~1m: up{job="postgres"} == 0 -> PostgresExporterDown pending -> firing
curl -s http://localhost:9090/api/v1/alerts | \
  python3 -c "import sys,json;d=json.load(sys.stdin);print([(a['labels'].get('alertname'),a['state']) for a in d['data']['alerts']])"
# restore
docker start multica-postgres-exporter
# next eval cycle: PostgresExporterDown resolves; PostgresDown re-engages (pg_up==0)
```

Observed transition: `PostgresExporterDown` inactive → **pending** (activeAt
12:12:16Z) → **firing** (12:14:03Z) → **resolved** after restart. The same
`up{job=…} == 0 for 1m` shape drives `CredentialServiceDown` (stop/start
`multica-backend-1`) — not exercised here to avoid disrupting concurrent agents.

---

## 6. The daemon-metrics gap (H2) — READ THIS

**Today, the rotation/credential-isolation alerts (§4.1, §4.2) will NOT fire from
Prometheus, by design.** The metrics `rotation_total`, `rotation_duration_seconds`,
`all_accounts_exhausted`, `accounts_available`, `exhaustion_detected_total`,
`credential_restore_total`, `cred_env_injection_total`, and
`credential_prepare_seconds` are emitted by the **daemon** process. That process
currently does **not** expose a metrics server (gap documented in
`docs/project/BACKLOG-detection.md`, line 306: “expor metrics server no processo
daemon”).

Verified live (this stream): on the backend `/metrics` endpoint,
`rotation_total` = **0 series**, `all_accounts_exhausted` = **0 series**,
`accounts_available` = **0 series**. So those alert expressions evaluate to empty
and stay **inactive** — they are written against the real metric names so they
light up the moment the daemon exports them (stream **H2**: add a metrics server to
the daemon process and point `prometheus.yml`’s `credential-service` job at it).

**Until H2, rotation is proven via the database, not Prometheus:**
```bash
docker exec -i multica-postgres-1 psql -U multica -d multica -c \
  "SELECT reason, count(*) FROM rotation_events GROUP BY reason ORDER BY 2 DESC;"
docker exec -i multica-postgres-1 psql -U multica -d multica -c \
  "SELECT vendor, status, count(*) FROM accounts GROUP BY vendor, status;"
```
These `rotation_events` / `accounts` queries are the source of truth for rotation
behavior today; the Prometheus alerts are the future automated layer on top.

---

## 7. What to watch day-to-day

1. **PostgresDown (`pg_up == 0`)** — currently a real condition. Investigate the
   exporter→DB connection (secrets, host). This is the highest-priority live item.
2. **CredentialServiceDown** — confirms the backend/daemon is scrapeable. If it
   fires, the backend is down or `METRICS_ADDR` drifted from `backend:9090`.
3. **PostgresExporterDown** — monitoring sidecar health; cheap to fix (`docker start`).
4. **Rotation (via DB today, via Prometheus after H2):** `rotation_events` volume
   by `reason`, `accounts` status mix per vendor, `last_error` on degraded accounts.
5. **Credential isolation (after H2):** `credential_restore_total` /
   `cred_env_injection_total` error rates; `credential_prepare_seconds` p95.
6. **Dashboards:** Grafana `http://localhost:3000` (provisioned datasource =
   Prometheus). Check “Credential Health”, “Accounts & Quota”, “Platform Health”.

---

## 8. Verification cheat-sheet

```bash
# rules loaded (expect 10 rules across 3 groups, no parse errors)
docker exec multica-prometheus promtool check rules /etc/prometheus/alerts.yml
curl -s http://localhost:9090/api/v1/rules | \
  python3 -c "import sys,json;d=json.load(sys.stdin);print([r['name'] for g in d['data']['groups'] for r in g['rules']])"
# alert states
curl -s http://localhost:9090/api/v1/alerts | \
  python3 -c "import sys,json;d=json.load(sys.stdin);print([(a['labels'].get('alertname'),a['state']) for a in d['data']['alerts']])"
# reload after editing alerts.yml
curl -s -X POST http://localhost:9090/-/reload
```

---

## 9. Files touched (this stream)

- `deploy/observability/alerts.yml` — rewritten: 10 rules, real metrics only,
  fake `secret_in_log_suspected_total` removed, drivable target-down alerts added.
- `docs/project/observability-runbook.md` — this file (NEW).
- **Not touched:** product Go, `prometheus.yml` (Opus-owned), Grafana dashboards.
