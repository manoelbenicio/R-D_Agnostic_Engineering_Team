# Prodex Next-Slice Audit (Tasks 3.1 - 3.3)
**Reviewer Provenance:** ANTIGRAVITY (Date: 2026-07-18T17:37:00Z)

## Grade: 3.1-3.3 PARTIAL

## Source Hashes Assessed
- `afe8b75f024e825eaea301c22566e28e653ccb57727e596905858a385489e4c8  multica-auth-work/server/internal/daemon/prodex_profiles.go`
- `312fef692eeca53b84413f429b2cb19136974f7326a133105379d18263bdf78e  multica-auth-work/server/internal/daemon/prodex_runtime_integration_test.go`

## Implementation Readiness & Test Coverage Matrix

| Task | Implementation | Test Coverage | Overall Grade | Notes & Finding |
| --- | --- | --- | --- | --- |
| **3.1 Mapping from Prodex profile names to isolated Codex slots** | **ACCEPT** | **MISSING** | **PARTIAL** | Implemented securely in `reconcileProdexProfiles` using `rotationStore.ListAccounts(ctx, "codex", ...)` (lines 33-36, 76-80). However, no offline tests exist for this logic. |
| **3.2 Audit/reconciliation using `prodex profile add --codex-home`** | **ACCEPT** | **MISSING** | **PARTIAL** | Implemented cleanly in `addProdexProfileReference` (line 201). Uses `exec.CommandContext` without credential copying. Missing unit tests. |
| **3.3 Enforce POSIX, approved-root containment, credential presence** | **ACCEPT** | **MISSING** | **PARTIAL** | Validations implemented correctly across lines 49, 61, 64, 89-93, 160-187 (`validateCodexSlotHome`, `validateApprovedPOSIXFilesystem`, mode 0700/0600 checks). Missing unit tests. |

## Missing-Test Recommendations
The implementation logic for Profile Reconciliation is mature, but the tests are entirely absent. A new test file (e.g., `multica-auth-work/server/internal/daemon/prodex_profiles_test.go`) should be created to include executable offline tests, mimicking the synthetic design of `prodex_runtime_integration_test.go`.
Specifically, tests must:
1. Use `t.Setenv` to mock `PRODEX_HOME` and `MULTICA_AGENT_CREDENTIAL_SLOTS_ROOT`.
2. Mock `rotationStore` to return synthetic test accounts.
3. Stub `exec.CommandContext` or provide a dummy script to intercept `prodex profile add --codex-home`.
4. Assert idempotency (re-running reconciliation doesn't duplicate).
5. Assert POSIX rejections (e.g. failing if directory mode != 0700 or file != 0600).

## Ownership-Conflict Scan
- **Ledger Checked:** `.planning/AGENT_LEDGER.md`
- **Result:** No conflicts. `Codex#5.5#D` owns deploy/scripts, while no agent currently claims `server/internal/daemon/prodex_profiles.go` or tests. I have locked this audit process cleanly via a Golden Rule `CHECK-IN`.

## AB-REQ/EV Mapping (Requirement vs Evidence)
- **AB-REQ 3.1 (Current validated inventory):** Evidenced by `prodex_profiles.go:33` where it calls `rotationStore.ListAccounts()`.
- **AB-REQ 3.2 (No credential copying via prodex add):** Evidenced by `prodex_profiles.go:201` executing `prodex profile add <name> --codex-home <codexHome>` with strict child-env limitations.
- **AB-REQ 3.3 (POSIX & Approved-Root checks):** Evidenced by `prodex_profiles.go:160` (`validateCodexSlotHome`) which enforces `filepath.Rel(slotsRoot, home)`, mode 0600/0700, and non-empty `auth.json` size checks.

## Explicit Non-Claims
- This was a strictly read-only audit. No test files were created.
- No real credentials, network, or live databases were used.
- I did NOT edit product/tests/spec/task checkboxes.
- I did not "ACCEPT" the final slice on behalf of the developer (never self-accept).
