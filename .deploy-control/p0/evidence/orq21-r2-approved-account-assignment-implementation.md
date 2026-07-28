# ORQ-21 R2 — approved-account assignment implementation

- Owner: `Codex56#B`
- Branch/worktree: `agent/codex56-b/orq21-r2` /
  `/home/ec2-user/workspace/worktrees/gtl-orq21-r2`
- Integration base: `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`
- Prepared: `2026-07-28T14:02:42Z`
- Status: **IMPLEMENTED / LOCAL COMMIT PENDING IN THIS EVIDENCE SNAPSHOT**
- Runtime boundary: no production database, container, credential, AWS resource, remote, or board
  mutation was performed.

## Contract implemented

The production task-claim path now resolves one persisted, tenant-approved account assignment for
the credential-bearing providers `kiro`, `codex`, `antigravity`, and the established `agy` alias.
Resolution joins only `assignments`, `agent`, `accounts`, and `approved_accounts`. It does not join
or read `credentials`, `secret_ref`, token, cookie, password, or credential payload columns.

The handler places only two routing fields in the daemon task response:

- an approved account home-directory path; and
- a boolean that makes the daemon-side check mandatory.

Missing approval, tenant mismatch, provider mismatch, unavailable account, absent worktype scope, or
invalid metadata fails closed. A task already claimed by the handler is cancelled before a generic
409/503 response; the daemon independently rejects required assignments that are missing, unsafe,
symlinked, non-canonical, or outside an existing directory.

The staging importer is transactionally idempotent. It validates the workspace/agent binding,
serializes one account import with an advisory transaction lock, upserts only account metadata and
approval, and inserts one assignment. It explicitly performs no operation on `credentials`.

## Exact implementation file lock

1. `multica-auth-work/scripts/staging/seed_approved_assignment.sql` (new)
2. `multica-auth-work/server/internal/credentialregistry/resolver.go` (new)
3. `multica-auth-work/server/internal/credentialregistry/resolver_test.go` (new)
4. `multica-auth-work/server/internal/credentialregistry/resolver_db_test.go` (new)
5. `multica-auth-work/server/internal/daemon/credential_account_home.go` (new)
6. `multica-auth-work/server/internal/daemon/credential_account_home_test.go` (new)
7. `multica-auth-work/server/internal/daemon/daemon.go`
8. `multica-auth-work/server/internal/daemon/types.go`
9. `multica-auth-work/server/internal/handler/agent.go`
10. `multica-auth-work/server/internal/handler/daemon.go`
11. `multica-auth-work/server/internal/handler/handler.go`
12. `multica-auth-work/server/internal/handler/orq21_credential_assignment_test.go` (new)

No migration, SQLC query, generated file, credential table, or unrelated handler route changed.
Migrations 123 (`rotation`) and 124 (`approved_accounts`) already provide the required tables and
constraints, so a new migration was neither necessary nor created.

## Gates

All Go commands used Go `1.26.1` with task-private mode-0700 caches. Peak cache use was `756 MiB`,
below the accepted `2 GiB` ceiling.

### Disposable PostgreSQL

- Host identity: ORQ1 `ip-172-31-18-217.sa-east-1.compute.internal`,
  Tailscale `100.118.244.61`.
- Image: already-local
  `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0`.
- Isolation: task-specific container, ORQ1 loopback port `15545`, private SSH tunnel to local
  loopback `25545`, disposable database only.
- `go run ./cmd/migrate up`: PASS, migrations through `126`.
- `go test -tags orq21db ./internal/credentialregistry
  -run '^TestResolverDBApprovedAssignmentContract$' -count=1 -v`: PASS, zero skip.
- `go test ./internal/handler -run '^TestORQ21' -count=1 -v`: PASS (2/2), zero skip.
- Import executed twice: first run created one account/approval/assignment; second run changed zero
  rows; final counts were `accounts=1`, `approvals=1`, `assignments=1`, `credentials=0`.
- The final resolver test also proved that a NULL worktype scope fails closed and a revoked
  approval no longer resolves.

The final container, tunnel, and private caches were removed. The local tunnel port refused a
post-teardown connection and the exact container name was absent. No image prune was run.

### Go and differential gates

- Focused race gate for `internal/credentialregistry`, `internal/daemon`, and the ORQ-21 handler
  paths: PASS.
- Complete `internal/daemon` package: PASS.
- `go vet ./internal/credentialregistry ./internal/daemon ./internal/handler ./cmd/server`: PASS.
- `go build ./cmd/server`: PASS.
- `gofmt` and `git diff --check`: PASS.
- Full candidate `internal/handler` package against the same disposable DB reported exactly three
  top-level failures:
  `TestCreateChatSession_Routing`, `TestCreateWorkspaceUsesRequestedSlug`, and
  `TestCreateWorkspace_DoesNotMarkOnboarded`.
- Exact base `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`, under the identical DB/environment, reported
  those same three failures plus `TestQuickCreateIssueParentTrustBoundary`.
- Therefore the baseline differential is PASS: the ORQ-21 candidate introduces zero new handler
  failures. No skip was accepted as a pass.

## Rollout boundary

This commit is dormant until integrated and deployed. Before enabling it in a runtime, the operator
must import the approved metadata assignment for every covered agent/provider; otherwise claims
correctly fail closed. Deployment, production import, credential provisioning, and rollback are
separate owner-authorized actions and are not authorized by this implementation package.

The pre-existing dirty/untracked ORQ-21 worktree at
`/home/ec2-user/multica_workspaces/20fce817-895d-447b-965a-49f5e279314a/e39199ed/workdir/repo`
was inspected read-only and left unchanged.
