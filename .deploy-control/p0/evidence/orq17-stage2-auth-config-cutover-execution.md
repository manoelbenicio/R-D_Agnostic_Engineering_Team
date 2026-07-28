# ORQ-17 — Stage 2 auth configuration cutover

- Completed: `2026-07-27T15:57:03Z`
- Target: ORQ1 only, hostname `ip-172-31-18-217.sa-east-1.compute.internal`,
  Tailscale `100.118.244.61`
- Authorization: owner-authorized Stage 2.
- Result: **PASS**.
- Secret safety: `.agents/skills/aws-secrets-manager/SKILL.md` was reloaded completely. No secret
  value, `GetSecretValue`, SMA endpoint, full `Config.Env`, secret-bearing override content, login,
  AWS, DB write, board, Serve/Funnel, push or PR action occurred.

## Pre-cutover state

Only the owner-allowlisted effective values were read. Email was reported as presence only and the
Google redirect as presence/host only:

| Setting | Before | Desired |
|---|---|---|
| `MULTICA_LOCAL_AUTH_BYPASS` | `true` | `false` |
| `MULTICA_LOCAL_AUTH_EMAIL` | `PRESENT` | `EMPTY` |
| `FRONTEND_ORIGIN` | `http://localhost:13100` | canonical HTTPS origin |
| `MULTICA_APP_URL` | `http://localhost:13100` | canonical HTTPS origin |
| `MULTICA_PUBLIC_URL` | empty | canonical HTTPS origin |
| `CORS_ALLOWED_ORIGINS` | empty | canonical HTTPS origin |
| `AUTH_TOKEN_TTL` | empty | `24h` |
| `MULTICA_TRUSTED_PROXIES` | empty | `127.0.0.1/32` |
| `APP_ENV` | `development` | `production` |
| `COOKIE_DOMAIN` | empty | empty |
| Google redirect | `PRESENT`, host `localhost:13100` | same callback, canonical host |

Before the cutover, anonymous `/api/me` returned `200`, confirming that the local bypass was active.
Both `/health` and `/readyz` returned `200`; frontend `127.0.0.1:13100` returned `200`; Serve and
Funnel status were `{}`.

## Queue and rollback gates

Two pre-mutation aggregate-only reads used all active states in this exact order:
`queued`, `dispatched`, `running`, `waiting_local_directory`.

| Read | Per-state counts | Total active |
|---|---|---:|
| T1 | `0|0|0|0` | 0 |
| T2, three seconds later | `0|0|0|0` | 0 |

The existing owner-mandated rollback set was explicitly extended to cover Stage 2:

`/home/ec2-user/.local/state/orq17-stage1-frontend-recovery-20260727T155000Z`

- backend container and image safe metadata: `0600`;
- backend recovery command: `0700`;
- original four Compose/override backups retained at `0600`;
- old backend image preserved under
  `multica-backend:orq17-stage2-rollback-60133934d8f9`;
- complete `SHA256SUMS` verification: PASS;
- no image prune.

The recovery command uses the original four files only and recreates only
`backend` with `--force-recreate --no-deps backend`.

## Applied configuration

A separate non-secret override was created:

`/home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml`

- mode/owner: `0600`, `ec2-user:ec2-user`;
- SHA-256: `510039dd01187a179d6d1ef79126bdbcd9ea0c6cb98f6f952d88955810a7705c`;
- copied into the private rollback set at `0600`;
- contains only the approved non-secret auth values.

Because the existing Google redirect was independently proven non-empty with host
`localhost:13100`, its host was changed to `orq1.tail96e2c0.ts.net` while preserving the documented
`/auth/callback` path. No Google credential was read or changed.

Compose used project `multica-dev-transition`, the exact four live-label files, followed only by the
new auth override:

1. `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml`
2. `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml`
3. `/home/ec2-user/.config/multica-transition/images.yml`
4. `/home/ec2-user/.config/multica-transition/backend-env.override.yml`
5. `/home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml`

The only recreate target was `backend`, using `--force-recreate --no-deps backend`.

## Validation

| Gate | Result |
|---|---|
| Backend container | changed from `dc719a1bf4e7d41fb9823f67877831e4bf7daf9059eaa3f9b6ff74400d0f1845` to `75e4416f06e92d51bdc9d6d973939e626d9479489551c82c70968f38deb8a928` |
| Backend image | unchanged: `sha256:60133934d8f99c72e05abc171304e5a7a41ce6827ef4590a873f5642f38bf101` |
| Backend health/readiness | `/health=200`, `/readyz=200`; running, restart count 0 |
| Anonymous `/api/me` | `401`, 34-byte error body discarded |
| Scoped anonymous `/api/issues` | synthetic workspace scope, `401`, 34-byte error body discarded; no issue data read |
| Frontend | `127.0.0.1:13100=200`; container/image unchanged |
| PostgreSQL | container/image unchanged |
| OmniRoute | container/image unchanged |
| Queue after cutover | `0|0|0|0`, total active 0 |
| Serve/Funnel | `{}` / `{}` |
| Rollback checksum set | PASS after adding post-cutover safe metadata |

Effective allowlist after recreate:

```text
MULTICA_LOCAL_AUTH_BYPASS=false
MULTICA_LOCAL_AUTH_EMAIL=EMPTY
FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net
MULTICA_APP_URL=https://orq1.tail96e2c0.ts.net
MULTICA_PUBLIC_URL=https://orq1.tail96e2c0.ts.net
CORS_ALLOWED_ORIGINS=https://orq1.tail96e2c0.ts.net
AUTH_TOKEN_TTL=24h
MULTICA_TRUSTED_PROXIES=127.0.0.1/32
APP_ENV=production
COOKIE_DOMAIN=
GOOGLE_REDIRECT_URI=PRESENT host=orq1.tail96e2c0.ts.net
```

Rollback was not needed. A second stability read three seconds later reproduced all health, auth,
container-isolation, queue and Serve/Funnel gates.

**Stage 2 verdict: PASS. Auth bypass is closed and the canonical HTTPS configuration is active,
while Serve/Funnel remain empty pending a separately authorized stage.**
