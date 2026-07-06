# PII Scrubbing Test — prodex-presidio + prodex-redaction

> **Phase:** P4 (State/Security) — Task 4.6
> **REQ:** REQ-28 (Redaction real via prodex-presidio + prodex-redaction)
> **Author:** Gemini#Pro
> **Date:** 2026-07-05
> **Status:** ACTIVE
> **Gate:** G8 (secrets redaction test)

## Overview

Task 4.6 requires testing PII scrubbing via the prodex native redaction engines. This document specifies the test matrix and expected behavior.

## Redaction Engines (prodex-native)

| Crate | Purpose | Technique |
|---|---|---|
| `prodex-presidio` | PII detection (NER-based) | Microsoft Presidio integration — entity recognition for names, emails, phone numbers, credit cards, SSNs |
| `prodex-redaction` | Log/diagnostic scrubbing | Regex-based pattern matching — tokens, JWTs, API keys, env vars |

Both crates are part of the `redaction` domain in the prodex crate map (ref: `Diligencias/00c_PRODEX_CRATE_COVERAGE.md`).

## Test Matrix

### prodex-redaction (regex-based)

| # | Input | Expected Output | Pattern |
|---|---|---|---|
| R1 | `ghp_ABCDEF1234567890ABCDEF1234567890ABCD` | `ghp_****REDACTED****` | GitHub PAT |
| R2 | `sk-proj-abc123def456ghi789jkl012mno345` | `sk-****REDACTED****` | OpenAI key |
| R3 | `sk-ant-api03-xyzabc123` | `sk-ant-****REDACTED****` | Anthropic key |
| R4 | `AKIAIOSFODNN7EXAMPLE` | `AKIA****REDACTED****` | AWS Access Key |
| R5 | `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U` | `JWT:****REDACTED****` | JWT (3-part) |
| R6 | `xoxb-1234567890-abcdef` | `xoxb-****REDACTED****` | Slack Bot Token |
| R7 | `AIzaSyA1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q` | `AIza****REDACTED****` | Google API key |
| R8 | `PRODEX_CLAUDE_PROXY_API_KEY=sk-real-secret-here` | `PRODEX_CLAUDE_PROXY_API_KEY=****REDACTED****` | Env var with secret |

### prodex-presidio (NER-based)

| # | Input | Expected Detection | Entity Type |
|---|---|---|---|
| P1 | `User John Smith logged in from 192.168.1.1` | `John Smith` → `[PERSON]`, `192.168.1.1` → `[IP_ADDRESS]` | PERSON, IP |
| P2 | `Contact: john@example.com, +1-555-0123` | `john@example.com` → `[EMAIL]`, `+1-555-0123` → `[PHONE]` | EMAIL, PHONE |
| P3 | `CC: 4111-1111-1111-1111 exp 12/28` | `4111-1111-1111-1111` → `[CREDIT_CARD]` | CREDIT_CARD |
| P4 | `SSN: 123-45-6789` | `123-45-6789` → `[US_SSN]` | US_SSN |

## Smoke Test Procedure

```bash
# Prerequisites: prodex binary available (post-P0)

# Test 1: prodex-redaction (regex)
echo "Token: ghp_ABCDEF1234567890ABCDEF1234567890ABCD" | prodex redact --engine regex
# Expected: Token: ghp_****REDACTED****

# Test 2: prodex-presidio (NER)
echo "User John Smith logged in" | prodex redact --engine presidio
# Expected: User [PERSON] logged in

# Test 3: Combined pipeline
echo "ghp_ABCDEF123 belongs to John Smith" | prodex redact --engine all
# Expected: ghp_****REDACTED**** belongs to [PERSON]
```

> **⚠️ NOTE:** Live smoke test requires prodex binary (P0 blocker). This document defines the test matrix. Evidence will be recorded in `.deploy-control/evidence/p4-redaction-smoke.md` after P0 is green.

## Surfaces Covered

| Surface | Engine | Status |
|---|---|---|
| Application logs (`PRODEX_AUDIT_LOG_DIR`) | Both | Pending live test |
| Error messages / stack traces | prodex-redaction | Pending live test |
| Audit events | prodex-redaction | Schema enforces no raw secrets |
| CLI stdout/stderr | prodex-redaction | Pending live test |
| Evidence files | Pre-commit scrub (manual) | Active — all evidence files scrubbed |

## Gate G8 Criteria
- [ ] prodex-redaction regex patterns match all 8 test cases (R1–R8)
- [ ] prodex-presidio NER detects all 4 entity types (P1–P4)
- [ ] Combined pipeline scrubs both regex and NER patterns
- [ ] No raw secret appears in any audit event payload
- [ ] Evidence scrubbed before commit
