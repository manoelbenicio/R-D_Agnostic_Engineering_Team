# Remote Recovery Snapshot — 2026-07-19

## Purpose

This snapshot preserves the current multi-stream work in progress against data loss. It is a recovery checkpoint, not an acceptance, promotion, release, or claim that every included change passes its delivery gates.

## Baseline and scope

- Baseline branch: `main`
- Baseline commit: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- Recovery branch: `backup/wip-snapshot-20260719T153042Z`
- Scope before exclusions: 477 staged, unstaged, and untracked file entries
- Included: legitimate source, tests, OpenSpec artifacts, planning/evidence, reports, project tooling, and incomplete lanes required for loss prevention

## Mandatory exclusions

The following local, secret-bearing, generated, or Windows-reserved artifacts are intentionally excluded:

- `opencode.json`
- `opencode.json.backup.20260718-110331`
- `files.txt`
- `nul`
- `multica-auth-work/NUL`
- `multica-auth-work/server/NUL`
- Five `reports/status-360-2026-07-19/report.html.tmp-*.verification-failure.png` artifacts

Ignored environment files are not force-added. In particular, `.env.bak`, `.env.bak-agentsetup`, and `deploy/observability/.env` remain local and excluded.

## Safety and governance result

- Path, filename, size, and signature-based credential audits were completed before staging.
- The two excluded OpenCode configuration files contain literal API-key values.
- No remaining candidate file exceeded 1 MiB or produced a high-confidence live credential finding in the independent audit.
- A second staged-only Gitleaks scan reported 37 signature matches; path/rule review confirmed they are synthetic credential strings in redaction/security tests and their audit evidence, not production credentials.
- `git diff --cached --check` reports pre-existing whitespace defects in mixed WIP artifacts. They are preserved for recovery and remain a promotion gate; they are not accepted as release quality by this snapshot.
- Known mixed/incomplete work remains on hold for later clean, atomic promotion.
- Prodex remains deferred as a final, default-off cold recovery lane; this snapshot does not activate it.

## Recovery policy

Do not merge this branch wholesale into `main`. Reconstruct accepted changes as clean atomic commits/branches after OpenSpec, observability, test, and ownership gates pass.
