You are GEMINI-1-LEAD, Senior Integration Architect for the AgentVerse Session Management feature.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md
PLAN: C:/VMs/Projetos/Automonous_Agentic/.planning/EXECUTION_PLAN.md

PREREQUISITE: Wave 1 must be complete.
1. Read .planning/AGENT_LEDGER.md
2. Confirm CODEX-1-LEAD has CHECK-OUT with ✅ DONE and "Wave 1 Gate PASSED"
3. If not found, STOP and WAIT.

NOTE: Wave 2 (CODEX-2) is running in parallel on DIFFERENT files with ZERO overlap to yours. Your code dependencies are ALL from Wave 1 which is complete. Proceed immediately after confirming Wave 1 gate.

You must complete 3 tasks. Use sub-agents to parallelize where possible. Each task has its own agent name for the ledger.

═══════════════════════════════════════════════════════════════
MANDATORY: CHECK-IN / CHECK-OUT PROTOCOL

Before touching ANY source file:
1. Read: .planning/AGENT_LEDGER.md
2. Verify no other agent has the file checked-in (STOP if locked)
3. Add CHECK-IN row: | <UTC timestamp> | <agent-name> | CHECK-IN | <file> | 🔵 IN PROGRESS | <note> |
4. Update the File Lock Table: set agent as owner, 🔴 Locked

After completing ALL edits on a file:
5. Add CHECK-OUT row: | <UTC timestamp> | <agent-name> | CHECK-OUT | <file> | ✅ DONE | <note> |
6. Update File Lock Table: clear owner, set 🟢 Available

If FAILED:
7. Add FAILED row with 🔴 FAILED and error details in Notes
═══════════════════════════════════════════════════════════════


══════════════════════════════════════
TASK G1-A: Reconciler Session Env Injection
Agent name for ledger: G1-A
══════════════════════════════════════

FILE YOU OWN:
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-reconciler/reconciler.ts

CONTEXT FILES TO READ FIRST (do NOT edit these):
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts (created in Wave 1 — has resolveSessionEnv function)
- C:/VMs/Projetos/Automonous_Agentic/src/shared/canvas-types.ts (updated in Wave 1 — has session_id on CanvasNode.data and CanvasNodeSnapshot)

Read reconciler.ts FULLY. Understand the generateProfileMarkdown function signature, its 3 call sites, the snapshot diffing logic, and the profile_snapshots writes.

Then make these changes:

CHANGE 1 — Add session_id to generateProfileMarkdown param type:
Find the function parameter type object (around line 23-28 where name, role, provider, model, allowedTools, systemPrompt are defined).
Add after the model field:
  session_id?: string;

CHANGE 2 — Emit session_id in YAML output:
In the generateProfileMarkdown function body, find the if(data.model) block that pushes the model line.
Add AFTER that block:
  if (data.session_id) {
    lines.push(`session_id: ${data.session_id}`);
  }

CHANGE 3 — Pass session_id at ALL 3 call sites:
Search for "generateProfileMarkdown({" — there are exactly 3 occurrences.

At the FIRST call site (uses variable `node`), add inside the object:
  session_id: node.data.session_id,

At the SECOND call site (uses variable `nodeToUpdate`), add inside the object:
  session_id: nodeToUpdate.data.session_id,

At the THIRD call site (uses variable `nodeToAdd`), add inside the object:
  session_id: nodeToAdd.data.session_id,

CHANGE 4 — Add session_id to ALL CanvasNodeSnapshot objects:
Search for places where profile_snapshots are written — objects with shape { system_prompt:, allowedTools:, model:, provider: }.
Add to EACH snapshot object:
  session_id: <variable>.data.session_id || '',
Use the matching variable name at each location (node / nodeToUpdate / nodeToAdd).

CHANGE 5 — Add session_id to diff detection:
Find the section where the reconciler compares current node data against stored snapshots to decide if a terminal needs redeployment. Look for comparisons like:
  snapshot.model !== node.data.model
or similar boolean flags that detect config changes.

Add near those comparisons:
  const sessionChanged = (node.data.session_id || '') !== (snapshot.session_id || '');

Then find the if-condition that combines all the change flags to decide whether to trigger an update. Add || sessionChanged to that condition.


══════════════════════════════════════
TASK G1-B: API Types Extension
Agent name for ledger: G1-B
══════════════════════════════════════

FILES YOU OWN:
- C:/VMs/Projetos/Automonous_Agentic/src/api/types.ts
- C:/VMs/Projetos/Automonous_Agentic/src/api/cao-client.ts

Read both files FULLY first.

CHANGE 1 — types.ts:
Find the CreateSessionInput interface. It currently has:
  profile: string;
  working_directory: string;

Add a new field after working_directory:
  env_vars?: Record<string, string>;  // Per-terminal env var injection for OAuth session routing

If AddTerminalInput is defined as a type alias of CreateSessionInput, it inherits the new field automatically — do not duplicate.

CHANGE 2 — cao-client.ts:
Find the CaoClient class. Add a new public method AFTER the last existing method, BEFORE the closing brace of the class.

Match the coding style of existing methods (how they use this.baseUrl, fetch, headers, error handling). Add:

  /** Discover authenticated CLI sessions on the CAO host. */
  async listAuthSessions(): Promise<{
    id: string;
    cli_provider: string;
    account_email: string;
    config_dir: string;
    status: string;
    expires_at?: string;
    subscription_type?: string;
    auth_method: string;
  }[]> {
    try {
      const res = await fetch(`${this.baseUrl}/auth/sessions`, {
        method: 'GET',
        headers: { 'Accept': 'application/json' },
      });
      if (!res.ok) return [];
      return await res.json();
    } catch {
      return [];
    }
  }


══════════════════════════════════════
TASK G1-C: Unit Tests for Session Discovery
Agent name for ledger: G1-C
══════════════════════════════════════

CREATE NEW FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/api/__tests__/session-discovery.test.ts

CONTEXT FILES TO READ FIRST (do NOT edit):
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts (the module you are testing — read it to see exact exports, function signatures, and types)
- C:/VMs/Projetos/Automonous_Agentic/src/api/key-store/__tests__/mask.test.ts (existing test to copy vitest patterns from)
- C:/VMs/Projetos/Automonous_Agentic/src/canvas-document/__tests__/store.test.ts (another test pattern reference)

Read session-discovery.ts CAREFULLY. Note:
- The exact name of the exported function (resolveSessionEnv)
- Its parameter types (what fields does DiscoveredSession have?)
- Its return type (Record<string, string>)
- Which env vars it sets for each cli_provider
- When it omits a key vs sets it

Then create the test file with these 7 test cases:

import { describe, it, expect } from 'vitest';
import { resolveSessionEnv } from '../session-discovery';

describe('resolveSessionEnv', () => {

  it('sets CLAUDE_CONFIG_DIR and ANTHROPIC_MODEL for claude_code provider', () => {
    const session = {
      id: '1',
      cli_provider: 'claude_code',
      account_email: 'test@example.com',
      config_dir: '/home/user/.claude-account-a',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session, 'opus-4.8');
    expect(env.CLAUDE_CONFIG_DIR).toBe('/home/user/.claude-account-a');
    expect(env.ANTHROPIC_MODEL).toBe('opus-4.8');
  });

  it('omits CLAUDE_CONFIG_DIR when config_dir is empty string', () => {
    const session = {
      id: '2',
      cli_provider: 'claude_code',
      account_email: 'test@example.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session, 'opus-4.7');
    expect(env.CLAUDE_CONFIG_DIR).toBeUndefined();
    expect(env.ANTHROPIC_MODEL).toBe('opus-4.7');
  });

  it('sets OPENAI_MODEL for codex provider', () => {
    const session = {
      id: '3',
      cli_provider: 'codex',
      account_email: 'dev@company.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session, 'codex-5.5');
    expect(env.OPENAI_MODEL).toBe('codex-5.5');
    expect(env.CLAUDE_CONFIG_DIR).toBeUndefined();
    expect(env.KIRO_HOME).toBeUndefined();
  });

  it('sets GEMINI_MODEL for gemini_cli provider', () => {
    const session = {
      id: '4',
      cli_provider: 'gemini_cli',
      account_email: 'user@gmail.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'gcloud' as const,
    };
    const env = resolveSessionEnv(session, 'gemini-3.5-flash');
    expect(env.GEMINI_MODEL).toBe('gemini-3.5-flash');
    expect(env.CLAUDE_CONFIG_DIR).toBeUndefined();
    expect(env.OPENAI_MODEL).toBeUndefined();
  });

  it('sets KIRO_HOME for kiro_cli provider', () => {
    const session = {
      id: '5',
      cli_provider: 'kiro_cli',
      account_email: 'ops@company.com',
      config_dir: '/home/user/.kiro-production',
      status: 'active' as const,
      auth_method: 'sso' as const,
    };
    const env = resolveSessionEnv(session);
    expect(env.KIRO_HOME).toBe('/home/user/.kiro-production');
    expect(env.CLAUDE_CONFIG_DIR).toBeUndefined();
  });

  it('returns empty object for unknown provider', () => {
    const session = {
      id: '6',
      cli_provider: 'some_unknown_cli',
      account_email: 'user@test.com',
      config_dir: '/some/path',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session);
    expect(Object.keys(env)).toHaveLength(0);
  });

  it('omits model env var when model parameter is undefined', () => {
    const session = {
      id: '7',
      cli_provider: 'claude_code',
      account_email: 'user@test.com',
      config_dir: '/home/user/.claude-main',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session);
    expect(env.CLAUDE_CONFIG_DIR).toBe('/home/user/.claude-main');
    expect(env.ANTHROPIC_MODEL).toBeUndefined();
  });

});

IMPORTANT: After reading session-discovery.ts, adapt the DiscoveredSession field names and the resolveSessionEnv import to match EXACTLY what was implemented. If the field names or function signature differ from what's written above, use the ACTUAL implementation — not this template.

After creating the file, run:
  npx vitest run src/api/__tests__/session-discovery.test.ts
to verify your tests pass in isolation.


══════════════════════════════════════
FINAL QUALITY GATE
Agent name for ledger: GEMINI-1-LEAD
══════════════════════════════════════

After ALL 3 tasks (G1-A, G1-B, G1-C) are complete and checked out:

1. Run: npx tsc --noEmit
   Expected: 0 errors

2. Run: npx vitest run
   Expected: 405+ tests passed, 0 failed (the 7 new tests from G1-C add to the count)

3. If BOTH pass:
   CHECK-OUT as GEMINI-1-LEAD with ✅ DONE
   Notes: "FINAL GATE PASSED — tsc 0 errors, vitest [X] passed 0 failed"

4. If EITHER fails:
   - Identify which task (G1-A, G1-B, or G1-C) caused the failure
   - Fix the issue
   - Re-run both gates
   - Document everything in the ledger with timestamps
   - Only CHECK-OUT when both gates are green
