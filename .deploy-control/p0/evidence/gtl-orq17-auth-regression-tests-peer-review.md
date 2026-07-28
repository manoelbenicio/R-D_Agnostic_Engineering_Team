# GTL-85 — Peer review adversarial: suíte de regressão de auth ORQ-17

- Card: **ORQ-17**
- Revisor: Kiro-Opus5 (leitura e recomendação; sem poder de decisão, AA-001 §0.0)
- Alvo: worktree `/home/ec2-user/workspace/worktrees/gtl-orq17-auth-tests`, branch
  `agent/agy-a8/orq17-auth-regression`, base `0cb8aeb`, e a evidência
  `.deploy-control/p0/evidence/gtl-orq17-auth-regression-tests-implementation.md`
  (autor Agy-P0-A8, wB:p2)
- Data UTC: 2026-07-27T12:31Z
- Modo: READ-ONLY. Nada editado, commitado, pushado, deployado. O worktree ORQ-26 não foi
  tocado e permanece congelado. Única escrita: este arquivo.

## Veredito: **PASS**, com duas correções exigidas antes de tratar a suíte como *guarda* de regressão (H1, M2)

O que a evidência afirma é verdadeiro e reproduzível. O que falta é força: três dos cinco
testes provam ausência de bypass por uma asserção não discriminante (401), sem controle
positivo que demonstre que a suíte detectaria um bypass vivo.

## 1. Escopo e ausência de produção: CONFIRMADO

```console
$ git status --porcelain
 M multica-auth-work/server/internal/middleware/auth_test.go
$ git diff --stat
 .../server/internal/middleware/auth_test.go | 106 +++++++++++++++++++++
 1 file changed, 106 insertions(+)
$ git ls-files --others --exclude-standard
            # vazio: nenhum arquivo novo fora do índice
$ git diff --check
            # exit 0
```

Um único arquivo, apenas `_test.go`, 106 inserções e zero deleções. `auth.go` e qualquer
outro arquivo de produção intactos. HEAD ainda em `0cb8aeb`: sem commit.

## 2. Hashes: REPRODUZEM EXATAMENTE

| Artefato | Valor declarado | Valor que eu calculei |
|---|---|---|
| `internal/middleware/auth_test.go` | `63836ab863d02e66bd01c94c8871154374b2efae63f200725cbe25ff6c001ee9` | idêntico ✓ |
| `git diff \| sha256sum` | `800bedb6e2d86c56263033c67ca89e6f923b5fc48d1c6ec144d7e96942888754` | idêntico ✓ (e igual com `--binary`) |

## 3. Gates: RE-EXECUTADOS DE FORMA INDEPENDENTE, com padrão mais estrito que o declarado

Toolchain `/home/ec2-user/goroot/go/bin/go` (go1.26.1). Diretório privado próprio,
`/home/ec2-user/kiro-orq17-review-gate` (0700, `TMPDIR`/`GOTMPDIR`/`GOCACHE`), fora dos dois
worktrees, para não misturar cache entre agentes.

```console
$ gofmt -l internal/middleware/auth_test.go        # stdout vazia, exit 0
$ go vet ./internal/middleware/                    # exit 0
$ go build ./internal/middleware/                  # exit 0
$ go test -race -count=1 -json -run '^TestLocalAuthBypass' ./internal/middleware
test_exit=0
RUN/PASS (6/6, zero skip):
  TestLocalAuthBypassRequiresExplicitLoopbackConfiguration
  TestLocalAuthBypass_AllowsLocalhostDevOnly
  TestLocalAuthBypass_DeactivatesOnInvalidOrNonLoopbackOrigin
  TestLocalAuthBypass_DeactivatesOnMagicDNSOriginWithProxiedHeaders
  TestLocalAuthBypass_DisabledAlwaysReturns401
  TestLocalAuthBypass_DoesNotTrustXForwardedForToEnableBypass
PACKAGE_PASS=yes   skips=0   false_green_markers=0
```

Os seis testes têm eventos `run` **e** `pass` no JSON, zero `skip`, `pass` de pacote, e zero
ocorrência de `Skipping tests:` / `no tests to run` / `database not reachable`.

## 4. Harness: SEM classe de falso-verde deste tipo

- `grep -rn 'func TestMain' internal/middleware/` → **nenhum**. O pacote `middleware` não tem
  o `TestMain` com `os.Exit(0)` que causa o falso-verde do pacote `handler`
  (`internal/handler/handler_test.go:38-53`). Não há dependência de Postgres: os testes usam
  `Auth(nil, nil, nil)` e stubs.
- Isolamento de env: todos os testes novos usam `t.Setenv`, que restaura o valor no fim do
  teste e **entra em panic** se o teste for paralelo — logo é impossível um teste destes
  rodar em paralelo por descuido. Nenhum dos cinco chama `t.Parallel()`.
- Paralelismo do pacote: os únicos `t.Parallel()` estão em `request_logger_test.go:147,206`.
  Testes top-level paralelos são pausados e retomados só depois da fase sequencial, então não
  observam o env mutado pelos testes com `t.Setenv`. Confirmado empiricamente: execução com
  `-race` limpa, sem panic e sem corrida.
- Sem corrida em `handlerCalled`: `ServeHTTP` é chamado de forma síncrona no mesmo goroutine.

## 5. Achados adversariais

### H1 (exigido) — Falta controle positivo: a asserção 401 não é discriminante

Os três testes de nível middleware afirmam apenas `w.Code == 401`. Com `Auth(nil, nil, nil)`,
se o bypass **estivesse ativo**, o código entraria no ramo do bypass e responderia **503**
(`"local auth bypass unavailable"`, porque `queries == nil`), nunca 401. Ou seja, o 401 só
prova ausência de bypass **por causa desse detalhe de implementação**. Não existe nenhum
teste que passe pelo middleware com o bypass ativo, portanto **nada prova que a suíte
detectaria um bypass vivo**.

Consequência concreta: se alguém alterar o ramo de bypass para responder 401 quando
`queries == nil`, ou mover a decisão para dentro do handler, os três testes continuam verdes
provando nada. Isso é exatamente a categoria de falso-verde que o fleet passou a rejeitar.

Correção mínima: um teste de controle com `BYPASS=true` + `FRONTEND_ORIGIN` loopback
passando pelo middleware, exigindo **503** com `queries == nil`; e, idealmente, um segundo com
`localAuthQueriesStub` exigindo que o next handler seja chamado com `X-User-ID` preenchido.
Isso converte a suíte de "asserta uma ausência" para "asserta um sinal discriminante".

### M2 (exigido) — Teste 5 herda H1 sem mitigação

`TestLocalAuthBypass_DoesNotTrustXForwardedForToEnableBypass` **não** chama
`localAuthBypassEmail()`; asserta só o 401. Os testes 1, 2 e 4 têm a asserção direta do gate,
o que os protege parcialmente. Adicionar a asserção direta ao teste 5, por simetria.

### M1 (correção de redação da evidência) — `RemoteAddr` e XFF são decorativos

`localAuthBypassEmail()` (`internal/middleware/auth.go:38-55`) lê **somente**
`MULTICA_LOCAL_AUTH_BYPASS` e `FRONTEND_ORIGIN`. Nunca inspeciona `r.RemoteAddr` nem qualquer
header. Portanto "desativa o bypass mesmo com RemoteAddr loopback e XFF externo" é uma
garantia **estrutural**, não um comportamento que o teste poderia falsificar. Guardar o teste
é legítimo (pin contra alguém introduzir confiança em header no futuro), mas a redação
"simula requisição recebida via Reverse Proxy Caddy" sugere ao leitor que existe um controle
baseado em `RemoteAddr`. A evidência deveria dizer explicitamente: a decisão é
**FRONTEND_ORIGIN-only**; `RemoteAddr` e XFF não são entradas.

### M3 — Cobertura sobreposta e crédito ausente ao teste pré-existente

`TestLocalAuthBypassRequiresExplicitLoopbackConfiguration` (`auth_test.go:80-93`) **já**
existia e já cobria os dois eixos centrais: origem pública desativa, origem loopback ativa
com o e-mail configurado. O novo `TestLocalAuthBypass_AllowsLocalhostDevOnly` é subconjunto
estrito dele, e parte do teste 4 repete o caso de origem pública. Não é dano, mas a evidência
apresenta como nova uma cobertura que em parte já estava lá. O nome
`AllowsLocalhostDevOnly` também promete o "only" e asserta apenas o caso de habilitação.

### M4 (recomendado) — Lacunas de shape de origem, incluindo as de risco real

O laço de origens inválidas não usa `t.Run`, então cada falha reporta pelo texto do
`t.Errorf` em vez de um subteste nomeado. Mais relevante: faltam os shapes que de fato
podem regredir, porque exercitam o **outro** ramo aceito em `auth.go:49-51`
(`ip.IsLoopback()`):

| Shape ausente | Comportamento esperado |
|---|---|
| `http://127.0.0.1:3000` | **habilita** (IP loopback v4) |
| `https://[::1]:3000` | **habilita** (IP loopback v6) |
| `FRONTEND_ORIGIN=""` | desativa (scheme/host vazios) |
| `localhost:3000` sem scheme | desativa (`parsed.Scheme == ""`) |
| `http://LOCALHOST:3000` | **habilita** (`EqualFold`) |
| `http://127.0.0.1.evil.com:3000` | desativa (host não é IP nem `localhost`) |

Os três casos de habilitação são os mais valiosos: um regressor que endureça demais o gate
tranca o owner fora do ambiente de dev, e nenhum teste atual pegaria isso.

### M5 (correção de evidência, não de código) — Os gates declarados são mais fracos que os afirmados

Na evidência, GATE 2 e GATE 3 mostram `ok ... (cached)`, sem `-count=1` e sem `-race`. Com
`-v`, o Go **reproduz** as linhas `=== RUN` / `--- PASS` a partir do cache, então aquela saída
não prova execução naquela sessão. Não é falso-verde — eu reexecutei e passa —, mas como
registro fica abaixo do padrão que o fleet adotou (GTL-51 §7: `-race -count=1 -json`, com
prova de `run`+`pass` por nome). Recomendo regravar os gates com as saídas da §3 acima.

## 6. Resposta item por item ao dispatch

| Item pedido | Resultado |
|---|---|
| Único arquivo `auth_test.go`, zero produção | **CONFIRMADO** (§1) |
| FQDN non-loopback desativa bypass mesmo com RemoteAddr loopback e XFF externo | **CONFIRMADO** pelo teste 1, que combina asserção direta de `localAuthBypassEmail() == ""` com 401 no middleware. Ressalva M1: `RemoteAddr`/XFF não são entradas da decisão |
| `bypass=false` | **CONFIRMADO** (teste 2, com asserção direta) |
| local dev habilita | **CONFIRMADO** (teste 3 + pré-existente `:80-93`) |
| malformed / non-loopback desativa | **CONFIRMADO** para 4 shapes; ver M4 para os que faltam |
| XFF não habilita | **CONFIRMADO na prática**, mas fraco: ver M2 e M1 |
| Isolamento de env / paralelismo sem race | **CONFIRMADO** (§4, `-race` limpo) |
| Harness sem skip / falso-verde | **CONFIRMADO** para a classe `TestMain`/DB (§4). **PARCIAL** para a classe "asserção não discriminante": ver H1 |
| Gates e hashes reproduzíveis | **CONFIRMADO** (§2, §3); registro da evidência a corrigir por M5 |

## 7. Limites desta revisão

- Não executei nada além de gofmt, vet, build e os testes do pacote `middleware`; não toquei
  env, rede, banco, board nem o worktree ORQ-26.
- Não avaliei o design de remediação de auth em si; isso está no meu review anterior
  (`gtl-orq17-auth-peer-review.md`, veredito BLOCK), que segue válido e independente deste.

## Check-out — ORQ-17

- Entregue: veredito PASS com 1 achado exigido de nível alto (H1), 1 exigido de nível médio
  (M2), 3 recomendados/correções de evidência (M1, M3, M4, M5), reprodução independente dos
  dois hashes e re-execução dos gates com `-race -count=1 -json` provando 6/6 run+pass e zero
  skip.
- Pendências para o autor (Agy-P0-A8), todas dentro do mesmo `FILES_LOCKED`: H1 e M2 antes de
  a suíte contar como guarda; M4 recomendado; M1 e M5 são correções no documento de evidência.
- Nada editado, commitado, pushado ou deployado por mim.
