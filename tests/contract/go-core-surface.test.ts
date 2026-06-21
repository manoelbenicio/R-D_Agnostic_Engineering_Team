/**
 * GO Core Contract Tests — RCA-2026-05-31-001 (A1)
 *
 * Execução: GO_CORE_LIVE=1 npx vitest run tests/contract/
 * Requer: GO Core Server rodando em VITE_GO_CORE_BASE_URL (default: http://127.0.0.1:8080)
 */
import { describe, it, expect, beforeAll } from 'vitest';
import { GoCoreClient } from '../../src/api/go-core-client';

const GO_CORE_LIVE = process.env.GO_CORE_LIVE === '1';
const GO_CORE_URL = process.env.VITE_GO_CORE_BASE_URL || 'http://127.0.0.1:8080';

// Skip all tests if GO Core is not live
const describeIfLive = GO_CORE_LIVE ? describe : describe.skip;

let client: GoCoreClient;

beforeAll(() => {
  client = new GoCoreClient(GO_CORE_URL);
});

describeIfLive('GO Core Contract — Health', () => {
  it('GET /health returns { status: "ok" }', async () => {
    const health = await client.getHealth();
    expect(health.status).toBe('ok');
  });
});

describeIfLive('GO Core Contract — Profiles', () => {
  it('GET /agents/profiles returns array', async () => {
    const profiles = await client.listProfiles();
    expect(Array.isArray(profiles)).toBe(true);
  });

  it('POST /agents/profiles/install accepts markdown with frontmatter', async () => {
    const markdown = `---
name: contract-test-agent
role: Developer
provider: codex
description: Contract test agent
---
# Contract Test Agent
`;
    const result = await client.installProfile(markdown);
    expect(result.name).toBe('contract-test-agent');
  });
});

describeIfLive('GO Core Contract — Providers', () => {
  it('GET /agents/providers returns ProviderAvailability[]', async () => {
    const providers = await client.listProviders();
    expect(Array.isArray(providers)).toBe(true);
    if (providers.length > 0) {
      expect(providers[0]).toHaveProperty('name');
      expect(providers[0]).toHaveProperty('installed');
    }
  });
});

describeIfLive('GO Core Contract — Sessions', () => {
  it('GET /sessions returns array', async () => {
    const sessions = await client.listSessions();
    expect(Array.isArray(sessions)).toBe(true);
  });

  it('POST /sessions accepts provider as query param', async () => {
    // This will succeed (201) or fail with 400 (invalid profile) — both prove
    // the endpoint exists and reads query params (not JSON body)
    try {
      await client.createSession({
        profile: 'contract-test-agent',
        working_directory: '/tmp',
        provider: 'codex',
      });
    } catch (err: unknown) {
      // 400 = endpoint exists but profile validation failed — that's OK
      // 404 = endpoint missing — FAIL
      // 422 = wrong param encoding — FAIL
      if (err && typeof err === 'object' && 'status' in err) {
        expect((err as { status: number }).status).not.toBe(404);
        expect((err as { status: number }).status).not.toBe(422);
      }
    }
  });
});

describeIfLive('GO Core Contract — Auth Sessions', () => {
  it('GET /auth/sessions returns array (may be empty)', async () => {
    const sessions = await client.listAuthSessions();
    expect(Array.isArray(sessions)).toBe(true);
  });
});

describeIfLive('GO Core Contract — WebSocket', () => {
  it('WS /terminals/:id/ws URL is constructed correctly', async () => {
    const { buildTerminalSocketUrl } = await import('../../src/api/connect-terminal-socket');
    const url = buildTerminalSocketUrl('test-id', GO_CORE_URL);
    expect(url).toMatch(/^ws:\/\//);
    expect(url).toContain('/terminals/test-id/ws');
  });
});
