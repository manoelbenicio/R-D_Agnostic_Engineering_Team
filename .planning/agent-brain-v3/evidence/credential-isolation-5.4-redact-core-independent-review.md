# EV-CREDISO-5.4-REDACT-CORE — distinct independent review of pkg/redact

Distinct independent review of the `pkg/redact` redaction core
(`SanitizeForLog`/`SanitizeSlogAttr`/`Text`/`IsSensitiveKey`/`InputMap`),
which was the RED blocker on agent-credential-isolation task 5.4
("Confirmar que nenhum segredo aparece em logs (sanitizeForLog)"). The
producer artifact `evidence/credential-isolation-redact-core-fix.md`
(Kiro/Opus-4.8, CRED-REDACT-FIX) reported the prior `TestSanitizeForLog`
query-secret bypass as **already remediated by a concurrent 16:15:46 edit**
and stopped without editing product code. This review independently
reproduces that remediation and inspects the full required-behavior surface.

## Reviewer identity (distinct)

- **Reviewer:** GLM52-auth-QA (Herdr pane `w4:p3`, workspace `w4`).
- **Producer (CRED-REDACT-FIX):** Kiro/Opus-4.8 — read-only, no product edit.
- **Concurrent remediation author:** unattributed concurrent edit at
  2026-07-18 16:15:46 (mtime on `redact.go`/`redact_test.go`), pre-existing
  in the working tree before this review session (21:44 UTC).
- **Adjudicator:** Kiro/Opus-4.8 (TL). This review does not self-accept; TL
  adjudicates and owns any EVIDENCE_INDEX entry / task checkbox.
- **Distinctness:** GLM52-auth-QA is a different agent identity from the
  producer (Kiro/Opus-4.8) and from the unattributed concurrent editor.

## Provenance

- **Host:** `manoelneto-laptop` (WSL2, Linux amd64).
- **Toolchain:** `/home/dataops-lab/go-sdk/bin/go` →
  `go version go1.26.4 linux/amd64`; `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.
- **Repository commit (HEAD):** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.
- **Review execution window:** 2026-07-18T21:44:00Z through 2026-07-18T22:05:00Z UTC.
- **Working directory:** `multica-auth-work/server`.
- **Method:** `read`/`grep`/`go test`/`sha256sum` only. No product/test/spec/
  task/index/git edit; no installs, network, DB, live provider, credentials,
  or env-value inspection.

## Task / EV / AB-REQ mapping

- **OpenSpec task:** `agent-credential-isolation/tasks.md:34` "5.4 Confirmar
  que nenhum segredo aparece em logs (sanitizeForLog)" — unchecked `[ ]`.
- **OpenSpec spec:** `agent-credential-isolation/specs/agent-credential-isolation/spec.md:39-46`,
  requirement "Não vazamento de segredo" + scenario "Log de diagnóstico sem
  segredo" ("o log contém caminho/tipo/mtime, nunca o conteúdo do token").
- **Existing EV:** `EV-CREDISO-5.4-EMAIL` (ACCEPT slice; email log-safety
  only) — `EVIDENCE_INDEX.md:136`. It explicitly recorded `pkg/redact.SanitizeForLog`
  as RED (`TestSanitizeForLog` secret bypass), blocking whole-task 5.4.
- **Proposed EV:** `EV-CREDISO-5.4-REDACT-CORE` — this review, proposed for TL
  indexing. It targets the **redaction-core** slice (clears the prior RED
  blocker on `SanitizeForLog`).
- **AB-REQ mapping (REQUIREMENTS.md):**
  - **AB-REQ-21** ("Secret-safe evidence: redige secrets/cookies/prompts/tool
    payloads/repo content/reasoning"; CLE; spec scenario "Upstream error
    includes an authorization value"; tasks 6.3, 8.3; owner Codex4/3;
    evidence EV-G4-03). `SanitizeForLog`/`SanitizeSlogAttr`/`Text` are the
    primary secret-safe-evidence mechanism; this review is a direct
    implementation-evidence contribution to AB-REQ-21.
  - **Cross-ref AB-REQ-12** (credential+quota lifecycle) and **AB-REQ-38**
    (operational handover) — both rely on redacted logs; not primary.
- **Honest scope note:** clearing the `SanitizeForLog` RED blocker is
  necessary but **not sufficient** for whole-task 5.4 acceptance — the
  codebase-wide "no secret in logs" confirmation (703 slog callsites via the
  global hook) is a separate slice tracked by `EV-CREDISO-5.4-EMAIL` and the
  w7:p2/w7:p1 audits. This review covers the **redaction core** only.

## Reproduction commands and results

### Full package ×20 (all 23 named tests)

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./pkg/redact -count=20
```

Result: exit 0; `ok github.com/multica-ai/multica/server/pkg/redact 0.047s`.
Verbose re-run (`-v`): **460 `--- PASS` / 0 `--- FAIL` / 0 `--- SKIP`**
(23 top-level tests × 20), **580 `=== RUN`** lines (includes 6 subtests
× 20 = 120: `TestRedactGenericCredentials/{API_KEY,DATABASE_URL,DB_PASSWORD}`
and `TestRedactPasswordEnvVar/{PASSWORD,SECRET,TOKEN}`). **Non-zero count
confirmed.** The `TestRedactHomeDirectory` test did **not** skip in this
environment (home dir + username available), so all 23 ran.

### Focused 3 producer-named tests ×20

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test ./pkg/redact -count=20 -run '^(TestSanitizeForLog|TestSanitizeForLogTypedNilError|TestSanitizeForLogIsBoundedAndCycleSafe)$'
```

Result: exit 0; `ok ... 0.014s`. Verbose: **60 `--- PASS` / 0 `--- FAIL`**
(3 tests × 20). This is the exact set the producer artifact cited; reproduced.

### Race (full package + focused)

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race -count=1 ./pkg/redact
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go test -race -count=1 ./pkg/redact -run '^(TestSanitizeForLog|TestSanitizeForLogTypedNilError|TestSanitizeForLogIsBoundedAndCycleSafe)$'
```

Results: both exit 0; full `ok 1.044s`, focused `ok 1.024s` (race clean).

### Vet and gofmt

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off /home/dataops-lab/go-sdk/bin/go vet ./pkg/redact
/home/dataops-lab/go-sdk/bin/gofmt -l pkg/redact/redact.go pkg/redact/redact_test.go
```

Results: vet exit 0, no diagnostics; gofmt `-l` exit 0, no files listed (both
clean).

## Required-behavior inspection (source-anchored)

### Key-based redaction (the 5.4 fix — `map[string]string`)

The producer's claim that the query-secret bypass is fixed is **verified**.
`redact.go:221-226` handles `case map[string]string:` by recursing with the
map key as the `key` argument (`sanitizeForLog(item, k, depth+1, path)`), so
`IsSensitiveKey(k)` (`:177-178`) redacts a `query.secret` value by key name.
`redact_test.go:229-232,264-269` exercises exactly this:
`query: map[string]string{"secret":"synthetic-query-sentinel","page":"2"}`,
asserting `query["secret"]` does **not** contain `synthetic-query-sentinel`
(`:264-266`) **and** `query["page"]=="2"` is preserved (`:267-269`). This is
a **genuine redaction fix, not test-weakening** — the non-secret `page`
value is explicitly asserted intact.

### `IsSensitiveKey` coverage (`redact.go:147-164`)

- Normalizes (`ToLower`, trims `" _-"`, replaces `-`/`.`→`_`) so
  `X-Api-Key`, `x_api_key`, ` client_secret` all match.
- Exact-match list: `authorization`, `proxy_authorization`, `cookie`,
  `set_cookie`, `auth`, `credential`, `credentials`, `password`, `passwd`,
  `secret`, `token`, `api_key`, `apikey`, `api_secret`, `access_token`,
  `auth_token`, `private_key`, `database_url`, `db_password`, `db_url`,
  `redis_url`.
- Suffix match: `_password`, `_passwd`, `_secret`, `_token`, `_api_key`,
  `_authorization` → catches `nested_token`, `client_secret`, etc.
- **Telemetry preservation:** `token_count` does NOT match (no suffix hit,
  not in exact list) — asserted at `redact_test.go:335-338`
  (`slog.Int("token_count", 42)` preserved as KindInt64/42).

### Value-based redaction (`Text`, `redact.go:90-102` + patterns `:21-57`)

12 regex patterns: AWS key/secret, PEM private key, GitHub/GitLab/OpenAI/Slack
tokens, JWT, Bearer, connection strings, credential-bearing JSON fields
(`:53`), generic `KEY=value` env patterns (`:56`). Plus home-dir path
masking (`:96-99`). All exercised: `TestRedactAWSAccessKey`,
`TestRedactAWSSecretKey`, `TestRedactPrivateKey`, `TestRedactGitHubToken`,
`TestRedactOpenAIKey`, `TestRedactSlackToken`, `TestRedactBearerToken`,
`TestRedactGitLabToken`, `TestRedactJWT`, `TestRedactConnectionString`,
`TestRedactCredentialFieldsInJSONBody`, `TestRedactGenericCredentials`,
`TestRedactPasswordEnvVar`, `TestRedactMultipleSecrets`, `TestRedactHomeDirectory`.
No-false-positive coverage: `TestNoFalsePositivesOnNormalText`
(5 benign strings unchanged).

### Structured containers (`SanitizeForLog`, `redact.go:169-249`)

- `string` → `Text` (`:201-202`).
- `[]string` → new slice, recurse (`:203-208`).
- `[]any` → new slice, recurse (`:209-214`).
- `map[string]any` → new map, recurse with key (`:215-220`).
- `map[string]string` → new map, recurse with key (the 5.4 fix, `:221-226`).
- `map[string][]string` (HTTP headers) → key-based redaction shortcut
  (`:230-233`) else recurse items (`:234-238`). `TestSanitizeForLog`
  exercises `Authorization`/`X-Api-Key` headers redacted, `User-Agent`
  preserved (`redact_test.go:224-227,253-261`).
- `error` → `redactedError{err: val}` wrapper whose `Error()` calls `Text`
  (`:241-245,264-269`); `TestSanitizeForLog` embeds
  `DB_PASSWORD: synthetic-error-sentinel` in a `mockError` and asserts the
  sentinel is absent (`:240,283-286`).
- **Never mutates input:** every container case allocates a new map/slice;
  the caller's value is never written back.

### Bounded / cycle-safe (`redact.go:180-198`)

- **Depth bound:** `maxLogSanitizeDepth = 16` (`:106`); `depth >= maxLogSanitizeDepth`
  returns `logRedactionReplacement` (`:180-181`). `TestSanitizeForLogIsBoundedAndCycleSafe`
  builds depth 17 (range `maxLogSanitizeDepth + 1`) and asserts non-nil
  (`redact_test.go:310-316`).
- **Cycle detection:** `logVisit{kind, ptr}` visited-set; on re-visit of the
  same map/slice pointer returns `logRedactionReplacement` (`:188-198`). The
  test builds a self-referential map (`cycle["self"] = cycle`) and asserts
  `safe` stays `"visible"` while `self` fails closed to `[REDACTED]`
  (`redact_test.go:299-308`).

### Typed-nil error (`redact.go:241-245, 251-258`)

`case error:` checks `val == nil` → returns `nil` (`:242-244`); `isNilValue`
handles Chan/Func/Interface/Map/Pointer/Slice (`:251-258`).
`TestSanitizeForLogTypedNilError` passes a typed-nil `*mockError` as an
`error` interface and asserts the result is `nil` (not a `redactedError`
with a nil inner) (`redact_test.go:319-326`). This prevents a nil-typed-error
from being wrapped and producing a misleading non-nil log value.

### `SanitizeSlogAttr` — the central slog hook (`redact.go:117-142`)

- **Key-first:** `IsSensitiveKey(attr.Key) || hasSensitiveGroup(groups)` →
  whole attr replaced with `[REDACTED]` (`:118-121`). This is the defense
  that redacts an opaque sentinel under a credential key even if `Text`
  wouldn't pattern-match it — `TestSanitizeSlogAttrUsesKeyAndPreservesSafeKinds`
  proves it with `nested_token` + `synthetic-opaque-sentinel`
  (`redact_test.go:330-333`).
- **Kind-preserving:** safe `KindInt64`/`KindString` keep their kind+value
  (`:123-128`; tested `:335-342`).
- **Group-aware:** `hasSensitiveGroup` checks nested group names
  (`:135-142`).
- **End-to-end through a real handler:** `TestSanitizeSlogAttrThroughHandler`
  wires `SanitizeSlogAttr` into `slog.NewJSONHandler` with `ReplaceAttr` and
  asserts the sentinel is absent from the JSON output while `safe_message`,
  `ready`, `"count":42` survive — including inside `slog.Group("request", …)`
  and `slog.group("credentials", …)` (`redact_test.go:345-367`). This is the
  strongest proof that the global-hook wiring (703 slog callsites per the
  w7:p2 audit) redacts at the structured boundary.

## Source SHA-256 manifest (revalidated)

```text
f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c  server/pkg/redact/redact.go
5a37941a1c7f1bd7263368a6479104c81300f7981e9f47bfb6b0cd17a602fec9  server/pkg/redact/redact_test.go
```

Both hashes **match the producer artifact's claimed hashes exactly**
(`credential-isolation-redact-core-fix.md:42-43`). Revalidated with
`sha256sum -c` after this artifact was written (exit 0).

## No-secret / no-DB / no-live verification

- **No real secrets in tests:** the test file uses only `synthetic-*`
  sentinels (`synthetic-query-sentinel`, `synthetic-error-sentinel`,
  `synthetic-array-sentinel`, `synthetic-opaque-sentinel`,
  `synthetic-json-sentinel`) and well-known **public** AWS/GitHub/Slack
  example tokens (`AKIAIOSFODNN7EXAMPLE`, `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ…`,
  `xoxb-123456789012-…`, `glpat-AbCdEfGhIjKlMnOpQrStUvWx`,
  `sk-proj-abc123def456ghi789jkl012mno345`) — these are documentation/example
  fixtures, not live credentials. No `os.Getenv`/`os.ReadFile` of real
  credential paths.
- **No DB:** zero `DATABASE_URL` env-gated tests; the `DATABASE_URL` at
  `redact_test.go:87` is a **test input string** (a regex-pattern exercise),
  not an env gate. No `t.Skip` on `DATABASE_URL`.
- **No live:** no network, provider, daemon, or process interaction. Pure
  string/struct functions + `reflect` + `regexp`.
- **Home-dir test:** `TestRedactHomeDirectory` reads `os.UserHomeDir()` /
  `user.Current()` (already resolved at `init`, `:80-85`) only to mask the
  *current user's* home path; it did **not** skip in this env. No credential
  under that path is read — only the path string is masked.

## Conflict scan (Golden Rule 2: disjunta)

- `FILE_OWNERSHIP.md` has **no bounded grant for `pkg/redact/**`** → no
  active owner collision. The concurrent 16:15:46 edit is a working-tree
  modification by another agent, pre-existing before this review session;
  this review is read-only and claims no `files_locked`.
- Concurrent worktree changes by other agents (mobile auth, model-picker,
  server cmd tests, daemon alerting) do not overlap `pkg/redact/`.

## Golden Rule check-in/out

Per `GOLDEN_RULES_E_CHECKIN.md`:
- **Rule 1 (sign-in/out):** this review touched no product/test file;
  evidence artifact + ledger row only.
- **Rule 2 (disjunta):** no `files_locked` — read-only; no overlap.
- **Rule 4 (nada inventado):** all file/symbol/test references are from
  `read`/`grep`/`go test` over current source; hashes revalidated.
- **Rule 5 (sem segredo):** no real credential/token/home content read or
  recorded; only synthetic sentinels and public example fixtures.
- **Rule 9 (só o TL commita / PARE e escale):** no commit; the
  redaction-core-blocker-clearance decision is escalated to the TL.

## Verdict

**ACCEPT (redaction-core slice; clears the prior RED blocker on
`SanitizeForLog`; whole-task 5.4 remains OPEN pending the codebase-wide
slice).**

The `pkg/redact.SanitizeForLog` query-secret bypass that made
`EV-CREDISO-5.4-EMAIL` flag the redaction core as RED is **genuinely
remediated** and **independently reproduced**:

- All 23 named tests pass ×20 (460/0/0), race clean, vet clean, gofmt clean.
- The 3 producer-named tests (`TestSanitizeForLog`,
  `TestSanitizeForLogTypedNilError`,
  `TestSanitizeForLogIsBoundedAndCycleSafe`) pass ×20 (60/0), race clean.
- The fix (`map[string]string` case at `redact.go:221-226`) is real
  key-based redaction, verified by the `query.secret`/`query.page`
  assertions — not test-weakening.
- Required-behavior coverage is complete: key-based, value-based
  (12 regex patterns), structured containers (map/slice/headers/error),
  bounded (depth 16), cycle-safe (pointer visited-set), typed-nil-safe,
  input-non-mutation, kind-preserving slog hook, end-to-end through a real
  JSON handler, no-false-positives.
- Both source hashes match the producer artifact exactly and revalidate.

**Scope boundary (honest):** this clears the `SanitizeForLog` RED blocker
on task 5.4. It does **not** accept whole-task 5.4 — the codebase-wide
"no secret in logs" confirmation (703 slog callsites, Claude stderr residual,
whole-codebase coverage PARTIAL per the w7:p1/w7:p2 critiques) is a separate
slice tracked under `EV-CREDISO-5.4-EMAIL` and remains OPEN. Task 5.4 stays
`[ ]` until both the redaction-core (this review) **and** the
codebase-wide slice are independently accepted by the TL.

**Not self-accepted.** Kiro/Opus-4.8 adjudicates and, if accepted, adds the
`EV-CREDISO-5.4-REDACT-CORE` entry to `EVIDENCE_INDEX.md` and decides
whether the redaction-core blocker on `EV-CREDISO-5.4-EMAIL`/task 5.4 is
cleared.

## Non-claims

This review does **not** claim:
- whole-task 5.4 acceptance (codebase-wide slice separate, OPEN);
- that the concurrent 16:15:46 edit followed a recorded pre-edit
  Golden-Rule check-in (its provenance is unattributed; this review
  judges the artifact on disk, not the edit's process hygiene — that is
  for the TL);
- live provider behavior, network, DB, or real-credential handling;
- any EVIDENCE_INDEX entry (proposed for TL; not self-added);
- any task checkbox change (TL-owned);
- that the 12 regex patterns are exhaustive against all possible secret
  formats (they are a defense-in-depth layer; the key-based
  `IsSensitiveKey` + `SanitizeSlogAttr` key-first path is the primary
  structural guarantee, and it is verified).
