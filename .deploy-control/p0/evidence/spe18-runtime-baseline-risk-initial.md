# SPE-18 runtime baseline risk — initial independent K3 evidence

- **Role:** K3 / Senior Data-Semantic, independent evidence review
- **First write (UTC):** `2026-07-31T23:35:23Z`
- **Frozen base:** `a57c12e8424cd88cda178499563b69569d244d84`
- **Scope:** one non-production evidence artifact only; no producer-code or OpenSpec edits
- **Verdict:** **BLOCK as a verified allocation baseline; CONDITIONAL as planning arithmetic**

## Executive finding

The requested envelope is internally consistent: a new workspace requests `3 Kiro + 5 Codex = 8` runtimes; the protected ORQ2-dev allocation is stated as `4 AGY + 2 Codex + 2 Kiro = 8`; and, if physical capacity is `11 Kiro + 5 Codex`, subtracting the protected Kiro and Codex homes yields upper bounds of `9 Kiro + 3 Codex`. That conditional Codex upper bound is at least two below demand: `5 - 3 = 2`.

The committed base does **not**, however, independently establish every premise. It proves eight observed runtime rows, canonical protection of existing ORQ2-dev bindings, four active AGY agents in the operational report, and the required secrecy/dynamic-discovery rules. It does not turn the observed row count into a contractual hard ceiling, does not state the requested complete `4/2/2` protected split, and does not contain a count-only source for `11 Kiro + 5 Codex` physical homes. The dedicated capacity snapshot measures host compute, memory, storage and pressure; it is not a provider-home inventory. Older committed account-home evidence reports only five Kiro homes and leaves Codex as an unspecified count. Therefore the arithmetic is verified, but the allocation inputs remain provenance-blocked and MUST NOT authorize enrollment or reassignment.

## Committed evidence and claim status

| Claim | Independent status | Evidence and interpretation |
|---|---|---|
| Eight-runtime ceiling | **PARTIAL** | `.deploy-control/p0/evidence/gemini-selector-p0-production-repair-20260727.md:27` records eight runtime rows and six online rows across AGY, Codex and Kiro in two workspaces. This verifies an observed eight-row topology, not a normative ceiling. No reviewed canonical requirement states that eight is a hard maximum. |
| Proposed new-workspace allocation: 3 Kiro + 5 Codex | **INPUT ONLY** | The requested split totals eight, but no reviewed committed artifact on the frozen base states this allocation. It is retained as the planning demand, not reported as deployed or authorized. |
| Protected ORQ2-dev allocation: 4 AGY + 2 Codex + 2 Kiro | **PARTIAL** | `openspec/changes/credential-account-home-restoration/design.md:18` makes existing ORQ2-dev bindings immutable, and the canonical spec requires protected-binding enrollment to fail before mutation. The operational report corroborates four active AGY agents, but the reviewed committed evidence does not establish the complete protected `4/2/2` split as one authoritative snapshot. |
| Physical capacity: 11 Kiro + 5 Codex | **BLOCKED** | `.deploy-control/p0/reports/orq-capacity/snapshots/orq-capacity-20260728T155304Z.v1.json` is a read-only host-pressure snapshot, not a provider-home count. Legacy committed evidence reports five Kiro homes and an unspecified Codex count, so it cannot substantiate `11/5`. A newer immutable count-only catalog snapshot is required. |
| Unassigned upper bounds: 9 Kiro + 3 Codex | **CONDITIONAL PASS** | Arithmetic passes only if `11/5` physical capacity and protected `2/2` Kiro/Codex consumption are independently established: `11 - 2 = 9`; `5 - 2 = 3`. These are upper bounds, not guarantees of health, uniqueness, eligibility or availability. |
| External Codex-home deficit | **CONDITIONAL BLOCK** | Demand is five while the conditional unassigned upper bound is three, so `5 - 3 = 2`: at least two additional eligible external Codex homes are required. “At least” is mandatory because an upper bound can fall after health, uniqueness, ownership, eligibility and assignment checks. |

## Semantic boundaries

1. **Runtime rows are not physical credential homes.** An eight-row runtime observation neither proves eight unique homes nor establishes provider-specific physical capacity.
2. **Capacity is not eligibility.** The `11/5` planning premise, if later evidenced, remains a gross physical count. A home is allocatable only after dynamic discovery, metadata-only validation, uniqueness, health, ownership, provider compatibility, exclusive assignment and current-generation checks.
3. **Unassigned is an upper bound.** Subtraction alone does not prove that nine Kiro or three Codex homes can launch work.
4. **Protected means unavailable to the new allocation.** Existing ORQ2-dev bindings cannot be moved, shared, borrowed, retired or silently rebound. Enrollment that references one must fail before mutation.

## Safety and design findings

### No raw paths or credential contents

The review read committed aggregate/control metadata only and did not read credential contents. This artifact contains repository-relative evidence references, opaque counts and arithmetic only; it reproduces no filesystem home, account identity, credential filename/data, token, cookie, environment value, process argument, prompt or provider response. The canonical contract reinforces this boundary in `openspec/changes/credential-account-home-restoration/specs/credential-account-home/spec.md:24,66-73,172`.

### No static allowlists

Historical operational evidence uses fixed home selections, but that mechanism is not valid target authority. The canonical target requires arbitrary-child controlled-root discovery, periodic/full reconciliation, overflow recovery, immutable opaque references and monotonic generations. `openspec/changes/credential-account-home-restoration/design.md:120-131` and the canonical spec at `:64-82` expressly reject fixed slot grammar, fixed capacity assumptions and per-home allowlists. Consequently, neither the legacy five-Kiro aggregate nor any enumerated home list may be converted into a static production allowlist.

## Migration risk

Migration identities `130` through `134` are reserved in the design for runtime standards, sessions/enrollments, credential-home catalog, runtime bindings and configuration snapshots. The reservation is conditional: C2 MUST recheck the canonical registrar immediately before implementation (`design.md:43-51`; canonical spec `:220-224`). Any collision blocks implementation and requires a canonical amendment before code or schema work. This review performed no migration, registrar mutation or implementation work.

## Authority risk

This evidence does not authorize producer-code changes, schema changes, enrollment, reassignment, credential provisioning, runner activity, deployment or production rollout. The proposal limits this wave to documentation/control-plane work and keeps production NO-GO pending independent K3 acceptance, integrated gates, Council acceptance and separate written owner authorization (`proposal.md:65-80`; `tasks.md:73-74`). No push, production action, runner use, credential read or SharePoint operation occurred.

## Decision and unblock conditions

**Decision:** block full `3 Kiro + 5 Codex` allocation as a verified baseline. Bounded non-production work may proceed only where it preserves protected bindings, uses opaque dynamic discovery, fails closed before mutation and does not depend on unsupported capacity claims.

Unblock requires one immutable committed count-only catalog snapshot, captured for the same generation, that:

1. proves gross physical counts of `11 Kiro` and `5 Codex` without exposing raw paths or account identities;
2. proves the protected ORQ2-dev split of `4 AGY + 2 Codex + 2 Kiro` from authoritative bindings;
3. distinguishes gross, protected, unassigned, healthy, unique, eligible and exclusively assignable homes;
4. shows that the eight-runtime value is either a canonical ceiling or only an observed topology; and
5. confirms at least two additional eligible external Codex homes before promising all five requested Codex allocations.

Until then, `9 Kiro / 3 Codex` and the two-home Codex deficit are valid **conditional risk bounds**, not deployment facts.
