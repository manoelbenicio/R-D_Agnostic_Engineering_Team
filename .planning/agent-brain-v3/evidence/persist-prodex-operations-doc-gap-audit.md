# Operational Documentation Gap Audit for persist-prodex-runtime-integration (Task 4.5)

## Grade: 4.5 MISSING

## Source Hashes Assessed
- `413029dc068402c7c26b070fec962b67789d21bca348695c1fc4f96e88b9ec63  docs/deploy/l2-sidecar-deploy-plan.md`
- `40a5464a987af22962c96d5199dbc51c761f692829e6f324b64e28682b2c5c0c  docs/deploy/prod-rollout-runbook.md`
- `b27e9f65d1b99df0648c66f8b45a37b1a3f757594054fda82f336dd40a85313c  docs/deploy/prodex-account-enrollment-runbook.md`
- `3f9b95b76f2683bb0a91a9d6a7bc6db939dfb3af3dd43d10977f27a756db0512  multica-auth-work/.env.example`

## Requirement-to-Doc Gap Matrix

| Requirement | Target Doc | Current Content | Missing / Ambiguous Content Gap |
| --- | --- | --- | --- |
| **Separate L2 adapter path** (`MULTICA_L2_SIDECAR_PATH`) | `multica-auth-work/.env.example`, `docs/deploy/l2-sidecar-deploy-plan.md` | Omitted entirely from `.env.example`. Plan only mentions `MULTICA_L2_MODE` and `MULTICA_L2_SIDECAR_TOKEN` (Line 110-111). | The exact environment variable, its purpose, and default path must be documented in both `.env.example` and the deploy plan. |
| **Pinned Prodex binary** (`MULTICA_PRODEX_PATH`) | `multica-auth-work/.env.example`, `docs/deploy/l2-sidecar-deploy-plan.md` | Omitted entirely. | Explicit definition of how to pin the Prodex path independently from the L2 sidecar path. |
| **Required-mode failure** (`MULTICA_PRODEX_REQUIRED`) | `multica-auth-work/.env.example`, `docs/deploy/prod-rollout-runbook.md` | Omitted entirely. | Definition of the flag and instructions on how fail-closed behavior operates when enabled. |
| **Profile reconciliation** (`prodex profile add --codex-home`) | `docs/deploy/prodex-account-enrollment-runbook.md` | Lines 54-75 instruct using `bash scripts/staging/enroll_account.sh` which copies credentials. | Must be updated to use the reference-only `prodex profile add --codex-home` strategy without copying raw credential material. |
| **Config-source / Runtime-auth health & Readiness** | `docs/deploy/prod-rollout-runbook.md` | Lines 94, 95 merely state "Verify Go daemon health" and "Verify prodex liveness". | Missing exact steps on how to verify the new config-source health, effective `rust_l2` authority, and adapter readiness independently. |
| **Rollback** | `docs/deploy/prod-rollout-runbook.md` | Line 181: "raw Codex rollback remains immediately available". | Lacks explicit CLI commands or exact configuration switches to explicitly disable L2 and fallback to raw Codex. |
| **Credential-safe evidence** | `docs/deploy/prodex-account-enrollment-runbook.md` | Line 134 requires `secrets_present=false`. | Missing evidence conventions to prove that the new reference-based reconciliation (`--codex-home`) succeeded without printing or copying secrets. |

## Unsafe Secret Examples Detected in Legacy Docs
- `scripts/staging/enroll_account.sh` copies secrets directly to an ext4 volume (documented in `docs/deploy/prodex-account-enrollment-runbook.md` Line 73: "source credential contents are copied"). This violates the reference-only invariant specified in task 4.5.

## Recommended Exact Doc Owners / Order
1. **Infra / Deployment Owner:** Update `multica-auth-work/.env.example` with `MULTICA_L2_SIDECAR_PATH`, `MULTICA_PRODEX_PATH`, and `MULTICA_PRODEX_REQUIRED`.
2. **Operations Owner:** Rewrite `docs/deploy/prodex-account-enrollment-runbook.md` to deprecate `enroll_account.sh` in favor of `prodex profile add --codex-home`.
3. **Operations Owner:** Update `docs/deploy/l2-sidecar-deploy-plan.md` and `docs/deploy/prod-rollout-runbook.md` with explicit checks for readiness, config-source health, and exact rollback commands.

## Explicit Non-Claims
- This audit did not inspect real credentials, Docker/network/DB/daemon state, or `.env` values.
- No docs, codebase files, or OpenSpec task checkboxes were modified.
- No real deployment topology was touched; solely read-only documentation gap analysis.
