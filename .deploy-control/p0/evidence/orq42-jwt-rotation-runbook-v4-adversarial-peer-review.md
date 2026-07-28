# ORQ-42 — V4 independent adversarial peer review

- Reviewed: `.deploy-control/p0/evidence/orq42-jwt-rotation-runbook-v4-design.md`
- Review time: `2026-07-27T15:44:32Z`
- Mode: READ-ONLY. The AWS Secrets Manager skill was read completely before review.
- Safety: no secret value, credential file, container environment or sensitive override was read.
  No `GetSecretValue`, `BatchGetSecretValue`, SMA access, `asm-exec`, Docker command, container
  action, login, rotation, restart or board mutation was performed.

## Verdict: BLOCK

V4 fixes the Compose spelling and project name, and it states the correct backend-only recreate
flags. It does not provide an executable, secret-safe forward/rollback runbook. The effective
four-file ORQ1 Compose configuration is still wrong/incomplete, the child-process boundary is not
encoded, the anti-literal and HTTP gates are incorrect, and secret metadata presence cannot
currently be proved by this execution identity.

## Adversarial findings

| ID | Result | Evidence and executable correction |
|---|---|---|
| F1 — Compose plugin | **PASS WITH PREFLIGHT** | V4:22-23 correctly uses `docker compose`; the V3 live ORQ1 snapshot recorded plugin `v5.3.1` and no `docker-compose` binary (`orq42-jwt-v3-adversarial-peer-review.md:49-58`). V4 must still require `docker compose version` immediately before the window and stop on failure. |
| F2 — exact four config paths | **BLOCK** | V4:25-29 lists only two paths, both under the ORQ2-style `/home/ec2-user/workspace/...` prefix. The live ORQ1 label snapshot records four files under `/home/ec2-user/R-D_Agnostic_Engineering_Team/...`, plus `/home/ec2-user/.config/multica-transition/{images.yml,backend-env.override.yml}` (V3 review:60-88). V4:36 passes only the first `-f`, so it can lose the pinned image and effective override. Before either recreate, re-read only `com.docker.compose.project.config_files`, require the exact four-item ordered set, and pass all four `-f` arguments. Do not read `backend-env.override.yml` contents. |
| F3 — project/service isolation | **PASS PARTIAL** | `-p multica-dev-transition` and `up -d --force-recreate --no-deps backend` at V4:31-38 are correct. The command is not usable until F2 is fixed. A preflight must also assert label project=`multica-dev-transition` and service=`backend`; labels only, never `.Config.Env`. |
| F4 — forward and rollback | **BLOCK** | V4 says the same recreate applies to both directions (33-38) but supplies only one command and no distinct old/new dynamic reference, version stage, restoration step or rollback acceptance gate. Forward and rollback must each use the same four `-f` files and backend-only flags, but different owner-approved references (for example an explicitly verified current/previous version stage). Rollback must restore the prior reference first, force-recreate the backend, then repeat liveness/readiness/auth gates. |
| F5 — no plaintext in parent shell | **BLOCK** | V4:40-42 is a claim, not an executable boundary. Snippets at 47-78 use `$JWT_SECRET` without showing that the entire temp-file creation, checks, Compose invocation and cleanup are inside the single `asm-exec -- sh -c '...'` child. As written, an operator could execute them in the parent shell. Publish one complete child script; the parent may hold only the literal dynamic reference and non-secret paths/statuses. Never use command substitution in the parent to capture a resolved value. |
| F6 — child-only length gate | **BLOCK** | `printf %s` is correct (V4:53-61), but the detached snippet does not establish child-only execution. The complete child must evaluate `test "$(printf %s "$JWT_SECRET" \| wc -c)" -ge 32` and emit only a boolean/status or length, never the value. |
| F7 — anti-literal effect check | **BLOCK** | V4:44-51 checks only an unresolved value that *starts* with `{{`; it neither matches literal `{{resolve:` anywhere nor proves the resulting container environment. Add a post-recreate effect check that emits only `FOUND`/empty, e.g. a Docker template iterating `.Config.Env` with `contains "{{resolve:"` and never printing entries. Criterion: empty. This is an authorized-window check and was not run here. |
| F8 — private temp/trap | **BLOCK** | `umask 077`, mode `0600`, and single-quoted trap are directionally correct (63-74). The runbook must reject a symlink/non-owned private directory, verify owner and mode, and make signal traps exit after cleanup. `echo "JWT_SECRET=..."` (78) is not robust for arbitrary bytes; use a documented env-file-safe secret encoding/character contract or a safe writer inside the child. Cleanup must cover forward, failure and rollback files. |
| F9 — health/readiness | **BLOCK** | V4:82 checks only `/health`, calls it readiness, targets `localhost:8080`, and uses `curl -sf`, which does not assert exact `200`. Source proves `/health` is liveness and `/readyz`/`/healthz` are readiness (`router.go:442-445`). ORQ1 evidence records host bind `127.0.0.1:18080->8080`. Require bounded polling with connect/total timeouts and exact HTTP code `200` for both `http://127.0.0.1:18080/health` and `/readyz`. |
| F10 — authenticated `/api/me` | **BLOCK** | V4:83-84 uses cookie `session`, but source defines `multica_auth` (`internal/auth/cookie.go:20-23`) and middleware accepts Authorization Bearer first, then that cookie (`middleware/auth.go:329-341`). `curl -f` on the old token cannot distinguish expected `401` from transport/other HTTP failure. Tokens in `-b "..."` also enter argv. Keep old/new credentials in private child-owned files and use a private curl config/cookie jar; assert exact old=`401`, new=`200`, bounded timeouts, and `GET /api/me`. Define how the owner creates the post-rotation token without exposing it. |
| F11 — secret-id presence | **BLOCK** | V4:86-90 names `prod/jwt-secret` but does not fix a JSON key/version stage or acceptance output. A metadata-only `DescribeSecret` attempt in `sa-east-1` returned `AccessDenied`; no value API was called and no value returned. Therefore presence is unverified and the operation must fail closed until an authorized human/MCP identity records metadata-only success. `DescribeSecret` cannot prove a JSON key exists; key/schema requires separate owner attestation without disclosing value. |
| F12 — ORQ2 daemon re-pair | **PASS DIRECTION / BLOCK EXECUTION** | V4:92-95 correctly requires the same operational window and a separate owner-authorized card. It must be a hard precondition before rotation, with the card ID, authorized operator, ready re-pair procedure, heartbeat/auth acceptance, rollback and stop condition recorded. A future unscheduled `ORQ-N` is insufficient. No ORQ2 action is implied by ORQ-42 authorization. |
| F13 — current ORQ1 configuration reconciliation | **BLOCK** | Later read-only ORQ1 evidence records a durable `dev.env` (`0600`), `backend-env.override.yml` referencing it without secret literals, and dedicated recreate/rollback helpers with hashes (`kanban-runtime-diagnosis-20260727.md:353-371`). V4 ignores this post-ORQ-30 state and proposes a new temp `--env-file` path. Before approval, the owner must choose whether to use the existing authorized helpers/durable source or supersede them, verify hashes, and prevent two competing sources of `JWT_SECRET`. No file contents may enter evidence. |

## Minimum executable acceptance contract

1. Fail closed unless the live ORQ1 label-only preflight yields project
   `multica-dev-transition` and the exact four config paths; use those four paths for both directions.
2. Obtain separate written authorizations for secret mutation, ORQ1 backend recreate, rollback, and
   same-window ORQ2 daemon re-pair. Rotation cannot begin until the re-pair procedure is ready.
3. Run one complete child-only secret-resolution script with private temp lifecycle, length check,
   Compose interpolation, backend-only recreate and cleanup; the parent receives only non-secret status.
4. Prove after forward: no literal dynamic reference in the resulting container, exact liveness `200`,
   readiness `200`, old `/api/me` `401`, new `/api/me` `200`, and ORQ2 daemon authenticated/healthy.
5. Prove rollback with the same four Compose files and `--force-recreate --no-deps backend`, then the
   corresponding exact HTTP and daemon gates. Any mismatch is STOP, not an improvised second action.

## Non-claims

- The ORQ1 Compose labels were not re-read live in this review; the latest preserved live snapshot was
  used because login and Docker execution were explicitly out of scope.
- Secret `prod/jwt-secret` was not proved present. Metadata access was denied; no secret value was
  requested, received or inferred.
- This review authorizes no AWS, Docker, container, daemon, login, rotation, restart or board action.
