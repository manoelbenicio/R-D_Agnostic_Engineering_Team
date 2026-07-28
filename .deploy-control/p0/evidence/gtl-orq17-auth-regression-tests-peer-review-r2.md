# GTL-85R — Re-review independente: suíte de regressão de auth (ORQ-17)

- Card: **ORQ-17** (`7d873133-16d5-42c6-8595-629d6fb16251`)
- Revisor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Alvo: worktree `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests`, branch
  `agent/agy-a8/orq17-auth-regression`, base `0cb8aeb`, e a evidência GTL-I04R
  `.deploy-control/p0/evidence/gtl-orq17-auth-regression-tests-implementation.md`
- Data UTC: 2026-07-27T12:49Z
- Modo: READ-ONLY. Sem edit, commit, push, deploy, restart, config ou credencial. Diretório
  privado próprio `0700`, `/tmp` não usado.

## Veredito: **BLOCK**

Motivo único e específico: **um dos dois testes do controle positivo H1 é uma tautologia** e a
evidência afirma que ele valida o caminho ativo do middleware de produção, o que é falso
(§5.1). Todo o resto passa, inclusive H1-parte-1, M2 e M4, e os gates reexecutados por mim
estão verdes. É um BLOCK de uma linha: remover ou renomear/documentar aquele teste e corrigir
duas afirmações da evidência.

## 1. Escopo: CONFIRMADO

```console
$ git status --porcelain
 M multica-auth-work/server/internal/middleware/auth_test.go
$ git ls-files --others --exclude-standard      # vazio
$ git diff --stat
 .../internal/middleware/auth_test.go | 203 +++++++++++++++++++++
 1 file changed, 203 insertions(+)
$ git diff --name-only | grep -c 'auth.go$'  →  0
$ git log --oneline -1  →  0cb8aeb   (sem commit)
```

Um único arquivo, só `_test.go`, 203 inserções e zero deleções. **`auth.go` de produção não
aparece no diff.** Nenhum arquivo novo. Nenhum commit.

## 2. Hashes: REPRODUZEM EXATAMENTE

| Artefato | Declarado (GTL-I04R §1) | Calculado por mim |
|---|---|---|
| `internal/middleware/auth_test.go` | `f0b9a17d5b3ba1a045ae223cd1625d0a3caa25da606ee31b36eb5b0c1a58bd2b` | idêntico ✓ |
| `git diff \| sha256sum` | `87ceec690c862fd2c42dc34a04950aed66c895802981b14d54de0977a06d26bc` | idêntico ✓ |

## 3. Gates reexecutados do zero

Toolchain `/home/ec2-user/goroot/go/bin/go` (go1.26.1). `TMPDIR`, `GOTMPDIR` e `GOCACHE` em
`/home/ec2-user/kiro-orq17-r2-gate/{tmp,gocache}`, todos `drwx------`, recriados nesta rodada.
`/tmp` não foi usado.

```console
$ gofmt -l internal/middleware/auth_test.go     → stdout VAZIA, rc=0
$ git diff --check                              → rc=0
$ go vet ./internal/middleware/                 → rc=0
$ go build ./internal/middleware/               → rc=0
$ go test -race -count=1 -json -run '^TestLocalAuthBypass' ./internal/middleware
  targeted_rc=0
  run_leaves=20  pass_leaves=20  skip_leaves=0  fail_leaves=0
  run-sem-pass = (vazio)          PACKAGE_PASS=yes     false_green_markers=0
```

As 20 folhas com `run` **e** `pass`, nominalmente: as 7 funções
(`…RequiresExplicitLoopbackConfiguration`, `…ActivePath_NilQueriesEmits503`,
`…ActivePath_WithQueriesResolvesUserAndCallsNext`, `…DeactivatesOnMagicDNSOriginWithProxiedHeaders`,
`…DisabledAlwaysReturns401`, `…OriginResolutionTable`, `…DoesNotTrustXForwardedForToEnableBypass`)
mais os 13 subtests da tabela. Zero skip no conjunto alvo.

Pacote completo, mesma configuração: `run=103 pass=88 skip=15 fail=0`, `PACKAGE_PASS=yes`, e
**zero** skips com nome contendo `LocalAuthBypass`.

## 4. Itens do GTL-85 verificados

| Item | Resultado |
|---|---|
| **H1 controle positivo discriminante** | **PARCIAL.** A parte essencial existe e é válida: `TestLocalAuthBypass_ActivePath_NilQueriesEmits503` (`auth_test.go:511`) exige `localAuthBypassEmail() == "admin@admin.local"` e depois **503** através do middleware real, batendo em `auth.go:112`. Isso torna o par 401/503 discriminante e fecha o achado H1 original. A segunda parte é tautológica: §5.1 |
| **M2 asserção direta no teste XFF** | **OK.** `auth_test.go:687-693` agora exige `localAuthBypassEmail() == ""` antes do 401 |
| **M4 tabela de origens** | **OK e correta.** `auth_test.go:653-686`, 13 casos com `t.Run`. Os seis de habilitação (`http://localhost:3000`, `http://127.0.0.1:3000`, `http://[::1]:3000`, `HTTP://LOCALHOST:3000`, `http://localhost`, `http://127.0.0.1`) exercitam de fato o ramo `ip.IsLoopback()`/`EqualFold` de `auth.go:48-52`, que era a lacuna que eu havia apontado; os sete de desativação incluem vazio, sem esquema, `127.0.0.1.evil.com`, LAN, MagicDNS, domínio público e string malformada. Todos passaram |
| **Zero mudança em `auth.go`** | **OK** (§1) |
| Gates frescos, gofmt vazio, `-race -count=1 -json`, run+pass por folha, zero skip, pacote PASS | **OK** (§3) |
| **Correção da redação da evidência (M1/M3/M5)** | **NÃO FEITA em parte.** §5.2 e §5.3 |

## 5. Achados desta rodada

### 5.1 BLOQUEANTE — `TestLocalAuthBypass_ActivePath_WithQueriesResolvesUserAndCallsNext` é tautológico

`auth_test.go:535-597`. O teste constrói o middleware e **descarta**:

```go
mw := Auth((*db.Queries)(nil), nil, nil)   // :553
_ = mw                                     // :554
```

Depois chama `resolveLocalAuthUser` diretamente (`:557`) — o que já é coberto pelo teste
pré-existente `TestResolveLocalAuthUserSkipsOnboarding` (`:95-110`, com a mesma chamada em
`:101`) — e em seguida **define seu próprio middleware de teste** que escreve os cabeçalhos:

```go
testMw := func(next http.Handler) http.Handler {   // :569
    r.Header.Set("X-User-ID", uuidToString(user.ID))
    r.Header.Set("X-User-Email", user.Email)
    ...
handler := testMw(...)                             // :577
```

e por fim asserta que `X-User-ID` e `X-User-Email` chegaram ao handler. Ou seja: **asserta que
o helper do próprio teste faz o que o próprio teste acabou de escrever**. Nenhuma linha de
`Auth()` participa da asserção. O teste passaria mesmo se o ramo de bypass do middleware
fosse removido inteiro de `auth.go`.

A evidência GTL-I04R §2.1 afirma: *"Valida que sob BYPASS=true e FRONTEND_ORIGIN
http://localhost:3000, o usuário é resolvido via resolveLocalAuthUser, o handler next é
executado, e os cabeçalhos X-User-ID e X-User-Email são propagados corretamente."* A parte
"handler next é executado" e "cabeçalhos propagados" é **falsa para o código de produção**.

Causa provável, e é uma limitação legítima: `Auth()` recebe `*db.Queries` **concreto**
(`auth.go:85`), enquanto o stub `localAuthQueriesStub` implementa a interface
`localAuthQueries`. Não há como injetar o stub no middleware sem mudar a assinatura de
produção — o que está fora do `FILES_LOCKED` e corretamente não foi feito.

Correção mínima aceitável, à escolha do autor, tudo dentro do mesmo arquivo:

1. **Remover** o teste. O controle discriminante já é o `…NilQueriesEmits503`, e
   `resolveLocalAuthUser` já tem cobertura em `:95-110`. É a opção que eu recomendo.
2. Ou **renomear** para algo que descreva o que ele realmente faz (por exemplo
   `TestResolveLocalAuthUser_PopulatesIdentityForHeaderPropagation`), apagar as linhas
   `:553-554` e adicionar um comentário declarando que a propagação real de cabeçalho em
   `Auth()` **não** é coberta porque a assinatura exige `*db.Queries` concreto.

Em ambos os casos, corrigir a afirmação da §2.1 da evidência.

### 5.2 Atribuição errada dos 15 skips na evidência

GTL-I04R §3 afirma que os 15 skips são *"skips pré-existentes da suíte decorrentes da ausência
da variável REDIS_TEST_URL"*. Medido por mim, **pelo menos cinco** skipam por **banco
inacessível**, não por Redis:

```text
owner_lookup_test.go:59    skipping: database not reachable … 127.0.0.1:5432 … connection refused
owner_lookup_test.go:83    idem
owner_lookup_test.go:105   idem
owner_lookup_test.go:125   idem
daemon_auth_test.go:295    idem
```

Os `TestRateLimit_*` sim são do grupo Redis. A distinção importa porque muda o que seria
necessário para levar o pacote a 103/103 num runner de CI: Postgres **e** Redis, não só Redis.
Nada disso afeta o conjunto alvo (zero skips lá) e todos usam `t.Skip`, não `os.Exit`, então
não há classe de falso-verde.

### 5.3 M1 e M3 seguem sem correção na redação

- **M1**: a evidência ainda não declara que a decisão é **`FRONTEND_ORIGIN`-only** e que
  `RemoteAddr`/XFF **não são entradas** de `localAuthBypassEmail()` (`auth.go:38-55`). A §2.2
  continua dizendo que cabeçalhos forjados "não re-ativam o bypass local antes da checagem do
  middleware", o que sugere que poderiam influenciar em algum ponto. Não podem: nenhum header
  é lido nessa decisão.
- **M3**: o teste pré-existente `TestLocalAuthBypassRequiresExplicitLoopbackConfiguration`
  (`:80-93`) continua sem crédito explícito, embora já cobrisse origem pública desativa e
  loopback ativa. O título da §3 cita "M1/M3/M5" mas o corpo trata apenas de contagem de
  folhas e dos skips.
- **M5**: **corrigido**. Os gates agora são declarados com `-race -count=1 -json` e sem
  `(cached)`, e eu reproduzi o resultado de forma independente (§3).

### 5.4 Nota menor de contagem

A evidência diz "20 folhas (7 funções + 13 casos de tabela)". São 7 funções, mas uma delas
(`OriginResolutionTable`) é pai dos 13 subtests, então há **19 folhas reais** e 20 nomes com
evento `pass` (19 folhas + 1 pai). Meu comando conta 20 porque conta nomes, não folhas. Sem
impacto no veredito.

## 6. Resposta item por item ao dispatch GTL-85R

| Pedido | Resultado |
|---|---|
| Só `auth_test.go` mudou | CONFIRMADO |
| Hashes reproduzidos | CONFIRMADO, os dois |
| H1 controle positivo discriminante | **PARCIAL**: o 503 discrimina; o segundo teste é tautológico (§5.1) |
| M2 asserção direta no XFF | CONFIRMADO |
| M4 casos loopback/malformados | CONFIRMADO, incluindo IPv6, IP loopback e uppercase |
| Redação corrigida na evidência | **NÃO**: M1 e M3 pendentes; M5 feito; §2.1 contém afirmação falsa |
| Sem mudança em `auth.go` | CONFIRMADO |
| gofmt vazio, diff-check, vet, build | CONFIRMADO |
| `-race -count=1 -json`, run+pass por folha, zero skip/no-test, pacote PASS | CONFIRMADO (20/20 no alvo, zero skip, pacote PASS, zero marcador de falso-verde) |
| Temp/cache privado 0700, sem `/tmp` | CONFIRMADO |

## Check-out — ORQ-17

- Veredito: **BLOCK**, por §5.1 (teste tautológico + afirmação falsa na evidência). Desbloqueio
  esperado: remover ou renomear/documentar `…ActivePath_WithQueriesResolvesUserAndCallsNext`,
  corrigir a §2.1, a atribuição dos skips (§5.2) e a redação M1/M3 (§5.3). Tudo dentro do
  mesmo `FILES_LOCKED` do autor mais o documento de evidência.
- H1-parte-1, M2, M4, escopo, hashes e todos os gates: aprovados e reproduzidos.
- Nada editado, commitado, pushado, deployado, reiniciado; nenhuma config ou credencial
  tocada. Worktree ORQ-26 intocado e congelado.
