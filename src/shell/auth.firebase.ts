/**
 * Firebase Auth backend for the AgentVerse shell.
 *
 * Loaded lazily by `auth.ts` only when VITE_AUTH_PROVIDER=firebase. Local-mode
 * bundles without that env var never reach this code path; Vite tree-shakes
 * the dynamic import.
 *
 * Required env vars (.env.local for dev, .env.production.local for prod):
 *   VITE_FIREBASE_API_KEY
 *   VITE_FIREBASE_AUTH_DOMAIN
 *   VITE_FIREBASE_PROJECT_ID
 *   VITE_FIREBASE_APP_ID
 */

import { initializeApp, getApps, type FirebaseApp } from 'firebase/app';
import {
  getAuth,
  GoogleAuthProvider,
  GithubAuthProvider,
  signInWithPopup,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
  sendPasswordResetEmail,
  signOut as firebaseSdkSignOut,
  onAuthStateChanged,
  type Auth,
  type User,
  type Unsubscribe,
} from 'firebase/auth';

export interface FirebaseAuthUserShape {
  uid: string;
  email: string | null;
  displayName: string | null;
  photoURL: string | null;
  providerId: string | null;
  getIdToken: () => Promise<string>;
}

let app: FirebaseApp | null = null;
let auth: Auth | null = null;

function ensureInit(): Auth {
  if (auth) return auth;

  const cfg = {
    apiKey: import.meta.env.VITE_FIREBASE_API_KEY ?? '',
    authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN ?? '',
    projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID ?? '',
    appId: import.meta.env.VITE_FIREBASE_APP_ID ?? '',
  };
  if (!cfg.apiKey || !cfg.projectId || !cfg.appId) {
    throw new Error(
      '[auth.firebase] missing required VITE_FIREBASE_* env vars; cannot initialize Firebase Auth.',
    );
  }

  app = getApps()[0] ?? initializeApp(cfg);
  auth = getAuth(app);
  return auth;
}

function adapt(user: User | null): FirebaseAuthUserShape | null {
  if (!user) return null;
  return {
    uid: user.uid,
    email: user.email,
    displayName: user.displayName,
    photoURL: user.photoURL,
    providerId: user.providerData[0]?.providerId ?? null,
    getIdToken: () => user.getIdToken(),
  };
}

export async function getCurrentFirebaseIdToken(): Promise<string | null> {
  const a = ensureInit();
  const u = a.currentUser;
  return u ? u.getIdToken() : null;
}

export async function getCurrentFirebaseUser(): Promise<FirebaseAuthUserShape | null> {
  const a = ensureInit();
  return adapt(a.currentUser);
}

// ─── OAuth: Google ─────────────────────────────────────────────────────────
export async function signInWithGoogle(): Promise<FirebaseAuthUserShape> {
  const a = ensureInit();
  const provider = new GoogleAuthProvider();
  provider.setCustomParameters({ prompt: 'select_account' });
  const result = await signInWithPopup(a, provider);
  const adapted = adapt(result.user);
  if (!adapted) throw new Error('Google sign-in succeeded but returned no user');
  return adapted;
}

// ─── OAuth: GitHub ─────────────────────────────────────────────────────────
export async function signInWithGitHub(): Promise<FirebaseAuthUserShape> {
  const a = ensureInit();
  const provider = new GithubAuthProvider();
  // Request the user's email so we can show it in the NavBar.
  provider.addScope('user:email');
  const result = await signInWithPopup(a, provider);
  const adapted = adapt(result.user);
  if (!adapted) throw new Error('GitHub sign-in succeeded but returned no user');
  return adapted;
}

// ─── Email + password ──────────────────────────────────────────────────────
export async function signInWithEmail(
  email: string,
  password: string,
): Promise<FirebaseAuthUserShape> {
  const a = ensureInit();
  const result = await signInWithEmailAndPassword(a, email, password);
  const adapted = adapt(result.user);
  if (!adapted) throw new Error('Email sign-in succeeded but returned no user');
  return adapted;
}

export async function signUpWithEmail(
  email: string,
  password: string,
): Promise<FirebaseAuthUserShape> {
  const a = ensureInit();
  const result = await createUserWithEmailAndPassword(a, email, password);
  const adapted = adapt(result.user);
  if (!adapted) throw new Error('Account creation succeeded but returned no user');
  return adapted;
}

export async function sendPasswordReset(email: string): Promise<void> {
  const a = ensureInit();
  await sendPasswordResetEmail(a, email);
}

// ─── Common ────────────────────────────────────────────────────────────────
export async function firebaseSignOut(): Promise<void> {
  const a = ensureInit();
  await firebaseSdkSignOut(a);
}

export function subscribeAuthChanges(
  callback: (user: FirebaseAuthUserShape | null) => void,
): Unsubscribe {
  const a = ensureInit();
  return onAuthStateChanged(a, (u) => callback(adapt(u)));
}
