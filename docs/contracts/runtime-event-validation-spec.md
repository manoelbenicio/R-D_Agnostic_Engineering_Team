# Runtime Event Validation Spec

Status: TESTABLE IMPLEMENTATION SPEC

Owner: Codex#5.5#A contract-owner lane

Contract source: `docs/contracts/runtime-events.schema.json`

Contract version: `rpp.l2.v1`

Scope: Multica Go event-ingest validation before any runtime event is written to ledger or observability. Events are evidence only; ingest must not use them to reroute an already committed request.

## Hard Reject Rules

Go event-ingest MUST reject the event and MUST NOT write ledger or observability when any of these rules fails:

1. `event_type` is absent or is not one of the schema enum values listed in this document.
2. `contract_version != "rpp.l2.v1"`.
3. `redaction.secrets_present == true`, or `redaction.secrets_present` is absent, non-boolean, or not exactly `false`.
4. Any common required envelope field is absent or has the wrong type/constraint.
5. Any event-specific required top-level field is absent or has the wrong type/constraint.
6. Any event-specific required nested object field is absent or has the wrong type/constraint.
7. Unknown top-level fields or unknown nested object fields are present. The schema uses `additionalProperties: false`.

## Common Required Envelope

Every event type MUST validate these fields first:

| Field | Required type / constraint |
|---|---|
| `contract_version` | string, exactly `rpp.l2.v1` |
| `event_id` | string, length 8-128 |
| `event_type` | string enum; see event matrix below |
| `occurred_at` | string, RFC 3339 / JSON Schema `date-time` |
| `severity` | string enum: `debug`, `info`, `warn`, `error`, `critical` |
| `producer` | object; required nested fields below |
| `producer.plane` | string, exactly `rust_l2` |
| `producer.component` | string enum: `sidecar`, `runtime_proxy`, `gateway`, `smart_context`, `redeem`, `policy`, `event_stream` |
| `redaction` | object; required nested fields below |
| `redaction.secrets_present` | boolean, exactly `false` |
| `redaction.scrubber_version` | string, length 1-64 |

Optional common fields, when present, MUST match the schema type and constraints:

| Field | Type / constraint |
|---|---|
| `producer.version` | string, max length 64 |
| `producer.commit` | string, max length 80 |
| `tenant_id` | string, length 1-128 |
| `workspace_id` | string, length 1-128 |
| `task_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `runtime_session_id` | string, length 1-128 |
| `runtime_request_id` | string, length 1-128 |
| `policy_id` | string, length 1-128 |
| `provider` | string, length 1-64 |
| `profile_id` | string, length 1-128 |
| `model` | string, length 1-128 |
| `message` | string, max length 512 |
| `payload_ref.ledger_id` | string, max length 128 |
| `payload_ref.sha256` | lowercase hex string matching `^[a-f0-9]{64}$` |
| `redaction.fields_scrubbed[]` | array of strings, each max length 128 |

## Event Type Enum

The exact accepted `event_type` values from the schema are:

- `sidecar_started`
- `sidecar_ready`
- `policy_applied`
- `account_registered`
- `session_started`
- `session_stopped`
- `selection`
- `affinity`
- `fallback`
- `redeem_attempt`
- `redeem_result`
- `rewrite_decision`
- `spend_savings`
- `guardrail`
- `quota_snapshot`
- `gateway_request`
- `gateway_response`
- `kill_switch_applied`
- `error`

## Event Validation Matrix

The "additional required top-level fields" column is in addition to the common required envelope. The "required nested fields" column lists fields that must exist inside the event-specific object when that object is required.

| `event_type` | Additional required top-level fields | Required nested fields and types |
|---|---|---|
| `sidecar_started` | none | none |
| `sidecar_ready` | none | none |
| `policy_applied` | none | none |
| `account_registered` | none | none |
| `session_started` | `tenant_id` string length 1-128; `session_id` string length 1-128 | none |
| `session_stopped` | `tenant_id` string length 1-128; `session_id` string length 1-128 | none |
| `selection` | `tenant_id` string length 1-128; `session_id` string length 1-128; `runtime_request_id` string length 1-128; `selection` object | `selection.decision_phase` string enum `pre_commit`; `selection.selected_profile_id` string length 1-128; `selection.selected_provider` string length 1-64; `selection.reason` string enum `fresh_request`, `policy_preferred`, `quota_available`, `provider_requested`, `manual_profile_requested`, `fallback_recovery`, `not_applicable`; `selection.committed` boolean |
| `affinity` | `tenant_id` string length 1-128; `session_id` string length 1-128; `runtime_request_id` string length 1-128; `affinity` object | `affinity.binding_type` string enum `previous_response_id`, `turn_state`, `session_id`; `affinity.binding_profile_id` string length 1-128; `affinity.binding_source` string enum `runtime_state`, `session_start`, `provider_metadata`; `affinity.overrode_fresh_selection` boolean, exactly `true` |
| `fallback` | `tenant_id` string length 1-128; `session_id` string length 1-128; `runtime_request_id` string length 1-128; `fallback` object | `fallback.phase` string enum `pre_commit`; `fallback.result` string enum `attempted`, `succeeded`, `failed`, `blocked`; `fallback.from_profile_id` string length 1-128; `fallback.reason` string enum `quota_exhausted`, `rate_limited`, `profile_unhealthy`, `transport_precommit_failure`, `provider_capability_rejected`, `kill_switch`, `fallback_budget_exhausted`; `fallback.committed` boolean, exactly `false` |
| `redeem_attempt` | `tenant_id` string length 1-128; `session_id` string length 1-128; `profile_id` string length 1-128; `redeem` object | `redeem.action` string enum `attempt`, `result`; `redeem.profile_id` string length 1-128; `redeem.guard_state` string enum `weekly_exhausted_no_other_profile`, `cooldown_active`, `reset_imminent`, `profile_not_exhausted`, `other_profile_eligible`, `provider_unsupported`, `not_applicable`; `redeem.result` string enum `attempted`, `succeeded`, `no_credit`, `rejected`, `blocked`, `error`, `not_applicable` |
| `redeem_result` | `tenant_id` string length 1-128; `session_id` string length 1-128; `profile_id` string length 1-128; `redeem` object | `redeem.action` string enum `attempt`, `result`; `redeem.profile_id` string length 1-128; `redeem.guard_state` string enum `weekly_exhausted_no_other_profile`, `cooldown_active`, `reset_imminent`, `profile_not_exhausted`, `other_profile_eligible`, `provider_unsupported`, `not_applicable`; `redeem.result` string enum `attempted`, `succeeded`, `no_credit`, `rejected`, `blocked`, `error`, `not_applicable` |
| `rewrite_decision` | `tenant_id` string length 1-128; `session_id` string length 1-128; `runtime_request_id` string length 1-128; `rewrite_decision` object | `rewrite_decision.mode` string enum `disabled`, `shadow`, `canary`, `live`, `exact`; `rewrite_decision.decision` string enum `pass_through`, `shadow_only`, `rewrite`, `segment_rollback`, `exact_fallback`, `blocked`; `rewrite_decision.fallback_exact` boolean; `rewrite_decision.validation_result` string enum `passed`, `failed_protocol`, `failed_continuation`, `failed_tool_integrity`, `failed_json_integrity`, `failed_mandatory_reference`, `not_applicable` |
| `spend_savings` | `tenant_id` string length 1-128; `session_id` string length 1-128; `runtime_request_id` string length 1-128; `spend_savings` object | none; object may be empty, but any present `spend_savings.*` field must match the optional field constraints below |
| `guardrail` | `tenant_id` string length 1-128; `session_id` string length 1-128; `guardrail` object | `guardrail.guardrail_type` string enum `kill_switch`, `redaction`, `profile_isolation`, `provider_capability`, `smart_context_integrity`, `redeem_guard`, `state_backend`, `single_router`; `guardrail.action` string enum `allowed`, `blocked`, `degraded`, `fallback_exact`, `fail_closed` |
| `quota_snapshot` | `tenant_id` string length 1-128; `session_id` string length 1-128; `profile_id` string length 1-128; `quota_snapshot` object | none; object may be empty, but any present `quota_snapshot.*` field must match the optional field constraints below |
| `gateway_request` | `tenant_id` string length 1-128; `session_id` string length 1-128 | none |
| `gateway_response` | `tenant_id` string length 1-128; `session_id` string length 1-128 | none |
| `kill_switch_applied` | none | none |
| `error` | `error` object | `error.code` string, max length 80 |

## Event-Specific Optional Field Constraints

When these event-specific objects or fields are present, Go event-ingest MUST validate them before writing:

| Field | Type / constraint |
|---|---|
| `selection.strategy` | string enum `policy_pool`, `quota_aware`, `provider_bridge`, `gateway_route`, `manual_profile`, `not_applicable` |
| `selection.candidate_count` | integer, minimum 0 |
| `affinity.binding_provider` | string, length 1-64 |
| `fallback.to_profile_id` | string, length 1-128 |
| `fallback.attempt_number` | integer, minimum 1 |
| `redeem.cooldown_seconds_remaining` | integer, minimum 0 |
| `redeem.idempotency_key` | string, length 8-128 |
| `rewrite_decision.rollout_mode` | string enum `shadow`, `canary_in`, `canary_out`, `live`, `exact`, `disabled` |
| `rewrite_decision.rollout_canary_percent` | integer, 0-100 inclusive |
| `rewrite_decision.transformed_segment_categories[]` | array of strings; each item enum `protocol_exact`, `continuation_exact`, `critical_exact`, `rehydratable_exact`, `lossless_transformable`, `condensable`, `droppable_duplicate` |
| `spend_savings.estimated_input_tokens_before` | integer, minimum 0 |
| `spend_savings.estimated_input_tokens_after` | integer, minimum 0 |
| `spend_savings.estimated_input_tokens_saved` | integer, minimum 0 |
| `spend_savings.rewrite_ratio_percent` | number, 0-100 inclusive |
| `spend_savings.body_bytes_before` | integer, minimum 0 |
| `spend_savings.body_bytes_after` | integer, minimum 0 |
| `spend_savings.body_bytes_saved` | integer, minimum 0 |
| `spend_savings.estimator_confidence` | string enum `high`, `medium`, `low`, `unknown` |
| `spend_savings.pressure_band` | string enum `low`, `medium`, `high`, `critical`, `unknown` |
| `guardrail.reason` | string, max length 256 |
| `guardrail.effective_at` | string enum `immediate`, `next_request`, `session_restart_required`, `not_applicable` |
| `quota_snapshot.classification` | string enum `available`, `thin`, `critical`, `weekly_exhausted`, `rate_limited`, `unknown` |
| `quota_snapshot.source` | string enum `codex_usage`, `vendor_balance`, `rate_limit_headers`, `custom_probe`, `cli_native_store`, `none` |
| `quota_snapshot.is_account_quota` | boolean |
| `error.retryable` | boolean |
| `error.safe_detail` | string, max length 512 |

## Minimum Test Requirements for Codex#5.5#C

Implement Go table-driven tests that prove validation happens before ledger/observability writes:

1. For each of the 19 accepted `event_type` values, a minimal valid event passes validation and reaches a fake ledger/observability sink exactly once.
2. An event with an unknown `event_type` is rejected and reaches no sink.
3. An event with `contract_version` absent or not equal to `rpp.l2.v1` is rejected and reaches no sink.
4. An event with `redaction.secrets_present == true` is rejected and reaches no sink.
5. For each common envelope field, a missing or wrong-typed value is rejected and reaches no sink.
6. For every row in the event validation matrix, each additional required top-level field missing in isolation is rejected and reaches no sink.
7. For every row with required nested fields, each nested field missing or wrong-typed in isolation is rejected and reaches no sink.
8. For all fields with enum/const/range constraints, an out-of-range value is rejected and reaches no sink.
9. Unknown top-level and nested fields are rejected and reach no sink.
10. Valid `selection`, `affinity`, and `fallback` events write observability/ledger only and do not invoke any Go rotation, account mutation, fallback, or reroute path.

This spec intentionally does not add product-code behavior beyond schema validation. It defines the acceptance target for closing the `RuntimeEventStream` validation gap in the F0 matrix.
