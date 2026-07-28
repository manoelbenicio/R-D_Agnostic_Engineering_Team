# ORQ-21 R3 — exclusive approved-account assignment correction

- Owner: `Codex56#B`
- Branch: `agent/codex56-b/orq21-r3`
- Immutable R2 parent: `182b7b3644c860f3149dee75292dc5dfe8ea6b6a`
- Code-only R3 commits:
  - `ee87f7b8483507269ddc6631d17af99d41b4d204`
  - `ec8f945f1522e543469ae8dba2f211961549d4a4` (existing task-root containment)
  - `aac222715c46dd3000b00c0df3ba43e83b94a73f` (canonical vendor at write, exact read)
  - `feccee4be2d9443e37f8b050de05473d1885ad38` (reclaim fixtures preserve the
    frozen snapshot and retry uses a fresh queued row)
- Status: **R3 PASS / FINAL ORQ-12 COMBINED GATE PASS**
- Safety boundary: no production database, product container, credential value, AWS API,
  Secrets Manager value, board, remote, push, merge or deployment was touched.

## Rulings implemented

1. The metadata importer takes an account-scoped advisory transaction lock, rejects a
   second agent with `E_ACCOUNT_ALREADY_ASSIGNED`, accepts only `GENERAL`, and never
   changes a pre-existing `allowed=false` approval back to true.
2. The resolver accepts `available` and `leased` only when the durable assignment belongs
   to the current agent and no other agent has the account. Tenant/workspace, provider and
   approval predicates remain fail-closed.
3. The daemon independently derives mandatory assignment for covered canonical providers
   when `MULTICA_CREDENTIAL_ASSIGNMENT_ENFORCED=true`. The server payload may tighten the
   check but cannot disable that local gate. Rollout order is server/schema first, then
   daemon gate on.
4. `agy` is canonicalized to `antigravity` before configuration lookup and execution
   environment preparation.
5. Account homes must be exactly `<controlled-root>/<slot>/home`. Root, slot, home and
   provider credential artifacts must be owned by the daemon uid, private, non-symlink and
   match the existing Codex, Kiro or Antigravity layout.
6. Assignment failure cancels the rejected attempt, but the normal rerun service creates a
   distinct queue row. A DB-backed test proves the corrected assignment can be claimed by
   that new attempt.
7. `AccountID` remains server-side metadata. It is logged only with task/agent correlation
   as an interim audit bridge and is forbidden in the daemon claim JSON. Durable
   `credential_account_id` storage remains in ORQ-12 ownership.
8. `accounts.vendor` is canonicalized at import time. Only Codex, Kiro and Antigravity
   are accepted; the legacy `agy` spelling is persisted as `antigravity`. The resolver
   compares the persisted value exactly and rejects noncanonical drift.
9. An account-bearing execution refuses a pre-existing task root and preserves its
   contents. It never enters the legacy `RemoveAll` path.

## Fail-closed defect found during the gate

The first real `psql` execution showed that PostgreSQL 17 treats `\quit 1` as `\quit` with
an ignored extra argument, returning success after a refusal. R3 replaces every such path
with a fixed-code `RAISE EXCEPTION` under `ON_ERROR_STOP`. The repeated gate then proved
revocation, unsupported scope and exclusivity return nonzero without exposing input values.

## Executed gates

- Disposable PostgreSQL: pinned local
  `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0`,
  isolated on ORQ1 loopback port `15546`, reached only through a private SSH local tunnel.
  Migrations `001` through `126` applied to database `orq21_test`; no product DB used.
- `go test -tags orq21db ./internal/credentialregistry -count=1 -json`:
  **5 tests PASS, 0 fail, 0 skip**, including the real seed script, idempotency, revoked
  rerun preservation, concurrent assignment and GENERAL-only rejection.
- `go test ./internal/handler -run '^TestORQ21' -count=1 -json`:
  **2 tests PASS, 0 fail, 0 skip**.
- The same handler focus under `-race`: **2 PASS, 0 fail, 0 skip**.
- Focused resolver/daemon `-race`: **48 PASS, 0 fail, 0 skip**.
- Complete credential-registry/daemon packages: **591 PASS, 0 fail**. The two daemon
  platform skips (`TestIsDriveRoot` and the macOS private-root case) are unrelated and
  were not added by R3.
- Complete handler differential, identical DB and environment:
  - R2 parent: 1336 pass, 3 fail, 37 skip.
  - R3 candidate: 1336 pass, 3 fail, 37 skip.
  - New failures: zero. Added skips: zero. Removed skips: zero.
  - The identical failures are
    `TestCreateChatSession_Routing`, `TestCreateWorkspaceUsesRequestedSlug`, and
    `TestCreateWorkspace_DoesNotMarkOnboarded`.
- `go vet ./internal/credentialregistry ./internal/daemon ./internal/handler ./cmd/server`:
  PASS.
- `go build ./cmd/server`: PASS.
- `gofmt`, `git diff --check`, staged scope and code-only commit inspection: PASS.
- Go toolchain: `go1.26.1 linux/amd64`. Private task cache remained below the approved
  2 GiB ceiling.
- Independent review:
  - `ee87f7b`: one narrow BLOCK for the account-bearing existing-root path.
  - `ec8f945`: PASS; the sentinel mutation test proves the old behavior fails.
  - `aac2227`: PASS; the reviewer independently reproduced all 5 tagged DB tests and
    exact canonical storage/read behavior.
    The review correctly notes that migrations through canonical `126` still have only
    `assignments(agent_id)` uniqueness plus a non-unique `account_id` index. The durable
    unique `assignments(account_id)` invariant exists only after applying ORQ-12's staged
    migration; R3's advisory-lock plus conflict checks are the pre-promotion enforcement.
  - `feccee4`: PASS delta. The reviewer independently proved the intentional sequencing:
    standalone ORQ-21 and materialized migrations through `127` fail because the snapshot
    column is absent; explicitly applying ORQ-12's unnumbered staged migration makes the
    two ORQ-21 handler tests and their `-race` run pass with zero skip, while the tagged
    registry gate remains 5/5 PASS. The staged schema also supplies the durable unique
    `assignments(account_id)` index.

## Final combined ORQ-12 gate

A private integration worktree combined ORQ-12 through:

- `de38d3db7d360b6db2a789dc3a2756f9e2af97f2`;
- `d93ad9f2acacae3ee0f15e5b94b50b57fca1f7aa`;
- `c146635b63b68ab3e77fe84fd46be37983aea4d3`;
- `12581e1b50fdb884aea014834f4a88c4ef74d674`;
- `a70072a52d0ef0d3ffa22297712e8241a945f0f4`; and
- `dd95e0a7ef78d8844ef55e0a1765d8849b7d18bf`,

plus the four ORQ-21 R3 commits above. Every final measurement used a newly created
disposable database with canonical migrations through materialized
`127_task_usage_thinking_level`, followed by the still-unmaterialized staged ORQ-12
migration. No migration number was invented.

- staged up → down → up: expected columns `2 → 0 → 2`; the unique
  `assignments(account_id)` index was present after up;
- real backfill fixture: both `accounts.vendor=' AGY '` and
  `agent_runtime.provider=' AGY '` became exact `antigravity`;
- combined handler acceptance matrix: **39 pass, 0 fail, 0 skip**;
- focused combined handler `-race`: **7 pass, 0 fail, 0 skip**;
- combined credential registry DB gate: **5 pass, 0 fail, 0 skip**;
- complete handler package: **1383 pass, 3 fail, 37 skip**. The failures and skip
  count exactly match the previously measured R2 baseline; the only failures remain
  `TestCreateChatSession_Routing`, `TestCreateWorkspaceUsesRequestedSlug`, and
  `TestCreateWorkspace_DoesNotMarkOnboarded`. New failures: zero. Added skips: zero;
- combined vet and server build: PASS.

Private JSON gate-record hashes (files removed during mandatory teardown):

- combined matrix: `747a70745915aa4b357699ec93f4a9d2eceed3a6d9dc036dc1c42bf4f74a8cad`;
- focused race: `7bd513245c4646e6aa8da6be0c935ce702ad0f6bb2f3bd6ef96c166a37929deb`;
- registry DB: `22e6e473f82958d9cc3a501d73365041acdd24aa0d6ee4f877d07c0b496615bf`;
- complete handler: `ae6816158fb22288e013590b1f9d0169ac3e910f77d32db124f3938245b452ae`.

The gate first caught and rejected four stale read-time alias-repair tests, then caught
the real ORQ-21/ORQ-12 sequencing mismatch: old ORQ-21 fixtures attempted to reclaim
covered dispatched rows with NULL snapshots. `feccee4` now gives the approved reclaim
fixture its already-frozen account and tests missing-assignment cancellation on a new
queued attempt. This preserves both rulings: reclaim never re-resolves, while a
recoverable retry is a distinct queue row.

## ORQ-12 handoff

The authoritative ADR is
`.deploy-control/p0/evidence/adr-orq21-orq12-producing-account-snapshot.md`.
ORQ-12 owns the migration/query/generated changes that add and atomically freeze
`agent_task_queue.credential_account_id`, and copies that existing-first snapshot into
`task_usage`. R3 does not edit those files or accept an account ID from the daemon.
The final combined ephemeral DB up/down/up and behavioral gate is complete; promotion
still requires Registrar materialization and the aggregate zero-active-queue deploy gate.

## File hashes

- `seed_approved_assignment.sql`:
  `7e307c88386e75d8c8762709f0622677bce537abdcac09446c06bd6e5dc30e74`
- `resolver.go`:
  `1460cd32d164e03799deaf40fc1c0b86d2569234bc173492753e91673c046d55`
- `credential_account_home.go`:
  `a6c1cb9c73a09d961191c9d77a0468f272eb551083ecc1efd697cda0d423a332`
- `execenv.go`:
  `fe6af9a498bbda395d9567901866d8d092d5071214f8c22d12924d64af103dfd`
