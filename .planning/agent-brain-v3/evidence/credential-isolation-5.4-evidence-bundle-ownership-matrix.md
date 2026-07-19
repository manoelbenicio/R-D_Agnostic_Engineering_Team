# 5.4 evidence-bundle ownership matrix (advisory, read-only)

- Author: Kiro/Opus-4.8, pane **w7:p2**. **Advisory only; Kiro TL adjudicates.** No acceptance, no checkbox.
- Task 5.4: "Confirmar que nenhum segredo aparece em logs (sanitizeForLog)." Whole task remains **OPEN**.
- Read-only: no shared STATE/LEDGER/EVIDENCE_INDEX, tasks/spec/source/test/git/index/ref edit; no
  credentials/env/network/DB/services.

## Check-IN / Check-OUT

- **Check-IN** 2026-07-18T21:58:00Z — read-only map of all 5.4 candidates; sole deliverable is this artifact.
- **Check-OUT** 2026-07-18T22:10:00Z — DONE. Hashes verified against HEAD `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
  at authoring time; drift flagged in §Drift. Nothing else written.

## A. Source / test candidates (current bytes @ HEAD b6571299)

| Path | Owner / provenance | SHA-256 | Git state | Inclusion | Dependencies | Independent reviewer | Governance gate |
|---|---|---|---|---|---|---|---|
| `pkg/redact/redact.go` | 5.4-core; central pkg (not a Codex1 hotspot); source author unnamed (concurrent edit 16:15:46) | `f409ba8a…a058fbf68a5c` | `M` | **INCLUDE** (core) | none (stdlib) | slice ACCEPT (`redact-core-review`); fix note (`redact-core-fix`) — both by this org, **not GLM** | **GLM core review PENDING**; Kiro TL PENDING |
| `pkg/redact/redact_test.go` | 5.4-core test; same provenance | `5a37941a…cd17a602fec9` | `M` | **INCLUDE** (core) | redact.go | same as above | same |
| `pkg/agent/claude.go` (MINIMAL 5.4 file) | generated = HEAD blob `41d7ac9c…` + 2-hunk delta; delta authored w7:p2 | target `c7922b7b…ded9d5ede9` | n/a (generated) | **INCLUDE as generated patch** | redact core (JSON-field regex) | clean-room proofs (w7:p2 = **self**, not independent) | independent review PENDING; Kiro TL PENDING |
| `pkg/agent/claude.go` (WORKING-TREE) | Codex3 (pkg/agent owner); carries argv/env WIP | `3f9dc4fb…925 92c6d2f54` | `M` | **EXCLUDE** (WIP) | — | — | — |
| `pkg/agent/claude_log_writer_redaction_test.go` | new; produced prior session (untracked) | `81d3e865…f46c33f7cae2ae40a` | `??` | **INCLUDE** | redact core; in-pkg `logWriter` | clean-room proofs (w7:p2 = **self**) | independent review PENDING; Kiro TL PENDING |
| `internal/logger/logger.go` | central slog hook (`ReplaceAttr: redact.SanitizeSlogAttr`) | `f5f705c1…cfcefd38010` | `M` | **INCLUDE (coverage)**; push-coupling TBD | redact core | slice reviewed (`5.4-codebase-log-safety-audit`, w7:p2) | Kiro TL PENDING; distinct reviewer PENDING |
| `internal/service/email.go` | email dev-mode slice | `43f36afd…ed27179d4c3` | `M` | **INCLUDE (slice)** | slog/redact | **EV-CREDISO-5.4-EMAIL ACCEPT** (independent, `email-log-safety-review`) | slice accepted; whole-task 5.4 PENDING |
| `internal/service/email_test.go` | email slice test | `865b43ee…e8c0d9be1807a` | `M` | **INCLUDE (slice)** | email.go | same (EV-CREDISO-5.4-EMAIL) | same |
| `internal/auth/cloud_pat.go` | internal/auth owner; **committed/clean at HEAD** | `98a4aadf…b53c67a09a7e2778` | clean (no diff) | **EXCLUDE from push (no change)**; **INCLUDE in coverage** | slog; deliberately drops token_last4/status (no token in logs) | none dedicated for 5.4 | coverage confirm PENDING (advisory) |
| `internal/auth/cloud_pat_test.go` | internal/auth owner; committed/clean | `2626a886…f902de8b75f62e72` | clean | **EXCLUDE from push (no change)** | cloud_pat.go | none | — |
| `internal/handler/auth.go` (Google OAuth seam) | handler owner; **compositional seam** `auth.go:656` logs `"body"` on error-only path | `d69877a9…4a80c8360b7a259e0` | `M` | **EXCLUDE from 5.4 push** (carries unrelated auth WIP) | **depends on redact core** (`Text()` JSON-field regex + `SanitizeSlogAttr`) | seam noted in `5.4-codebase-log-safety-audit` (w7:p2) | distinct reviewer PENDING; Kiro TL PENDING |

## B. Evidence artifacts (current bytes)

| Artifact | SHA-256 | Role | Independent? |
|---|---|---|---|
| `credential-isolation-email-log-safety-review.md` | `3a3018b4…6fc4b5529` | EV-CREDISO-5.4-EMAIL slice ACCEPT | independent (adjudicator-separated) |
| `credential-isolation-redact-core-review.md` | `521cef31…7fa8c12` | redact-core slice ACCEPT | independent core reviewer (not GLM) |
| `credential-isolation-redact-core-fix.md` | `f73fa02e…994301ea` | core "not-reproducible/already-remediated" note | producer note (not acceptance) |
| `credential-isolation-5.4-codebase-log-safety-audit.md` | `2b060da6…b42b870dedb` | whole-codebase PASS w/ residuals | w7:p2 (this author) |
| `credential-isolation-5.4-claude-stderr-clean-room-isolated-patch.md` | `53242f91…f4d32b0c` | 2-file patch FAILS (dep on core) | w7:p2 |
| `credential-isolation-5.4-redact-core-plus-claude-clean-room-atomic-review.md` | `2f25c316…fab3daac0b` | 4-file unit technical-candidate PASS | w7:p2 |
| `credential-isolation-5.4-redact-core-plus-claude-root-integration-manifest.md` | `8dc7b612…329687b004` | root integration recipe | w7:p2 |

## C. Compositional seams (safety depends on the core)

- **Cloud PAT** (`cloud_pat.go`): verifies `mcn_` tokens against Fleet; **sends token plaintext over the network to
  Fleet** (by design) but **logs no token material** (deliberately drops `token_last4`/status/reason). Committed/clean
  — not a 5.4 push item; its non-leaking logging is an **independent coverage surface** to confirm, not modify.
- **Google OAuth** (`auth.go:656`): `slog.Error("google oauth token exchange returned error", "status", …, "body",
  string(tokenBody))` — **error-only** path (status≠200 ⇒ Google error body carries no token), and `body` is scanned
  by `redact.Text` (JSON-field regex for `access_token`/`id_token`/`refresh_token`) + `SanitizeSlogAttr`. **Its
  log-safety is coupled to the redact core**, so it is a dependency-consumer, not an independent unit. Excluded from
  the 5.4 push (auth.go carries unrelated auth WIP).

## D. Exclusions (MUST NOT enter a 5.4 bundle)

- Working-tree `pkg/agent/claude.go` in full (`3f9dc4fb…`) — argv redaction WIP (`redactedAgentArgValue`,
  `path/filepath`) + env WIP. Ship only the regenerated 2-hunk delta (`c7922b7b…`).
- Untracked pkg/agent WIP: `environment.go`, `environment_test.go`, `models_process_test.go`,
  `models_windows_test.go`, `proc_unsupported.go`; modified `models.go`, `claude_test.go`.
- `auth.go` full working-tree diff (unrelated auth changes); `cloud_pat*.go` (no change).

## E. Drift flags (verify before any action)

- **No drift** since prior 5.4 evidence for: `redact.go` `f409ba8a…`, `redact_test.go` `5a37941a…`, claude test
  `81d3e865…`, `email.go` `43f36afd…`, `email_test.go` `865b43ee…`, `logger.go` `f5f705c1…` — all match.
- **Distinction (not drift):** working-tree `claude.go` = `3f9dc4fb…` (WIP) is **not** the minimal 5.4 file
  `c7922b7b…` (HEAD + 2 hunks). Any consumer must regenerate the delta from HEAD blob `41d7ac9c…`.
- `cloud_pat.go`/`cloud_pat_test.go` are committed/clean (no working-tree change) — map as coverage, not push.

## F. Unresolved governance gates (all PENDING)

1. **GLM core review** of `pkg/redact` 5.4-core.
2. **Independent expanded review** (distinct reviewer, NOT w7:p2) of the 4-file logical unit + claude test.
3. **Kiro TL adjudication** — whole-task 5.4 OPEN; email + core are slices only.
4. **Root / GitHub auth** for any integration commit/push.
5. Advisory coverage confirm of the cloud-PAT and Google-OAuth seams under the accepted core.

## Non-claims
- Advisory map only. No acceptance, no checkbox, no push. Read-only: no shared STATE/LEDGER/INDEX/tasks/spec/
  source/test/git/index/ref edit; no credentials/env/network/DB/services. Hashes verified at authoring time against
  HEAD `b6571299`; re-verify before acting. Kiro TL adjudicates.
