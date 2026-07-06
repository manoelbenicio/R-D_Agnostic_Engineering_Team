# prodex AS-IS readiness — REAL inspection evidence
- by: opus-4.8-orchestrator ; at: 2026-07-04T22:09:47Z ; prodex 0.246.0 (pinned)
## prodex info
```
[ Info ] =====================================================================================================
Profiles:               0
Active profile:         -
Providers:              none
Provider routes:        none
Runtime policy:         disabled
Runtime preset:         default
Runtime proxy contract: scoped gateway, policy-visible selection, bounded precommit retry, cheap hot path,
                        quota/transport split, structured observability, connection reuse, profile-isolated
                        secrets
Secret backend:         file
Runtime logs:           /tmp (text)
Audit logs:             /tmp/prodex-audit.log (missing)
Runtime metrics:        -
Runtime workers:        workers proxy=12, long-lived=24, async=4, probe-refresh=4; active=84, queue=192; lanes
                        responses=63, compact=6, websocket=24, standard=24; ws-connect workers=16, queue=128,
                        overflow=512; ws-dns workers=8, queue=32, overflow=64
Runtime budgets:        precommit=12x/3000ms, pressure-precommit=6x/800ms, continuation=24x/12000ms;
                        admission=750ms, pressure-admission=200ms, long-lived=750ms, pressure-long-lived=200ms
Runtime transport:      http-connect=5000ms, stream-idle=300000ms, sse-lookahead=1000ms; ws-connect=15000ms,
                        ws-progress=8000ms, ws-happy=200ms, ws-stale-reuse=60000ms; inflight soft/hard=4/8
Prodex version:         0.246.0 (up to date)
Prodex processes:       Yes (2 total, 0 runtime; pids: 573629, 573632)
Recent load:            No active prodex runtime detected
Codex quota data:       No quota-compatible profiles
```
## prodex capability
```
List Prodex capabilities and local availability.

Usage: prodex capability <COMMAND>

Commands:
  list          List Prodex capabilities and local availability
  super-doctor  Diagnose the local optimizer stack used by `prodex s` / `prodex super` [aliases: s-doctor]
  help          Print this message or the help of the given subcommand(s)

Options:
  -h, --help  Print help

Examples:
  prodex capability list
  prodex capability list --json
```
## prodex doctor
```
[ Doctor ] ===================================================================================================
Prodex root:            /home/dataops-lab/.prodex
State file:             /home/dataops-lab/.prodex/state.json (missing)
Profiles root:          /home/dataops-lab/.prodex/profiles
Default CODEX_HOME:     /home/dataops-lab/.codex (exists)
Codex binary:           codex (/tmp/prodex-codex-xP1RZ8/codex)
Kiro binary:            kiro-cli-chat (/home/dataops-lab/.local/bin/kiro-cli-chat)
Quota endpoint:         https://chatgpt.com/backend-api/wham/usage
Runtime policy:         disabled
Runtime proxy contract: scoped gateway, policy-visible selection, bounded precommit retry, cheap hot path,
                        quota/transport split, structured observability, connection reuse, profile-isolated
                        secrets
Secret backend:         file
Runtime logs:           /tmp (text)
Audit logs:             /tmp/prodex-audit.log (missing)
Runtime metrics:        -
Import auth journals:   None
Profiles:               0
Active profile:         -
```
## prodex capability list
```
[ Capabilities ] =============================================================================================
codex:               available; runtime; Codex CLI frontend
claude:              available; runtime; Claude Code frontend
caveman:             built-in; mode-assets; embedded Caveman Codex/Claude plugin assets
rtk:                 available; optimizer; upstream shell-output token reduction
sqz:                 missing; optimizer; downstream context reuse MCP
token-savior:        missing; optimizer; symbol navigation MCP
codebase-memory-mcp: missing; optimizer; structural codebase graph MCP
claw-compactor:      missing; optimizer; local deterministic code-summary aid
ponytail:            built-in; optimizer-plugin; managed checkout loaded as a Codex plugin in Prodex overlays
prodex-inspect:      built-in; diagnostics; read-only MCP diagnostics for Prodex status, profiles, and runtime
                     logs
prodex-memory:       built-in; memory; local-first SQLite memory MCP without Mem0 Cloud auth
smart-context:       built-in; runtime; runtime proxy context compaction and rehydration
runtime-doctor:      built-in; diagnostics; runtime log and pressure diagnostics
```
## prodex profile list
```
[ Profiles ] =================================================================================================
Status:         No profiles configured.
Create:         prodex profile add <name>
Import:         prodex profile import-current
Import Copilot: prodex profile import copilot
```
## READINESS VERDICT (opus-4.8-orchestrator)
- prodex install: DONE (v0.246.0 pinned, runnable).
- prodex config: NOT READY — 0 profiles, no providers, policy disabled, no quota-compatible profiles.
- Real QA (C1-C6/redaction-live/kill-switch-live/fail-closed) + deploy: BLOCKED on profile enrollment, which needs a healthy codex account (single account d3982266 has revoked/contended tokens).
- Next real gate: enroll >=1 working codex profile via prodex login/profile add (needs interactive account auth).
