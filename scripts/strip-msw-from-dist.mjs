#!/usr/bin/env node
/**
 * Strip MSW (Mock Service Worker) artifacts from the production bundle.
 *
 * MSW configures `workerDirectory: ["public"]` in package.json, which copies
 * `mockServiceWorker.js` into `dist/` at build time. The runtime code in
 * `src/main.tsx` only registers the worker when `VITE_USE_MSW === 'true'`, so
 * the file is dormant in production — but per the v1 ship gate, no mock
 * infrastructure may ship in the production bundle. This script removes the
 * worker file from `dist/` after `vite build` completes.
 *
 * Invoked automatically via the `postbuild` npm script.
 */
import { existsSync, rmSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..');
const targets = [resolve(repoRoot, 'dist', 'mockServiceWorker.js')];

let removed = 0;
for (const target of targets) {
  if (existsSync(target)) {
    rmSync(target);
    console.log(`[strip-msw-from-dist] removed ${target.replace(repoRoot + '/', '')}`);
    removed += 1;
  }
}

if (removed === 0) {
  console.log('[strip-msw-from-dist] nothing to remove (already clean)');
}
