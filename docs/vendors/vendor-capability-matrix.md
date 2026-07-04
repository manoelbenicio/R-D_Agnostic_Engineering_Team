# Vendor Capability Matrix

Status: PRE-DEPLOY REQUIRED

This matrix is capability-based, not marketing-label-based.

## Capability Schema

```text
ProviderCapability {
  launch_mode: native_cli | codex_provider_bridge | openai_compatible_api | anthropic_compatible_api
  auth_mode: oauth_profile | api_key | cloud_iam | cli_native_store
  quota_mode: codex_usage | vendor_balance | rate_limit_headers | custom_probe | none
  rotation_mode: profile_pool | key_pool | gateway_route | unsupported
  continuation_mode: response_id | session_id | cli_thread | none
  smart_context_mode: proxy_rewrite | pre_tool_output_filter | disabled_shadow_only
  reset_claim_mode: codex_redeem | unsupported
}
```

## Matrix

| Product | Status | Launch | Auth | Quota | Rotation | Continuation | Smart Context | Reset |
|---|---|---|---|---|---|---|---|---|
| OpenAI/Codex | verified/not_validated mix | native/prodex | oauth_profile | codex_usage | profile_pool | response_id/session | proxy_rewrite | codex_redeem not_validated |
| prodex | verified in repo docs, runtime not fully validated | wrapper/gateway | profile/API depending provider | mixed | precommit profile/key/gateway | response/session binding | proxy_rewrite | codex_redeem |
| Kiro | verified docs, adapter not_validated | native_cli/import | oauth/profile/native | custom_probe/none | profile_pool not_validated | cli_thread | proxy/pre-filter depending path | unsupported |
| Antigravity | verified repo, adapter not_validated | native_cli | Google sign-in/native | custom_probe/none | unsupported/profile not_validated | cli_thread | pre-filter/disabled | unsupported |
| Cline | verified docs, deploy path not_validated | editor/provider config | api_key/gateway | provider-dependent | gateway_route/key_pool | provider-dependent | gateway/proxy | unsupported |
| OpenCode | not_validated | to verify | to verify | to verify | to verify | to verify | to verify | unsupported |
| DeepSeek | verified API docs | openai_compatible_api | api_key | vendor_balance | key_pool/gateway | API response | proxy_rewrite | unsupported |
| Bedrock | verified AWS docs | openai_compatible_api | cloud_iam/API gateway | AWS-managed | gateway_route | API response | proxy_rewrite | unsupported |

## Deploy Rule

Only capabilities marked `verified` or explicitly accepted as `not_validated`
by owner may be enabled. Unknown capabilities default to disabled.
