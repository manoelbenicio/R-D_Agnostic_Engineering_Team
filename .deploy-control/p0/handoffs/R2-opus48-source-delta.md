# R2-DESIGN — Kiro/Opus48 exact minimal source-delta plan (handoff → W1)

- agent: Codex56#B · lane: R2-DESIGN · task: P0-OPUS48-SOURCE-DELTA · pane: w7:p4 (HERDR_ENV=1)
- covers: OpenSpec 5.8 (Kiro/Opus48 delivery), 8.1, 8.2
- mode: **READ-ONLY on product source** (no edits until `FLEET_SATURATED=GREEN` and source locks frozen). Only mutable artifact = this file. No live run.
- repo: HEAD `a6d5098`, branch `integration/dev-transition-candidate-20260719`
- check-in: `.deploy-control/p0/checkins/Codex56-B__P0-OPUS48-SOURCE-DELTA__20260721T223812Z.json`
- consumes: `.deploy-control/p0/handoffs/A4-opus48.md`; `evidence/authoritative-route-matrix-D-V3-27.md`; frozen contracts in `internal/daemon/{brain,gateway,runtimeenv}` + `daemon.go`/`config.go`.

## HEADLINE VERDICT

**The Kiro/Opus48 gateway route requires ZERO required product-source lines.** It is delivered by
**configuration + an external OmniRoute registry publication**, reusing the already-accepted Anthropic
frontend (`claude-code`) end to end. The single unresolved input is the exact Opus48 AWS `RouteModel`
string — an **isolated external blocker** on the OmniRoute registry owner. No credential/account/quota/
retry/failover ownership is created or touched.

---

## 1. Tool preflight (exact)

| Tool | Path | Result |
|---|---|---|
| git | `/usr/bin/git` | `git version 2.50.1` |
| python3 | `/usr/bin/python3` | `Python 3.9.25` |
| rg | `/usr/local/bin/rg` | `ripgrep 15.2.0` |
| Go 1.26.1 | `/home/ec2-user/goroot/go/bin/go` | `go version go1.26.1 linux/amd64` (verified; final corrected path) |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | present/executable |
| Registry (no secret) | `curl http://127.0.0.1:20128/v1/models` | connection refused — no local gateway; retrieval is OmniRoute-owned (secret + running gateway) |

No compile/live run performed (read-only design lane). No secret/credential/account value read or written.

---

## 2. Proven resolution chain: agent config → CLIKind + RouteModel → launch

All symbols verified read-only at HEAD `a6d5098`.

1. **Config load** — `config.go` `loadAgentBrainIntegrationConfig` (lines 841–842):
   - `CLIKind ← firstConfigured(overrides.AgentBrainCLIKind, os.Getenv("AGENT_BRAIN_CLI_KIND"))` → `brain.ParseCLIKind`.
   - `RouteModel ← firstConfigured(overrides.AgentBrainRouteModel, os.Getenv("AGENT_BRAIN_ROUTE_MODEL"))` → `brain.ParseRouteModel`.
   - ⇒ Both are **pure configuration**; no model string is hard-coded in source.
2. **Config validation** — `config.go` `AgentBrainIntegrationConfig.Validate` (line 179): in
   gateway-required dev mode, `CLIKind` must be **`claude-code` or `codex`** (claude-code = the accepted
   Anthropic frontend); `RouteModel` must merely parse; `CapacityTier` must be `CapacityTier20`.
3. **Immutable executable mapping** — `config.go` `agentBrainBuiltInCLIFor`:
   `CLIClaudeCode → {Provider:"claude", Command:"claude"}` (Codex → codex; default error). No
   `MULTICA_*_PATH`/profile override honored in gateway mode (`resolveAgentBrainBuiltInEntry`).
4. **Task→runtime resolution** — `daemon.go` `resolveTaskAgentEntry` (gateway branch, ~L3255–3275):
   rejects custom args; computes `builtIn = agentBrainBuiltInCLIFor(cfg.AgentBrain.CLIKind)`; **requires
   `claimedProvider == builtIn.Provider`** (i.e. `"claude"`) else `builtin_runtime_provider_mismatch`;
   looks up `cfg.Agents["claude"]` (must be installed with a path).
5. **Admission** — `brain_integration.go` `admitTask` (L152): uses `r.config.CLIKind` + `r.config.RouteModel`;
   requires `LegacyProviderCLIKind(provider) == r.config.CLIKind` and, when `legacyModel` non-empty,
   `ParseRouteModel(legacyModel) == r.config.RouteModel`. `legacyModel` = `task.Agent.Model` else
   `entry.Model` (`daemon.go` L3316–3319). Adapter/profile resolved via
   `runtimeenv.CredentiallessAdapterContract(claude-code)=AdapterReady/anthropic-messages` and
   `gateway.LookupRuntimeProfile(anthropic-messages, claude-code)=ProfileAnthropicMessages`.
6. **Route policy** — `gateway.FrozenTier20CanaryPolicy().Validate(RouteModel)` (`gateway/policy.go`
   `RoutePolicy.Validate`): checks only parse + `RouterOwner=omniroute` + rotation/affinity/retry/
   circuit/smart-context validity + cross-model-fallback shape. **There is NO RouteModel allow-list** —
   any well-formed Opus48 id is accepted structurally. Nothing to extend here.
7. **Credential source** — `admitTask` fails closed with `credential_source_unavailable` unless
   `dependencies.CredentialSource` is injected. That injection is **OmniRoute-owned** (PD-08); the Brain
   never reads a credential value.

**Consequence:** with `AGENT_BRAIN_CLI_KIND=claude-code` and `AGENT_BRAIN_ROUTE_MODEL=<OPUS48_AWS_ID>`,
a task whose claimed `provider="claude"` and `model=<OPUS48_AWS_ID>` (or empty) is admitted and launched
on the `claude` (Claude Code) command against OmniRoute — with no source change.

---

## 3. Minimal delta = configuration only (no source patch)

| Change | Kind | Where | Value |
|---|---|---|---|
| `AGENT_BRAIN_CLI_KIND` | env/config (not source) | daemon env / `Overrides.AgentBrainCLIKind` | `claude-code` |
| `AGENT_BRAIN_ROUTE_MODEL` | env/config (not source) | daemon env / `Overrides.AgentBrainRouteModel` | `<OPUS48_AWS_ID>` (**external — see §5**) |
| `AGENT_BRAIN_GATEWAY_REQUIRED` / base URL / secret-file ref | env/config | existing gateway config | unchanged mechanism; secret is OmniRoute-owned |
| "claude" agent installed | runtime prerequisite | `cfg.Agents["claude"]` (Claude Code on PATH) | required by `resolveTaskAgentEntry` |
| Kanban agent/task | data/config | agent record | `provider="claude"`, `model=<OPUS48_AWS_ID>` |

**Product source diff = ∅.** No file in `internal/daemon/**` or `pkg/agent/**` needs editing for the
route to admit, launch, and stream. This matches the A4 finding ("zero new Main-Brain code") and is now
proven down to the config loader and admission call site.

---

## 4. Explicitly rejected non-minimal alternatives (do NOT implement)

- **Adding `case "kiro": return CLIClaudeCode` to `brain.LegacyProviderCLIKind`** — insufficient AND
  wrong-layer. Even with it, `resolveTaskAgentEntry` still rejects `claimedProvider != "claude"`
  (`builtin_runtime_provider_mismatch`), so a `provider="kiro"` task cannot flow through the gateway
  without ALSO changing `agentBrainBuiltInCLIFor`/`resolveTaskAgentEntry`. That is a larger,
  credential-adjacent change to the immutable executable mapping → **forbidden and non-minimal**.
- **Native Kiro frontend (`pkg/agent/kiro.go`, `execenv/kiro_home.go`, `kiro-cli acp`, `XDG_DATA_HOME`,
  `kiro-cli/data.sqlite3`, `KIRO_*`/`AWS_*`)** — this is the native-credential path. Wiring it as the
  Opus48 route would create a new credential/account owner → **forbidden** (OmniRoute-only) and violates
  D-V3-27 "Kiro is a persona/model route, not a credential owner."
- **Extending a route allow-list** — none exists (`FrozenTier20CanaryPolicy.Validate` has no model list),
  so there is nothing to extend; inventing one would be new scope.

Decision: **Kiro/Opus48 = the `claude-code` frontend + the Opus48 RouteModel via config**, not a native
provider and not a new CLIKind.

---

## 5. Isolated external blocker (the ONLY unresolved input)

- **Blocker:** exact Opus48 AWS `RouteModel` string is `BLOCKED_EXTERNAL`.
- **Owner:** OmniRoute registry owner (same owner A3 depends on for exact GLM-5.2 / Kimi-K2.7 ids).
- **Required action:** publish, in the OmniRoute `/v1/models` document consumed by
  `gateway.Client.FetchModels`→`buildSnapshot`, the exact Opus48-on-AWS row with `id`,
  `protocol="anthropic-messages"`, `tools`/`reasoning`/`streaming`/`structured_output` booleans,
  `context_limit>0`, `account_pool`, `available=true`, plus provenance (endpoint/version/digest + UTC).
- **Isolation guarantee:** the id appears in exactly two config values (`AGENT_BRAIN_ROUTE_MODEL` and the
  agent record `model`) and in the external registry. No source symbol embeds it. When published, no code
  changes; only config values are filled and the registry serves the row.
- Note: strings `anthropic:claude-opus-4.8` (`internal/metrics/pricing.go`) and `claude-opus-4-8`
  (`pkg/agent/thinking.go`) are **billing/effort keys, NOT gateway RouteModel ids** — do not substitute.

---

## 6. File / symbol / test matrix (verification when `<OPUS48_AWS_ID>` is known)

No source files change. The matrix below is the **proof surface** W1 runs once the id is published; all
targets already exist. `<ID>` = the published Opus48 AWS RouteModel.

| # | Concern (task) | Symbol / file (read-only anchor) | Focused test to run/extend | Expected |
|---|---|---|---|---|
| 1 | Config accepts route (5.8) | `config.go` `loadAgentBrainIntegrationConfig`, `AgentBrainIntegrationConfig.Validate` | `internal/daemon/brain_integration_test.go` `loadAgentBrainIntegrationConfig` cases (L458/476) with `CLIKind=claude-code`, `RouteModel=<ID>` | config valid; no error |
| 2 | Admission maps provider→CLIKind + model match (5.8/8.2) | `brain_integration.go` `admitTask`; `brain.LegacyProviderCLIKind` | `brain_integration_test.go` admission tests (L72/219) called as `admitTask(ctx, task, "claude", "<ID>")` | plan admitted; provider "kiro"/wrong model → deterministic fail-closed (`legacy_contract_rejected`/`route_model_not_approved`) |
| 3 | Adapter + profile (8.1) | `runtimeenv.CredentiallessAdapterContract(claude-code)`; `gateway.LookupRuntimeProfile(anthropic-messages, claude-code)` | `internal/daemon/runtimeenv` `adapter`/`gateway_acceptance_test.go`; `gateway` `profiles` coverage | `AdapterReady`/`anthropic-messages`; `ProfileAnthropicMessages` (`/v1/messages`, root, SSE) |
| 4 | Registry capability (8.1/8.2) | `gateway.Registry.LookupModel`/`ValidateCapability`; `buildSnapshot` | `internal/daemon/gateway` `registry_test.go` / `g4_protocol_conformance_test.go` with a synthetic `ModelsDocument` row `{id:<ID>, protocol:anthropic-messages, tools/reasoning/streaming:true, context_limit>0, available:true}` | model `Available`, protocol anthropic-messages; tools/reasoning/streaming validated |
| 5 | Selection policy (8.2) | `runtimeenv.GatewayModelPolicy.ValidateSelection` | `internal/daemon/runtimeenv` `model_test.go` | `(claude-code,<ID>)` accepted; wrong CLI→`ErrCLIModelNotApproved`; wrong proto→`ErrProtocolNotCompatible`; unknown→`ErrModelNotApproved` |
| 6 | Route policy (8.1) | `gateway.FrozenTier20CanaryPolicy().Validate(<ID>)` | `internal/daemon/gateway` policy coverage | valid (no allow-list); owner omniroute |
| 7 | Reasoning/effort axis (8.2) | `pkg/agent/thinking.go` `claude-opus-4-8` schema | `pkg/agent/thinking_test.go` (existing) | effort levels resolve; reuse — do not duplicate |
| 8 | Terminal/tools/cancel/usage (8.2) | Anthropic-Messages protocol path (shared with accepted Claude route #2) | **single reserved live non-prod run (W1)** | protocol/tools/reasoning/usage/cancel/error captured once |

Compile/run (fleet toolchain; not executed by this lane):
`/home/ec2-user/goroot/go/bin/go test ./internal/daemon/... ./pkg/agent/ -run 'AgentBrain|RouteModel|AnthropicMessages|Selection|Thinking|Admit'`

**Overlap closure:** items 1–7 are offline/synthetic and may run pre-live. Item 8 is the **one** reserved
live non-prod acceptance per family (`control.json` `live_runs.opus48`), which closes overlapping
5.8/8.1/8.2 in a single pass. Additionally gated by **D-V3-25(B)** security-stop until the Owner confirms
key revocation. No second live run, no broad regression, no QA-A/B/C.

---

## 7. Zero credential ownership (invariants preserved)

- No auth/credential/token/quota/window/expiry/refresh/revocation/401/403/429/5xx/circuit/retry/
  account-selection/rotation/quarantine/failover logic is added or edited. `CredentialSource` stays an
  injected OmniRoute-owned dependency (`admitTask` fails closed without it).
- No native provider credentials: `claude-code` runs credentialless against OmniRoute base URL + stable
  secret env only (tasks 5.5/5.10 pre-launch assertion). Native Kiro/AWS selectors remain blocked.
- Single router owner `omniroute`; no dual router; no Prodex (Phase 3/HOLD untouched).

---

## 8. Out-of-scope surfaces (flagged, not this lane)

- **Frontend model catalog / persona label** (`packages/core/runtimes/models.ts`,
  `model-picker.tsx`) — whether the UI shows a "Kiro" persona bound to `<ID>` is a frontend concern
  owned by A7; it is **not required** for the Brain route to function (route works with
  `provider="claude"` + `model=<ID>`). Reported, not designed here.
- **OpenSpec spec wording** (`specs/omniroute-agent-routing/spec.md:4` generic five-vendor phrasing that
  omits Kiro=Opus48/AWS) — council/W8 process per D-V3-27; not edited.

---

## 9. W1_HANDOFF

- **Delta:** none in product source. Set `AGENT_BRAIN_CLI_KIND=claude-code`,
  `AGENT_BRAIN_ROUTE_MODEL=<OPUS48_AWS_ID>`; ensure `claude` (Claude Code) installed; create the Kanban
  agent as `provider="claude"`, `model=<OPUS48_AWS_ID>`.
- **Blocked on:** (1) exact `<OPUS48_AWS_ID>` from OmniRoute registry owner; (2) live acceptance
  security-stop (D-V3-25(B)) until Owner confirms key revocation.
- **Do not:** add a `kiro` CLIKind/provider mapping, touch `agentBrainBuiltInCLIFor`/
  `resolveTaskAgentEntry`, wire native `kiro-cli`, or invent an id/fallback.
- **Verify:** run matrix §6 items 1–7 offline once `<ID>` is published; reserve the single live run for
  item 8.
