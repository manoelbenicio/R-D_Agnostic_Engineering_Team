# Agent Brain v3 — Kiro/Opus-4.8 + Codex#56#A leadership handover

Effective: 2026-07-18 · authorized directly by the product owner.

## Authority and role split

- Kiro/Opus-4.8 (`w3:p3`, `Kiro#Opus48-TL`) owns planning, architecture adjudication,
  prioritization and acceptance decisions.
- Codex#56#A (`w3:p1`) owns live Herdr transport, independent disk/pane verification,
  authoritative state/document updates and execution control.
- Codex1–4 implement through disjoint file ownership. Co-leads do not issue competing prompts.
- Claude/GLM-5.2 `w3:p5` was closed by the owner. Its G0/G1 history is retained but its
  terminal summary is not current authority.

## Authoritative current state

- G0/G1/G2: COMPLETE in their explicitly authorized scopes.
- G2A `w3:pD`: EV-G2A-01..05 DONE.
- G2B `w3:p8`: EV-G2B-01..07 DONE.
- G2C `w3:p9`: EV-G2C-01..10 DONE for G2 scope; 5.6–5.8 remain fail-closed contracts,
  not accepted native routes.
- G2D `w3:pA`: EV-G2D-01..07 DONE as packages/specifications; no live service action.
- G3: READY and authorized for serial Codex1 integration only.

Read in this order: `.planning/agent-brain-v3/STATE.md`, `DECISIONS.md`,
`FILE_OWNERSHIP.md`, `AGENT_LEDGER.md`, `EVIDENCE_INDEX.md`, `phases/G3/PLAN.md`,
`DISPATCH_QUEUE.md`, then OpenSpec `tasks.md`.

## Hard gates

- PD-01: preserve the dirty baseline; no reset/stash/revert/discard.
- PD-08: no credential/auth read, copy, rewrite, rotation, quarantine or mutation.
- No Prodex removal, default cutover, production action or tier 50/100.
- The system is non-production. Production canary/soak is removed; development integration,
  security, failure, rollback, isolation and bounded-capacity acceptance remain mandatory.
- Keep MUL-2..MUL-25 parked. Reconcile/supersede MUL-11/12/15 because OmniRoute exclusively
  owns provider credentials/accounts/rotation.
- Do not dispatch Codex through the current Multica daemon until G3 credentialless wiring and
  isolation smoke pass. Use isolated Herdr panes.

## Immediate mission

Dispatch the single current G3 entry in `.planning/agent-brain-v3/DISPATCH_QUEUE.md` to
Codex1 `w3:pD`, then monitor state transitions without repeatedly ingesting full transcripts.
No G4 fan-out until G3 evidence is independently validated.

Prepared next-phase packet: `.planning/agent-brain-v3/G4_ACCELERATED_PACKET.md`.
