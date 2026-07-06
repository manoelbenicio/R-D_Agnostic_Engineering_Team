# MCP Conformance Live Evidence

Status: RED - live sidecar endpoint unreachable
Timestamp: 2026-07-05T03:00:00Z
Executor: Codex

## Objective

Rerun live MCP/provider passthrough conformance after 03-01 reportedly completed
and the L2 sidecar was said to be up.

## Live Event Stream Attempt

Command:

```text
SMOKE_ALLOW_EXECUTE=1 SMOKE_TARGET_ENV=test L2_BEARER_TOKEN=<dummy> \
  bash scripts/smoke/event-stream-smoke.sh --execute --base-url http://127.0.0.1:43117 --min-events 1
```

Result:

```text
code=1
curl: (7) Failed to connect to 127.0.0.1 port 43117 after 0 ms: Couldn't connect to server
expected at least 1 event(s), got 0
```

## Discovery Notes

The contractual endpoint `127.0.0.1:43117` was not listening. Probing local
listeners found unrelated services (registry, Hadoop, Grafana/Prometheus,
blackbox, AGY panes), but no endpoint implementing both `/readyz` and
`/v1/events/stream` with the `rpp.l2.v1` contract.

## Verdict

Live MCP/provider passthrough is NOT green. The earlier Rust fixture evidence
still proves offline provider conformance coverage, but no live MCP event or
tool-call passthrough was observed in this rerun.
