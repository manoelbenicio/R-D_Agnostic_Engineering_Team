# PROMPT CX5 — Codex 5.5 High Thinking
# Workstream: Test Reliability + Contract Tests GO Core
# Sprint 3 | CRIT-004 + HIGH-002 + RCA A1+A2+A3

## SEU PAPEL
Você é CX5, responsável por tornar o test suite confiável e criar os contract tests do GO Core.
Você pode escalar 2 sub-agentes: **CX5-A** e **CX5-B**.

## OBRIGATORIO antes de qualquer edição
1. Leia `.planning/AGENT_LEDGER_S3.md`
2. Registre CHECK-IN para cada arquivo
3. Verifique GATE 1 aprovado (NM1 done) antes de iniciar tests que dependam de typecheck

---

### CX5-A: CRIT-004 — Test Suite Reliability

**Problema:** Vitest > 4 min em UNC path, Zustand stores poluídos entre testes.

#### `vitest.config.ts`
Adicione dentro do bloco `test: { ... }`:
```typescript
test: {
  globals: true,
  environment: 'jsdom',
  setupFiles: ['./src/__tests__/setup.ts'],
  css: true,
  testTimeout: 60000,        // ← ADD: evita timeouts em UNC path
  pool: 'forks',             // ← ADD: melhor isolamento que threads
  poolOptions: {
    forks: { singleFork: false }
  },
  clearMocks: true,          // ← ADD: limpa mocks entre testes
  restoreMocks: true,        // ← ADD: restaura spies entre testes
  // Contract tests são gate-behind GO_CORE_LIVE=1
  exclude:
    process.env.GO_CORE_LIVE === '1'
      ? configDefaults.exclude
      : [...configDefaults.exclude, 'src/api/__tests__/contract/**', 'tests/e2e/**'],
  coverage: { ... },  // manter igual ao atual
}
```

**IMPORTANTE:** O `exclude` atual usa `CAO_LIVE === '1'` — troque para `GO_CORE_LIVE === '1'`.

#### `src/__tests__/setup.ts`
Leia o arquivo atual primeiro. Adicione no final um `beforeEach` que reseta os stores:
```typescript
import { beforeEach, afterEach } from 'vitest';

// Limpar Zustand stores entre testes para evitar poluição
beforeEach(() => {
  // Reset session store
  try {
    const { useSessionStore } = await import('@/api/session-store');
    useSessionStore.setState({ sessions: [], loading: false, error: null, lastRefreshed: null });
  } catch { /* store pode nao existir em todos os contexts */ }
});
```

Verifique o arquivo atual e adapte conforme o que já existe nele.

**Verificação CX5-A:**
```bash
npx vitest run --reporter=verbose 2>&1 | tail -30
# Deve completar em menos de 120s
```
Documente o tempo e resultado no ledger.

---

### CX5-B: RCA A1+A3 — Contract Tests GO Core

**Contexto:** RCA-2026-05-31-001 exige contract tests contra o GO Core real.
O GO Core NÃO está rodando agora — crie os testes estruturalmente corretos
que serão executados quando GO_CORE_LIVE=1.

**Criar:** `tests/contract/go-core-surface.test.ts`

```typescript
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
  it('WS /terminals/:id/ws URL is constructed correctly', () => {
    const { buildTerminalSocketUrl } = require('../../src/api/connect-terminal-socket');
    const url = buildTerminalSocketUrl('test-id', GO_CORE_URL);
    expect(url).toMatch(/^ws:\/\//);
    expect(url).toContain('/terminals/test-id/ws');
  });
});
```

**Criar:** `scripts/pre-ship-gate.sh` (RCA A3):
```bash
#!/usr/bin/env bash
# Pre-ship gate — RCA-2026-05-31-001 (A3)
# Runs: unit tests → typecheck → lint → build → contract tests (if GO_CORE_LIVE=1)
set -euo pipefail

echo "=== PRE-SHIP GATE: AgentVerse — GO Core ==="
echo "1. TypeScript..."
npx tsc --noEmit

echo "2. Lint..."
npm run lint

echo "3. Unit tests..."
npm run test

echo "4. Build..."
npm run build
node scripts/check-bundle-size.mjs

if [[ "${GO_CORE_LIVE:-0}" == "1" ]]; then
  echo "5. Contract tests (GO_CORE_LIVE=1)..."
  GO_CORE_LIVE=1 npx vitest run tests/contract/
else
  echo "5. Contract tests SKIPPED (set GO_CORE_LIVE=1 to enable)"
fi

echo "=== ALL GATES PASSED ==="
```

**Verificação CX5-B:**
```bash
# Sem GO Core rodando — todos devem SKIP (não FAIL)
npx vitest run tests/contract/ --reporter=verbose 2>&1 | tail -20
```

---

## GATE de CX5
1. Test suite < 120s
2. Contract tests estruturados corretamente (skip quando GO_CORE_LIVE != 1)
3. Pre-ship gate script criado
4. Registrar CHECK-OUT no ledger
5. Commit: `test(contract): GO Core contract tests + test reliability — Sprint-3 — RCA-A1-A3`

## REGRAS ABSOLUTAS
- NUNCA rodar `npm install` ou `npm audit fix`
- SEMPRE registrar no ledger