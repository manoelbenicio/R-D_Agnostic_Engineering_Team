import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Button, StatusBadge, FormField } from '../index';

describe('SENTINEL Components Tests', () => {
  describe('StatusBadge', () => {
    it('renders the correct glyph and style for error status', () => {
      const { container } = render(<StatusBadge status="error" label="Failed Connection" />);
      
      const glyph = container.querySelector('.status-glyph');
      expect(glyph?.textContent).toBe('✕');
      expect(screen.getByText('Failed Connection')).toBeInTheDocument();
      
      const badge = container.querySelector('.sentinel-badge');
      expect(badge).toHaveStyle({
        color: 'var(--indra-error)',
      });
    });

    it('renders the correct glyph and style for completed status', () => {
      const { container } = render(<StatusBadge status="completed" />);
      
      const glyph = container.querySelector('.status-glyph');
      expect(glyph?.textContent).toBe('✓');
      
      const badge = container.querySelector('.sentinel-badge');
      expect(badge).toHaveStyle({
        color: 'var(--indra-success)',
      });
    });

    it('renders the correct glyph and pulsing class for processing status', () => {
      const { container } = render(<StatusBadge status="processing" />);
      
      const glyph = container.querySelector('.status-glyph');
      expect(glyph?.textContent).toBe('●');
      expect(glyph).toHaveClass('blink');
      
      const badge = container.querySelector('.sentinel-badge');
      expect(badge).toHaveStyle({
        // DSS info variant: cyan family. Indra-sky is used for foreground per DSS spec.
        color: 'var(--indra-sky)',
      });
    });
  });

  describe('Button', () => {
    it('renders and contains focus-visible properties', () => {
      render(<Button id="test-btn">Click Me</Button>);
      const btn = screen.getByRole('button', { name: 'Click Me' });
      expect(btn).toBeInTheDocument();
      
      btn.focus();
      expect(document.activeElement).toBe(btn);
      expect(btn).toHaveClass('sentinel-btn-sys');
    });
  });

  describe('FormField', () => {
    it('wires the label htmlFor attribute to the input id', () => {
      render(
        <FormField label="Secret Token" id="secret-input">
          <input type="password" />
        </FormField>
      );
      
      const label = screen.getByText('Secret Token');
      const input = screen.getByLabelText('Secret Token');
      
      expect(label).toHaveAttribute('for', 'secret-input');
      expect(input).toHaveAttribute('id', 'secret-input');
    });

    it('renders error message when errorText is present', () => {
      render(
        <FormField label="API Key" id="api-input" errorText="Key is invalid">
          <input type="text" />
        </FormField>
      );
      
      const errorMsg = screen.getByText('Key is invalid');
      expect(errorMsg).toBeInTheDocument();
      expect(errorMsg).toHaveAttribute('id', 'api-input-error');
      
      const input = screen.getByLabelText('API Key');
      expect(input).toHaveAttribute('aria-invalid', 'true');
      expect(input).toHaveAttribute('aria-describedby', 'api-input-error');
    });
  });
});
