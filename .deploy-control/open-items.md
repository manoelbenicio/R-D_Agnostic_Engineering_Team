# Open Items — Owner Decisions

## Owner decisions (ONLY these two)
1. **F5 vendor sign-off** — `docs/vendors/owner-acceptance-request.md` (8 not_validated + 2 borderline cells): ACCEPT / REJECT.
2. **F0 deploy (canary) go** — authorizes the live/canary validation of the live-gated gates (G1/G5/G7/F6/G10-live), which by design run DURING the F0 canary, not before.

## Non-issues (do NOT escalate)
- **Account/profile isolation** is ALREADY RESOLVED (since HerdMaster; preserved by prodex AS-IS — hard affinity + per-profile isolation + rotate-before-commit, per ADR-001 + `prodex-runtime-invariants`). The transient "please sign in again" on codex panes C/D was an **operational session hiccup** (C recovered to Full Access), **not a design gap**. Closed — no owner action. (Prior "Option-A isolation" entry retracted.)

_Reconciled by opus-4.8-orchestrator per Tech-Lead directive, 2026-07-04._
