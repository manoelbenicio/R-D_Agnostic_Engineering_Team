// GO Core Server base URL.
// CRIT-003.2: prod-readiness-critical-fixes

const DEFAULT_GO_CORE_BASE_URL = 'http://127.0.0.1:8080';

function normalizeBaseUrl(value: string): string {
  const trimmed = value.trim();
  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed;
}

export const GO_CORE_BASE_URL = normalizeBaseUrl(
  import.meta.env.VITE_GO_CORE_BASE_URL || DEFAULT_GO_CORE_BASE_URL
);
