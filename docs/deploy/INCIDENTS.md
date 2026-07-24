# Deploy Incidents

Canonical log of deploy/ops incidents on the self-host stack (orq1 backend/frontend, orq2 daemon/agents). Newest first. Prevention rules born from incidents live in the rollback runbook (`rollback-runbook.md`) so they are enforced, not just remembered.

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
