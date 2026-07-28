[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Peer Review READ-ONLY GTL-52 — Design de Remediação de Auth e Acesso LAN (gtl-orq17-auth-remediation-design.md)

1. VALIDAÇÃO ADVERSARIAL DOS REQUISITOS:
- Ordem Fail-Closed: Binds downstream reconfigurados para 127.0.0.1 ANTES da publicação pública no Proxy HTTPS (L143).
- Auth & Sessão: Cookie multica_session com atributos HttpOnly, Secure, SameSite=Lax e Path=/. Validação síncrona em /api/* contra a tabela user_session retornando HTTP 401 (L54).
- Reverse Proxy & ACL: Caddy HTTPS com matcher @allowed_ips (100.110.178.47, 127.0.0.1, ::1) e resposta 403 Forbidden em acessos não autorizados (L70).
- Proteções Específicas: CORS restrito por hostname, CSRF via X-Requested-With/Origin, WebSockets via upgrade transparente e Uploads via streaming sem buffering (L117).
- Binds Locais Estritos: Next.js em 127.0.0.1:13100, Go API em 127.0.0.1:18080, Postgres DB em 127.0.0.1:5432 / Unix socket (L127).
- Zero-Secret & Rollback: Zero segredos impressos (asm-exec dinâmico) e rollback de emergência via script tunnel-multica.sh preservado (L155).

2. VEREDITO FINAL:
- STATUS: PASS (APROVADO SEM RESSALVAS).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq17-auth-remediation-peer-review.md.
Operação 100% Read-Only. Zero mutações.
