# tech-debt-keystore-validator-coverage — Implementation Tasks

> Owner: **IF** (Infra Dev — `src/api/`).
> Parallel-safe: only modifies `package.json`, `.github/workflows/`, `docs/`, and creates new files under `src/api/key-store/__tests__/contract/`. No conflicts with other tech-debt changes.

## 1. New Contract Test Directory (IF)

- [x] 1.1 Create `src/api/key-store/__tests__/contract/` directory
- [x] 1.2 Add `src/api/key-store/__tests__/contract/README.md` documenting:
  - The `KEYSTORE_LIVE=1` gate
  - Required env vars per provider (table)
  - How to skip when keys are unavailable
  - How to obtain disposable / sandbox keys for each provider
- [x] 1.3 Add `src/api/key-store/__tests__/contract/_helpers.ts` exposing:
  - `requireEnv(name: string): string | null` — returns value or `null`; emits `console.warn` and returns null when missing
  - `liveOrSkip(): void` — calls `it.skip` when `process.env.KEYSTORE_LIVE !== '1'`

## 2. Per-Provider Contract Tests (IF)

Each contract test reads its key from env, skips if absent or `KEYSTORE_LIVE` unset, calls the validator against the real endpoint, and asserts:
- `res.ok === true`
- `Array.isArray(res.models) && res.models.length > 0`
- `res.error === undefined`

- [x] 2.1 `openai.contract.ts` — env: `OPENAI_API_KEY`
- [x] 2.2 `anthropic.contract.ts` — env: `ANTHROPIC_API_KEY`
- [x] 2.3 `google.contract.ts` — env: `GOOGLE_API_KEY`
- [x] 2.4 `aws.contract.ts` — env: `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY`
- [x] 2.5 `azure.contract.ts` — env: `AZURE_OPENAI_ENDPOINT` + `AZURE_OPENAI_API_KEY`
- [x] 2.6 `moonshot.contract.ts` — env: `MOONSHOT_API_KEY`
- [x] 2.7 `copilot.contract.ts` — env: `GITHUB_COPILOT_TOKEN`
- [x] 2.8 `opencode.contract.ts` — env: `OPENCODE_ENDPOINT` + `OPENCODE_API_KEY`

## 3. Tooling (IF)

- [x] 3.1 Add `package.json` script:
  ```json
  "test:keystore-contract": "KEYSTORE_LIVE=1 vitest run --dir src/api/key-store/__tests__/contract"
  ```
- [x] 3.2 Add `.github/workflows/keystore-contract-nightly.yml`:
  - Trigger: `schedule` (cron `0 4 * * *`, UTC) + `workflow_dispatch`
  - Mirror the `contract-nightly.yml` pattern — install, run `npm run test:keystore-contract`, raise an issue on failure
  - Pull each provider key from a GitHub secret (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, …); the contract test skips when its env var is empty so a missing secret will simply skip that provider
- [x] 3.3 Author `docs/keystore-contract-tests.md` covering:
  - Purpose (catch silent provider drift; complement the deterministic MSW unit tests, do not replace them)
  - Env-var reference table (provider → vars → how to obtain)
  - How to run locally (`KEYSTORE_LIVE=1 OPENAI_API_KEY=… npm run test:keystore-contract`)
  - CI failure triage (which validator → which provider docs link)

## 4. Verify (IF)

- [x] 4.1 Default `npm test` is unchanged (contract dir lives outside the default vitest scope)
- [x] 4.2 Default `npm run test:keystore-contract` without `KEYSTORE_LIVE` skips every test (assertion: 0 failures, all skipped)
- [x] 4.3 With `KEYSTORE_LIVE=1` and a single env var set (e.g. only `OPENAI_API_KEY`), only the OpenAI test runs to completion; the others skip with a clear message
- [x] 4.4 Lint + typecheck pass

## Out of Scope

- Replacing or weakening the MSW unit tests in `src/api/key-store/__tests__/validators/`
- Token-leak fuzzing (already covered by `agentverse/no-secret-in-error` ESLint rule)
- Cost monitoring on contract runs (provider tokens are negligible; revisit in `finops-tier2`)
