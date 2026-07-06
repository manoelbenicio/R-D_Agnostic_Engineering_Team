# SPEC — Phase 11: Vendor Validation

phase: 11-vendor-validation
milestone: v2.1
author: Kiro/Principal

## WHAT this phase delivers (not HOW)
Proof that Smart Context works for each of the 4 real vendors, expressed as a capability matrix with
zero `not_validated` cells and a per-vendor measurement of tokens_saved > 0.

## Acceptance criteria
- REQ: capability matrix (docs/vendors/vendor-capability-matrix.md) has 0 `not_validated` cells.
- REQ: for each vendor {OpenAI/Codex, Antigravity, OpenCode/GLM5.2, Kiro/Opus4.8}, a documented
  payload shape and a measured tokens_saved.
- REQ: measurement provenance recorded per EVIDENCE_CONTRACT.md.

## Known caveat (must be stated, not hidden)
Local measurements return `local_estimate` because the local gateway is 404 (no real upstream). Real
per-vendor numbers are obtained in P12's live session. Until then P11 = PASS_WITH_CAVEAT, and any
"tokens_saved" carrying `measurement_source=local_estimate` must be labeled as such.

## Explicit gap
OpenCode/GLM5.2 was not measured (a Cline row was measured instead — Cline is not a target vendor).
This gap closes in P12.12.3.

## Out of scope (this phase)
Real provider round-trips (that is P12). No fake-upstream substitute is acceptable as a stand-in.
