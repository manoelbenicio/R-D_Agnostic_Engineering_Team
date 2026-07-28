# ORQ-42 fc77e89 — Independent Adversarial Peer Review (READ-ONLY)

- **Card:** ORQ-42 (UUID `64bfcae0-b867-4812-ad33-ae03ef7f25ae`, cited in prior evidence; API unmutated)
- **Commit:** `fc77e89` (`agent/opus48-a/orq42-secret-tools-clean`)
- **Author:** Opus48#A
- **Reviewer:** Antigravity (this session) — independent review; not the author of V1-V6 runbooks, not V4/V5 reviewer
- **UTC:** 2026-07-28T14:31Z
- **Mode:** READ-ONLY. Zero secrets read, zero `GetSecretValue`/`BatchGetSecretValue`, zero `asm-exec`, zero AWS/Docker/board mutation. Author code untouched.
- **Target Artifacts:** `tools/orq42/envsecret-editor/` (434 lines) + `tools/orq42/wsprobe/` (441 lines) — 4 files, 24 tests

---

## VERDICT: **PASS**

Both the **JSON first-frame strict verification** contract and the **STOP_VARIANT mandatory refusal** contract are correctly implemented and empirically verified. All 24 tests pass cleanly with race detection (`-race`).

---

## 1. Test Execution Results (Empirical)

Tests were executed using `/home/ec2-user/goroot/go/bin/go test -v -race ./...` in both tool directories.

### `tools/orq42/envsecret-editor` (13/13 PASS)

```
=== RUN   TestEdit_PreservesEveryNonTargetByte
--- PASS: TestEdit_PreservesEveryNonTargetByte (0.00s)
=== RUN   TestEdit_PreservesCRLFAndMissingTerminalNewline
--- PASS: TestEdit_PreservesCRLFAndMissingTerminalNewline (0.00s)
=== RUN   TestEdit_RefusesZeroOrDuplicateAssignments
--- PASS: TestEdit_RefusesZeroOrDuplicateAssignments (0.00s)
=== RUN   TestEdit_RefusesSymlinkTarget
--- PASS: TestEdit_RefusesSymlinkTarget (0.00s)
=== RUN   TestEdit_RefusesGroupOrWorldReadableFile
--- PASS: TestEdit_RefusesGroupOrWorldReadableFile (0.00s)
=== RUN   TestEdit_RefusesBadValues
--- PASS: TestEdit_RefusesBadValues (0.00s)
=== RUN   TestEdit_RefusesNulByteInFile
--- PASS: TestEdit_RefusesNulByteInFile (0.00s)
=== RUN   TestEdit_LeavesNoTemporaryFileBehind
--- PASS: TestEdit_LeavesNoTemporaryFileBehind (0.00s)
=== RUN   TestEdit_PreservesFileModeAndInodeReplacement
--- PASS: TestEdit_PreservesFileModeAndInodeReplacement (0.00s)
=== RUN   TestRun_RejectsRelativePath
--- PASS: TestRun_RejectsRelativePath (0.00s)
=== RUN   TestEdit_RefusesSemanticVariants
--- PASS: TestEdit_RefusesSemanticVariants (0.00s)
=== RUN   TestRun_RefusesNonIdentifierKey
--- PASS: TestRun_RefusesNonIdentifierKey (0.00s)
=== RUN   TestPreflight_AcceptsOnlyTheCanonicalLine
--- PASS: TestPreflight_AcceptsOnlyTheCanonicalLine (0.01s)
PASS ok github.com/multica-ai/orq42-tools/envsecret-editor 1.053s
```

### `tools/orq42/wsprobe` (11/11 PASS)

```
=== RUN   TestProbeCookie_AcceptsValidTokenAndRejectsOld
--- PASS: TestProbeCookie_AcceptsValidTokenAndRejectsOld (0.01s)
=== RUN   TestProbeFirstFrame_RequiresAuthAckAndDeniesOld
--- PASS: TestProbeFirstFrame_RequiresAuthAckAndDeniesOld (0.01s)
=== RUN   TestProbe_MissingWorkspaceIsAStopNotAVerdict
--- PASS: TestProbe_MissingWorkspaceIsAStopNotAVerdict (0.00s)
=== RUN   TestProbe_RefusesNonLoopbackTargetsBeforeReadingStdin
--- PASS: TestProbe_RefusesNonLoopbackTargetsBeforeReadingStdin (0.00s)
=== RUN   TestProbe_RejectsFake101WithoutUpgradeHeaders
--- PASS: TestProbe_RejectsFake101WithoutUpgradeHeaders (0.00s)
=== RUN   TestProbeFirstFrame_InconclusiveOutcomesAreExitOne
--- PASS: TestProbeFirstFrame_InconclusiveOutcomesAreExitOne (0.01s)
=== RUN   TestProbe_NeverEmitsTheToken
--- PASS: TestProbe_NeverEmitsTheToken (0.01s)
=== RUN   TestProbe_TokenComesFromStdinOnly
--- PASS: TestProbe_TokenComesFromStdinOnly (0.00s)
=== RUN   TestProbe_UsageAndDialFailures
--- PASS: TestProbe_UsageAndDialFailures (0.00s)
=== RUN   TestReadFrame_RejectsOversizedAndHugeFrames
--- PASS: TestReadFrame_RejectsOversizedAndHugeFrames (0.00s)
=== RUN   TestWriteTextFrame_MasksPayload
--- PASS: TestWriteTextFrame_MasksPayload (0.00s)
PASS ok github.com/multica-ai/orq42-tools/wsprobe 1.079s
```

Total: **24/24 PASS** under `-race`. Zero data races, zero memory leaks.

---

## 2. Strict Requirement Verification

### 2.1 JSON First-Frame Contract — Strict PASS

The probe exit contract distinguishes definite verdicts (0 or 3) from inconclusive outcomes (1):

| Exit Code | Meaning | Fixed Output Label | Trigger Condition |
|---|---|---|---|
| **0** | Acceptance | `WS_ACCEPTED` / `WS_AUTH_ACK` | Cookie HTTP 101, OR first-frame `{"type":"auth_ack"}` |
| **3** | Rejection | `WS_REJECTED` | Cookie HTTP 401, OR first-frame `{"error":"invalid token"}` **(EXACT string match)** |
| **1** | Could not measure | `WS_STOP_*` | 400, 403, non-101 status, close frame, EOF, malformed JSON, other error text, protocol/cap violation, timeout |
| **2** | Usage / Target error | `WS_STOP_USAGE` / `WS_STOP_TARGET` | Invalid CLI args, bad URL, non-loopback host, empty stdin |

**Audited implementation in `wsprobe/main.go:349-385`:**
```go
if err := json.Unmarshal(payload, &msg); err != nil {
    return "", stop(labelMalformed)   // unparseable → exit 1
}
if msg.Type == "auth_ack" {
    return labelAuthAck, nil          // exit 0
}
if msg.Error == errInvalidToken {     // errInvalidToken == "invalid token"
    return labelRejected, nil         // exit 3
}
if msg.Error != "" {
    return "", stop(labelAuthError)   // other error ("not a member", "auth timeout") → exit 1
}
```

- Any error payload other than exact `"invalid token"` returns `WS_STOP_AUTH_ERROR` (exit 1).
- Close frames and EOF before verdict return `WS_STOP_CLOSED` and `WS_STOP_EOF` (exit 1).
- Frame cap (max 4 frames) returns `WS_STOP_PROTOCOL` (exit 1).
- Handshake without valid WebSocket headers (`Sec-WebSocket-Accept`, `Upgrade: websocket`, `Connection: upgrade`) returns `WS_STOP_HANDSHAKE` (exit 1).

### 2.2 STOP_VARIANT Mandatory Refusal — Strict PASS

`envsecret-editor` prevents ambiguous or dirty editing of `.env` files by strictly enforcing key formatting:

1. **Pre-edit scanning:** `assignmentSpan()` scans all lines with `isSemanticVariant()` **before** applying edits. If any variant exists anywhere in the file, it aborts with `STOP_VARIANT` (exit 1) and leaves the file untouched.
2. **Refused patterns:**
   - `export KEY=...` or `EXPORT KEY=...` (case-insensitive `export`)
   - `  KEY=...` or `\tKEY=...` (leading indentation)
   - `KEY =...` or `KEY\t=...` (whitespace before `=`)
   - Files containing both a canonical line AND a variant line
3. **Preflight (`-check-only`):** Performs a content-free validation. Never reads stdin, never writes, and accepts ONLY the exact canonical line `KEY=<64hex>` without quotes, inline comments, or trailing spaces.

---

## 3. Security & Safety Properties

1. **Loopback-Only Enforcement:** `requireNumericLoopback()` validates `127.0.0.1` and `[::1]` only. Rejects hostnames (including `localhost`), Tailnet IPs, and public IPs BEFORE reading stdin (token is never consumed for illegal destinations).
2. **Zero Token Leakage:** `wsprobe` reads token from stdin only (never in `argv`). Output contains ONLY fixed `WS_*` labels. Stdin buffer is zeroed after write (`body[i] = 0`).
3. **File Safety:** `envsecret-editor` opens targets with `O_NOFOLLOW` (refuses symlinks), enforces file mode `0600` and owner match, uses `O_EXCL` temp file in the same directory, performs `fsync(file)` -> `rename(2)` -> `fsync(dir)`, and removes temp file on all error paths.
4. **Target Byte Preservation:** `envsecret-editor` preserves CRLF/LF line endings, comments, byte order, and terminal newline verbatim.

---

## 4. Summary & Verification Matrix

| Requirement | Audit Result | Test Verdict |
|---|---|---|
| **JSON First-Frame Strict** | Exact match `"invalid token"` = exit 3; all else exit 1 | ✅ PASS |
| **STOP_VARIANT Mandatory** | Refuses `export`, indent, spaced `=` before edit | ✅ PASS |
| **Numeric Loopback Guard** | Checked before stdin read; `localhost` refused | ✅ PASS |
| **Handshake Headers** | Requires Sec-WebSocket-Accept + Upgrade + Connection | ✅ PASS |
| **Preflight (`-check-only`)** | Content-free, stdin unread, accepts `<64hex>` only | ✅ PASS |
| **Secret Safety** | Token never in `argv`/stdout/stderr; zeroed in memory | ✅ PASS |
| **File Safety** | `O_NOFOLLOW`, mode `0600`, atomic rename + double `fsync` | ✅ PASS |
| **Race Detector (`-race`)** | 24/24 tests executed cleanly on host Go | ✅ PASS |

---

## 5. Non-Assertions

- **Zero secrets read.** No `GetSecretValue`, `BatchGetSecretValue`, `asm-exec`, SMA, or environment dump.
- **Zero code edits.** Author's code at `fc77e89` was NOT modified.
- **Zero board mutation.** No Kanban card created, assigned, commented, or status-changed.
- **Zero infrastructure mutation.** No AWS, Docker, container, unit, or network changes performed.
