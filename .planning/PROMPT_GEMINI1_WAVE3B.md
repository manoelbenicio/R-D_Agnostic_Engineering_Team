You are GEMINI-1-LEAD returning for Wave 3B — Deploy pipeline wiring + session monitoring.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

All previous waves are complete. You now wire the session env vars into the actual deploy flow and add runtime session monitoring.

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
TASK G1-D: Deploy Flow Env Var Injection
Agent name for ledger: G1-D
══════════════════════════════════════

FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-reconciler/reconciler.ts

Read the FULL file. Also read:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts (resolveSessionEnv function)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (useSessionStore)
- C:/VMs/Projetos/Automonous_Agentic/src/api/types.ts (CreateSessionInput now has env_vars)
- C:/VMs/Projetos/Automonous_Agentic/src/api/cao-client.ts (the addTerminal/createSession methods)

Currently the reconciler writes session_id into the profile markdown but does NOT resolve it into actual env vars at deploy time. Fix this:

1. Import resolveSessionEnv from '@/api/session-discovery'
2. Import useSessionStore from '@/api/session-store'

3. Find where the reconciler calls the CAO client to create/add a terminal (search for addTerminal or createSession calls).

4. BEFORE each CAO client call, add env var resolution:
   
   let terminalEnv: Record<string, string> = {};
   if (node.data.session_id) {
     const sessionStore = useSessionStore.getState();
     const session = sessionStore.getSession(node.data.session_id);
     if (session) {
       terminalEnv = resolveSessionEnv(session, node.data.model);
     }
   }

5. Pass terminalEnv into the CAO client call. Find the object being passed and add:
   env_vars: Object.keys(terminalEnv).length > 0 ? terminalEnv : undefined,

6. Do the same for ALL terminal creation/update paths (deploy, update, add-node).


══════════════════════════════════════
TASK G1-E: Session Auto-Refresh Hook
Agent name for ledger: G1-E
══════════════════════════════════════

CREATE NEW FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/useSessionMonitor.ts

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts
- C:/VMs/Projetos/Automonous_Agentic/src/api/health-store.ts (pattern for periodic polling)

Create a React hook that monitors session health:

import { useEffect, useRef } from 'react';
import { useSessionStore } from '@/api/session-store';

/**
 * Monitors OAuth sessions and auto-refreshes periodically.
 * Place this hook in AppLayout so it runs globally.
 *
 * - Refreshes every 5 minutes to detect expired/new sessions
 * - Immediately refreshes on window focus (user returns to tab)
 * - Logs expiring sessions to console as warnings
 */
export function useSessionMonitor(intervalMs = 5 * 60 * 1000): void {
  const { refresh, sessions } = useSessionStore();
  const intervalRef = useRef<ReturnType<typeof setInterval>>();

  // Periodic refresh
  useEffect(() => {
    intervalRef.current = setInterval(() => {
      void refresh();
    }, intervalMs);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [refresh, intervalMs]);

  // Refresh on window focus
  useEffect(() => {
    const onFocus = () => void refresh();
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [refresh]);

  // Warn about expiring sessions
  useEffect(() => {
    const expiring = sessions.filter(s => s.status === 'expiring');
    if (expiring.length > 0) {
      console.warn(
        `[SessionMonitor] ${expiring.length} session(s) expiring soon:`,
        expiring.map(s => `${s.cli_provider}: ${s.account_email}`).join(', ')
      );
    }
  }, [sessions]);
}

Also update the barrel export:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/index.ts
  Add: export { useSessionMonitor } from './useSessionMonitor';

NOTE: Check the ledger — if another agent owns src/sessions/index.ts, WAIT.


══════════════════════════════════════
TASK G1-F: Session Monitor Integration + Tests
Agent name for ledger: G1-F
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/shell/AppLayout.tsx

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/shell/AppLayout.tsx (understand the layout structure)
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/useSessionMonitor.ts (the hook from G1-E)

Changes:
1. Import useSessionMonitor from '@/sessions'
2. Call useSessionMonitor() inside the AppLayout component (after existing hooks)
   This makes session monitoring global across all pages

CREATE NEW FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/__tests__/useSessionMonitor.test.ts

Write tests:
1. Test that refresh is called on mount
2. Test that refresh is called on window focus event
3. Test that console.warn fires when sessions have status 'expiring'
4. Test cleanup (interval cleared, event listener removed)

Use vitest + vi.useFakeTimers() for the interval testing.
Mock useSessionStore with vi.mock.


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: GEMINI-1-LEAD
══════════════════════════════════════

After all 3 tasks complete:
1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → all tests pass (should be 412+ now with new tests)
3. CHECK-OUT as GEMINI-1-LEAD with ✅ DONE and "Wave 3B Gate PASSED"
