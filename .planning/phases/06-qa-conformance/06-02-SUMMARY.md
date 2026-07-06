---
phase: 06-qa-conformance
plan: 02
status: completed_with_blockers
completed_at: 2026-07-05T02:52:42Z
requirements: [REQ-16, REQ-17, REQ-18, REQ-26]
artifacts:
  - .deploy-control/evidence/c5-smart-context.md
  - .deploy-control/evidence/c6-isolation.md
  - .deploy-control/evidence/herdr-smoke.md
  - .deploy-control/evidence/mcp-conformance.md
---

# SUMMARY 06-02 — QA Conformance C5/C6/Herdr/MCP

## Resultado

Executado conforme `.planning/phases/06-qa-conformance/06-02-PLAN.md`, sem
marcar dry-run/live-blocked como DONE.

## Evidências Criadas

- `.deploy-control/evidence/c5-smart-context.md`
- `.deploy-control/evidence/c6-isolation.md`
- `.deploy-control/evidence/herdr-smoke.md`
- `.deploy-control/evidence/mcp-conformance.md`

## Verificações

- C5 Smart Context:
  - replay strict: GREEN;
  - `cargo test -p prodex-runtime-proxy smart_context`: 113 passed;
  - live shadow->canary->live: BLOCKED.
- C6 isolation:
  - static source review: GREEN;
  - synthetic file/env isolation probe: GREEN;
  - live triple `CODEX_HOME x prodex x Herdr`: BLOCKED.
- Herdr smoke:
  - discovery: 9 agents, 10 panes;
  - `agent send`: OK;
  - notification: shown;
  - agent-status event/wait: OK.
- MCP/provider conformance:
  - Rust `provider_conformance`: 13 passed;
  - Rust `provider_conformance_v1`: 14 passed;
  - `npm run test:gemini-schema`: FAILED;
  - live MCP provider passthrough: BLOCKED.

## Bloqueadores

- `prodex profile list`: no profiles configured.
- `prodex info`: providers none, provider routes none, runtime policy disabled,
  no active prodex runtime, no quota-compatible profiles.
- Live C5/C6/MCP proof requires configured prodex profiles/providers and
  controlled runtime traffic.
- Gemini schema parity has missing snippets/schema expectations and must be
  fixed before claiming full MCP/Gemini conformance.

## Gate P6

P6 is NOT closed by this plan. Evidence was produced and offline/local checks
were run, but live-blocked and failed checks remain IN_PROGRESS/BLOCKED rather
than DONE.
