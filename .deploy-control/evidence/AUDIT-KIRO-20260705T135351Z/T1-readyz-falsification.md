# T1 readyz falsification evidence

- Agent: Codex#5.5#C
- Evidence directory: `.deploy-control/evidence/AUDIT-KIRO-20260705T135351Z/`
- Sidecar command: `MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117`

## Commands executed

```bash
curl -s -H 'Authorization: Bearer audit-test-token' http://127.0.0.1:43117/readyz | tee T1-readyz-before.json
docker stop deploy-postgres-1
curl -s -H 'Authorization: Bearer audit-test-token' http://127.0.0.1:43117/readyz | tee T1-readyz-after-pg-down.json
docker start deploy-postgres-1
```

## T1-readyz-before.json

```json
{"checks":[{"details":{"backend_type":"postgres","connection_status":"ok"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"name":"runtime_proxy","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

## T1-readyz-after-pg-down.json

```json
{"checks":[{"details":{"backend_type":"postgres","connection_status":"ok"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"name":"runtime_proxy","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

## Observation

`/readyz` reported `status: ready`, `shared_state_backend.status: pass`, and `connection_status: ok` both before and after `deploy-postgres-1` was stopped.
