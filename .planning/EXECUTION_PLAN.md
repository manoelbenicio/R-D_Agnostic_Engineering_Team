# 🏗️ Multi-Agent Orchestration Plan v3 — Session Management

> **Owner**: Manoel (activates each agent manually)
> **Architect**: Gemini 2.5 Pro (plan design + coordination rules)
> **Ledger**: [AGENT_LEDGER.md](./AGENT_LEDGER.md) — mandatory check-in/check-out for every agent.
> **Location**: This file lives in the repo so ALL agents and the owner can read it in real-time.

---

## Agent Fleet

| Lead Agent | Model | Sub-Agents | Workstream |
|------------|-------|------------|------------|
| **CODEX-1** | Codex 5.5 High Thinking | C1-A, C1-B, C1-C | Backend / Data Layer |
| **CODEX-2** | Codex 5.5 High Thinking | C2-A, C2-B, C2-C | Frontend / UI Layer |
| **GEMINI-1** | Gemini 3.5 Flash High Thinking | G1-A, G1-B, G1-C | Integration / QA / Reconciler |

---

## ⚠️ MANDATORY PROTOCOL — All Agents

Every agent MUST follow this. **No exceptions. No shortcuts.**

```
═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
═══════════════════════════════════════════════════════════════

Before touching ANY source file:

1. Read:  .planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add a CHECK-IN row with your agent name, timestamp, file, 🔵 IN PROGRESS
4. Update the File Lock Table: set yourself as owner, 🔴 Locked

After completing ALL edits:

5. Add a CHECK-OUT row with ✅ DONE
6. Update File Lock Table: clear owner, set 🟢 Available

If your work FAILS:

7. Add a FAILED row with 🔴 FAILED and the error in Notes
═══════════════════════════════════════════════════════════════
```

---

## Execution Waves

```
Wave 1: Foundation (no dependencies)
  CODEX-1 → C1-A, C1-B, C1-C run in parallel
  Gate: npx tsc --noEmit = 0 errors

Wave 2: UI + Integration (depends on Wave 1)
  CODEX-2 → C2-A, C2-B, C2-C run in parallel
  Gate: npx tsc --noEmit = 0 errors

Wave 3: Deploy Pipeline + Tests (depends on Wave 1 + 2)
  GEMINI-1 → G1-A, G1-B, G1-C run in parallel
  Gate: npx tsc --noEmit = 0 errors + npx vitest run = 405+ passed
```

---

## File Ownership Matrix

| File | Wave | Owner | Lock Rule |
|------|------|-------|-----------|
| `src/shared/canvas-types.ts` | 1 | C1-A | Single writer |
| `src/canvas-document/schema.ts` | 1 | C1-A | Single writer |
| `src/api/session-discovery.ts` (NEW) | 1 | C1-B | Creator only |
| `src/api/session-store.ts` (NEW) | 1 | C1-C | Creator only |
| `src/sessions/SessionsPage.tsx` (NEW) | 2 | C2-A | Creator only |
| `src/sessions/sessions.css` (NEW) | 2 | C2-A | Creator only |
| `src/sessions/index.ts` (NEW) | 2 | C2-A | Creator only |
| `src/canvas-builder/BlockConfigurationPanel.tsx` | 2 | C2-B | Single writer |
| `src/shell/router.tsx` | 2 | C2-C | Single writer |
| `src/shell/NavBar.tsx` | 2 | C2-C | Single writer |
| `src/canvas-reconciler/reconciler.ts` | 3 | G1-A | Single writer |
| `src/api/types.ts` | 3 | G1-B | Single writer |
| `src/api/cao-client.ts` | 3 | G1-B | Single writer |
| `src/api/__tests__/session-discovery.test.ts` (NEW) | 3 | G1-C | Creator only |

---

# WAVE 1 — CODEX-1 Lead + 3 Sub-Agents

## CODEX-1 Lead Prompt

```
You are CODEX-1-LEAD, the lead agent for the Backend/Data Layer workstream.

You coordinate 3 sub-agents: C1-A, C1-B, C1-C.
They work in PARALLEL on non-conflicting files.

Your responsibilities:
1. Verify all 3 sub-agents have completed by reading .planning/AGENT_LEDGER.md
2. Run: npx tsc --noEmit
3. If errors, identify which sub-agent's file caused it and report
4. CHECK-IN/CHECK-OUT in .planning/AGENT_LEDGER.md with name CODEX-1-LEAD

MANDATORY: Read .planning/AGENT_LEDGER.md before any action.
Add CHECK-IN before gate. Add CHECK-OUT with result after.

Wave 1 Gate: npx tsc --noEmit = 0 errors
Report: list all files changed, all ledger entries, gate result.
```

---

### C1-A: Type Definitions + Zod Schema

```
You are C1-A, sub-agent of CODEX-1. Your task: add OAuth session types and session_id.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file, you MUST:
1. Read: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add a CHECK-IN row: | <timestamp> | C1-A | CHECK-IN | <file> | 🔵 IN PROGRESS | <note> |
4. Update the File Lock Table: set C1-A as owner, status 🔴 Locked

After completing ALL edits:
5. Add CHECK-OUT row: | <timestamp> | C1-A | CHECK-OUT | <file> | ✅ DONE | <note> |
6. Update File Lock Table: clear owner, set 🟢 Available
═══════════════════════════════════════════════════════════════

FILES YOU OWN (and ONLY these):
- C:/VMs/Projetos/Automonous_Agentic/src/shared/canvas-types.ts
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-document/schema.ts

Read both files first. Then make these EXACT changes:

1. In canvas-types.ts, AFTER the CanvasDocument interface (after line 63), add:

/** Represents an authenticated CLI session detected on the host machine. */
export interface AuthSession {
  id: string;
  cli_provider: string;
  account_email: string;
  config_dir: string;
  status: 'active' | 'expiring' | 'expired';
  expires_at?: string;
  subscription_type?: string;
  billing_label?: string;
  auth_method: 'oauth' | 'sso' | 'gcloud' | 'api_key';
}

2. In canvas-types.ts, add session_id to CanvasNode.data (after the color field on line 20):
    /** Links this agent to a specific OAuth session for billing/auth routing. */
    session_id?: string;

3. In canvas-types.ts, add session_id to CanvasNodeSnapshot (after provider field):
    session_id?: string;

4. In schema.ts, add to the canvasNodeSchema data Zod object (after the color field):
    session_id: z.string().optional(),

Run npx tsc --noEmit after your edits to verify.
Do NOT touch any other files.
```

---

### C1-B: Session Discovery Module

```
You are C1-B, sub-agent of CODEX-1. Create the session discovery module.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file, you MUST:
1. Read: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add CHECK-IN row: | <timestamp> | C1-B | CHECK-IN | src/api/session-discovery.ts | 🔵 IN PROGRESS | Creating new file |
4. Update File Lock Table: set C1-B as owner, 🔴 Locked

After completing:
5. Add CHECK-OUT row with ✅ DONE
6. Update File Lock Table: 🟢 Available
═══════════════════════════════════════════════════════════════

FILE YOU CREATE (NEW — does not exist yet):
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts

First read the CAO client to understand patterns:
- C:/VMs/Projetos/Automonous_Agentic/src/api/cao-client.ts

Create src/api/session-discovery.ts with:

1. DiscoveredSession interface:
   { id, cli_provider, account_email, config_dir, status, expires_at?, subscription_type?, auth_method }

2. discoverSessions() async function:
   - Try GET {caoBaseUrl}/auth/sessions (parse JSON)
   - If 404/error, fallback: GET {caoBaseUrl}/agents/providers, map available ones to session stubs
   - Return DiscoveredSession[]

3. resolveSessionEnv(session, model?) function:
   - claude_code → { CLAUDE_CONFIG_DIR: session.config_dir, ANTHROPIC_MODEL: model }
   - codex → { OPENAI_MODEL: model }
   - gemini_cli → { GEMINI_MODEL: model }
   - kiro_cli → { KIRO_HOME: session.config_dir }
   - Return Record<string, string>

4. triggerLogin(cliProvider, configDir?) async function:
   - POST {caoBaseUrl}/auth/login with { provider, config_dir }

Read cao-client.ts to get the base URL pattern. Use fetch() directly.
Do NOT modify any existing files.
```

---

### C1-C: Session Zustand Store

```
You are C1-C, sub-agent of CODEX-1. Create the session Zustand store.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file, you MUST:
1. Read: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add CHECK-IN row: | <timestamp> | C1-C | CHECK-IN | src/api/session-store.ts | 🔵 IN PROGRESS | Creating new file |
4. Update File Lock Table: set C1-C as owner, 🔴 Locked

After completing:
5. Add CHECK-OUT row with ✅ DONE
6. Update File Lock Table: 🟢 Available
═══════════════════════════════════════════════════════════════

FILE YOU CREATE (NEW):
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts

First read existing Zustand stores to match patterns:
- C:/VMs/Projetos/Automonous_Agentic/src/api/key-store/store.ts
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-reconciler/deploy-store.ts

Create src/api/session-store.ts with:

import { create } from 'zustand';
import { discoverSessions, triggerLogin, type DiscoveredSession } from './session-discovery';

State: sessions, loading, error, lastRefreshed
Actions: refresh(), addSession(), getSession(), getSessionsForProvider(), clearError()

Follow the exact same Zustand create() pattern as key-store/store.ts.
Do NOT modify any existing files.
```

---

# WAVE 2 — CODEX-2 Lead + 3 Sub-Agents

> ⚠️ Only start after CODEX-1-LEAD confirms Wave 1 gate GREEN in AGENT_LEDGER.md

## CODEX-2 Lead Prompt

```
You are CODEX-2-LEAD, the lead agent for the Frontend/UI Layer workstream.

You coordinate 3 sub-agents: C2-A, C2-B, C2-C.

IMPORTANT: Wave 1 must be complete. Verify by reading:
C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md
Confirm C1-A, C1-B, C1-C all have CHECK-OUT rows with ✅ DONE.

Your responsibilities:
1. Verify all 3 sub-agents have completed
2. Run: npx tsc --noEmit
3. CHECK-IN/CHECK-OUT in .planning/AGENT_LEDGER.md

Wave 2 Gate: npx tsc --noEmit = 0 errors
Report: list all files changed, all ledger entries, gate result.
```

---

### C2-A: Session Management Page

```
You are C2-A, sub-agent of CODEX-2. Create the Session Management page.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Agent name: C2-A
1. Read .planning/AGENT_LEDGER.md
2. CHECK-IN before creating files
3. CHECK-OUT after completion
═══════════════════════════════════════════════════════════════

FILES YOU CREATE (NEW):
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/index.ts
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/SessionsPage.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/sessions.css

First read to understand design system and page patterns:
- C:/VMs/Projetos/Automonous_Agentic/src/index.css (CSS variables)
- C:/VMs/Projetos/Automonous_Agentic/src/dashboard/DashboardPage.tsx (page pattern)
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/canvas-builder.css (component styles)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (Wave 1 output)

Create:

1. src/sessions/index.ts — barrel:
   export { SessionsPage } from './SessionsPage';

2. src/sessions/sessions.css — premium dark-mode styling matching the app

3. src/sessions/SessionsPage.tsx:
   - Import useSessionStore from '@/api/session-store'
   - useEffect → refresh() on mount
   - Group sessions by cli_provider
   - Each session card: status dot (🟢/🟡/🔴), email, config_dir, expires_at
   - Header: "AUTH SESSIONS" + [🔄 Refresh] button
   - [+ Add Session] button
   - Loading spinner, error banner, empty state
   - Premium glassmorphism card styling, status badges with glow
   - document.title = 'Sessions · AgentVerse'

Do NOT modify any existing files.
```

---

### C2-B: Config Panel Session Dropdown

```
You are C2-B, sub-agent of CODEX-2. Add session selector to config panel.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Agent name: C2-B
1. Read .planning/AGENT_LEDGER.md
2. CHECK-IN before editing
3. CHECK-OUT after completion
═══════════════════════════════════════════════════════════════

FILE YOU OWN:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/BlockConfigurationPanel.tsx

Read first:
- The full BlockConfigurationPanel.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts (Wave 1)
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-builder/provider-options.ts

Changes:
1. Import useSessionStore from '@/api/session-store'
2. Get sessions filtered by current provider
3. Add session dropdown AFTER provider dropdown, BEFORE model dropdown
4. Options: "🟢 email" / "🟡 email (expiring)" / "Auto (default session)"
5. onChange → patchData({ session_id: value })
6. useEffect to refresh sessions on mount if empty
7. Show helper text if no sessions available

Do NOT touch any other files.
```

---

### C2-C: Router + Nav Integration

```
You are C2-C, sub-agent of CODEX-2. Add /sessions route and nav link.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Agent name: C2-C
1. Read .planning/AGENT_LEDGER.md
2. CHECK-IN before editing
3. CHECK-OUT after completion
═══════════════════════════════════════════════════════════════

FILES YOU OWN:
- C:/VMs/Projetos/Automonous_Agentic/src/shell/router.tsx
- C:/VMs/Projetos/Automonous_Agentic/src/shell/NavBar.tsx

Read both files first. Then:

1. router.tsx — add import + route:
   // eslint-disable-next-line agentverse/no-sideways-capability-imports
   import { SessionsPage } from '@/sessions';

   Add route AFTER 'memory', BEFORE 'settings/providers':
   { path: 'sessions', element: <SessionsPage /> },

2. NavBar.tsx — add nav link AFTER Memory link, BEFORE </div>:
   <NavLink to="/sessions" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`} id="nav-link-sessions">
     Sessions
   </NavLink>

Do NOT touch any other files.
```

---

# WAVE 3 — GEMINI-1 Lead + 3 Sub-Agents

> ⚠️ Only start after CODEX-2-LEAD confirms Wave 2 gate GREEN in AGENT_LEDGER.md

## GEMINI-1 Lead Prompt

```
You are GEMINI-1-LEAD, the lead for Integration/QA.
You coordinate G1-A, G1-B, G1-C.

Verify Wave 2 complete: read .planning/AGENT_LEDGER.md, confirm all C2-* CHECK-OUT ✅.

Your responsibilities:
1. Verify all 3 sub-agents completed
2. Run: npx tsc --noEmit AND npx vitest run
3. CHECK-IN/CHECK-OUT in .planning/AGENT_LEDGER.md

Final Gate: tsc = 0 errors + vitest = 405+ passed, 0 failed
```

---

### G1-A: Reconciler Session Env Injection

```
You are G1-A, sub-agent of GEMINI-1. Wire session into the reconciler.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Agent name: G1-A
1. Read .planning/AGENT_LEDGER.md
2. CHECK-IN before editing
3. CHECK-OUT after completion
═══════════════════════════════════════════════════════════════

FILE YOU OWN:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-reconciler/reconciler.ts

Read first:
- reconciler.ts (full)
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts
- C:/VMs/Projetos/Automonous_Agentic/src/shared/canvas-types.ts

Changes:
1. Add session_id?: string to generateProfileMarkdown param type
2. If data.session_id → add session_id line in YAML output
3. Pass session_id at all 3 call sites
4. Add session_id to all CanvasNodeSnapshot objects
5. Add sessionChanged to diff detection logic

Do NOT touch any other files.
```

---

### G1-B: API Types Extension

```
You are G1-B, sub-agent of GEMINI-1. Extend CAO API types.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Agent name: G1-B
1. Read .planning/AGENT_LEDGER.md
2. CHECK-IN before editing
3. CHECK-OUT after completion
═══════════════════════════════════════════════════════════════

FILES YOU OWN:
- C:/VMs/Projetos/Automonous_Agentic/src/api/types.ts
- C:/VMs/Projetos/Automonous_Agentic/src/api/cao-client.ts

Changes:
1. types.ts: add env_vars?: Record<string, string> to CreateSessionInput
2. cao-client.ts: add listAuthSessions() method → GET /auth/sessions

Do NOT touch any other files.
```

---

### G1-C: Unit Tests

```
You are G1-C, sub-agent of GEMINI-1. Write session discovery tests.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL
Agent name: G1-C
1. Read .planning/AGENT_LEDGER.md
2. CHECK-IN before creating file
3. CHECK-OUT after completion
═══════════════════════════════════════════════════════════════

FILE YOU CREATE (NEW):
- C:/VMs/Projetos/Automonous_Agentic/src/api/__tests__/session-discovery.test.ts

Read:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts
- Any existing test file for vitest patterns

Tests:
1. resolveSessionEnv for claude_code → CLAUDE_CONFIG_DIR + ANTHROPIC_MODEL
2. resolveSessionEnv for codex → OPENAI_MODEL
3. resolveSessionEnv for gemini_cli → GEMINI_MODEL
4. resolveSessionEnv for kiro_cli → KIRO_HOME
5. resolveSessionEnv for unknown → empty object

Use vitest. Do NOT modify any existing files.
```

---

## Quality Gates

| Gate | Command | Criteria | Who Runs |
|------|---------|----------|----------|
| Wave 1 | `npx tsc --noEmit` | 0 errors | CODEX-1-LEAD |
| Wave 2 | `npx tsc --noEmit` | 0 errors | CODEX-2-LEAD |
| Wave 3 | `npx tsc --noEmit` + `npx vitest run` | 0 errors + 405+ passed | GEMINI-1-LEAD |

---

## Dependency Graph

```
Wave 1 (parallel — no deps):
  C1-A (types)──────┐
  C1-B (discovery)───┤ parallel, no file conflicts
  C1-C (store)───────┘
       │
  [GATE: tsc — run by CODEX-1-LEAD]
       │
Wave 2 (parallel — depends on Wave 1):
  C2-A (page)────── reads C1-C output
  C2-B (panel)───── reads C1-C output
  C2-C (router)──── reads C2-A output
       │
  [GATE: tsc — run by CODEX-2-LEAD]
       │
Wave 3 (parallel — depends on Wave 1+2):
  G1-A (reconciler)── reads C1-A + C1-B
  G1-B (API types)─── independent
  G1-C (tests)─────── reads C1-B
       │
  [FINAL GATE: tsc + vitest — run by GEMINI-1-LEAD]
```