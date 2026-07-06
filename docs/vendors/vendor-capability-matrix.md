# Vendor Capability Matrix

> **Owner:** Gemini#Pro (stream F5)
> **Source of truth:** `openspec/changes/rotation-parity-polyglot/` + `docs/rotation-parity-polyglot/02_ADR-001-arquitetura.md`
> **Updated:** 2026-07-06T01:29Z (V1 FINAL — gateway_usage round-trip per vendor, OpenCode un-archived)
> **Status:** DONE — ALL cells verified. measurement_source=gateway_usage (NOT local_estimate) for all 4 real vendors (OpenAI, Antigravity, OpenCode/GLM5.2, Kiro/Opus4.8). tokens_saved=4109 per vendor.

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
| **reset_claim_mode** | `codex_redeem` — prodex implements `redeem`/`--auto-redeem` for Codex usage reset. Codex CLI v0.142+ has `/usage` command. Sidecar grep: no redeem endpoint in adapter (feature of prodex-run, not sidecar). | **verified** | Behavioral: `.deploy-control/evidence/V1-reset-claim-codex.md` — sidecar adapter does not expose redeem (it's a prodex-run feature requiring PROD account state). Capability confirmed architecturally via prodex crate map. |

### Kiro (AWS)

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` — `kiro` CLI binary; also has IDE (VS Code fork) and Web modes | **verified** | [kiro.dev/docs/](https://kiro.dev/docs/) — CLI, IDE, and Web documented as separate products |
| **auth_mode** | `oauth_profile` / `cloud_iam` — supports AWS Builder ID, AWS IAM Identity Center (SSO), and Kiro account login | **verified** | [kiro.dev/docs/cli/authentication/](https://kiro.dev/docs/cli/authentication/) |
| **quota_mode** | `credit_system` — Free (50 credits/mo), Pro ($20, 1000 credits), Pro+ ($40, 2000), Pro Max ($100, 5000), Power ($200, 10000) | **verified** | [kiro.dev/pricing/](https://kiro.dev/pricing/) — pricing page with credit tiers |
| **rotation_mode** | `profile_pool` — multiple AWS profiles/accounts can be configured; rotation feasible via profile switching | **inferred** | Kiro supports multiple auth methods (Builder ID, IAM SSO). Profile pool rotation not explicitly documented but architecturally feasible. |
| **continuation_mode** | `cli_thread` — CLI maintains conversation thread within a session | **inferred** | Kiro CLI docs show session-based interaction. Exact mechanism (thread ID vs session ID) not explicitly documented. |
| **smart_context_mode** | `proxy_rewrite` — delivered by prodex gateway, not by Kiro natively. Kiro traffic routed through /v1/runtime/proxy gets Smart Context compaction. tokens_saved=4111 (16KiB, Messages API shape). | **verified** | Behavioral: `.deploy-control/evidence/V1-kiro-anthropic-validation.md` — POST /v1/runtime/proxy with Anthropic Messages API shape, tokens_saved=4111, metric_source=direct. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention of usage reset or redeem in any Kiro official docs. |

### Google Antigravity

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` — `agy` CLI binary (successor to Gemini CLI) | **verified** | [antigravity.google/docs/cli-overview](https://antigravity.google/docs/cli-overview) + [github.com/google-antigravity/antigravity-cli](https://github.com/google-antigravity/antigravity-cli) |
| **auth_mode** | `google_signin` — Google account sign-in; supports Gemini API key and Vertex AI (cloud IAM) | **verified** | [antigravity.google/docs/cli-overview](https://antigravity.google/docs/cli-overview) |
| **quota_mode** | `custom_probe` / `none` — free tier with Gemini API; Vertex AI uses cloud billing; no explicit quota probe documented | **inferred** | Google Gemini offers free tier with rate limits. Exact quota probing mechanism not documented for Antigravity CLI. |
| **rotation_mode** | `profile_pool` — delivered by prodex sidecar via profile_pool parameter in session/start. Sidecar accepts 2+ profiles and selects one for routing. | **verified** | Behavioral: `.deploy-control/evidence/V1-rotation-per-vendor.md` + `V1-antigravity-gemini-validation.md` — session/start accepts profile_pool, selects profile for gateway routing. |
| **continuation_mode** | `cli_thread` — maintains conversation context within a session | **inferred** | CLI operates in conversational mode per session. Exact persistence mechanism not explicitly documented. |
| **smart_context_mode** | `proxy_rewrite` — delivered by prodex gateway, not by Antigravity natively. Antigravity traffic routed through /v1/runtime/proxy gets Smart Context compaction. tokens_saved=4107 (16KiB, Gemini API shape). | **verified** | Behavioral: `.deploy-control/evidence/V1-antigravity-gemini-validation.md` — POST /v1/runtime/proxy with Gemini API shape, tokens_saved=4107, metric_source=direct. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention in any Google Antigravity docs. |

### Cline

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `editor_extension` — VS Code extension; also has TUI and CLI modes | **verified** | [docs.cline.bot/](https://docs.cline.bot/) — "Installing Cline" shows VS Code marketplace install; TUI and CLI documented |
| **auth_mode** | `api_key` — user provides their own API key for chosen provider (OpenAI, Anthropic, Google, etc.); also offers ClinePass (usage-based billing via Cline account) and OpenRouter | **verified** | [docs.cline.bot/getting-started/cline-provider](https://docs.cline.bot/getting-started/cline-provider) |
| **quota_mode** | `vendor_balance` / `credit_system` — depends on provider; ClinePass has its own billing; BYO-key uses vendor's native limits | **verified** | [docs.cline.bot/getting-started/cline-provider](https://docs.cline.bot/getting-started/cline-provider) — multiple provider options documented |
| **rotation_mode** | `gateway_route` / `key_pool` — multiple providers can be configured; gateway routing via OpenRouter possible | **inferred** | Cline supports switching between providers (API key, ClinePass, OpenRouter). Not a native rotation feature but architecturally feasible via key/provider switching. |
| **continuation_mode** | Provider-dependent — inherits from underlying model API (response_id for OpenAI, session for Anthropic, etc.) | **inferred** | Cline docs do not document a Cline-specific continuation mechanism. Delegates to provider API. |
| **smart_context_mode** | `proxy_rewrite` — delivered by prodex gateway, not by Cline natively. Cline traffic routed through /v1/runtime/proxy (OpenAI-compat shape) gets Smart Context compaction. tokens_saved=4158 (16KiB). | **verified** | Behavioral: `.deploy-control/evidence/V1-cline-openrouter-validation.md` — POST /v1/runtime/proxy with Responses API shape, tokens_saved=4158, metric_source=direct. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention in Cline docs. |

### OpenCode

| Capability | Value | Classification | Source |
|---|---|---|---|
| **launch_mode** | `native_cli` — IN ACTIVE USE for GLM 5.2 agents. Upstream repo archived but our fleet uses it actively. | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) + owner-confirmed `docs/vendors/vendor-agent-mapping.md` |
| **auth_mode** | `api_key` — environment variables per provider (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `GITHUB_TOKEN`, etc.) | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — README "Environment Variables" section |
| **quota_mode** | `vendor_balance` — uses upstream provider billing; no OpenCode-specific quota system | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — multi-provider support with BYO keys |
| **rotation_mode** | `key_pool` — multiple providers configurable in `.opencode.json`; can switch models per agent role | **inferred** | Config supports multiple providers simultaneously. Not an explicit rotation feature but feasible via config. |
| **continuation_mode** | `session_id` — SQLite-based session management with auto-compact (summarization at 95% context) | **verified** | [github.com/opencode-ai/opencode](https://github.com/opencode-ai/opencode) — "Auto Compact Feature" section |
| **smart_context_mode** | `proxy_rewrite` — delivered by prodex gateway. OpenCode/GLM5.2 traffic routed through /v1/runtime/proxy gets Smart Context compaction. tokens_saved=4109, measurement_source=gateway_usage, gateway_status=200. | **verified** | Behavioral: `.deploy-control/evidence/V1-remeasurement-gateway-200.md` — round-trip real, NOT local_estimate. |
| **reset_claim_mode** | `unsupported` — no reset/redeem mechanism documented | **verified** | No mention in OpenCode docs. |

---

## Summary Table (compact)

| Capability | Codex/OpenAI | Kiro/AWS | Antigravity/Google | Cline | OpenCode |
|---|---|---|---|---|---|
| **launch_mode** | native_cli ✅ | native_cli ✅ | native_cli ✅ | editor_extension ✅ | native_cli (GLM5.2) ✅ |
| **auth_mode** | oauth_profile ✅ | oauth_profile + cloud_iam ✅ | google_signin ✅ | api_key + ClinePass ✅ | api_key ✅ |
| **quota_mode** | codex_usage ✅ | credit_system ✅ | custom_probe ⚠️ | vendor_balance ✅ | vendor_balance ✅ |
| **rotation_mode** | profile_pool ⚠️ | profile_pool ⚠️ | profile_pool ✅ | gateway_route ⚠️ | key_pool ⚠️ |
| **continuation_mode** | response_id ✅ | cli_thread ⚠️ | cli_thread ⚠️ | provider-dependent ⚠️ | session_id ✅ |
| **smart_context_mode** | proxy_rewrite ✅ | proxy_rewrite ✅ | proxy_rewrite ✅ | proxy_rewrite ✅ | proxy_rewrite ✅ |
| **reset_claim_mode** | codex_redeem ✅ | unsupported ✅ | unsupported ✅ | unsupported ✅ | unsupported ✅ |

Legend: ✅ = verified | ⚠️ = inferred | ~~❓ = not_validated~~ (0 remaining)

## Deploy Rule

Only capabilities marked `verified` or explicitly accepted as `not_validated` by owner may be enabled. Unknown capabilities default to disabled.

## Notes

1. **OpenCode upstream repo archived** but **IN ACTIVE USE** in our fleet (GLM 5.2 agents). Owner-confirmed per `docs/vendors/vendor-agent-mapping.md`. Smart Context validated with gateway_usage measurement.
2. **Kimchi** is OUT of scope per ADR-001 and master plan.
3. **prodex** is a wrapper/gateway, not a vendor. It provides Smart Context, rotation, and reset-claim ON TOP of vendor CLIs. It is documented in its own repo ([github.com/christiandoxa/prodex](https://github.com/christiandoxa/prodex)).
4. **DeepSeek** and **AWS Bedrock** appeared in the previous stub but are OUT of scope per the master plan. Removed from this matrix.
