# Deploy Incidents

Canonical log of deploy/ops incidents on the self-host stack (orq1 backend/frontend, orq2 daemon/agents). Newest first. Prevention rules born from incidents live in the rollback runbook (`rollback-runbook.md`) so they are enforced, not just remembered.

---

## 2026-08-01 — Unsafe branch synchronization and credential-home lifecycle reconciliation

**Severity:** release-blocking integration incident with historical operational storage cleanup; no production deployment was performed.  
**Scope:** accepted Multica integration on coordinator host `21LAPGLMVPJ4`, ORQ2 synchronization targets, and credential-home lifecycle evidence.  
**Canonical RCA:** [`MULTICA_BRANCH_SYNC_AND_CREDENTIAL_HOME_RCA.md`](../../MULTICA_BRANCH_SYNC_AND_CREDENTIAL_HOME_RCA.md)

### What happened
A merge-based integration candidate would have inherited 133 unintended donor deletions. It was rejected in favor of additive N0v2 `9c0ad3428dd77638edde4020ff2df6c5177475d8`, which exists only on the coordination host and is absent from the ORQ2 local object graph. The detached ORQ2 root is not the accepted release tree and cannot be used to classify missing or differing generated files or migrations as RC defects.

Historical cleanup-time evidence found 30 physical credential-slot folders and retained the then-only live `slot-185` while removing 29 unreferenced folders under lock and filesystem-boundary protection. That single-slot result is explicitly historical chronology, not current state.

**Current credential-home state (owner-confirmed correction):** the eight homes were created by intentional agent logins and mapped one-to-one to eight legacy registry terminals and live panes at the reconciliation audit. No duplicate mapping or unreferenced physical home existed, no cleanup was needed, and no credential or production mutation occurred. The earlier single-slot result remains historical cleanup evidence; the later eight-home count is expected and is not independently a release or restart blocker. Installed registry v1 remains legacy, while validated v2 stable agent/subscription enforcement is a separately authorized future rollout.

### Corrections completed
- The accepted composed RC tree is complete on the coordinator host. Independent transferred-byte verification passed: archive SHA-256 `07ff862254a097cc2529216141b733ccedc1566779ec1acdc6266e06fba080b6` at 44,343,032 bytes; sidecar 3189/3189; superseding manifest 59/59; final repairs inventory 21/21 with zero path overlap; pinned migration/generated hashes; migration 128 up/down blobs; sqlc v1.31.1 deterministic generation; consolidated Go, PostgreSQL, Runtime Manager, C4, and SPE-7 gates. The ORQ2 workspace root is not the accepted final tree and must not be used to classify missing or differing generated files or migrations as RC defects. The validated two-file allocator supplement extends the governed synchronization set to 82 canonical repository-relative paths and passed 2/2 hash verification.
- The installed ORQ2 registry remains legacy v1 and no credential home was mutated. Validated source v2 remains a future rollout item; the current eight intentionally bound homes require no cleanup.
- GIT-PREFLIGHT-005 completed for the exact four-agent active topology: direct ACKs from `w5:pC`, `wP:p1`, and `wP:p2`; `w5:p9` remained sole Git authority; absent pane IDs returned `pane_not_found`; sanitized process evidence found zero Git-like processes.
- The separately validated allocator supplement adds two repository-root scripts, producing an exact 82-path synchronization scope with zero pairwise overlap.

### Current release decision
**LOCAL RC TECHNICAL BASELINE AND ITEM #3 GOVERNANCE PASS; ITEM #4 EXTERNAL-ACTION HOLD.** The accepted local RC bytes, technical gates, and Item #3 local Git-governance work are complete under w5:p9’s exclusive authority. Item #4 remains HOLD for external actions: push, deployment, and restart are eligible only after asking the owner and receiving fresh explicit approval immediately before each action; no such authorization is currently granted. The owner authorized only persistent backup, exact documentation correction, exact 82-file synchronization, and non-Git validation in the current bounded execution; Git, staging, commit, push, deployment, restart, and credential mutation remain prohibited.

### Prevention
- Reject deletion-bearing integration candidates and synchronize only exact reviewed path inventories.
- Preserve historical evidence with explicit chronology qualifiers; do not treat the detached ORQ2 tree as accepted release evidence.
- Require one physical credential home per active agent/subscription binding; count alone is not proliferation.
- Preserve the historical archive and create separate, verified rollback/final evidence for later bounded synchronization.
- Keep Git exclusive to `w5:p9`; require separate fresh owner approval for push, deployment, and restart.
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
