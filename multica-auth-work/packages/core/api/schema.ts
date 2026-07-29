import type { ZodType } from "zod";
import { type Logger, noopLogger } from "../logger";

// Module-level logger for schema warnings. Defaults to no-op so test
// runs don't spam stderr; the platform layer wires a real logger via
// `setSchemaLogger` at app boot.
let schemaLogger: Logger = noopLogger;

export function setSchemaLogger(logger: Logger): void {
  schemaLogger = logger;
}

export interface ParseOptions {
  /** Endpoint identifier used in the warning log so we can grep for which
   *  contract drifted in production telemetry. */
  endpoint: string;
}

/** Raised when a production API response violates its declared contract. */
export class ApiContractError extends Error {
  readonly endpoint: string;
  readonly issues: unknown;

  constructor(endpoint: string, issues: unknown) {
    super(`API response failed schema validation: ${endpoint}`);
    this.name = "ApiContractError";
    this.endpoint = endpoint;
    this.issues = issues;
  }
}

/**
 * Validate a parsed API response and fail closed on contract drift.
 *
 * The third parameter remains temporarily for source compatibility with
 * existing callers, but is deliberately ignored: malformed production data
 * must never be replaced with an empty or successful-looking synthetic record.
 * Callers receive a typed error so their normal query/mutation error path can
 * render an honest failure and roll back optimistic state.
 */
export function parseWithFallback<T>(
  data: unknown,
  schema: ZodType,
  _legacyFallback: T,
  opts: ParseOptions,
): T {
  const result = schema.safeParse(data);
  if (result.success) return result.data as T;
  schemaLogger.warn(
    `API response failed schema validation: ${opts.endpoint}`,
    {
      endpoint: opts.endpoint,
      issues: result.error.issues,
    },
  );
  throw new ApiContractError(opts.endpoint, result.error.issues);
}
