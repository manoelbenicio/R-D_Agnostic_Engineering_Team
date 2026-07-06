# prodex L2 Event Emission Contract

Status: TARGET-MILESTONE EMITTER CONTRACT. Documentation only; no product code
or deploy.

Scope: Rust/prodex L2 must emit the runtime events below so Multica Go L4 can
validate them against `docs/contracts/runtime-events.schema.json` and
`docs/contracts/runtime-event-validation-spec.md`.

Pinned prodex source: official `github.com/christiandoxa/prodex` commit
`7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.

## Source Basis

Official prodex docs used:

- [architecture.md](https://raw.githubusercontent.com/christiandoxa/prodex/7750da9b6a5c91a6d429e18e6a4d422cab4bc144/docs/architecture.md):
  runtime proxy hot path, hard affinity, rotate-before-commit, observability
  crates, runtime logs, audit log, metrics, and redaction helpers.
- [state-model.md](https://raw.githubusercontent.com/christiandoxa/prodex/7750da9b6a5c91a6d429e18e6a4d422cab4bc144/docs/state-model.md):
  durable profile registry, response/session bindings, quota/health state, and
  runtime log marker discipline.
- [runtime-policy.md](https://raw.githubusercontent.com/christiandoxa/prodex/7750da9b6a5c91a6d429e18e6a4d422cab4bc144/docs/runtime-policy.md):
  runtime log format, gateway adaptive routing, virtual-key dimensions,
  observability sinks, guardrails, usage, ledger, metrics, and audit events.
- [smart-context.md](https://raw.githubusercontent.com/christiandoxa/prodex/7750da9b6a5c91a6d429e18e6a4d422cab4bc144/docs/smart-context.md):
  Smart Context safety model, rollout, fallback, telemetry, and replay fields.
- [provider-conformance.md](https://raw.githubusercontent.com/christiandoxa/prodex/7750da9b6a5c91a6d429e18e6a4d422cab4bc144/docs/provider-conformance.md):
  provider capability and transform classification direction.

Local Multica contract sources:

- `docs/contracts/runtime-events.schema.json`
- `docs/contracts/runtime-event-validation-spec.md`

## Status Labels

- `AS-IS source signal`: official prodex docs confirm the runtime behavior,
  state, log field, audit field, or telemetry source exists.
- `a validar adapter`: official prodex docs do not confirm direct emission of
  the exact Multica `rpp.l2.v1` event shape. The fork/adapter must map prodex
  source signals into the schema and prove validation.

No requested event type is confirmed by official prodex docs as already emitted
AS-IS in the exact Multica schema. All six schema events below therefore require
fork/adapter work before Go L4 may accept them as runtime events.

## Common Envelope

Every emitted event MUST include exactly schema-allowed top-level fields. The
schema has `additionalProperties: false`, so adapters must drop or reject
unknown fields before emission.

Required for every event:

| Field | Required value / constraint | Source status |
|---|---|---|
| `contract_version` | exactly `rpp.l2.v1` | `a validar adapter` |
| `event_id` | string, length 8-128, unique enough for ingest idempotency | `a validar adapter` |
| `event_type` | one of the event names in this document | `a validar adapter` |
| `occurred_at` | RFC3339 / JSON Schema `date-time` string | `a validar adapter` |
| `severity` | `debug`, `info`, `warn`, `error`, or `critical` | `a validar adapter` |
| `producer.plane` | exactly `rust_l2` | `a validar adapter` |
| `producer.component` | schema enum matching source component | `a validar adapter` |
| `redaction.secrets_present` | exactly `false` | `a validar adapter`; prodex redaction helpers are AS-IS source support |
| `redaction.scrubber_version` | string, length 1-64 | `a validar adapter` |

Optional common fields, when present, must obey the schema:
`producer.version`, `producer.commit`, `tenant_id`, `workspace_id`, `task_id`,
`session_id`, `runtime_session_id`, `runtime_request_id`, `policy_id`,
`provider`, `profile_id`, `model`, `message`, `payload_ref`, and
`redaction.fields_scrubbed`.

Emitter rules:

1. Do not emit secrets, bearer tokens, OAuth material, API keys, cookies, raw
   prompts, raw tool outputs, or full provider request/response bodies.
2. Use opaque identifiers only for tenant, workspace, task, session, runtime
   session, request, policy, provider, profile, and model fields.
3. Runtime events are observability and ledger evidence only. Go L4 must not
   use them to reroute or re-decide an in-flight request.
4. If the adapter cannot build the required envelope or event-specific object,
   it must not emit a partial event.

## Emission Matrix

| Event | Official prodex AS-IS source signal | Schema-valid emission status |
|---|---|---|
| `selection` | Runtime proxy owns live transport orchestration; quota/health state is advisory for fresh selection before commit. | `a validar adapter`: no official doc confirms direct `selection` event emission. |
| `affinity` | State model documents hard `previous_response_id`, turn-state, and `session_id` profile bindings. | `a validar adapter`: binding state exists, exact event emission does not. |
| `fallback` | Architecture documents rotate-before-commit and bounded pre-commit routing behavior. | `a validar adapter`: fallback behavior exists, exact event emission does not. |
| `rewrite_decision` | Smart Context logs rollout, validation, fallback, and transformed segment telemetry. | `a validar adapter`: source telemetry exists, exact event object requires mapping. |
| `spend_savings` | Smart Context telemetry includes size, token estimate, pressure, and rewrite-ratio fields. | `a validar adapter`: source telemetry exists, exact event object requires mapping. |
| `guardrail` | Gateway policy documents blocked keywords, output blocks, model allowlists, Presidio/PII redaction, prompt-injection checks, webhooks, and audit/observability surfaces. | `a validar adapter`: guardrails exist, exact event emission does not. |

## `selection`

Purpose: record the fresh pre-commit profile/provider decision made by Rust L2.

Producer:

- `producer.component`: `runtime_proxy`
- Normal severity: `info`; use `warn` only when selection is degraded but still
  valid.

Required top-level fields beyond the common envelope:

| Field | Constraint |
|---|---|
| `tenant_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `runtime_request_id` | string, length 1-128 |
| `selection` | object |

Required `selection` object fields:

| Field | Constraint |
|---|---|
| `selection.decision_phase` | exactly `pre_commit` |
| `selection.selected_profile_id` | string, length 1-128 |
| `selection.selected_provider` | string, length 1-64 |
| `selection.reason` | `fresh_request`, `policy_preferred`, `quota_available`, `provider_requested`, `manual_profile_requested`, `fallback_recovery`, or `not_applicable` |
| `selection.committed` | boolean |

Optional `selection` fields:

| Field | Constraint | Source status |
|---|---|---|
| `selection.strategy` | `policy_pool`, `quota_aware`, `provider_bridge`, `gateway_route`, `manual_profile`, or `not_applicable` | `a validar adapter` |
| `selection.candidate_count` | integer, minimum 0 | `a validar adapter` |

AS-IS vs adapter:

- AS-IS source signal: prodex owns runtime proxy selection before commit.
- Required fork/adapter: construct the Multica envelope and normalize the
  selected profile/provider, reason, commitment state, and optional strategy
  into schema enums.

## `affinity`

Purpose: record that hard continuation/session affinity overrode fresh
selection.

Producer:

- `producer.component`: `runtime_proxy`
- Normal severity: `info`

Required top-level fields beyond the common envelope:

| Field | Constraint |
|---|---|
| `tenant_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `runtime_request_id` | string, length 1-128 |
| `affinity` | object |

Required `affinity` object fields:

| Field | Constraint |
|---|---|
| `affinity.binding_type` | `previous_response_id`, `turn_state`, or `session_id` |
| `affinity.binding_profile_id` | string, length 1-128 |
| `affinity.binding_source` | `runtime_state`, `session_start`, or `provider_metadata` |
| `affinity.overrode_fresh_selection` | exactly `true` |

Optional `affinity` fields:

| Field | Constraint |
|---|---|
| `affinity.binding_provider` | string, length 1-64 |

AS-IS vs adapter:

- AS-IS source signal: official state docs confirm continuation bindings and
  require continuation affinity to beat selection heuristics.
- Required fork/adapter: emit the schema event at the decision point where Rust
  applies the binding, and map the binding source to the schema enum without
  adding raw continuation tokens.

## `fallback`

Purpose: record a pre-commit fallback attempt, success, failure, or block.

Producer:

- `producer.component`: `runtime_proxy`
- Normal severity: `info` for attempted/succeeded, `warn` for failed/blocked,
  `error` only when fallback handling itself fails.

Required top-level fields beyond the common envelope:

| Field | Constraint |
|---|---|
| `tenant_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `runtime_request_id` | string, length 1-128 |
| `fallback` | object |

Required `fallback` object fields:

| Field | Constraint |
|---|---|
| `fallback.phase` | exactly `pre_commit` |
| `fallback.result` | `attempted`, `succeeded`, `failed`, or `blocked` |
| `fallback.from_profile_id` | string, length 1-128 |
| `fallback.reason` | `quota_exhausted`, `rate_limited`, `profile_unhealthy`, `transport_precommit_failure`, `provider_capability_rejected`, `kill_switch`, or `fallback_budget_exhausted` |
| `fallback.committed` | exactly `false` |

Optional `fallback` fields:

| Field | Constraint |
|---|---|
| `fallback.to_profile_id` | string, length 1-128 |
| `fallback.attempt_number` | integer, minimum 1 |

AS-IS vs adapter:

- AS-IS source signal: official architecture requires rotation only before
  commit and no mid-stream rotation after model output starts.
- Required fork/adapter: emit fallback only for pre-commit decisions and prove
  `fallback.committed` is always `false`.

## `rewrite_decision`

Purpose: record the Smart Context decision for a runtime request.

Producer:

- `producer.component`: `smart_context`
- Normal severity: `info`; use `warn` for exact fallback, segment rollback, or
  blocked rewrite decisions.

Required top-level fields beyond the common envelope:

| Field | Constraint |
|---|---|
| `tenant_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `runtime_request_id` | string, length 1-128 |
| `rewrite_decision` | object |

Required `rewrite_decision` object fields:

| Field | Constraint |
|---|---|
| `rewrite_decision.mode` | `disabled`, `shadow`, `canary`, `live`, or `exact` |
| `rewrite_decision.decision` | `pass_through`, `shadow_only`, `rewrite`, `segment_rollback`, `exact_fallback`, or `blocked` |
| `rewrite_decision.fallback_exact` | boolean |
| `rewrite_decision.validation_result` | `passed`, `failed_protocol`, `failed_continuation`, `failed_tool_integrity`, `failed_json_integrity`, `failed_mandatory_reference`, or `not_applicable` |

Optional `rewrite_decision` fields:

| Field | Constraint | Source status |
|---|---|---|
| `rewrite_decision.rollout_mode` | `shadow`, `canary_in`, `canary_out`, `live`, `exact`, or `disabled` | AS-IS source telemetry; schema mapping `a validar adapter` |
| `rewrite_decision.rollout_canary_percent` | integer, 0-100 | AS-IS source telemetry; schema mapping `a validar adapter` |
| `rewrite_decision.transformed_segment_categories[]` | items from schema segment-category enum | AS-IS source telemetry; schema mapping `a validar adapter` |

AS-IS vs adapter:

- AS-IS source signal: Smart Context documents exact preservation rules,
  shadow/canary rollout, validation gates, whole-request fallback, and
  transformed segment telemetry.
- Required fork/adapter: normalize prodex Smart Context markers into the
  required `mode`, `decision`, `fallback_exact`, and `validation_result` enums.
  The exact marker-to-enum mapping is `a validar` in the fork because official
  docs do not define the Multica event schema.

## `spend_savings`

Purpose: record Smart Context size/token savings telemetry for a runtime
request.

Producer:

- `producer.component`: `smart_context`
- Normal severity: `debug` or `info`

Required top-level fields beyond the common envelope:

| Field | Constraint |
|---|---|
| `tenant_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `runtime_request_id` | string, length 1-128 |
| `spend_savings` | object; may be empty by schema, but should include available scrubbed telemetry |

Optional `spend_savings` fields:

| Field | Constraint | Source status |
|---|---|---|
| `spend_savings.estimated_input_tokens_before` | integer, minimum 0 | AS-IS source has token estimates; schema field mapping `a validar adapter` |
| `spend_savings.estimated_input_tokens_after` | integer, minimum 0 | AS-IS source has token estimates; schema field mapping `a validar adapter` |
| `spend_savings.estimated_input_tokens_saved` | integer, minimum 0 | `a validar adapter`; derive only from scrubbed estimate fields |
| `spend_savings.rewrite_ratio_percent` | number, 0-100 | AS-IS source telemetry |
| `spend_savings.body_bytes_before` | integer, minimum 0 | AS-IS source telemetry |
| `spend_savings.body_bytes_after` | integer, minimum 0 | AS-IS source telemetry |
| `spend_savings.body_bytes_saved` | integer, minimum 0 | AS-IS source telemetry |
| `spend_savings.estimator_confidence` | `high`, `medium`, `low`, or `unknown` | AS-IS source telemetry |
| `spend_savings.pressure_band` | `low`, `medium`, `high`, `critical`, or `unknown` | AS-IS source telemetry |

AS-IS vs adapter:

- AS-IS source signal: Smart Context telemetry documents byte counts, token
  estimates, rewrite ratio, estimator confidence, and pressure band while
  avoiding source contents by default.
- Required fork/adapter: emit only schema-approved field names. In particular,
  map prodex token estimate telemetry into the `estimated_input_tokens_*` names
  only when the source value is present and scrubbed.

## `guardrail`

Purpose: record a runtime guardrail decision from Rust L2.

Producer:

- `producer.component`: use the component that applied the guardrail:
  `gateway`, `smart_context`, `runtime_proxy`, `redeem`, `policy`, or
  `event_stream`.
- Normal severity: `info` for allowed/degraded, `warn` for blocked/fallback
  exact, `critical` for fail-closed kill-switch state.

Required top-level fields beyond the common envelope:

| Field | Constraint |
|---|---|
| `tenant_id` | string, length 1-128 |
| `session_id` | string, length 1-128 |
| `guardrail` | object |

Required `guardrail` object fields:

| Field | Constraint |
|---|---|
| `guardrail.guardrail_type` | `kill_switch`, `redaction`, `profile_isolation`, `provider_capability`, `smart_context_integrity`, `redeem_guard`, `state_backend`, or `single_router` |
| `guardrail.action` | `allowed`, `blocked`, `degraded`, `fallback_exact`, or `fail_closed` |

Optional `guardrail` fields:

| Field | Constraint |
|---|---|
| `runtime_request_id` | top-level string, length 1-128, when request-scoped |
| `guardrail.reason` | string, max length 256; scrubbed, no raw payload |
| `guardrail.effective_at` | `immediate`, `next_request`, `session_restart_required`, or `not_applicable` |

AS-IS vs adapter:

- AS-IS source signal: official gateway policy docs confirm guardrail controls
  and observability/audit surfaces; provider conformance docs confirm explicit
  degraded/rejected/unsupported classification direction; Smart Context docs
  confirm exact fallback on integrity risk.
- Required fork/adapter: normalize each source guardrail into the schema enum,
  include tenant/session dimensions, and fail closed instead of emitting
  incomplete guardrail events.

## Adapter Validation Checklist

The fork/adapter is complete only when all checks pass:

1. A minimal valid event for each of `selection`, `affinity`, `fallback`,
   `rewrite_decision`, `spend_savings`, and `guardrail` validates against
   `runtime-events.schema.json`.
2. Unknown top-level and nested fields are rejected before emission.
3. `redaction.secrets_present` is always exactly `false`.
4. Event generation never reads or copies raw prompt, raw tool output, cookies,
   API keys, OAuth material, bearer tokens, or full provider payloads.
5. `selection`, `affinity`, and `fallback` are emitted only from Rust runtime
   decision points and do not trigger Go routing.
6. `fallback.phase` is always `pre_commit` and `fallback.committed` is always
   `false`.
7. Smart Context events are emitted from Smart Context source signals only; Go
   must not rewrite payloads to manufacture them.
8. Guardrail events include `tenant_id` and `session_id`; request-scoped
   guardrails also include `runtime_request_id`.
9. The adapter has tests for every schema enum and required-field failure named
   by `docs/contracts/runtime-event-validation-spec.md`.
