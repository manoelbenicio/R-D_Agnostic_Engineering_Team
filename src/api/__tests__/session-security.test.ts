import { describe, it, expect, vi, afterEach } from 'vitest';
import { maskEmail, maskConfigDir, isExpiringSoon, sanitizeForLog } from '../session-security';

describe('maskEmail', () => {
  it('masks standard email showing first 2 chars', () => {
    expect(maskEmail('john.doe@example.com')).toBe('jo***@example.com');
  });

  it('masks single-char local part', () => {
    expect(maskEmail('a@b.com')).toBe('a***@b.com');
  });

  it('returns original string when no @ sign present', () => {
    expect(maskEmail('invalid')).toBe('invalid');
  });
});

describe('maskConfigDir', () => {
  it('shows only last segment for unix paths', () => {
    expect(maskConfigDir('/home/user/.claude-test')).toBe('…/.claude-test');
  });

  it('shows only last segment for windows paths', () => {
    expect(maskConfigDir('C:\\Users\\test\\.codex')).toBe('…/.codex');
  });

  it('returns empty string for empty input', () => {
    expect(maskConfigDir('')).toBe('');
  });
});

describe('isExpiringSoon', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns false when expiresAt is undefined', () => {
    expect(isExpiringSoon(undefined)).toBe(false);
  });

  it('returns false when expiry is far in the future', () => {
    const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString(); // 1 hour
    expect(isExpiringSoon(futureDate, 30)).toBe(false);
  });

  it('returns true when expiry is within threshold', () => {
    const soonDate = new Date(Date.now() + 10 * 60 * 1000).toISOString(); // 10 minutes
    expect(isExpiringSoon(soonDate, 30)).toBe(true);
  });
});

describe('sanitizeForLog', () => {
  it('redacts sensitive keys', () => {
    const result = sanitizeForLog({ id: '1', auth_token: 'secret123' });
    expect(result).toEqual({ id: '1', auth_token: '[REDACTED]' });
  });
});
