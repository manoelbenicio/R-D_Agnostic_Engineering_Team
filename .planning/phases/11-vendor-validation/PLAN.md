# PLAN — Phase 11: Vendor Validation (Behavioral)

phase: 11-vendor-validation
milestone: v2.1
status: IN_PROGRESS
depends_on: v2.0 COMPLETE

## Objective

Validate ALL `not_validated` cells in vendor-capability-matrix.md through BEHAVIORAL testing against the real runtime /v1/runtime/proxy. The capability is delivered BY PRODEX (not the vendor).

## Approach

The prodex gateway + sidecar already proved Smart Context works (tokens_saved=4139/16476/65827). The `not_validated` cells exist because VENDOR docs don't mention these features. But OUR architecture delivers them through the proxy — so validate empirically by POSTing each vendor's API shape through /v1/runtime/proxy and measuring results.

## Cells to Validate

| Vendor | Capability | Current Status | Validation Method |
|:---|:---|:---|:---|
| Codex/OpenAI | smart_context_mode | inferred | POST Responses API shape → tokens_saved>0 |
| Codex/OpenAI | reset_claim_mode | not_validated | Exercise prodex redeem endpoint |
| Kiro/Anthropic | smart_context_mode | not_validated | POST Messages API shape → tokens_saved>0 |
| Antigravity/Gemini | rotation_mode | not_validated | profile_pool with 2+ profiles → rotation |
| Antigravity/Gemini | smart_context_mode | not_validated | POST Gemini API shape → tokens_saved>0 |
| Cline/OpenRouter | smart_context_mode | not_validated | POST OpenRouter shape → tokens_saved>0 |
| OpenCode | smart_context_mode | not_validated | ARCHIVED → Crush (document superseded) |

## Task Breakdown

### 11.1 Smart Context Per-Vendor (PARALLEL — 4 agents)

For each vendor, create a session via /v1/session/start, then POST a 16KiB body in that vendor's API shape to /v1/runtime/proxy?session_id=X. Measure tokens_saved.

**Payload shapes:**
1. **Codex/OpenAI (Responses API):** `{model:"gpt-4.1", input:[{role:"user", content:"<16KiB>"}], instructions:"..."}`
2. **Kiro/Anthropic (Messages API):** `{model:"claude-sonnet-4-20250514", messages:[{role:"user", content:"<16KiB>"}], max_tokens:1}`
3. **Antigravity/Gemini:** `{model:"gemini-2.5-pro", contents:[{role:"user", parts:[{text:"<16KiB>"}]}]}`
4. **Cline/OpenRouter:** Same as OpenAI or Anthropic shape depending on configured provider

**Acceptance:** tokens_saved>0 for EACH vendor with DIRECT metric source (not inferred).

**Agent:** Codex#A=OpenAI, Codex#C=Kiro, Codex#D=Antigravity, Codex#E=Cline
**Evidence:** .deploy-control/evidence/V1-smart-context-per-vendor.md

### 11.2 Rotation Per-Vendor

POST /v1/session/start with profile_pool containing 2+ profiles. Verify sidecar accepts and selects a profile. Use kill-switch to force fallback to alternate profile.

**Agent:** Codex#D (Antigravity — primary rotation candidate)
**Evidence:** .deploy-control/evidence/V1-rotation-per-vendor.md

### 11.3 Reset-Claim (Codex)

Investigate if sidecar exposes redeem/reset-claim endpoint. If yes, exercise it. If no, document that reset-claim is a prodex-run feature requiring PROD account state.

**Agent:** Codex#A
**Evidence:** .deploy-control/evidence/V1-reset-claim-codex.md

### 11.4 OpenCode Disposition

Document OpenCode as ARCHIVED/superseded by Crush. Update vendor-capability-matrix.md.

**Agent:** Codex#E
**Evidence:** .deploy-control/evidence/V1-opencode-disposition.md

### 11.5 Matrix Update

After all validations, update vendor-capability-matrix.md changing validated cells from not_validated to verified with evidence pointer.

**Agent:** TL (after agents report)

### 11.6 GATE P11

All not_validated cells either verified or documented as not_applicable. Commit + push.

## Rules

- Only B edits prodex-sidecar/ (if needed for fixes)
- All others: read-only measurement
- Evidence-gated: no DONE without raw evidence
- Elite Rust skills: Diligencias/SKILLS_ELITE_RUST/
