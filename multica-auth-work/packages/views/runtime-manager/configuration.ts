import {
  isRuntimeDigest,
  isRuntimeReasonCode,
  RUNTIME_MANAGER_MAX_REASON_LENGTH,
  RUNTIME_MANAGER_REASON_PATTERN,
  type RuntimeCapabilityModel,
  type RuntimeCapabilityProjection,
  type RuntimeConfigurationDocument,
} from "@multica/core/types";

export interface RuntimeConfigurationDiff {
  field: string;
  before: string;
  after: string;
}

/**
 * Display allowlist over the frozen C3 value set.
 *
 * Every scalar C3 can return is listed, so an operator sees the whole effective
 * configuration rather than a subset. It stays an allowlist rather than an
 * iteration over the response because a stored document is untrusted input: an
 * unexpected key is never rendered, which is what keeps a path, account or
 * secret out of the UI even if one somehow reached storage.
 *
 * Composite policies (limits beyond the scalars below, concurrency, retry,
 * flags, env, skills, mcp_tools, permissions, eligibility, health, fallback)
 * are deliberately excluded from this flat projection: they are structured
 * objects and are surfaced separately rather than string-coerced here.
 */
const SAFE_FIELDS = [
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
] as const;

export type SafeRuntimeConfigurationField = (typeof SAFE_FIELDS)[number];

export function safeRuntimeDigest(value: unknown): string {
  return isRuntimeDigest(value) ? value : "invalid digest";
}

export function safeRuntimeConfiguration(
  configuration: RuntimeConfigurationDocument | null | undefined,
): Record<SafeRuntimeConfigurationField, string> {
  const values = configuration?.values;
  const limits = values?.limits;
  return {
    transport_binding: values?.transport_binding ?? "—",
    cli_kind: values?.cli_kind ?? "—",
    provider: values?.provider ?? "—",
    subscription_ref: values?.subscription_ref ?? "—",
    provider_catalog_version: values?.provider_catalog_version ?? "—",
    capability_digest: values?.capability_digest
      ? safeRuntimeDigest(values.capability_digest)
      : "—",
    model: values?.model ?? "—",
    reasoning_mode: values?.reasoning_mode ?? "—",
    reasoning_effort: values?.reasoning_effort ?? "runtime default",
    reasoning_budget: numberText(values?.reasoning_budget),
    max_context_tokens: numberText(limits?.max_context_tokens),
    max_input_tokens: numberText(limits?.max_input_tokens),
    max_output_tokens: numberText(limits?.max_output_tokens),
    max_total_tokens: numberText(limits?.max_total_tokens),
    max_tool_calls: numberText(limits?.max_tool_calls),
    wall_timeout_ms: numberText(limits?.wall_timeout_ms),
    idle_timeout_ms: numberText(limits?.idle_timeout_ms),
  };
}

export function diffRuntimeConfigurations(
  before: RuntimeConfigurationDocument | null | undefined,
  after: RuntimeConfigurationDocument | null | undefined,
): RuntimeConfigurationDiff[] {
  const left = safeRuntimeConfiguration(before);
  const right = safeRuntimeConfiguration(after);
  return SAFE_FIELDS.flatMap((field) =>
    left[field] === right[field]
      ? []
      : [{ field, before: left[field], after: right[field] }],
  );
}

export function capabilityModels(
  projection: RuntimeCapabilityProjection | null | undefined,
): Array<RuntimeCapabilityModel & { model_id: string }> {
  if (!projection) return [];
  if (Array.isArray(projection.models)) {
    return projection.models.flatMap((model) => {
      const id = model.model_id ?? model.id;
      return id ? [{ ...model, model_id: id }] : [];
    });
  }
  return Object.entries(projection.models).map(([modelId, model]) => ({
    ...model,
    model_id: model.model_id ?? model.id ?? modelId,
  }));
}

export function withModelAndReasoning(
  base: RuntimeConfigurationDocument,
  model: string,
  reasoningEffort: string,
): RuntimeConfigurationDocument {
  const values = { ...base.values };
  delete values.reasoning_effort;
  return {
    ...base,
    values: {
      ...values,
      model,
      ...(reasoningEffort ? { reasoning_effort: reasoningEffort } : {}),
      ...(base.values.limits ? { limits: { ...base.values.limits } } : {}),
    },
    ...(base.delegability
      ? { delegability: { ...base.delegability } }
      : {}),
  };
}

/**
 * C3 requires the audit reason to be a symbolic code, so prose is rejected with
 * a 400 rather than stored. Validating here lets the UI refuse before the round
 * trip and explain why, instead of surfacing an opaque invalid-argument error.
 */
export function describeReasonCodeIssue(value: string): string | null {
  const reason = value.trim();
  if (reason.length === 0) return "A reason code is required.";
  if (reason.length > RUNTIME_MANAGER_MAX_REASON_LENGTH) {
    return `A reason code is at most ${RUNTIME_MANAGER_MAX_REASON_LENGTH} characters.`;
  }
  if (!RUNTIME_MANAGER_REASON_PATTERN.test(reason)) {
    return "A reason code uses lowercase letters, digits, underscore, hyphen and dot only.";
  }
  return null;
}

export function isValidReasonCode(value: string): boolean {
  return isRuntimeReasonCode(value.trim());
}

/**
 * Suggests a compliant code for prose the operator already typed. It is offered
 * as a suggestion the operator must accept rather than applied silently: an
 * audit reason is the operator's own words and must not be rewritten without
 * them seeing the result.
 */
export function suggestReasonCode(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_\-.]+/g, "_")
    .replace(/_{2,}/g, "_")
    .replace(/^[_\-.]+|[_\-.]+$/g, "")
    .slice(0, RUNTIME_MANAGER_MAX_REASON_LENGTH);
}

function numberText(value: number | undefined): string {
  return value === undefined ? "—" : String(value);
}
