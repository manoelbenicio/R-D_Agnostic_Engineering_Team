import React from 'react';
import { isAuthRequired, isAuthEnabled } from './auth';
import { useAuthStore } from './auth-store';
import { LoginScreen } from './LoginScreen';

/**
 * Wraps the app's main content with an auth requirement check.
 *
 * - VITE_AUTH_REQUIRED=false (default, local mode):
 *     Always renders children. Users can browse anonymously.
 *
 * - VITE_AUTH_REQUIRED=true (cloud mode, baked in by deploy-cloud.sh):
 *     - status=idle / loading → returns null (small flash; the auth subscription
 *       fires once almost immediately to resolve idle).
 *     - status=unauthenticated → shows the LoginScreen.
 *     - status=authenticated → renders children.
 *     - status=error → shows LoginScreen so the user can retry; the error
 *       is surfaced inside the screen.
 */
export const RequireAuthGate: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const status = useAuthStore((s) => s.status);
  const user = useAuthStore((s) => s.user);

  if (!isAuthRequired() || !isAuthEnabled()) {
    return <>{children}</>;
  }

  if (status === 'idle' || status === 'loading') {
    return null;
  }

  if (!user) {
    return <LoginScreen />;
  }

  return <>{children}</>;
};

export default RequireAuthGate;
