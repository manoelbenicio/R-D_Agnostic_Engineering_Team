import { existsSync, readFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { describe, expect, it } from "vitest";

const webRoot = process.cwd();
const repoRoot = resolve(webRoot, "../..");

function source(relativePath: string): string {
  return readFileSync(join(repoRoot, relativePath), "utf8");
}

describe("onboarding auth source gate", () => {
  it("keeps removed marketing and email-code surfaces out of the web app", () => {
    for (const removedPath of [
      "app/(landing)",
      "features/landing",
      "content/use-cases",
      "public/usecases",
    ]) {
      expect(existsSync(join(webRoot, removedPath)), removedPath).toBe(false);
    }

    const apiClient = source("packages/core/api/client.ts");
    expect(apiClient).toContain('this.fetch<unknown>("/auth/login"');
    expect(apiClient).not.toMatch(/\b(?:sendCode|verifyCode)\s*\(/);

    const packageJson = source("apps/web/package.json");
    expect(packageJson).not.toMatch(/fumadocs|input-otp/);
  });

  it("locks login colors to the shared Kanban and Agents token vocabulary", () => {
    const loginPage = source("packages/views/auth/login-page.tsx");

    for (const token of [
      "bg-background",
      "text-foreground",
      "bg-card",
      "border-border",
      "text-muted-foreground",
      "text-destructive",
    ]) {
      expect(loginPage, token).toContain(token);
    }
    expect(loginPage).not.toMatch(
      /(?:bg|text|border)-(?:white|black|slate|gray|zinc|neutral)(?:-|\b)/,
    );
  });

  it("keeps root and onboarding CTAs away from deleted internal routes", () => {
    expect(source("apps/web/app/page.tsx")).toContain('redirect("/login")');

    const onboardingSources = [
      source("packages/views/onboarding/steps/step-welcome.tsx"),
      source("packages/views/onboarding/steps/step-platform-fork.tsx"),
    ].join("\n");
    expect(onboardingSources).toContain("DESKTOP_RELEASES_URL");
    expect(onboardingSources).not.toMatch(/["']\/download["']/);
  });
});
