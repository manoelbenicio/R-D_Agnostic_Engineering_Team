# ORQ-40 — Codex CLI alignment completion

Date: 2026-07-29 UTC
Target host: ORQ1 (`ec2-user@100.118.244.61`)
Authorized target: exact package `@openai/codex@0.145.0`
Rollback: `npm install -g @openai/codex@0.144.6`

## Scope

The correction was executed only through BatchMode SSH to ORQ1. ORQ2, project-local and nested `node_modules`, credentials, agents, daemon, backend, database, AWS resources, environment configuration, systemd units, and containers were not changed.

## Measured pre-state on ORQ1

A fresh remote login shell reported:

- Hostname: `ip-172-31-18-217.sa-east-1.compute.internal`
- Node: `v22.23.1`
- npm prefix: `/home/ec2-user/.nvm/versions/node/v22.23.1`
- npm root: `/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules`
- Command: `/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex`
- Realpath: `/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules/@openai/codex/bin/codex.js`
- Binary version: `codex-cli 0.144.6`
- Package version: `0.144.6`

## Mutation

Executed on ORQ1 in its effective global npm prefix:

```text
npm install -g @openai/codex@0.145.0
```

npm completed successfully (`changed 2 packages in 4s`). Immediate readback returned the expected global executable, `codex-cli 0.145.0`, and package version `0.145.0`.

## Independent post-install verification

Two new, separate BatchMode SSH login-shell connections were opened after the installation. Both independently returned the same content-free metadata and passed exact assertions:

```text
hostname=ip-172-31-18-217.sa-east-1.compute.internal
node=v22.23.1
npm_prefix=/home/ec2-user/.nvm/versions/node/v22.23.1
npm_root=/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules
codex_command=/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex
codex_realpath=/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules/@openai/codex/bin/codex.js
codex_binary_version=codex-cli 0.145.0
codex_package_version=0.145.0
```

Result: ORQ1's login-shell-visible global Codex CLI installation is persistently aligned to exact version `0.145.0`.
