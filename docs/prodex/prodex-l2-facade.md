# prodex L2 Facade Target

Status: TARGET SPEC. The exact facade endpoints below are Multica fork targets,
not confirmed prodex-as-is HTTP endpoints. Anything not confirmed in official
prodex docs/repo is marked `a validar`.

Pinned prodex source: official `github.com/christiandoxa/prodex` tag `0.246.0`,
commit `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`, Apache-2.0.

Runtime event contract: every Rust L2 event emitted to Go L4 must validate
against `docs/contracts/runtime-events.schema.json` before it is accepted as
observability or ledger input.

## Source Basis

Official prodex sources used:

- `docs/architecture.md`: command path, runtime proxy hot path, state and
  persistence, observability, session/profile/shared Codex FS, crate boundaries.
- `docs/runtime-policy.md`: policy file/env model, gateway keys, gateway state
  backends, runtime proxy contract, structured runtime log requirements.
- `docs/state-model.md`: profile registry, profile auth isolation,
  continuation bindings, quota/health, runtime logs.
- `docs/provider-conformance.md`: provider conformance split and target provider
  core direction.
- `docs/provider-capabilities.md`: generated provider capability matrix.
- `docs/smart-context.md`: Smart Context safety, rollout, telemetry, replay,
  and remaining risks.
- `docs/deployment.md`: gateway deployment, persistent paths, admin endpoints,
  file/SQLite single-node limit, Postgres/Redis shared state options.
- `Cargo.toml`: workspace crate list and package metadata.

Local Multica contract source:

- `docs/contracts/runtime-events.schema.json`: event types, producer components,
  redaction rules, and conditional requirements.

## Facade Endpoints

All facade endpoints are target loopback sidecar endpoints for the forked Rust
L2. Official prodex docs confirm the backing runtime capabilities and crates,
but not these exact facade endpoint names.

| Facade endpoint | Target purpose | prodex backing to reuse | Status |
|---|---|---|---|
| `GET /healthz` | Process liveness for the Rust L2 sidecar. Should prove the sidecar process can answer locally without consulting broad runtime state. | `prodex-app` command orchestration, runtime launch scaffolding, `prodex-runtime-log` path helpers, `prodex-runtime-broker` DTOs/registry where useful. | `a validar`: exact health endpoint is not an official prodex-as-is surface. |
| `GET /readyz` | Readiness for accepting Go desired-state and runtime sessions. Should check policy loaded, state backend reachable, runtime log path usable, and optional gateway state backend readiness. | `prodex-runtime-policy`, `prodex-runtime-tuning`, `prodex-state`, `prodex-runtime-state`, `prodex-runtime-store`, `prodex-runtime-doctor`, gateway state backends documented in `docs/runtime-policy.md` and `docs/deployment.md`. | `a validar`: readiness composition is target-sidecar behavior. |
| `POST /policy.apply` | Apply Go L4 desired policy: routing mode, budgets, Smart Context mode, gateway controls, provider/profile constraints, and kill-switch state. | `prodex-runtime-policy`, `prodex-runtime-tuning`, `crates/prodex-app/src/runtime_policy.rs`, gateway policy keys, Smart Context env/policy controls, runtime proxy tuning contract. | `a validar`: prodex-as-is reads `policy.toml`/env; no official hot policy-apply endpoint confirmed. |
| `POST /accounts.register` | Register approved tenant/provider/profile accounts from Go L4 into Rust L2 runtime scope. | Profile/state surfaces in `prodex-state`, `prodex-core`, `prodex-shared-codex-fs`, `prodex-profile-identity`, `prodex-profile-export`, `prodex-secret-store`; gateway admin keys/SCIM are separate gateway admin surfaces. | `a validar`: approved-account registration facade is Multica target behavior, not confirmed prodex-as-is API. |
| `POST /session.start` | Start one runtime session under Rust L2 authority and bind the Go session id to runtime state. | `prodex-runtime-launch`, app `runtime_launch/*`, app `runtime_proxy/*`, `prodex-runtime-state` continuation/session binding models, `prodex-session-store`, provider runtime crates. | `a validar`: official prodex run/session commands exist, but this sidecar session-start endpoint is target fork work. |
| `POST /session.stop` | Stop one runtime session, release/flush runtime state as allowed, and emit a terminal event. | app `runtime_proxy/*`, `prodex-runtime-state`, `prodex-runtime-store`, `prodex-session-store`, runtime persistence, runtime broker/log crates. | `a validar`: exact stop endpoint and shutdown semantics require fork implementation and tests. |
| `POST /killswitch` | Disable Smart Context, gateway, redeem, provider, profile, or tenant traffic before the next request. | Runtime policy/tuning crates, Smart Context rollout controls, gateway guardrail/policy keys, runtime proxy selection/admission/fallback code. | `a validar`: local ADR requires kill switches; official prodex docs confirm controls but not a unified kill-switch facade. |
| `GET /events.stream` | Stream Rust L2 runtime events to Go L4 for observability and ledger only. | `prodex-runtime-log`, `prodex-runtime-broker`, `prodex-runtime-broker-log`, `prodex-runtime-metrics`, `prodex-audit-log`, `prodex-redaction`, gateway observability sinks, structured runtime log markers. | `a validar`: prodex has logs/audit/metrics, but schema-valid Multica event streaming is target fork work. |

## Event Schema Requirements

`docs/contracts/runtime-events.schema.json` is the acceptance gate for all
events crossing from Rust L2 to Go L4.

Required top-level fields:

- `contract_version` must be `rpp.l2.v1`.
- `event_id`, `event_type`, `occurred_at`, `severity`, `producer`, and
  `redaction` are required.
- `producer.plane` must be `rust_l2`.
- `producer.component` must be one of `sidecar`, `runtime_proxy`, `gateway`,
  `smart_context`, `redeem`, `policy`, or `event_stream`.

Allowed `event_type` values include:

- sidecar/control: `sidecar_started`, `sidecar_ready`, `policy_applied`,
  `account_registered`, `session_started`, `session_stopped`,
  `kill_switch_applied`;
- routing/runtime: `selection`, `affinity`, `fallback`, `quota_snapshot`,
  `error`;
- redeem: `redeem_attempt`, `redeem_result`;
- Smart Context: `rewrite_decision`, `spend_savings`;
- gateway/guardrails: `gateway_request`, `gateway_response`, `guardrail`.

Rules:

1. Go L4 must reject or quarantine any Rust L2 event that fails the schema.
2. Runtime events are observability/ledger only; they must not re-decide an
   already in-flight request.
3. Event payloads must not contain secrets, bearer tokens, OAuth material, API
   keys, cookies, raw prompts, raw tool outputs, or full provider
   request/response bodies.
4. Event fields must use opaque ids for tenant, workspace, task, session,
   runtime session, request, policy, provider, profile, and model identifiers.
5. Endpoint handlers that emit events must map them to the schema event types:
   `/policy.apply -> policy_applied`, `/accounts.register ->
   account_registered`, `/session.start -> session_started`,
   `/session.stop -> session_stopped`, `/killswitch ->
   kill_switch_applied`, `/events.stream -> sidecar_ready` or stream lifecycle
   events as applicable.

## Endpoint-To-Crate Notes

### `healthz`

Confirmed backing: prodex has runtime launch/log/broker/doctor support, and
official deployment docs describe gateway process deployment.

`a validar`: no official prodex doc confirms a generic sidecar liveness endpoint
named `/healthz`. The fork must define whether health is pure process liveness
or also includes loopback listener binding.

### `readyz`

Confirmed backing: prodex policy, runtime log, state, gateway, and deployment
docs identify the components readiness must inspect. File/SQLite state are
single-node; Postgres/Redis are documented shared-state options for gateway
state.

`a validar`: readiness must be precise enough not to block on hot request paths
or broad disk reads, and must distinguish process liveness from runtime
readiness.

### `policy.apply`

Confirmed backing: prodex reads `policy.toml`/environment overrides; runtime
proxy tuning and gateway policy keys are documented. Smart Context rollout has
official shadow/canary controls.

`a validar`: hot application of Go desired-state through a sidecar endpoint is
not confirmed in official prodex docs. The fork must define accepted fields,
versioning, persistence, and kill-switch precedence without moving runtime
decision authority to Go.

### `accounts.register`

Confirmed backing: prodex profiles, profile identity, shared Codex filesystem,
secret-store primitives, state models, and gateway admin/SCIM surfaces exist.

`a validar`: Multica approved-account registration is not the same as prodex
profile commands or gateway virtual-key admin. The fork must define how Go
approved accounts map onto prodex profiles without mutating profile auth
unexpectedly.

### `session.start` / `session.stop`

Confirmed backing: official prodex architecture maps runtime launch, runtime
proxy, hard affinity, session/profile binding state, and session store crates.

`a validar`: official prodex CLI commands are not a long-lived sidecar
session-control API. The fork must define idempotency, duplicate start behavior,
stop semantics, state flush, and event emission.

### `killswitch`

Confirmed backing: prodex has policy/tuning controls, Smart Context rollout
controls, gateway guardrails, runtime proxy selection/admission, and redeem
guards.

`a validar`: a unified tenant/provider/profile kill-switch facade is Multica
target behavior. The fork must prove kill switches take effect before the next
request and cannot be bypassed by profile affinity or adaptive routing.

### `events.stream`

Confirmed backing: prodex has structured runtime logs, audit log helpers,
broker log parsing, metrics rendering, redaction helpers, and gateway
observability sinks.

`a validar`: official prodex docs do not confirm direct emission of
`docs/contracts/runtime-events.schema.json` events. The fork must implement a
schema-valid adapter from prodex runtime markers/audit/metrics into the Multica
runtime event stream.

## Non-Goals

- Do not add Go runtime routing logic behind these endpoints.
- Do not treat runtime events as commands.
- Do not use file/SQLite shared state for multi-host L2 deployment.
- Do not add payload rewriting outside Rust L2 Smart Context.
- Do not claim prodex-as-is already exposes these exact facade endpoints.
