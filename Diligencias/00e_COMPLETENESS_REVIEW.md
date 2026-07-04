# Revisão de Completude — varredura de "o que pode ter ficado esquecido"

> Varredura sistemática de TODAS as dimensões contra a fonte (prodex source + Multica repo).
> Status por dimensão + gaps NOVOS achados. Complementa 00c (crates) e 00d (env/CLI).

## Dimensões varridas

| Dimensão | Achado | Status |
|----------|--------|--------|
| prodex config | lê `config.toml` (por-task em CODEX_HOME) | ✅ coberto (execenv) |
| **prodex BROWSER (Playwright/Chromium)** | 58 `browser` + 13 `playwright` + `PRODEX_BROWSER_STDIO_REPORT` | 🔴 **GAP → REQ-37** |
| **prodex MEMORY (Mem0)** | 136 `mem0` + `memory_backend` (backend ativo) | 🔴 amplia REQ-27 → **REQ-37b** |
| prodex npm | pacote `prodex-workspace` (distribuição alt) | ✅ nota em 00b |
| Multica rotas × contrato | handler tem rotas HTTP; prodex/l2 **não** no handler (está no daemon) — correto | 🟡 mapear rotas formalmente |
| **Migrations Postgres** | **322** .sql (up/down reversíveis); ex. `124_approved_accounts` | ✅ REQ-03 satisfeito (confirmar 124) |
| **CI Go** | só `go test -race ./...` — **sem vet/lint/security** | 🟠 **GAP → REQ-38** |
| **Deploy** | `deploy/helm` + `deploy/observability` + `docker-compose.selfhost` (K8s/self-host) | 🟠 **GAP → REQ-39** (P7 runbook) |
| Datastores | Postgres pg17 + Redis (docker) | ✅ |
| Isolamento/execenv | per-task CODEX_HOME + per-account HOME + copia auth.json | ✅ |
| 44 crates | matriz 00c | ✅ |
| env/subcomandos/providers | 00d | ✅ |
| Caveman/hook (RCE) | REQ-34 (OFF por padrão) | ✅ |
| MCP | REQ-26 | ✅ |

## Gaps NOVOS (incorporados)
- **REQ-37** — **Browser automation (Playwright/Chromium)** do prodex: é capability + **superfície de segurança** (execução de browser, `PRODEX_BROWSER_STDIO_REPORT`). Decidir escopo (habilitar/disabled), sandbox, e implicação de segurança (allowlist de domínios, headless, sem exfiltração). Cobrir em fork-map (P2), conformance (P6) e segurança (P4).
- **REQ-37b** — **Memory Mem0** (`prodex-memory`, `memory_backend`): backend de memória ativo — escopo (on/off), privacidade/redaction (PII em memória), contrato. (amplia REQ-27).
- **REQ-38** — **Hardening do CI**: além de `go test -race`, adicionar `go vet`, lint (golangci-lint) e security scan (`govulncheck`/gitleaks) como gates. (O CI-parity foi bypassado por decisão do dono, mas o CI em si é fino.)
- **REQ-39** — **Deploy runbook (P7)** deve referenciar o mecanismo real: **Helm** (`deploy/helm`) + `docker-compose.selfhost` + `deploy/observability`; migrations reversíveis (322 .sql) aplicadas no deploy.

## Confirmado COBERTO (não é gap)
Migrations reversíveis (up/down) já são convenção do repo; config.toml por-task; npm como distribuição alt;
integração prodex vive no daemon (não no handler HTTP) — arquitetura correta.

## Dimensões que AINDA valem varredura futura (declaradas, não esquecidas)
- Flags detalhadas de cada subcomando prodex (run/s/redeem/mcp/doctor…).
- Mapa formal rota-a-rota do Multica × contrato L2 (P1/P3).
- CI do prodex (o que ele testa) para espelhar no nosso P6.