# REQUIREMENTS — Milestone v2.0 (Rotation-Parity Polyglot: Fundação + Deploy Correto)

> Requisitos escopados e rastreáveis (REQ-IDs). Cada REQ mapeia para fase(s) no ROADMAP.
> Aterrados no estado real verificado (ver PROJECT.md §2).

## Fundação (o que faltou no plano anterior)
- **REQ-01** — Provisionar o binário prodex: a partir do source em `/tmp/prodex-audit-7750da9` (commit `7750da9b`), instalar toolchain Rust, `cargo build --release`, mover o source/binário para local estável (não `/tmp`), verificar **pin (versão+commit) e integridade** (hash/attestation).
- **REQ-02** — Ambiente dev/deploy pronto: confirmar Postgres/Redis (docker) alcançáveis; toolchain de build (docker golang) validado; `.env`/vars (`MULTICA_PRODEX_ENABLED/PATH/VERSION/COMMIT`, `PRODEX_HOME`) definidos.
- **REQ-03** — Migrations Postgres reversíveis para gateway/ledger/approved-accounts do runtime.

## Contrato & Integração
- **REQ-04** — Contrato Go↔L2 (`rpp.l2.v1`) + schema de eventos, com invariante de **roteador único por sessão** validado em teste.
- **REQ-05** — Integração Go: lançar/lifecycle do prodex (sidecar), policy push, event ingest, **kill-switch**; Go **não** roteia request em voo.
- **REQ-06** — Ingest de runtime-events validado (não dispara rotação no Go).

## Capabilities & Vendors
- **REQ-07** — Matriz de capability por provider (fonte primária): Codex/Kiro/Antigravity/Cline/OpenCode. `verified|inferred|not_validated`.
- **REQ-08** — Reavaliar **OpenCode** (arquivado → sucessor Crush): manter disabled, descopar, ou migrar. Decisão documentada.
- **REQ-09** — Smart Context/token-saver via prodex (mandatório), com fallback exato; **não** reimplementar em Go.

## State & Segurança
- **REQ-10** — Postgres para estado compartilhado (SQLite proibido); secrets boundary.
- **REQ-11** — Redaction: sem segredo em logs/traces/errors/audit — **teste com evidência**.
- **REQ-12** — Taxonomia de audit: account selection, redeem attempt, fallback, continuation binding, context-rewrite decision.

## QA exaustivo (SEM bypass — gate duro)
- **REQ-13** — Conformance C1–C6 por capability (não por rótulo), com evidência em container.
- **REQ-14** — Replay: long-session, tool-calls, previous_response_id, compact, SSE, WebSocket.
- **REQ-15** — Troca de perfil **fail-closed** provada.
- **REQ-16** — Smart Context validado shadow→canary→live (medição antes/depois + fallback automático).
- **REQ-17** — Tripla-interação `CODEX_HOME × prodex × Herdr` coexistindo sem clobber (isolamento provado).
- **REQ-18** — Coordenação Herdr operacional (agent send/notification/events) provada em smoke.

## Deploy & Rollback
- **REQ-19** — **Kill-switch testado** (real, não só documentado) por tenant/provider/profile.
- **REQ-20** — **Rollback em 1 comando** testado (volta a `codex` cru).
- **REQ-21** — Deploy **direto em PROD** (sem canary/staging) atrás de REQ-19/REQ-20 verdes + QA exaustivo verde + logs scrubbed.

## MCP (superfície de tool-calls via prodex)
- **REQ-26** — Suporte a MCP do prodex mapeado e coberto: `prodex-mcp-stdio` (framing stdio) + tradução/passthrough de tool-calls MCP no runtime (anthropic/gemini/deepseek). Contrato/eventos devem cobrir tool-calls MCP; afinidade preserva estado de tool_call/continuation; conformance testa passthrough. Segurança: MCP servers stdio são superfície — declarar quais são confiáveis.

## Superfícies prodex adicionais (varredura dos 44 crates — ver Diligencias/00c)
- **REQ-27** — Memory MCP backend (`prodex-memory`, Mem0-compat): escopo (on/off), contrato, privacidade.
- **REQ-28** — Redaction real via `prodex-presidio`+`prodex-redaction`: amarrar G8/P4 ao motor nativo; testar PII.
- **REQ-29** — Runtime broker (`prodex-runtime-broker`): health/registry/metrics → contrato L2.
- **REQ-30** — Cookie relay (`prodex-runtime-cookies`): auditar superfície de auth/sessão.
- **REQ-31** — Quota adapters (`prodex-quota`/`runtime-quota`): ampliar quota_mode por vendor.
- **REQ-32** — Plugin Caveman (`prodex-caveman-assets`): investigar/escopar/segurança.

## Config/ENV + Segurança do prodex (varredura — ver Diligencias/00d)
- **REQ-33** — Inventário completo dos `PRODEX_*` + defaults seguros; `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`; chaves via secret-store, nunca em log.
- **REQ-34** — Caveman/hook (`PRODEX_CAVEMAN_HOOK_*`): **DESABILITADO por padrão** (RCE/supply-chain); se usado: allowlist+timeout+sem marketplace externo. Gate de segurança pré-deploy.
- **REQ-35** — Mapa dos subcomandos prodex usados pelo Multica (run/s/redeem/mcp/auth/doctor/quota/status/profile/replay-report).
- **REQ-36** — Mapa provider(prodex: anthropic/gemini/deepseek/copilot/openai) × vendor(Multica) + declarar os 7 vendors out-of-scope.

## Superfícies extras (varredura de completude — ver Diligencias/00e)
- **REQ-37** — Browser automation (Playwright/Chromium) do prodex: escopo (on/off), **sandbox/segurança** (allowlist domínios, headless, sem exfiltração); cobrir fork-map/conformance/segurança.
- **REQ-37b** — Memory Mem0 (`prodex-memory`/`memory_backend`): escopo, privacidade/redaction de PII em memória, contrato.
- **REQ-38** — Hardening do CI: além de `go test -race`, adicionar `go vet` + lint (golangci) + security scan (govulncheck/gitleaks).
- **REQ-39** — Deploy runbook (P7) referencia mecanismo real: Helm (`deploy/helm`) + `docker-compose.selfhost` + `deploy/observability`; migrations reversíveis (322 .sql) no deploy.

## Reset-claim (baixa prioridade — por último)
- **REQ-22** — Matriz reset-claim (planning) + validação **empírica** com contas reais (guardas: idempotência, cooldown, audit); só quando o estado ocorrer.

## Meta / Processo
- **REQ-23** — Tasks rastreáveis (o change anterior tinha 0 tasks no CLI); dependências entre fases formalizadas.
- **REQ-24** — Arquivar `rotation-router` (SUPERSEDED) e reconciliar docs/board.
- **REQ-25** — Reconciliar contradição "deploy direto × QA exaustivo": QA exaustivo em container ANTES; deploy direto DEPOIS.