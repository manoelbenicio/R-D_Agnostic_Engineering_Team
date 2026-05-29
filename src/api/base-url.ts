const DEFAULT_CAO_BASE_URL = 'http://127.0.0.1:9889';

function normalizeBaseUrl(value: string): string {
  const trimmed = value.trim();
  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed;
}

export const CAO_BASE_URL = normalizeBaseUrl(
  import.meta.env.VITE_CAO_BASE_URL || DEFAULT_CAO_BASE_URL
);
