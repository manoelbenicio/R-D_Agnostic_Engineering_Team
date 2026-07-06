---
phase: 06-qa-conformance
plan: 02-live-rerun
status: failed
completed_at: 2026-07-05T03:00:00Z
requirements: [REQ-16, REQ-17, REQ-18, REQ-26]
artifacts:
  - .deploy-control/evidence/c5-smart-context-live.md
  - .deploy-control/evidence/c6-isolation-live.md
  - .deploy-control/evidence/mcp-conformance-live.md
---

# SUMMARY 06-02 LIVE RERUN

## Resultado

Golden Rule cumprida antes da execução:

- `.deploy-control/Codex__PLAN-06-02-LIVE__20260705T025940Z.md`

Os testes live foram reexecutados contra o endpoint contratual
`http://127.0.0.1:43117`, mas falharam por conexão recusada.

## Comandos Executados

- `policy-apply-smoke.sh --execute --base-url http://127.0.0.1:43117`
- `kill-switch-smoke.sh --execute --base-url http://127.0.0.1:43117 --feature smart_context`
- `session-start-stop-smoke.sh --execute --base-url http://127.0.0.1:43117`
- `event-stream-smoke.sh --execute --base-url http://127.0.0.1:43117 --min-events 1`

All commands used:

```text
SMOKE_ALLOW_EXECUTE=1
SMOKE_TARGET_ENV=test
L2_BEARER_TOKEN=<dummy>
```

## Discovery

- `ss -ltnp`: no listener on `127.0.0.1:43117`.
- `docker ps`: no visible L2 sidecar container.
- `curl /healthz` and `/readyz` on `127.0.0.1:43117`: connection refused.
- Local active ports did not expose the required `rpp.l2.v1` `/readyz` and
  `/v1/events/stream` surfaces.
- `prodex info`: `Profiles: 0`, `Providers: none`, `Runtime policy: disabled`,
  `No active prodex runtime detected`.

## Verdict

- C5 live: RED.
- C6 live: RED.
- MCP live: RED.

P6 remains not closed. The statement that the sidecar is up could not be
verified from this host/namespace.
