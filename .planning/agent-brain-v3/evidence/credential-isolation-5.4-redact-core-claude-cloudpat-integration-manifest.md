# Integration manifest — 5.4 redact-core + minimal Claude + Cloud PAT accepted slice

- Author: Kiro/Opus-4.8, pane **w7:p2**. **Read-only manifest.** Constructs nothing: no worktree/source/test/
  shared-planning/spec/tasks/git/index/ref mutation; no credentials/env/network/DB/services. Only this file created/edited.
- Extends the previously verified 4-file logical unit
  (`credential-isolation-5.4-redact-core-plus-claude-clean-room-atomic-review.md`) with the TL-accepted
  **EV-CREDISO-5.4-CLOUDPAT** slice.
- **Technical candidate only. No push authorization.** Owner waiver, cross-family Claude review, codebase-wide 5.4
  gates, and Kiro/root auth remain PENDING.
- Base: HEAD **`b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`**. Hashes = SHA-256 of file **content**.

## ⚠️ CORRECTION NOTE (2026-07-18T22:38:00Z)

A prior revision of this manifest **inspected the wrong file** and reached a **false "zero-overlay"** conclusion for
the Cloud PAT slice. That is now corrected:

- **Wrong (prior):** treated the pre-existing, committed `internal/auth/cloud_pat_test.go` (`2626a886…`) as the
  accepted slice and concluded it added no overlay.
- **Correct:** the TL-accepted producer/reviewer chain concerns a **different, new, untracked** file
  `internal/auth/cloud_pat_log_redaction_test.go` (`1896f90d…`), producer evidence
  `credential-isolation-5.4-cloud-pat-body-log-test.md` (`99b50ea5…`), independent review (GLM52#B)
  `credential-isolation-5.4-cloud-pat-body-log-independent-review.md` (**SHA `8ecb3a36…`**).
- Consequence: the Cloud PAT slice **IS a push overlay** (untracked new file) and **depends on the redact core**.
  The logical unit grows from **4 → 5 overlay items**. The false zero-overlay conclusion is withdrawn.

## Check-IN / Check-OUT
- **Check-IN** 2026-07-18T22:14:00Z — original manifest.
- **Check-OUT** 2026-07-18T22:28:00Z — (superseded).
- **Correction Check-IN** 2026-07-18T22:34:00Z — re-opened actual file + git state + producer/review evidence.
- **Correction Check-OUT** 2026-07-18T22:40:00Z — DONE. Corrected 5-file unit below; re-hash of this artifact recorded at end.

## Logical unit — 5 overlay items (corrected)

| # | Path | Action | SHA-256 (content) | Git state |
|---|---|---|---|---|
| 1 | `pkg/redact/redact.go` | **overlay** current bytes | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` | `M` |
| 2 | `pkg/redact/redact_test.go` | **overlay** current bytes | `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9` | `M` |
| 3 | `pkg/agent/claude.go` | **generated minimal patch only** | target `c7922b7bf92826f5b7889c91ef4d710700be602ef69ad0156044ffded9d5ede9` | `M` (WIP; not overlaid) |
| 4 | `pkg/agent/claude_log_writer_redaction_test.go` | **overlay** (new file) | `81d3e8659125df2fda25c27da16b24d88e1861282255e41f46c33f7cae2ae40a` | `??` |
| 5 | `internal/auth/cloud_pat_log_redaction_test.go` | **overlay** (new file) — **EV-CREDISO-5.4-CLOUDPAT** | `1896f90dcd791a9b455f1c940928e7b608ce64f1403b7bcf2986b80a30a55d49` | `??` |

Generated claude.go patch base = HEAD blob `41d7ac9cf9aa040b5ee7b767ec628608ab601c4d`; two-hunk delta (redact import
+ `logWriter.Write` → `redact.Text(...)`, preserving `w.prefix`/`return len(p)`) is verbatim in the prior atomic
review. Pin File 3 by the **target content hash** `c7922b7b…`.

## Cloud PAT slice (File 5) — accepted, corrected framing

- **Producer:** `credential-isolation-5.4-cloud-pat-body-log-test.md` (`99b50ea5…`).
- **Independent review:** GLM52#B, `credential-isolation-5.4-cloud-pat-body-log-independent-review.md`
  (**SHA `8ecb3a3666ae9582da626bdff7fb17147ce0126511a43643e0ebdc4d200da8c4`**) — technical PASS (bounded); 5.4 OPEN.
- **What it is:** 3 tests (`TestCloudPATVerifyNon200BodyRedactsAccessTokenAndAPIKeySentinels`,
  `…RedactsBearerTokenSentinel`, `…SafeNon200BodyIsPreservedForDiagnostics`) that drive the **real**
  `CloudPATVerifier.Verify/fetch` non-200 path via a synthetic `http.RoundTripper` (no listener/network) and assert
  the production `slog.Warn("cloud_pat: verify returned non-200", …, "body", snippet)` at `cloud_pat.go:359`, routed
  through the production `redact.SanitizeSlogAttr` hook, does not leak `access_token`/`api_key`/bearer sentinels
  while status/diagnostic context survive. Test 3 is a non-vacuous control.
- **Distinct from** the pre-existing committed `internal/auth/cloud_pat_test.go` (`2626a886…`, `TestCloudPATVerifier_*`)
  — that file is **not** part of this unit and must not be confused with File 5.

## Dependencies

- File 3 (Claude patch) **depends on** redact core (Files 1–2) — `access_token` JSON-field regex (proven: 2-file
  claude patch fails on HEAD; 4-file unit passes).
- File 4 (Claude test) depends on Files 1–3.
- **File 5 (Cloud PAT test) depends on: redact core (Files 1–2)** — it wires `redact.SanitizeSlogAttr` and asserts
  `[REDACTED`; **and on `internal/auth/cloud_pat.go`** (the `slog.Warn` call site under test). `cloud_pat.go` is
  **unchanged at HEAD** (`98a4aadf…`) — present in the pristine tree, **not overlaid, not edited**. File 5 does
  **not** depend on the Claude files (3–4).
- The whole 5-item unit shares the redact core (Files 1–2) as its common dependency; all ship together.

## Exclusion set (MUST NOT enter the integration commit)

- **`internal/auth/cloud_pat_test.go`** (`2626a886…`, pre-existing/committed) — distinct pre-existing file; **not**
  part of this unit; do not stage/confuse with File 5.
- **`internal/auth/cloud_pat.go`** (`98a4aadf…`) — **unchanged production call site; excluded from overlay** (no
  edit); required present at HEAD as the code under test, which it already is.
- **Full working-tree `pkg/agent/claude.go`** (`M`, argv/env WIP) — ship only the regenerated two-hunk delta
  (`c7922b7b…`), never the working-tree file.
- **`internal/handler/auth.go`** (`M`, Google OAuth WIP) — unrelated; excluded.
- Untracked pkg/agent WIP: `environment.go`, `environment_test.go`, `models_process_test.go`,
  `models_windows_test.go`, `proc_unsupported.go`; modified `models.go`, `claude_test.go`. Untracked auth WIP:
  `jwt_configuration_test.go`, `recent_auth.go`, `recent_auth_test.go`; modified `jwt.go`. All other unrelated source.

## Isolated-worktree verification recipe (for ROOT to run later — NOT executed here)

Separate pristine worktree off HEAD (root owns invocation; this manifest runs nothing):
1. `git worktree add <path> b6571299`.
2. Overlay Files 1, 2, 4, 5 by their pinned content hashes; regenerate File 3 from HEAD blob `41d7ac9c…`; verify it
   hashes to `c7922b7b…`. Do **not** touch `cloud_pat.go`/`cloud_pat_test.go` (already at HEAD).
3. Gates (offline, pinned `/home/dataops-lab/go-sdk/bin/go`, `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`):
   - `sha256sum` Files 1–5 == pins above.
   - `git diff --cached --name-only` == exactly Files 1–5 (five paths; `cloud_pat.go` and pre-existing
     `cloud_pat_test.go` NOT staged).
   - `gofmt -l` (Files 1,2,4,5 + generated 3) clean; `go build ./pkg/redact/ ./pkg/agent/ ./internal/auth/` exit 0;
     `go vet ./pkg/redact ./pkg/agent ./internal/auth` exit 0.
   - **full** `go test ./pkg/redact` ok; `go test ./pkg/agent` ok; **`go test ./internal/auth` ok** (Cloud PAT
     `TestCloudPATVerifyNon200Body*` green; Redis-gated cache tests SKIP with `REDIS_TEST_URL not set` — pre-existing,
     unrelated).
   - named redact `-count=20` = 60 PASS; six Claude logWriter `-count=20` = 120 PASS; **three Cloud PAT
     `-count=20` = 60 PASS**; `-race` clean (per prior proofs + GLM52#B review).
   - secret gate: staged diff shows only synthetic sentinels inside the test files, never real values.

## Pending gates (integration BLOCKED until all clear)
1. **Owner waiver** for the 5.4 slice/core integration.
2. **Cross-family Claude review** (distinct reviewer, not w7:p2) of the Claude logWriter slice.
3. **Codebase-wide 5.4 gates** — whole-task 5.4 remains OPEN; email/core/claude/cloud-pat are slices.
4. **Kiro TL adjudication** and **root / GitHub auth** for any worktree/commit/push.

## Provenance / non-claims
- Pane **w7:p2**; independent author; **no self-acceptance, no push authorization, no checkbox**. Read-only:
  no worktree/source/test/shared-planning/spec/tasks/git/index/ref edit; no credentials/env/network/DB/services.
  Content hashes verified against HEAD `b6571299` at correction time. File 5 confirmed **untracked/new** and
  redact-core-dependent; `cloud_pat.go` unchanged/excluded; pre-existing `cloud_pat_test.go` distinct/excluded.
  Technical candidate only; Kiro TL adjudicates; root integrates.

## Re-hash of this corrected artifact
- After writing, this file's SHA-256 is recomputed and reported in the session log (the value changes with this
  correction; consumers should re-hash before relying on it).
