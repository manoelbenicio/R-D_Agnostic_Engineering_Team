# Herdr Credential Isolation — ORQ2 Recovery and Operations Runbook

Status date: 2026-07-20 (America/Sao_Paulo)

## 1. Executive status

The credential-isolation incident on `orq2` has been contained and the isolation guard has been installed.

Current verified state:

- `orq1` (`100.118.244.61`) owns the single-node DEV application stack.
- `orq2` (`100.110.178.47`) is reserved for Herdr agent execution.
- Three active Codex processes and one active Kiro process were stopped with `SIGTERM` before remediation.
- No provider logout, token revocation, account reset, or credential-content read was performed.
- Herdr 0.7.4 and Codex were already installed on `orq2`.
- Before remediation, `orq2` had only the shared global `~/.codex/auth.json` and no `~/.agent-cred-homes` registry.
- The isolation script is installed and sourced from every new interactive Bash shell.
- Global credential migration is disabled by default and explicitly disabled on `orq2`.
- The deployed sandbox harness passes.
- A new shell receives a private physical slot with mode `700`, a Codex shell wrapper, isolated XDG paths, and no inherited `auth.json`.
- All Codex, Kiro, OpenCode, Antigravity, and GROK processes were confirmed stopped after validation.

No old Herdr pane may be trusted to have the new environment. Existing panes must be replaced or their shell must be restarted before any provider CLI is launched.

## 2. Incident and root cause

### AS-IS before containment

`orq2` had:

- a shared directory `/home/ec2-user/.codex`;
- a regular `/home/ec2-user/.codex/auth.json` used as the default Codex credential;
- no `/home/ec2-user/.agent-cred-homes` directory;
- no isolation bootstrap in `/home/ec2-user/.bashrc`;
- three concurrent Codex processes and one Kiro process;
- no repository checkout containing the approved isolation implementation.

With this state, a second login for the same provider writes to the same global credential store. A token refresh or login can therefore replace the identity observed by another agent.

### Root cause

Herdr provides terminal multiplexing, pane identity, and agent lifecycle integration. It does not automatically create independent provider credential homes.

The isolation failure occurred because the provider processes started before the project isolation bootstrap was installed. They inherited the default user home and therefore converged on shared vendor credential paths.

The failure is not fixed by creating more panes. New panes can inherit environment variables from a parent pane, and pane IDs can change when panes are compacted. Isolation must be based on a stable Herdr terminal identity plus a local fail-closed lifecycle guard.

### Why repeated login is unsafe

Running another login against the same shared global home does not create a second account. It overwrites or refreshes the same credential store. During an incident:

1. stop provider processes;
2. do not run logout, revoke, reset, or additional login;
3. install and validate isolation;
4. restart panes;
5. enroll exactly one account in each isolated pane.

## 3. Target architecture

### Node responsibilities

| Node | Role | Credential rule |
|---|---|---|
| `orq1` | PostgreSQL, backend, frontend, later approved infrastructure | Does not own interactive agent-provider logins |
| `orq2` | Herdr panes and coding agents | Every new shell receives an isolated credential environment |

### Stable terminal to slot mapping

The bootstrap obtains `terminal_id` from `herdr pane get "$HERDR_PANE_ID"`. The stable terminal identity is stored in:

```text
/home/ec2-user/.agent-cred-homes/registry.json
```

The registry is protected by `flock`, written atomically, and maps each stable terminal to one physical slot:

```text
/home/ec2-user/.agent-cred-homes/slots/slot-NN/
```

If Herdr identity lookup is unavailable, the bootstrap allocates a unique fail-safe fallback identity from the TTY, pane ID, or process identity. It never falls back to a shared provider home.

### Per-slot vendor paths

| Runtime | Isolated variable/path |
|---|---|
| Codex base slot | `CODEX_HOME=$SLOT_ROOT/codex` |
| Codex explicit login | Fresh physical `$ROOT/codex-logins/login-*` owned by the current shell lifecycle |
| Kiro | `XDG_DATA_HOME=$SLOT_ROOT/xdg-data`, native store under `kiro-cli/` |
| OpenCode | `XDG_DATA_HOME` and `XDG_CONFIG_HOME` under the slot |
| GLM through OpenCode | Slot-local OpenCode or explicit `glm/` XDG directories |
| Cline | `CLINE_DATA_DIR=$SLOT_ROOT/cline` and slot-local sandbox data |
| Antigravity/agy | `HOME=$SLOT_ROOT/home`, including `.gemini/antigravity-cli` |
| GROK | Explicit physical profiles A–D with exclusive leases |

### Codex lifecycle guard

The sourced script defines a Bash `codex` function. On the first Codex invocation in a shell, or on every explicit `codex login`, it:

1. creates a fresh physical login directory;
2. creates and locks a `.lease` file;
3. exports that directory as `CODEX_HOME`;
4. rejects symlinked or inherited unsafe bindings;
5. invokes the real CLI only after the binding is verified.

The cleanup policy is report-only. The script does not automatically delete credential directories.

### No implicit legacy migration

`AGENT_CRED_ISOLATION_MIGRATE_LEGACY` now defaults to `0`.

This is a security boundary. Copying a shared global credential into every new slot would create several independent copies of the same identity and would hide an allocation error. A one-time migration is allowed only after explicit owner authorization by setting the variable to `1` for that controlled migration process.

On `orq2`, the variable is permanently configured as:

```bash
export AGENT_CRED_ISOLATION_MIGRATE_LEGACY=0
```

## 4. Changes executed on ORQ2

### Containment

The following process names were inspected by count and stopped with `SIGTERM`:

- `codex`: 3 before, 0 after;
- `kiro-cli`: 1 before, 0 after;
- `opencode`: 0;
- `agy`: 0;
- `grok`: 0.

No command-line arguments or credential contents were collected.

### Installed files

```text
/home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh
/home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/tests/agent-cred-isolation-harness.sh
```

Both files are executable and owned by `ec2-user`.

### Shell integration

The previous Bash configuration was backed up as:

```text
/home/ec2-user/.bashrc.pre-agent-cred-isolation.20260720T085002Z
```

The following managed block was appended to `/home/ec2-user/.bashrc`:

```bash
# >>> agent credential isolation >>>
export AGENT_CRED_ISOLATION_HOST_HOME="/home/ec2-user"
export AGENT_CRED_ISOLATION_HOST_XDG_DATA_HOME="/home/ec2-user/.local/share"
export AGENT_CRED_ISOLATION_HOST_XDG_CONFIG_HOME="/home/ec2-user/.config"
export AGENT_CRED_ISOLATION_ROOT="/home/ec2-user/.agent-cred-homes"
export AGENT_CRED_ISOLATION_ENABLE_VENDOR_SLOTS=1
export AGENT_CRED_ISOLATION_MIGRATE_LEGACY=0
source "/home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh"
# <<< agent credential isolation <<<
```

The shared legacy `.codex` directory was restricted to mode `700`; its existing `auth.json` remains mode `600`. Its value and account identity were not read.

## 5. Validation evidence

### Static validation

```bash
bash -n ~/.local/lib/agent-credential-isolation/scripts/ops/agent-cred-isolation.sh
bash -n ~/.local/lib/agent-credential-isolation/scripts/ops/tests/agent-cred-isolation-harness.sh
```

### Sandbox harness

```bash
bash ~/.local/lib/agent-credential-isolation/scripts/ops/tests/agent-cred-isolation-harness.sh
```

Verified result:

```text
PASS: Codex no-delete leases, GROK A-D physical profiles/device-login isolation, 6-vendor migration, recompaction, fallback, and flock allocators
```

The test uses synthetic credential markers in a temporary directory. It does not read or modify real provider credentials.

### Deployed shell validation

Verified facts from a new Bash shell:

```text
CODEX_WRAPPER=function
SLOT=slot-01
SLOT_ROOT=/home/ec2-user/.agent-cred-homes/slots/slot-01
CODEX_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-01/codex
XDG_DATA_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-01/xdg-data
XDG_CONFIG_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-01/xdg-config
CLINE_DATA_DIR=/home/ec2-user/.agent-cred-homes/slots/slot-01/cline
ISOLATED_AUTH_PRESENT=no
SLOT_MODE=700
```

Verified permissions:

```text
700 /home/ec2-user/.agent-cred-homes
600 /home/ec2-user/.agent-cred-homes/registry.json
700 /home/ec2-user/.agent-cred-homes/slots/slot-01
```

## 6. Mandatory restart and login procedure

### Phase A — replace stale panes

Every Herdr pane created before installation must be treated as stale. Stop any provider CLI in it, then close and recreate the pane or restart its shell with:

```bash
exec bash -l
```

Do not preserve or manually export the old `CODEX_HOME`, `HOME`, `XDG_DATA_HOME`, `XDG_CONFIG_HOME`, or `CLINE_DATA_DIR` values.

### Phase B — pre-login checks in every pane

Run:

```bash
test "${HERDR_ENV:-}" = "1"
type -t codex
agent_cred_isolation_status
stat -c '%a %U:%G %n' "$AGENT_CRED_ISOLATION_SLOT_ROOT"
```

Required results:

- `HERDR_ENV=1`;
- `type -t codex` returns `function`;
- each pane reports a different `slot-NN` unless it is the same stable terminal;
- slot mode is `700`;
- no path points to `/home/ec2-user/.codex`;
- no path is a symlink.

Compare two panes without showing secrets:

```bash
printf 'terminal=%s slot=%s codex=%s data=%s config=%s\n' \
  "$AGENT_CRED_ISOLATION_TERMINAL_ID" \
  "$AGENT_CRED_ISOLATION_SLOT_NAME" \
  "$CODEX_HOME" "$XDG_DATA_HOME" "$XDG_CONFIG_HOME"
```

Do not proceed if two different stable terminals report the same slot or credential path.

### Phase C — one account per pane

Only after Phase B passes:

```bash
codex login
```

The owner completes the provider login interactively. Never automate the OAuth/device confirmation and never paste tokens into shell history, chat, logs, or documentation.

For Kiro/OpenCode/Antigravity/Cline, launch the normal CLI only after confirming the pane-local XDG/HOME variables. One pane owns one account identity.

After login, confirm presence only:

```bash
test -f "$CODEX_HOME/auth.json" && echo 'codex credential present'
stat -c '%a %U:%G %n' "$CODEX_HOME" "$CODEX_HOME/auth.json"
```

Do not use `cat`, `jq`, checksum, grep, strings, database dump, or screenshots on credential files.

## 7. Prohibited operations

The following operations bypass or weaken isolation and are prohibited:

- launching `/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex` directly;
- using `command codex` manually;
- running provider CLIs with `sudo`;
- exporting a global `CODEX_HOME`, `HOME`, or XDG path over the managed values;
- enabling `AGENT_CRED_ISOLATION_MIGRATE_LEGACY=1` during ordinary startup;
- symlinking an isolated credential back to `~/.codex/auth.json`;
- copying one pane's credential tree into another pane;
- running logout, revoke, reset, rotation, or token deletion through an agent;
- reading or printing credential contents for diagnosis;
- reusing an old pane without restarting its shell;
- deleting `registry.json`, slots, login directories, or lease files while agents run.

## 8. Incident diagnosis

### Safe status commands

```bash
agent_cred_isolation_status
type -a codex
stat -c '%F %a %U:%G %n' \
  "$AGENT_CRED_ISOLATION_ROOT" \
  "$AGENT_CRED_ISOLATION_REGISTRY" \
  "$AGENT_CRED_ISOLATION_SLOT_ROOT" \
  "$CODEX_HOME"
```

### Detect a global-home regression

```bash
case "$CODEX_HOME" in
  "$AGENT_CRED_ISOLATION_ROOT"/*) echo PASS ;;
  *) echo BLOCKED_GLOBAL_OR_UNKNOWN_HOME; return 1 ;;
esac
```

### Detect duplicate slot assignment

Use the metadata printed by `agent_cred_isolation_status` in each pane. Two different `terminal` values must never share the same `slot` or `root`.

Do not inspect `registry.json` while writing a public report because terminal identifiers are operational metadata. If an ownership mismatch is reported, stop provider processes and preserve the registry for offline analysis.

### If Herdr is unavailable

The script allocates a private fallback slot. This is fail-safe but not a durable Herdr binding. Provider work may continue only if the owner explicitly accepts temporary fallback operation. Otherwise stop and restore Herdr first.

## 9. Recovery and rollback

### Isolation bootstrap failure

If a new shell reports an isolation error:

1. do not launch a provider CLI;
2. run the syntax and sandbox tests;
3. verify `flock`, `python3`, `sha256sum`, and `cp` exist;
4. verify root/registry ownership and modes;
5. compare the installed script checksum with the repository version;
6. repair the bootstrap, then recreate the pane.

### Suspected credential overwrite

If an account appears to have changed:

1. stop all processes for that provider;
2. preserve path/type/owner/mode/mtime metadata only;
3. do not perform another login or logout;
4. identify which pane and slot were expected;
5. have the owner reauthenticate the correct account in a fresh isolated pane;
6. rotate/revoke only after an explicit owner/security decision.

The system cannot reconstruct a credential that a provider CLI already overwrote unless a separately authorized credential backup exists. The global `~/.codex/auth.json` on `orq2` was deliberately not read, copied, or claimed as a specific account.

### Shell configuration rollback

Rollback is available from:

```text
/home/ec2-user/.bashrc.pre-agent-cred-isolation.20260720T085002Z
```

Rollback procedure:

1. stop all provider CLIs;
2. preserve the current `.bashrc` and isolation registry;
3. restore the prior `.bashrc` only for diagnostic rollback;
4. do not resume multi-agent provider work without an alternative isolation control.

Removing the isolation block restores the unsafe global-home behavior and is not an operational resolution.

## 10. Persistence and backup policy

Credential homes are deliberately excluded from Git and ordinary application backups.

Backup rules:

- never place `.agent-cred-homes`, `.codex`, Kiro/OpenCode auth stores, or provider tokens in the repository;
- transfer credential backups only after explicit owner authorization through an encrypted channel;
- preserve modes `700` for directories and `600` for files;
- do not generate checksums of credential contents for public evidence;
- prefer provider reauthentication on a new agent node over uncontrolled copying;
- document account ownership and slot assignment without documenting tokens.

The application/database backup transferred to `orq1` does not contain the `orq2` provider credential homes.

## 11. Herdr operating model

- Herdr pane IDs are not durable and may compact.
- Always resolve current pane IDs from Herdr rather than hardcoding them.
- The isolation registry uses Herdr's stable `terminal_id` when available.
- A new pane or replaced shell must rerun the bootstrap.
- Never assume a restored screen proves the credential binding is current.
- Agent status and terminal multiplexing are coordination signals, not credential authority.
- Disk paths, registry ownership, process environment, and the provider's isolated store are authoritative.

## 12. Acceptance gate before restarting the fleet

All items must pass:

- [ ] Every old pane has been replaced or restarted.
- [ ] Every pane reports `HERDR_ENV=1`.
- [ ] `type -t codex` returns `function` in every Codex pane.
- [ ] Every different terminal has a different slot/root.
- [ ] All slot directories are physical directories with mode `700`.
- [ ] `AGENT_CRED_ISOLATION_MIGRATE_LEGACY=0`.
- [ ] No pane points at `/home/ec2-user/.codex`.
- [ ] The sandbox harness passes on `orq2`.
- [ ] Exactly one owner-approved account is enrolled per pane.
- [ ] No provider process is started before its pane passes metadata checks.
- [ ] No agent performs credential rotation, revocation, or account reset.
- [ ] The orchestrator records pane, terminal, slot, runtime, and assigned task without secret values.

## 13. Known safe limitation

Codex login directories are bound to the current shell lifecycle and are never automatically reattached after a pane/shell replacement. This prevents a new shell from inheriting another shell's credential. The safe consequence is that the owner may need to reauthenticate after replacing a Codex pane.

Automatic persistent account reattachment is not enabled on `orq2`. It requires a separately reviewed account-to-slot registry and must not be improvised by symlink or directory copy.

## 14. Authoritative files

- `scripts/ops/agent-cred-isolation.sh`
- `scripts/ops/tests/agent-cred-isolation-harness.sh`
- `openspec/changes/agent-credential-isolation/specs/agent-credential-isolation/spec.md`
- `openspec/changes/agent-credential-isolation/design.md`
- `openspec/changes/agent-credential-isolation/tasks.md`
- `openspec/changes/persist-prodex-runtime-integration/specs/prodex-runtime-continuity/spec.md`
- `docs/operations/CREDENTIAL_ISOLATION_REFERENCES.md`
- `docs/herdr/README.md`

This runbook is the operational authority for the `orq2` recovery performed on 2026-07-20. OpenSpec remains the behavioral contract, and the installed script plus passing harness are the executable implementation evidence.
