import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

// Per cao-integration/spec.md and tasks 1.1, dev server MUST be 5173 to match
// CAO_CORS_ORIGINS / CAO_WS_ALLOWED_CLIENTS.
export default defineConfig({
  plugins: [react()],
  // Scan only the real SPA entry. Without this, Vite globs every *.html in the
  // repo (snake-game/, docs/, data_expert_skills/, playwright-report/, …) as
  // entries and dependency optimization fails.
  optimizeDeps: {
    entries: ['index.html'],
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    // Canonical dev runs natively (Windows) with reliable native file watching.
    // Polling is only needed when serving from a mounted filesystem (e.g. WSL on
    // /mnt/c), where inotify events don't fire. Opt in with VITE_WATCH_POLLING=true.
    // See docs/dev-environment.md.
    watch:
      process.env.VITE_WATCH_POLLING === 'true'
        ? { usePolling: true, interval: 300 }
        : undefined,
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
