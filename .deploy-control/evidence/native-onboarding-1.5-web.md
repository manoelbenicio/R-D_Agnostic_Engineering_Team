# Native runtimes onboarding task 1.5 — web frontend evidence

- Change: `native-runtimes-onboarding`
- Task: `1.5` Agent-5 frontend-auth/marketing removal
- Check-in: `.deploy-control/Codex-Agent-5__NATIVE-ONBOARDING-1.5-WEB__20260718T200604Z.md`
- Evidence captured: 2026-07-18T20:34:35Z
- Result: **BLOCKED (verification residual; OpenSpec task remains unchecked)**
- Network, credentials, DB, live services, Docker, and backend: not used or changed

## Current implementation verified on disk

- `apps/web/app/page.tsx` redirects `/` to `/login`.
- `apps/web/app/(landing)`, `apps/web/features/landing`, and
  `apps/web/content/use-cases` do not exist. The web static gate also rejects
  sponsor/marketing and email-code surfaces.
- `AuthService` and `SimpleAuthService` exist in `packages/core/auth/service.ts`;
  the simple implementation delegates exactly to `api.login(email, password)`.
- `ApiClient.login` sends `POST /auth/login` with email/password.
- `packages/views/auth/login-page.tsx` renders the design-system password form
  and delegates through the auth store/service boundary.
- Google OAuth construction, validated CLI callbacks/token return, the OAuth
  callback page, and the authenticated desktop `multica://auth/callback`
  handoff remain present.
- No task-1.6 design-token, color-parity, i18n-cleanup, broad build harness, or
  QA file was edited. Existing mobile work was not touched.

## Product edit in this pass

- `apps/web/app/(auth)/login/page.tsx`: added the used translation function
  `t` to the desktop-handoff effect dependency list, resolving the focused
  `react-hooks/exhaustive-deps` warning.
- The pre-existing staged model-discovery change in
  `packages/core/api/client.ts` was preserved and not edited by this pass.

## Executed evidence

### Core focused tests — PASS, assertions executed

```text
pnpm --filter @multica/core exec vitest run api/client.test.ts auth/service.test.ts auth/store.test.ts --maxWorkers=1
Test Files  3 passed (3)
Tests       33 passed (33)
exit 0
```

These assertions cover the login request/response contract, the
`SimpleAuthService` delegation, successful store state/token behavior, and
failed-login state preservation.

### TypeScript — PASS

```text
pnpm --filter @multica/core typecheck       # exit 0
pnpm --filter @multica/views typecheck      # exit 0
pnpm --filter @multica/web typecheck        # exit 0
```

After the one-line effect-dependency fix, web lint and typecheck were rerun as
one `eslint ... && pnpm typecheck` command and exited 0.

### Focused lint — PASS after one owned fix

The first root-level ESLint command was invalid and is not evidence: it exited
2 because the monorepo has workspace-local flat configs. Corrected commands:

```text
packages/core: pnpm exec eslint auth api/client.ts api/client.test.ts --fix=false
exit 0, no findings

packages/views: pnpm exec eslint auth --fix=false
exit 0, no findings

apps/web: pnpm exec eslint <owned auth/root/callback/static-gate files> --fix=false
initial: exit 0 with one react-hooks/exhaustive-deps warning at login/page.tsx:111
post-fix targeted login lint plus pnpm typecheck: exit 0, no output
```

### Deterministic source gate — PASS, 15 assertions executed

The shell verifier used `test` and `rg -q` assertions for:

1. all three removed marketing directories absent;
2. root redirect to `/login`;
3. `AuthService` interface;
4. `SimpleAuthService implements AuthService`;
5. exact service delegation to `api.login(email, password)`;
6. `/auth/login` API path;
7. password input type;
8. `current-password` autocomplete;
9. auth-store login call;
10. no `sendCode`/`verifyCode` calls in owned production scope;
11. Google OAuth URL preserved;
12. CLI callback validator preserved;
13. CLI callback redirect preserved;
14. desktop CLI-token exchange preserved;
15. desktop deep-link and OAuth callback CLI state preserved.

Result: `PASS: 15 deterministic onboarding/auth source assertions`, exit 0.

### Diff checks — PASS

```text
git diff --check -- 'apps/web/app/(auth)/login/page.tsx'
exit 0
```

## Verification residual — assertions did not execute

The focused views and web Vitest files could not start a worker on the loaded
mounted workspace. This is **not** recorded as a test pass:

```text
packages/views login-page.test.tsx + auth-locale-parity.test.ts
fork pool: no tests; 2 worker-start timeout errors; exit 1
single-thread pool retry: no tests; 2 worker-start timeout errors; exit 1
single locale-file retry: no tests; 1 worker-start timeout error; exit 1

apps/web onboarding-auth-gate.test.ts + login/page.test.tsx + callback/page.test.tsx
no tests; 3 worker-start timeout errors; exit 1
```

Vitest reported `[vitest-pool-runner]: Timeout waiting for worker to respond`
before transform/import/tests. Therefore the password form, Google OAuth, CLI
callback, and desktop handoff UI assertions are source/type verified but not
currently executed by Vitest. Task 1.5 is deliberately left unchecked.

No broad web build or UAT is claimed; those remain task 1.6 / Wave 2-3 owner
lanes.

## Source SHA-256 manifest

```text
14390e9f3c37c4429bb4eaa3f31d250f61960725638131fc25fa9534f81be9fb  apps/web/app/page.tsx
a5ead9a772b7a31629190124a4721b62d654f6817a4fce3378f06ac8773bf4c7  apps/web/app/(auth)/login/page.tsx
4b176203725a6f2b692b549d54b97a60a9eea3c105030b4d1b515af884aa1d14  apps/web/app/(auth)/login/page.test.tsx
cd4e36849170df039e88f1e371e40a2402a6fcefd6f35f980c72a0b02ec210e8  apps/web/app/auth/callback/page.tsx
83f80167e9d81a81327e6d7b1a529c0cf4a7b0423972c93ea836c4cd4cdda4b1  apps/web/app/auth/callback/page.test.tsx
80066e7d47650ebe96bc24ce2172c238de5eeda892fd7d3b99c7f74542c1805c  apps/web/test/onboarding-auth-gate.test.ts
937cadb61759d935dbf226050dffd895de349e67694d9784ff7d2f37a5f755ff  packages/views/auth/login-page.tsx
f3b632f9bbd1637c6405ff1869544b792c0c9d11b63455507a3508521e0a536d  packages/views/auth/login-page.test.tsx
2af77c72b12d6ac1b39a1dfca61cee6ed7b6c49fca67af53e233daf4293611ef  packages/core/auth/service.ts
2add5c81097326164d0e33e51f2d5ad2b7d25bca92d5117af72a71ca52f50e17  packages/core/auth/service.test.ts
bd0d7ac9560a04d9e37e0b00d2c659e55f68c06b551cd6ae1872b9140a6d279a  packages/core/auth/store.ts
39fdaca276de65bdc8b4fc399069a1f13861f075bea4c0bb62dedf14710f4ee7  packages/core/auth/store.test.ts
c0a0af82e72ba014ef33c6eff1f675b15e9353738cbcc46123967c5338cfaf59  packages/core/api/client.ts
ee502aa4323c285967cb40d2d7ef73f3e2b5a0bbf3878fa84e2a34ab96c62fcc  packages/core/api/client.test.ts
```
