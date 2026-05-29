import { defineConfig, devices } from '@playwright/test';

const PORT = 5173;

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: `http://localhost:${PORT}`,
    trace: 'on-first-retry',
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    port: 5173,
    env: {
      VITE_USE_MSW: 'true',
      VITE_CAO_BASE_URL: 'http://127.0.0.1:9889',
      VITE_ALLOW_CANVAS2D: 'true',
    },
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});
