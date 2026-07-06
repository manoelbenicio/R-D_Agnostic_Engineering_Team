# Deploy-Rollback Spec Test Proof

- timestamp_utc: 2026-07-06T03:21:27Z
- runner: Codex#5.5#B
- task: prove deploy-rollback kill-switch and rollback by test
- spec: openspec/changes/rotation-parity-polyglot/specs/deploy-rollback/spec.md
- sidecar: /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
- prodex_bin: /home/dataops-lab/.nvm/versions/node/v24.17.0/bin/prodex
- codex_bin: /home/dataops-lab/.nvm/versions/node/v24.17.0/bin/codex
- temp_root: /tmp/tmp.tHWKUWiKAT
- secrets_present: false

## Provenance

```text
$ hostname
manoelneto-laptop

$ git rev-parse --short HEAD
5be5c99

$ prodex --version
prodex 0.246.0

$ codex --version
codex-cli 0.142.5

$ sha256sum multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
2c21b3a83e87f6b130534face6b3d7f091bc7f26026094a1077001c7a7872266  /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
```

## Preflight
- PASS pinned prodex version: prodex 0.246.0

## Isolated Sidecar Start
```text
$ PRODEX_HOME=<tmp>/prodex-home CODEX_HOME=<tmp>/codex-home MULTICA_PRODEX_PATH=/home/dataops-lab/.nvm/versions/node/v24.17.0/bin/prodex PRODEX_GATEWAY_LISTEN=127.0.0.1:55859 MULTICA_L2_BEARER_TOKEN=<redacted> /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:35031
```

### healthz after start
```text
{"contract_version":"rpp.l2.v1","sidecar":{"commit":"smoke","name":"prodex-sidecar","version":"0.1.0"},"status":"alive"}```

### readyz observation after start
```text
{"checks":[{"details":{"connection_status":"error","error":"missing_config"},"name":"shared_state_backend","status":"fail"},{"name":"kill_switch","status":"pass"},{"details":{"listen_addr":"127.0.0.1:55859","pid":384019},"name":"runtime_proxy","status":"pass"},{"name":"event_stream","status":"pass"}],"contract_version":"rpp.l2.v1","status":"error"}```
- PASS healthz status: alive

## Kill-Switch Tests

### 1. Tenant-scope smart_context disables Smart Context

### ks1-baseline-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks1-baseline-session.json
http_status=200
```

### ks1-baseline-session body
```text
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:35031/v1/events/stream?session_id=baseline-smart-context","gateway":{"listen_addr":"127.0.0.1:55859","smart_context_enabled":true},"request_id":"req-baseline-smart-context","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:35031/v1/runtime/proxy?session_id=baseline-smart-context","runtime_log_ref":"prodex-gateway://127.0.0.1:55859","runtime_session_id":"rt-1783308088789412267","smart_context_mode":"proxy_rewrite"}```
- PASS baseline smart_context HTTP: 200
- PASS baseline smart_context mode: proxy_rewrite

### ks1-disable-smart-context command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/killswitch/apply -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks1-disable-smart-context.json
http_status=200
```

### ks1-disable-smart-context body
```text
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"immediate","request_id":"req-deploy-rollback-20260706T032127Z-tenant-smart_context-disabled"}```
- PASS tenant smart_context disable applied: True

### ks1-status-disabled command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=smart_context\&provider=codex\&profile_id=profile-proof -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks1-status-disabled body
```text
{"active":true,"contract_version":"rpp.l2.v1","feature":"smart_context","profile_id":"profile-proof","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS tenant smart_context active: True

### ks1-after-disable-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks1-after-disable-session.json
http_status=200
```

### ks1-after-disable-session body
```text
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:35031/v1/events/stream?session_id=after-smart-context-disabled","gateway":{"listen_addr":"127.0.0.1:55859","smart_context_enabled":true},"request_id":"req-after-smart-context-disabled","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:35031/v1/runtime/proxy?session_id=after-smart-context-disabled","runtime_log_ref":"prodex-gateway://127.0.0.1:55859","runtime_session_id":"rt-1783308089709389590","smart_context_mode":"exact"}```
- PASS after tenant smart_context HTTP: 200
- PASS after tenant smart_context mode: exact

### ks1-enable-smart-context command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/killswitch/apply -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks1-enable-smart-context.json
http_status=200
```

### ks1-enable-smart-context body
```text
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"immediate","request_id":"req-deploy-rollback-20260706T032127Z-tenant-smart_context-enabled"}```

### ks1-status-enabled command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=smart_context\&provider=codex\&profile_id=profile-proof -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks1-status-enabled body
```text
{"active":false,"contract_version":"rpp.l2.v1","feature":"smart_context","profile_id":"profile-proof","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS tenant smart_context inactive after enable: False

### ks1-after-enable-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks1-after-enable-session.json
http_status=200
```

### ks1-after-enable-session body
```text
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:35031/v1/events/stream?session_id=after-smart-context-enabled","gateway":{"listen_addr":"127.0.0.1:55859","smart_context_enabled":true},"request_id":"req-after-smart-context-enabled","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:35031/v1/runtime/proxy?session_id=after-smart-context-enabled","runtime_log_ref":"prodex-gateway://127.0.0.1:55859","runtime_session_id":"rt-1783308090721932694","smart_context_mode":"proxy_rewrite"}```
- PASS after tenant smart_context restore HTTP: 200
- PASS after tenant smart_context restore mode: proxy_rewrite

### 2. Provider-scope gateway disables routing

### ks2-baseline-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks2-baseline-session.json
http_status=200
```

### ks2-baseline-session body
```text
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:35031/v1/events/stream?session_id=baseline-gateway","gateway":{"listen_addr":"127.0.0.1:55859","smart_context_enabled":true},"request_id":"req-baseline-gateway","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:35031/v1/runtime/proxy?session_id=baseline-gateway","runtime_log_ref":"prodex-gateway://127.0.0.1:55859","runtime_session_id":"rt-1783308091140168608","smart_context_mode":"proxy_rewrite"}```
- PASS baseline gateway HTTP: 200

### ks2-disable-gateway command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/killswitch/apply -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks2-disable-gateway.json
http_status=200
```

### ks2-disable-gateway body
```text
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"immediate","request_id":"req-deploy-rollback-20260706T032127Z-provider-gateway-disabled"}```

### ks2-status-disabled command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=gateway\&provider=codex\&profile_id=profile-proof -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks2-status-disabled body
```text
{"active":true,"contract_version":"rpp.l2.v1","feature":"gateway","profile_id":"profile-proof","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS provider gateway active: True

### ks2-after-disable-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks2-after-disable-session.json
http_status=423
```

### ks2-after-disable-session body
```text
{"error":"gateway disabled by kill switch"}```
- PASS provider gateway blocked HTTP: 423

### ks2-other-provider-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks2-other-provider-session.json
http_status=200
```

### ks2-other-provider-session body
```text
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:35031/v1/events/stream?session_id=other-provider-not-blocked","gateway":{"listen_addr":"127.0.0.1:55859","smart_context_enabled":true},"request_id":"req-other-provider-not-blocked","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:35031/v1/runtime/proxy?session_id=other-provider-not-blocked","runtime_log_ref":"prodex-gateway://127.0.0.1:55859","runtime_session_id":"rt-1783308092506478372","smart_context_mode":"proxy_rewrite"}```
- PASS other provider unaffected HTTP: 200

### ks2-enable-gateway command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/killswitch/apply -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks2-enable-gateway.json
http_status=200
```

### ks2-enable-gateway body
```text
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"immediate","request_id":"req-deploy-rollback-20260706T032127Z-provider-gateway-enabled"}```

### ks2-status-enabled command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=gateway\&provider=codex\&profile_id=profile-proof -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks2-status-enabled body
```text
{"active":false,"contract_version":"rpp.l2.v1","feature":"gateway","profile_id":"profile-proof","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS provider gateway inactive after enable: False

### ks2-after-enable-session command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/session/start -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks2-after-enable-session.json
http_status=200
```

### ks2-after-enable-session body
```text
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:35031/v1/events/stream?session_id=after-gateway-enabled","gateway":{"listen_addr":"127.0.0.1:55859","smart_context_enabled":true},"request_id":"req-after-gateway-enabled","router_owner":"rust_l2","runtime_endpoint":"http://127.0.0.1:35031/v1/runtime/proxy?session_id=after-gateway-enabled","runtime_log_ref":"prodex-gateway://127.0.0.1:55859","runtime_session_id":"rt-1783308093583402178","smart_context_mode":"proxy_rewrite"}```
- PASS provider gateway restored HTTP: 200

### 3. Profile-scope auto_redeem disables auto-redeem state

### ks3-disable-auto-redeem command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/killswitch/apply -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks3-disable-auto-redeem.json
http_status=200
```

### ks3-disable-auto-redeem body
```text
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"immediate","request_id":"req-deploy-rollback-20260706T032127Z-profile-auto_redeem-disabled"}```
- PASS profile auto_redeem disable applied: True

### ks3-status-disabled command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=auto_redeem\&provider=codex\&profile_id=profile-proof -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks3-status-disabled body
```text
{"active":true,"contract_version":"rpp.l2.v1","feature":"auto_redeem","profile_id":"profile-proof","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS profile auto_redeem active: True

### ks3-status-other-profile command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=auto_redeem\&provider=codex\&profile_id=profile-other -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks3-status-other-profile body
```text
{"active":false,"contract_version":"rpp.l2.v1","feature":"auto_redeem","profile_id":"profile-other","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS other profile auto_redeem unaffected: False

### ks3-enable-auto-redeem command/status
```text
$ curl -sS --max-time 8 -X POST http://127.0.0.1:35031/v1/killswitch/apply -H Authorization:\ Bearer\ \<redacted\> -H Content-Type:\ application/json --data-binary @ks3-enable-auto-redeem.json
http_status=200
```

### ks3-enable-auto-redeem body
```text
{"applied":true,"contract_version":"rpp.l2.v1","effective_at":"immediate","request_id":"req-deploy-rollback-20260706T032127Z-profile-auto_redeem-enabled"}```

### ks3-status-enabled command/status
```text
$ curl -sS --max-time 8 -X GET http://127.0.0.1:35031/v1/killswitch/status\?tenant_id=tenant-proof\&feature=auto_redeem\&provider=codex\&profile_id=profile-proof -H Authorization:\ Bearer\ \<redacted\>
http_status=200
```

### ks3-status-enabled body
```text
{"active":false,"contract_version":"rpp.l2.v1","feature":"auto_redeem","profile_id":"profile-proof","provider":"codex","session_id":"","tenant_id":"tenant-proof"}```
- PASS profile auto_redeem inactive after enable: False

Auto-redeem implementation note: this sidecar exposes auto_redeem as kill-switch state and event only; session_block currently checks runtime_proxy/gateway/provider_bridge, while smart_context changes session mode. The test therefore verifies auto_redeem disable/restore through the status API and profile scope isolation.

## One-Command Rollback Test

### rollback before env
```text
MULTICA_CODEX_PATH=/home/dataops-lab/.nvm/versions/node/v24.17.0/bin/prodex
MULTICA_PRODEX_ENABLED=1
MULTICA_PRODEX_PATH=/home/dataops-lab/.nvm/versions/node/v24.17.0/bin/prodex
MULTICA_PRODEX_VERSION=v0.246.0
MULTICA_PRODEX_COMMIT=7750da9b6a5c91a6d429e18e6a4d422cab4bc144
PRODEX_HOME=/tmp/tmp.tHWKUWiKAT/isolated-prodex-home
CODEX_HOME=/tmp/tmp.tHWKUWiKAT/isolated-codex-home
MULTICA_L2_ENABLED=1
MULTICA_L2_BASE_URL=http://127.0.0.1:35031
MULTICA_L2_BEARER_TOKEN=<redacted>
MULTICA_L2_SIDECAR_ARGS=--isolated-profile profile-proof
```

Before rollback behavior:
```text
$ "$MULTICA_CODEX_PATH" --version
prodex 0.246.0
```

One command executed:
```text
$ ROLLBACK_ALLOW_EXECUTE=1 ROLLBACK_TARGET_ENV=smoke bash /mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/scripts/deploy/rollback-to-raw-codex.sh --env-file /tmp/tmp.tHWKUWiKAT/multica-runtime.env --codex-path /home/dataops-lab/.nvm/versions/node/v24.17.0/bin/codex --execute
```

### rollback command output
```text
[rollback-to-raw-codex] PASS rollback_id=rollback-20260706T032136Z backup=/tmp/tmp.tHWKUWiKAT/multica-runtime.env.rollback-20260706T032136Z.bak
```

### rollback after env
```text
CODEX_HOME=/tmp/tmp.tHWKUWiKAT/isolated-codex-home
MULTICA_CODEX_PATH=/home/dataops-lab/.nvm/versions/node/v24.17.0/bin/codex
MULTICA_PRODEX_ENABLED=0
MULTICA_L2_ENABLED=0
MULTICA_ROLLBACK_ID=rollback-20260706T032136Z
```

### raw codex behavior after rollback
```text
codex-cli 0.142.5
```
- PASS rollback MULTICA_CODEX_PATH: /home/dataops-lab/.nvm/versions/node/v24.17.0/bin/codex
- PASS rollback MULTICA_PRODEX_ENABLED: 0
- PASS rollback MULTICA_L2_ENABLED: 0
- PASS rollback removed prodex/L2 routing keys
- PASS raw codex version after rollback: codex-cli 0.142.5

## Verdict
PASS. Kill-switch and rollback behavior were proven by executable tests with before/after captures.

## Scrub Check
```text
0 matches
```
