# prodex Fork Map

Status: PRE-DEPLOY REQUIRED FOR FORK MILESTONE

## 1. Source

Primary source: official `christiandoxa/prodex` repo and npm package.

## 2. Relevant Areas

Runtime hot path:

- runtime proxy;
- runtime launch;
- quota;
- state;
- gateway;
- Smart Context;
- provider core/conformance;
- redeem.

Docs to preserve as design input:

- `docs/architecture.md`
- `docs/runtime-policy.md`
- `docs/state-model.md`
- `docs/provider-conformance.md`
- `docs/provider-capabilities.md`
- `docs/smart-context.md`
- `docs/deployment.md`
- `docs/testing.md`

## 3. Fork Boundary

Keep:

- hard affinity model;
- rotate-before-commit invariant;
- Smart Context safety model;
- replay/canary/shadow controls;
- gateway policy machinery;
- provider capability/conformance fixtures.

Change for Multica:

- externalize policy from Multica Go;
- emit Multica runtime events schema;
- use Postgres/Redis shared state;
- enforce Multica kill switches;
- remove/disable product branding conflicts;
- add deployment gates and redaction checks.

## 4. Do Not Change Without Tests

- continuation binding;
- SSE/WebSocket handling;
- first committed response logic;
- Smart Context validation;
- credential/profile isolation;
- redeem guards.

