# AUTHORITY AMENDMENT 001 — HUMAN OWNER IS THE SOLE DECISION MAKER

**Amends:** `MISSION_CHARTER_v3.md` → charter becomes **v3.1**
**Issued:** 2026-07-25 ~14:40 -03 by owner directive
**Status:** BINDING, effective immediately. Supersedes any conflicting text in the charter,
`DISPATCH_001_TM_TO_CODEX56.md`, or any prior instruction to any agent.
**Applies to:** Kiro-CLI TM, Codex56-TL, and all 9 executors — no exceptions, no seniority override.

---

## §0.0 ABSOLUTE AUTHORITY (overrides everything, including this charter)

**NO AGENT IN THIS PROJECT HAS DECISION POWER. ALL DECISION POWER BELONGS EXCLUSIVELY TO THE HUMAN
OWNER.** This applies without exception to Kiro-CLI TM, Codex56-TL, Kiro-Opus5, and all 9 executors.

- There is **no delegated authority**. No role, title, seniority level, "TL", "audit authority",
  "orchestrator" or "tech manager" label confers the right to decide anything in §0.1.
- Codex56-TL's **veto** (right to REJECT unverified/off-scope work) is a *quality gate only*. It can
  stop work; it can never authorise work. A rejection is not a decision, and an approval by the TL is
  not an authorisation to proceed on a STOP-AND-WAIT item.
- Codex56-TL's **commit authority** is conditional and downstream: it may only commit work whose
  underlying decisions the owner has already approved in writing (§0.1, §0.3).
- The TM's planning authority is **advisory**. Plans, waves, ETAs and prompts are *proposals* until the
  owner approves them in writing.
- Agreement between the two leads carries **zero** decision weight. Two agents agreeing is still not a
  decision — it is a recommendation with two signatures.
- No agent may infer, assume, imply or "reasonably conclude" owner consent. Only an explicit written
  answer from the owner counts. Silence, urgency, idleness, a blocked queue, a deadline, or a
  previously approved similar action are **never** substitutes for authorisation.
- Any agent that acts on a §0.1 item without the owner's written decision has committed a **critical
  defect**: it must stop, check out as `BLOCKED`, report exactly what it did, and the change must be
  proposed for revert to the owner.

---

## §0 HUMAN AUTHORITY GATE (implementation of §0.0)

The **human owner is the sole decision maker.** The TM and Codex56-TL are advisory and executional
only. Our joint-decision doctrine (§1.3) governs how we *form a recommendation* — it never grants us
authority to *decide*.

### 0.1 STOP-AND-WAIT class (requires owner decision **in writing** before any action)
Work halts at the decision point and waits. No partial execution, no "starting the easy part first".
- Any **refactor** of existing code or architecture.
- Any **rebuild** (images, binaries, runtimes) — including the OmniRoute affinity image.
- **Installing or removing** any package, runtime or service on ORQ2 or orq1 (e.g. a container runtime).
- Any **infrastructure change**: EC2, EBS, volumes, containers, ports, networking, Tailscale, systemd.
- Any **deletion or pruning**: images, containers, volumes, branches, files — even to reclaim disk.
- Any **spec authority change**: archiving, unarchiving, superseding, or reinstating an OpenSpec change.
- Any **commit, push, merge or PR**.
- Any change to **credentials, auth, access control** or secret handling.
- **Promotion to production** or any change touching a live environment.
- Anything else with **broad blast radius or hard reversibility** — when in doubt, it is in this class.

### 0.2 What the leads MAY do without waiting (read-only and reversible-local)
- Read code, docs, specs, logs, panes, and live inventory (read-only commands).
- Analyse, measure, diff, and reconcile.
- Produce plans, prompts, task breakdowns, ETAs, risk registers and recommendations.
- Ask the owner questions.

### 0.3 Required format for every escalation
Each item in `.deploy-control/escalations/` needing a ruling must carry:
`OPTIONS` (≥2, additive-first per §1.3) · `RECOMMENDATION` + reasoning · `RISK` · `REVERSIBILITY`
· `BLAST RADIUS` · `COST OF WAITING` · `WHAT UNBLOCKS IT`.
No item is "decided" until the owner has answered **in writing**. Verbal or inferred consent does not
count. Silence is not approval.

### 0.4 Reporting duty
The leads must proactively advise the owner *before* the decision point is reached — not after work has
begun. If an agent discovers mid-task that its unit requires a STOP-AND-WAIT action, it must halt that
unit, check out as `BLOCKED` with the exact reason, and escalate. Continuing is a defect.

### 0.5 Interaction with the zero-idle rule
The zero-idle rule (§5) never overrides this gate. If a lane is blocked awaiting an owner decision,
the agent is re-tasked onto other *approved* pending work — it is **never** unblocked by acting without
authorisation. "The agent was idle" is not a justification for proceeding.

---

## §0.6 LIVE INVENTORY CORRECTION (found while this amendment was drafted)
Codex56-TL's read-only inventory of orq1 contradicts the handover: **orq1 root is 91% full with only
2.4 GiB free**, not ~71%. The two broken OmniRoute affinity images alone hold 3.24 GB and 5.66 GB.
ORQ2 has ~27 GiB free. This materially changes blocker **B1** — any build-capacity decision now needs a
bounded build/cache policy and a disk guard, and the obvious "reclaim space by pruning images" move is
squarely in the STOP-AND-WAIT class (§0.1) because those images are the documented OmniRoute rollback
path. **No disk action of any kind without the owner's written decision.**

---
*Owner written decision recorded below (required before any STOP-AND-WAIT item proceeds):*

| # | Item | Owner decision | Date |
|---|---|---|---|
| B1 | Build capacity: runtime on ORQ2 vs build on orq1 (91% full) | _pending_ | |
| B2 | Canonical OpenSpec tree + contradictory/archived change authority | _pending_ | |
| B3 | Sync Cashless spec to ORQ2 | _pending_ | |
| B4 | Commit/push policy on the integration branch | _pending_ | |
| B5 | orq1 disk remediation (images are the rollback path) | _pending_ | |
