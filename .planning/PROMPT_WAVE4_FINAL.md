You are CODEX-1-LEAD executing Wave 4 — FINAL integration, commit, and ship verification.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

PREREQUISITE: ALL previous waves must be complete.
1. Read .planning/AGENT_LEDGER.md
2. Confirm ALL leads have CHECK-OUT with ✅ DONE:
   CODEX-1-LEAD, CODEX-2-LEAD, GEMINI-1-LEAD
3. If any agent is still 🔵 IN PROGRESS, STOP and WAIT.

This is the FINAL wave. No more waves after this.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Same as all previous waves. Agent name: WAVE-4-FINAL
═══════════════════════════════════════════════════════════════


══════════════════════════════════════
TASK FINAL-A: Stage and Commit All Remaining Work
Agent name for ledger: FINAL-A
══════════════════════════════════════

1. Run: git status --short
   Review all modified/untracked files

2. Run: npx tsc --noEmit
   Must be 0 errors. If errors, fix them.

3. Run: npx vitest run
   Must be 430+ tests passed, 0 failed. If failures, fix them.

4. Stage everything:
   git add -A

5. Create the final commit:
   git commit -m "feat(sessions): Final integration — all waves merged

   Session Management feature complete:
   - OAuth session discovery, store, and monitoring
   - Premium Sessions page at /sessions
   - Auth Session dropdown in canvas config panel
   - Session env var injection in deploy pipeline
   - Session status badge in navbar
   - Dashboard session summary widget
   - FinOps session cost grouping
   - Terminal tab + agent node OAuth indicators
   - Canvas 32-inch expansion + fullscreen toggle
   - Add Session dialog modal
   - Session revoke support
   - Security utilities (maskEmail, sanitize)
   - Auto-refresh session monitoring (5min + focus)
   - Keyboard accessibility (WCAG AA)
   - Provider icons and branding
   - Full documentation at docs/session-management.md
   - 430+ unit tests, 5 E2E tests, 0 failures"


══════════════════════════════════════
TASK FINAL-B: Full Verification Suite
Agent name for ledger: FINAL-B
══════════════════════════════════════

Run ALL verification gates one final time:

1. TypeScript:
   npx tsc --noEmit
   Expected: 0 errors

2. Unit Tests:
   npx vitest run
   Expected: 430+ passed, 0 failed

3. E2E Tests:
   npx playwright test tests/e2e/sessions.spec.ts
   Expected: 5/5 passed

4. Lint (if available):
   npx eslint src/sessions/ src/api/session-discovery.ts src/api/session-store.ts --max-warnings=0
   Expected: 0 errors, 0 warnings (or note any pre-existing issues)

5. Build check:
   npx vite build
   Expected: builds successfully

Document ALL results in the ledger.


══════════════════════════════════════
TASK FINAL-C: Ship Report
Agent name for ledger: FINAL-C
══════════════════════════════════════

CREATE FILE:
- C:/VMs/Projetos/Automonous_Agentic/.planning/SHIP_REPORT.md

Write a comprehensive ship report covering:

# Ship Report — Session Management Feature

## Summary
- Feature: OAuth-first Session Management
- Total waves: X
- Total agents deployed: X
- Total sub-tasks: X
- Total commits: X
- Total tests added: X
- Time span: <first wave timestamp> → <final wave timestamp>

## Files Created (NEW)
List all new files created across all waves.

## Files Modified
List all existing files that were modified.

## Test Coverage
- Unit tests: X passed
- E2E tests: X passed
- Total: X passed, 0 failed

## Quality Gates
List all gate results from the ledger (wave, result, test count).

## Architecture Decisions
- OAuth-first: CLAUDE_CONFIG_DIR / KIRO_HOME per-process isolation
- Zustand for session state management
- Auto-refresh every 5 minutes + window focus
- env_vars injection at CAO terminal spawn

## Known Limitations
- Revoke depends on CAO /auth/sessions/:id DELETE endpoint (backend stub)
- Session discovery falls back to provider listing if /auth/sessions not available
- No persistent session preferences across browser reloads (stored in Zustand memory only)

## Ledger Summary
Copy the final ledger statistics: total CHECK-INs, CHECK-OUTs, BLOCKEDs, FAILEDs.


══════════════════════════════════════
FINAL QUALITY GATE
Agent name for ledger: WAVE-4-FINAL
══════════════════════════════════════

After ALL tasks complete:
1. All gates green
2. Final commit created
3. Ship report written
4. CHECK-OUT as WAVE-4-FINAL with ✅ DONE and "SESSION MANAGEMENT FEATURE SHIPPED"
