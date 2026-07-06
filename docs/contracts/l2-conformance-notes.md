# L2 Conformance Notes - Downstream Check Against `rpp.l2.v1`

Status: VERIFICATION NOTES

Verifier: Codex#5.5#A

Date: 2026-07-04

Scope:

- Contract: `docs/contracts/l2-runtime-contract.md`
- F2 fork/runtime docs: `docs/prodex/prodex-fork-map.md`, `docs/prodex/prodex-runtime-invariants.md`
- F3 Go integration: `multica-auth-work/server/internal/l2runtime/client.go`, `multica-auth-work/server/internal/daemon/prodex.go`, and relevant daemon callsites

No product code was edited in this pass.

Worktree provenance: this verification used the current working tree. At review
time, `multica-auth-work/server/internal/l2runtime/` and
`multica-auth-work/server/internal/daemon/prodex.go` were untracked files, and
`docs/prodex/*` had pre-existing drift outside this verifier's file lock. Those
files were read for conformance evidence but not edited.

## Summary

F2 is broadly conformant to `rpp.l2.v1` as target-milestone documentation. It preserves Rust/prodex as runtime authority, keeps Smart Context and pre-commit routing out of Go, calls out loopback sidecar endpoints, and marks live-account/reset evidence as not validated.

F3 is only partially conformant. `internal/l2runtime/client.go` implements the nominal `rpp.l2.v1` client surface, loopback enforcement, bearer auth, fail-closed health/readiness checks, `router_owner == "rust_l2"` validation, and event `secrets_present == false` rejection. However, the package is not imported anywhere under `multica-auth-work/server`, so the daemon does not yet call `HealthCheck`, `ApplyPolicy`, `RegisterAccounts`, `StartSession`, `RuntimeEventStream`, `KillSwitch`, or `StopSession`.

The main conformance blocker is one-router-per-session: legacy Go rotation remains active in daemon execution paths and is not gated off by an L2 `StartSession` result. That is acceptable only for the current F0/prodex-as-is launch path, not for the `rpp.l2.v1` F3 sidecar contract.

## Endpoint Matrix

| Contract surface | F3 status | Evidence |
|---|---:|---|
| `HealthCheck` / `GET /healthz` | Client present, not wired | `Health` validates `contract_version` and `status == alive` in `multica-auth-work/server/internal/l2runtime/client.go:88`. No imports of `internal/l2runtime` exist under `multica-auth-work/server`. |
| Readiness / `GET /readyz` | Client present, not wired | `Ready` validates `status == ready` and every check status `pass` in `client.go:99`. No daemon lifecycle callsite found. |
| `ApplyPolicy` / `POST /v1/policy/apply` | Client present, not wired | Types and method are present in `client.go:121` and `client.go:155`. No policy push callsite found. |
| `RegisterAccounts` / `POST /v1/accounts/register` | Client present, not wired | Types and method are present in `client.go:167` and `client.go:188`. No approved-account registration callsite found. |
| `StartSession` / `POST /v1/session/start` | Client present, not wired | Method validates `router_owner == "rust_l2"` in `client.go:223`. No daemon session-start callsite found. |
| `StopSession` / `POST /v1/session/stop` | Client present, not wired | Method posts `/v1/session/stop` in `client.go:242`. No daemon stop callsite found. |
| `KillSwitch` / `POST /v1/killswitch/apply` | Client present, not wired | Types and method are present in `client.go:247` and `client.go:269`. Current daemon only injects `PRODEX_KILL_SWITCH_DEFAULT_ON` env in `prodex.go:71`. |
| `RuntimeEventStream` / `GET /v1/events/stream` | Client present, not wired; schema validation incomplete | `StreamEvents` reads NDJSON and rejects version mismatch or `secrets_present == true` in `client.go:303`. It does not validate full `runtime-events.schema.json`, and no ingest callsite was found. |

## Conformant Areas

1. F2 keeps the correct authority split. `prodex-runtime-invariants.md:17` says Go owns cold control-plane state and `prodex-runtime-invariants.md:20` says Rust/prodex owns runtime proxy, affinity, pre-commit routing, fallback, Smart Context, and redeem.
2. F2 explicitly forbids splitting runtime decisions. `prodex-fork-map.md:83` says the fork should keep `prodex-app/src/runtime_proxy` as runtime authority, and `prodex-fork-map.md:176` says not to move in-flight profile selection, quota fallback, Smart Context rewriting, continuation binding, or redeem attempts into Go.
3. F2 matches the target endpoint list. `prodex-fork-map.md:161` calls for loopback JSON sidecar health/readiness, `ApplyPolicy`, `RegisterAccounts`, `StartSession`, `StopSession`, event stream, and kill switches.
4. F2 preserves Smart Context boundary. `prodex-runtime-invariants.md:111` keeps Smart Context inside Rust L2 and `prodex-runtime-invariants.md:112` limits Go to desired mode and kill switch.
5. F2 preserves no-secret requirements. `prodex-runtime-invariants.md:194` says secrets must not appear in logs, traces, evidence, check-ins, or event payloads.
6. F3 client enforces loopback endpoint construction in `validateLoopbackURL` at `client.go:396`.
7. F3 client generates a 32-byte random bearer token with base64url encoding in `client.go:57`.
8. F3 client sets `Authorization: Bearer ...` internally in `client.go:392`; error messages observed in the client do not include the token value.
9. F3 daemon launch path pins prodex before enabling it. `prodex.go:26` requires version and commit, and `config.go:331` replaces the codex agent entry with the resolved prodex entry only when enabled.
10. F3 blocks user `custom_env` override of `MULTICA_*` and `PRODEX_*` values in `daemon.go:4541`, reducing accidental override of sidecar/launch guardrails.

## Findings

### BLOCKER: F3 sidecar client is not integrated into daemon lifecycle

`multica-auth-work/server/internal/l2runtime/client.go` implements the client skeleton, but `rg "internal/l2runtime|l2runtime\\." multica-auth-work/server -g'*.go'` found no import or callsite. As a result, F3 currently cannot prove:

- sidecar readiness before traffic;
- policy apply before session start;
- account registration before session start;
- `router_owner == "rust_l2"` persisted before traffic;
- event stream established for production rollout;
- kill switch applied through the `rpp.l2.v1` endpoint;
- session stop idempotency.

Required next action for F3: wire the client into daemon/session lifecycle, and fail closed before launching runtime traffic if readiness, policy push, account registration, session start, or production event stream setup fails.

### BLOCKER: Legacy Go rotation remains active and is not gated off for L2 sessions

The contract requires exactly one runtime router per session. F2 also says after `StartSession`, Go must not call legacy Go runtime routing for that session (`docs/prodex/prodex-runtime-invariants.md:23`).

Current daemon code still initializes the Go rotation service in `daemon.go:272`, checks proactive rotation before execution in `daemon.go:3192`, retries after reactive exhaustion in `daemon.go:3643`, and can trigger proactive rotation from streamed text in `daemon.go:4222`. The actual account switch is performed through `rotateTaskWithReason` in `daemon.go:3948`.

This is a direct non-conformance for the F3 sidecar path unless all those paths are disabled or bypassed for sessions that have successfully started with `router_owner == "rust_l2"`.

Required next action for F3: persist a per-task/session `runtime_router_owner = rust_l2` after successful `StartSession`, then guard `maybeProactiveRotateFromLedger`, `maybeProactiveRotateOnText`, `rotateTaskOnExhaustion`, and retry-after-rotation logic so Go cannot rotate a task/session owned by L2.

### HIGH: `StartSession` result is validated but not persisted

The client correctly rejects any `StartSession` response whose `router_owner` is not `rust_l2` (`client.go:229`). The contract also requires Go to persist the session as `runtime_router_owner = rust_l2` before sending traffic. No persistence or daemon state integration was found.

Required next action for F3: add storage or in-memory session tracking that records `session_id`, `runtime_session_id`, `runtime_router_owner`, and the L2 event stream URL before execution starts.

### HIGH: Runtime event stream does not validate the full schema

`StreamEvents` rejects contract mismatches and `redaction.secrets_present == true` (`client.go:339` and `client.go:342`). It does not validate required fields or event-specific payload shape from `docs/contracts/runtime-events.schema.json`.

This means malformed `selection`, `affinity`, `fallback`, `rewrite_decision`, `spend_savings`, or `guardrail` events could reach a handler as long as they carry `contract_version: rpp.l2.v1` and `secrets_present: false`.

Required next action for F3: either generate Go validation from `runtime-events.schema.json` or add equivalent typed validation before event ingestion writes observability/ledger records. At minimum, enforce required event-specific fields and reject unknown `event_type`.

### MEDIUM: `StopSession` does not validate response contract

`StopSession` posts to `/v1/session/stop` and returns transport status only (`client.go:242`). If the sidecar returns a body with the wrong `contract_version` or a failed stop acknowledgement, the client will not catch it.

Required next action for F3: define and validate a stop response shape, or document that `204 No Content` is the only success response for `StopSession`.

### MEDIUM: Current kill switch integration is env-only in daemon

The daemon applies `PRODEX_KILL_SWITCH_DEFAULT_ON` in `prodex.go:71`, but no `ApplyKillSwitch` callsite exists. Env-only launch guardrails do not satisfy the target `KillSwitch` endpoint for tenant/provider/profile/session scoping.

Required next action for F3: map control-plane kill switch state to `/v1/killswitch/apply`, including scope and `effective_at`, then fail closed if the sidecar cannot confirm it.

### MEDIUM: F2 notes mention a gap-hardening list that is not present in `docs/prodex`

The current `docs/prodex` directory contains only:

- `docs/prodex/prodex-fork-map.md`
- `docs/prodex/prodex-runtime-invariants.md`

Any downstream checklist expecting `docs/prodex/prodex-gap-hardening-list.md` is stale. This does not affect `rpp.l2.v1` conformance directly, but status boards should not cite the missing path as current F2 evidence.

## No-Secret Review

No real secret values were found in the reviewed F2/F3 surfaces. The F3 l2runtime client stores the bearer token in memory and injects it into request headers, but observed error paths do not print the token. The daemon blocks user overrides for `MULTICA_*` and `PRODEX_*` keys, and F2 explicitly requires secret-free logs/events/evidence.

Residual risk: because `StreamEvents` preserves raw event JSON in memory (`client.go:293` and `client.go:338`) and only trusts `redaction.secrets_present`, Go cannot independently prove that a misbehaving L2 event does not contain a secret. Full schema validation plus downstream redaction tests remain required.

## Conformance Verdict

- F2 docs: PASS for target-milestone contract alignment, with live/reset validations still explicitly not validated.
- F3 client package: PARTIAL PASS as an unintegrated client skeleton.
- F3 daemon integration: BLOCKED for `rpp.l2.v1` conformance until sidecar lifecycle wiring exists and legacy Go rotation is disabled for L2-owned sessions.

Recommended handoff:

- Codex#5.5#C should wire `l2runtime.Client` into daemon lifecycle and add the `runtime_router_owner` gate before traffic.
- Codex#5.5#C should disable Go rotation paths for L2-owned sessions while preserving current F0/prodex-as-is behavior behind a separate mode.
- Codex#5.5#B should ensure Rust emits schema-valid runtime events and implements the target facade endpoints or clearly documents any endpoint still a validar.

## Testable Acceptance Spec: One-Router Gate

Purpose: make the `rpp.l2.v1` single-router invariant directly implementable by Codex#5.5#C.

### Required Gate

After a successful L2 `StartSession` response, Go must persist the runtime router owner before any task traffic is sent to the runtime:

```text
session_id = <task/session id>
runtime_session_id = <StartSession.runtime_session_id>
runtime_router_owner = rust_l2
runtime_router_owner_source = l2_start_session
runtime_router_owner_started_at = <timestamp>
```

Persistence may be durable DB state or daemon-owned in-memory state for the first implementation, but it must be queryable by every Go rotation path before that path can select or switch credentials.

The gate condition is:

```text
if runtime_router_owner(session_id) == rust_l2:
    Go rotation paths must return no-op and must not call rotationService.OnExhaustion,
    rotationService.SelectNext, rotationStore.RecordRotation, or mutate task credential home.
```

This gate applies to at least these daemon paths:

- proactive ledger rotation before execution;
- proactive text/banner rotation while streaming;
- reactive exhaustion rotation after execution;
- retry-after-rotation logic;
- any future Go code path that selects a next account/profile for the same session.

Allowed Go behavior for L2-owned sessions:

- send desired policy/account/kill-switch state before session start;
- ingest L2 runtime events for observability/ledger;
- stop the L2 session;
- fail closed if L2 readiness, policy, account registration, start, event stream, or kill-switch confirmation fails.

Forbidden Go behavior for L2-owned sessions:

- selecting a replacement profile/account;
- changing `credentialAccountHome`;
- clearing `PriorSessionID` or `PriorWorkDir` as part of account rotation;
- retrying the same request on a different account because Go detected quota text;
- interpreting L2 runtime events as commands to reroute an in-flight request.

### Required Observable No-Op Result

Each gated Go rotation path should return a distinguishable no-op reason so tests and logs can prove the gate fired without exposing secrets:

```text
rotation_noop_reason = l2_router_owner
runtime_router_owner = rust_l2
session_id = <opaque id>
```

Do not log profile auth material, bearer tokens, raw prompts, raw tool output, or provider response bodies.

### Acceptance Tests

1. `StartSession` persistence test

   Given an L2 `StartSession` response with `router_owner: "rust_l2"`, the daemon records `runtime_router_owner == rust_l2` for the session before executing the backend. If persistence fails, execution fails closed and no backend task starts.

2. Proactive ledger no-op test

   Given a session with `runtime_router_owner == rust_l2` and an exhausted current assignment, calling the proactive ledger rotation path returns no account, records/reports `rotation_noop_reason == l2_router_owner`, and does not call `rotationService.OnExhaustion` or mutate credential home.

3. Proactive stream-text no-op test

   Given a session with `runtime_router_owner == rust_l2` and streamed text containing a quota warning, the text/banner rotation path returns no account, reports `rotation_noop_reason == l2_router_owner`, and does not call any Go rotation service/store mutation.

4. Reactive exhaustion no-op test

   Given a session with `runtime_router_owner == rust_l2` and a backend result that the Go exhaustion detector would normally classify as exhausted, `rotateTaskOnExhaustion` returns no account and does not trigger retry-after-rotation.

5. F0 compatibility test

   Given no recorded `runtime_router_owner` for the session, the existing F0/prodex-as-is behavior remains unchanged. This proves the gate only disables Go rotation for L2-owned sessions and does not silently remove current launch-mode behavior.

6. Event-ingest non-routing test

   Given valid L2 `selection`, `affinity`, or `fallback` runtime events, Go writes observability/ledger evidence only. It must not call Go rotation selection or mutate the runtime profile/account for the same `runtime_request_id`.

7. Exactly-one-router assertion

   For a test session, collect:

   ```text
   go_rotation_decision_count(session_id)
   go_rotation_noop_count(session_id, reason=l2_router_owner)
   l2_selection_count(session_id, runtime_request_id)
   l2_fallback_after_committed_count(session_id, runtime_request_id)
   persisted_runtime_router_owner(session_id)
   ```

   The test passes only if:

   ```text
   persisted_runtime_router_owner(session_id) == rust_l2
   go_rotation_decision_count(session_id) == 0
   go_rotation_noop_count(session_id, reason=l2_router_owner) >= 1
   l2_selection_count(session_id, runtime_request_id) <= 1
   l2_fallback_after_committed_count(session_id, runtime_request_id) == 0
   ```

This acceptance spec is the implementation bar for closing the one-router gate blocker.
