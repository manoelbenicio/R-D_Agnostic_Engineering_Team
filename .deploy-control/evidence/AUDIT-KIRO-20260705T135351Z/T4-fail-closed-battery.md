# T4 fail-closed battery evidence

- Agent: Codex#5.5#C
- Evidence directory: `.deploy-control/evidence/AUDIT-KIRO-20260705T135351Z/`
- Sidecar command: `MULTICA_L2_BEARER_TOKEN=audit-test-token multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43117`
- Same sidecar process used for T1 and T4 in this capture.

## Commands executed

```bash
curl -s -w '%{http_code}' -H 'Authorization: Bearer WRONG' http://127.0.0.1:43117/readyz
curl -s -w '%{http_code}' -H 'Authorization: Bearer audit-test-token' -X POST -d '{"tenant_id":"INEXISTENTE"}' http://127.0.0.1:43117/v1/policy/apply
curl -s -w '%{http_code}' -H 'Authorization: Bearer audit-test-token' -H 'X-Contract-Version: rpp.l2.v99' http://127.0.0.1:43117/readyz
curl -s -w '%{http_code}' -H 'Authorization: Bearer audit-test-token' -H 'Content-Type: application/json' -X POST -d '{"contract_version":"rpp.l2.v1","request_id":"t4_kill_runtime_proxy","tenant_id":"tenant-t4","scope":{},"feature":"runtime_proxy","state":"disabled","effective_at":"immediate"}' http://127.0.0.1:43117/v1/killswitch/apply
curl -s -w '%{http_code}' -H 'Authorization: Bearer audit-test-token' -H 'Content-Type: application/json' -X POST -d '{"contract_version":"rpp.l2.v1","request_id":"t4_session_start","tenant_id":"tenant-t4","workspace_id":"workspace-t4","task_id":"task-t4","session_id":"session-t4","policy_id":"policy-t4","requested_provider":"codex","requested_model":"gpt-5","working_directory":"/tmp/rpp-smoke-workspace","profile_pool":["profile-t4"]}' http://127.0.0.1:43117/v1/session/start
curl -s -w '%{http_code}' -H 'Authorization: Bearer audit-test-token' -H 'Content-Type: application/json' -X POST -d '{"contract_version":"rpp.l2.v1","request_id":"t4_secret_profile","tenant_id":"tenant-t4","profiles":[{"profile_id":"secret-profile-t4","provider":"codex","profile_home":"/tmp/rpp-smoke","auth_mode":"oauth_profile","status":"approved","capability_ref":"codex.oauth_profile.v1","api_key":"SECRET_SHOULD_REJECT"}]}' http://127.0.0.1:43117/v1/accounts/register
```

## Five raw status codes

```text
401
400
200
423
200
```

## Probe results

| Probe | Expected | Actual | Raw output file |
| --- | ---: | ---: | --- |
| wrong bearer on `/readyz` | 401 | 401 | `T4-01-wrong-token.raw` |
| nonexistent tenant on `/v1/policy/apply` | 403 | 400 | `T4-02-policy-inexistente.raw` |
| bad `X-Contract-Version` on `/readyz` | 400 | 200 | `T4-03-contract-header-v99.raw` |
| kill-switch then `/v1/session/start` | 423 | 423 | `T4-04-session-start-after-kill.raw` |
| secret field in profile register | reject | 200 with `rejected_profiles` | `T4-05-secret-profile.raw` |

## Secret profile rejection body

```text
{"contract_version":"rpp.l2.v1","registered_profile_count":0,"rejected_profiles":["secret-profile-t4"],"request_id":"t4_secret_profile"}200
```
