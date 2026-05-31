## 1. Durable dependency integrity (replace the band-aid)

- [ ] 1.1 From WSL/Linux, remove the mixed tree and reinstall from lockfile: `rm -rf node_modules && npm ci`
- [ ] 1.2 Confirm `node_modules/@rollup/` contains the linux binary and that no manual `--no-save` install is needed
- [ ] 1.3 Verify `npm run build` succeeds on the clean install (proves the lockfile is sufficient)

## 2. Preflight environment doctor

- [ ] 2.1 Add `scripts/check-dev-env.mjs` that resolves the rollup native binary for the current `process.platform`/`arch` and exits non-zero with remediation if missing
- [ ] 2.2 Wire it into `package.json` as `predev` and `prebuild` hooks
- [ ] 2.3 Manually verify: rename the linux binary dir → doctor fails with the exact fix message; restore → doctor passes

## 3. Vite configuration (formalize, document)

- [ ] 3.1 Keep `optimizeDeps.entries: ['index.html']` in `vite.config.ts` with a comment referencing this change
- [ ] 3.2 Keep `server.watch.usePolling` with a comment justifying it for mounted filesystems

## 4. Documentation

- [ ] 4.1 Create `docs/dev-environment.md`: canonical WSL/Linux workflow, the exact install/build/dev commands, the "never npm install from Windows" rule, and the move-to-WSL-FS long-term option
- [ ] 4.2 Link it from `README.md`

## 5. Zero-placeholder / mock guarantee (audit evidence)

- [ ] 5.1 After build, assert no `dist/assets/*.js` runtime file contains `setupWorker`/`msw/browser` and `dist/mockServiceWorker.js` is absent
- [ ] 5.2 Record the audit findings (done in design.md); confirm the two out-of-scope items (finops prices, ChatView TODO) are logged for separate changes, not fixed here

## 6. Verification (prove durability)

- [ ] 6.1 Clean-room proof: `rm -rf node_modules && npm ci && npm run build` green on Linux with no manual binary step
- [ ] 6.2 `npm test` (vitest) green
- [ ] 6.3 `npm run lint` and `npm run typecheck` green
- [ ] 6.4 Confirm the doctor blocks a simulated cross-platform corruption and passes when healthy
