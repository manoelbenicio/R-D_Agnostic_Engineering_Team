# Secrets Redaction Policy

Status: PRE-DEPLOY REQUIRED

## 1. Rule

No secret may appear in:

- logs;
- traces;
- runtime events;
- check-in/check-out records;
- evidence files;
- dashboards;
- screenshots;
- error messages;
- command output pasted into docs.

## 2. Secret Classes

Redact:

- bearer tokens;
- OAuth tokens;
- refresh tokens;
- cookies;
- API keys;
- database URLs;
- Redis URLs;
- profile auth payloads;
- `auth.json` contents;
- private keys;
- signed URLs;
- session cookies;
- provider account identifiers when unnecessary.

## 3. Allowed Identifiers

Allowed when useful:

- profile alias;
- tenant id;
- provider id;
- model id;
- event id;
- hashed account id.

Account emails should be avoided in public evidence unless explicitly needed.

## 4. Redaction Format

Use:

```text
[REDACTED:<kind>:sha256:<first12>]
```

Examples:

```text
[REDACTED:bearer:sha256:1a2b3c4d5e6f]
[REDACTED:dburl:sha256:9f8e7d6c5b4a]
```

## 5. Required Scrub Tests

Before deploy, run a scrub test against:

- sidecar logs;
- Go daemon logs;
- runtime event stream;
- deploy runbook commands;
- QA evidence snippets;
- error paths.

The test must inject fake markers:

```text
sk-test-secret
Bearer test-secret
postgres://user:pass@example/db
redis://:pass@example:6379
```

Expected result: none appear unredacted.

## 6. Fail-Closed Conditions

Deployment blocks if:

- scrubber disabled;
- event schema allows `secrets_present = true`;
- logs contain raw token-like values;
- evidence contains raw `auth.json`;
- command output includes raw connection string.
