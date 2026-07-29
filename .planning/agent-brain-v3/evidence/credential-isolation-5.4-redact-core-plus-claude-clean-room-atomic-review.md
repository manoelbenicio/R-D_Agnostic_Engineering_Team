# Clean-room 5.4 atomic proof — redact core + claude stderr logWriter (4-file logical unit)

- Verifier: independent clean-room (Kiro/Opus-4.8), operating pane **w7:p2** (per assignment). Read-only vs the
  repository; a private `/tmp` copy of committed HEAD was used and removed.
- Builds on the prior finding (`…5.4-claude-stderr-clean-room-isolated-patch.md`): the claude stderr slice is
  **not** a 2-file patch; it depends on the 5.4-core `redact.go`. This proof tests the **4-file logical unit**.
- **Verdict is a technical candidate only.** The redact core still requires a distinct reviewer + Kiro TL
  adjudication; root integrates. **No self-acceptance.**
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; offline.

## Result: **CLEAN-ROOM PASSES for the 4-file logical unit (technical candidate).**

On pristine committed HEAD, overlaying only current `pkg/redact/redact.go` + `redact_test.go`, applying the same
minimal 2-hunk `claude.go` patch, and overlaying only `claude_log_writer_redaction_test.go`: gofmt/vet/build clean,
**full `pkg/redact` ok, full `pkg/agent` ok**, named redact tests 60/60, six Claude tests 120/120, all race-clean.
No dependency on `environment.go` or argv WIP.

## Method (no repository git/index mutation)

1. `mktemp -d /tmp/credroom.54atomic.XXXXXX` → `/tmp/credroom.54atomic.edBtiw`; `git archive HEAD
   multica-auth-work/server | tar -x`. Saved pristine `claude.go.HEAD`.
2. Baseline verified: HEAD `redact.go` = 97 lines (pre-5.4-core); HEAD `claude.go` had 0 redact/argv refs;
   `environment.go` and the claude test were **absent** from HEAD.
3. Overlaid **only** current `pkg/redact/redact.go` and `pkg/redact/redact_test.go`.
4. Applied the **same minimal 5.4 delta** to HEAD `claude.go` (redact import group + wrap `text` in
   `redact.Text(...)` in `logWriter.Write`, preserving `w.prefix` and `return len(p)`) — **not** the full WIP claude.go.
5. Overlaid **only** `claude_log_writer_redaction_test.go`.
6. Verified the generated `claude.go` diff = exactly the two hunks; verified `environment.go`/argv absence.
7. Ran the offline suite; removed only `/tmp/credroom.54atomic.edBtiw` (confirmed gone).

## Four-file logical manifest (SHA-256, in clean room)

| Logical file | SHA-256 | Origin |
|---|---|---|
| `pkg/redact/redact.go` | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` | current (== EV-CREDISO-5.4-CORE) |
| `pkg/redact/redact_test.go` | `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9` | current (== EV-CREDISO-5.4-CORE) |
| `pkg/agent/claude.go` | `c7922b7bf92826f5b7889c91ef4d710700be602ef69ad0156044ffded9d5ede9` | **generated** = HEAD + 2-hunk patch (NOT working-tree claude.go) |
| `pkg/agent/claude_log_writer_redaction_test.go` | `81d3e8659125df2fda25c27da16b24d88e1861282255e41f46c33f7cae2ae40a` | current |

- **Generated minimal claude.go patch** hash (captured `diff -u`, includes ephemeral tmp paths/timestamps →
  not byte-stable): `48a8c810b54a71c323a42ff4c64cef2ad51dc07f90d41dee5bbedfbba0e9ac51`. The **semantic** hunks are
  verbatim-stable and identical to the prior review:

```diff
@@ -12,6 +12,8 @@
 	"strings"
 	"sync"
 	"time"
+
+	"github.com/multica-ai/multica/server/pkg/redact"
 )
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

`grep -c "redactedAgentArgValue\|path/filepath\|environment\."` on patched claude.go = **0**; `environment.go` absent.

## Commands & actual counts (working dir = temp `multica-auth-work/server`)

- `gofmt -l` (4 files) → clean (exit 0).
- `go build ./pkg/redact/ ./pkg/agent/` → **exit 0**.
- `go vet ./pkg/redact ./pkg/agent` → **exit 0**.
- `go test ./pkg/redact` (full) → **ok**.
- `go test ./pkg/agent` (full) → **ok** (7.017s). `go test ./pkg/agent -run LogWriter` → **ok**.
  (The prior 2-file run's only failure — `TestLogWriterRedactsErrorBodyTokenField` — is resolved by the core.)
- Named redact tests `-run '^(TestSanitizeForLog|TestSanitizeSlogAttrThroughHandler|TestRedactCredentialFieldsInJSONBody)$'
  -v -count=20` → **60 `--- PASS`** (3 × 20).
- Six Claude tests `-run '^(TestLogWriterRedactsAPIKeySentinel|…BearerTokenSentinel|…ErrorBodyTokenField|
  …PreservesSafeStderrContent|…EmptyOrWhitespaceEmitsNothing|…ReturnedByteCountMatchesInputRegardlessOfRedaction)$'
  -v -count=20` → **120 `--- PASS`** (6 × 20), package **ok**.
- Six Claude tests `-race -count=20` → **ok** (1.252s). Named redact `-race -count=20` → **ok** (1.090s).

## Dependency boundaries

- The claude stderr 5.4 slice's minimal dependency-complete unit is **exactly these four logical files**:
  the 5.4-core `redact.go` (+ its `redact_test.go`), the 2-hunk `claude.go` delta, and the claude test.
- **Confirmed independent of**: untracked `pkg/agent/environment.go`, `proc_unsupported.go`, `models.go` WIP, and
  the argv-redaction WIP (`redactedAgentArgValue`/`path/filepath`). Patched HEAD `claude.go` built, vetted, and
  passed offline without any of them.
- **Integration caveat:** the working-tree `pkg/agent/claude.go` carries additional argv/env WIP and is **not** the
  file in this unit. An atomic 5.4 push must ship the **regenerated 2-hunk delta** (manifest hash `c7922b7b…`),
  not the full working-tree claude.go. `redact.go`/`redact_test.go` are shipped as the whole accepted-core files.

## Verdict

**Technical candidate: PASS** — the 4-file logical unit is dependency-complete, builds, vets, gofmt-clean, and
passes full `pkg/redact` + full `pkg/agent` + the named/six focused tests ×20 and race, on pristine HEAD.
**Not acceptance.** The redact 5.4-core still requires a **distinct independent reviewer** and **Kiro TL
adjudication**; whole-task 5.4 remains OPEN; **root controls integration**. This review authorizes nothing and
sets no checkbox.

## Provenance / non-claims
- Pane **w7:p2**; independent verifier; no self-acceptance. Read-only vs repo: no product/spec/task/shared-doc/
  git/index edit; no credentials/env values; no DB/network/services. The private HEAD copy under
  `/tmp/credroom.54atomic.edBtiw` was created from committed HEAD and removed after validation (confirmed gone).
  Only the three current files (redact.go, redact_test.go, claude test) were overlaid; the working-tree `claude.go`
  was never overlaid — its 2-hunk delta was regenerated onto HEAD.
