import { describe, expect, it } from "vitest";
import {
  capabilityModels,
  describeReasonCodeIssue,
  diffRuntimeConfigurations,
  isValidReasonCode,
  safeRuntimeConfiguration,
  safeRuntimeDigest,
  suggestReasonCode,
  withModelAndReasoning,
} from "./configuration";

// The contract version travels as `version`. C3 defines no `schema_version`
// alias and rejects unknown request fields, so the wrong key is a 400.
const unsafeConfiguration = {
  version: "v1" as const,
  values: {
    transport_binding: "native_credential_home" as const,
    cli_kind: "claude",
    provider: "anthropic",
    model: "claude-sonnet",
    reasoning_effort: "medium",
    limits: { max_input_tokens: 1000 },
    path: "/home/example/.credentials",
    account_email: "person@example.test",
    secret_value: "do-not-render",
  },
  delegability: {},
};

describe("Runtime Manager safe configuration projection", () => {
  it("accepts only the frozen raw lowercase 64-hex digest", () => {
    const raw64 = "0123456789abcdef".repeat(4);
    expect(safeRuntimeDigest(raw64)).toBe(raw64);
    expect(safeRuntimeDigest(`sha256:${raw64}`)).toBe("invalid digest");
    expect(safeRuntimeDigest(raw64.toUpperCase())).toBe("invalid digest");
    expect(safeRuntimeDigest(raw64.slice(1))).toBe("invalid digest");
    expect(safeRuntimeDigest(`${raw64}0`)).toBe("invalid digest");
    expect(safeRuntimeDigest("g".repeat(64))).toBe("invalid digest");
  });

  it("allowlists display fields and never renders arbitrary path/account/secret data", () => {
    const rendered = JSON.stringify(safeRuntimeConfiguration(unsafeConfiguration));
    expect(rendered).toContain("claude-sonnet");
    expect(rendered).not.toContain("/home/example");
    expect(rendered).not.toContain("person@example.test");
    expect(rendered).not.toContain("do-not-render");
    expect(rendered).not.toContain("path");
    expect(rendered).not.toContain("account_email");
  });

  it("projects every frozen scalar field, defaulting absent ones", () => {
    const projected = safeRuntimeConfiguration(unsafeConfiguration);
    expect(Object.keys(projected)).toEqual([
      "transport_binding",
      "cli_kind",
      "provider",
      "subscription_ref",
      "provider_catalog_version",
      "capability_digest",
      "model",
      "reasoning_mode",
      "reasoning_effort",
      "reasoning_budget",
      "max_context_tokens",
      "max_input_tokens",
      "max_output_tokens",
      "max_total_tokens",
      "max_tool_calls",
      "wall_timeout_ms",
      "idle_timeout_ms",
    ]);
    expect(projected.max_input_tokens).toBe("1000");
    expect(projected.max_total_tokens).toBe("—");
    expect(projected.subscription_ref).toBe("—");
    expect(projected.reasoning_effort).toBe("medium");
  });

  it("rejects a non-conforming capability digest inside the document values", () => {
    const raw64 = "0123456789abcdef".repeat(4);
    expect(
      safeRuntimeConfiguration({
        version: "v1",
        values: { capability_digest: `sha256:${raw64}` },
      }).capability_digest,
    ).toBe("invalid digest");
    expect(
      safeRuntimeConfiguration({
        version: "v1",
        values: { capability_digest: raw64 },
      }).capability_digest,
    ).toBe(raw64);
  });

  it("diffs model and reasoning changes without diffing unknown fields", () => {
    const changed = withModelAndReasoning(unsafeConfiguration, "claude-opus", "high");
    const diff = diffRuntimeConfigurations(unsafeConfiguration, changed);
    expect(diff).toEqual([
      { field: "model", before: "claude-sonnet", after: "claude-opus" },
      { field: "reasoning_effort", before: "medium", after: "high" },
    ]);
    expect(JSON.stringify(diff)).not.toMatch(/path|account|secret|example\.test/);
  });

  it("preserves the version key when deriving a candidate document", () => {
    const changed = withModelAndReasoning(unsafeConfiguration, "claude-opus", "high");
    expect(changed.version).toBe("v1");
    expect(changed).not.toHaveProperty("schema_version");
  });

  it("normalizes capability maps and arrays to opaque model IDs", () => {
    expect(
      capabilityModels({
        generation: 3,
        models: {
          opus: { reasoning_efforts: ["medium", "high"] },
        },
      }),
    ).toEqual([{ model_id: "opus", reasoning_efforts: ["medium", "high"] }]);

    expect(
      capabilityModels({
        generation: 4,
        models: [{ id: "sonnet", reasoning_efforts: ["low"] }],
      }),
    ).toEqual([{ id: "sonnet", model_id: "sonnet", reasoning_efforts: ["low"] }]);
  });
});

describe("Runtime Manager symbolic reason codes", () => {
  it("accepts only the symbolic charset C3 enforces", () => {
    expect(isValidReasonCode("use_approved_model")).toBe(true);
    expect(isValidReasonCode("rollback-2.1")).toBe(true);
    expect(isValidReasonCode("Use approved model")).toBe(false);
    expect(isValidReasonCode("use approved model")).toBe(false);
    expect(isValidReasonCode("USE_APPROVED_MODEL")).toBe(false);
    expect(isValidReasonCode("")).toBe(false);
    expect(isValidReasonCode("a".repeat(129))).toBe(false);
    expect(isValidReasonCode("a".repeat(128))).toBe(true);
  });

  it("explains why prose is rejected instead of surfacing an opaque 400", () => {
    expect(describeReasonCodeIssue("")).toBe("A reason code is required.");
    expect(describeReasonCodeIssue("   ")).toBe("A reason code is required.");
    expect(describeReasonCodeIssue("Use approved model")).toMatch(/lowercase letters/);
    expect(describeReasonCodeIssue("a".repeat(129))).toMatch(/at most 128/);
    expect(describeReasonCodeIssue("use_approved_model")).toBeNull();
  });

  it("suggests a compliant code without silently rewriting operator intent", () => {
    expect(suggestReasonCode("Use approved model")).toBe("use_approved_model");
    expect(suggestReasonCode("  Rollback: bad activation!  ")).toBe("rollback_bad_activation");
    expect(isValidReasonCode(suggestReasonCode("Pin Opus 4.8 for QA"))).toBe(true);
  });
});
