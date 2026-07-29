# LANE E — RUNTIMES ONLINE — PREP READY (read-only; execute later after LANE D + go)

- agent: **Opus48#C** · lane **E** · task **E-RUNTIMES-ONLINE-PREP** · pane `w8:p1`
- as-of (UTC): `2026-07-24T10:53Z` · reporting to Opus48-Kiro (`w5:p1`)
- **READ-ONLY prep. NO runtime started, NO daemon restart, NO source edit, NO secret value read/printed. Execution is serialized AFTER LANE D (daemon deploy) AND an explicit go.**

## ⚠️ Two hard caveats (must resolve before relying on auth status)
1. **Host identity unconfirmed.** This pane is on `ip-172-31-30-9.sa-east-1.compute.internal`. I could NOT confirm it is **orq1**. If runtimes must come online on a different orq1 host, re-run this prep there. All findings below are for THIS host.
2. **`$HOME` is a credential-isolation SLOT** (`~/.agent-cred-homes/slots/slot-109/home`), not the real runtime/account home. Runtime auth is owned by the credential-isolation enrollment (`~/.agent-cred-homes/` — `registry.json`, `codex-logins/`, per-account slot homes), **not** the user home. So per-runtime auth below is an **indirect signal + the authoritative command to run on orq1**, marked UNVERIFIED — I did not (and must not) read secret contents.

## 1. CLI install status (read-only `command -v`) — all PRESENT
| Runtime | CLI | Path |
|---|---|---|
| codex | `codex` | `~/.nvm/versions/node/v22.23.1/bin/codex` ✓ |
| cline | `cline` | `~/.nvm/versions/node/v22.23.1/bin/cline` ✓ |
| antigravity | `agy` | `~/.local/bin/agy` ✓ |
| kiro | `kiro-cli` | `~/.local/bin/kiro-cli` ✓ |
| opencode | `opencode` | `~/.opencode/bin/opencode` ✓ |
| (claude, online) | `claude` | `~/.nvm/versions/node/v22.23.1/bin/claude` ✓ |

## 2. Per-runtime AUTH status (indirect signal + authoritative check; all UNVERIFIED from this pane)
Real-home (`/home/ec2-user`) artifact presence (ls only, no contents) + auth verb (`--help`):
| Runtime | Auth artifact (real home) | Auth verb | Provisional | Authoritative check (run on orq1 as runtime user) |
|---|---|---|---|---|
| codex | `~/.codex/auth.json` **PRESENT**; `~/.agent-cred-homes/codex-logins/` present | `codex login` / `codex doctor` | likely-authed | `codex doctor` (diagnoses auth/runtime health) |
| cline | `~/.cline/data/settings/providers.json` **PRESENT** | `cline auth [provider]` | likely-configured | `cline auth <provider>` (verify configured, non-mutating check) |
| kiro | `~/.local/share/kiro-cli` **PRESENT**, `~/.kiro` **PRESENT** | (interactive; AWS Builder ID) | likely-authed | `kiro-cli` login/whoami status (Builder ID) |
| antigravity | `~/.gemini/antigravity-cli` **ABSENT** (real home) | (interactive; Google OAuth eligibility) | **likely NOT authed** | `agy` eligibility/login → token dir `~/.gemini/antigravity-cli` populated. NB: agy 1.1.4 IPv6 eligibility regression (see A5) — needs `GODEBUG=netdns=cgo` reachability |
| opencode | `~/.local/share/opencode/auth.json` **ABSENT**; `~/.config/opencode` present | `opencode auth login` (`opencode providers`) | **likely NOT authed** | `opencode auth login` → `auth.json` present |

**Summary:** installed = 5/5. Auth (provisional, must confirm on orq1): codex/cline/kiro = likely-authed; **antigravity + opencode = likely NOT authed (auth artifacts absent)** → these two are the probable blockers to flipping online.

## 3. How a runtime is started/registered (how Claude 588ebcba came online)
Runtimes are **daemon-registered, not individually launched**:
- The daemon resolves `daemon_id` (`MULTICA_DAEMON_ID` or persisted) + `runtime name` (`MULTICA_AGENT_RUNTIME_NAME`, default `"Local Agent"`) — `internal/daemon/config.go:525-559`.
- On startup it calls **`POST /api/daemon/register`** (`internal/daemon/client.go:459-461`) → backend returns **`RegisterResponse.Runtimes []Runtime`** (`client.go:452-457`), each assigned a UUID. **`588ebcba` is the backend Runtime row for the Claude-backed runtime this daemon registered.** A runtime is **ONLINE** while its daemon is running + registered + heartbeating; `Deregister` (`client.go:445`) / stop takes it offline.
- Runtime set = built-in per `CLIKind` (`brain/registry.go StaticRuntimeRegistry`) plus optional **custom runtime profiles** (`RuntimeProfile{protocol_family, command_name}` via `/api/daemon/workspaces/{id}/runtime-profiles`, written by `multica runtime profile set-path` — `config.go:111`).
- Agents are bound to runtimes → when a runtime is online, its agents flip online.
- Management/observe: `multica runtime list|usage|activity|update|delete` (`cmd/multica/cmd_runtime.go`).
⇒ There is **no per-CLI "start runtime" command**; a runtime flips online when the (deployed) daemon registers it, which requires its CLI installed+**authed** and its agent/runtime configured. This is exactly why execution is **serialized after LANE D (daemon deploy)**.

## 4. START / REGISTER PLAN per provider (draft; DO NOT execute until LANE D done + go)
For each of `codex, kiro, agy, cline, opencode`, on **orq1** as the runtime user, in order:
1. **Confirm/complete auth** (authoritative, §2): codex `codex doctor`; cline `cline auth <provider>`; kiro `kiro-cli` Builder-ID login; **agy** Google eligibility/login (ensure `~/.gemini/antigravity-cli` populated; apply `GODEBUG=netdns=cgo` if 1.1.4 eligibility fails); **opencode** `opencode auth login`. (agy + opencode are the expected work items.)
2. **Ensure the runtime/agent is configured** so the daemon registers it: a built-in CLIKind agent OR a custom runtime profile (`multica runtime profile set-path --command-name <cli> …`), plus the workspace agent bound to it.
3. **LANE D deploys/restarts the daemon** → daemon `POST /api/daemon/register` re-registers → each authed+configured runtime flips **ONLINE**; its agents flip online. (No manual per-runtime start.)
4. **Verify online:** `multica runtime list` shows `codex/kiro/agy/cline/opencode` ONLINE alongside claude `588ebcba`; confirm agents online in the Runtimes UI.

**Serialization / ordering:** do steps 1–2 (auth + config) only when authorized; step 3 (online flip) happens via LANE D's daemon deploy; do NOT restart the daemon myself. One serialized pass; verify with `multica runtime list` (read-only) after.

## 5. Scope / non-claims
- Read-only prep only: `command -v` + `<cli> --help` (no network, no auth, no state change) + `ls` presence (no file contents) + code inspection. **No runtime started, no `login`/`register`/`daemon` run, no daemon restart, no source/config edited, no secret read/printed.**
- Auth statuses are provisional/indirect and **UNVERIFIED** authoritatively (host≠confirmed-orq1; `$HOME`=cred slot); the authoritative checks in §2/§4.1 must be run on orq1 as the runtime user before claiming authed.
- No `multica runtime list` executed (would query the live backend); drafted as the verification command.

## 6. Status
**PREP READY.** Installed 5/5; auth provisional (codex/cline/kiro likely-authed; agy+opencode likely NOT — the expected work); registration path = daemon `POST /api/daemon/register` (588ebcba is a registered Runtime row); plan drafted (auth → configure → LANE D daemon deploy auto-registers → verify). Awaiting LANE D completion + explicit go before any execution.
