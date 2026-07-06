# W1-C1 Daemon Real Prodex Evidence

status: GREEN
timestamp_utc: 2026-07-05T22:35:40Z
agent: Codex#5.5#C
stream: W1-C1-DAEMON-REAL

## Scope

- Reviewed `multica-auth-work/server/internal/daemon/l2_runtime.go`.
- Reviewed `multica-auth-work/server/internal/daemon/prodex.go`.
- Implemented daemon-side launch normalization so `MULTICA_L2_SIDECAR_ARGS` can be configured as either:
  - `run --profile <profile> -- <codex args>`
  - `prodex run --profile <profile> -- <codex args>`
  - `<pinned-prodex-path> app-server-broker ...`
- Rejected shim executable forms such as `/path/to/prodex-sidecar ...`; the daemon always executes the configured pinned `MULTICA_PRODEX_PATH` and treats `MULTICA_L2_SIDECAR_ARGS` as prodex subcommand/args after normalization.
- Preserved single-router behavior: once `StartSession` records `runtime_router_owner=rust_l2`, legacy Go rotation paths return no-op reason `l2_router_owner`.

## Code Evidence

- `multica-auth-work/server/internal/daemon/l2_runtime.go:130` parses `MULTICA_L2_SIDECAR_ARGS` against the configured prodex path.
- `multica-auth-work/server/internal/daemon/l2_runtime.go:207` launches `exec.CommandContext(ctx, s.daemon.cfg.Prodex.Path, args...)`.
- `multica-auth-work/server/internal/daemon/l2_runtime.go:257-320` normalizes full `prodex ...` commands to subcommand args and rejects shim executable tokens.
- `multica-auth-work/server/internal/daemon/prodex.go:107-155` forces safe prodex sidecar env (`PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`) and loopback `NO_PROXY` entries.
- `multica-auth-work/server/internal/daemon/prodex_test.go:105-168` covers accepted prodex forms, pinned path stripping, shim rejection, and safe env.
- Existing single-router tests remain in `multica-auth-work/server/internal/daemon/daemon_test.go`:
  - `TestL2OwnedTaskSuppressesLegacyGoRotationPaths`
  - `TestExactOneRouterForL2OwnedSessionHasZeroGoRotations`
  - `TestEventIngestNonRoutingDoesNotTriggerGoRotation`

## Test Evidence

Initial sandboxed run failed because `httptest` could not bind loopback under the managed sandbox:

```text
go test -race ./internal/daemon
FAIL ./internal/daemon [setup failed]
open /home/dataops-lab/.cache/go-build/...: read-only file system
```

Rerun with writable cache reached test execution but the sandbox blocked loopback:

```text
env GOCACHE=/tmp/codex-go-build /home/dataops-lab/.cache/codex-go/go/bin/go test -race ./internal/daemon
panic: httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted
```

Escalated rerun passed:

```text
env GOCACHE=/tmp/codex-go-build /home/dataops-lab/.cache/codex-go/go/bin/go test -race ./internal/daemon
ok  	github.com/multica-ai/multica/server/internal/daemon	20.220s
```

Fresh non-cached unit gate passed:

```text
env GOCACHE=/tmp/codex-go-build /home/dataops-lab/.cache/codex-go/go/bin/go test -race -count=1 ./internal/l2runtime ./internal/daemon
ok  	github.com/multica-ai/multica/server/internal/l2runtime	1.025s
ok  	github.com/multica-ai/multica/server/internal/daemon	19.807s
```

## Notes

- No secrets were printed.
- No live prodex provider session was executed by this C1 unit gate.
- The real-process guarantee here is daemon launch selection: the child process is the pinned prodex binary from `MULTICA_PRODEX_PATH`; shim executable tokens in `MULTICA_L2_SIDECAR_ARGS` fail closed.
