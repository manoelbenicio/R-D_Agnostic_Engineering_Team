# SPE-18 — K3 immutable count-only credential-home baseline snapshot

- **Role:** K3 / independent evidence owner (no producer authority)
- **Worktree / branch:** `/home/ec2-user/workspace/worktrees/spe18-baseline-risk` on `audit/spe18-baseline-risk`
- **Frozen base:** `80e7924d6859b2a15dc468e6eb30fad6a932e1dc`
- **Artifact class:** immutable, append-never; a correction requires a new timestamped artifact, never an edit of this one
- **Scope:** one non-production evidence artifact; no producer source, schema, OpenSpec or configuration edit; no push; no production action
- **Verdict:** **PARTIAL PROVE / EXPLICIT BLOCK** — protected split and the healthy/unique/eligible distinctions are proven; `11 Kiro + 5 Codex` physical capacity is **BLOCKED**; the eight-value is proven to be an **observed enrollment total, not a canonical ceiling**

## 1. Snapshot identity (single generation)

Every count in section 3 was read inside **one** `BEGIN TRANSACTION READ ONLY ISOLATION LEVEL REPEATABLE READ` … `COMMIT`, so all figures share one MVCC snapshot and cannot be interleaved with concurrent writes.

| Field | Value |
|---|---|
| Snapshot clock (UTC) | `2026-08-01 00:35:26.409949+00` |
| Snapshot xmin | `303600` |
| WAL LSN at snapshot | `0/26DEC710` |
| Isolation / mode | `repeatable read` / `read only` |
| Server | PostgreSQL `17.10` (Debian), database `multica_transition` |
| Control-plane host | ORQ2 transition control plane, container `multica-dev-transition-postgres-1` |
| Deployed backend image | `multica-backend:orq134154-edd7b93-20260730` |
| Max applied migration | `129_task_usage_price_snapshot` (166 rows applied) |

**Generation caveat, stated up front:** the canonical design's monotonic credential-home *generation* does not exist yet in this schema. The snapshot proves `0` columns named `generation`/`catalog_generation`/`discovery_generation`, `0` of the target tables (`credential_home`, `credential_home_catalog`, `runtime_binding`, `runtime_standard`, `credential_home_generation`), and `0` applied migrations in the reserved `130`–`134` range. "Same generation" is therefore established here by MVCC snapshot identity (xmin + LSN), which is the strongest available equivalent — not by a canonical generation counter.

## 2. Secrecy and authority boundary

No credential content was read. This artifact was produced from aggregate control-plane metadata only. It emits **counts, status labels, vendor labels, workspace slugs, timestamps and a snapshot identity** and nothing else: no `home_dir` or `config_dir` path, no account identifier, no `secret_ref`, no token, cookie, environment value, process argument, prompt or provider response. Uniqueness and root-grouping were computed **inside SQL** as `count(DISTINCT …)` so path values never entered the reviewer's context or this file. `credentials.secret_ref` was counted, never selected. No producer source file was edited or created. No enrollment, assignment, rotation, migration, deployment or push occurred.

## 3. Count-only measurements

### 3.1 Gross enrolled homes

| Vendor | Gross homes | `available` | `degraded` |
|---|---|---|---|
| antigravity | 4 | 4 | 0 |
| codex | 2 | 1 | 1 |
| kiro | 2 | 2 | 0 |
| **total** | **8** | **7** | **1** |

Vendor domain size: `3`. Rows with missing/empty `home_dir`: `0`.

### 3.2 Uniqueness

| Measure | Value |
|---|---|
| Home rows | 8 |
| Distinct normalized `home_dir` | 8 |
| Distinct normalized `config_dir` | 8 |
| Distinct parent roots | 8 |
| Duplicate-home groups | 0 |
| Distinct normalized homes, per vendor | antigravity 4/4, codex 2/2, kiro 2/2 |

Uniqueness holds exactly: gross = unique, no collisions, no shared parent directory.

### 3.3 Healthy

Health predicate: `status = 'available'` **AND** no active cooldown **AND** no recorded `last_error`.

| Vendor | Healthy | Not healthy |
|---|---|---|
| antigravity | 4 | 0 |
| codex | 1 | 1 |
| kiro | 2 | 0 |
| **total** | **7** | **1** |

Accounts in active cooldown: `0`. Accounts carrying a non-empty `last_error`: `1` (the single `degraded` codex home). Healthy is therefore strictly smaller than gross and unique: `7 < 8`.

### 3.4 Eligible

Eligibility predicate: healthy **AND** approved (`approved_accounts.allowed IS TRUE`) **AND** not exclusively assigned.

| Measure | Value |
|---|---|
| Approval rows / allowed rows | 8 / 8 |
| Approved-allowed by vendor | antigravity 4, codex 2, kiro 2 |
| Assignment rows / distinct accounts / distinct agents | 8 / 8 / 8 |
| Accounts shared by more than one agent | 0 |
| Unassigned accounts (any vendor) | **0** |
| **Eligible total** | **0** |

Eligible collapses to zero for every vendor. Approval is universal and exclusivity is intact (a strict 1:1 account↔agent mapping, zero sharing), but because all eight homes are already exclusively assigned, **no home is allocatable to a new workspace at this snapshot**. This proves the three distinctions are materially different at the same instant: gross 8 → unique 8 → healthy 7 → eligible 0.

### 3.5 Protected ORQ2-dev split

From authoritative bindings (`assignments` joined to `agent` joined to `workspace` where `slug = 'orq2-dev'`):

| Vendor | Protected accounts | Distinct bound agents |
|---|---|---|
| antigravity (AGY) | 4 | 4 |
| codex | 2 | 2 |
| kiro | 2 | 2 |
| **total** | **8** | **8** |

Accounts assigned to any agent **outside** `orq2-dev`: `0`.

**Status: PROVEN.** The protected split is exactly `4 AGY + 2 Codex + 2 Kiro = 8`, it accounts for 100% of enrolled homes, and every protected home is bound to exactly one distinct agent.

### 3.6 Physical capacity claim `11 Kiro + 5 Codex`

**Status: BLOCKED — contradicted, not merely unproven.** Three independent grounds inside or adjacent to this snapshot:

1. The authoritative catalog reports gross `kiro = 2` and `codex = 2` at this snapshot. `11` and `5` exceed the enrolled population by `9` and `3` respectively.
2. No same-generation provider-home inventory can exist by construction: dynamic discovery, the credential-home catalog tables and the generation counter are all absent (section 1). Any `11/5` figure is necessarily out-of-band, unversioned and un-reconcilable against a generation.
3. A bounded, count-only, read-nothing directory probe on the ORQ2 control-plane host (`ip-172-31-18-217.sa-east-1.compute.internal`, `2026-08-01T00:35:55Z`) found `0` Kiro-home directories, `0` Antigravity-home directories and `≤1` Codex-home directory under the probed roots. This probe is explicitly **outside** the MVCC snapshot and is reported as corroboration only, not as inventory. It scanned this host only; homes on other machines remain unenumerable from any authoritative same-generation source.

`11 Kiro + 5 Codex` MUST NOT be used as an allocation input.

### 3.7 Eight-ceiling semantics

**Status: PROVEN to be an observed total, NOT a canonical ceiling.**

| Probe | Result |
|---|---|
| Enrolled homes | 8 |
| Protected ORQ2-dev homes | 8 |
| DB constraints bounding a count at `≤ 8` on `accounts`/`agent`/`agent_runtime`/`assignments`/`workspace` | **0** |
| `agent_runtime` rows (all online) | **15** |
| Runtime rows by workspace | `orq2-dev` 3, `html-sharepoint` 3, `sp-exec-premium-8x-20260731-1715` 9 |
| Non-archived agents by workspace | `orq2-dev` 12, `html-sharepoint` 5, `sp-exec-premium-8x-20260731-1715` 8 |
| Workspaces | 3 |

The number eight is the enrolled-home total and the protected-home total. It is *not* a runtime ceiling: 15 runtime rows exist, one workspace alone carries 9. It is *not* a schema invariant: zero constraints bound anything at eight. It is *not* an agent ceiling: `orq2-dev` has 12 non-archived agents against 8 protected homes, so the agent population already exceeds the home population by 4 within the protected workspace itself. Any document treating "eight" as a hard maximum is asserting a convention, not a measured or enforced limit.

### 3.8 New-workspace exposure (incidental finding, same snapshot)

Workspace `sp-exec-premium-8x-20260731-1715` already carries `3 kiro + 5 codex + 1 antigravity = 9` online runtime rows and `8` non-archived agents, while `AGENTS_WITH_ACCOUNT_ASSIGNMENT` returns rows for `orq2-dev` only. The requested `3 Kiro + 5 Codex` shape is therefore **already materialized at the runtime layer with zero credential-home backing**.

This supersedes the earlier "at least two additional external Codex homes" estimate. With eligible = 0 for every vendor, the true unmet requirement at this snapshot is the full **5 Codex + 3 Kiro** eligible homes, plus whatever the extra antigravity runtime row implies. The earlier `9 Kiro / 3 Codex` unassigned upper bound is now **falsified**: measured unassigned is `0`, not `9/3`, because it was derived from the unproven `11/5` premise.

## 4. Claim disposition

| Claim | Disposition | Basis |
|---|---|---|
| Physical capacity `11 Kiro + 5 Codex` | **BLOCKED (contradicted)** | §3.6 — gross is 2/2; no generation/discovery exists; host probe found none |
| Protected `4 AGY + 2 Codex + 2 Kiro` | **PROVEN** | §3.5 — authoritative bindings, 8/8 accounts, 8 distinct agents, 0 external |
| Healthy ≠ unique ≠ eligible | **PROVEN (distinct)** | §3.2–3.4 — gross 8, unique 8, healthy 7, eligible 0 |
| Exclusive assignability | **PROVEN** | §3.4 — 8 assignments, 8 distinct accounts, 8 distinct agents, 0 shared |
| Eight is a canonical ceiling | **BLOCKED** | §3.7 — 0 constraints, 15 runtimes, 12 agents in an 8-home workspace |
| Eight is an observed enrollment total | **PROVEN** | §3.7 — enrolled = protected = 8 |
| Unassigned upper bound `9 Kiro / 3 Codex` | **FALSIFIED** | §3.8 — measured unassigned = 0 |
| Deficit "at least 2 Codex" | **SUPERSEDED / WORSE** | §3.8 — eligible = 0, so 5 Codex + 3 Kiro are unmet |

## 5. Consequences and unblock conditions

This artifact does **not** authorize enrollment, reassignment, credential provisioning, schema work, deployment or production rollout. Protected ORQ2-dev bindings remain immutable; the snapshot shows no spare capacity, so any allocation to `sp-exec-premium-8x-20260731-1715` would necessarily rebind a protected home and MUST fail closed before mutation.

To convert the blocked items into deployment facts, a future artifact must supply, in one generation:

1. an authoritative provider-home inventory produced by controlled-root dynamic discovery, carrying a monotonic generation identifier (requires migrations `130`–`134`, currently `0` applied);
2. enrollment of the additional homes, so gross Kiro/Codex counts are measurable rather than asserted;
3. a non-zero eligible count per vendor at the same generation, with health, approval and non-assignment each measured separately;
4. an explicit canonical statement of whether any per-workspace runtime/home maximum exists, since none is currently enforced; and
5. re-measured protected counts, to confirm the `4/2/2` split is unchanged after any of the above.

## Appendix A — reproduction

Executed as a single read-only transaction against the control plane; output is count-only. Key measurement statements, verbatim in the form used:

```sql
BEGIN TRANSACTION READ ONLY ISOLATION LEVEL REPEATABLE READ;
SELECT 'A00_SNAPSHOT_ID', clock_timestamp()::text,
       pg_snapshot_xmin(pg_current_snapshot())::text, pg_current_wal_lsn()::text;

-- gross
SELECT vendor, count(*) FROM accounts GROUP BY vendor;
SELECT vendor, status, count(*) FROM accounts GROUP BY vendor, status;

-- uniqueness (paths never leave SQL)
SELECT vendor, count(*), count(DISTINCT lower(rtrim(btrim(home_dir),'/')))
FROM accounts GROUP BY vendor;
SELECT count(DISTINCT regexp_replace(rtrim(btrim(home_dir),'/'), '/[^/]+$','')) FROM accounts;

-- healthy
SELECT vendor, count(*) FROM accounts
WHERE status='available' AND (cooldown_until IS NULL OR cooldown_until <= now())
  AND (last_error IS NULL OR btrim(last_error)='') GROUP BY vendor;

-- protected split from authoritative bindings
SELECT ac.vendor, count(DISTINCT s.account_id), count(DISTINCT s.agent_id)
FROM assignments s JOIN agent ag ON ag.id=s.agent_id
JOIN workspace w ON w.id=ag.workspace_id JOIN accounts ac ON ac.account_id=s.account_id
WHERE w.slug='orq2-dev' GROUP BY ac.vendor;

-- eligible = healthy AND approved AND unassigned
SELECT count(DISTINCT a.account_id) FROM accounts a
JOIN approved_accounts ap ON ap.account_id=a.account_id AND ap.allowed IS TRUE
WHERE a.status='available' AND (a.cooldown_until IS NULL OR a.cooldown_until <= now())
  AND (a.last_error IS NULL OR btrim(a.last_error)='')
  AND NOT EXISTS (SELECT 1 FROM assignments s WHERE s.account_id=a.account_id);

-- eight-ceiling probes
SELECT count(*) FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid
WHERE t.relname IN ('accounts','agent','agent_runtime','assignments','workspace')
  AND pg_get_constraintdef(c.oid) ~ '(<=|<)\s*8\b';
SELECT count(*), count(*) FILTER (WHERE status='online') FROM agent_runtime;

-- generation readiness
SELECT count(*) FROM information_schema.columns
WHERE table_schema='public' AND column_name IN ('generation','catalog_generation','discovery_generation');
SELECT count(*) FROM schema_migrations WHERE version ~ '^13[0-4]';
COMMIT;
```
