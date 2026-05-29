export interface FontOverrideRecord {
  display?: string;
  body?: string;
  mono?: string;
}

/**
 * Applies custom font override variables onto the document root.
 */
export function applyFontOverrides(record: FontOverrideRecord): void {
  if (typeof document === 'undefined') return;

  const root = document.documentElement;

  if (record.display) {
    root.style.setProperty('--font-display', record.display);
  }
  if (record.body) {
    root.style.setProperty('--font-body', record.body);
  }
  if (record.mono) {
    root.style.setProperty('--font-mono', record.mono);
  }
}
