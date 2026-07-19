# Persist Prodex runtime integration 2.1-2.2 — implementation design

Date: 2026-07-18
Owner: Codex Design
Disposition: design only; **no implementation or acceptance claim**

## Scope and confirmed baseline

This artifact designs the smallest secure follow-up for OpenSpec change
`persist-prodex-runtime-integration`, tasks 2.1 and 2.2. The confirmed readiness
audit classifies both tasks as MISSING: no persistent service imports a
mode-0600 Prodex environment file, and no local persistent environment template
or installer establishes the L2 adapter and PostgreSQL configuration. Task 2.3
is outside this lane.

Authoritative anchors:

- `openspec/changes/persist-prodex-runtime-integration/tasks.md:9-10` — exact
  task 2.1 and 2.2 wording.
- `openspec/changes/persist-prodex-runtime-integration/specs/prodex-runtime-continuity/spec.md:3-23`
  — durable activation, fail-closed startup, and separate executable validation.
- That spec at `:62-85` — filesystem modes, L2 readiness, and redacted operator
  visibility.
- `.planning/agent-brain-v3/evidence/persist-prodex-runtime-2.1-2.3-readiness-audit.md:13-15`
  — MISSING/MISSING/implemented classifications.
- That audit at `:18-21,28-31` — secret-export risk, restart-continuity gap, and
  recommendation to use a disjoint service/template owner.

Baseline SHA-256 values observed before this artifact was written:

| File | SHA-256 |
|---|---|
| OpenSpec tasks | `de661603af6b0ec1aece2ffe442f446ece86cc9dd91aa596fa81fc1553b3eaea` |
| OpenSpec design | `2bf341e8323467f1fc8235190c6f42711ffae392d081849a7c7127063ffb7feb` |
| OpenSpec specification | `4a47a6cfc92b5c63acdd8baf83556063f84d47dbd9a9e8270b38e6988375c251` |
| Readiness audit | `b8d2847443c1a090c979bb0567a122896a8c14e94419875937b3507112bf5b1f` |
| Agent Brain requirements | `f95fbc6a1323f86c8e00707843ecc407e98288f9fb8bdc00ce3903ed259fdfdc` |

## Conflict inspection and ownership boundary

The working tree already contains concurrent edits in the central integration
surfaces, including:

- `multica-auth-work/server/cmd/multica/cmd_daemon.go`
- `multica-auth-work/server/internal/daemon/config.go`
- `multica-auth-work/server/internal/daemon/daemon.go`
- `multica-auth-work/server/internal/daemon/health.go`
- `multica-auth-work/server/internal/daemon/l2_runtime.go`
- `multica-auth-work/server/internal/daemon/prodex.go`
- `multica-auth-work/.env.example`
- `multica-auth-work/scripts/ops/agent-cred-isolation.sh`
- `docs/operations/RUNBOOK_ISOLAMENTO_CREDENCIAL_PANES.md`

Those files are excluded from this slice. Central daemon/config/health work is
owned by another active lane, and the existing 1.1-1.3 Prodex baseline is still
open despite a technical review. Implementation must therefore start only after
that baseline is frozen and must use new, disjoint files. If implementation
discovers that Go-native `_FILE` support is required, it must stop and hand that
change to the central owner serially; it must not opportunistically edit the
dirty Go files.

Relevant current source anchors, inspected read-only:

- `server/cmd/multica/cmd_daemon.go:177-183,185-365,367-429` — foreground and
  self-backgrounding paths. A service must invoke `daemon start --foreground`.
- `server/internal/daemon/config.go:209-232,458-475` — Prodex/L2 configuration
  fields and required-mode loading.
- `server/internal/daemon/prodex.go:16-100,146-159` — executable/version/commit,
  L2 token/tenant checks, PostgreSQL forwarding, and unsafe-child-env denial.
- `server/internal/daemon/l2_runtime.go:69-96,191-211,260-267` — L2 lifecycle,
  readiness, adapter launch, and current loopback adapter default.
- `server/internal/daemon/health.go` — current task-2.3 visibility surface; no
  modification is planned here.

Observed source SHA-256 values:

| File | SHA-256 |
|---|---|
| `cmd_daemon.go` | `0460a0e3a52b75b14a29a2a7591a592224da7adf12c08cb71d383aa91f05de73` |
| `config.go` | `9a8a33f6cc6ad2ff95cb9034d23900a8ca9bdac5b1eb815eb8db979a642189cf` |
| `prodex.go` | `82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7` |
| `l2_runtime.go` | `a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de` |
| `health.go` | `4b1650e059a52f7951fd83af5ca707aa5b42b111b2c4dd107d4f11d501dbc3a8` |

## Proposed file and function map

All paths below are proposals, not files created by this design task.

### 1. `multica-auth-work/deploy/systemd/multica-prodex-daemon.service.in`

Add a Linux user-service template with rendered absolute paths:

- `EnvironmentFile=%h/.../prodex.env` without the optional-file `-` prefix, so
  absence is fatal.
- `ExecStartPre=<absolute launcher> validate <absolute environment file>`.
- `ExecStart=<absolute launcher> run <absolute multica binary> daemon start --foreground`.
- `Type=simple`, `UMask=0077`, `Restart=on-failure`, bounded restart/start limits,
  `NoNewPrivileges=true`, `PrivateTmp=true`, and only hardening directives that
  preserve required workspace/home writes.
- Normal systemd signal handling for stop. Never invoke the CLI background
  path, create a competing PID file, or use `multica daemon restart`.

### 2. `multica-auth-work/scripts/ops/multica-prodex-launcher.sh`

Proposed functions:

- `main`
- `require_linux_systemd_user`
- `validate_environment_contract`
- `require_regular_nonsymlink_mode_0600`
- `require_owned_by_service_user`
- `require_approved_posix_filesystem`
- `require_absolute_executable`
- `require_loopback_l2_topology`
- `read_secret_reference`
- `build_clean_environment`
- `run_foreground_daemon`

The launcher must never `source` or `eval` the environment file. Under systemd,
it consumes only allowlisted inherited keys. For direct validation, it parses a
strict `KEY=VALUE` grammar and rejects unknown or duplicate keys, control
characters, command substitutions, and relative executable/secret paths.

It validates all nonsecret metadata before reading a secret reference. It then
reads approved mode-0600, owner-matching, non-symlink secret files into process
memory, exposes the current names expected by the Go process only in that
process environment, and executes via `env -i` with a minimal system allowlist.
Provider credentials and untrusted inherited settings are absent. Secrets must
never appear in argv, stdout/stderr, the environment file, or generated evidence.

### 3. `multica-auth-work/scripts/ops/configure-multica-prodex-service.sh`

Proposed functions:

- `validate_nonsecret_input`
- `render_environment_file`
- `atomic_install_mode_0600`
- `render_user_unit`
- `install_user_service`
- `backup_current_revision`
- `rollback_previous_revision`
- `print_redacted_summary`

The configuration interface accepts nonsecret values and absolute secret-file
references only, never raw secret values. With `umask 077`, it renders into a
same-directory temporary file, validates, sets mode 0600, flushes where the
platform permits, and atomically renames. Unit and reference-only environment
file revisions are retained in a mode-0700 state directory with mode-0600 files.
The operator summary reports enabled/required state, executable identity,
adapter/base URL, tenant/policy identifiers as approved opaque values, and
secret presence/source metadata only; it never prints secret values or database
URLs. Installation and restart are explicit separate operations.

### 4. `multica-auth-work/deploy/systemd/prodex.env.example`

Add a nonsecret template containing variable names and placeholders only:

- required/enabled flags;
- absolute pinned Prodex executable, version, and commit;
- opaque `MULTICA_PRODEX_CONFIG_SOURCE=systemd_environment_file` label;
- L2 enabled flag, absolute sidecar path, loopback bind argument, matching
  loopback base URL, timeout, policy ID, and tenant ID;
- POSIX Prodex home and `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`;
- `MULTICA_L2_BEARER_TOKEN_FILE=/absolute/reference` and
  `PRODEX_PG_URL_FILE=/absolute/reference`, not their secret values.

If the reconciler also needs a separate database value, use a distinct
`DATABASE_URL_FILE` reference. The launcher resolves references in memory to the
existing environment variable names. Because systemd imports only nonsecrets
and references, the unit environment does not contain the bearer token or
PostgreSQL URL before `ExecStartPre` validation.

For the current host/WSL topology, the L2 adapter and base URL must agree on
loopback; the current source default is port 43117. AB-REQ-34's literal port
20128 describes the separate Agent Brain/OmniRoute target and must not be copied
into this Prodex L2 template.

### 5. Pure offline harnesses

Add:

- `multica-auth-work/scripts/ops/tests/multica-prodex-launcher-harness.sh`
- `multica-auth-work/scripts/ops/tests/multica-prodex-service-harness.sh`

They use temporary synthetic files and fake executables only. They must not call
`systemctl`, start a daemon, connect to a database, access a provider, or require
network access.

## Startup validation and failure semantics

Validation order is security-significant:

1. Require Linux plus an available systemd user-service environment.
2. Validate the environment file as regular, non-symlink, service-user-owned,
   mode 0600 on an approved POSIX filesystem.
3. Parse the strict allowlist and required nonsecret fields.
4. Validate absolute executable/sidecar paths and executable status; validate
   pinned identity where the frozen 1.1-1.3 contract provides it.
5. Validate state/parent directory ownership and modes.
6. Require an HTTP loopback L2 base URL and an adapter bind that resolves to the
   same endpoint; reject wildcard and non-loopback binds.
7. Validate secret-reference metadata, then and only then read the references.
8. Construct a clean environment, applying trusted values last and setting only
   the opaque configuration-source label.
9. `exec` the foreground daemon.
10. Retain `LoadConfig` and L2 readiness as the second validation layer.

Every error exits nonzero. Required mode has no raw/native/provider fallback;
systemd may retry only within bounded limits.

## Rollback contract

The installer keeps one last-known-good rendered unit and reference-only
environment revision. It validates a candidate before atomic replacement. If
activation fails, it restores the previous pair, reloads the user service, and
restarts only that previous required-L2 configuration. If no good revision
exists, it leaves the service stopped and reports a redacted failure.

Rollback must never automatically launch a native/raw route, copy provider
credentials, establish a second router, or weaken a file mode. A break-glass
disable flow requires a separate authorized task and is not part of 2.1-2.2.

## Platform boundaries

- **Linux:** user-systemd only for this slice.
- **WSL:** supported only when systemd is enabled and files reside on a verified
  POSIX filesystem. Otherwise fail closed with an actionable message; never
  suggest `source`/`export` as fallback.
- **Containers:** out of scope. A later slice must use orchestrator secret mounts
  and container DNS, not the host unit or a container-local loopback assumption.
- **macOS/Windows:** launchd and Windows service integration are out of scope.
  Validation returns a stable unsupported-platform result before reading secrets.

## Pure offline security acceptance tests

The implementation evidence should include deterministic assertions for:

1. mode 0600, service-user ownership, regular/non-symlink files, mode-0700
   private parents, and an approved POSIX filesystem;
2. rejection of 0640/0644, symlinks, wrong owner, relative/non-executable paths,
   missing files, duplicate/unknown keys, control characters, and shell syntax;
3. injectable filesystem/platform probes proving rejection of drvfs/9p/CIFS,
   WSL without systemd, macOS, and Windows without touching secret contents;
4. loopback-only HTTP base URL and exact adapter/base-URL agreement, with
   wildcard/non-loopback/container assumptions rejected;
5. atomic mode-0600 generation and preservation of the prior revision when
   candidate validation fails;
6. sentinel secrets absent from stdout, stderr, argv capture, diffs, and the
   generated reference-only environment file;
7. a fake child environment recorder proving `env -i` strips provider variables
   and trusted values override inherited values;
8. an exact foreground command with no self-backgrounding or competing PID file;
9. restoration of prior hashes and modes on rollback, with a stopped/fail-closed
   result when no prior good revision exists;
10. static unit/template assertions, `bash -n`, available `shellcheck`, each
    harness repeated x20, and optional `systemd-analyze verify` when installed.

These are future acceptance criteria. None was executed as acceptance evidence
by this design-only assignment.

## Task, requirement, and evidence mapping

| OpenSpec task | Security/architecture mapping | Proposed future evidence IDs |
|---|---|---|
| 2.1 persistent launcher/service and mode-0600 environment file | AB-REQ-04 fail-closed readiness; AB-REQ-20 restricted Linux secret source; AB-REQ-21 secret-safe evidence; AB-REQ-22 no fallback; AB-REQ-34 topology principle only; AB-REQ-36 safe rollback; AB-REQ-38 operator handover | `EV-PP-2.1-LAUNCHER`, then independent `EV-PP-2.1-2.2-REVIEW` |
| 2.2 persistent L2/adapter/PostgreSQL configuration with redacted secrets | AB-REQ-16 one service-secret boundary; AB-REQ-17 inherited-provider denial; AB-REQ-18 trusted configuration last; AB-REQ-20/21 secret handling; AB-REQ-34 host/WSL versus container topology; AB-REQ-37 legacy-removal gate | `EV-PP-2.2-ENV`, then independent `EV-PP-2.1-2.2-REVIEW` |

The EV identifiers are proposals following the existing `EV-PP-*` convention;
they are not registered, awarded, or accepted here. Tasks 2.1-2.2 are a legacy
integration slice and cannot by themselves satisfy the final-target claims in
AB-REQ-16/17. The mapping means they must not regress those constraints.

## Non-conflicting implementation order

1. Freeze and record the accepted input contract from the still-open 1.1-1.3
   lane; do not edit its locked files.
2. An ops owner adds only the four new deploy/ops files and two new test
   harnesses proposed above.
3. A security reviewer checks the reference-only secret model before activation
   behavior is implemented or tested.
4. Run pure offline harnesses and static checks. Do not exercise systemd, DB,
   network, credentials, or providers.
5. Obtain independent evidence review before any OpenSpec checkbox changes.

No work is proposed for task 2.3, central Go files, `.env.example`, the existing
credential-isolation script, runbooks, OpenSpec artifacts, or planning ledgers.

## Explicit non-claims

- No product, test, OpenSpec, task, planning-ledger, or evidence-index file was
  changed by this assignment.
- No credential or environment-file contents were read.
- No DB, network, provider, daemon, or systemd service was invoked.
- No task checkbox, evidence grade, EV registration, or acceptance status is
  changed.
- This artifact is an implementation plan only; tasks 2.1 and 2.2 remain MISSING.
