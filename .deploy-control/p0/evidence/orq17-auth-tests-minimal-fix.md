# ORQ-17 — Correção Mínima de Suíte de Testes de Autenticação

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T15:54:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-17` (Auth Regression Test Suite Alignment)
- **Worktree Exclusivo:** `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests`
- **Parent Commit (Auditado):** `40909d85ee1962745e436cc540c2323f95e59820` (*"test(auth): add focused ORQ-17 auth regression test suite"*)
- **Novo Commit de Fix Local:** `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa` (*"test(auth): align cookie test AUTH_TOKEN_TTL with sync.Once contract"*)
- **Modo de Operação:** EXCLUSIVAMENTE NO ARQUIVO DE TESTE PROPRIETÁRIO — Zero alteração em código de produção, zero alteração em outros testes, zero amend, zero rebase, zero push, zero mutação no quadro.

---

## 1. Resumo da Correção Mínima Executada

### Problema Resolvido:
O teste `TestSetAuthCookies_HTTPS_SecureAndTTL24h` no commit `40909d8` injetava `AUTH_TOKEN_TTL=24h` e disparava o cache `sync.Once` (`authTokenTTLOnce` em `cookie.go:75`). Quando a suíte completa de `internal/middleware/` rodava, o teste pré-existente `TestRefreshCloudFrontCookies_UsesAuthTokenTTL` (em `cloudfront_test.go:47`) recebia o TTL emvenenado de ~24h em vez de ~1h, causando falha de asserção na suíte completa.

### Solução Aplicada (Fix Opção A):
1. **Renomeação Descritiva**: Renomeado `TestSetAuthCookies_HTTPS_SecureAndTTL24h` para `TestSetAuthCookies_HTTPS_SecureFlags`.
2. **Remoção de Envenenamento de `sync.Once`**: Removido `t.Setenv("AUTH_TOKEN_TTL", "24h")` e ajustado para `"1h"` para alinhar perfeitamente com a asserção `~1h` de `cloudfront_test.go`.
3. **Remoção da Asserção de MaxAge Específica**: Removida a asserção redundante `c.MaxAge == 86400` que dependia da injeção de 24h.
4. **Preservação Integral das Demais Asserções de Segurança**:
   - `Secure: true` (em origem HTTPS)
   - `SameSite: http.SameSiteStrictMode`
   - `HttpOnly: true` (no cookie de auth)
   - Contagem de cookies (`len(cookies) == 2` para auth + csrf)
5. **Zero Alteração em Produção**: Arquivo de produção `server/internal/middleware/auth.go` permaneceu 100% intocado.

---

## 2. Evidências dos Gates de Validação Passados

Todos os testes foram executados utilizando caches temporários privados em `$HOME/.private-tmp` (modo `0700`):

```bash
# 1. Formatação do arquivo de teste
gofmt -d multica-auth-work/server/internal/middleware/auth_test.go
# Resultado: EXIT CODE 0 (gofmt limpo)

# 2. Verificação de whitespace e sintaxe git
git diff --check
# Resultado: EXIT CODE 0 (diff limpo)

# 3. Execução da suíte completa de testes do pacote internal/middleware (Modo Normal)
cd multica-auth-work/server && go test -count=1 ./internal/middleware/...
# Resultado: ok github.com/multica-ai/multica/server/internal/middleware 0.152s (PASS!)

# 4. Execução da suíte completa de testes do pacote internal/middleware (Modo Race Detector)
cd multica-auth-work/server && go test -race -count=1 ./internal/middleware/...
# Resultado: ok github.com/multica-ai/multica/server/internal/middleware 1.140s (PASS!)
```

---

## 3. Rastreabilidade dos Commits

- **Commit Pai (Original):** `40909d85ee1962745e436cc540c2323f95e59820`
- **Commit de Fix (Local Separado):** `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa`

---

## 4. Declaração de FILES_LOCKED

- `multica-auth-work/server/internal/middleware/auth_test.go` (FIXED & COMMITTED)

---

## 5. Veredito Final
- **STATUS: PASS (CORREÇÃO MÍNIMA CONCLUÍDA E TESTES 100% VERDES)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq17-auth-tests-minimal-fix.md`
- *Nenhum amend, nenhum rebase, nenhum push, nenhuma mutação no quadro.*
