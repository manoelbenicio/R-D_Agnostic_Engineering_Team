# L2 Metrics and Alerts

Status: DRAFT FOR OWNER REVIEW - PROD DEPLOY NO-GO

Runtime metrics come from Rust/prodex L2 events ingested by Multica Go. Events are observability and ledger evidence only; Go must not use them to re-decide an in-flight committed request.

## 1. Event and Metric Sources

Required sources:

- prodex lifecycle and runtime events;
- Go sidecar lifecycle checks;
- Postgres ledger writes;
- kill switch audit state;
- Smart Context shadow/canary decisions;
- redaction scrubber results;
- rollback/deploy control records.

Every ingested runtime event must validate against `docs/contracts/runtime-events.schema.json`, include `contract_version`, and have `secrets_present=false`. Invalid events are rejected with redacted error logs only.

## 2. Runtime Health Metrics

```text
l2_sidecar_alive{tenant,host}
l2_sidecar_ready{tenant,host}
l2_sidecar_start_total{tenant,result}
l2_sidecar_restart_total{tenant,reason}
l2_sidecar_health_failures_total{tenant,check}
l2_readiness_failures_total{tenant,check}
l2_event_stream_connected{tenant,session}
l2_event_stream_disconnect_total{tenant,reason}
l2_event_ingest_reject_total{tenant,reason}
l2_event_audit_write_fail_total{tenant}
```

Critical alerts:

- `l2_sidecar_ready == 0` for a deployed tenant for more than 60 seconds;
- any readiness failure for `shared_state_backend`, `kill_switch`, `event_stream`, `attestation`, or `profile_permissions`;
- any audit write failure after deploy.

## 3. Routing and Session Metrics

```text
l2_sessions_started_total{tenant,provider,result}
l2_sessions_stopped_total{tenant,reason}
l2_runtime_router_owner{tenant,session,owner}
l2_route_selection_total{tenant,provider,result}
l2_affinity_bind_total{tenant,provider}
l2_affinity_reuse_total{tenant,provider}
l2_fallback_attempted_total{tenant,provider,reason}
l2_fallback_succeeded_total{tenant,provider}
l2_fallback_failed_total{tenant,provider,reason}
l2_committed_fallback_rejected_total{tenant,provider}
l2_legacy_go_router_invocation_total{tenant,session}
```

Critical alerts:

- `l2_legacy_go_router_invocation_total > 0` for a prodex-owned session;
- any fallback after commit;
- profile switch fail-open event;
- continuation/affinity mismatch.

Warning alerts:

- fallback failure rate above owner-approved threshold over 15 minutes;
- selection latency p95 above owner-approved baseline;
- affinity reuse unexpectedly drops for continuation-heavy sessions.

## 4. Smart Context Metrics

```text
l2_smart_context_mode{tenant,provider,profile}
l2_smart_context_shadow_total{tenant,provider}
l2_smart_context_canary_total{tenant,provider,result}
l2_smart_context_rewrite_total{tenant,provider,result}
l2_smart_context_exact_fallback_total{tenant,provider,reason}
l2_smart_context_validation_fail_total{tenant,provider,reason}
l2_smart_context_tokens_before{tenant,provider}
l2_smart_context_tokens_after{tenant,provider}
l2_smart_context_rewrite_ratio_percent{tenant,provider}
l2_smart_context_overhead_ms{tenant,provider,quantile}
```

Critical alerts:

- Smart Context live/canary enabled without approval record;
- protocol, continuation, tool-call, JSON, or mandatory-context integrity failure without exact fallback;
- exact fallback failure;
- secret detected in Smart Context evidence.

Warning alerts:

- canary error rate above baseline;
- p95 rewrite overhead above owner-approved threshold;
- rewrite ratio is extreme or unexpectedly zero during canary.

## 5. State Backend Metrics

```text
l2_postgres_ready{cluster}
l2_postgres_migration_version{cluster}
l2_postgres_state_errors_total{cluster,table,operation}
l2_postgres_write_latency_ms{cluster,table,quantile}
l2_sqlite_shared_backend_detected_total{tenant,path_hash}
l2_redis_ready{cluster}
l2_redis_operation_errors_total{cluster,operation}
```

Critical alerts:

- Postgres unavailable;
- migration version mismatch;
- any shared SQLite backend detected;
- kill switch audit write failure;
- deployment approval record write failure.

## 6. Security and Redaction Metrics

```text
l2_secret_scrub_failures_total{source,kind}
l2_redaction_smoke_result{source}
l2_event_secrets_present_total{tenant,source}
l2_profile_permission_violation_total{tenant,profile}
l2_profile_on_9p_detected_total{tenant,profile}
l2_bearer_token_auth_fail_total{tenant,source}
l2_kill_switch_applied_total{tenant,feature,scope,result}
l2_kill_switch_unavailable_total{tenant}
l2_guardrail_block_total{tenant,feature,reason}
```

Critical alerts:

- any secret scrub failure;
- any event with `secrets_present != false`;
- any credential/profile path on 9p;
- any credential file not mode 0600;
- kill switch unavailable;
- bearer token authentication disabled or failing open.

## 7. Redeem Metrics

Auto-redeem is disabled for initial PROD rollout unless separately approved.

```text
l2_redeem_enabled{tenant,provider}
l2_redeem_attempted_total{tenant,provider,profile}
l2_redeem_succeeded_total{tenant,provider,profile}
l2_redeem_rejected_total{tenant,provider,profile,reason}
l2_redeem_no_credit_total{tenant,provider,profile}
l2_redeem_cooldown_block_total{tenant,provider,profile}
```

Critical alert:

- any redeem attempt while auto-redeem is disabled.

Warning alerts:

- redeem rejected unexpectedly;
- repeated cooldown blocks;
- empirical reset-claim validation event missing required audit fields.

## 8. Deploy and Rollback Metrics

```text
l2_deploy_approval_state{deploy_id}
l2_deploy_precheck_result{deploy_id,check}
l2_deploy_smoke_result{deploy_id,check}
l2_rollback_started_total{reason}
l2_rollback_succeeded_total{reason}
l2_rollback_failed_total{reason}
l2_raw_codex_smoke_result{rollback_id}
```

Critical alerts:

- deploy attempts while `deploy_owner_approved=false`;
- rollback command unavailable during active prodex rollout;
- rollback failed to restore raw Codex session smoke.

## 9. Dashboards

Required dashboard panels:

- deploy gate and owner approval state;
- sidecar health/readiness;
- profile filesystem and permission invariant state;
- sessions and router owner;
- profile affinity and fallback;
- Smart Context shadow/canary savings and failures;
- Postgres/Redis state backend health;
- event ingest and audit writes;
- redaction/security guardrails;
- kill switch state;
- rollback readiness and rollback history.

Dashboards must display profile aliases or hashed account ids only. They must not display raw tokens, connection strings, account emails unless explicitly required, raw prompts, raw tool output, or `auth.json`.
