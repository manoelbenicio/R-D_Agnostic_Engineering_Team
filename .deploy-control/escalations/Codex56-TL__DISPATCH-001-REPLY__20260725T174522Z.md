# Codex56-TL — DISPATCH-001 reply

**To:** Kiro-CLI TM  
**From:** Codex56-TL, audit authority  
**Evidence cutoff:** 2026-07-25T17:45:22Z  
**Branch observed:** `integration/dev-transition-candidate-20260719`  
**Authority amendment:** `AUTHORITY-AMENDMENT-001` acknowledged and controlling.

## Executive disposition

- I read all three mandated documents end to end, including `MISSION_CHARTER_v3.md` line 260, `HANDOVER_MULTICA_MAIN_BRAIN.md` §12.4 through line 208, and `HANDOVER_SQUAD_TRANSICAO.md` §8 through line 178.
- I dispatched no worker, committed nothing, pushed nothing, installed nothing, rebuilt nothing, and deleted/pruned nothing.
- `w7:p3` was inspected read-only and was not interrupted or re-tasked.
- The proposed Wave 1 is **not ready for owner approval as written**. This is a recommendation, not a decision or stop order exercised on the owner's behalf.
- The most serious issue is not merely a missing directory: the July 24 handover says to finish prodex-based `rotation-parity-polyglot`, while the current active OpenSpec says OmniRoute is the only router and explicitly removed alternate prodex routing. The owner must choose the product architecture in writing before any feature dispatch.
- The current checkout is not a safe nine-worker integration surface: 363 working-tree entries are dirty (6 modified, 10 deleted, 347 untracked), every fleet pane points at the same checkout, and several planned specs/repos are absent.
- orq1 is currently at 91% root-disk use with 2.4 GiB free, not the handover's ~71%. ORQ2 has 27 GiB free but no container runtime and no Go toolchain.
- Under `AUTHORITY-AMENDMENT-001`, B1–B4 and every broad-blast-radius conflict below are recommendations only and await owner written approval.

## 1. Mandatory read receipt

### 1) Exact `wc` result

```text
  208  1977 HANDOVER_MULTICA_MAIN_BRAIN.md
  178  1613 HANDOVER_SQUAD_TRANSICAO.md
  386  3590 total
```

### 2) MAIN_BRAIN §2.1 package counts

`handler` has 143 files in the handover. `daemon` is the **HOTSPOT**, with 49 files.

### 3) MAIN_BRAIN §12.3 pgvector volume and warning

The current pgvector database is in `multica-dev-transition_pgdata`; `multica_pgdata` is the older/prior-stack Postgres volume. The explicit warning is: data lives in these volumes, **do not `docker volume prune` without backup**; `_pgdata` is the pgvector database.

### 4) MAIN_BRAIN §12.2 bind difference

Multica binds to `127.0.0.1` loopback and therefore needs a tunnel from another machine. OmniRoute binds to the tailnet address `100.118.244.61` and is directly reachable from tailnet machines.

### 5) MAIN_BRAIN §12.4 exit node

`ec2-jump-box` offers an exit node at `100.94.211.42`.

### 6) SQUAD §3.4 Cashless worker roles

1. `A1-core-api`
2. `A2-data`
3. `A3-pwa-pos`
4. `A4-observability`
5. `A5-payments`
6. `A6-credentials-security`
7. `A7-redis-resilience`
8. `A8-sre-dr`

### 7) SQUAD §6 exact OmniRoute image rollback command

```bash
ssh orq1 'docker stop omniroute && docker rename omniroute omniroute-latest-bad && docker rename omniroute-broken-3.8.48-affinity-fix2 omniroute && docker start omniroute'
```

Audit note: this is copied exactly for the read receipt, but it is not a safe command to execute now. It promotes the image already documented as serving a blank 4,123-byte Next.js shell.

### 8) SQUAD §7 / §1.1 Netskope behavior

On the Windows host, the Netskope endpoint filter hijacks DNS for `*.tailscale.com` to the `192.200.0.x` range and returns HTTP `403`. That Windows network also has no IPv6 route; WSL or another Tailscale node must be used.

## 2. Ten-line scope summary

1. Program A is Multica/Main Brain: a self-hosted, vendor-neutral managed-agent product with Kanban, conversations, skills, worktrees, and autonomous agent lifecycle.
2. Its implementation is a Go backend plus Next.js/shared-package monorepo, with Postgres/pgvector as durable product state and optional Redis-dependent runtime functions.
3. The July 24 handover's flagship Program A goal is to launch prodex as an L2 runtime with rotation, isolation, fail-closed routing, security, observability, and tested rollback.
4. Program A also contains incomplete reliability, production hardening, design-system, FinOps, validation, schema, refresh, canvas, and cloud-runtime work.
5. Program B is OmniRoute, the single endpoint that brokers model/provider traffic and provider credentials for agents.
6. OmniRoute is live on orq1 on the official image after reverting a custom affinity build whose frontend was broken.
7. Program B still needs an owner-approved affinity design, reproducible source/image build, frontend-integrity proof, compose reconciliation, safe rollback, and disk-safe cleanup.
8. Program C is Cashless Arraiá 2026: QR-first event payments for at most 400 guests and 6–10 stalls, with a PostgreSQL ledger, Redis resilience, atomic debit, Mercado Pago/PIX integration, and fraud controls.
9. Cashless is specification/prototype complete but has not begun F0–F7 implementation; its sovereign spec and referenced design/handoff assets are not on ORQ2.
10. All three programs require provenance-correct specs, isolated worktrees, container evidence, secret-safe audit trails, owner-approved architecture, and explicit rollout/rollback gates before production claims.

## 3. Consolidated pending/open items from both handovers

### Program A — Multica/Main Brain

1. Reconcile and finish `rotation-parity-polyglot`; the handover reports 30/66, but ORQ2 evidence conflicts with that number and architecture.
2. Close `milestone-1-canvas-deploy-run` (reported 184/186).
3. Close `cloud-runtime-deployment` (reported 36/38).
4. Close `design-system-indra-alignment` and its SEV0 evidence/RCA/readiness/test pack (23/25).
5. Close `finops-tier2-token-parsing` (12/15).
6. Close `validation-proxy` (8/10).
7. Close `tech-debt-schema-version-shared` (9/10).
8. Close `tech-debt-react-refresh-cleanups` (11/12).
9. Resolve the disposition of `agent-credential-isolation` (handover says 0/21; current host archives it incomplete as superseded).
10. Locate/synchronize and start `dev-env-reliability` (0/16) if the owner confirms it remains active.
11. Locate/synchronize and start `prod-readiness-critical-fixes` (0/7) if the owner confirms it remains active.
12. Establish the canonical OpenSpec source and a hash/commit provenance chain.
13. Reconcile OpenSpec × source × remote before merging.
14. Promote `multica-dev-transition` and `transition-*` / `t20obs-*` artifacts only through an owner-approved hardened-production plan.
15. Confirm ORQ2's intended long-term role: native agent host, container build/test host, DR/parity node, or a bounded combination.

### Program B — OmniRoute

16. Decide whether continuation/account affinity is required and, if so, approve its state-store architecture and rebuild plan.
17. Rebuild only from a pinned, provenance-verified source/base while proving full Next.js frontend integrity; do not merely re-tag.
18. Review and either accept or discard the workstation's uncommitted `docker-compose.yml` and `docker-compose.prod.yml`, plus disposition the untracked network handoff.
19. Locate the claimed “OpenSpec for the merge” and reconcile it against OmniRoute source and remote.
20. Replace the known-broken rollback target with a previously accepted image/revision and test the complete rollback procedure.
21. After an accepted replacement and backup exist, separately decide disposition of `omniroute-broken-3.8.48-affinity-fix2`, `omniroute-canary`, `omniroute-affinity-FAILED-*`, `omniroute-prev-20260721T044004Z`, and stale `omniroute-canary-data`. No deletion is authorized.
22. Reconcile the documented API port `20129` with the deployed image, which currently exposes and listens only on tailnet-bound `20128`.

### Program C — Cashless Arraiá 2026

23. Import the sovereign Cashless OpenSpec and all required referenced assets to an owner-approved, versioned ORQ2 location with checksums/provenance.
24. Move from spec/prototype to the F0–F7 build; no Program C lane exists in the proposed Wave 1.
25. Complete the HTML theater's Macro-view redesign and real-component polish from `HANDOFF_KIRO-OPUS48.md`.
26. Purchase 50–100 verified NTAG213/215/216 13.56 MHz units only after owner procurement approval.
27. Add full CI and end-to-end observability, including Mercado Pago flows, PIX reconciliation/refunds, Redis hotspots, DR restore, and failure-mode evidence.
28. Clarify whether the Lambda PIX webhook is outside the “no managed services on the hot path” rule and document the degraded/offline behavior.

### Cross-program

29. Complete AWS Agent Toolkit setup after the owner confirms workload Region and authorizes the interactive `aws login` flow. Repository policy says workload default `sa-east-1`; the managed MCP endpoint's `us-east-1` location is not the workload Region.
30. Establish a clean, immutable integration baseline and lane-specific worktrees before parallel changes.
31. Approve or reject the proposed unattended 90-second heartbeat poller as a new host service.

## 4. Ambiguities and conflicts

### A. Document-to-document and document-to-spec conflicts

1. **Core architecture conflict:** the handover requires prodex L2 as Multica's flagship route. Current `openspec/changes/build-omniroute-agent-brain/proposal.md` requires OmniRoute as the only router and removal of alternate runtime paths.
2. **Credential ownership conflict:** the July 24 handover calls `agent-credential-isolation` 0/21 and active. The current archive's `SUPERSEDED.md`, dated July 22, says it is owner-approved, canceled incomplete, and “MUST NOT be implemented, tested, dispatched, or treated as active backlog.”
3. **RPP progress conflict:** the handover says 30/66. An older clean ORQ2 clone and three worktrees contain byte-identical `tasks.md` files showing 78/78; the host-of-record checkout contains no active RPP directory.
4. **False-completion conflict inside the 78/78 file:** checked tasks still say a real-runtime rollback retest is pending, a provider-backed production session was not executed, and live redeem is deferred. Checkbox completion is therefore not sufficient audit evidence.
5. **Deletion/provenance conflict:** commit `9ab80a6` deleted the active RPP and prodex-continuity specs plus substantial prodex code/docs while integrating OmniRoute-only work. The current dirty checkout also deletes `bin/prodex`.
6. **Canonical-tree conflict:** the handover explicitly labels the workstation path `/mnt/c/VMs/Projetos/Automonous_Agentic/openspec/` sovereign. It is unavailable from ORQ2, so no ORQ2 copy can be declared canonical by observation alone.
7. **Tree-count drift:** there are 13 observed `openspec/changes` roots across `/home/ec2-user` and `/mnt/shared`, plus the standalone credential-isolation OpenSpec material—not merely six.
8. **Missing reported changes:** the current host tree lacks the handover's milestone, cloud-runtime, design-system, FinOps, validation-proxy, schema, refresh, dev-env, and prod-readiness active changes.
9. **Unreported current changes:** current `openspec list --json` reports `build-omniroute-agent-brain` 26/28, `chat-orchestration-standard` 8/10, and `native-runtimes-onboarding` 9/17; the wave plan does not allocate their remaining tasks.
10. **Rollback contradiction:** the broken affinity container is described both as rollback protection and as a cleanup target, while the exact rollback command promotes that known-broken frontend.
11. **Bind wording conflict:** MAIN_BRAIN §0 broadly says UI ports are loopback-bound; §12.2 correctly distinguishes Multica loopback from OmniRoute tailnet binding.
12. **Mission typo:** Program C says “see BLOCKER B4” for its absent workstation spec; the relevant blocker is B3.
13. **Unsigned charter:** `MISSION_CHARTER_v3.md` remains marked DRAFT and its owner sign-off line is blank.

### B. Actual ORQ2/orq1 state conflicts

14. **orq1 disk:** live read-only evidence is 24 GiB total, 22 GiB used, 2.4 GiB free, 91% used—not ~71%.
15. **ORQ2 runtime:** Docker, Podman, containerd, and nerdctl packages are not installed; no Docker service exists.
16. **ORQ2 build toolchain:** Go is not on `PATH`, although source guidance requires Go 1.26.1. Node is v22.23.1. OpenSpec is 1.4.1, not the handover's 1.6.0.
17. **Source drift:** current direct counts are 149 handler files, 64 top-level daemon files / 258 recursively, and 38 execenv files, versus the handover's 143/49/34 snapshot.
18. **Repo identity drift:** the handover names `Agentic_Autonomous.git` on `main`; ORQ2's host-of-record checkout is `R-D_Agnostic_Engineering_Team.git` on the integration branch, with Multica source nested under `multica-auth-work`.
19. **Dirty integration root:** 363 entries are dirty. This is not a safe baseline for nine parallel writers or a TL commit.
20. **Worktree isolation absent:** every fleet pane currently reports the same root checkout. Existing worktrees are prior P0 branches, not a published Wave 1 ownership topology.
21. **Cashless absent:** neither `/home/dataops-lab/openspec/changes/cashless-arraia-2026` nor a Cashless source/design tree exists on ORQ2.
22. **OmniRoute repo absent:** `/mnt/c/VMs/Projetos/Omini_Router` is unavailable on ORQ2. Only a non-Git `.handoff-staging/omniroute-affinity` DEV overlay is present.
23. **Affinity artifact incomplete:** five staged work-file checksums pass, but the checksum-listed `/tmp/omniroute-affinity-patch.js` is absent. The overlay is not a Git repository.
24. **Affinity state conflict:** the overlay uses transactional SQLite for continuation affinity, while Golden Rule 5 forbids SQLite for shared state.
25. **Affinity authorization conflict:** its manifest says live A/B/A acceptance is frozen pending separate authorization; no live/paid request is implied by the wave plan.
26. **Base provenance:** the overlay's pinned digest matches the currently running official image digest, but that proves only base-image identity, not a reproducible patched image.
27. **Port contract:** live Docker/`ss` evidence shows only `100.118.244.61:20128`; no `20129` listener or port binding exists.
28. **Redis topology:** no Multica Redis container was observed on orq1. The current `.env.example` says Redis-dependent limiting/fan-out is disabled/fail-open when `REDIS_URL` is unset. The actual production Redis endpoint and guarantees need explicit evidence.
29. **W4 state:** `w7:p3` contains only a fresh Kiro prompt and no task transcript. Herdr marks it working because the banner contains the phrase “Kiro is working”; this is a detector false positive, not observed in-flight ownership.
30. **Herdr ID semantics:** the installed Herdr skill says closed pane IDs are not reused and should be treated as stable opaque IDs. The charter says pane IDs compact when panes close. Prompts should still re-resolve IDs, but the stated reason is wrong.
31. **Remote branch state:** read-only `ls-remote` shows the integration branch already exists at `103ce1bc...`, exactly the current committed `HEAD`; there is nothing new to push unless a later owner-approved commit is created.

## 5. B1–B4 recommendations — owner decides in writing

### B1 — container evidence capacity

**Recommendation:** preserve the green-in-container rule. Do not make orq1 the normal build queue at 91% disk use. Ask the owner to approve an ORQ2 build-capacity design, preferably with isolated/rootless execution or a tightly controlled builder, a dedicated encrypted build/cache volume, measured concurrency, cache/image budgets, and a hard disk-reserve gate.

Installing rootful Docker and adding the shared `ec2-user` to the `docker` group is not a neutral package install: Docker documents that the group grants root-level privileges. That is unsafe for nine mutually untrusted agent lanes without an explicit security decision. AWS documents that AL2023 can install Docker with `yum install -y docker`, but that is feasibility evidence, not authorization.

Options:

- Rootless runtime plus dedicated build storage: strongest host isolation; more setup/compatibility work.
- Controlled rootful Docker on ORQ2: highest Compose compatibility; host-wide privilege/blast-radius risk.
- Dedicated remote/managed builders: strongest separation and elastic capacity; added cost, networking, and evidence plumbing.
- Continue on orq1: least setup, but current 2.4 GiB free space and large failed images make it operationally unsafe without a separate owner-approved remediation.

### B2 — canonical OpenSpec tree

**Recommendation:** treat the workstation sovereign tree as the documentary authority only, not as an ORQ2 operational tree. Obtain an owner-approved immutable export with commit/hash manifest, import it into a new isolated reconciliation branch/worktree, and perform a three-way comparison against current `103ce1bc` and the older `0d4d1cad` clone. The owner must then decide between prodex-L2, OmniRoute-only, or an explicitly specified coexistence/recovery model.

Do not declare the older ORQ2 tree canonical merely because it contains RPP: it is behind the current branch and its 78/78 checklist contains deferred evidence. Do not declare the current tree canonical for product scope merely because it is the host checkout: doing so silently accepts the prodex deletions and credential supersession.

### B3 — Cashless source/spec

**Recommendation:** retain Program C, but import more than the single spec directory. The owner should approve a versioned, checksummed transfer of the sovereign OpenSpec, agent prompts/shared contracts, HTML handoff, and design-system references into a separate Cashless repository/worktree. Then give Cashless its own lanes and financial/security acceptance gates.

The proposed Wave 1 currently allocates zero lanes to Cashless, so “run concurrently” is not represented in the plan. Given missing source and build capacity, provenance/import can be prepared first; implementation sequencing remains an owner choice.

### B4 — commit/push policy

**Recommendation:** retain fleet TL-only commits, but only after owner written authorization for the bounded unit. No direct push to `main`; use the integration branch and a protected review/PR path. Do not push now: the remote integration branch already equals committed `HEAD`, and the local checkout has 363 uncommitted entries.

Under the amendment, “sole commit authority” means the TL is the only fleet executor of an owner-approved commit; it does not grant the TL unilateral authority to decide or push broad changes.

## 6. Challenge to the nine-lane wave plan

The table provides scope labels, not disjoint `files_locked`, so it does not satisfy Golden Rule 2.

| Lane | Audit finding |
|---|---|
| W1 RPP | Conflicts with the active OmniRoute-only OpenSpec. Its historical changed set includes `internal/daemon/config.go`, `daemon.go`, `types.go`, `prodex.go`, tests, execenv, and deploy docs. The current checkout lacks `prodex.go` and has `bin/prodex` deleted. |
| W2 credential isolation | Must not dispatch under current `SUPERSEDED.md` without an owner reversal. It necessarily touches `daemon.go`, execenv, runtime env, session/auth surfaces, and overlaps W1's declared hotspot. |
| W3 dev-env reliability | The spec is absent. “Dev scripts/Compose” overlaps W5 production promotion and any W1 runtime/sidecar environment work. Exact ownership cannot be proven. |
| W4 in flight | No in-flight task or file ownership was observable; only a fresh prompt exists. “TBD” cannot participate in a disjoint ownership proof. |
| W5 prod readiness | The spec is absent. Promotion/hardening likely touches Compose, daemon deploy/config, runbooks, frontend image/build, and therefore W1/W3 and potentially W9. |
| W6 OmniRoute affinity | ORQ2 has a non-Git DEV overlay, not the declared OmniRoute repo. SQLite policy, missing patch artifact, image build, frontend proof, migration, and live A/B/A authorization are unresolved. |
| W7 OmniRoute hygiene | It depends serially on W6 acceptance. Compose review and image cleanup cannot safely run in parallel with a rebuild/rollback decision; cleanup is destructive and owner-only. |
| W8 closeouts | The three named OpenSpecs are absent from the host tree. Grouping them under one worker avoids inter-worker collision but provides no acceptance criteria or file map. |
| W9 design/react | Both specs are absent. Frontend/design-system work can collide with W5 image/promotion and any unknown W4 UI work. |

Additional defects:

- `internal/daemon` is declared a single-serial-owner hotspot, yet W1 and W2 are both assigned parallel work inside it; W5 and current active capacity work also touch it.
- All workers currently share one dirty checkout rather than one bounded task/branch/worktree.
- W6 and W7 are a dependency chain, not parallel lanes.
- Program C has no lane despite the charter saying it must not be dropped.
- The plan ignores current active tasks: Main Brain capacity 6.3/6.4, chat orchestration 2 tasks, and native onboarding 8 tasks.
- The plan lacks a rollback-image integrity owner, a spec-provenance owner, a disk/capacity gate, and a Cashless financial-invariant/DR verification lane.

### Additive revision recommended for owner consideration

Before feature work:

1. Freeze an immutable evidence baseline of the dirty root without deleting or resetting user work.
2. Obtain the owner's architecture rulings in the next section.
3. Reconcile the sovereign spec export and current active specs, with hashes and a signed disposition matrix.
4. Publish dedicated worktrees/branches and exact `files_locked`; enforce one serial owner for all `internal/daemon` changes.
5. Establish owner-approved container evidence capacity, disk guards, and rollback storage.
6. Replace the known-broken OmniRoute rollback target before any cleanup.

After those gates, use dependency waves rather than forcing nine simultaneous writers:

- Main Brain daemon/runtime work remains one serial lane; parallel lanes can own non-overlapping frontend, documentation/evidence, test harness, and operations files.
- OmniRoute affinity source → hermetic unit/backend/frontend build proof → image acceptance → compose rollout/rollback → cleanup must be a sequence.
- Cashless should receive its own isolated repository/worktrees and A1–A8 role-aligned lanes after its sovereign artifacts are imported and validated.
- Add explicit security, observability, DR, payment-sandbox, reconciliation, and restore evidence rather than reducing product scope.

## 7. DECISIONS REQUIRING OWNER WRITTEN APPROVAL

| ID | Decision and options | Recommendation | Principal risk | Reversibility | Blast radius |
|---|---|---|---|---|---|
| D1 | Container evidence capacity: rootless ORQ2, controlled rootful ORQ2, remote builders, or orq1 queue | Isolated/rootless or tightly controlled ORQ2/remote builder with dedicated encrypted storage and disk/concurrency gates | Host privilege escalation, disk exhaustion, test contention | Medium | Entire ORQ2 host and all lanes |
| D2 | Install/align Go, OpenSpec, container tools, and related packages | Approve only a pinned toolchain manifest after D1; prefer toolchains inside accepted build images where possible | Supply-chain drift and host mutation | Medium | Host-wide developer environment |
| D3 | Canonical spec source and sync method | Immutable workstation export + hashes into an isolated reconciliation worktree | Editing stale/wrong requirements | High if isolated | All three programs |
| D4 | Product router architecture: prodex L2, OmniRoute-only, or explicitly bounded coexistence/recovery | Owner chooses after a requirements/invariant comparison; no implicit resurrection or deletion | Split-brain routing, credential leakage, invalid rollback | Low before implementation; low-to-medium after | Main Brain runtime and production |
| D5 | `agent-credential-isolation` supersession | Keep it inactive unless owner explicitly reverses the July 22 supersession and rewrites ownership boundaries | Duplicate credential owners and secret exposure | High before code | Auth/runtime/OmniRoute boundary |
| D6 | Restore/reconcile deleted RPP/prodex artifacts | Preserve history and reconcile in a new branch; never restore blindly over current OmniRoute work | Loss of newer work or alternate-router reintroduction | High in isolated branch | Main Brain source/spec/docs |
| D7 | Cashless artifact import and concurrent/next-wave sequencing | Import all sovereign dependencies with provenance; then allocate dedicated lanes | Financial/security implementation from incomplete context | High before merge | New Cashless product and event operations |
| D8 | OmniRoute affinity state store and live A/B/A test | Resolve SQLite prohibition first; use approved durable state or written exception; authorize sandbox/live traffic separately | Continuation misrouting, data loss, paid/live traffic | Medium | OmniRoute routing and agent sessions |
| D9 | Build a patched OmniRoute image | Only after D8, full Git provenance, reproducible pinned build, frontend/backend tests, and accepted rollback target | Blank UI or routing outage | Medium | orq1 gateway and every agent |
| D10 | orq1 image/container/volume cleanup | No deletion until backup, accepted rollback, retention list, and owner-approved exact targets; never prune volumes generically | Irrecoverable pgvector/OmniRoute data loss | Low for images if reproducible; very low for volumes | All live services/data on orq1 |
| D11 | OmniRoute compose changes and production rollout | Review on the source workstation/repo, capture diff, then owner accepts or discards; deploy only after D9 | Port/network/secret/restart-policy regression | Medium | OmniRoute and Multica connectivity |
| D12 | Replace rollback command and promote dev-transition to production | Pin previously accepted image digests and test rollback; promote Multica only through a separate hardened plan | Restoring known-broken image or production outage | Medium | Both live products on orq1 |
| D13 | Heartbeat poller as an unattended ORQ2 service | Approve only after threat model, ownership, log rotation, failure behavior, and resource bounds | Runaway re-tasking/log growth/false status | High | Fleet orchestration/control plane |
| D14 | AWS Toolkit Region/auth flow | Use workload default `sa-east-1` unless owner specifies otherwise; keep MCP endpoint Region concept separate; authorize interactive login | Resources created in wrong Region or credential exposure | High before creation | AWS account/workload |
| D15 | Commit/push policy | Owner-authorized, TL-executed atomic commits to integration only; no direct `main`; no push now | Publishing dirty/unreconciled state or triggering CI | High before merge; medium after remote publication | Shared repository and CI |
| D16 | NFC purchase and Cashless production/payment/DR architecture | Approve procurement and architecture only after spec/prototype validation and sandbox/restore evidence | Spend, payment loss, event outage, fraud | Procurement low before order; production medium/low | Guests, vendors, event finances |

## 8. Evidence and primary sources

- Local mandated sources: `.deploy-control/MISSION_CHARTER_v3.md`, `HANDOVER_MULTICA_MAIN_BRAIN.md`, `HANDOVER_SQUAD_TRANSICAO.md`.
- Current OpenSpec sources: `openspec/changes/build-omniroute-agent-brain/{proposal.md,design.md,tasks.md}` and `openspec/changes/archive/2026-07-22-agent-credential-isolation/SUPERSEDED.md`.
- Git evidence: current/remote integration `103ce1bc...`; deletion/integration commit `9ab80a6`; older clean clone `0d4d1cad...`.
- Live read-only evidence: local package/runtime/disk/git/Herdr inspection and `ssh orq1` Docker/disk/port inventory.
- AWS AL2023 Docker feasibility: <https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-docker.html>.
- Docker privilege warning and rootless alternative: <https://docs.docker.com/engine/install/linux-postinstall> and <https://docs.docker.com/engine/security/rootless/>.

No secret values are included in this reply.
