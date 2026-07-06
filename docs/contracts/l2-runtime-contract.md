# L2 Runtime Contract - Go Control Plane to Rust Runtime Plane

Status: PRE-DEPLOY REQUIRED

Owner: Codex#5.5#A draft / Opus 4.8 Tech-Lead approval

Contract version: `rpp.l2.v1`

Scope note: this document defines the Multica target sidecar facade for the fork/polyglot milestone. It does not claim that prodex AS-IS already exposes every endpoint below; endpoint wiring in the fork is a validar by Codex#5.5#B/C.

## 1. Authority Boundary

Multica Go L4 is the cold control plane. It owns tenants, workspaces, approved accounts, product policy, budgets, lifecycle orchestration, kill-switch state, aggregated observability, and ledger ingestion.

Rust/prodex L2 is the hot runtime plane. It owns the local runtime proxy/gateway, session/profile affinity, pre-commit account selection, bounded pre-commit fallback, Smart Context rewrite/exact fallback, guarded reset-claim/redeem attempts, and runtime event emission.

The boundary is one-way for control and one-way for evidence:

```text
Go desired state  ->  Rust runtime decisions
Go ledger ingest  <-  Rust runtime events
```

Events are evidence only. Go must not use a Rust runtime event to re-decide a request that Rust has already committed to a profile/provider.

Primary design sources:

- `openspec/changes/rotation-parity-polyglot/design.md`
- `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`
- prodex README and docs: `README.md`, `docs/smart-context.md`, `docs/runtime-policy.md`, `docs/state-model.md`, `docs/provider-capabilities.md`

## 2. Transport

The sidecar contract uses HTTP JSON over loopback:

- bind to `127.0.0.1` or `[::1]` only in local sidecar mode;
- require an ephemeral high-entropy bearer token generated per sidecar start;
- never put the bearer token, OAuth material, API keys, cookies, provider tokens, raw prompts, raw tool outputs, or full request/response bodies in logs, events, check-ins, examples, or fixtures;
- include `contract_version: "rpp.l2.v1"` in every request, response, and event;
- set a timeout for every Go->L2 call;
- retry only idempotent calls with the same `request_id`;
- no FFI;
- no subprocess per request.

Headers:

```text
Authorization: Bearer <ephemeral-sidecar-token>
Content-Type: application/json
Accept: application/json
```

The placeholder above is not an example secret.

## 3. Shared Types

### Envelope

Every mutating request includes:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_20260704_000001",
  "tenant_id": "tenant-alpha"
}
```

`request_id` is the idempotency key for control-plane calls. `tenant_id`, `workspace_id`, `session_id`, `profile_id`, and `provider` are opaque identifiers, not secret material.

### Feature Keys

Feature keys used by policy and kill switch:

- `runtime_proxy`
- `gateway`
- `smart_context`
- `auto_redeem`
- `provider_bridge`

### Provider Capability

Go may send the provider capability matrix as desired state, but Rust enforces runtime behavior. Capability values follow the ADR shape:

```json
{
  "provider": "codex",
  "launch_mode": "native_cli",
  "auth_mode": "oauth_profile",
  "quota_mode": "codex_usage",
  "rotation_mode": "profile_pool",
  "continuation_mode": "response_id",
  "smart_context_mode": "proxy_rewrite",
  "reset_claim_mode": "codex_redeem",
  "validation_status": "verified"
}
```

Allowed `validation_status` values: `verified`, `inferred`, `not_validated`.

## 4. Endpoints

### HealthCheck

`GET /healthz`

Purpose: liveness only. It must not prove readiness for traffic.

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "alive",
  "sidecar": {
    "name": "prodex",
    "version": "0.246.0",
    "commit": "pin-to-validated-commit"
  }
}
```

`GET /readyz`

Purpose: readiness for production traffic.

Readiness must fail closed unless all required checks pass:

- policy loaded for the requested tenant or global default;
- approved account registry loaded;
- state backend for shared product state is Postgres/Redis, not shared SQLite;
- runtime log directory is writable;
- loopback bearer auth is active;
- kill-switch store is reachable;
- event stream can be established in production rollout mode;
- no startup integrity/attestation failure for the pinned sidecar build.

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "ready",
  "checks": [
    { "name": "policy_loaded", "status": "pass" },
    { "name": "approved_accounts_loaded", "status": "pass" },
    { "name": "shared_state_backend", "status": "pass" }
  ]
}
```

### ApplyPolicy

`POST /v1/policy/apply`

Go pushes desired policy, budgets, feature rollout, and kill-switch defaults. Rust must treat it as an authorization envelope, not as a request-by-request routing decision.

Request:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_policy_000001",
  "tenant_id": "tenant-alpha",
  "policy_id": "policy-20260704-a",
  "revision": 7,
  "allowed_providers": ["codex", "kiro"],
  "allowed_profiles": ["codex-main", "codex-backup"],
  "budgets": {
    "max_requests_per_session": 200,
    "max_estimated_input_tokens_per_request": 180000,
    "max_redeem_attempts_per_profile_per_day": 1
  },
  "smart_context": {
    "mode": "shadow",
    "canary_percent": 0,
    "exact_mode_allowed": true
  },
  "auto_redeem": {
    "enabled": false,
    "cooldown_seconds": 86400
  },
  "gateway": {
    "enabled": true,
    "adaptive_routing": "shadow"
  },
  "provider_capabilities": [],
  "kill_switches": []
}
```

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_policy_000001",
  "policy_id": "policy-20260704-a",
  "revision": 7,
  "applied": true
}
```

Idempotency: same `request_id`, `policy_id`, and `revision` must return the same effective result. Older revisions must be rejected unless explicitly marked as rollback by the control plane.

### RegisterAccounts

`POST /v1/accounts/register`

Go pushes approved profile metadata. Secrets are forbidden in this payload. Use profile references and local managed paths only.

Request:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_accounts_000001",
  "tenant_id": "tenant-alpha",
  "profiles": [
    {
      "profile_id": "codex-main",
      "provider": "codex",
      "profile_home": "$PRODEX_HOME/profiles/codex-main",
      "auth_mode": "oauth_profile",
      "status": "approved",
      "capability_ref": "codex.oauth_profile.v1"
    }
  ]
}
```

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_accounts_000001",
  "registered_profile_count": 1,
  "rejected_profiles": []
}
```

Rust must fail closed if `profile_home` resolves outside the managed profile root or if profile auth cannot be isolated from another profile.

### StartSession

`POST /v1/session/start`

Go starts a runtime session. Rust may pre-bind a profile, but Rust remains the only router for in-flight requests inside that session.

Request:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_session_start_000001",
  "tenant_id": "tenant-alpha",
  "workspace_id": "workspace-42",
  "task_id": "task-123",
  "session_id": "session-abc",
  "policy_id": "policy-20260704-a",
  "requested_provider": "codex",
  "requested_model": "gpt-5",
  "working_directory": "/workspaces/tenant-alpha/project",
  "profile_pool": ["codex-main", "codex-backup"],
  "continuation": {
    "previous_response_id": null,
    "session_binding_hint": null
  }
}
```

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_session_start_000001",
  "runtime_session_id": "rt-session-abc",
  "router_owner": "rust_l2",
  "event_stream_url": "http://127.0.0.1:43117/v1/events/stream?session_id=session-abc",
  "runtime_endpoint": "http://127.0.0.1:43117/v1/runtime/session-abc",
  "runtime_log_ref": "prodex-runtime-20260704-000001"
}
```

`runtime_log_ref` is a reference only, not a path containing user secrets.

### StopSession

`POST /v1/session/stop`

Stop is idempotent. Rust emits `session_stopped` once per effective stop transition.

Request:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_session_stop_000001",
  "tenant_id": "tenant-alpha",
  "session_id": "session-abc",
  "runtime_session_id": "rt-session-abc",
  "reason": "operator_requested"
}
```

Allowed reasons: `completed`, `operator_requested`, `policy_revoked`, `kill_switch`, `timeout`, `runtime_error`.

### KillSwitch

`POST /v1/killswitch/apply`

Go disables a feature by global, tenant, provider, profile, or session dimension. More-specific dimensions override less-specific dimensions.

Request:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_kill_000001",
  "tenant_id": "tenant-alpha",
  "scope": {
    "provider": "codex",
    "profile_id": "codex-main",
    "session_id": null
  },
  "feature": "smart_context",
  "state": "disabled",
  "reason": "operator_guardrail",
  "effective_at": "next_request"
}
```

Allowed `effective_at` values:

- `immediate`: applies before any new runtime commit;
- `next_request`: committed streams may finish, next request uses the disabled feature state;
- `session_restart_required`: L2 cannot safely change the feature for the running session and must emit a guardrail event.

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "request_id": "req_kill_000001",
  "applied": true,
  "effective_at": "next_request"
}
```

### RouteDecisionEvent

Rust reports routing decisions through the event stream using `event_type` values:

- `selection`
- `affinity`
- `fallback`

These are not Go commands. Go may ingest them into observability and ledger tables only.

### RuntimeEventStream

`GET /v1/events/stream`

The stream is newline-delimited JSON. Each line conforms to `docs/contracts/runtime-events.schema.json`.

Minimum required runtime event families:

- `selection`
- `affinity`
- `fallback`
- `redeem_attempt`
- `rewrite_decision`
- `spend_savings`
- `guardrail`

The stream may include lifecycle/control acknowledgements (`sidecar_started`, `policy_applied`, `session_started`, `session_stopped`, `kill_switch_applied`) when useful for readiness and audit.

## 5. Testable Single-Router Invariant

Invariant: for each `session_id`, exactly one runtime router has authority over in-flight request selection from session start until session stop.

Testable requirements:

1. `StartSession` response must include `router_owner: "rust_l2"`.
2. Go must persist the session as `runtime_router_owner = rust_l2` before sending traffic.
3. Go must not call any legacy Go rotation-router path for that `session_id` after successful `StartSession`.
4. Rust must emit exactly one `selection` event with `decision_phase: "pre_commit"` for each fresh runtime request that needs a new route.
5. Rust must emit `affinity` instead of fresh `selection` when continuation binding exists (`previous_response_id`, turn state, or session binding).
6. Fallback may occur only while `committed: false`.
7. Once any event for the request has `committed: true`, later `fallback` events for the same `runtime_request_id` must be rejected by schema/consumer validation unless they are marked `blocked` with a guardrail reason.
8. Events from Rust must never trigger Go to change the selected profile for the same `runtime_request_id`.

Suggested conformance assertion:

```text
group events by (tenant_id, session_id, runtime_request_id)
assert count(selection where decision_phase=pre_commit) <= 1
assert no fallback where committed=true
assert if affinity present then selected_profile_id equals affinity.binding_profile_id
assert Go legacy router invocation count for session_id == 0
```

## 6. Runtime Invariants

Required invariants inherited from ADR/design/prodex docs:

- profile auth isolation is stronger than convenience;
- hard continuation affinity beats fresh selection heuristics;
- rotation and fallback are pre-commit only;
- `previous_response_id`, turn state, and session bindings must not be overridden by load balancing;
- Smart Context control-plane and continuation metadata stay exact;
- Smart Context rewrite may affect eligible payload segments only and must fall back to exact pass-through when protocol, continuation, tool-call structure, critical signal, mandatory references, or JSON integrity are at risk;
- reset-claim/redeem is low-priority, guarded, idempotent, and never attempted merely for thin/critical windows when another eligible profile exists;
- generic upstream `429` is not account quota unless payload identifies quota or rate-limit exhaustion;
- route-scoped health must not poison unrelated routes without evidence;
- runtime hot paths must avoid broad disk reads, quota probes, and blocking state saves;
- runtime logs are diagnostics, not source of truth;
- shared product state uses Postgres/Redis; shared SQLite is forbidden;
- secrets are redacted in diagnostics, runtime logs, traces, events, evidence, and check-ins.

## 7. Failure Behavior

Go must fail closed when:

- sidecar readiness fails;
- policy apply fails;
- account registration rejects a required profile;
- selected profile metadata is missing;
- profile home or working directory validation fails;
- bearer auth cannot be established;
- shared SQLite is detected for shared product state;
- kill switch cannot be confirmed;
- production event stream cannot be established.

Rust must fail closed when:

- profile switch points to invalid or non-isolated auth;
- continuation binding references an unavailable profile;
- provider capability is unknown or not validated for the requested path;
- Smart Context validation fails;
- fallback budget is exhausted before commit;
- reset/redeem guard conditions are not satisfied;
- kill switch disables the requested runtime feature.

## 8. Required Evidence Before DONE

- JSON schema validates.
- Contract examples validate against their described shape where applicable.
- HealthCheck and readiness smoke planned for Go integration.
- Policy apply, account register, session start/stop, kill switch, and event stream smoke planned for Go integration.
- Runtime event redaction smoke planned with no secret-bearing examples.
- Single-router invariant has an explicit conformance assertion.
- Handoff notes call out endpoints as Multica target facade, not proven prodex AS-IS RPC.
