You are CODEX-2-LEAD returning for Wave 2C — Settings persistence + comprehensive E2E coverage.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

Other agents are working in parallel. DO NOT touch these files:
- src/dashboard/DashboardPage.tsx (CODEX-1)
- src/sessions/SessionStatusBadge.tsx (CODEX-1)
- src/sessions/provider-icons.tsx (CODEX-1)
- src/api/session-discovery.ts (GEMINI-1)
- src/api/session-store.ts (GEMINI-1)
- src/sessions/SessionsPage.tsx (GEMINI-1)

Always CHECK the ledger before editing ANY file.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file:
1. Read: .planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add CHECK-IN row: | <UTC timestamp> | <agent-name> | CHECK-IN | <file> | 🔵 IN PROGRESS | <note> |
4. Update File Lock Table: add entry with agent as owner, 🔴 Locked

After completing:
5. Add CHECK-OUT row with ✅ DONE
6. Update File Lock Table: clear owner, set 🟢 Available
═══════════════════════════════════════════════════════════════


══════════════════════════════════════
TASK C2-G: Session Settings Persistence
Agent name for ledger: C2-G
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/settings/settings-store.ts
- C:/VMs/Projetos/Automonous_Agentic/src/settings/routes.tsx

Read FULLY first. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/settings/__tests__/settings-store.test.ts (test patterns)

Add session-related preferences to the existing settings store:

1. In settings-store.ts, add to the settings state:
   sessionAutoRefreshInterval: number;  // minutes, default 5
   sessionShowExpiredWarnings: boolean;  // default true
   sessionMaskEmails: boolean;          // default false (privacy mode)

2. Add defaults for these new fields in the initial state.

3. In routes.tsx, find or create a settings section. Add a "Sessions" settings panel:
   - Auto-refresh interval: dropdown [1 min, 5 min, 15 min, 30 min, Off]
   - Show expiring warnings: toggle switch
   - Mask emails in UI: toggle switch (privacy mode)
   - Style these controls matching existing settings page patterns


══════════════════════════════════════
TASK C2-H: Comprehensive E2E Test Suite
Agent name for ledger: C2-H
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/tests/e2e/sessions.spec.ts (EXISTING — add more tests)
- C:/VMs/Projetos/Automonous_Agentic/tests/e2e/canvas-session.spec.ts (NEW)

Read first:
- C:/VMs/Projetos/Automonous_Agentic/tests/e2e/sessions.spec.ts (existing 5 tests from Wave 1C)
- C:/VMs/Projetos/Automonous_Agentic/tests/e2e/smoke.spec.ts (e2e patterns)

1. Add to sessions.spec.ts:
   - Test: Add Session button opens dialog
   - Test: Provider sections are collapsible
   - Test: Refresh button shows loading state
   - Test: Page title is "Sessions · AgentVerse"

2. Create canvas-session.spec.ts:
   - Test: Canvas fullscreen toggle works (Ctrl+Shift+F or button)
   - Test: Zoom controls display current zoom percentage
   - Test: Fit View button resets zoom
   - Test: Config panel has Auth Session dropdown
   - Test: Session dropdown shows "Auto (default session)" as default option


══════════════════════════════════════
TASK C2-I: Canvas Keyboard Shortcut Help Overlay
Agent name for ledger: C2-I
══════════════════════════════════════

CREATE NEW FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/KeyboardShortcutsHelp.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/keyboard-shortcuts-help.css

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/CanvasBuilderPage.tsx (to see existing shortcuts)
- C:/VMs/Projetos/Automonous_Agentic/src/index.css (design tokens)

Create a keyboard shortcuts help overlay triggered by pressing "?" on the canvas:

KeyboardShortcutsHelp.tsx:
- Props: { isOpen: boolean; onClose: () => void }
- Modal overlay showing all canvas keyboard shortcuts in a clean grid:
  | Shortcut | Action |
  | Ctrl+Shift+F | Toggle fullscreen |
  | Ctrl+0 | Fit view |
  | Ctrl+= | Zoom in |
  | Ctrl+- | Zoom out |
  | ? | Show this help |
  | Delete | Remove selected node |
  | Escape | Close panel / exit fullscreen |
- Close on Escape or clicking outside
- Glassmorphism card style matching app theme

keyboard-shortcuts-help.css:
- Modal with backdrop blur
- Two-column grid layout
- Keyboard key styling (rounded kbd tags)
- Match dark theme


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: CODEX-2-LEAD
══════════════════════════════════════

After all 3 tasks complete:
1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → all tests pass
3. Run: npx playwright test tests/e2e/sessions.spec.ts tests/e2e/canvas-session.spec.ts → all pass
4. Git commit only YOUR files:
   git add src/settings/settings-store.ts src/settings/routes.tsx tests/e2e/sessions.spec.ts tests/e2e/canvas-session.spec.ts src/canvas-builder/KeyboardShortcutsHelp.tsx src/canvas-builder/keyboard-shortcuts-help.css .planning/AGENT_LEDGER.md
   git commit -m "feat(sessions): Wave 2C — settings persistence, E2E coverage, keyboard shortcuts help"
5. CHECK-OUT as CODEX-2-LEAD with ✅ DONE and "Wave 2C Gate PASSED"
