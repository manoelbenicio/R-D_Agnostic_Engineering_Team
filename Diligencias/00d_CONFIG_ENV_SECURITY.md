# Superfície de Config/ENV + Segurança do prodex (varredura)

> Fecha o furo: env vars, subcomandos e providers do prodex — com os itens de **segurança** destacados.
> Base: `/tmp/prodex-audit-7750da9` (commit 7750da9b).

## 1. Subcomandos do prodex (o que o Multica pode invocar)
`run`, `s` (launch), `redeem` (reset-claim), `mcp` (MCP server), `auth`, `login` (credencial),
`doctor`, `super-doctor` (diagnóstico/health), `profile`, `quota`, `status`, `compact-output`, `replay-report`.
- **Multica usa hoje:** `run`/`s` (F3), `redeem` (F9), `mcp` (P1/P6). 
- **Mapear/decidir:** `auth`/`login` (fluxo de credencial vs isolamento), `doctor`/`status`/`quota` (health/observability), `profile` (isolamento), `replay-report`/`compact-output` (QA replay/Smart Context).

## 2. 🔴 SEGURANÇA — plugin/hook "Caveman"
Env vars revelam execução de comando via hook:
`PRODEX_CAVEMAN_HOOK_COMMAND`, `PRODEX_CAVEMAN_HOOK_SCRIPT`, `PRODEX_CAVEMAN_HOOK_MARKER`,
`PRODEX_CAVEMAN_HOOK_TIMEOUT_SEC`, `PRODEX_CAVEMAN_MARKETPLACE_NAME`, `PRODEX_CAVEMAN_PLUGIN_*`, `PRODEX_CAVEMAN_SOURCE_REPO`.
- **Risco:** hook = execução arbitrária de comando/script + marketplace/source repo externo → RCE/supply-chain.
- **Ação obrigatória (pré-deploy):** por padrão **DESABILITAR** Caveman/hook; se necessário, allowlist explícita, timeout, sem marketplace externo não-auditado. **Gate de segurança (P4/P6).**

## 3. 🔴 ENV sensíveis (travar/auditar)
- `PRODEX_ALLOW_UNSAFE_CHILD_ENV` → **deve ficar OFF** (vazamento de env pro child).
- `PRODEX_CLAUDE_PROXY_API_KEY` / chaves → **nunca em log**; via secret-store.
- `PRODEX_AGY_BIN`, `PRODEX_CLAUDE_BIN` → binários de vendor pinados/validados.
- `PRODEX_ANTHROPIC_*` (base_url/model), `PRODEX_AUDIT_LOG_DIR`, `PRODEX_HOME`, `PRODEX_SMART_CONTEXT_*`, kill-switch.
- **Ação:** inventário completo dos `PRODEX_*` + defaults seguros documentados; scrubbing garantido.

## 4. Providers do prodex × Vendors do Multica (eixos diferentes)
- **Providers backend do prodex:** anthropic, claude, gemini, **deepseek**, **copilot**, openai.
- **Vendors (agent CLIs) do Multica (12+):** Claude Code, Codex, Copilot CLI, OpenClaw, OpenCode, Hermes, Gemini, Pi, Cursor, Kimi, Kiro, Qoder.
- **Escopo rotation-parity:** 5 (Codex, Kiro, Antigravity, Cline, OpenCode) — **decisão**; os demais 7 = out-of-scope-agora (documentar explicitamente).
- **Mapear:** qual vendor Multica → qual runtime/provider prodex (ex.: Codex→openai/anthropic-compat; Antigravity→gemini; etc.). Ver vendor matrix (P5).

## REQs adicionados
- **REQ-33** — Superfície ENV/config do prodex: inventário completo dos `PRODEX_*` + defaults seguros; `ALLOW_UNSAFE_CHILD_ENV=off`; chaves via secret-store, nunca em log.
- **REQ-34** — Caveman/hook: **DESABILITADO por padrão** (RCE/supply-chain); se usado, allowlist+timeout+sem marketplace externo. Gate de segurança pré-deploy.
- **REQ-35** — Mapa subcomandos prodex usados pelo Multica (run/s/redeem/mcp/auth/doctor/quota/status/...).
- **REQ-36** — Mapa provider(prodex)×vendor(Multica) + declarar os 7 vendors out-of-scope.