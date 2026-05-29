# Keystore Live Contract Tests

This directory holds **live** provider contract tests for the eight key-store
validators. They complement (not replace) the deterministic MSW unit tests in
`../validators/`.

| Layer | Location | Purpose | Runs in CI |
|-------|----------|---------|------------|
| Unit (MSW) | `../validators/*.test.ts` | Deterministic shape + happy/sad path coverage of each validator | Every PR |
| **Live contract** (this dir) | `*.contract.test.ts` | Catch silent provider drift — URLs, headers, response shape | Nightly only |

## The `KEYSTORE_LIVE` gate

Every test in this directory is wrapped in `liveOrSkip()` from `./_helpers.ts`.
The wrapper resolves to `it` only when **both** of the following are true:

1. `process.env.KEYSTORE_LIVE === '1'`
2. The provider's required env vars (table below) are non-empty

Otherwise the test is registered with `it.skip` and reported as skipped — zero
failures, regardless of which secrets are missing. This is what makes the suite
safe to run in any environment.

## Required env vars per provider

| File | Provider | Env vars |
|------|----------|----------|
| `openai.contract.test.ts` | OpenAI | `OPENAI_API_KEY` |
| `anthropic.contract.test.ts` | Anthropic | `ANTHROPIC_API_KEY` |
| `google.contract.test.ts` | Google Gemini | `GOOGLE_API_KEY` |
| `aws.contract.test.ts` | AWS (STS GetCallerIdentity) | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| `azure.contract.test.ts` | Azure OpenAI | `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_KEY` |
| `moonshot.contract.test.ts` | Moonshot AI | `MOONSHOT_API_KEY` |
| `copilot.contract.test.ts` | GitHub Copilot | `GITHUB_COPILOT_TOKEN` |
| `opencode.contract.test.ts` | OpenCode CLI | `OPENCODE_ENDPOINT`, `OPENCODE_API_KEY` |

## Running locally

Skip every test (no secrets needed):

```bash
npm run test:keystore-contract
```

Run a single provider:

```bash
KEYSTORE_LIVE=1 OPENAI_API_KEY=sk-... npm run test:keystore-contract
```

Run all providers (rare; usually CI does this):

```bash
KEYSTORE_LIVE=1 \
  OPENAI_API_KEY=... \
  ANTHROPIC_API_KEY=... \
  GOOGLE_API_KEY=... \
  AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... \
  AZURE_OPENAI_ENDPOINT=... AZURE_OPENAI_API_KEY=... \
  MOONSHOT_API_KEY=... \
  GITHUB_COPILOT_TOKEN=... \
  OPENCODE_ENDPOINT=... OPENCODE_API_KEY=... \
  npm run test:keystore-contract
```

## Obtaining disposable keys

See `docs/keystore-contract-tests.md` for the full triage + key-acquisition
guide (free tiers, sandbox endpoints, console URLs).

## Why a separate `*.contract.test.ts` extension

The MSW server in `src/__tests__/setup.ts` rejects unhandled requests with
`onUnhandledRequest: 'error'`, which would otherwise block live provider
calls. The `passthroughMsw()` helper in `_helpers.ts` re-listens with
`onUnhandledRequest: 'bypass'` — but only when `KEYSTORE_LIVE=1`. This means
default `npm test` keeps the strict MSW boundary; tests in this directory
self-skip and never touch the network.

## See also

- Proposal: `openspec/changes/tech-debt-keystore-validator-coverage/proposal.md`
- Tasks:    `openspec/changes/tech-debt-keystore-validator-coverage/tasks.md`
- Doc:      `docs/keystore-contract-tests.md`
- Workflow: `.github/workflows/keystore-contract-nightly.yml`
