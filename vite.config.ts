import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

// Per cao-integration/spec.md and tasks 1.1, dev server MUST be 5173 to match
// CAO_CORS_ORIGINS / CAO_WS_ALLOWED_CLIENTS.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
  },
  preview: {
    port: 4173,
    strictPort: true,
  },
  build: {
    sourcemap: true,
    target: 'es2022',
    rollupOptions: {
      output: {
        manualChunks: {
          // Keep heavy libs in their own chunks so the main bundle stays slim
          // and contributes to the 21.3 bundle-size budget.
          xterm: ['@xterm/xterm', '@xterm/addon-fit', '@xterm/addon-webgl', '@xterm/addon-search', '@xterm/addon-unicode11', '@xterm/addon-web-links'],
          flow: ['@xyflow/react'],
          monaco: ['@monaco-editor/react'],
          recharts: ['recharts'],
        },
      },
    },
  },
});
