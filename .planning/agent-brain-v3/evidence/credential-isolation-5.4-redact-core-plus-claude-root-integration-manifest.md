# Root integration manifest — 5.4 redaction-core + minimal Claude logical unit

- Author: independent (Kiro/Opus-4.8), pane **w7:p2**. **Read-only manifest.** Constructs nothing: no worktree,
  no commit, no git/index/ref/product/shared/spec/task mutation. It is a recipe for **root** to execute later.
- Purpose: define the exact overlays + generated minimal patch to build an integration commit in a **separate
  pristine worktree**, taking the 5.4 logical unit **without** unrelated working-tree `claude.go` hunks.
- Reuses the clean-room proofs (see Evidence). **Technical candidate only — not acceptance.**
- Base: HEAD `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`. Toolchain for verification:
  `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, offline.

## Logical unit (4 files) — exact overlays + generated patch

| # | Path | Action | SHA-256 | Current git state |
|---|---|---|---|---|
| 1 | `multica-auth-work/server/pkg/redact/redact.go` | **overlay** current bytes | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` | modified (`M`) |
| 2 | `multica-auth-work/server/pkg/redact/redact_test.go` | **overlay** current bytes | `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9` | modified (`M`) |
| 3 | `multica-auth-work/server/pkg/agent/claude.go` | **generated minimal patch only** (not overlay) | target `c7922b7bf92826f5b7889c91ef4d710700be602ef69ad0156044ffded9d5ede9` | modified (`M`, carries unrelated WIP) |
| 4 | `multica-auth-work/server/pkg/agent/claude_log_writer_redaction_test.go` | **overlay** (new file) | `81d3e8659125df2fda25c27da16b24d88e1861282255e41f46c33f7cae2ae40a` | untracked (`??`) |

Files 1, 2, 4 are shipped as whole current bytes. File 3 is **NOT** overlaid from the working tree (it contains
unrelated argv/env WIP); instead the two-hunk delta below is applied to the **HEAD** blob.

## Generated minimal `claude.go` patch (HEAD → patched)

- Patch base: HEAD `claude.go` blob `41d7ac9cf9aa040b5ee7b767ec628608ab601c4d` (anchors verified present:
  import block close at line 15; `w.logger.Debug(w.prefix + text)` at line 854).
- Target patched-file SHA-256: `c7922b7bf92826f5b7889c91ef4d710700be602ef69ad0156044ffded9d5ede9`.
- Semantic hunks (verbatim; the only two):

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

(A captured `diff -u` file hashes non-deterministically because it embeds tmp paths/timestamps; pin File 3 by the
**target patched-file hash** `c7922b7b…`, not by a diff-file hash.)

## Exclusions (MUST NOT enter the integration commit)

- Working-tree `pkg/agent/claude.go` in full (argv redaction WIP: `redactedAgentArgValue`, `path/filepath`).
- Untracked pkg/agent WIP: `environment.go`, `environment_test.go`, `models_process_test.go`,
  `models_windows_test.go`, `proc_unsupported.go`; modified `models.go`, `claude_test.go`.
- Any other credential-isolation task files (4.1/4.2/4.3/4.4/5.2/5.3 sources) — out of this unit.
- No `.env`, auth home, config, credential, or environment-value file. No shared docs/ledger/EVIDENCE_INDEX/STATE,
  OpenSpec spec/`tasks.md`/checkbox.

## Construction recipe (for ROOT to run later — NOT executed here)

In a **separate pristine worktree** off HEAD (illustrative; root owns exact invocation):
1. `git worktree add <path> b6571299` (new worktree; does not touch the working repo/index).
2. Overlay files 1, 2, 4 by their pinned hashes.
3. Regenerate file 3 by applying the two hunks above to the HEAD `claude.go` blob (`41d7ac9c…`); verify result
   hashes to `c7922b7b…`.
4. Stage exactly these four paths; confirm `git status`/`git diff --cached --stat` shows **only** them.
5. Commit (message per root convention); push only under explicit root/GH auth.

## Verification gates (run in the worktree before commit; must all pass)

- Manifest rehash gate: `sha256sum` files 1,2,3,4 == the four hashes above (esp. claude.go == `c7922b7b…`).
- Diff-scope gate: `git diff --cached --name-only` == exactly the four paths; the claude.go staged diff == the two
  hunks above and nothing else (`grep -c "redactedAgentArgValue\|path/filepath\|environment\." pkg/agent/claude.go` = 0).
- Secret gate: no sentinel/secret in staged diff (`git diff --cached | grep -nE "sk-|Bearer |access_token|PASSWORD="`
  should show only synthetic test sentinels inside `claude_log_writer_redaction_test.go`/`redact_test.go`, never real values).
- Build/test gates (offline, pinned): `gofmt -l` (4 files) clean; `go build ./pkg/redact/ ./pkg/agent/` exit 0;
  `go vet ./pkg/redact ./pkg/agent` exit 0; **full** `go test ./pkg/redact` ok and `go test ./pkg/agent` ok;
  named redact tests `-count=20` = 60 PASS; six Claude logWriter tests `-count=20` = 120 PASS; both `-race` ok.
  (All reproduced in the clean-room proof below.)

## Evidence (already produced; read-only)

- `credential-isolation-5.4-claude-stderr-clean-room-isolated-patch.md` — proves the 2-file claude patch FAILS on
  HEAD (dependency on core `redact.go`); failing test `TestLogWriterRedactsErrorBodyTokenField`.
- `credential-isolation-5.4-redact-core-plus-claude-clean-room-atomic-review.md` — proves this 4-file unit builds,
  vets, gofmt-clean, and passes full `pkg/redact` + full `pkg/agent` + named(60)/six(120) ×20 + race on pristine HEAD.
- `credential-isolation-email-log-safety-review.md`, `credential-isolation-redact-core-review.md`,
  `credential-isolation-redact-core-fix.md` — prior 5.4 slice/core evidence context.

## Pending gates (integration BLOCKED until all clear)

1. **GLM core review** of `pkg/redact` 5.4-core — PENDING.
2. **Independent expanded review** (distinct reviewer, not this author) of the 4-file unit — PENDING.
3. **Kiro TL adjudication** / whole-task 5.4 acceptance — PENDING (5.4 remains OPEN; this unit is a slice+core, not the whole task).
4. **Root / GitHub auth** for worktree creation, staging, commit, and push — PENDING; root-controlled.

## Provenance / non-claims
- Pane **w7:p2**; independent author; **no self-acceptance**, sets no checkbox, authorizes no push. Read-only:
  no git/index/ref mutation, no worktree/commit created, no product/shared/spec/task edit; no credentials/env values;
  no DB/network/services. Hashes/anchors verified against HEAD `b6571299` at authoring time; root must re-run the
  rehash + diff-scope + secret + build/test gates in the pristine worktree before committing.
