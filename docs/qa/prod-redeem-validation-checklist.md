# PROD Redeem Validation Checklist

Status: PRE-DEPLOY REQUIRED FOR AUTO-REDEEM

## 1. Decision

Manual `prodex redeem <profile>` may be validated in controlled PROD only with
owner approval and scrubbed evidence.

`--auto-redeem` must remain disabled until this checklist is satisfied.

## 2. Required Guard Conditions

Redeem may be attempted only when:

- provider is OpenAI/Codex profile;
- target profile is approved;
- weekly window is exhausted;
- no other eligible profile has weekly quota;
- reset is not imminent;
- kill switch `auto_redeem` is not disabled;
- cooldown allows attempt;
- audit event sink is healthy.

Redeem must not be attempted when:

- 5h-only exhaustion;
- thin/critical but other profiles available;
- provider is not OpenAI/Codex;
- profile auth invalid;
- audit unavailable;
- owner has not approved validation.

## 3. Matrix

| Case | Expected |
|---|---|
| No credit | `redeem_no_credit`, no retry storm |
| Credit present | `redeem_succeeded`, same profile retried |
| Near natural reset | `redeem_rejected`, reason reset imminent |
| Weekly exhausted | eligible if no other weekly quota exists |
| 5h-only exhausted | rejected |
| All profiles exhausted | eligible if credit present |
| Non-OpenAI provider | unsupported/rejected |
| Invalid profile | fail closed |

## 4. Evidence

Evidence must include:

- profile alias only;
- timestamp;
- quota state summary;
- command summary without tokens;
- event id;
- result;
- scrubber confirmation.

Forbidden:

- raw OAuth tokens;
- cookies;
- full `auth.json`;
- raw account secret;
- raw backend response containing sensitive values.

## 5. Auto-Redeem Promotion Gate

`--auto-redeem` can be enabled only after:

- at least one no-credit case validated;
- at least one rejected guard validated;
- at least one success or explicit not-available outcome recorded;
- cooldown/idempotency verified;
- owner approval recorded.

