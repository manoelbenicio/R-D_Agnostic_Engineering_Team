import { describe, it, expect } from "vitest";
import {
  checkQuickCreateCliVersion,
  isNewerCliRelease,
} from "./cli-version";

describe("checkQuickCreateCliVersion", () => {
  it("returns ok for a tagged release at or above the minimum", () => {
    expect(checkQuickCreateCliVersion("v0.2.21").state).toBe("ok");
    expect(checkQuickCreateCliVersion("0.3.1").state).toBe("ok");
  });

  it("returns too_old for a tagged release below the minimum", () => {
    expect(checkQuickCreateCliVersion("v0.2.20").state).toBe("too_old");
    expect(checkQuickCreateCliVersion("v0.2.15").state).toBe("too_old");
  });

  it("returns missing for empty or unparsable input", () => {
    expect(checkQuickCreateCliVersion("").state).toBe("missing");
    expect(checkQuickCreateCliVersion(undefined).state).toBe("missing");
    expect(checkQuickCreateCliVersion("not-a-version").state).toBe("missing");
  });

  it("treats git-describe dev builds as ok regardless of base tag", () => {
    expect(checkQuickCreateCliVersion("v0.2.15-235-gdaf0e935").state).toBe("ok");
    expect(checkQuickCreateCliVersion("v0.2.15-235-gdaf0e935-dirty").state).toBe("ok");
    expect(checkQuickCreateCliVersion("0.1.0-1-gabc1234").state).toBe("ok");
  });
});

describe("isNewerCliRelease", () => {
  it("compares stable published versions", () => {
    expect(isNewerCliRelease("v0.4.0", "v0.3.17")).toBe(true);
    expect(isNewerCliRelease("v0.3.17", "0.3.17")).toBe(false);
    expect(isNewerCliRelease("v0.3.16", "v0.3.17")).toBe(false);
  });

  it("does not offer release updates for source builds", () => {
    expect(isNewerCliRelease("v99.0.0", "dev-2599e33")).toBe(false);
    expect(
      isNewerCliRelease("v99.0.0", "v0.2.15-235-gdaf0e935-dirty"),
    ).toBe(false);
  });

  it("fails closed for malformed and prerelease values", () => {
    expect(isNewerCliRelease("latest", "v0.3.17")).toBe(false);
    expect(isNewerCliRelease("v0.4.0", "unknown")).toBe(false);
    expect(isNewerCliRelease("v0.4.0-beta.1", "v0.3.17")).toBe(false);
  });
});
