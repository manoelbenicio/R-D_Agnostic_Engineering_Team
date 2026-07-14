"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@multica/ui/components/ui/card";
import { Input } from "@multica/ui/components/ui/input";
import { Button } from "@multica/ui/components/ui/button";
import { Label } from "@multica/ui/components/ui/label";
import { useAuthStore } from "@multica/core/auth";
import { workspaceKeys } from "@multica/core/workspace/queries";
import { api } from "@multica/core/api";
import type { User } from "@multica/core/types";
import { useT } from "../i18n";

interface GoogleAuthConfig {
  clientId: string;
  redirectUri: string;
  /** Opaque state passed through Google OAuth (e.g. "platform:desktop"). */
  state?: string;
}

interface CliCallbackConfig {
  /** Validated localhost callback URL. */
  url: string;
  /** Opaque state to pass back to the CLI. */
  state: string;
}

interface LoginPageProps {
  logo?: ReactNode;
  /** Called after the workspace list is available in the Query cache. */
  onSuccess: () => void;
  google?: GoogleAuthConfig;
  cliCallback?: CliCallbackConfig;
  /** Called after a token is obtained (for example, to set the web marker cookie). */
  onTokenObtained?: () => void;
  /** Desktop can override Google login to open the system browser. */
  onGoogleLogin?: () => void;
  extra?: ReactNode;
}

export function redirectToCliCallback(url: string, token: string, state: string) {
  const separator = url.includes("?") ? "&" : "?";
  window.location.href = `${url}${separator}token=${encodeURIComponent(token)}&state=${encodeURIComponent(state)}`;
}

/** Accept loopback and RFC 1918 HTTP callbacks used by local and WSL CLIs. */
export function validateCliCallback(cliCallback: string): boolean {
  try {
    const cbUrl = new URL(cliCallback);
    if (cbUrl.protocol !== "http:") return false;
    const host = cbUrl.hostname;
    if (host === "localhost" || host === "127.0.0.1") return true;
    if (/^10\./.test(host)) return true;
    if (/^172\.(1[6-9]|2\d|3[01])\./.test(host)) return true;
    return /^192\.168\./.test(host);
  } catch {
    return false;
  }
}

export function LoginPage({
  logo,
  onSuccess,
  google,
  cliCallback,
  onTokenObtained,
  onGoogleLogin,
  extra,
}: LoginPageProps) {
  const { t } = useT("auth");
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<"credentials" | "cli_confirm">("credentials");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [existingUser, setExistingUser] = useState<User | null>(null);
  const authSourceRef = useRef<"cookie" | "localStorage">("cookie");

  // Preserve the existing CLI authorization shortcut. Cookie auth wins over
  // a potentially stale desktop/localStorage bearer token.
  useEffect(() => {
    if (!cliCallback) return;

    api.setToken(null);
    api
      .getMe()
      .then((user) => {
        authSourceRef.current = "cookie";
        setExistingUser(user);
        setMode("cli_confirm");
      })
      .catch(() => {
        const token = localStorage.getItem("multica_token");
        if (!token) return;

        api.setToken(token);
        api
          .getMe()
          .then((user) => {
            authSourceRef.current = "localStorage";
            setExistingUser(user);
            setMode("cli_confirm");
          })
          .catch(() => {
            api.setToken(null);
            localStorage.removeItem("multica_token");
          });
      });
  }, [cliCallback]);

  const handleLogin = useCallback(
    async (event: FormEvent) => {
      event.preventDefault();
      const normalizedEmail = email.trim();
      if (!normalizedEmail) {
        setError(t(($) => $.common.email_required));
        return;
      }
      if (!password) {
        setError(t(($) => $.common.password_required));
        return;
      }

      setLoading(true);
      setError("");
      try {
        const { token } = await useAuthStore
          .getState()
          .login(normalizedEmail, password);
        onTokenObtained?.();

        if (cliCallback) {
          // The CLI needs the bearer token even when the web app itself uses
          // the HttpOnly cookie set by POST /auth/login.
          localStorage.setItem("multica_token", token);
          api.setToken(token);
          redirectToCliCallback(cliCallback.url, token, cliCallback.state);
          return;
        }

        const workspaces = await api.listWorkspaces();
        queryClient.setQueryData(workspaceKeys.list(), workspaces);
        onSuccess();
      } catch (loginError) {
        setError(
          loginError instanceof Error
            ? loginError.message
            : t(($) => $.errors.login_failed),
        );
      } finally {
        setLoading(false);
      }
    },
    [cliCallback, email, onSuccess, onTokenObtained, password, queryClient, t],
  );

  const handleCliAuthorize = async () => {
    if (!cliCallback) return;
    setLoading(true);
    setError("");

    try {
      let token: string;
      if (authSourceRef.current === "localStorage") {
        const stored = localStorage.getItem("multica_token");
        if (!stored) throw new Error("token missing");
        token = stored;
      } else {
        token = (await api.issueCliToken()).token;
      }
      onTokenObtained?.();
      redirectToCliCallback(cliCallback.url, token, cliCallback.state);
    } catch {
      setError(t(($) => $.errors.cli_auth_failed));
      setExistingUser(null);
      setMode("credentials");
      setLoading(false);
    }
  };

  const handleGoogleLogin = () => {
    if (onGoogleLogin) {
      onGoogleLogin();
      return;
    }
    if (!google) return;
    const params = new URLSearchParams({
      client_id: google.clientId,
      redirect_uri: google.redirectUri,
      response_type: "code",
      scope: "openid email profile",
      access_type: "offline",
      prompt: "select_account",
    });
    if (google.state) params.set("state", google.state);
    window.location.href = `https://accounts.google.com/o/oauth2/v2/auth?${params}`;
  };

  if (mode === "cli_confirm" && existingUser) {
    return (
      <main className="flex min-h-svh items-center justify-center bg-background p-4 text-foreground">
        <Card className="w-full max-w-sm border-border bg-card shadow-sm">
          <CardHeader className="text-center">
            {logo && <div className="mx-auto mb-4">{logo}</div>}
            <CardTitle className="text-2xl">{t(($) => $.cli.title)}</CardTitle>
            <CardDescription>
              {t(($) => $.cli.description, { email: existingUser.email })}
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Button onClick={handleCliAuthorize} disabled={loading} size="lg">
              {loading ? t(($) => $.cli.authorizing) : t(($) => $.cli.authorize)}
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setExistingUser(null);
                setMode("credentials");
              }}
            >
              {t(($) => $.cli.different_account)}
            </Button>
            {error && (
              <p className="text-center text-sm text-destructive" role="alert">
                {error}
              </p>
            )}
          </CardContent>
        </Card>
      </main>
    );
  }

  return (
    <main className="flex min-h-svh items-center justify-center bg-background p-4 text-foreground">
      <Card className="w-full max-w-sm border-border bg-card shadow-sm">
        <CardHeader className="text-center">
          {logo && <div className="mx-auto mb-4">{logo}</div>}
          <CardTitle className="text-2xl">{t(($) => $.signin.title)}</CardTitle>
          <CardDescription>{t(($) => $.signin.description)}</CardDescription>
        </CardHeader>
        <CardContent>
          <form id="login-form" onSubmit={handleLogin} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="login-email">{t(($) => $.common.email)}</Label>
              <Input
                id="login-email"
                type="email"
                autoComplete="email"
                placeholder={t(($) => $.common.email_placeholder)}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                autoFocus
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="login-password">{t(($) => $.common.password)}</Label>
              <Input
                id="login-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
            </div>
            {error && (
              <p className="text-sm text-destructive" role="alert">
                {error}
              </p>
            )}
          </form>
        </CardContent>
        <CardFooter className="flex flex-col gap-3">
          <Button
            type="submit"
            form="login-form"
            className="w-full"
            size="lg"
            disabled={loading}
          >
            {loading ? t(($) => $.signin.signing_in) : t(($) => $.signin.submit)}
          </Button>
          {(google || onGoogleLogin) && (
            <>
              <div className="relative w-full">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t border-border" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">
                    {t(($) => $.signin.divider)}
                  </span>
                </div>
              </div>
              <Button
                type="button"
                variant="outline"
                className="w-full"
                size="lg"
                onClick={handleGoogleLogin}
                disabled={loading}
              >
                <svg aria-hidden="true" className="mr-2 h-4 w-4" viewBox="0 0 24 24">
                  <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" />
                  <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
                  <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
                  <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
                </svg>
                {t(($) => $.signin.google)}
              </Button>
            </>
          )}
          {extra && <div className="w-full pt-1 text-center">{extra}</div>}
        </CardFooter>
      </Card>
    </main>
  );
}
