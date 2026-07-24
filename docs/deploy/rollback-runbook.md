# Main Brain / OmniRoute rollback runbook

## Goal

Restore the last accepted Main Brain + OmniRoute release/config while preserving Postgres product state, Kanban/workspace/task/session state, terminal results, redacted evidence and the single-router invariant.

## Rules

- Never roll back to direct-provider/native execution or provider-native credentials.
- Never start an alternate router.
- If neither current nor previous OmniRoute is ready, keep model admission closed.
- Do not delete audit/terminal evidence or run destructive database operations.

## Steps

1. Declare the incident and record trigger, operator, owner, affected cohort and immutable revisions.
2. Close new affected admissions; cancel or drain active tasks according to commit/tool safety.
3. Select the previous accepted OmniRoute image/config/state generation.
4. Wait for strict readiness for the selected protocol/model.
5. Select the previous accepted Main Brain release/config and restart only the required component.
6. Verify health reports OmniRoute as the sole router owner, no direct provider path, reconciled capacity counters and intact terminal persistence.
7. Reopen only the approved bounded cohort; otherwise remain fail-closed.
8. Store redacted timing, status, correlation and persistence evidence; do not store secrets or task content.

## Success criteria

- exactly one router owner: `omniroute`;
- strict readiness passes before admission;
- Postgres and product data remain healthy;
- cancellation/task/terminal counters reconcile;
- no provider credential, direct endpoint or alternate runtime is present;
- owner receives a scrubbed completion record.
