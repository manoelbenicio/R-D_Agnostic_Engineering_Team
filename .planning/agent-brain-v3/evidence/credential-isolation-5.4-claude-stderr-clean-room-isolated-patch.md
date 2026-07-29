# Clean-room 5.4 isolated-patch proof — claude stderr logWriter redaction

- Verifier: independent clean-room (Kiro/Opus-4.8), operating pane **w7:p2** (per assignment). Read-only vs the
  repository; a private `/tmp` copy of committed HEAD was used and removed.
- Scope: prove whether the task-5.4 claude stderr slice is a **dependency-complete isolated patch** =
  (HEAD `claude.go` + minimal redact hunk) + current `claude_log_writer_redaction_test.go`, applied to pristine HEAD.
- **Does not self-accept; Kiro TL adjudicates; root integrates.**
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; offline.

## Result: **CLEAN-ROOM CONDITION FAILS (as a 2-file patch).**

The 2-file patch is `gofmt`/`vet`/`build`-clean on pristine HEAD, but **`go test` fails**: 1 of the 6 focused
tests (`TestLogWriterRedactsErrorBodyTokenField`) fails every iteration because **HEAD `pkg/redact/redact.go`
lacks the credential-bearing JSON-field regex** (`"access_token":"…"`). That regex is part of the untracked/modified
5.4-core `redact.go`. **The isolated patch is therefore NOT dependency-complete as two files; it also requires the
5.4-core `redact.go`** (EV-CREDISO-5.4-CORE, `f409ba8a…`). It does **not** depend on untracked `environment.go` or
argv WIP.

## Method (no repository git/index mutation)

1. `CR=$(mktemp -d /tmp/credroom.5.4.XXXXXX)` → `/tmp/credroom.5.4.MFQXzh`.
2. `git -C <repo> archive HEAD multica-auth-work/server | tar -x -C "$CR"` (committed HEAD, tracked files only).
   Saved pristine `claude.go.HEAD` for diffing.
3. Verified HEAD archive **lacked** `claude_log_writer_redaction_test.go`, `environment.go`, and any
   `pkg/redact`/`redactedAgentArgValue` reference in `claude.go` (clean baseline; `environment.go` is untracked, not at HEAD).
4. Applied **exactly the 5.4 delta** to the temp HEAD `claude.go`: added the `pkg/redact` import group and changed
   only the `logWriter.Write` log call to wrap `text` in `redact.Text(...)`, preserving `w.prefix` and `return len(p)`.
5. Overlaid **only** the current `claude_log_writer_redaction_test.go`. Did **not** overlay the working-tree
   `claude.go` (argv/env WIP) or any other WIP.
6. Verified the temp `claude.go` diff contained **only** the two intended hunks; ran the offline suite.
7. Supplementary diagnostic: overlaid current `redact.go` to pinpoint the exact missing dependency.
8. Removed only `/tmp/credroom.5.4.MFQXzh` (confirmed gone).

## Overlay / patch hashes (SHA-256)

| Item | SHA-256 |
|---|---|
| temp patched `claude.go` (HEAD + 2 hunks; NOT working-tree claude.go) | `c7922b7bf92826f5b7889c91ef4d710700be602ef69ad0156044ffded9d5ede9` |
| overlaid `claude_log_writer_redaction_test.go` (current) | `81d3e8659125df2fda25c27da16b24d88e1861282255e41f46c33f7cae2ae40a` |
| captured `claude_5.4.patch` file (diff -u; includes ephemeral tmp paths/timestamps) | `48452b1ed5653133d71ebb6131d788d6c3614ba4dfdbc0043ac87c9fb44bc280` |
| working-tree `redact.go` overlaid in diagnostic (== EV-CREDISO-5.4-CORE) | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` |

## The isolated `claude.go` diff (verbatim; the only two hunks)

```diff
@@ -12,6 +12,8 @@
 	"strings"
 	"sync"
 	"time"
+
+	"github.com/multica-ai/multica/server/pkg/redact"
 )
 
 // claudeBackend implements Backend by spawning the Claude Code CLI
@@ -851,7 +853,7 @@
 func (w *logWriter) Write(p []byte) (int, error) {
 	text := strings.TrimSpace(string(p))
 	if text != "" {
-		w.logger.Debug(w.prefix + text)
+		w.logger.Debug(w.prefix + redact.Text(text))
 	}
 	return len(p), nil
 }
```

`grep -c "redactedAgentArgValue\|path/filepath\|environment\."` on the patched file = **0** (no argv/env WIP leaked).

## Commands & results (working dir = temp `multica-auth-work/server`)

Primary — HEAD + 2 files only:
- `gofmt -l pkg/agent/claude.go pkg/agent/claude_log_writer_redaction_test.go` → clean (exit 0).
- `go build ./pkg/agent/` → **exit 0** (compiles; import resolves against HEAD redact.go).
- `go vet ./pkg/agent` → exit 0.
- `go test ./pkg/agent -run <6 tests> -v -count=20` → **100 `--- PASS` / 20 `--- FAIL`**, package **FAIL**.
- `go test ./pkg/agent -run <6 tests> -race -count=20` → **FAIL**.
- Failing test: `TestLogWriterRedactsErrorBodyTokenField` (`claude_log_writer_redaction_test.go:76`):
  `synthetic error-body token field leaked … "…access_token…synthetic-error-body-token-sentinel…"`.
- Root cause (verified): HEAD `pkg/redact/redact.go` (97 lines) has only `sk-`/`Bearer`/older patterns; it has
  **no** `"access_token":"…"` JSON-field regex. Working-tree `redact.go` (269 lines) adds it.

Supplementary diagnostic — same clean room + current `redact.go` overlaid (`f409ba8a…`):
- `go test ./pkg/agent -run '^TestLogWriterRedactsErrorBodyTokenField$'` → **PASS**.
- `go test ./pkg/agent -run <6 tests> -v -count=20` → **120 `--- PASS`**, package **ok**.
- Confirms the missing dependency is **precisely** `pkg/redact/redact.go` at the 5.4-core version, nothing else.

## Dependency conclusion

- The claude stderr 5.4 slice is **not** an atomic 2-file patch. Its **minimal dependency-complete unit is 3 files**:
  1. `pkg/agent/claude.go` — the 2-hunk delta above,
  2. `pkg/agent/claude_log_writer_redaction_test.go` — the overlaid test,
  3. `pkg/redact/redact.go` — the 5.4-core (`f409ba8a…`, EV-CREDISO-5.4-CORE).
- It does **not** depend on untracked `environment.go`, `proc_unsupported.go`, `models.go` WIP, or argv redaction:
  the patched HEAD `claude.go` built and vetted cleanly without them, and the only failure was the redact-pattern gap.
- Whole-task 5.4 remains OPEN regardless (per prior EV-CREDISO-5.4 reviews); this proof concerns only the claude
  stderr slice's isolated build/behavior completeness.

## Provenance / non-claims
- Pane **w7:p2**; independent verifier; no self-acceptance. Read-only vs repo: no product/spec/task/shared-doc/
  git/index edit; no credentials/env values; no DB/network/services. The private HEAD copy under
  `/tmp/credroom.5.4.MFQXzh` was created from committed HEAD and removed after validation (confirmed gone). Only
  the current test file (and, in the labeled diagnostic, the current `redact.go`) were overlaid; the working-tree
  `claude.go` was never overlaid. Kiro TL adjudicates; root controls any push.
