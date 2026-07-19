---
name: herdr-sync
description: "Sync GSD Agent Brain v3 state to the live Herdr fleet: read STATE.md/AGENT_LEDGER, surface per-phase blockers/ETA/risks, and report to the product owner in the mandated format. Use when the user asks for a status report, blocker review, or to reconcile GSD state with what the Herdr panes actually show. Read-only governance."
---

# herdr-sync — GSD v3 state ↔ live Herdr fleet reconciliation

Read-only governance skill. Keeps `.planning/agent-brain-v3/` state aligned with what the
live Herdr panes actually show, and produces owner reports in the mandatory format.

## Preconditions

1. Herdr gate (only if you need to inspect live panes):
   ```bash
   test "${HERDR_ENV:-}" = 1
   ```
   Fail → you can still produce a GSD-only report (no live cross-check); say so.

2. Never mutate production. This skill reads STATE.md/AGENT_LEDGER/EVIDENCE_INDEX and
   (optionally) herdr panes, and writes only to GSD governance files under
   `.planning/agent-brain-v3/` (STATE.md, AGENT_LEDGER.md, EVIDENCE_INDEX.md). It does not
   write product code, does not dispatch agents, and does not run prodex.

## Source files (the GSD v3 ledger)

Located at `C:\VMs\Projects\RD_Agnostic_Engineering_Team\.planning\agent-brain-v3\`:

- `STATE.md` — live per-phase state, last verified fact, blockers, next authorized action.
- `AGENT_LEDGER.md` — check-in/out per Codex stream.
- `EVIDENCE_INDEX.md` — immutable evidence IDs per phase (PLANNED = not yet produced).
- `ROADMAP.md` — G0–G8, gates, ETA (not-linear).
- `DECISIONS.md` — decisions v3 + absorbed legacy + pending-owner decisions PD-01..07.
- `RISKS.md` — risk register.
- `TRACEABILITY.md` / `COMPONENT_REGISTER.md` / `INTERFACE_REGISTER.md` /
  `REMOVAL_REGISTER.md` / `FILE_OWNERSHIP.md` / `REQUIREMENTS.md` — full registers.

## Cross-check live fleet (inside Herdr only)

```bash
herdr --help
herdr workspace list
herdr pane list --workspace "$HERDR_WORKSPACE_ID"
herdr pane get <pane-id>     # agent, agent_status (idle/working/blocked/done/unknown)
```
- `idle` = waiting, result seen; `done` = finished, result unseen. Both = completed.
- Parse IDs from JSON; never derive from sidebar order. Do not run bare `herdr`. Do not
  mutate via this skill (this is the read-only governance skill).

Reconcile: does the AGENT_LEDGER match what the panes show? Is a pane `blocked` but its
ledger row claims IN_PROGRESS? Flag mismatches as findings, do not silently fix code.

## Mandatory owner-report format (from CLAUDE_GLM52_TL_PROFILE.md / handover §13)

Every report to the owner contains exactly these fields, in Portuguese:

```text
phase
task IDs
agents ativos
owners / files locked
progresso real
evidence IDs
decisões tomadas
blockers
riscos novos
ETA atualizado
próxima ação que exige autorização
```
- Never declare "completo" from a code description alone. Cite real artifacts/evidence.
- Separate reviewed · implemented · verified · accepted.
- Authorization gate reminder: §7.1 = AGUARDAR means implementation is NOT authorized;
  the report must say so and not imply progress on implementation.

## When to update STATE.md / AGENT_LEDGER (writes this skill may do)

- After a verified status change from a `herdr-dispatch` loop (worker DONE/BLOCKED).
- After the owner resolves a pending decision (PD-0x) — record it in DECISIONS.md and
  reflect in STATE.md blockers.
- After an orphan-audit pass (task 0.6) — update TRACEABILITY.md §C.
- Never fabricate progress. Never mark an evidence ID DONE without a real artifact path
  in EVIDENCE_INDEX.md. Keep honest BLOCKED entries; do not delete fabrication history.

## Hard rules

- Do not edit production code or the OpenSpec change under
  `openspec/changes/build-omniroute-agent-brain/` (OpenSpec is the product contract; TL
  edits GSD governance, proposes OpenSpec deltas to the owner rather than rewriting them).
- No secrets in any report or state file: never include the OmniRoute key, provider creds,
  cookies, prompts, repo content, or tool payloads.
- Do not close panes/workspaces/sessions; do not run `herdr server stop`.
- If the live fleet contradicts the GSD state, surface that contradiction to the owner
  rather than papering over it.

## Reference

- Governance master: `.planning/agent-brain-v3/PROJECT.md`, `MASTER_PLANNING_AND_GOVERNANCE.md`.
- ETA (non-linear): `ROADMAP.md`. Best 5–7 / likely 8–14 / pessimistic 15–25 business days.
- Authorization gate: `OMNIROUTE_ARCHITECT_RESPONSE.md` §7.1.
