import { describe, it, expect } from 'vitest';
import { resolveSessionEnv } from '../session-discovery';

describe('resolveSessionEnv', () => {
  it('sets CLAUDE_CONFIG_DIR and ANTHROPIC_MODEL for claude_code', () => {
    const session = {
      id: '1',
      cli_provider: 'claude_code',
      account_email: 't@t.com',
      config_dir: '/home/.claude-test',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session, 'opus-4.8');
    expect(env.CLAUDE_CONFIG_DIR).toBe('/home/.claude-test');
    expect(env.ANTHROPIC_MODEL).toBe('opus-4.8');
  });

  it('omits CLAUDE_CONFIG_DIR if config_dir is empty', () => {
    const session = {
      id: '2',
      cli_provider: 'claude_code',
      account_email: 't@t.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session, 'opus-4.7');
    expect(env.CLAUDE_CONFIG_DIR).toBeUndefined();
    expect(env.ANTHROPIC_MODEL).toBe('opus-4.7');
  });

  it('sets OPENAI_MODEL for codex', () => {
    const session = {
      id: '3',
      cli_provider: 'codex',
      account_email: 't@t.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session, 'codex-5.5');
    expect(env.OPENAI_MODEL).toBe('codex-5.5');
  });

  it('sets GEMINI_MODEL for gemini_cli', () => {
    const session = {
      id: '4',
      cli_provider: 'gemini_cli',
      account_email: 't@t.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'gcloud' as const,
    };
    const env = resolveSessionEnv(session, 'gemini-3.5-flash');
    expect(env.GEMINI_MODEL).toBe('gemini-3.5-flash');
  });

  it('sets KIRO_HOME for kiro_cli', () => {
    const session = {
      id: '5',
      cli_provider: 'kiro_cli',
      account_email: 't@t.com',
      config_dir: '/home/.kiro-test',
      status: 'active' as const,
      auth_method: 'sso' as const,
    };
    const env = resolveSessionEnv(session);
    expect(env.KIRO_HOME).toBe('/home/.kiro-test');
  });

  it('returns empty object for unknown provider', () => {
    const session = {
      id: '6',
      cli_provider: 'unknown',
      account_email: 't@t.com',
      config_dir: '',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session);
    expect(Object.keys(env)).toHaveLength(0);
  });

  it('omits model env var when model is undefined', () => {
    const session = {
      id: '7',
      cli_provider: 'claude_code',
      account_email: 't@t.com',
      config_dir: '/home/.claude',
      status: 'active' as const,
      auth_method: 'oauth' as const,
    };
    const env = resolveSessionEnv(session);
    expect(env.CLAUDE_CONFIG_DIR).toBe('/home/.claude');
    expect(env.ANTHROPIC_MODEL).toBeUndefined();
  });
});
