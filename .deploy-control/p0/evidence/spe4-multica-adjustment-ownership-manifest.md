# SPE-4 Multica Adjustment Ownership and Dependency Lock

**Issue:** SPE-4
**Workspace:** `56422d5a-539f-4973-ab1c-49beebbc45d0`
**Authority source:** `/tmp/multica-authoritative-schedule-20260731T214806Z/pending-task-schedule.md`
**Producer:** K1 / TL-ORCHESTRATOR
**Scope:** documentation and future ownership freeze only
**Decision:** architecture/planning is **GREENLIGHT WITH CONDITIONS**; implementation and production are **NO-GO**.

This manifest records existing assignments; it does not create, rename, move, share, or reassign an agent, runtime, credential home, account, subscription, workspace, project, squad, daemon, container, or provider adapter. Kanban remains the only executable dispatch surface. This file is not implementation authorization, rollout authorization, or SharePoint authorization.

## Immutable eight-slot mapping

The new workspace is limited to **exactly eight persistent agents and exactly eight logical runtime rows: three Kiro and five Codex**. These are scheduling aliases bound to existing agents. Subagents are nonpersistent, inherit their parent runtime/configuration unless an explicitly governed task override is allowed, and do not consume or create persistent runtime rows.

| Slot | Family | Exact existing agent | Frozen accountability | Exclusive future ownership |
|---|---|---|---|---|
| K1 | Kiro | **TL-ORCHESTRATOR** | Sole integration owner; contract/governance lead; merge and release orchestration authority | Shared interface freeze; `server/cmd/server/router.go`; `server/pkg/protocol/{events,messages}.go`; top-level `server/internal/daemon/{daemon,client,wakeup}.go`; shared `execenv/{execenv,codex_home,kiro_home}.go`; final claim wiring |
| K2 | Kiro | **SENIOR-PLATFORM-AUTOMATION-SECURITY** | Daemon catalog and operations owner | `server/internal/daemon/credentialcatalog/**`; `server/internal/daemon/credential_home.go`; discovery, reconciliation, lifecycle, drift, watermarks, incident controls and their tests; never K1 shared daemon files |
| K3 | Kiro | **SENIOR-DATA-SEMANTIC** | Independent assurance reviewer | Baseline-risk, evaluation-verdict, Council-review and release-recommendation evidence only; may inspect and test but never edits producer files |
| C1 | Codex | **SENIOR-BACKEND-API** | Credential registry and security-boundary owner | `server/internal/credentialregistry/**`; registry/admin and daemon-reconciliation handlers; C1-owned unit/DB/handler tests; never `router.go` |
| C2 | Codex | **SENIOR-EXEC-DASHBOARD-DATA-VIZ** | Database, data-contract and migration owner | Migrations `128` and `130`-`134`; DB queries; all generated DB output; clean/upgrade/rollback fixtures; sole SQLC regeneration owner |
| C3 | Codex | **SENIOR-M365-INTEGRATION-ENGINEER** | Runtime Standard/configuration integration API owner | `server/internal/service/runtimeconfig/**`; `server/internal/handler/{runtime_standard,runtime_configuration}.go`; API unit tests; never `router.go` |
| C4 | Codex | **PRINCIPAL-FRONTEND-ARCH-DESIGN-SYSTEMS** | Runtime Manager UI/CLI and evaluation-spec owner | Runtime-manager `apps/web` pages/components; runtime configuration under `packages/views`; API types/client under `packages/core/api`; runtime-standard CLI under `server/cmd/multica`; C4 evaluation specs/fixtures |
| C5 | Codex | **SENIOR-QA-PERFORMANCE-A11Y-RELEASE** | Independent test/evidence harness owner | New cross-module integration/validation and evidence harnesses, performance/accessibility/release gates and end-to-end traces; never producer modules |

**Integration authority lock:** K1 is the sole integration authority. Only K1 may merge reviewed commits into `integration/spe9-runtime-manager-rc`, in dependency order, or edit shared hotspots. Producers submit reviewed commits or patch specifications; they do not edit K1 files. K3 and C5 cannot self-approve producer work. Every implementation stream must use its frozen branch/worktree and must not stage, reset, rebase, or commit another stream.

## Existing infrastructure and reservation invariants

1. Reuse the existing `orq2-credential-runtime-v1` daemon, installed provider adapters, and existing runtime rows. No replacement or alternate daemon and no new runtime infrastructure may be created.
2. Preserve the immutable ORQ2-dev baseline of **4 AGY (Antigravity) + 2 Codex + 2 Kiro exclusive assignments** exactly as-is. Their account/home/runtime bindings may not move, share, lend, borrow, or be stolen for the new workspace.
3. The new workspace row shape is exactly **3 Kiro + 5 Codex** persistent agents/logical runtimes. It does not alter the protected ORQ2-dev assignments.
4. There is a deficit of **at least two external independent Codex subscription homes**. A CTO-designated credential custodian, not an agent, must legitimately provision and attest them. Until dynamic discovery reports at least five healthy unassigned Codex homes, downstream integration is blocked.
5. No credential or credential home may be copied, moved, deleted, truncated, sanitized, overwritten, or shared. Existing `auth.json` files must never be copied. Agents must not execute authentication. Product APIs, logs, metrics, traces, and evidence expose opaque references only.
6. Running tasks pin assignment generation and configuration version. New configuration applies only to new tasks unless an approved hard-revoke policy explicitly requires termination.

## Frozen future stream/file/worktree ownership

| Issue / stream | Accountable | Exact future files/modules | Frozen worktree / branch |
|---|---|---|---|
| SPE-4 ownership/DAG lock | K1 | This ownership manifest and deferred documentation-phase payload only; no product files | `/home/ec2-user/workspace/worktrees/spe4-inventory-lock` / `plan/spe4-inventory-lock` |
| SPE-5 interface/OpenSpec freeze | K1 | `openspec/changes/credential-account-home-restoration/{proposal.md,design.md,tasks.md,specs/credential-account-home/spec.md}` and authority amendment in the conflicting OpenSpec only | `/home/ec2-user/workspace/worktrees/spe5-runtime-contracts` / `plan/spe5-runtime-contracts` |
| SPE-18 baseline risk | K3 | Independent review/evidence only | `/home/ec2-user/workspace/worktrees/spe18-baseline-risk` / `audit/spe18-baseline-risk` |
| SPE-6 R3 registry core | C1 | `server/internal/credentialregistry/**` plus its unit/DB tests | `/home/ec2-user/workspace/worktrees/spe6-r3-registry-core` / `feature/spe6-r3-registry-core` |
| SPE-6 runtime schema | C2 | `server/migrations/{128,130,131,132,133,134}_*`; `server/pkg/db/queries/{agent,credential_registry,runtime_standard,runtime_configuration}.sql`; all generated DB output | `/home/ec2-user/workspace/worktrees/spe6-runtime-schema` / `feature/spe6-runtime-schema` |
| SPE-6 manager API | C3 | New `server/internal/service/runtimeconfig/**`; `server/internal/handler/{runtime_standard,runtime_configuration}.go`; unit tests; no `router.go` | `/home/ec2-user/workspace/worktrees/spe6-runtime-manager-api` / `feature/spe6-runtime-manager-api` |
| SPE-6 manager UI/CLI | C4 | Runtime-manager `apps/web`; runtime configuration `packages/views`; API types/client `packages/core/api`; runtime-standard CLI `server/cmd/multica` | `/home/ec2-user/workspace/worktrees/spe6-runtime-manager-ui` / `feature/spe6-runtime-manager-ui` |
| SPE-6 eight-runtime fabric | K1 | Documentary integration/reconciliation manifest only until SPE-9; no producer files | `/home/ec2-user/workspace/worktrees/spe6-eight-runtime-fabric` / `plan/spe6-eight-runtime-fabric` |
| SPE-10 credential catalog | K2 | New `server/internal/daemon/credentialcatalog/**`; `server/internal/daemon/credential_home.go`; tests; no shared daemon files | `/home/ec2-user/workspace/worktrees/spe10-credential-catalog` / `feature/spe10-credential-catalog` |
| SPE-10 capacity closure | CTO-designated credential custodian | External legitimate isolated source homes only; no repository module and no agent authentication | No agent worktree |
| SPE-8 registry boundaries | C1 | `server/internal/handler/{credential_registry_admin,daemon_credential_reconciliation}.go`; C1 registry extensions and tests; no `router.go` | `/home/ec2-user/workspace/worktrees/spe8-registry-boundaries` / `feature/spe8-registry-boundaries` |
| SPE-7 evaluations | C4 | C4-owned evaluation specifications and fixtures only; no producer code | `/home/ec2-user/workspace/worktrees/spe7-golden-forbidden` / `test/spe7-golden-forbidden` |
| SPE-9 integration RC | K1 | `server/cmd/server/router.go`; `server/pkg/protocol/{events,messages}.go`; `server/internal/daemon/{daemon,client,wakeup}.go`; shared `execenv/{execenv,codex_home,kiro_home}.go`; final claim integration | `/home/ec2-user/workspace/worktrees/spe9-runtime-manager-rc` / `integration/spe9-runtime-manager-rc` |
| SPE-20 independent harness | C5 | New C5-owned cross-module integration/validation harness; no producer source | `/home/ec2-user/workspace/worktrees/spe20-runtime-harness` / `test/spe20-runtime-harness` |
| SPE-21 E2E evidence | C5 | C5-owned evidence harness/output only; no producer source | `/home/ec2-user/workspace/worktrees/spe21-e2e-evidence` / `evidence/spe21-e2e-trace` |
| SPE-19 evaluation verdict | K3 | Independent K3 review/evidence only | `/home/ec2-user/workspace/worktrees/spe19-eval-verdict` / `audit/spe19-eval-verdict` |
| SPE-24 recurring controls | K2 | Operations/control definitions and catalog metrics/tests; no credential-deletion tooling | `/home/ec2-user/workspace/worktrees/spe24-recurring-controls` / `ops/spe24-recurring-controls` |
| SPE-22 Council review | K3 | K3 Council packet only; producer files read-only | `/home/ec2-user/workspace/worktrees/spe22-council-review` / `review/spe22-council` |
| SPE-23 release packet | K3 | Release recommendation/evidence only | `/home/ec2-user/workspace/worktrees/spe23-release-packet` / `review/spe23-release-packet` |
| SPE-23 separately authorized rollout | K1 | No source editing; deploy only an exactly pinned accepted SPE-9 RC and additive schema | Existing release worktree; no new implementation branch |

## Complete SPE-4/5/6/7/8/9/10/18/19/20/21/22/23/24 DAG and greenlight gates

A successor remains blocked until **all** predecessors and its stated acceptance gate pass. A conditional architecture greenlight never advances an implementation or rollout gate.

| Node | Mandatory predecessors | Greenlight / acceptance gate | Current authorization |
|---|---|---|---|
| SPE-4 | CTO-provided issue and existing-agent mappings | Existing issue, accountable owner, non-overlapping module, worktree, dependency, reviewer and gate recorded; exactly eight persistent agent/runtime rows | Documentation lock authorized; no implementation |
| SPE-5 | SPE-4 accepted | Strict OpenSpec validation; no contradictory active authority; K3 signs frozen interfaces; every field has validation, precedence/delegability and hot/restart class | Planning only; implementation blocked |
| SPE-18 | SPE-4 plus independent evidence source | K3 independently proves no ORQ2-dev reassignment, hidden allowlist, path/ID disclosure, or estimate based on unverified capacity | Audit only |
| SPE-6 C1 registry | SPE-5 frozen contract | Exactly one workspace/runtime/agent/account/home binding; reviewed final R3 chain; no credential query; tests/race pass; revoked/stale fails closed | NO-GO |
| SPE-6 C2 schema | SPE-5; migration registrar recheck; isolated DB fixture | Clean install/upgrade and SQLC parity; actionable duplicate preflight; protected assignments unchanged; no destructive rollback | NO-GO |
| SPE-6 C3 API | SPE-5; C2 schema contract | Precedence/delegability enforced; unsupported model/reasoning/limits fail before activation; explicit apply class; rollback activates prior version | NO-GO |
| SPE-6 C4 UI/CLI | SPE-5 API contract; C3 mocks | No path/account identity; reuse default; custom path explicit; agent/runtime IDs stable on subscription change | NO-GO |
| SPE-6 fabric correction | All SPE-6 producer gates; SPE-18 | Exactly eight persistent agents/logical runtimes, 3 Kiro + 5 Codex; subagents excluded; same daemon/rows; no protected movement/sharing | Documentary correction only; implementation NO-GO |
| SPE-10 catalog | SPE-5 catalog/event contracts | Dynamic discovery without static slots or restart; invalid/duplicate quarantine; no source/active credential copy/move/delete | NO-GO |
| SPE-10 capacity | Owner authorization and legitimate external provisioning | At least five healthy unassigned Codex homes, including at least two additional independent homes; protected bindings/source homes unchanged | External owner action blocked/not delegated |
| SPE-8 | C1 registry; C2 schema; SPE-5 API contract | Cross-workspace/task actors denied; opaque/pathless responses; stale/revoked generation cannot claim or launch | NO-GO |
| SPE-7 | SPE-5 | Positive and negative fixtures for every mandatory path; forbidden action fails before mutation/process launch | Evaluation documentation only until approved |
| SPE-9 | All SPE-6, SPE-8 and SPE-10 producer gates; SPE-18; Codex capacity closure | K1-only merge; zero overlap; full build/vet/race/migration/integration gates; eight-runtime fixture; old/new compatibility; fail-closed rollback pinned | Merge and implementation NO-GO |
| SPE-20 | SPE-9 RC | Zero skips; named census; one winner under 100-way race; no path/secret leakage; compatibility/rollback pass; independent reproduction | NO-GO until RC |
| SPE-21 | SPE-9; operational SPE-20 harness | Reproducible opaque trace from session/workspace/runtime through config/home generation/task/result/rollback; no secret/path/account/prompt | NO-GO until predecessors |
| SPE-19 | SPE-7, SPE-20, SPE-21 | Independent K3 ACCEPT; no producer self-approval; all capabilities/reasoning/forbidden actions pass; unresolved evidence blocks | Review only after evidence |
| SPE-24 | SPE-10, SPE-20 | Every control has owner/cadence/evidence/escalation; watermarks tested; no automated active/source credential deletion | Design after predecessors; activation needs rollout approval |
| SPE-22 | SPE-18, SPE-19, SPE-20, SPE-21, SPE-24 design | Council ACCEPT; zero unresolved P0/P1; each residual P2 has owner, deadline, detection and rollback | Review only after complete packet |
| SPE-23 recommendation | SPE-22 ACCEPT; Codex capacity closed | Written final GREENLIGHT; exact RC/schema/config/evidence/capacity/rollback pinned; separate Multica rollout approval required; SharePoint remains blocked | Recommendation only |
| SPE-23 rollout | SPE-23 recommendation; separate written CTO production authorization; backup/restore rehearsal; zero active tasks | Verified backup/restore; additive order DB → server → existing daemon reconciliation; non-inference health/canary; rollback; protected assignments unchanged | **PRODUCTION NO-GO** |

Canonical dependency summary:

```text
SPE-4 -> SPE-5
SPE-4 -> SPE-18
SPE-5 -> SPE-6(C1,C2,C3,C4) and SPE-10(catalog) and SPE-7
SPE-6(all) + SPE-18 -> SPE-6(fabric correction)
SPE-6(C1,C2) + SPE-5 -> SPE-8
SPE-10(catalog) + SPE-10(external capacity) + SPE-6(all) + SPE-8 + SPE-18 -> SPE-9
SPE-9 -> SPE-20 -> SPE-21
SPE-7 + SPE-20 + SPE-21 -> SPE-19
SPE-10 + SPE-20 -> SPE-24
SPE-18 + SPE-19 + SPE-20 + SPE-21 + SPE-24 -> SPE-22
SPE-22 ACCEPT + SPE-10(capacity closed) -> SPE-23(recommendation)
SPE-23(recommendation) + separate CTO production authorization + backup/restore + zero active tasks -> SPE-23(rollout)
```

## SPE-6 mandatory documentary correction

After separate authorization, the obsolete title `reconcile isolated 24-agent fabric in staging` must become:

> **Reconcile the isolated 8-agent / 8-logical-runtime fabric on the existing `orq2-credential-runtime-v1` daemon in staging**

Its body and criteria must replace every 24-agent target with: an eight-persistent-agent/eight-logical-runtime hard ceiling; 3 Kiro + 5 Codex for the new workspace; nonpersistent subagents that inherit or use explicitly governed overrides; existing daemon/adapters/runtime rows only; no daemon/container/runtime/workspace/project/squad/agent creation; no ORQ2-dev movement or sharing; attach legitimate new subscriptions without recreating agents/runtimes; and “24 agents” retained only as obsolete documentary history. This manifest records the correction but does not edit the issue or call an API.

## Explicit no-go boundary

- **Implementation:** NO-GO until SPE-4 and SPE-5 gates close and written CTO implementation approval exists.
- **Production:** NO-GO until SPE-22 ACCEPT, SPE-23 recommendation, and separate written CTO rollout approval; no production action is part of this documentation phase.
- **Credentials/runtime/auth/daemon:** no action authorized; no copy, move, delete, retrieval, login, mutation, deployment, or lifecycle execution.
- **SPE-11 through SPE-17:** **PROHIBITED and unexecuted**. No assignment, status advancement, execution, rerun, delivery, SharePoint activity, or coupling is permitted until successful Multica rollout and observation plus a later, separate written CTO SharePoint approval.
- Missing evidence, missed gates, unresolved Codex capacity, contradictory OpenSpec authority, mixed-version safety, root safety, or backup/restore evidence fails closed and slips every dependent date.

## Documentation-phase disposition

SPE-4 producer documentation may be proposed for independent acceptance, but no Kanban mutation has been sent. The companion JSON is a deferred, not-sent operation plan with no invented issue UUID. Only after authorized UUID resolution and precondition checks may an operator apply the documented `in_progress` and comment operations; only after independent acceptance may the issue be proposed for `in_review`. This manifest itself grants no such execution authority.
