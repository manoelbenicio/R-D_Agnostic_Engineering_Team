# R1 — Cline focused-test IMPLEMENTATION (DONE)

- agent: `Opus48#A`  ·  lane: `A9-R1`  ·  task: `P0-R1-TEST-IMPLEMENTATION`  ·  pane: `w6:p1`
- control lock: `.deploy-control/p0/handoffs/R1-test-implementation.md`
- product test locks (edited under lock): 5 files in `internal/daemon/runtimeenv/`
- authorization: `implementation_authorized=true`, phase `IMPLEMENTATION_W1_SERIAL`; W1 **D1+D2 APPLIED & greened** signal received and independently verified in source.
- created: 2026-07-21T23:20Z (UTC) · toolchain `/home/ec2-user/goroot/go/bin/{go,gofmt}` (go1.26.1)

---

## 1. D1+D2 source verified before editing tests (read-only)

- `runtimeenv/adapter.go` — `CredentiallessAdapterContract(brain.CLIOpenAICompatible)` returns
  `AdapterContract{State: AdapterReady, Protocol: brain.ProtocolOpenAIChat}, nil`; Kimi/NIM/Antigravity
  remain `AdapterFailClosed` with their gates; `default` → `GateOpenAICompatibleUnaccepted`.
- `runtimeenv/env.go` — `AdapterEnvironment.ClineDataDir` field added; `trustedAdapterEntries` has a
  `case brain.CLIOpenAICompatible` that validates `ClineDataDir` as a physical dir and injects
  `CLINE_DATA_DIR` (trusted-local) + `CLINE_OMNIROUTE_API_KEY` (= `ClineOmniRouteAPIKeyEnv`,
  trusted-secret), returning `secretKey="CLINE_OMNIROUTE_API_KEY"`.

Tests below were written to match this **actual** applied contract.

## 2. Exact edits (only the 5 locked files; cline_test.go intentionally unchanged)

| File | Change |
|---|---|
| `runtimeenv/adapter_test.go` (**new**) | `TestOpenAICompatibleAdapterReadyForOmniRouteChat` — asserts Ready + `ProtocolOpenAIChat`, empty gate, no fallback frontends (D1 positive proof). |
| `runtimeenv/model_test.go` | **Flip**: removed `{CLIOpenAICompatible, GateOpenAICompatibleUnaccepted}` row from `TestCredentialBearingNativeAdaptersFailClosed` (kept Kimi/NIM/Antigravity). **Add**: `TestGatewayModelPolicyAcceptsClineGLMOverChat` (A3-frozen `cp/cline-pass/glm-5.2` over Chat; fail-closed on wrong CLI / thinking / unknown model) + `TestGatewayModelPolicyClineKimiHeldPendingExactID` (`t.Skip` BLK-KIMI, no id invented). |
| `runtimeenv/env_test.go` | **Add**: `TestBuildGatewayEnvironmentClineUsesDedicatedKeyName` (trusted-last `CLINE_DATA_DIR` + `CLINE_OMNIROUTE_API_KEY`, inherited untrusted dropped, secret redacted in diagnostics) + `TestClineTrustedKeysRejectedFromCustomEnvironment` (deny-list rejects both keys via custom). |
| `runtimeenv/isolation_g4_test.go` | **Flip**: removed `{CLIOpenAICompatible, GateOpenAICompatibleUnaccepted}` row from `TestG4NativeCredentialBearingAdaptersStayFailClosed` (kept Kimi/NIM/Antigravity fail-closed with zero child env). |
| `runtimeenv/cline_test.go` | **No change** (locked; contract unaffected by D1/D2; already green). No Kimi id invented. |

## 3. Validation — exact command & result

gofmt (touched files): `gofmt -w …` then `gofmt -l` over all 5 locked files → **clean (empty)**.

Command (exactly as directed, from `multica-auth-work/server`):
```
/home/ec2-user/goroot/go/bin/go test ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4' -count=1
```
Result:
```
ok  github.com/multica-ai/multica/server/internal/daemon/runtimeenv  0.014s   (exit 0)
```

Per-test (verbose selector run):
- **NEW** `TestOpenAICompatibleAdapterReadyForOmniRouteChat` — PASS
- **NEW** `TestBuildGatewayEnvironmentClineUsesDedicatedKeyName` — PASS
- **NEW** `TestClineTrustedKeysRejectedFromCustomEnvironment` — PASS
- **NEW** `TestGatewayModelPolicyAcceptsClineGLMOverChat` — PASS
- **NEW** `TestGatewayModelPolicyClineKimiHeldPendingExactID` — SKIP (intended, BLK-KIMI)
- **FLIPPED** `TestCredentialBearingNativeAdaptersFailClosed` (kimi/nim/antigravity) — PASS
- **FLIPPED** `TestG4NativeCredentialBearingAdaptersStayFailClosed` (kimi/nim/antigravity) — PASS
- unaffected existing (regression-safe): all `TestNewClineConfigContract*`, `TestValidateClineConfigBytesDetectsTamper`, `TestBuildGatewayEnvironment{Claude,Codex,RejectsNoncanonicalTrustedHomes}`, `TestG4{ClaudeAndCodexTrustedGatewayProtocols,ClaudeAndCodexCancellationAndDeterministicErrors,ChildHomeProcessTreeAndLogIsolation}`, `TestGatewayModelPolicyAcceptsApprovedOmniRouteIDWithoutNativeDiscovery`, `TestNativeFallbackIsNeverAutomatic` — PASS

## 4. Scope discipline

- Edited **only** the locked files; no unplanned file touched; no other package built/tested; no live run.
- Kimi/NIM/Antigravity remain deterministically fail-closed (5.7 boundary: Cline→Kimi uses the Cline
  OpenAI-compatible frontend, never native `CLIKimi`).
- No Kimi RouteModel invented (BLK-KIMI held via `t.Skip`).
- Requirements advanced (offline): 5.6 (Cline→GLM adapter/env/model-policy), 8.1 (protocol/model), 8.2
  (deterministic-error half). Availability + tools/reasoning/usage/terminal remain for the single
  reserved GLM live run (BLK-AVAIL / D-V3-25(B)); Kimi 5.7 exact-model + live remain external-blocked.

## 5. Agent status block

- STATUS: DONE
- DELIVERED: 1 new + 4 edited-in-place focused runtimeenv tests implementing the D1 flip and the
  accepted-Cline adapter/env/model-policy coverage; all green; Kimi held (no id invented).
- FILES: control `R1-test-implementation.md` + `runtimeenv/{adapter_test.go(new),model_test.go,env_test.go,isolation_g4_test.go}`; `cline_test.go` locked-unchanged.
- VALIDATION: `go test ./internal/daemon/runtimeenv/ -run 'Adapter|Env|Model|Cline|G4' -count=1` → `ok 0.014s` (exit 0); `gofmt -l` clean on all 5 locked files.
- EVIDENCE: this artifact (§3 command/result + per-test list).
- BLOCKERS/LIMITATIONS: none for this focused delta. Downstream external blockers unchanged: BLK-KIMI (Kimi exact id), BLK-AVAIL (enriched registry), D-V3-25(B) (live-run security stop).
