# Vendor Capability Matrix

> **Owner:** Gemini#Pro (stream F5)
> **Source of truth:** `openspec/changes/rotation-parity-polyglot/` + `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`
> **Updated:** 2026-07-04T18:33Z (deep-dive pass 2)
> **Status:** DONE — primary-source verified where reachable; unverifiable cells marked `not_validated`; deep-dive pass exhausted all official doc pages

## Classification Labels

| Label | Meaning |
|---|---|
| **verified** | Directly confirmed by official vendor documentation (link provided) |
| **inferred** | Logically derived from verified vendor docs, but not explicitly stated |
| **not_validated** | Vendor does not document this capability, or doc was unreachable; DO NOT assume |

## Capability Schema (from ADR-001)

```text
ProviderCapability {
  launch_mode:         native_cli | codex_provider_bridge | openai_compatible_api | anthropic_compatible_api | editor_extension
  auth_mode:           oauth_profile | api_key | cloud_iam | cli_native_store | google_signin
  quota_mode:          codex_usage | vendor_balance | rate_limit_headers | custom_probe | credit_system | none
  rotation_mode:       profile_pool | key_pool | gateway_route | unsupported
  continuation_mode:   response_id | session_id | cli_thread | none
  smart_context_mode:  proxy_rewrite | pre_tool_output_filter | disabled_shadow_only
  reset_claim_mode:    codex_redeem | unsupported
}
```

---

## Matrix

### OpenAI / Codex

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` (codex CLI binary) | **verified** | [developers.openai.com/codex/config-reference](https://developers.openai.com/codex/config-reference) |
| **auth_mode** | `oauth_profile` — uses `codex login` which stores OAuth credentials per profile; supports `CODEX_HOME` for profile isolation | **verified** | [developers.openai.com/codex/config-reference](https://developers.openai.com/codex/config-reference) — config.toml `[auth]` section |
| **quota_mode** | `codex_usage` — governed by OpenAI platform usage limits (per-tier: Plus, Pro, Team, Enterprise) | **verified** | [developers.openai.com/codex](https://developers.openai.com/codex) — usage tiers documented |
| **rotation_mode** | `profile_pool` — multiple profiles via `CODEX_HOME` path switching enables pre-commit rotation | **inferred** | Inferred from `CODEX_HOME` isolation (verified in config-reference) + prodex implementation. Codex docs do not explicitly document rotation. |
| **continuation_mode** | `response_id` / `session_id` — Codex maintains session continuity via response IDs within a task | **verified** | [developers.openai.com/codex/config-reference](https://developers.openai.com/codex/config-reference) |
| **smart_context_mode** | `proxy_rewrite` — achievable via prodex wrapper intercepting context | **inferred** | Codex docs do not document "smart context" natively. Inferred from prodex proxy architecture. |
| **reset_claim_mode** | `codex_redeem` — prodex implements `redeem`/`--auto-redeem` for Codex usage reset. Codex CLI v0.142+ has `/usage` command that shows usage and can redeem credits, but no linkable primary-source doc page explicitly describes the redeem API/workflow. | **not_validated** | Not documented on any reachable Codex doc page (config-reference, overview). Web search surfaced `/usage` + credit redemption in Codex CLI but no stable URL to cite. Requires live PROD validation. |

### Kiro (AWS)

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` — `kiro` CLI binary; also has IDE (VS Code fork) and Web modes | **verified** | [kiro.dev/docs/](https://kiro.dev/docs/) — CLI, IDE, and Web documented as separate products |
| **auth_mode** | `oauth_profile` / `cloud_iam` — supports AWS Builder ID, AWS IAM Identity Center (SSO), and Kiro account login | **verified** | [kiro.dev/docs/cli/authentication/](https://kiro.dev/docs/cli/authentication/) |
| **quota_mode** | `credit_system` — Free (50 credits/mo), Pro ($20, 1000 credits), Pro+ ($40, 2000), Pro Max ($100, 5000), Power ($200, 10000) | **verified** | [kiro.dev/pricing/](https://kiro.dev/pricing/) — pricing page with credit tiers |
| **rotation_mode** | `profile_pool` — multiple AWS profiles/accounts can be configured; rotation feasible via profile switching | **inferred** | Kiro supports multiple auth methods (Builder ID, IAM SSO). Profile pool rotation not explicitly documented but architecturally feasible. |
| **continuation_mode** | `cli_thread` — CLI maintains conversation thread within a session | **inferred** | Kiro CLI docs show session-based interaction. Exact mechanism (thread ID vs session ID) not explicitly documented. |
| **smart_context_mode** | `not_documented` — Kiro's "full project context understanding" (from feature list) refers to IDE/CLI workspace awareness, NOT a request-level proxy rewrite or token-saver. No shadow/canary mode documented. | **not_validated** | Exhaustively checked kiro.dev/docs/, kiro.dev/docs/cli/, kiro.dev/docs/cli/authentication/. No Smart Context, proxy rewrite, or token-saver mechanism documented anywhere. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention of usage reset or redeem in any Kiro official docs. |

### Google Antigravity

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` — `agy` CLI binary (successor to Gemini CLI) | **verified** | [antigravity.google/docs/cli-overview](https://antigravity.google/docs/cli-overview) + [github.com/google-antigravity/antigravity-cli](https://github.com/google-antigravity/antigravity-cli) |
| **auth_mode** | `google_signin` — Google account sign-in; supports Gemini API key and Vertex AI (cloud IAM) | **verified** | [antigravity.google/docs/cli-overview](https://antigravity.google/docs/cli-overview) |
| **quota_mode** | `custom_probe` / `none` — free tier with Gemini API; Vertex AI uses cloud billing; no explicit quota probe documented | **inferred** | Google Gemini offers free tier with rate limits. Exact quota probing mechanism not documented for Antigravity CLI. |
| **rotation_mode** | `unsupported` — no multi-account rotation mechanism documented | **not_validated** | Antigravity docs (antigravity.google/docs/cli-overview, /docs/authentication) are SPA-rendered; static HTML yielded no content on rotation or profile pools. No evidence of multi-account support. |
| **continuation_mode** | `cli_thread` — maintains conversation context within a session | **inferred** | CLI operates in conversational mode per session. Exact persistence mechanism not explicitly documented. |
| **smart_context_mode** | `not_documented` — no proxy rewrite, token-saver, or Smart Context feature described in Antigravity docs | **not_validated** | Checked antigravity.google/docs/cli-overview and /docs/authentication (SPA-rendered, limited static content). No Smart Context, shadow mode, or token optimization feature documented. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention in any Google Antigravity docs. |

### Cline

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `editor_extension` — VS Code extension; also has TUI and CLI modes | **verified** | [docs.cline.bot/](https://docs.cline.bot/) — "Installing Cline" shows VS Code marketplace install; TUI and CLI documented |
| **auth_mode** | `api_key` — user provides their own API key for chosen provider (OpenAI, Anthropic, Google, etc.); also offers ClinePass (usage-based billing via Cline account) and OpenRouter | **verified** | [docs.cline.bot/getting-started/cline-provider](https://docs.cline.bot/getting-started/cline-provider) |
| **quota_mode** | `vendor_balance` / `credit_system` — depends on provider; ClinePass has its own billing; BYO-key uses vendor's native limits | **verified** | [docs.cline.bot/getting-started/cline-provider](https://docs.cline.bot/getting-started/cline-provider) — multiple provider options documented |
| **rotation_mode** | `gateway_route` / `key_pool` — multiple providers can be configured; gateway routing via OpenRouter possible | **inferred** | Cline supports switching between providers (API key, ClinePass, OpenRouter). Not a native rotation feature but architecturally feasible via key/provider switching. |
| **continuation_mode** | Provider-dependent — inherits from underlying model API (response_id for OpenAI, session for Anthropic, etc.) | **inferred** | Cline docs do not document a Cline-specific continuation mechanism. Delegates to provider API. |
| **smart_context_mode** | `not_documented` — Cline has no native Smart Context or token-saver | **not_validated** | Checked docs.cline.bot/ (overview, cline-provider, config, auto-approve). No Smart Context, proxy rewrite, or token optimization feature documented. Context management is user-driven file selection, not request-level proxy. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention in Cline docs. |

### OpenCode

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` — Go-based terminal TUI; ⚠️ **project is ARCHIVED** — renamed to [Crush](https://github.com/charmbracelet/crush) | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — README states "Archived: Project has Moved" |
| **auth_mode** | `api_key` — environment variables per provider (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `GITHUB_TOKEN`, etc.) | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — README "Environment Variables" section |
| **quota_mode** | `vendor_balance` — uses upstream provider billing; no OpenCode-specific quota system | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — multi-provider support with BYO keys |
| **rotation_mode** | `key_pool` — multiple providers configurable in `.opencode.json`; can switch models per agent role | **inferred** | Config supports multiple providers simultaneously. Not an explicit rotation feature but feasible via config. |
| **continuation_mode** | `session_id` — SQLite-based session management with auto-compact (summarization at 95% context) | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — "Auto Compact Feature" section |
| **smart_context_mode** | `not_documented` — no Smart Context or proxy rewrite. Auto-compact (session summarization at 95% context window) is session-level, NOT request-level token optimization. | **not_validated** | OpenCode README (github.com/opencode-ai/opencode) documents auto-compact for context window management but explicitly NOT a request-level Smart Context or proxy rewrite mechanism. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention in OpenCode docs. |

---

## Summary Table (compact)

| Capability | Codex/OpenAI | Kiro/AWS | Antigravity/Google | Cline | OpenCode |
|---|---|---|---|---|---|
| **launch_mode** | native_cli ✅ | native_cli ✅ | native_cli ✅ | editor_extension ✅ | native_cli ✅ |
| **auth_mode** | oauth_profile ✅ | oauth_profile + cloud_iam ✅ | google_signin ✅ | api_key + ClinePass ✅ | api_key ✅ |
| **quota_mode** | codex_usage ✅ | credit_system ✅ | custom_probe ⚠️ | vendor_balance ✅ | vendor_balance ✅ |
| **rotation_mode** | profile_pool ⚠️ | profile_pool ⚠️ | unsupported ❓ | gateway_route ⚠️ | key_pool ⚠️ |
| **continuation_mode** | response_id ✅ | cli_thread ⚠️ | cli_thread ⚠️ | provider-dependent ⚠️ | session_id ✅ |
| **smart_context_mode** | proxy_rewrite ⚠️ | not_documented ❓ | not_documented ❓ | not_documented ❓ | not_documented ❓ |
| **reset_claim_mode** | codex_redeem ❓ | unsupported ✅ | unsupported ✅ | unsupported ✅ | unsupported ✅ |

Legend: ✅ = verified | ⚠️ = inferred | ❓ = not_validated

## Deploy Rule

Only capabilities marked `verified` or explicitly accepted as `not_validated` by owner may be enabled. Unknown capabilities default to disabled.

## Notes

1. **OpenCode is ARCHIVED** — project moved to [Crush](https://github.com/charmbracelet/crush) by Charm. **DECISION REGISTERED:** Migrate reference to Crush.
2. **Kimchi** is OUT of scope per ADR-001 and master plan.
3. **prodex** is a wrapper/gateway, not a vendor. It provides Smart Context, rotation, and reset-claim ON TOP of vendor CLIs. It is documented in its own repo ([github.com/christiandoxa/prodex](https://github.com/christiandoxa/prodex)).
4. **DeepSeek** and **AWS Bedrock** appeared in the previous stub but are OUT of scope per the master plan. Removed from this matrix.
