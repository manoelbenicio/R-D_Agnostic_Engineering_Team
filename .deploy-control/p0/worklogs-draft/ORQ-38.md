# WORKLOG DRAFT — ORQ-38

STATE: DRAFT_NOT_POSTED
IDEMPOTENCY_MARKER: WORKLOG-V1:ORQ-38:4e955a7bfed1be5944b917c7f8582feb0e3e7274d911ed93fad06efde377abd5

- **Card Identifier:** ORQ-38
- **Card UUID:** fd5c4d55-8ce5-412f-9f15-db666831bfdb
- **Title:** Kanban issue-number identifier collision (reporting-level)
- **Measured Time / Agent:** 2026-07-27T13:42:00Z | Codex56#A | Duration: NOT_MEASURED
- **Evidence Artifact & SHA256:** `.deploy-control/p0/evidence/orq38-issue-get-workspace-contract-audit.md` (`4e955a7bfed1be5944b917c7f8582feb0e3e7274d911ed93fad06efde377abd5`)
- **Commits:** NONE (READ_ONLY audit)
- **Gates & Verdict:** PASS — Auditoria do contrato do manipulador `GET /api/issues/{id}` realizada com sucesso.
- **Rework & Retractions:** Retratação de premissa anterior: provado que `GET /api/issues/{id}` sem `workspace_id` retorna `HTTP 400` fail-closed (e não objeto nulo). O relato falso foi gerado por erro de parsing do cliente ao omitir a verificação de código HTTP.
- **Runtime Mutation & Rollback:** Zero edições no servidor, zero escrita em banco.
- **Blocker & Next Steps:** Adição de testes unitários herméticos no handler cobrindo exigência de `workspace_id` e padronização da verificação no cliente da frota.
