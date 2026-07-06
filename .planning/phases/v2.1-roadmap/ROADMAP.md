# ROADMAP — v2.1 Full Vendor Validation + PROD Deploy

## Phase V1 — Behavioral Validation of not_validated Capabilities

**Goal:** Prove every `not_validated` cell in the vendor capability matrix through BEHAVIORAL testing via our runtime (/v1/runtime/proxy). The capability is delivered BY PRODEX, not by the vendor — so validate through our proxy.

**Tasks:**

- [ ] V1.1 Smart Context per-vendor validation: POST to /v1/runtime/proxy with each vendor's typical payload shape (Codex/OpenAI Responses API, Kiro/Anthropic Messages API, Antigravity/Gemini API, Cline/OpenRouter). Prove tokens_saved>0 for EACH. Evidence in .deploy-control/evidence/V1-smart-context-per-vendor.md
- [ ] V1.2 Rotation validation: Start session with profile_pool containing 2+ profiles per vendor. Trigger rotation event. Prove new profile is selected without session interruption. Evidence in .deploy-control/evidence/V1-rotation-per-vendor.md
- [ ] V1.3 Reset-claim validation (Codex): Exercise prodex redeem/--auto-redeem through sidecar. Prove claim-reset event emitted. Evidence in .deploy-control/evidence/V1-reset-claim-codex.md
- [ ] V1.4 OpenCode disposition: Document as ARCHIVED/superseded by Crush. Update vendor-capability-matrix.md. Evidence in .deploy-control/evidence/V1-opencode-disposition.md
- [ ] V1.5 Update vendor-capability-matrix.md: Change all validated cells from not_validated to verified with evidence pointer.
- [ ] V1.6 GATE V1: All not_validated cells either verified or documented as not_applicable. Commit + push.

## Phase V2 — PROD Deploy + Live Test

**Goal:** Deploy to PROD and validate with real provider-backed session.

**Tasks:**

- [ ] V2.1 Pre-deploy checklist: Verify all GATE P0-P7 green, kill-switch tested, rollback tested, readyz-falsification passed.
- [ ] V2.2 Execute prod-rollout-runbook: Follow docs/deploy/prod-rollout-runbook.md step by step.
- [ ] V2.3 PROD session: Start real provider-backed session through deployed sidecar. Capture evidence.
- [ ] V2.4 PROD kill-switch test: Apply kill-switch in PROD, verify behavior. Rollback.
- [ ] V2.5 PROD logs scrubbed: Verify no secrets in PROD logs.
- [ ] V2.6 GATE V2: PROD session evidence + kill-switch + rollback + scrubbed logs. Commit + push.

**Success criteria:** All not_validated → verified. PROD session real. Kill-switch + rollback LIVE. Commit SHA pushed.
