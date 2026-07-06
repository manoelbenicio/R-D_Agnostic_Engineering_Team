# ROADMAP — Milestone v2.1

author: Kiro/Principal (Opus 4.8)
goal: Prove the Rotation-Parity Polyglot delivery for real — real per-vendor Smart Context in PROD,
      with kill-switch + rollback proven live — with zero fabricated evidence.

## Phases

### Phase 11 — Vendor Validation  [PASS_WITH_CAVEAT]
Prove Smart Context per real vendor (OpenAI/Codex, Antigravity, OpenCode/GLM5.2, Kiro/Opus4.8).
- DONE: capability matrix, 0 not_validated cells.
- CAVEAT: savings are `local_estimate` (gateway 404). Real numbers deferred to P12 live session.
- OPEN: OpenCode/GLM5.2 never measured. Closes naturally in P12.12.3.

### Phase 12 — PROD Deploy + Live Test  [BLOCKED — owner creds/host]
Depends on: owner-supplied real provider credentials + real PROD host (PREREQUISITES.md).
Task IDs 12.0–12.7 in phases/12-prod-deploy/PLAN.md. Definition of "real" in EVIDENCE_CONTRACT.md.
Gate: PROD up + real per-vendor session (gateway 200, real usage, distinct numbers) + kill-switch
      LIVE + rollback LIVE + logs scrubbed, all committed + pushed.

## Exit criteria for v2.1 (milestone DONE)
1. P11 real numbers backfilled from P12 live session (no more local_estimate).
2. All 4 real vendors have a real gateway-200 session with distinct tokens_saved.
3. Kill-switch + rollback proven on the real PROD host.
4. Zero INVALID/fabricated evidence remaining; VERIFICATION.md clean.
5. STATE.md reflects reality; committed to origin/main.
