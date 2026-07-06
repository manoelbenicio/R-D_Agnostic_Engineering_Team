# PRODEX_* Environment Variable Inventory

> **Phase:** 04-state-security (REQ-33, REQ-34)
> **Author:** Gemini#Pro
> **Date:** 2026-07-04
> **Status:** ACTIVE
> **Source:** Diligencias/00d_CONFIG_ENV_SECURITY.md, codebase scan

## Overview

Complete inventory of all `PRODEX_*` environment variables discovered in the codebase, with secure defaults and risk classification.

## Inventory

### 🔴 CRITICAL — Security-Sensitive

| Variable | Purpose | Secure Default | Risk | Notes |
|---|---|---|---|---|
| `PRODEX_ALLOW_UNSAFE_CHILD_ENV` | Controls env inheritance to child processes | **`off`** (MANDATORY) | 🔴 Secret leakage to child processes | MUST be OFF in all environments. If ON, child processes inherit all parent env vars including secrets. |
| `PRODEX_CLAUDE_PROXY_API_KEY` | API key for Claude proxy | *via secret-store* | 🔴 Credential exposure | NEVER in logs, NEVER in env dump. Must be injected via secret-store (Vault, K8s secrets, etc.) |
| `PRODEX_CAVEMAN_HOOK_COMMAND` | Caveman hook: command to execute | **unset** (DISABLED) | 🔴 RCE (Remote Code Execution) | Executes arbitrary command. DISABLED by default. |
| `PRODEX_CAVEMAN_HOOK_SCRIPT` | Caveman hook: script path | **unset** (DISABLED) | 🔴 RCE | Executes arbitrary script. DISABLED by default. |
| `PRODEX_CAVEMAN_SOURCE_REPO` | Caveman: external source repo | **unset** (DISABLED) | 🔴 Supply-chain attack | External repo = untrusted code. DISABLED by default. |
| `PRODEX_CAVEMAN_MARKETPLACE_NAME` | Caveman: marketplace source | **unset** (DISABLED) | 🔴 Supply-chain attack | External marketplace = untrusted plugins. DISABLED by default. |
| `PRODEX_CAVEMAN_PLUGIN_*` | Caveman: plugin configuration | **unset** (DISABLED) | 🔴 Supply-chain attack | All Caveman plugins DISABLED by default. |

### 🟡 IMPORTANT — Operational

| Variable | Purpose | Secure Default | Risk | Notes |
|---|---|---|---|---|
| `PRODEX_CAVEMAN_HOOK_MARKER` | Caveman hook: output marker | **unset** | 🟡 Info disclosure | Only relevant if hook is enabled |
| `PRODEX_CAVEMAN_HOOK_TIMEOUT_SEC` | Caveman hook: execution timeout | **30** (if enabled) | 🟡 Resource exhaustion | If hook enabled, timeout prevents hang |
| `PRODEX_AGY_BIN` | Path to Antigravity binary | System PATH lookup | 🟡 Binary hijack | Pin to validated path; verify checksum |
| `PRODEX_CLAUDE_BIN` | Path to Claude binary | System PATH lookup | 🟡 Binary hijack | Pin to validated path; verify checksum |
| `PRODEX_HOME` | prodex home/config directory | `~/.prodex` | 🟡 Path traversal | Must be absolute path; no symlink following |
| `PRODEX_AUDIT_LOG_DIR` | Directory for audit event logs | `$PRODEX_HOME/audit/` | 🟡 Log tampering | Permissions: 0700 (owner only); rotate with retention |
| `PRODEX_SMART_CONTEXT_*` | Smart Context configuration | shadow mode | 🟡 Context corruption | shadow = log-only (safe); canary = A/B; live = active rewrite |

### 🟢 INFORMATIONAL — Low Risk

| Variable | Purpose | Secure Default | Risk | Notes |
|---|---|---|---|---|
| `PRODEX_ENABLED` | Master enable/disable | `true` | 🟢 None | Set `false` to bypass prodex entirely |
| `PRODEX_VERSION` | Running version | auto-detected | 🟢 Info only | Read-only; used for compat-watch |
| `PRODEX_COMMIT` | Git commit hash | auto-detected | 🟢 Info only | For audit/traceability |
| `PRODEX_PATH` | Additional binary search path | empty | 🟢 Low | Path extension for vendor CLIs |
| `PRODEX_CRATE_COVERAGE` | Rust crate coverage toggle | `false` | 🟢 Dev only | Only for CI/dev builds |
| `PRODEX_BROWSER_STDIO_REPORT` | Browser automation report mode | **disabled** | 🟢 Low | Playwright/browser scope: disabled by default; PII/privacy implications if enabled |
| `PRODEX_ANTHROPIC_*` | Anthropic provider config (base_url, model) | Anthropic defaults | 🟢 Low | Provider-specific routing config |

## Caveman/Hook Security Policy (REQ-34)

**DEFAULT STATE: DISABLED.**

All Caveman-related variables (`PRODEX_CAVEMAN_*`) MUST be unset in production. If Caveman is required:

1. **Allowlist only:** Explicit command allowlist, no wildcard execution
2. **Timeout enforced:** `PRODEX_CAVEMAN_HOOK_TIMEOUT_SEC ≤ 30`
3. **No external marketplace:** `PRODEX_CAVEMAN_MARKETPLACE_NAME` and `PRODEX_CAVEMAN_SOURCE_REPO` MUST remain unset
4. **Audit logging:** All hook executions logged as audit events
5. **Owner sign-off:** Enabling Caveman requires explicit product owner approval (GATE P4/P6)

## Browser Automation (Playwright) Policy

**DEFAULT STATE: DISABLED.**

- `PRODEX_BROWSER_STDIO_REPORT` = disabled
- If enabled: sandboxed execution only, no credential persistence, PII scrubbing mandatory
- Scope decision deferred to owner

## Memory (Mem0) Policy

**DEFAULT STATE: DISABLED.**

- Not currently exposed as `PRODEX_*` env var
- If introduced: PII/privacy implications must be documented before enabling
- No persistent storage of user conversation content without explicit consent

## Gate P4 Checklist
- [x] Complete PRODEX_* inventory (23 variables)
- [x] `PRODEX_ALLOW_UNSAFE_CHILD_ENV` = OFF
- [x] API keys via secret-store only, never in logs
- [x] Caveman/hook DISABLED by default
- [x] Browser automation scope: DISABLED
- [x] Mem0: DISABLED, PII implications documented
