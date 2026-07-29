import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { mapAuthError } from "./auth-error";

describe("mobile password-login contract", () => {
  it("uses secure password entry and replaces the route after login", () => {
    const source = readFileSync(
      resolve(process.cwd(), "app/(auth)/login.tsx"),
      "utf8",
    );

    expect(source).toContain("secureTextEntry");
    expect(source).toContain('textContentType="password"');
    expect(source).toContain('autoComplete="current-password"');
    expect(source).toContain('router.replace("/")');
    expect(source).not.toContain("/verify");
  });

  it("maps credential failures to one generic message", () => {
    expect(mapAuthError(new Error("invalid credentials"), "fallback")).toBe(
      "Invalid email or password.",
    );
    const unauthorized = Object.assign(new Error("request failed"), {
      status: 401,
    });
    expect(mapAuthError(unauthorized, "fallback")).toBe(
      "Invalid email or password.",
    );
  });
});
