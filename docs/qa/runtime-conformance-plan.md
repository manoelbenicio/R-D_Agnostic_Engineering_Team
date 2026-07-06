# Runtime Conformance Plan

Status: PLAN + ACCEPTANCE CRITERIA. LIVE EXECUTION F0-GATED.

Owner: GLM#52#A / Opus 4.8 approval

Execution boundary: this document defines QA conformance criteria, dry-run
evidence, and the future PROD validation checklist. Any real sidecar traffic,
real provider traffic, live prodex rollout, or PROD validation is **F0-GATED**
and requires owner approval.

## 1. Purpose

Prove that Multica Go + prodex/Rust L2 preserves the required runtime
invariants before any real PROD deploy.

## 2. Required Invariants

- one runtime router per session;
- Go sends desired state only;
- Rust decides in-flight runtime route;
- rotate only before commit;
- continuation affinity beats selection heuristics;
- profile switch fails closed;
- Smart Context preserves protocol/tool-call/continuation integrity;
- kill switch works;
- no secrets in logs/events/evidence.

## 3. Smoke Tests

### S1 - Sidecar readiness

Expected:

- `healthz` ok;
- `readyz` ok;
- Postgres reachable;
- no shared SQLite backend;
- kill switch state readable.

### S2 - Policy apply

Expected:

- valid policy accepted;
- unknown tenant rejected;
- policy with disabled provider rejected or ignored according to schema;
- no secrets in payload.

### S3 - Account register

Expected:

- approved profiles registered by reference only;
- missing profile home fails closed;
- raw auth material rejected.

### S4 - Session start/stop

Expected:

- session starts with policy id;
- runtime event emitted;
- session stops idempotently.

### S5 - Kill switch

Expected:

- disable Smart Context for tenant;
- next request exact/pass-through;
- event emitted.

## 4. Conformance Tests

### C1 - One router

Inject a session with Go desired policy. Verify only Rust/prodex emits
`selection`, `affinity`, or `fallback` runtime events. Go must not run runtime
load-balance/fallback for request in flight.

### C2 - Fail-closed profile switch

Switch from valid profile A to invalid/missing profile B.

Expected:

- task does not reuse profile A silently;
- event `profile_switch_fail_closed` or equivalent;
- session fails before commit.

### C3 - Continuation affinity

Start response with profile A. Send continuation with `previous_response_id`.

Expected:

- continuation remains bound to profile A;
- load balance does not move it to profile B.

### C4 - Precommit fallback

Simulate quota/rate/provider failure before first committed response.

Expected:

- fallback attempted before commit;
- event emitted;
- no mid-stream rotation.

### C5 - Smart Context exact fallback

Feed malformed artifact reference or protocol-sensitive payload.

Expected:

- exact fallback;
- no corrupted JSON;
- tool-call ids preserved;
- continuation ids preserved.

### C6 - Event redaction

Inject fake secrets into env/log path and error path.

Expected:

- scrubbed output only;
- event schema has `secrets_present=false`.

## 4A. Testable C1-C6 Acceptance Matrix

All live executions in this matrix are F0-GATED. Before F0, acceptable evidence
is static review, unit/contract evidence, smoke dry-run output, and schema
validation proof.

| Case | Setup / stimulus | Dry-run evidence now | F0-GATED live proof | Pass criteria |
|---|---|---|---|---|
| C1 - One router | Start an L2-owned session with Go desired policy and `router_owner=rust_l2`. Send one fresh request. | Unit/contract evidence that L2-owned tasks suppress legacy Go rotation paths; planned event contract for `selection`/`affinity`/`fallback`. | Controlled real session; collect router counters and event ids. | `go_rotation_decision_count == 0`; `go_fallback_invocation_count == 0`; Rust emits at most one pre-commit `selection` for a fresh request; events are observability/ledger only. |
| C2 - Fail-closed profile switch | Request profile B that is missing, disabled, unhealthy, or lacks required auth while profile A exists. | Static policy/daemon review and dry-run launch failure evidence. | Controlled session attempts profile switch before first commit. | No silent reuse of profile A; request fails before commit or emits schema-valid `guardrail`/`error`; no secret/auth material in evidence. |
| C3 - Continuation affinity | Start response on profile A; send continuation with `previous_response_id`, turn state, or session binding while profile B is eligible. | State/invariant docs show hard affinity; event schema requires `affinity.overrode_fresh_selection=true`. | Real multi-turn continuation under L2 authority. | Bound continuation remains on profile A; no load-balance/fallback to profile B unless Rust marks binding unavailable by explicit policy; Go does not override owner. |
| C4 - Precommit fallback | Simulate quota exhaustion, rate limit, unhealthy profile, provider capability rejection, or precommit transport failure before upstream commit. | Contract evidence that fallback event requires `phase=pre_commit` and `committed=false`. | Controlled request with injectable precommit failure. | Fallback occurs only before first committed response/stream; `fallback.committed=false`; no fallback after output starts; fallback budget is bounded. |
| C5 - Smart Context exact fallback | Feed malformed artifact reference, invalid JSON candidate, protocol-sensitive payload, continuation/tool-call risk, or mandatory-reference risk. | `docs/qa/smart-context-shadow-canary-plan.md` exact fallback probes and prodex Smart Context invariants. | Controlled Smart Context shadow/canary/live gate execution. | Original or exact-safe body is sent; JSON/tool/continuation integrity preserved; `rewrite_decision.fallback_exact=true` when fallback is used; kill switch forces next request exact. |
| C6 - Event redaction | Inject fake secrets into allowed test-only env/log/error paths and malformed runtime events. | Go validation spec rejects `redaction.secrets_present != false`; dry-run redaction smoke evidence. | Controlled L2 event emission and ingest with fake secret probes. | No fake secret reaches logs/events/evidence; malformed or secret-bearing events are rejected before ledger/observability writes; accepted events have `contract_version=rpp.l2.v1` and `secrets_present=false`. |

## 4B. Common Test Harness Requirements

- Every test must record `tenant_id`, `session_id`, `runtime_request_id` where
  available, but only as opaque identifiers.
- Every event sample must validate against
  `docs/contracts/runtime-events.schema.json` and
  `docs/contracts/runtime-event-validation-spec.md`.
- Runtime events are evidence only; no test may use an event to drive Go-side
  runtime rerouting.
- Live harnesses must support immediate stop/kill switch and rollback to exact
  pass-through/raw Codex path.
- Evidence must be scrubbed before it is written under `.deploy-control`.
- Any case with missing evidence is `BLOCKED`, not pass.

## 5. Replay Coverage

Minimum replay scenarios:

- 30+ turn continuation;
- repeated build/test output;
- compiler/runtime errors;
- large diffs;
- repository navigation;
- multi-file refactor;
- changing static instructions;
- missing/corrupted artifacts;
- duplicate tool calls/output;
- noisy binary-like command output;
- failure recovery;
- 16k/32k/128k/200k context windows if available.

## 6. PROD Validation Entry Criteria

Before controlled PROD validation:

- smoke S1-S5 pass;
- conformance C1-C6 pass or accepted exception recorded;
- deploy runbook approved;
- rollback command tested;
- kill switch tested;
- redaction smoke passed.

## 6A. PROD Validation Checklist

Status: F0-GATED. Do not execute this checklist until owner approval names the
UTC window, tenant/profile scope, rollback owner, and maximum Smart Context mode.

### Pre-Window Checklist

| Item | Required evidence | Gate |
|---|---|---|
| Owner approval | Written approval with window, scope, runtime mode, and rollback owner. | F0-GATED |
| Pin/integrity | prodex version/commit pin and integrity evidence from deployment runbook. | Required |
| State backend | Shared PROD state is Postgres/Redis where applicable; no shared file/SQLite multi-host state. | Required |
| Kill switch | Smart Context, gateway/runtime, and prodex launch disable paths are dry-run tested. | Required |
| Event validation | Runtime event ingest rejects unknown types, malformed fields, and `secrets_present=true`. | Required |
| Redaction | Fake secret smoke passes; evidence/log paths are scrubbed. | Required |
| Rollback | Raw Codex/prodex-as-is rollback path documented and dry-run tested. | Required |
| Smart Context | Shadow default and canary percent are set according to `docs/qa/smart-context-shadow-canary-plan.md`. | Required |

### Controlled PROD Steps

All steps below are F0-GATED live actions:

1. Start prodex-backed session for the approved scope only.
2. Verify sidecar readiness and policy/account/session surfaces.
3. Run C1 one-router assertion and collect counters.
4. Run C3 continuation affinity on a bounded multi-turn task.
5. Run C4 precommit fallback only with approved injectable/safe failure; do not
   force real account damage.
6. Run C5 Smart Context in shadow first; canary/live only after separate owner
   approval per gate.
7. Run C6 redaction with fake secrets only.
8. Confirm events are written to observability/ledger only and never trigger Go
   reroute/account mutation.
9. Execute kill-switch proof: next request exact/pass-through.
10. Record rollback readiness and stop condition status.

### PROD Pass / Fail Criteria

Pass requires all of:

- S1-S5 are green in the approved environment;
- C1-C6 pass with no unapproved exception;
- no event or evidence contains secrets;
- no Go runtime router path fires for an L2-owned session;
- no fallback occurs after commit;
- Smart Context exact fallback and kill switch are proven;
- rollback remains immediately available.

Fail any one of:

- `go_rotation_decision_count > 0` for L2-owned session;
- continuation moves to another profile without Rust-owned stale/dead binding
  policy;
- `fallback.committed=true` or fallback after first model output;
- Smart Context corrupts JSON, tool-call ids, continuation ids, or mandatory
  references;
- `redaction.secrets_present=true` reaches ledger/observability;
- kill switch does not affect the next request;
- rollback cannot be executed.

### Stop And Rollback Conditions

Stop the validation window and roll back when:

- any fail criterion occurs;
- event ingest is unavailable or accepting malformed/secret-bearing events;
- p95 rewrite overhead exceeds owner-approved threshold;
- user-visible task regression is linked to Smart Context rewrite;
- prodex sidecar readiness flaps or loses state backend;
- evidence cannot be written scrubbed.

## 7. Evidence Format

Store scrubbed evidence under:

```text
.deploy-control/evidence/
```

Required:

- command summary;
- result;
- event ids;
- no raw secrets;
- owner;
- timestamp.

Additional required fields for F0-GATED PROD validation:

- `gate`: `S1`-`S5` or `C1`-`C6`;
- `mode`: `dry-run`, `shadow`, `canary`, `live`, or `exact`;
- `f0_approved`: boolean;
- `runtime_router_owner`: expected `rust_l2` for L2-owned sessions;
- `event_ids`: schema-valid runtime event ids, if emitted;
- `rollback_ready`: boolean;
- `kill_switch_verified`: boolean;
