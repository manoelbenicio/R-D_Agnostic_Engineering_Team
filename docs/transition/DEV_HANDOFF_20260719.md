# DEV Transition Handoff — 2026-07-19

> Expanded authority: start with `docs/transition/README.md`, then read the complete restart dossier, Docker/Redis inventory, secrets/access register, and fresh-environment runbook linked there. This file preserves the original zero-loss handoff and deployment evidence.

## Executive disposition

- Source preservation: COMPLETE. Every discovered remote, uncommitted, and local-only source state has a remote ref.
- Canonical release candidate: ASSEMBLED and pushed at `integration/dev-transition-candidate-20260719`.
- New DEV deployment: RUNNING from commit `6a2aba3` under Docker project `multica-dev-transition`.
- Cutover: NO-GO for production. The isolated DEV deployment is green, but supplier pinning, security evidence, Agent Brain live-provider acceptance, and owner decisions remain open.
- Migration method: blue/green. Keep the old stack available only as rollback until the new DEV stack passes acceptance.

## Immutable source inventory

| Purpose | Remote ref | Commit |
|---|---|---|
| Current main baseline | `origin/main` | `b657129` |
| Deployed DEV candidate snapshot | `dev-deploy-20260719-candidate` | `6a2aba3` |
| Agent Brain integrated P0 | `origin/integration/agent-brain-p0` | `29056e5` |
| Agent Brain planning/governance | `origin/planning/agent-brain-observability-freeze` | `043e641` |
| Cline OmniRoute adapter | `origin/work/agent-brain-w3-cline-omniroute` | `3f73594` |
| NIM OmniRoute correction | `origin/topic/agent-brain-p0-nim-omniroute-delta` | `f1c79f7` |
| OmniRoute health correction | `origin/topic/agent-brain-p0-omniroute-health` | `46abd8a` |
| Antigravity resolver correction | `origin/topic/antigravity-agy-1.1.4-resolver` | `7735bdc` |
| New DEV pre-sync NIM recovery | `origin/backup/dev-transition-home-wip-20260719T231359Z` | `3ca8dca` |
| Old environment credential-isolation WIP | `origin/backup/dev-transition-old-main-wip-20260719T231359Z` | `d29ff1c` |
| Five formerly local-only observability commits | `origin/backup/dev-transition-local-w3-20260719T231359Z` | `0ba88da` |
| Disposable integration recovery head | `origin/backup/dev-transition-disposable-integration-20260719T231359Z` | `54910a8` |

No branch above may be deleted, force-pushed, or rebased until transition acceptance is signed off.

## Source-of-truth hierarchy during transition

1. `integration/dev-transition-candidate-20260719` — canonical reconciled DEV candidate.
2. `transition/dev-handoff-20260719` — original transition manifest and recovery instructions.
3. `planning/agent-brain-observability-freeze` — planning, OpenSpec, decisions, risks, and evidence to reconcile into the candidate.
4. Topic/work branches — reviewed deltas integrated one at a time, never bulk-overwritten.
5. `main` — stable baseline; remains unchanged until the candidate passes all gates.
6. `backup/dev-transition-*` — immutable recovery evidence; never used as an automatic merge source.

## Mandatory reconciliation order

1. Create `integration/dev-transition-candidate-20260719` from `29056e5`.
2. Merge the planning branch with file-by-file conflict resolution; do not use global `ours` or `theirs` strategies.
3. Reconcile OpenSpec counters and dispositions before changing implementation task status.
4. Integrate the OmniRoute health, NIM, Antigravity, and Cline topic branches one at a time.
5. Compare the recovered observability branch against the already-integrated W5 history; retain only independently reviewed, non-duplicate deltas.
6. Compare credential-isolation recovery WIP against the candidate. The Kiro wrapper contains an old `/mnt/c` path and must be parameterized before promotion.
7. Keep the earlier new-DEV NIM recovery branch as evidence unless review proves a required behavior absent from the newer implementation.

## Recovery branch disposition after candidate assembly

- `backup/dev-transition-local-w3-20260719T231359Z`: ARCHIVE. Its `observability/e2e` tree is byte-equivalent to the accepted W5 tree already present in the candidate; merging it would duplicate history without changing source.
- `backup/dev-transition-old-main-wip-20260719T231359Z`: ARCHIVE/REVIEW-ONLY. The candidate contains a substantially newer credential-isolation implementation. The additional Kiro wrapper hard-codes the retired `/mnt/c` source path and is not portable to the new DEV environment.
- `backup/dev-transition-home-wip-20260719T231359Z`: ARCHIVE/REVIEW-ONLY. It is an earlier divergent NIM implementation based on `610c847`; the candidate contains the later mainline NIM implementation plus the OmniRoute correction.
- `backup/dev-transition-disposable-integration-20260719T231359Z`: ARCHIVE. It is a disposable intermediate integration head and is superseded by the canonical candidate.

These dispositions preserve every commit remotely while preventing obsolete recovery trees from silently replacing newer accepted code.

## Hard acceptance gates

- Git: clean candidate; all expected refs fetched; no local-only commits.
- OpenSpec: strict validation; task counts agree with `STATE`, `ROADMAP`, evidence, and decisions.
- Security: no provider credentials in child environments, argv, logs, traces, images, or repository history.
- Router ownership: OmniRoute is the only hot router; Prodex is default-OFF cold recovery and mutually exclusive.
- Supplier pinning: replace `omniroute:latest` with an approved digest before cutover.
- Backend: focused Go tests, race tests for changed concurrency paths, vet, build, migrations up/down.
- Frontend: install from lockfile, typecheck, unit tests, production build.
- Runtime: model discovery, streaming, tools, cancellation, retry-before-output, no replay-after-output, affinity, and deterministic failure classification.
- Observability: continuous metadata-only trace; structural secret/content leakage scan clean.
- Operations: health/readiness, database reachability, backup/restore, kill switch, rollback, and restart-from-clean-clone.

## Validation and deployment evidence

- Git integrity: local candidate and `origin/integration/dev-transition-candidate-20260719` both resolved to `6a2aba3550aaf6b0468a37bfdf2f00c7faaae084` at deployment time; the working tree was clean.
- OpenSpec: strict validation passed for `build-omniroute-agent-brain` and `persist-prodex-runtime-integration`; Agent Brain implementation progress is `53/96`.
- Backend: Go 1.26.1 container tests passed for the full `internal/daemon` package plus focused Agent Brain, gateway, runtime environment, observability, and agent packages; focused `go vet` passed.
- Frontend: frozen-lockfile install, workspace typecheck, and production build passed.
- Images: `multica-backend:transition-6a2aba3` and `multica-web:transition-6a2aba3` built from the candidate checkout.
- Isolation: Docker project `multica-dev-transition` uses dedicated network, volumes, images, and loopback ports `15433`, `18080`, and `13100`.
- Database: fresh isolated PostgreSQL 17 initialized successfully and migrations completed through migration `126`.
- Runtime probes: candidate backend `/health` returned HTTP `200`; candidate frontend `/login` returned HTTP `200`.
- Non-regression: the old backend on `8080` and frontend on `3100` continued returning HTTP `200`; no old container, volume, or port was replaced.
- Runtime secrets: generated outside the repository in owner-only `/home/dataops-lab/.config/multica-transition/dev.env`; the directory is mode `700`, files are mode `600`, and no credential was committed or printed.

## DEV stack operations

Run these commands from `multica-auth-work`. They operate only on the isolated `multica-dev-transition` Docker project.

```bash
docker compose -p multica-dev-transition \
  --env-file /home/dataops-lab/.config/multica-transition/dev.env \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f /home/dataops-lab/.config/multica-transition/images.yml \
  up -d
```

Use the same prefix with `ps`, `logs`, `restart`, or `down`. Do not add `--volumes` to `down` unless destruction of the isolated DEV database is explicitly authorized.

## Blue/green deployment sequence

1. Build only from the clean candidate under `/home/dataops-lab`.
2. Use separate project names, ports, volumes, and secret files for the new DEV stack.
3. Restore a scrubbed database copy or create a fresh DEV database; never point unvalidated code at the old writable database.
4. Apply and reverse migrations in the DEV database before startup.
5. Start infrastructure, then OmniRoute, backend, frontend, and observability.
6. Run synthetic/offline acceptance first, followed by explicitly authorized live-provider checks.
7. Prove rollback to the old stack or baseline commit in one documented command.
8. Cut over only after acceptance evidence is committed and owner/security gates are resolved.
9. Stop and archive the old stack only after the agreed observation window.

## Fresh-clone recovery

```bash
git clone https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
cd R-D_Agnostic_Engineering_Team
git fetch --all --tags --prune
git branch -r | sort
git switch --create transition/dev-handoff-20260719 --track origin/transition/dev-handoff-20260719
```

Before implementation or deployment, verify every commit in the inventory resolves with `git cat-file -e <sha>^{commit}`.

## Current blockers requiring explicit disposition

- OmniRoute image digest is not pinned.
- Smart Context SC01–SC10 requires implementation evidence or a formal waiver.
- Shared-state topology for higher capacity tiers is undecided.
- Product naming and final sign-offs remain open.
- Historical credential exposure requires completed remediation/rotation evidence.
- Tier-20 resource limits, workload, thresholds, evidence owner, and retention require ratification.

This document preserves work and controls transition sequencing; it does not declare production readiness.
