# ORQ-26 final compatibility gate — PASS

- **Recorded UTC:** 2026-07-28T14:33:21Z
- **Verdict:** **PASS**
- **Scope:** disposable detached composition of immutable ORQ-26 HEAD plus the two requested squad/chat fixes, in exact order.
- **Technical disposition:** the full-package blocker observed on bare `20cab478` is removed by the tested composition. This is compatibility evidence, not a push, merge or production change.

## Immutable composition

| Position | Commit | Subject | Exact patch scope |
|---|---|---|---|
| Base | `20cab478a04f4119be7f1ecd109a328cabf01f6b` | `ci(orq26): pin the changed-file range base instead of the default branch` | existing ORQ-26 candidate |
| Fix 1 | `11ef7156ad77e838132d4d8e3ca86ffa1a70ba1a` | `fix(workspace): stop creating a default squad with no leader` | `multica-auth-work/server/internal/handler/workspace.go` only |
| Fix 2 | `67e9a4c7b643c729603bcb6498c57ed0489d20b7` | `test(chat): fix TestCreateChatSession_Routing setup` | `multica-auth-work/server/internal/handler/chat_test.go` only |

Both patches were first applied through a temporary Git index to prove conflict-free order without touching refs. They were then cherry-picked in the same order in an independent object database with detached HEAD, hooks/signing disabled and deterministic committer metadata.

- Tree after fix 1: `1ca626e910fa0625caccd5b8e691c167cd0a9a84`
- Final canonical composite tree: `9cee675d17b918d3a9b1184688fead0d1e52ba14`
- Disposable synthetic HEAD: `4b288e055e1e0bc8efc6c6c7077ec703abed82eb` — **must not be integrated**; it existed only in the deleted clone.
- `workspace.go` SHA-256: `5d57ac41397ad0ec6867484367c007d35fde48bdc27acd5aa023ec46b22c1c7a`
- `chat_test.go` SHA-256: `8cc30a97e4dd1ac96fd8f972d03435b909e9bace2654034f5bf6fa79d297c18e`
- ORQ-26 `file.go` locked SHA-256 remained `70f45ebdcd07e24d892657a9add7a624b2c10bb2c657d9af1118af453c0fc50e`.
- ORQ-26 `file_test.go` locked SHA-256 remained `b1c3c515e31d97cfa4ed26a07ee7ae2c1c824d07e61fc05929c7a0ed92fd8f1a`.

Set equalities were proven independently:

1. `0cb8aebb...20cab478` = exactly the original three ORQ-26 paths.
2. `20cab478...composite` = exactly `workspace.go` and `chat_test.go`.
3. `0cb8aebb...composite` = exactly those five paths combined.

The composite checkout remained clean and at the same HEAD/tree after every gate.

## Disposable environment

- Exact Go: `go1.26.1 linux/amd64`, copied from the existing ORQ1 installation into a private disposable ORQ2 executor.
- Race compiler: existing GCC 11.5 on ORQ2; no package installation.
- Module source: existing ORQ1 module cache copied privately; `GOPROXY=off`, `GOSUMDB=off`, `GOWORK=off`, `GOFLAGS=-mod=readonly`.
- Private roots/caches were mode `0700`.
- PostgreSQL image: `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0`.
- Local image ID: `sha256:ca46538882b0ab2dcbd6fd51fd918602200e48df7fae5cb079efaadba739fcd6`.
- Pull policy: `--pull=never`; no pull occurred.
- DB container: `orq26-compat-20260728t142452z`, `--rm`, PGDATA on tmpfs, `.Mounts` count `0`, no volume/network created.
- ORQ1 binding: loopback-only `127.0.0.1:32770`.
- Private tunnel: ORQ2 `127.0.0.1:55201` → ORQ1 `127.0.0.1:32770`.
- Disposable identity: `orq26_gate|orq26_gate`.

## Static and migration gates — PASS

```text
gofmt(file.go,file_test.go,workspace.go,chat_test.go)=clean
git diff --check=clean
db_identity=orq26_gate
go vet ./...=PASS
go build ./...=PASS
migrations_expected=163
migrations_applied=163
expected_applied_set_equality=PASS
```

## Targeted ORQ-26 race/JSON gate — PASS

The exact workflow regex was run with `go test -race -count=1 -json`. The 11 required leaves each had `run` and `pass`, none had `skip`; package-level PASS existed.

```text
required_leaves=11/11
package_pass=true
fail_events=0
race_warnings=0
false_green_markers=0
elapsed_seconds=3.713
json_sha256=dc8325bcf545db7e030660e072a693a8c293865a40c10622af41bf5f0104c995
```

A harness-only aggregation typo (`jq` without slurp mode) occurred after the test command and after the per-leaf checks. The captured JSON was parsed correctly without rerunning the targeted test command.

## Full handler package race/JSON gate — PASS

Command: `go test -race -count=1 -json ./internal/handler`.

```text
package_pass=true
fail_events=0
race_warnings=0
false_green_markers=0
unique_test_pass_events=1350
unique_skip_events=37
required_upload_regressions=5/5 PASS
former_blocking_tests=3/3 PASS
elapsed_seconds=13.938
json_sha256=3e52a80cfb23656b28ce09d4b7dc8ba36afa2b0431cb1f1c7c4b45ad259ba952
```

Required upload regressions passing:

- `TestUploadFileForeignWorkspace`
- `TestUploadFileResolvesWorkspaceViaSlugHeader`
- `TestUploadFileResolvesWorkspaceViaIDHeaderStill`
- `TestUploadFile_AttachesToChatSession`
- `TestUploadFile_RejectsForeignChatSession`

The three failures from the bare-`20cab478` run now pass:

- `TestCreateChatSession_Routing`
- `TestCreateWorkspaceUsesRequestedSlug`
- `TestCreateWorkspace_DoesNotMarkOnboarded`

## Teardown and non-impact — PASS

Completed at `2026-07-28T14:32:27Z`:

- gate container absent;
- ORQ1 remote directory absent;
- ORQ1 and ORQ2 tunnel ports bindable;
- private clone, copied toolchain, module cache, Go caches, control socket/root and state file absent;
- no `orq26-compat` volume or network;
- exact four-product-container before/after fingerprint unchanged: `29f3f6c01360dd3e96d7af226398c6e788f46bbb92cc8192d11de638dde41d79`;
- backend HTTP `200`, frontend HTTP `200`, product PostgreSQL `healthy`, OmniRoute `healthy`;
- `refs/heads/ci/orq26-db-gate` remained exactly `20cab478a04f4119be7f1ecd109a328cabf01f6b`.

Global repository refs/status fingerprints changed during the run because other repository activity was concurrent. No reset, clean or attempt to overwrite that activity was made. This gate used an independent clone and made no branch/ref mutation. The differing global fingerprints are recorded as concurrency, not falsely claimed as gate output.

## Technical blocker disposition

**REMOVED for this exact composition.** Bare `20cab478` remains historically BLOCK by its own full-package result; the tested compatibility candidate `20cab478 + 11ef715 + 67e9a4c` is green across the complete local gate. No product or integration action occurred.

## Integration sequence — future authorized action only

1. Create a fresh isolated integration worktree/branch at exact `20cab478a04f4119be7f1ecd109a328cabf01f6b`; do not alter the existing `ci/orq26-db-gate` ref in place.
2. Apply the original commits, in order:
   ```text
   git cherry-pick 11ef7156ad77e838132d4d8e3ca86ffa1a70ba1a
   git cherry-pick 67e9a4c7b643c729603bcb6498c57ed0489d20b7
   ```
3. Require tree `1ca626e910fa0625caccd5b8e691c167cd0a9a84` after the first pick and final tree `9cee675d17b918d3a9b1184688fead0d1e52ba14` after the second when starting exactly at `20cab478`. Never cherry-pick the disposable synthetic commit `4b288e...`.
4. Reconfirm the exact two-file compatibility delta and unchanged ORQ-26 locked hashes, then rerun the complete gate if the target base/tree differs or any commit is regenerated.
5. Preserve the workflow's existing three-file changed-range guard as the historical ORQ-26 gate contract. Adding the two fixes to that branch makes the full range five files; do not silently weaken or broaden the workflow guard. Use this explicit compatibility evidence or a separately reviewed compatibility-gate change.
6. Only after owner/integrator authorization, perform the intended push/review/merge through the normal integration lane. No such action was taken here.
