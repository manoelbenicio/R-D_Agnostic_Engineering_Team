# W2-C2 Zero Go Rotation Evidence

status: GREEN_WITH_DEPENDENCY
timestamp_utc: 2026-07-05T22:42:20Z
agent: Codex#5.5#C
stream: W2-C2-ZERO-ROTATION

## Scope

- Verified one-router gate in `multica-auth-work/server/internal/daemon/`.
- Added explicit sentinel `ErrL2Owned` for the Go rotation block when L2 owns the session.
- Added unit coverage proving:
  - L2 is enabled.
  - `StartSession` returns `router_owner=rust_l2`.
  - Go records the router owner before execution.
  - The Go rotation path returns/observes `ErrL2Owned` via `legacyGoRotationBlockError`.
  - `rotateTaskWithReason` does not call the legacy Go rotation service.
- Did not edit `multica-auth-work/prodex-sidecar/`.

## Code Evidence

- `multica-auth-work/server/internal/daemon/daemon.go:3896-3903`
  - Defines `ErrL2Owned`.
  - `legacyGoRotationBlockError` returns it when `runtimeRouterOwnerForTask(task) == rust_l2`.
- `multica-auth-work/server/internal/daemon/daemon.go:3905-3921`
  - `legacyGoRotationAllowed` blocks legacy Go rotation and logs `rotation_noop_reason=l2_router_owner`.
- `multica-auth-work/server/internal/daemon/daemon.go:3923-4022`
  - Proactive ledger, proactive text, reactive exhaustion, and explicit rotation all consult the gate before touching Go rotation service.
- `multica-auth-work/server/internal/daemon/daemon_test.go:1684-1725`
  - `TestL2StartSessionRouterOwnerReturnsErrL2OwnedForGoRotationPath` exercises the requested path.

## Unit Test Evidence

```text
env GOCACHE=/tmp/codex-go-build /home/dataops-lab/.cache/codex-go/go/bin/go test -race -count=1 ./internal/daemon
ok  	github.com/multica-ai/multica/server/internal/daemon	20.501s
```

```text
env GOCACHE=/tmp/codex-go-build /home/dataops-lab/.cache/codex-go/go/bin/go test -race -count=1 ./internal/l2runtime ./internal/daemon
ok  	github.com/multica-ai/multica/server/internal/l2runtime	1.027s
ok  	github.com/multica-ai/multica/server/internal/daemon	20.642s
```

## Existing Sidecar Contract Probe

B1 runtime-real handoff is not complete in this workspace:

```text
.deploy-control/Codex-5.5-B__PASSO0-RUNTIME-INVESTIGATION__20260705T222850Z.md
status: IN_PROGRESS
```

Because B1 is pending, W2-C2 used the existing compiled local sidecar as the broker contract dependency and did not modify Rust sidecar files.

Start sidecar:

```text
env MULTICA_L2_BEARER_TOKEN=w2-c2-test multica-auth-work/prodex-sidecar/target/release/prodex-sidecar 127.0.0.1:43137
prodex-sidecar listening on 127.0.0.1:43137
```

Health:

```json
{"contract_version":"rpp.l2.v1","sidecar":{"commit":"smoke","name":"prodex-sidecar","version":"0.1.0"},"status":"alive"}
```

Ready:

```json
{"checks":[{"details":{"backend_type":"postgres","connection_status":"ok"},"name":"shared_state_backend","status":"pass"},{"name":"kill_switch","status":"pass"},{"name":"runtime_proxy","status":"pass"}],"contract_version":"rpp.l2.v1","status":"ready"}
```

Session start:

```json
{"contract_version":"rpp.l2.v1","event_stream_url":"http://127.0.0.1:43117/v1/events/stream?session_id=task-l2-owned","request_id":"w2-c2-start","router_owner":"rust_l2","runtime_endpoint":"loopback","runtime_log_ref":"memory","runtime_session_id":"rt-1783291391707829751","smart_context_mode":"shadow"}
```

## Conclusion

The Go daemon blocks all legacy Go rotation paths for an L2-owned session. The requested explicit error signal exists as `ErrL2Owned`, and the unit test proves the path from L2 `StartSession(router_owner=rust_l2)` to zero Go rotation service calls.
