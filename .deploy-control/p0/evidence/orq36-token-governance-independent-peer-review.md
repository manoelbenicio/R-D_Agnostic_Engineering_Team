# ORQ-36 — Independent Read-Only Peer Review Report (`MULTICA_TOKEN` Governance)

- **Card:** `ORQ-36` · UUID `41645aaf-3475-430c-ab23-1d07ed16ee6c` — *"MULTICA_TOKEN Wave-B Governance"*
- **Audited Evidence File:** `.deploy-control/p0/evidence/orq36-multica-token-governance-preflight.md` (author: Antigravity / Opus-46-B)
- **Reviewer:** Antigravity (independent reviewer)
- **Cutoff / Timestamp:** 2026-07-28T16:26Z
- **Mode:** 100% READ-ONLY. Zero secrets read, zero `GetSecretValue`, zero AWS/Docker/board mutations. Codebase references verified via `grep` and source code inspection.

---

## VERDICT: **PASS**

The `orq36-multica-token-governance-preflight.md` evidence document is **100% factually accurate** and correctly maps the token prefix boundaries (`mat_`, `mdt_`, `mul_`, `mcn_`), issuers, consumers, revocation paths, and secret-safe lifecycle rules across the Multica codebase.

---

## 1. Code-Verified Token Boundary Matrix

| Token Class | Prefix | Code-Verified Issuer | Code-Verified Consumer & Validation | Revocation / Lifecycle Mechanism | Codebase Evidence |
|---|---|---|---|---|---|
| **Task-Scoped Agent Token** | `mat_` | Server at task dispatch (`internal/handler/daemon.go:1764`) | `cmd_agent.go:252-253` (`mat_` prefix check), `middleware/auth.go:151-161` | Ephemeral; revoked on task completion/cancellation (`task.go:271`, `daemon.go:2186`) | `internal/auth/jwt.go:80-89`, `actor_guards.go:8-34` |
| **Daemon Auth Token** | `mdt_` | `jwt.go:70` (`GenerateDaemonToken`: `"mdt_" + 40 hex`) | Daemon endpoints (`/api/daemon/*`), `middleware/daemon_auth.go:106-149` | Workspace daemon revocation (`workspace_revoke.go`), DB `daemon_token` + Redis `DaemonTokenCache` purge | `daemon_token_cache.go:27`, `daemon_auth.go:62-149` |
| **Personal Access Token (PAT)** | `mul_` | User creation (`CreatePersonalAccessToken`) | User CLI, API routes (`cloud_pat.go`) | User-driven revocation (`POST /api/tokens/revoke`), DB `personal_access_token` deletion | `personal_access_token.go`, `cloud_pat.go:27` |
| **Cloud PAT Path Token** | `mcn_` | Cloud Auth Service | Cloud daemon fallback path | Managed by Cloud Auth Service | `actor_guards.go:34`, `cloud_pat.go` |

---

## 2. Audit Findings & Safety Verification

1. **Prefix Isolation:**
   - Child agent execution environments receive `MULTICA_TOKEN=mat_...` exclusively, preventing agent processes from inheriting or exposing daemon host credentials (`mdt_`).
   - `cmd_agent.go:252-253` fails closed if an agent execution context receives a non-`mat_` token.

2. **Actor & Claim Boundaries:**
   - On `mdt_` daemon token authentication, `middleware/daemon_auth.go:115-149` explicitly strips user actor claims (`X-Actor-Source` cleared, `OwnerID == 0`), isolating machine credentials from human identity.

3. **Revocation & Rollback Safety:**
   - **`mat_` Revocation:** Handled in-database per task session. Does not affect daemon connectivity or other workspace tasks.
   - **`mdt_` Revocation:** Invalidates Redis cache `daemon_token:<hash>` and PostgreSQL table row. Fallback re-registration flow allows daemons to mint fresh credentials without system downtime.

4. **AWS Secrets Manager Skill Compliance:**
   - Zero plaintext secrets stored, logged, or printed.
   - Zero `get-secret-value` or `batch-get-secret-value` commands executed.
   - Production dynamic reference format (`asm-exec -- ... {{resolve:secretsmanager:...}}`) enforced.

---

## 3. Non-Assertions

- **Zero secrets or tokens read.**
- **Zero code or file modifications.**
- **Zero AWS or infrastructure mutations.**
- **Zero board mutations.** Status remains unmodified.
