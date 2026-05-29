/**
 * Reactive auth state for the shell. Subscribes to Firebase Auth changes once
 * at module load, exposes a Zustand selector hook for the UI.
 *
 * When VITE_AUTH_PROVIDER is unset (local mode default), the store stays in a
 * permanent { user: null, status: 'unauthenticated' } state and the
 * subscription is a no-op.
 */
import { create } from 'zustand';
import {
  isAuthEnabled,
  signIn as authSignIn,
  signUp as authSignUp,
  signOut as authSignOut,
  sendPasswordReset as authSendPasswordReset,
  subscribeAuthChanges,
  type AuthUser,
  type SignInArgs,
} from './auth';

export type AuthStatus = 'idle' | 'loading' | 'authenticated' | 'unauthenticated' | 'error';

interface AuthStoreState {
  user: AuthUser | null;
  status: AuthStatus;
  error: string | null;
  /** True when auth is wired up at all (env var enabled). */
  enabled: boolean;
  /** Trigger the provider sign-in flow. */
  signIn: (args: SignInArgs) => Promise<void>;
  /** Create a new email-password account. */
  signUp: (args: { email: string; password: string }) => Promise<void>;
  /** Send a password-reset email. */
  sendPasswordReset: (email: string) => Promise<void>;
  /** Sign the current user out. */
  signOut: () => Promise<void>;
  /** Clear any pending error so the UI can retry cleanly. */
  clearError: () => void;
  /** Internal: replace the current user (called by the auth subscription). */
  _setUser: (user: AuthUser | null) => void;
}

export const useAuthStore = create<AuthStoreState>((set) => ({
  user: null,
  status: isAuthEnabled() ? 'idle' : 'unauthenticated',
  error: null,
  enabled: isAuthEnabled(),

  signIn: async (args) => {
    if (!isAuthEnabled()) return;
    set({ status: 'loading', error: null });
    try {
      const user = await authSignIn(args);
      set({ user, status: 'authenticated', error: null });
    } catch (err: unknown) {
      const msg = humanizeAuthError(err);
      set({ status: 'error', error: msg });
      throw err;
    }
  },

  signUp: async ({ email, password }) => {
    if (!isAuthEnabled()) return;
    set({ status: 'loading', error: null });
    try {
      const user = await authSignUp({ method: 'email', email, password });
      set({ user, status: 'authenticated', error: null });
    } catch (err: unknown) {
      const msg = humanizeAuthError(err);
      set({ status: 'error', error: msg });
      throw err;
    }
  },

  sendPasswordReset: async (email) => {
    if (!isAuthEnabled()) return;
    set({ status: 'loading', error: null });
    try {
      await authSendPasswordReset(email);
      // After sending, drop back to unauthenticated so the UI can re-render.
      set({ status: 'unauthenticated', error: null });
    } catch (err: unknown) {
      const msg = humanizeAuthError(err);
      set({ status: 'error', error: msg });
      throw err;
    }
  },

  signOut: async () => {
    if (!isAuthEnabled()) return;
    set({ status: 'loading', error: null });
    try {
      await authSignOut();
      set({ user: null, status: 'unauthenticated', error: null });
    } catch (err: unknown) {
      const msg = humanizeAuthError(err);
      set({ status: 'error', error: msg });
      throw err;
    }
  },

  clearError: () => set({ error: null, status: 'unauthenticated' }),

  _setUser: (user) => {
    const status: AuthStatus = user ? 'authenticated' : 'unauthenticated';
    set({ user, status, error: null });
  },
}));

/**
 * Friendly error mapper. Surfaces the actionable bit of the Firebase error
 * code instead of the raw 'auth/operation-not-allowed' string.
 */
function humanizeAuthError(err: unknown): string {
  if (typeof err === 'object' && err !== null && 'code' in err) {
    const code = String((err as { code: unknown }).code);
    switch (code) {
      case 'auth/popup-closed-by-user':
      case 'auth/cancelled-popup-request':
        return 'Sign-in cancelled.';
      case 'auth/popup-blocked':
        return 'The browser blocked the sign-in popup. Allow popups for this site and try again.';
      case 'auth/operation-not-allowed':
        return 'This sign-in method is not enabled in Firebase. Enable it in the Firebase Console → Authentication → Sign-in method.';
      case 'auth/invalid-credential':
      case 'auth/wrong-password':
      case 'auth/user-not-found':
        return 'Email or password is incorrect.';
      case 'auth/email-already-in-use':
        return 'An account with that email already exists. Sign in instead.';
      case 'auth/weak-password':
        return 'Password is too weak. Use at least 6 characters.';
      case 'auth/invalid-email':
        return 'That email address is not valid.';
      case 'auth/network-request-failed':
        return 'Network error contacting Firebase. Check your connection and try again.';
      case 'auth/too-many-requests':
        return 'Too many failed attempts. Try again in a few minutes.';
      case 'auth/account-exists-with-different-credential':
        return 'An account with that email already exists, signed up via a different provider.';
      default:
        // fallthrough to generic message
        break;
    }
  }
  return err instanceof Error ? err.message : String(err);
}

// Subscribe once at module load. Auth-disabled builds resolve to a no-op
// subscription per `subscribeAuthChanges` contract.
let unsubscribe: (() => void) | null = null;

export function bootstrapAuthStore(): void {
  if (unsubscribe) return;
  if (!isAuthEnabled()) return;
  unsubscribe = subscribeAuthChanges((user) => {
    useAuthStore.getState()._setUser(user);
  });
}

export function teardownAuthStore(): void {
  if (unsubscribe) {
    unsubscribe();
    unsubscribe = null;
  }
}

export function useAuthUser(): AuthUser | null {
  return useAuthStore((s) => s.user);
}

export function useAuthStatus(): AuthStatus {
  return useAuthStore((s) => s.status);
}

export function useAuthEnabled(): boolean {
  return useAuthStore((s) => s.enabled);
}
