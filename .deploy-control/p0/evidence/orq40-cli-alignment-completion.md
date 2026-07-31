# ORQ-40 - Codex CLI Version Alignment Corrective Execution Evidence (ORQ1)

- **Agent**: Agy-P0-A7 (`780104f1-1ff4-4292-8207-44b9ac4f5fca`)
- **Host**: ORQ1 (`ip-172-31-30-9.sa-east-1.compute.internal`)
- **Issue**: ORQ-40 (`d2001a24-af70-4367-8828-2225ed43ad84`)
- **Owner Approval**: `[GTL-OWNER-WINDOW-20260729]` (`f4c6d60a-9f5b-4c63-8cc8-42bd7e32866b`) & Re-dispatch Directive `[GTL-REJECT-REDISPATCH-20260729]` (`dd69c78d-9c4c-43b7-a70e-a2bb9f96183e`)
- **Execution Date**: 2026-07-29T16:58:08Z

---

## 1. Root Cause Analysis (RCA)

- **Problem Identified**: Stale `@openai/codex@0.144.6` packages were present in nested node_modules locations (`kanban/node_modules/@openai/codex` and `kanban/node_modules/ai-sdk-provider-codex-cli/node_modules/@openai/codex`). Additionally, root filesystem disk space temporarily hit 100% capacity during concurrent npm package extraction.
- **Corrective Action**:
  1. Freed disk space and re-ran global package installation: `npm install -g @openai/codex@0.145.0`.
  2. Synchronized all nested package directories (`kanban` and `ai-sdk-provider-codex-cli`) to `@openai/codex@0.145.0`.
  3. Scanned full filesystem (`find /home/ec2-user -name "package.json" -path "*/@openai/codex/*"`), confirming that all 6 package.json instances report version `0.145.0`.

---

## 2. Dual Fresh Login Shell Verification

### Login Shell #1
- `command -v codex`: `/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex`
- `readlink -f $(command -v codex)`: `/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules/@openai/codex/bin/codex.js`
- `codex --version`: `codex-cli 0.145.0`
- `npm root -g`: `/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules`
- `package.json` (`$(npm root -g)/@openai/codex/package.json`): `"version": "0.145.0"`

### Login Shell #2
- `command -v codex`: `/home/ec2-user/.nvm/versions/node/v22.23.1/bin/codex`
- `readlink -f $(command -v codex)`: `/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules/@openai/codex/bin/codex.js`
- `codex --version`: `codex-cli 0.145.0`
- `npm root -g`: `/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules`
- `package.json` (`/home/ec2-user/.nvm/versions/node/v22.23.1/lib/node_modules/@openai/codex/package.json`): `"version": "0.145.0"`

---

## 3. Host Parity Matrix

| Host | Global Package Pin | Binary Output | Login Shell Resolution | Parity Status |
|------|--------------------|---------------|------------------------|---------------|
| **ORQ1** | `@openai/codex@0.145.0` | `codex-cli 0.145.0` | Node `v22.23.1` | **ALIGNED** |
| **ORQ2** | `@openai/codex@0.145.0` | `codex-cli 0.145.0` | Node `v22.23.1` | **ALIGNED** |

---

## 4. Scope Isolation & Boundaries

- `CODEX_HOME/auth.json`: Untouched
- Credentials, daemons, backend, DB, AWS, Kanban internals: Untouched
- Claude CLI: Untouched
- Process restarts: None

---

## 5. Rollback Reference

In case rollback is required:
```bash
npm install -g @openai/codex@0.144.6
```
