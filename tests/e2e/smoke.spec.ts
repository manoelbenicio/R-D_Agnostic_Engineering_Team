import { test, expect } from '@playwright/test';
import { disableAnimations } from './helpers/disable-animations';
import { installSpeechRecognitionMock } from './helpers/speech-recognition-mock';

test.describe('AgentVerse App Shell Smoke Tests', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', (msg) => console.log('BROWSER LOG:', msg.text()));
    page.on('pageerror', (err) => console.error('BROWSER EXCEPTION:', err.message));

    // Strip CSS animations / transitions on every navigation so Playwright's
    // actionability checks don't see infinitely-animating elements (e.g. the
    // voice 🛑 pulse-mic button) as "unstable" and reject the click.
    await disableAnimations(page);

    // Headless Chromium has no Web Speech API; install a deterministic
    // polyfill so clicking 🎤 drives the real
    // getSTTEngine() → VoiceCapture → setFinalTranscript →
    // matchRuntimeCommand → executeRuntimeCommand pipeline end-to-end.
    await installSpeechRecognitionMock(page, { transcript: 'focus on supervisor' });

    // Clear IndexedDB by visiting a static resource (no JS running) to avoid blocking connections
    await page.goto('/favicon.svg');
    await page.evaluate(async () => {
      return new Promise<void>((resolve, reject) => {
        const req = indexedDB.deleteDatabase('AgentVerse');
        req.onblocked = () => resolve();
        req.onsuccess = () => resolve();
        req.onerror = () => reject(req.error);
      });
    });
  });

  test('should load the home page (Canvas List coming soon)', async ({ page }) => {
    // 1. Load home page to let the React app initialize the DB and object stores
    await page.goto('/');
    await expect(page.locator('#app-navbar')).toBeVisible();

    // 2. Now write wizard_completed to IndexedDB directly on the same page
    await page.evaluate(async () => {
      return new Promise<void>((resolve, reject) => {
        const req = indexedDB.open('AgentVerse', 1);
        req.onupgradeneeded = () => {
          const db = req.result;
          if (!db.objectStoreNames.contains('app_state')) {
            db.createObjectStore('app_state', { keyPath: 'key' });
          }
        };
        req.onsuccess = () => {
          const db = req.result;
          const tx = db.transaction('app_state', 'readwrite');
          const store = tx.objectStore('app_state');
          const reqPut = store.put({ key: 'wizard_completed', value: true });
          reqPut.onsuccess = () => resolve();
          reqPut.onerror = () => reject(reqPut.error);
        };
        req.onerror = () => reject(req.error);
      });
    });

    // 3. Reload page to bypass wizard
    await page.goto('/');

    // Check document title
    await expect(page).toHaveTitle(/AgentVerse/);

    // Verify navbar is visible
    const navbar = page.locator('#app-navbar');
    await expect(navbar).toBeVisible();

    // Verify home page displays Canvas List Placeholder
    const placeholder = page.locator('text=Canvas List');
    await expect(placeholder).toBeVisible();

    // Verify CAO health pill exists and displays ONLINE status
    const healthPill = page.locator('#cao-health-pill');
    await expect(healthPill).toBeVisible();
    await expect(healthPill).toContainText('CAO ONLINE');
  });

  test('should navigate to 404 page for unknown routes', async ({ page }) => {
    // 1. Load home page to let the React app initialize the DB and object stores
    await page.goto('/');
    await expect(page.locator('#app-navbar')).toBeVisible();

    // 2. Now write wizard_completed to IndexedDB directly on the same page
    await page.evaluate(async () => {
      return new Promise<void>((resolve, reject) => {
        const req = indexedDB.open('AgentVerse', 1);
        req.onupgradeneeded = () => {
          const db = req.result;
          if (!db.objectStoreNames.contains('app_state')) {
            db.createObjectStore('app_state', { keyPath: 'key' });
          }
        };
        req.onsuccess = () => {
          const db = req.result;
          const tx = db.transaction('app_state', 'readwrite');
          const store = tx.objectStore('app_state');
          const reqPut = store.put({ key: 'wizard_completed', value: true });
          reqPut.onsuccess = () => resolve();
          reqPut.onerror = () => reject(reqPut.error);
        };
        req.onerror = () => reject(req.error);
      });
    });

    // 3. Navigate to unknown route
    await page.goto('/some-invalid-path-123');

    // Verify 404 container is loaded
    const notFound = page.locator('#not-found-page');
    await expect(notFound).toBeVisible();
    await expect(notFound).toContainText('PAGE NOT FOUND');

    // Clicking Return to Canvas takes back home
    const backLink = page.locator('#back-home-link');
    await backLink.click();

    // Expect to be back on the canvas page
    await expect(page).toHaveURL('/');
    const placeholder = page.locator('text=Canvas List');
    await expect(placeholder).toBeVisible();
  });

  test('should complete the critical path flow (configure provider -> create canvas -> drop nodes/edges -> deploy -> see terminal -> voice command)', async ({ page }) => {
    // Mock the external Anthropic model check URL
    await page.route('https://api.anthropic.com/v1/models', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [{ id: 'claude-3-5-sonnet-latest' }],
        }),
      });
    });

    // 1. Go to root page - onboarding wizard should be visible
    await page.goto('/');

    // Skip the onboarding wizard to navigate to the normal Canvas List view
    const skipBtn = page.locator('button:has-text("Skip Setup")');
    await expect(skipBtn).toBeVisible();
    await skipBtn.click();

    // Verify normal Canvas List view is visible
    await expect(page.locator('text=Canvas List')).toBeVisible();

    // 2. Go to settings/providers to configure provider keys
    await page.goto('/settings/providers');

    // Fill in API key input element for Anthropic
    const apiKeyInput = page.locator('#anthropic-apiKey');
    await expect(apiKeyInput).toBeVisible();
    await apiKeyInput.fill('sk-ant-test-key-123456');

    // Click Validate & Save in the Anthropic card
    const anthropicCard = page.locator('.sentinel-card').filter({ hasText: 'Anthropic' });
    const validateBtn = anthropicCard.locator('button:has-text("Validate & Save")');
    await expect(validateBtn).toBeVisible();
    await validateBtn.click();

    // Wait for success validation toast
    await expect(page.locator('text=Successfully validated and saved Anthropic!')).toBeVisible();

    // 3. Return to root page and create canvas via template
    await page.goto('/');
    await expect(page.locator('text=Canvas List')).toBeVisible();

    // Open Templates picker
    const templatesBtn = page.locator('button:has-text("Templates"), button:has-text("Browse Templates")').first();
    await expect(templatesBtn).toBeVisible();
    await templatesBtn.click();

    // Select the 'Code Review Pipeline' template
    const templateCard = page.locator('.template-picker-card').filter({ hasText: 'Code Review Pipeline' });
    const useTemplateBtn = templateCard.locator('button:has-text("Use Template")');
    await expect(useTemplateBtn).toBeVisible();
    await useTemplateBtn.evaluate((el) => (el as HTMLButtonElement).click());

    // Verify navigation to canvas page
    await expect(page).toHaveURL(/\/canvas\/[0-9a-f-]+/);
    const canvasUrl = page.url();
    const canvasIdMatch = canvasUrl.match(/\/canvas\/([0-9a-f-]+)/);
    const canvasId = canvasIdMatch ? canvasIdMatch[1] : '';
    expect(canvasId).not.toBe('');

    // Verify nodes from template are loaded (Supervisor, Developer, Reviewer)
    await expect(page.locator('.react-flow__node').filter({ hasText: 'Supervisor' }).first()).toBeVisible();
    await expect(page.locator('.react-flow__node').filter({ hasText: 'Developer' }).first()).toBeVisible();
    await expect(page.locator('.react-flow__node').filter({ hasText: 'Reviewer' }).first()).toBeVisible();

    // 4. Click Deploy to run
    await page.waitForTimeout(1000);
    const deployBtn = page.locator('button:has-text("Deploy")');
    await expect(deployBtn).toBeVisible();
    await expect(deployBtn).toBeEnabled();
    await deployBtn.click();

    // Wait for deployment status to show deployed (check status badge or text)
    const deployedBadge = page.locator('.canvas-toolbar-meta').filter({ hasText: 'deployed' });
    await expect(deployedBadge).toBeVisible({ timeout: 15000 });

    // 5. Navigate to Dashboard to verify terminal output (xterm) is visible
    await page.goto('/dashboard');
    
    // Wait for the Dashboard to load
    await expect(page.locator('h1', { hasText: 'Dashboard' })).toBeVisible();

    // Check what is rendered
    const emptyMsg = page.locator('.dashboard-empty');
    if (await emptyMsg.count() > 0) {
      console.log('BROWSER STATE: Found dashboard-empty messages:');
      for (let i = 0; i < await emptyMsg.count(); i++) {
        console.log(`- ${await emptyMsg.nth(i).textContent()}`);
      }
    }

    const errorCard = page.locator('.terminal-error-card');
    if (await errorCard.count() > 0) {
      console.log(`BROWSER STATE: Found terminal error: ${await errorCard.textContent()}`);
    }

    const terminalHost = page.locator('.terminal-host').first();
    await expect(terminalHost).toBeVisible();
    const xtermContainer = terminalHost.locator('.xterm');
    await expect(xtermContainer).toBeVisible();

    // 6. Go back to canvas page to test voice command. We pin the STT engine
    //    to 'webspeech' here so the SpeechRecognition polyfill installed in
    //    beforeEach is exercised on the next page load. (The app's default is
    //    'whisper', which requires real MediaRecorder + getUserMedia and would
    //    immediately error out under headless Chromium.)
    await page.evaluate(async () => {
      return new Promise<void>((resolve, reject) => {
        const req = indexedDB.open('AgentVerse', 1);
        req.onsuccess = () => {
          const db = req.result;
          if (!db.objectStoreNames.contains('settings')) {
            db.close();
            resolve();
            return;
          }
          const tx = db.transaction('settings', 'readwrite');
          const store = tx.objectStore('settings');
          const reqPut = store.put({ key: 'sttEngine', value: 'webspeech' });
          reqPut.onsuccess = () => resolve();
          reqPut.onerror = () => reject(reqPut.error);
        };
        req.onerror = () => reject(req.error);
      });
    });

    await page.goto(`/canvas/${canvasId}`);
    await expect(page.locator('.react-flow__node').filter({ hasText: 'Supervisor' }).first()).toBeVisible();

    // Open Mic modal via Ctrl+Shift+V keyboard shortcut
    await page.keyboard.press('Control+Shift+V');

    // Wait for the idle mic button (🎤) to be visible, to ensure modal is open
    const startMicBtn = page.locator('text=🎤');
    await expect(startMicBtn).toBeVisible();

    // Click 🎤 to drive the real voice pipeline. The SpeechRecognition polyfill
    // installed in beforeEach delivers a final transcript ('focus on supervisor')
    // through getSTTEngine() → VoiceCapture.onresult → setFinalTranscript(...).
    await startMicBtn.click();

    // Wait for voice panel overlay to be listening (containing the 🛑 button)
    const stopMicBtn = page.locator('text=🛑');
    await expect(stopMicBtn).toBeVisible();

    // Click stop mic to fire stopListening() → matchRuntimeCommand →
    // executeRuntimeCommand. With animations disabled, the pulse-mic is now
    // "stable" so a normal click (no force) passes Playwright's actionability
    // check.
    await stopMicBtn.click();

    // Expect to be navigated to the terminal output route
    await expect(page).toHaveURL(new RegExp(`/canvas/${canvasId}/terminal/term-.+`));
  });
});
