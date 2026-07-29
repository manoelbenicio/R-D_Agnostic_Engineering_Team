# W1 — GLM Live Readiness (P0-W1-LIVE-READINESS)

- agent: `Opus48#B` · pane `w6:p2` · lane `W1-LIVE` · task `P0-W1-LIVE-READINESS`
- lock (only mutable file): `.deploy-control/p0/handoffs/W1-live-readiness.md`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-LIVE-READINESS__20260722T000445Z.json`
- repo: branch `integration/dev-transition-candidate-20260719`, HEAD `a6d50986…`; Go `/home/ec2-user/goroot/go/bin/go` (go1.26.1)
- posture: **readiness record only — no test rerun, no live execution, no source edit.**
- supersedes: `P0-W1-FINAL-INTEGRATION-EVIDENCE` (DONE). Full detail: `W1-implementation-D6.md`, `W1-final-integration-evidence.md`.

## 1. Implementation complete — D1–D7 (evidence refs, not re-run)

W1 source (7): `runtimeenv/adapter.go` (D1), `runtimeenv/env.go` (D2/D6-4), `internal/daemon/config.go`
(D3/D4), `internal/daemon/execenv/cline_home.go` (D5), `internal/daemon/brain_integration.go` (D6),
`runtimeenv/assert.go` (D6-1/2/3), `pkg/agent/models.go` (D7 → `cp/cline-pass/glm-5.2`).
W1 tests (2): `runtimeenv/assert_test.go`, `pkg/agent/models_test.go`.

Focused results (cited): runtimeenv Assert/Cline `ok 0.006s`; pkg/agent StaticModels/Cline `ok 0.016s`;
`go build` daemon+execenv+runtimeenv+pkg/agent OK; gofmt clean.
- **R1** (`Opus48-A__P0-R1-FINAL-INTEGRATION-VALIDATION__20260721T232302Z`): **DONE — PASS 0.017s, no edits.**
- **R3** (`Opus48-D__P0-R3-FINAL-INTEGRATION-VALIDATION__20260721T235649Z`): **DONE — NO-REVALIDATION-NEEDED.**

## 2. No source locks / no edits needed

W1 holds **no product-source locks** now: all source-lock tasks (`P0-W1-SOURCE-LOCK-ACTIVATION`,
`P0-W1-IMPLEMENTATION`, `P0-W1-IMPLEMENTATION-D6`) are checked out DONE and released their locks. The
working tree carries exactly the 7 D1–D7 source files + coupled tests (R1/R3-owned test files included),
all build-clean and validated. **The GLM live run requires no further source edit, no new lock, and no
integration change** — it exercises the already-integrated launch path.

## 3. Terminal state — BLOCKED solely on external GLM live prerequisites

| Prerequisite | State | Owner |
|---|---|---|
| `control.json live_runs.cline_glm.authorized` | **false** | Principal Orchestrator (issues token) |
| **BLK-AVAIL** — OmniRoute enriched `/v1/models` rows for `cp/cline-pass/glm-5.2` | unpublished | OmniRoute registry owner |
| **BLK-25B** — live-provider security stop until exposed key revoked | active | Product/OmniRoute security |

## 4. Next action (on external clear)

When BLK-AVAIL + BLK-25B clear and the Principal authorizes `live_runs.cline_glm`:
1. W1 verifies **integrated provenance** (post-integration commit/build/config: HEAD, `go build` clean,
   frozen `cp/cline-pass/glm-5.2` RouteModel, no pending diff) — provenance-only, no test rerun.
2. W1 hands the **single reserved live-run token** to **Codex56#A** (`P0-GLM-LIVE-ACCEPTANCE`,
   `evidence/R1-glm-live-acceptance.md`), who runs the **one** Kanban→terminal GLM acceptance closing
   overlapping 5.6/8.1/8.2. Exactly one run per family; no duplicate.

Out of scope / still external: Kimi 5.7 (BLK-KIMI), Opus48 5.8 (BLK-OPUS48), Antigravity live scenario
(BLK-25B); `8.5–8.7` OmniRoute-only. None is W1 source work.

## 5. Status: BLOCKED — external GLM live prerequisites only
- **Blocker:** `live_runs.cline_glm.authorized=false` + BLK-AVAIL + BLK-25B.
- **Owner:** OmniRoute registry / Product security (BLK-AVAIL, BLK-25B) + Principal Orchestrator (live token).
- **Next:** on external clear, W1 verifies integrated provenance and hands one run token to Codex56#A;
  no W1 test rerun, live execution, or source edit.
