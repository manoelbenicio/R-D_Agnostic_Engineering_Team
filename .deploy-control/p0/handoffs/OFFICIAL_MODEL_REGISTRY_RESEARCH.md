# Official model identity and OmniRoute mapping research

- Research owner: Principal/Kiro (management/research only)
- Access date: **2026-07-22**
- Scope: active P0 GLM, Kimi, Claude Opus/AWS Bedrock, plus the NVIDIA GLM fallback required by the frozen route matrix
- Product-code effect: **none**
- Rule: an official commercial name, a manufacturer/provider API ID, and an internal OmniRoute `RouteModel` are different identifiers. A public name or API ID is never promoted into an OmniRoute namespace without registry evidence.

## Executive result

| Manufacturer/provider | Official commercial/model name | Official API/provider ID | Current repository/route assumption | OmniRoute `RouteModel` status | Required correction/action |
|---|---|---|---|---|---|
| Z.ai | **GLM-5.2** | Z.ai API: **`glm-5.2`**; official weight repository: `zai-org/GLM-5.2` | `cp/cline-pass/glm-5.2` primary; `nvidia/z-ai/glm-5.2` fallback | **Confirmed by OmniRoute v3.8.48 source.** ClinePass declares alias `cp` and model `cline-pass/glm-5.2`; the catalog constructs `alias/model.id`. | No product correction to the primary literal. Keep `BLK-AVAIL` until an active runtime exposes and validates the row. |
| NVIDIA NIM (GLM fallback) | **GLM-5.2** by Z.ai | NVIDIA NIM catalog identity: **`z-ai/glm-5.2`** | `nvidia/z-ai/glm-5.2` | **Confirmed by OmniRoute v3.8.48 source.** NVIDIA declares alias `nvidia` and model `z-ai/glm-5.2`. The leading namespace remains OmniRoute-owned. | No identity correction. Keep fallback non-Brain-selectable and retain pool/conformance gates. |
| Moonshot AI/Kimi | **Kimi K2.7 Code**; HighSpeed is the same model served faster | **`kimi-k2.7-code`**; **`kimi-k2.7-code-highspeed`** | Product catalog lacks the `cp/` prefix; `cline-kimi-k2.7-dedicated` and `kimi-sub` are separate combos | **Exact route confirmed by OmniRoute v3.8.48 source:** `cp/cline-pass/kimi-k2.7-code` (OpenAI-format ClinePass). | The exact-ID part of `BLK-KIMI` is resolved. Product correction remains unauthorized; runtime availability/capability and the authorized live run remain blocked. |
| Anthropic | **Claude Opus 4.8** | Claude API: **`claude-opus-4-8`** | Opus48 semantic intent | **Two non-AWS routes confirmed in OmniRoute v3.8.48:** `cc/claude-opus-4-8` (Claude OAuth) and `anthropic/claude-opus-4-8` (Anthropic API key). | Do not use either as proof of the requested AWS route. They resolve generic Anthropic mapping only. |
| AWS Bedrock / Anthropic | **Claude Opus 4.8** | Base: **`anthropic.claude-opus-4-8`**; geo: **`us.`**, **`eu.`**, **`jp.`**, **`au.`** + base; global: **`global.anthropic.claude-opus-4-8`** | “Kiro → Opus48 from AWS” | **Route construction proven; target literal unresolved.** OmniRoute discovers AWS foundation-model/profile IDs, persists them verbatim, and publishes `bedrock/<literal-id>`. Neither v3.8.48 nor current `main` statically lists Opus 4.8. | Require the literal `modelId`/`inferenceProfileId` returned for the OmniRoute account/region and its active `/v1/models` row. Current read-only AWS calls were IAM-denied; no runtime endpoint was reachable. Never infer among base/geo/global IDs. |

## 1. Z.ai — GLM-5.2

### Official identity, release, and API

Z.ai’s release notes list **GLM-5.2** on **2026-06-16**. The official model guide uses `model: "glm-5.2"` with `POST https://api.z.ai/api/paas/v4/chat/completions`; its OpenAI-compatible SDK example uses base URL `https://api.z.ai/api/paas/v4/`.

Official limits and behavior:

- text input and text output;
- **1M-token context** and **128K maximum output**;
- thinking modes/effort, streaming, function calling, context caching, structured output, and MCP integration;
- long-horizon/project-scale coding and agentic execution are the stated positioning.

### Published architecture/design

The official `zai-org/GLM-5.2` model card describes:

- **IndexShare**, reusing one indexer across every four sparse-attention layers;
- a stated **2.9× reduction in per-token FLOPs at 1M context**;
- an improved MTP layer for speculative decoding, with acceptance length improved by up to 20%;
- MIT-licensed open weights.

The Z.ai pages checked do not state a total parameter count. NVIDIA’s provider model card separately describes the served model as a 753B MoE with GLM DSA/IndexShare; that is recorded as an NVIDIA provider disclosure, not silently reattributed to Z.ai.

### Repository and OmniRoute comparison

- Manufacturer API ID: `glm-5.2`.
- Official weight-repository ID: `zai-org/GLM-5.2`.
- NVIDIA provider identity: `z-ai/glm-5.2`.
- Internal primary route: `cp/cline-pass/glm-5.2`.
- Internal fallback route: `nvidia/z-ai/glm-5.2`.

The OpenSpec checklist lists `cp/cline-pass/glm-5.2`, and the official OmniRoute v3.8.48 ClinePass registry independently declares `alias: "cp"` plus model ID `cline-pass/glm-5.2`. The v3.8.48 catalog constructs the public row as `${alias}/${model.id}`, so this is source-backed rather than inferred. Public Z.ai sources still certify only the upstream identity, not the OmniRoute namespace.

**Disposition:** GLM upstream and internal identities are resolved. `BLK-AVAIL` remains because `/v1/models` emits provider rows only for active, model-eligible connections, and no local OmniRoute runtime was reachable to prove availability, pool behavior, or end-to-end conformance.

## 2. Moonshot AI — Kimi K2.7 Code

### Official identity and API

Moonshot’s current model list names:

- `kimi-k2.7-code` — Kimi’s dedicated coding model, 256K context;
- `kimi-k2.7-code-highspeed` — the same model with higher serving speed.

The official guide uses the commercial name **Kimi K2.7 Code**, an OpenAI-compatible client with base URL `https://api.moonshot.ai/v1`, and `model="kimi-k2.7-code"`. The official pages checked do not provide a separate release date; they currently list both IDs and do not place them in the deprecated-model section. Kimi K3 is newer, but K2.7 Code remains listed.

Official behavior and constraints:

- **256K context**;
- image/video plus text input in the official API;
- thinking is always on and cannot be disabled;
- Preserved Thinking is always on; prior `reasoning_content` must be retained through multi-step interactions;
- multi-step tool invocation is supported; `tool_choice` accepts `auto` or `none`, not `required`;
- fixed sampling constraints include temperature 1.0, top-p 0.95, and `n=1`;
- OpenAI-compatible Chat Completions API.

### Published architecture/design

The official Moonshot model card discloses:

- Mixture-of-Experts, **1T total parameters / 32B activated**;
- 61 layers, 384 experts, 8 selected experts per token, and 1 shared expert;
- MLA attention and SwiGLU activation;
- 160K vocabulary and 256K context;
- MoonViT vision encoder with 400M parameters;
- Modified MIT license.

### Repository and OmniRoute comparison

The manufacturer has now resolved the upstream model identity: **`kimi-k2.7-code`**, not a generic API ID `kimi-k2.7`.

OmniRoute v3.8.48 resolves the route directly in source, without textual inference:

- provider ID `clinepass`; public alias `cp`; format `openai`; upstream endpoint `https://api.cline.bot/api/v1/chat/completions`;
- registry model ID `cline-pass/kimi-k2.7-code`;
- catalog composition `${alias}/${model.id}`;
- exact public `RouteModel`: **`cp/cline-pass/kimi-k2.7-code`**.

This does not conflate the other local names:

- `cline-pass/kimi-k2.7-code` without `cp/` is the current Multica-side catalog literal and is not the complete public row;
- `cline-kimi-k2.7-dedicated` and `kimi-sub` are runtime combo names, whose targets live in the installation database;
- `claude_code_kimi_2.7_Code` is a Claude Code/Anthropic-Messages combo family, not ClinePass.

The same v3.8.48 shared registry also lists direct Moonshot rows `moonshot/kimi-k2.7-code` and `moonshot/kimi-k2.7-code-highspeed`, plus Kimi Coding rows under `kmc/` and `kmca/`. Those are different provider routes and do not replace the required ClinePass route.

**Disposition:** the exact-ID component of **`BLK-KIMI` is resolved** as `cp/cline-pass/kimi-k2.7-code`. This finding authorizes no product edit and no inference run. The remaining Kimi gates are the Principal-controlled product correction, an active `/v1/models` availability/capability row, pool/fallback verification, `BLK-25B`, and `live_runs.cline_kimi.authorized=true`.

## 3. Anthropic — Claude Opus 4.8

### Official identity, release, and API

Anthropic announced **Claude Opus 4.8** on **2026-05-28** and documents Claude API model ID **`claude-opus-4-8`**.

Official limits and behavior:

- **1M context** and **128K maximum output**;
- Claude Messages API;
- adaptive thinking controlled with `thinking: {type: "adaptive"}` and output effort; default effort is high;
- tools and platform features inherited from Opus 4.7;
- fast mode research preview on the Claude API, up to 2.5× output speed;
- long-horizon agentic coding, tool triggering, compaction recovery, and enterprise work are the stated focus;
- non-default `temperature`, `top_p`, or `top_k` values are rejected for Messages API requests.

### Published architecture/design

Anthropic’s release and model documentation checked do **not** disclose low-level architecture, parameter count, expert count, attention design, or weight topology. The defensible architecture statement is therefore: **proprietary/undisclosed in the official pages checked**. Behavioral capabilities must not be presented as low-level architecture facts.

### OmniRoute v3.8.48 mapping

The official registry publishes two exact non-AWS routes for the same manufacturer model:

- `cc/claude-opus-4-8` — provider `claude`, alias `cc`, OAuth, Anthropic Messages;
- `anthropic/claude-opus-4-8` — provider/alias `anthropic`, API key, Anthropic Messages.

These rows prove that OmniRoute knows Opus 4.8, but they do not prove AWS provenance and therefore do not satisfy the P0 requirement “Kiro → Opus48 from AWS.”

## 4. AWS Bedrock — Claude Opus 4.8

AWS’s official model card reports:

- launch date **2026-05-28**;
- lifecycle **Active**; EOL **N/A**;
- **1M context**, **128K max output**, reasoning supported;
- text/image input and text output;
- Invoke, Converse, and Messages support; streaming support;
- `bedrock-runtime` and `bedrock-mantle` endpoints.

Official programmatic identifiers:

- base model ID: `anthropic.claude-opus-4-8`;
- geo inference IDs: `us.anthropic.claude-opus-4-8`, `eu.anthropic.claude-opus-4-8`, `jp.anthropic.claude-opus-4-8`, `au.anthropic.claude-opus-4-8`;
- global inference ID: `global.anthropic.claude-opus-4-8`.

The Bedrock card’s `bedrock-mantle` Messages URL is `https://bedrock-mantle.{region}.api.aws/anthropic/v1/messages`; `bedrock-runtime` requests use the selected model or inference-profile ID.

### OmniRoute executor, discovery, persistence, and publication

The Bedrock registries in v3.8.48 and current `main` (package version 3.8.49 when checked) are equivalent for this question: provider and alias are `bedrock`, executor is `bedrock`, format is `openai`, `passthroughModels: true`, and the static Opus entries stop at 4.6/4.7. Opus 4.8 is not static.

The v3.8.48 Bedrock executor uses AWS `BedrockRuntimeClient` with `ConverseCommand`/`ConverseStreamCommand`. It passes the selected route suffix literally as `payload.modelId = model`; it does not translate “Opus 4.8” or choose a profile. Therefore the runtime rule is:

```text
RouteModel = bedrock/<literal modelId or inferenceProfileId accepted by AWS Converse>
```

The official discovery/sync path is fully traced:

1. `/api/providers/[connection-id]/models` calls `discoverBedrockNativeModels` for provider `bedrock`.
2. Discovery calls AWS `ListFoundationModels` for text-output models and paginated `ListInferenceProfiles` with `typeEquals=SYSTEM_DEFINED`.
3. `normalizeBedrockDiscoveredModels` retains literal `modelId` and `inferenceProfileId` values (and recognizes foundation-model ARNs) as discovered row IDs.
4. `persistDiscoveredModels` writes the per-connection discovery cache; `/sync-models` fetches that endpoint with `refresh=true&excludeCustom=true` and calls `importManagedModels`.
5. `importManagedModels` removes stale imported custom rows, writes discovered rows through `replaceSyncedAvailableModelsForConnection`, and prunes stale/inactive connection caches.
6. `/v1/models` loads `getAllSyncedAvailableModels`; when a provider has any synced rows, its static catalog is suppressed. Each eligible synced Bedrock row is published as `${alias}/${sm.id}`, hence `bedrock/<literal-id>`.

Publication also requires an active Bedrock connection. `hasEligibleConnectionForModel` accepts the row when at least one active connection does not exclude it through `excludedModels`/`excluded_models` wildcard patterns; hidden/blocked-provider, `hidePaidModels`, and API-key filters can still remove it. `passthroughModels: true` permits forwarding a literal, but by itself neither creates nor certifies a `/v1/models` row.

Consequently, these are only candidate forms documented by AWS, not certified RouteModels for this installation:

```text
bedrock/anthropic.claude-opus-4-8
bedrock/us.anthropic.claude-opus-4-8
bedrock/global.anthropic.claude-opus-4-8
```

The geo variants for EU, Japan, and Australia are equally non-selectable until returned for the target account/region. No candidate may be chosen by textual similarity.

### Kiro/AWS CodeWhisperer is not a Bedrock substitute

Official OmniRoute history closes the Kiro hypothesis. PR #3131 historically added `claude-opus-4.8` to Kiro, but PR #6170 removed Opus 4.8/4.7/4.6 and other fabricated IDs after live VPS validation recorded `kiro/claude-opus-4.8 → 400 Invalid model`; the maintainer stated Kiro did not offer Opus. Current v3.8.48 and `main` Kiro registries confirm the removal. Kiro uses Amazon Q/CodeWhisperer `generateAssistantResponse`, not generic Bedrock Converse. Therefore **`kr/claude-opus-4.8` is invalid** and must not be restored, inferred, or used as AWS evidence.

### Read-only account/runtime evidence

Two catalog-only AWS calls were attempted in `us-east-1`; neither invoked a model:

- `bedrock:ListFoundationModels` filtered to Anthropic/TEXT: `AccessDeniedException` because the current role has no identity-based permission for the action.
- `bedrock:ListInferenceProfiles` with `typeEquals=SYSTEM_DEFINED`: `AccessDeniedException` for the same missing identity-based permission class.

The errors were recorded without credentials or tokens. They prove only that this role cannot retrieve the required account/region metadata; they do not prove model absence.

The declared OmniRoute endpoints were probed without authentication and without inference. `127.0.0.1:20128` and `192.168.1.27:20128` were unreachable for `/health`, `/api/health`, and `/v1/models`; `omniroute` did not resolve; no TCP listener existed on port 20128. Docker and Podman are absent locally. Thus no installation catalog or DB row could be inspected.

**Disposition:** `BLK-OPUS48-ID` is narrowed to two concrete missing runtime facts: (1) the literal AWS `modelId` or `inferenceProfileId` returned by `ListFoundationModels`/`ListInferenceProfiles` under the actual OmniRoute account and configured region; and (2) the corresponding active OmniRoute `/v1/models` row `bedrock/<literal-id>`. Satisfying this requires granting those two read-only catalog actions to the runtime role or running the same discovery through the authorized OmniRoute connection, then capturing the row. It does not require inference. Capability, pool/fallback, and live conformance remain separately under `BLK-AVAIL` and live-run authorization.

## 5. Blocker impact

### Resolved by this manufacturer/provider research

1. GLM-5.2 is a real Z.ai release; canonical Z.ai API ID is `glm-5.2`.
2. Kimi K2.7 is specifically **Kimi K2.7 Code**; canonical API IDs are `kimi-k2.7-code` and the separate HighSpeed serving ID.
3. Claude Opus 4.8 is a real Anthropic release; canonical Claude API ID is `claude-opus-4-8`.
4. AWS Bedrock officially serves Opus 4.8 under the base, geo, and global IDs listed above; lifecycle is Active.
5. NVIDIA’s official NIM catalog identifies its GLM serving target as `z-ai/glm-5.2`.

### Resolved by direct OmniRoute v3.8.48 source inspection

1. ClinePass Kimi exact `RouteModel`: **`cp/cline-pass/kimi-k2.7-code`**. This closes only the exact-ID question formerly recorded as `BLK-KIMI`.
2. Generic Anthropic Opus 4.8 routes: **`cc/claude-opus-4-8`** and **`anthropic/claude-opus-4-8`**. Neither is AWS.
3. ClinePass GLM and NVIDIA fallback are independently source-confirmed as `cp/cline-pass/glm-5.2` and `nvidia/z-ai/glm-5.2`.
4. Bedrock synced discovery publishes `bedrock/<literal modelId-or-inferenceProfileId>`; the executor forwards that suffix verbatim to AWS Converse.
5. `kr/claude-opus-4.8` is invalid: it was removed after a live `400 Invalid model` result and is absent from current Kiro registries.

### Not resolved; remains runtime-specific

1. `BLK-OPUS48-ID`: literal AWS Opus48 `modelId`/`inferenceProfileId` returned for the OmniRoute account/region and the matching active `bedrock/<literal-id>` `/v1/models` row.
2. `BLK-AVAIL`: active `/v1/models` rows, eligible connections, enriched capabilities, pool/fallback state, and live conformance for GLM/Kimi/Opus48 AWS.
3. Kimi product catalog/fixture correction and all other product changes remain unauthorized.
4. `BLK-25B`: live-provider security stop/key revocation.
5. Explicit per-family live-run authorization tokens and accepted production executions.
6. OmniRoute-owned authentication, credentials, accounts, quota, retry/failover, provider/account selection, circuits, and upstream error handling.

No `5.x`, `8.1`, or `8.2` acceptance item is closed by source research alone. All `live_runs.*.authorized` values remain `false`, and no inference was executed.

## 6. Official sources

All URLs accessed 2026-07-22.

### OmniRoute official v3.8.48

- Release/tag and signed commit `7ee5bbc64dbb03e967521227f2afffeb7c9dad1e`: https://github.com/diegosouzapw/OmniRoute/releases/tag/v3.8.48
- Provider registry generator: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providerRegistry.ts
- ClinePass row: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/registry/clinepass/index.ts
- Kimi/Moonshot shared models: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/shared.ts
- Claude OAuth row: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/registry/claude/index.ts
- Anthropic API row: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/registry/anthropic/index.ts
- AWS Bedrock row: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/registry/bedrock/index.ts
- Bedrock executor (literal `modelId` to Converse): https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/executors/bedrock.ts
- Bedrock discovery normalization: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/bedrock.ts
- Bedrock discovery service: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/services/bedrock.ts
- Provider discovery route: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/src/app/api/providers/%5Bid%5D/models/route.ts
- Sync/persistence route: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/src/app/api/providers/%5Bid%5D/sync-models/route.ts
- Connection model-exclusion rule: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/src/domain/connectionModelRules.ts
- Historical Kiro addition/removal: https://github.com/diegosouzapw/OmniRoute/pull/3131 and https://github.com/diegosouzapw/OmniRoute/pull/6170
- NVIDIA and Z.ai rows: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/registry/nvidia/index.ts and https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/open-sse/config/providers/registry/zai/index.ts
- `/v1/models` catalog composition: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/src/app/api/v1/models/catalog.ts
- API/CLI docs: https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/docs/reference/API_REFERENCE.md and https://github.com/diegosouzapw/OmniRoute/blob/v3.8.48/docs/reference/CLI-TOOLS.md

### Z.ai / Zhipu

- Release notes: https://docs.z.ai/release-notes/new-released
- GLM-5.2 model/API guide: https://docs.z.ai/guides/llm/glm-5.2
- Official Z.ai organization model card: https://huggingface.co/zai-org/GLM-5.2/blob/main/README.md

### Moonshot AI / Kimi

- Current model list: https://platform.kimi.ai/docs/models
- Kimi K2.7 Code guide: https://platform.kimi.ai/docs/guide/kimi-k2-7-code-quickstart
- Model parameter reference: https://platform.kimi.ai/docs/api/models-overview
- Official Moonshot model card: https://huggingface.co/moonshotai/Kimi-K2.7-Code

### Anthropic

- Release announcement: https://www.anthropic.com/news/claude-opus-4-8
- Claude Opus 4.8 developer documentation: https://platform.claude.com/docs/en/about-claude/models/whats-new-claude-4-8
- System card (PDF; architecture details not extracted as public low-level specifications): https://www.anthropic.com/claude-opus-4-8-system-card

### AWS Bedrock

- Claude Opus 4.8 model card and programmatic IDs: https://docs.aws.amazon.com/bedrock/latest/userguide/model-card-anthropic-claude-opus-4-8.html

### NVIDIA NIM

- Z.ai GLM-5.2 NIM model/API reference: https://docs.api.nvidia.com/nim/reference/z-ai-glm-5.2

## 7. OmniRoute runtime and local evidence consulted

- Official release provenance: v3.8.48, signed commit `7ee5bbc64dbb03e967521227f2afffeb7c9dad1e`.
- Local code-only snapshot: `.handoff-staging/omniroute-affinity/base/package.json` reports `3.8.48`; `PATCH_MANIFEST.md` pins base image `diegosouzapw/omniroute@sha256:badb560971fdc23c2fb84b3e8695116239ff215b4cca4b07076201a8efae7f0d`. The snapshot intentionally contains only patch-relevant files, so registry evidence came from the matching official tag.
- Runtime probe: no `omniroute` binary, Docker, or Podman; no TCP listener on port 20128. The three metadata paths `/health`, `/api/health`, and `/v1/models` were unreachable at both declared addresses `127.0.0.1:20128` and `192.168.1.27:20128`; Docker DNS name `omniroute` did not resolve. No secret content was read or emitted.
- AWS metadata probe: `ListFoundationModels` and `ListInferenceProfiles` in `us-east-1` both returned sanitized `AccessDeniedException` results for missing identity-based permissions. No inference API was called.
- Official catalog behavior: provider rows require active connections; synced Bedrock IDs are literal discovery results, and eligibility is exclusion-based per connection. Combos and synced/custom models are installation DB state. Therefore source proves the construction rule, not this installation’s selected literal or availability.
- `openspec/changes/build-omniroute-agent-brain/omniroute-architecture-acceptance-checklist.md` — lists `cp/cline-pass/glm-5.2`, `nvidia/z-ai/glm-5.2`, and leaves the approved Kimi model “Architect to confirm” (now superseded for the exact-ID question by v3.8.48 source).
- `.planning/agent-brain-v3/evidence/g1-model-route-matrix.md` — freezes GLM primary/fallback identities, confirms Chat endpoint family, and records exact Kimi ID as `UNDECLARED`.
- `openspec/changes/build-omniroute-agent-brain/OMNIROUTE_ARCHITECT_RESPONSE.md` — confirms Kimi/GLM/NVIDIA use `/v1/chat/completions` and identifies combo aliases without promoting them to exact model rows.
- `.deploy-control/p0/handoffs/A3-route-freeze.md` and `.deploy-control/p0/handoffs/A4-opus48.md` — local reconciliation and blocker ownership.
- `multica-auth-work/server/pkg/agent/models.go` — current catalog literals: correct `cp/cline-pass/glm-5.2` and incomplete `cline-pass/kimi-k2.7-code`; the source-backed Kimi row includes the `cp/` prefix. No product edit was made.

## Final adjudication

Direct inspection resolves the static routes exactly: `cp/cline-pass/glm-5.2`, `cp/cline-pass/kimi-k2.7-code`, `nvidia/z-ai/glm-5.2`, and the non-AWS Opus routes `cc/claude-opus-4-8` and `anthropic/claude-opus-4-8`. Official history also rejects `kr/claude-opus-4.8` as an invalid fabricated Kiro ID. For Bedrock, the exact internal construction is now proven: OmniRoute discovers and persists the AWS literal, publishes `bedrock/<literal-id>`, and sends that literal unchanged to Converse. The remaining identifier gap is not semantic: the current role lacks `bedrock:ListFoundationModels` and `bedrock:ListInferenceProfiles`, and no declared OmniRoute runtime is reachable, so the target account/region’s literal and `/v1/models` row cannot be observed. `BLK-OPUS48-ID` is therefore precisely that missing catalog metadata; `BLK-AVAIL`, `BLK-25B`, product-change authorization, and every live-run token remain open. No product code or live-run authorization changed, and no inference was executed.
