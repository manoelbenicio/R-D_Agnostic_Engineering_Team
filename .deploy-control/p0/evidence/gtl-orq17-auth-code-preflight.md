# GTL-72 — Preflight READ-ONLY de Código Backend-Auth (ORQ-17 Design V2)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T12:12:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Alvo:** Mapeamento de rotas, bypasses e proposta de patch mínimo test-first para endurecimento de autenticação no Go Backend
- **Modo:** SOMENTE LEITURA / PREFLIGHT DE CÓDIGO — Zero alterações em código, rede, banco ou variáveis de ambiente.

---

## 1. Mapeamento Completo de Rotas Públicas, Protegidas e Bypasses

### 1.1 Mecanismo de Bypass Local (`MULTICA_LOCAL_AUTH_BYPASS`)
- **Localização no Código**: `server/internal/middleware/auth.go:38-55`
- **Condições Exigidas para Ativação**:
  1. `MULTICA_LOCAL_AUTH_BYPASS` deve ser igual a `"true"` (case-insensitive).
  2. `FRONTEND_ORIGIN` deve ser não-nula e parseável como URL.
  3. O hostname de `FRONTEND_ORIGIN` deve ser `"localhost"` ou um IP de loopback (`127.0.0.1`, `::1`).
  4. `MULTICA_LOCAL_AUTH_EMAIL` deve conter um e-mail de operador válido no banco (`users.email`).
- **Comportamento em Execução**: Quando ativo, o middleware `Auth` atribui automaticamente o usuário estático a TODAS as requisições que passam por `middleware.Auth`, ignorando tokens JWT/cookies e retornando o header `X-User-ID`.
- **Risco Identificado**: Se `FRONTEND_ORIGIN` for configurado como `http://localhost:3000` em um servidor exposto na LAN na porta 13100/18080, requisições externas diretas contornam a autenticação sem apresentar token.

### 1.2 Mapeamento de Rotas do Servidor (`server/cmd/server/router.go`)

| Categoria | Rota | Método | Middleware | Descrição |
|---|---|---|---|---|
| **Health / Probes** | `/health`, `/readyz`, `/healthz` | GET | Nenhum | Probes de liveness/readiness públicas |
| **Realtime Metrics** | `/health/realtime` | GET | Token / Loopback | Protegida por `REALTIME_METRICS_TOKEN` ou loopback |
| **WebSockets** | `/ws` | GET | Custom (realtime) | Handshake WS com auth por query/cookie interna |
| **Local Uploads** | `/uploads/*` | GET | Storage local | Servimento estático (quando em LocalStorage) |
| **Auth Pública** | `/auth/login`, `/auth/google` | POST | RateLimit | Login público com limite por IP |
| **Auth Pública** | `/auth/logout` | POST | Nenhum | Limpa cookie `multica_auth` |
| **Public API** | `/api/config` | GET | Nenhum | Configuração pública do cliente |
| **Public API** | `/api/contact-sales` | POST | RateLimit | Formulário de vendas público |
| **Webhooks** | `/api/webhooks/autopilots/{token}` | POST | Token na URL | Autenticação por token no path |
| **Webhooks** | `/api/webhooks/github`, `/api/webhooks/stripe` | POST | HMAC / Signature | Autenticado por assinatura de payload |
| **Daemon API** | `/api/daemon/*` | ALL | `DaemonAuth` | Requer token de daemon (`mdt_`), task (`mat_`), cloud (`mcn_`) ou PAT (`mul_`) |
| **Protegidas User** | `/api/me`, `/api/workspaces/*`, etc. | ALL | `Auth` | Requer JWT, cookie com CSRF ou PAT (`mul_`) |

---

## 2. Análise de CORS, CSRF, WebSockets e File Uploads

1. **CORS (`router.go:424-440`)**:
   - `allowedOrigins()` resolve `CORS_ALLOWED_ORIGINS` -> `FRONTEND_ORIGIN` -> `localhost:3000,5173,5174`.
   - `AllowCredentials: true` e `AllowedHeaders` explícitos (`X-Workspace-ID`, `X-CSRF-Token`, `Authorization`, etc.).
2. **CSRF (`auth.go:145-149`)**:
   - `auth.ValidateCSRF(r)` é acionado APENAS quando a autenticação vem de cookie (`fromCookie == true`).
   - Rejeita requisições mutativas (`POST`, `PUT`, `PATCH`, `DELETE`) sem header `X-CSRF-Token` válido com **HTTP 403 Forbidden**.
3. **WebSockets Realtime (`router.go:468`)**:
   - Rota `/ws` aberta na raiz do roteador, com checagem de origem via `realtime.SetAllowedOrigins(origins)` e `realtime.SetTrustedProxies(signupConfig.TrustedProxies)`.
4. **File Uploads (`router.go:567`)**:
   - Rota `/api/upload-file` montada dentro do grupo protegido por `middleware.Auth`. Requer autenticação de usuário válida.

---

## 3. Proposta de Patch Mínimo Test-First (Design V2 de Auth)

### 3.1 Endurecimento do Loopback Bypass (Trava de Segurança Código-Level)
Reforçar `localAuthBypassEmail()` em `server/internal/middleware/auth.go`:
```go
// Garantir que a requisição de fato veio de um IP loopback quando o bypass local está ativo:
if localEmail != "" {
    remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
    ip := net.ParseIP(remoteIP)
    if ip != nil && !ip.IsLoopback() {
        slog.Warn("auth: rejected non-loopback request under local bypass", "remote_ip", remoteIP)
        http.Error(w, `{"error":"local auth bypass restricted to loopback callers"}`, http.StatusForbidden)
        return
    }
}
```

### 3.2 Feature Flag / Config de Compatibilidade
- **Variável**: `MULTICA_AUTH_HARDENING_V2=true` (default: `false` para preservar ambientes legados de dev).
- Quando ativada, bloqueia qualquer tentativa de usar bypass local em interfaces não-loopback.

### 3.3 Plano de Rollback
- Reverter `MULTICA_AUTH_HARDENING_V2=false` via variável de ambiente systemd restaura o comportamento anterior sem necessidade de rebuild.

---

## 4. Declaração de FILES_LOCKED Reais

Para evitar sobreposição e garantir o princípio de Escritor Único:

### Arquivos Bloqueados (NÃO TOCAR / SINGLE WRITER):
- `multica-auth-work/server/internal/handler/file.go` (LOCKED — ORQ-26 Fix)
- `multica-auth-work/server/internal/handler/file_test.go` (LOCKED — ORQ-26 Tests)
- `multica-auth-work/server/internal/middleware/auth.go` (LOCKED — Auth Middleware)
- `multica-auth-work/server/internal/middleware/auth_test.go` (LOCKED — Auth Middleware Tests)

---

## 5. Suíte de Testes Negativos 401/403 e Regressões Recomendada

1. `TestAuth_MissingAuthorizationReturns401`: Requisição sem header/cookie em `/api/me` deve retornar HTTP 401.
2. `TestAuth_InvalidJWTTokenReturns401`: Token JWT expirado ou com assinatura inválida retorna HTTP 401.
3. `TestAuth_CSRFFailureOnCookieReturns403`: Requisição via cookie sem `X-CSRF-Token` em `POST /api/workspaces` retorna HTTP 403.
4. `TestLocalAuthBypass_RejectsRemoteIP`: Com `MULTICA_LOCAL_AUTH_BYPASS=true`, requisição com `RemoteAddr="192.168.1.50:12345"` é rejeitada com HTTP 403.
5. `TestLocalAuthBypass_AllowsLoopbackIP`: Requisição com `RemoteAddr="127.0.0.1:12345"` é aceita com o usuário configurado.

---

## 6. Veredito Final
- **STATUS: PASS (PREFLIGHT DE CÓDIGO CONCLUÍDO COM SUCESSO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/gtl-orq17-auth-code-preflight.md`
- *Operação 100% Read-Only. Nenhuma alteração de código, banco de dados ou rede.*
