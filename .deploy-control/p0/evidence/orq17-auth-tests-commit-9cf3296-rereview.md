# ORQ-17 — Re-review focado do commit de correção `9cf3296` (READ-ONLY)

- Card: **ORQ-17** · worktree `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests`
- Commit: `9cf3296fcf2c8d5ff72f439df16492e6cd76c1fa` — *"test(auth): align cookie test AUTH_TOKEN_TTL with
  sync.Once contract"*, autor `kiro-lead`, 2026-07-27 15:53:53 +0000
- Sobre: `40909d8` (revisado antes → BLOCK), que está sobre `0cb8aeb`
- Diff: **1 arquivo, +2/-5**, apenas `multica-auth-work/server/internal/middleware/auth_test.go`
- Revisor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:56Z
- Modo: **READ-ONLY**. Nenhum edit, amend, rebase, commit, push, board ou deploy.

## VEREDITO: **PASS** — o BLOCK anterior está resolvido; 1 observação residual sem bloqueio

## 1. O diff é exatamente o mínimo, nada além

```diff
-func TestSetAuthCookies_HTTPS_SecureAndTTL24h(t *testing.T) {
+func TestSetAuthCookies_HTTPS_SecureFlags(t *testing.T) {
 	t.Setenv("FRONTEND_ORIGIN", "https://orq1.domain.ts.net:13100")
-	t.Setenv("AUTH_TOKEN_TTL", "24h")
+	t.Setenv("AUTH_TOKEN_TTL", "1h")
...
-		if c.MaxAge != 86400 {
-			t.Errorf("cookie %q MaxAge = %d, want 86400 (24h)", c.Name, c.MaxAge)
-		}
```

Três mudanças, todas dentro do escopo pedido:

1. **asserção de TTL removida** (`MaxAge != 86400`) — era a origem do envenenamento;
2. **env alinhado** de `24h` para `1h`, o mesmo valor que `cloudfront_test.go:47` define;
3. **renome** de `..._SecureAndTTL24h` para `..._SecureFlags` — resolve também o achado secundário 3 do
   review anterior, em que o nome prometia "24h" como contrato quando o default real é 30 dias
   (`internal/auth/cookie.go:23`).

Nenhum outro arquivo, nenhuma linha de produção, nenhum teste removido.

## 2. O envenenamento do `sync.Once` foi de fato eliminado

Levantamento de todos os pontos que tocam a variável no pacote:

```text
internal/middleware/auth_test.go:652        t.Setenv("AUTH_TOKEN_TTL", "1h")
internal/middleware/cloudfront_test.go:47   t.Setenv("AUTH_TOKEN_TTL", "1h")
internal/middleware/cloudfront_test.go:68   espera ~1h
```

Os dois únicos escritores agora concordam em `1h`, então **qualquer ordem de execução** produz o mesmo
valor cacheado por `authTokenTTLOnce` (`internal/auth/cookie.go:74-88`). A falha anterior —
`"expires in 23h59m59s; expected ~1h"` nas três cookies CloudFront — não é mais reproduzível.

## 3. Execução: pacote completo, plain e `-race`

Go 1.26.1, caches privados fora de `/tmp` (tmpfs em 99%), todos `0700`:

```text
/home/ec2-user/.private-tmp/orq17-review          700
/home/ec2-user/.private-tmp/orq17-review/gocache  700
/home/ec2-user/.private-tmp/orq17-review/gotmp    700

go test ./internal/middleware/ -count=1            -> ok  0.120s
go test ./internal/middleware/ -race -count=1      -> ok  1.111s
go test ./internal/middleware/ -race -count=2      -> ok  1.718s   (2 rodadas, sem flake)
git show 9cf3296 --check                           -> exit 0
git status --porcelain                             -> vazio
```

Comparação direta com o estado anterior, que é o que fecha o ciclo:

| árvore | plain | -race |
|---|---|---|
| base `0cb8aeb` (sem os dois commits) | ok 0.112s | — |
| `40909d8` (só testes) | **FAIL** | **FAIL** |
| `9cf3296` (com a correção) | **ok 0.120s** | **ok 1.111s / 1.718s** |

## 4. A cobertura útil permaneceu intacta

`grep` dos testes ORQ-17 no arquivo após a correção:

| linha | teste | estado |
|---|---|---|
| 511 | `TestLocalAuthBypass_ActivePath_NilQueriesEmits503` | mantido |
| 535 | `TestLocalAuthBypass_DeactivatesOnMagicDNSOriginWithProxiedHeaders` | mantido |
| 565 | `TestLocalAuthBypass_DisabledAlwaysReturns401` | mantido |
| 589 | `TestLocalAuthBypass_OriginResolutionTable` (13 subcasos) | mantido |
| 623 | `TestLocalAuthBypass_DoesNotTrustXForwardedForToEnableBypass` | mantido |
| 650 | `TestSetAuthCookies_HTTPS_SecureFlags` | mantido, sem asserção de TTL |
| 677 | `TestValidateJWTConfiguration_ProductionRequirements` | mantido |
| 80 | `TestLocalAuthBypassRequiresExplicitLoopbackConfiguration` (pré-existente) | intacto |

Asserções que sobrevivem no teste de cookies (linhas 660-673): `len(cookies) != 2` (auth + csrf),
`c.Secure`, `c.SameSite == http.SameSiteStrictMode`, e `HttpOnly` no cookie de auth. Ou seja, o valor de
segurança do teste — as flags — foi **preservado**; só saiu a asserção que dependia de estado global.
Nada de CSRF em requisição é afirmado, e a contagem de 2 cookies continua legítima
(`internal/auth/cookie.go:167-179` emite mesmo o `CSRFCookieName`).

## 5. Observação residual (não bloqueia)

`t.Setenv("AUTH_TOKEN_TTL","1h")` permaneceu no teste de cookies **embora ele já não asserte nada sobre
TTL**. Funciona hoje só porque o valor coincide com o que `cloudfront_test.go` espera; é um acoplamento
implícito entre dois arquivos via variável global. Duas formas de endurecer, ambas opcionais:

- remover a linha 652 por completo — o teste não precisa dela; ou
- deixar um comentário de uma linha explicando que o valor **precisa** coincidir com
  `cloudfront_test.go:47` por causa do `sync.Once`, para que um futuro editor não o mude sozinho.

A causa estrutural (TTL global cacheado em `sync.Once`, sem hook de reset para teste) continua no
produto e merece card próprio; não é problema deste commit de testes.

## 6. Não-alegações

- Não editei, amendei, rebasei, commitei nem pushei; não toquei board nem deploy. HEAD segue `9cf3296`
  e `git status` está vazio.
- Rodei apenas `./internal/middleware/`, não a suíte completa do servidor.
- Não verifiquei o pacote `internal/auth` nesta rodada (não foi tocado pelo diff).
- Não provei por mutação o poder de detecção dos testes de bypass; mantenho a avaliação por inspeção do
  review anterior (`auth.go:43-52`).
- Nenhum valor de segredo real foi lido ou impresso.
