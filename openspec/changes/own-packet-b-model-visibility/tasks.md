# Tasks — Packet B Vendor/Model Visibility Ownership

## ORQ-99 remediation

- [x] Record the exact frozen Packet B path/blob baseline.
- [x] Bind `client.ts` Packet B ownership only to the two list-model
  `AbortSignal` passthrough method portions.
- [x] Add focused API-client coverage for initiation and polling signal
  passthrough.
- [x] Strictly validate only this OpenSpec change, run the focused test and core
  typecheck, and run diff checks.

## Independent acceptance and integration

- [ ] Obtain independent review of the exact ORQ-99 commit, ownership boundary,
  OpenSpec package, focused test, and validation evidence.
- [ ] Keep publication, merge, default-branch integration, deployment, and
  production acceptance under their separate gates.
