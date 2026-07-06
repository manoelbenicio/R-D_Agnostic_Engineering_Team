# Redaction Policy

> **Phase:** 04-state-security (REQ-11)
> **Author:** Gemini#Pro
> **Date:** 2026-07-04
> **Status:** ACTIVE
> **Applies to:** All log surfaces, traces, errors, audit events, CLI output, evidence files

## 1. Scope

This policy governs redaction (masking) of sensitive data across ALL output surfaces in the Multica + prodex stack:
- Application logs (`PRODEX_AUDIT_LOG_DIR`)
- Error messages / stack traces
- Audit events (see `audit-taxonomy.md`)
- CLI stdout/stderr
- Evidence files committed to `.deploy-control/evidence/`
- Observability/tracing payloads

## 2. Mandatory Redaction Patterns

| Pattern | Type | Redacted To | Example |
|---|---|---|---|
| `ghp_[A-Za-z0-9]{36,}` | GitHub PAT | `ghp_****REDACTED****` | `ghp_ABCDEF123456…` → `ghp_****REDACTED****` |
| `sk-[A-Za-z0-9]{20,}` | OpenAI API key | `sk-****REDACTED****` | `sk-proj-abc…` → `sk-****REDACTED****` |
| `sk-ant-[A-Za-z0-9]{20,}` | Anthropic API key | `sk-ant-****REDACTED****` | |
| `AKIA[0-9A-Z]{16}` | AWS Access Key | `AKIA****REDACTED****` | |
| `eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}` | JWT (3-part) | `JWT:****REDACTED****` | |
| `xoxb-[0-9]{10,}-[A-Za-z0-9]+` | Slack Bot Token | `xoxb-****REDACTED****` | |
| `AIzaSy[A-Za-z0-9_-]{33}` | Google API key | `AIza****REDACTED****` | |
| `PRODEX_CLAUDE_PROXY_API_KEY=.*` | prodex proxy key | `PRODEX_CLAUDE_PROXY_API_KEY=****REDACTED****` | |
| Any env var value matching `key\|token\|secret\|password` | Generic secrets | `****REDACTED****` | |

## 3. Redaction Engines

| Engine | Role | Surface |
|---|---|---|
| **prodex-presidio** | NER-based PII detection | Audit logs, traces |
| **prodex-redaction** | Regex-based pattern matching | CLI output, errors, evidence |
| **log scrubbing** (pre-commit) | Git hook / CI check | Evidence files before commit |

## 4. Rules

1. **Never log raw secrets.** All surfaces MUST apply redaction BEFORE write.
2. **Fail-closed:** If redaction engine is unavailable, the surface MUST suppress the entire field rather than emit unredacted.
3. **Evidence scrubbing:** Any evidence file committed to `.deploy-control/evidence/` MUST be scrubbed. Known test stubs (jwt.io example, `ghp_ABCDEF…` placeholder) are permitted but MUST be annotated as `[TEST-STUB]`.
4. **No secret in env dump:** `PRODEX_ALLOW_UNSAFE_CHILD_ENV` MUST be OFF. Child processes MUST NOT inherit secrets via environment.
5. **Audit trail:** Redaction events themselves are logged (event type: `redaction_applied`, count of patterns matched) but WITHOUT the original value.

## 5. Smoke Test

```bash
# Inject fake token and verify masked
echo "ghp_ABCDEF1234567890ABCDEF1234567890ABCD" | prodex-redaction --check
# Expected output: ghp_****REDACTED****

echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U" | prodex-redaction --check
# Expected output: JWT:****REDACTED****
```

## 6. Exceptions

- Test fixtures in `tests/` may contain example tokens annotated with `[TEST-STUB]`.
- Hashed values (SHA-256, bcrypt) are NOT secrets and do NOT require redaction.
