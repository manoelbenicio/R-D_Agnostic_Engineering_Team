# PLAN 00-02 Evidence — Prodex Daemon Environment

created_at_utc: 20260705T023004Z
status: DONE
depends_on: PLAN 00-01

## PLAN 00-01 Reference

- attestation: `/home/dataops-lab/runtime/prodex-src/attestations/prodex-build-20260705T022506Z.sha256`
- binary: `/home/dataops-lab/runtime/prodex-src/target/release/prodex`
- binary_sha256: `5568ae664e2fa5b776a9e2df813175e57a24dc31c4c1dfbeb029a2d3db8e7758`
- source_commit: `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`

## Daemon Environment File

- env_file: `/home/dataops-lab/runtime/prodex.env`
- env_file_mode: `0600`
- env_file_owner: `dataops-lab:dataops-lab`
- contains_secrets: `false`

Configured non-secret values:

```text
MULTICA_PRODEX_ENABLED=true
MULTICA_PRODEX_PATH=/home/dataops-lab/runtime/prodex-src/target/release/prodex
MULTICA_PRODEX_VERSION=0.246.0
MULTICA_PRODEX_COMMIT=7750da9b6a5c91a6d429e18e6a4d422cab4bc144
PRODEX_HOME=/home/dataops-lab/runtime/prodex-home
```

## Runtime Home

- prodex_home: `/home/dataops-lab/runtime/prodex-home`
- filesystem: `ext2/ext3` as reported by WSL for the Linux home filesystem
- mode: `0700`
- owner: `dataops-lab:dataops-lab`

## Verification

- `MULTICA_PRODEX_PATH` is executable.
- Current binary SHA-256 matches PLAN 00-01 attestation.
- Minimal Go `exec.LookPath(os.Getenv("MULTICA_PRODEX_PATH"))` in `golang:1.26-alpine` resolved:

```text
/home/dataops-lab/runtime/prodex-src/target/release/prodex
```

## Guardrails

- No secrets printed or recorded.
- No daemon live launch executed.
- PLAN 00-03 not started.
