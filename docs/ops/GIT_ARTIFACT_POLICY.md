# Git artifact policy

This repository treats generated session/audit exports, root Windows-drive-like paths, and unapproved large blobs as publication hazards. Restricted custody material is local rollback/audit evidence only. It must not be published, used, reused, deleted, or assigned an expiry without separate owner and data-owner authority.

## Size tiers

The gate uses decimal bytes exactly:

- `size > 100000000`: unconditional rejection. An allowlist entry cannot override it.
- `size >= 10485760`: rejection unless `.git-large-files.allow` has the exact path, decimal size, and Git blob OID.
- `size >= 5242880`: warning.
- Exactly `100000000` bytes remains in the exact-allowlist tier by design.

Allowlist rows are `path<TAB>bytes<TAB>blob_oid`. Paths are repository-relative and OIDs are lowercase SHA-1 Git object IDs.

## Enforcement

- `.githooks/pre-commit` scans introduced or changed staged blob identities.
- `.githooks/pre-push` receives Git's target remote name and scans commits newly reachable relative only to `refs/remotes/<target>/*`. Missing remote mapping or target baseline fails closed; refs from another remote cannot suppress scanning.
- Root CI scans `trusted-baseline..tip`: the PR base SHA for pull requests, the before SHA for existing-branch pushes, and the merge base with the fetched default branch for new branches. A missing default ref or merge base fails closed, and inherited history before the baseline is not rescanned.
- The rollback-only source ref `refs/heads/task16-rc-16bfcb4` is denied.
- Known restricted omission blob identities are denied in every outgoing range.
- Root session/audit export names and files under designated `export`, `exports`, `session`, `sessions`, or `audit-exports` directories are rejected regardless of size. Structured repository evidence under `.deploy-control` is not an export and remains permitted.

The hook files are repository source only. `make -C multica-auth-work hook-install` is the documented local provisioning command for `core.hooksPath=.githooks`; it must be run only under separate configuration authorization. This change does not execute that target or modify local, user, or system Git configuration.

## Publisher boundary

A construction candidate that retains rollback refs or quarantined omission objects is `NEVER_PUBLISH`. Only a separately validated clean single-branch publisher clone with no rollback ref, omission objects, alternates, or hardlinks may be considered for a future push. That future action still requires separate configuration, security/data-owner, and exact-OID authorization, an explicit `refs/heads/task16-m3-robust:refs/heads/task16-m3-robust` refspec, and revalidation of the installed pre-push hook. `--all` and `--mirror` are forbidden.

## Restricted M3 custody

The omitted JSON artifacts are classified `RESTRICTED_SESSION_AUDIT_POTENTIALLY_SENSITIVE_AND_POTENTIALLY_LIVE`. The omitted DOCX artifacts are `RESTRICTED_BUSINESS_DATA_POTENTIAL_PII`. Scanner categories remain `UNCERTAIN_POTENTIALLY_LIVE`.

No later push, pull request, merge, deployment, or credential use is authorized by this policy. Before any later push, separate security and data-owner approval must either evidence rotation/revocation or explicitly classify the findings as non-live. Publication, content reuse, and value inspection remain prohibited.

## Idle autostop source

`scripts/ops/agent-idle-autostop.sh` performs read-only Git queries and supports linked worktrees through Git's own directory discovery. Dirty, untracked, detached, no-upstream, unpushed, or indeterminate state vetoes poweroff. This repository does not install the script, a service, or a timer.
