# Cookie Relay Audit — prodex-runtime-cookies

> **Phase:** P4 (State/Security) — Task 4.7
> **REQ:** REQ-30 (Cookie relay: auditar superfície de auth/sessão)
> **Author:** Gemini#Pro
> **Date:** 2026-07-05
> **Status:** ACTIVE
> **Source:** Diligencias/00c_PRODEX_CRATE_COVERAGE.md (crate `prodex-runtime-cookies`)

## Overview

The `prodex-runtime-cookies` crate handles cookie relay for authentication and session persistence across vendor API interactions. This audit documents the security surface and required controls.

## Crate Purpose

`prodex-runtime-cookies` provides:
- **Cookie persistence** between requests to vendor APIs (maintaining authenticated sessions)
- **Session affinity** via cookie jar (vendor APIs that use cookies for session state)
- **Auth relay** for providers that use cookie-based authentication (e.g., browser-based auth flows)

## Security Surface Analysis

### 1. Attack Vectors

| # | Vector | Risk | Mitigation |
|---|---|---|---|
| C1 | **Cookie theft via logs** | 🔴 HIGH — session cookies in logs = session hijack | All cookies MUST be redacted by `prodex-redaction` before logging |
| C2 | **Cross-vendor cookie leakage** | 🔴 HIGH — vendor A's cookies sent to vendor B | Cookie jar MUST be isolated per vendor/profile |
| C3 | **Cookie persistence on disk** | 🟡 MED — stored cookies accessible to other processes | Cookie jar location MUST have `chmod 600`; in POSIX FS only (see 4.11) |
| C4 | **Cookie replay attack** | 🟡 MED — stolen cookies reused after session ends | Cookies MUST have expiry; session cookies cleared on `StopSession` |
| C5 | **PRODEX_ALLOW_UNSAFE_CHILD_ENV cookie leak** | 🔴 HIGH — child process inherits cookie env | Cookies MUST NOT be passed via environment; `UNSAFE_CHILD_ENV=off` enforced |
| C6 | **Cookie in audit events** | 🟡 MED — cookie value appears in audit trail | Audit events MUST only contain cookie name + hash, never value |

### 2. Required Controls

| Control | Requirement | Verification |
|---|---|---|
| **Isolation** | Cookie jar per vendor × per profile. No cross-contamination. | Inspect `$PRODEX_HOME/profiles/<name>/cookies/` — one jar per vendor |
| **Redaction** | Cookie values NEVER in logs/traces/audit. Only name + existence. | Inject test cookie, verify redacted in all surfaces |
| **Persistence** | Cookie jar file permissions = `0600`. POSIX FS only. | `stat -c '%a' <cookie_jar>` == `600` |
| **Lifecycle** | Session cookies cleared on `StopSession` event. | Start session, get cookies, stop session, verify jar empty |
| **Encryption** | Cookie jar at rest: encrypted or in secure enclave (KMS ref). | Inspect storage format — must not be plaintext JSON |
| **Scope** | Only relay cookies to same-origin vendor API. No third-party relay. | Inspect request headers — `Cookie` header only to matching domain |

### 3. Integration with Redaction Policy

The redaction policy (`docs/security/redaction-policy.md`) MUST include:

```
| Cookie value pattern | cookie=<value> | cookie=****REDACTED**** |
| Set-Cookie header    | Set-Cookie: ... | Set-Cookie: ****REDACTED**** |
```

### 4. Integration with Audit Taxonomy

Cookie-related events fit under existing audit types:
- `account_selection` — may include cookie jar initialization
- `continuation_binding` — session affinity via cookie
- No new event type needed — cookie is a transport detail, not a business event

## Test Cases

| # | Test | Expected | Gate |
|---|---|---|---|
| T1 | Log contains `Set-Cookie:` header from vendor API | Cookie value redacted | G8 |
| T2 | Cookie jar for vendor A inspected during vendor B session | Different jar / no cross-contamination | — |
| T3 | `StopSession` called | Session cookies cleared from jar | — |
| T4 | Cookie jar file permissions | `0600` on POSIX FS | G9 |
| T5 | Audit event includes cookie reference | Only cookie name + hash, never value | G8 |

> **⚠️ NOTE:** Live testing requires prodex binary (P0 blocker). This audit defines the security surface and test cases. Evidence in `.deploy-control/evidence/p4-cookie-relay.md` after P0.

## Recommendations

1. **DEFAULT:** Cookie relay ENABLED only for vendors that require it (session-based APIs)
2. **AUDIT:** Cookie jar access logged as part of `account_selection` event
3. **CLEANUP:** Cookie jars older than 24h auto-purged
4. **ENCRYPTION:** Cookie jar files encrypted at rest (AES-256 or via OS keyring)
