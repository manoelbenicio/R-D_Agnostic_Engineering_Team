# Reconciled Local RC Evidence

## Historical authority and source parity

- Pre-ORQ-114 OpenSpec baseline: `/home/ec2-user/workspace/worktrees/spe6-runtime-schema`.
- Pre-ORQ-114 authority document SHA-256: `cd156e4db5ac14854c124ca5d86cf0428ce002c2200209bdea88c873d4e34013`; it does not validate the amended ORQ-114 bytes.
- Authoritative app archive SHA-256: `07ff862254a097cc2529216141b733ccedc1566779ec1acdc6266e06fba080b6`; sidecar verification 3189/3189.
- Superseding source manifest 59/59, repairs inventory 21/21, allocator supplement 2/2.
- Final local source parity and post-parity integrity: PASS.

## Migration and generated-source pins

- Migration 128 Git-blob SHA-1: up `4b0894920069336efc36db645b05fae775a3b130`; down `a2b2ea2b56d10f57fe0144bc6998a88277039643`.
- Migration 129 up SHA-256: `6279b861de52797e0686a1547ef2dae0b5cc33a4a9e87e9dc1ec704ea827270c`.
- Migration 135 up SHA-256: `f08d4d4a5eb894a340b72e12e830d10ddb890ce9f010677e363aa426c6672b80`.
- `task_usage.sql`: `44b912e2a8bdb5a7989d2490a1d99b3e913083b0777e7ef189e0c91f5a3a0d10`.
- sqlc v1.31.1 output: `models.go` `79e81bb2abcd8d98a0cd01ef5a42c32b63b94df2d283a35adcae284bef449acb`; `runtime_manager.sql.go` `a0b5ac3022aa722b2ac3db85a6086634e57329e811f60c4e4a4a74507862b896`.

## Historical behavior evidence

- Before ORQ-114, clean PostgreSQL migration through 135, reversible 134/133/132 verification, exact migration-131 refusal, Runtime Manager reservation primitives, normal/race tests and vet: historical PASS.
- Before ORQ-114, Runtime Manager API/UI, core/views tests and typechecks, production web build, Go compile/build/vet, allocator harness and exact SPE-7 suite: historical baseline PASS.
- Stable allocator-v2 source requires canonical agent UUID plus subscription fingerprint, fails before allocation, removes ephemeral/random fallback, and preserves live references.
- N0v2 pricing preserves `TaskUsage.price_version`, `TaskUsage.computed_cost_usd` and `SchemaMigration`.
- Runtime families preserve AGY tier-in-model-ID and structured Kiro/Codex thinking levels.

## ORQ-114 authority and dependency state

- Owner authority selects `native_credential_home` as the only current solution. OmniRouter/OmniRoute is not a positive or fallback authority.
- Active, draining, referenced, reserved, admissible, unproven, and quarantined homes are preserved.
- Whole-home deletion requires `INACTIVE_PROVEN`, age greater than 24 hours, independently accepted ORQ-96 database/admission/fencing/no-bypass gates, and separate destructive cutover authorization. Age is necessary and never sufficient; catalog metadata lifecycle alone cannot delete.
- ORQ-107 candidate `1800df1bf2c87857a5c000ddda94f84554304ce8` supplies proposed atomic assignment rotation but remains pending independent acceptance and cleanup lock-order integration.
- ORQ-108 is an uncommitted alerting candidate. Alerts and expiry presentation are observability only and cannot prove inactivity or authorize cleanup.
- ORQ-109 candidate `075a3c03b69145ad4302201b6854a6980ab6393a` supplies proposed pathless lifetime/readiness schema only; it remains pending independent audit and has no cleanup executor or production wiring.
- Focused strict validation of `credential-account-home-restoration` after the ORQ-114 authority correction: PASS (`Change 'credential-account-home-restoration' is valid`).
- Residual active-package scan is intentionally not claimed clean: `build-omniroute-agent-brain` still assigns positive sole-router/account/credential authority to OmniRoute. ORQ-114 does not edit that separately owned package, so cross-package handoff/deployment remains fail closed.
- This five-file documentation correction performs no credential-home, runtime, service, timer, database, production, push, or deployment action. Prior validation is not current validation of these bytes.

## Historical candidate validation evidence

See `reconciliation/VALIDATION_RESULTS.md`, `reconciliation/SCENARIO_REVIEW.md` and
`reconciliation/TARGET_CHANGESET.sha256`.
