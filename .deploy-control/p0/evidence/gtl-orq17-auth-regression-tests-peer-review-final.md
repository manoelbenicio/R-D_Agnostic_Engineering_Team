# GTL-85R2 — Re-review final: suíte de regressão de auth (ORQ-17)

- Card: **ORQ-17** (`7d873133-16d5-42c6-8595-629d6fb16251`)
- Revisor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Alvo: worktree `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests`, branch
  `agent/agy-a8/orq17-auth-regression`, base `0cb8aeb`, e a evidência GTL-I04R2
  `.deploy-control/p0/evidence/gtl-orq17-auth-regression-tests-implementation.md`
  (autor Agy-P0-A8, wB:p2, 2026-07-27T12:51:51Z)
- Data UTC: 2026-07-27T12:53Z
- Modo: READ-ONLY. Nenhum edit, commit, push, deploy, restart, config ou credencial.
  Diretório privado próprio `0700`; `/tmp` não usado.

## Veredito: **PASS**

O único item bloqueante do GTL-85R foi resolvido pela via que eu havia recomendado (remoção),
o controle discriminante permaneceu, e as três correções de redação da evidência foram feitas.
Todos os gates reexecutados por mim estão verdes. Resta apenas uma nota de contagem sem
impacto (§6).

## 1. Limpeza solicitada: CONFIRMADA

O teste tautológico **não existe mais**. Busca por qualquer vestígio dele no arquivo:

```console
$ grep -n 'WithQueriesResolvesUserAndCallsNext\|_ = mw\|testMw :=' internal/middleware/auth_test.go
ABSENT (nenhuma ocorrência)
```

Nada de `mw := Auth(...)` descartado, nada de middleware falso definido dentro do teste. O
diff encolheu de 203 para **139 inserções**, coerente com a remoção de um teste de ~64 linhas.

Funções `TestLocalAuthBypass` remanescentes, todas legítimas:

| Linha | Função |
|---|---|
| `:80` | `TestLocalAuthBypassRequiresExplicitLoopbackConfiguration` (pré-existente) |
| `:511` | `TestLocalAuthBypass_ActivePath_NilQueriesEmits503` |
| `:535` | `TestLocalAuthBypass_DeactivatesOnMagicDNSOriginWithProxiedHeaders` |
| `:565` | `TestLocalAuthBypass_DisabledAlwaysReturns401` |
| `:589` | `TestLocalAuthBypass_OriginResolutionTable` (13 subtests) |
| `:623` | `TestLocalAuthBypass_DoesNotTrustXForwardedForToEnableBypass` |

## 2. Controle positivo segue discriminante: CONFIRMADO

`auth_test.go:511-533`, lido linha a linha: configura `BYPASS=true` e
`FRONTEND_ORIGIN=http://localhost:3000`, **exige** `localAuthBypassEmail() == "admin@admin.local"`
(prova que o bypass está ativo, não desligado por acidente de env), constrói o middleware de
produção com `Auth(nil, nil, nil)`, falha se o `next` for chamado, e **exige 503** —
`auth.go:112`. É o par que torna as asserções 401 dos outros testes discriminantes: bypass
ativo ⇒ 503, bypass inativo ⇒ 401. Nenhum atalho, nenhum middleware sintético.

## 3. Casos de origem e XFF preservados: CONFIRMADO

- `TestLocalAuthBypass_OriginResolutionTable` (`:589`) mantém os **13** casos, com `t.Run`
  nomeado. Os seis de habilitação exercitam o ramo `ip.IsLoopback()`/`EqualFold` de
  `auth.go:48-52`: `localhost` com e sem porta, `127.0.0.1` com e sem porta, `[::1]:3000` e
  `HTTP://LOCALHOST:3000`. Os sete de desativação: vazio, sem esquema,
  `127.0.0.1.evil.com`, LAN privada, MagicDNS FQDN, domínio público e string malformada.
- `TestLocalAuthBypass_DoesNotTrustXForwardedForToEnableBypass` (`:623-640`) mantém a asserção
  direta `localAuthBypassEmail() == ""` **antes** do 401, com `RemoteAddr` loopback,
  `X-Forwarded-For: 127.0.0.1` e `X-Forwarded-Host: localhost`.

## 4. Escopo e produção: CONFIRMADO

```console
$ git status --porcelain
 M multica-auth-work/server/internal/middleware/auth_test.go
$ git ls-files --others --exclude-standard      # vazio
$ git diff --stat
 .../internal/middleware/auth_test.go | 139 +++++++++++++++++++++
 1 file changed, 139 insertions(+)
$ git diff --name-only | grep -c 'auth\.go$'  →  0
$ git log --oneline -1  →  0cb8aeb    (sem commit)
```

Um único arquivo, só `_test.go`, zero deleções de linhas do arquivo original,
**`auth.go` de produção fora do diff**, nenhum arquivo novo, nenhum commit.

## 5. Hashes: REPRODUZEM EXATAMENTE

| Artefato | Declarado (GTL-I04R2 §1) | Calculado por mim |
|---|---|---|
| `internal/middleware/auth_test.go` | `aa84df9ebea043dd3059ee9440a4da48a1ba3bf35c75ec792a2bb4df6d0d8c94` | idêntico ✓ |
| `git diff \| sha256sum` | `ea6ff6450d60e4f5d53783157ae9d0c7f9420fefa3e8bc6346d861c2e352b2d5` | idêntico ✓ |

## 6. Correções de redação da evidência: FEITAS

| Item do GTL-85R | Estado no GTL-I04R2 |
|---|---|
| Decisão é `FRONTEND_ORIGIN`-only; `RemoteAddr`/XFF não são entradas | **FEITO.** §2.3 afirma textualmente que `localAuthBypassEmail()` é decidida estritamente por `FRONTEND_ORIGIN` e `MULTICA_LOCAL_AUTH_BYPASS`, e que `RemoteAddr`, `X-Forwarded-For` e `X-Forwarded-Host` nunca ativam ou desativam a flag |
| Crédito à cobertura pré-existente | **FEITO.** §2.4 credita `auth_test.go:80-93` (`TestLocalAuthBypassRequiresExplicitLoopbackConfiguration`) |
| Atribuição correta dos skips do pacote | **FEITO.** §3 agora atribui os 15 skips à ausência de **`REDIS_TEST_URL` e `DATABASE_URL`**, o que confere com a medição: 6 linhas de `skipping: database not reachable` (Postgres) e 5 nomes `TestRateLimit_*` (Redis) |
| Afirmação falsa sobre propagação de header | **RESOLVIDA** pela remoção do teste (§1); a §2.1 agora descreve a remoção e o motivo |
| Gates com `-race -count=1 -json`, sem `(cached)` | **FEITO** e reproduzido por mim (§7) |

Nota sem impacto: a §3 e a §4 contam "19 folhas (6 funções + 13 casos)". Contando folhas
reais, são **18** (5 funções folha + 13 subtests), porque `OriginResolutionTable` é pai, não
folha; o total de **nomes** com evento `pass` é 19. A aritmética do autor conta o pai como
folha. Nenhum efeito sobre a cobertura ou sobre o veredito.

## 7. Gates frescos reexecutados

Toolchain `/home/ec2-user/goroot/go/bin/go` (go1.26.1). `TMPDIR`, `GOTMPDIR` e `GOCACHE` em
`/home/ec2-user/kiro-orq17-r3-gate/{tmp,gocache}`, criados nesta rodada, todos `drwx------`.
`/tmp` não foi usado em nenhum passo.

```console
$ gofmt -l internal/middleware/auth_test.go        → stdout VAZIA, rc=0
$ git diff --check                                 → rc=0
$ go vet ./internal/middleware/                    → rc=0
$ go build ./internal/middleware/                  → rc=0
$ go test -race -count=1 -json -run '^TestLocalAuthBypass' ./internal/middleware
  rc=0
  names_run=19   names_pass=19   skip=0   fail=0
  real_leaves_pass=18            run_without_pass=[]
  PACKAGE_PASS=yes               false_green_markers=0
```

Todos os 19 nomes (18 folhas reais + o pai da tabela) têm evento `run` **e** `pass`; o
conjunto "executou mas não passou" é vazio; zero `skip`, zero `fail`, zero ocorrência de
`no tests to run`, `Skipping tests:` ou `database not reachable` no alvo; e há evento `pass`
de pacote.

Pacote completo, mesma configuração: `run=102 pass=87 skip=15 fail=0`, `PACKAGE_PASS=yes`, e
**zero** skips com nome contendo `LocalAuthBypass`. Os 15 skips são pré-existentes, usam
`t.Skip` (não `os.Exit`), e se dividem entre Postgres (6 mensagens `database not reachable`)
e Redis (5 nomes `TestRateLimit_*`), exatamente como a evidência passou a declarar.

## 8. Resposta item por item ao dispatch GTL-85R2

| Pedido | Resultado |
|---|---|
| Teste tautológico `WithQueries…` ausente | CONFIRMADO (§1) |
| `NilQueriesEmits503` segue discriminante | CONFIRMADO (§2) |
| 19 folhas / casos de origem / XFF preservados | CONFIRMADO (§3, §7); 19 nomes, 18 folhas reais |
| Só `auth_test.go` mudou | CONFIRMADO (§4) |
| `auth.go` de produção intocado | CONFIRMADO (§4) |
| Evidência declara decisão `FRONTEND_ORIGIN`-only | CONFIRMADO (§6) |
| Evidência credita cobertura pré-existente | CONFIRMADO (§6) |
| Evidência atribui skips a PostgreSQL e Redis | CONFIRMADO (§6), e confere com a medição |
| Hashes reproduzidos | CONFIRMADO (§5) |
| gofmt vazio, diff-check, vet, build | CONFIRMADO (§7) |
| `-race -count=1 -json`, todas as folhas run+pass, zero skip/fail/no-test, pacote PASS | CONFIRMADO (§7) |
| Temp/cache privado 0700, sem `/tmp` | CONFIRMADO (§7) |

## Check-out — ORQ-17

- Veredito final: **PASS**. Nenhuma pendência bloqueante. A suíte agora é um guarda de
  regressão real: bypass ativo produz 503 pelo middleware de produção, bypass inativo produz
  401, e a tabela de origens fixa os dois lados da decisão `FRONTEND_ORIGIN`-only, incluindo
  IPv4/IPv6 loopback e `localhost` em maiúsculas.
- Única observação remanescente, não bloqueante: aritmética de folhas na §3/§4 da evidência
  (18 folhas reais contra 19 nomes).
- Estado do trabalho: não commitado, não pushado, na branch `agent/agy-a8/orq17-auth-regression`
  sobre `0cb8aeb`. Promoção, commit, push ou merge continuam exigindo decisão escrita do owner
  (AA-001 §0.1).
- Eu não editei, commitei, pushei, deployei nem reiniciei nada; nenhuma config ou credencial
  tocada. Worktree ORQ-26 intocado e congelado.
