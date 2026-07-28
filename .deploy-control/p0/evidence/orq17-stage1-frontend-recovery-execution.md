# ORQ-17 — Stage 1 frontend recovery execution

- Time: `2026-07-27T15:52:09Z`
- Authorization: owner-authorized Stage 1, ORQ1 only.
- Result: **PASS — rebuild safely avoided.**
- Safety skill: `.agents/skills/aws-secrets-manager/SKILL.md` read completely before the operation.
  No secret value, `Config.Env`, backend override content, login, AWS, DB, board, Serve or Funnel
  mutation occurred.

## Identity and preflight

| Check | Result |
|---|---|
| Local executor | ORQ2: `orq2`, Tailscale `100.110.178.47` |
| Remote target | ORQ1: hostname `ip-172-31-18-217.sa-east-1.compute.internal`, Tailscale `100.118.244.61`, DNS `orq1.tail96e2c0.ts.net` |
| Source | `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work` |
| Rejected source | `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team/...` is the ORQ2 proposal and was not used |
| Compose | plugin `docker compose`, version `v5.3.1` |
| Project/service | `multica-dev-transition` / `frontend` |
| Ports | frontend `3000/tcp -> 127.0.0.1:13100`; backend `8080/tcp -> 127.0.0.1:18080` |
| Initial health | frontend `/` = `200`; backend `/health` = `200` |
| Tailscale | `serve status --json={}` and `funnel status --json={}` |

The frontend label records the compose base file. The effective project label on the backend records
the four-file stack:

1. `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml`
2. `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml`
3. `/home/ec2-user/.config/multica-transition/images.yml`
4. `/home/ec2-user/.config/multica-transition/backend-env.override.yml`

No contents from item 4 were read or printed.

## Mandatory rollback gate

Rollback directory:
`/home/ec2-user/.local/state/orq17-stage1-frontend-recovery-20260727T155000Z`

- directory and `files/`: mode `0700`, owner `ec2-user`;
- four config backups, Serve JSON, safe container/image metadata and `SHA256SUMS`: mode `0600`;
- recovery command: mode `0700`, SHA-256
  `87463b4280c81b65e93dae18ea8ab22e94143cdffaa5690d4929611933ee93c7`;
- `sha256sum -c SHA256SUMS`: eight `OK`, zero failures;
- saved Serve JSON: `{}`;
- prior frontend image:
  `sha256:cf8017e3d2fd2b19e7e2edba029b9a8454e923824420797cc23b69833bd5416a`;
- immutable rollback tag:
  `multica-web:orq17-stage1-rollback-cf8017e3d2fd`;
- no image prune was executed.

The recovery command restores only `frontend`, using
`--force-recreate --no-deps frontend`; it was recorded but not executed.

## Public bundle decision gate

The current public root HTML and its referenced JavaScript were scanned first. A subsequent
exhaustive scan covered all **453** public static JavaScript files in the frontend container:

- absolute incompatible assignment to `apiBaseUrl`/`NEXT_PUBLIC_API_URL`: **0**;
- relative `/api/` tokens: **8**;
- root bundles independently contained ten `/api/me` strings;
- `NEXT_PUBLIC_API_URL` had no incompatible absolute value embedded;
- `https://api.multica.ai` occurs only as the explicit cloud target of the “connect remote” UI
  (`packages/views/runtimes/components/connect-remote-dialog.tsx:30`), not as the current
  `apiBaseUrl`;
- requests therefore retain the intended same-origin/relative `/api` behavior. When the UI is
  reached at the ORQ1 HTTPS FQDN, those relative requests resolve to that origin without requiring
  a hard-coded literal.

The source Dockerfile does not declare `ARG` or `ENV NEXT_PUBLIC_API_URL`; it supports
`REMOTE_API_URL`, `NEXT_PUBLIC_WS_URL` and `NEXT_PUBLIC_APP_VERSION`. The ORQ1 source checkout also
contains extensive unrelated changes. A rebuild solely to force a literal URL would therefore be
unsupported and unsafe.

## Final state

| Resource | Before and after |
|---|---|
| Frontend container | `1b3b6c9ee32a74d8eb5fc08c47ffea96625f644045dae34184300e30fd4cca2d` |
| Frontend image | `sha256:cf8017e3d2fd2b19e7e2edba029b9a8454e923824420797cc23b69833bd5416a` |
| Backend container | `dc719a1bf4e7d41fb9823f67877831e4bf7daf9059eaa3f9b6ff74400d0f1845` |
| Backend image | `sha256:60133934d8f99c72e05abc171304e5a7a41ce6827ef4590a873f5642f38bf101` |
| Frontend HTTP | `127.0.0.1:13100/` = `200` |
| Backend health | `127.0.0.1:18080/health` = `200` |
| Serve/Funnel | `{}` / `{}` |

No container was rebuilt, recreated or restarted. The only Docker metadata mutation was adding the
task-specific rollback tag to the already-running frontend image, as required by the owner backup
gate. The independent monitor likewise observed no container-ID drift and preserved green health.

**Stage 1 verdict: PASS with rebuild safely avoided and rollback artifacts preserved.**
