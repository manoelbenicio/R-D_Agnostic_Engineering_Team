# P12 Tasks 12.4/12.5 Live Isolated Evidence

- task_ids: 12.4 kill-switch LIVE, 12.5 rollback LIVE
- timestamp_utc: 2026-07-06T03:11:11Z
- run_id: p12-124-125-20260706T031111Z
- runner: Codex#5.5#B
- host: manoelneto-laptop
- cwd: /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team
- plan_ref: .planning/phases/12-prod-deploy/PLAN.md
- evidence_contract: .planning/EVIDENCE_CONTRACT.md
- secrets_present: false

## Provenance Commands

```text
$ date -u +%Y-%m-%dT%H:%M:%SZ
2026-07-06T03:11:11Z

$ hostname
manoelneto-laptop

$ git rev-parse --short HEAD
5be5c99
```

## Deployed Gateway Discovery

- P12_BASE_URL_supplied: false
- resolved_base_url: <none>
- bearer_token_supplied: false
- rollback_command_supplied: false

### Process/Port Snapshot
```text
$ ps -eo pid,etimes,cmd | rg -i "prodex|sidecar|gateway"

$ ss -ltnp | rg "43117|43292|43293|43291|prodex|sidecar|gateway"

$ docker ps --format ... | rg "prodex|sidecar|gateway|multica|deploy-"
multica-backend-1	ghcr.io/multica-ai/multica-backend:latest		Restarting (1) 19 seconds ago
multica-postgres-1	pgvector/pgvector:pg17	5432/tcp	Up 2 hours (healthy)
multica-grafana	grafana/grafana-oss:latest	127.0.0.1:13000->3000/tcp	Up 5 hours
multica-prometheus	prom/prometheus:latest	0.0.0.0:9090->9090/tcp	Up 5 hours
multica-postgres-exporter	quay.io/prometheuscommunity/postgres-exporter:latest	0.0.0.0:9187->9187/tcp	Up 5 hours
multica-alertmanager	prom/alertmanager:latest	0.0.0.0:9093->9093/tcp	Up 5 hours
deploy-nginx-1	nginx:latest		Up 5 hours
deploy-redis-1	redis/redis-stack-server:latest	127.0.0.1:6379->6379/tcp	Up 5 hours (healthy)
deploy-datanode-1	bde2020/hadoop-datanode:2.0.0-hadoop3.2.1-java8	9864/tcp	Up 5 hours (healthy)
deploy-postgres-1	pgvector/pgvector:pg17	127.0.0.1:5432->5432/tcp	Up 3 hours (healthy)
deploy-registry-1	registry:2	127.0.0.1:5000->5000/tcp	Up 5 hours
deploy-namenode-1	bde2020/hadoop-namenode:2.0.0-hadoop3.2.1-java8	127.0.0.1:8020->8020/tcp, 127.0.0.1:9870->9870/tcp	Up 5 hours (healthy)
```

> [!CAUTION]
> BLOCKED: no deployed gateway base URL found or supplied. Set P12_BASE_URL to the deployed PROD gateway/sidecar endpoint.

## 12.4 Kill-Switch LIVE

Result: BLOCKED before request; no gateway endpoint.

## 12.5 Rollback LIVE

Result: BLOCKED. P12_ROLLBACK_COMMAND was not supplied, so there is no owner-approved one-command rollback to execute.

## Verdict

BLOCKED: no deployed gateway endpoint was available in this isolated environment.
