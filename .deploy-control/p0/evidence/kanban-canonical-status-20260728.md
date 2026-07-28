# P0 Kanban canonical status snapshot — 2026-07-28

- **Workspace:** `20fce817-895d-447b-965a-49f5e279314a`
- **Cutoff UTC:** `2026-07-28T13:19:48Z`
- **Mode:** authenticated, read-only. No issue, comment, assignee, status, date or task was mutated to produce this snapshot.
- **Authority order:** authenticated API/DB state at cutoff; latest terminal task outcome and human-only `/note`; then fingerprinted local operational evidence. A terminal `completed` task does not by itself mean issue acceptance.
- **Stale rule:** `YES` only when current verified work state materially contradicts the board column; it is not inferred merely because a historical task is terminal.

## Totals

| Dimension | Counts |
|---|---|
| Open issues | **29** |
| Board status | `todo=17`, `in_review=7`, `blocked=5` |
| Priority | `urgent=13`, `high=11`, `medium=1`, `low=4` |
| Assignee type | `agent=12`, `member=1`, `squad=2`, `none=14` |
| Active task queue | **0** (`queued/dispatched/running=0`) |
| Board-status stale | **7**: ORQ-13, ORQ-16, ORQ-17, ORQ-18, ORQ-37, ORQ-41, ORQ-42 |

## Canonical issue matrix

`Owner` is the creating member/agent recorded by the API. `ETA` is factual only: a board due date, hard reservation deadline, or explicit condition; `not stated` is not an estimate.

| Issue | Board / priority | Owner | Assignee | Evidence-backed actual status / current activity | Verifiable blocker | ETA | Stale? |
|---|---|---|---|---|---|---|---|
| ORQ-11 | in_review / low | owner | E2E-Codex-Test (archived agent) | Latest task completed; root count rerun returned `1`; waiting acceptance. | No technical blocker evidenced; review/acceptance only. | not stated | NO |
| ORQ-12 | todo / urgent | owner | Opus48-A | No active task; historical run failed `runtime_offline`; preserved in serial LANE-DB queue. | ORQ-21 predecessor and canonical DB ordering must stabilize before account identity is added to usage. | after ORQ-21; no date | NO |
| ORQ-13 | todo / high | owner | Codex-A | **CODE PASS / EVIDENCE PASS / integration-ready conditional** on clean branch `c0e93a2`; migration 127 exclusive; no active task. | Independent review and owner merge authorization; at snapshot, `RES-ORQ13-001` expires `2026-07-28T15:37:12Z`; end-to-end also needs complementary gateway-admission lane. | hard reservation deadline `15:37:12Z`; integration date not stated | **YES → in_review/conditional** |
| ORQ-14 | blocked / urgent | owner | Codex-C | Diagnostic task completed and intentionally left card blocked. Codex reports real usage; AGY 1.1.7 and Kiro 2.13 ACP expose no truthful per-turn totals. | External provider interfaces do not expose required token totals; estimates would violate acceptance. | not stated | NO |
| ORQ-15 | todo / urgent | owner | Gemini-3.6-Flash-A | No active task; historical pre-start failure; multi-account DAG remains queued. | Requires ORQ-21 registry/assignments and downstream usage/cost acceptance; historical task is not replay authorization. | after dependency chain; no date | NO |
| ORQ-16 | in_review / high | owner | Gemini-3.6-Flash-B | **Reconciliation PASS / closeout evidenced**: allowlist 141,145,146; A7/A8 retries completed; six runtimes online; queue zero. | No technical blocker verified; only owner acceptance. Slot-150 remains excluded by policy, not a blocker. | ready for acceptance; no date | **YES → done candidate** |
| ORQ-17 | todo / high | owner | Opus48-B | Code-only candidate `9cf3296...` passed focused review; release remains not ready; no active task. | Owner credential provisioning through approved secret-safe path and real authenticated login/cutover verification. | after owner credential action; no date | **YES → blocked** |
| ORQ-18 | todo / high | owner | Codex-B | Isolated UI patch exists, but independent test audit is BLOCK; no active task. | Must follow ORQ-26 clean-base validation; targeted tests currently fail and browser QA/permissions remain required. | after ORQ-26 and test repair; no date | **YES → blocked** |
| ORQ-19 | blocked / medium | owner | Codex-C | Read-only diagnosis completed; obsolete runtimes remain linked to archived agents. | Deletion requires destructive `--cascade`; explicit owner and TL consent absent. | not stated | NO |
| ORQ-20 | blocked / high | owner | Opus48-A | Catalog probing completed; all active agents have models; Codex-D fixed; thinking-level mapping proposed. | Owner must choose cost-affecting thinking levels; ORQ-13 pricing/persistence dependency remains. | after owner decision/ORQ-13; no date | NO |
| ORQ-21 | todo / urgent | owner | Codex-B | No active task; historical run failed `runtime_offline`; canonical LANE-DB predecessor position 1, not stabilized. | Approved accounts/assignments authority and implementation must stabilize before ORQ-12/13. | first in DB chain; no date | NO |
| ORQ-23 | todo / high | owner | Codex-A | No active task; historical run failed `runtime_offline`; preserved, no replay. | Depends on completed Phase-3 usage/cost chain and rollback gate evidence. | after ORQ-21→12→13/usage gates; no date | NO |
| ORQ-26 | in_review / urgent | owner | **UNASSIGNED** | Branch `ci/orq26-db-gate` at immutable `20cab478...`; run `30294480915` had zero steps and no gates executed. | **HOLD_EXTERNAL_BILLING / JOB_NOT_STARTED**. Owner must fix GitHub billing and rerun the same run; not a code/test failure. | after billing repair; no date | NO |
| ORQ-30 | in_review / urgent | owner | owner | Durable backend environment/JWT remediation evidenced across two recreates; waiting review. | No current technical blocker verified; acceptance review remains. | not stated | NO |
| ORQ-31 | in_review / urgent | owner | **UNASSIGNED** | Canonical card is Security Wave A containment; no task history or active task; identifier collision was resolved as reporting-only. | No completion evidence or explicit current blocker on card; reviewer/owner acceptance evidence is missing. | not stated | NO |
| ORQ-33 | todo / urgent | owner | **UNASSIGNED** | Rev-token Wave-B intake only; no task dispatched; waiting Navy_Seals distribution. | Leader distribution/owner-approved secret rotation controls. | board due `2026-07-28` | NO |
| ORQ-34 | todo / urgent | owner | **UNASSIGNED** | OPENAI_API_KEY Wave-B intake only; no task dispatched; waiting Navy_Seals distribution. | Leader distribution plus secret-safe infrastructure/rotation approval. | board due `2026-07-28` | NO |
| ORQ-35 | todo / urgent | owner | **UNASSIGNED** | PostgreSQL/DATABASE_URL Wave-B intake only; no task dispatched. | Leader distribution, backup/rollback and credential-hardening approval. | board due `2026-07-29` | NO |
| ORQ-36 | todo / urgent | owner | **UNASSIGNED** | MULTICA_TOKEN Wave-B intake only; no task dispatched. | Leader distribution and token lifecycle/rollback authorization. | board due `2026-07-29` | NO |
| ORQ-37 | todo / urgent | owner | Navy_Seals squad | Leader task completed after escalating ORQ-34 scope, infrastructure and credential-startup blockers; no active worker task. | Owner decisions and Wave-B infrastructure/credential prerequisites; claimed delegation did not populate ORQ-33..36 assignees. | board due `2026-07-30` | **YES → blocked** |
| ORQ-38 | in_review / high | owner | **UNASSIGNED** | Latest task completed; UUID+GET-back protocol and ORQ-39 re-registration verified; waiting acceptance. | No technical blocker verified; review/acceptance only. | not stated | NO |
| ORQ-39 | blocked / high | owner | **UNASSIGNED** | Design PASS / execution BLOCK; accidental task was cancelled; no active task. | Depends on ORQ-26 executable CI/browser foundation and explicit unfreeze/authorization. | after ORQ-26; no date | NO |
| ORQ-40 | blocked / low | owner | **UNASSIGNED** | Read-only pin/release verification completed; no package installation or mutation; active queue is now zero. | Written owner authorization for CLI alignment remains absent; execution must use approved maintenance/rollback gate. | not stated | NO |
| ORQ-41 | todo / high | owner | **UNASSIGNED** | **CODE PASS / EVIDENCE PASS**, commit `e0b0155` dormant; flag remains off. | Activation blocked by serial LANE-DB plus W2/W3 and owner merge/activation authorization. | after LANE-DB/W2/W3; no date | **YES → blocked** |
| ORQ-42 | todo / urgent | owner | **UNASSIGNED** | Q-H/Q-I implemented on clean branch `agent/opus48-a/orq42-secret-tools-clean` at `cebc96ae...`; independent peer review underway. | Must receive independent PASS before H1 integration/push; original local branch remains without rewrite. | after peer-review PASS; no date | **YES → in_review** |
| ORQ-43 | todo / high | owner | **UNASSIGNED** | Daemon-token lifecycle design and independent peer review PASS; ORQ43A remains queue-position-only, no implementation/DDL. | `RES-ORQ43A-001=QUEUED_NO_NUMBER` after ORQ-21→12→13; root SQLC delta must clear and global scan repeat. | after DB predecessors; no date | NO |
| ORQ-44 | todo / high | owner | **UNASSIGNED** | OmniRoute inference-key lifecycle design and independent review exist; no active implementation task. | Owner-approved rotation window, rollback and secret-safe execution path not evidenced as authorized. | not stated | NO |
| ORQ-45 | todo / low | owner | **UNASSIGNED** | Zero-task validation card; still has no task and no current execution evidence. | Scope/owner decision absent; no operational blocker otherwise evidenced. | not stated | NO |
| ORQ-46 | in_review / low | Codex-A | Navy_Seals squad | Priority-triage task completed; waiting review. | No technical blocker verified; review/acceptance only. | not stated | NO |

## Agents without an active task

All **12 non-archived agents** returned `idle` and DB/API correlation found `0` active tasks each:

`Agy-P0-A7`, `Agy-P0-A8`, `Codex-A`, `Codex-B`, `Codex-C`, `Codex-D`, `Gemini-3.6-Flash-A`, `Gemini-3.6-Flash-B`, `Opus-46#A`, `Opus-46-B`, `Opus48-A`, `Opus48-B`.

Additionally, ORQ-11 points to archived idle agent `E2E-Codex-Test`; it is not part of the 12-agent active fleet.

## Cards without assignee

**14 cards:** `ORQ-26`, `ORQ-31`, `ORQ-33`, `ORQ-34`, `ORQ-35`, `ORQ-36`, `ORQ-38`, `ORQ-39`, `ORQ-40`, `ORQ-41`, `ORQ-42`, `ORQ-43`, `ORQ-44`, `ORQ-45`.

Squad-assigned cards are not counted as unassigned: ORQ-37 and ORQ-46 are assigned to `Navy_Seals`.

## Evidence and limitations

Authenticated reads used: `GET /api/issues`, `GET /api/agents`, `GET /api/workspaces/{id}/members`, `GET /api/squads`, `GET /api/agent-task-snapshot`, plus each open issue's `comments` and `task-runs`. DB verification joined `agent`, `issue` and `agent_task_queue` read-only and confirmed zero `queued`, `dispatched` or `running` rows for the workspace.

Primary local corroboration includes:

- `.deploy-control/p0/evidence/orq13-squad-integration-readiness.md`
- `.deploy-control/p0/evidence/gtl-migration-registrar-governance.md`
- `.deploy-control/p0/evidence/release-control-work-measurement-baseline-20260727.md`
- `.deploy-control/p0/evidence/orq-9-issues-exit-criteria.md`
- `.deploy-control/p0/evidence/RCA-HANDOVER-20260727.md`
- `.deploy-control/p0/evidence/orq26-push-executed-run-billing-blocked.md`
- `.deploy-control/p0/evidence/agy-slot150-lifecycle-containment-20260727.md`
- `.deploy-control/p0/evidence/kanban-leader-live-execution-20260727.md`
- `.deploy-control/p0/worklogs-draft/REGISTRAR_QUEUE_INDEX.md`

No secret, cookie, CSRF, password or token value was read or printed. The authenticated cookie was consumed only by `curl` from its mode-0600 private file. ETAs not explicitly present in due dates or evidence are intentionally `not stated`; this snapshot does not invent delivery dates.
