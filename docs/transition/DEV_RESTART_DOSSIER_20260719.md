# DEV Restart Dossier — Complete Transition State

Snapshot date: 2026-07-19 23:27 America/Sao_Paulo / 2026-07-20 02:27 UTC

Repository: `https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git`

Canonical transition branch at the beginning of this dossier: `integration/dev-transition-candidate-20260719`

Deployed source snapshot: `dev-deploy-20260719-candidate` → `6a2aba3550aaf6b0468a37bfdf2f00c7faaae084`

## 1. Executive conclusion

### What is safe and complete

- Every discovered local-only commit, divergent worktree, topic branch, integration branch, planning branch, and transition artifact has a remote Git reference.
- The Agent Brain P0 integration history and the later planning/governance history were reconciled into one canonical candidate.
- The OmniRoute health correction, NIM routing correction, Antigravity resolver correction, and Cline OmniRoute configuration were merged into that candidate.
- The candidate passed strict OpenSpec validation, focused and broad daemon Go tests, focused Go vet, frozen-lockfile frontend installation, workspace typecheck, and production frontend/backend builds.
- A fresh isolated DEV PostgreSQL database was migrated through migration 126.
- A blue/green DEV stack is running from the new `/home/dataops-lab` checkout without replacing the legacy `/mnt/c` stack.
- The new backend and frontend return HTTP 200, while the legacy backend and frontend also remain available as rollback.
- The current DEV database and uploads volume have owner-only backup artifacts and SHA-256 checksums.
- No subordinate collaboration agents remain active. Only the primary session was present at the last audit.

### What is not complete

- Agent Brain is 53/96, not 96/96.
- The new Docker web/backend stack does not itself start the host-side Agent Brain daemon.
- The new Multica DEV backend is not connected to Redis. It uses in-memory realtime fan-out and bounded local rate limiting.
- OmniRoute is still deployed from `diegosouzapw/omniroute:latest`; its digest is not approved/pinned in configuration.
- Live authenticated route acceptance, full failure injection, G4-OBS, capacity tiers, cold-recovery reconciliation, production cutover, and debranding remain open.
- Historical credential exposure and secret-file permission findings still require owner/security handling.
- `main` remains unchanged at `b657129`; the canonical candidate is intentionally on a protected integration branch.

### Decision

- **Continue isolated DEV:** GO.
- **Resume implementation from canonical candidate:** GO, subject to task/file ownership and credential stop rules.
- **Merge candidate to `main`:** NO-GO until the documented gates are explicitly dispositioned.
- **Production cutover:** NO-GO.
- **Stop/delete legacy stack or volumes:** NO-GO until rollback observation is complete and the owner explicitly authorizes it.

## 2. Repository and environment topology

### Current new environment

- Workspace root: `/home/dataops-lab`
- Repository: `/home/dataops-lab/R-D_Agnostic_Engineering_Team`
- Application workspace: `/home/dataops-lab/R-D_Agnostic_Engineering_Team/multica-auth-work`
- Candidate branch: `integration/dev-transition-candidate-20260719`
- Secure runtime configuration: `/home/dataops-lab/.config/multica-transition`
- Runtime backup directory: `/home/dataops-lab/.local/share/multica-transition/backups`

### Legacy environment retained for rollback

- Legacy repository/worktree: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team`
- Legacy application workspace: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work`
- Legacy Docker project: `multica`
- Legacy observability project: `multica-observability`

The legacy path is not the development source of truth. Do not edit it to continue current work. Keep it read-only except for an explicitly authorized rollback or data-export action.

## 3. Git source-of-truth map

### Primary refs

| Purpose | Remote ref or tag | Snapshot commit | Disposition |
|---|---|---:|---|
| Stable baseline | `origin/main` | `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f` | Preserve; not yet advanced |
| Canonical transition branch | `origin/integration/dev-transition-candidate-20260719` | Read the remote tip after clone | Resume here |
| Exact deployed code | `dev-deploy-20260719-candidate` | `6a2aba3550aaf6b0468a37bfdf2f00c7faaae084` | Immutable deployment/rollback anchor |
| Agent Brain P0 integration input | `origin/integration/agent-brain-p0` | `29056e5aaf52d1e10fbfa0745b69b66685febd54` | Integrated; retain as provenance |
| Planning/governance input | `origin/planning/agent-brain-observability-freeze` | `043e6415fdb7f3ecc7ebcabce076c6e1f970981c` | Integrated; retain as provenance |
| Original transition handoff | `origin/transition/dev-handoff-20260719` | `7c60b83b9ff027e2feddf2e82c44cfdbb3f166b3` | Historical handoff |

### Integrated topic refs

| Topic | Ref | Commit | Candidate merge |
|---|---|---:|---:|
| OmniRoute installed health contract | `origin/topic/agent-brain-p0-omniroute-health` | `46abd8a06fda212e97396f89c3eaed61cc188def` | `196e39c` |
| NIM routes through OmniRoute | `origin/topic/agent-brain-p0-nim-omniroute-delta` | `f1c79f77e7fac6df764b9f9cce31f94cea994881` | `65c5710` |
| Antigravity 1.1.4 IPv4 resolver | `origin/topic/antigravity-agy-1.1.4-resolver` | `7735bdcac20ecc6571e170c99e684c7dc7e3915e` | `57f2f46` |
| Cline OmniRoute configuration | `origin/work/agent-brain-w3-cline-omniroute` | `3f735949fd0970480522124e6f874c10354f4983` | `6be1ffa` |

### Worker provenance refs

| Ref | Commit | Status |
|---|---:|---|
| `origin/work/agent-brain-w1` | `3711eb4da77bd8e20848c2beaa6bb143503ad6de` | Integrated into P0; keep |
| `origin/work/agent-brain-w2` | `528d1bb9801bdaf203eca1e8aeb0d8b32b28d55e` | Integrated into P0; keep |
| `origin/work/agent-brain-w3` | `1716186b25a4025fee97559316528d5ef5b7f0e1` | Integrated into P0; keep |
| `origin/work/agent-brain-w4` | `0a291d9fe2779459d3154829344e487adf06e70c` | Integrated into P0; keep |
| `origin/work/agent-brain-w5` | `fd4aa4d203ef16faf0d401514cacab1c2d06415f` | Integrated into P0; keep |
| `origin/work/agent-brain-w6` | `a715b0ab573e37f4feefa05198cf307fd73deb13` | Produced observability work; not automatically accepted |
| `origin/work/agent-brain-w8` | `6aa1e50398314b6ba4d677da1397c568d705dcc6` | Draft sibling reviews; preserve |

### Recovery refs

| Recovery ref | Commit | Exact disposition |
|---|---:|---|
| `origin/backup/dev-transition-home-wip-20260719T231359Z` | `3ca8dca8b5dff83ca20749cd2c05ae19b6622a44` | Earlier divergent NIM state; archive/review-only |
| `origin/backup/dev-transition-old-main-wip-20260719T231359Z` | `d29ff1cfff0b68b8082abe8b724b65cc6447dc4d` | Older credential-isolation WIP plus non-portable Kiro wrapper; archive/review-only |
| `origin/backup/dev-transition-local-w3-20260719T231359Z` | `0ba88da61588bef8bb9b263d8ecf65348f670121` | Observability tree byte-equivalent to accepted W5 tree; archive |
| `origin/backup/dev-transition-disposable-integration-20260719T231359Z` | `54910a813c0e85461e72bdb4ee0f0aa5f807daf8` | Superseded disposable merge candidate; archive |
| `origin/backup/wip-snapshot-20260718T202300Z` | `5106de35b2f0ec2e0b44938547ba5011bdc8e5dc` | Earlier multi-stream safety snapshot; retain |
| `origin/backup/wip-snapshot-20260719T153042Z` | `da42282372d42f61c24c3b8b67bc79e86dc85473` | Planning recovery baseline; retain |

No recovery branch is an automatic merge source. Compare its tree and intent against the canonical candidate before selecting individual commits.

### Immutable transition tags

The repository contains thirteen transition tags: twelve `dev-freeze-20260719-*` recovery tags plus `dev-deploy-20260719-candidate`. Fetch tags before any audit. Never rewrite or delete them.

## 4. Candidate assembly already completed

Do not repeat this integration sequence. It has already been executed and pushed.

1. Candidate created from `origin/integration/agent-brain-p0`.
2. Planning branch merged at `dc60083`.
3. Only two conflicts occurred: `.planning/agent-brain-v3/AGENT_LEDGER.md` and `.planning/agent-brain-v3/EVIDENCE_INDEX.md`.
4. Conflict resolution preserved the integrated worker rows and the later planning ledger/evidence entries.
5. OmniRoute health merged at `196e39c`.
6. NIM correction merged at `65c5710`.
7. Antigravity resolver merged at `57f2f46`.
8. Cline configuration merged at `6be1ffa`.
9. Transition documentation was then added. Documentation commits after `6a2aba3` do not change application binaries.

## 5. AS-IS architecture

### Application plane

- Frontend: Next.js 16 standalone image.
- Backend: Go server image built with Go 1.26 Alpine.
- Primary application database: PostgreSQL 17 with pgvector.
- Host daemon: separate from the backend web/API container; this is where Agent Brain integration resides.
- Runtime routing: OmniRoute is the intended sole hot router for gateway-required Agent Brain tasks.
- Prodex/L2: retained in source as default-OFF, mutually-exclusive cold platform recovery; it is not authorized as an automatic or per-request fallback.
- Redis support: implemented by backend source but optional at runtime. Current candidate has `REDIS_URL` unset.

### Current DEV runtime behavior

- Backend local-auth bypass is enabled only for loopback development.
- Candidate local identity is `owner@local.test`.
- PostgreSQL identity is `multica_transition` on database `multica_transition`.
- Realtime fan-out is in-memory and therefore single-node.
- Rate limiting uses the bounded local fallback.
- Email is not configured; DEV verification behavior is used.
- S3, CloudFront, Lark, and external analytics are disabled/unconfigured in the candidate.

### Agent Brain runtime status

- Core neutral brain contracts, gateway package, runtime-environment isolation, core integration, diagnostics, and a first vertical slice exist.
- Native Kimi/GLM/NVIDIA/Agy route completion is still open under tasks 5.6–5.8.
- Live authenticated route/failure acceptance is still open under 8.1, 8.2, and 8.4–8.7.
- The Docker web/backend stack being healthy does not prove the host daemon or Agent Brain live routing is healthy.
- OmniRoute currently runs as a legacy standalone container attached to `bridge` and `multica_default`; the new `multica-dev-transition_default` network does not contain it.
- The host daemon can reach OmniRoute through `http://127.0.0.1:20128`; container DNS `http://omniroute:20128` is only valid on a network containing the OmniRoute container.

## 6. TO-BE architecture

### Required end state

- One canonical code line promoted through reviewed integration into `main`.
- OmniRoute is the only hot routing and credential-owning component.
- Agent child processes receive a stable, limited OmniRoute credential only through approved restricted secret-file projection; provider-native credentials never reach child env, argv, logs, task homes, or images.
- Every approved model/protocol path has live non-production evidence for discovery, streaming, tools, reasoning, cancellation, usage, retry, failure classification, and affinity.
- End-to-end metadata-only observability covers ingress, queue, daemon, CLI process, OmniRoute/provider, persistence, UI delivery, and complete trace assembly.
- Structural leak scanning proves no prompts, results, tools, repository content, credentials, cookies, emails, connection strings, or opaque reasoning appear in telemetry.
- Tier 20 is enabled only after G4-OBS and capacity thresholds pass. Tiers 50/100 require a shared-state decision and their own sustained/recovery evidence.
- Redis is either explicitly omitted for a documented single-node topology or deployed as a dedicated authenticated, persistent, health-checked Multica dependency. A shared external AOP Redis must not be silently reused.
- Prodex remains default-OFF, operator-gated, mutually exclusive, and usable only as cold platform recovery after OmniRoute is quiesced.
- All mutable secrets reside outside Git with owner-only permissions and a documented rotation/recovery owner.
- Every image is pinned by approved digest or reproducibly rebuilt from an immutable source tag.
- Backup/restore, restart, kill switch, rollback, and clean-clone recovery are proven and recorded.

## 7. OpenSpec program status

### Authoritative task counts

| Change | Complete | Open | Total | Disposition |
|---|---:|---:|---:|---|
| `build-omniroute-agent-brain` | 53 | 43 | 96 | Main Brain authority; active |
| `persist-prodex-runtime-integration` | 0 | 16 | 16 | Cold-recovery-only stream; default-OFF; held |
| `chat-orchestration-standard` | 4 | 6 | 10 | Separate sibling stream |
| `native-runtimes-onboarding` | 9 | 8 | 17 | Separate sibling stream |
| `agent-credential-isolation` | 4 | 17 | 21 | Separate security stream; credential STOP applies |
| `rotation-parity-polyglot` | 78 | 0 | 78 | Complete historical program |

Do not sum sibling streams into an Agent Brain completion percentage. The owner-approved reporting rule says Agent Brain is reported only as N/96.

### Known documentation discrepancy

- Current checkboxes and the latest ledger establish 53/96 after owner-approved closure of 0.1 and 0.7.
- Older current-summary prose still contains 51/96 and says 0.1/0.7 are open.
- Historical event entries that say 51/96 are valid snapshots of their time and must not be rewritten as if they had always been 53/96.
- New agents must use checkbox recount plus the latest append-only ledger entry, not an older paragraph.

### Main Brain open work: 43 tasks

#### P0 supplier/native/live functional work: 11

- `1.3` — complete OmniRoute architecture checklist and pin version/image digest.
- `1.4` — complete Prodex parity matrix including SC01–SC10, reset/redeem, waivers, owners, restrictions, and dates.
- `5.6` — accepted Kimi/GLM/NVIDIA adapter; remove/replace native direct-NVIDIA credential path.
- `5.7` — accepted Kimi provider registry or documented Claude/Codex frontend fallback.
- `5.8` — Agy native endpoint only if proven; otherwise enforce approved fallback and disable direct native path.
- `8.1` — authenticated discovery plus streaming/non-streaming completion for every approved protocol/model route.
- `8.2` — accepted Claude, Codex, Kimi, GLM/NVIDIA, and Antigravity paths with tools, reasoning, cancellation, usage, and deterministic errors.
- `8.4` — concurrent strict round-robin and continuation/prompt-cache/tool affinity.
- `8.5` — token expiry/revocation, quota, 401/403/429, 5xx, timeout, and malformed-upstream behavior.
- `8.6` — retry-before-output, no replay after partial output/tool action, deduplication, and cancellation slot release.
- `8.7` — account lifecycle plus OmniRoute restart/config rollback under load.

#### G4-OBS stop-gate: 11

- `OBS-1` correlation schema and `secrets_present=false` contract.
- `OBS-2` ingress span.
- `OBS-3` queue span.
- `OBS-4` daemon admission/lifecycle span.
- `OBS-5` structurally-redacted CLI-process span.
- `OBS-6` safe OmniRoute/provider span.
- `OBS-7` terminal-persistence span.
- `OBS-8` WebSocket/UI-delivery span.
- `OBS-9` complete eight-hop trace assembly.
- `OBS-10` structural secret/content leakage scan.
- `OBS-11` dashboards, alerts, and consolidated acceptance.

#### Post-gate capacity, cutover, recovery, and naming: 21

- `9.1–9.7` — tier 20/50/100 measurement, enablement, operational evidence, and sign-off.
- `10.1–10.7` — gateway-required default, cohort observation, legacy drain/removal, cold recovery, obsolete credential-path removal, and reconciled runbooks.
- `11.1–11.7` — final name inventory, compatibility-safe rename, migrations, operational rename, alias retirement, and final sign-off.

### Sibling open work

- Credential isolation: `0.1–0.3`, `1.1–1.3`, `2.1–2.3`, `3.1–3.3`, `4.3–4.4`, `5.1`, `5.3–5.4`.
- Native onboarding: `1.5–1.6`, `2.4–2.5`, `3.1–3.4`.
- Chat orchestration: `0.1`, `1.2–1.3`, `2.1–2.3`.
- Persist Prodex: `1.1–1.3`, `2.1–2.3`, `3.1–3.5`, `4.1–4.5`.

## 8. Validation already performed

Do not repeat these only to rediscover whether the candidate compiles. Repeat them when source changes, the toolchain changes, or formal acceptance requires fresh evidence.

### OpenSpec

- `openspec validate build-omniroute-agent-brain --strict` passed.
- `openspec validate persist-prodex-runtime-integration --strict` passed.

### Backend

- Host Go was unavailable; validation used the repository-required Go 1.26.1 container.
- Focused packages passed: `internal/daemon/brain`, `internal/daemon/gateway`, `internal/daemon/runtimeenv`, `internal/daemon/observability/e2e`, and `pkg/agent`.
- Full `internal/daemon` package tests passed using the existing module cache with `GOPROXY=off` and `GOSUMDB=off`.
- Focused `go vet` passed for the Agent Brain/gateway/runtime/observability/agent packages.
- Docker `--init` was required for orphan-process reaping tests. Failures without `--init` were container PID 1 behavior, not application regressions.
- A full repository-wide `go test ./...` was not claimed.

### Frontend

- `pnpm install --frozen-lockfile` completed.
- `pnpm typecheck` passed across eight non-mobile packages.
- `pnpm build` passed, including the web production build.
- The host lacked `go`, so desktop CLI bundling was skipped by the workspace build.
- `server/bin/linux-amd64/multica` was absent; desktop would install the latest binary at runtime. This is not acceptable as a production reproducibility guarantee and remains an operational follow-up.

### Container builds and runtime

- Backend image: `multica-backend:transition-6a2aba3`, image ID `sha256:c8ba7dc56be057c9bff5e076d6020d7d550cf4406a344729defc1eba94057dd0`.
- Frontend image: `multica-web:transition-6a2aba3`, image ID `sha256:29e4bc52c351443234ac593eefecb4a712e205f5c9602b66fc6d5add39a8952d`.
- PostgreSQL image ID: `sha256:dd467f03ca5c5581222490e5217e48a262864ccb659be559f8491bbafdc97da0`.
- Migrations completed through 126.
- Candidate backend `/health`: HTTP 200 with `{"status":"ok"}`.
- Candidate frontend `/login`: HTTP 200.
- Legacy backend `/health`: HTTP 200 after candidate launch.
- Legacy frontend `/login`: HTTP 200 after candidate launch.

### Not yet validated

- Live provider traffic and provider-authenticated model discovery.
- Full Agent Brain host-daemon start from the new environment.
- G4-OBS complete eight-hop trace and leak scan.
- Race testing for every changed concurrency path across the whole repository.
- Migration down/re-up against a disposable clone.
- Actual database restore into a second isolated PostgreSQL instance.
- Redis-backed Multica operation.
- OmniRoute rebuild/recreate from an approved pinned digest.
- Capacity profiles at 20, 50, or 100 tasks.
- Clean-clone reconstruction on a separate machine.

## 9. Runtime data preservation

### Candidate Docker volumes

- `multica-dev-transition_pgdata` → PostgreSQL data.
- `multica-dev-transition_backend_uploads` → application uploads.

Never use `docker compose down --volumes` unless destruction is explicitly authorized and verified backups exist.

### Current owner-only backups

Directory: `/home/dataops-lab/.local/share/multica-transition/backups` with mode 700.

- `postgres-20260719T233100-0300.dump` — PostgreSQL custom-format dump, mode 600, 230414 bytes.
- `uploads-20260719T233100-0300.tar.gz` — uploads archive, mode 600, 87 bytes.
- `SHA256SUMS-20260719T233100-0300` — checksum manifest, mode 600.

Recorded checksums:

- PostgreSQL: `699f708858bc81f6baca4c30ac35b189fae2e542a1cfba909c8e85a35a6a9f51`
- Uploads: `1f15f6f5b101399d20f2154e4ead5f1b631164054ae2d65c7fd0cdc20d4c44e8`

These artifacts are not in Git and must be transferred through an owner-approved encrypted channel if runtime state must move to another machine.

## 10. Decision register

### Decisions already made; do not reopen without owner direction

- OmniRoute is the only hot router/credential owner.
- Agent Brain receives one stable, limited OmniRoute secret.
- `Agent Brain` is a provisional name only.
- First intended capacity tier is 20.
- Logical independent request is the rotation unit; continuations use explicit affinity.
- Single-node SQLite/state is acceptable only for tier-20 validation.
- Use strangler extraction, not a global rewrite.
- Retry/fallback is pre-commit only.
- Secrets originate from Linux permission-restricted storage, never a world-readable Windows path.
- Prodex is retained as default-OFF cold platform recovery and must be mutually exclusive with OmniRoute.
- G4-OBS is a stop-gate before capacity/cutover.
- Observability execution remains lower priority than functional P0 unless the owner changes priority.
- Existing Kanban work remains parked until credentialless dispatch is accepted.
- Agents report facts and evidence; owner alone authorizes accept/stop/defer and all credential/session mutations.

### Pending owner decisions

| ID | Decision | Blocks |
|---|---|---|
| `PD-02` | Approve/pin OmniRoute image digest | Reproducible cutover |
| `PD-03` | Approve SC01–SC10 implementation plan or product/security waiver | Parity, G5 |
| `PD-04` | Choose single-node vs shared state | Tier 50/100, horizontal scale |
| `PD-05` | Choose final product name | Debranding/G8 |
| `PD-06` | Complete product/architecture/security sign-offs | Final readiness |
| `PD-07` | Rotate key associated with earlier partial-prefix exposure | Security/cutover |
| `PD-08` | Complete approved remediation of the exposed legacy Windows credential and confirm rotation/revocation | Live-auth and cutover |
| `REDIS-01` | Decide whether DEV/target Multica is intentionally single-node without Redis or receives a dedicated authenticated persistent Redis | Multi-node realtime/rate limiting |
| `DEPLOY-01` | Define target new-environment hostname, TLS/reverse proxy, observation window, and cutover owner | External access/cutover |

## 11. Critical risks requiring continued control

- Floating `:latest` images make rebuilds non-reproducible.
- The legacy OmniRoute container is standalone and not declared by the new candidate Compose project.
- The shared AOP Redis is authenticated but external, uses a floating image, has no Docker volume mount, and stores its secret in a Windows-mounted `.env` observed as mode 777 from Linux.
- Legacy Multica observability secret files under `/mnt/c` were also observed as mode 777 from Linux.
- Reusing shared Redis can create cross-project key collisions, shared failure domains, unclear ownership, and accidental credential coupling.
- Running without Redis is acceptable only for one backend node; it does not provide cross-node realtime fan-out or distributed state/rate limiting.
- Candidate frontend/backend containers have no Docker healthcheck, so `running` alone is insufficient; use HTTP probes.
- Candidate configuration was initially created with a temporary override path. Persistent copies now exist under `/home/dataops-lab/.config/multica-transition`; always use those paths for future Compose commands.
- A clean source tree does not prove all runtime state is backed up; preserve volumes and external secrets separately.
- Historical planning summaries can lag the append-only ledger; always recount checkboxes.
- Direct credential handling remains prohibited for agents.

## 12. No-repeat work ledger

The following work must not be assigned again as discovery:

- Locating the remote repository and configuring Git push authentication.
- Identifying and preserving the four transition recovery branches.
- Comparing the recovered observability branch with W5; it is tree-equivalent and archived.
- Determining the old credential-isolation recovery is older and contains a hard-coded `/mnt/c` wrapper.
- Determining the new-DEV NIM recovery is older than the integrated NIM path.
- Reconciling Agent Brain integration with planning.
- Integrating OmniRoute health, NIM, Antigravity, and Cline topics.
- Running focused Agent Brain/daemon tests and vet.
- Installing frontend dependencies, typechecking, and production-building.
- Proving blue/green port/volume/image isolation.
- Proving candidate and legacy HTTP health simultaneously.
- Determining that Redis exists externally but is not connected to candidate Multica.
- Determining current Redis requires authentication.
- Determining candidate runtime secrets are owner-only and outside Git.
- Producing the initial PostgreSQL and uploads backups.

Future work should consume these facts and move directly to the next open acceptance item.

## 13. Exact next-action sequence

1. Clone/fetch and verify all immutable refs using `FRESH_ENV_RESTART_RUNBOOK.md`.
2. Recreate or securely transfer runtime secrets; never copy them through Git or chat.
3. Verify Docker prerequisites and reconstruct only the isolated DEV project.
4. Restore the DEV database/uploads only if state continuity is required; otherwise retain the fresh database.
5. Start and verify the web/backend/PostgreSQL stack.
6. Decide `REDIS-01`; do not silently point Multica at the shared AOP Redis.
7. Provision the host-side daemon and OmniRoute secret path without reading or printing credential values.
8. Resolve/pin `PD-02` before rebuilding OmniRoute.
9. Continue P0 tasks 1.3, 1.4, 5.6–5.8, and 8.1–8.7 under explicit no-credential rules.
10. Complete G4-OBS before any capacity task.
11. Execute tier-20 acceptance only after owner thresholds and evidence ownership are recorded.
12. Reconcile Prodex cold recovery, then controlled cohort observation and legacy drain.
13. Resolve product naming and complete final operational/security sign-off.
14. Merge to `main` only after all mandatory gates are evidence-backed and owner-authorized.

## 14. Required first report from any resumed agent team

Before editing, the orchestration lead must report:

- Checked-out branch and exact `HEAD`.
- Remote candidate tip and deployed tag resolution.
- Clean/dirty/untracked Git status.
- OpenSpec counts by change, with Agent Brain reported separately as N/96.
- Docker project/container state and health probes.
- Whether runtime secrets and backups exist, reporting paths/modes only.
- Active agent names, task IDs, branch/worktree, locked files, and evidence owner.
- Pending owner decisions and credential/security stop conditions.
- First concrete task and why it does not duplicate completed work.

If any of those facts differ from this snapshot, record the delta before continuing.
