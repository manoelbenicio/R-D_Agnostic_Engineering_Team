/**
 * Auth-aware fetch boundary.
 *
 * Behaviour:
 *   - If VITE_AUTH_PROVIDER is unset / 'none' (default), this is a thin
 *     pass-through over `fetch()`. Local-mode bundles never carry auth code.
 *   - If VITE_AUTH_PROVIDER=firebase and the user is signed in, the current
 *     Firebase ID token is attached as `Authorization: Bearer <jwt>` on every
 *     outgoing request. The token is fetched lazily so SSR / unauthenticated
 *     pages do not pay the SDK cost.
 *
 * The module preserves the original signature so all existing call sites and
 * the lint rule `agentverse/no-direct-go-core-fetch` keep working unchanged.
 */
import { getAuthToken, isAuthEnabled } from './auth';

export async function appFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  if (!isAuthEnabled()) {
    return fetch(input, init);
  }

  const token = await getAuthToken();
  if (!token) {
    return fetch(input, init);
  }

  // Merge headers without mutating the caller's init object.
  const headers = new Headers(init?.headers ?? {});
  if (!headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  return fetch(input, { ...(init ?? {}), headers });
}
