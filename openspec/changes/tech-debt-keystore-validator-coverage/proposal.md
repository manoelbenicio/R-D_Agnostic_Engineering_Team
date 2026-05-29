# tech-debt-keystore-validator-coverage

## Why

The eight provider validators in `src/api/key-store/validators/` now ship with MSW-mocked unit tests (`src/api/key-store/__tests__/validators/*.test.ts`). Coverage is healthy:

- Statements: 98.78%
- Branches:   73.43%
- Functions:  100%
- Lines:      98.78%

That satisfies the milestone-1 ≥70% gate for logic-heavy modules (task 22.1) and gives a deterministic CI signal. It does **not** prove the validators still talk to live providers — provider URLs, auth headers, and response shapes can drift silently while the mocked tests stay green.

## What Changes

Add an optional **contract test** layer that hits the real provider endpoints, gated behind a `KEYSTORE_LIVE=1` env flag mirroring the existing `CAO_LIVE=1` pattern (`src/api/__tests__/contract/`):

1. New script in `package.json`:
   ```
   "test:keystore-contract": "KEYSTORE_LIVE=1 vitest run --dir src/api/key-store/__tests__/contract"
   ```
2. New directory `src/api/key-store/__tests__/contract/` with one file per provider:
   ```
   anthropic.contract.ts
   aws.contract.ts
   azure.contract.ts
   copilot.contract.ts
   google.contract.ts
   moonshot.contract.ts
   openai.contract.ts
   opencode.contract.ts
   ```
3. Each contract test reads its key from env (`ANTHROPIC_API_KEY`, `AWS_ACCESS_KEY_ID`+`AWS_SECRET_ACCESS_KEY`, `AZURE_OPENAI_ENDPOINT`+`AZURE_OPENAI_API_KEY`, etc.), skips if absent, and calls the validator against the real endpoint asserting:
   - `res.ok === true`
   - `res.models.length > 0`
   - `res.error === undefined`
4. New nightly CI job runs the contract suite when secrets are present in the environment, similar to `.github/workflows/contract-nightly.yml`.
5. New doc `docs/keystore-contract-tests.md` listing the env vars per provider and how to obtain disposable keys.

## Impact

- **Affected:** `src/api/key-store/__tests__/contract/` (new), `package.json`, `.github/workflows/`, `docs/`.
- **Not affected:** existing MSW unit tests in `src/api/key-store/__tests__/validators/` — they stay as the deterministic CI gate. Validator source code stays unchanged.
- **CI:** default `npm test` cost unchanged. Nightly contract job adds 1 token per provider per run.
- **Risk:** flaky on provider degradation; mitigated by the `KEYSTORE_LIVE` gate (no impact on PR builds).

## Out of scope

- Replacing or weakening MSW unit tests.
- Token-leak fuzzing (already covered by ESLint rule `agentverse/no-secret-in-error`).
- Cost monitoring on contract runs.
