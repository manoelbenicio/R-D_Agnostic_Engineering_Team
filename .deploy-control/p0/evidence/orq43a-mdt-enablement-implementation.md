# ORQ-43A — `mdt_` enablement implementation

**Status:** PROPOSAL implementation committed locally; peer review requested.
**Owner/worktree:** `agent/codex-a/orq43-mdt-enablement`,
`/home/ec2-user/workspace/worktrees/gtl-orq43-mdt-enablement`.

## Gap closed

The reviewed design proved `GenerateDaemonToken` and the `CreateDaemonToken`
SQLC method existed without production callers. This commit connects them via
an owner/admin-only route:

`POST /api/workspaces/{id}/daemon-tokens`

The handler validates request shape and future expiry, generates `mdt_`, stores
only `auth.HashToken(raw)` with workspace/daemon binding, and returns the raw
token once as the explicit issuance contract. ORQ-43B rotation, secret
provisioning, AWS, and production mutation remain out of scope.

## Locked files and commit

Only the router registration hunk, new handler, and focused tests changed:

- `server/cmd/server/router.go` (single ORQ-43A route hunk)
- `server/internal/handler/daemon_token.go`
- `server/internal/handler/daemon_token_test.go`

Local commit: `368c5a973467aab3b9c8719d683c04ec2035fe74`.

## Verification

- Focused handler tests: PASS.
- Focused `-race` tests with `CGO_ENABLED=1`: PASS.
- `go vet` for handler/server packages: PASS.
- `gofmt` and `git diff --check`: PASS.
- Private build cache removed after tests.

No secret value, AWS API, `asm-exec`, database, container, production daemon,
rotation, board, push, or PR was touched. The route hunk is reserved to this
ORQ-43A lane; W3 must not edit it.
