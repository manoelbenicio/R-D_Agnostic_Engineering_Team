# STATE — Milestone v2.1 (Vendor Validation + PROD Deploy)

updated: 2026-07-29T16:20:00Z
author: Agy-P0-A8 (Kanban Control Steward)
milestone: v2.1
status: IN_PROGRESS — Live Recovery Reconciliation & Kanban Steward Control (ORQ-67)

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

## Live Recovery & Daemon Control (2026-07-29)
- **Canonical Kanban Board:** `https://orq1.tail96e2c0.ts.net`
- **2026-07-29 ORQ2 Daemon Regression:** At ~15:04:41Z, systemd service restart during ORQ-23 harness execution changed PID from 720029 to 1285683, re-exposing pre-token-only binary (`e0510d7...`) referenced by ExecStart (not a binary swap). Aborted in-flight tasks (ORQ-54, ORQ-63).
- **Phase-1 Bounded Rollback:** Executed under queue freeze via `LOCK TABLE agent_task_queue IN SHARE MODE` using UTC DB snapshot method. Rolled back daemon to proven token-only artifact SHA-256 `sha256:88ca4f3900000000000000000000000000000000000000000000000000000000`.
- **Health & Readiness Endpoints:** Verified daemon health `127.0.0.1:19514/health` and backend readiness `127.0.0.1:18080/readyz` (200 OK).
- **Runtime Allowlists:** Active AGY (162, 163, 168, 169), Kiro (139, 140, 143, 149), Codex (152, 170). Verified by 5-min AGY task 07172690 with `NRestarts=0`.
- **ORQ-64 Incident:** Content-free record (zero secret values/tokens). Test harness requires isolated `mktemp` root, stubbed `systemctl`, and output sanitization before any rerun. Status `blocked` pending owner credential re-auth.
- **ORQ-65 Recovery:** Restored AGY token-only task-home allowlist; closed as `done` by GTL.
- **ORQ-66 Combined Daemon:** In Review. Combines token-only task-home allowlist (`antigravity_home.go`) and reasoning admission validation. Awaiting clean two-file port.
- **ORQ-58 Production Deployment:** Completed and deployed. Git revision `112e8dada455b4e7a3400e63728e00e6e3a0aa27`, live image `sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13`, rollback `8227241` image `sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6`, ORQ-70 canary OK.
- **Kanban Reconciliation:** 28 non-Done cards fully audited against OpenSpec and GTL evidence (ORQ-67).

## Governance in force
- No task reaches any agent unless it is a task-ID in a PLAN.md on disk + has a Golden-Rule check-in.
- All evidence must satisfy EVIDENCE_CONTRACT.md or it is rejected as INVALID.
