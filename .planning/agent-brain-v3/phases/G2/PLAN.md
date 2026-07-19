# PLAN — G2: Quatro streams paralelas (Codex 1-4)

gate saída: entregas isoladas contra contratos congelados; agentes 2-4 NÃO fiam no daemon.

## G2A — Brain core (Codex1) — tasks.md §3
- 3.1 neutral coordinator/task-executor/registry sem mover cred lógica
- 3.2 neutral fields CLIKind/RouteModel/RouterOwner/correlation/policy
- 3.3 compat translations + measurable legacy-use events
- 3.4 gateway-required admission/readiness + fail-closed (sem habilitar path novo)
- 3.5 preserve workspace/repo/worktree/cancel/watchdog/context/skills/recovery/stream atrás de interfaces neutras
Evidence: EV-G2A-01..05

## G2B — OmniRoute gateway (Codex2) — tasks.md §4
- 4.1 client (auth redacted, base URL config, timeouts, cancel, correlation)
- 4.2 liveness/readiness + /v1/models authenticated + error classification
- 4.3 registry versionada (protocol/tools/reasoning/stream/context/structured)
- 4.4 trusted runtime profiles (Messages/Responses/Chat/Antigravity)
- 4.5 route-policy types (strict RR, affinity, retry deadline, same/cross-model fallback, circuit, SC flags)
- 4.6 parse telemetry (no content/secrets)
- 4.7 protocol fixtures (synthetic creds/content)
Evidence: EV-G2B-01..07

## G2C — Credentialless runtime/CLI (Codex3) — tasks.md §5
- 5.1 minimal inherited-env builder (remove provider keys/cookies/direct URLs)
- 5.2 custom-env validation deny + trusted applied last
- 5.3 remove provider-auth copying from homes
- 5.4 controlled Codex custom-provider (Responses, stable-key, HTTP/SSE, correlation, sem auth.json)
- 5.5 Claude Code trusted root URL/token; markers não vazam override
- 5.6 Kimi/GLM/NVIDIA adapter + NIM→gateway (no direct NVIDIA key)
- 5.7 Kimi registry path ou fallback Claude/Codex
- 5.8 Agy native SE só se proved; senão fallback agy/ via Claude/Codex
- 5.9 model/thinking validation gateway-aware
- 5.10 pre-launch assertion (só OmniRoute secret + dados locais)
Evidence: EV-G2C-01..10

## G2D — Ops/paridade/evidence (Codex4) — tasks.md §6
- 6.1 Linux restricted secret (sem copiar valor p/ repo/image/logs/screenshots)
- 6.2 endpoint config (host loopback; futuras containeriza; sem hardcode DNS p/ host daemon)
- 6.3 structured redacted events/metrics p/ cada evento hot-path
- 6.4 dashboards/alerts
- 6.5 capacity/failure harness spec (20/50/100)
- 6.6 runbooks (backup/restore, hot-change, rotation key, upgrade/rollback, incident, escalo)
- 6.7 staged flags/controlled cohorts/rollback triggers/evidence locations
Evidence: EV-G2D-01..07

Pré-requisitos: G1. STATUS: COMPLETE 2026-07-18 (authorized no-secret scope).

Acceptance: G2A EV-G2A-01..05; G2B EV-G2B-01..07; G2C EV-G2C-01..10 with
5.6–5.8 retained as fail-closed no-secret contracts; G2D EV-G2D-01..07. Live panes
`w3:pD/p8/p9/pA` were reconciled idle with final worker reports. No active daemon wiring,
credential mutation, cutover, provider traffic or tier activation occurred.
