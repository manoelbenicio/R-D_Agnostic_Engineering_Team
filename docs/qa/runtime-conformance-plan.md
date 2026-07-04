# Runtime Conformance Plan

Status: PRE-DEPLOY REQUIRED

Owner: GLM#52#A / Opus 4.8 approval

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
`route_selected`. Go must not run runtime load-balance/fallback for request in
flight.

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

