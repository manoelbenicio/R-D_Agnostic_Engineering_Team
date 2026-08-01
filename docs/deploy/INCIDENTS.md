# Deploy Incidents

Canonical log of deploy/ops incidents on the self-host stack (orq1 backend/frontend, orq2 daemon/agents). Newest first. Prevention rules born from incidents live in the rollback runbook (`rollback-runbook.md`) so they are enforced, not just remembered.

---

## 2026-08-01 — Unsafe branch synchronization and credential-home proliferation

**Severity:** release-blocking integration and operational storage incident; no production deployment was performed.  
**Scope:** accepted Multica integration on coordination host `21LAPGLMVPJ4`, ORQ2 source worktrees and credential-home allocator, and shared C2/C3/SPE-10 runtime-manager contracts.  
**Canonical RCA:** [`MULTICA_BRANCH_SYNC_AND_CREDENTIAL_HOME_RCA.md`](../../MULTICA_BRANCH_SYNC_AND_CREDENTIAL_HOME_RCA.md)

### What happened
A merge-based integration candidate would have inherited 133 unintended donor deletions. It was rejected in favor of additive N0v2 `9c0ad3428dd77638edde4020ff2df6c5177475d8`, which is present only on the coordination host and absent from the ORQ2 local object graph. The detached ORQ2 root is a partial staging tree, not the accepted release tree; reviewed changes must be imported selectively into the accepted `/tmp/multica-n1-c1` tree, never by whole-branch merge.

In parallel, ORQ2 accumulated 30 physical credential-slot folders because the installed allocator used ephemeral Herdr/process identity and random fallback UUIDs. A cleanup-time `/proc` audit identified `slot-185` as the only live slot, and the other 29 folders were removed under lock with filesystem-boundary protection. The installed allocator was not hot-edited. That single-slot statement is historical, not current.

**Current restart-readiness blocker (read-only audit at 2026-08-01 07:05 BRT):** while the synchronized allocator fix remained undeployed, credential-home proliferation recurred. ORQ2 now has eight directories: `slot-184`, `slot-185`, `slot-186`, `slot-187`, `slot-188`, `slot-192`, `slot-193`, and `slot-194`. Metadata-only `/proc` reference counts are 6, 6, 6, 6, 6, 4, 4, and 6 respectively, so every directory is live and no deletion is safe. No cleanup or production mutation was performed. The source allocator and harness remain validated, but installed behavior is unremediated because deployment and restart are unauthorized. Release and restart readiness therefore remain HOLD independently of the completed source gates.

### Corrections completed in bounded source lanes
- Migration 128 was reverified unchanged: up `4b0894920069336efc36db645b05fae775a3b130`, down `a2b2ea2b56d10f57fe0144bc6998a88277039643`.
- Pinned sqlc v1.31.1 generation, generated build, credential-registry tests, and reported accepted-tree agent/realtime/handler/full-build gates passed.
- The source allocator now requires a stable agent UUID plus provider-subscription fingerprint, fails before allocation on invalid identity, removes Herdr/pane/TTY/PID/random fallback, preserves live OS references, and maintains one physical folder per active binding with zero historical physical folders. The installed ORQ2 script and live `slot-185` remain unchanged.
- C2/C3/SPE-10 owned source implements UUID lifecycle identity, durable immutable generations, compare-and-swap publication, idempotency/concurrency controls, immediate tombstone persistence, and strict runtime-configuration contracts.
- SPE-7 passed 28/28 with Git absent from `PATH`; its hot-apply document uses `version: v1` and its source ledger records six landed handwritten Runtime Manager symbols. C4 final source passed core 11/11 plus typecheck and views 12/12 plus typecheck; all nine imported files matched the tested frozen source byte-for-byte. No missing test dependency was installed.
- Final ORQ2 shared-source targeted tests, runtimeconfig/catalog race tests, all-package Go compile sweep, vet, and `go build ./...` passed. Strict OpenSpec validation passed for `credential-account-home-restoration` and `build-omniroute-agent-brain`.
- Accepted-tree Lane J passed daemon compile, full execenv, full daemon, and `go build ./...` after exact three-file restoration of Agent Brain configuration, daemon runtime/observability/admission wiring, profile suppression, and credentialless Prepare/Reuse. Because `agent.Result` has no real `ProcessID`, CLI span `proc_id` remains fail-closed and unfabricated.
- Accepted-tree Lane I completed six exact files. Imported declarations and files were independently verified against two byte-identical non-N0 authorities; focused and full affected-package tests, the Lane I compile sweep, and `go build ./...` passed. Rejected N0 was not accepted as provenance.
- Accepted-tree Lane K completed five exact files for ingress/queue observability, atomic failed-issue reconciliation, generated query output, and one handler usage-call signature. Four dependency hashes matched non-N0 authority; pinned sqlc v1.31.1 changed only generated `issue.sql.go`; migration 128 stayed exact. Full/race middleware and service tests, handler, cmd/server, whole compile sweep, and build passed.
- Accepted-tree Lane L completed bounded verified-JWT `auth_time`, password provisioning, recent-authentication, and handler provisioning-field changes while preserving `recent_auth.go` at SHA-256 `e800814c59e5ea55295d6b8c2209bf57fe776595e5d1e541ff511f4f892db94b`, the generated `db.User` 14-field shape, and local credential-registry wiring. Initial no-DB handler invocation compiled with its `TestMain` database suite skipped. Subsequent clean disposable-PostgreSQL runs passed migration through 135, protected-row 134/133/132 down/up, exact 131 refusal, reservation primitives, full non-race all-package tests, `go vet ./...`, and full race all-package tests. The disposable database was automatically cleaned up. All I/J/K/L and local test/DB blockers are closed.
- Shared-source composition is corrected: `server/migrations/135_migration_checksums.up.sql` now canonically creates `schema_migrations` if absent before `ALTER TABLE`, using `version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now()` to match `cmd/migrate`; this is a runtime no-op. Migration-135 ordinary SHA-256 is `f08d4d4a5eb894a340b72e12e830d10ddb890ce9f010677e363aa426c6672b80`. Direct pinned sqlc v1.31.1 passed with migration 135 present and no exclusion workaround. It necessarily exposed `SchemaMigration` in `models.go`; a second generation had zero byte changes. Source-only generated hashes are runtime-manager Go `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896` and `models.go` `80b974a3087ea5264a8fd2d936e8bf9a3120f88d1c56b719c348a8aaaf337f57`; compile sweep and `go build ./...` passed. The accepted composed tree must retain N0v2 migration 129 and matching `task_usage.sql`; deterministic composed `models.go` is authorized at `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb`, preserving `TaskUsage.price_version`, `TaskUsage.computed_cost_usd`, and `SchemaMigration`.
- The superseding 59-line manifest `/tmp/multica-selective-transfer-full-files.sha256` verifies 59/59 files with zero missing/failures and is authoritative at SHA-256 `52b5c6aaa8320613134638335ce8152951f9d1b715a4cfc364fa4f65bce9b53a`. Its exact source root is `/home/ec2-user/workspace/worktrees/spe6-runtime-schema/multica-auth-work`; lines are lowercase SHA-256, two spaces, relative path, and LF. Corrected hashes remain query `737f2dfad69332e90009c3ffa36455ec50f0df002667df86d8e7d3ef6c019e8b` and runner `e1719a126c3129db101b62da57b47fe72583f1838f7641992c3601c5adb1129a`. Generated files are outside the manifest's 59 copied-file set. Target sqlc regeneration may change only runtime-manager Go to `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896` and composed `models.go` to `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb`, with zero further spill. Migration 129 and `task_usage.sql` must remain byte-exact; both `TaskUsage` fields/types/JSON tags and `SchemaMigration` must remain; all other declarations must be unchanged; and a second sqlc v1.31.1 run must be deterministic. Manifest `be1b1bcd31fcc834ff8e9677ec45c9eec1ca05575d3e25c0f37c73f4e9019017`, earlier manifest `22c0a149edbcc1eedccbe8f30e10198fee29444c1dad06517537246514d8a712`, and runner `6d7f91905a9560c769ad2c6b493df3e12a697f72dfaaa8fa091ced26a9a7f7bf` are superseded provenance and must not be transferred.
- `131_runtime_sessions.down.sql` must remain unchanged: down is required to fail with SQLSTATE `55000` and `migration 131 is non-destructive and cannot be rolled down`. The approved DB gate migrates cleanly through 135, tests reversible down/up only for 134/133/132 with protected-row preservation, asserts the 131 refusal separately, never runs 130/128 down, proves 128 by forward application plus pinned blobs, and runs `TestRuntimeManagerReservationPrimitives` on the fully migrated database.

### Current release decision
**HOLD / NOT READY.** All local source, corrected runner/manifest, migration, reservation, non-race, vet, race, direct sqlc, compile, and build gates passed, and the shared-source composition blocker is resolved. Authorized target action remains: restore only canonical `server/migrations/135_migration_checksums.up.sql` into `/tmp/multica-n1-c1/multica-auth-work`, verify all 59 ordinary hashes against manifest `52b5c6aaa8320613134638335ce8152951f9d1b715a4cfc364fa4f65bce9b53a`, run sqlc v1.31.1, accept only generated hashes `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896` and composed `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb`, preserve migration 129, `task_usage.sql`, additive router/main/auth routes, and all accepted repairs, then run protected-path/placeholder checks and consolidated target Go/C4/SPE-7 gates. No stage or integration commit exists. No Git push, deployment, allocator installation, production mutation, or restart is authorized.

### Prevention
- Reject deletion-bearing integration candidates before construction; use additive path invariants and selective import.
- Never treat the detached ORQ2 root or an isolated package build as release evidence for the accepted remote tree.
- Keep generated Go pinned to sqlc v1.31.1 and reverify immutable migration blobs around schema integration.
- Require stable credential binding identity and fail closed before allocation; retain tombstone metadata when required for non-reuse, but retain zero historical credential folders.
- Run complete companion, production composition, real-database, and consolidated final-tree gates before commit or release authorization.

---

## 2026-07-24 — Backend crash-loop after image redeploy (missing canonical env-file)

**Severity:** production API outage (~12:20–12:27 UTC, backend down / crash-looping).
**Scope:** `multica-dev-transition` backend container on orq1. Daemon, frontend, Postgres, OmniRoute unaffected. No data loss. No user logout.

### What happened
While redeploying the backend onto the fresh nim-removed image (`994aa2284b55`, server `a05415c4`, consistent with daemon `b635556`), the container was recreated with a bare `docker compose up --force-recreate backend`. That recreate **regenerated the container environment from the compose files + defaults only** — it did **not** load the deployment's canonical env-file. The backend's real secrets/config are **not** in the repo `.env`; they live in `~/.config/multica-transition/dev.env` (16 vars: `JWT_SECRET`, `POSTGRES_USER/PASSWORD/DB`, `BACKEND_PORT`, …), which the original deploy loaded via `--env-file`.

### Failure cascade
1. `POSTGRES_*` fell back to compose defaults (`${POSTGRES_USER:-multica}` …) → `postgres://multica:multica@…`, but Postgres runs as `multica_transition` → **`password authentication failed`** crash loop.
2. After patching DB creds inline (whack-a-mole), migrations ran ("Done."), then a **second** crash: **`JWT_SECRET must be a non-placeholder secret ≥32 bytes`** (JWT had also defaulted to a placeholder).

The image, DB, and migrations were all healthy — the failure was **purely missing environment**, so an image rollback would not have helped (same root cause).

### Root cause
The recreate did not supply the canonical deployment env-file; critical secrets defaulted to broken values.

### Resolution (Principal-endorsed forward-fix)
Recreated with the **single canonical env source**:
```
docker compose -p multica-dev-transition \
  -f docker-compose.selfhost.yml -f docker-compose.selfhost.build.yml \
  -f ~/.config/multica-transition/images.yml \
  --env-file /home/ec2-user/.config/multica-transition/dev.env \
  up -d --no-deps --force-recreate backend
```
Guardrails applied: pre-verify env-file keys (without printing secret values), bounded ~90s health gate, and an **armed auto-rollback** to the prior image (`5e8882da1a85`) if health did not return. Result: **`/health=200` in ~10s**, no rollback needed. Because the same canonical `JWT_SECRET` was used, **no sessions were invalidated**.

### Prevention (now a hard rule — see `rollback-runbook.md`)
**ALL backend/daemon container recreates MUST pass `--env-file /home/ec2-user/.config/multica-transition/dev.env` as the single env source.** Never a bare `docker compose up`, which silently drops the deployment secrets and defaults them to broken values.

### Follow-ups opened
- Daemon runs as an **unsupervised foreground process** from `/tmp/multica-auth-fixed` (no systemd/auto-restart; a reboot or `/tmp` cleanup kills it). Move to a stable path + supervision — needs a restart window + owner OK.
- Prune stale `/tmp` build artifacts/backups when idle (low priority, safe).
