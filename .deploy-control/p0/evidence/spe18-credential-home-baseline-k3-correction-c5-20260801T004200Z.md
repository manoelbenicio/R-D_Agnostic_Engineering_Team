# SPE-18 — C5 correction, superseding the K3 baseline snapshot's blocker framing

- **Role:** K3 / independent evidence owner (evidence and review only; no producer authority)
- **Worktree / branch:** `/home/ec2-user/workspace/worktrees/spe18-baseline-risk` on `audit/spe18-baseline-risk`
- **Corrects:** `f37487ece188be5c4853e0b87f80d87713848293` — `.deploy-control/p0/evidence/spe18-credential-home-baseline-k3-snapshot-20260801T003526Z.md`
- **Correction authority:** SPE-18 C5
- **Disposition of the corrected commit:** **PRESERVED IMMUTABLE.** `f37487ece188` is not rewritten, amended, reverted or edited. It remains the authoritative record of the measured counts. This artifact supersedes only the four framing defects listed in §2 and MUST be read together with it.
- **Verdict after correction:** **PARTIAL PROVE / BLOCK ON UNESTABLISHED ORQ2 DISCOVERY** — the measured counts stand unchanged; the physical-capacity blocker is restated as *not established* rather than *contradicted*; frozen exact-eight is additionally blocked by the extra AGY runtime row.

## 1. Measured facts carried forward unchanged

These are accepted and are **not** modified by this correction. All were read inside one read-only `REPEATABLE READ` transaction (xmin `303600`, LSN `0/26DEC710`, clock `2026-08-01 00:35:26.409949+00`, PostgreSQL 17.10):

| Fact | Value | Status |
|---|---|---|
| Registered credential-home metadata rows | 8 registered / 8 assigned | unchanged |
| Unique normalized homes | 8 of 8 (0 duplicate groups) | unchanged |
| Healthy registered homes | 7 (AGY 4, Codex 1, Kiro 2) | unchanged |
| Eligible registered homes | 0 (all vendors) | unchanged |
| Protected ORQ2-dev split | 4 AGY + 2 Codex + 2 Kiro | unchanged, **PROVEN** |
| Exclusivity | 8 assignments / 8 accounts / 8 agents, 0 shared | unchanged |
| Target workspace `sp-exec-premium-8x-20260731-1715` | 9 runtime rows = 3 Kiro + 5 Codex + 1 AGY | unchanged |
| Generation readiness | 0 generation columns, 0 target catalog tables, 0 migrations in `130`–`134` | unchanged |

## 2. Corrections

### 2.1 Host identity — the probed host was ORQ1, not ORQ2

**Defect.** `f37487ece188` §1 labelled the database and frontend host as the "ORQ2 transition control plane", and §3.6 ground 3 described the directory probe as running on "the ORQ2 control-plane host `ip-172-31-18-217.sa-east-1.compute.internal`".

**Correction.** `ip-172-31-18-217.sa-east-1.compute.internal` / `100.118.244.61`, which runs container `multica-dev-transition-postgres-1`, the transition backend and the frontend, is **ORQ1**. The **ORQ2** model host is `ip-172-31-30-9.sa-east-1.compute.internal` / `100.110.178.47`, which is where the credential-home daemon runs. Committed evidence establishes both mappings independently:

- `.deploy-control/p0/evidence/gtl-cli-alignment-second-peer-review.md:20-21` — `ORQ1 = ip-172-31-18-217 / 172.31.18.217 / 100.118.244.61`; `ORQ2 = ip-172-31-30-9 / 172.31.30.9 / 100.110.178.47`.
- `.deploy-control/p0/evidence/gate0-tool-permission-preflight-report.md:8-9` — ORQ2 is the orchestration host `ip-172-31-30-9`; ORQ1 is the production target `100.118.244.61` (`ip-172-31-18-217`).
- `.deploy-control/MISSION_CHARTER_v3.md:5` — host of record ORQ2 `ip-172-31-30-9`, tailnet `100.110.178.47`.
- `.deploy-control/p0/checkins/CHECKOUT__Opus48#A__A__GTL-50-CLI-ALIGNMENT-SECOND-AUDIT__20260727T115800Z.json:12-13` — ORQ1 `ip-172-31-18-217` runs the four containers and **no** multica daemon unit or process; ORQ2 `ip-172-31-30-9` runs the active user unit `multica-daemon-orq2-credential.service` and the credential-home daemon process.

**Consequence.** The directory probe reported in `f37487ece188` §3.6 ground 3 ran against **ORQ1**, which by committed evidence hosts no credential-home daemon. It therefore **did not inventory ORQ2 physical homes at all**, and its `0` Kiro / `0` AGY / `≤1` Codex directory counts carry no information about ORQ2 physical capacity. That probe result is withdrawn as corroboration for the physical-capacity blocker. The database counts themselves are unaffected — the control plane is legitimately on ORQ1 and was read correctly; only its host label was wrong.

This review session itself runs on ORQ2 (`ip-172-31-30-9`, `100.110.178.47`), verified in-session. No substitute ad-hoc ORQ2 filesystem probe was performed, deliberately: an ad-hoc directory count cannot carry a generation identifier and would reproduce exactly the out-of-band, unversioned measurement class this review rejects. Physical ORQ2 capacity must come from authoritative discovery, not from a reviewer's `find`.

### 2.2 Registered gross is catalog state, not physical capacity

**Defect.** `f37487ece188` §3.6 ground 1 treated gross `kiro = 2` / `codex = 2` as evidence that `11 Kiro + 5 Codex` exceeds reality, and §4 recorded the claim as "BLOCKED (contradicted)".

**Correction.** `kiro = 2` and `codex = 2` are the **registered/enrolled catalog state** of the control plane at the snapshot. They are a count of rows in `accounts`, and nothing more. Because dynamic discovery does not exist yet (0 generation columns, 0 catalog tables, 0 applied migrations in `130`–`134`), the catalog cannot be assumed complete with respect to the filesystem: an unregistered but legitimate home is invisible to it by construction. Catalog state therefore **cannot contradict** a physical-capacity claim; it can only fail to confirm one.

### 2.3 Physical-capacity blocker restated

**Replaces** the `11 Kiro + 5 Codex` disposition in `f37487ece188` §3.6 and §4.

> **`11 Kiro + 5 Codex` is NOT ESTABLISHED by an authoritative same-generation ORQ2 discovery.**

Grounds, stated exactly:

1. No authoritative discovery mechanism exists at the snapshot: `0` generation/`catalog_generation`/`discovery_generation` columns, `0` of the target tables (`credential_home`, `credential_home_catalog`, `runtime_binding`, `runtime_standard`, `credential_home_generation`), `0` applied migrations in the reserved `130`–`134` range, max applied migration `129_task_usage_price_snapshot`. No figure of any kind can be attributed to a generation.
2. The registered catalog reports 2 Kiro and 2 Codex, which neither confirms nor refutes `11/5` (§2.2).
3. No ORQ2 physical inventory was taken by this review (§2.1).

The word **contradicted** is withdrawn. `11 Kiro + 5 Codex` is neither proven nor disproven; it is **unestablished**, and remains unusable as an allocation input until authoritative same-generation ORQ2 discovery reports it.

### 2.4 Eligible = 0 — scope of the claim narrowed

**Defect.** `f37487ece188` §3.4 and §3.8 permitted `eligible = 0` to be read as an absence of allocatable homes in general, and §3.8 asserted a `5 Codex + 3 Kiro` unmet requirement on that basis.

**Correction.** `eligible = 0` proves exactly one thing: **there is no currently allocatable *registered* credential home** — every one of the 8 registered homes is healthy-or-not but already exclusively assigned and protected, so none can be handed to a new workspace without rebinding a protected home. It does **not** prove the absence of legitimate unregistered homes on ORQ2 or elsewhere. Unregistered homes may exist and may be enrollable; this snapshot is blind to them.

The `5 Codex + 3 Kiro` figure in `f37487ece188` §3.8 is accordingly restated as a **registered-supply gap**, not a physical procurement requirement: the target workspace's runtimes have no registered home backing, and closing the gap may be satisfied either by enrolling existing legitimate unregistered homes or by provisioning new ones. Which of the two applies is unknown until discovery runs.

The withdrawal of "contradicted" in §2.3 does not soften the operational conclusion: with `eligible = 0`, any allocation to `sp-exec-premium-8x-20260731-1715` at this snapshot would still have to rebind a protected ORQ2-dev home and MUST fail closed before mutation.

## 3. New explicit blocker — the extra AGY runtime row breaks frozen exact-eight

**Status: BLOCKER, added by this correction.**

The requested envelope is `3 Kiro + 5 Codex = 8`, and the protected ORQ2-dev allocation is likewise `4 + 2 + 2 = 8`. The snapshot measures the target workspace `sp-exec-premium-8x-20260731-1715` at **9 runtime rows: 3 Kiro + 5 Codex + 1 AGY**, all online.

The **1 AGY row is unaccounted for by the eight-shaped envelope**. Three consequences, each independently sufficient to block a frozen exact-eight semantic:

1. **Arithmetic.** `3 + 5 + 1 = 9 ≠ 8`. The live population of the target workspace already exceeds the envelope by one before any credential home is allocated to it. An envelope stated as an exact eight is falsified by its own target workspace.
2. **Unbudgeted vendor.** The envelope names only Kiro and Codex. The AGY row introduces a third vendor with no stated demand, no protected-side headroom (all 4 AGY registered homes are protected ORQ2-dev bindings) and no registered eligible AGY home (`eligible = 0` for AGY). Its home requirement is undefined rather than zero.
3. **No enforcement.** `0` schema constraints bound any of `accounts`, `agent`, `agent_runtime`, `assignments`, `workspace` at `≤ 8`. Nothing prevents a tenth row. "Eight" is convention, and the ninth row demonstrates the convention is already not held.

Combined with the previously proven facts that total runtime rows are `15` across 3 workspaces and that `orq2-dev` carries `12` non-archived agents against `8` protected homes, **eight MUST NOT be treated as a frozen exact ceiling.** It is an observed enrollment and protection total only. Any plan, spec or gate that relies on exact-eight must first reconcile the extra AGY row and state whether AGY demand is zero, one, or unbounded.

## 4. Corrected disposition table

| Claim | Corrected disposition | Change from `f37487ece188` |
|---|---|---|
| Physical capacity `11 Kiro + 5 Codex` | **NOT ESTABLISHED** by authoritative same-generation ORQ2 discovery | was "BLOCKED (contradicted)"; "contradicted" withdrawn |
| Registered gross Kiro 2 / Codex 2 | **PROVEN as catalog state only** | reclassified; no longer capacity evidence |
| ORQ1 vs ORQ2 host labelling | **CORRECTED**: DB/frontend host `ip-172-31-18-217` is ORQ1; ORQ2 model host is `ip-172-31-30-9` | mislabelled as ORQ2 |
| ORQ1 directory probe as capacity corroboration | **WITHDRAWN** — did not inventory ORQ2 | was ground 3 of the block |
| Protected `4 AGY + 2 Codex + 2 Kiro` | **PROVEN** | unchanged |
| Registered gross 8 / unique 8 / healthy 7 / eligible 0 | **PROVEN** | unchanged |
| Meaning of `eligible = 0` | **PROVEN**: no currently allocatable *registered* home; silent on legitimate unregistered homes | scope narrowed |
| Exclusive assignability | **PROVEN** (0 shared) | unchanged |
| Frozen exact-eight ceiling | **BLOCKED**, now also by the extra AGY runtime row (9 = 3+5+1) | blocker strengthened and made explicit |
| Eight as observed enrollment/protection total | **PROVEN** | unchanged |
| `5 Codex + 3 Kiro` gap | **Registered-supply gap**, satisfiable by enrolment or provisioning | reclassified from procurement requirement |

## 5. Unblock conditions after correction

1. Authoritative discovery over validated ORQ2 controlled roots, emitting a monotonic generation identifier — requires migrations `130`–`134`, currently `0` applied.
2. A same-generation report of ORQ2 physical home counts per vendor, which is the only artifact that can establish or refute `11 Kiro + 5 Codex`.
3. A same-generation reconciliation of physical versus registered homes, so that legitimate unregistered homes are enumerated rather than assumed absent.
4. A non-zero eligible count per vendor at the same generation, with health, approval and non-assignment measured separately.
5. An explicit canonical ruling on whether any per-workspace runtime or home maximum exists, and on the AGY demand implied by the extra runtime row.
6. Re-measured protected counts, confirming `4 AGY + 2 Codex + 2 Kiro` is unchanged after any of the above.

## 6. Boundary

No credential content was read. This correction performed repository reads and one in-session hostname verification only; it issued no database write, no filesystem probe, no enrollment, no assignment, no rotation, no migration, no deployment and no push. No producer source, schema, OpenSpec artifact or configuration was created or edited. `f37487ece188` is preserved byte-for-byte. This artifact authorizes nothing; production remains NO-GO pending authoritative discovery, integrated gates and separate written owner authorization.
