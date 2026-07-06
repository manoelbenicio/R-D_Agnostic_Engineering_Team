---
agent: GLM#52#CLINE#B
stream: G1-TRIPLE
phase: G1
task: Validate TRIPLE-interaction CODEX_HOME x prodex x Herdr-codex-integration coexistence; produce docs/qa/triple-interaction-coexistence.md (NEW, disjoint) — how all 3 touch CODEX_HOME/hooks, conflict risks, and a TEST PLAN proving they coexist without clobbering the account pool or isolation. LIVE proof F0-GATED; deliver plan+criteria now.
priority: P1
status: DONE
progress: 100
eta: done
started_at: 2026-07-04T20:28:35Z
finished_at: 2026-07-04T20:34:22Z
depends_on: [F2 prodex fork map, F3 Go integration skeleton]
blockers: none
build_result: green — docs/qa/triple-interaction-coexistence.md delivered (446 lines): 3 CODEX_HOME touchers mapped (daemon execenv, prodex, Herdr-codex-integration), hooks unmanaged surface flagged, 8 conflict risks (R1-R8), 10 gated tests (T1-T10) with pass/fail-closed criteria, 10 isolation invariants, F0-gated acceptance criteria. READ-ONLY honored; no product code edited; no live execution; no deploy.
notes: DONE. LIVE proof F0-GATED — plan+criteria only. Key findings: prodex.go does NOT set CODEX_HOME (inherits daemon per-task — single owner, T2 asserts); dominant risk R1 (auth.json clobber via shared ~/.codex symlink when AccountHome unset, or Herdr-pane codex defaults to ~/.codex) mitigated by making AccountHome mandatory in coexistence topologies; subtle risk R5 (hooks unmanaged) requires T7 to close. ACKs for STATUS_REPORTING_STANDARD + HERDR_COMMS_GUIDE recorded in GLM-52-CLINE-B__REDACTION-AUDIT__20260704T194546Z.md. Reach POC via ping-opus.sh.
---

# Check-in: GLM#52#CLINE#B — G1 Triple-Interaction Coexistence

## Scope (from dispatch)

Validate that three CODEX_HOME touchers coexist without clobbering the account pool or per-account isolation:

1. **Multica daemon execenv** — per-account `CODEX_HOME` isolation (restored `auth.json`, memory, skills, sandbox).
2. **prodex (Rust L2 runtime plane)** — launches codex; may set/override `CODEX_HOME` / `PRODEX_HOME` env for the codex it runs.
3. **Herdr-codex-integration** — a codex agent running inside a Herdr pane (tmux-style multiplexer), which also reads `CODEX_HOME` and possibly `hooks`.

## Deliverable

`docs/qa/triple-interaction-coexistence.md` — disjoint from existing `docs/qa/` files (prod-redeem-validation-checklist, runtime-conformance-plan, smart-context-shadow-canary-plan).

## Constraints

- LIVE proof F0-GATED — no live execution this pass; deliver plan + acceptance criteria.
- READ-ONLY for product code; only the new QA doc + this check-in are written.
- No deploy.

## Status

DONE — see build_result above and `docs/qa/triple-interaction-coexistence.md`.
