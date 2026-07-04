# Matriz de Cobertura — 44 crates do prodex × plano

> Garantia VERIFICÁVEL de que nenhuma superfície do prodex ficou de fora. Cada crate → área do plano →
> coberto? Gaps viram REQ/tasks. Base: `/tmp/prodex-audit-7750da9/crates` (commit 7750da9b).

| Crate | Função | Área do plano | Coberto? |
|-------|--------|---------------|----------|
| prodex-core | paths/primitives | fundação | ✅ P0 |
| prodex-shared-types | tipos internos | contrato/fork-map | ✅ P1/P2 |
| prodex-shared-codex-fs | ops no ~/.codex | isolamento por conta | ✅ execenv/P2 |
| prodex-profile-identity | identidade de perfil | isolamento | ✅ P2 |
| prodex-profile-export | crypto export de perfil | isolamento/segurança | ✅ P4 |
| prodex-secret-store | secret storage | segurança/secrets boundary | ✅ P4 |
| prodex-session-store | metadata de sessão codex | afinidade/continuation | ✅ P1/P2 |
| prodex-codex-config | parse config codex | fundação/config | ✅ P0 |
| prodex-context | Smart Context (audit/compress) | Smart Context | ✅ P2/G5 |
| prodex-mcp-stdio | framing MCP stdio | **MCP** | ✅ P1/P2/P6 (REQ-26) |
| prodex-runtime-anthropic | tradução Anthropic (+MCP tools) | contrato/conformance | ✅ P1/P6 |
| prodex-runtime-gemini(-cli-compat) | Gemini runtime/compat | conformance/vendor | ✅ P5/P6 |
| prodex-runtime-claude | launch Claude Code | vendor/launch | ✅ P5 |
| prodex-provider-core | catálogo/adapter/cost | vendor matrix | ✅ P5 |
| prodex-runtime-capabilities | detecção de compat | conformance | ✅ P6 |
| prodex-audit-log | audit log | audit taxonomy | ✅ P4 |
| prodex-runtime-policy | policy parse/validate | ApplyPolicy (contrato) | ✅ P1/P3 |
| prodex-runtime-launch | launch planning | integração/lifecycle | ✅ P3 |
| prodex-runtime-proxy / proxy-config | proxy boundary/config | fork-map runtime | ✅ P2 (enumerar) |
| prodex-runtime-state / prodex-state / runtime-store | state/merge | state backend | ✅ P4 (Postgres) |
| prodex-runtime-metrics / runtime-log | métricas/log | observability | ✅ P8 |
| prodex-terminal-ui | render terminal | (fora — UI CLI) | N/A |
| prodex-update-notice / housekeeping / bench-support / app-reports / cli / app | infra CLI/app | fundação/build | ✅ P0 |
| **prodex-memory** | **memory MCP backend (Mem0-compat)** | **NOVO** | 🔴 REQ-27 |
| **prodex-presidio** | **PII/redaction (Presidio)** | redaction | 🔴 REQ-28 (amarrar a G8/P4) |
| **prodex-redaction** | **redaction de logs/diag** | redaction | 🔴 REQ-28 |
| **prodex-runtime-broker(+log)** | **broker registry/health/metrics** | contrato L2 (HealthCheck/Route) | 🔴 REQ-29 |
| **prodex-runtime-cookies** | **cookie relay (auth/sessão)** | auth/segurança | 🔴 REQ-30 |
| **prodex-quota / runtime-quota** | **adapters de quota** | quota_mode/rotação | 🟠 REQ-31 (ampliar P5) |
| **prodex-caveman-assets** | **plugin assets (Caveman)** | plugin surface | 🟠 REQ-32 (investigar/escopar) |
| prodex-runtime-tuning | tuning overrides | runtime/fork-map | ✅ P2 |

## REQs adicionados (gaps → plano)
- **REQ-27** — Memory MCP backend (`prodex-memory`, Mem0-compat): decidir escopo (habilitar/disabled), contrato, e implicação de privacidade/redaction.
- **REQ-28** — Redaction real via `prodex-presidio` + `prodex-redaction`: amarrar o gate G8/P4 ao motor nativo do prodex (não só regra genérica); testar PII scrubbing.
- **REQ-29** — Runtime broker (`prodex-runtime-broker`): mapear health/registry/metrics ao contrato L2 (HealthCheck, RouteDecisionEvent, RuntimeEventStream).
- **REQ-30** — Cookie relay (`prodex-runtime-cookies`): auditar superfície de auth/sessão (cookies) — segurança/redaction.
- **REQ-31** — Quota adapters (`prodex-quota`/`runtime-quota`): ampliar cobertura de quota_mode por vendor (rotação proativa).
- **REQ-32** — Plugin Caveman (`prodex-caveman-assets`): investigar o que é, decidir escopo/segurança.

## Regra
P2 (fork-map) DEVE enumerar TODOS os 44 crates com esta matriz; nenhum crate "genérico/assumido".