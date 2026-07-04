You are CODEX-1-LEAD returning for Wave 1D — FinOps session integration + terminal session indicators.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

Other agents are working in parallel. DO NOT touch these files (they are locked by other agents):
- src/canvas-builder/canvas-builder.css (CODEX-2)
- src/canvas-builder/CanvasBuilderPage.tsx (CODEX-2)
- src/sessions/SessionsPage.tsx (GEMINI-1)
- src/api/session-discovery.ts (GEMINI-1)
- src/api/session-store.ts (GEMINI-1)

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
TASK C1-J: Session-Aware Terminal Tab Headers
Agent name for ledger: C1-J
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/terminal-grid/TabBar.tsx

Read FULLY first. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/terminal-grid/TerminalGrid.tsx (parent context)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (useSessionStore — READ ONLY, do not edit)
- C:/VMs/Projetos/Automonous_Agentic/src/shared/canvas-types.ts (CanvasNode with session_id)

Changes:
When a terminal tab represents an agent with a session_id bound, show a small visual indicator:

1. Import useSessionStore from '@/api/session-store' (read-only usage)
2. For each tab, if the corresponding canvas node has a session_id:
   - Look up the session: useSessionStore.getState().getSession(session_id)
   - If found, render a small badge/dot next to the tab label showing:
     - 🟢 if session active
     - 🟡 if session expiring
     - 🔴 if session expired
   - Tooltip: "OAuth: email@example.com"
3. If no session_id, show nothing (default behavior)
4. Keep it subtle — small dot, no layout disruption


══════════════════════════════════════
TASK C1-K: FinOps Session Cost Grouping
Agent name for ledger: C1-K
══════════════════════════════════════

FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/finops/FinopsPage.tsx

Read FULLY first. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/finops/token-cost.ts
- C:/VMs/Projetos/Automonous_Agentic/src/finops/token-usage.ts

Changes:
The FinOps page currently shows costs per provider/model. Add a "Group by Session" toggle:

1. Add state: const [groupBy, setGroupBy] = useState<'provider' | 'session'>('provider')
2. Add a toggle button group in the page header: [By Provider] [By Session]
3. When groupBy === 'session':
   - Import useSessionStore from '@/api/session-store' (read-only)
   - Group cost data by session_id instead of provider
   - Show session email as the group header (e.g., "beniciosmsnoel@ — Claude Code")
   - Show "Unassigned" for tokens without a session_id
4. When groupBy === 'provider': keep existing behavior unchanged
5. Style the toggle to match the existing page design


══════════════════════════════════════
TASK C1-L: Agent Node Session Indicator
Agent name for ledger: C1-L
══════════════════════════════════════

FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/AgentNode.tsx

Read FULLY first. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/shared/canvas-types.ts (CanvasNode.data.session_id)

Changes:
When an agent node on the canvas has a session_id assigned, show a small OAuth badge:

1. Check if data.session_id exists on the node
2. If yes, render a small lock/shield icon or "OAuth" text badge in the bottom-right corner of the node
3. Use a subtle style — small, semi-transparent, doesn't clutter the node
4. Tooltip: "OAuth session bound" or the session email if available
5. If no session_id, render nothing (default)

CSS: Add styles inline or in the existing canvas-builder.css section for agent nodes.
NOTE: Do NOT edit canvas-builder.css if CODEX-2 has it locked. Check the ledger first. If locked, use inline styles or a separate small CSS file.


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: CODEX-1-LEAD
══════════════════════════════════════

After all 3 tasks complete:
1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → all tests pass
3. Git commit: git add -A; git commit -m "feat(sessions): Wave 1D — terminal session indicators, FinOps grouping, node badges"
4. CHECK-OUT as CODEX-1-LEAD with ✅ DONE and "Wave 1D Gate PASSED"
