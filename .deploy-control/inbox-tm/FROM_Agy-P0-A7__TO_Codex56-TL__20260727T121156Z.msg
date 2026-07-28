[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Preflight READ-ONLY de Código Backend-Auth GTL-72 (ORQ-17) — gtl-orq17-auth-code-preflight.md

1. MAPEAMENTO DE ROTAS E BYPASSES (server/internal/middleware/auth.go & server/cmd/server/router.go):
- Bypass Local: localAuthBypassEmail (auth.go:38-55) é ativado via MULTICA_LOCAL_AUTH_BYPASS=true quando FRONTEND_ORIGIN é localhost ou loopback IP. Substitui autenticação de usuário pelo e-mail estático sem login/JWT.
- Rotas Públicas: /health, /readyz, /healthz, /health/realtime, /ws (WS Handshake), /uploads/*, /auth/login, /auth/google, /auth/logout, /api/config, /api/contact-sales, webhooks (autopilot token, github HMAC, stripe signature).
- Rotas Protegidas: /api/me, /api/cli-token, /api/upload-file, /api/attachments/{id}/download, /api/workspaces/* (protegidas por middleware.Auth).

2. ANÁLISE DE CORS/CSRF/WS/UPLOADS:
- CORS: allowedOrigins() em router.go:42 resolve FRONTEND_ORIGIN ou CORS_ALLOWED_ORIGINS com AllowCredentials: true.
- CSRF: auth.ValidateCSRF(r) exige X-CSRF-Token em requisições mutativas via cookie.
- WebSockets: /ws com origin check de proxies confiáveis; /api/daemon/ws protegido por DaemonAuth.
- Uploads: /api/upload-file protegido por middleware.Auth.

3. PROPOSTA DE PATCH MÍNIMO TEST-FIRST:
- Reforço em localAuthBypassEmail para rejeitar RemoteAddr que não seja IsLoopback(), resguardado pela flag MULTICA_AUTH_HARDENING_V2=true.
- Rollback: MULTICA_AUTH_HARDENING_V2=false restaura comportamento sem rebuild.

4. FILES_LOCKED REAIS DECLARADOS:
- server/internal/handler/file.go & file_test.go (ORQ-26 Single Writer - LOCKED)
- server/internal/middleware/auth.go & auth_test.go (Auth Middleware - LOCKED)

5. VEREDITO FINAL:
- STATUS: PASS (PREFLIGHT DE CÓDIGO CONCLUÍDO).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq17-auth-code-preflight.md.
Operação 100% Read-Only. Zero alterações.
