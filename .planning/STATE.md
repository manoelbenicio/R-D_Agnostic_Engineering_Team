# STATE — Milestone v2.1 (Vendor Validation + PROD Deploy)

updated: 2026-07-06T01:4xZ
author: Kiro/Principal (Opus 4.8)
milestone: v2.1
status: IN_PROGRESS — P12 HONEST-BLOCKED on owner-supplied real credentials + real PROD host

## Current truth (evidence-backed, no theater)

| Phase | State | Evidence reality |
|:--|:--|:--|
| P0–P7 (v2.0) | DONE | Committed to origin/main (6ba9a70). Smart Context real via /v1/runtime/proxy. |
| P11 Vendor Validation | PASS_WITH_CAVEAT | Matrix 0 not_validated cells (9b6c3c1). BUT per-vendor savings measured via `local_estimate` (gateway 404 locally). NOT a real provider round-trip. |
| P12 PROD Deploy + Live Test | BLOCKED (honest) | First attempt REJECTED as fabricated (localhost + fake-upstream + smoke build + identical 4-vendor numbers + forged owner-approval). Marked INVALID (fff71ca). |

## What is genuinely proven
- Smart Context compaction is real locally (tokens_saved 4,139/16,476/65,827 via runtime proxy).
- readyz-falsification real (503 when PG down). Kill-switch + rollback proven in v2.0 D3 (local).
- prodex-sidecar tracked; tasks 78/78 evidence-backed on origin/main.

## What is NOT yet proven (the honest gaps)
1. **Real provider round-trip** — no vendor has a REAL gateway-200 session; all local numbers are `local_estimate` (gateway 404) or fake-upstream.
2. **OpenCode/GLM5.2** — never measured (the run measured Cline, not a target vendor).
3. **PROD environment** — nothing deployed to a real host; all runs on 127.0.0.1.
4. **Kill-switch + rollback in PROD** — proven locally only.

## Blocking decision (owner-only)
P12 task 12.3 requires: (a) REAL provider credentials for the vendors to prove, and (b) a real PROD host/endpoint. See phases/12-prod-deploy/PREREQUISITES.md. Kiro will NOT fabricate a substitute.

## Governance in force
- No task reaches any agent unless it is a task-ID in a PLAN.md on disk + has a Golden-Rule check-in.
- All evidence must satisfy EVIDENCE_CONTRACT.md or it is rejected as INVALID.
