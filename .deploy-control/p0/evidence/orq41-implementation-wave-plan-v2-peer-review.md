# ORQ-41 — V2 implementation-wave plan independent re-review

- Auditor: Codex56#B (`w7:p4`)
- Card: ORQ-41, UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`
- Mode: READ-ONLY. No migration, sqlc, build, test, live DB, board or source mutation.
- Reviewed plan SHA-256: `629163c23f9c63b804a71e59ffb00970273564a93c5dc353ea1d20fba0b8e59c`

## Verdict

**BLOCK for implementation.** V2 improves ownership and rollback wording, but its numeric
reservation is not canonical and directly conflicts with the materialized ORQ-13 worktree and
with the latest Z01/reservation evidence. No writer should start LANE-DB until one durable
reservation is signed by the GTL and the materialized ORQ-13 SQL is either retained at 127 or
explicitly renumbered/ported.

## Findings

| ID | Result | Evidence and required disposition |
|---|---|---|
| R1 | **BLOCK** | V2 plan lines 251–267 assigns 127=`task_execution_request`, 128=`task_trigger_audit`, 129=`issue_workflow_authorization`, 130=index and 131=ledger. `orq41-migration-reservation-audit.md:71,84-98,109-112,167-179` proves the only materialized new migration is untracked `127_task_usage_thinking_level` in ORQ-13; its proposal preserves that at 127 and shifts ledger to 128. Z01’s canonical control says ledger=127/account+usage=128 (`gtl-z01-master-integration-control.md:19-21,30`) and later addendum still calls the reservation pending durable confirmation. V2 therefore is not “aligned to Z01” and must not claim the range reserved. |
| R2 | **BLOCK (collision characterization)** | `task_execution_request` is a different table/name from `127_task_usage_thinking_level`, so there is no direct SQL object-name collision merely from both basenames beginning 127. There **is** a migration-reservation/order collision: the runner identity is the full basename and sorts lexically (`orq41-migration-reservation-audit.md:39-55`), so both would apply at 127 in an order nobody approved. ORQ-13 also already modifies shared `queries/task_usage.sql` and `pkg/db/generated/{models.go,task_usage.sql.go}` (`:79-85,146-163`); V2’s single LANE-DB is necessary but cannot cure the unresolved semantic reservation. |
| R3 | **CONDITIONAL PASS** | The order 127 request / 128 audit / 129 auth / 130 active-state index / 131 ledger is internally executable only after R1 is resolved. It is not acceptable as the current canonical order because it disagrees with both Z01 and the materialized ORQ-13 proposal. The plan must state the signed winner, exact basenames, and whether ORQ-13 is renumbered/ported before S1. |
| R4 | **PASS WITH GATE** | D1 (lines 228–249) correctly makes migrations, queries and generated a serial LANE-DB. D2 (440–442) correctly maps placeholders mechanically, but its mapping is invalid until the reservation ruling (R1). S1/S3 timing (242–246) is coherent, except 131 is described as S1 **or** S2 (263), which is not a deterministic dependency gate; choose one. |
| R5 | **CONDITIONAL PASS** | D3 (lines 269–293) correctly removes `service/task.go` overlap and identifies `router.go` as shared. Current `cmd/server/router.go:723-760` has the workspace issue route, `POST /{id}/rerun` and `GET /task-runs`, but no ledger-summary route. Assigning the file to W3 and requiring W4 to submit a registration patch to W3 is safe only if W3 is the single writer, W4 cannot edit the file, and sequencing/acceptance includes route auth, handler ownership and one integrated diff. The “or W4 after W3” alternative leaves an unresolved owner; replace with one mandatory choice. |
| R6 | **PASS WITH GATE** | D4 (440–444; detail 295–314) distinguishes reassign call sites K1/K2 from terminal cancel/delete K3–K6. This is a useful ownership correction; acceptance must retain the six-site regression tests, not only the two removals. |
| R7 | **CONDITIONAL PASS** | D5 (316–336, 440–444) correctly records that `MULTICA_EXECUTION_TRIGGER_DECOUPLED` is absent and assigns creation to W2. “off” is not literally byte-identical because the plan explicitly permits `task_trigger_audit` writes (333–336); the operational exception is acceptable only if the gate names the audit-only delta and tests invalid/missing values as off. |
| R8 | **BLOCK** | D6 (338–391, 445) calls the matrix “32”, but row 31 bundles three named tests (`MissingCostAck`, `NonUUIDv4Key`, `WrongRBAC`) into one row. The plan must publish the exact executable count and names, not a row count. Its anti-skip rule is directionally correct, but the existing handler TestMain/DB skip behavior means the gate needs a separate fail-fast test target; otherwise exit 0 can still be a false green. |
| R9 | **PASS WITH GATE** | D7 (393–411, 446) correctly forbids recreating `idx_one_pending_task_per_issue` and requires the vigente `(issue_id,agent_id)` queued/dispatched index. The down migration still needs an ephemeral-DB execution gate proving the exact definition and squad parallelism. |
| R10 | **BLOCK** | D8 (413–432, 447) gives `go run ./cmd/migrate up/down`, but does not identify an explicit disposable DSN/provision/teardown boundary. Without an unambiguous test database, the command can target a live database despite the prose “descartável”; no migration gate is reproducible or safe until DSN injection and non-live proof are specified. |
| R11 | **CONDITIONAL PASS** | D9 (338–350, 448) correctly prohibits `t.Skip` for the ORQ-41 matrix and requires TestMain fail-fast. It must also prove `m.Run` executes and report exact test names/counts (the GTL-51 harness finding); otherwise a TestMain that exits before `m.Run` remains a false-green path. |

## Required order before any writer

1. GTL resolves one durable 127–131 table, explicitly superseding the conflicting Z01 and
   reservation-audit proposals; preserve/port ORQ-13’s materialized SQL or assign it a new number.
2. LANE-DB owner records exact migration basenames and a disposable-DB DSN gate; only then S1
   runs. Generated output remains exclusively serial.
3. Freeze `router.go` to W3 as sole writer; W4 supplies handler/interface changes and a route
   registration patch, consumed by W3 in one ordered integration step.
4. Publish an executable 32/34-name test manifest and fail-fast TestMain proof; require the
   migration up→down→up cycle on the disposable database.

No implementation, migration, sqlc, build, live DB, board or runtime action was performed.
