# V1 — OpenCode Disposition (CORRECTED)

> Date: 2026-07-06T01:29Z
> Corrected by: Owner directive (vendor-agent-mapping.md)

## Previous Disposition (WRONG)

OpenCode was marked ARCHIVED/superseded by Crush per ADR-001 decision 4.
All not_validated cells were marked not_applicable.

## Corrected Disposition

**OpenCode is IN ACTIVE USE** in our fleet (GLM 5.2 agents). The upstream repo
(github.com/opencode-ai/opencode) was archived by the original maintainer, but
our team actively uses OpenCode to run GLM 5.2 models.

**Owner-confirmed:** `docs/vendors/vendor-agent-mapping.md`

## Smart Context Validation (REAL)

| Metric | Value |
|:---|:---|
| gateway_status | **200** |
| tokens_saved | **4,109** |
| measurement_source | **gateway_usage** (NOT local_estimate) |
| input_tokens_before_estimate | 4,117 |
| input_tokens_after_observed_or_estimate | 8 |

Evidence: `.deploy-control/evidence/V1-remeasurement-gateway-200.md`

## Verdict

OpenCode/GLM5.2 **VERIFIED** — Smart Context compaction works via prodex gateway
with real round-trip measurement (gateway_usage, gateway_status=200).
