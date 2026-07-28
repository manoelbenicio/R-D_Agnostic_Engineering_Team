# ORQ-26 — GitHub authentication unblock runbook (human-only, one retry)

- Issue: **ORQ-26**
- Prepared by/pane: `Codex56#B` / `w7:p4`
- Prepared: `2026-07-27T13:39:51Z`
- Mode: READ-ONLY preparation. No login, token read/paste, push, workflow dispatch, rerun,
  PR, merge, or cleanup was performed.
- Current local target: worktree `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate`, branch
  `ci/orq26-db-gate`, commit
  `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee`, parent
  `0cb8aebb5aff79cb430b3740d22fadc53c0116fd`.
- Remote: `https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git`.
- Existing failure: the first authorized push exited 128 locally with
  `could not read Username for 'https://github.com'`; no remote branch or Actions run exists.

## Safety boundary

Only the human owner/operator may execute the authentication and retry steps below. The agent
must not receive, print, paste, put in an argument, put in an environment variable, log, diff,
or persist any token, OAuth code, cookie, SSH private key, or credential-store value. Never use
`gh auth status --show-token`, `gh auth token`, `gh auth login --with-token`, `--insecure-storage`,
`GH_TOKEN`, `GITHUB_TOKEN`, shell tracing (`set -x`), or a command transcript containing secrets.

The safe default is the browser flow. GitHub CLI documents that the default web flow stores the
credential in the system credential store when available, but can fall back to plaintext when no
store is available; if that fallback is reported, **STOP** and provision a secure credential store
instead of continuing. See the [official `gh auth login` manual](https://cli.github.com/manual/gh_auth_login).

## 1. Human preflight (read-only)

Run from the temporary worktree, with no edits:

```bash
cd /home/ec2-user/workspace/worktrees/ci-orq26-db-gate
git status --short --branch                 # must be clean on ci/orq26-db-gate
test "$(git rev-parse HEAD)" = \
  d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee
git diff --check HEAD^ HEAD
git ls-remote --get-url origin              # must be the repository above
git ls-remote --exit-code --heads origin ci/orq26-db-gate
```

The last command is expected to exit `2` before the retry because the remote temporary branch is
absent. Any local diff, changed HEAD, unexpected remote, or existing remote branch is a STOP.

## 2. Recommended human authentication

Use HTTPS and the browser/device flow; do not supply a token on stdin or in a command line:

```bash
gh auth login --hostname github.com --git-protocol https --web
gh auth setup-git --hostname github.com
```

`gh auth login` documents `--web` as the browser option; `gh auth setup-git` configures Git to use
GitHub CLI as its credential helper. The human completes the browser/device approval privately.
The CLI manual states that `gh auth setup-git` fails when the host is not authenticated, which is
useful as a fail-closed checkpoint. References:

- [`gh auth login`](https://cli.github.com/manual/gh_auth_login)
- [`gh auth setup-git`](https://cli.github.com/manual/gh_auth_setup-git)

Do not add `--scopes` casually. The CLI manual's `--with-token` section lists classic-token
minimums (`repo`, `read:org`, `gist`) for that separate input mode; that mode is deliberately not
used here. Do not use `--clipboard` on this host unless the human explicitly wants the one-time
device code copied locally and it never enters agent context.

## 3. Pre-retry verification (metadata only)

The human verifies identity and storage without revealing a token:

```bash
gh auth status --hostname github.com
gh auth status --hostname github.com --json hosts \
  --jq '.hosts[] | {host,activeUser,gitProtocol}'
git config --get-regexp '^credential\.' || true
git ls-remote --exit-code --heads origin ci/orq26-db-gate
```

Accept only an authenticated account with write access to this exact repository and an HTTPS Git
protocol. The status command must not include `--show-token`; the official manual explicitly
documents that flag as printing the token. A successful authenticated `ls-remote` proves the
credential helper can reach GitHub, but does not authorize a push by itself. The expected branch
probe remains exit `2` until the push.

## 4. Public-repository and workflow permission reconciliation

There are two separate permissions:

1. **Human Git reference creation.** The account must be the repository owner or a member/collaborator
   with write permission. GitHub documents that fine-grained tokens can be restricted to one
   repository and permissions, but a fine-grained token cannot write to a public repository that is
   not owned by the user or an organization of which the user is a member. For this remote, the
   human must confirm that ownership/membership condition; do not infer it from the URL alone.
2. **Workflow job token.** The committed workflow intentionally declares `permissions: contents: read`.
   GitHub's workflow syntax says unspecified permissions become `none`, and `contents: read` is
   enough for checkout/read operations. The job does not need `contents: write`, `actions: write`,
   `issues: write`, or secrets. Repository/organization Actions settings may still disable Actions
   or restrict allowed actions; those settings are independent of the human Git push.

The `.github/workflows/orq26-db-gate.yml` path has an additional restriction. GitHub's contents API
documentation states that classic OAuth/PAT access needs `repo` **and** `workflow` to modify files
under `.github/workflows`; for a fine-grained token it requires repository `Contents: write` and
`Workflows: write` (the exact-repository grant). The official fine-grained permission table also
requires write access for creating Git refs. If browser OAuth cannot create this workflow branch,
the human should use an approved, short-lived fine-grained credential with those exact repository
permissions, or a classic credential with the documented `repo` + `workflow` capability. The agent
must never see or handle either credential.

Sources:

- [Managing personal access tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
- [Create/update repository contents permissions](https://docs.github.com/en/rest/repos/contents)
- [Fine-grained token endpoint permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens)
- [Workflow syntax and `permissions`](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)
- [Repository Actions settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository)

## 5. The single retry (only after sections 1–4 PASS)

This is the one and only authorized retry command:

```bash
cd /home/ec2-user/workspace/worktrees/ci-orq26-db-gate
git push --set-upstream origin ci/orq26-db-gate
```

Capture only exit code, remote commit SHA, and URLs/IDs. If it exits nonzero, if authentication is
requested again, or if the branch does not appear remotely, **STOP**: no second push, amend,
force-push, PR, merge, workflow dispatch, or rerun without a new GTL/owner ruling. A push made by
the human credential is expected to create the `push`-triggered run; the workflow's own
`GITHUB_TOKEN` must not be used to push, because GitHub documents that most events created by that
token do not create a new workflow run.

## 6. Post-push verification and immutable run record

Immediately after a successful push, record metadata without printing environment variables or
tokens:

```bash
git ls-remote origin refs/heads/ci/orq26-db-gate
gh run list --repo manoelbenicio/R-D_Agnostic_Engineering_Team \
  --workflow orq26-db-gate.yml --branch ci/orq26-db-gate --limit 1 \
  --json databaseId,headSha,status,conclusion,url,createdAt,updatedAt
```

Require the run's `headSha` to equal `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee` and wait for a
terminal status. Use the run ID returned by GitHub, not a prediction:

```bash
gh run watch <RUN_ID> --repo manoelbenicio/R-D_Agnostic_Engineering_Team --exit-status
gh run view <RUN_ID> --repo manoelbenicio/R-D_Agnostic_Engineering_Team \
  --json databaseId,headSha,status,conclusion,url,jobs
```

Verify the named job conclusion, all eight leaf-test steps, and zero skipped leaves from the job
metadata/log summary. Do not use `gh run view --log` into an agent-visible transcript; if immutable
logs are required, the human downloads/reviews them privately and records only redacted hashes,
job names, conclusions, and timestamps. If the run does not trigger or is not terminal, STOP under
the same no-rerun rule.

## 7. Credential cleanup and approved B4 cleanup

After terminal run evidence is captured, and only if the existing B4 authorization still applies:

```bash
gh auth logout --hostname github.com --user <HUMAN_ACCOUNT>
gh auth status --hostname github.com    # expected unauthenticated/nonzero
git ls-remote --exit-code --heads origin ci/orq26-db-gate
git push origin --delete ci/orq26-db-gate
git worktree remove /home/ec2-user/workspace/worktrees/ci-orq26-db-gate
git branch -D ci/orq26-db-gate
```

The `gh auth logout` manual states that logout removes the local stored authentication configuration
but does **not** revoke tokens; if the human wants revocation, use GitHub Settings → Applications →
GitHub CLI and revoke it privately. Remove only the exact GitHub CLI credential-helper entry if
needed; do not wipe unrelated global Git credentials. Before B4, prove the remote branch exists and
after B4 prove it is absent. Preserve `/home/ec2-user/workspace/worktrees/gtl-orq26` and the durable
ORQ-18 source worktree; do not delete, reset, or checkout either source worktree.

Reference: [`gh auth logout`](https://cli.github.com/manual/gh_auth_logout).

## Stop conditions and evidence checklist

Stop immediately on any of the following: secure credential store unavailable; account/owner or
write permission mismatch; unexpected HEAD/diff/remote; authentication prompt during push; nonzero
push; no branch; no workflow run; nonterminal run; failed job; skipped leaf; missing run SHA; secret
appearing in output; or any request to rerun/force-push/merge. A stop produces a blocked evidence
record, not another attempt.

The human handoff must contain: authenticated account name (not token), exact local commit, remote
commit SHA, run ID and URL, terminal job conclusion, eight leaf conclusions and skip count, redacted
log/evidence hashes, cleanup receipts, and confirmation that no secret entered agent context.
