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
