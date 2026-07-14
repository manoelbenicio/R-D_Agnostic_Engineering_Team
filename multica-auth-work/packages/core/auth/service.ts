import type { ApiClient, LoginResponse } from "../api/client";

/** Authentication boundary used by the app state.
 * A future Firebase adapter can implement this contract without changing
 * views, routing, token persistence, or post-login navigation. */
export interface AuthService {
  login(email: string, password: string): Promise<LoginResponse>;
}

/** Current local credential provider backed by POST /auth/login. */
export class SimpleAuthService implements AuthService {
  constructor(private readonly api: Pick<ApiClient, "login">) {}

  login(email: string, password: string): Promise<LoginResponse> {
    return this.api.login(email, password);
  }
}
