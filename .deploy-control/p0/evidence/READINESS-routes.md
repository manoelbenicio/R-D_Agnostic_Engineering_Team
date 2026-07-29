# OmniRoute Declared Readiness Routes (Consume-Only Metadata)

**Lane**: `readiness-metadata` (`wK:p2`)  
**Task ID**: `6.1-READINESS-METADATA`  
**Timestamp**: 2026-07-22T21:57:15Z  
**Repository HEAD**: `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`  
**Policy**: Consume-only metadata inspection via existing gateway secret-file references. Zero secrets, tokens, or credential values read or recorded (`SecretsPresent == false`).  

---

## 1. Declared Ready OmniRoute Models (Codex / Auto-Coding Relevant)

The following route model IDs are declared ready for consume-only gateway admission across Codex, Cline, and Claude Code auto-coding execution lanes:

| Route Model ID | Target Purpose | Protocol / Endpoint | Selection Authority | Status |
|---|---|---|---|---|
| **`cp/cline-pass/glm-5.2`** | Primary GLM Auto-Coding | OpenAI Chat (`/v1/chat/completions`) | Main Brain Selectable (Cline runtime) | **FROZEN & READY** |
| **`nvidia/z-ai/glm-5.2`** | Bounded Fallback GLM Auto-Coding | OpenAI Chat (`/v1/chat/completions`) | OmniRoute-Owned Fallback (Not Brain-selectable) | **FROZEN (Fallback)** |
| **`claude_code_kimi_2.7_Code`** | Claude Code / Kimi Auto-Coding | Anthropic Messages (`/v1/messages`) | Main Brain Selectable (Claude Code runtime) | **FROZEN & APPROVED** |

---

## 2. Model Catalog Metadata & Capability Contracts

### 2.1 Primary Auto-Coding Route: `cp/cline-pass/glm-5.2`
- **Model ID**: `cp/cline-pass/glm-5.2`
- **Adapter Class**: `CLIOpenAICompatible` (Cline runtime)
- **Protocol**: OpenAI Chat Completions (`/v1/chat/completions`)
- **Capability Schema**:
  - `streaming`: `true`
  - `tools`: `true`
  - `reasoning`: `true`
  - `structured_output`: `true`
  - `context_limit`: `128000`
- **Readiness Gate**: `SelectedModelReady == true` when published with `available = true` in `/v1/models` catalog snapshot.

### 2.2 Fallback Route: `nvidia/z-ai/glm-5.2`
- **Model ID**: `nvidia/z-ai/glm-5.2`
- **Adapter Class**: `CLIOpenAICompatible` (NIM / NVIDIA pool)
- **Protocol**: OpenAI Chat Completions (`/v1/chat/completions`)
- **Routing Scope**: Bounded fallback owned strictly by OmniRoute gateway; Main Brain cannot explicitly select or force fallback to `nvidia/z-ai/glm-5.2` (`isNVIDIAOwnedRoute == true` -> fail-closed).

### 2.3 Claude Code Route: `claude_code_kimi_2.7_Code`
- **Model ID**: `claude_code_kimi_2.7_Code`
- **Adapter Class**: `CLIClaudeCode` (Claude Code runtime)
- **Protocol**: Anthropic Messages (`/v1/messages`)
- **Capability Schema**:
  - `streaming`: `true`
  - `tools`: `true`
  - `reasoning`: `true`
  - `structured_output`: `true`
  - `context_limit`: `200000`

---

## 3. Consume-Only Security & Fail-Closed Guardrails

1. **Secret Non-Disclosure**: Zero secret values, bearer tokens, or API keys are written, hashed, or embedded in this report or telemetry spans.
2. **Black-Box Consumption**: Main Brain reads boolean readiness signals (`Live`, `Authenticated`, `ModelRegistryReady`, `SelectedModelReady`, `SelectedProtocolReady`) via `GatewayReadinessChecker`. It performs zero internal probing of OmniRoute provider accounts, rotation pools, or session tables.
3. **Fail-Closed Admission**: Unrecognized or un-approved route model IDs return `AdmissionCapabilityRejected` immediately before process launch.
