# ORQ-43A — independent review and integration sequencing

- Commit reviewed: `7d2f36fc485d0218f9e9e72bdcef3bb955a9bd88`
- Parent: `368c5a973467aab3b9c8719d683c04ec2035fe74`
- Mode: READ-ONLY. No migration number, DB, build, sqlc, production, push, or board action.
- Verdict: **BLOCK for integration of the two-commit stack as dormant or complete.**

## What 7d2f36fc closes

- The router attaches `RequireHumanActor` to issuance and registers individual revocation.
- JSON decoding rejects unknown fields and successful token responses use `Cache-Control: no-store`.
- Revocation is bound to token UUID, workspace UUID, and daemon ID, returns only the hash to the
  server, and invalidates that one cache entry.
- Expired-token deletion is invoked before issuance.
- `StoreDaemonToken` is a currently uncalled client helper that writes through a private temporary
  file, fsyncs, atomically renames, and refuses an existing symlink target.

## Blocking findings

1. The stack is not dormant. Parent `368c5a9` registers
   `POST /api/workspaces/{id}/daemon-tokens`; `7d2f36f` registers the corresponding DELETE route.
   Integrating both commits exposes issuance and revocation immediately.
2. Durable idempotency is absent. The handler does not require or store `Idempotency-Key`; retry
   after an ambiguous response can mint another overlap token.
3. Issuance does not prove that the supplied daemon ID belongs to the URL workspace. It stores the
   caller-provided string directly. UUID/workspace parsing is not cross-daemon binding.
4. The focused test added by 7d2f36f covers only strict JSON. There are no route-level owner/admin
   versus machine/plain-member tests, no cross-daemon rejection, no single-revoke/cache/overlap
   test, and no expiry or idempotency DB gate.
5. The commit edits `pkg/db/generated/daemon_token.sql.go` directly in the code lane. Repository
   governance assigns generated output to the serial LANE-DB/sqlc owner.
6. The placeholder FILES_LOCKED list is incomplete: adding `idempotency_key` to `daemon_token`
   changes the generated `DaemonToken` model, so `pkg/db/generated/models.go` must also be owned as
   sqlc output. Other generated files may be added only if a clean `sqlc generate` proves they
   changed.

## Dormant integration boundary

Only the client storage helper and its focused tests are inherently dormant:

- `server/internal/daemon/client.go`
- `server/internal/daemon/client_test.go`

They may be extracted into an independent commit after review. The current commit cannot be
cherry-picked as that dormant unit because it also contains active router/handler/query changes.

Handler and query code could be staged dormant only with both daemon-token routes withheld. That
would require a new, explicitly reviewed split; the current two-commit stack does not satisfy it.

## Minimal next sequence

### S0 — canonical reservation and ownership

The registrar/GTL reserves one canonical migration number and assigns the serial LANE-DB owner.
No filename is materialized before that ruling.

### S1 — LANE-DB schema/query/sqlc

Exact FILES_LOCKED:

1. `server/migrations/<RESERVED>_daemon_token_idempotency.up.sql`
2. `server/migrations/<RESERVED>_daemon_token_idempotency.down.sql`
3. `server/pkg/db/queries/daemon_token.sql`
4. `server/pkg/db/generated/daemon_token.sql.go` — sqlc output only
5. `server/pkg/db/generated/models.go` — sqlc output only
6. Any additional `server/pkg/db/generated/*.go` only when the clean generator diff proves it.

Contract: nullable bounded idempotency key; unique partial
`(workspace_id, daemon_id, idempotency_key)`; create enforces existence of the workspace/daemon
binding and handles a concurrent uniqueness loss without exposing the discarded raw token;
metadata-only lookup supports deterministic retry conflict; down drops only the exact index and
column.

### S2 — handler and route acceptance

Exact FILES_LOCKED:

1. `server/internal/handler/daemon_token.go`
2. `server/internal/handler/daemon_token_test.go`
3. `server/cmd/server/router.go` — only the formally granted ORQ-43A hunk

Require a bounded non-empty `Idempotency-Key`, strict single-JSON-document decoding, owner/admin
human route tests, rejection of task/cloud machine actors and plain members, cross-daemon rejection,
raw-once/no-store success, content-free failures, individual revocation, cache invalidation, overlap
survival, and expiry behavior.

### S3 — dormant client delivery

Exact FILES_LOCKED:

1. `server/internal/daemon/client.go`
2. `server/internal/daemon/client_test.go`

Re-review owner/mode checks, overwrite behavior, and content-free failures, then integrate the
uncalled helper independently or together with the final gated stack.

### S4 — final ephemeral DB gate

On a private disposable DB only: migration up/down/up, clean and repeatable `sqlc generate`,
HashToken-only persistence, raw-once, same-key retry with one row, concurrent same-key race,
different-key overlap, old revoke/new accepted, cache invalidation, expiry, actor/role and
cross-daemon failures, zero skips, and no raw token in failure output.

## ETA

- Dormant client-only extraction/re-review: **20–30 minutes**.
- After canonical reservation and LANE-DB ownership: **90–150 minutes** for S1–S4, including the
  ephemeral DB gate and independent re-review.
- Reservation wait is external and is not represented as an invented ETA.

Until S0–S4 pass, neither `368c5a9` nor `7d2f36f` should enter a deployable integration branch.
