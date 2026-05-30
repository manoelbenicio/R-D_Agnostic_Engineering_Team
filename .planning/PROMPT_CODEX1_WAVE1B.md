You are CODEX-1-LEAD, returning for Wave 1B — additional backend work while Wave 2 and Wave 3 run in parallel on other files.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

You completed Wave 1 successfully. Now you have 3 new tasks on files that NO other agent is touching.

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
TASK C1-D: Session Store Unit Tests
Agent name for ledger: C1-D
══════════════════════════════════════

CREATE NEW FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/api/__tests__/session-store.test.ts

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (module to test)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts (dependency — discoverSessions, triggerLogin)
- C:/VMs/Projetos/Automonous_Agentic/src/api/key-store/__tests__/mask.test.ts (test pattern)
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-document/__tests__/store.test.ts (zustand test pattern)

Write tests for useSessionStore:

import { describe, it, expect, vi, beforeEach } from 'vitest';

Mock the session-discovery module:
vi.mock('../session-discovery', () => ({
  discoverSessions: vi.fn(),
  triggerLogin: vi.fn(),
}));

Test cases:
1. Initial state — sessions empty, loading false, error null, lastRefreshed null
2. refresh() — calls discoverSessions, sets sessions, sets lastRefreshed, loading goes true then false
3. refresh() error — sets error message, loading goes false
4. getSession(id) — returns matching session or undefined
5. getSessionsForProvider('claude_code') — filters correctly
6. getSessionsForProvider('unknown') — returns empty array
7. clearError() — clears error state
8. addSession() — calls triggerLogin then calls refresh

Use act() from the zustand store testing patterns. Reset store state between tests using beforeEach.

After creating, run:
  npx vitest run src/api/__tests__/session-store.test.ts


══════════════════════════════════════
TASK C1-E: Schema Validation Tests
Agent name for ledger: C1-E
══════════════════════════════════════

FILE TO EDIT (add test cases):
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-document/__tests__/store.test.ts

Read the full test file first. Find the existing tests that validate canvas document schema parsing.

Add NEW test cases (do NOT remove any existing tests) that verify:

1. A canvas document with session_id on a node parses successfully:
   - Create a valid CanvasDocument object with a node that has session_id: 'test-session-123'
   - Parse it through the schema
   - Assert session_id is preserved

2. A canvas document with session_id undefined parses successfully:
   - Create a valid node WITHOUT session_id
   - Parse through schema
   - Assert it parses without error

3. A canvas document with session_id as empty string parses:
   - session_id: ''
   - Assert it parses

If the existing test file structure doesn't have schema validation tests, create a new describe('session_id schema support') block with these tests using the project's Zod schema imports.

After editing, run:
  npx vitest run src/canvas-document/__tests__/store.test.ts


══════════════════════════════════════
TASK C1-F: Session Status Badge Component
Agent name for ledger: C1-F
══════════════════════════════════════

CREATE NEW FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/SessionStatusBadge.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/session-status-badge.css

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/shell/NavBar.tsx (to see the health pill pattern — the green/yellow/red badge in the navbar)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (useSessionStore)
- C:/VMs/Projetos/Automonous_Agentic/src/index.css (CSS variables)

Create a compact SessionStatusBadge component that can be placed in the navbar:

SessionStatusBadge.tsx:
- Import useSessionStore
- On mount, call refresh() if sessions empty
- Show a small pill/badge with:
  - Count of active sessions: "4 sessions"
  - Color: green if all active, yellow if any expiring, red if any expired
  - Tooltip showing breakdown: "3 active, 1 expiring"
  - Clicking navigates to /sessions page (use useNavigate from react-router-dom)
- Keep it compact — it sits in the navbar next to the health pill
- Match the exact visual style of the existing health pill in NavBar.tsx

session-status-badge.css:
- Match the .health-pill styling patterns from the existing CSS
- Use CSS variables from index.css
- Subtle glow effect based on status color
- Smooth transitions

Also update the barrel export in src/sessions/index.ts:
  export { SessionsPage } from './SessionsPage';
  export { SessionStatusBadge } from './SessionStatusBadge';

IMPORTANT: Do NOT edit NavBar.tsx (C2-C owns it in Wave 2). Just create the component — it will be integrated into the navbar after Wave 2 completes.


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: CODEX-1-LEAD
══════════════════════════════════════

After all 3 tasks complete:
1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → all tests pass (including your new ones)
3. CHECK-OUT as CODEX-1-LEAD with ✅ DONE and "Wave 1B Gate PASSED"
