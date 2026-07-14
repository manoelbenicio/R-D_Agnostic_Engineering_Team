import { describe, expect, it } from "vitest";
import en from "../locales/en/auth.json";
import ja from "../locales/ja/auth.json";
import ko from "../locales/ko/auth.json";
import zhHans from "../locales/zh-Hans/auth.json";

function leafKeys(value: unknown, prefix = ""): string[] {
  if (typeof value !== "object" || value === null) return [prefix];
  return Object.entries(value).flatMap(([key, child]) =>
    leafKeys(child, prefix ? `${prefix}.${key}` : key),
  );
}

describe("auth locale parity", () => {
  it("keeps every supported locale aligned with the password-login contract", () => {
    const expected = leafKeys(en).sort();

    for (const locale of [ja, ko, zhHans]) {
      expect(leafKeys(locale).sort()).toEqual(expected);
    }

    expect(expected).toContain("common.password");
    expect(expected).toContain("signin.submit");
    expect(expected.some((key) => /code|resend|verify|download/.test(key))).toBe(
      false,
    );
  });
});
