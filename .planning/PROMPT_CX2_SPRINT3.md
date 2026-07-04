# PROMPT CX2 — Codex (Instância 2)
# Workstream: E2E Tests + Smoke Flow GO Core
# Sprint 3 | CRIT-004 (E2E) + RCA A2 + HIGH-003

## SEU PAPEL
Você é CX2, responsável pelos testes E2E Playwright e validação do fluxo completo de deploy.
Você pode escalar 2 sub-agentes: **CX2-A** e **CX2-B**.

## DEPENDÊNCIA
Aguarde GATE 1 de NM1 no ledger antes de iniciar.

## OBRIGATORIO antes de qualquer edição
1. Leia `.planning/AGENT_LEDGER_S3.md`
2. Registre CHECK-IN
3. Leia `docs/patterns/testing.md` — padrão para testes E2E deste projeto

---

### CX2-A: Atualizar smoke spec para GO Core

**Arquivo:** `tests/e2e/smoke.spec.ts`
Leia o arquivo atual. Verifique se há referências a `9889` ou `CAO` hard-coded.
Substitua por configuração via `process.env.VITE_GO_CORE_BASE_URL || 'http://127.0.0.1:8080'`.

**Padrão do projeto (do docs/patterns/testing.md):**
- Use `disableAnimations(page)` no `beforeEach` antes de `page.goto`
- Use `installSpeechRecognitionMock(page, { transcript: '...' })` para testes de voz
- NÃO force-click. NÃO mutate Zustand store via `page.evaluate`
- NÃO use `setTimeout` no spec — use `findByText` ou `waitFor`

---

### CX2-B: Criar canvas-deploy E2E spec (RCA A2 — HIGH-003)

**RCA Finding:** Testes mocked não pegos falhas de integração real. Criar fluxo real (mas com MSW).

**Criar:** `tests/e2e/canvas-deploy.spec.ts`

```typescript
/**
 * Canvas Deploy E2E — RCA-2026-05-31-001 (A2)
 * 
 * Smoke test do fluxo completo: Draft → Deploy → Deployed/Degraded
 * Não requer GO Core real — usa MSW handlers para simular runtime.
 * 
 * Para testar com GO Core real: GO_CORE_LIVE=1 npx playwright test
 */
import { test, expect } from '@playwright/test';
import { disableAnimations } from './helpers/disable-animations';

test.describe('Canvas Deploy Flow', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
    await page.goto('/');
  });

  test('opens canvas builder from sidebar', async ({ page }) => {
    // Navegar para canvas builder
    await expect(page.getByRole('link', { name: /canvas/i })).toBeVisible({ timeout: 10000 });
    await page.getByRole('link', { name: /canvas/i }).click();
    await expect(page.getByRole('heading', { name: /canvas/i })).toBeVisible({ timeout: 5000 });
  });

  test('canvas node transitions draft → deploying → deployed', async ({ page }) => {
    await page.goto('/canvas');

    // Aguardar canvas builder carregar
    await expect(page.locator('[data-testid="canvas-builder"]')).toBeVisible({ timeout: 10000 });

    // Verificar que botão Deploy existe e está habilitado
    const deployBtn = page.getByRole('button', { name: /deploy/i });
    await expect(deployBtn).toBeVisible({ timeout: 5000 });

    // Click deploy (MSW intercept faz o session criado)
    await deployBtn.click();

    // Esperar transição de estado
    await expect(page.locator('[data-node-status="deploying"]')).toBeVisible({ timeout: 10000 });

    // Após MSW retornar session OK, deve virar deployed
    await expect(page.locator('[data-node-status="deployed"]')).toBeVisible({ timeout: 15000 });
  });

  test('health pill shows green when GO Core is reachable', async ({ page }) => {
    // MSW handler responde /health com { status: "ok" }
    await expect(page.locator('[data-testid="health-pill"][data-status="healthy"]'))
      .toBeVisible({ timeout: 10000 });
  });
});
```

**IMPORTANTE:** Se os data-testid's não existirem nos componentes, documente
no CHECK-OUT quais precisam ser adicionados (mas não adicione você mesmo — isso
é escopo de NM2 ou MED tasks).

---

### Verificação CX2:
```bash
# Deve completar sem crashed — alguns podem FAIL por falta de data-testid (documente)
npx playwright test tests/e2e/canvas-deploy.spec.ts --reporter=list 2>&1 | tail -30
```

## GATE de CX2
1. Smoke spec atualizado sem referências hard-coded a :9889 ou "CAO"
2. Canvas deploy spec criado
3. Resultado documentado no ledger (incluindo failures esperados por falta de data-testid)
4. Commit: `test(e2e): canvas deploy flow spec + GO Core smoke — Sprint-3 — RCA-A2`

## REGRAS ABSOLUTAS
- Seguir padrões de `docs/patterns/testing.md` estritamente
- NUNCA force-click ou mutate store diretamente no spec
- SEMPRE registrar no ledger