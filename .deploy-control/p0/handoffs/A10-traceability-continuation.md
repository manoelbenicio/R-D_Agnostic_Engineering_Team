# A10 — P0 Traceability CONTINUATION (evidence map → external live-acceptance gate)

- agent: **Opus48#C** · lane **A10** · task **P0-TRACEABILITY-CONTINUATION** · pane `w8:p1`
- sole mutable lock: `.deploy-control/p0/handoffs/A10-traceability-continuation.md`
- check-in: `.deploy-control/p0/checkins/Opus48-C__P0-TRACEABILITY-CONTINUATION__20260722T000500Z.json`
- supersedes: `handoffs/A10-traceability.md` (P0-TRACEABILITY, checked out DONE 00:05Z) — prior gap/design/freeze history there.
- as-of (UTC): `2026-07-22T00:05Z` · HEAD `a6d50986…` · `control.json`: phase `IMPLEMENTATION_W1_SERIAL`, `implementation_authorized=true`, gate **GREEN**, `live_runs.*=false`.
- posture: READ-ONLY. **No product edit / no test / no live run.** A10 never marks the OpenSpec/GSD checkbox.

## 0. Headline

**All Main-Brain-owned P0 code is complete and offline-validated.** D1–D7 (Cline GLM wiring) + D6 (pre-launch `AssertPreLaunch`/`ChildEnvironment.clineDataDir`) are implemented and integrated; R1 and R3 focused validations passed on the integrated build; production integrity is clean. **The entire remaining P0 surface is the external live-acceptance gate** — one authorized live run per changed family — blocked by (a) no Principal `live_run` token, (b) external OmniRoute RouteModels (Kimi, Opus48), (c) enriched-registry availability, (d) the D-V3-25(B) key-revocation security stop. `8.5–8.7` remain OmniRoute-external.

## 1. Completed & validated (DONE, evidence on disk)

| Work | Lane / task | Evidence artifact | Result |
|---|---|---|---|
| **D1–D7** Cline GLM wiring (adapter flip, trusted env inject, `CLINE_DATA_DIR`, `WriteCredentiallessClineConfig`, `buildLaunch` branch, config-bytes, SUB-1 `cp/cline-pass/glm-5.2`) | W1 `P0-W1-IMPLEMENTATION` (DONE 23:41) | `handoffs/W1-implementation.md` | integrated |
| **D6** pre-launch fail-closed contract (assert.go ×3 + `env.go ChildEnvironment.clineDataDir`) | adjudicated: `A10-DESIGN P0-D6-PRELAUNCH-ADJUDICATION` (DONE 23:38); implemented: W1 `P0-W1-IMPLEMENTATION-D6` (DONE 23:58) | `handoffs/D6-prelaunch-adjudication.md` (sha256 `610a3bb5…`) + `handoffs/W1-implementation-D6.md` | implemented per §2/§3 of the adjudication (containment + trusted-entry + switch case + clineDataDir field) |
| **R1 focused validation** (Cline/GLM: adapter/env/execenv/config/model/launch + fail-closed matrix) | A9-R1 `P0-R1-FINAL-INTEGRATION-VALIDATION` (DONE 00:00:18) | `handoffs/R1-final-integration-validation.md` (+ `R1-test-implementation.md`) | **PASS** on integrated build (one focused pass; no duplicate) |
| **R3 focused validation** (lifecycle: L3c admit+launch, L7b cleanup, cancel/usage/terminal, D6 assert cases) | A9-R3 `P0-R3-FINAL-INTEGRATION-VALIDATION` (DONE 00:00:00) | `handoffs/R3-final-integration-validation.md` (+ `R3-test-implementation.md`) | **PASS** on integrated build |
| **Production integrity** FE/BE | A7 `P0-PROD-INTEGRITY-FE` (DONE) · A8 `P0-PROD-INTEGRITY-BE` (DONE) | `handoffs/A7-production-integrity-frontend.md` · `handoffs/A8-production-integrity-backend.md` | A8 = **NO_REACHABLE_RESIDUAL** |
| **Antigravity equivalence** | A5 (DONE) | `evidence/A5-antigravity-equivalence.md` | **STALE** — one minimal scenario reserved (not run) |
| Route/lifecycle gap + design | A1–A6, R1/R2/R3-DESIGN (DONE) | respective handoffs | frozen inputs consumed |

## 2. Requirement closure map (what is done vs the one thing left)

| Req | Code/impl | Focused validation | Remaining to close | External blocker(s) |
|---|---|---|---|---|
| **5.6** Cline→GLM-5.2 | **DONE** (D1–D7 + D6 integrated) | **PASS** (R1) | one authorized GLM live Kanban→terminal run | `live_runs.cline_glm=false` (Principal) · **BLK-AVAIL** (enriched OmniRoute registry availability) · **BLK-25B** (key-revocation security stop) |
| **5.7** Cline→Kimi-K2.7 | shared Cline base **DONE & green** (GLM reuse path); Kimi-specific = ID + `models.go:563` SUB-2 (held) | rides R1 (structural green); Kimi exact-model tests HELD | publish exact Kimi RouteModel → apply SUB-2 → one live run | **BLK-KIMI** (exact versioned Kimi RouteModel UNDECLARED; owner OmniRoute architect) · `live_runs.cline_kimi=false` |
| **5.8** Opus48 | **zero product-source** (config-only, proven) | rides gateway/runtimeenv green | publish exact Opus48 AWS RouteModel → config + one live run | **BLK-OPUS48-ID** (owner OmniRoute registry owner) · `live_runs.opus48=false` |
| **5.8** Antigravity | n/a (evidence-reuse) | A5 = STALE | the one minimal A5 scenario (not run) | **BLK-25B** (security stop); W1 R2 token |
| **8.1** model/protocol/availability | **DONE** (registry/profile/projection green) | PASS via R1/gateway | availability confirmed by the same per-family live run | BLK-AVAIL / external RouteModels |
| **8.2** tools/reasoning/cancel/usage/terminal/error | **DONE** (lifecycle + D6) | **PASS** (R3) | tools/reasoning/usage/terminal observed in the same one live run | same live-run gate |
| **8.5–8.7** | — | — | **EXTERNAL to Main Brain P0** — OmniRoute-owned; no Multica lane/lock/run | OmniRoute owner |

**Dedup preserved:** one authorized live run per family closes its overlapping 5.x/8.1/8.2 in a single pass — no QA-A/B/C, no broad regression, no second acceptance.

## 3. The single remaining P0 gate — external live acceptance

All of the following are **BLOCKED**, each waiting on the same external gate (Principal token + OmniRoute IDs/availability + D-V3-25(B)); none is Main-Brain implementation work:

| Lane / task | State | Reserved evidence consumer (empty) | Waiting on |
|---|---|---|---|
| A9-GLM `P0-GLM-LIVE-ACCEPTANCE` | BLOCKED | `evidence/R1-glm-live-acceptance.md` | `cline_glm` token + BLK-AVAIL + BLK-25B |
| A9-OPUS48 `P0-OPUS48-LIVE-ACCEPTANCE` | BLOCKED | `evidence/R2-opus48-live-acceptance.md` | BLK-OPUS48-ID + `opus48` token |
| R1-KIMI `P0-KIMI-IMPLEMENTATION` | BLOCKED | — (SUB-2 held) | BLK-KIMI (exact ID) |
| A9-R3-EVIDENCE `P0-R3-LIVE-EVIDENCE-CONSUMER` | BLOCKED | `evidence/R3-glm-lifecycle-evidence.md` | the GLM live run |
| A7-EVIDENCE `P0-TERMINAL-UI-EVIDENCE` | BLOCKED | `evidence/A7-terminal-ui-evidence.md` | the GLM live run (UI terminal delivery) |
| A8-EVIDENCE `P0-TERMINAL-BACKEND-EVIDENCE` | BLOCKED | `evidence/A8-terminal-backend-evidence.md` | the GLM live run (backend terminal persistence) |
| W1-EVIDENCE `P0-W1-FINAL-INTEGRATION-EVIDENCE` | BLOCKED | `handoffs/W1-live-readiness.md` | live-run authorization |

## 4. Consolidated external blocker register (owner + action) — all external to Main-Brain code

| ID | Blocks | Owner | Required action |
|---|---|---|---|
| **BLK-TOKEN** | 5.6/5.8 live (and the overlapping 8.1/8.2) | **Principal Orchestrator** | set `live_runs.<family>.authorized=true` with integrated commit/build provenance; one run per changed family |
| **BLK-25B** (D-V3-25(B)) | **all** live runs incl. Antigravity scenario | **Product/OmniRoute Owner** | confirm UI invalidation/revocation of the exposed key (security stop) |
| **BLK-KIMI** | 5.7 | **OmniRoute architect** | publish exact versioned Cline→Kimi-K2.7 RouteModel + protocol + availability (then SUB-2 at `models.go:563` + one-line fixtures) |
| **BLK-OPUS48-ID** | 5.8 (Opus48) | **OmniRoute registry owner** | publish exact Opus48-on-AWS RouteModel row (anthropic-messages caps + provenance) |
| **BLK-AVAIL** | 5.6/5.7 live admission, 8.1 | **OmniRoute architect** | publish enriched `/v1/models` rows (`cp/cline-pass/glm-5.2` + Kimi ID): protocol/stream/tools/reasoning/context/pool/available |
| `8.5–8.7` | (external cert) | **OmniRoute owner** | certify inside OmniRoute; no Multica lane/QA/live run |

## 5. Orphan / hygiene
- No orphaned requirement: 5.6/5.7/5.8/8.1/8.2 all map to DONE code + PASS focused evidence + a single reserved live run. 8.5–8.7 correctly external.
- No duplicate validation: R1/R3 ran exactly one focused pass each on the integrated build; live acceptance is one run per family; no QA-A/B/C, no broad regression.
- No fabricated ID/evidence: Kimi/Opus48 RouteModels remain `BLOCKED_EXTERNAL` (never invented); Antigravity decision hash-pinned (A5); live-evidence consumers are reserved-empty until the authorized run.
- Checkbox closure remains Principal-only, after the accepted live run(s).

## 6. Living-doc note
As of `2026-07-22T00:05Z`, Main-Brain P0 is **code-complete + offline-validated**; the only open work is external live acceptance. **Refresh triggers (state re-read only — no rerun/second review):** (1) Principal issues a `live_run` token + D-V3-25(B) clears → the one GLM live run populates `R1-glm-live-acceptance.md` / `R3-glm-lifecycle-evidence.md` / A7/A8 terminal evidence, closing 5.6+8.1+8.2; (2) OmniRoute publishes BLK-KIMI / BLK-OPUS48-ID / BLK-AVAIL → 5.7 / 5.8 live runs; (3) Antigravity A5 scenario under the token. On accepted live evidence, this map appends final refs and checks out.
