# EV-G1-ADAPTERPREP — Runtime/CLI credential and endpoint surface

## Evidence status

- Evidence ID: `EV-G1-ADAPTERPREP`
- Agent: Codex 3 — Runtime/CLI Security
- Scope: G1 read-only preparation for OpenSpec task `1.5` input and tasks `5.1`–`5.10`
- Traceability: `AB-REQ-07`, `AB-REQ-16`–`AB-REQ-19`, `AB-REQ-21`, `AB-REQ-22`, `AB-REQ-34`
- Result: **G1 PREP CONTRACT COMPLETE**; runtime implementation and acceptance remain pending
- Not claimed: implementation, runtime verification, protocol acceptance, secret isolation acceptance, or route acceptance
- Inspection window: `2026-07-18T00:30:37Z`–`2026-07-18T00:35:10Z`; reconciled against the Codex 1 freeze at `2026-07-18T00:47:04Z`

This document records names and paths only. No secret value or secret-file content was read, copied, hashed, logged, or included in evidence.

## Provenance

- Host: `manoelneto-laptop`
- OS: `Linux 6.18.35.2-microsoft-standard-WSL2 x86_64 GNU/Linux`
- Repository commit: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- Worktree: dirty before this task; the pre-existing dirty baseline was preserved. None of the scoped Go sources was edited.
- Provider source set: 5 files; aggregate SHA-256 of the sorted `sha256sum` manifest: `cbdd1800dc6c75d411bb1cb3eb1c3cd36f69c8fcaaaa15784180bc02236d6a87`
- `execenv` source set: 40 `*.go` files, including tests; aggregate SHA-256 of the sorted `sha256sum` manifest: `ad85fe4c5770214a0e248503a681bd2b2351d1c4e8234279409594fc1c538805`
- OmniRoute supplier baseline referenced by the approved planning record: version `3.8.48`, mutable image tag `diegosouzapw/omniroute:latest`, port `20128`. A pinned image digest is still absent, so this evidence is not OmniRoute runtime acceptance evidence.

Scoped provider files:

- `multica-auth-work/server/pkg/agent/claude.go`
- `multica-auth-work/server/pkg/agent/codex.go`
- `multica-auth-work/server/pkg/agent/kimi.go`
- `multica-auth-work/server/pkg/agent/nim.go`
- `multica-auth-work/server/pkg/agent/antigravity.go`
- every `multica-auth-work/server/internal/daemon/execenv/*.go` file

## Frozen control-plane inputs consumed by this contract

Codex 1 froze the following neutral service configuration names in
`g1-codex1-contract-freeze.md` and `server/internal/daemon/brain/config.go` after the initial
adapter inspection:

- `AGENT_BRAIN_CONTROL_URL`
- `AGENT_BRAIN_GATEWAY_REQUIRED`
- `AGENT_BRAIN_GATEWAY_BASE_URL`
- `AGENT_BRAIN_GATEWAY_SECRET_FILE`
- `AGENT_BRAIN_GATEWAY_READINESS_POLICY`
- `AGENT_BRAIN_TASK_CAPACITY_TIER`
- `AGENT_BRAIN_LEGACY_EXECUTION_ENABLED`

These are trusted Agent Brain service inputs, not inherited/custom child overrides. In
gateway-required mode the Brain resolves and validates them first. The child-environment
builder then removes untrusted/provider-native inputs and applies only the adapter-specific
projection of the validated gateway profile last. In particular,
`AGENT_BRAIN_GATEWAY_SECRET_FILE` is a restricted service-side reference; its path and the
secret-file contents are not forwarded wholesale to a child CLI.

## Current injection surface

### 1. Shared child environment is permissive

`buildEnv` starts from `os.Environ()`, removes only Claude's internal process markers, then appends every key from `Config.Env`. Claude, Codex, Kimi, and Antigravity all use this builder. Therefore any provider credential or endpoint present in the daemon environment or agent custom environment reaches these children unless a gateway-required sanitizer removes it.

The existing filter removes `CLAUDECODE`, `CLAUDECODE_*`, `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_CODE_EXECPATH`, `CLAUDE_CODE_SESSION_ID`, and `CLAUDE_CODE_SSE_PORT`. It intentionally permits the public `CLAUDE_CODE_*` namespace, including the source-observed direct-provider selectors `CLAUDE_CODE_USE_BEDROCK` and `CLAUDE_CODE_USE_VERTEX`.

Source anchors: `claude.go:66`, `claude.go:653-695`, `codex.go:558`, `kimi.go:65-70`, `antigravity.go:78`.

### 2. Codex task homes import provider authentication and routing

Every Codex prepare/reuse path creates a per-task `CODEX_HOME`. The current preparer:

- resolves its source from the parent `CODEX_HOME`, otherwise `~/.codex`;
- copies source `auth.json` into task `auth.json`, even when no per-account home is selected;
- copies `config.json` and `config.toml` from the shared home;
- permits copied `config.toml` provider definitions containing `model_provider`, `[model_providers.*].base_url`, and `env_key`;
- symlinks `sessions/` and exposes a shared plugin cache. These are state surfaces, not provider-auth inputs, and are outside this credential/base-URL removal contract.

Gateway-required mode must not call the auth-copy branch and must not copy uncontrolled `config.json` or `config.toml`. It must generate a controlled task-local config and ensure task `auth.json` is absent.

Source anchors: `execenv.go:275-285`, `codex_home.go:18-24`, `codex_home.go:38-48`, `codex_home.go:59-102`, `codex_home.go:145-158`, `codex_home.go:279-320`.

### 3. NIM is a direct-provider HTTP path

The native NIM backend reads `NVIDIA_API_KEY` first from `Config.Env`, then from the daemon process environment. Its endpoint precedence is an injected backend URL, then `NIM_BASE_URL`, then the hard-coded direct URL `https://integrate.api.nvidia.com/v1`. `execenv` can also copy `<CredentialAccountHome>/NVIDIA_API_KEY` to `<envRoot>/nim-home/NVIDIA_API_KEY`, read the file, and inject its value as `NVIDIA_API_KEY`.

Gateway-required mode must disable the per-account preparation and direct fallback. If the temporary native NIM adapter is retained, both `NIM_BASE_URL` and the legacy `NVIDIA_API_KEY` slot must be overwritten last with trusted OmniRoute values; otherwise the native NIM path must be disabled.

Source anchors: `nim.go:20`, `nim.go:103-106`, `nim.go:271-285`, `execenv.go:340-348`, `execenv.go:651-655`, `nim_home.go:11-22`, `nim_home.go:31-65`.

### 4. Antigravity copies a complete native token directory

When an account home is selected, `execenv` recursively copies `<CredentialAccountHome>/.gemini/antigravity-cli/**` into the isolated task `HOME` and applies that `HOME` to the Agy child. With no account home, the shared builder leaves the parent `HOME` available, so native discovery remains possible.

The scoped source contains no proven Agy base-URL override. Gateway-required mode must not copy the token directory and must not launch the native Agy path until an accepted version proves a controlled endpoint. The approved interim route is Claude/Codex with an `agy/...` OmniRoute model.

Source anchors: `execenv.go:298-306`, `execenv.go:625-627`, `antigravity_home.go:10-17`, `antigravity_home.go:21-44`, `antigravity.go:78`.

### 5. Kimi upstream routing is opaque to the adapter

The Kimi adapter starts `kimi acp` with the same permissive environment and appends `KIMI_MODEL_THINKING_EFFORT` last. ACP controls the local agent session; this file does not configure the upstream model HTTP endpoint. The only upstream host visible in the scoped source is the diagnostic reference `api.kimi.com`.

Gateway-required mode must remove Kimi credential/endpoint settings and must not approve native Kimi until its installed provider-registry contract is proven. Until then it must use an accepted Claude/Codex frontend for the Kimi OmniRoute model route.

Source anchors: `kimi.go:58-70`, `kimi.go:86-89`.

### 6. Claude upstream routing is entirely environment/native-state driven here

The Claude adapter adds no controlled gateway authentication or endpoint. It receives the permissive inherited/custom environment. The scoped `execenv` code has no Claude-specific clean home or authentication-file preparer, so any Claude-native credential discovery reachable through inherited `HOME` remains outside current controls.

Gateway-required mode must start from a sanitized environment, prevent native provider login/discovery, and apply only the trusted Anthropic-compatible OmniRoute pair described below.

Source anchors: `claude.go:66`, `claude.go:653-695`.

### 7. Additional credential projection exists elsewhere in `execenv`

Although the assigned adapter files are Claude, Codex, Kimi, NIM, and Antigravity, the
required `execenv/*.go` inspection also found native credential projection for other legacy
providers:

- Kiro copies `<CredentialAccountHome>/kiro-cli/data.sqlite3` and projects the task copy
  through `XDG_DATA_HOME`; inherited `KIRO_API_KEY` is also an explicitly documented bypass.
- Cline recursively copies an account `.cline/` or `cline/` tree (including detected
  `data/settings/providers.json` or `settings/providers.json`) and projects it with
  `CLINE_DATA_DIR`, `CLINE_SANDBOX`, and `CLINE_SANDBOX_DATA_DIR`.
- OpenCode/GLM recursively copies account data/config roots, including OpenCode
  `auth.json` and provider configuration, and projects them through `XDG_DATA_HOME` and
  `XDG_CONFIG_HOME`.
- OpenClaw configuration discovery inherits `OPENCLAW_CONFIG_PATH`,
  `OPENCLAW_STATE_DIR`, `OPENCLAW_HOME`, and `OPENCLAW_INCLUDE_ROOTS`. The task wrapper can
  `$include` `~/.openclaw/openclaw.json` or write a resolved snapshot that may contain API
  keys/model-provider tokens; its `gateway.host`, `gateway.port`, and
  `gateway.auth.token` can also preserve a non-OmniRoute route.

Gateway-required mode must not invoke any of those credential projection branches. Because
none is in the frozen initial `CLIKind`/route release, admission must reject them unless and
until a controlled OmniRoute adapter is separately specified and accepted.

Source anchors: `execenv.go:287-338`, `execenv.go:610-658`, `kiro_home.go:11-53`,
`cline_home.go:10-82`, `opencode_home.go:10-85`, `openclaw_config.go:17-29`,
`openclaw_config.go:121-176`, `openclaw_config.go:235-272`, and
`openclaw_config.go:433-560`.

## Gateway-required deny/remove contract

The sanitizer must operate on inherited environment, custom environment, generated task homes/config, and provider-specific preparation. Exact keys listed below are mandatory, while family rules close the current arbitrary-`Config.Env` and future-variable gap.

### Environment variables

| Provider/surface | Remove or reject before trusted injection | Reason/disposition |
|---|---|---|
| Anthropic/Claude | `ANTHROPIC_API_KEY`, untrusted `ANTHROPIC_AUTH_TOKEN`, untrusted `ANTHROPIC_BASE_URL`, `ANTHROPIC_CUSTOM_HEADERS`; any `ANTHROPIC_*` credential/header/endpoint alias | Prevent direct Anthropic auth, endpoint, or header injection. The two trusted names are re-added last. |
| Claude cloud selectors | `CLAUDE_CODE_USE_BEDROCK`, `CLAUDE_CODE_USE_VERTEX`, and any other `CLAUDE_CODE_USE_*` provider selector; provider-auth variables usable by the selected cloud path | The current filter explicitly lets the first two through. Safe non-routing settings such as temp/output limits may remain only after allowlist review. |
| Claude internal markers | `CLAUDECODE`, `CLAUDECODE_*`, `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_CODE_EXECPATH`, `CLAUDE_CODE_SESSION_ID`, `CLAUDE_CODE_SSE_PORT` | Already stripped; retain this behavior, then add only controlled correlation inputs. |
| OpenAI/Codex | `OPENAI_API_KEY`, `OPENAI_API_KEYS`, untrusted `OPENAI_BASE_URL`, `CODEX_API_KEY`, untrusted `CODEX_HOME`; any `OPENAI_*`/`CODEX_*` credential, bearer, cookie, provider, or endpoint alias | Prevent built-in OpenAI login/routing and redirection to an arbitrary home. A controlled `CODEX_HOME` and dedicated OmniRoute key variable are re-added last. |
| Kimi | `KIMI_*` from inherited/custom input, including any key/token/base URL/provider-registry variable | Program interface register requires broad removal. A safe thinking-effort value may be restored from validated task policy only after routing is fixed. |
| NVIDIA/NIM | `NVIDIA_API_KEY`, untrusted `NIM_BASE_URL`, and any `NVIDIA_*`/`NIM_*` credential or endpoint alias | Prevent copied/direct NVIDIA auth and the direct NIM fallback. Trusted legacy-slot values may be re-added last only for the accepted transitional adapter. |
| Google/Gemini/Agy | `GOOGLE_API_KEY`, `GEMINI_API_KEY`, `GOOGLE_*_API_KEY`, and Google/Gemini/Agy credential, OAuth, cookie, base-URL, API-base, or endpoint aliases | Prevent Antigravity/Gemini direct-provider discovery. Native Agy remains disabled until its endpoint contract is proven. |
| Kiro legacy surface | `KIRO_API_KEY`; untrusted `XDG_DATA_HOME` when used for Kiro credential discovery; any `KIRO_*` credential, token, provider, or endpoint alias | `execenv` documents the API-key bypass and copied SQLite store. Kiro is not in the initial accepted gateway release. |
| Cline legacy surface | untrusted `CLINE_DATA_DIR`, `CLINE_SANDBOX_DATA_DIR`, and routing/auth-bearing `CLINE_*` values | Prevent discovery of copied `providers.json` and provider state. `CLINE_SANDBOX=1` is non-secret but may be restored only by an accepted controlled adapter. |
| OpenCode/GLM legacy surface | untrusted `XDG_DATA_HOME`, `XDG_CONFIG_HOME`, and provider/API-key/base-URL variables consumed by OpenCode-compatible runtimes | Prevent copied `auth.json` and provider-config discovery. OpenCode/GLM are not in the initial accepted gateway release. |
| OpenClaw legacy surface | `OPENCLAW_CONFIG_PATH`, `OPENCLAW_STATE_DIR`, `OPENCLAW_HOME`, `OPENCLAW_INCLUDE_ROOTS`, and any provider/gateway credential or endpoint variable used by config substitution | Prevent live inclusion or resolved copying of user provider keys, model-provider definitions, and a non-OmniRoute gateway. OpenClaw is not in the initial accepted gateway release. |
| Native discovery roots | Untrusted `HOME`, `XDG_DATA_HOME`, and `XDG_CONFIG_HOME` wherever they expose a user/provider credential or routing store | Claude, Kimi, Antigravity, Kiro, and OpenCode-compatible runtimes can discover native state through these roots. Replace them with a clean controlled root when required by an accepted adapter; do not inherit the user's root. |
| Generic auth | provider-scoped `*_ACCESS_TOKEN`, `*_REFRESH_TOKEN`, `*_AUTH_TOKEN`, `*_BEARER_TOKEN`, `*_OAUTH_*`, `*_COOKIE`, `*_COOKIES`, `*_COOKIE_FILE`, and authorization/custom-header injections | The credentialless spec forbids OAuth tokens, refresh tokens, cookies, and alternate auth headers regardless of provider spelling. |
| Generic routing | provider-scoped `*_BASE_URL`, `*_API_BASE`, `*_API_URL`, `*_ENDPOINT`, `*_HOST`, and provider-registry/config selectors | Required because arbitrary custom keys currently pass through and native clients can add aliases without changes in this repository. |

Family matching must be case-normalized for the target OS. A denied custom key must be rejected with a redacted policy reason; inherited denied keys may be removed with a redacted removal event. Values must never appear in logs or errors.

### Authentication and routing files/directories

| Provider | Current source-known path | Gateway-required rule |
|---|---|---|
| Codex | `<source CODEX_HOME>/auth.json` or `~/.codex/auth.json`; `<envRoot>/codex-home/auth.json` | Do not read/copy; remove a stale task copy before launch; pre-launch assertion requires absence. |
| Codex | `<source CODEX_HOME>/config.toml`, `<source CODEX_HOME>/config.json`; task copies under `<envRoot>/codex-home/` | Do not copy uncontrolled provider config. Generate controlled `config.toml`; remove all inherited `model_provider`, `[model_providers.*]`, `base_url`, `env_key`, bearer/header, and provider-auth definitions before writing the single OmniRoute provider. |
| NIM | `<CredentialAccountHome>/NVIDIA_API_KEY`; `<envRoot>/nim-home/NVIDIA_API_KEY` | Do not prepare or read in gateway-required mode; remove stale task copy. |
| Antigravity | `<CredentialAccountHome>/.gemini/antigravity-cli/**`; `<envRoot>/antigravity-home/.gemini/antigravity-cli/**` | Do not copy; use a clean task home if native Agy is ever accepted, otherwise disable native Agy. |
| Claude | Any provider-native auth/config reachable through inherited `HOME` (no exact path is encoded in the scoped source) | Do not expose the user home as an auth source. Use a clean/controlled home or a CLI-supported credentialless mode. Exact native store inventory is an acceptance prerequisite, not something this source snapshot proves. |
| Kimi | Any provider-native auth/provider registry reachable through inherited `HOME` (no exact path is encoded in the scoped source) | Do not expose the user home as an auth source. Native Kimi remains unapproved until its exact config/auth store and endpoint override are documented and sanitized. |
| Kiro | `<CredentialAccountHome>/kiro-cli/data.sqlite3`; task copy under `<envRoot>/kiro-data-home/kiro-cli/data.sqlite3`; inherited user `XDG_DATA_HOME/kiro-cli/data.sqlite3` | Do not copy or expose. Reject Kiro in gateway-required mode until a controlled adapter is accepted. |
| Cline | `<CredentialAccountHome>/.cline/**`, `<CredentialAccountHome>/cline/**`, or account roots detected by `data/settings/providers.json` / `settings/providers.json`; task copies under `<envRoot>/cline-data-dir/**` | Do not copy or expose through `CLINE_DATA_DIR`. Reject Cline until a controlled adapter is accepted. |
| OpenCode/GLM | account data roots `.local/share/opencode/**` or `opencode/**`, including `auth.json`; config roots `.config/opencode/**`, `config/opencode/**`, or `opencode-config/**`, including provider config; corresponding task XDG copies | Do not copy or expose through XDG variables. Reject these legacy frontends until a controlled adapter is accepted. |
| OpenClaw | active user config selected by `OPENCLAW_CONFIG_PATH` / `OPENCLAW_STATE_DIR` / `OPENCLAW_HOME` (normally `~/.openclaw/openclaw.json`); task `openclaw-config.json`; task `openclaw-user-snapshot.json`; any nested `$include` target | Do not include, resolve, or snapshot uncontrolled provider/gateway/auth definitions. Reject OpenClaw until a controlled adapter is accepted. |

The path-class rules for Claude and Kimi are intentionally broader than a guessed filename: the current repository delegates discovery to the installed CLI. Gateway-required mode must isolate the discovery root, not rely on a filename denylist that the CLI can change.

### Direct base URLs/endpoints

| Surface | Direct/untrusted destination to eliminate | Trusted target or decision |
|---|---|---|
| Claude | Any inherited/custom `ANTHROPIC_BASE_URL` and Claude's native provider default | `http://127.0.0.1:20128` on the current host/WSL topology; no trailing `/v1` |
| Codex | Any copied/custom provider `base_url`, built-in direct OpenAI route, or untrusted `OPENAI_BASE_URL` | Controlled provider `base_url=http://127.0.0.1:20128/v1`, `wire_api="responses"`, HTTP/SSE |
| Kimi | Native `api.kimi.com` route and any untrusted registry/base URL | No native target accepted yet; use an approved Claude/Codex frontend and exact Kimi OmniRoute model until proven |
| NIM | `https://integrate.api.nvidia.com/v1`; any untrusted `NIM_BASE_URL`; injected backend URL not derived from trusted gateway profile | `http://127.0.0.1:20128/v1` only if the transitional NIM adapter is accepted; otherwise disable native NIM |
| Antigravity | Native Agy destination and any token-home-mediated direct route | No native target accepted yet; use Claude/Codex `agy/...` fallback until a supported Agy endpoint override is proven |
| Kiro/Cline/OpenCode/GLM | Any endpoint resolved from copied provider state, config files, or inherited/custom variables | No accepted native gateway target in G1; fail closed |
| OpenClaw | Any user/snapshot `providers`, `gateway.host`/`gateway.port`, nested include, or env-substituted endpoint | No accepted OpenClaw gateway adapter in G1; fail closed |

Containerized execution must substitute a separately approved reachable Docker DNS/host-gateway endpoint. It must not reuse host loopback blindly.

## Trusted apply-last contract

The implementation order is binding:

1. Build a minimal safe inherited environment.
2. Remove all provider credentials, provider auth/header/cookie variables, direct endpoint variables, and unsafe routing selectors.
3. Add non-secret task/workspace/runtime inputs.
4. Validate and merge allowed custom settings; reject every denied name without logging its value.
5. Create controlled task-local CLI homes/config with no copied auth material.
6. Apply the following trusted adapter values last.
7. Assert immediately before launch that no denied variable, auth path, provider definition, or direct destination remains.

| Adapter | Trusted values applied last | Notes |
|---|---|---|
| Shared Brain policy | `AGENT_BRAIN_GATEWAY_REQUIRED`, `AGENT_BRAIN_GATEWAY_BASE_URL`, `AGENT_BRAIN_GATEWAY_SECRET_FILE`, `AGENT_BRAIN_GATEWAY_READINESS_POLICY`, `AGENT_BRAIN_TASK_CAPACITY_TIER`, and `AGENT_BRAIN_LEGACY_EXECUTION_ENABLED`; exact `CLIKind`, `RouteModel`, `RouterOwner=omniroute`; task/session/request correlation values | These exact neutral names are frozen. They are validated service inputs; only the minimum adapter projection is emitted to the child. `AGENT_BRAIN_CONTROL_URL` remains control-plane-only. |
| Claude Code | `ANTHROPIC_BASE_URL=<trusted OmniRoute root>`; `ANTHROPIC_AUTH_TOKEN=<stable OmniRoute key>` | Current host root is `http://127.0.0.1:20128`; remove both untrusted instances first. |
| Codex | `CODEX_HOME=<controlled task home>`; `<DEDICATED_CODEX_OMNIROUTE_KEY_ENV>=<stable OmniRoute key>` referenced by the generated provider `env_key`; controlled provider base URL `<trusted OmniRoute root>/v1`; controlled correlation headers `X-Session-Id` and `X-Request-Id` | The service-side secret reference is frozen as `AGENT_BRAIN_GATEWAY_SECRET_FILE`; the dedicated child key variable is intentionally still unnamed and must be frozen before implementation. It must not reuse `OPENAI_API_KEY`. `auth.json` must be absent. HTTP/SSE Responses only. |
| Transitional NIM | `NIM_BASE_URL=<trusted OmniRoute root>/v1`; `NVIDIA_API_KEY=<stable OmniRoute key>` | Reuse of the legacy NVIDIA slot is allowed only as the accepted transitional adapter; never source it from account-home preparation. |
| Kimi | validated `KIMI_MODEL_THINKING_EFFORT` may be restored after route policy | No trusted native auth/base-URL pair exists in the scoped source; native launch remains fail-closed pending provider-registry proof. |
| Native Agy | controlled clean `HOME` only after endpoint support is accepted | No trusted native auth/base-URL pair exists in the scoped source; use Claude/Codex fallback meanwhile. |

The stable OmniRoute key value comes only from the restricted service-secret mechanism owned by the operations stream. It is never stored in this document, a general config file, a copied provider home, logs, command arguments, diagnostics, or screenshots.

## Required implementation/acceptance assertions

- Gateway-required mode never calls provider-account auth preparation for Codex, NIM, Antigravity, Kiro, Cline, OpenCode, or GLM, and never resolves/includes the user's OpenClaw configuration.
- The child environment contains at most the one stable OmniRoute inference secret and no provider-native secret.
- Custom env cannot replace trusted base URLs, auth variables, provider config, transport, model route, router owner, or correlation.
- Codex `auth.json` is absent and controlled `config.toml` contains exactly one accepted OmniRoute provider plus non-provider sandbox/session/skill settings.
- No endpoint resolves to `integrate.api.nvidia.com`, `api.kimi.com`, a built-in direct Anthropic/OpenAI/Google destination, or an arbitrary custom/copied provider URL.
- Kimi and native Agy fail closed until their exact installed-version endpoint/auth contracts pass review; approved Claude/Codex model-route fallbacks remain available.
- Kiro, Cline, OpenCode/GLM, and OpenClaw fail closed in gateway-required mode until separately controlled adapters are specified and accepted.
- A redacted pre-launch diagnostic may report key names, path presence/absence, route class, and trusted host identity, but never values, tokens, cookies, headers, prompts, tool payloads, or repository content.
- Wave 3 evidence must inspect child environments, task homes, process trees, logs, and diagnostics and show zero provider-native credentials/auth files/direct endpoints (`OpenSpec 8.3`).

## Open blockers handed to G1/G2

1. The neutral service configuration and stable secret-file reference are frozen. Before Codex adapter implementation, Codex 1 and Codex 3 must freeze the dedicated, non-`OPENAI_API_KEY` child variable named by the generated Codex provider's `env_key`.
2. Kimi's installed provider registry, auth-store root, and endpoint override are not represented in the scoped adapter; native Kimi is not approved from this inspection.
3. Native Agy exposes no source-proven endpoint override; native Agy is not approved from this inspection.
4. Claude and Kimi native auth filenames are not encoded in `execenv`; clean discovery-root isolation is mandatory, and exact installed-version store inventory is still required for acceptance.
5. OmniRoute `3.8.48` is referenced through a mutable `latest` tag without a pinned digest. Protocol/runtime acceptance remains blocked on the supplier evidence gate.

## Reproduction commands

The following read-only command classes produced this evidence. Paths were restricted to source/planning artifacts; no `.env`, credential home, auth file, token directory, or secret source was opened.

```text
git status --short
git rev-parse HEAD
hostname
uname -srmo
rg --files pkg/agent internal/daemon/execenv
wc -l server/pkg/agent/{claude,codex,kimi,nim,antigravity}.go server/internal/daemon/execenv/*.go
rg -n '<environment/auth/base-url patterns>' server/pkg/agent/{claude,codex,kimi,nim,antigravity}.go server/internal/daemon/execenv/*.go
sed -n '<scoped line ranges>' <scoped Go source and approved planning artifacts>
nl -ba <scoped Go source> | sed -n '<scoped line ranges>'
sha256sum <five scoped provider files> | sha256sum
sha256sum <sorted execenv/*.go files> | sha256sum
openspec status --change build-omniroute-agent-brain --json
openspec instructions apply --change build-omniroute-agent-brain --json
```

No product code, central daemon/config/health entrypoint, secret, or secret file was modified.
