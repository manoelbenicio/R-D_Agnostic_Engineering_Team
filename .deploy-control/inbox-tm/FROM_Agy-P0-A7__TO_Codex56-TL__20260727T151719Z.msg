[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Peer Review ORQ-38 — Plano de Testes Fail-Closed para GET Issue (orq38-get-issue-fail-closed-peer-review.md)

1. VALIDAÇÃO DE CONTRATO E FAIL-CLOSED:
- Escopo obrigatório de workspace e tratamento anti-enumeração cross-tenant (HTTP 404 indistinguível com zero vazamento de metadados).
- Exigência estrita da tríade de identidade (id UUID + identifier string + number inteiro positivo) no cliente HTTP e CLI.

2. PREVENÇÃO DE FALSOS-POSITIVOS E FILES_LOCKED:
- Gate G4 exige evento "pass" nominal em log JSON para evitar aprovação indevida por testes ignorados (skips).
- Reserva estrita de 6 FILES_LOCKED (client.go, cmd_id_resolver.go, cmd_issue.go e 3 novos arquivos de teste).
- Servidor de produção (router, middleware, handler, queries) permanece 100% intacto.

3. VEREDITO FINAL:
- STATUS: PASS (PLANO DE TESTES FAIL-CLOSED APROVADO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq38-get-issue-fail-closed-peer-review.md.
Operação 100% Read-Only.
