# P7 Rollback Test Evidence

timestamp_utc: 2026-07-05T05:19:41Z
task: 7.2 Rollback em 1 comando para raw Codex
status: PASS
secrets_present: false

## Scope

Validated in a temporary env-file harness. The test starts with prodex/L2 launch
keys enabled, runs the rollback command once, then verifies the resulting config
selects raw Codex and removes prodex/L2 routing keys.

## Commands

```bash
SMOKE_ALLOW_EXECUTE=1 DEPLOY_OWNER_APPROVED=true \
  bash scripts/smoke/rollback-smoke.sh --execute

bash scripts/smoke/rollback-smoke.sh --dry-run
```

The actual rollback command exercised by the smoke:

```bash
ROLLBACK_ALLOW_EXECUTE=1 ROLLBACK_TARGET_ENV=smoke \
  bash scripts/deploy/rollback-to-raw-codex.sh \
  --env-file <temporary-env-file> \
  --codex-path <temporary-raw-codex> \
  --execute
```

## Results

- `rollback-to-raw-codex.sh`: PASS.
- `rollback-smoke.sh --execute`: PASS.
- `rollback-smoke.sh --dry-run`: PASS.

## Assertions Covered

- One command rewrites launch config to:
  - `MULTICA_CODEX_PATH=<raw codex executable>`;
  - `MULTICA_PRODEX_ENABLED=0`;
  - `MULTICA_L2_ENABLED=0`.
- prodex/L2 routing keys are removed from the rolled-back env file.
- Raw Codex executable is invoked successfully after rollback.
- Roll-forward recovery path is verified by restoring the rollback backup and
  confirming prodex launch keys are present again.

## Evidence Hygiene

- Only temporary fake `codex`/`prodex` executables were used.
- The temporary bearer token and temp paths are not recorded.
- No production env file was modified.
