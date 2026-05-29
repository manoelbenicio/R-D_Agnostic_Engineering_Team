/**
 * DSS Universal Standard v3.0 (Indra) — Token Parity Test
 *
 * SEV-0 ISSUES covered: ISSUE-001, ISSUE-008, ISSUE-009, ISSUE-011, ISSUE-012
 *
 * Source of truth: `src/design-system/frontend/styles.css` (the DSS canon).
 * System under test: `src/design-system/tokens.css`.
 *
 * Contract: every CSS custom property defined in DSS MUST be present in tokens.css
 * with an equivalent resolved value. tokens.css MAY define additional tokens
 * (AgentVerse extensions). It MUST NOT contradict DSS.
 *
 * This test FAILS until the SEV-0 fix lands.
 */
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const REPO = resolve(__dirname, '../../..');
const DSS_PATH = resolve(REPO, 'src/design-system/frontend/styles.css');
const TOKENS_PATH = resolve(REPO, 'src/design-system/tokens.css');

/**
 * Parse `:root { --name: value; ... }` blocks. Returns a Map<name, value>.
 * Whitespace inside `value` is collapsed for stable equality.
 */
function parseRootTokens(css: string): Map<string, string> {
  const tokens = new Map<string, string>();
  // Capture each `:root { ... }` block (DSS has one; tokens.css has one or more).
  const rootBlockRe = /:root\s*\{([^}]*)\}/g;
  let block: RegExpExecArray | null;
  while ((block = rootBlockRe.exec(css)) !== null) {
    const body = block[1] ?? '';
    const declRe = /(--[a-z0-9-]+)\s*:\s*([^;]+);/gi;
    let m: RegExpExecArray | null;
    while ((m = declRe.exec(body)) !== null) {
      const name = (m[1] ?? '').trim();
      const value = (m[2] ?? '').replace(/\s+/g, ' ').trim();
      if (name) tokens.set(name, value);
    }
  }
  return tokens;
}

const dssCss = readFileSync(DSS_PATH, 'utf8');
const tokensCss = readFileSync(TOKENS_PATH, 'utf8');
const dss = parseRootTokens(dssCss);
const cur = parseRootTokens(tokensCss);

// Helper: resolve `var(--x)` chains inside the current tokens map (1-deep is
// enough for our cases; deeper aliases are reduced inline).
function resolveVar(name: string, source: Map<string, string>, depth = 4): string | undefined {
  let v = source.get(name);
  while (v !== undefined && /^var\(--[a-z0-9-]+\)$/i.test(v) && depth-- > 0) {
    const inner = v.match(/^var\((--[a-z0-9-]+)\)$/i)?.[1];
    if (!inner) break;
    v = source.get(inner);
  }
  return v;
}

describe('DSS Universal Standard v3.0 — Token Parity (ISSUE-001/008/009/011/012)', () => {
  it('parses both stylesheets', () => {
    expect(dss.size).toBeGreaterThan(0);
    expect(cur.size).toBeGreaterThan(0);
  });

  describe('Brand palette — 12 canonical swatches', () => {
    const expected = {
      '--indra-deep': '#002B3A',
      '--indra-dark': '#003E50',
      '--indra-primary': '#06596E',
      '--indra-secondary': '#346679',
      '--indra-teal': '#3F96AE',
      '--indra-cyan': '#00B0BD',
      '--indra-light': '#7A9CAE',
      '--indra-blue-gray': '#B3C1DA',
      '--indra-sky': '#BADFF3',
      '--indra-warm-gray': '#B0B4BD',
      '--indra-off-white': '#F2F5F6',
    };
    for (const [name, value] of Object.entries(expected)) {
      it(`${name} === ${value}`, () => {
        const resolved = resolveVar(name, cur);
        expect(resolved).toBeDefined();
        expect((resolved ?? '').toUpperCase()).toBe(value.toUpperCase());
      });
    }
  });

  describe('Status palette', () => {
    const expected = {
      '--indra-success': '#27AE60',
      '--indra-warning': '#FF9800',
      '--indra-error': '#E91E63',
      '--indra-gold': '#FFC107',
    };
    for (const [name, value] of Object.entries(expected)) {
      it(`${name} === ${value}`, () => {
        const resolved = resolveVar(name, cur);
        expect(resolved).toBeDefined();
        expect((resolved ?? '').toUpperCase()).toBe(value.toUpperCase());
      });
    }
  });

  describe('Typography (ISSUE-008)', () => {
    it('--font-sans places "Segoe UI" before "Inter"', () => {
      const fontSans = resolveVar('--font-sans', cur) ?? '';
      const segoe = fontSans.toLowerCase().indexOf('segoe ui');
      const inter = fontSans.toLowerCase().indexOf('inter');
      expect(segoe).toBeGreaterThanOrEqual(0);
      expect(inter).toBeGreaterThanOrEqual(0);
      expect(segoe).toBeLessThan(inter);
    });

    it('--font-mono lists "JetBrains Mono" first', () => {
      const fontMono = resolveVar('--font-mono', cur) ?? '';
      expect(fontMono.toLowerCase()).toMatch(/^['"]?jetbrains mono['"]?/);
    });
  });

  describe('Spacing scale (ISSUE-001 — DSS extended scale)', () => {
    const expected = {
      '--space-1': '4px',
      '--space-2': '8px',
      '--space-3': '12px',
      '--space-4': '16px',
      '--space-6': '24px',
      '--space-8': '32px',
      '--space-12': '48px',
      '--space-16': '64px',
      '--space-20': '80px',
      '--space-30': '120px',
    };
    for (const [name, value] of Object.entries(expected)) {
      it(`${name} === ${value}`, () => {
        expect(resolveVar(name, cur)).toBe(value);
      });
    }
  });

  describe('Timing tokens (ISSUE-001)', () => {
    const expected = {
      '--duration-fast': '200ms',
      '--duration-normal': '300ms',
      '--duration-slow': '500ms',
      '--duration-emphasis': '800ms',
    };
    for (const [name, value] of Object.entries(expected)) {
      it(`${name} === ${value}`, () => {
        expect(resolveVar(name, cur)).toBe(value);
      });
    }
  });

  describe('Easing tokens (ISSUE-001)', () => {
    const expected = {
      '--ease-out': 'cubic-bezier(0.16, 1, 0.3, 1)',
      '--ease-in-out': 'cubic-bezier(0.65, 0, 0.35, 1)',
      '--ease-spring': 'cubic-bezier(0.34, 1.56, 0.64, 1)',
    };
    for (const [name, value] of Object.entries(expected)) {
      it(`${name} === ${value}`, () => {
        expect(resolveVar(name, cur)).toBe(value);
      });
    }
  });

  describe('Geometry (ISSUE-009 — sharp button corners)', () => {
    it('--radius-button === 0 (sharp DSS corporate corners)', () => {
      const v = resolveVar('--radius-button', cur);
      expect(v).toBe('0');
    });
    it('--radius-card === 8px', () => {
      expect(resolveVar('--radius-card', cur)).toBe('8px');
    });
    it('--radius-badge === 9999px (full pill)', () => {
      expect(resolveVar('--radius-badge', cur)).toBe('9999px');
    });
    it('exposes --radius-glass-card === 16px (DSS glass-card)', () => {
      expect(resolveVar('--radius-glass-card', cur)).toBe('16px');
    });
  });

  describe('Section rhythm utilities (ISSUE-011)', () => {
    it('tokens.css declares .section utility with 120px vertical padding', () => {
      expect(tokensCss).toMatch(/\.section\s*\{[^}]*padding:\s*var\(--space-30\)\s*0/);
    });
    it('tokens.css declares .section-title with 40px / weight 300', () => {
      expect(tokensCss).toMatch(/\.section-title\s*\{[^}]*font-size:\s*40px/);
      expect(tokensCss).toMatch(/\.section-title\s*\{[^}]*font-weight:\s*300/);
    });
  });

  describe('Smooth scroll (ISSUE-012)', () => {
    it('tokens.css sets html { scroll-behavior: smooth; }', () => {
      expect(tokensCss).toMatch(/html\s*\{[^}]*scroll-behavior:\s*smooth/);
    });
    it('tokens.css sets html { scroll-padding-top: 80px; }', () => {
      expect(tokensCss).toMatch(/html\s*\{[^}]*scroll-padding-top:\s*80px/);
    });
  });

  describe('Motion keyframes (ISSUE-003)', () => {
    it('declares @keyframes indra-fade-in (24px translateY → 0)', () => {
      expect(tokensCss).toMatch(/@keyframes\s+indra-fade-in\s*\{/);
    });
    it('declares @keyframes indra-slide-in-x (translateX(-20px) → 0)', () => {
      expect(tokensCss).toMatch(/@keyframes\s+indra-slide-in-x\s*\{/);
    });
    it('declares [data-animate] base + .animate-in trigger', () => {
      expect(tokensCss).toMatch(/\[data-animate\][^{]*\{/);
      expect(tokensCss).toMatch(/\[data-animate\]\.animate-in[^{]*\{/);
    });
    it('declares stagger utilities .stagger-1 through .stagger-5', () => {
      for (const n of [1, 2, 3, 4, 5]) {
        expect(tokensCss).toMatch(new RegExp(`\\.stagger-${n}\\b`));
      }
    });
  });

  describe('DSS subset compliance — every DSS token has a matching value in tokens.css', () => {
    // Tokens that DSS declares but tokens.css aliases differently are allowed
    // ONLY IF the resolved value matches.
    const ignoredKeys = new Set<string>([
      // DSS uses --font-sans only; AgentVerse adds --font-body, --font-display.
    ]);
    for (const [name, dssValue] of dss.entries()) {
      if (ignoredKeys.has(name)) continue;
      it(`tokens.css resolves ${name} to ${dssValue.slice(0, 60)}…`, () => {
        const curValue = resolveVar(name, cur);
        expect(curValue, `Missing or mismatched token ${name}`).toBeDefined();
        // Normalize: lowercase, strip quotes around font names.
        const normalize = (s: string): string =>
          s
            .toLowerCase()
            .replace(/['"]/g, '')
            .replace(/\s+/g, ' ')
            .trim();
        expect(normalize(curValue ?? '')).toBe(normalize(dssValue));
      });
    }
  });
});
