# Blocker escalation policy — 2026-07-28

Owner ruling: delivery speed is critical; cost is not a blocker. A blocker is valid only when recorded with all fields below:

1. **Why:** exact failing condition, command, code path, permission, external answer or collision.
2. **Who:** single person/agent/owner capable of resolving it.
3. **Where:** exact card, worktree, file, service or external system.
4. **When:** ETA or explicit external condition; `not stated` is escalated within 15 minutes.
5. **Unblock action:** smallest executable next action, not a generic request for more review.

Rules:

- “Waiting review”, “waiting Gate 0”, “governance” or “cost” alone are not blockers.
- A self-created sequencing contradiction must be corrected by the General Tech Manager, not propagated to workers.
- Work that is independent of the blocker proceeds in parallel.
- A reviewer must return PASS/BLOCK with minimal correction, not reopen unrelated scope.
- After a correction, re-review is immediate and stays with an independent reviewer.
- Kanban status must be reconciled with evidence at each milestone.
- Secret exposure, conflicting ownership, destructive data action without backup/rollback, and external permission denial remain legitimate stop conditions.
