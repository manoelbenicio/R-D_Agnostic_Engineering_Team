[FROM Codex56#B w7:p4][ORQ-17 READ-ONLY AUDIT DONE][2026-07-27T11:30:02Z]
Evidencia consolidada em .deploy-control/p0/evidence/orq-9-issues-exit-criteria.md:653-723.
Veredito ORQ-17: PRESERVADA/OWNER. auth.go:107-110 aplica bypass a toda API browser; auth.go:43-52 so o desliga quando FRONTEND_ORIGIN deixa de ser loopback.
docker-compose.selfhost.yml:3-8 diz literalmente "Do NOT change these bindings to 0.0.0.0"; next.config.ts:50-66 permite publicar apenas frontend por proxy same-origin.
Saida exige: URL HTTPS/ACL decididas pelo owner, senha real provisionada, /api/me anonimo=401, backend/Postgres loopback, CSRF/WS/uploads e rollback do proxy validados.
Nenhuma mutacao, teste live, rerun, deploy, restart, credencial ou board tocado; as 9 issues seguem preservadas.
Backups .pre-token-only/.pre-agy-fix/.previous sao known-bad e reintroduzem AGY task-incapaz; baseline green futuro e 88ca4f397ef841edac091f15e77b2ba52754045a471acaa311c280827d2dd1f8.
