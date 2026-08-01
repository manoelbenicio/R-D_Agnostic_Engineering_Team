# Unified SPE-4 / SPE-5 Authority Baseline

**Authority owner:** K1 / TL-ORCHESTRATOR
**Purpose:** one ancestry-complete authority for all Multica implementation and review
**Included accepted histories:**

- SPE-4 ownership/DAG lock: `e0e54d798d0eb742da653353f105489f1966317e`
- SPE-5 reconciled OpenSpec authority: `a57c12e8424cd88cda178499563b69569d244d84`
- K3 independent documentation review: `eb2668e61271c641e4265f9945074d745e6a6129`

This branch merges the three accepted histories without rebasing, cherry-picking, amending, or rewriting any accepted SHA. Commit `88e2f2255995aeb56878c2268cb654d6107a915e` is independently accepted by C5 as the clean ancestry repair for the SPE-5 and SPE-4 parents only. The later K3-review merge and this authority addendum require independent acceptance as one final descendant before implementation integration. That accepted descendant, rooted at `88e2f225`, is the only authority baseline permitted for implementation acceptance and K1 integration; sibling producer bases are never integration roots.

## Canonical accountability and module locks

The SPE-4 mapping is authoritative. Engineering agents in Herdr panes may execute a delegated bounded task, but pane identity or model does not change the accountable owner or file lock.

| Slot | Exact accountable agent | Frozen responsibility | Exclusive files/modules |
|---|---|---|---|
| K1 | TL-ORCHESTRATOR | Sole integration, shared contracts, merge, RC and rollout authority | Router, protocol, top-level daemon/client/wakeup, shared execenv and final claim wiring |
| K2 | SENIOR-PLATFORM-AUTOMATION-SECURITY | Credential catalog and operations | `server/internal/daemon/credentialcatalog/**`, credential-home discovery/reconciliation/lifecycle/drift/watermarks/tests; no K1 shared files |
| K3 | SENIOR-DATA-SEMANTIC | Independent assurance | Evidence/review only; never producer source |
| C1 | SENIOR-BACKEND-API | R3 credential registry and registry security boundaries | `server/internal/credentialregistry/**`, credential registry/admin/reconciliation handlers and tests; no router |
| C2 | SENIOR-EXEC-DASHBOARD-DATA-VIZ | Schema, migrations and generated DB | Migrations 128 and 130-134, DB queries, generated DB output and DB fixtures |
| C3 | SENIOR-M365-INTEGRATION-ENGINEER | Runtime Standard/configuration service and API | `server/internal/service/runtimeconfig/**`, runtime standard/configuration handlers and tests; no router |
| C4 | PRINCIPAL-FRONTEND-ARCH-DESIGN-SYSTEMS | Runtime Manager UI/CLI and SPE-7 evaluations | Web/pages/components, views, API client/types, runtime CLI and evaluation fixtures |
| C5 | SENIOR-QA-PERFORMANCE-A11Y-RELEASE | Independent harness and release assurance | Cross-module harness/evidence only; never producer modules |

Real pane agents assigned to SPE-8 execute under C1 accountability. Real pane agents assigned to UI or SPE-7 execute under C4 accountability. A cross-host agent assigned to catalog executes under K2 accountability. C5 and K3 remain independent and do not become producers.

## Superseded conflicting mapping

The role table in commit `677754a9cd9585c6e1cb809c034877840ef7aa3b` (`spe6-eight-runtime-fabric-initial.md`) is invalid where it assigns K2 to UI, C4 to registry boundaries, C5 to catalog, or K3 as the SPE-7 owner. Those assignments are superseded by this baseline and must not be used for acceptance, integration, release or deployment. A follow-up correction is required on that branch before its evidence can be accepted.

## Mandatory dependency and acceptance order

Speculative implementation may continue only inside frozen disjoint modules. Acceptance and merge follow this order:

```text
unified SPE-4 + SPE-5 + K3 authority baseline
  -> SPE-18 independent baseline gate
  -> C1 registry core and C2 schema (parallel)
  -> C1-owned SPE-8 boundaries and K2 catalog (parallel after compatible C1/C2 interfaces)
  -> C3 Runtime Standard/configuration service
  -> C4 Runtime Manager UI/CLI
  -> C4 SPE-7 executable evaluations
  -> C5/K3 independent acceptance
  -> K1 integration and immutable RC
```

Rules:

1. No downstream branch is accepted or merged before all predecessors have reviewed immutable SHAs.
2. Preparation against frozen interfaces may proceed early, but does not become accepted evidence.
3. K1 alone merges the authority baseline and producer commits.
4. Before producer acceptance, K1 merges this authority branch into the clean producer branch without rewriting the producer commit, then the producer reruns only its targeted gate and reports the new clean merge SHA.
5. OAuth-blocked or credit-blocked reviewers provide no acceptance. K1/C5 must assign an operational independent reviewer.
6. The effective configuration digest wire/storage form is raw lowercase 64 hexadecimal characters. The `sha256:` prefix is forbidden.
7. Catalog acceptance requires durable tombstone/non-reuse state, active-reference protection, used TTL and missing/draining/retired states, working hint/full reconciliation with overflow recovery, non-truncating watermarks, and launch-time metadata/path revalidation. Process-memory tombstones or silent healthy-entry truncation are not acceptable.

## Current merge prohibitions

- Do not merge catalog commit `ada64779f4cf` by itself.
- Do not merge runtime-config commit `496c0e7463fe84e2da437d5c135b1bc924de4fa1` while it emits or tests `sha256:`-prefixed digests.
- Do not merge SPE-18 commit `f37487ece188` as capacity proof. Preserve it and require a superseding artifact that identifies ORQ1 as the DB/frontend host and ORQ2 (`ip-172-31-30-9`) as the model host, treats physical capacity as not established without authoritative same-generation ORQ2 discovery, reports eligible-zero only as no allocatable registered homes, and makes the target workspace's current 3 Kiro + 5 Codex + 1 AGY row shape an explicit exact-eight blocker.
- Do not accept the conflicting SPE-6 role table in `677754a9cd9585c6e1cb809c034877840ef7aa3b`; only its follow-up correction commit may supersede it.
- Do not start K1 integration until the order above is satisfied.

GitLab Runner and SharePoint SPE-11 through SPE-17 remain excluded. No credential-home mutation, new runtime infrastructure, ORQ2 change, or production rollout is authorized by this authority document.
