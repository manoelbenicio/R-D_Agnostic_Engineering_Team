# G4 Phase 1 evidence automation and reconciled consolidation

- Evidence record: `EV-G4-08`
- Disposition: **Partial**
- Schema: `agent-brain.g4-evidence.v1`
- Execution scope: deterministic synthetic/reference-only development validation
- Initial synthetic run: `2026-07-18T03:14:35Z`
- Gateway consolidation checkpoint: `2026-07-18T03:18:43Z`
- Runtime consolidation checkpoint: `2026-07-18T03:18:43Z`
- Reconciliation checkpoint: `2026-07-18T13:41:33Z`
- Acceptance claim: **no**
- G3 security-correction prerequisite: **SATISFIED — REVIEW-G3-02 ACCEPT**

The two required G3 correction artifacts are present, provenance-pinned, and
accepted by the independent `REVIEW-G3-02` re-review. That prerequisite is
satisfied. It does not convert synthetic G4 evidence into live/provider,
host-resource, tier, cutover, or active-path acceptance.

## Produced automation

The owned observability package now provides:

- a typed result schema whose prohibited fields exclude authorization material,
  credentials, cookies, account identity, prompts, message content, tool payloads,
  repository content, and reasoning content;
- a provenance manifest that accepts only supplied synthetic bytes and never opens
  or crawls a filesystem path;
- exact frozen-catalog coverage for all 116 `AC-*` checklist rows and all 44 parity
  rows (`P01`–`P34`, `SC01`–`SC10`);
- a consolidation gate that requires both independent G4 inputs, their paths, and
  SHA-256 provenance before consolidation; and
- a separate fail-closed G3 correction gate requiring both correction artifacts,
  SHA-256 provenance, and explicit independent pB re-review acceptance; and
- an invariant that synthetic evidence cannot have a `Supported` disposition,
  assert acceptance, or enable a capacity tier.

Implementation: `internal/daemon/observability/evidence.go`,
`internal/daemon/observability/synthetic.go`, and `g4_test.go`.

## Current evidence records

These are honest Phase 1 dispositions. `Partial` for a gateway-owned row means
the offline gateway portion independently passed while the runtime/provider or
approved system boundary remains unproven. `Not-supported` means the required
acceptance artifact is absent; it is not a negative result from a fabricated
protocol/provider run.

| Evidence ID | Phase 1 disposition | Frozen checklist/parity mapping | Produced evidence | Blocker retained |
|---|---|---|---|---|
| EV-G4-01 | Partial | AC-2.*, AC-6.*, AC-12.4; P15–P21, P27 | Independently rerun gateway G4 tests pass for offline mock Messages, Responses, Chat, and compatible Antigravity shapes, including synthetic model/capability health/readiness | no live wire/model-route result; Antigravity native path and runtime/provider combination remain unproven |
| EV-G4-02 | Partial | AC-2.2.*–AC-2.5.*; P15–P20 | Independently rerun runtime tests pass synthetic trusted-gateway Claude/Codex tools, reasoning signals, usage, cancellation, correlation, and deterministic errors | no installed CLI/live route; Kimi, GLM/NVIDIA/NIM, and Antigravity native contracts remain fail-closed or unaccepted |
| EV-G4-03 | Partial | AC-1.2, AC-7.*, AC-12.7; P01, P03, P30, P31 | Runtime tests pass synthetic child environment/home, test-owned Linux helper process-tree, command-line, and redacted diagnostic isolation | no live daemon/CLI/provider process or approved service security review |
| EV-G4-04 | Partial | AC-3.*, AC-9.6; P04–P06 | Gateway test proves an exact three-slot cycle across 96 overlapping independent selections plus three continuation-affinity families; capacity model is even at four synthetic slots | no OmniRoute concurrency/eligibility-duration measurement and runtime combination is incomplete |
| EV-G4-05 | Partial | AC-4.*, AC-5.*, AC-10.1–AC-10.7; P08–P14 | Gateway tests pass synthetic classifier/scope cases for expiry, revocation, quota, 401/403/429, 5xx, timeout, and malformed input | no live account/auth/quota/circuit transition or approved system boundary was exercised |
| EV-G4-06 | Partial | AC-5.7, AC-5.9, AC-6.3–AC-6.5; P07, P08, P21, P32 | Gateway test passes pre-output retry, post-output/tool no-replay, completed-request dedup, and exactly-once synthetic cancellation release | no runtime process-tree cancellation or live upstream capacity reconciliation |
| EV-G4-07 | Partial | AC-1.3–AC-1.5, AC-3.8–AC-3.10, AC-8.*, AC-10.10–AC-10.12; P02, P23–P26, P28–P29 | Gateway test passes in-memory registry/hot-change (P24), add/remove/quarantine/re-entry, stopped/restarted state, and snapshot rollback with concurrent workers | no OmniRoute/service restart, persisted-state recovery, or operational sign-off |
| EV-G4-08 | Partial | all 116 checklist IDs and all 44 parity IDs | exact catalog, record schema, provenance contract, both independently verified A/B inputs, and `g4-consolidated-matrix.md` with 160 row dispositions | 24 checklist and 5 parity rows remain Not-supported, no waivers exist, and live/provider/capacity gates remain open |
| EV-G4-CAP | Partial | AC-9.*, AC-12.6; P22, P33, P34 | deterministic development-only 20-task aggregate in `g4-synthetic-capacity-phase1.md` | values are virtual/modeled, thresholds are unapproved, and task 9.2 remains closed |

No `Supported` disposition is recorded in this Phase 1 artifact. No waiver is
present or inferred. Every `Not-supported` and live/provider/host-resource
blocker continues to stop cutover despite the satisfied G3 correction
prerequisite.

## G3 security-correction gate

| Required condition | State | G4 effect |
|---|---|---|
| `g3-security-corrections.md` | Present; SHA-256 `074807c51aa66ec67909393d76e694668a41cb3a8a3bfbcd20ed13e95b85b33c`; 3,684 bytes | accepted by REVIEW-G3-02 |
| `g3-security-corrections-adapters.md` | Present; SHA-256 `1efa057ba7bda4cab8813fc243b5d91402fc7a38704cd6a0d96517005695df37`; 3,286 bytes | accepted by REVIEW-G3-02 |
| `.planning/agent-brain-v3/evidence/g3-independent-security-rereview.md` / REVIEW-G3-02 | **ACCEPT**; SHA-256 `0806e2f54b0049396323ec83f171084fd5d3fe4a9f0326c4c2ecd4c89d9f665e`; 3,377 bytes; artifact timestamp `2026-07-18T04:10:50Z` | G3 correction prerequisite satisfied; no G4/tier acceptance implied |

The automation gate inputs now reconcile: both correction paths/digests are
present and the independent re-review is explicitly accepted. Remaining G4
blockers are the unsupported/live/provider/host-resource conditions recorded
in the matrix, not the superseded G3 correction gate.

## Consolidation gate checkpoint

| Required input | State at checkpoint | Consolidation action |
|---|---|---|
| `.planning/agent-brain-v3/evidence/g4-gateway-tests.md` | Present; SHA-256 `163b12c4403fb73dc5031e5d17770d61af8d7f83dd1b1f10cbad3cae7693eabd`; 15,500 bytes; current checkpoint has 16 G4 tests including 6 property-style tests and 59 top-level tests; the complete 25-file gateway set is pinned in `g4-provenance-manifest.md`; contained offline coverage reran at `2026-07-18T13:09:54Z` and reported 82.5%; earlier 15-test and 77.6%/79.1% figures are historical and superseded; toolchain `golang:1.26` / Go 1.26.5 linux/amd64, image digest `sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6` | accept only its synthetic offline gateway portion as `Partial` |
| `.planning/agent-brain-v3/evidence/g4-runtime-isolation.md` | Present; SHA-256 `17576dd1e61f2b93c52b6a2bc2ab72034be4530e1a922b0910bf4f3a8274f695`; focused/full, ten-count G4, race, and vet rerun passed | accept synthetic Claude/Codex and test-owned isolation portions as `Partial`; keep native paths closed |

The documentary substance of OpenSpec 8.8 is recorded in
`g4-consolidated-matrix.md`: all 116 checklist and 44 parity IDs have explicit
dispositions, evidence bases, and blockers. Task 8.8 documentary completion is
preserved. Unsupported rows and live/provider/host-resource gates stop cutover;
no waiver is supplied or inferred.

At the `2026-07-18T13:41:33Z` provenance checkpoint, the independently reviewed
RSS lifecycle correction disposition is **ACCEPT** for the exact source set:
`realtime_process.go` SHA-256
`d84fc659dd8bd5873760ed9a5965adb6007d9a454af7cdd30b3441fe607a293e`
(16,168 bytes), `realtime_process_linux.go` SHA-256
`f143cda7e712883481372f73666424e930036f261ab1ba8b5b59f17d6a66b131`
(5,409 bytes), and `realtime_process_linux_test.go` SHA-256
`4b53206c7a819323df4730a247c0b187e40cb07d6b4281783913e3ca84a5a81f`
(14,765 bytes). The accepted correction removes only the observed
RSS/process-exit portability defect; it is not task 9.1 execution, host-capacity
acceptance, tier authorization, or cutover evidence.

## Boundary statement

The independent gateway and runtime reruns used the pinned local Go container
and did not edit those packages. This work performed no live provider, OmniRoute, daemon dispatch, production,
cutover, Prodex removal, credential/auth/secret operation, tier enablement, tier
50/100 run, or native 5.6–5.8 acceptance. It did not edit central, gateway, or
runtimeenv implementation or entrypoints.
