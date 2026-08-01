# SPE-4 / SPE-5 Final Independent Documentation Review

**Reviewer:** K3 / `SPE45-FINAL-INDEPENDENT-REVIEW-K3-05`
**Review worktree:** `/home/ec2-user/workspace/worktrees/spe4-spe5-doc-review`
**Exact SPE-4 commit:** `e0e54d798d0eb742da653353f105489f1966317e`
**Exact reconciled SPE-5 HEAD:** `a57c12e8424cd88cda178499563b69569d244d84`
**Common base:** `2d91148a58fe1a7cb43fee2e54df1860535bd03d`
**Review scope:** documentation/static source inspection only; no API, authentication, secret, cloud, inference, runtime, daemon, container, database, SharePoint, deployment, or production action
**Final verdict:** **ACCEPT WITH CONDITIONS**

## Executive verdict

- **SPE-4 documentation gate: ACCEPT.** The exact ownership/capacity/topology/DAG/no-go contract remains complete and internally consistent.
- **SPE-5 reconciled documentation gate: ACCEPT.** The prior active-authority and deferred-payload defects are corrected at exact HEAD `a57c12e8424cd88cda178499563b69569d244d84`. Both active OpenSpecs strict-validate. The active normative surfaces jointly admit exactly one pinned `omniroute` or `native_credential_home` binding, with non-overlapping authority, binding-local readiness and recovery, no cross-binding or cross-home fallback, no universal native prohibition, no automatic source-home deletion, and path/account/credential secrecy.
- **Condition still open:** the exact producer branches are locally clean but have **no configured upstream**. The newly documented future pre-handoff/deploy gate requires a configured non-main upstream, exact local/upstream SHA equality, and `0/0` divergence. No publish/push was authorized or performed. This does not invalidate the documentation content because the contract expressly describes a future handoff/deployment gate, but it blocks any claim that the repository synchronization gate has already been operationally satisfied.
- **Implementation and production remain NO-GO.** This acceptance is documentation/interface acceptance only. Separate written implementation authorization, downstream assurance/Council gates, exact release pinning, and separate rollout authorization remain mandatory. SPE-11 through SPE-17 and SharePoint activity remain prohibited.

## 1. Exact topology, commit chain, and path scope

The commits are sibling workstreams from one base, not a single linear chain:

```text
2d91148a58fe1a7cb43fee2e54df1860535bd03d
├── e0e54d798d0eb742da653353f105489f1966317e  SPE-4
└── 8e548e274026fc88f078d82c9a8eb2a2361edcef  SPE-5 initial
    └── a57c12e8424cd88cda178499563b69569d244d84  SPE-5 reconciliation
```

Verified facts:

- `e0e54d7^` is the exact common base.
- `8e548e2^` is the exact common base.
- `a57c12e^` is exactly `8e548e274026fc88f078d82c9a8eb2a2361edcef`.
- SPE-4 is not an ancestor of SPE-5; their merge base is the exact common base.
- The review therefore accepts the exact pair of branch commits; it does not falsely claim that SPE-4 content is contained in the SPE-5 Git tree.

### SPE-4 exact path scope

`git diff-tree --root --no-commit-id --name-status -r e0e54d7...` reports exactly two additions:

1. `.deploy-control/p0/evidence/spe4-multica-adjustment-ownership-manifest.md`
2. `.deploy-control/p0/worklogs-draft/SPE-4-documentation-phase-payload.json`

Both are mode `100644`.

### SPE-5 cumulative base-to-reconciled-HEAD scope

`git diff --name-status 2d91148... a57c12e...` reports 23 documentation/JSON paths:

- one deferred payload JSON;
- root `README.md`;
- active `build-omniroute-agent-brain` proposal, design, tasks, six normative specs, and eight auxiliary/historical documents;
- the four canonical `credential-account-home-restoration` files.

Every path is mode `100644`; the only creation is the JSON payload. `git diff --summary` reports no executable creation or mode change. No `.go`, `.ts`, `.tsx`, `.js`, `.sql`, migration, shell script, YAML, deployment configuration, generated output, or product executable is changed by either reviewed result.

### Patch hygiene

The following all exited `0` with no whitespace diagnostics:

```text
git show --check --oneline e0e54d798d0eb742da653353f105489f1966317e^!
git show --check --oneline a57c12e8424cd88cda178499563b69569d244d84^!
git show --check --oneline e0e54d798d0eb742da653353f105489f1966317e..a57c12e8424cd88cda178499563b69569d244d84
```

## 2. Producer cleanliness and synchronization condition

Read-only producer checks immediately before writing this report:

| Producer worktree | Branch | Exact HEAD | Porcelain status | Upstream |
|---|---|---|---|---|
| `/home/ec2-user/workspace/worktrees/spe4-inventory-lock` | `plan/spe4-inventory-lock` | `e0e54d798d0eb742da653353f105489f1966317e` | empty / clean | unconfigured |
| `/home/ec2-user/workspace/worktrees/spe5-runtime-contracts` | `plan/spe5-runtime-contracts` | `a57c12e8424cd88cda178499563b69569d244d84` | empty / clean | unconfigured |

The review worktree had only this required review artifact untracked and no other staged, modified, or untracked path. This report is not staged or committed.

The reconciled contracts correctly document the synchronization gate in all required normative locations:

- `build-omniroute-agent-brain/design.md` migration plan item 6;
- `build-omniroute-agent-brain/specs/brain-cutover-operations/spec.md`, Repository/OpenSpec synchronization requirement;
- `build-omniroute-agent-brain/tasks.md` 7.3;
- `credential-account-home-restoration/design.md` section 13;
- `credential-account-home-restoration/spec.md` REQ-23;
- `credential-account-home-restoration/tasks.md` 5.4.

That gate requires: both strict validations, clean cross-authority scan, zero staged/modified/untracked owned files, exact commit published to a configured **non-main** upstream, matching local/upstream SHA, `0/0` ahead/behind, and deployment evidence pinned to that SHA. Because both current branches have no upstream, only the local-cleanliness part is presently proven. Any future handoff/deployment claim must remain fail closed until publication and SHA/divergence checks are independently evidenced.

## 3. Strict OpenSpec validation

Both validations were run from the read-only reconciled SPE-5 producer worktree:

```text
openspec validate credential-account-home-restoration --strict
Change 'credential-account-home-restoration' is valid
exit 0

openspec validate build-omniroute-agent-brain --strict
Change 'build-omniroute-agent-brain' is valid
exit 0
```

Strict validation is necessary but not treated as proof of cross-change semantic consistency; the independent residual review below supplies that check.

## 4. Deferred payload JSON and exact current source contract

### JSON results

Both exact payload blobs parse with `jq -e .`:

| Payload | SHA-256 | State |
|---|---|---|
| SPE-4 at `e0e54d7` | `888d5156f198b10c21fefc41ee493a4e7ff045329f77a4766def2c23df227e1e` | deferred, unsent, execution unauthorized, issue UUID null, zero requests |
| SPE-5 at `a57c12e` | `6ee9283c703150ecc48aac01d772fb32a97a0c99d7cd13fe8d444d4884c5b2ce` | `DEFERRED_UNSENT`, sent false, workspace/issue UUIDs null |

Recursive structural assertions pass for both payloads:

- at least one PUT exists;
- every PUT path/path-template ends with `/`;
- the comments route ends with `/comments`;
- every planned comments POST has explicit body `"type":"comment"`.

Exact planned operations are:

```text
PUT  /api/issues/{issueUUID}/           {"status":"in_progress"}
POST /api/issues/{issueUUID}/comments   {"content":"...","type":"comment"}
PUT  /api/issues/{issueUUID}/           {"status":"in_review"}
```

and equivalently with SPE-5 placeholder `{issue_uuid}`.

### Current source proof

Current product source in the producer tree registers:

```text
r.Route("/api/issues", ...)
r.Route("/{id}", ...)
r.Put("/", h.UpdateIssue)
r.Post("/comments", h.CreateComment)
```

`UpdateIssueRequest` exposes `Status *string json:"status"`. `CreateCommentRequest` exposes `Content string json:"content"` and `Type string json:"type"`; the handler also defaults an omitted type to `comment`, but the reconciled payload no longer relies on that default.

Both payloads prohibit invented UUIDs and require unique authoritative resolution, exact workspace/identifier verification, fresh-state checks, idempotency/read-back, and K3 acceptance before `in_review`. No request was sent.

## 5. SPE-4 original-requirement re-review

The exact SPE-4 manifest still satisfies the complete planning lock:

- exact aliases and existing agents: K1 `TL-ORCHESTRATOR`, K2 `SENIOR-PLATFORM-AUTOMATION-SECURITY`, K3 `SENIOR-DATA-SEMANTIC`, and C1-C5 with their exact named roles;
- exactly eight persistent agents/logical runtime rows, **3 Kiro + 5 Codex**; nonpersistent subagents inherit bounded snapshots and create no persistent rows;
- K1 sole integration/shared-hotspot authority; K3 and C5 independent and unable to self-approve producer work;
- exact non-overlapping future file/module/worktree ownership across SPE-4/5/6/7/8/9/10/18/19/20/21/22/23/24;
- reuse of `orq2-credential-runtime-v1`, installed adapters, and existing rows; no replacement daemon/runtime infrastructure;
- immutable ORQ2-dev baseline of **4 AGY (Antigravity) + 2 Codex + 2 Kiro** exclusive assignments, with no move/share/lend/borrow/steal;
- external deficit of at least two legitimate independent Codex homes remains owner/custodian work, never agent authentication or `auth.json` copying;
- complete predecessor DAG and greenlight gates through SPE-23 rollout and SPE-24 recurring controls;
- SPE-6 obsolete “24-agent” wording is frozen for later authorized correction to exactly eight agents/eight logical runtimes;
- credentials, runtime/auth/daemon changes, implementation, production, and SharePoint are explicit NO-GO;
- SPE-11 through SPE-17 remain prohibited until successful Multica rollout/observation plus later separate written CTO SharePoint approval.

**SPE-4 result: ACCEPT.**

## 6. SPE-5 original-requirement and reconciliation review

### Runtime Manager and catalog contract coverage

The exact reconciled canonical change retains all original SPE-5 requirements:

- owner-global immutable-versioned Runtime Standards;
- owner-global reusable accountless Runtime Sessions;
- enrollment onto existing workspace/agent/runtime rows and existing daemon without recreation;
- stable session/runtime/agent/daemon identity when subscriptions/homes attach;
- opaque dynamic arbitrary-child controlled-root discovery with no slot grammar or fixed allowlist;
- metadata-only layout validation, filesystem-identity deduplication, quarantine, hint/full-scan reconciliation, watcher overflow/loss recovery, monotonic atomic generations, TTL, watermarks, tombstones, and active-reference retention;
- one exclusive healthy native home per active persistent binding; concurrency remains policy-driven and never derives from account inventory;
- complete versioned configuration groups for transport/provider/subscription, literal model, reasoning, limits, timeout/retry, concurrency, flags, environment, skills, MCP/tools, filesystem/network/process permissions, eligibility, health, and bounded fallback;
- deterministic `platform > standard > runtime > explicitly delegable task` precedence, subagent inheritance, and no lower-layer widening;
- per-leaf schema/type, source/effective value, capability predicate, redaction, delegability, audit representation, and `hot|restart` apply class;
- pure non-inference validation; compare-and-swap activation; acknowledgement; immutable history; drift blocking; rollback by prior immutable activation; running-task pinning;
- immutable claim/reclaim snapshots of session/runtime/agent/workspace IDs, standard/config versions and digest, binding/catalog generations, optional opaque home ref, and capability digest;
- exact REST methods/paths/bodies, status codes, idempotency, pagination, error envelope, event envelope/type set, at-least-once deduplication, and generation-gap reconciliation;
- owner/admin/daemon/workspace authorization and cross-workspace denial;
- reserved additive migrations 130-134 subject to registrar recheck and active-reference-safe rollback;
- reuse of existing ORQ2 topology, no implementation/production authority, and SharePoint prohibition.

### Non-overlapping transport authority

Active normative surfaces now agree:

- exactly one immutable `TransportBinding` is required per launch: `omniroute` or `native_credential_home`;
- `omniroute`: OmniRoute is the sole inference router and account/credential owner, no native home is resolved, and strict selected route/model/protocol readiness is required;
- `native_credential_home`: R3 resolves exactly one approved exclusive opaque home for one existing logical runtime/agent, only daemon-local isolated-home references are supplied, and OmniRoute is not probed/contacted/used;
- missing, multiple, unknown, stale, unauthorized, unhealthy, unsupported, or conflicting authority fails closed before launch;
- recovery and rollback preserve the same pinned binding and never promote/switch/translate/remap to the other binding or another home;
- within-binding fallback is bounded/capability-validated but may not change transport, native `home_ref`, workspace, generation, or ORQ2-dev reservation.

Residual scanning found 49 exact binding declarations across the root README and active normative OpenSpec surfaces. No active normative requirement still says every model task must use OmniRoute or that native mode is universally forbidden.

### Auxiliary and operational documentation

All eight auxiliary documents inside `build-omniroute-agent-brain` begin with the same explicit banner: historical, superseded, non-normative; current authority is the two reconciled changes; old OmniRoute-only text applies only as history or within `omniroute` and must not be read as universally forbidding approved native mode or authorizing source-home mutation.

The current `docs/deploy/prod-rollout-runbook.md` and `rollback-runbook.md` are explicitly titled **Main Brain / OmniRoute** runbooks and describe selecting/deploying/restoring Main Brain plus OmniRoute. Their prohibition on switching to native execution is therefore binding-local rollback safety, consistent with the normative no-cross-binding-fallback rule, not a global prohibition on separately approved `native_credential_home`. Root `README.md` labels deployment runbooks binding-specific and separately documents native readiness.

Preserved `.planning` and `.deploy-control` evidence contains point-in-time OmniRoute-only statements, but those are execution/evidence history, not current normative transport authority. No executable instruction in the reconciled active changes delegates authority back to them.

### No source-home deletion and secrecy

The old automatic 24-hour slot-destruction requirement is removed from the active normative spec. Current authority uniformly requires:

- never copy, move, delete, truncate, sanitize, overwrite, chmod, ownership-change, authenticate, or automatically age-destroy a source credential home;
- TTL/retention may retire/tombstone metadata only;
- cleanup is limited to task-local non-source material after clear active-reference checks;
- raw source paths, isolated-home paths, global HOME, filesystem identity, account identity, credential names/data, argv/env values, prompts, provider responses, and secrets stay out of APIs, events, logs, metrics, traces, diagnostics, errors, and evidence.

A scoped scan of active normative surfaces returned no `destroyed automatically`, `deleted automatically`, `automatic deletion`, or `older than 24h` requirement.

### Fail-closed, ORQ2-dev, release, and SharePoint boundaries

Both active changes require fail-closed admission and same-binding recovery. Global HOME, cross-workspace, stale generation, replacement-home, cross-binding, and ORQ2-dev fallback are forbidden. Existing ORQ2-dev bindings are immutable. Documentation completion does not imply implementation, production, inference, credential provisioning, or SharePoint authority. Independent gates, Council acceptance, exact release pinning, and separate rollout approval remain mandatory.

**SPE-5 result: ACCEPT.**

## 7. Final gate matrix

| Check | SPE-4 `e0e54d7` | SPE-5 `a57c12e` |
|---|---:|---:|
| Exact object/parent chain | PASS | PASS (`base → 8e548e2 → a57c12e`) |
| Documentation/JSON-only path scope | PASS | PASS |
| No executable or executable-bit change | PASS | PASS |
| `git show --check` | PASS | PASS |
| Producer exact HEAD and clean worktree | PASS | PASS |
| Both affected OpenSpecs strict-valid | N/A | PASS |
| Payload valid JSON / unsent / no fake UUID | PASS | PASS |
| Exact PUT trailing slash | PASS | PASS |
| Explicit comment `type:"comment"` | PASS | PASS |
| Original SPE-4 requirements | PASS | Preserved |
| Original SPE-5 requirements | N/A | PASS |
| Non-overlapping `omniroute` / `native_credential_home` authority | N/A | PASS |
| No universal native prohibition | N/A | PASS |
| No automatic source-home deletion | PASS | PASS |
| Fail-closed / no cross-binding fallback | PASS | PASS |
| Path/account/credential secrecy | PASS | PASS |
| ORQ2-dev reservation | PASS | PASS |
| SharePoint boundary | PASS | PASS |
| Local/upstream cleanliness/SHA gate documented | N/A | PASS |
| Current local producer cleanliness | PASS | PASS |
| Current configured upstream SHA equality / 0/0 | **NOT ESTABLISHED** | **NOT ESTABLISHED** |
| Documentation/interface gate | **ACCEPT** | **ACCEPT** |
| Implementation authorization | **NO-GO** | **NO-GO** |
| Production/SharePoint authorization | **NO-GO** | **NO-GO** |

## 8. Exact remaining conditions

1. **Before any handoff/deployment claim:** configure an authorized non-main upstream for the applicable exact accepted branch, publish the exact commit without rewriting it, prove local HEAD equals upstream SHA, prove ahead/behind `0/0`, prove zero staged/modified/untracked owned files, rerun both strict validations and the cross-authority scan, and pin deployment evidence to that SHA. Current upstream state is unconfigured; this condition is not yet met.
2. **Before implementation:** obtain separate written CTO implementation authorization and preserve the frozen ownership/DAG, migration registrar recheck, capacity, ORQ2-dev, secrecy, and independent-review gates.
3. **Before production:** complete the SPE-18 through SPE-24 assurance/Council/recommendation chain, close legitimate capacity, pin the accepted RC/schema/config/rollback evidence, complete backup/restore and zero-active-task gates, and obtain separate written rollout authorization.
4. **Before any SharePoint SPE-11 through SPE-17 activity:** complete successful Multica rollout and observation and obtain a later, separate written CTO SharePoint approval.

## Final disposition

**ACCEPT WITH CONDITIONS** the exact reconciled SPE-4/SPE-5 documentation pair for interface/planning purposes. The two prior SPE-5 documentation blockers are closed: active authority is reconciled and the payload now uses exact trailing-slash PUT paths plus explicit comment type. The only current repository condition observed is the absence of configured upstreams; therefore no synchronized handoff/deployment claim may be made yet. This verdict authorizes no API mutation, implementation, credential/runtime action, deployment, production operation, or SharePoint work.
