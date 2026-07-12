# CHECKIN Agent-3 Cline-Core DONE

- UTC: `2026-07-12T05:56:00Z`
- Change: `native-runtimes-onboarding`
- Task: `1.3`
- Status: DONE

## Implemented

- Added `multica-auth-work/server/pkg/agent/cline.go`.
- Added `multica-auth-work/server/pkg/agent/cline_test.go`.
- Native launch contract: `cline --acp --json`.
- Reused the shared `hermesClient` ACP JSON-RPC lifecycle used by Kimi/Kiro.
- Supports initialize, session new/resume, model selection, prompt streaming, session ID, token usage, MCP conversion, permission approval, provider-error promotion, timeout/cancellation, and Cline tool-name normalization.
- Passes `ThinkingLevel` through `--thinking` and blocks custom overrides of `--acp`, `--json`, and `--thinking`.
- Integrates existing Cline credential isolation through `Config.Env`: the backend preserves `CLINE_DATA_DIR` and `CLINE_SANDBOX_DATA_DIR` prepared by `execenv/cline_home.go`.
- Integrates existing reactive rotation signals by promoting ACP/provider rate-limit failures through `newACPProviderErrorSniffer("cline")`; daemon screen/output detection remains owned by `rotation_detector_cline.go`.
- Marked OpenSpec task `1.3` complete.

## Container Evidence

Green, Go `1.26.4` container (`golang:1.26-alpine`):

```text
go test ./pkg/agent -run '^TestCline' -count=1
ok github.com/multica-ai/multica/server/pkg/agent 0.035s

go vet ./pkg/agent
PASS (exit 0)
```

An attempted broad `go test ./pkg/agent -count=1` reached an unrelated concurrent failure in `TestCodexStaticModelsExposesGPT55`: `models.go`/`models_test.go` were already modified by Agent-4 and currently disagree on the GPT-5.5 default. No Cline test failed.

## Wiring Patch For Kiro (Wave 2)

Do not apply from this task; shared files were intentionally untouched.

1. `server/internal/daemon/config.go`
   - Probe `MULTICA_CLINE_PATH`, default executable `cline`, model env `MULTICA_CLINE_MODEL`.
   - Register the result as `agents["cline"]`.
   - Add `cline` to the no-agent-found guidance and executable lists where applicable.
2. `server/pkg/agent/agent.go`
   - Add `cline` to `SupportedTypes`.
   - Add `case "cline": return &clineBackend{cfg: cfg}, nil` in `New`.
   - Add `cline` to the `New` error/support text and package/config comments.
   - Add launch header `"cline": "cline --acp --json"`.
3. Keep existing daemon credential wiring for `cline`: `requiresCredentialIsolation` already includes it and `execenv.CredentialEnv("cline")` emits the isolated data/sandbox variables.

## Ownership Check

- `server/internal/daemon/config.go`: not edited.
- `server/pkg/agent/agent.go`: not edited.
- `requiresCredentialIsolation`: not edited.
