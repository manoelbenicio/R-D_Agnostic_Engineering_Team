You are GEMINI-1-LEAD returning for Wave 3C — Session revoke, security hardening, and documentation.

PROJECT: C:/VMs/Projetos/Automonous_Agentic
LEDGER: C:/VMs/Projetos/Automonous_Agentic/.planning/AGENT_LEDGER.md

All core session management features are implemented. Now add the missing revoke capability, security hardening, and generate documentation.

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
TASK G1-G: Session Revoke Support
Agent name for ledger: G1-G
══════════════════════════════════════

FILES:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts
- C:/VMs/Projetos/Automonous_Agentic/src/sessions/SessionsPage.tsx

Read ALL three files FULLY first.

Changes:

1. In session-discovery.ts — add revokeSession function:

   /**
    * Revoke/logout an OAuth session by asking the CAO backend
    * to clear the credentials for a specific config directory.
    */
   export async function revokeSession(sessionId: string, cliProvider: string, configDir: string): Promise<boolean> {
     try {
       const res = await fetch(
         `${caoClient.baseUrl}/auth/sessions/${sessionId}`,
         {
           method: 'DELETE',
           headers: { 'Content-Type': 'application/json' },
           body: JSON.stringify({ provider: cliProvider, config_dir: configDir }),
         }
       );
       return res.ok;
     } catch {
       return false;
     }
   }

2. In session-store.ts — add revokeSession action:

   Add to SessionState interface:
     revokeSession: (sessionId: string) => Promise<boolean>;

   Add implementation in the create() block:
     revokeSession: async (sessionId: string) => {
       const session = get().sessions.find(s => s.id === sessionId);
       if (!session) return false;
       const { revokeSession: revoke } = await import('./session-discovery');
       const success = await revoke(sessionId, session.cli_provider, session.config_dir);
       if (success) {
         await get().refresh();
       }
       return success;
     },

3. In SessionsPage.tsx — enable the Revoke button:
   Find the Revoke button (it should be disabled or non-functional).
   Wire it to call useSessionStore().revokeSession(session.id).
   Add a confirmation prompt before revoking: window.confirm('Revoke session for ' + session.account_email + '?')
   Show loading state on the specific card during revoke.
   After success, the refresh() will update the list automatically.


══════════════════════════════════════
TASK G1-H: Credential Security Hardening
Agent name for ledger: G1-H
══════════════════════════════════════

CREATE NEW FILE:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-security.ts

Read first:
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-discovery.ts
- C:/VMs/Projetos/Automonous_Agentic/src/api/session-store.ts

Create security utilities for session management:

/**
 * Session security utilities.
 * Ensures credentials are handled safely in the frontend layer.
 */

/** Mask email for display: "john.doe@example.com" → "jo***@example.com" */
export function maskEmail(email: string): string {
  const [local, domain] = email.split('@');
  if (!domain) return email;
  const visibleChars = Math.min(2, local.length);
  return local.slice(0, visibleChars) + '***@' + domain;
}

/** Mask config directory path for logs: show only last segment */
export function maskConfigDir(configDir: string): string {
  if (!configDir) return '';
  const parts = configDir.replace(/\\/g, '/').split('/');
  const last = parts[parts.length - 1] || parts[parts.length - 2] || '';
  return '…/' + last;
}

/** Check if a session token is expiring within the given minutes */
export function isExpiringSoon(expiresAt: string | undefined, withinMinutes = 30): boolean {
  if (!expiresAt) return false;
  const expiry = new Date(expiresAt).getTime();
  const threshold = Date.now() + withinMinutes * 60 * 1000;
  return expiry <= threshold;
}

/** Sanitize session data before logging — strip any token/key fragments */
export function sanitizeForLog(session: Record<string, unknown>): Record<string, unknown> {
  const sanitized = { ...session };
  const sensitiveKeys = ['token', 'secret', 'key', 'password', 'credential', 'auth_token'];
  for (const key of Object.keys(sanitized)) {
    if (sensitiveKeys.some(sk => key.toLowerCase().includes(sk))) {
      sanitized[key] = '[REDACTED]';
    }
  }
  return sanitized;
}

Also create tests:
- C:/VMs/Projetos/Automonous_Agentic/src/api/__tests__/session-security.test.ts

Test cases:
1. maskEmail('john.doe@example.com') → 'jo***@example.com'
2. maskEmail('a@b.com') → 'a***@b.com'
3. maskEmail('invalid') → 'invalid'
4. maskConfigDir('/home/user/.claude-test') → '…/.claude-test'
5. maskConfigDir('C:\\Users\\test\\.codex') → '…/.codex'
6. maskConfigDir('') → ''
7. isExpiringSoon(undefined) → false
8. isExpiringSoon(future date 1 hour from now, 30) → false
9. isExpiringSoon(date 10 minutes from now, 30) → true
10. sanitizeForLog({ id: '1', auth_token: 'secret123' }) → { id: '1', auth_token: '[REDACTED]' }


══════════════════════════════════════
TASK G1-I: Session Management Documentation
Agent name for ledger: G1-I
══════════════════════════════════════

CREATE NEW FILE:
- C:/VMs/Projetos/Automonous_Agentic/docs/session-management.md

Write comprehensive documentation covering:

1. Overview — what the session management feature does
2. Architecture — how OAuth sessions are discovered, stored, and routed to agents
3. Per-CLI authentication:
   - Claude Code: CLAUDE_CONFIG_DIR + OAuth token
   - Codex: OPENAI_MODEL (API key or OAuth)
   - Gemini CLI: GEMINI_MODEL + gcloud auth
   - Kiro CLI: KIRO_HOME + AWS SSO
4. How session_id flows: canvas node → reconciler → CAO terminal env vars
5. Security considerations: no tokens in frontend, env var isolation, config dir sandboxing
6. UI guide: Sessions page, config panel dropdown, status badge
7. API endpoints: /auth/sessions (GET), /auth/login (POST), /auth/sessions/:id (DELETE)
8. Troubleshooting: common issues and solutions

Use markdown with mermaid diagrams for the architecture flow.


══════════════════════════════════════
QUALITY GATE
Agent name for ledger: GEMINI-1-LEAD
══════════════════════════════════════

After all 3 tasks complete:
1. Run: npx tsc --noEmit → 0 errors
2. Run: npx vitest run → 436+ tests passed (new security tests add to count)
3. CHECK-OUT as GEMINI-1-LEAD with ✅ DONE and "Wave 3C Gate PASSED"
