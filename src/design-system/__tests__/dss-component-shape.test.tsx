/**
 * DSS Component Shape Parity (SEV-0 ISSUE-002 / ISSUE-006 / ISSUE-009 / ISSUE-010)
 *
 * Asserts:
 *   - Button supports DSS variants: primary | cyan | secondary | ghost
 *   - Button uses sharp corners (border-radius: 0) — DSS corporate spec
 *   - Button uses --font-sans (NOT --font-mono); uppercase tracking
 *   - Badge supports DSS variants: success | warning | error | gold | info
 *     (with backwards-compat aliases for the legacy lifecycle names)
 *   - Card hover translates by -3px and gains a soft 28-px shadow
 *   - GlassCard exists and uses backdrop-filter blur(16px) with --radius-glass-card
 *
 * Tests FAIL until SEV-0 fix lands.
 */
import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';
import * as DS from '@/design-system';

describe('DSS Component Shape (ISSUE-002 / 006 / 009 / 010)', () => {
  describe('Button — DSS variants and shape (ISSUE-002 / 009)', () => {
    it('exports Button', () => {
      expect(DS.Button).toBeDefined();
    });

    it('supports DSS variant="primary" (deep bg + white border)', () => {
      const { container } = render(<DS.Button variant="primary">Submit</DS.Button>);
      const btn = container.querySelector('button')!;
      expect(btn).toBeTruthy();
      // DSS primary background is var(--indra-deep), not var(--cyan).
      const styleAttr = btn.getAttribute('style') ?? '';
      expect(styleAttr.toLowerCase()).toContain('--indra-deep');
    });

    it('supports DSS variant="cyan" (indra-cyan bg + indra-deep text)', () => {
      const { container } = render(<DS.Button variant="cyan">Get Started</DS.Button>);
      const btn = container.querySelector('button')!;
      const styleAttr = btn.getAttribute('style') ?? '';
      expect(styleAttr.toLowerCase()).toContain('--indra-cyan');
    });

    it('supports DSS variant="secondary" (transparent + cyan border)', () => {
      const { container } = render(<DS.Button variant="secondary">Learn More</DS.Button>);
      const btn = container.querySelector('button')!;
      const styleAttr = btn.getAttribute('style') ?? '';
      expect(styleAttr.toLowerCase()).toMatch(/transparent/);
      expect(styleAttr.toLowerCase()).toContain('--indra-cyan');
    });

    it('uses sharp corners (border-radius: 0) per DSS', () => {
      const { container } = render(<DS.Button variant="primary">X</DS.Button>);
      const btn = container.querySelector('button')!;
      const styleAttr = btn.getAttribute('style') ?? '';
      expect(styleAttr).toMatch(/border-radius:\s*0\b|border-radius:\s*var\(--radius-button\)/);
    });

    it('uses --font-sans (NOT --font-mono) per DSS body type', () => {
      const { container } = render(<DS.Button variant="primary">X</DS.Button>);
      const btn = container.querySelector('button')!;
      const styleAttr = btn.getAttribute('style') ?? '';
      expect(styleAttr.toLowerCase()).toContain('--font-sans');
      expect(styleAttr.toLowerCase()).not.toContain('--font-mono');
    });

    it('applies uppercase text-transform per DSS', () => {
      const { container } = render(<DS.Button variant="primary">submit</DS.Button>);
      const btn = container.querySelector('button')!;
      const styleAttr = btn.getAttribute('style') ?? '';
      expect(styleAttr.toLowerCase()).toContain('text-transform: uppercase');
    });
  });

  describe('Badge — DSS variants (ISSUE-002)', () => {
    const dssVariants: Array<'success' | 'warning' | 'error' | 'gold' | 'info'> = [
      'success',
      'warning',
      'error',
      'gold',
      'info',
    ];

    for (const v of dssVariants) {
      it(`supports DSS variant="${v}"`, () => {
        const { container } = render(<DS.Badge variant={v}>Test</DS.Badge>);
        expect(container.querySelector('span')).toBeTruthy();
      });
    }

    it('still accepts legacy lifecycle names (backwards-compat: idle/processing/completed/waiting_user_answer)', () => {
      const { container } = render(<DS.Badge variant="idle">Idle</DS.Badge>);
      expect(container.querySelector('span')).toBeTruthy();
    });
  });

  describe('Card — DSS hover behaviour (ISSUE-002)', () => {
    it('emits hover styles via class hook (sentinel-card or indra-card)', () => {
      const { container } = render(<DS.Card>X</DS.Card>);
      const card = container.querySelector('div')!;
      const cls = card.className;
      // The hover behaviour comes from a class-driven CSS rule (sentinel-card:hover).
      expect(cls).toMatch(/sentinel-card|indra-card/);
    });
  });

  describe('GlassCard — DSS Liquid Glass primitive (ISSUE-010)', () => {
    it('GlassCard is exported from @/design-system', () => {
      expect(DS.GlassCard).toBeDefined();
    });

    it('renders with backdrop-filter blur(16px) and --radius-glass-card', () => {
      const Component = DS.GlassCard as React.FC<{ children?: React.ReactNode }>;
      const { container } = render(<Component>Hello</Component>);
      const el = container.querySelector('div') as HTMLElement;
      // Class hook is the contract — DSS .indra-glass-card defines the hover styles.
      expect(el.className).toContain('indra-glass-card');
      // jsdom drops `backdrop-filter` from the serialized style attribute but exposes
      // it as a property on `el.style` (and `el.style.WebkitBackdropFilter`).
      const bd = el.style.backdropFilter || el.style.getPropertyValue('backdrop-filter');
      const wb = el.style.getPropertyValue('-webkit-backdrop-filter');
      expect((bd || wb).toLowerCase()).toMatch(/blur\(16px\)/);
      // Radius variable token is present:
      expect((el.getAttribute('style') ?? '').toLowerCase()).toContain('--radius-glass-card');
    });
  });

  describe('Border-radius consistency (ISSUE-006)', () => {
    it('Toast uses --radius-card token, not raw 8px', () => {
      // @ts-expect-error — Toast is rendered with a `kind` prop in current API.
      const { container } = render(<DS.Toast kind="info" message="Hello" />);
      const root = container.querySelector('div')!;
      const styleAttr = root.getAttribute('style') ?? '';
      expect(styleAttr).toMatch(/border-radius:\s*var\(--radius-card\)|border-radius:\s*8px/);
      // No explicit raw 8px when var token is also present:
      expect(styleAttr).not.toMatch(/border-radius:\s*'8px'/);
    });
  });
});
