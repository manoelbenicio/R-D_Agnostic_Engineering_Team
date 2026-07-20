# Secrets, Users, Passwords, and Access Register

Snapshot: 2026-07-19 23:27 America/Sao_Paulo / 2026-07-20 02:27 UTC

## 1. Purpose

This register answers which identities and passwords exist, where they are controlled, and how a fresh environment restores access without placing credentials in Git.

Passwords and secret values are deliberately not written in this repository. Complete documentation means recording the secret’s purpose, identity, variable name, location, owner, permissions, provisioning method, validation method, and rotation requirement—not exposing its value.

## 2. Credential authority rule

- Only the owner may authorize login, logout, account reset, password reset, key rotation, token revocation, credential migration, session replacement, or reading a secret value.
- Agents may verify that a path exists, ownership/mode are correct, a required variable name is present, and authentication succeeds or fails without displaying the value.
- Agents must never read, copy, hash into public evidence, print, screenshot, log, or commit provider credentials or authentication files.
- Secrets transferred to a new host must use an owner-approved encrypted channel or be regenerated on the target.
- A file path in documentation is not authorization to read the file.

## 3. Candidate Multica DEV identities

### Host and Git operator identities

| Purpose | Identity | Authentication storage |
|---|---|---|
| Linux/WSL operator and file owner | `dataops-lab` | Host account; not stored in repository |
| Repository commit author | `mbenicios <mbenicios@users.noreply.github.com>` | Repository-local Git configuration |
| GitHub HTTPS push | GitHub credential available through Windows Git Credential Manager in the current host | External credential manager; token/password never stored in Git or this documentation |

### Application local identity

| Field | Current DEV value |
|---|---|
| Mode | Loopback-only local authentication bypass |
| Email identity | `owner@local.test` |
| Variable | `MULTICA_LOCAL_AUTH_EMAIL` |
| Enable flag | `MULTICA_LOCAL_AUTH_BYPASS=true` |
| Environment | `APP_ENV=development` |
| Secret/password | No user password is used while this bypass is enabled |
| Exposure control | Backend is bound to `127.0.0.1:18080`; frontend to `127.0.0.1:13100` |

This identity is for isolated DEV only. It must not be enabled on a publicly reachable or production environment.

### PostgreSQL identity

| Field | Current DEV value |
|---|---|
| Database | `multica_transition` |
| User | `multica_transition` |
| Password variable | `POSTGRES_PASSWORD` |
| Connection variable | `DATABASE_URL`, composed inside the backend container |
| Secret location | `/home/dataops-lab/.config/multica-transition/dev.env` |
| Permissions | Directory 700; file 600; owner `dataops-lab` |
| Value in Git | Prohibited and absent |

### Backend JWT

| Field | Value |
|---|---|
| Purpose | Sign/validate application authentication tokens |
| Variable | `JWT_SECRET` |
| Secret location | `/home/dataops-lab/.config/multica-transition/dev.env` |
| Generation | Cryptographically random local value generated with OpenSSL |
| Permissions | File mode 600 |
| Rotation effect | Existing tokens become invalid; coordinate restart and user reauthentication |

### Other supported backend secret variables

The backend container contract includes these secret-bearing variables even when blank/unconfigured:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `CLOUDFRONT_PRIVATE_KEY`
- `GITHUB_WEBHOOK_SECRET`
- `GOOGLE_CLIENT_SECRET`
- `MULTICA_LARK_SECRET_KEY`
- `RESEND_API_KEY`
- `SMTP_PASSWORD`
- `MULTICA_DEV_VERIFICATION_CODE`

Blank variables are not credentials. Before enabling a related integration, provision its value outside Git, record its owner and rotation, and verify logs do not expose it.

## 4. Legacy Multica identities

| Service | Identity | Secret source | Status |
|---|---|---|---|
| Legacy local app login | `admin@admin.local` with loopback bypass | Legacy Compose environment under `/mnt/c` | Active rollback stack; DEV-only |
| Legacy PostgreSQL | user `multica`, database `multica` | Legacy self-host `.env`/Compose configuration | Active rollback database |
| Legacy backend JWT | `JWT_SECRET` | Legacy self-host environment | Do not copy into candidate by default |

The legacy and candidate databases intentionally use distinct users, passwords, ports, networks, and volumes.

## 5. Agent Brain and OmniRoute secret

### Frozen contract

| Field | Contract |
|---|---|
| Daemon variable | `AGENT_BRAIN_GATEWAY_SECRET_FILE` |
| Approved target path | `/etc/agent-brain/secrets/omniroute-inference-key` |
| Dedicated child variable | `AGENT_BRAIN_OMNIROUTE_API_KEY` |
| Required ownership | Dedicated runtime/service owner; least privilege |
| Required mode | Directory 700 or stricter; secret file 600 or stricter |
| Child exposure | Exactly one approved stable gateway secret; no provider-native keys |
| Logging | Path presence may be boolean; path/value must not appear in general telemetry |

### Current new-environment fact

`/etc/agent-brain/secrets/omniroute-inference-key` was absent at the snapshot. Therefore the new environment does not yet have the frozen Agent Brain gateway secret path provisioned.

This is a blocker for ordinary gateway-required host-daemon execution. It does not block source builds, synthetic tests, or the web/backend/PostgreSQL DEV stack.

The standalone OmniRoute container being healthy does not prove that the Agent Brain daemon has a usable stable key.

### Provisioning rule

The owner/security operator must either:

1. securely transfer an approved rotated stable OmniRoute key to the target path; or
2. generate/issue a new restricted key and revoke the predecessor according to the rotation plan.

An agent may create the directory and verify mode/ownership after explicit authorization, but may not retrieve or print the value.

## 6. Redis identity and password

### Active Redis

| Field | Current fact |
|---|---|
| Container | `deploy-redis-1` |
| Owner | External AOP project `deploy` |
| ACL user | Default Redis ACL user |
| Password variable | `REDIS_PASSWORD` |
| Secret source | `/mnt/c/VMs/Projects/AOP/deploy/.env` |
| Authentication proof | Unauthenticated `PING` returned `NOAUTH Authentication required` |
| Network exposure | Loopback only at `127.0.0.1:6379` |
| Multica usage | None; candidate has no `REDIS_URL` |

The password exists, but it is not a Multica-owned secret and is not documented by value. The AOP `.env` was observed as mode 777 from Linux. Treat this as a security and portability finding.

### How Multica would reference Redis

The backend expects `REDIS_URL`, normally shaped as a secret-bearing URI. Documentation and logs must show only a redacted form such as:

```text
redis://default:<redacted>@redis-host:6379/0
```

Do not place a literal URI in Compose committed to Git. Use a secret environment file or secret-file launcher. The existing self-host Compose does not pass `REDIS_URL`; adding Redis requires a reviewed deployment change.

### Redis password recovery

- Current external password recovery is owned by the AOP environment owner.
- Do not run commands that echo the AOP `.env` or `REDIS_PASSWORD` into terminal history or agent output.
- If a dedicated Multica Redis is approved, generate a new unrelated password, use a dedicated ACL user where supported, persist Redis data, and record backup/rotation procedures.

## 7. Legacy observability identities

| Component | Identity/secret | Current location | Finding |
|---|---|---|---|
| Grafana | Admin password via `GF_SECURITY_ADMIN_PASSWORD__FILE`; default username unless separately configured | `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/deploy/observability/secrets/grafana_admin_password` | Linux observed mode 777 |
| PostgreSQL exporter | DB username | `.../deploy/observability/secrets/pg_user` | Linux observed mode 777 |
| PostgreSQL exporter | DB password | `.../deploy/observability/secrets/pg_pass` | Linux observed mode 777 |

Do not migrate these files as-is. Reissue secrets into Linux owner-only storage, update Compose secret mounts, and rotate the underlying accounts/passwords if the legacy files were broadly readable.

## 8. External AOP database identity

| Field | Value |
|---|---|
| PostgreSQL user | `aop_dev` |
| Database | `aop` |
| Password variable | `POSTGRES_PASSWORD` |
| Secret location | `/mnt/c/VMs/Projects/AOP/deploy/.env` |
| Ownership | External AOP project |

This is not a Multica database or credential.

## 9. External HerdMaster credentials

Observed paths:

- `/home/dataops-lab/.aop-runtime/herdmaster.token` — mode 600.
- `/home/dataops-lab/.aop-runtime/prometheus.token` — mode 644 at snapshot.
- `/home/dataops-lab/.config/herdmaster` — bind-mounted into remediation service.

The mode-644 Prometheus token is an external security finding. Do not modify it without the HerdMaster/AOP owner, but do not copy or expose it during the Multica migration.

## 10. Secret-file inventory for candidate operations

| Path | Mode | Contains | Git status |
|---|---:|---|---|
| `/home/dataops-lab/.config/multica-transition` | 700 | Candidate local runtime control directory | Outside Git |
| `/home/dataops-lab/.config/multica-transition/dev.env` | 600 | PostgreSQL password, JWT secret, non-secret ports/identity/settings | Outside Git |
| `/home/dataops-lab/.config/multica-transition/images.yml` | 600 | Non-secret unique image override | Outside Git |
| `/etc/agent-brain/secrets/omniroute-inference-key` | Required 600; currently absent | Stable OmniRoute inference key | Never Git |
| `/home/dataops-lab/.local/share/multica-transition/backups` | 700 | Runtime database/uploads backup | Outside Git |

## 10.1 External access items not audited by this repository

The host also contains external Chatwoot, HerdMaster, AOP, Docuseal, registry, Hadoop, and observability containers. Their complete human-user/password inventories are outside this repository’s authority. This transition inspected only the minimum safe metadata needed to identify ownership, secret paths, modes, and collision risk.

Not audited by value or account list:

- Chatwoot application/admin users and Rails secrets.
- Chatwoot Redis/PostgreSQL passwords.
- HerdMaster Grafana/admin users and full token inventory.
- Docuseal users and application secrets.
- AOP registry authentication, nginx TLS keys, and Hadoop service identities.

These are explicit external dependencies/unknowns, not assumed absent. Their owners must provide separate access registers before those projects are migrated or modified.

## 11. Fresh-environment secret bootstrap

### Regenerate DEV-only PostgreSQL/JWT secrets

Use this only when a fresh isolated DEV identity is acceptable. Do not use it when continuity of existing authentication tokens or database credentials is required.

```bash
install -d -m 700 "$HOME/.config/multica-transition"
umask 077
DB_PASSWORD="$(openssl rand -hex 24)"
JWT_SECRET="$(openssl rand -hex 48)"
```

Write the values directly into the owner-only environment file without echoing them to chat, evidence, or command output. After writing:

```bash
chmod 600 "$HOME/.config/multica-transition/dev.env"
stat -c '%a %U:%G %n' "$HOME/.config/multica-transition/dev.env"
```

Expected mode is 600.

### Preserve existing runtime identity

If the new host must restore the current DEV database and continue existing sessions, securely transfer `dev.env` and the backup artifacts together. Verify file ownership/mode before starting containers. Do not transmit them through GitHub, issue trackers, chat, screenshots, or agent prompts.

## 12. Rotation register

| Secret | Current rotation status | Required action |
|---|---|---|
| Candidate PostgreSQL password | Newly generated for isolated DEV | Rotate when changing owner/host or after exposure |
| Candidate JWT secret | Newly generated for isolated DEV | Rotate before any production use |
| OmniRoute stable key | Target file absent; historical exposure concerns exist | Owner/security provision approved rotated key; complete `PD-02`, `PD-07`, `PD-08` |
| External AOP Redis password | Present and required; external ownership | AOP owner review/rotate; do not reuse silently |
| Legacy Grafana password | Legacy secret file observed permissive from Linux | Rotate and move to restricted Linux storage before promotion |
| Legacy exporter DB password | Legacy secret file observed permissive from Linux | Rotate and move to restricted Linux storage before promotion |
| External HerdMaster Prometheus token | Mode 644 | External owner hardening/rotation review |

## 13. Safe validation commands

These verify presence and permissions without printing values:

```bash
stat -c '%a %U:%G %n' "$HOME/.config/multica-transition/dev.env"
test -s "$HOME/.config/multica-transition/dev.env"
awk -F= '/^[A-Za-z_][A-Za-z0-9_]*=/ {print $1}' \
  "$HOME/.config/multica-transition/dev.env" | sort
```

Do not use `cat`, `set -x`, `env`, `docker inspect` without filtering, or shell tracing around secret files.

## 14. Access readiness checklist

- [ ] Candidate DEV environment file exists, owner is correct, and mode is 600.
- [ ] PostgreSQL/JWT values were regenerated or securely transferred according to continuity requirements.
- [ ] Agent Brain OmniRoute secret path is provisioned by the owner/security operator.
- [ ] No provider-native credential exists in child env, argv, task home, image, log, trace, or documentation.
- [ ] Redis architecture decision is recorded before adding `REDIS_URL`.
- [ ] Any Redis selected for Multica has explicit ownership, authentication, persistence, backup, health, and rotation.
- [ ] Legacy Windows credential remediation/rotation is owner-confirmed.
- [ ] Legacy observability secrets are not reused from permissive `/mnt/c` files.
- [ ] Production identities replace DEV local-auth bypass.
- [ ] Secret recovery owners and rotation dates are recorded without values.
