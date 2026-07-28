# ORQ-33 - Peer review independente do runbook V2 de rotacao do `JWT_SECRET`

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T15:22Z
- independencia: **nao sou autor da V1** (Agy-P0-A8), **nao sou o GTL-87**, **nao sou o autor da V2**
  (Opus48#B). Primeira vez que reviso este artefato.
- auditado: `.deploy-control/p0/evidence/orq33-jwt-secret-rotation-runbook-v2.md`
- card citado no artefato: `ORQ-33` / `dfeabbdc-33e1-4ab8-9460-27b43df227db` (nao revalidei por GET;
  ver secao 12)
- modo: READ-ONLY. **Nenhum segredo lido**, zero `GetSecretValue`/`BatchGetSecretValue`, **zero
  `asm-exec` executado**, zero restart, zero mutacao de AWS, codigo, banco, container ou unit.

---

## VEREDITO: **BLOCK**

A V2 e um salto de qualidade real sobre a V1: o alvo esta certo, os cinco consumidores estao certos,
o `sync.Once` esta certo, o gate de fila esta certo, e a honestidade sobre ser evento disruptivo esta
certa. **Confirmo 8 dos 10 itens** que me pediram para verificar.

O BLOCK e por **um defeito de seguranca critico** no Passo 2 + Passo 3, que nem a V1 nem o GTL-87
tinham como pegar porque a V2 os introduziu juntos, mais **um defeito operacional** que faria o
comando do Passo 3 agir na arvore errada.

---

## 1. ✅ CONFIRMADO - alvo `JWT_SECRET` e chave unica com `sync.Once`

`internal/auth/jwt.go:25-40`, literal, confere com o citado:
```go
25 var (
26 	jwtSecret     []byte
27 	jwtSecretOnce sync.Once
28 )
30 func JWTSecret() []byte {
31 	jwtSecretOnce.Do(func() {
32 		secret := os.Getenv("JWT_SECRET")
33 		if secret == "" { secret = defaultJWTSecret }
36 		jwtSecret = []byte(secret)
37 	})
39 	return jwtSecret
40 }
```
Uma unica `jwtSecret`, resolvida **uma vez por processo**. Sem `kid`, sem keyset, sem chave
secundaria. A conclusao da V2 - **recreate, nao restart/reload** - esta correta e bem fundamentada.

`ValidateJWTConfiguration` (`jwt.go:46-58`) confere: recusa placeholder conhecido
(`knownInsecureJWTSecrets`, com `"change-me-in-production"` em `:20`) e recusa
`len < minimumProductionJWTSecretBytes`, **exceto** quando `APP_ENV` e `dev`/`development`/`test`
(`:47-50`). O erro e `ErrInsecureJWTConfiguration` (`:23`), cujo texto cita "at least 32 bytes".
A exigencia de >= 32 bytes da V2 esta correta.

## 2. ✅ CONFIRMADO - exatamente **cinco** consumidores, com os papeis certos

`grep -rn "JWTSecret()" --include=*.go .` sem testes, resultado integral:
```
internal/auth/jwt.go:30            (definicao)
internal/handler/auth.go:213       token.SignedString(auth.JWTSecret())      <- ASSINA
internal/handler/auth.go:226       token.SignedString(auth.JWTSecret())      <- ASSINA
internal/middleware/auth.go:295    return auth.JWTSecret(), nil              <- VERIFICA (API HTTP)
internal/middleware/daemon_auth.go:234  return auth.JWTSecret(), nil         <- VERIFICA (daemon)
internal/realtime/hub.go:691       return auth.JWTSecret(), nil              <- VERIFICA (WebSocket)
```
Cinco call sites, dois de assinatura e **tres** superficies de verificacao. A tabela da secao 2 da V2
esta **exata**, linha por linha. E a afirmacao de que a V1 subestimava em 3 consumidores procede.

## 3. ✅ CONFIRMADO - fila com os 4 estados ativos

O predicado da V2 (`queued`, `dispatched`, `running`, `waiting_local_directory`) e o correto, derivado
do CHECK de `109_agent_task_waiting_local_directory.up.sql:15`. Endosso tambem a recomendacao
transversal da V2: este predicado ja errou em **tres** runbooks distintos que eu revisei hoje
(GTL-32, GTL-43 e o ORQ-32), e deveria existir **uma** fonte reutilizavel derivada da constraint.

## 4. ✅ CONFIRMADO - rota autenticada barata **existe**, e a V2 pode fechar essa nao-afirmacao

A V2 declara em `10` que nao verificou qual rota usar e chama `/api/me` de "ilustrativo". Verifiquei:
```
cmd/server/router.go:552   r.Get("/api/me", h.GetMe)
cmd/server/router.go:553   r.Patch("/api/me", h.UpdateMe)
```
`GET /api/me` e real, esta no grupo autenticado e e leve. Serve exatamente ao proposito de V2/V3 da
secao 6. **Nao usar `/api/config`** (`router.go:492`): ele esta **fora** do grupo autenticado, logo
responderia `200` com token invalido e produziria falso sucesso no teste de `old_token = 401`.
Correcao simples: fixar `GET /api/me` e remover a nao-afirmacao.

## 5. ✅ CONFIRMADO - `old=401` / `new=200` e o gate certo, pelo motivo certo

A justificativa da V2 - "reiniciar o processo nao prova que a chave nova carregou, porque
`JWTSecret()` resolve lazily na primeira chamada (`jwt.go:31`)" - esta correta e e o insight mais
valioso do documento. O criterio 2 (`old_token = 200` significa falso sucesso) e exatamente a
armadilha que um runbook comum deixaria passar.

Endosso tambem a checagem de log: `internal/middleware/auth.go:298` e literalmente
`slog.Warn("auth: invalid token", "path", r.URL.Path, "error", err)` - loga `path` e `error`, **nunca**
o token. A contagem por `grep -c` da secao 8 e secret-safe.

## 6. ✅ CONFIRMADO - segredo nunca em argv, e a autocritica de R4 procede

A V2 acerta ao proibir `--secret-string '{"token":"..."}'` e ao oferecer
`--generate-secret-string` (geracao server-side, valor nunca no cliente) ou
`fileb:///dev/stdin`. G2 e G4 tambem sao secret-safe: `test -n` nao imprime, e
`printf %s | wc -c` imprime **comprimento**, nao valor. Nenhum passo do runbook imprime segredo.

## 7. ✅ CONFIRMADO - rollback e disruptivo nos dois sentidos

Correto e honesto: chave unica implica que ir e voltar sao ambos eventos disruptivos, `reload` nao
reverte pelo mesmo `sync.Once`, e o segredo antigo nao pode ser deletado antes da validacao. A
ressalva herdada sobre nao reaplicar branch antiga nem copia bruta de credencial (AGY/`cli.log`) esta
mantida corretamente.

---

## 8. 🔴 BLOQUEADOR 1 - `asm-exec` **nao resolve placeholder dentro de arquivo**, e o Passo 2+3 conta com isso

Este e o motivo do BLOCK, e o efeito e pior que uma falha: e um **sucesso aparente com segredo
publico**.

O Passo 2 manda escrever na configuracao do backend:
```
JWT_SECRET={{resolve:secretsmanager:prod/multica/jwt-secret:SecretString:jwt_secret}}
```
e o Passo 3 executa `asm-exec -- docker compose -f ... up -d --force-recreate --no-deps backend`.

Escopo real do `asm-exec`, `.agents/skills/aws-secrets-manager/SKILL.md:53-55`, literal:
> *"`asm-exec` is a wrapper that resolves `{{resolve:...}}` references in **command arguments and
> environment variables**, then `exec`s the target command."*

Ou seja: **argumentos e variaveis de ambiente do processo filho**. O `backend-env.override.yml` e um
**arquivo lido pelo docker compose**, nao um argumento nem uma variavel de ambiente do `asm-exec`.
Portanto o placeholder **nao e resolvido**: o compose le a string literal
`{{resolve:secretsmanager:prod/multica/jwt-secret:SecretString:jwt_secret}}` e a injeta como valor de
`JWT_SECRET` no container.

Por que isso passa silenciosamente pelas guardas existentes:
1. a string literal tem **74 bytes**, logo passa o `len < 32` de `jwt.go:55`;
2. ela **nao** esta em `knownInsecureJWTSecrets` (`jwt.go:19-21`), logo passa o teste de placeholder;
3. `ValidateJWTConfiguration` retorna `nil`, o backend **sobe normalmente**;
4. `health=200`, `readyz=200`, `old_token=401` (porque a chave mudou de fato) e `new_token=200`
   (porque assina e verifica com a mesma string). **Todos os cinco criterios de aceite da secao 6
   passam.**

Resultado: a rotacao seria declarada bem-sucedida com o JWT do produto assinado por uma string
**previsivel e publica**, presente em documentacao e em qualquer log de compose. E estritamente pior
que o estado atual.

Correcoes aceitaveis, escolha do executor:
- **(a)** manter o `{{resolve}}` em **variavel de ambiente do proprio `asm-exec`** e o compose apenas
  **repassar**: no override, `JWT_SECRET: ${JWT_SECRET:?...}` (interpolacao do compose), e o comando
  ser `asm-exec -- env JWT_SECRET='{{resolve:...}}' docker compose ... up -d ...`. Aqui o placeholder
  esta em argumento/env do `asm-exec`, dentro do escopo documentado;
- **(b)** usar `--env-file` gerado dentro do processo filho, `0600`, apagado em seguida - com a
  ressalva de que o valor toca disco;
- **(c)** provar, com teste em ambiente descartavel, que o `asm-exec` desta instalacao resolve
  tambem em arquivo - **eu nao afirmo que nao resolve alem do documentado**, afirmo que a **skill
  documenta apenas argumentos e ambiente**, e um runbook nao pode depender de comportamento nao
  documentado.

Gate novo obrigatorio, seja qual for a opcao: **provar que o valor efetivo no container nao e um
placeholder**, sem imprimir o valor. Por exemplo comparar comprimento e hash com o esperado, ou
verificar que `docker inspect` do container **nao** contem a substring `{{resolve:`:
```bash
docker inspect multica-dev-transition-backend-1 \
  --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -c '{{resolve:'
# criterio: 0.  Qualquer valor > 0 significa placeholder literal injetado -> ABORTAR
```
Esse comando **nao** imprime o segredo, apenas conta ocorrencias do marcador.

## 9. 🔴 BLOQUEADOR 2 - os `-f` do Passo 3 sao relativos, e a arvore correta **nao e a que parece**

O Passo 3 usa nomes relativos: `-f docker-compose.selfhost.yml -f docker-compose.selfhost.build.yml`.
Os `config_files` reais do container em execucao, lidos por `docker inspect` no ORQ1, sao **absolutos**:
```
/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml
/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml
/home/ec2-user/.config/multica-transition/images.yml
/home/ec2-user/.config/multica-transition/backend-env.override.yml
project/service = multica-dev-transition/backend
```
Duas consequencias:
1. **caminho relativo depende do cwd.** Executado do diretorio errado, o compose falha ou - pior -
   usa um `docker-compose.selfhost.yml` de **outra** arvore, recriando o container com configuracao
   diferente.
2. **a arvore e outra.** O compose aponta para `/home/ec2-user/R-D_Agnostic_Engineering_Team/...`.
   Verifiquei no ORQ1 que **`/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` nao existe lá** -
   esse caminho e do ORQ2, onde nos trabalhamos. Ou seja: quem executar no ORQ1 tem de usar a arvore
   sob `$HOME`, e um operador acostumado ao caminho do ORQ2 erraria.

Correcao: usar **os quatro caminhos absolutos** exatamente como o label reporta, e adicionar
`-p multica-dev-transition` explicito. E incluir um gate previo que **releia** o label, porque a
lista pode mudar:
```bash
docker inspect multica-dev-transition-backend-1 \
  --format '{{index .Config.Labels "com.docker.compose.project.config_files"}}'
# criterio: a lista usada no comando e IDENTICA a esta saida, na mesma ordem
```
A V2 ja declara em `10` que montou a lista a partir do label e que a ordem precisa ser confirmada -
credito por isso -, mas o comando escrito no Passo 3 **contradiz a propria nao-afirmacao** ao usar
relativos. Um executor apressado copia o comando, nao a ressalva.

---

## 10. Achados menores, nao bloqueantes

- **M1.** G5 e a pergunta certa, mas nao tem criterio de sucesso executavel. Sugestao secret-safe, so
  por **nome de chave**: `grep -l 'JWT_SECRET' <os quatro arquivos>` e, no override, confirmar
  presenca da chave sem imprimir a linha (`grep -c`). Isso responde G5 sem abrir valor.
- **M2.** O Passo 4 (re-emparelhar daemons) nao tem procedimento nem criterio de aceite; a secao 6
  cobra "heartbeat atualizado em `agent_runtime`", mas nao diz **como** re-emparelhar. Como o daemon
  do ORQ2 e user unit, isso provavelmente envolve reiniciar a unit - o que e mutacao no **outro
  host** e merece autorizacao propria. Hoje o passo esta subespecificado.
- **M3.** O gatilho de rollback ("taxa de 401/403 acima do esperado") e qualitativo, como a propria V2
  admite. Com `old_token=401` sendo o comportamento **desejado**, uma taxa alta de 401 e esperada por
  definicao apos a rotacao; o gatilho precisa ser reescrito em termos de **usuarios que nao conseguem
  relogar**, nao de contagem de 401.
- **M4.** A secao 6 guarda tokens em `/run/secrets-tmp/*.jwt`. Nao verifiquei se esse diretorio
  existe no ORQ1 nem seu modo; o runbook deve criar com `0700` e o arquivo `0600`, e o `rm` final
  precisa ser incondicional (`trap`), senao um `curl` que falhe deixa token em disco.
- **M5.** `defaultJWTSecret` existe (`jwt.go:34`) como fallback quando `JWT_SECRET` esta vazio. Se o
  Passo 2 apagar a variavel por engano em vez de trocar, com `APP_ENV` nao-production o backend sobe
  com o default de desenvolvimento. Vale um gate explicito de que `JWT_SECRET` esta **presente e
  nao-vazio** apos a edicao - o compose ja tem `:?` para isso (`docker-compose.selfhost.yml:58`), mas
  so na variavel de ambiente do compose, e o override pode contornar.

## 11. Correcoes exigidas para virar PASS

1. **C1 (bloqueador 1)** eliminar a dependencia de resolucao de placeholder **dentro de arquivo**;
   adotar (a), (b) ou (c) da secao 8, e adicionar o gate `grep -c '{{resolve:'` no `docker inspect`
   com criterio **0**.
2. **C2 (bloqueador 2)** trocar os `-f` relativos pelos **quatro caminhos absolutos** do label, com
   `-p multica-dev-transition`, e adicionar o gate de releitura do label antes de executar.
3. **C3** fixar `GET /api/me` (`router.go:552`) como rota de validacao e remover a nao-afirmacao;
   proibir explicitamente `/api/config` (`router.go:492`, fora do grupo autenticado).
4. **C4** dar a G5 um criterio executavel por nome de chave (M1).
5. **C5** especificar o Passo 4, com procedimento de re-emparelhamento, host afetado e autorizacao
   propria (M2).
6. **C6** reescrever o gatilho de rollback em termos de falha de **novo login**, nao de contagem de
   401 (M3).
7. **C7** `trap`/`0600` para os tokens temporarios (M4) e gate de `JWT_SECRET` nao-vazio (M5).

Nenhuma dessas exige refundar o runbook. O desenho central - evento disruptivo unico, recreate,
old=401/new=200, fila de 4 estados - esta correto e eu o endosso.

## 12. Nao-afirmacoes desta revisao

- Nao revalidei o UUID do ORQ-33 por `GET`; usei o citado no artefato. Sob a regra de unfreeze, quem
  for executar deve confirmar o par UUID+numero.
- Nao inspecionei `~/.config/multica-transition/backend-env.override.yml`, pelo mesmo motivo que a V2
  nao inspecionou: pode conter segredo. Portanto **nao sei** se `JWT_SECRET` e definido lá ou no
  compose principal - e por isso G5 continua sendo gate, e nao conclusao.
- Nao consultei o Secrets Manager: nao sei se `prod/multica/jwt-secret` existe.
- Nao executei `asm-exec`, nao li nem resolvi segredo, nao gerei token, nao chamei
  `GetSecretValue`/`BatchGetSecretValue`, nao reiniciei nem recriei container, e nao mutei AWS, codigo,
  banco, unit ou board. Nao alterei assignee nem postei comentario. Nenhum card criado.
- As leituras que fiz no ORQ1 foram `docker inspect` (labels e nomes de arquivo, **sem** `.Config.Env`)
  e `ls -d` de diretorio.
