# ORQ-26 local ephemeral-DB gate — immutable `20cab478` — BLOCK

- **Recorded (UTC):** 2026-07-28T13:57:50Z
- **Card:** ORQ-26 (`e966922d-c6a5-4812-87bb-8b9576ccbc60`)
- **Verdict:** **BLOCK**
- **Meaning:** this local execution is the practical equivalent of `.github/workflows/orq26-db-gate.yml` and supersedes the GitHub Actions billing path as execution evidence. It does **not** produce a false PASS: the targeted ORQ-26 gate passed, but the workflow-required full handler package failed.
- **Mutations excluded:** no product/source edit, checkout/reset/clean of the dirty main worktree, commit, push, merge, issue status/assignee change, product DB access, or product-container mutation.

## 1. Immutable inputs

| Input | Verified value |
|---|---|
| Branch | `ci/orq26-db-gate` |
| Locked HEAD | `20cab478a04f4119be7f1ecd109a328cabf01f6b` |
| Locked base | `0cb8aebb5aff79cb430b3740d22fadc53c0116fd` (ancestor: yes) |
| Range cardinality | exactly 3 files |
| Range files | `.github/workflows/orq26-db-gate.yml`; `multica-auth-work/server/internal/handler/file.go`; `multica-auth-work/server/internal/handler/file_test.go` |
| `file.go` SHA-256 | `70f45ebdcd07e24d892657a9add7a624b2c10bb2c657d9af1118af453c0fc50e` |
| `file_test.go` SHA-256 | `b1c3c515e31d97cfa4ed26a07ee7ae2c1c824d07e61fc05929c7a0ed92fd8f1a` |
| Go | `go1.26.1 linux/amd64` |
| DB image | `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` |
| Local image ID | `sha256:ca46538882b0ab2dcbd6fd51fd918602200e48df7fae5cb079efaadba739fcd6` |
| Pull policy | `--pull=never`; pinned image was already present; no pull occurred |
| Gate environment | `CI=true`, `ORQ26_EPHEMERAL_DB_GATE=1` |

Preflight proved the exact branch HEAD, base ancestry, three-file union range, both locked hashes and `git diff --check`. Final fingerprints in both disposable checkouts again proved the same HEAD, clean diff and locked hashes.

## 2. Private disposable topology

- ORQ1 private workspace: `/home/ec2-user/.cache/orq26-gate.uX3r3P`, mode `0700`; its `tmp`, `gocache` and `gotmp` were also `0700`.
- Source arrived through a private full-history Git bundle; checkout was detached at the exact HEAD and the bundle was deleted.
- Ephemeral container: `orq26-gate-20260728t134309z`, `--rm`, exact digest, `PGDATA` on tmpfs, no persistent volume.
- DB binding: ORQ1 `127.0.0.1:32768` only.
- Private SSH tunnel: ORQ2 `127.0.0.1:59843` → ORQ1 `127.0.0.1:32768`, with control/executor root `/tmp/o26.01Wyvs` mode `0700`.
- Disposable DB identity through the tunnel: `orq26_gate|orq26_gate`.
- No package installation and no network module resolution were used.

ORQ1 had Go 1.26.1 but `CGO_ENABLED=0` and no `gcc`, `cc` or `clang`; the first `-race` command therefore stopped before running any test with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. Every already-local ORQ1 image was checked; none contained both Go and a C compiler. The no-install fallback copied ORQ1's existing Go 1.26.1 toolchain and module cache into the private disposable ORQ2 executor, used existing GCC 11.5, set `GOPROXY=off GOSUMDB=off`, and reached the same ORQ1 disposable DB only through the private tunnel.

## 3. Pre-test gates — PASS

```text
locked_hashes=PASS
range_files=3 exact_set=PASS
gofmt=clean
git_diff_check=clean
db_identity=orq26_gate
go_vet=PASS
go_build=PASS
migrations_expected=163
migrations_applied=163
```

## 4. Targeted ORQ-26 race/JSON gate — PASS

Command contract: all exact named leaves under `go test -race -count=1 -json`, requiring a JSON `run` and `pass` event for every leaf, no `skip`, package-level PASS, no false-green marker and no `WARNING: DATA RACE`.

```text
ORQ26_TARGETED_PASS
leaf_tests=11
package_pass=true
false_green=false
race_warning=false
elapsed_seconds=1.908
JSON_SHA256=3876fb486d70e57bff9c76469fe0ba63370de22a1edd38332405ed0ef55f7924
```

The 11 passed leaves were:

1. `TestUploadFile_ContextlessWithoutEntityRefsReturnsEmptyIDAndStorageLinks`
2. `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/issue_id`
3. `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/comment_id`
4. `TestUploadFile_ContextlessWithEntityRefsRejectedPreUpload/chat_session_id`
5. `TestUploadFile_InsertFailureCleansUpAndReturns500`
6. `TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop`
7. `TestUploadFile_SuccessShapesAlwaysCarryContractFields/workspace_shape`
8. `TestUploadFile_SuccessShapesAlwaysCarryContractFields/contextless_shape`
9. `TestCleanupOrphanObject_DeletesExactKeyNotDerivedFromURL`
10. `TestCleanupOrphanObject_RunsWithCanceledRequestContext`
11. `TestUploadFile_InsertFailureDeletesOriginalKeyWhenKeyFromURLCollapses`

## 5. Full handler package race/JSON gate — BLOCK

The workflow-required command `go test -race -count=1 -json ./internal/handler` returned non-zero.

```text
package_pass=false
unique_test_pass_events=1347
unique_failed_tests=3
unique_skipped_tests=37
required_upload_regressions_passed=5/5
false_green=false
race_warning=false
package_elapsed_seconds=12.511
JSON_SHA256=7fbdfb2dc996a4ed91e68db2a82e117ba1d1931acc5fc402c74a14c000ee2d06
```

All five required upload regressions passed:

- `TestUploadFileForeignWorkspace`
- `TestUploadFileResolvesWorkspaceViaSlugHeader`
- `TestUploadFileResolvesWorkspaceViaIDHeaderStill`
- `TestUploadFile_AttachesToChatSession`
- `TestUploadFile_RejectsForeignChatSession`

Exactly three tests failed:

```text
TestCreateChatSession_Routing
  chat_test.go:461: CreateChatSession direct explicit: expected 201, got 400:
  {"error":"invalid workspace id"}

TestCreateWorkspaceUsesRequestedSlug
  handler_test.go:2354: CreateWorkspace: expected 201, got 500:
  {"error":"failed to create default squad: ERROR: null value in column \"leader_id\" of relation \"squad\" violates not-null constraint (SQLSTATE 23502)"}

TestCreateWorkspace_DoesNotMarkOnboarded
  workspace_test.go:73: CreateWorkspace: expected 201, got 500:
  {"error":"failed to create default squad: ERROR: null value in column \"leader_id\" of relation \"squad\" violates not-null constraint (SQLSTATE 23502)"}
```

The later commit `11ef7156ad77e838132d4d8e3ca86ffa1a70ba1a` is titled `fix(workspace): stop creating a default squad with no leader`, but it is not an ancestor of locked HEAD `20cab478`; applying it would violate this gate's immutable input. No source repair or rerun was attempted.

## 6. Teardown and product non-impact — PASS

All temporary resources were removed even though the gate verdict is BLOCK:

- SSH ControlMaster/socket gone; local port `59843` bindable.
- ORQ1 gate container absent from `docker ps -a`; remote port `32768` bindable.
- Container `.Mounts` was empty before removal, proving no attached Docker volume; no named `orq26-gate` volume or network exists.
- Remote workspace `/home/ec2-user/.cache/orq26-gate.uX3r3P` absent.
- Local `/tmp/o26.01Wyvs` and `/tmp/kiro-orq26-gate.current` absent. Go's read-only module cache required `chmod -R u+w` only on that disposable private tree before deletion.

The four product containers retained images/ports and start times predating the gate cutoff `2026-07-28T13:43:09Z`:

| Product container | Image | Port | Started UTC | Final check |
|---|---|---|---|---|
| `multica-dev-transition-backend-1` | `multica-backend:agy-status-20260727T102815Z` | `127.0.0.1:18080→8080` | `2026-07-27T15:56:31.478491626Z` | HTTP 200 |
| `multica-dev-transition-frontend-1` | `multica-web:transition-6a2aba3` | `127.0.0.1:13100→3000` | `2026-07-27T03:11:24.477348893Z` | HTTP 200 |
| `multica-dev-transition-postgres-1` | `pgvector/pgvector:pg17` | `127.0.0.1:15433→5432` | `2026-07-21T17:10:49.731230109Z` | healthy |
| `omniroute` | `diegosouzapw/omniroute:latest` | `100.118.244.61:20128→20128` | `2026-07-24T18:36:12.135713768Z` | healthy |

A separate concurrent container `orq12-review-1785246393-221832` appeared after the original baseline on `127.0.0.1:32769`; its name, port and lifecycle are unrelated to ORQ-26, and it was deliberately not touched.

## 7. Kanban receipt

- **Idempotency marker:** `ORQ26-LOCAL-GATE-V1:20cab478`
- **Note ID:** `f5226819-e168-4999-98c0-203d21e9edb0`
- **Verified receipt:** action `created`; marker count `1`; trigger preview `agents=[]`; issue task-runs `2→2`; comments `4→5`; issue status/assignee/project/dates fingerprint unchanged.
- **Safety contract:** `/note` was the first token. Authentication used one runtime-resolved full JSON `SecretString` inside an `asm-exec` child; the JWT existed only in private `0600` temporary files. The private ORQ2→ORQ1 API tunnel and all note-client files were removed, and its local port was proven bindable.

## Final disposition

**BLOCK the immutable `20cab478` gate.** The GitHub billing condition is no longer the evidence-path blocker: the exact local ephemeral-DB execution ran. The remaining blocker is the observed full-package test failure at the locked commit. Preserve the successful 11/11 targeted result as useful evidence, but do not promote it to workflow PASS until a new immutable candidate includes the necessary fixes and the complete workflow-equivalent gate passes.
