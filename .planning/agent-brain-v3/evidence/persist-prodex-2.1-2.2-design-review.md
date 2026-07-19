# Design-Readiness Review — persist-prodex 2.1-2.2 (Codex56#A)

- reviewer: Kiro (principal, independent)
- date: 2026-07-18T20:55:00Z
- mode: READ-ONLY product/spec/task/git. No env/credential contents read, no systemd/DB/network/provider invoked, no product/test/spec edits, no stage/commit/push.
- reviewed artifact: `.deploy-control/evidence/persist-prodex-2.1-2.2-design.md`
  - SHA-256 `fa560db1431dcf0da1a335c4235c3d05c6e00481c34b60e6a2dc977ada8ec1df`
- check-in: `.deploy-control/Kiro__PRODEX-2.1-2.2-DESIGN-REVIEW__20260718T205400Z.md`

## VERDICT: ACCEPT (design-readiness only) — implementation-gated, non-blocking recommendations below

The design is secure, implementable, rollback-safe, cross-platform-honest, and disjoint from the dirty central ownership. It makes no acceptance/implementation claim. It is ready to proceed **as a design**, subject to the sequencing gates it already names. This verdict is design-readiness only; **Kiro TL adjudicates and owner gates remain owner-only.** Staged/authored state is not acceptance.

## Provenance verification (independent, live tree)

**All source hashes in the artifact match the current tree exactly** (no drift despite the active multi-agent tree):

| File | Artifact SHA | Live SHA | Match |
|---|---|---|---|
| tasks.md | de661603… | de661603af6b0ec1aece2ffe442f446ece86cc9dd91aa596fa81fc1553b3eaea | ✅ |
| OpenSpec design.md | 2bf341e8… | 2bf341e8323467f1fc8235190c6f42711ffae392d081849a7c7127063ffb7feb | ✅ |
| spec.md | 4a47a6cf… | 4a47a6cfc92b5c63acdd8baf83556063f84d47dbd9a9e8270b38e6988375c251 | ✅ |
| readiness audit | b8d28474… | b8d2847443c1a090c979bb0567a122896a8c14e94419875937b3507112bf5b1f | ✅ |
| REQUIREMENTS.md | f95fbc6a… | f95fbc6a1323f86c8e00707843ecc407e98288f9fb8bdc00ce3903ed259fdfdc | ✅ |
| cmd_daemon.go | 0460a0e3… | 0460a0e3a52b75b14a29a2a7591a592224da7adf12c08cb71d383aa91f05de73 | ✅ |
| config.go | 9a8a33f6… | 9a8a33f6cc6ad2ff95cb9034d23900a8ca9bdac5b1eb815eb8db979a642189cf | ✅ |
| prodex.go | 82035719… | 82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7 | ✅ |
| l2_runtime.go | a54fb79d… | a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de | ✅ |
| health.go | 4b1650e0… | 4b1650e059a52f7951fd83af5ca707aa5b42b111b2c4dd107d4f11d501dbc3a8 | ✅ |

## Claim-by-claim verification

| Design claim | Verdict | Evidence |
|---|---|---|
| Tasks 2.1/2.2 wording | **CONFIRMED** | tasks.md L9 "secure persistent launcher/service that imports the mode-0600 Prodex EnvironmentFile"; L10 "Extend the local persistent Prodex environment with L2 endpoint, adapter, Postgres, and redacted secret settings". Task 2.3 correctly excluded. |
| All 11 AB-REQ ids exist | **CONFIRMED** | AB-REQ-04/16/17/18/20/21/22/34/36/37/38 each present in REQUIREMENTS.md. |
| AB-REQ-34 = 20128 is the OmniRoute/Agent-Brain target, NOT the Prodex L2 port | **CONFIRMED + strong** | REQUIREMENTS.md L72: `host/WSL usa 127.0.0.1:20128 … Host daemon launches Codex`. Design correctly forbids copying 20128 into the Prodex L2 template. |
| Current L2 default port is 43117 | **CONFIRMED** | `l2_runtime.go:266` → `[]string{"127.0.0.1:43117"}`. |
| Service must invoke `daemon start --foreground`, never the background path | **CONFIRMED + correct** | `cmd_daemon.go:81,178-179,303` implement foreground + self-backgrounding; running the CLI background path under systemd would create a competing/detached process + PID file. |
| Launcher must resolve `_FILE` refs in memory to the Go-expected names (no Go edit) | **CONFIRMED** | Go reads `MULTICA_L2_BEARER_TOKEN` (`prodex.go:73`) and `PRODEX_PG_URL` (`prodex.go:152-153`); no native `*_FILE` resolution exists for these. The only `_FILE` precedent is `AGENT_BRAIN_GATEWAY_SECRET_FILE` (a different subsystem). So the launcher approach is the correct disjoint path. |
| `MULTICA_PRODEX_CONFIG_SOURCE=systemd_environment_file` label accepted by Go | **CONFIRMED** | `prodex.go:47` `envOrDefault("MULTICA_PRODEX_CONFIG_SOURCE","process_env")` accepts any opaque label. |
| Spec anchors (fs modes / redacted visibility) | **CONFIRMED** | spec.md:62-85 = "Filesystem and permission enforcement" (0700/0600, drvfs/9p/CIFS rejection) + "Redacted operational visibility". |

## Dimension assessment

- **Security / threat model — STRONG.** Never `source`/`eval`; strict `KEY=VALUE` allowlist rejecting unknown/duplicate keys, control chars, command substitution, relative paths. `env -i` + minimal allowlist strips provider vars; trusted values applied last (AB-REQ-17/18). Secrets only via mode-0600, owner-matched, non-symlink `_FILE` references; never in argv/stdout/stderr/env-file/evidence. Nonsecret metadata validated **before** any secret read (ordering minimizes exposure window). systemd unit env holds only references pre-`ExecStartPre`.
- **Quoting / permissions / atomic-write — SOUND.** `umask 077`, same-dir temp → validate → chmod 0600 → flush → atomic rename; rejects 0640/0644/symlink/wrong-owner. (Minor: make parent-dir `fsync`-after-rename explicit for crash durability.)
- **Secret-at-rest boundary — SOUND.** Reference-only model; one service-secret boundary (AB-REQ-16); no copying provider credentials into `PRODEX_HOME`.
- **Failure semantics — SOUND / fail-closed.** Every error exits nonzero; required mode has no raw/native/provider fallback; only bounded systemd retries. Matches spec durable-activation + fail-closed.
- **Rollback — SAFE.** One last-known-good unit+env revision; validate candidate before atomic replace; on failure restore prior pair + reload + restart prior required-L2 config; if none, stopped + redacted failure. Never auto-launches raw route, copies creds, adds a second router, or weakens a mode (AB-REQ-36).
- **Cross-platform honesty — HONEST.** Linux user-systemd only; WSL only with systemd + verified POSIX FS else fail closed (never suggests `source`/`export`); containers explicitly out of scope with correct rationale (orchestrator secret mounts + container DNS, not host unit/loopback); macOS/Windows return a stable unsupported result before reading secrets.
- **Offline test plan — COMPREHENSIVE + genuinely offline.** 10 deterministic assertion groups; temp synthetic files + fake executables; explicitly no `systemctl`/daemon/DB/provider/network; `bash -n`, `shellcheck`, harness ×20, optional `systemd-analyze verify`.
- **Disjointness from dirty ownership — CLEAN.** Excludes the 9 dirty central files; proposes only NEW files under `deploy/systemd/` and `scripts/ops/`; mandates freezing 1.1-1.3 first and serial hand-off if Go-native `_FILE` is ever required. Consistent with Golden Rule 2 (single-owner hotspots).

## Non-blocking recommendations (address at implementation, not design defects)

1. **Pin the launcher→Go resolution contract explicitly**: enumerate that `MULTICA_L2_BEARER_TOKEN_FILE`→`MULTICA_L2_BEARER_TOKEN` and `PRODEX_PG_URL_FILE`→`PRODEX_PG_URL` (verified Go-expected names) against the frozen 1.1-1.3 contract, so a rename in that lane can't silently break resolution.
2. **Directory durability**: state `fsync` of the containing directory after the atomic rename for the env file and unit.
3. **Service-user/topology assumption**: make the "service user" and `%h` home explicit (which account runs the daemon; ensures owner checks + `EnvironmentFile=%h/...` resolve on the target host). The current documented host path is `~/runtime/prodex.env` on ext4 — align the template default with it.
4. **`DATABASE_URL_FILE`**: only introduce if the reconciler truly needs a value distinct from `PRODEX_PG_URL`; otherwise drop to avoid a redundant secret reference.

## Implementation gates (sequencing, not design defects)

- **G-a:** 1.1-1.3 Prodex baseline must be FROZEN/accepted first (design correctly states this; that lane is still open).
- **G-b:** independent security review of the reference-only secret model before activation behavior is implemented/tested.
- **G-c:** offline harnesses + static checks green before any OpenSpec checkbox change.
- **G-d:** TL adjudication + owner gate; only the TL commits (Golden Rule 9).

## Explicit non-claims / provenance

- I changed no product/spec/task/planning/git state; I created only this review artifact and my check-in.
- I did not read any credential or environment-file contents, and invoked no systemd/DB/network/provider/daemon.
- Verdict is **design-readiness only**; it is not an implementation acceptance, EV award, or checkbox change. Tasks 2.1/2.2 remain MISSING/unimplemented.
- Steer carried: staged authorship (e.g. the 11-file frontend "G1" set in the push-scope matrix) is **not** acceptance; treat as PENDING/EXCLUDED until task ownership + independent evidence + TL adjudication exist. The same principle applies here — this ACCEPT is for the *design*, not for shipping.
- AB-REQ/EV identifiers (`EV-PP-2.1-LAUNCHER`, `EV-PP-2.2-ENV`, `EV-PP-2.1-2.2-REVIEW`) are unregistered proposals; this review does not register or award them.
