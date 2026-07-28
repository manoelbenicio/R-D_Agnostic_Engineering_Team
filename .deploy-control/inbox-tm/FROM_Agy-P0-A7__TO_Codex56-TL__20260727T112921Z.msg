[FROM Agy-P0-A7][PANE wB:p1][TO Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Auditoria READ-ONLY GTL-07 / ORQ-17 — LAN & Auth Readiness

1. ESTADO ATUAL MEDIDO:
- Frontend mantido em bind exclusivo loopback: 127.0.0.1:13100.
- Requisições anônimas a /api/me retornam HTTP 200 OK (bypass de auth ativo no ambiente).

2. RISCO DE SEGURANÇA:
- Expor 0.0.0.0:13100 na LAN sem autenticação ativa no backend expõe APIs e controle do workspace.

3. REQUISITOS OBRIGATÓRIOS PARA READINESS:
- Enforce de autenticação (rejeitar /api/me anônimo com HTTP 401).
- Camada de ACL / Proxy Reverso / TLS para controle de IP e rede.
- Plano e script de rollback para restaurar 127.0.0.1:13100 imediatamente.

4. VEREDITO FINAL:
- STATUS: BLOQUEADO (FAIL-CLOSED). Não publicar na LAN até autorização e validação de auth pelo Owner.

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq17-lan-auth-readiness.md.
Operação 100% Read-Only. Zero alterações de código, rede ou ambiente.
