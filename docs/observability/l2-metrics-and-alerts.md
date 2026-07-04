# L2 Metrics and Alerts

Status: PRE-DEPLOY REQUIRED

## 1. Metrics Source

Runtime metrics come from Rust/prodex L2 runtime events and are ingested by
Multica Go for aggregated dashboards.

Events are observability/ledger only. Go must not re-decide an in-flight
request based on events.

## 2. Required Metrics

Runtime:

- `l2_sidecar_ready`
- `l2_sessions_started_total`
- `l2_sessions_stopped_total`
- `l2_route_selected_total`
- `l2_fallback_attempted_total`
- `l2_fallback_succeeded_total`
- `l2_fallback_failed_total`
- `l2_profile_affinity_bind_total`

Smart Context:

- `l2_smart_context_shadow_total`
- `l2_smart_context_rewrite_total`
- `l2_smart_context_exact_fallback_total`
- `l2_smart_context_tokens_before`
- `l2_smart_context_tokens_after`
- `l2_smart_context_rewrite_ratio_percent`

Redeem:

- `l2_redeem_attempted_total`
- `l2_redeem_succeeded_total`
- `l2_redeem_rejected_total`
- `l2_redeem_no_credit_total`

Security:

- `l2_secret_scrub_failures_total`
- `l2_kill_switch_applied_total`
- `l2_guardrail_block_total`

Health:

- `l2_sidecar_health_failures_total`
- `l2_event_stream_disconnect_total`
- `l2_postgres_state_errors_total`

## 3. Required Alerts

Critical:

- raw secret detected;
- sidecar unavailable;
- kill switch unavailable;
- profile switch not fail-closed;
- Smart Context protocol failure without exact fallback.

Warning:

- fallback failure rate above threshold;
- event stream disconnected;
- Postgres state latency high;
- canary error rate above baseline;
- redeem rejected unexpectedly.

## 4. Dashboards

Add or update dashboards:

- runtime sessions;
- profile/account routing;
- Smart Context savings;
- fallback and errors;
- redeem attempts;
- sidecar health;
- security/redaction;
- deploy/rollback events.
