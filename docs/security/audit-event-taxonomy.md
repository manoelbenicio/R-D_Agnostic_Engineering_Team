# Audit Event Taxonomy

Status: PRE-DEPLOY REQUIRED

## 1. Required Audit Events

Runtime:

- `sidecar_started`
- `sidecar_ready`
- `policy_applied`
- `session_started`
- `session_stopped`
- `route_selected`
- `affinity_bound`
- `fallback_attempted`
- `fallback_succeeded`
- `fallback_failed`

Smart Context:

- `smart_context_shadow`
- `smart_context_rewrite`
- `smart_context_fallback_exact`

Redeem:

- `redeem_attempted`
- `redeem_succeeded`
- `redeem_no_credit`
- `redeem_rejected`

Security:

- `kill_switch_applied`
- `guardrail_block`
- `secret_scrub_failure`
- `profile_switch_fail_closed`

Deploy:

- `deploy_runbook_presented`
- `deploy_owner_approved`
- `deploy_started`
- `deploy_succeeded`
- `deploy_failed`
- `rollback_started`
- `rollback_succeeded`
- `rollback_failed`

## 2. Required Fields

Every audit event must include:

- event id;
- timestamp;
- tenant id;
- session id if applicable;
- actor or component;
- event type;
- result;
- redaction metadata;
- source component;
- correlation id.

## 3. Forbidden Fields

Never include:

- raw token;
- raw cookie;
- raw auth payload;
- database password;
- full connection string;
- full `auth.json`.

## 4. Retention

Minimum:

- runtime events: 30 days;
- deploy/audit/security events: 180 days;
- owner approval events: 1 year.
