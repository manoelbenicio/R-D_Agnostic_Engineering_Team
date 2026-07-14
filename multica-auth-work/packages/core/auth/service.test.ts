import { describe, expect, it, vi } from "vitest";
import type { LoginResponse } from "../api/client";
import { SimpleAuthService } from "./service";

describe("SimpleAuthService", () => {
  it("delegates credentials to ApiClient.login", async () => {
    const response = {
      token: "jwt",
      user: { id: "user-1", email: "user@example.com" },
    } as LoginResponse;
    const login = vi.fn().mockResolvedValue(response);
    const service = new SimpleAuthService({ login });

    await expect(
      service.login("user@example.com", "correct horse battery staple"),
    ).resolves.toBe(response);
    expect(login).toHaveBeenCalledWith(
      "user@example.com",
      "correct horse battery staple",
    );
  });
});
