# ORQ-21 R2 account exclusivity and mixed-version rollout ruling

Date: 2026-07-28. Authority: General Tech Manager under the owner's immediate-execution ruling.

1. A persisted approved credential account/home is exclusive to one non-archived agent assignment at a time. Sharing the same credential home across agents is prohibited because it permits concurrent mutation of the same provider session.
2. ORQ-21 must reject a second agent assignment with `E_ACCOUNT_ALREADY_ASSIGNED` before mutation and the resolver must not treat `leased` as generally shareable.
3. ORQ-12 migration 128 must add the durable database invariant: a unique partial index on `assignments(account_id)` for the rows covered by the active assignment model, with symmetric rollback and an explicit preflight for existing duplicates.
4. Mixed-version rollout must not delegate all enforcement to a server-supplied boolean. The daemon must support an explicit rollout gate. Deployment order is server first, verify metadata responses, then enable the daemon gate; when enabled, covered providers require an approved assignment regardless of an omitted legacy-server boolean.
5. `worktype_scope` is approval metadata only until an authoritative task worktype vocabulary and comparison are implemented. It must not be represented as an enforced scope.
6. Missing/malformed assignment is a recoverable configuration failure. Tests must prove that after correcting the assignment the issue can enqueue a new task; otherwise the cancellation path must be replaced with a recoverable failure transition.
7. The original `182b7b3` remains immutable. A clean branch/commit containing code only is required for integration; evidence/check-outs remain outside the code commit.
