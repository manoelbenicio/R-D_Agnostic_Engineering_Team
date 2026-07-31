# SPE-6 eight-runtime fabric - initial integration control

- Authority: `K1/TL-ORCHESTRATOR`, sole integration/shared-file writer.
- First write UTC: `2026-07-31T23:32:11Z`.
- Worktree: `/home/ec2-user/workspace/worktrees/spe6-eight-runtime-fabric`.
- Branch: `plan/spe6-eight-runtime-fabric`.
- Frozen source commit: `a57c12e8424cd88cda178499563b69569d244d84`.
- Scope: dependency-safe, non-production first wave for SPE-18/SPE-6/SPE-7/SPE-8/SPE-10. No push, production deployment, runner, SharePoint, live inference, secret resolution, or infrastructure mutation is authorized.

## Exact eight-seat reconciliation

The fabric has exactly eight persistent logical seats: three Kiro and five Codex. These are identity-preserving bindings to existing product agent/runtime rows in the target new workspace; they are not permission to insert, clone, rename, archive, reassign, or delete rows. Before any later attachment, the implementation must resolve the exact existing `workspace_id`, `agent_id`, `runtime_id`, and daemon by read-only lookup and fail closed on absence, ambiguity, workspace mismatch, duplicate name, stale generation, or protected assignment. No UUID may be invented.

| Seat | CLI composition | Responsibility | First-wave branch/worktree | Initial immutable SHA |
|---|---|---|---|---|
| K1 | Kiro | TL orchestration, integration and all shared-file decisions; no producer code in this wave | `plan/spe6-eight-runtime-fabric` / `spe6-eight-runtime-fabric` | `a57c12e8424cd88cda178499563b69569d244d84` |
| K2 | Kiro | Runtime Manager UI/UX consumer | `feature/spe6-runtime-manager-ui` / `spe6-runtime-manager-ui` | `a57c12e8424cd88cda178499563b69569d244d84` |
| K3 | Kiro | independent baseline/risk and golden/forbidden verifier; non-author acceptance | `audit/spe18-baseline-risk` / `spe18-baseline-risk`, then `test/spe7-golden-forbidden` / `spe7-golden-forbidden` | `a57c12e8424cd88cda178499563b69569d244d84` |
| C1 | Codex | R3 registry core | `feature/spe6-r3-registry-core` / `spe6-r3-registry-core` | `a57c12e8424cd88cda178499563b69569d244d84` |
| C2 | Codex | runtime schema and migrations 130-134, subject to registrar recheck | `feature/spe6-runtime-schema` / `spe6-runtime-schema` | `a57c12e8424cd88cda178499563b69569d244d84` |
| C3 | Codex | Runtime Manager API against frozen pathless contracts | `feature/spe6-runtime-manager-api` / `spe6-runtime-manager-api` | `a57c12e8424cd88cda178499563b69569d244d84` |
| C4 | Codex | registry authorization, workspace, secrecy and boundary enforcement | `feature/spe8-registry-boundaries` / `spe8-registry-boundaries` | `a57c12e8424cd88cda178499563b69569d244d84` |
| C5 | Codex | dynamic opaque credential-home catalog | `feature/spe10-credential-catalog` / `spe10-credential-catalog` | `a57c12e8424cd88cda178499563b69569d244d84` |

K3 owns two serial verifier worktrees but remains one logical seat. K1 is the ninth Git worktree only because integration is isolated from producers; K1 is one of the eight product seats and does not create a ninth agent/runtime row. At wave start, all eight producer worktrees listed above existed, were clean (`status=0`), and pointed exactly to the frozen source SHA.

## Existing-row and protected-assignment reconciliation

The contract reuses the existing daemon `orq2-credential-runtime-v1`, target workspace row, agent rows, and runtime rows. It creates no workspace, project, squad, agent, runtime, daemon, container, runtime installation, credential home, provider account, subscription, or infrastructure.

The current ORQ2-dev workspace is protected and is not the target pool for the new-workspace composition:

- protected workspace: `orq2-dev`, `workspace_id=20fce817-895d-447b-965a-49f5e279314a`;
- protected online Codex runtime: `0f7133db-ba65-4373-9c6c-884cc4731700`;
- protected online Kiro runtime: `6d0d721a-5ffa-4955-94c0-cddcc1bb3475`;
- protected online Antigravity runtime: `405b751d-e831-4da3-8fd5-bb3744c49334`;
- protected historical offline runtime rows: Claude `588ebcba-f36c-45d9-8b3b-0fe17f5b83aa` with three active agent references (`8101fcf3`, `b2f5448`, `2954cddf`) and Kiro `eb41a0c9-0005-4138-8d41-47975a642230` with five active agent references (`24b10605`, `bd1ebd0a`, `d09b1fc1`, `1e0f4914`, `c37759f7`).

Those ORQ2-dev assignments and historical references are evidence, not migration inventory. They must not be moved, borrowed, shared, deleted, archived, renamed, repurposed, or used as fallback for the new workspace. Any lookup that resolves a fabric seat to one of those protected bindings fails closed and returns to K1 for adjudication.

A subscription/account-home is attached separately to the already-resolved target binding. Attachment changes only the binding's opaque `subscription_ref` or exclusive `home_ref` plus generation; it must leave `workspace_id`, `agent_id`, `runtime_id`, `runtime_session_id`, and `daemon_id` unchanged. One native home is exclusive to one active persistent binding; an OmniRoute binding has no native home. Missing, stale, ambiguous, unhealthy, unsupported, unauthorized, conflicting, cross-workspace, or protected state fails closed. There is no global-HOME, cross-home, cross-workspace, or cross-transport fallback.

## Immutable shared interfaces

All first-wave work starts from commit `a57c12e8424cd88cda178499563b69569d244d84`. The following Git blob SHAs are immutable producer interfaces; a producer must not change them. Any required interface amendment stops the producer, returns to K1, and is committed separately before rebase/restart of affected work.

| Frozen interface | Git blob SHA |
|---|---|
| `openspec/changes/credential-account-home-restoration/design.md` | `3d605f2b278837418db363d60d7ddf83df671ba3` |
| `openspec/changes/credential-account-home-restoration/specs/credential-account-home/spec.md` | `6cb9e596d2845c06415e87d39b05e290b7e0a54d` |
| `openspec/changes/credential-account-home-restoration/tasks.md` | `705a1fcc986437823d140f85c866892e6f5e0e3c` |
| `openspec/changes/build-omniroute-agent-brain/design.md` | `d7d211055cb81bdec03d8cc5da1fb3ebd3c0a9d4` |
| `openspec/changes/build-omniroute-agent-brain/specs/agent-brain-runtime/spec.md` | `f9770e5d085f01553b6a6d11e5ce4f3bdfde2659` |
| `openspec/changes/build-omniroute-agent-brain/specs/brain-cutover-operations/spec.md` | `9a8945aace771691c1e41c8c57de779d2920bfa5` |
| `openspec/changes/build-omniroute-agent-brain/specs/credentialless-agent-execution/spec.md` | `9376d91f1889d44e6060f275bcbf86bd54a001aa` |

Also frozen: migration order `130_runtime_standards` -> `131_runtime_sessions` -> `132_credential_home_catalog` -> `133_runtime_bindings` -> `134_runtime_configuration_snapshots`; pathless REST methods/bodies/status/error envelope; authenticated event envelope/types with at-least-once deduplication and generation-gap recovery; precedence `platform > standard > runtime > explicitly delegable task`; claim-time immutable IDs/versions/generations/digests; and mutually exclusive `omniroute` versus `native_credential_home` transport bindings.

## Dependency and merge order

1. **Gate 0 - K3/SPE-18:** independently inventory baseline risk and confirm no protected-row, secret, infrastructure, producer-file, or shared-interface violation. C2 also performs the mandatory global migration-registrar recheck; any collision stops the wave.
2. **Foundation - C1 and C2:** C1 registry core and C2 schema may proceed in parallel only against the frozen interfaces and disjoint files. Neither edits shared daemon/router/protocol wiring.
3. **Boundaries - C4 and C5:** after reviewed C1+C2 interface-compatible commits, C4 lands authorization/workspace/secrecy boundaries and C5 lands catalog discovery/lifecycle. They may run in parallel on disjoint files.
4. **API - C3:** consume reviewed C1+C2+C4+C5 interfaces; do not invent compatibility shims or alter frozen contracts.
5. **UI - K2:** consume the reviewed C3 API and opaque/pathless types. UI may prepare against the freeze earlier, but its merge waits for C3's immutable reviewed SHA.
6. **Verification - K3/SPE-7:** compose the reviewed producer SHAs and run golden/forbidden, migration, race, privacy, authorization, capability, recovery, drift, rollback, and end-to-end non-production gates. K3 does not review its own SPE-18/SPE-7 authored evidence as producer acceptance; a distinct reviewer is required where independence would otherwise collapse.
7. **Integration - K1:** serially merge only reviewed immutable producer commits in this order: SPE-18 evidence -> C1 -> C2 -> C4 -> C5 -> C3 -> K2 -> SPE-7 evidence. K1 records each accepted commit SHA before merge. Shared router/protocol/daemon wiring is deferred to SPE-9 and is not authorized by this initial file.

A downstream commit is never merged before all declared predecessors have a reviewed immutable SHA. Merge conflict, interface drift, overlapping ownership, dirty producer state, skipped test, or missing evidence is a hard stop, not permission for K1 to rewrite producer code.

## Production and release gates

This first wave is non-production and does not authorize deployment. Production remains NO-GO until all of the following are separately evidenced:

1. both affected OpenSpec changes strict-validate and the cross-authority scan passes;
2. every producer and integration tree is clean for owned files, with no untracked/staged residue;
3. C2 proves migrations 130-134 on an isolated ephemeral database, including up/down safety, active-reference protection, uniqueness/race behavior, and no destructive source-home action;
4. K3 and an independence-preserving peer accept golden/forbidden, privacy/redaction, authorization, capability, recovery, drift, rollback, and end-to-end results with zero skips and no live inference used as a basic health check;
5. the integrated exact commit is published only to an approved non-main branch, local/upstream SHAs are equal, ahead/behind is `0/0`, and deployment evidence pins that exact SHA;
6. Council acceptance and K3 release recommendation are recorded;
7. the owner gives separate written production-rollout authorization and a bounded rollback window.

Until those gates pass, there is no push, production change, daemon/runtime/container restart, runner execution, secret or credential operation, subscription mutation, infrastructure creation, ORQ2-dev reassignment, or SharePoint work.
