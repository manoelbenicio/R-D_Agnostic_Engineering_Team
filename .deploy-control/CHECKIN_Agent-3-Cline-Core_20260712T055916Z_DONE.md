# CHECKIN Agent-3 Cline-Core DONE

- UTC: `2026-07-12T05:59:16Z`
- OpenSpec: `native-runtimes-onboarding`, task `1.3`
- Implementation commit: `58e1f24 feat(agent): add native cline ACP backend`
- Status: DONE

## Delivered

- `multica-auth-work/server/pkg/agent/cline.go`
  - Native `cline --acp --json` process backend.
  - Reuses the shared Hermes/Kimi ACP JSON-RPC client and lifecycle.
  - Supports initialize, new/resumed sessions, model selection, MCP servers,
    streamed messages/tool events, token usage, timeout/cancellation, upstream
    provider error promotion, and native `--thinking` effort selection.
  - Preserves `Config.Env`, integrating the existing Cline isolation variables
    `CLINE_DATA_DIR` and `CLINE_SANDBOX_DATA_DIR` prepared by `cline_home.go`.
  - Provider/quota failures are surfaced through the shared ACP error sniffer,
    allowing existing `rotation_detector_cline.go` handling to observe them.
- `multica-auth-work/server/pkg/agent/cline_test.go`
  - Covers native argv, blocked protocol overrides, ACP lifecycle, streaming,
    model/thinking selection, token usage, malformed MCP configuration, and
    propagation of isolated Cline environment variables.
- `openspec/changes/native-runtimes-onboarding/tasks.md`
  - Task `1.3` marked complete.

## Green-In-Container Evidence

Image: `golang:1.26-alpine` (`Go 1.26.4`). Working directory mounted at
`multica-auth-work/server`.

```text
$ go build ./...
exit 0

$ go test ./pkg/agent/ -count=1
ok github.com/multica-ai/multica/server/pkg/agent 5.532s
```

## Wave 2 Wiring Patch For Kiro

Apply to the shared files after the Wave 1 backends have landed:

```diff
diff --git a/multica-auth-work/server/internal/daemon/config.go b/multica-auth-work/server/internal/daemon/config.go
@@
 	if e, ok := probe("MULTICA_KIRO_PATH", "kiro-cli", "MULTICA_KIRO_MODEL"); ok {
 		agents["kiro"] = e
 	}
+	if e, ok := probe("MULTICA_CLINE_PATH", "cline", "MULTICA_CLINE_MODEL"); ok {
+		agents["cline"] = e
+	}

diff --git a/multica-auth-work/server/pkg/agent/agent.go b/multica-auth-work/server/pkg/agent/agent.go
@@
 var SupportedTypes = []string{
@@
 	"kiro",
+	"cline",
 	"antigravity",
 }
@@
 	case "kiro":
 		return &kiroBackend{cfg: cfg}, nil
+	case "cline":
+		return &clineBackend{cfg: cfg}, nil
 	case "antigravity":
@@
 	"kiro":        "kiro-cli acp",
+	"cline":       "cline --acp --json",
```

Also update the adjacent supported-provider comments and the `New` unknown-type
error text to include `cline`, plus the `config.go` no-agent-found message and
executable-name list where those enumerate providers.

## Shared-File Confirmation

Commit `58e1f24` contains only the Cline backend/test, its START/DONE check-ins,
and the OpenSpec task checkbox. It contains no path matching either shared file.

- `multica-auth-work/server/internal/daemon/config.go`: NOT TOUCHED.
- `multica-auth-work/server/pkg/agent/agent.go`: NOT TOUCHED.
