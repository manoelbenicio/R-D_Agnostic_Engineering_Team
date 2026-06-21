/**
 * Canvas Deploy E2E — RCA-2026-05-31-001 (A2)
 *
 * Smoke test do fluxo completo: Draft → Deploy → Deployed/Degraded
 * Não requer GO Core real — usa MSW handlers para simular runtime.
 *
 * Para testar com GO Core real: GO_CORE_LIVE=1 npx playwright test
 *
 * Sprint-3 | CRIT-004 (E2E) + HIGH-003
 *
 * -----------------------------------------------------------------
 * MISSING data-testid REPORT (não adicionado — escopo de NM2/MED):
 *   • [data-testid="canvas-builder"]  — CanvasBuilder root element
 *   • [data-node-status="deploying"]  — CanvasNode during deploy transition
 *   • [data-node-status="deployed"]   — CanvasNode after successful deploy
 *   • [data-testid="health-pill"][data-status="healthy"] — GO Core health badge
 * These must be wired into the respective components before GATE 3 can pass.
 * -----------------------------------------------------------------
 */
import { test, expect } from '@playwright/test';
import { disableAnimations } from './helpers/disable-animations';

test.describe('Canvas Deploy Flow', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
    await page.goto('/');
  });

  test('opens canvas builder from sidebar', async ({ page }) => {
    // Navegar para canvas builder via sidebar link
    await expect(page.getByRole('link', { name: /canvas/i })).toBeVisible({ timeout: 10000 });
    await page.getByRole('link', { name: /canvas/i }).click();
    await expect(page.getByRole('heading', { name: /canvas/i })).toBeVisible({ timeout: 5000 });
  });

  test('canvas node transitions draft → deploying → deployed', async ({ page }) => {
    await page.goto('/canvas');

    // Aguardar canvas builder carregar
    // NOTE: data-testid="canvas-builder" must be added to CanvasBuilder component (NM2/MED scope)
    await expect(page.locator('[data-testid="canvas-builder"]')).toBeVisible({ timeout: 10000 });

    // Verificar que botão Deploy existe e está habilitado
    const deployBtn = page.getByRole('button', { name: /deploy/i });
    await expect(deployBtn).toBeVisible({ timeout: 5000 });

    // Click deploy (MSW intercept faz o session criado)
    await deployBtn.click();

    // Esperar transição de estado
    // NOTE: data-node-status="deploying" must be added to CanvasNode component (NM2/MED scope)
    await expect(page.locator('[data-node-status="deploying"]')).toBeVisible({ timeout: 10000 });

    // Após MSW retornar session OK, deve virar deployed
    // NOTE: data-node-status="deployed" must be added to CanvasNode component (NM2/MED scope)
    await expect(page.locator('[data-node-status="deployed"]')).toBeVisible({ timeout: 15000 });
  });

  test('health pill shows green when GO Core is reachable', async ({ page }) => {
    // MSW handler responde /health com { status: "ok" }
    // NOTE: data-testid="health-pill" with data-status="healthy" must be wired into HealthPill
    //       component (NM2/MED scope).
    await expect(page.locator('[data-testid="health-pill"][data-status="healthy"]'))
      .toBeVisible({ timeout: 10000 });
  });
});
