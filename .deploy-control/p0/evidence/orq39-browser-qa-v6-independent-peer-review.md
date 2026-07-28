# ORQ-39 V6 — independent adversarial peer review

**Workflow SHA:** `5ee1dabb24b10be50ef5d5b7b2932db332a1ede280f037c6c635de90ddf6d576`
**Evidence SHA:** `27b6ac8a3ebb30b9b3c7e609763b18efe497d43b8afdbf76e2343cfd9d9196ea`

**Verdict: BLOCK for push/execution.** The workflow is materially improved and
its static controls are coherent, but the exact six E2E specs required by its
own trigger package are absent from the worktree. It must not be pushed until
those specs are supplied, reviewed and included in the same seven-path commit.

## Six V4 blockers

1. **cwd/env guard — PASS with scope gate.** The root guard explicitly uses
   `github.workspace`; normal build/test steps intentionally use
   `multica-auth-work`. It rejects root and nested env files and credential-like
   files. The exact package gate prevents unrelated paths.
2. **PostgreSQL health — PASS.** The service has immutable pgvector digest and
   `pg_isready` health options; GitHub waits for a healthy service before steps.
   The migration step also constrains the DSN host to `postgres`.
3. **Process readiness/propagation/cleanup — PASS statically.** Backend and web
   are separate `setsid` process groups, readiness checks prove process state
   and HTTP endpoints, `wait -n -p` catches first death, final probes close the
   simultaneous-exit race, and EXIT/INT/TERM cleanup is bounded.
4. **Immutable supply chain — PASS.** Checkout/setup-go actions and Playwright/
   pgvector images use full SHA-256 pins. Corepack proves pnpm 10.28.2; Go
   1.26.1 is asserted. No mutable action/image tag is present.
5. **Trigger/permissions — PASS with execution gate.** Only push to
   `ci/orq39-browser-qa` is configured; no PR/dispatch trigger. Job/workflow
   permissions are contents-read, actions-none, id-token-none, and checkout
   credentials are not persisted. The branch remains ephemeral by policy.
6. **Anti-skip/package/verification/timeout — BLOCK on missing specs.** The
   exact seven-path delta and six explicit spec paths are enforced; JSON parsing
   rejects missing files, absent tests, skips, fixmes/unexpected/flaky results,
   malformed reports and service death. The fixture reads the real
   `verification_code.code` row after `/auth/send-code`, with the dev-code
   override absent. Timeout is bounded at 25 minutes and Playwright has
   `--max-failures=5`. However, all six required specs are currently absent,
   so no trigger commit can satisfy the package gate.

## ORQ26 and integration boundary

The worktree status contains only the local workflow/evidence preparation; no
ORQ26 workflow or lock is changed. No workflow run, migration, container,
remote, push, PR or board action was performed. The workflow is correctly
designed to fail before installation if the six specs are missing.

## Exact next action

Keep the workflow unpushed. Have the owners of the six specs deliver and review
the exact files, then assemble one seven-path commit and rerun static YAML/shell
and package checks. Only after an independent review of that combined package
may a separate owner authorize the single push-triggered run.
