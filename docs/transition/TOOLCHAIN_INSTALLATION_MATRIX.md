# Toolchain Installation Matrix — ORQ1 and ORQ2

Status date: 2026-07-20

## 1. Purpose

This is the authoritative installation inventory for rebuilding the DEV application node (`orq1`) and the Herdr agent node (`orq2`). Install only the tools required by a node's role. Avoid filling the 8 GB agent node with application images, browser bundles, mobile SDKs, or duplicate language caches.

Version precedence:

1. repository-pinned version;
2. transition runbook version;
3. currently validated environment version;
4. newest compatible version only when no pin exists.

## 2. GSD clarification

GSD is required as the project's execution/governance framework, but this repository does not contain or reference a required `gsd` executable or pinned external GSD package.

The installed project GSD is the versioned document system under `.planning/`, especially:

- `.planning/agent-brain-v3/PROJECT.md`
- `.planning/agent-brain-v3/ROADMAP.md`
- `.planning/agent-brain-v3/STATE.md`
- `.planning/agent-brain-v3/TRACEABILITY.md`
- `.planning/agent-brain-v3/DECISIONS.md`
- `.planning/agent-brain-v3/RISKS.md`
- `.planning/agent-brain-v3/HERDR_TRANSPORT.md`
- `.planning/agent-brain-v3/phases/`

Therefore:

- no unpinned `npm install -g gsd` is authorized;
- no random “Get Shit Done” package may be installed by name;
- Git, Markdown-capable agents, OpenSpec, and the repository files provide the required GSD workflow;
- an external GSD CLI/plugin may be added only after its upstream identity, version, checksum, installation method, and compatibility are approved and recorded.

## 3. Required on both nodes

| Tool | Target | Purpose | Required |
|---|---|---|---|
| Amazon Linux 2023 base utilities | Supported current image | Host operating system | Yes |
| Bash | System version | Runbooks, isolation bootstrap, smoke scripts | Yes |
| Git | 2.40+ | Source, branches, worktrees, evidence | Yes |
| OpenSSH client/server | System-supported | Encrypted administration and transfer | Yes |
| Tailscale | Validated fleet version or compatible | Private node connectivity | Yes |
| rsync | 3.x | Resumable backup/source transfer | Yes |
| curl | Current OS package | Health probes and installers | Yes |
| CA certificates | Current OS package | TLS validation | Yes |
| tar, gzip, xz, unzip | Current OS packages | Backups and tool archives | Yes |
| coreutils/findutils | Current OS packages | Checksums, file inspection, automation | Yes |
| util-linux `flock` | Current OS package | Credential registry and lease locking | Yes |
| Python | 3.11+; 3.12 validated | JSON validation, dashboards, harnesses | Yes |
| ripgrep `rg` | 14+; 15 validated | Fast source/task searches | Yes |
| jq | Current OS package | Operational JSON inspection | Yes |
| OpenSSL | Current OS package | DEV secret generation and TLS utilities | Yes |
| GNU Make | Current OS package | Existing repository targets | Yes |

Helpful but not a runtime dependency: `tree`, `lsof`, `procps-ng`, `bind-utils`, and `net-tools` when an older diagnostic specifically requires it.

## 4. ORQ1 — application node requirements

### Mandatory baseline

| Tool | Required version/contract | Notes |
|---|---|---|
| Docker Engine | Modern version supporting Compose v2 and BuildKit | Docker 25 was previously observed; newer compatible is acceptable |
| Docker Compose v2 plugin | `docker compose`, not legacy `docker-compose` | Required by the restart runbook |
| Docker Buildx | Compatible with installed Docker | Required for reliable local image builds |
| OpenSpec CLI | 1.4.1 validated | Required for strict change validation |
| Node.js | 22.x | Repository host frontend validation target |
| Corepack | Compatible with Node 22 | Activates the pinned package manager |
| pnpm | Exactly 10.28.2 | Pinned in `multica-auth-work/package.json` |

Activate the repository pnpm version with:

```bash
corepack enable
corepack prepare pnpm@10.28.2 --activate
pnpm --version
```

Expected: `10.28.2`.

### Backend toolchain

The backend module declares Go `1.26.1`. Choose one approved mode:

1. install Go 1.26.1 on the host; or
2. use the pinned `golang:1.26.1` container documented by the restart runbook.

The container mode is preferred when host installation would duplicate a large toolchain.

### Database and state tools

| Tool | Requirement | Purpose |
|---|---|---|
| PostgreSQL client tools | Version 17 compatible | `psql`, `pg_dump`, `pg_restore` diagnostics/restores |
| SQLite CLI | Current version | `quick_check`, backup and restore verification |
| `sha256sum` | From coreutils | Backup integrity |

PostgreSQL client installation is recommended even though restore commands may run inside the PostgreSQL container.

### ORQ1 tools not required for baseline

- Herdr;
- interactive provider CLIs;
- Rust/Cargo unless an explicitly assigned Prodex-sidecar task builds on ORQ1;
- Kubernetes tooling;
- mobile SDKs;
- Playwright browsers unless browser E2E is explicitly assigned.

## 5. ORQ2 — Herdr agent node requirements

### Mandatory orchestration toolchain

| Tool | Required version/contract | Purpose |
|---|---|---|
| Herdr | 0.7.4 validated | Workspaces, panes, agent coordination |
| OpenSpec CLI | 1.4.1 validated | Task/spec inspection and strict validation |
| Git | 2.40+ | Shared clone plus isolated worktrees |
| Node.js | 22.x; 22.23.1 observed | Codex and JavaScript agents/tooling |
| Corepack | Node-compatible | pnpm activation |
| pnpm | Exactly 10.28.2 for project work | Frontend tasks only |
| Python | 3.11+ | Automation, JSON parsing, isolation harness |
| `flock` | util-linux | Credential isolation correctness |
| ripgrep | 14+ | Fast repository inspection |
| jq | Current OS package | Herdr/OpenSpec JSON operations |
| rsync | 3.x | Worktree/evidence transfer when required |

### Agent CLIs

Install only CLIs assigned to actual fleet accounts:

| CLI/runtime | Status |
|---|---|
| OpenAI Codex CLI | Required for Codex panes; already observed on ORQ2 |
| Kiro CLI | Required only for assigned Kiro pane(s) |
| OpenCode | Required only for assigned OpenCode/GLM pane(s) |
| Antigravity/`agy` | Required only for assigned Antigravity pane(s) |
| GROK CLI | Required only if GROK A–D profiles are approved and assigned |
| Cline runtime/CLI | Required only for assigned Cline work |

Every provider CLI must run only after the credential-isolation bootstrap passes. Installation does not authorize authentication. The owner performs one account login per isolated pane.

### ORQ2 credential-isolation dependencies

The installed isolation control requires Bash, `flock`, Python 3, `sha256sum`, GNU `cp`, and Herdr for stable terminal identity. Validate with:

```bash
bash /home/ec2-user/.local/lib/agent-credential-isolation/scripts/ops/tests/agent-cred-isolation-harness.sh
```

### ORQ2 capacity policy

ORQ2 has 8 GB. Keep it lean:

- use one canonical clone plus Git worktrees;
- do not duplicate `node_modules` unnecessarily;
- do not keep Docker application images on ORQ2 unless a task explicitly needs them;
- avoid Playwright browser downloads unless assigned;
- avoid separate Go/Rust caches per pane;
- inspect disk use without reading credential contents.

## 6. Conditional engineering toolchains

Install these only when an assigned task requires them.

### Rust / Prodex sidecar

- `rustup`;
- current approved stable Rust toolchain;
- `cargo`;
- `rustfmt`;
- `clippy`;
- C linker/toolchain and required system headers.

Additional release/security gates when explicitly assigned:

- `cargo-audit` 0.22.1;
- `cargo-deny` 0.19.0;
- `cross` only for cross-platform release builds.

Prodex is cold-recovery/default-OFF in the current target architecture. Do not install or activate its complete runtime merely because Rust source exists.

### Frontend browser E2E

Install Playwright browser/system dependencies only for assigned browser tests. Do not download every browser for typecheck, unit-test, or production-build tasks.

### Kubernetes deployment

Install `kubectl`, Helm 3, authenticated cluster tooling, and any required cloud CLI only if Kubernetes deployment is explicitly selected. Kubernetes is not required for the current Compose baseline.

### GitHub workflow

- GitHub CLI `gh` is optional for PR/workflow operations.
- Git LFS is required only if `.gitattributes` or `git lfs ls-files` proves repository usage.

### Security and quality tools

Recommended only at relevant integration/release gates:

- ShellCheck;
- `gitleaks`;
- `golangci-lint` only with repository-pinned configuration;
- `hadolint` for Dockerfile-specific work;
- `yamllint` for YAML-heavy deployment changes.

Do not introduce a formatter or linter without a repository configuration or gate.

## 7. OpenSpec installation contract

OpenSpec CLI is required on both nodes and version 1.4.1 is the validated baseline.

Required commands:

```bash
openspec --version
openspec list
openspec validate build-omniroute-agent-brain --strict
openspec validate persist-prodex-runtime-integration --strict
```

The repository also contains agent-specific OpenSpec skills/workflows under `.agent`, `.codex`, `.cline`, `.kiro`, and `.opencode`. These versioned files travel with Git and do not replace the CLI.

## 8. Version verification commands

Run on each node:

```bash
git --version
ssh -V
tailscale version
rsync --version | head -1
python3 --version
rg --version | head -1
jq --version
openssl version
make --version | head -1
openspec --version
```

Run on ORQ1:

```bash
docker --version
docker compose version
docker buildx version
node --version
corepack --version
pnpm --version
go version || true
psql --version || true
sqlite3 --version
```

Run on ORQ2:

```bash
herdr --version
node --version
corepack --version
pnpm --version
type -a codex
test "${HERDR_ENV:-}" = "1" || true
```

Do not print environment values, auth files, tokens, or secret-bearing URLs during verification.

## 9. Minimum acceptance checklist

### ORQ1

- [ ] Git works and the canonical repository is current.
- [ ] Docker Engine responds.
- [ ] `docker compose version` succeeds.
- [ ] Buildx responds.
- [ ] OpenSpec 1.4.1 responds.
- [ ] Node 22 is active.
- [ ] pnpm 10.28.2 is active.
- [ ] Go 1.26.1 is installed or its Docker image is usable.
- [ ] PostgreSQL/SQLite restore tooling is available directly or through approved containers.
- [ ] Tailscale and SSH connectivity work.

### ORQ2

- [ ] Herdr 0.7.4 responds.
- [ ] OpenSpec 1.4.1 responds.
- [ ] Git and worktrees work.
- [ ] Node 22 and pnpm 10.28.2 are available.
- [ ] Python, `flock`, `rg`, `jq`, and rsync are available.
- [ ] Required provider CLIs are installed.
- [ ] Credential-isolation harness passes.
- [ ] Every provider pane reports a unique isolated slot before login.

## 10. Explicit non-requirements

Do not block the current DEV baseline waiting for:

- a separate GSD CLI;
- Kubernetes;
- Redis;
- Rust/Prodex activation;
- mobile SDKs;
- every provider CLI;
- every optional linter;
- Git LFS when the repository does not use it;
- Playwright browsers for non-browser work.

Install tools from assigned work, repository pins, and acceptance gates—not from a generic developer-workstation checklist.
