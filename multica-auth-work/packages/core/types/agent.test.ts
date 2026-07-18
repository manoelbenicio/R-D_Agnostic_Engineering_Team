import { describe, expect, it } from "vitest";
import { RUNTIME_PROFILE_PROTOCOL_FAMILIES } from "./agent";

describe("runtime profile protocol families", () => {
  it("matches backend-supported custom profile families and keeps qoder built-in-only", () => {
    expect(RUNTIME_PROFILE_PROTOCOL_FAMILIES).toEqual([
      "claude",
      "codebuddy",
      "cline",
      "codex",
      "copilot",
      "nim",
      "opencode",
      "openclaw",
      "hermes",
      "gemini",
      "pi",
      "cursor",
      "kimi",
      "kiro",
      "antigravity",
    ]);
    expect(RUNTIME_PROFILE_PROTOCOL_FAMILIES).not.toContain("qoder");
  });
});
