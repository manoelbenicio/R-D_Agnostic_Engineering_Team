# Keystore Contract Tests

Live provider contract tests that complement the deterministic MSW unit tests
for `src/api/key-store/validators/`.

## Purpose

The MSW unit tests in `src/api/key-store/__tests__/validators/*.test.ts` give
us a hermetic CI signal — every PR proves the eight validators still parse the
shapes we expect. They do **not** prove the validators still talk to live
providers: a provider can rename a header, change a URL, or shift response
fields silently while the mocked tests stay green.

The contract suite catches that drift. Each test:

1. Reads its key(s) from `process.env`.
2. Skips when `KEYSTORE_LIVE !== '1'` or any required env var is empty.
3. Calls the validator against the **real** provider endpoint via `appFetch`.
4. Asserts the validator returns `{ ok: true, models: string[] (length > 0) }`
   with no `error`.

## Env-var reference

| Provider | Env vars | Endpoint hit | How to obtain |
|----------|----------|--------------|---------------|
| OpenAI | `OPENAI_API_KEY` | `GET https://api.openai.com/v1/models` | https://platform.openai.com/api-keys (free trial / paid) |
| Anthropic | `ANTHROPIC_API_KEY` | `GET https://api.anthropic.com/v1/models` | https://console.anthropic.com/settings/keys |
| Google Gemini | `GOOGLE_API_KEY` | `GET https://generativelanguage.googleapis.com/v1beta/models` | https://aistudio.google.com/app/apikey (free tier available) |
| AWS | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | `POST https://sts.amazonaws.com/` (`GetCallerIdentity`) | Create a programmatic IAM user with **only** `sts:GetCallerIdentity` allowed |
| Azure OpenAI | `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_KEY` | `GET ${endpoint}/openai/models?api-version=2024-02-01` | Azure portal → Cognitive Services → keys & endpoint |
| Moonshot AI | `MOONSHOT_API_KEY` | `GET https://api.moonshot.cn/v1/models` | https://platform.moonshot.cn/console/api-keys |
| GitHub Copilot | `GITHUB_COPILOT_TOKEN` | `GET https://api.github.com/copilot_internal/v2/token` | A GitHub PAT or OAuth token from an account with Copilot access |
| OpenCode CLI | `OPENCODE_ENDPOINT`, `OPENCODE_API_KEY` | `GET ${endpoint}/models` | The deployed OpenCode instance's admin UI / config |

> **Disposable / sandbox tip:** for the cloud providers (OpenAI, Anthropic,
> Google, Moonshot) it is sufficient to use the cheapest model tier — the
> validators only call list-models endpoints, which never consume tokens.
> For AWS, attach an IAM policy with **only** `sts:GetCallerIdentity` to keep
> blast radius zero.

## Running locally

The script lives in `package.json`:

```jsonc
"test:keystore-contract": "KEYSTORE_LIVE=1 vitest run --dir src/api/key-store/__tests__/contract"
```

Skip every test (no secrets needed):

```bash
npm run test:keystore-contract
# All eight providers report as skipped (zero failures).
```

Run a single provider:

```bash
KEYSTORE_LIVE=1 OPENAI_API_KEY=sk-... npm run test:keystore-contract
# Only the OpenAI test executes; the other seven skip with a warning.
```

Run every provider:

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

## CI

`.github/workflows/keystore-contract-nightly.yml` runs the suite nightly at
04:00 UTC. Every provider key is wired to a GitHub secret of the same name.
Missing secrets cause the corresponding provider to skip — the job only fails
when a present key produces an unexpected response (true drift signal).

On failure the workflow opens a GitHub issue tagged `bug`, `contract-drift`,
`keystore`.

## CI failure triage

When the nightly job opens a `contract-drift` issue, walk the failed
provider's docs first — the validators below are intentionally thin and the
break is almost always a provider-side change.

| Provider | Validator | Provider docs |
|----------|-----------|---------------|
| OpenAI | `src/api/key-store/validators/openai.ts` | https://platform.openai.com/docs/api-reference/models/list |
| Anthropic | `src/api/key-store/validators/anthropic.ts` | https://docs.anthropic.com/en/api/models-list |
| Google Gemini | `src/api/key-store/validators/google.ts` | https://ai.google.dev/api/models#method:-models.list |
| AWS | `src/api/key-store/validators/aws.ts` | https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html |
| Azure OpenAI | `src/api/key-store/validators/azure.ts` | https://learn.microsoft.com/azure/ai-services/openai/reference#list-models |
| Moonshot AI | `src/api/key-store/validators/moonshot.ts` | https://platform.moonshot.cn/docs/api/chat |
| GitHub Copilot | `src/api/key-store/validators/copilot.ts` | https://docs.github.com/copilot |
| OpenCode CLI | `src/api/key-store/validators/opencode.ts` | https://github.com/sst/opencode |

After confirming the drift, open a follow-up change in
`openspec/changes/` patching the validator and add the new shape to the
matching MSW unit test under
`src/api/key-store/__tests__/validators/*.test.ts`.

## Out of scope

- Replacing or weakening the MSW unit tests — they stay as the deterministic
  PR gate.
- Token-leak fuzzing — already covered by the `agentverse/no-secret-in-error`
  ESLint rule.
- Cost monitoring on contract runs — the validators only call list-models
  endpoints; cost per nightly run is negligible. Revisit in `finops-tier2`.

## See also

- `src/api/key-store/__tests__/contract/README.md` — per-test quick reference
- `openspec/changes/tech-debt-keystore-validator-coverage/proposal.md`
- `openspec/changes/tech-debt-keystore-validator-coverage/tasks.md`