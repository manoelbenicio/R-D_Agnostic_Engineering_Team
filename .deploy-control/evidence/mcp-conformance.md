# MCP / Provider Conformance Evidence

Status: GREEN - local L2/MCP smoke and provider translation conformance passed
Timestamp: 2026-07-05T06:27:39Z
Executor: Codex-A
Task: 6.8 MCP conformance
Secrets present: false

## Scope

This evidence covers the 06-06 task scope:

- MCP/L2 passthrough over a real local `rpp.l2.v1` sidecar process;
- provider translation conformance fixtures in the pinned prodex source;
- Multica Go L2/client/daemon/MCP config tests.

No live provider traffic or real credentials were used. The sidecar bearer token
was a dummy smoke token and is not recorded.

## Pinned Prodex Facade

Command:

```text
bin/prodex --version
```

Result:

```text
prodex 0.246.0
```

## Local L2/MCP Smoke

Setup:

```text
sidecar: multica-auth-work/prodex-sidecar/target/release/prodex-sidecar
bind: ephemeral 127.0.0.1 loopback port
auth: dummy smoke bearer token
```

Commands:

```text
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=<dummy> \
  bash scripts/smoke/readyz-smoke.sh --execute --base-url <loopback>

SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=<dummy> \
  bash scripts/smoke/policy-apply-smoke.sh --execute --base-url <loopback> --tenant-id tenant-mcp-smoke

SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=<dummy> SMOKE_SESSION_ID=session-mcp-smoke \
  bash scripts/smoke/session-start-stop-smoke.sh --execute --base-url <loopback>

SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=<dummy> SMOKE_SESSION_ID=session-mcp-smoke \
  bash scripts/smoke/event-stream-smoke.sh --execute --base-url <loopback> --min-events 1
```

Results:

```text
readyz-smoke: PASS
policy-apply-smoke: PASS
session-start-stop-smoke: PASS
event-stream-smoke: PASS
validated_events=2
```

Assertions covered:

- sidecar readiness includes `contract_version=rpp.l2.v1`;
- policy apply accepts the safe desired-state envelope;
- session start returns `router_owner=rust_l2`;
- session stop is accepted;
- event stream emits NDJSON with `secrets_present=false`.

## Go MCP / L2 Tests

Commands:

```text
/home/dataops-lab/.cache/codex-go/go/bin/go test ./internal/l2runtime ./internal/daemon ./cmd/multica
/home/dataops-lab/.cache/codex-go/go/bin/go test ./internal/daemon/execenv
```

Results:

```text
ok github.com/multica-ai/multica/server/internal/l2runtime
ok github.com/multica-ai/multica/server/internal/daemon
ok github.com/multica-ai/multica/server/cmd/multica
ok github.com/multica-ai/multica/server/internal/daemon/execenv
```

Regression fixed during this pass:

- `cursorProjectRoot` now ignores invalid/empty `.git` sentinels such as
  `/tmp/.git`, so managed Cursor MCP config writes stay inside the task workdir
  and do not collide with host-level `/tmp/.cursor/mcp.json`.

## Rust Provider Translation Conformance

Commands:

```text
CARGO_TARGET_DIR=<tmp> cargo test -p prodex-provider-core --test provider_conformance -- --nocapture
CARGO_TARGET_DIR=<tmp> cargo test -p prodex-provider-core --test provider_conformance_v1 -- --nocapture
```

Results:

```text
provider_conformance: 13 passed
provider_conformance_v1: 14 passed
```

Relevant passed assertions include:

- translated Gemini and DeepSeek fixture coverage;
- request/response/stream shapes for current provider fixtures;
- `gemini_request_preserves_continuation_metadata_explicitly`;
- `deepseek_request_fixture_preserves_continuation_metadata_explicitly`;
- passthrough endpoint fixture coverage where docs claim support.

## Verdict

MCP/L2 passthrough, event stream validation, Multica MCP config behavior, and
provider translation conformance are green for the local sidecar and pinned
prodex fixture scope required by task 6.8.
