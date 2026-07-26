# EXECUTION PLAN v3 — AGENTIC PARALLEL PLAN (PROPOSAL)

> **STATUS: PROPOSAL. NOT AUTHORISED. NOTHING IN THIS PLAN EXECUTES.**
> Per `AUTHORITY_AMENDMENT_001.md` §0.0 the **human owner is the sole decision maker**. This document is
> a recommendation with two signatures (TM + Codex56-TL) and carries zero decision weight.
> **Author:** Kiro-CLI TM (`.planning/**` is TM-authored per EXECUTION_PLAN_v2.1 rule 4)
> **Audit:** Codex56-TL (pending — DISPATCH-001 in flight)
> **Date:** 2026-07-25 · **Framework:** GSD + OpenSpec · **Fleet:** 9 executors on ORQ2

---

## 1. HEADLINE — WHAT THE EVIDENCE SAYS

The handover documents and the live ORQ2 state **do not agree**, and the gap is large enough that
dispatching 9 agents today would send most of them at specs that do not exist on the host.

| Handover claims (2026-07-24) | Verified on ORQ2 (2026-07-25) |
|---|---|
| 16 openspec changes incl. flagship `rotation-parity-polyglot` **30/66** | Canonical tree has **4** active changes; **no `rotation-parity-polyglot` directory** |
| `agent-credential-isolation` **0/21**, to be started | **Archived** with an explicit *MUST NOT be dispatched* supersession |
| orq1 root disk ~**71%** used | **91% used, 2.4 GiB free** (Codex56-TL read-only inventory) |
| — | `rotation-router` = SUPERSEDED **by a change that is absent** → runtime authority orphaned |
| — | Six competing `openspec/` trees on ORQ2 alone |

**Verified canonical open work is 12 tasks, not ~120.** Total remaining scope cannot be sized until the
OpenSpec authority question (B2) is closed by the owner.

---

## 2. CANONICAL OPEN TASKS (verified on disk, ORQ2 project tree)

Legend — **GATE**: 🔴 = requires owner **written** decision before it can run (Amendment §0.1) ·
🟢 = executable once authorised · **ETA** = estimated agent-hours, **estimate not commitment**;
confidence L/M/H.

### 2.1 `build-omniroute-agent-brain` — 26/28
| Task | OpenSpec ref | Slot | ETA | Conf | Dep | GATE | Proof |
|---|---|---|---|---|---|---|---|
| Validate + approve 20-task bounded capacity profile; hold lower limit until evidence passes | 6.3 | W5 | 3h | M | — | 🔴 *"approve" is an owner act, not an agent act* | capacity run evidence in `.deploy-control/evidence/` |
| Validate 50/100-task profiles after lower tier + observability gates pass | 6.4 | W5 | 4h | L | 6.3 | 🔴 | tiered capacity evidence + observability gate green |

### 2.2 `chat-orchestration-standard` — 8/10
| Task | OpenSpec ref | Slot | ETA | Conf | Dep | GATE | Proof |
|---|---|---|---|---|---|---|---|
| Smoke: chat `@codex` reaches the agent directly (escape hatch works) | 2.2 | W8 | 1.5h | M | — | 🟢 | smoke transcript + scrubbed log |
| Check-ins DONE + evidence in `.deploy-control/` | 2.3 | W8 | 0.5h | H | 2.2 | 🟢 | check-out file + evidence artifact |

### 2.3 `native-runtimes-onboarding` — 9/17
| Task | OpenSpec ref | Slot | ETA | Conf | Dep | GATE | Proof |
|---|---|---|---|---|---|---|---|
| Agent-5 FRONTEND onboarding: remove `(landing)`/`features/landing`/`content/use-cases`/sponsors + email-code flow; `AuthService` iface + `SimpleAuthService` | 1.5 | W2 | 6h | M | — | 🔴 **refactor + deletions** | web build green + UAT 3.3 |
| Agent-6 design parity (tokens, kanban/agent colours), i18n cleanup, web build/test harness + QA. **Does not touch marketing (that is 1.5)** — disjoint files | 1.6 | W3 | 5h | M | — | 🟢 (no deletions) | design parity diff + harness green |
| Rebuild `server/bin/multica` + backend image; restart daemon; `nim`/`cline` runtimes online | 2.4 | W1 | 3h | M | 1.5,1.6 | 🔴 **rebuild + daemon restart** | binary sha + runtimes online evidence |
| Build + run web locally; validate new onboarding | 2.5 | W3 | 2h | M | 1.5,1.6,2.4 | 🔴 (build) | web up + onboarding walkthrough |
| Green Go + web tests **in container** | 3.1 | W4 | 2h | L | 2.4, **B1** | 🔴 **needs a container runtime — none on ORQ2** | container test run, IPv6 off |
| Smoke: create agent in `nim` and `cline`, run 1 task, observe execution + tokens | 3.2 | W6 | 2h | M | 2.4 | 🟢 | smoke evidence + token counts |
| UAT onboarding (no sponsors/email-code; colours identical) | 3.3 | W7 | 1.5h | M | 2.5 | 🟢 | UAT record |
| Check-ins DONE + integration report in `.deploy-control/` | 3.4 | W9 | 1h | H | 3.1,3.2,3.3 | 🟢 | integration report |

**Canonical subtotal: 12 tasks · 31.5 agent-hours · 7 of 12 gated on owner decisions.**

### 2.4 `rotation-router` — status only, no tasks.md
SUPERSEDED 2026-07-04 by `rotation-parity-polyglot`, **which is not present on ORQ2**. Runtime authority
(selection, rotation, fallback, load-balancing, in-flight reset → prodex/Rust L2) is therefore
**orphaned**: the superseded change is here, the superseding one is not. Go L4 keeps Account Registry,
policy definition, observability, governance. **No task can be planned against this until B2 is ruled.**

---

## 3. WAVE 0 — RECONCILIATION (read-only, no owner decision needed to *perform*)

This is the only work I recommend starting immediately: it is entirely read-only, it produces the
evidence the owner needs to rule on B1–B5, and it cannot damage anything.

| # | Task | Owner | ETA | Conf | Output |
|---|---|---|---|---|---|
| R1 | OpenSpec **authority map**: reconcile 6 trees; locate/confirm `rotation-parity-polyglot`; resolve the orphaned supersession chain; flag every contradictory or archived-but-planned change | TM + Codex56-TL | 3h | M | `AUTHORITY_MAP.md` + canonical-tree recommendation |
| R2 | **Handover ↔ canonical delta**: every change the handover cites that is absent on ORQ2, with where it actually lives | Codex56-TL | 2h | M | delta table for owner |
| R3 | **orq1 disk risk register** (read-only): 91%/2.4 GiB free vs 3.24 GB + 5.66 GB rollback images | W5 | 1h | H | risk register + options (no action) |
| R4 | Reconcile **`w7:p3` in-flight work** — read-only inspection, do not interrupt | Codex56-TL | 0.5h | M | in-flight report |
| R5 | Extend the **single-writer file-ownership matrix** to all 9 lanes (GOLDEN RULE 2; `internal/daemon` serial) | TM | 2h | M | ownership matrix |
| R6 | **Teach 9 workers the comms protocol** + verify round-trip to `w5:pB` | Codex56-TL | 1.5h | H | 9 ack receipts |
| R7 | **Heartbeat poller spec** for the 90s zero-idle audit (spec only — building it needs owner approval) | TM | 1h | H | poller spec |
| R8 | **Program C sync plan** — how to bring `cashless-arraia-2026` to ORQ2 (plan only) | W8 | 1h | M | sync proposal |

**Wave 0: 12 agent-hours · ~4h wall-clock with parallelism · 0 owner decisions required to start.**

---

## 4. 🔴 OWNER DECISION GATE (all waves after 0 are blocked here)

Nothing past Wave 0 runs until each is answered **in writing** (Amendment §0.3 format).

| # | Decision needed | Blocks |
|---|---|---|
| **D1/B1** | Build capacity: install runtime on ORQ2 (27 GiB free) **with** bounded build/cache policy + disk guard, vs build on orq1 (2.4 GiB free) | NRO 3.1, 2.4, 2.5; all container evidence |
| **D2/B2** | Canonical OpenSpec tree + what to do about the orphaned `rotation-parity-polyglot` supersession | every Program A lane |
| **D3/B3** | Sync Cashless spec to ORQ2; concurrent with A+B or after | Program C entirely |
| **D4/B4** | Commit/push policy on `integration/dev-transition-candidate-20260719` | every DONE |
| **D5/B5** | orq1 disk remediation — the broken images **are** the documented rollback path | Program B |
| **D6** | Approve the frontend refactor + deletions in NRO 1.5 | NRO 1.5, 2.5, 3.3 |
| **D7** | Approve rebuild + daemon restart in NRO 2.4 | NRO 2.4, 3.1, 3.2 |
| **D8** | OmniRoute affinity image rebuild — yes/no and acceptance criteria | Program B |
| **D9** | Approve building the heartbeat poller on ORQ2 | unattended zero-idle enforcement |
| **D10** | Capacity profile approval (BOAB 6.3/6.4) — an owner act by the task's own wording | BOAB closeout |

---

## 5. WAVE 1+ — PARALLEL EXECUTION (shape only; assignments final after D1–D10)

Wave 1 is drawn only from 🟢 tasks plus whatever D1–D10 unlocks. Every unit carries: acceptance
criteria, exact proof command, disjoint `files_locked`, `depends_on`, and the herdr comms block.

```
WAVE 1 (parallel, no gated deps)          ~5h wall-clock
  W3  NRO 1.6  design parity + harness            5h   🟢
  W8  COS 2.2  escape-hatch smoke                1.5h  🟢
  W8  COS 2.3  check-ins + evidence              0.5h  🟢 (dep 2.2)
  W2/W1/W4/W5/W6/W7/W9 → Wave 0 residue + prompt authoring; NOT idle,
      but NOT permitted to touch 🔴 work (Amendment §0.5)

WAVE 2 (unlocks on D6 + D7)               ~8h wall-clock
  W2  NRO 1.5  frontend refactor                  6h   🔴 D6
  W1  NRO 2.4  rebuild + daemon restart           3h   🔴 D7 (dep 1.5,1.6)
  W3  NRO 2.5  web build + onboarding validate    2h   🔴 (dep 2.4)

WAVE 3 (unlocks on D1)                    ~3h wall-clock
  W4  NRO 3.1  container-green Go + web           2h   🔴 D1
  W6  NRO 3.2  nim/cline smoke                    2h   🟢 (dep 2.4)
  W7  NRO 3.3  UAT onboarding                   1.5h   🟢 (dep 2.5)
  W9  NRO 3.4  integration report                 1h   🟢 (dep 3.1-3.3)

WAVE 4 (unlocks on D10)                   ~7h wall-clock
  W5  BOAB 6.3 / 6.4 capacity profiles            7h   🔴 D10

BLOCKED-UNSIZED (needs D2/D3 before it can even be decomposed)
  rotation-parity-polyglot (flagship), Program B OmniRoute, Program C Cashless,
  and every handover item absent from the canonical tree
```

**Sized total: 43.5 agent-hours (12 canonical + Wave 0). Critical path ≈ 20h wall-clock**, and that path
runs almost entirely through owner decisions, not through engineering effort. Unsized scope (flagship +
Programs B and C) is deliberately not estimated — estimating specs I cannot read would be invented work
(GOLDEN RULE 4).

---

## 6. GSD + OPENSPEC CONFORMANCE
- **OpenSpec:** every task above is traced to a real change + task number verified on disk. No task is
  invented. Changes are only touched via delta specs; archive/unarchive/supersede is 🔴 owner-only.
- **GSD:** spec → plan → execute → verify per lane; per-task proof command = the verification step;
  evidence under `EVIDENCE_CONTRACT.md`; check-in/out per `GOLDEN_RULES_E_CHECKIN.md` into
  `.deploy-control/`; atomic commits by the TL only, after owner approval (Amendment §0.0).
- **Gates:** no bypass (GOLDEN RULE 8). Container-green evidence stands as a requirement — which is
  exactly why D1 blocks NRO 3.1 rather than being waived.
- **Zero-idle:** 90s pane audit per charter §5. Blocked-on-owner lanes are re-tasked to approved work,
  **never** unblocked by acting without authorisation (Amendment §0.5).

## 7. ETA CAVEAT
All ETAs are estimates in agent-hours with confidence marks, derived from task text and observed repo
state — not commitments. Confidence L means the task text is ambiguous or its dependency is unresolved.
Wall-clock assumes disjoint ownership and no rework. Every ETA is re-baselined after Codex56-TL's audit
and after the owner rules on D1–D10.

---
*TM signature: Kiro-CLI TM, 2026-07-25 · TL audit: pending · **Owner authorisation: NOT GIVEN***
