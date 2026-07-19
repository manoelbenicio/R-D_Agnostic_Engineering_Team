# Independent QA — agent-credential-isolation task 5.4 email log-safety (Gemini/Antigravity P5-4)

- Reviewer: Kiro/Opus-4.8 (TL co-lead), independent QA — 2026-07-18
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; no Docker/network/DB/credentials/email delivery/env-value inspection.
- Target (review-only, not edited): `multica-auth-work/server/internal/service/email.go`, `email_test.go`; producer artifact `.deploy-control/evidence/Antigravity-P5-4-Evidence.md`.
- Task 5.4: "Confirmar que nenhum segredo aparece em logs (sanitizeForLog)."

## Verdict

**ACCEPT (email.go dev-mode log-safety slice) — but OpenSpec task 5.4 REMAINS OPEN.**
The email verification-code and invitation-URL dev-mode log leaks are fixed and independently
proven. However, task 5.4's named mechanism `sanitizeForLog` (`pkg/redact.SanitizeForLog`) is
currently **failing** (active secret-in-logs bypass, out of this review's edit scope), so the
task as worded is not fully satisfied. This review does not check 5.4 and does not accept the
unrelated `pkg/redact` work.

## Manifest (SHA-256, current disk)

| File | SHA-256 |
|---|---|
| `server/internal/service/email.go` | `43f36afd8abd17ec6037e22a67205cd6934de8ced8cefb0f1a766ed27179d4c3` |
| `server/internal/service/email_test.go` | `865b43ee3032754ae39326430b5046c99bd012c5867877c5d51e8c0d9be1807a` |
| producer evidence `Antigravity-P5-4-Evidence.md` | (diff-only; self-reports tests could not run) |

## What changed (verified against disk)

- `SendVerificationCode` dev branch: `fmt.Printf("[DEV] Verification code for %s: %s", to, code)` →
  `slog.Info("[DEV] Verification email generated", "to", to, "code", "[REDACTED CREDENTIAL]")` (email.go:340).
- `SendInvitationEmail` dev branch: `fmt.Printf(... inviteURL)` →
  `slog.Info("[DEV] Invitation email generated", "to", to, "inviter", ..., "workspace", ..., "invite_url", "[REDACTED CREDENTIAL]")` (email.go:367).
- The secret (code / invite URL) is replaced with a literal placeholder **before** logging — redaction by construction, not string-scrubbing.

## Executable evidence (reproduced by reviewer; producer could not run)

The producer artifact recorded no test results ("could not run … preventing dependencies download").
Reviewer reproduced on the warmed host cache:

- Focused `-count=20 -v` on `./internal/service` for
  `TestSendVerificationCode_DevModeRedactsCode`, `TestSendInvitationEmail_DevModeRedactsURL`,
  `TestSanitizeSubjectField`, `TestBuildInvitationParams_SubjectStripsControls`,
  `TestBuildInvitationParams_EscapesHTMLInBody`: **exit 0**, **5 distinct parent tests RUN**
  (non-zero match confirmed), `ok`.
- `-race` on the same set: **ok**, no data races.
- `go vet ./internal/service`: **exit 0**.

## Static scan for raw-secret log bypasses (email.go)

- No `fmt.Print*`/`log.*`/`println` logs a verification `code`, `inviteURL`/invite token, password, or message `body`.
- Remaining `fmt.Printf`/`Println` (email.go:194,210,224,226,228) log only construction-time config:
  hostname-fallback warning, unrecognized `SMTP_TLS` mode, SMTP relay `host:port (tls) from=<addr>`,
  Resend `from=<addr>`, and a static DEV-mode hint. `from` is a configured sender address, not a per-user secret.

## Acceptance criteria assessment

- "Secret values never reach stdout/logs": **met for email.go** dev-mode code/URL paths (SMTP/Resend paths log no body/code).
- "Safe dev observability remains": **met** — `to`/`inviter`/`workspace` still logged so a dev sees an email was generated.

## Residuals / non-claims (task 5.4 NOT fully satisfied)

1. **Active bypass (blocks 5.4):** `pkg/redact.SanitizeForLog` fails `TestSanitizeForLog`
   ("query secret not redacted: mysecretvalue", redact_test.go:262). Task 5.4 explicitly names
   `sanitizeForLog`; while it is red, codebase-wide "no secret in logs" is not confirmable.
   This `pkg/redact` work is **out of scope** for this review and is **not accepted or edited** here.
2. **Stale comment (minor):** email.go:228 still says "codes printed to stdout" — inaccurate after
   redaction; no secret leaks, doc-only nit.
3. **Scope:** this review covers `internal/service/email.go` only. Broader handler/response logging
   of verification codes was not audited here and remains for the codebase-wide 5.4 confirmation.
4. No live credentials/network/DB/email delivery/env-value inspection; no product code edited; no
   OpenSpec checkbox changed by this review.

## Disposition

Record the email slice as independently ACCEPTED evidence (`EV-CREDISO-5.4-EMAIL`). **Do not check
task 5.4** until (a) the `pkg/redact.SanitizeForLog` bypass is repaired and accepted, and (b) a
codebase-wide "no secret in logs" confirmation is produced. Queue: repair CRED-REDACT-FIX → then
5.4 codebase confirmation.
