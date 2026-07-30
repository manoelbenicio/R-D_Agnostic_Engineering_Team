## 1. Secret ownership and resolution

- [ ] 1.1 Record the exact secret identity, ARN, Region, version stage, and accountable owner without retrieving plaintext.
- [ ] 1.2 Obtain owner approval for least-privilege metadata-only IAM or an owner-reviewed loopback Secrets Manager Agent on port 2773.
- [ ] 1.3 Prove the selected resolution path exposes metadata only and does not retrieve or log plaintext secret material.

## 2. Authorized cutover preparation

- [ ] 2.1 Record the exact canonical root, custom TMPDIR, and `*.service` unit tuple for both apply and rollback.
- [ ] 2.2 Test the byte-exact rollback using that same tuple in an isolated environment.
- [ ] 2.3 Obtain a separately authorized window requiring two queue-zero reads and a PostgreSQL admission lock.

## 3. Runtime verification

- [ ] 3.1 Apply only during the authorized window and verify daemon identity, process state, health, restart count, and queue state.
- [ ] 3.2 Execute a bounded credential-mode and MCP canary without reading or printing credential values.
- [ ] 3.3 Record rollback or acceptance evidence and release admission only after all runtime checks pass.

## 4. Separation

- [ ] 4.1 Keep ORQ-58 deployment and acceptance work separate from this runtime hardening cutover.
