# ORQ-26 — official ephemeral-Postgres CI execution

- Agent/pane: `Codex56#B` / `w7:p4`
- Date: `2026-07-27` UTC
- Scope: owner-approved B1–B4 for ORQ-26; this record covers B1 preparation and the single authorized push attempt.
- Final verdict: **BLOCKED_AUTH — no remote object was transmitted, no workflow run exists, and B4 was not executed.**

## Authorization and stop condition

The owner authorized exactly one push of branch `ci/orq26-db-gate`. The accepted checkpoint required an immediate stop if the workflow failed to trigger: no second push, amend, rerun, PR, merge, or cleanup without a new General-TL ruling.

## Frozen inputs

| Item | Verified value |
|---|---|
| Base commit | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` |
| Source worktree | `/home/ec2-user/workspace/worktrees/gtl-orq26` |
| Source branch | `agent/kiro-opus5/orq-26-contract-fix` |
| Source `file.go` SHA-256 | `48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161` |
| Source `file_test.go` SHA-256 | `815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d` |
| Frozen two-file diff SHA-256 | `6ce6a0a4f403063eaf4506b2eefac44b956e842313c9b1a3e4544cc936ae4bc0` |
| Temporary worktree | `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate` |
| Temporary branch | `ci/orq26-db-gate` |

The source worktree was inspected without switching or modifying it. The frozen two-file diff was applied mechanically to the temporary worktree, and the resulting file hashes matched the source hashes above.

## B1 content and pre-push gates

The temporary commit contains exactly these three paths:

1. `.github/workflows/orq26-db-gate.yml`
2. `multica-auth-work/server/internal/handler/file.go`
3. `multica-auth-work/server/internal/handler/file_test.go`

Verified immutable identifiers:

| Item | Value |
|---|---|
| Commit | `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee` |
| Parent | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` |
| Author | `Codex <codex@openai.com>` |
| Subject | `ci(orq26): add temporary ephemeral-Postgres gate for upload contract tests` |
| Workflow SHA-256 | `3494ae8a17d5f35dbbfcd12b1f729ae0089eb3c0882245e329b3ec40d00000f7` |
| Staged three-file diff SHA-256 | `845dd1bb658dfb478539731eaf97722db221d74b135f7e96f221fcbf6322b03f` |

Pre-push verification passed:

- working tree clean after the commit;
- exactly three committed paths and no unrelated file;
- frozen `file.go` and `file_test.go` hashes unchanged;
- `git diff --check` clean and `gofmt -l` empty for the Go files;
- workflow YAML parsed successfully with the repository's existing YAML module;
- every workflow `run:` block passed `bash -n`;
- push-only trigger is limited to branch `ci/orq26-db-gate` and the three paths;
- permissions are `contents: read`; concurrency does not cancel an in-flight run;
- checkout/setup-go actions and the pgvector service are immutable-pinned;
- `persist-credentials: false`;
- no `secrets.*`, artifact-upload action, private-key marker, token assignment, credential value, or secret value was found;
- the workflow includes the eight exact ORQ-26 leaf tests, zero-skip/false-green guards, full handler suite, vet/build/migration gates, and secret-safe summaries.

## Single authorized push attempt

Command executed once:

```text
git push -u origin ci/orq26-db-gate
```

Result:

```text
exit 128
fatal: could not read Username for 'https://github.com': No such device or address
```

The failure occurred locally during HTTPS authentication. Read-only, secret-safe diagnostics established:

- GitHub CLI is installed but has no authenticated account;
- Git has no configured credential helper;
- no GitHub token environment variable was present (environment names only were inspected);
- `git ls-remote --exit-code --heads origin ci/orq26-db-gate` returned exit `2` after the attempt, proving the remote temporary branch is absent.

No credential value was requested, read, printed, persisted, or placed in an argument. No alternative transport or authentication method was attempted.

## Workflow/run evidence

Because the branch was never created remotely:

- remote commit SHA: **not created**;
- workflow run ID: **not created**;
- workflow URL: **not created**;
- job conclusion: **not available**;
- eight leaf-test conclusions/skip count: **not available**;
- immutable Actions logs: **not created**.

This is not a CI test failure and is not a CI PASS. It is a pre-trigger authentication block.

## Stop and preservation

The accepted stop condition was applied immediately:

- no second push;
- no amend, rerun, PR, merge, or workflow dispatch;
- no B4 remote-branch deletion, temporary-worktree removal, or local-branch deletion, because B4 requires terminal run evidence;
- temporary branch/worktree preserved at commit `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee`;
- durable source worktree `/home/ec2-user/workspace/worktrees/gtl-orq26` preserved with its original two modified files and `.gtl-orq26-gate/`;
- ORQ-18 source worktree preserved with its existing four-file safe-boundary diff.

## Required ruling

The next action requires both:

1. a human-provisioned, secret-safe GitHub authentication path that does not disclose credentials to agent context; and
2. explicit General-TL/owner authorization for a new push attempt.

Until both exist, ORQ-26 official CI execution remains **BLOCKED_AUTH**.
