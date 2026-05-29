import { describe, it, expect } from 'vitest';
import { maskKey } from '../mask';

describe('maskKey', () => {
  it('correctly masks standard keys with prefix and last 4 chars', () => {
    expect(maskKey('sk-abc123XYZ')).toBe('sk-…3XYZ');
    expect(maskKey('sk-proj-12345')).toBe('sk-…2345');
  });

  it('correctly handles short keys', () => {
    expect(maskKey('abc')).toBe('…bc');
    expect(maskKey('a')).toBe('…a');
    expect(maskKey('')).toBe('');
  });

  it('handles keys without dashes', () => {
    expect(maskKey('abcdefghijklmnop')).toBe('…mnop');
  });
});
