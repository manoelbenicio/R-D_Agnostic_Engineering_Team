# R1-DESIGN — Cline Route Source Delta (P0-CLINE-ROUTE-SOURCE-DELTA)

- agent: `Codex56#A`  ·  lane: `R1-DESIGN`  ·  task: `P0-CLINE-ROUTE-SOURCE-DELTA`  ·  pane: `w7:p3`
- output lock: `.deploy-control/p0/handoffs/R1-cline-route-source-delta.md` (only mutable artifact)
- covers OpenSpec: **5.6** (Cline→GLM-5.2, implementable) · **5.7** (Cline→Kimi-K2.7, ID-blocked) · **8.1**
- product source: **READ-ONLY** until `FLEET_SATURATED=GREEN`. This is a **design-only** patch plan; no source edited.
- consumes: `A1-A2-cline-foundation.md` (gap matrix C1–C10) + `A3-route-freeze.md` (frozen IDs / blockers).
- boundaries honored: no invented IDs; **no auth/credential/quota/retry/failover**; NVIDIA stays OmniRoute-only; Prodex untouched.

> STATUS: **exact minimal patch plan produced.** The Cline OpenAI-compatible frontend wiring (7
> edits) is a single model-agnostic delta that is **fully implementable now for GLM-5.2** using the
> A3-frozen `cp/cline-pass/glm-5.2`. The **Kimi-K2.7 route reuses the identical code path** and needs
> **only** an exact `RouteModel` string that is **BLOCKED_EXTERNAL** (A3 BLK-KIMI) — no separate
> implementation. See §4 (implementable), §5 (Kimi-blocked), §6 (test delta).

---

## 0. Preflight (exact, this pane)

| Tool | Path | Version | Result |
|---|---|---|---|
| HERDR_PANE_ID | env | `w7:p3` | matches Codex56#A ✓ |
| git | `/usr/bin/git` | `2.50.1` | ✓ |
| python3 | `/usr/bin/python3` | `3.9.25` | ✓ (p0_control check-in OK) |
| rg | `/usr/local/bin/rg` | `15.2.0` | ✓ |
| herdr | `/home/ec2-user/.local/bin/herdr` | `0.7.4` | ✓ |
| Go (verify only, per A1-A2 §1) | `/home/ec2-user/goroot/go/bin/go` | `go1.26.1 linux/amd64` | not run (read-only phase) |

Signatures below were read directly from source (AST lookup + line reads), not from the A1-A2
summary, so old-behavior snippets are authoritative.

---

## 1. Design invariant — one code path, two routes

5.6 and 5.7 are **not two implementations**. Both use:
- the **same CLIKind** → `brain.CLIOpenAICompatible` (`"openai-compatible"`); executable `cline`;
- the **same protocol/profile** → `brain.ProtocolOpenAIChat` (`gateway/profiles.go` already lists
  `CLIOpenAICompatible` in `AllowedCLI`);
- the **same carrier contract** → `runtimeenv/cline.go` `NewClineConfigContract` (already model-parameterized, zero delta — A1 [EQUIV]).

They differ **only** by the `brain.RouteModel` value carried in `plan.Task.Request.RouteModel`:
- GLM-5.2 → **`cp/cline-pass/glm-5.2`** (A3 FROZEN) → **implementable now**.
- Kimi-K2.7 → **UNDECLARED** (A3 BLK-KIMI) → route-selection/live-validation blocked; wiring shared.

Therefore the entire wiring delta (§4) is written **once**; enabling it makes the GLM route work
end-to-end and simultaneously makes the Kimi route work the instant its exact ID is published — with
no further code.

---

## 2. Ownership map (who applies each edit; A3/R1-DESIGN applies none — design only)

| File | Symbol(s) | Lane owner (per FILE_OWNERSHIP P0 override) | GLM/Kimi |
|---|---|---|---|
| `runtimeenv/cline.go` | contract | A1 — **no delta** ([EQUIV]) | shared |
| `runtimeenv/adapter.go` | `CredentiallessAdapterContract` | R1 (`runtimeenv/**`), **W1-serial** | shared (D1) |
| `runtimeenv/env.go` | `trustedAdapterEntries`, `BuildGatewayEnvironment` | **W1** (shared switch) | shared (D2) |
| `internal/daemon/config.go` | `agentBrainBuiltInCLIFor`, `AgentBrainIntegrationConfig.Validate` | **W1** hotspot | shared (D3, D4) |
| `internal/daemon/execenv/cline_home.go` (**new**) | `WriteCredentiallessClineConfig`, `prepareCredentiallessClineHome` | **W1** hotspot family | shared (D5) |
| `internal/daemon/brain_integration.go` | `buildLaunch` | **W1** hotspot | shared (D6) |
| `server/pkg/agent/models.go` | `clineStaticModels` | **W1** hotspot | **GLM-only** (D7 = A3 SUB-1) |

R1-DESIGN (this artifact) edits **none** of the above; it specifies exact old→new for W1/A1.

---

## 3. Root-cause of "route currently fails closed" (verified)

The Cline route is not merely unwired — it is **actively fail-closed** at four points, by design,
until acceptance. Each must flip:
1. `adapter.go` `CredentiallessAdapterContract(CLIOpenAICompatible)` → returns
   `AdapterFailClosed{GateOpenAICompatibleUnaccepted}` + error. (Also gates `BuildGatewayEnvironment`,
   which calls the contract first.)
2. `env.go` `trustedAdapterEntries` `default: ErrAdapterFailClosed` (no `CLIOpenAICompatible` case).
3. `config.go` `agentBrainBuiltInCLIFor` `default: error` (no `cline` executable mapping).
4. `config.go` `AgentBrainIntegrationConfig.Validate` switch rejects everything but ClaudeCode/Codex.
Plus one missing capability: no `providers.json` writer / `buildLaunch` branch (5).

---

## 4. IMPLEMENTABLE NOW (GLM-5.2 route + shared frontend) — exact minimal patch plan

### D1 — `runtimeenv/adapter.go` · `CredentiallessAdapterContract` (owner: R1, W1-serial)
- OLD behavior:
  ```go
  case brain.CLIOpenAICompatible:
      contract := AdapterContract{CLI: cli, State: AdapterFailClosed, Gate: GateOpenAICompatibleUnaccepted}
      return contract, &AdapterGateError{Gate: contract.Gate}
  ```
- NEW behavior:
  ```go
  case brain.CLIOpenAICompatible:
      return AdapterContract{CLI: cli, State: AdapterReady, Protocol: brain.ProtocolOpenAIChat}, nil
  ```
- Rationale: this single gate flip converts the Cline frontend from fail-closed to selectable for
  the OpenAI-Chat profile. **Leave `CLIKimi`, `CLINIM`, `CLIAntigravity` fail-closed** (see §7).
- Blast radius: unblocks `BuildGatewayEnvironment` (calls the contract first) and
  `GatewayModelPolicy.ValidateSelection`.
- Tests: D1-T (§6.1).

### D2 — `runtimeenv/env.go` · `trustedAdapterEntries` (+ `BuildGatewayEnvironment`) (owner: W1)
- OLD: `switch profile.CLI { case CLIClaudeCode…; case CLICodex…; default: ErrAdapterFailClosed }`
  and `BuildGatewayEnvironment` sets `child.codexHome` only for `CLICodex`.
- NEW: add a `case brain.CLIOpenAICompatible:` that validates the controlled Cline data dir and
  injects **trusted-last**:
  ```go
  case brain.CLIOpenAICompatible:
      if err := validatePhysicalControlledDirectory(profile.ClineDataDir, "Cline data dir"); err != nil {
          return nil, "", "", err
      }
      entries["CLINE_DATA_DIR"] = environmentEntry{key: "CLINE_DATA_DIR", value: profile.ClineDataDir, origin: originTrustedLocal}
      entries[ClineOmniRouteAPIKeyEnv] = environmentEntry{key: ClineOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
      return entries, root, ClineOmniRouteAPIKeyEnv, nil
  ```
  (`ClineOmniRouteAPIKeyEnv = "CLINE_OMNIROUTE_API_KEY"` already exists in `cline.go`.)
- Supporting field decision (**minimal**): add `ClineDataDir string` to `AdapterEnvironment`
  (`env.go:107-113`). Do **NOT** overload `CodexHome`. No new `ChildEnvironment` struct field is
  required — the child process receives `CLINE_DATA_DIR` via `entries`/`Exec()`. (Only add a
  `ChildEnvironment.clineDataDir` field if a launch consumer must read it back; none identified.)
- Invariant (§7): `CLINE_DATA_DIR` + `CLINE_OMNIROUTE_API_KEY` are both rejected by
  `ClassifyEnvironmentKey` (DenyCredentialRoot / DenyProviderCredential) — they are legal ONLY as
  trusted entries merged after validation. Never source them from inherited/local/custom env.
- Tests: D2-T (§6.2).

### D3 — `internal/daemon/config.go` · `agentBrainBuiltInCLIFor` (owner: W1)
- OLD (`config.go:141-150`): only `CLIClaudeCode→{claude,claude}`, `CLICodex→{codex,codex}`, else error.
- NEW: add
  ```go
  case brain.CLIOpenAICompatible:
      return agentBrainBuiltInCLI{Provider: "cline", Command: "cline"}, nil
  ```
- Note: `resolveAgentBrainBuiltInEntry` already `exec.LookPath`s the returned `Command`; no other
  change. Gateway mode intentionally ignores `MULTICA_CLINE_PATH`/profile overrides (correct
  credentialless posture) — keep it.
- Tests: D3-T (§6.4).

### D4 — `internal/daemon/config.go` · `AgentBrainIntegrationConfig.Validate` (owner: W1)
- OLD (`config.go:194-198`):
  ```go
  switch c.CLIKind {
  case brain.CLIClaudeCode, brain.CLICodex:
  default:
      return fmt.Errorf("agent brain development mode supports only the accepted Claude Code or Codex frontend")
  }
  ```
- NEW: add `brain.CLIOpenAICompatible` to the accepted case and update the message, e.g.
  `"...only the accepted Claude Code, Codex, or OpenAI-compatible (Cline) frontend"`.
- Tests: D4-T (§6.4).

### D5 — `internal/daemon/execenv/cline_home.go` (**new file**) (owner: W1)
- Mirror `WriteCredentiallessCodexConfig` (`codex_home.go:87-102`) exactly, but write the Cline
  carrier at the **frozen subpath** `<clineDataDir>/settings/providers.json`:
  ```go
  func WriteCredentiallessClineConfig(clineDataDir string, raw []byte) error {
      if strings.TrimSpace(clineDataDir) == "" || !filepath.IsAbs(clineDataDir) || filepath.Clean(clineDataDir) == string(filepath.Separator) {
          return fmt.Errorf("credentialless cline data dir must be an absolute non-root path")
      }
      if len(raw) == 0 || len(raw) > 64<<10 {
          return fmt.Errorf("credentialless Cline configuration is empty or too large")
      }
      if err := prepareCredentiallessClineHome(clineDataDir); err != nil { // mkdir 0700 clineDataDir + settings/
          return err
      }
      path := filepath.Join(clineDataDir, "settings", "providers.json")
      if err := os.WriteFile(path, append([]byte(nil), raw...), 0o600); err != nil {
          return fmt.Errorf("write credentialless Cline configuration: %w", err)
      }
      return os.Chmod(path, 0o600)
  }
  ```
- `prepareCredentiallessClineHome` mirrors `prepareCredentiallessCodexHome` (physical-dir checks,
  0700 dir, no symlink) and creates the `settings/` subdir. Freeze subpath = `settings/providers.json`
  (matches `execenv/cline_home.go` `resolveClineSourceDir` recognized shape `<root>/settings/providers.json`).
- Tests: D5-T (§6.3).

### D6 — `internal/daemon/brain_integration.go` · `buildLaunch` (owner: W1)
- OLD (`brain_integration.go:277-361`): writes config + manifest **only** under
  `if plan.Task.Request.CLIKind == brain.CLICodex { … NewCodexConfigContract … WriteCredentiallessCodexConfig … manifest=[config.toml,sessions,skills] }`.
- NEW: add a sibling branch for the Cline frontend (does **not** disturb the Codex branch):
  ```go
  } else if plan.Task.Request.CLIKind == brain.CLIOpenAICompatible {
      clineDataDir := filepath.Join(env.RootDir, "cline-data")           // controlled, task-scoped
      contract, configErr := runtimeenv.NewClineConfigContract(
          r.config.Neutral.Gateway.BaseURL, plan.Task.Request.RouteModel, launchUpdatedAt) // see decision below
      if configErr != nil { return configErr }
      if err := execenv.WriteCredentiallessClineConfig(clineDataDir, contract.Bytes()); err != nil { return err }
      // manifest stays EMPTY: the carrier lives in CLINE_DATA_DIR, excluded from the task-home manifest (§7).
  }
  ```
- Wire the data dir into the environment: set `Adapter.ClineDataDir = clineDataDir` in the
  `runtimeenv.AdapterEnvironment{…}` literal (currently only sets `CodexHome: env.CodexHome`). The dir
  must be created (0700) before `BuildGatewayEnvironment` validates it — create it in this branch
  (mirror the `taskHome` mkdir pattern at the top of `buildLaunch`), or before the `WithCredential`
  closure.
- `AssertPreLaunch` is called with `CodexConfig: nil` and an **empty** `manifest` for Cline; the Cline
  carrier is validated by `contract.Validate()` inside `NewClineConfigContract` + tamper-checked by
  `ValidateClineConfigBytes`, so it does not need the manifest path.
- **Decision needed (W1): `updatedAt` source.** `NewClineConfigContract` requires a non-empty
  `validHeaderValue(updatedAt)` (RFC3339-style, no control chars). Codex uses `Correlation`; Cline
  uses a timestamp string. Recommend a launch-time UTC value, e.g.
  `time.Now().UTC().Format(time.RFC3339)`, sourced from an injectable clock for deterministic tests.
  This is a materialization detail, **not** auth. Freeze the exact format with A1.
- Tests: D6-T (§6.5).

### D7 — `server/pkg/agent/models.go` · `clineStaticModels` — **GLM-only** (owner: W1) = A3 SUB-1
- OLD (`models.go:562`): `{ID: "cline-pass/glm-5.2", Label: "GLM-5.2", Provider: "cline-pass"},`
- NEW: `{ID: "cp/cline-pass/glm-5.2", Label: "GLM-5.2", Provider: "cline-pass"},`
- Rationale: the UI/discovery catalog ID must equal the A3-frozen registry `RouteModel` so a picked
  "GLM-5.2" resolves to an admissible `RouteModel`. Update `models_test.go:172` expectation in lockstep.
- **Do NOT touch line 563** (Kimi row) — see §5. **Do NOT touch line 572** (`z-ai/glm-5.2` NIM-direct
  default) — that is the A3 REVIEW-1 obsolete-direct-alternative decision (out of this delta's scope).

---

## 5. KIMI-BLOCKED (5.7) — no separate implementation; ID + fixtures + live run only

The frontend wiring D1–D6 is model-agnostic and **already covers Kimi**. The Kimi route needs
**zero additional wiring code**. The only Kimi-specific items are all gated on **A3 BLK-KIMI**:

| Item | File:symbol | Blocked on | Action when unblocked |
|---|---|---|---|
| K1 canonical Kimi `RouteModel` | (external — OmniRoute registry) | A3 BLK-KIMI (owner: OmniRoute architect via Principal) | publish exact versioned Kimi model ID + protocol=openai-chat + availability |
| K2 test route fixture | `runtimeenv/cline_test.go:19` `clineAgentBrainRoutes[0]` = `"cline-kimi-k2.7-dedicated"` (combo **alias**, not exact model) | K1 | replace with the K1 exact ID |
| K3 static catalog Kimi row | `server/pkg/agent/models.go:563` `"cline-pass/kimi-k2.7-code"` (unattested) | K1 | replace with the K1 exact ID; update mirroring tests (`models_test.go:172`, `handler/agent_thinking_test.go:164`) |
| K4 availability | OmniRoute enriched registry row | A3 BLK-AVAIL | OmniRoute publishes enriched row `available=true` |
| K5 single live run | Kanban→terminal (reserved token) | K1+K4 + D1–D6 integrated | one live run closes 5.7/8.1/8.2 for the Kimi family |

**Hard rule (from A3 + A1-A2 §7):** do **not** select a Kimi ID by plausibility; do **not** map the
Kimi route to `brain.CLIKimi`; do **not** reuse `claude_code_kimi_2.7_Code` (that is the Claude Code
family). A model whose *name* contains "kimi" is not the `CLIKimi` native frontend.

**GLM independence:** none of D1–D7 depends on K1–K5. GLM-5.2 is fully implementable and live-runnable
now; Kimi rides the same rails the moment K1 lands.

---

## 6. Test delta (focused; author with each owning edit; run only affected package)

Existing green, no change (both routes already exercised, no-secret/NVIDIA-reject/tamper):
`runtimeenv/cline_test.go` `TestNewClineConfigContract*`, `TestValidateClineConfigBytesDetectsTamper`.
(Their GLM fixture `cp/cline-pass/glm-5.2` is already canonical; the Kimi fixture is K2 above.)

New focused tests:
- **6.1 D1-T** `runtimeenv/adapter_test.go`: `CredentiallessAdapterContract(brain.CLIOpenAICompatible)`
  returns `State==AdapterReady`, `Protocol==ProtocolOpenAIChat`, `err==nil`; `CLIKimi/CLINIM/CLIAntigravity`
  still fail-closed (regression guard for §7).
- **6.2 D2-T** `runtimeenv/env_test.go`: `BuildGatewayEnvironment` with `AdapterEnvironment{CLI:
  CLIOpenAICompatible, ClineDataDir:<ctrl dir>, …}` injects `CLINE_DATA_DIR` + `CLINE_OMNIROUTE_API_KEY`
  as trusted-last; secret absent from `String()`/`Keys()`, present only in `Exec()`; a
  custom/local attempt to set either key is rejected pre-merge.
- **6.3 D5-T** `execenv` test: `WriteCredentiallessClineConfig` writes `0600` `settings/providers.json`
  under a `0700` dir at the frozen subpath; rejects empty/oversized/relative/root paths.
- **6.4 D3-T/D4-T** `internal/daemon` `config_test.go`: `agentBrainBuiltInCLIFor(CLIOpenAICompatible)`
  == `{Provider:"cline",Command:"cline"}`; `AgentBrainIntegrationConfig.Validate()` accepts
  `CLIOpenAICompatible` in gateway/tier-20 mode and still rejects unknown kinds.
- **6.5 D6-T** `internal/daemon` `brain_integration_test.go`: `buildLaunch` for `CLIOpenAICompatible`
  writes `providers.json` into `CLINE_DATA_DIR`, passes an **empty** task-home manifest (carrier
  excluded), and `AssertPreLaunch` succeeds with `CodexConfig==nil`.
- **6.6 D7-T** `server/pkg/agent/models_test.go`: `clineStaticModels()` GLM row == `cp/cline-pass/glm-5.2`
  (update existing `:172` expectation).
- **6.7 policy** `runtimeenv/model_test.go`: `GatewayModelPolicy.ValidateSelection(CLIOpenAICompatible,
  brain.RouteModel("cp/cline-pass/glm-5.2"), "")` passes once D1 lands. (Kimi variant added with K1.)

No broad regression, no second QA, no duplicate live run. Verify per package:
```
cd multica-auth-work/server
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/runtimeenv/...
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/execenv/...
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/...
/home/ec2-user/goroot/go/bin/go test ./pkg/agent/...
/home/ec2-user/goroot/go/bin/gofmt -l internal/daemon/runtimeenv internal/daemon internal/daemon/execenv pkg/agent
```

---

## 7. Non-negotiable invariants W1 must preserve

- No secret in `providers.json` — `apiKey` stays `ClineSecretReferenceSentinel`;
  `ValidateClineConfigBytes` enforces it (rejects any concrete value).
- `CLINE_DATA_DIR` + `CLINE_OMNIROUTE_API_KEY` injected **only** as trusted entries (never inherited/local/custom).
- Carrier lives in `CLINE_DATA_DIR`, **excluded** from the task-home manifest (`home.go`
  `ValidateTaskHomeManifest` forbids `providers.json`/`.cline`).
- Keep `CLIKimi`/`CLINIM`/`CLIAntigravity` fail-closed; **NVIDIA (`nvidia/*`) stays OmniRoute-only**
  and Brain-non-selectable (`cline.go` `isNVIDIAOwnedRoute` already rejects it).
- Both Cline routes share one CLIKind/profile/contract; only `RouteModel` differs.
- **No auth/credential/quota/retry/failover** added to the Brain (OmniRoute-owned).
- `${ENV}` expansion open item (A1-A2 §5, BLOCKED_EXTERNAL/behavior): confirm Cline 3.0.44 expands
  `${CLINE_OMNIROUTE_API_KEY}` in `settings.apiKey` **before** the reserved live run. If it does not
  and write-time substitution is disallowed (it is, by the no-embed invariant), this blocks the live
  run only — not the D1–D7 build.

---

## 8. Integration order for W1 (serial)

1. **D1** adapter flip (smallest; unblocks env + policy).
2. **D2** trusted env injection (+ `AdapterEnvironment.ClineDataDir` field).
3. **D5** `WriteCredentiallessClineConfig` (new execenv file).
4. **D6** `buildLaunch` Cline branch (creates dir, writes carrier, empty manifest, `updatedAt` decision).
5. **D3 + D4** config executable mapping + validation accept.
6. **D7** static catalog GLM prefix (A3 SUB-1) + test lockstep.
7. GLM live run (reserved single token) → closes 5.6/8.1/8.2 for the GLM family.
8. When A3 BLK-KIMI clears: apply K2/K3 (one-line fixtures), then the Kimi single live run.

---

## 9. Blockers / limitations

- **BLOCKED_EXTERNAL (A3 BLK-KIMI):** exact Kimi `RouteModel` undeclared → 5.7 selection/live run
  blocked; wiring (D1–D6) is shared and unaffected. Owner: OmniRoute architect via Principal.
- **BLOCKED_EXTERNAL (A3 BLK-AVAIL):** enriched OmniRoute registry rows (GLM + Kimi) unpublished →
  runtime admissibility of `cp/cline-pass/glm-5.2` pending; the D1–D7 build is independent of this.
- **BLOCKED_EXTERNAL (behavior, A1-A2 §5):** Cline `${ENV}` expansion in `settings.apiKey` unproven
  in-repo → blocks the live run only.
- This phase is **READ-ONLY**; all D/K items are handoff intent for W1/A1, not applied edits.
- Live acceptance additionally gated by D-V3-25(B) security stop (key revocation) per
  `authoritative-route-matrix-D-V3-27.md` — out of this lane's scope, recorded for W1.

---

## 10. Agent status block

- STATUS: DONE (read-only design; exact minimal patch plan)
- DELIVERED: root-cause of fail-closed (§3); 7-edit implementable GLM/shared delta with exact
  old→new per file:symbol (§4, D1–D7); Kimi external-blocker isolation with zero extra code (§5,
  K1–K5); focused test delta (§6); invariants (§7); W1 serial order (§8); blockers (§9).
- FILES: `.deploy-control/p0/handoffs/R1-cline-route-source-delta.md` (only mutable file).
- VALIDATION: none run (no product code changed); exact focused commands in §6. Old-behavior snippets
  verified by direct source AST lookup (agentBrainBuiltInCLIFor, Validate, trustedAdapterEntries,
  buildLaunch, WriteCredentiallessCodexConfig, AdapterEnvironment, AgentBrainIntegrationConfig).
- EVIDENCE: this artifact; `A1-A2-cline-foundation.md`; `A3-route-freeze.md`; source citations by file:line.
- BLOCKERS/LIMITATIONS: §9.
- W1_HANDOFF: §4 (D1–D7), §8 (order), §7 (invariants). GLM implementable now; Kimi one-line fixtures on BLK-KIMI clear.
