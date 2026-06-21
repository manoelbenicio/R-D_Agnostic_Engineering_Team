import { configDefaults, defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/__tests__/setup.ts'],
    css: true,
    testTimeout: 60000,        // evita timeouts em UNC path
    pool: 'forks',             // melhor isolamento que threads
    poolOptions: {
      forks: { singleFork: false },
    },
    clearMocks: true,          // limpa mocks entre testes
    restoreMocks: true,        // restaura spies entre testes
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      exclude: [
        'node_modules/**',
        'dist/**',
        'playwright/**',
        'src/main.tsx',
        'src/**/*.d.ts',
        'src/**/__tests__/**',
        'src/**/index.ts',
      ],
    },
    // Contract tests are gated behind GO_CORE_LIVE=1; exclude by default.
    exclude:
      process.env.GO_CORE_LIVE === '1'
        ? configDefaults.exclude
        : [...configDefaults.exclude, 'src/api/__tests__/contract/**', 'tests/e2e/**'],
  },
});
