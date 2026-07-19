# Independent audit — agent-credential-isolation task 5.4 whole-codebase log-safety

- Reviewer: independent security reviewer (Kiro/Opus-4.8), read-only static + bounded-test audit — 2026-07-18
- Role: independent; does NOT self-accept — **Kiro TL adjudicates** final acceptance and any `tasks.md` checkbox.
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; offline.
- Task 5.4: "Confirmar que nenhum segredo aparece em logs (sanitizeForLog)."
- Spec requirement "Não vazamento de segredo": SHALL NOT log credential content when resolving/mounting
  per-account dirs; only metadata (path/type/mtime) allowed.

## Golden Rule check-in / check-out

- **Check-IN** 2026-07-18T20:43:00Z — claimed: read-only source/static scan of logging/error/event-emission
  surfaces + one planning evidence artifact (this file). No files_locked for edit beyond this artifact.
- Scope owned: static enumeration + bounded offline `pkg/redact` tests (pre-existing, unmodified).
- Excluded (honored): no credential/token/auth-home/env **value** inspected; no product/test/spec/`tasks.md`
  edits; no DB/network/live-provider/service; no git stage/commit/push; no secret fixtures created.
- **Check-OUT** 2026-07-18T20:57:00Z — DONE; verdict below; no product code or OpenSpec checkbox touched.

## Provenance (SHA-256, current disk — VERIFIED matching accepted EV-CREDISO-5.4-CORE artifact)

| File | SHA-256 |
|---|---|
| `server/pkg/redact/redact.go` | `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` |
| `server/pkg/redact/redact_test.go` | `5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9` |

Hashes match those pinned in `credential-isolation-redact-core-fix.md`; the accepted core module is
unchanged on disk.

## Executable evidence (non-zero, reproduced by reviewer)

`go test -count=1 -v ./pkg/redact` → **PASS**, `ok …/pkg/redact 0.017s`, exit 0.
25 parent tests RUN + subtests, including the log-safety-critical ones:
`TestSanitizeForLog`, `TestSanitizeForLogIsBoundedAndCycleSafe`, `TestSanitizeForLogTypedNilError`,
`TestSanitizeSlogAttrUsesKeyAndPreservesSafeKinds`, `TestSanitizeSlogAttrThroughHandler`,
`TestRedactCredentialFieldsInJSONBody`. Non-zero match confirmed (not a vacuous/skip pass).

## Logging-sink enumeration (whole codebase, server/)

1. **`slog.*` (structured)** — 703 callsites / 82 files. All route through the process-global default
   logger installed by `logger.Init()` (`internal/logger/logger.go:38` `slog.SetDefault`) with
   `ReplaceAttr: redact.SanitizeSlogAttr` (logger.go:36,50). `logger.Init()` is called by every
   entrypoint: `cmd/server/main.go:123`, `cmd/migrate/main.go:107`,
   `cmd/backfill_codex_usage_cache/main.go:59`, `cmd/backfill_task_usage_hourly/main.go:58`.
2. **Standard `log.*` package** — **zero** Print/Printf/Println/Fatal/Panic callsites in server code
   (verified by anchored regex). No sink bypasses slog via the std log package.
3. **`fmt.Print*` (stdout/stderr)** — confined to (a) `cmd/multica/*` interactive CLI command output
   (operator-owned terminal, not daemon diagnostic logs), (b) `internal/service/email.go` (already
   independently ACCEPTED as EV-CREDISO-5.4-EMAIL), (c) `*_test.go` fixtures. No server daemon path
   uses `fmt.Print*` for secret-bearing diagnostics outside email.go.
4. **WebSocket / event-emitter broadcast** — single agent-output ingestion point
   `internal/handler/daemon.go:2225-2228` (`ReportMessages`): `msg.Content`/`msg.Output` →
   `redact.Text`, `msg.Input` → `redact.InputMap`, applied **before** DB persist and broadcast
   (in-code comment: "Redact sensitive information before persisting or broadcasting.").
5. **Agent comment / error surfaces** — `internal/service/task.go:1298,1328,1459,1471,2385` wrap
   comment bodies / error messages through `redact.Text` before create/broadcast.

## Sensitive-field flow → sink coverage (by file:line)

| Sensitive flow | Sink | Coverage mechanism | Verdict |
|---|---|---|---|
| Per-task credential secret carrier | any `%v/%+v/%s/%#v` fmt path | `runtimeenv.StableSecret` `String()/GoString()/Format()` all return `[REDACTED]` (env.go:54-57); test env_test.go:95,98 | COVERED by construction |
| OpenClaw gateway bearer token | debug/log/JSON dump | `OpenclawGatewayPin.String()` masks Token `***` (openclaw_config.go:83-90) + `MarshalJSON()` masks; test openclaw_runtime_config_test.go:104 | COVERED (hardened for #3260) |
| Inherited/child env values | diagnostics | `runtimeenv.MinimalEnvironment` keeps values private; diagnostics operate on key names only (env.go) | COVERED by design |
| Agent custom_env reveal/update | `agent_env.go` audit | audit records key symmetric-difference, never values (agent_env.go comment + slog only logs error+agent_id) | COVERED |
| Structured attrs w/ credential keys | all `slog.*` | `SanitizeSlogAttr` → `IsSensitiveKey` (key-name, suffix-aware) + `Text()` value scan + recursive `SanitizeForLog` for `KindAny` | COVERED |
| Agent stdout/output/tool-input | DB + WS broadcast | `redact.Text`/`redact.InputMap` at daemon.go:2226-2228 | COVERED |
| Google OAuth token-exchange body | `auth.go:656` slog `"body"` | error-only path (status!=200 → no token in Google error body); `Text()` JSON-field regex also redacts access_token/id_token/refresh_token | COVERED (pattern-dependent) |
| Task tokens (generate/persist/revoke) | daemon.go:1792,1806; task.go:190 | logs error + IDs only, never token value | COVERED |

## Unsanitized / unknown paths (flagged, non-blocking)

- **R-5.4-A (structural, general codebase):** slog `ReplaceAttr` sanitizes **attributes only**, not the
  log **message** string. A hypothetical `slog.X(fmt.Sprintf("…"+secret))` would bypass redaction.
  Targeted scan (`slog.*` × secret keywords × `fmt.Sprintf`) found **no** credential-bearing message
  interpolation in credential-isolation paths; convention is structured attrs + redacting carrier types.
  Cannot exhaustively prove all 703 callsites; recommend a lightweight lint/CI guard. **No confirmed leak.**
- **R-5.4-B (pattern-dependency):** values placed under a non-sensitive key rely on `Text()` regex
  pattern coverage. Novel/vendor-specific token shapes not matching a pattern could slip. Mitigated by
  key-name redaction + redacting carriers + ingestion-path redaction. **No confirmed leak.**
- **R-5.4-C (CLI, out of daemon-log scope):** `cmd/multica/cmd_autopilot.go:579,584` print webhook URLs
  (may embed tokens) to the operator's own terminal; `cmd_auth.go:415` prints a PAT **prompt** only
  (no value). Interactive CLI output, not the spec's "log de diagnóstico" surface.

## AB-REQ / EV mapping

- **AB-REQ (spec "Não vazamento de segredo"):** SHALL NOT log credential content on resolve/mount;
  metadata only → **SATISFIED** — credential-isolation surfaces (env injection, StableSecret,
  OpenclawGatewayPin, agent_env audit) are value-free/redacted by construction.
- **EV-CREDISO-5.4-CORE** (pkg/redact module): re-verified ACCEPT; hashes pinned above; tests green.
- **EV-CREDISO-5.4-EMAIL** (email.go dev-mode slice): prior ACCEPT unchanged; not re-adjudicated here.
- **EV-CREDISO-5.4-CODEBASE** (this audit): whole-codebase log-safety = mechanism comprehensively wired
  (global slog hook + ingestion redaction + redacting carriers), no unsanitized credential path found.

## Conflict scan

- No conflict with EV-CREDISO-5.4-CORE (hashes identical; core unmodified).
- No conflict with EV-CREDISO-5.4-EMAIL (email.go untouched; independent slice).
- No product/spec/task edits by this audit; AGENT_LEDGER prior 5.4 entries consistent (both left 5.4 OPEN
  pending exactly this codebase confirmation).

## Verdict: **PASS** (whole-codebase log-safety), with documented non-blocking residuals

Basis: (1) the named mechanism `sanitizeForLog`/`pkg/redact` is green and its central slog hook is
globally installed across all 4 entrypoints; (2) the sole agent-output broadcast/persist sink redacts
before emission; (3) credential-isolation-specific secret carriers redact by construction; (4) no
standard `log` package leaks and no server-daemon `fmt.Print*` secret path outside the accepted email
slice; (5) no unsanitized credential path was found. Residuals R-5.4-A/B/C are hardening opportunities
with **no confirmed leak**.

## Non-claims / recommendation to TL

- Reviewer does **not** self-accept and did **not** check `tasks.md` 5.4. TL to adjudicate.
- Recommend: accept 5.4 log-safety confirmation; optionally open a **non-blocking** hardening ticket for
  R-5.4-A (CI lint forbidding `fmt.Sprintf` of secret-typed values into slog messages).
- No live credentials/network/DB/provider traffic; no secret values inspected; no product code edited.
