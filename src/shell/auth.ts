/**
 * Auth scaffolding — Firebase Auth integration is opt-in and disabled by
 * default. It activates when the SPA is built with VITE_AUTH_PROVIDER=firebase
 * and the standard VITE_FIREBASE_* env vars are present.
 *
 * Sign-in methods supported (when VITE_AUTH_PROVIDER=firebase):
 *   - Google OAuth        signIn({ method: 'google' })
 *   - GitHub OAuth        signIn({ method: 'github' })
 *   - Email + password    signIn({ method: 'email', email, password })
 *                         signUp({ method: 'email', email, password })
 *                         sendPasswordReset(email)
 *
 * The Firebase SDK is loaded lazily so local-mode bundles without
 * VITE_AUTH_PROVIDER stay free of Firebase code at runtime.
 */

export type AuthProvider = 'none' | 'firebase';
export type SignInMethod = 'google' | 'github' | 'email';

export interface AuthUser {
  uid: string;
  email: string | null;
  displayName: string | null;
  photoURL: string | null;
  providerId: string | null;
}

export interface SignInArgs {
  method: SignInMethod;
  email?: string;
  password?: string;
}

export function getAuthProviderName(): AuthProvider {
  const value = (import.meta.env.VITE_AUTH_PROVIDER ?? '').toString().toLowerCase();
  return value === 'firebase' ? 'firebase' : 'none';
}

export function isAuthEnabled(): boolean {
  return getAuthProviderName() !== 'none';
}

export function isAuthRequired(): boolean {
  return (import.meta.env.VITE_AUTH_REQUIRED ?? '').toString().toLowerCase() === 'true';
}

/** Returns the current JWT, or null when auth is disabled / no user. */
export async function getAuthToken(): Promise<string | null> {
  if (!isAuthEnabled()) return null;
  if (getAuthProviderName() === 'firebase') {
    try {
      const { getCurrentFirebaseIdToken } = await import('./auth.firebase');
      return await getCurrentFirebaseIdToken();
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('[auth] Firebase token fetch failed:', err);
      return null;
    }
  }
  return null;
}

/**
 * Trigger the configured sign-in flow.
 *
 * Examples:
 *   signIn({ method: 'google' })
 *   signIn({ method: 'github' })
 *   signIn({ method: 'email', email: 'a@b.com', password: '…' })
 */
export async function signIn(args: SignInArgs): Promise<AuthUser> {
  if (getAuthProviderName() !== 'firebase') {
    throw new Error('signIn() called but VITE_AUTH_PROVIDER is not "firebase"');
  }
  const fb = await import('./auth.firebase');

  const adapted = await (async () => {
    switch (args.method) {
      case 'google':
        return fb.signInWithGoogle();
      case 'github':
        return fb.signInWithGitHub();
      case 'email': {
        if (!args.email || !args.password) {
          throw new Error('Email sign-in requires both email and password');
        }
        return fb.signInWithEmail(args.email, args.password);
      }
      default:
        throw new Error(`Unknown sign-in method: ${args.method as string}`);
    }
  })();

  return {
    uid: adapted.uid,
    email: adapted.email,
    displayName: adapted.displayName,
    photoURL: adapted.photoURL,
    providerId: adapted.providerId,
  };
}

/**
 * Create a new account using email + password. Currently the only sign-up
 * path; OAuth providers create accounts automatically on first sign-in.
 */
export async function signUp(args: { method: 'email'; email: string; password: string }): Promise<AuthUser> {
  if (getAuthProviderName() !== 'firebase') {
    throw new Error('signUp() called but VITE_AUTH_PROVIDER is not "firebase"');
  }
  if (args.method !== 'email') {
    throw new Error(`signUp() only supports method='email' (got ${args.method})`);
  }
  const { signUpWithEmail } = await import('./auth.firebase');
  const u = await signUpWithEmail(args.email, args.password);
  return {
    uid: u.uid,
    email: u.email,
    displayName: u.displayName,
    photoURL: u.photoURL,
    providerId: u.providerId,
  };
}

/** Send a password-reset email to the given address. */
export async function sendPasswordReset(email: string): Promise<void> {
  if (getAuthProviderName() !== 'firebase') {
    throw new Error('sendPasswordReset() called but VITE_AUTH_PROVIDER is not "firebase"');
  }
  const { sendPasswordReset: sendReset } = await import('./auth.firebase');
  await sendReset(email);
}

/** Sign the current user out of the configured provider. */
export async function signOut(): Promise<void> {
  if (getAuthProviderName() !== 'firebase') return;
  const { firebaseSignOut } = await import('./auth.firebase');
  await firebaseSignOut();
}

/**
 * Subscribe to auth state changes. Fires once with the current user (or
 * null) immediately, then again on every change. Returns the unsubscribe
 * function.
 *
 * When auth is disabled this is a no-op subscription.
 */
export function subscribeAuthChanges(
  callback: (user: AuthUser | null) => void,
): () => void {
  if (getAuthProviderName() !== 'firebase') {
    queueMicrotask(() => callback(null));
    return () => undefined;
  }

  let unsubscribe: (() => void) | null = null;
  let cancelled = false;

  void (async () => {
    try {
      const { subscribeAuthChanges: subscribe } = await import('./auth.firebase');
      if (cancelled) return;
      unsubscribe = subscribe((u) =>
        callback(
          u
            ? {
                uid: u.uid,
                email: u.email,
                displayName: u.displayName,
                photoURL: u.photoURL,
                providerId: u.providerId,
              }
            : null,
        ),
      );
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('[auth] subscribe failed:', err);
      callback(null);
    }
  })();

  return () => {
    cancelled = true;
    if (unsubscribe) unsubscribe();
  };
}

/** Convenience: returns the current session snapshot. */
export async function getAuthSession(): Promise<{
  provider: AuthProvider;
  signedIn: boolean;
  user: AuthUser | null;
}> {
  const provider = getAuthProviderName();
  if (provider === 'none') return { provider: 'none', signedIn: false, user: null };
  if (provider === 'firebase') {
    const { getCurrentFirebaseUser } = await import('./auth.firebase');
    const u = await getCurrentFirebaseUser();
    return {
      provider,
      signedIn: u !== null,
      user: u
        ? {
            uid: u.uid,
            email: u.email,
            displayName: u.displayName,
            photoURL: u.photoURL,
            providerId: u.providerId,
          }
        : null,
    };
  }
  return { provider, signedIn: false, user: null };
}
