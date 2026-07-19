# G3 Security Correction — Claude/Codex Adapter Logging

- Owner: Codex 3 — runtime/CLI security
- Evidence: EV-G3-SEC-ADAPTERS
- Recorded: 2026-07-18T03:32:36Z
- Reviewer finding: Claude and Codex logged final argv verbatim, while model/config/custom arguments could carry routing, authentication or private paths.
- Status: IMPLEMENTED; independent pB re-review pending

## Correction

Claude and Codex now send command diagnostics through one adapter-local safe argv projection before logging. Process argv remains unchanged.

The projection:

- logs only the executable basename;
- retains argument count, safe subcommands, safe flag names and safe non-sensitive diagnostic values;
- redacts separate and inline values for routing, auth, config, settings, home, model, resume, provider, base-URL/endpoint, token, key, environment/header and prompt/schema flags;
- recognizes compound adapter-local variants such as provider-config, auth-file, key-file, home-dir, routing-url and settings-sources;
- redacts inline credential-, authorization-, bearer-, cookie-, password-, private-key-, provider-, model-, endpoint- and config-like assignments;
- explicitly preserves Claude's standalone `--strict-mcp-config` flag without misclassifying the following safe flag as its value.

No raw final-argv `agent command` log call remains in either adapter.

## Gateway-mode decision

The existing adapter-local `ExecOptions` and backend `Config` contracts have no Claude/Codex gateway-required signal. Adding an adapter allowlist would require inventing a shared API or guessing from environment state, neither of which is appropriate for this scoped correction.

Therefore:

- no adapter-mode heuristic or shared API change was added;
- central gateway-required `CustomArgs` rejection remains the primary High-severity fix;
- safe argv logging is defense in depth for legacy and non-gateway execution paths.

## Regression evidence

Claude and Codex regression tests use synthetic strings only. Each test proves that synthetic model/provider identifiers, resume IDs, config/settings/home paths, base URLs, auth/header/key values, inline environment assignments and instruction content are absent from the log. They also prove that executable basename, safe flag names, safe execution modes, argument count and the redaction marker remain available for diagnosis.

Verification passed from `multica-auth-work/server` with the pinned Go toolchain:

- focused redaction tests repeated ten times;
- full `go test ./pkg/agent`;
- `go test -race ./pkg/agent -run 'Test(Claude|Codex)CommandLoggingRedactsSensitiveArgv' -count=10`;
- `go vet ./pkg/agent`;
- `gofmt` and scoped `git diff --check`;
- static source check confirming both adapter command logs use the safe projection.

## Scope and non-claims

Only `pkg/agent/claude.go`, `codex.go` and their tests were changed. No daemon/central, gateway, deploy, observability, runtime environment or credential source was edited. No credential, auth file or secret was read, copied, printed, rewritten or mutated. PD-01 was preserved without reset, stash, revert or discard. This evidence does not claim that arbitrary custom arguments are safe to execute in gateway-required mode; that is the central High fix awaiting independent pB re-review.
