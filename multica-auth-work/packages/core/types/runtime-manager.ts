// SPE-6 C4: Runtime Manager client DTOs.
//
// These types mirror the frozen C3 wire contract exactly. C3 is authoritative:
// `internal/handler/runtime_configuration.go` and `runtime_standard.go` own the
// JSON tags, and `internal/service/runtimeconfig/types.go` owns the field
// vocabulary. Nothing here may rename, widen or reinterpret a field.
//
// Two properties of the C3 decoder make exactness mandatory rather than
// stylistic. Request bodies are decoded with unknown fields rejected and no
// trailing content allowed, so a single extra or misspelled key is a 400 rather
// than a silently ignored value. Responses are re-encoded from the typed
// contract instead of echoed from storage, so a field absent here is a field
// the UI can never receive.

export type RuntimeTransportBinding =
  | "omniroute"
  | "native_credential_home";

export type RuntimeApplyClass = "hot" | "restart";

/**
 * Frozen effective-configuration digest: raw lowercase SHA-256 hex, exactly
 * 64 characters, with no algorithm prefix or separator. C3 emits `version`,
 * `configuration_digest` and `capability_digest` as external names and defines
 * no aliased or prefixed spelling of any of them.
 */
declare const runtimeDigestBrand: unique symbol;
export type RuntimeDigest = string & { readonly [runtimeDigestBrand]: true };

export function isRuntimeDigest(value: unknown): value is RuntimeDigest {
  return typeof value === "string" && /^[0-9a-f]{64}$/.test(value);
}

/**
 * The frozen configuration field vocabulary, in C3 registry order. Delegability
 * keys are drawn from this set, so typing them as arbitrary strings would let a
 * caller declare policy on a field the resolver does not know.
 */
export type RuntimeConfigurationField =
  | "transport_binding"
  | "cli_kind"
  | "provider"
  | "subscription_ref"
  | "provider_catalog_version"
  | "capability_digest"
  | "model"
  | "reasoning_mode"
  | "reasoning_effort"
  | "reasoning_budget"
  | "max_context_tokens"
  | "max_input_tokens"
  | "max_output_tokens"
  | "max_total_tokens"
  | "max_tool_calls"
  | "wall_timeout_ms"
  | "idle_timeout_ms"
  | "concurrency"
  | "retry"
  | "flags"
  | "env"
  | "skills"
  | "mcp_tools"
  | "permissions"
  | "eligibility"
  | "health"
  | "fallback";

// ---------------------------------------------------------------------------
// Bounds enforced by C3. Mirrored so the UI can fail before a round trip.
// ---------------------------------------------------------------------------

/** Default page size when `limit` is omitted. */
export const RUNTIME_MANAGER_DEFAULT_PAGE_SIZE = 50;
/** C3 clamps any larger `limit` down to this value. */
export const RUNTIME_MANAGER_MAX_PAGE_SIZE = 200;
/** A longer cursor is rejected as invalid. */
export const RUNTIME_MANAGER_MAX_CURSOR_LENGTH = 200;
/** A longer Idempotency-Key is rejected. */
export const RUNTIME_MANAGER_MAX_IDEMPOTENCY_KEY_LENGTH = 200;
/** Maximum request body C3 will read. */
export const RUNTIME_MANAGER_MAX_BODY_BYTES = 64 * 1024;
/** Maximum length of the audit reason on every mutation. */
export const RUNTIME_MANAGER_MAX_REASON_LENGTH = 128;
/**
 * C3 requires a symbolic reason code, not prose: lowercase letters, digits,
 * underscore, hyphen and dot only. Any space or uppercase character is a 400.
 */
export const RUNTIME_MANAGER_REASON_PATTERN = /^[a-z0-9_\-.]+$/;

export function isRuntimeReasonCode(value: unknown): value is string {
  return (
    typeof value === "string" &&
    value.length > 0 &&
    value.length <= RUNTIME_MANAGER_MAX_REASON_LENGTH &&
    RUNTIME_MANAGER_REASON_PATTERN.test(value)
  );
}

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

/**
 * C3 emits `next_cursor` on every page without `omitempty`, so it is always
 * present and null on the last page rather than absent.
 */
export interface RuntimeManagerPage<T> {
  items: T[];
  next_cursor: string | null;
}

// ---------------------------------------------------------------------------
// Configuration document
// ---------------------------------------------------------------------------

/** Optional execution ceilings. An omitted limit inherits; an explicit zero is rejected. */
export interface RuntimeConfigurationLimits {
  max_context_tokens?: number;
  max_input_tokens?: number;
  max_output_tokens?: number;
  max_total_tokens?: number;
  max_tool_calls?: number;
  wall_timeout_ms?: number;
  idle_timeout_ms?: number;
}

export interface RuntimeConcurrencyPolicy {
  max_sessions?: number;
  max_tasks?: number;
  max_subagents?: number;
  max_queue?: number;
}

export interface RuntimeRetryPolicy {
  max_attempts?: number;
  backoff_ms?: number;
  max_backoff_ms?: number;
  jitter_percent?: number;
  deadline_budget_ms?: number;
  retryable_classes?: string[];
}

/** CLI flag order is significant. Credential- and path-bearing flags are rejected. */
export interface RuntimeFlagPolicy {
  ordered: string[];
}

/** Exactly one of `value` and `reference` may be present. */
export interface RuntimeEnvironmentEntry {
  key: string;
  value?: string;
  reference?: string;
}

export interface RuntimeEnvironmentPolicy {
  entries: RuntimeEnvironmentEntry[];
}

/** Secret-free immutable artifact reference. */
export interface RuntimeVersionedRef {
  id: string;
  version: string;
  digest: string;
}

export interface RuntimeSkillPolicy {
  allowed: RuntimeVersionedRef[];
}

export interface RuntimeMCPToolRef {
  server_id: string;
  tool_id: string;
  scope: string;
  version: string;
  timeout_ms?: number;
}

export interface RuntimeMCPToolPolicy {
  allowed: RuntimeMCPToolRef[];
}

/** Symbolic policy IDs only. Raw filesystem paths and inline endpoints are not representable. */
export interface RuntimePermissionPolicy {
  filesystem_policies?: string[];
  network_policies?: string[];
  process_grants?: string[];
  tool_grants?: string[];
}

export interface RuntimeEligibilityPolicy {
  providers?: string[];
  runtime_kinds?: string[];
  daemon_classes?: string[];
  workspace_classes?: string[];
  required_capabilities?: string[];
}

export interface RuntimeHealthPolicy {
  freshness_ttl_ms?: number;
  readiness_percent?: number;
  circuit_state?: string;
  probe_class?: string;
}

export interface RuntimeFallbackRoute {
  route_id: string;
  failure_classes?: string[];
  max_attempts?: number;
}

export interface RuntimeFallbackPolicy {
  routes: RuntimeFallbackRoute[];
}

/**
 * The complete frozen pathless value set. It carries no credential, cookie,
 * endpoint, prompt, raw filesystem path or arbitrary map, which is what makes a
 * response structurally incapable of leaking one.
 */
export interface RuntimeConfigurationValues {
  transport_binding?: RuntimeTransportBinding;
  cli_kind?: string;
  provider?: string;
  subscription_ref?: string;
  provider_catalog_version?: string;
  capability_digest?: string;
  model?: string;
  reasoning_mode?: string;
  reasoning_effort?: string;
  reasoning_budget?: number;
  limits?: RuntimeConfigurationLimits;
  concurrency?: RuntimeConcurrencyPolicy;
  retry?: RuntimeRetryPolicy;
  flags?: RuntimeFlagPolicy;
  env?: RuntimeEnvironmentPolicy;
  skills?: RuntimeSkillPolicy;
  mcp_tools?: RuntimeMCPToolPolicy;
  permissions?: RuntimePermissionPolicy;
  eligibility?: RuntimeEligibilityPolicy;
  health?: RuntimeHealthPolicy;
  fallback?: RuntimeFallbackPolicy;
}

/**
 * Secret-free, pathless configuration document accepted and returned by the
 * frozen API.
 *
 * The contract version travels as `version`. C3 defines no `schema_version`
 * alias, and because request decoding rejects unknown fields, sending
 * `schema_version` is a 400 rather than a tolerated synonym.
 *
 * Delegability is tri-state per field: an absent key inherits lower policy,
 * true explicitly allows a task or subagent value, and false explicitly denies
 * it.
 */
export interface RuntimeConfigurationDocument {
  version: "v1";
  values: RuntimeConfigurationValues;
  delegability?: Partial<Record<RuntimeConfigurationField, boolean>>;
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

/**
 * Stable validation issue classifications. The first fourteen come from the
 * typed resolver; the last three are minted at the handler boundary.
 */
export type RuntimeConfigurationIssueCode =
  | "invalid_version"
  | "required_field"
  | "invalid_field"
  | "unknown_field"
  | "policy_not_allowed"
  | "task_field_not_delegable"
  | "invalid_capabilities"
  | "provider_mismatch"
  | "unsupported_model"
  | "unsupported_reasoning"
  | "limit_exceeded"
  | "context_window_exceeded"
  | "invalid_parent_digest"
  | "no_rollback_target"
  | "capabilities_unavailable"
  | "platform_layer_unavailable"
  | "invalid_configuration";

/**
 * An issue carries a stable code and the field it concerns, never the rejected
 * value, so it is safe to log and to render. `retryable` is emitted without
 * `omitempty` and is therefore always present.
 */
export interface RuntimeConfigurationIssue {
  code: RuntimeConfigurationIssueCode | string;
  field?: string;
  retryable: boolean;
}

/**
 * Only a failure to read a dependency is retryable. Every resolver verdict is
 * deterministic, so resending an identical document yields an identical
 * rejection.
 */
export const RUNTIME_RETRYABLE_ISSUE_CODES: readonly RuntimeConfigurationIssueCode[] = [
  "capabilities_unavailable",
  "platform_layer_unavailable",
];

/**
 * `capability_digest` is typed as a plain string rather than a RuntimeDigest:
 * when a document is rejected before the capability declaration is read, C3
 * reports an empty digest, which is not 64-hex. `apply_class` is present only
 * when `valid` is true. `issues` is always an array, empty rather than null.
 */
export interface RuntimeConfigurationValidation {
  valid: boolean;
  capability_digest: string;
  apply_class?: RuntimeApplyClass;
  issues: RuntimeConfigurationIssue[];
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

/**
 * One immutable stored version. C3 renders a version only after reproducing its
 * canonical document and confirming the stored digest matches those bytes, so
 * `configuration_digest` is always a valid 64-hex digest here.
 */
export interface RuntimeConfigurationVersion {
  id: string;
  version_number: number;
  configuration: RuntimeConfigurationDocument;
  configuration_digest: RuntimeDigest;
  reason: string;
  state?: string;
  apply_class?: RuntimeApplyClass;
  validation?: RuntimeConfigurationValidation;
  created_at: string;
}

// ---------------------------------------------------------------------------
// Standards
// ---------------------------------------------------------------------------

export interface RuntimeStandardSummary {
  id: string;
  name: string;
  description: string | null;
  active_version_id: string | null;
  active_version_number?: number | null;
  updated_at: string;
}

/**
 * `versions` is a page object, not a bare array: C3 embeds
 * `runtimeManagerPage[...]` under the `versions` key so standard detail is
 * pageable through the same cursor mechanism as every list endpoint.
 */
export interface RuntimeStandardDetail extends RuntimeStandardSummary {
  active_version: RuntimeConfigurationVersion | null;
  versions: RuntimeManagerPage<RuntimeConfigurationVersion>;
}

export interface CreateRuntimeStandardRequest {
  name: string;
  description?: string;
}

// ---------------------------------------------------------------------------
// Activation, rollback and compare-and-swap
// ---------------------------------------------------------------------------

/**
 * `expected_active_version_id` is required on every activation and rollback,
 * including the first one, where explicit null is the assertion that no active
 * version exists. Omitting the key is a 400; it is not equivalent to null.
 *
 * `reason` must satisfy RUNTIME_MANAGER_REASON_PATTERN.
 */
export interface RuntimeActivationRequest {
  expected_active_version_id: string | null;
  reason: string;
}

/**
 * Rollback targets any prior immutable version.
 *
 * `expected_binding_generation` is optional because one request type serves two
 * scopes with different fences. Binding rollback requires a positive generation
 * and rejects a request without one; standard rollback has no generation and,
 * because C3 rejects unknown fields, rejects a request that carries one. Supply
 * it for binding scope and omit it for standard scope.
 */
export interface RuntimeRollbackRequest extends RuntimeActivationRequest {
  target_version_id: string;
  expected_binding_generation?: number;
}

/** Binding activation is fenced on both the active version and the generation. */
export interface RuntimeBindingActivationRequest extends RuntimeActivationRequest {
  expected_binding_generation: number;
}

/**
 * `previous_version_id` is emitted without `omitempty` and is therefore always
 * present, null when there was no prior active version. A transition that moves
 * no value reports apply class `hot` with `pending_acknowledgement` false.
 */
export interface RuntimeActivationResult {
  active_version_id: string;
  previous_version_id: string | null;
  apply_class: RuntimeApplyClass;
  pending_acknowledgement: boolean;
  capability_digest: string;
}

// ---------------------------------------------------------------------------
// Canonical error envelope
// ---------------------------------------------------------------------------

/**
 * The status-to-code vocabulary C3 emits. `generation_conflict` and
 * `catalog_unavailable` are additionally minted when a store reports a failed
 * compare-and-swap or an unreadable capability catalog.
 */
export type RuntimeManagerErrorCode =
  | "invalid_argument"
  | "unauthenticated"
  | "forbidden"
  | "not_found"
  | "version_conflict"
  | "generation_conflict"
  | "capability_unsupported"
  | "invalid_configuration"
  | "capacity_exhausted"
  | "catalog_unavailable"
  | "internal_error";

/**
 * `request_id` and `retryable` are emitted without `omitempty` and are always
 * present. `message` is a fixed operator-facing string chosen by the server and
 * never carries backend error detail, a path, an account or a credential.
 */
export interface RuntimeManagerErrorBody {
  code: RuntimeManagerErrorCode | string;
  message: string;
  field?: string;
  request_id: string;
  retryable: boolean;
}

/** Every C3 error response is this single nested envelope. */
export interface RuntimeManagerErrorEnvelope {
  error: RuntimeManagerErrorBody;
}

export function isRuntimeManagerErrorEnvelope(
  value: unknown,
): value is RuntimeManagerErrorEnvelope {
  if (typeof value !== "object" || value === null) return false;
  const error = (value as { error?: unknown }).error;
  if (typeof error !== "object" || error === null) return false;
  const body = error as Partial<RuntimeManagerErrorBody>;
  return typeof body.code === "string" && typeof body.message === "string";
}

// ---------------------------------------------------------------------------
// Projections without a frozen C3 endpoint
// ---------------------------------------------------------------------------
//
// Everything below is consumed by the C4 UI but has NO route in the frozen C3
// surface. C3 serves exactly two scopes: `/api/runtime-standards...` and
// `/api/workspaces/{workspaceId}/runtime-bindings/{bindingId}/configuration-versions...`
// plus that binding's `/rollback`.
//
// It serves no session list or create, no session enrollment, no binding list,
// no binding detail, no credential-home list and no home assignment. These
// shapes therefore have no server-side authority yet: their field names are the
// C4 proposal, not a verified contract, and they must be reconciled against the
// owning lane before any endpoint is treated as available.
//
// Reported as the primary Lane D blocking gap rather than silently assumed.

export type RuntimeEnrollmentState = "active" | "draining" | "inactive";

export type RuntimeHomeState =
  | "candidate"
  | "healthy"
  | "degraded"
  | "quarantined"
  | "missing"
  | "draining"
  | "retired"
  | "unassigned"
  | "assigned";

/** UNSERVED by C3. Requires a session endpoint owner. */
export interface RuntimeSessionSummary {
  id: string;
  standard_id: string;
  name: string;
  provider: string;
  runtime_kind: string;
  enrollments?: RuntimeSessionEnrollment[];
  created_at: string;
  updated_at: string;
}

/** UNSERVED by C3. Requires a session enrollment endpoint owner. */
export interface RuntimeSessionEnrollment {
  id: string;
  session_id: string;
  workspace_id: string;
  runtime_id: string;
  agent_id: string;
  state: RuntimeEnrollmentState;
  created_at: string;
  updated_at: string;
}

/** UNSERVED by C3. Requires a session create endpoint owner. */
export interface CreateRuntimeSessionRequest {
  name: string;
  provider: string;
  runtime_kind: string;
  standard_id: string;
}

/** UNSERVED by C3. Capability projections are internal to the resolver today. */
export interface RuntimeCapabilityModel {
  id?: string;
  model_id?: string;
  display_name?: string;
  reasoning_efforts?: string[];
  max_input_tokens?: number;
  max_output_tokens?: number;
  context_window_tokens?: number;
  max_tool_calls?: number;
}

/** UNSERVED by C3. */
export interface RuntimeCapabilityProjection {
  version?: string;
  generation?: number;
  provider?: string;
  capability_digest?: string;
  models: RuntimeCapabilityModel[] | Record<string, RuntimeCapabilityModel>;
}

/**
 * UNSERVED by C3. Lane B additionally reserves a separate canonical
 * `name_ref` (`name_<43>`) alongside the UUID `home_ref`; this projection has
 * no field for it yet, which is an open reconciliation item.
 */
export interface RuntimeHomeProjection {
  home_ref: string;
  generation: number;
  provider: string;
  state: RuntimeHomeState;
  health: string;
  first_seen_at?: string;
  last_seen_at?: string;
}

/** UNSERVED by C3. */
export interface RuntimeBindingSummary {
  id: string;
  workspace_id: string;
  runtime_id: string;
  agent_id: string;
  session_id: string;
  standard_id: string;
  provider: string;
  transport_binding: RuntimeTransportBinding;
  binding_generation: number;
  catalog_generation?: number | null;
  home_ref?: string | null;
  home_state?: RuntimeHomeState | null;
  active_configuration_version_id: string | null;
  applied_configuration_version_id?: string | null;
  state: string;
}

/** UNSERVED by C3. */
export interface RuntimeBindingDetail extends RuntimeBindingSummary {
  session?: RuntimeSessionSummary;
  active_configuration: RuntimeConfigurationDocument | null;
  versions: RuntimeConfigurationVersion[];
  capabilities: RuntimeCapabilityProjection | null;
}
