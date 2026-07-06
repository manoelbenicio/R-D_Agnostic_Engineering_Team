# RPP L2 Runtime Contract

Status: P1 contract baseline
Contract version: `rpp.l2.v1`
Event schema: [`rpp-l2-v1-event-schema.json`](./rpp-l2-v1-event-schema.json)

## Purpose

This document defines the formal interface between Multica Go L4, the cold
control plane, and prodex/Rust L2, the hot runtime plane. Go owns tenants,
workspaces, approved accounts, policy, budgets, lifecycle, kill-switch state,
and ledger/observability ingestion. L2 owns in-flight runtime routing:
session/profile affinity, pre-commit selection, bounded pre-commit fallback,
Smart Context decisions, guarded redeem/reset attempts, MCP tool-call
continuation integrity, and runtime event emission.

The central invariant is one runtime router per session:

```text
Go desired state  ->  L2 runtime decisions
Go evidence ingest <- L2 runtime events
```

After `StartSession` returns `router_owner: "rust_l2"`, Go must persist that
owner before sending traffic and must not run legacy Go rotation/router paths
for that `session_id`. L2 events are evidence only; they must never cause Go to
re-route an already committed runtime request.

## Transport And Envelope

The target sidecar contract is HTTP JSON over loopback with an ephemeral bearer
token. This contract describes the versioned facade required for integration;
it does not claim prodex AS-IS already exposes every endpoint.

Every control request includes:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_000001",
  "tenant_id": "tenant-alpha"
}
```

`request_id` is the idempotency key. Identifiers such as `tenant_id`,
`session_id`, `profile_id`, and `runtime_request_id` are opaque and non-secret.
Payloads, logs, events, fixtures, and evidence must not include bearer tokens,
OAuth material, API keys, cookies, raw prompts, raw tool outputs, or full
provider request/response bodies.

## Operation Summary

| Operation | Direction | Endpoint / channel |
|---|---|---|
| HealthCheck | Go -> L2 | `GET /healthz`, `GET /readyz` |
| ApplyPolicy | Go -> L2 | `POST /v1/policy/apply` |
| RegisterAccounts | Go -> L2 | `POST /v1/accounts/register` |
| StartSession | Go -> L2 | `POST /v1/session/start` |
| StopSession | Go -> L2 | `POST /v1/session/stop` |
| RouteDecisionEvent | L2 -> Go | RuntimeEventStream event family |
| RuntimeEventStream | L2 -> Go | `GET /v1/events/stream` NDJSON |
| KillSwitch | Go -> L2 | `POST /v1/killswitch/apply` |

## HealthCheck

Direction: Go -> L2.

Payload:

- Liveness: `GET /healthz` has no body. Response includes
  `contract_version: "rpp.l2.v1"`, `status: "alive"`, and non-secret sidecar
  build metadata.
- Readiness: `GET /readyz` has no body. Response includes
  `contract_version: "rpp.l2.v1"`, `status: "ready"`, and named checks.

Error handling:

- Go fails closed if contract version differs, liveness status is not `alive`,
  readiness status is not `ready`, any required readiness check is not `pass`,
  bearer auth fails, or the endpoint is not loopback.
- Readiness must fail closed when policy, approved accounts, state backend,
  runtime log directory, kill-switch store, event stream, or build attestation
  is unavailable.

Idempotency:

- Safe and idempotent. Go may retry with normal health-check cadence.

## ApplyPolicy

Direction: Go -> L2.

Payload:

- Request references `rpp.l2.v1`, `request_id`, `tenant_id`, `policy_id`,
  `revision`, allowed providers/profiles, budgets, Smart Context rollout,
  auto-redeem guard config, gateway policy, provider capabilities, and
  kill-switch defaults.
- Response echoes `contract_version`, `request_id`, `policy_id`, `revision`,
  and `applied: true`.

Error handling:

- Go fails closed when the response contract version differs, `applied` is not
  true, revision is rejected, or L2 reports invalid policy state.
- L2 rejects older revisions unless Go marks the request as an explicit
  rollback in policy metadata.

Idempotency:

- Same `request_id`, `policy_id`, and `revision` returns the same effective
  result. Different payload under the same `request_id` is a protocol error.

## RegisterAccounts

Direction: Go -> L2.

Payload:

- Request contains approved profile metadata only: `profile_id`, `provider`,
  managed `profile_home`, `auth_mode`, `status: "approved"`, and
  `capability_ref`.
- Secrets are forbidden. Auth material remains in managed profile homes.
- Response includes registered count and `rejected_profiles`.

Error handling:

- Go fails closed if any required profile is rejected, if response contract
  version differs, or if L2 cannot prove profile isolation.
- L2 rejects profiles whose `profile_home` escapes the approved root, whose auth
  mode is unsupported, whose status is not approved, or whose capability is
  unknown.

Idempotency:

- Same `request_id` and profile set is idempotent. Re-registering the same
  approved profile refreshes metadata without duplicating profile state.

## StartSession

Direction: Go -> L2.

Payload:

- Request includes `workspace_id`, `task_id`, `session_id`, `policy_id`,
  requested provider/model, working directory, approved `profile_pool`, and
  continuation hints such as `previous_response_id` or session binding.
- Response includes `runtime_session_id`, `router_owner: "rust_l2"`,
  `event_stream_url`, `runtime_endpoint`, and scrubbed `runtime_log_ref`.

Error handling:

- Go fails closed if L2 does not return `router_owner: "rust_l2"`, contract
  version differs, no event stream/runtime endpoint is provided, or profile pool
  validation fails.
- L2 fails closed if requested continuation state points to an unavailable or
  non-isolated profile.

Idempotency:

- Same `request_id` and `session_id` returns the same `runtime_session_id` and
  router owner. A conflicting retry for an already-started session is rejected.

Session affinity:

- L2 must preserve tool-call, continuation, turn-state, and session affinity
  across rotation. Hard affinity beats fresh load balancing. Rotation and
  fallback are pre-commit only.

## StopSession

Direction: Go -> L2.

Payload:

- Request includes `session_id`, `runtime_session_id`, and a stop reason such as
  `completed`, `operator_requested`, `policy_revoked`, `kill_switch`,
  `timeout`, or `runtime_error`.
- Effective stop emits one `session_stopped` runtime event.

Error handling:

- Go treats transport/auth/contract failures as stop-confirmation failures and
  preserves local evidence for recovery.
- L2 must drain or interrupt according to policy and must not emit secret
  details in stop diagnostics.

Idempotency:

- Idempotent. Repeating the same stop request for an already stopped session
  returns success/no-op and must not emit duplicate effective stop transitions.

## RouteDecisionEvent

Direction: L2 -> Go.

Payload:

- Delivered through `RuntimeEventStream`.
- Covers `route_decision` events and may decompose decisions into selection,
  affinity, or fallback subtypes in the event payload.
- Events include `tenant_id`, `session_id`, `runtime_request_id`, provider,
  profile, route phase, decision reason, and committed state.

Error handling:

- Go rejects events that fail
  [`rpp-l2-v1-event-schema.json`](./rpp-l2-v1-event-schema.json), have
  `contract_version` other than `rpp.l2.v1`, declare
  `redaction.secrets_present: true`, or contain unknown top-level fields.
- Rejected route events never trigger a Go-side route change.

Idempotency:

- Events are append-only evidence. `event_id` is unique. Duplicate `event_id`
  ingestion is a no-op.

## RuntimeEventStream

Direction: L2 -> Go.

Payload:

- `GET /v1/events/stream` returns newline-delimited JSON events.
- Every line validates against
  [`rpp-l2-v1-event-schema.json`](./rpp-l2-v1-event-schema.json).
- Required event types include lifecycle/control events, route decisions, MCP
  tool-call events, and kill-switch events:
  `sidecar_started`, `session_started`, `session_stopped`, `route_decision`,
  `policy_applied`, `health_status`, `mcp_tool_call`, `mcp_tool_result`, and
  `kill_switch_activated`.

MCP coverage:

- `mcp_tool_call` events surface tool name, non-secret call id, phase, and
  scrubbed input reference/hash.
- `mcp_tool_result` events surface the matching call id, status, scrubbed output
  reference/hash, and continuation binding.
- Tool-call event ordering and continuation state must survive profile rotation.
  If preserving tool-call state is unsafe, L2 must emit a guardrail/failure
  event and fail closed rather than silently switching profiles mid-tool.

Error handling:

- Go closes/retries the stream according to integration policy on transport
  errors, but invalid/secret-bearing events are rejected before ledger or
  observability sinks.
- L2 must not place raw prompts, raw tool payloads, or secrets on the stream.

Idempotency:

- Event ingestion is idempotent by `event_id`.

## KillSwitch

Direction: Go -> L2.

Payload:

- Request includes scope (`provider`, `profile_id`, `session_id` as needed),
  feature (`runtime_proxy`, `gateway`, `smart_context`, `auto_redeem`,
  `provider_bridge`), state (`disabled` or `enabled`), reason, and
  `effective_at` (`immediate`, `next_request`, or
  `session_restart_required`).
- Response includes `applied: true` and confirmed `effective_at`.

Error handling:

- Go fails closed if kill switch cannot be confirmed, if response contract
  version differs, if `applied` is false, or if `effective_at` is unknown.
- L2 must enforce the most specific matching scope and emit
  `kill_switch_activated` when a feature is disabled.

Idempotency:

- Same `request_id` and scope/feature/state is idempotent. Reapplying an
  already-effective switch confirms the current state.

## Testable Single-Router Assertions

For each `session_id`:

1. `StartSession` returns and Go persists `router_owner: "rust_l2"` before
   runtime traffic.
2. Go legacy rotation/router invocation count is zero after L2 ownership.
3. L2 emits at most one pre-commit route decision per fresh
   `runtime_request_id`.
4. Continuation or MCP tool-call state emits affinity rather than a fresh
   profile selection.
5. Fallback is allowed only while `committed` is false.
6. Runtime events are observability/ledger only and do not alter the in-flight
   route in Go.

## Failure Policy

The contract is fail-closed. Missing auth, invalid profile home, failed
readiness, rejected account registration, invalid policy, unknown provider
capability, malformed event, secret-bearing event, or unconfirmed kill-switch
state blocks the affected operation. Evidence must be scrubbed.
