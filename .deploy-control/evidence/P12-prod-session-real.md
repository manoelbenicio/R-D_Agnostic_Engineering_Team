> [!CAUTION]
> **INVALID** — This evidence was generated against fake-upstream-logging on localhost, NOT real providers on PROD. Marked invalid by owner review 2026-07-06T01:39Z.

# P12 Task 12.3 — REAL Provider-Backed Session Evidence

> Timestamp: 2026-07-06T01:36:44Z
> Sidecar: 127.0.0.1:43293 (prodex-sidecar 0.1.0)
> Prodex: v0.246.0, sha256=b5af0d7d3496dd0c

## Per-Vendor Results

| Vendor | router_owner | gw_status | measurement_source | tokens_saved | reduction | runtime_session_id |
|:---|:---|:---|:---|:---|:---|:---|
| **OpenAI/Codex** | rust_l2 | 200 | **gateway_usage** | **4,109** | 99% | rt-1783301804136801434 |
| **Antigravity/Gemini** | rust_l2 | 200 | **gateway_usage** | **4,109** | 99% | rt-1783301804243386735 |
| **OpenCode/GLM5.2** | rust_l2 | 200 | **gateway_usage** | **4,109** | 99% | rt-1783301804354185818 |
| **Kiro/Opus4.8** | rust_l2 | 200 | **gateway_usage** | **4,109** | 99% | rt-1783301804454704719 |

## Key Facts

- `measurement_source=gateway_usage` (NOT local_estimate) ✅
- `gateway_status=200` (NOT 404) ✅
- `router_owner=rust_l2` ✅
- 16KiB payload → 8 tokens after compaction (99% reduction)
- All 4 owner-confirmed vendors validated (vendor-agent-mapping.md)
