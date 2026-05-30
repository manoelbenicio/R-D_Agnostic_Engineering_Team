import { expect, test, type Page } from '@playwright/test';

async function completeFirstRun(page: Page) {
  await page.goto('/favicon.svg');
  await page.evaluate(async () => {
    await new Promise<void>((resolve, reject) => {
      const deleteRequest = indexedDB.deleteDatabase('AgentVerse');
      deleteRequest.onblocked = () => resolve();
      deleteRequest.onsuccess = () => resolve();
      deleteRequest.onerror = () => reject(deleteRequest.error);
    });
  });

  await page.goto('/');
  await page.evaluate(async () => {
    await new Promise<void>((resolve, reject) => {
      const openRequest = indexedDB.open('AgentVerse', 2);
      openRequest.onupgradeneeded = () => {
        const db = openRequest.result;
        if (!db.objectStoreNames.contains('app_state')) {
          db.createObjectStore('app_state', { keyPath: 'key' });
        }
      };
      openRequest.onsuccess = () => {
        const db = openRequest.result;
        const transaction = db.transaction('app_state', 'readwrite');
        const putRequest = transaction
          .objectStore('app_state')
          .put({ key: 'wizard_completed', value: true });
        putRequest.onsuccess = () => resolve();
        putRequest.onerror = () => reject(putRequest.error);
      };
      openRequest.onerror = () => reject(openRequest.error);
    });
  });
}

async function openDraftCanvas(page: Page) {
  await page.goto('/canvas/e2e-canvas');
  await expect(page.locator('.canvas-flow-shell')).toBeVisible();
}

async function instantiateCodeReviewTemplate(page: Page) {
  await openDraftCanvas(page);
  await page.getByRole('button', { name: 'Browse Templates' }).click();
  const template = page
    .locator('.template-picker-card')
    .filter({ hasText: 'Code Review Pipeline' });
  await template
    .getByRole('button', { name: 'Use Template' })
    .evaluate((element) => (element as HTMLButtonElement).click());
  await expect(
    page.locator('.react-flow__node').filter({ hasText: 'Supervisor' }).first(),
  ).toBeVisible();
}

test.describe('Canvas session controls', () => {
  test.beforeEach(async ({ page }) => {
    await completeFirstRun(page);
  });

  test('fullscreen toggle works from the canvas toolbar', async ({ page }) => {
    await openDraftCanvas(page);
    await page.getByRole('button', { name: 'Fullscreen' }).click();
    await expect(page.locator('.canvas-builder-page')).toHaveClass(/canvas-fullscreen-mode/);

    await page.keyboard.press('Control+Shift+F');
    await expect(page.locator('.canvas-builder-page')).not.toHaveClass(/canvas-fullscreen-mode/);
  });

  test('zoom controls display the current zoom percentage', async ({ page }) => {
    await openDraftCanvas(page);
    const zoom = page.locator('.canvas-zoom-level');
    await expect(zoom).toHaveText('100%');

    await page.locator('.canvas-floating-toolbar').getByRole('button', { name: 'Zoom in' }).click();
    await expect(zoom).not.toHaveText('100%');
  });

  test('Fit View button resets zoom', async ({ page }) => {
    await instantiateCodeReviewTemplate(page);
    const zoom = page.locator('.canvas-zoom-level');
    const zoomIn = page
      .locator('.canvas-floating-toolbar')
      .getByRole('button', { name: 'Zoom in' });
    await zoomIn.click();
    await zoomIn.click();
    const zoomed = await zoom.textContent();

    await page
      .locator('.canvas-floating-toolbar')
      .getByRole('button', { name: 'Fit View' })
      .click();
    await expect.poll(() => zoom.textContent()).not.toBe(zoomed);
  });

  test('config panel has Auth Session dropdown', async ({ page }) => {
    await instantiateCodeReviewTemplate(page);
    await page.locator('.react-flow__node').filter({ hasText: 'Supervisor' }).first().click();

    await expect(page.locator('#block-session')).toBeVisible();
  });

  test('session dropdown shows Auto (default session) as default option', async ({ page }) => {
    await instantiateCodeReviewTemplate(page);
    await page.locator('.react-flow__node').filter({ hasText: 'Supervisor' }).first().click();

    await expect(page.locator('#block-session option').first()).toHaveText(
      'Auto (default session)',
    );
  });

  test('? opens keyboard shortcut help and Escape closes it', async ({ page }) => {
    await openDraftCanvas(page);
    await page.keyboard.press('?');
    await expect(page.getByRole('dialog', { name: 'Keyboard Shortcuts' })).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Keyboard Shortcuts' })).toBeHidden();
  });
});
