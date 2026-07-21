import { type ZodType } from "zod";

export interface ParseOptions {
  endpoint: string;
}

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
 * Mobile API contract boundary. The legacy fallback argument is ignored so a
 * malformed server response can never masquerade as an empty successful
 * result. React Query receives the error and keeps error/empty states distinct.
 */
export function parseWithFallback<T>(
  data: unknown,
  schema: ZodType,
  _legacyFallback: T,
  opts: ParseOptions,
): T {
  const result = schema.safeParse(data);
  if (result.success) return result.data as T;
  console.warn(`[api] schema validation failed: ${opts.endpoint}`, {
    issues: result.error.issues,
  });
  throw new ApiContractError(opts.endpoint, result.error.issues);
}
