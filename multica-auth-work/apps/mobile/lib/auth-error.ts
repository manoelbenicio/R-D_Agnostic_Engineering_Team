/**
 * Map backend auth errors to user-facing strings. The backend returns raw
 * English messages that are fine for logs but should not surface as-is —
 * we map the known shapes to friendlier copy and fall back to the caller's
 * default for anything unrecognised.
 */
export function mapAuthError(err: unknown, fallback: string): string {
  if (!(err instanceof Error)) return fallback;
  const msg = err.message.toLowerCase();
  const status =
    "status" in err ? (err as Error & { status?: unknown }).status : undefined;
  if (
    status === 401 ||
    /invalid|incorrect|wrong|credential|unauthorized/.test(msg)
  ) {
    return "Invalid email or password.";
  }
  if (/rate.?limit|too many|throttle/.test(msg)) {
    return "Too many attempts. Wait a moment and try again.";
  }
  if (/network|fetch|timeout|unreachable/.test(msg)) {
    return "Can't reach Multica. Check your connection and retry.";
  }
  return fallback;
}
