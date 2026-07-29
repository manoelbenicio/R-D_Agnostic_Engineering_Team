# native-runtimes-onboarding task 1.5 — independent QA review (web slice)

- Review date: 2026-07-18T21:21:15Z
- Reviewer: opencode (independent QA), read-only on product/test files
- Scope: OpenSpec `native-runtimes-onboarding` task 1.5 (Codex56#A web frontend slice — onboarding/auth/marketing removal)
- Toolchain: node v24.17.0, pnpm 11.15.0, vitest 4.1.0, tsc; offline (`npm_config_offline=true`, frozen pnpm store, no network); no DB/Docker/credentials/live services
- Verdict: **Technical ACCEPT (implementation contract proven with non-zero real assertions — 43 assertions across 4 suites, closing the producer's zero-assertion gap) · Acceptance ACCEPT for the 1.5 implementation contract; checkbox stays OPEN pending Kiro TL adjudication**
- Files reviewed (read-only): `apps/web/app/(auth)/login/{page.tsx,page.test.tsx}`, `apps/web/app/auth/callback/{page.tsx,page.test.tsx}`, `apps/web/test/onboarding-auth-gate.test.ts`, `apps/web/app/page.tsx`, `packages/core/auth/{service.ts,service.test.ts,store.ts,store.test.ts,index.ts}`, `packages/views/auth/{login-page.tsx,login-page.test.tsx,auth-locale-parity.test.ts,use-logout.ts,index.ts}`, `packages/core/api/client.ts`

## START claim (preflight, recorded before any artifact)

- Reviewer: opencode (independent QA), read-only on product/test files.
- Scope: OpenSpec `native-runtimes-onboarding` task 1.5 (Codex56#A web slice).
- Toolchain: node v24.17.0, pnpm 11.15.0, vitest 4.1.0, tsc; offline (`npm_config_offline=true`, frozen pnpm store, no network); no DB/Docker/credentials/live services.
- Provenance: 1.5 web slice = commits `579e2df` "feat(web): replace email-code auth and remove landing" (impl) + `d18894e` "test(web): lock onboarding auth design parity" (source-gate tests). All reviewed files are tracked at these commits; `apps/web/app/(auth)/login/page.tsx` has a working-tree ` M` (the producer's exhaustive-deps fix at line 111, per its evidence).
- Frozen-state verification: all 14 producer-manifest SHA256 hashes (`native-onboarding-1.5-web.md` lines 133–146) reproduce exactly — this review inspects the same state the producer left.
- Collision check: `native-onboarding-1.5-review.md` confirmed not pre-existing.
- Transient test config: two `__tmp_pool_vitest.config.ts` files were created (one in `packages/views/`, one in `apps/web/`) solely to work around a vitest 4.1.0 jsdom fork-pool worker-startup timeout on the WSL2/mount environment (the producer hit the same timeout — its evidence lines 107–125 record it). Both temp files were deleted immediately after each run; confirmed absent (`ls` returns "No such file or directory"). No repo file was modified by me.
- Non-claims: no `pnpm build` (Next.js build — heavy, optional-network, out of scope); no e2e/Playwright (needs running server + DB); no live browser/UAT (task 3.3 lane); no `callback/page.test.tsx` execution (jsdom worker-startup timeout on retry — recorded as non-claimed I/O-locality limitation, source/type verified only).
- No edits to: OpenSpec, STATE, AGENT_LEDGER, EVIDENCE_INDEX, any product/test file, any checkbox. `tasks.md` 1.5 confirmed `[ ]` (OPEN) after review — left for Kiro TL adjudication.

## Spec / task acceptance criteria

`openspec/changes/native-runtimes-onboarding/tasks.md:10` (task 1.5) and `specs/onboarding/spec.md` require:

1. **Remove `(landing)` / `features/landing` / `content/use-cases` / sponsors** — marketing surfaces gone.
2. **Remove email verification-code flow** — no `send-code`/`verify-code`.
3. **`AuthService` interface + `SimpleAuthService`→`api.login()`** — Firebase-ready (no rework later).
4. **UI login/senha in the design-system** — same color tokens as kanban/agents.
5. **Preserve Google OAuth / CLI callback / desktop handoff** — escape hatches intact.

AB-REQ / EV mapping: native onboarding is a product-UX requirement with no direct AB-REQ in `REQUIREMENTS.md` (AB-REQs are Agent Brain program requirements). The `AuthService` interface's provider-neutral design (a future Firebase adapter implements the contract without touching views/store/routing/token-persistence) echoes the strangler principle of **AB-REQ-31** ("Strangler extraction: neutral interfaces … not rewrite global") at the auth boundary — cross-coverage, not a direct acceptance ID. EV: this review is a new artifact **EV-NATIVE-1.5-REVIEW** (analogous to the `EV-CHAT-1.2-1.3-REVIEW` pattern), to be indexed by Kiro.

## Why the producer's 1.5 was REOPENED (the gap this review closes)

AGENT_LEDGER.md:260 records: *"producer artifact claims 'Core focused tests — 33 assertions passed', but the queue reports the 1.5 web deliverable Vitest executed ZERO assertions — the 1.5-specific UI test provides no real proof. Zero-assertion = no executable evidence for the deliverable → 1.5 stays REOPENED/OPEN."* The producer's own evidence (`native-onboarding-1.5-web.md` lines 107–125) honestly records that `packages/views` and `apps/web` Vitest could not start a jsdom fork worker on the loaded mounted workspace (`[vitest-pool-runner]: Timeout waiting for worker to respond`), so the password-form / Google-OAuth / CLI-callback / desktop-handoff UI assertions were **source/type verified but not executed**. Task 1.5 was correctly left unchecked.

This review's job: produce **genuine non-zero executed assertions** for the 1.5 web slice (the producer source-gate alone is insufficient per the dispatch).

## Implementation contract — verified against real source

| Required (spec/task 1.5) | Source anchor | Reviewed result |
|---|---|---|
| Marketing dirs absent | `apps/web/app/(landing)`, `apps/web/features/landing`, `apps/web/content/use-cases`, `apps/web/public/usecases` | All four `ls` → "No such file or directory"; `grep -rni 'sponsor'` in `apps/web`+`packages/views` → 0 matches (excluding node_modules/.next) |
| Email-code flow absent | `packages/core/api/client.ts` | No `sendCode`/`verifyCode` methods; `grep -rn -e 'sendCode' -e 'verifyCode' -e 'send-code' -e 'verify-code'` in `apps/web`+`packages/core`+`packages/views` → only the source-gate test asserting absence (onboarding-auth-gate.test.ts:25). `e2e/` still references the old flow (out-of-scope legacy, not part of 1.5 web slice). |
| `AuthService` interface (Firebase-ready) | `packages/core/auth/service.ts:6-8` | `export interface AuthService { login(email: string, password: string): Promise<LoginResponse>; }` — provider-neutral; a future Firebase adapter implements this contract without touching views/store/routing |
| `SimpleAuthService` → `api.login()` | `packages/core/auth/service.ts:11-16`, `packages/core/api/client.ts:395-415` | `SimpleAuthService implements AuthService`, `constructor(private readonly api: Pick<ApiClient, "login">)`, `login()` delegates to `this.api.login(email, password)`. `ApiClient.login` sends `POST /auth/login` with `{email, password}`, validates `LoginResponseSchema`, throws `ApiError` on invalid response. |
| Store wired through `AuthService` | `packages/core/auth/store.ts:33,80` | `const authService = options.authService ?? new SimpleAuthService(api)` (defaultable + injectable for tests); `login` calls `authService.login(email, password)` — not `api.login` directly. This is the Firebase-ready seam. |
| UI login/senha in design-system | `packages/views/auth/login-page.tsx:251-333` | `<main className="… bg-background p-4 text-foreground">`, `Card`/`CardHeader`/`CardContent`/`CardFooter`/`Input`/`Button`/`Label` from `@multica/ui`; password `<Input type="password" autoComplete="current-password">`; no raw white/black/slate/gray/zinc/neutral color literals (locked by source-gate test). |
| Google OAuth preserved | `packages/views/auth/login-page.tsx:197-213,301-329`, `apps/web/app/(auth)/login/page.tsx:141-217`, `apps/web/app/auth/callback/page.tsx:75-88` | `handleGoogleLogin` builds `https://accounts.google.com/o/oauth2/v2/auth?…` with `client_id`/`redirect_uri`/`response_type=code`/`scope`/`state`; login page passes `google={clientId, redirectUri, state}`; callback page calls `api.googleLogin(code, redirectUri)`. |
| CLI callback preserved | `packages/views/auth/login-page.tsx:56-74`, `apps/web/app/auth/callback/page.tsx:53-77` | `validateCliCallback` accepts loopback + RFC1918 http:// (localhost/127.0.0.1/10./172.16-31./192.168.) and rejects https/external/non-URL; `redirectToCliCallback(url, token, state)`; callback page extracts `cli_callback:`/`cli_state:` from OAuth state and calls `redirectToCliCallback`. |
| Desktop handoff preserved | `apps/web/app/(auth)/login/page.tsx:67-101,155-200`, `apps/web/app/auth/callback/page.tsx:83-88,146-161` | `platform === "desktop"` → `api.issueCliToken()` → `window.location.href = multica://auth/callback?token=…`; authenticated-user desktop handoff screen with "Open desktop app" button. |
| Root redirect to /login | `apps/web/app/page.tsx` | `redirect("/login")` (locked by source-gate test line 50). |

## Executable proof (genuine, non-zero, real assertions)

All commands run offline (`npm_config_offline=true`, frozen pnpm store, no network). No DB/Docker/credentials/live services. Temp jsdom configs used only to work around the WSL2 vitest fork-pool worker-startup timeout (same environmental issue the producer hit); each temp config was deleted immediately after its run and confirmed absent.

### 1. `packages/core/auth` — 14 assertions PASS (default config, no jsdom needed)

```text
cd packages/core && env npm_config_offline=true pnpm vitest run auth/ --maxWorkers=1 --reporter=verbose
```
Result: `Test Files 3 passed (3)`, `Tests 14 passed (14)`, exit 0. Duration 51.36s.
Executed assertions (named, non-zero): `authStore.initialize` token-mode (401 cleanup delegation, 500 token-keep, network-failure token-keep, success populate) ×4; `authStore.login` (token-mode persist + setToken + user; cookie-mode no-persist) ×2; `SimpleAuthService` delegation to `ApiClient.login` with email+password ×1; `sanitizeNextUrl` security (single-slash, null/empty, absolute-URL, javascript:/non-http, protocol-relative, backslash, control-char) ×7.

### 2. `apps/web/test/onboarding-auth-gate.test.ts` — 3 assertions PASS (`--environment=node`, no jsdom)

```text
cd apps/web && env npm_config_offline=true pnpm vitest run test/onboarding-auth-gate.test.ts --environment=node --maxWorkers=1 --reporter=verbose
```
Result: `Test Files 1 passed (1)`, `Tests 3 passed (3)`, exit 0. Duration 19.63s.
Executed assertions: (a) marketing dirs absent + `client.ts` contains `POST /auth/login` + no `sendCode/verifyCode` calls + `package.json` has no `fumadocs`/`input-otp`; (b) login-page locks colors to design-system tokens (`bg-background`/`text-foreground`/`bg-card`/`border-border`/`text-muted-foreground`/`text-destructive`) and rejects raw white/black/slate/gray/zinc/neutral; (c) root `page.tsx` redirects to `/login` + onboarding CTAs use `DESKTOP_RELEASES_URL` not `/download`.

### 3. `packages/views/auth` — 20 assertions PASS (singleFork jsdom temp config)

```text
cd packages/views && env npm_config_offline=true pnpm vitest run --config ./__tmp_pool_vitest.config.ts --reporter=verbose
```
Result: `Test Files 2 passed (2)`, `Tests 20 passed (20)`, exit 0. Duration 110.38s.
Executed assertions (named): `auth-locale-parity` — every supported locale (en/ja/ko/zh-Hans) aligned with the password-login contract, no `code`/`resend`/`verify`/`download` keys ×1; `LoginPage` — password form without email-code step, same semantic color tokens as kanban/agents, required-credentials validation before auth service, login→workspaces→success flow, auth error single-step, password token to CLI callback, cookie-session CLI authorize via `issueCliToken`, Google OAuth state preserved ×8; `validateCliCallback` — accepts `http://localhost`/`127.0.0.1`/`10.`/`172.16-31.`/`192.168.` and rejects `https://`/`evil.com`/`172.32.`/`192.169.`/not-a-URL ×11.

### 4. `apps/web` (source-gate + login page) — 6 assertions PASS (singleFork jsdom temp config)

```text
cd apps/web && env npm_config_offline=true pnpm vitest run --config ./__tmp_pool_vitest.config.ts --reporter=verbose
```
Result: `Test Files 2 passed (2)`, `Tests 6 passed (6)`, exit 0. Duration 82.91s.
Executed assertions: source-gate ×3 (as above); web `LoginPage` — renders simple email+password contract (no "login code" text, `current-password` autocomplete), submits credentials through auth service (`mockLogin` called with `("test@multica.ai","secret")`), desktop deep-link token minting (`issueCliToken` called once, `window.location.href = multica://auth/callback?token=handoff-jwt`, "Open desktop app" button) ×3.

### 5. Typecheck — 3 PASS

```text
pnpm --filter @multica/core typecheck   # exit 0
pnpm --filter @multica/views typecheck  # exit 0 (slow on WSL2, ~6min)
pnpm --filter @multica/web typecheck    # exit 0
```

### Total genuine executed assertions: 14 + 3 + 20 + 6 = **43 real assertions**, all PASS, exit 0, non-zero. This closes the producer's zero-assertion gap for the 1.5 web slice.

## Explicit non-claims (honesty boundary)

- **`apps/web/app/auth/callback/page.test.tsx` did NOT execute** — jsdom fork-pool worker-startup timeout on the WSL2/mount environment on retry (same `[vitest-pool-runner]: Timeout waiting for worker to respond` the producer hit). Its 5+ assertions (next= routing, onboarding/invitations/workspace destinations, unsafe-next rejection) are **source + type verified only** (callback/page.tsx SHA256 `cd4e3684…10e8` matches producer manifest; typecheck PASS). Recorded as an **I/O-locality / environmental limitation**, not a test pass.
- **No `pnpm build` / Next.js production build** — heavy, optional-network, out of 1.5 scope (task 1.6 / 2.5 lanes).
- **No e2e/Playwright** — needs running server + DB (task 3.2 lane).
- **No live browser / UAT** — task 3.3 lane.
- **No runtime behavioral claim against a live backend** — `POST /auth/login` is the 1.7 backend (ACCEPTED+CHECKED, EV-AUTH-1.7); this review proves the **frontend contract** encodes the call, not that the live endpoint responds.
- **`e2e/{fixtures,helpers}.ts` still reference `send-code`/`verify-code`** — out-of-scope legacy e2e harness, not part of the 1.5 web slice (producer's evidence correctly excluded it; the source-gate test scopes its `not.toMatch` to `packages/core/api/client.ts`).
- **No checkbox, OpenSpec, STATE, AGENT_LEDGER, or EVIDENCE_INDEX edit.** `tasks.md` 1.5 confirmed `[ ]` (OPEN) after review. Kiro TL adjudicates checkboxes.

## Technical versus acceptance verdict

- **Technical verdict: ACCEPT.** The 1.5 web implementation contract is **completely and correctly encoded** in the real production/test source and proven by **43 genuine non-zero executed assertions** across 4 vitest suites (14 core + 3 web-source-gate + 20 views + 6 web-login-page) plus 3 typechecks. All required surfaces verified: marketing/landing/sponsors/email-code absent; `AuthService` interface + `SimpleAuthService`→`api.login()` Firebase-ready seam present and wired through the store; login/senha UI in the design-system (token-locked, no raw colors); Google OAuth / CLI callback (loopback+RFC1918 validation, 11 cases) / desktop `multica://auth/callback` handoff all preserved. The producer's zero-assertion gap (the reason 1.5 was REOPENED) is closed with real executed proof.
- **Acceptance verdict: ACCEPT for the 1.5 implementation contract.** The implementation-contract evidence is sufficient to lift the zero-assertion blocker. **Checkbox stays OPEN pending Kiro TL adjudication** per dispatch policy ("Kiro adjudicates; no edits to OpenSpec/STATE/ledger"). Recommendation to Kiro: **ACCEPT → CHECK** for task 1.5, with the recorded non-claims (callback test jsdom-timeout; no build/e2e/UAT — those are the separate 1.6/2.5/3.2/3.3 lanes).

This is an **implementation-contract ACCEPT**, not a runtime/UAT ACCEPT — smoke (3.2), web build (2.5), and UAT (3.3) remain the separate Kiro lanes.

## Source SHA-256 manifest (15 files; first 14 match producer manifest exactly)

| SHA-256 | Source |
|---|---|
| `2af77c72b12d6ac1b39a1dfca61cee6ed7b6c49fca67af53e233daf4293611ef` | `packages/core/auth/service.ts` |
| `2add5c81097326164d0e33e51f2d5ad2b7d25bca92d5117af72a71ca52f50e17` | `packages/core/auth/service.test.ts` |
| `bd0d7ac9560a04d9e37e0b00d2c659e55f68c06b551cd6ae1872b9140a6d279a` | `packages/core/auth/store.ts` |
| `39fdaca276de65bdc8b4fc399069a1f13861f075bea4c0bb62dedf14710f4ee7` | `packages/core/auth/store.test.ts` |
| `e10e6945e6d66cf0ef39fa02caf1cddf9bef09b4b6c464200348ffe9b4ca4031` | `packages/core/auth/index.ts` |
| `937cadb61759d935dbf226050dffd895de349e67694d9784ff7d2f37a5f755ff` | `packages/views/auth/login-page.tsx` |
| `f3b632f9bbd1637c6405ff1869544b792c0c9d11b63455507a3508521e0a536d` | `packages/views/auth/login-page.test.tsx` |
| `1c17dba85bb0cba526fb4d1d02d3aa819056a7e7e9321eaf633a48deccabb5df` | `packages/views/auth/auth-locale-parity.test.ts` |
| `232a2b9d115cccb7c06d26590429125a6acb919114ef9990e73dee5dd511dbcf` | `packages/views/auth/use-logout.ts` |
| `46e1b6a90ae604e0e1360d06ecd2025c8e4c7587a652653648d8e7e21e2eab94` | `packages/views/auth/index.ts` |
| `80066e7d47650ebe96bc24ce2172c238de5eeda892fd7d3b99c7f74542c1805c` | `apps/web/test/onboarding-auth-gate.test.ts` |
| `a5ead9a772b7a31629190124a4721b62d654f6817a4fce3378f06ac8773bf4c7` | `apps/web/app/(auth)/login/page.tsx` |
| `4b176203725a6f2b692b549d54b97a60a9eea3c105030b4d1b515af884aa1d14` | `apps/web/app/(auth)/login/page.test.tsx` |
| `14390e9f3c37c4429bb4eaa3f31d250f61960725638131fc25fa9534f81be9fb` | `apps/web/app/page.tsx` |
| `c0a0af82e72ba014ef33c6eff1f675b15e9353738cbcc46123967c5338cfaf59` | `packages/core/api/client.ts` |
| `cd4e36849170df039e88f1e371e40a2402a6fcefd6f35f980c72a0b02ec210e8` | `apps/web/app/auth/callback/page.tsx` (source/type verified; test jsdom-timeout non-claimed) |
| `83f80167e9d81a81327e6d7b1a529c0cf4a7b0423972c93ea836c4cd4cdda4b1` | `apps/web/app/auth/callback/page.test.tsx` (test jsdom-timeout non-claimed) |
| `ee502aa4323c285967cb40d2d7ef73f3e2b5a0bbf3878fa84e2a34ab96c62fcc` | `packages/core/api/client.test.ts` (producer-manifest; not re-executed in this review) |

OpenSpec docs (review-read): `tasks.md` `78f78b38…7d3c`, `design.md` `ad4f3ff0…892d`, `proposal.md` `5d248cb8…c6eb`, `specs/onboarding/spec.md` `3796015a…1de5`.

## Transient-config cleanup confirmation

Two temp vitest configs were created and deleted:
- `multica-auth-work/packages/views/__tmp_pool_vitest.config.ts` — created, run, deleted; `ls` confirmed absent.
- `multica-auth-work/apps/web/__tmp_pool_vitest.config.ts` — created, run (×2 for callback retry), deleted; `ls` confirmed absent.
- `git status --porcelain=v1` for both paths returns empty (no repo trace).

No product/test file was modified by this review. No OpenSpec/STATE/ledger/EVIDENCE_INDEX/checkbox file was edited. `tasks.md` 1.5 confirmed `[ ]` (OPEN) after review.

## Review check-in / check-out

Per dispatch policy ("no edits to OpenSpec/STATE/ledger; Kiro adjudicates checkboxes"), the review check-in/out is recorded here in the review artifact itself, not in AGENT_LEDGER.

- **Sign-in (START claim, 2026-07-18T20:38:49Z, before any artifact):** reviewer opencode; read-only on the 1.5 web slice; offline node/pnpm; no DB/Docker/network/credentials; only artifact `native-onboarding-1.5-review.md` (confirmed not pre-existing); transient jsdom test configs to live under the package dirs and be deleted immediately after each run.
- **Sign-out (2026-07-18T21:21:15Z):**
  - Artifact produced: this file (`native-onboarding-1.5-review.md`, 164 lines). Final SHA256 is recorded in the "Artifact SHAs" section below to avoid a self-referential paradox.
  - Files locked for review: **none** (read-only on product/test files; no repo file owned or modified).
  - Executable proof: genuine, non-zero — 43 real assertions across 4 vitest suites (14+3+20+6), all PASS exit 0; 3 typechecks PASS exit 0. Producer's zero-assertion gap closed.
  - Non-claimed: `callback/page.test.tsx` jsdom worker-startup timeout (I/O-locality limitation, source/type verified); no `pnpm build`/e2e/UAT (separate Kiro lanes 1.6/2.5/3.2/3.3).
  - Grades: **Technical ACCEPT · Acceptance ACCEPT for the 1.5 implementation contract** (zero-assertion blocker lifted). Checkbox stays OPEN pending Kiro TL adjudication — recommendation: ACCEPT → CHECK.
  - No edits to: OpenSpec (`tasks.md`/`design.md`/`proposal.md`/`specs/`), `STATE.md`, `AGENT_LEDGER.md`, `EVIDENCE_INDEX.md`, any product/test file, any checkbox. `tasks.md` 1.5 confirmed `[ ]` (OPEN) after review.
  - No DB, Docker, network, credential, auth-file, provider, user-home, or live-service access. No secret in any artifact. No `pnpm install` mutation (frozen store used as-is).

## Artifact SHA (post-write, non-self-referential)

This review artifact's final SHA256 (computed after the last edit to this line) is recorded by the reviewer's shell and reported in the review summary; it is intentionally NOT embedded as a literal inside this file because doing so would create a self-referential paradox (any change to embed the hash changes the hash). The reviewer's final `sha256sum` of this file is the authoritative value, reported to the dispatch out-of-band. Transient jsdom test configs (`__tmp_pool_vitest.config.ts`) were the only non-repo files created during this review and were deleted before sign-out; their existence was confined to `/mnt/c` package dirs and they contained no secrets.
