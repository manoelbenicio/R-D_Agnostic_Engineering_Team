# Single-Router-Per-Session Invariant

Keyword: single-router

Status: P1 gate artifact

Contract version: `rpp.l2.v1`

## Formal Statement

For any session `S`, exactly one entity routes in-flight requests. Go L4 holds desired state and pushes policy. Rust L2 performs runtime routing. Event ingest by Go MUST NOT trigger Go-side rotation.

In operational terms:

```text
session_started(S) -> router_owner(S) = rust_l2
output_stream_started(S) -> Go rotate(S) is forbidden until session_stopped(S)
RuntimeEventStream(S) -> Go ledger/observability only, no routing side effect
```

## Authority Split

Go is allowed to:

- publish desired state through `ApplyPolicy`, `RegisterAccounts`, `StartSession`, `StopSession`, and `KillSwitch`;
- record events in observability and ledger stores;
- fail closed before a session begins when policy, account state, sidecar health, or kill-switch state is invalid.

Go is not allowed to:

- rotate, rebind, retry-after-rotation, or select an account for an already-active L2-owned session;
- treat `RuntimeEventStream` or `RouteDecisionEvent` as a command to run Go-side rotation;
- rotate mid-flight after output begins.

Rust L2 is allowed to:

- select the profile/provider before commit;
- preserve hard affinity through `previous_response_id`, turn state, `session_id`, and MCP `tool_call_id`;
- perform bounded pre-commit fallback;
- emit runtime events for Go to validate and ingest.

## Testable Properties

1. End-to-end session routing: when a session runs from `session_started` to `session_stopped`, only the L2 runtime routes the in-flight request.
2. No mid-flight Go rotation: after output stream begins, Go never calls rotate/rebind/retry-after-rotation until the session is stopped.
3. Inert runtime events: runtime events ingested by Go are validation, observability, and ledger inputs only; they cannot call Go rotation.
4. Rotate-before-commit only: rotation may occur only at a session boundary or before a runtime request is committed, never mid-stream.
5. Affinity preservation: MCP tool-call and continuation state remains bound to the same runtime router for the active session.

## Test Approach

### Unit Test

Mock the L2 runtime and Go rotation service:

1. Start an L2-owned session and persist `router_owner = rust_l2`.
2. Begin an output stream.
3. Inject quota warning text, route decision events, MCP tool-call events, fallback events, and quota snapshots.
4. Assert Go rotation service call count remains zero until `session_stopped`.

Acceptance assertion:

```text
for event in runtime_events_between(session_started, session_stopped):
    ingest(event)
    assert go_rotate_call_count(session_id) == 0
```

### Integration Test

Run Go against a controlled mock L2 sidecar:

1. `StartSession` returns `runtime_router_owner = rust_l2`.
2. Mock sidecar streams valid `rpp.l2.v1` events, including `mcp_tool_call`.
3. Go validates and stores events.
4. Go does not call legacy Go rotation paths and does not mutate account/profile binding.

### Property Test

For every generated sequence:

```text
session_started
[runtime_event | tool_call | warning_text | quota_snapshot]*
session_stopped
```

The property is:

```text
Go rotate() is never called between session_started and session_stopped.
```

Any counterexample is a P1/P3 contract failure.

## Fixture Link

Positive and negative event examples are in `docs/contract/fixtures/`.

## GATE P1 Assertion

- [x] Single-router invariant is formally stated.
- [x] At least three testable properties are defined.
- [x] Unit and integration test approaches are documented.
- [x] Go pushes desired-state only; Rust routes in-flight requests.
- [x] Go NEVER rotates mid-flight after output begins.
