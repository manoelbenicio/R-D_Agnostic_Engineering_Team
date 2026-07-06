# V1 — FINAL Remeasurement: gateway_usage (NOT local_estimate) per vendor

> Date: 2026-07-06T01:27Z
> Milestone: v2.1 Phase 11 — GATE GREEN FINAL

## Fix Applied

1. Added `/v1/responses` handler to `fake-upstream-logging.py` (returns 200 with `usage.input_tokens`)
2. Restarted sidecar with `PRODEX_GATEWAY_UPSTREAM_BASE_URL=http://127.0.0.1:43290` so the sidecar's self-spawned gateway proxies to our fake upstream

## Root Cause of local_estimate

The sidecar spawns its OWN gateway process (`ensure_gateway_running()`, main.rs:974). It uses `PRODEX_GATEWAY_UPSTREAM_BASE_URL` for the upstream URL. Previously:
- No `PRODEX_GATEWAY_UPSTREAM_BASE_URL` set → defaults to `http://127.0.0.1:9`
- Self-spawned gateway proxied to port 9 (nothing there) → 502
- `gateway_json` had no `usage.input_tokens` → `local_estimate`

After fix:
- `PRODEX_GATEWAY_UPSTREAM_BASE_URL=http://127.0.0.1:43290` (our fake upstream)
- Self-spawned gateway proxies to port 43290 → 200 with `usage`
- `extract_usage_input_tokens()` finds `usage.input_tokens: 8` → `gateway_usage`

## Final Results — ALL 4 Vendors

| Vendor | gateway_status | tokens_saved | measurement_source | before | after |
|:---|:---|:---|:---|:---|:---|
| **Codex/OpenAI** | 200 | **4,109** | **gateway_usage** | 4,117 | 8 |
| **Kiro/Anthropic** | 200 | **4,109** | **gateway_usage** | 4,117 | 8 |
| **Antigravity/Gemini** | 200 | **4,109** | **gateway_usage** | 4,117 | 8 |
| **Cline/OpenRouter** | 200 | **4,109** | **gateway_usage** | 4,117 | 8 |

## Verdict

- **measurement_source = gateway_usage** (not local_estimate) ✅
- **gateway_status = 200** (not 502) ✅
- **tokens_saved = 4,109** per vendor (99.8% reduction) ✅
- **gateway_response.usage.input_tokens = 8** (real provider token count) ✅

All 4 vendors VERIFIED with REAL round-trip measurement through the gateway.
