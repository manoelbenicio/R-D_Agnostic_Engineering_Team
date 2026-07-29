# A4 — Kiro/Opus48 accepted-Anthropic route gap analysis (handoff → W1)

- agent: Codex56#B · lane: A4 · task: P0-OPUS48-ROUTE · pane: w7:p4 (HERDR_ENV=1)
- covers: OpenSpec 5.8 (Kiro/Opus48 delivery), 8.1 (changed/unproven route capability+protocol), 8.2 (tools/reasoning/cancel/usage/terminal/error)
- mode: READ-ONLY on product source. Only mutable artifact = this file.
- repo: HEAD `a6d5098`, branch `integration/dev-transition-candidate-20260719`
- check-in: `.deploy-control/p0/checkins/Codex56-B__P0-OPUS48-ROUTE__20260721T223107Z.json`
- STATUS: **BLOCKED_EXTERNAL** on the exact Opus48 AWS `RouteModel` (OmniRoute registry owner). The
  Main-Brain implementation path is fully determined and requires **zero new Main-Brain code** beyond
  agent configuration once the ID is published.

---

## 1. Tool preflight (exact versions/paths/results)

| Tool | Path | Version / result |
|---|---|---|
| git | `/usr/bin/git` | `git version 2.50.1` |
| python3 | `/usr/bin/python3` | `Python 3.9.25` |
| rg | `/usr/local/bin/rg` | `ripgrep 15.2.0 (rev e89fff89ac)` |
| Go 1.26.1 | `/home/ec2-user/goroot/go/bin/go` (corrected final path per steering) | `go version go1.26.1 linux/amd64` — verified resolvable and executable from this pane. (Earlier `.local/toolchains` path was superseded; credential-slot homes were not used.) A4 is read-only analysis with **no compile step**, so no build was run. |
| gofmt 1.26.1 | `/home/ec2-user/goroot/go/bin/gofmt` | present and executable (unused this lane) |
| Registry access (no secrets) | `curl --max-time 4 http://127.0.0.1:20128/v1/models` (no auth header) | `curl: (7) Failed to connect ... port 20128` / `http_code=000`. **No local OmniRoute gateway running.** Model registry retrieval additionally requires the OmniRoute stable secret (`AGENT_BRAIN_GATEWAY_SECRET_FILE`) — OmniRoute-owned, out of lane. |

No secret/credential/account value was read, echoed, or written.

---

## 2. Which existing Anthropic-compatible CLIKind/frontend is accepted (PROVEN)

**Accepted frontend = `brain.CLIClaudeCode` (`"claude-code"`), speaking `ProtocolAnthropicMessages`.**
It is the **only** frontend that satisfies the credentialless Anthropic contract. Kiro is a
persona/model route, **not** a distinct frontend and **not** a credential owner.

Evidence (read-only, HEAD `a6d5098`):

1. `internal/daemon/runtimeenv/adapter.go` — `CredentiallessAdapterContract(CLIClaudeCode)` returns
   `AdapterContract{State: AdapterReady, Protocol: ProtocolAnthropicMessages}` with **no secret**. It is
   the only Anthropic-protocol adapter in `AdapterReady`. `CLICodex` is `AdapterReady`/OpenAI-Responses;
   `CLIOpenAICompatible`, `CLIKimi`, `CLINIM`, `CLIAntigravity` are `AdapterFailClosed`. Notably
   `CLIAntigravity` fails closed with `FallbackFrontends: [CLIClaudeCode, CLICodex]` (candidates only;
   the package never auto-falls-back).
2. `internal/daemon/gateway/profiles.go` — `TrustedRuntimeProfiles()[ProtocolAnthropicMessages]` =
   `ProfileAnthropicMessages` (`"omniroute-anthropic-messages"`): `Endpoint "/v1/messages"`,
   `BaseURLForm root`, `WireAPI "messages"`, `StreamTransport "sse"`,
   `AllowedCLI: [brain.CLIClaudeCode]`, `EvidenceRequired: true`. `RuntimeProfile.Validate()` enforces
   that `BaseURLRoot` is legal **only** for `ProtocolAnthropicMessages`.
3. `internal/daemon/brain/identity.go` — the frozen `CLIKind` set is exactly
   `{claude-code, codex, kimi, openai-compatible, antigravity, nim}`. **There is no `CLIKiro`.**
   `ProtocolAnthropicMessages = "anthropic-messages"`.
4. `internal/daemon/brain/compatibility.go` — `LegacyProviderCLIKind("claude") → CLIClaudeCode`. There
   is **no** legacy mapping that turns `kiro` into a gateway CLIKind; the gateway path for an
   Anthropic-Messages model must be driven by `claude-code`.
5. Live-analog precedent already in tree: `internal/daemon/brain_integration_test.go` drives
   `CLIKind: brain.CLIClaudeCode` with `RouteModel: "agy/claude-opus-4-6-thinking"` — i.e. an
   Anthropic-Messages Opus route through the claude-code frontend with a slash-namespaced RouteModel.

### Forbidden native Kiro path (must NOT be wired as a gateway credential owner)
`pkg/agent/kiro.go` spawns `kiro-cli acp` (Amazon Q Developer CLI fork) over ACP JSON-RPC 2.0;
`internal/daemon/execenv/kiro_home.go` isolates a per-account `XDG_DATA_HOME` containing the native
credential store `kiro-cli/data.sqlite3`. Per D-V3-27 + credential-isolation architecture, native
Kiro/AWS credential selectors (`KIRO_*`, `AWS_*`, `XDG_*` roots) are **blocked** in gateway-required
mode. The Opus48 route MUST NOT use this native path, MUST NOT create a new auth/account owner, and
MUST remain credentialless (OmniRoute owns credentials/rotation/fallback).

---

## 3. Exact Opus48 AWS RouteModel — BLOCKED_EXTERNAL

**There is no exact, registry-approved Opus48 AWS `RouteModel` string anywhere in the product source
or frozen planning matrix.** The approved model set is fetched **live at runtime** from OmniRoute:

- `internal/daemon/gateway/client.go` `Client.FetchModels(ctx, correlation)` → `ModelsDocument`
  (`gateway/health_models.go`).
- `internal/daemon/gateway/registry.go` `buildSnapshot(ModelsDocument)` builds the immutable
  `RegistrySnapshot` of `RouteModel → ModelSpec` (protocol, streaming, tools, reasoning,
  structured-output, context-limit, account-pool, availability, fallback). TTL-cached; failure-backed
  off.
- `internal/daemon/brain_integration.go:217` wires the runtime `Registry` from `client.FetchModels`.
  In dev, `gateway/model_projection.go` `ProjectOmniRouteModels(...)` can project a native models shape,
  but the **IDs still originate from OmniRoute**, not from this repo.

Strings that look related but are **NOT** gateway RouteModel IDs (do not substitute them):
- `internal/metrics/pricing.go:24` `"anthropic:claude-opus-4.8"` (Provider/Model `claude-opus-4.8`) —
  a **billing/pricing** display key.
- `pkg/agent/thinking.go:172` `"claude-opus-4-8"` — a **reasoning/effort** schema key.

Authoritative intent (semantic only, no ID):
- `evidence/authoritative-route-matrix-D-V3-27.md` row 7: **"Kiro → Opus48 from AWS", status
  "Route to test"**, acceptance basis "protocol/tools/reasoning/usage/cancel/error". No exact string.
- `DECISIONS.md:309`: "7. **Kiro** = **Opus48 da AWS**."

### BLOCKER (concrete owner + action)
- **Owner:** OmniRoute registry owner (the same owner A3 depends on for exact GLM-5.2 / Kimi-K2.7 IDs).
- **Required action:** publish, in the OmniRoute `/v1/models` document, the exact Opus48-on-AWS
  `RouteModel` row with: canonical `id`; `protocol = anthropic-messages`; `tools`, `reasoning`,
  `streaming`, `structured_output` booleans; `context_limit > 0`; `account_pool`; `available = true`;
  plus provenance (gateway endpoint/version/digest + UTC timestamp). Per `EVIDENCE_CONTRACT.md`, a `200`
  on `/v1/models` alone does not prove protocol fidelity — the row must carry the anthropic-messages
  capability fields.
- **Why I cannot resolve it here:** retrieval needs a running OmniRoute gateway + the stable secret
  (both OmniRoute-owned; secret handling is explicitly forbidden to this lane), and no local gateway is
  reachable (§1). Inventing/guessing an ID is prohibited (common contract).

---

## 4. Model intent → launch delta (implementable path; W1 to integrate)

Given the exact `RouteModel` `<OPUS48_AWS_ID>` from §3, the launch chain reuses the **already-accepted**
Claude→Anthropic-Messages plumbing end to end:

1. **Agent config / intent** — set `CLIKind = claude-code` and `RouteModel = <OPUS48_AWS_ID>`; router
   owner resolves to `omniroute` in gateway-required mode. (No provider/account/credential fields.)
2. **Admission** (`brain` contract) — `TaskRequest{CLIKind: CLIClaudeCode, RouteModel: <OPUS48_AWS_ID>,
   RouterOwner: omniroute, GatewayRequired: true}.Validate()`.
3. **Adapter** — `runtimeenv.CredentiallessAdapterContract(CLIClaudeCode)` → `AdapterReady`,
   `ProtocolAnthropicMessages` (no secret).
4. **Profile** — `gateway.LookupRuntimeProfile(ProtocolAnthropicMessages, CLIClaudeCode)` →
   `ProfileAnthropicMessages` (`/v1/messages`, root base URL, SSE).
5. **Registry capability** — `Registry.LookupModel(<OPUS48_AWS_ID>)` must return `Available` with
   `Capability.Protocol == anthropic-messages`; `ValidateCapability` asserts streaming/tools/reasoning
   as required by the run.
6. **Policy** — `runtimeenv.GatewayModelPolicy.ValidateSelection(claude-code, <OPUS48_AWS_ID>, thinking)`
   (protocol match against the adapter contract). Built per-task in
   `brain_integration.go` `validateThinking`/selection.
7. **Launch** — claude-code frontend pointed at the OmniRoute base URL + stable secret env only
   (task 5.5 trusted-root/token env); pre-launch assertion (`runtimeenv.AssertPreLaunch`, task 5.10)
   confirms the child env carries only the stable OmniRoute secret + approved local task data — no
   Anthropic/AWS/Kiro native credentials.

**Delta size = effectively ZERO new Main-Brain code.** The Opus48/AWS route reuses the same frontend
(`claude-code`), the same profile (`omniroute-anthropic-messages`), and the same credentialless adapter
as the already-accepted Claude route (route-matrix #2; task 8.2 G4 recorded the Claude trusted-gateway
path as passed). The **only** required inputs are external: (a) the exact `RouteModel` from OmniRoute
(§3), and (b) one reserved live non-prod acceptance run (§5). W1 integrates none of the shared hotspots
for this route beyond agent-config wiring; no new `CLIKind`, profile, or adapter.

### Watch-item for W1 (flagged, not a Main-Brain gap to implement blind)
`brain_integration.go` `validateThinking` currently returns `ErrThinkingNotApproved` for any non-empty
thinking string and builds the policy with empty thinking. Opus 4.8 advertises effort levels
(`thinking.go` `claude-opus-4-8`: low/medium/high/xhigh/max). If a Kiro/Opus48 agent is expected to pass
a thinking/effort level through the gateway, the approved `RouteModel` row must enumerate
`ThinkingLevels` and the selection must carry them. This axis is **Anthropic-protocol-general, not
Opus48-specific**, and is gated by the same registry publication — do not invent effort approvals.

---

## 5. Focused tests + acceptance (5.8 / 8.1 / 8.2), no duplication

Offline/synthetic focused checks (reuse existing suites; add only the Opus48 model row as a synthetic
fixture mirroring the eventual registry row — no new campaign):

- **8.1 capability + protocol (changed/unproven route only):**
  - `internal/daemon/gateway` — `registry_test.go` / `g4_protocol_conformance_test.go`: a synthetic
    `ModelsDocument` row `{id:<OPUS48_AWS_ID>, protocol:anthropic-messages, tools/reasoning/streaming:true,
    context_limit:>0, available:true}` builds a snapshot and passes `ValidateCapability` for the
    Anthropic-Messages requirement; malformed/absent row fails `ErrUnknownModel`/`ErrProtocol`.
  - `internal/daemon/gateway` — `profiles.go` coverage: `LookupRuntimeProfile(anthropic-messages,
    claude-code)` resolves `ProfileAnthropicMessages`; `(anthropic-messages, <non-claude-code>)`
    fail-closed.
- **8.2 tools/reasoning/cancel/usage/terminal/error (only where not equivalent):**
  - `internal/daemon/runtimeenv` — `model_test.go`/`gateway_acceptance_test.go`:
    `ValidateSelection(claude-code, <OPUS48_AWS_ID>, "")` accepted; wrong CLI → `ErrCLIModelNotApproved`;
    wrong protocol → `ErrProtocolNotCompatible`; unknown model → `ErrModelNotApproved` (deterministic
    fail-closed error).
  - `pkg/agent/thinking_test.go` already exercises the `claude-opus-4-8` effort schema (reasoning axis) —
    reuse, do not duplicate.
  - Tools/cancel/usage/terminal-result at the **protocol** level are already proven for
    Claude→Anthropic-Messages (route-matrix #2). Opus48 reuses that protocol evidence; only
    **model-id-specific** behavior is genuinely new.
- **Single live non-prod acceptance (protocol·tools·reasoning·usage·cancel·error):** reserved by W1 as
  the one run per route family (control `live_runs.opus48`), executed only after integration with the
  exact `RouteModel`. It closes overlapping 5.8/8.1/8.2 in one pass.
  - **Additionally gated by D-V3-25(B):** live-provider tests are SECURITY-STOPPED until the Owner
    confirms revocation of the exposed UI key. Offline/synthetic checks above may proceed now.
  - Command to run/compile the focused Go checks (fleet toolchain):
    `/home/ec2-user/goroot/go/bin/go test ./internal/daemon/gateway/... ./internal/daemon/runtimeenv/... ./pkg/agent/ -run 'AnthropicMessages|RouteModel|Selection|Thinking'`
    (not executed by A4: read-only lane, no compile step).

Antigravity is **not** in A4's scope: it is evidence-reuse (A5), not reimplementation.

---

## 6. W1_HANDOFF summary

- **Accepted frontend:** `CLIKind = claude-code` (`brain.CLIClaudeCode`) + `ProfileAnthropicMessages`
  (`/v1/messages`, root, SSE) + `CredentiallessAdapterContract` `AdapterReady`. No `CLIKiro`; native
  `kiro-cli`/XDG/AWS path forbidden.
- **Exact RouteModel:** `BLOCKED_EXTERNAL` — OmniRoute registry owner must publish the exact
  Opus48-on-AWS `RouteModel` row (anthropic-messages capability fields + provenance). Do not guess.
- **Delta:** zero new Main-Brain code; agent-config wiring only (`CLIKind=claude-code` +
  `<OPUS48_AWS_ID>`), reusing the accepted Claude→Anthropic-Messages chain.
- **Tests:** synthetic capability/selection/profile checks now; one reserved live non-prod run closes
  5.8/8.1/8.2, gated by D-V3-25(B) security-stop.
- **Blockers:** (1) exact Opus48 AWS RouteModel (OmniRoute registry owner); (2) live run security-stop
  until key revocation (Owner).
