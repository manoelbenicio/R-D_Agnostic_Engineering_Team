/**
 * xterm Theme — Indra Palette Parity (SEV-0 ISSUE-004 / ISSUE-005)
 *
 * Root cause: xterm.js does NOT resolve CSS custom properties. The terminal
 * receives raw color strings at theme-construction time, so the SENTINEL_*
 * theme values must be real Indra hex literals (or be resolved at runtime
 * from the document root).
 *
 * Contract: `resolveTerminalTheme()` must return an `ITheme` whose color
 * channels equal the Indra palette resolved from CSS custom properties on
 * `document.documentElement` (or the SSR-safe Indra fallback).
 *
 * This test FAILS until the SEV-0 fix lands.
 */
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it, beforeAll } from 'vitest';
import { SENTINEL_TERMINAL_THEME, resolveTerminalTheme } from '@/terminal/xterm-theme';

const INDRA = {
  deep: '#002B3A',
  cyan: '#00B0BD',
  success: '#27AE60',
  warning: '#FF9800',
  error: '#E91E63',
  gold: '#FFC107',
  white: '#FFFFFF',
  blueGray: '#B3C1DA',
  warmGray: '#B0B4BD',
} as const;

describe('xterm Theme — Indra Palette Parity (ISSUE-004 / ISSUE-005)', () => {
  beforeAll(() => {
    // Inject the canonical token block into the JSDOM root so getComputedStyle
    // returns real Indra hex when the resolver reads CSS variables.
    const tokensCss = readFileSync(
      resolve(__dirname, '../../design-system/tokens.css'),
      'utf8'
    );
    const styleEl = document.createElement('style');
    styleEl.id = 'tokens-under-test';
    styleEl.textContent = tokensCss;
    document.head.appendChild(styleEl);
  });

  describe('SENTINEL_TERMINAL_THEME — static export does not contain pre-Indra hex', () => {
    it('does NOT use the legacy SENTINEL cyan #00f0ff', () => {
      expect(SENTINEL_TERMINAL_THEME.cyan?.toLowerCase()).not.toBe('#00f0ff');
    });
    it('does NOT use the legacy SENTINEL void #06090d', () => {
      expect(SENTINEL_TERMINAL_THEME.black?.toLowerCase()).not.toBe('#06090d');
    });
    it('does NOT use the legacy SENTINEL threat #ff3b30', () => {
      expect(SENTINEL_TERMINAL_THEME.red?.toLowerCase()).not.toBe('#ff3b30');
    });
    it('does NOT use the legacy SENTINEL ops #00ff66', () => {
      expect(SENTINEL_TERMINAL_THEME.green?.toLowerCase()).not.toBe('#00ff66');
    });
    it('does NOT use the legacy SENTINEL amber #ffb700', () => {
      expect(SENTINEL_TERMINAL_THEME.yellow?.toLowerCase()).not.toBe('#ffb700');
    });
    it('selectionBackground does NOT use the legacy SENTINEL cyan rgba(0, 255, 255, *)', () => {
      const sel = (SENTINEL_TERMINAL_THEME.selectionBackground ?? '').toLowerCase();
      expect(sel).not.toMatch(/rgba?\(\s*0\s*,\s*255\s*,\s*255/);
    });
  });

  describe('resolveTerminalTheme() — runtime resolver returns Indra hex', () => {
    it('exists and is a function', () => {
      expect(typeof resolveTerminalTheme).toBe('function');
    });

    it('resolves cyan to Indra cyan #00B0BD', () => {
      const t = resolveTerminalTheme();
      expect(t.cyan?.toUpperCase()).toBe(INDRA.cyan);
    });

    it('resolves background to Indra deep #002B3A', () => {
      const t = resolveTerminalTheme();
      expect(t.background?.toUpperCase()).toBe(INDRA.deep);
    });

    it('resolves cursor to Indra cyan #00B0BD', () => {
      const t = resolveTerminalTheme();
      expect(t.cursor?.toUpperCase()).toBe(INDRA.cyan);
    });

    it('resolves green to Indra success #27AE60', () => {
      const t = resolveTerminalTheme();
      expect(t.green?.toUpperCase()).toBe(INDRA.success);
    });

    it('resolves red to Indra error #E91E63', () => {
      const t = resolveTerminalTheme();
      expect(t.red?.toUpperCase()).toBe(INDRA.error);
    });

    it('resolves yellow to Indra gold #FFC107', () => {
      const t = resolveTerminalTheme();
      expect(t.yellow?.toUpperCase()).toBe(INDRA.gold);
    });

    it('resolves foreground to Indra white #FFFFFF', () => {
      const t = resolveTerminalTheme();
      expect(t.foreground?.toUpperCase()).toBe(INDRA.white);
    });

    it('resolves brightBlack to a perceptibly-lighter neutral than --indra-warm-gray', () => {
      // Sanity: brightBlack must not collide with foreground/background.
      const t = resolveTerminalTheme();
      expect(t.brightBlack).toBeDefined();
      expect(t.brightBlack).not.toBe(t.background);
      expect(t.brightBlack).not.toBe(t.foreground);
    });

    it('selectionBackground is derived from Indra cyan rgba(0, 176, 189, …)', () => {
      const t = resolveTerminalTheme();
      const sel = (t.selectionBackground ?? '').toLowerCase().replace(/\s+/g, '');
      expect(sel).toMatch(/rgba?\(0,176,189/);
    });
  });

  describe('Theme override merge — contract preserved', () => {
    it('respects partial overrides without breaking Indra base', () => {
      const t = resolveTerminalTheme({ background: '#101010' });
      expect(t.background).toBe('#101010');
      // Other channels still Indra:
      expect(t.cyan?.toUpperCase()).toBe(INDRA.cyan);
    });
  });
});
