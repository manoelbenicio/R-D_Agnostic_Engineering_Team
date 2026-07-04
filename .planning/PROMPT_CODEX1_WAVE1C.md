You are CODEX-1-LEAD returning for Wave 1C — NavBar integration + final wiring.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

SessionStatusBadge is built and exported from src/sessions/index.ts. Now wire it into the app.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file:
1. Read: .planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add CHECK-IN row: | <UTC timestamp> | <agent-name> | CHECK-IN | <file> | 🔵 IN PROGRESS | <note> |
4. Update File Lock Table: add file entry with agent as owner, 🔴 Locked

After completing:
5. Add CHECK-OUT row with ✅ DONE
6. Update File Lock Table: clear owner, set 🟢 Available
═══════════════════════════════════════════════════════════════


══════════════════════════════════════
TASK C1-G: Wire SessionStatusBadge into NavBar
Agent name for ledger: C1-G
══════════════════════════════════════

FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/shell/NavBar.tsx

Read FULLY first. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/SessionStatusBadge.tsx (the component to integrate)
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/session-status-badge.css (its styles)

Changes:
1. Add import at the top:
   // eslint-disable-next-line agentverse/no-sideways-capability-imports
   import { SessionStatusBadge } from '@/sessions';

2. In the navbar-right section, BEFORE the health pill button, add:
   <SessionStatusBadge />

That's it. The component handles its own state, styling, and click navigation.


══════════════════════════════════════
TASK C1-H: E2E Smoke Test for Sessions Page
Agent name for ledger: C1-H
══════════════════════════════════════

FILE:
- C:/VMs/Projetos/Automonous_Agentic/tests/e2e/sessions.spec.ts (NEW)

Read first:
- C:/VMs/Projetos/Automonous_Agentic/tests/e2e/smoke.spec.ts (existing e2e test pattern)
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/SessionsPage.tsx (to know what elements to target)

Create a Playwright E2E test:

import { test, expect } from '@playwright/test';

test.describe('Sessions Page', () => {

  test('navigates to /sessions from navbar', async ({ page }) => {
    await page.goto('/');
    await page.click('#nav-link-sessions');
    await expect(page).toHaveURL(/sessions/);
    await expect(page.locator('h1')).toContainText('AUTH SESSIONS');
  });

  test('shows provider sections', async ({ page }) => {
    await page.goto('/sessions');
    await expect(page.locator('text=CLAUDE CODE')).toBeVisible();
    await expect(page.locator('text=CODEX')).toBeVisible();
    await expect(page.locator('text=GEMINI CLI')).toBeVisible();
    await expect(page.locator('text=KIRO CLI')).toBeVisible();
  });

  test('refresh button exists and is clickable', async ({ page }) => {
    await page.goto('/sessions');
    const refreshBtn = page.locator('button:has-text("Refresh")');
    await expect(refreshBtn).toBeVisible();
    await refreshBtn.click();
  });

  test('shows empty state when no sessions detected', async ({ page }) => {
    await page.goto('/sessions');
    // With no CAO backend, should show empty state or fallback
    await expect(page.locator('.sessions-page')).toBeVisible();
  });

  test('session status badge appears in navbar', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.session-status-badge')).toBeVisible();
  });
});

Match the existing e2e test patterns for imports, configuration, and selectors.


══════════════════════════════════════
TASK C1-I: Git Commit All Pending Work
Agent name for ledger: C1-I
══════════════════════════════════════

After tasks C1-G and C1-H are complete and verified:

1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → all 423+ tests pass
3. Stage all changes: git add -A
4. Commit with message:

git commit -m "feat(sessions): Wave 1B+1C — StatusBadge, NavBar integration, tests

- SessionStatusBadge component with auto-refresh + nav to /sessions
- StatusBadge wired into NavBar (navbar-right section)
- 8 session-store unit tests
- session_id schema coverage tests
- E2E smoke tests for Sessions page
- Total: 423+ tests passing, tsc 0 errors"


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: CODEX-1-LEAD
══════════════════════════════════════

After all tasks complete:
1. npx tsc --noEmit → 0 errors
2. npx vitest run → 423+ tests passed
3. Git commit created
4. CHECK-OUT as CODEX-1-LEAD with ✅ DONE and "Wave 1C Gate PASSED"