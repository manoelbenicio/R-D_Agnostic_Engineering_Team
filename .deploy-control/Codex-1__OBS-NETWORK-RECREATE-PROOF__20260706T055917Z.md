# Codex#1 - OBS-NETWORK-RECREATE-PROOF

agent: Codex#1
task: PROVAR que o override de observabilidade sobrevive a recreate
status: IN_PROGRESS
timestamp_utc: 20260706T055917Z

## Evidence

```console
$ cd '/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work' && docker compose -f docker-compose.selfhost.yml -f docker-compose.override.yml up -d --force-recreate backend
 Container multica-postgres-1 Running 
 Container multica-frontend-1 Running 
 Container multica-backend-1 Recreate 
 Container multica-backend-1 Recreated 
 Container multica-postgres-1 Waiting 
 Container multica-postgres-1 Healthy 
 Container multica-backend-1 Starting 
 Container multica-backend-1 Started 
[exit_code=0]
```

```console
$ sleep 15

[exit_code=0]
```

```console
$ docker inspect multica-backend-1 --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}'
multica-observability multica_default 
[exit_code=0]
```

```console
$ curl -s -X POST http://127.0.0.1:9090/-/reload

[exit_code=0]
```

```console
$ sleep 6

[exit_code=0]
```

```console
$ curl -s 'http://127.0.0.1:9090/api/v1/targets?state=active'
{"status":"success","data":{"activeTargets":[{"discoveredLabels":{"__address__":"backend:9090","__metrics_path__":"/metrics","__scheme__":"http","__scrape_interval__":"15s","__scrape_timeout__":"10s","component":"credential-daemon","job":"credential-service"},"labels":{"component":"credential-daemon","instance":"backend:9090","job":"credential-service"},"scrapePool":"credential-service","scrapeUrl":"http://backend:9090/metrics","globalUrl":"http://backend:9090/metrics","lastError":"","lastScrape":"2026-07-06T05:59:31.617051191Z","lastScrapeDuration":0.016383134,"health":"up","scrapeInterval":"15s","scrapeTimeout":"10s"},{"discoveredLabels":{"__address__":"postgres-exporter:9187","__metrics_path__":"/metrics","__scheme__":"http","__scrape_interval__":"15s","__scrape_timeout__":"10s","component":"postgres","job":"postgres"},"labels":{"component":"postgres","instance":"postgres-exporter:9187","job":"postgres"},"scrapePool":"postgres","scrapeUrl":"http://postgres-exporter:9187/metrics","globalUrl":"http://postgres-exporter:9187/metrics","lastError":"","lastScrape":"2026-07-06T05:59:37.538131107Z","lastScrapeDuration":1.003671799,"health":"up","scrapeInterval":"15s","scrapeTimeout":"10s"},{"discoveredLabels":{"__address__":"localhost:9090","__metrics_path__":"/metrics","__scheme__":"http","__scrape_interval__":"15s","__scrape_timeout__":"10s","component":"prometheus","job":"prometheus"},"labels":{"component":"prometheus","instance":"localhost:9090","job":"prometheus"},"scrapePool":"prometheus","scrapeUrl":"http://localhost:9090/metrics","globalUrl":"http://e9870160e633:9090/metrics","lastError":"","lastScrape":"2026-07-06T05:59:37.56353257Z","lastScrapeDuration":0.003076846,"health":"up","scrapeInterval":"15s","scrapeTimeout":"10s"}],"droppedTargets":[],"droppedTargetCounts":null}}
[exit_code=0]
```

```console
$ curl -s http://127.0.0.1:8080/readyz
{"status":"ok","checks":{"db":"ok","migrations":"ok"}}
[exit_code=0]
```

## Result
backend_networks: multica-observability multica_default 
network_multica_observability_present: yes
credential_service_up_after_force_recreate: yes
backend_readyz_db_migrations_ok: yes
status: DONE
