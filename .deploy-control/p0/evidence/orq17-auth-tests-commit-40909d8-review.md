# ORQ-17 — Code review independente do commit local `40909d8` (READ-ONLY)

- Card: **ORQ-17** · worktree `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests`
- Commit revisado: `40909d85ee1962745e436cc540c2323f95e59820` — *"test(auth): add focused ORQ-17 auth
  regression test suite"*, autor `kiro-lead`, 2026-07-27 15:40:04 +0000
- Diff: **1 arquivo, +190/-0** — `multica-auth-work/server/internal/middleware/auth_test.go`
- Revisor: Codex56#A (`w7:p3`) · UTC 2026-07-27T15:49Z
- Modo: **READ-ONLY**. Nenhum amend, rebase, edit, commit, push, mutação de board ou deploy. Só leitura,
  execução de teste em caches privados `0700` e `git diff --check`.

## VEREDITO: **BLOCK** — o commit quebra um teste pré-existente do próprio pacote

Reproduzido, com o mesmo Go (1.26.1) e caches privados:

```text
worktree (com 40909d8):   go test ./internal/middleware/ -count=1        -> FAIL
worktree (com 40909d8):   go test ./internal/middleware/ -race -count=1  -> FAIL
base (sem 40909d8):       go test ./internal/middleware/ -count=1        -> ok  0.112s
```

Teste que passa a falhar: **`TestRefreshCloudFrontCookies_UsesAuthTokenTTL`** (pré-existente,
`cloudfront_test.go:47,68`).

### Causa raiz, provada por execução

`internal/auth/cookie.go:74-88` — `AuthTokenTTL()` é **cacheado em `sync.Once`**
(`authTokenTTLOnce`, `authTokenTTLCached`, default `30 * 24 * time.Hour` em `:23`). O primeiro teste do
processo que chamar a função **fixa o valor para todo o resto do binário de teste**.

O novo `TestSetAuthCookies_HTTPS_SecureAndTTL24h` faz `t.Setenv("AUTH_TOKEN_TTL","24h")` e dispara o
`Once` primeiro (log observado: `INFO auth token TTL configured seconds=86400`). O teste antigo, que
define `AUTH_TOKEN_TTL=1h` (`cloudfront_test.go:47`) e espera ~1h, recebe o valor envenenado:

```text
cloudfront_test.go:68: cookie "CloudFront-Policy" expires in 23h59m59.84s; expected ~1h (AUTH_TOKEN_TTL),
                       got what looks like 30-day hardcode
(idem CloudFront-Signature e CloudFront-Key-Pair-Id)  -> --- FAIL
```

Prova de que é acoplamento de ordem e não defeito do teste antigo: rodando **isolado**,
`go test -run TestRefreshCloudFrontCookies_UsesAuthTokenTTL` → `ok`. E rodando os dois juntos, o novo
passa e o antigo falha.

**Fix exato (escolher um):**
- **A (mínimo):** remover a asserção de `MaxAge`/TTL do novo teste e o `t.Setenv("AUTH_TOKEN_TTL")`,
  mantendo `Secure`, `SameSite`, `HttpOnly` e a contagem de cookies — nada disso depende do `Once`;
- **B:** cobrir TTL num teste **do pacote `internal/auth`** que já é o dono do `Once`, coordenado com
  os testes existentes de `cookie_test.go`;
- **C (mais correto, mas é mudança de produção, outro card):** expor um reset/injeção de TTL para teste
  e eliminar o `sync.Once` global — não fazer dentro deste commit de testes.

## 1. Qualidade comportamental dos testes (o que passou no meu crivo)

| teste | é comportamental? | cobre o quê | avaliação |
|---|---|---|---|
| `TestLocalAuthBypass_ActivePath_NilQueriesEmits503` | sim | bypass ativo + `queries == nil` → **503**, `next` nunca chamado | **BOM**: casa com `internal/middleware/auth.go:112` (`"local auth bypass unavailable"` → `StatusServiceUnavailable`); é fail-closed real, não restatement |
| `TestLocalAuthBypass_DeactivatesOnMagicDNSOriginWithProxiedHeaders` | sim | origem MagicDNS não-loopback → bypass off → **401** mesmo com `X-Forwarded-For` | **BOM**: é exatamente a raiz do incidente (config de produção copiada virando instância anônima), conforme o comentário `auth.go:34-37` |
| `TestLocalAuthBypass_DisabledAlwaysReturns401` | sim | `MULTICA_LOCAL_AUTH_BYPASS=false` → 401 mesmo com origem loopback | **BOM** |
| `TestLocalAuthBypass_OriginResolutionTable` (13 casos) | sim | loopback/localhost/IPv6/uppercase/sem porta habilitam; vazio, sem scheme, `127.0.0.1.evil.com`, LAN privada, MagicDNS, domínio público e string malformada **não** | **MUITO BOM**: `127.0.0.1.evil.com` e `192.168.1.50` são os casos que pegariam um `strings.HasPrefix` ingênuo; a implementação usa `url.Parse` + `net.ParseIP().IsLoopback()` (`auth.go:43-52`) e os casos batem |
| `TestLocalAuthBypass_DoesNotTrustXForwardedForToEnableBypass` | sim | XFF/`X-Forwarded-Host` **não** reabilitam bypass | **BOM** — é o item "trusted proxy" do brief, coberto pelo ângulo correto: header de proxy não é fonte de confiança para bypass |
| `TestSetAuthCookies_HTTPS_SecureAndTTL24h` | parcial | `Secure`, `SameSite=Strict`, `HttpOnly` no cookie de auth, 2 cookies, `MaxAge=86400` | **DEFEITUOSO** pela asserção de TTL (ver BLOCK). As demais asserções são válidas e casam com `cookie.go:149-179` |
| `TestValidateJWTConfiguration_ProductionRequirements` | sim | `default_secret_key` rejeitado, `<32` bytes rejeitado, 43 bytes aceito, com `APP_ENV=production` | **BOM**: valida política, não implementação |

**Nomes de rota reais:** todos usam `/api/me`, que existe de fato (medido antes: `GET /api/me` → 200 no
backend vivo). Nenhum endpoint inventado.

**Credenciais:** nenhum segredo real. Os valores são literais óbvios de teste
(`admin@admin.local`, `valid-jwt-token-string`, `deployment-owned-secret-0123456789-abcdefgh`). O teste
de JWT usa string de 43 caracteres só para satisfazer o mínimo de 32 — aceitável.

**CSRF:** o brief pedia para não afirmar comportamento de CSRF inexistente. Verifiquei: `CSRFCookieName`
**existe** e `SetAuthCookies` emite mesmo 2 cookies (`cookie.go:167-179`, `HttpOnly: false` no CSRF).
Portanto a asserção "2 cookies (auth + csrf)" é **legítima**, e o teste **não** afirma validação de CSRF
em requisição — o que seria o erro. Aprovado neste ponto.

## 2. "Falha antes do fix?" — resposta honesta

Não se aplica como escrito: `40909d8` **não contém fix**, é 100% teste (+190/-0), e a proteção de
loopback já está no código base (`auth.go:38-55`, com comentário explicando a intenção). Logo os testes
são **regressão/caracterização**, não prova de correção de um bug aberto. Verifiquei o valor deles por
inspeção: se alguém removesse a checagem `IsLoopback` de `auth.go:50`, os casos MagicDNS/LAN/`evil.com`
da tabela falhariam — ou seja, **têm poder de detecção**. Não executei essa mutação (seria edição de
código, proibido aqui).

## 3. Verificações de higiene

```text
git diff --check                      -> exit 0 (worktree limpo, sem whitespace error)
git show 40909d8 --check              -> exit 0
git status --porcelain                -> vazio
go test focado (-count=1 -v)          -> PASS nos 7 testes novos (13 subcasos da tabela inclusos)
go test pacote completo               -> FAIL (TestRefreshCloudFrontCookies_UsesAuthTokenTTL)
go test -race pacote completo         -> FAIL (mesma causa; nenhum data race reportado)
caches privados                       -> /home/ec2-user/.private-tmp/orq17-review{,/gocache,/gotmp} todos 0700
```

Observação: usei caches fora de `/tmp` porque o `tmpfs` do ORQ2 está em 99%; sem isso o resultado seria
falha de infraestrutura, não de teste.

## 4. Achados secundários (não bloqueiam)

1. O novo teste de cookies duplica cobertura já existente em `internal/auth/cookie_test.go:62,116`
   (`TestSetAuthCookies_HTTPSelfHost`, `TestSetAuthCookies_HTTPSProduction`). Vale consolidar em vez de
   manter asserção equivalente em dois pacotes.
2. `TestLocalAuthBypass_ActivePath_NilQueriesEmits503` passa `Auth(nil, nil, nil)`; o teste depende de
   ordem de parâmetros posicional. Um construtor nomeado deixaria a intenção explícita — melhoria, não
   defeito.
3. O nome `TestSetAuthCookies_HTTPS_SecureAndTTL24h` promete "24h" como contrato do produto, mas o
   default real é **30 dias** (`cookie.go:23`); 24h é só o valor injetado. Renomear evita leitura errada.
4. `t.Setenv` já força `-p 1` implícito para o pacote (testes não podem ser paralelos), então o problema
   do `Once` não é corrida e sim ordem — coerente com `-race` não reportar nada.

## 5. Não-alegações

- Não editei, amendei, rebasei, commitei ou pushei nada; não toquei board nem deploy.
- Não executei a suíte completa do servidor, apenas `./internal/middleware/` (focado, completo e `-race`).
- Não provei por mutação que os testes falham sem a proteção de loopback: isso exigiria editar código.
- Não avaliei os testes pré-existentes do arquivo (linhas 1-507), fora do escopo do diff.
- Nenhum valor de segredo real foi lido ou impresso.
