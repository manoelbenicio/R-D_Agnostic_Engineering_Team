/**
 * Indra-aligned xterm theme.
 *
 * xterm.js does NOT resolve CSS custom properties at render time — it expects
 * raw color strings. This module exposes:
 *
 *   - SENTINEL_TERMINAL_THEME: a static fallback theme (for SSR / pre-mount /
 *     test scaffolds). Values are the Indra hex literals from the DSS canon.
 *
 *   - resolveTerminalTheme(override?): the recommended runtime entry point.
 *     Reads `getComputedStyle(document.documentElement)` so the theme always
 *     mirrors whatever `tokens.css` currently exposes — including any user
 *     font/color overrides applied via `applyFontOverrides()` or future
 *     theme-switcher work.
 *
 * Contract proven by `src/terminal/__tests__/xterm-theme-indra-parity.test.ts`.
 */
import type { ITheme } from '@xterm/xterm';

/* ----- Indra hex (DSS Universal Standard v3.0) ----- */
const INDRA = {
  deep: '#002B3A',
  dark: '#003E50',
  primary: '#06596E',
  cyan: '#00B0BD',
  teal: '#3F96AE',
  light: '#7A9CAE',
  blueGray: '#B3C1DA',
  warmGray: '#B0B4BD',
  white: '#FFFFFF',
  success: '#27AE60',
  warning: '#FF9800',
  error: '#E91E63',
  gold: '#FFC107',
} as const;

/**
 * Static fallback theme — used when document is not available (SSR, tests
 * that don't mount tokens.css). All values match Indra hex literals; no
 * pre-Indra SENTINEL relics remain.
 */
export const SENTINEL_TERMINAL_THEME: ITheme = {
  background: INDRA.deep,
  foreground: INDRA.white,
  cursor: INDRA.cyan,
  cursorAccent: INDRA.deep,
  selectionBackground: 'rgba(0, 176, 189, 0.25)', // Indra cyan @ 25%
  selectionForeground: INDRA.white,

  /* ANSI 16 — Indra-mapped */
  black: INDRA.deep,
  red: INDRA.error,
  green: INDRA.success,
  yellow: INDRA.gold,
  blue: INDRA.primary,
  magenta: INDRA.error, // Indra has no dedicated magenta; reuse error per DSS spec.
  cyan: INDRA.cyan,
  white: INDRA.blueGray,

  brightBlack: INDRA.warmGray,
  brightRed: INDRA.error,
  brightGreen: INDRA.success,
  brightYellow: INDRA.gold,
  brightBlue: INDRA.teal,
  brightMagenta: INDRA.error,
  brightCyan: INDRA.cyan,
  brightWhite: INDRA.white,
};

/**
 * Read a CSS custom property from `:root`. Returns `fallback` if the document
 * is not available (SSR / tests without DOM) or the property is empty.
 *
 * The value is normalized: surrounding whitespace stripped, common helpers
 * like `rgb(0,176,189)` left as-is for xterm to consume.
 */
function readVar(name: string, fallback: string): string {
  if (typeof document === 'undefined' || !document.documentElement) {
    return fallback;
  }
  const raw = getComputedStyle(document.documentElement).getPropertyValue(name);
  const trimmed = (raw ?? '').trim();
  if (!trimmed) return fallback;
  // If the property was declared as `var(--other)` and the resolver gave us
  // back the literal `var(...)` we can't resolve to hex — fall back. In normal
  // browsers `getPropertyValue` returns the resolved leaf value already.
  if (trimmed.startsWith('var(')) return fallback;
  return trimmed;
}

/**
 * Construct an xterm.js ITheme from the current CSS variable state, layered
 * over the optional caller-supplied override.
 */
export function resolveTerminalTheme(override: Partial<ITheme> = {}): ITheme {
  const base: ITheme = {
    background: readVar('--indra-deep', INDRA.deep),
    foreground: readVar('--indra-white', INDRA.white),
    cursor: readVar('--indra-cyan', INDRA.cyan),
    cursorAccent: readVar('--indra-deep', INDRA.deep),
    selectionBackground: 'rgba(0, 176, 189, 0.25)',
    selectionForeground: readVar('--indra-white', INDRA.white),

    black: readVar('--indra-deep', INDRA.deep),
    red: readVar('--indra-error', INDRA.error),
    green: readVar('--indra-success', INDRA.success),
    yellow: readVar('--indra-gold', INDRA.gold),
    blue: readVar('--indra-primary', INDRA.primary),
    magenta: readVar('--indra-error', INDRA.error),
    cyan: readVar('--indra-cyan', INDRA.cyan),
    white: readVar('--indra-blue-gray', INDRA.blueGray),

    brightBlack: readVar('--indra-warm-gray', INDRA.warmGray),
    brightRed: readVar('--indra-error', INDRA.error),
    brightGreen: readVar('--indra-success', INDRA.success),
    brightYellow: readVar('--indra-gold', INDRA.gold),
    brightBlue: readVar('--indra-teal', INDRA.teal),
    brightMagenta: readVar('--indra-error', INDRA.error),
    brightCyan: readVar('--indra-cyan', INDRA.cyan),
    brightWhite: readVar('--indra-white', INDRA.white),
  };

  return { ...base, ...override };
}

/**
 * Legacy alias kept for backwards compatibility with consumers that imported
 * `createTerminalTheme()`. New code should call `resolveTerminalTheme()` so
 * intent is explicit.
 */
export function createTerminalTheme(themeOverride?: Partial<ITheme>): ITheme {
  return resolveTerminalTheme(themeOverride ?? {});
}
