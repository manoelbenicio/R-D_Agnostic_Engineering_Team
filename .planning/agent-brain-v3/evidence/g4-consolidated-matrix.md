# EV-G4-08 — consolidated G4 checklist and parity record

- Consolidation checkpoint: `2026-07-18T13:09:54Z`
- Gateway input: validated, SHA-256 `163b12c4403fb73dc5031e5d17770d61af8d7f83dd1b1f10cbad3cae7693eabd` (15,500 bytes)
- Runtime input: validated, SHA-256 `17576dd1e61f2b93c52b6a2bc2ab72034be4530e1a922b0910bf4f3a8274f695`
- Scope: synthetic/offline/reference-only G4 development evidence
- Waivers: none supplied or inferred
- Cutover: stopped for every remaining blocker row
- G3 security-correction prerequisite: **SATISFIED** — both correction artifacts
  are present and `REVIEW-G3-02` independently records **ACCEPT**

## Consolidation method

The matrix starts from the frozen G1 dispositions and admits only independently rerun A/B evidence. A synthetic result may upgrade a previously Not-supported row to Partial when it proves a bounded package behavior; it never upgrades a row to Supported. Supported rows below are unchanged G1 architectural/documentary results, not new live provider or capacity acceptance. Native Kimi, GLM/NVIDIA/NIM, and Antigravity paths remain fail-closed or unaccepted.

Independent reruns passed: gateway focused/full/race/vet; runtimeenv plus agent focused tests, ten-count G4 run, G4 race, and vet. No live endpoint, service, credential, daemon dispatch, or tier action occurred.

These A/B results do not establish a live/provider active path or host-resource
capacity. The two required G3 correction artifacts are present and the
independent `REVIEW-G3-02` re-review records **ACCEPT**. That closes only the G3
correction prerequisite; it does not accept G4, close a task, enable a tier, or
clear any blocker in this matrix.

## Disposition summary

| Catalog | Rows | Supported | Partial | Not-supported |
|---|---:|---:|---:|---:|
| OmniRoute checklist | 116 | 8 | 84 | 24 |
| Prodex parity P01-P34 + SC01-SC10 | 44 | 0 | 39 | 5 |

## Blocker legend

| Code | Meaning |
|---|---|
| B-PROVIDER-LIVE | exact live provider/model route, protocol, stream, or installed CLI boundary is absent |
| B-SECURITY-SYSTEM | synthetic child isolation is not an approved live service/process/security review |
| B-CAP-HOST | latency/resource values are virtual or modeled; approved host/process threshold evidence is absent |
| B-OMNI-LIVE | only in-memory gateway lifecycle/failure behavior was exercised; no live OmniRoute state/account operation |
| B-EXACT-MODEL | exact approved model-route conformance is absent |
| B-OMNI-OPS | persistent state, policy, health, or operational execution remains unproven |
| B-PARITY-GAP | the frozen parity remediation remains open |
| B-SC-NO-WAIVER | SC01-SC10 deterministic fixtures and an approved waiver are both absent |
| B-G1-GAP | frozen checklist remediation remains open and was not closed by A/B evidence |

## OmniRoute acceptance checklist (116 rows)

| ID | G4 disposition | Evidence basis | Explicit blocker |
|---|---|---|---|
| AC-1.1 | Partial | G1 + RT/EV-G4-03 + GW/EV-G4-07 | B-G1-GAP |
| AC-1.2 | Partial | G1 + RT/EV-G4-03 + GW/EV-G4-07 | B-G1-GAP |
| AC-1.3 | Partial | G1 + RT/EV-G4-03 + GW/EV-G4-07 | B-G1-GAP |
| AC-1.4 | Not-supported | G1 + RT/EV-G4-03 + GW/EV-G4-07 | B-G1-GAP |
| AC-1.5 | Partial | G1 + RT/EV-G4-03 + GW/EV-G4-07 | B-G1-GAP |
| AC-2.1.1 | Partial | G1 + GW/EV-G4-01 | B-PROVIDER-LIVE |
| AC-2.1.2 | Partial | G1 + GW/EV-G4-01 | B-PROVIDER-LIVE |
| AC-2.1.3 | Not-supported | G1 + GW/EV-G4-01 | B-PROVIDER-LIVE |
| AC-2.1.4 | Partial | G1 + GW/EV-G4-01 | B-PROVIDER-LIVE |
| AC-2.2.1 | Supported | G1 + GW/EV-G4-01 + RT/EV-G4-02 | none for frozen row; broader G4 gates remain |
| AC-2.2.2 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.2.3 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.2.4 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.2.5 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.2.6 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.2.7 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.2.8 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.1 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.2 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.3 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.4 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.5 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.6 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.7 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.3.8 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.4.1 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.4.2 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.4.3 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.4.4 | Not-supported | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.5.1 | Not-supported | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.5.2 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.5.3 | Partial | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.5.4 | Not-supported | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-2.5.5 | Not-supported | G1 + GW/EV-G4-01 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-3.1 | Supported | G1 + GW/EV-G4-04 | none for frozen row; broader G4 gates remain |
| AC-3.2 | Supported | G1 + GW/EV-G4-04 | none for frozen row; broader G4 gates remain |
| AC-3.3 | Supported | G1 + GW/EV-G4-04 | none for frozen row; broader G4 gates remain |
| AC-3.4 | Partial | G1 + GW/EV-G4-04 | B-G1-GAP |
| AC-3.5 | Partial | G1 + GW/EV-G4-04 | B-G1-GAP |
| AC-3.6 | Supported | G1 + GW/EV-G4-04 | none for frozen row; broader G4 gates remain |
| AC-3.7 | Partial | G1 + GW/EV-G4-04 | B-G1-GAP |
| AC-3.8 | Partial | G1 + GW/EV-G4-04 | B-G1-GAP |
| AC-3.9 | Partial | G1 + GW/EV-G4-04 | B-G1-GAP |
| AC-3.10 | Not-supported | G1 + GW/EV-G4-04 | B-G1-GAP |
| AC-4.1 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.2 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.3 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.4 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.5 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.6 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.7 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.8 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.9 | Not-supported | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.10 | Partial | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-4.11 | Not-supported | G1 + GW/EV-G4-05 | B-G1-GAP |
| AC-5.1 | Supported | G1 + GW/EV-G4-05/06 | none for frozen row; broader G4 gates remain |
| AC-5.2 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.3 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.4 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.5 | Not-supported | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.6 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.7 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.8 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.9 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.10 | Partial | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-5.11 | Not-supported | G1 + GW/EV-G4-05/06 | B-G1-GAP |
| AC-6.1 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-6.2 | Not-supported | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-6.3 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-6.4 | Not-supported | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-6.5 | Not-supported | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-6.6 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| AC-7.1 | Partial | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.2 | Partial | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.3 | Partial | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.4 | Partial | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.5 | Not-supported | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.6 | Not-supported | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.7 | Not-supported | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-7.8 | Not-supported | G1 + RT/EV-G4-03 + EV-G2D-01/03 | B-SECURITY-SYSTEM |
| AC-8.1 | Partial | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-8.2 | Not-supported | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-8.3 | Partial | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-8.4 | Partial | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-8.5 | Not-supported | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-8.6 | Partial | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-8.7 | Not-supported | G1 + GW/RT + EV-G2D-03/04 | B-G1-GAP |
| AC-9.1 | Supported | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | none for frozen row; broader G4 gates remain |
| AC-9.2 | Partial | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-9.3 | Partial | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-9.4 | Partial | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-9.5 | Not-supported | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-9.6 | Partial | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-9.7 | Partial | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-9.8 | Partial | G1 + GW/EV-G4-04 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-10.1 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.2 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.3 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.4 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.5 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.6 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.7 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.8 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.9 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.10 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.11 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-10.12 | Partial | G1 + GW/EV-G4-05/06/07 | B-OMNI-LIVE |
| AC-12.1 | Supported | G1 + EV-G4-08 + SYN20/EV-G4-CAP | none for frozen row; broader G4 gates remain |
| AC-12.2 | Partial | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-G1-GAP |
| AC-12.3 | Partial | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-G1-GAP |
| AC-12.4 | Not-supported | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-EXACT-MODEL |
| AC-12.5 | Partial | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-G1-GAP |
| AC-12.6 | Not-supported | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-CAP-HOST |
| AC-12.7 | Partial | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-G1-GAP |
| AC-12.8 | Partial | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-G1-GAP |
| AC-12.9 | Partial | G1 + EV-G4-08 + SYN20/EV-G4-CAP | B-G1-GAP |

## Prodex feature parity (44 rows)

| ID | G4 disposition | Evidence basis | Explicit blocker / waiver |
|---|---|---|---|
| P01 | Partial | G1 + RT/EV-G4-03 + OPS | B-PARITY-GAP |
| P02 | Partial | G1 + RT/EV-G4-03 + OPS | B-PARITY-GAP |
| P03 | Partial | G1 + RT/EV-G4-03 + OPS | B-PARITY-GAP |
| P04 | Partial | G1 + GW/EV-G4-04 | B-PARITY-GAP |
| P05 | Partial | G1 + GW/EV-G4-04 | B-PARITY-GAP |
| P06 | Partial | G1 + GW/EV-G4-04 | B-PARITY-GAP |
| P07 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P08 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P09 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P10 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P11 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P12 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P13 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P14 | Partial | G1 + GW/EV-G4-05/06 | B-PARITY-GAP |
| P15 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P16 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P17 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P18 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P19 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P20 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P21 | Partial | G1 + GW/EV-G4-01/06 + RT/EV-G4-02 | B-PROVIDER-LIVE |
| P22 | Partial | G1 + SYN20/EV-G4-CAP | B-CAP-HOST |
| P23 | Not-supported | G1 + GW/EV-G4-07 + OPS | B-OMNI-OPS |
| P24 | Partial | G1 + GW/EV-G4-07 + OPS | B-OMNI-OPS |
| P25 | Not-supported | G1 + GW/EV-G4-07 + OPS | B-OMNI-OPS |
| P26 | Not-supported | G1 + GW/EV-G4-07 + OPS | B-OMNI-OPS |
| P27 | Partial | G1 + GW/EV-G4-01 + OPS | B-OMNI-OPS |
| P28 | Not-supported | G1 + GW/EV-G4-07 + OPS | B-OMNI-OPS |
| P29 | Partial | G1 + GW/EV-G4-07 + OPS | B-OMNI-OPS |
| P30 | Partial | G1 + RT/EV-G4-03 + GW/EV-G4-06 | B-PARITY-GAP |
| P31 | Not-supported | G1 + RT/EV-G4-03 + GW/EV-G4-06 | B-PARITY-GAP |
| P32 | Partial | G1 + RT/EV-G4-03 + GW/EV-G4-06 | B-PARITY-GAP |
| P33 | Partial | G1 + SYN20/EV-G4-CAP | B-CAP-HOST |
| P34 | Partial | G1 + SYN20/EV-G4-CAP | B-CAP-HOST |
| SC01 | Partial | G1 only | B-SC-NO-WAIVER |
| SC02 | Partial | G1 only | B-SC-NO-WAIVER |
| SC03 | Partial | G1 only | B-SC-NO-WAIVER |
| SC04 | Partial | G1 only | B-SC-NO-WAIVER |
| SC05 | Partial | G1 only | B-SC-NO-WAIVER |
| SC06 | Partial | G1 only | B-SC-NO-WAIVER |
| SC07 | Partial | G1 only | B-SC-NO-WAIVER |
| SC08 | Partial | G1 only | B-SC-NO-WAIVER |
| SC09 | Partial | G1 only | B-SC-NO-WAIVER |
| SC10 | Partial | G1 only | B-SC-NO-WAIVER |

## Metrics cross-reference

The deterministic Phase 1 profile reports 20 offered/admitted tasks, 17 completed, 1 failed, 2 cancelled, peak queue 6, selection p50/p95/p99 6/10/10 ms, queue 164/333/351 ms, first output 70/100/100 ms, request 451/666/666 ms, 4 retries, 4 fallbacks, and zero modeled four-slot fairness deviation. Modeled peaks are 590 mCPU, 80 MiB, and 10 sockets. Current gateway evidence contains 16 G4 tests, including six property-style tests, within 59 top-level package tests: 24 seeded round-robin iterations at 96–160 requests, 48 affinity keys with 32 continuations each, 384 cancellations, 512 retry plans plus 128 duplicate deliveries, 64 circuit timelines, and 128 lifecycle operations with 32 concurrent selections. It also proves an exact three-slot cycle over 96 overlapping independent selections and continuation affinity. The freshly pinned complete gateway set passed the contained offline coverage command at 82.5%; earlier 15-test and 77.6%/79.1% figures are historical and superseded. These values are not live provider, OmniRoute, host SLO, or tier-20 acceptance measurements.

## Final G4 Phase 1 disposition

EV-G4-08 is Partial: both synthetic A/B artifacts are present, provenance-validated, independently rerun, and mapped to every frozen row. The matrix intentionally retains unsupported blockers and no waivers; therefore cutover, native adapter acceptance, OpenSpec 9.1 capacity acceptance, and Codex1-only tier enablement remain stopped. No task 9.2 action is authorized by this record.

`REVIEW-G3-02` satisfies the G3 correction prerequisite. This record still does
not mark G4 accepted or claim a live/provider active path. Task 8.8 documentary
completion is preserved. The independently reviewed RSS lifecycle correction
is **ACCEPT** at source-set SHA-256
`53f25cf1f01ed7cd66cd097b9ec0f71713188c6a9a1641f05c346881531c94c8`;
it changes no matrix row and does not satisfy capacity evidence. **Recommendation
for task 9.1: BLOCK** because its only 20-task result is virtual/modeled, numeric
SLO thresholds are unapproved, and approved host/process CPU, memory, socket,
and deployed-topology measurements do not exist.
