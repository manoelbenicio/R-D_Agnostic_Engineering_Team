# Independent QA — push-safety backup/WIP snapshot review

- Reviewer: Codex QA
- Review window: 2026-07-18T20:55:06Z–2026-07-18T21:03:49Z
- Source artifact: `.planning/agent-brain-v3/evidence/push-safety-review-backup-wip-snapshot.md`
- Source artifact SHA-256: `3ab4211fdbc33fb938b9a5df8afc81138ec09d2ce58d76a6c8259e3b24f55b4e`
- Mode: read-only Git/filesystem validation; only this critique and its Golden
  Rule check-in/out were written
- Overall QA disposition: **PARTIAL**. Snapshot recovery claims pass. Historical
  timing/count claims are not fully reproducible, and the proposed G1
  ready-to-commit classification is rejected by current authoritative evidence.
- This is a critique, not snapshot/task/push acceptance. Root retains integration
  authority.

## Claim-by-claim verdicts

| Claim | Verdict | Independent result |
|---|---|---|
| Snapshot ref resolves to `5106de35b2f0ec2e0b44938547ba5011bdc8e5dc` | **PASS** | `show-ref --verify` and `rev-parse --verify` both returned that exact commit. |
| Commit → tree → parent identity | **PASS** | Commit type is `commit`; tree is `5da8e6fe017698a11be6f0cbe8ebf85a3f9899dc`; parent is `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`; both referenced objects exist with expected types. |
| Parent equals current `main` and `HEAD`; parent is an ancestor | **PASS** | `main` and `HEAD` both resolved to `b6571299…`; `merge-base --is-ancestor` exited 0. This is a timestamped current-state fact, not a permanent property. |
| Snapshot is branch-reachable/not dangling | **PASS** | Exact branch ref exists; `git fsck --no-reflogs --unreachable` did not report the commit or tree. |
| Snapshot tree contains 3,795 paths | **PASS** | Independent recursive tree count returned exactly `3795`; NUL-delimited path-list SHA-256 is `72c1c5f9bd49161cb82e444b1e203e9ac0d0d758ddb39d72b054f446c74d3b70`. |
| Zero snapshot paths are physically missing | **PASS** | A NUL-safe loop checked every tree pathname using existence-or-symlink tests: `snapshot_paths=3795 physical_missing=0`. This proves current physical presence, not byte identity for every file. |
| Five spot-check blobs equal current files | **PASS** | All five full object IDs matched exactly; see the hash table below. |
| The 242 `D` entries relative to the snapshot are a comparison artifact | **PASS** | Current `git diff --name-status <snapshot>` has 244 paths: exactly `242 D` and `2 M`, while the physical path loop reports zero missing snapshot paths. The source explanation is correct: the snapshot captured paths absent from the parent/index baseline. |
| No nonignored working file at 20:23Z was omitted | **PARTIAL** | The snapshot is a full subset of the current tracked/untracked-nonignored union (`snapshot_only_union_count=0`), but a later set-membership check cannot reconstruct the complete 20:23Z worktree. A path present then, omitted, and removed before review would be invisible. The source's `comm` method supports current coverage but does not alone prove the historical universal claim. |
| Ignored secret-bearing `.env*` files were omitted intentionally | **PARTIAL** | The two currently visible ignored names `.env.bak` and `.env.bak-agentsetup` match `.gitignore:21` and are absent from the snapshot. No content was read. The historical “~30 ignored entries” count was not independently reconstructible. Also, `.env.example` is tracked and modified, so the blanket later exclusion “`.env*` must never be committed” is overbroad. |
| Exactly 15 files were created after the snapshot | **PARTIAL** | The source records a contemporaneous `3810 − 3795 = 15`, but that historic union was not saved as a hashed manifest. Current measurements drifted from 38 post-snapshot paths at 20:57 to 42 at 21:03:19 and a 3,839-path union (therefore 44 post-snapshot paths, with zero snapshot-only paths) at 21:03:49. The original count is plausible but not independently replayable. |
| `verify.tsx` is the only physically missing path described, absent from snapshot but recoverable from parent | **PASS** | Worktree status is ` D`; disk is missing; snapshot lookup is absent; parent contains blob `6ac2cf01ea2f83ecf25570423a048d2e1be2ed03`. No restore was attempted. |
| Index has exactly 11 staged frontend files | **PASS** | Exact count is 11; name/status manifest SHA-256 is `9512f50480949563eacfe729ac31a79d5889a931d14ecefd3911ce7995f26110`; no unmerged index entries; cached diff whitespace check produced no diagnostic. |
| The staged files form one coherent feature and none are rejected | **PARTIAL** | The paths collectively implement runtime/model picker behavior, but `packages/core/api/client.ts` is a shared native-onboarding surface and the set spans core API, runtime catalog, agent types, and views. Coherence does not establish acceptance or push eligibility. |
| G1 is “ready to commit as-is” | **REJECT** | `integration-push-scope-hygiene-audit.md:33-47` classifies all 11 as staged violations from pending/rejected lanes. `AGENT_LEDGER.md:287` explicitly rejects the ready-to-commit statement and freezes the set pending task identification and independent acceptance. Native tasks 1.5/1.6 remain open and EV-VIS is pending final review. |
| Current scope is 92 modified, 5 added, 71 untracked, 1 deleted | **PARTIAL** | It may describe the source review instant, but it is mutable and lacks a saved manifest hash. At 21:03:49Z the independently captured porcelain manifest was `M=93 A=5 D=1 untracked=285`, SHA-256 `0d319be35931403abcbc3961eaeb83caec2f81cf88c1b7e9607eee216a2ece24`. |
| Remote push is blocked by absent credentials | **PARTIAL** | Not retested: the assignment prohibited credential inspection and network access. This review makes no remote-auth or push-readiness claim. |

## Immutable object and physical-presence evidence

### Object provenance

```text
git rev-parse --verify refs/heads/backup/wip-snapshot-20260718T202300Z
5106de35b2f0ec2e0b44938547ba5011bdc8e5dc

git show-ref --verify refs/heads/backup/wip-snapshot-20260718T202300Z
5106de35b2f0ec2e0b44938547ba5011bdc8e5dc refs/heads/backup/wip-snapshot-20260718T202300Z

git cat-file -p 5106de35b2f0ec2e0b44938547ba5011bdc8e5dc | sed -n '1,6p'
tree 5da8e6fe017698a11be6f0cbe8ebf85a3f9899dc
parent b6571299b00c8e388abefe7ef9dcbcf8ac715d7f
author mbenicios <mbenicios@users.noreply.github.com> 1784406346 -0300
committer mbenicios <mbenicios@users.noreply.github.com> 1784406346 -0300
```

`git cat-file -t` returned `commit`, `tree`, and `commit` for the snapshot,
tree, and parent respectively. `git rev-parse main HEAD` returned the parent
twice. `git merge-base --is-ancestor <parent> <snapshot>` exited 0.

### Whole-tree path verification

The count used:

```text
git ls-tree -r --name-only 5106de35b2f0ec2e0b44938547ba5011bdc8e5dc | wc -l
3795
```

The physical check consumed `git ls-tree -r -z --name-only` in a Bash loop,
incremented `total`, and counted a path missing only when both `test -e` and
`test -L` failed. Result:

```text
snapshot_paths=3795 physical_missing=0
```

This avoided checkout, blob extraction, and credential/environment content
reads. The tree's NUL-delimited pathname stream was independently hashed:

```text
72c1c5f9bd49161cb82e444b1e203e9ac0d0d758ddb39d72b054f446c74d3b70
```

### Spot-check blob equality

| Path | Snapshot blob | Current file blob | Verdict |
|---|---|---|---|
| `.planning/agent-brain-v3/EVIDENCE_CONTRACT.md` | `fe94eca1c1a45e6e126ed103c517bf1922b629a1` | same | PASS |
| `.deploy-control/Kiro__PRODEX-RUNTIME-1.1-1.3__20260718T200600Z.md` | `8cc5e09d0a47633ed9dc2e05e8af4465c540a853` | same | PASS |
| `.planning/agent-brain-v3/evidence/credential-isolation-session-api-audit.md` | `f80d053aff05999e75cd5fcd855ec19dc97e3c61` | same | PASS |
| `.claude/skills/herdr-fleet/SKILL.md` | `4498a3af0df7b1b7ab9cf4c289e5b7d09a05d698` | same | PASS |
| `multica-auth-work/server/internal/daemon/prodex_runtime_integration_test.go` | `9e74dd8aef7ac2eb57a34be08de750415af55e8f` | same | PASS |

These five samples support the source's bounded byte-identity statement. They do
not convert the full 3,795-path presence check into a full byte comparison.

## Worktree drift and historical-count limitation

At 20:57, before this critique existed, read-only recomputation returned:

```text
current_union_count=3833
post_snapshot_count=38
snapshot_only_union_count=0
```

The QA check-in itself is one of those 38. A single in-memory snapshot at
21:03:19Z then found 42 post-snapshot paths with sorted-list SHA-256
`9089346584b4d58b27b98070690c967bd459db1976babea129fda7f5178afeac`.
Thirty seconds later, ongoing agents had increased the current union to 3,839
paths, sorted-list SHA-256
`43018f32fb3caf18c26b806a7c2a90d97718dd36aa2c1834bfd56a9f99e03e50`.
Because the snapshot-only count remained zero, that implies 44 post-snapshot
union paths at that instant.

Observed additions during the QA window included new Prodex review/design
records, credential-isolation review records, a native-onboarding diagnostic
check-in, and this QA check-in. This is normal concurrent-agent drift. It makes
the source's `3810` and `15` historical values unsuitable as current push-scope
counts; it does not weaken snapshot recoverability.

## Staged 11-file classification

Exact current staged manifest:

```text
M multica-auth-work/packages/core/api/client.ts
A multica-auth-work/packages/core/runtimes/models.test.tsx
M multica-auth-work/packages/core/runtimes/models.ts
A multica-auth-work/packages/core/types/agent.test.ts
M multica-auth-work/packages/core/types/agent.ts
A multica-auth-work/packages/views/agents/components/inspector/model-picker.test.tsx
M multica-auth-work/packages/views/agents/components/inspector/model-picker.tsx
A multica-auth-work/packages/views/agents/components/model-dropdown.test.tsx
M multica-auth-work/packages/views/agents/components/model-dropdown.tsx
A multica-auth-work/packages/views/agents/components/runtime-picker.test.tsx
M multica-auth-work/packages/views/agents/components/runtime-picker.tsx
```

The index has five additions and six modifications, no conflicts. The staged
manifest is mechanically coherent, but acceptance evidence is not. The
pre-existing hygiene audit already marked these paths as staged violations;
the current ledger's root adjudication says “staged ≠ accepted” and orders
FREEZE/EXCLUDE without unstaging while agents are active. Therefore the source's
inventory is correct and its ready-to-commit conclusion is not.

## Proposed atomic groups and exclusions

| Proposal | Verdict | Current evidence-based critique |
|---|---|---|
| **G1 — staged web agent model/runtime picker** | **REJECT** | Exact 11-file membership passes, but push readiness does not. EV-VIS remains `PRODUCED/PENDING FINAL REVIEW`; native 1.5/1.6 remain open; root ledger line 287 explicitly excludes this set. |
| **G2 — persist-prodex-runtime-integration** | **PARTIAL** | Current status has the three modified files and four new Go files named by the proposal plus the five-file OpenSpec change. Tasks 1.1-1.3 now have independent offline acceptance, but tasks 2.1-2.2 remain MISSING. The broad OpenSpec directory therefore mixes accepted and missing work; `config.go` is also a shared central hotspot, and the proposal omits the evidence artifacts it says should accompany code/spec. Requires owner-produced exact manifest after active lanes freeze. |
| **G3 — agent-credential-isolation** | **REJECT** | The pattern currently spans modified execenv/wakeup files, nine new session/rotation files, and the OpenSpec task file. Architecture 1.1-1.3 still requires product-owner decisions; task 4.3 remains unchecked; current evidence-integrity critique identifies stale hash/transcript limitations. This is multiple lanes and evidence grades, not one atomic commit. |
| **G4 — mobile auth** | **PARTIAL** | Current mobile set is bounded to six modified/deleted paths plus three new deterministic tests. It appears mechanically coherent, including the `verify.tsx` deletion, but no independent acceptance artifact was found. The ledger classifies it as follow-on `MOBILE-AUTH-MIGRATE`, while native 1.5 remains open. Plausible future group; not push-ready. |
| **G5 — remaining server modules** | **REJECT** | The proposal itself says to split by owning change, so it is not an atomic group. Current named server areas contain at least 48 modified and 84 untracked paths across brain, gateway, auth, handler, middleware, observability, deployment, and tests. Each accepted lane needs an exact owner/evidence manifest. |
| **G6 — planning and coordination docs** | **REJECT** | Current counts are `.planning=93`, `.deploy-control=33`, `.claude=4`, `.codex=4`. These include accepted, pending, review, active check-ins, and tool-skill files; `.claude/**` and `.codex/**` are not merely coordination prose. A single housekeeping commit would erase provenance/ownership boundaries. Root must select a narrow documentary manifest or leave active coordination state uncommitted. |
| `nul`, `multica-auth-work/NUL`, `multica-auth-work/server/NUL` | **PASS** exclusion | All three exist as untracked mode-0777 files and remain Windows path hazards. No deletion was performed. |
| `files.txt`, `opencode.json.backup.20260718-110331` | **PASS** exclusion | Both exist untracked. Treat as quarantined scratch/backup paths. Contents were not read. |
| `opencode.json` review before any inclusion | **PASS** quarantine | Exists untracked. It remains unreviewed here because content inspection could cross the local-secret/path boundary. Exclude unless separately authorized and cleared. |
| `multica-auth-work/.env*` must never be committed | **REJECT** as written | `.env.bak` and `.env.bak-agentsetup` are correctly ignored and absent from the snapshot; keep them excluded. But tracked `multica-auth-work/.env.example` is currently modified, so a glob-wide prohibition incorrectly includes a versioned template. The safe rule is: exclude actual/local environment files and backups; review the tracked example as a nonsecret template in its owning atomic group. No environment contents were inspected. |

## Evidence provenance

| Evidence source | SHA-256 | Use |
|---|---|---|
| Source push-safety review | `3ab4211fdbc33fb938b9a5df8afc81138ec09d2ce58d76a6c8259e3b24f55b4e` | Claims under review |
| Integration push-scope matrix | `d37efb3624b01235135e4a535472eb45fa6d0239f8d5cb2dd8532b0e53bb4fce` | Existing acceptance/exclusion grades |
| Integration hygiene audit | `ad44e4f37840c190c290e187d607044b5af73a7a40104f97541b52bc1e5b2d5d` | Staged violations and hazardous paths |
| Vendor/model visibility evidence | `67cdb00d8a930a9c6ca02991e8b431b22f00a5fdde2dad74712b23af175b9106` | EV-VIS produced/pending state |
| Agent ledger | `b06262cdb86c4cb563b9cba4692b3c8c1ce69a4363ab7f0baadde790eed6f1b6` at 21:02:50Z | Root G1 adjudication and lane states; see drift note below |
| Persist Prodex 1.1-1.3 review | `ed27595e600a33594b1003cddb2c14b4f60594065ca40e7356c97c0d87fe825d` | Accepted bounded PP source behavior |
| Persist Prodex 2.1-2.3 readiness audit | `b8d2847443c1a090c979bb0567a122896a8c14e94419875937b3507112bf5b1f` | 2.1/2.2 MISSING classification |
| Credential isolation 1.1-1.3 decision | `e9949a7fc8cfb02228256fdb709631acda01a4a374bab8c9f046d793f53dbc1a` | Product-owner gate |
| Credential 4.3/5.3 integrity critique | `b91877f956326f5f09fdca5202ea6cd6a82f8818a703abf2503b1dee3f52cb98` | Evidence drift/provenance limitation |
| Native onboarding tasks | `78f78b383f26dbd6128a43b4fcfbaf1375911f1671b069a0baef462f0b9e7d3c` | 1.5/1.6 checkbox state |

These are current-disk hashes captured during the review. Concurrent agents can
change untracked planning artifacts after the timestamp; they are provenance
pins for this critique, not immutable Git-commit provenance.

The ledger changed concurrently after the provenance table was captured: its
SHA-256 became `f5c66efa607dc840d90d493cc8b1fa2fd81e22ac240eded831ace772ddf95df7`
at 21:05:41Z. A post-drift `rg` check confirmed that the decisive push-scope
adjudication remains at line 287 with the same `staged ≠ accepted` and
`FREEZE/EXCLUDE` ruling. This critique preserves both observed hashes rather than
silently replacing the earlier provenance point.

## Exact command classes and tools

Read-only commands used:

```text
git rev-parse --verify <ref>; git show-ref --verify <ref>
git cat-file -t <object>; git cat-file -p <commit>
git merge-base --is-ancestor <parent> <snapshot>
git fsck --no-reflogs --unreachable
git ls-tree -r [-z] --name-only <snapshot>
git hash-object -- <noncredential sample path>
git ls-files -co --exclude-standard; git ls-files -u
git diff --cached --name-status; git diff --cached --check
git diff --name-status <snapshot>; git diff --check
git status --porcelain=v1
git check-ignore -v -- <environment backup path names>
test -e; test -L; stat; wc; sort; comm; rg; sha256sum; date
```

Tool identities: Git 2.43.0; GNU coreutils `stat`/`sha256sum` 9.4.

Both authorized documentation files passed isolated `diff --check`. The final
repository-wide `git diff --check` exited 2 only for pre-existing whitespace in
`multica-auth-work/server/internal/handler/chat_test.go:474,477,512`, the same
excluded chat lane identified by the hygiene audit; this QA did not edit it.

No checkout, reset, add, restore, commit, push, deletion, ref/index mutation,
network request, credential helper, credential content, environment content,
database, or live service/provider operation was used.

## Final boundary

The backup branch is presently recoverable and no captured path is physically
lost. That conclusion does not authorize a push. Root should preserve the
snapshot, freeze/exclude the staged G1 set per the existing adjudication, and
replace broad G2-G6 proposals with exact owner/evidence manifests after ongoing
agents finish. No acceptance, integration, index action, or git-state mutation
is performed or implied by this QA.
