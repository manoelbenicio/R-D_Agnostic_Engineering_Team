# DEV Transition Documentation Index

Snapshot date: 2026-07-19 23:27 America/Sao_Paulo / 2026-07-20 02:27 UTC

This directory is the entry point for resuming the repository and DEV environment transition. Read the documents in the order below before changing source, credentials, Docker state, OpenSpec checkboxes, planning status, branches, tags, or running agents.

## Authoritative reading order

1. [`DEV_RESTART_DOSSIER_20260719.md`](DEV_RESTART_DOSSIER_20260719.md) — complete AS-IS, TO-BE, source topology, completed work, validation, task status, decisions, blockers, risks, and next actions.
2. [`DOCKER_AND_REDIS_INVENTORY_20260719.md`](DOCKER_AND_REDIS_INVENTORY_20260719.md) — every Docker container observed on the host, ownership boundaries, networks, ports, volumes, image identity, Redis status, and persistence risks.
3. [`SECRETS_AND_ACCESS_REGISTER_20260719.md`](SECRETS_AND_ACCESS_REGISTER_20260719.md) — service identities, authentication modes, secret variable names, storage locations, permissions, rotation requirements, and prohibited handling.
4. [`FRESH_ENV_RESTART_RUNBOOK.md`](FRESH_ENV_RESTART_RUNBOOK.md) — exact clone, verification, secret bootstrap, build, restore, deployment, health, agent-resumption, rollback, and evidence commands.
5. [`DEV_HANDOFF_20260719.md`](DEV_HANDOFF_20260719.md) — original zero-loss freeze and deployment handoff. This remains historical evidence; the dossier above is the expanded current authority.

## Source-of-truth precedence

When records disagree, use this order:

1. Git object IDs and immutable tags.
2. Current OpenSpec task checkboxes in `openspec/changes/*/tasks.md`.
3. Latest append-only entries in `.planning/agent-brain-v3/AGENT_LEDGER.md` and `.planning/agent-brain-v3/EVIDENCE_INDEX.md`.
4. The transition dossier and infrastructure inventory in this directory.
5. Summary fields near the top of historical planning files.
6. Older narrative entries, chat transcripts, pane output, or remembered status.

The Agent Brain task file currently contains 53 checked tasks out of 96. Some earlier prose in `.planning/agent-brain-v3/STATE.md` and the OBS task introduction still says 51/96 because those passages predate the owner-approved closure of tasks 0.1 and 0.7. The latest ledger entry and actual checkboxes establish 53/96 as authoritative.

## Non-negotiable transition rules

- Never commit a password, token, cookie, provider credential, private key, connection string, or secret-bearing `.env` file.
- Never print secret values into agent output, logs, screenshots, evidence, or documentation.
- Never delete, force-push, rebase, or rewrite a `dev-freeze-*`, `dev-deploy-*`, or `backup/dev-transition-*` recovery point.
- Never merge directly to `main` until the documented security, observability, live-route, capacity, and owner gates are closed.
- Never activate Prodex and OmniRoute as simultaneous hot routers.
- Never authorize login, logout, account reset, key rotation, credential mutation, or session replacement from an agent. Those actions are owner-only.
- Never treat a running container as proof that its source, configuration, data, or secrets can be recreated.
- Never resume parallel agents before assigning disjoint ownership and recording the task IDs, branch, files, evidence contract, and stop conditions.

## Immediate resume pointer

The next authorized engineering work starts from remote branch `integration/dev-transition-candidate-20260719`. The deployed application snapshot is immutable tag `dev-deploy-20260719-candidate` at commit `6a2aba3550aaf6b0468a37bfdf2f00c7faaae084`. Documentation commits after that snapshot do not change the deployed binaries.

The new DEV stack is Docker project `multica-dev-transition`:

- Frontend: `http://127.0.0.1:13100`
- Backend health: `http://127.0.0.1:18080/health`
- PostgreSQL: `127.0.0.1:15433`

The stack is safe for continued isolated DEV work. It is not approved for production cutover.
