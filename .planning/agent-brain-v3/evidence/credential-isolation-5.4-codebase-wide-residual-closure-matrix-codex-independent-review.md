# Credential-isolation 5.4 residual closure matrix — Codex independent review

## Golden Rule CHECK-IN — 2026-07-18T22:24:44Z

- Reviewer: **Codex56#B**, cross-family and independent of matrix author Kiro/Opus-4.8 `w8:p2`, prior critic Kiro/Sonnet, and adjudicator Kiro TL.
- Reviewed artifact: `credential-isolation-5.4-codebase-wide-residual-closure-matrix.md`; requested SHA prefix `97fbbc24...`; actual SHA-256 **`97fbbc24fcf7783753486f926e9279bdd164c33edfbb64f3f113b9cf529402bf` — PASS**.
- Prior critique: `.deploy-control/evidence/credential-isolation-5.4-codebase-critique.md`, SHA-256 `84702b60ee2ec9746cbbcb7904bbe23d52f796d08c9ffb2dae75e356669bf1de`.
- Scope: current local source and durable evidence, static/read-only. Sole write is this artifact.
- Exclusions honored: no source/test/spec/task/shared-planning/git/index/ref edit; no authentication/token/environment-value access; no DB, network, provider, or service access. No tests were run.
- This review does not accept, rescope, or close task 5.4.

## Technical verdict

| Field | Verdict | Finding |
|---|---|---|
| Artifact/hash integrity | **PASS** | Requested artifact and cited primary audit/critique hashes reproduce. |
| 1088 / 877 / 703 count definitions | **PASS with material scope correction** | All three reproduce, but they are different grep populations—not interchangeable callsite totals or evidence of exhaustive sink coverage. |
| Thirteen raw-argv logging adapters | **PASS** | Exactly 14 production `"agent command"`/`"args"` callsites exist: Claude is projected; the other 13 log raw argv. `opencode_mcp.go` is not a 14th logging sink. |
| Residual classes A–H | **PARTIAL** | The cited callsites exist, but B’s test state is stale, C/“covered” guarantees are misclassified, and significant dynamic sinks are omitted. |
| Central-hook wiring | **PARTIAL** | Both explicit production `slog.New` sites use the sanitizer, but `cmd/multica` has a production package-level `slog.Warn` without initializing the central logger. |
| Structural versus pattern guarantees | **REJECT** | Calling `redact.Text` explicitly guarantees routing through a regex sanitizer, not content-independent removal. Claude stderr and ReportMessages remain pattern-dependent for unknown secret shapes. |
| Absolute closure | **PASS that it is NOT met** | The matrix correctly says literal task closure is not established, although its reason set is incomplete. |
| Bounded/risk closure statement | **REJECT as currently supportable** | “All sinks route through a redacting logger” and “remaining exposure is exactly R-5.4-B” are not proven and are false for the current CLI slog entrypoint. |
| Overall matrix | **PARTIAL / not closure-grade** | Useful inventory and count reconciliation; not exhaustive evidence and not sufficient for whole-task acceptance. |

**Task-level consequence:** credential-isolation 5.4 remains **OPEN `[ ]`**. Existing accepted bounded slices remain bounded; this review creates no acceptance, rescope, or implementation grant.

## Hash snapshot

| Current file/evidence | SHA-256 |
|---|---|
| original codebase audit | `2b060da6c8d817256b30b9f8ab372888105833a8a4478b70ce127b42b870dedb` |
| prior critique | `84702b60ee2ec9746cbbcb7904bbe23d52f796d08c9ffb2dae75e356669bf1de` |
| remaining-gaps review | `5a927fbdf8543a1b1000a2f7820f45971944ee4dcfbfe84ba255c30ccab3fc4c` |
| redact core | `multica-auth-work/server/pkg/redact/redact.go` `f409ba8a9f3e63618d59c5a8692296f8f7c019c9e558576b8786a058fbf68a5c` |
| central logger | `internal/logger/logger.go` `f5f705c1d1433db10d84496ff6dcaf42b62dcad5a415239b9cb38cfcefd38010` |
| canonical cross-family redact-core review | `4e4827a505efcf652c96f01544fbef84e9e717902260ec704fbe58dd5600344a` |
| Cloud PAT review / current test | `8ecb3a3666ae9582da626bdff7fb17147ce0126511a43643e0ebdc4d200da8c4` / `1896f90dcd791a9b455f1c940928e7b608ce64f1403b7bcf2986b80a30a55d49` |
| task / spec | `3bdbc1e1916629d87c14db544aee88050daf0e0aaa2c07c3c933798678e7a0e3` / `02b0a5c131c0fd2cee2732dbf4116155d29e87c55d37cc7391994cfdf20da63b` |

Relevant current source hashes are: `handler/auth.go` `d69877a9dcabec59726717628c103b2b1ab0c9a4c7673d14a80c8360b7a259e0`; `auth/cloud_pat.go` `98a4aadf4dd9a236a388bcdfaa9434d83a779916d25d8afdb53c67a09a7e2778`; `daemon/daemon.go` `a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07`; `daemon/auto_update.go` `1470a887fcb21987a7c7467993227c42b0829504f4d58140418bc48d9cae1a67`; `daemon/execenv/git.go` `230befcd27a6628ae8562894c161eb3938ba3d25e93669755cb2e536c9e4af21`; `daemon/gc.go` `cb702a39ce9e184064882ebef82e07930fea25a45a6aa52914b48e79afe28afd`; `handler/agent_template.go` `4dd0b8b1107ae79975009c52a3eb7d33cab3449526a1e8a8c60fb4d5678f6bc8`; `handler/daemon.go` `0e1cbb54c1d733afba473bab1fe42c62bda58aa9af439871267c1c062b3abec6`; and `service/email.go` `43f36afd8abd17ec6037e22a67205cd6934de8ced8cefb0f1a766ed27179d4c3`.

## Count reproduction and definitions

Commands were run from `multica-auth-work/server` against production `.go` files, excluding `_test.go`:

```text
grep -roE 'slog\.(Debug|Info|Warn|Error|Log|DebugContext|InfoContext|WarnContext|ErrorContext|LogAttrs)\(' --include='*.go' . | grep -v '_test\.go' | wc -l
703

grep -rlE 'slog\.(Debug|Info|Warn|Error|Log|DebugContext|InfoContext|WarnContext|ErrorContext|LogAttrs)\(' --include='*.go' . | grep -v '_test\.go' | wc -l
82

grep -rn 'slog\.' --include='*.go' . | grep -v '_test\.go' | wc -l
877

grep -ro 'slog\.' --include='*.go' . | grep -v '_test\.go' | wc -l
879

grep -rnE 'slog\.(Info|Warn|Error|Debug)|logger\.(Info|Warn|Error|Debug)|\.logger\.(Info|Warn|Error|Debug)|\.Logger\.(Info|Warn|Error|Debug)' --include='*.go' . | grep -v '_test\.go' | wc -l
1088

same expression with grep -roE
1088

grep -rnE '\blog\.(Print|Printf|Println|Fatal|Panic)' --include='*.go' . | grep -v '_test\.go' | wc -l
0

grep -rnE 'fmt\.Print(f|ln)?\(' --include='*.go' cmd | grep -v '_test\.go' | wc -l
59

grep -rnE 'fmt\.Print(f|ln)?\(' --include='*.go' . | grep -v '_test\.go' | grep -v '^\./cmd/' | wc -l
5
```

The 703 figure is an exact occurrence count of selected **package-level `slog` methods** and reproduces the prior audit. The 877 figure is a **line count of all `slog.` namespace references**, including types, constants, constructors, and non-sink references; it is not a callsite count. The 1088 figure is a line/occurrence count for four levels on selected receiver spellings (`slog`, `logger`, field `.logger`, field `.Logger`). It omits valid logger variables such as `taskLog` and `log`, while including only the receiver names its regex recognizes. Therefore:

- `1088 - 703` is not a meaningful uncovered/covered delta;
- 877 is not evidence of 877 sinks;
- none of the three proves that every dynamic value or message expression was inspected;
- the matrix’s dynamic-key grep is heuristic, not a complete data-flow analysis.

The `fmt.Print*` counts do reproduce exactly under the narrow expression. All five non-command hits are in `internal/service/email.go:194,210,224,226,228`. That confirms the count, not the stronger assertion that every possible output/log API has been enumerated.

## Thirteen raw-argv adapters

Current source has exactly 14 production `agent command` calls carrying an `args` attr. Claude calls `safeAgentArgvForLog`; the following **13** pass their argv slice directly:

| Adapter | Callsite | SHA-256 |
|---|---:|---|
| antigravity | `antigravity.go:73` | `96ee0c982cab104cd5690eba71b59536f4bef2306c184bf52471198dd36887a1` |
| cline | `cline.go:68` | `9497ebfccaeb143cef0e08b2ae4f59f5192a40d118d2f68ff208f9ae1322ede0` |
| codebuddy | `codebuddy.go:107` | `ecb85d968c1b60283e09174d3bc37a7dfa80126193105c2e97ce8382109bbcf9` |
| copilot | `copilot.go:213` | `80111abb1aa00045d7d31a777c8d233a57b41f7cdfaafe3b07fd49f21391d07b` |
| cursor | `cursor.go:41` | `f38115ae48ccc5bcfac0a028ad375dd99fb7394d0f7029791d0757be922e192e` |
| gemini | `gemini.go:38` | `260ffcf6d8066ad3e9f15c086381f5e062910043dba76d6c2de7421d79567555` |
| hermes | `hermes.go:64` | `3752b611d5f9fd1961079fa25a78187057ba9291cf7daa817838202a4e1ba3d9` |
| kimi | `kimi.go:61` | `53271c50affe13088d98a9f9b3f3db711b908a8b6a0e4fcae7e2031eec10cd2e` |
| kiro | `kiro.go:68` | `0b4d3bd7f274623fa4d45db34639a247e969074cf04ac5dce5de2b7322657410` |
| openclaw | `openclaw.go:79` | `ebd450c2c3911db39df078bc362a749d0bf0bd68d1250e85299e362cfdc4291a` |
| opencode | `opencode.go:85` | `4db9a414e13743c8cc672b36d30f6ba2f649530f75daeb13e42a9c27db448d4c` |
| pi | `pi.go:210` | `46f1ed17f664f2c316944f42e0a134ca86a460ba8bcd777e81aac6d27d1994da` |
| qoder | `qoder.go:99` | `7bfb0d23039911c2f206aab34b4cb1eb3885929dd471924a8eac7682b8042618` |

Claude’s current full-file hash is `3f9dc4fbdb28eecfcc4886f2a518a07943f8a0493cf7aa3525b745992c6d2f54`; its call at `:66` uses the projection. The raw 13 are materially relevant: several argv builders include prompt/system-prompt, model, resume IDs, configuration paths, and user-controlled custom arguments. The global hook sees `args` as `KindAny`/`[]string` and applies `Text()` element-by-element, which is pattern coverage only.

The prior same-family matrix review and current ledger suggested `opencode_mcp.go` as a possible 14th raw logging adapter. Static source disproves that specific callsite claim: `openCodeCommand` normalizes MCP configuration `command`/`args` at `opencode_mcp.go:354-380`, but the file contains **zero Debug/Info/Warn/Error calls**. Codex also has no `agent command` argv log. They may handle argv, but they are not additional class-D logging sinks. The matrix’s 13-callsite count is correct under its stated log-sink definition.

## Residual classes A–H

| Class | Static result | Correction/consequence |
|---|---|---|
| A — Google OAuth body | **CONFIRMED** at `handler/auth.go:656` | `body` is not a sensitive key; coverage is regex-based. No current site-level Google test was found. |
| B — Cloud PAT body | **CONFIRMED** at `auth/cloud_pat.go:359` | Matrix is stale: `cloud_pat_log_redaction_test.go` now exists and its bounded slice is indexed. It proves listed synthetic shapes, not arbitrary secret shapes. |
| C — Claude stderr | **CALLSITE CONFIRMED; GUARANTEE MISGRADED** at `claude.go:972-985` | Explicit `redact.Text(text)` is a structural sanitizer-routing step, but `Text` is a fixed regex set. Unknown shapes can survive; it is not absolute structural erasure. |
| D — argv | **13 CONFIRMED** | Largest explicitly enumerated raw surface. `safeAgentArgvForLog` itself is flag/term heuristic, stronger than raw+regex but still requires a defined allow/deny contract for absolute claims. |
| E — agent error content | **CONFIRMED** at `daemon.go:4542` | Pattern-dependent. Matrix omits normal agent text logged at `daemon.go:4535`. |
| F — update output | **FOUR CONFIRMED** at `auto_update.go:163,167` and `daemon.go:2297,2305` | Pattern-dependent; likelihood grading does not change task wording. |
| G — git output | **TWO CONFIRMED, THREE OMITTED** at `execenv/git.go:108,116`; additionally `daemon/gc.go:621,647,675` | Five current git-output sinks, not two. All are dynamic and pattern-dependent. |
| H — malformed extra-skill ID | **CONFIRMED** at `agent_template.go:497-503` | Description “raw agent-template parse input” is broad; exact value is a malformed user-supplied `extra_skill_id`. It can still carry arbitrary attacker text. |

No listed A–H callsite is fabricated. The false positives are in the claimed guarantees and in treating the inventory as exact/exhaustive, not in the existence of A–H.

## Material omissions beyond A–H

1. **Normal agent content:** `daemon.go:4535` logs `truncateLog(msg.Content, 200)` under key `text`. This is a direct agent-controlled dynamic sink alongside class E. Redaction occurs after truncation at the handler, so a truncated or novel-shaped secret can evade the fixed patterns.
2. **GC/git output:** `daemon/gc.go:621,647,675` logs raw command output and is absent from class G.
3. **Generic errors:** production has many `"error", err` attrs. Key `error` is not sensitive (`redact.go:147-164`), so `SanitizeForLog(error)` wraps `err.Error()` through `Text()` (`:241-268`). An error embedding an unknown credential shape remains a residual. A–H does not enumerate or bound this population.
4. **Unhooked CLI slog entrypoint:** explicit production logger construction is centralized, but `cmd/multica/main.go` does not call `logger.Init`. `cmd/multica/cmd_id_resolver.go:42` uses package-level `slog.Warn`; it therefore uses Go’s default handler, not `SanitizeSlogAttr`. Its current value is an issue ID, so this is not a confirmed credential leak, but it disproves the matrix’s universal central-hook statement and leaves any future/different package-level CLI slog attrs unprotected.
5. **Other attr names/message expressions:** the dynamic scan is keyed to selected names. Values under `text`, `query`, `path`, `url`, `data`, `payload`, `result`, and interpolated messages are outside A–H unless separately noticed. The 1088 enumeration does not resolve their provenance or sensitivity.

These omissions corroborate the prior critique’s non-exhaustiveness finding and go beyond its original Claude/message example.

## Central hook: what is and is not guaranteed

Static construction search finds exactly two production `slog.New(` occurrences, both in `internal/logger/logger.go:38,52`; both handlers set `ReplaceAttr: redact.SanitizeSlogAttr` at `:36,50`. No production `slog.NewTextHandler` or `slog.NewJSONHandler` exists. Server/migrate/backfill entrypoints call `logger.Init`; the daemon obtains `logger.NewLogger`.

That proves a strong fact: **logs emitted through those constructed handlers traverse the sanitizer hook**. It does not prove:

- every production entrypoint installed one of those handlers (`cmd/multica` does not);
- every caller-injected `*slog.Logger` uses it;
- every traversing value is removed—only recognized sensitive keys are unconditionally replaced, while strings/errors/arrays use `Text()` pattern matching;
- message interpolation or values under benign keys are safe for unknown secret formats.

Accordingly, “all production loggers wire the hook” and “every dynamic value below is at least pattern-dependently redacted” are too broad. They are true for the server/daemon logger path, not universally for the repository.

## Structural versus pattern guarantees

The matrix conflates **structural routing** with **structural confidentiality**:

- `redact.Text` at `redact.go:87-102` always invokes a fixed list of regex replacements. Explicitly calling it ensures the value reaches that pattern engine; it does not remove arbitrary credentials.
- `SanitizeSlogAttr` unconditionally removes only recognized sensitive keys/groups (`:117-132`, `:147-164`). Benign keys such as `body`, `args`, `content`, `output`, `raw`, `text`, and `error` fall to pattern processing.
- `SanitizeForLog` recursively reaches supported container types, but string and error leaves still call `Text()` (`:169-248`, `:260-268`). Unsupported custom types may pass through unchanged.
- ReportMessages at `handler/daemon.go:2226-2228` explicitly calls `Text`/`InputMap` before persistence/broadcast. That is a valuable structural placement guarantee, but content/output and non-string map values are not absolutely secret-free.
- Claude stderr has the same distinction: explicit pre-log `Text` is defense in depth, not an absolute guarantee for opaque/new secret shapes.

An absolute content-independent guarantee would require omission/constant replacement, a narrowly typed safe projection/allowlist, or another proof that the input cannot contain credential material. Key-based replacement is absolute only within the recognized-key contract, not for all structured attrs.

## Absolute versus bounded closure and task consequence

The matrix is correct that literal absolute closure is **not met**. Its proposed bounded statement is not yet evidence-backed because it says all sinks use the hook and the residual is exactly R-5.4-B; the unhooked CLI slog call and omitted dynamic classes disprove both universals.

The current task remains exactly: `tasks.md:34` — “Confirmar que nenhum segredo aparece em logs (sanitizeForLog).” This review neither changes nor narrows that language. Consequences:

- Task 5.4 must remain **OPEN**; no whole-task PASS follows from the 703/877/1088 counts or A–H.
- Accepted email, redact-core, and Cloud-PAT evidence remain bounded slice evidence only.
- Claude pre-redaction and ReportMessages are meaningful defense-in-depth but cannot be cited as absolute for unknown secret shapes.
- A future bounded-risk statement may be useful operationally, but it is not equivalent to satisfying the current unqualified task and cannot be treated as acceptance without an authorized rescope—which this review does not perform or recommend.
- Closure evidence must either establish exhaustive, content-independent handling for all log-capable paths or remain explicitly partial. Tests of selected sentinels demonstrate those shapes only.

## Non-claims

- No test, live service, DB, provider, credential, token, environment value, or network behavior was exercised.
- No source/test/spec/task/shared record/git/index/ref was changed.
- No acceptance, checkbox, EV registration, implementation authorization, owner waiver, or rescope is issued.
- Kiro TL remains adjudicator.

## Golden Rule CHECK-OUT — 2026-07-18T22:28:03Z

Review complete: counts **reproduced with definition limits**; 13 raw-argv loggers **confirmed**; A–H **PARTIAL/non-exhaustive**; central hook **PARTIAL**; structural-vs-pattern claim **REJECT**; absolute task closure **not met**; bounded closure statement **not currently supportable as universal**. Task 5.4 remains OPEN.
