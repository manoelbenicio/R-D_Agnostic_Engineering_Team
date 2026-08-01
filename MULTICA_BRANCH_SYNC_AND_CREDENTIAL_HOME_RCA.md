# Multica Branch Synchronization and Credential-Home Incident RCA

**Document status:** Active corrective-action record; release remains HOLD and unpushed  
**Prepared:** 2026-08-01T04:14:26Z  
**Reconciled:** 2026-08-01 after bounded implementation and gate review  
**Coordinator host:** `21LAPGLMVPJ4` (`/home/dataops-lab`)  
**Affected/inspected hosts:** ORQ2 and ORQ1  
**Classification:** Internal engineering; credential-safe metadata only  
**Deployment status:** No production deployment performed  
**Remote publication status:** No Git branch or commit pushed

## 1. Executive summary

Two related workstreams were investigated: (1) synchronization and safe integration of the Multica SPE branches, and (2) uncontrolled growth of isolated credential-home folders on ORQ2.

The original merge-based integration candidate, N0, was proven unsafe because it would have introduced 133 true donor deletions, including rotation, NIM, billing/labs UI, smoke scripts, and prodex content. It was rejected and preserved only as forensic evidence. A replacement additive base, N0v2, was built from the accepted parent and contains exactly 2,710 added paths with zero modifications, deletions, or renames. It preserves immutable migration 128 and all protected capabilities.

The accepted additive integration base, N0v2 (`9c0ad3428dd77638edde4020ff2df6c5177475d8`), exists only on the coordination host and is absent from the ORQ2 local object graph. The detached ORQ2 root is a partial staging tree, not the accepted integration and not a release target. Integration therefore proceeds only by reviewed selective import into `/tmp/multica-n1-c1` on `21LAPGLMVPJ4`; no whole-branch merge, overwrite of the ORQ2 root, or inference from ORQ2 root builds is permitted.

C1+C2 schema and credential-registry reconciliation advanced beyond the initial failure recorded in Sections 5 and 6. Migration 128 remains byte-exact, pinned sqlc v1.31.1 generation and the generated package build passed, and credential-registry tests passed in the accepted tree. Bounded companion repairs subsequently cleared the original ExactEnv/delivery-observer handler blocker, and targeted `pkg/agent`, `pkg/redact`, realtime, handler, and full build gates reported pass. Lane J then restored the accepted-tree Agent Brain configuration/capacity/built-in resolution, daemon constructor/runtime/observability wiring, admission-before-prepare controls, profile suppression, and credentialless Prepare/Reuse across exactly `internal/daemon/config.go`, `internal/daemon/daemon.go`, and `internal/daemon/execenv/execenv.go`; full daemon, full execenv, and build gates passed. Lane I then completed six accepted-tree files across `cmd/multica`, `internal/cli`, `internal/auth`, and `pkg/redact`; all imported declarations and full-file authorities were independently matched to two byte-identical non-N0 sources, and focused/full affected-package gates plus `go build ./...` passed. Lane K then completed exactly five files: ingress/queue observability in `internal/middleware/request_logger.go` and `internal/service/task.go`, workspace-scoped atomic failed-issue reconciliation in `pkg/db/queries/issue.sql` with pinned sqlc v1.31.1 output limited to `pkg/db/generated/issue.sql.go`, and the bounded `CaptureTaskUsage` signature correction in `internal/handler/daemon.go`. Migration 128 remained exact; middleware/service race and full tests, handler, cmd/server, compile sweep, and build passed. Lane L then completed bounded password provisioning and recent-authentication work across `internal/middleware/auth.go`, `internal/handler/auth_provider.go`, `internal/handler/auth.go`, and `internal/handler/handler.go`, while `internal/auth/recent_auth.go` remained exact at SHA-256 `e800814c59e5ea55295d6b8c2209bf57fe776595e5d1e541ff511f4f892db94b`. All normalized non-N0 hashes, the generated `db.User` 14-field shape, and local credential-registry wiring were preserved; passwordtest, auth, middleware, whole compile sweep, and build gates passed. A later clean disposable-PostgreSQL run closed the database evidence gap: migrations applied through 135; protected rows survived reversible 134/133/132 down/up; migration 131 refused down with exact SQLSTATE `55000` and policy message; `TestRuntimeManagerReservationPrimitives` passed in 0.165s; and `SPE6_RUNTIME_MANAGER_DATABASE_URL=<clean disposable> go test -count=1 ./...`, `go vet ./...`, and `go test -race -count=1 ./...` passed all packages. Non-race examples include `pkg/db/generated` 0.160s and daemon 57.673s; race examples include daemon 60.624s, passwordtest 4.322s, and `pkg/db/generated` 1.135s. The disposable database was automatically cleaned up. All I/J/K/L companion blockers and all local test/database blockers are now closed. Shared source now also resolves the migration-135 composition blocker, and direct pinned sqlc, deterministic regeneration, compile, and build gates pass. The accepted composed RC tree is complete on the coordinator host. Independent transferred-byte verification passed: archive SHA-256 `07ff862254a097cc2529216141b733ccedc1566779ec1acdc6266e06fba080b6` at 44,343,032 bytes; sidecar 3189/3189; superseding manifest 59/59; final repairs inventory 21/21 with zero path overlap; pinned migration/generated hashes; migration 128 up/down blobs; sqlc v1.31.1 deterministic generation; consolidated Go, PostgreSQL, Runtime Manager, C4, and SPE-7 gates. The ORQ2 workspace root is not the accepted final tree and must not be used to classify missing or differing generated files or migrations as RC defects. The validated two-file allocator supplement extends the governed synchronization set to 82 canonical repository-relative paths and passed 2/2 hash verification.

Shared C2/C3/SPE-10 work defines UUID `home_ref`, separate canonical `name_ref`, durable immutable lifecycle generations, compare-and-swap publication, restart fencing, immediate tombstone persistence, idempotent/concurrent behavior, strict runtime-configuration contracts, and fail-closed composition boundaries. The ORQ2 shared source, non-database gates, corrected runner, clean disposable-PostgreSQL evidence, canonical migration-135 self-contained DDL, and deterministic direct sqlc generation passed. The accepted composed RC tree is complete on the coordinator host. Independent transferred-byte verification passed: archive SHA-256 `07ff862254a097cc2529216141b733ccedc1566779ec1acdc6266e06fba080b6` at 44,343,032 bytes; sidecar 3189/3189; superseding manifest 59/59; final repairs inventory 21/21 with zero path overlap; pinned migration/generated hashes; migration 128 up/down blobs; sqlc v1.31.1 deterministic generation; consolidated Go, PostgreSQL, Runtime Manager, C4, and SPE-7 gates. The ORQ2 workspace root is not the accepted final tree and must not be used to classify missing or differing generated files or migrations as RC defects. The validated two-file allocator supplement extends the governed synchronization set to 82 canonical repository-relative paths and passed 2/2 hash verification.

Separately, ORQ2 had 30 credential-slot folders and a registry containing 90 historical slot/terminal records. An independent Linux `/proc` reference audit found exactly one live slot, `slot-185`. With explicit approval, the other 29 direct `slot-*` directories were deleted under the registry lock and with filesystem-boundary protection. Post-cleanup verification found exactly `slot-185` before and after deletion. The slot root then allocated 945,508,352 bytes, and the filesystem reported 29,301,944,320 bytes available at 61% utilization. This is retained as a historical cleanup-time observation, not the current live state.

**Current ORQ2 credential-home state (owner-confirmed correction):** Eight homes were expected because eight agents were intentionally logged in. The read-only three-layer reconciliation proved one-to-one physical-slot / legacy-registry-terminal / live-pane mapping at that audit: `slot-184=w7:p3`, `slot-185=w5:p9`, `slot-186=w6:p1`, `slot-187=w7:p5`, `slot-188=w6:p2`, `slot-192=wP:p2`, `slot-193=wP:p1`, and `slot-194=w5:pC`. There were no duplicate pane-to-slot mappings and no unreferenced physical homes. Count alone is not credential-home proliferation, no cleanup is needed or authorized, and this expected eight-home state is not a restart-readiness blocker.

The installed registry remains legacy v1. Validated source v2 adds stable agent-plus-subscription fingerprint enforcement, fail-closed identity validation, live-reference protection, and one physical home per stable binding, but it is a later rollout item. No installed allocator, registry, credential home, or credential content was modified. Deployment and restart require fresh explicit owner approval; the current eight intentionally bound homes do not by themselves demonstrate recurrence or a defect.

**Current release conclusion:** The accepted composed RC tree is complete on the coordinator host. Independent transferred-byte verification passed: archive SHA-256 `07ff862254a097cc2529216141b733ccedc1566779ec1acdc6266e06fba080b6` at 44,343,032 bytes; sidecar 3189/3189; superseding manifest 59/59; final repairs inventory 21/21 with zero path overlap; pinned migration/generated hashes; migration 128 up/down blobs; sqlc v1.31.1 deterministic generation; consolidated Go, PostgreSQL, Runtime Manager, C4, and SPE-7 gates. The ORQ2 workspace root is not the accepted final tree and must not be used to classify missing or differing generated files or migrations as RC defects. The validated two-file allocator supplement extends the governed synchronization set to 82 canonical repository-relative paths and passed 2/2 hash verification.

The accepted local RC bytes, technical gates, and Item #3 local Git-governance work are complete under w5:p9’s exclusive authority. Item #4 remains HOLD for external actions: push, deployment, and restart are eligible only after asking the owner and receiving fresh explicit approval immediately before each action; no such authorization is currently granted. The owner authorized only persistent backup, exact documentation correction, exact 82-file synchronization, and non-Git validation in the current bounded execution; Git, staging, commit, push, deployment, restart, and credential mutation remain prohibited.

**Reading rule:** Sections 4–9 preserve investigation-time chronology and may describe failures or pending work that was later resolved. The reconciled current gate state is authoritative in this executive summary and Sections 10–16.

## 2. Scope and governing constraints

### 2.1 Integration constraints

- The exact accepted additive integration base is `9c0ad3428dd77638edde4020ff2df6c5177475d8`.
- C1 source tip is `940f74692135731b83296afec6041bb6014017ee`; donor base is `2d91148a58fe1a7cb43fee2e54df1860535bd03d`.
- The old C1 branch must not be merged or cherry-picked wholesale.
- C1 is a selective final-tree import: 20 true additions and 21 OpenSpec modifications.
- Existing `resolver.go` and `resolver_test.go` must not be blindly overwritten.
- Generated Go must be produced by pinned sqlc v1.31.1 and never hand-merged.
- Migration 128 is immutable. Expected blobs are:
  - up: `4b0894920069336efc36db645b05fae775a3b130`
  - down: `a2b2ea2b56d10f57fe0144bc6998a88277039643`
- Migration `131_runtime_sessions.down.sql` is also immutable policy evidence. Its nonzero SQLSTATE `55000` refusal with exact message `migration 131 is non-destructive and cannot be rolled down` is required behavior, not a defect. No gate may weaken or cross this irreversible boundary.
- Work remains local until exact gates pass and explicit push authorization is given.
- No deployment is permitted during this integration phase.

### 2.2 Credential-home constraints

- Folder count must equal the exact number of active agent/subscription bindings.
- Historical copies, buffers, and age-based retention are prohibited.
- Herdr must not be an operational dependency for retention or identity.
- Current emergency active-set authority is Linux `/proc`: filtered slot-home environment values, cwd/root links, and open file descriptors.
- Future identity must be explicit and stable across restarts: agent identity plus provider-subscription fingerprint.
- Missing identity must fail closed; no generated UUID fallback may allocate a new slot.
- Credential values, tokens, cookies, or secret contents must never be printed or transmitted.

## 3. Systems and artifacts involved

### 3.1 Coordination host

- Host: `21LAPGLMVPJ4`
- Working directory: `/home/dataops-lab`
- Accepted additive worktree: `/tmp/multica-n0v2-additive`
- C1 integration worktree: `/tmp/multica-n1-c1`
- Unsafe original disposable clone: `/tmp/multica-branch-sync-a3t3Um/repo`
- Rejected forensic N0 worktree: `/tmp/multica-n0-integration`

### 3.2 ORQ2

- Dirty C2 worktree: `/home/ec2-user/workspace/worktrees/spe6-runtime-schema`
- C3 worktree awaiting audit: `/home/ec2-user/workspace/worktrees/spe6-runtime-manager-api`
- C4 worktree awaiting audit: `/home/ec2-user/workspace/worktrees/spe6-runtime-manager-ui`
- Installed allocator inspected read-only: `/home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh`
- Credential slot root: `/home/ec2-user/.agent-cred-homes/slots`
- Registry lock: `/home/ec2-user/.agent-cred-homes/registry.lock`

### 3.3 ORQ1

- No `/home/ec2-user/.agent-cred-homes` root exists.
- No alternate credential-slot root was found under `/home`, `/var/lib`, `/opt`, or `/srv`.
- No process references to credential-slot paths were found.
- No cleanup was required.

## 4. Incident A: unsafe branch integration base

### 4.1 Detection

Ten synchronized branches and their exact SHAs were inspected. `origin` and `orq2-source` were identical for all ten branches, and their exact octopus common ancestor was established as `2d91148a58fe1a7cb43fee2e54df1860535bd03d`.

The first merge-based base, N0, was then compared against the donor trees. That analysis found 133 true donor deletions. The affected material included rotation, NIM, billing/labs UI, smoke scripts, and prodex. These were not harmless merge noise; accepting them would have removed required existing capabilities.

### 4.2 Root cause

The integration strategy treated divergent branch trees as safe merge inputs even though donor branches contained deletion histories not intended for the restart baseline. A conventional merge inherited those deletions. The process lacked an additive-only path invariant before N0 was constructed.

### 4.3 Impact and risk

If merged or pushed, N0 could have silently removed operational and product functionality. The broad deletion surface created a high regression and restart risk.

### 4.4 Correction

- N0 commit `8002d39c9284999585cf0317b68e916828bea1c7` was rejected.
- It is retained unchanged on local branch `integration/spe-shared-base-local` at `/tmp/multica-n0-integration` for forensic comparison only.
- It must never be amended or pushed.
- A new additive base, N0v2, was constructed from parent `8b75cd012b513c0cec283166c21992e07a9b670c`.
- N0v2 commit is `9c0ad3428dd77638edde4020ff2df6c5177475d8` on local branch `integration/spe-shared-base-additive-local`.

### 4.5 Verification evidence

N0v2 has:

- one parent and no merge commit;
- exactly 2,710 added paths;
- zero modified, deleted, or renamed paths;
- zero paths under `server/migrations`;
- zero paths under `openspec/changes/archive`;
- zero intersection between imported additions and the origin/main tree;
- exact migration 128 blobs;
- preserved rotation, NIM, billing/labs, smoke, and prodex content;
- exclusions for `sqlc.staged-orq12.yaml`, all eight rename destinations, orphan `SUPERSEDED.md`, and two prohibited `.html` files.

## 5. Incident B: C1 credential-registry contract mismatch

### 5.1 Detection

C1 was selectively imported into `/tmp/multica-n1-c1` from accepted N0v2. The staged delta is 43 paths: 21 additions and 22 modifications. sqlc v1.31.1 generated `runtime_manager.sql.go` and updated `models.go` without unexpected generated files.

Validation results:

- `go build ./pkg/db/generated`: passed.
- `go build ./internal/credentialregistry`: failed.

The compile failure identifies missing C1 contract symbols used by `postgres_store.go`: `Store`, `Request`, `CandidateValidator`, `Candidate`, and `ReleaseRequest`. It also shows that `NewPostgresStore(pool)` cannot satisfy the legacy `QueryRower` constructor contract.

### 5.2 Resolver anomaly

Two C1 paths represented as additions already exist in N0v2/main:

- `multica-auth-work/server/internal/credentialregistry/resolver.go`
- `multica-auth-work/server/internal/credentialregistry/resolver_test.go`

The trees are semantically incompatible:

- accepted main: `NewResolver(db QueryRower) *Resolver`, with `Resolve(ctx, agentID, provider)`;
- C1: `NewResolver(store Store) *Resolver`, with fenced `Resolve(ctx, Request)` and `Release(ctx, ReleaseRequest)`.

Current accepted callers include handler and daemon code plus credential-registry database tests. Blind replacement would break those callers and alter assignment semantics from host paths/account metadata to opaque `HomeRef` admission snapshots.

### 5.3 Root cause

C1 combined a new atomic reservation store with a replacement resolver API, but the selective integration contract correctly prohibited overwriting shared resolver behavior before ownership and caller migration were established. The branch presentation also classified already-existing resolver paths as additions, obscuring the collision. There was no compatibility boundary between legacy read-only assignment resolution and the new fenced task-admission resolver.

### 5.4 Safe correction direction

The minimum-risk design is additive:

1. Preserve the accepted legacy `Resolver`, `NewResolver(QueryRower)`, and current handler behavior.
2. Extract C1's fenced contract types and validation into a separately named resolver, such as `FencedResolver`/`NewFencedResolver`.
3. Make `PostgresStore` implement only the new `Store` contract.
4. Expose a separately named PostgreSQL fenced composition point.
5. Wire shared handlers only after C3/API ownership and task lifecycle semantics are audited.
6. Build both credential-registry and handler packages and run resolver/store tests before committing.

This direction has been analyzed but not yet implemented or committed.

## 6. Incident C: C2 ownership ambiguity and schema recovery

### 6.1 Audit performed

ORQ2 C2 worktree was inspected read-only:

- path: `/home/ec2-user/workspace/worktrees/spe6-runtime-schema`
- branch: `feature/spe6-runtime-schema`
- HEAD: `72c07a97c09ba860f1e12f8848fa471c83c4b4f2`

Dirty paths include migration 132, runtime-manager query/fixture/gate test and generated outputs, plus untracked migration 135 and a gate script. Migration 128 remains exact.

### 6.2 Findings

C2 adds:

- `credential_home_catalog.lifecycle_generation`;
- `credential_home_catalog_lifecycle`;
- lifecycle states and non-reuse/tombstone enforcement;
- generated-query source operations to load lifecycle state, advance generation, and save lifecycle records;
- migration 135 future-only checksum columns for `schema_migrations`;
- rollback protection once checksum evidence exists.

Historical migration rows, including migration 128, remain NULL/UNKNOWN for checksums. C2 is additive and future-facing in that respect.

Critically, C2 contains no resolver or handler implementation. Therefore the assumption that C2 owned or supplied the missing `NewResolver(Store)` integration was disproven. The missing contract comes from C1's resolver tree.

### 6.3 Remaining correction

Only handwritten C2 changes should be imported into the C1 worktree: migration 132 up/down, runtime-manager SQL query, fixture, tests, migration 135, and gate script. Generated files must be regenerated locally with pinned sqlc v1.31.1. Migration 128 must be hashed before and after. This work is pending.

## 7. Incident D: ORQ2 credential-slot proliferation and disk consumption

### 7.1 Initial condition

ORQ2 initially had:

- 30 direct `slot-*` folders;
- approximately 29.06 GiB allocated by metadata inventory;
- 90 registry slot records;
- 90 registry terminal records.

The requested invariant is one folder per active agent/subscription binding and zero historical retention.

### 7.2 Active-set determination

Herdr-based identity mapping was rejected because Herdr failure must not cause credential-home lifecycle failure. A Linux `/proc` audit was used instead. It inspected all processes for:

- filtered environment references to slot-home paths;
- cwd and root symlink targets;
- open file descriptor targets.

Exactly one referenced slot was found: `slot-185`. Six processes referenced it, involving `kiro-cli`, `kiro-cli-chat`, `bun`, `python`, and `uv`. No credential files or values were read.

### 7.3 Deletion preview and approval

The canonical keep set was exactly `slot-185`; the approved deletion set was the other 29 direct `slot-*` directories. The initial projected remaining slot allocation was approximately 1.755 GiB, with approximately 27.308 GiB recoverable.

### 7.4 Execution and correction

The first Python `shutil.rmtree` attempt stopped safely on a root-owned `Dockerfile` with `PermissionError`; it partially reduced `slot-139` and made no other unsafe change.

Before retrying, the active set was revalidated as exactly `slot-185`. The approved deletion was then performed while holding `registry.lock`, using `sudo rm -rf --one-file-system` against the exact direct `slot-*` paths only. This constrained traversal to approved folders and prevented crossing mounted filesystem boundaries.

### 7.5 Postcondition evidence

- Active set immediately before deletion: `slot-185`.
- Remaining directory set after deletion: `slot-185` only.
- Active set after deletion: `slot-185`.
- Postcondition: success.
- Slot-root allocated bytes: 945,508,352.
- Filesystem total bytes: 75,082,215,424.
- Filesystem used bytes: 45,780,271,104.
- Filesystem available bytes: 29,301,944,320.
- Filesystem utilization: 61%.

The difference between initial metadata estimates and final `du` allocation reflects partial first-pass deletion and measurement timing; the authoritative final state is the post-cleanup directory enumeration, `/proc` revalidation, `du`, and `df` output.

### 7.6 Root cause

The installed allocator contains two coupled defects:

1. `agent_cred_isolation_terminal_id_locked` attempts `herdr pane get` and uses `herdr:<terminal_id>` as allocation identity. Terminal IDs are ephemeral and are not stable agent/subscription identities.
2. If Herdr lookup fails, the code derives a pane/TTY/process fallback key and creates/persists a random UUID. `agent_cred_isolation_allocate_slot_locked` allocates a new sequential slot for every unseen identity and increments `next_slot` indefinitely.

This design guarantees unbounded growth under terminal recreation, Herdr lookup failure, process changes, or repeated login/restart events. The registry's 90 historical terminal/slot entries are consistent with this behavior.

### 7.7 Contributing factors

- Identity, session, and storage-retention concerns were conflated.
- The allocator had no required stable identity supplied by orchestration.
- Random fallback converted control-plane uncertainty into durable allocation.
- There was no active-binding reconciliation or zero-retention reap path.
- Registry monotonic allocation was not paired with bounded garbage collection.
- Monitoring focused on folder existence rather than cardinality versus active bindings.

### 7.8 Durable correction required

The synchronized source copy—not the installed production script—must be changed and reviewed to:

- require an explicit `AGENT_CRED_ISOLATION_IDENTITY` or equivalent stable identifier;
- derive it from stable agent UUID plus provider-subscription fingerprint;
- reject path separators, control characters, and ambiguous forms;
- fail closed when identity is absent or invalid;
- remove all Herdr, pane, TTY, PID, and random UUID identity fallbacks;
- map one stable identity to exactly one slot across restarts;
- reconcile registry and filesystem to active stable bindings;
- delete only OS-unreferenced superseded paths;
- retain zero historical copies;
- add tests proving repeated login/restart does not increase slot count;
- add tests proving missing identity allocates nothing;
- add tests proving concurrent allocation for one identity converges to one folder.

The installed ORQ2 script has not been hot-edited. This avoids an unreviewed production divergence. Until the source fix is deployed, ORQ2 can regress and create new slots during future terminal/session churn.

## 8. ORQ1 comparison

ORQ1 had no credential-home root, no alternate root in the inspected standard locations, and zero process references to slot paths. No cleanup or mutation was performed. At RCA publication time, ORQ1 also had no detected Kiro/Codex/Gemini/Claude/Antigravity/OpenCode agent process and its local Herdr pane service was unavailable with a missing socket/path error.

This historical comparison documented the earlier cleanup condition only. The later owner-confirmed audit established eight expected one-to-one agent/subscription homes on ORQ2 with no duplicate or unreferenced physical home; count alone is not proliferation. Installed registry v1 remains legacy, and validated source v2 is a separately gated future rollout.

## 9. Communication and agent-state findings

An earlier exact-recipient search did not find a pane named `CODEX-TK-SHAREPOINT`. Similar labels existed but were intentionally not treated as exact matches without authorization.

For this RCA distribution, the owner explicitly authorized sharing with all agents online. A fresh process and pane audit found:

- Historical distribution-time topology: ORQ2 had one live agent pane, `w5:p9`; other named panes were idle. This is chronology, not current topology.
- Owner-confirmed credential reconciliation: eight physical homes mapped one-to-one to eight registry terminals and live panes, with no duplicates or unreferenced homes.
- Later GIT-PREFLIGHT-005 topology: exactly four responsive agents (`w5:p9`, `w5:pC`, `wP:p1`, `wP:p2`); absent pane IDs returned `pane_not_found`, which proves current absence but not historical retirement.
- ORQ1 historical comparison: no matching live agent processes; Herdr pane listing was unavailable.

The RCA contains metadata, code/integration facts, and operational findings only. It excludes credential contents, tokens, cookies, and secret values.

## 10. Validation and control evidence

### 10.1 Completed and passed

- Ten-branch SHA/mirror inventory.
- Exact common-ancestor determination.
- N0 deletion audit and rejection.
- N0v2 additive-only path classification.
- N0v2 protected-content preservation checks.
- Migration 128 blob verification.
- Pinned sqlc v1.31.1 generation for current C1 schema/query set.
- `go build ./pkg/db/generated`.
- Read-only C2 worktree/status/schema audit.
- ORQ2 full-process `/proc` active-slot audit before deletion.
- Registry-locked, filesystem-bounded approved deletion.
- ORQ2 post-delete folder and `/proc` active-set verification.
- ORQ2 post-delete `du`/`df` verification.
- ORQ1 candidate-root and process-reference audit.

### 10.2 Passed after bounded reconciliation

- Migration 128 up/down blobs were reverified unchanged.
- Pinned sqlc v1.31.1 generation, generated package build, and credential-registry tests passed.
- ExactEnv and delivery-observer companion repairs cleared the original handler compile blocker.
- Reported accepted-tree gates include full `pkg/redact`, `pkg/agent`, `internal/realtime`, and `internal/handler` tests plus `go build ./...`.
- The source allocator syntax check, full harness, fail-closed non-root reconcile test, and independent two-file review passed without changing the installed legacy-v1 registry, any credential home, or credential content.
- C3 owned runtime-configuration/composition tests, race tests, handler build, and vet passed; SPE-10 catalog tests and race tests passed.
- C4 final source gates passed: core runtime-manager client 11/11 plus TypeScript `--noEmit`, and views runtime-manager 12/12 plus TypeScript `--noEmit`. All nine imported C4 files were verified byte-identical with `cmp` against the tested frozen source. The shared tree did not have Vitest installed; no dependency installation was attempted.
- SPE-7 passed 28/28 with Git absent from `PATH`. The canonical hot-apply document uses `version: v1`, and the stale absent-adapter ledger now records `implemented_source_symbols` for the six landed handwritten Runtime Manager symbols.
- C3 runtime-configuration normal and race tests passed after all four remaining external validation issue labels changed from `schema_version` to `version`. The only remaining `schema_version` text is in two private digest-preimage JSON keys in `digest.go`, intentionally preserved to avoid changing established digests; no external issue or fixture field uses it.
- Final ORQ2 shared-source gates passed: targeted runtimeconfig/catalog/handler/cmd-server tests, runtimeconfig and catalog race tests, all-package Go compile sweep, vet, and `go build ./...`.
- `openspec validate credential-account-home-restoration --strict` and `openspec validate build-omniroute-agent-brain --strict` passed.
- Accepted-tree Lane J passed daemon compile, full `internal/daemon/execenv` (0.351s), full `internal/daemon` (58.530s), and `go build ./...` after bounded changes to exactly three production files. `agent.Result` has no real `ProcessID`, so CLI span `proc_id` remains absent rather than fabricated; recorder wiring remains active.
- Accepted-tree Lane I completed and refroze exactly six files: `cmd/multica/cmd_id_resolver.go`, `cmd/multica/cmd_user.go`, `cmd/multica/cmd_issue.go`, `internal/cli/client.go`, `internal/auth/jwt.go`, and `pkg/redact/redact.go`. Non-N0 provenance was independently verified at declaration or full-file level against two byte-identical healthy sources. Focused issue/password/expected-status/JWT/redaction tests, full affected-package gates, the whole compile sweep for Lane I, and `go build ./...` passed.
- Accepted-tree Lane K completed and refroze exactly five files: `internal/middleware/request_logger.go`, `internal/service/task.go`, `pkg/db/queries/issue.sql`, `pkg/db/generated/issue.sql.go`, and `internal/handler/daemon.go`. Four dependency hashes matched non-N0 authority; sqlc v1.31.1 was required and generated-file snapshots proved only `issue.sql.go` changed; migration 128 remained exact. Full and race middleware/service gates, handler, cmd/server, whole compile sweep, and `go build ./...` passed.
- Accepted-tree Lane L completed bounded modifications to `internal/middleware/auth.go`, `internal/handler/auth_provider.go`, `internal/handler/auth.go`, and `internal/handler/handler.go`; `internal/auth/recent_auth.go` remained byte-exact at SHA-256 `e800814c59e5ea55295d6b8c2209bf57fe776595e5d1e541ff511f4f892db94b`. All normalized non-N0 hashes, verified JWT `auth_time` propagation, the generated `db.User` 14-field scan shape, and local `CredentialAssignments`/credentialregistry resolver/constructor wiring were preserved. Passwordtest (0.244s), auth (0.997s), and middleware (0.558s) executed and passed; the handler package invocation compiled but its `TestMain` database suite was skipped because PostgreSQL was unavailable. Whole `go test -run '^$' ./...` and `go build ./...` passed.
- Shared-source composition blocker resolution is canonical: `server/migrations/135_migration_checksums.up.sql` now creates `schema_migrations` if absent, before `ALTER TABLE`, with table shape `version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now()` and an explanatory comment. This matches `cmd/migrate` and is a runtime no-op; its ordinary SHA-256 is `f08d4d4a5eb894a340b72e12e830d10ddb890ce9f010677e363aa426c6672b80`. Direct pinned sqlc v1.31.1 passed with migration 135 present and no temporary exclusion. The first generation necessarily exposed `SchemaMigration` in `models.go`; a second direct generation produced zero byte changes. Source-only generated hashes are `runtime_manager.sql.go` `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896` and `models.go` `80b974a3087ea5264a8fd2d936e8bf9a3120f88d1c56b719c348a8aaaf337f57`; compile sweep and `go build ./...` passed. The accepted composed tree must not force the source-only `models.go`: its deterministic sqlc v1.31.1 output is authorized at `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb` because accepted N0v2 migration 129 and matching `task_usage.sql` add and preserve `TaskUsage.price_version` and `computed_cost_usd`.
- The superseding 59-line selective-transfer manifest at `/tmp/multica-selective-transfer-full-files.sha256` verifies 59/59 files with zero missing/failures and is authoritative at SHA-256 `52b5c6aaa8320613134638335ce8152951f9d1b715a4cfc364fa4f65bce9b53a`. It is newline-terminated and uses `<64 lowercase hex><two spaces><source-relative path><LF>` from source root `/home/ec2-user/workspace/worktrees/spe6-runtime-schema/multica-auth-work`. Corrected source hashes remain: `server/pkg/db/queries/runtime_manager.sql` `737f2dfad69332e90009c3ffa36455ec50f0df002667df86d8e7d3ef6c019e8b` and runner `e1719a126c3129db101b62da57b47fe72583f1838f7641992c3601c5adb1129a`. Generated outputs are outside the 59 copied-file set and are regenerated on target with sqlc v1.31.1. The only authorized generated change set is `runtime_manager.sql.go` at `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896` plus composed `models.go` at `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb`, with zero further spill. Acceptance also requires migration 129 and `task_usage.sql` byte preservation, `TaskUsage` retaining both generated fields/types/JSON tags, `SchemaMigration` remaining present, all other generated declarations unchanged, and a second deterministic sqlc run. Manifest `be1b1bcd31fcc834ff8e9677ec45c9eec1ca05575d3e25c0f37c73f4e9019017`, earlier manifest `22c0a149edbcc1eedccbe8f30e10198fee29444c1dad06517537246514d8a712`, and runner `6d7f91905a9560c769ad2c6b493df3e12a697f72dfaaa8fa091ced26a9a7f7bf` remain superseded provenance and must not be transferred.
- Clean disposable-PostgreSQL evidence passed: migration through 135; protected-row preservation across reversible 134/133/132 down/up; exact migration-131 SQLSTATE `55000` refusal; `TestRuntimeManagerReservationPrimitives` 0.165s; full non-race `go test -count=1 ./...`; `go vet ./...`; and full `go test -race -count=1 ./...`. The database URL existed only in child test environments and the disposable database was automatically cleaned up. No local test or database blocker remains.

### 10.3 Current governance and release state

- The accepted composed RC tree is complete on the coordinator host. Independent transferred-byte verification passed: archive SHA-256 `07ff862254a097cc2529216141b733ccedc1566779ec1acdc6266e06fba080b6` at 44,343,032 bytes; sidecar 3189/3189; superseding manifest 59/59; final repairs inventory 21/21 with zero path overlap; pinned migration/generated hashes; migration 128 up/down blobs; sqlc v1.31.1 deterministic generation; consolidated Go, PostgreSQL, Runtime Manager, C4, and SPE-7 gates. The ORQ2 workspace root is not the accepted final tree and must not be used to classify missing or differing generated files or migrations as RC defects. The validated two-file allocator supplement extends the governed synchronization set to 82 canonical repository-relative paths and passed 2/2 hash verification.
- The accepted local RC bytes, technical gates, and Item #3 local Git-governance work are complete under w5:p9’s exclusive authority. Item #4 remains HOLD for external actions: push, deployment, and restart are eligible only after asking the owner and receiving fresh explicit approval immediately before each action; no such authorization is currently granted. The owner authorized only persistent backup, exact documentation correction, exact 82-file synchronization, and non-Git validation in the current bounded execution; Git, staging, commit, push, deployment, restart, and credential mutation remain prohibited.
- The installed registry remains legacy v1; validated source v2 is a separately authorized future rollout. The current eight intentionally bound homes require no cleanup.
- No local commit, push, deployment, or restart has occurred in this Item #4 execution.
## 11. Risk assessment

### Critical/high risks

1. **Silent functional deletion if rejected N0 is reused.** Mitigation: retain it as forensic-only evidence; synchronize only the exact canonical 82-path set from verified authorities.
2. **Legacy-v1 identity enforcement pending future rollout.** The installed registry cannot encode the validated v2 subscription fingerprint. Monitor physical-home cardinality against active agent/subscription bindings; do not classify expected one-to-one homes as proliferation, and do not mutate homes without fresh zero-reference proof and explicit approval.
3. **Synchronization scope drift.** Mitigation: exact 82-path path accounting, persistent pre-sync rollback evidence, atomic per-file replacement, and 82/82 post-copy hashes with no deletions.
4. **Premature external action.** Local technical and governance evidence does not authorize push, deployment, or restart; each requires fresh explicit owner approval immediately before execution.
### Medium risks

1. **Registry/filesystem reconciliation.** At the owner-confirmed audit, eight physical homes mapped one-to-one to eight registry terminals and live panes, with no duplicates or unreferenced homes. Installed registry v1 remains legacy; any future v2 rollout must preserve live bindings and require fresh evidence before cleanup.
2. **Lifecycle-policy mismatch.** C2 includes tombstone/non-reuse lifecycle concepts while the owner requires zero historical folder retention. Metadata tombstones may be retained only if they do not retain credential folders and are required for non-reuse/security; this contract needs explicit review.
3. **Migration checksum rollout.** Migration 135 is future-only. Tooling must tolerate historical NULL checksums and block rollback only when checksum evidence exists.
4. **Communication ambiguity.** Similar SharePoint pane labels are not interchangeable with the requested exact recipient.

## 12. Corrective and preventive action register

| ID | Action | Status | Acceptance evidence |
|---|---|---|---|
| CAPA-01 | Permanently reject merge-based N0 from release flow | Complete locally | N0 retained forensic-only at `8002d39...`; N0v2 accepted |
| CAPA-02 | Use exact additive N0v2 base | Complete locally | `9c0ad342...`, 2,710 A / 0 M/D/R |
| CAPA-03 | Selectively integrate C1 without resolver overwrite | Complete in accepted RC bytes | Accepted tree and exact selective authorities preserved; no whole-branch merge |
| CAPA-04 | Import C2 schema/query work and regenerate sqlc | Complete in accepted RC bytes | Migration 128 exact; pinned sqlc v1.31.1 deterministic generation and generated hashes passed |
| CAPA-05 | Reconcile credential-registry contracts | Implemented in accepted tree | Credential-registry tests and handler build passed |
| CAPA-06 | Integrate C3/C4 and remaining synchronized work | Complete in accepted RC bytes | Consolidated Go, PostgreSQL, Runtime Manager, C4, and SPE-7 gates passed |
| CAPA-07 | Remove unreferenced ORQ2 folders | Historical cleanup complete; no current cleanup needed | Historical cleanup retained `slot-185`; later owner-confirmed reconciliation found eight intentionally bound, live-referenced homes with no duplicates or unreferenced physical homes |
| CAPA-08 | Implement stable agent/subscription slot identity in source | Complete in source; not deployed | Restart/relogin/concurrency/stale-reconcile harness and independent review passed |
| CAPA-09 | Remove Herdr/random fallback allocation | Complete in source; not deployed | Missing/invalid identity fails before allocation; installed script unchanged |
| CAPA-10 | Reconcile registry entries safely | Validated in source v2; future rollout separately gated | Installed registry remains legacy v1; validated v2 uses stable agent/subscription binding and live-reference protection; current eight homes require no cleanup |
| CAPA-11 | Add cardinality/disk monitoring | Pending | Alert when folders differ from active bindings or grow unexpectedly |
| CAPA-12 | Close systemic companions and consolidated gates | Complete in accepted RC bytes | Authoritative archive 3189/3189, manifest 59/59, repairs 21/21 with zero overlap, supplement 2/2, pinned hashes, sqlc v1.31.1, Go/PostgreSQL/Runtime Manager/C4/SPE-7 gates PASS |
| CAPA-13 | Complete approved integration commit | Pending | Local SHA plus complete evidence; still no push |
| CAPA-14 | Obtain explicit push/deploy authorization | Pending | Written authorization after restart-readiness certification |

## 13. Restart-readiness decision

**Decision: NOT READY for push, deployment, or restart.**

The accepted local RC bytes, technical gates, and Item #3 local Git-governance work are complete under w5:p9’s exclusive authority. Item #4 remains HOLD for external actions: push, deployment, and restart are eligible only after asking the owner and receiving fresh explicit approval immediately before each action; no such authorization is currently granted. The owner authorized only persistent backup, exact documentation correction, exact 82-file synchronization, and non-Git validation in the current bounded execution; Git, staging, commit, push, deployment, restart, and credential mutation remain prohibited.
## 14. Immediate operating guidance

- Do not use, amend, or push rejected N0.
- Do not push or deploy any current integration worktree.
- Do not replace accepted `resolver.go` wholesale with C1's resolver.
- Do not hand-edit generated sqlc Go.
- Do not hot-edit the installed ORQ2 allocation script without reviewed source and deployment approval.
- Monitor physical-home cardinality against active agent/subscription bindings. Do not classify expected one-to-one homes as proliferation, and do not mutate homes without fresh zero-reference proof and explicit approval.
- Any future cleanup requires fresh zero-reference proof, registry-lock and filesystem-boundary protection, exact-path approval, and before/after verification; current homes require no cleanup.
- Never expose credential contents while investigating folder identity or liveness.

## 15. Evidence locations and immutable identifiers

- Accepted N0v2 (coordination host only; absent from ORQ2 local object graph): `9c0ad3428dd77638edde4020ff2df6c5177475d8`
- N0v2 parent: `8b75cd012b513c0cec283166c21992e07a9b670c`
- Rejected N0: `8002d39c9284999585cf0317b68e916828bea1c7`
- Common donor ancestor: `2d91148a58fe1a7cb43fee2e54df1860535bd03d`
- C1 source tip: `940f74692135731b83296afec6041bb6014017ee`
- C2 worktree HEAD: `72c07a97c09ba860f1e12f8848fa471c83c4b4f2`
- Migration 128 up blob: `4b0894920069336efc36db645b05fae775a3b130`
- Migration 128 down blob: `a2b2ea2b56d10f57fe0144bc6998a88277039643`
- Pinned generator: sqlc v1.31.1
- Accepted base worktree: `/tmp/multica-n0v2-additive`
- C1 staging worktree: `/tmp/multica-n1-c1`
- C2 source worktree: `/home/ec2-user/workspace/worktrees/spe6-runtime-schema` on ORQ2

## 16. Final statement

The investigation rejected the unsafe deletion-bearing integration, preserved the additive accepted tree and immutable migration 128, and completed the accepted RC technical baseline. Historical cleanup retained the then-only live `slot-185`; a later owner-confirmed audit established eight expected one-to-one agent/subscription homes with no duplicates or unreferenced physical homes, so the earlier critical-proliferation interpretation is withdrawn. Authoritative transferred bytes passed archive 3189/3189, manifest 59/59, repairs 21/21 with zero overlap, supplement 2/2, pinned hash, sqlc v1.31.1, Go, PostgreSQL, Runtime Manager, C4, and SPE-7 verification. The accepted local RC bytes, technical gates, and Item #3 local Git-governance work are complete under w5:p9’s exclusive authority. Item #4 remains HOLD for external actions: push, deployment, and restart are eligible only after asking the owner and receiving fresh explicit approval immediately before each action; no such authorization is currently granted. The owner authorized only persistent backup, exact documentation correction, exact 82-file synchronization, and non-Git validation in the current bounded execution; Git, staging, commit, push, deployment, restart, and credential mutation remain prohibited.