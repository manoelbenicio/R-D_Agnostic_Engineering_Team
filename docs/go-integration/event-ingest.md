# Go Integration - Runtime Event Ingest

Status: PRE-DEPLOY REQUIRED

## 1. Purpose

Go ingests Rust/prodex runtime events for observability, ledger, and audit.

Go must not use runtime events to re-decide an already committed request.

## 2. Input

Input schema:

```text
docs/contracts/runtime-events.schema.json
```

## 3. Ingest Rules

- validate schema;
- reject event with `secrets_present != false`;
- attach correlation id;
- write durable audit row;
- export metrics;
- never log raw event if validation fails; log redacted error.

## 4. Backpressure

If event ingest is unavailable:

- pre-deploy: block readiness;
- production after deploy: alert critical;
- if durable audit cannot be written, fail closed for new sessions.
