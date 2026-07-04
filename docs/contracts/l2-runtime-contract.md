# L2 Runtime Contract - Go Control Plane to Rust Runtime Plane

Status: PRE-DEPLOY REQUIRED

Owner: Codex#5.5#A (draft) / **Opus 4.8 — Sr. SME AI Solutions Architect & Principal Agentic Planning Orchestrator** (Tech-Lead / POC) approval

## 1. Scope

This contract defines the boundary between:

- Multica Go L4: product/control plane.
- Rust/prodex L2: runtime/hot path plane.

The contract prevents competing routers. Go decides desired state. Rust decides
the in-flight runtime request inside the authorized envelope.

## 2. Authority Model

Go owns:

- tenant and workspace identity;
- approved accounts;
- account enrollment state;
- policy and budget;
- kill switch state;
- orchestration lifecycle;
- aggregated observability and ledger ingestion;
- operator approval and audit trail.

Rust/prodex owns:

- runtime proxy;
- gateway;
- session/profile affinity;
- route selection before commit;
- retry/fallback before commit;
- Smart Context rewrite/fallback;
- redeem/reset attempt;
- runtime event emission.

Rust must not create product policy. Go must not re-route committed runtime
requests.

## 3. Transport

Preferred transport:

```text
HTTP JSON over 127.0.0.1
```

Required:

- bind only to loopback for local sidecar mode;
- bearer token generated per sidecar start;
- token passed by env or secure local process channel;
- schema version on every request and event;
- timeouts on every call;
- retry only for idempotent control-plane calls;
- no FFI;
- no subprocess-per-request.

## 4. Versioning

All requests include:

```json
{
  "contract_version": "rpp.l2.v1"
}
```

Backward incompatible changes require:

- new contract version;
- updated schema;
- migration note;
- Opus approval.

## 5. Endpoints

### GET /healthz

Purpose: liveness.

Response:

```json
{
  "contract_version": "rpp.l2.v1",
  "status": "alive",
  "build": {
    "name": "prodex",
    "version": "0.246.0",
    "commit": "7750da9b6a5c91a6d429e18e6a4d422cab4bc144"
  }
}
```

### GET /readyz

Purpose: readiness for traffic.

Must validate:

- policy loaded;
- state backend reachable;
- account registry loaded;
- runtime log directory writable;
- kill switch reachable;
- no shared SQLite backend selected.

### POST /v1/policy/apply

Go pushes desired policy.

Required fields:

- `tenant_id`
- `policy_id`
- `allowed_profiles`
- `allowed_providers`
- `smart_context`
- `auto_redeem`
- `gateway`
- `budgets`
- `kill_switches`

Policy apply is idempotent by `policy_id`.

### POST /v1/accounts/register

Go pushes approved account/profile metadata.

Secrets are not allowed in this payload. Use profile references and local path
references only.

Required fields:

- `tenant_id`
- `profiles[].profile_id`
- `profiles[].provider`
- `profiles[].profile_home`
- `profiles[].auth_mode`
- `profiles[].status`

### POST /v1/session/start

Go starts a runtime session.

Required fields:

- `tenant_id`
- `workspace_id`
- `task_id`
- `session_id`
- `policy_id`
- `requested_provider`
- `requested_model`
- `working_directory`
- `profile_pool`

Rust returns:

- `runtime_session_id`
- local endpoint or process handle;
- effective profile binding if preselected;
- runtime log path.

### POST /v1/session/stop

Go stops a runtime session.

Must be idempotent. Stop reason is required.

### POST /v1/killswitch/apply

Go applies kill switch.

Dimensions:

- global;
- tenant;
- provider;
- profile;
- feature.

Features:

- `smart_context`
- `auto_redeem`
- `gateway`
- `provider_bridge`
- `runtime_proxy`

Kill switch must take effect for new requests immediately. If disabling a
feature cannot affect an already committed stream, Rust emits an event with
`effective_at = next_request`.

### GET /v1/events/stream

Rust emits newline-delimited JSON events.

Go ingests events but must not use events to re-decide an already committed
request.

Event schema:

```text
docs/contracts/runtime-events.schema.json
```

## 6. Runtime Invariants

Required:

- one runtime router per session;
- rotate only before commit;
- continuation affinity beats fresh selection;
- `previous_response_id` binding must not be overridden by load balance;
- generic `429` is not account quota unless payload identifies quota/rate;
- route-scoped health must not poison unrelated lanes without evidence;
- no broad disk reads on request/stream commit path;
- no secrets in request/response/event logs.

## 7. Failure Behavior

Go must fail closed when:

- sidecar is not ready;
- policy apply fails;
- selected profile missing;
- profile home missing auth material;
- sidecar reports forbidden shared SQLite;
- kill switch cannot be confirmed;
- event stream cannot be established for production rollout.

Rust must fail closed when:

- profile switch points to invalid auth;
- continuation binding references unavailable profile;
- Smart Context validation fails;
- reset/redeem guard conditions are not satisfied;
- provider capability is unknown for requested path.

## 8. Required Pre-Deploy Test Evidence

- `GET /readyz` success.
- policy apply success.
- account register success with no secret payload.
- session start/stop smoke.
- kill switch smoke.
- event stream smoke.
- redaction smoke.
- Go container green.
- Rust sidecar healthy.
