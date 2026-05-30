import { test, expect, type Page } from '@playwright/test';

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
        const tx = db.transaction('app_state', 'readwrite');
        const store = tx.objectStore('app_state');
        const putRequest = store.put({ key: 'wizard_completed', value: true });
        putRequest.onsuccess = () => resolve();
        putRequest.onerror = () => reject(putRequest.error);
      };
      openRequest.onerror = () => reject(openRequest.error);
    });
  });
}

test.describe('Sessions Page', () => {
  test.beforeEach(async ({ page }) => {
    await completeFirstRun(page);
  });

  test('navigates to /sessions from navbar', async ({ page }) => {
    await page.goto('/');
    await page.click('#nav-link-sessions');
    await expect(page).toHaveURL(/sessions/);
    await expect(page.locator('h1')).toContainText('AUTH SESSIONS');
  });

  test('shows provider sections', async ({ page }) => {
    await page.goto('/sessions');
    await expect(page.getByRole('heading', { name: 'CLAUDE CODE' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'CODEX' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'GEMINI CLI' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'KIRO CLI' })).toBeVisible();
  });

  test('refresh button exists and is clickable', async ({ page }) => {
    await page.goto('/sessions');
    const refreshBtn = page.locator('button:has-text("Refresh")');
    await expect(refreshBtn).toBeVisible();
    await refreshBtn.click();
  });

  test('shows empty state when no sessions detected', async ({ page }) => {
    await page.goto('/sessions');
    await expect(page.locator('.sessions-page')).toBeVisible();
  });

  test('session status badge appears in navbar', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.session-status-badge')).toBeVisible();
  });
});
