# ORQ-17 Stage3B — Gate 0 tool/permission preflight

- **UTC:** 2026-07-27T17:03:21Z
- **Recipient:** General-TL / Codex56-TL
- **State:** `REPORTED_PENDING_ACCEPTANCE`; execution remains blocked on the explicit items below.
- **Scope:** CloudFormation KMS/Secrets Manager bootstrap infrastructure plus ORQ1 Stage3B/login/Serve/Kanban validation.

## 1. Required tools and measured versions

### ORQ2 control/build host
- AWS CLI `2.36.5`: `/home/ec2-user/.local/bin/aws`
- `cfn-lint 1.46.0`: `/home/ec2-user/.local/bin/cfn-lint` (isolated venv supplied by General-TL)
- Python `3.9.25`
- OpenSSH `8.7p1`; curl `8.17.0`; GNU `sha256sum`
- `asm-exec`: `/home/ec2-user/.local/bin/asm-exec`, SHA-256 `d55eb38ad33a5b76f584ca180f633ecc120cf39b8fd29427ffbe11a8fbf19556`
- Go executable pin used by V2: Go `1.26.1 linux/amd64`, SHA-256 `548e61b2d08ae52043be2f1924ed3c1d2b2c41967e360f3e317667f6fa912fc2`

### ORQ1 production target
- Docker + Docker Compose `5.3.1`
- Tailscale `1.98.9`
- curl `8.17.0`; OpenSSH; `psql`; `pg_dump`; `sha256sum`
- Go `1.26.1 linux/amd64`, `/usr/local/go/bin/go`, matching SHA-256 `548e61b...`
- AWS principal: `arn:aws:sts::809809509961:assumed-role/cw-agent-orquestradores/i-0d9d441dd364039f9`

## 2. Missing tools

- **BLOCK:** `asm-exec` is absent on ORQ1. General-TL must make the pinned `d55eb38...` executable available in a task-private path without modification; do not substitute another build.
- `cfn-guard` is absent on ORQ2. It is a recommended compliance layer, not required for the already authorized minimal stack create; no installation was attempted.

## 3. IAM/access requirements

Observed hard denials for ORQ2 role `arn:aws:iam::809809509961:role/cw-agent-orquestradores`:

1. **BLOCK:** `cloudformation:CreateStack` on `arn:aws:cloudformation:sa-east-1:809809509961:stack/prod-multica-bootstrap-owner/*`.
2. `cloudformation:DescribeStacks` on the same stack ARN (metadata/reuse verification).
3. `secretsmanager:ListSecrets` on `*` (exact-name metadata/reuse lookup; list action is not resource-scoped).

The first exact grant is required before retry. Because no CloudFormation service role is supplied, the caller will also need stack-execution actions for the template when creation begins: dedicated KMS key/alias creation, rotation enablement/tagging/key policy, Secrets Manager secret creation/tagging/resource policy, and metadata reads. Grant these only to this stack workflow and exact secret/key where AWS supports resource scoping. Stage3B resolution additionally requires only `secretsmanager:GetSecretValue` for the exact created secret and `kms:Decrypt`/`kms:DescribeKey` for its dedicated key via `secretsmanager.sa-east-1.amazonaws.com`; the template resource/key policies scope resolver access to the exact role. Never authorize `BatchGetSecretValue`.

No AWS resource was created by the denied call.

## 4. Services, ports and dependencies

- ORQ1 exact identity: `ip-172-31-18-217.sa-east-1.compute.internal`, Tailscale `100.118.244.61`.
- `127.0.0.1:13100` LISTEN and HTTP `200` (frontend).
- `127.0.0.1:18080` LISTEN; `/health=200`, `/readyz=200` (backend).
- `127.0.0.1:443` CLOSED; Serve `{}` and Funnel `{}` before Stage3B.
- Compose project services healthy: backend up, frontend up, Postgres healthy.
- PostgreSQL is an internal Compose dependency on `postgres:5432`; no host/public DB exposure is required.
- Required network paths: SSH/Tailscale ORQ2→ORQ1; HTTPS 443 from execution host to AWS resolver backend/CloudFormation/KMS/Secrets Manager; later Tailscale Serve HTTPS 443.
- Stage3B V2 remains BLOCK pending synchronized F1–F5 package and independent PASS; latest helper/runner hashes were still changing during this preflight.

## 5. Temporary/cache space

- ORQ2 `/`: 17,118,269,440 bytes free (74% used); `/tmp`: 7,864,344,576 bytes free.
- Existing private V2 build closure/cache: 1.7 GiB.
- ORQ1 `/`: 5,231,284,224 bytes free (80% used); `/tmp`: 3,774,242,816 bytes free.
- Plan: reproduce/build on ORQ2 and transfer only the approved binary/runner/metadata to ORQ1. Estimated additional ORQ2 working space ≤2 GiB; ORQ1 task-private execution/backup allowance ≤250 MiB. Do not duplicate the 1.7-GiB module closure on ORQ1.

## Gate decision

`BLOCKED_PENDING_GTL_ACCEPTANCE_AND_UNBLOCKS`: grant exact CloudFormation access, stage pinned `asm-exec` on ORQ1, and return independent PASS for the immutable V2 package. Secret/KMS mutation, DB bootstrap, login and Serve changes cannot begin until applicable blockers are cleared.

## Post-report validation result

- `cfn-lint 1.46.0 -r sa-east-1`: **PASS**, exit `0`, `0` findings.
- Template: `.deploy-control/p0/evidence/orq17-bootstrap-owner-secret-stack.yaml`
- Template SHA-256 before any deployment: `7c6d2b547a071cd8d7ad06437a80055a3879ee2764caa4fc45cccaf68a60c582`.
- Authorized `CreateStack` was attempted after validation and failed at IAM authorization before resource creation; exact denial is recorded in §3.
