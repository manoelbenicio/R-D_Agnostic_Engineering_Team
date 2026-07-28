# Runbook V2 - rotacao de `JWT_SECRET` (ORQ-33) - EVENTO DISRUPTIVO DE CHAVE UNICA

- **Autor da V2:** Opus48#B (ORQ2, pane w6:p2)
- **Cartao:** `ORQ-33` · UUID `dfeabbdc-33e1-4ab8-9460-27b43df227db`
- **Substitui:** `orq33-rev-token-rotation-runbook.md` (V1, Agy-P0-A8) e o veredito PASS proprio dele
- **Base factual:** `gtl-orq33-rev-token-rotation-peer-review.md` (GTL-87, BLOCK) e
  `orq33-rev-token-independent-review.md` (minha revisao independente, BLOCK ampliado)
- **Data UTC:** 2026-07-27T15:15Z
- **Modo:** READ-ONLY / DESENHO. **Nenhum segredo lido**, zero `GetSecretValue`, zero `asm-exec`
  executado, zero restart, zero mutacao de AWS, codigo, banco, container ou unit.
- **Escritor/executor:** Codex56-TL (GENERAL-TECH-LEAD). Autoridade final: **owner humano**.

---

## 0. O QUE MUDOU DA V1 PARA A V2, E POR QUE

A V1 desenhava a rotacao de um artefato chamado `rev-token`, com janela de sobreposicao
primary/secondary e drenagem por TTL. Medido no codigo, **nada disso existe**:

| premissa da V1 | realidade medida | acao na V2 |
|---|---|---|
| segredo `prod/multica/rev-token/{primary,secondary}` | `grep -rniE 'rev_token\|revToken\|REV_TOKEN' --include='*.go'` = **zero**; o compose define `JWT_SECRET` (`docker-compose.selfhost.yml:58`) e `APP_ENV` (`:84`), **nao** `REV_TOKEN` | alvo real passa a ser **`JWT_SECRET`** |
| dupla chave com validacao simultanea | `jwt.go:25-40` tem **uma** `jwtSecret`; keyfunc em `middleware/auth.go:291-296` devolve uma chave e nao le `kid`; assinatura HS256 simetrica (`handler/auth.go:206`, `:218`) | secao 3 da V1 **removida**; V2 e evento disruptivo unico |
| drenagem de 15 min por TTL de cache | `AuthCacheTTL = 10 * time.Minute` (`internal/auth/pat_cache.go:19`) e `MembershipCacheTTL = 5 * time.Minute` (`internal/auth/membership_cache.go:19`) governam PAT e membership, **nao** validade de JWT | Passo 4 de drenagem **removido** |
| `systemctl reload multica-backend` | nao existe essa unit; o backend e container Docker no ORQ1 | recreate de container (secao 5) |
| 3 consumidores | **5** call sites de `JWTSecret()` (secao 2) | lista completa |
| Gate de fila com 2 status | `migrations/109:13-15` define 7 status, 3 terminais -> **4 ativos** | Gate 1 corrigido (secao 4) |
| `--secret-string '{"token":"..."}'` em argv | valor de segredo em linha de comando e visivel em `ps` e em historico | proibido (secao 1, R4) |

**Declaracao central da V2, que a V1 nao fazia:** rotacionar `JWT_SECRET` no codigo atual **nao e
rotacao sem downtime**. E um **evento disruptivo unico** que invalida, no mesmo instante, todos os
tokens em **tres** superficies de verificacao. Deve ser tratado como janela de manutencao anunciada.

---

## 1. REGRAS DE SEGREDO (mantidas da V1, com uma correcao)

Herdadas da V1 e reafirmadas, porque estavam corretas:
- **R1.** NUNCA chamar `GetSecretValue` ou `BatchGetSecretValue` via CLI, SDK, MCP ou script que
  imprima a resposta.
- **R2.** NUNCA ler o Secrets Manager Agent local em `localhost:2773`.
- **R3.** NUNCA imprimir, logar, commitar ou colar valor de segredo em evidencia, chat ou parametro.
- **R4 (CORRIGIDA na V2).** NUNCA passar valor de segredo em **argumento de linha de comando**.
  A V1 violava a propria R3 no Passo 1, com
  `aws secretsmanager create-secret --secret-string '{"token":"..."}'`: argv e visivel em `ps` para
  qualquer processo do host e vai para historico de shell. Formas aceitas:
  - geracao **server-side**, sem o valor jamais existir no cliente:
    ```bash
    aws secretsmanager create-secret \
      --name "prod/multica/jwt-secret" \
      --description "Backend JWT signing secret (HS256)" \
      --generate-secret-string 'PasswordLength=48,ExcludePunctuation=true,RequireEachIncludedType=false'
    ```
  - ou, se o valor precisar vir de fora, `--secret-string fileb:///dev/stdin` alimentado por pipe,
    com o arquivo temporario em `0600` e removido em seguida.
- **R5.** Consumo apenas por referencia dinamica dentro de processo filho:
  `{{resolve:secretsmanager:prod/multica/jwt-secret:SecretString:jwt_secret}}` via `asm-exec`.
- **R6.** O valor **nunca** entra em contexto de agente. Nenhum passo deste runbook imprime o segredo.

> Nota de escopo: a V2 **nao autoriza** criar o segredo. O comando de R4 esta aqui como **forma
> correta**, nao como passo executavel. Criar segredo e mutacao de AWS e exige autorizacao do owner.

---

## 2. ALVO REAL E OS CINCO CONSUMIDORES

Alvo unico: variavel de ambiente **`JWT_SECRET`** do backend.

Definicao, `server/internal/auth/jwt.go:25-40` (literal):
```go
var (
	jwtSecret     []byte
	jwtSecretOnce sync.Once
)

func JWTSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = defaultJWTSecret
		}
		jwtSecret = []byte(secret)
	})

	return jwtSecret
}
```

Guarda de configuracao ja existente, `jwt.go:46-58`: `ValidateJWTConfiguration` recusa placeholder
conhecido e segredo com menos de **32 bytes**, exceto quando `APP_ENV` e `dev`, `development` ou
`test`. O compose reforca: `JWT_SECRET: ${JWT_SECRET:?JWT_SECRET must be set to a generated value of
at least 32 bytes}` (`docker-compose.selfhost.yml:58`). Portanto o segredo novo **deve** ter >= 32
bytes, senao o backend recusa subir com `APP_ENV=production`.

### Os cinco call sites de `JWTSecret()` (todos afetados pela troca)

| # | arquivo:linha | papel | efeito da rotacao |
|---|---|---|---|
| 1 | `internal/handler/auth.go:213` | **assina** token de sessao (`token.SignedString`) | passa a assinar com a chave nova |
| 2 | `internal/handler/auth.go:226` | **assina** segundo tipo de token | idem |
| 3 | `internal/middleware/auth.go:295` | verifica - **API HTTP** | todo JWT antigo -> `401` |
| 4 | `internal/middleware/daemon_auth.go:234` | verifica - **auth de daemon** | daemons perdem auth; exige re-emparelhamento |
| 5 | `internal/realtime/hub.go:691` | verifica - **WebSocket / realtime** | conexoes autenticadas caem |

Assinatura e HS256 (`handler/auth.go:206`, `:218`), isto e **HMAC simetrico**: a mesma chave assina e
verifica. Nao existe `kid`, nao existe chave secundaria, nao existe conjunto de chaves.

**Consequencia que precisa constar em comunicado de manutencao:** usuarios sao deslogados, daemons
precisam re-emparelhar e sessoes WebSocket reconectam. A V1 descrevia isso como zero-downtime; e falso.

### Superficies NAO afetadas (para nao superestimar o raio)
- **PAT** (`mul_...`) e **daemon token** (`mdt_...`) sao gerados por `crypto/rand`
  (`jwt.go:61-68` e seguinte) e validados contra o banco, **nao** assinados com `JWT_SECRET`. Nao sao
  invalidados pela rotacao.
- `internal/auth/cloud_pat.go` faz revogacao por **TTL de PAT de Cloud** - mecanismo distinto,
  fora deste runbook (`cloud_pat.go:43`: "the TTL itself IS the revocation").
- Os caches `AuthCacheTTL` (10 min) e `MembershipCacheTTL` (5 min) guardam PAT e membership; nao
  prolongam nem atrasam a invalidacao de JWT.

---

## 3. TOPOLOGIA MEDIDA

```
ORQ1 (100.118.244.61) - containers Docker
  multica-dev-transition-backend-1    127.0.0.1:18080 -> 8080   <- ALVO da recriacao
  multica-dev-transition-frontend-1   127.0.0.1:13100 -> 3000
  multica-dev-transition-postgres-1   127.0.0.1:15433 -> 5432
  omniroute                           100.118.244.61:20128
  compose project : multica-dev-transition
  compose service : backend
  config_files    : docker-compose.selfhost.yml, docker-compose.selfhost.build.yml,
                    ~/.config/multica-transition/images.yml,
                    ~/.config/multica-transition/backend-env.override.yml

ORQ2 (100.110.178.47) - systemd user units
  multica-daemon-orq2-credential.service   (daemon executor)
  multica-orq1-backend-tunnel.service      (tunel ORQ2 -> ORQ1:18080)
```
**Nao existe `multica-backend.service`.** Rotacionar `JWT_SECRET` significa recriar o container
`backend` do projeto `multica-dev-transition` no **ORQ1**.

Atencao a dependencia cruzada: o daemon do ORQ2 alcanca o backend **pelo tunel**, que e user unit do
ORQ2. A recriacao do container no ORQ1 nao derruba o tunel, mas o daemon perde auth pelo call site 4
e precisa re-emparelhar.

Note tambem `backend-env.override.yml` na lista de config files: **e ali que a variavel efetiva pode
estar sendo definida**. Antes de qualquer execucao e obrigatorio determinar **qual** arquivo fornece
`JWT_SECRET` hoje - eu nao inspecionei esse override, porque ele pode conter segredo.

---

## 4. GATES DE PRONTIDAO (antes de qualquer mutacao)

| Gate | Criterio | Comando (secret-safe) |
|---|---|---|
| **G1 - Fila zero, 4 estados** | `active_tasks = 0` | ver abaixo |
| **G2 - Resolucao do segredo** | referencia dinamica resolve sem erro e **sem imprimir valor** | `asm-exec -- sh -c 'test -n "$JWT_SECRET_NEW" && echo resolved'` |
| **G3 - Saude do backend** | `200` no liveness | `curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18080/health` |
| **G4 - Comprimento minimo** | segredo novo >= 32 bytes, sem imprimir valor | `asm-exec -- sh -c 'printf %s "$JWT_SECRET_NEW" \| wc -c'` |
| **G5 - Origem da variavel** | saber qual config file define `JWT_SECRET` hoje | inspecao dos 4 config files por **nome de chave**, nunca por valor |
| **G6 - Janela anunciada** | comunicado de manutencao enviado (deslogue + re-emparelhamento) | evidencia humana |

### G1 - predicado correto, com os 4 estados ativos
Derivado da constraint em `server/migrations/109_agent_task_waiting_local_directory.up.sql:13-15`,
que lista 7 status dos quais 3 sao terminais (`completed`, `failed`, `cancelled`):
```sql
SELECT count(*) AS active_tasks
FROM agent_task_queue
WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
```
Aceite: `active_tasks = 0`, com a **saida real anexada**. `dispatched` e `waiting_local_directory`
**nao podem** ser omitidos - foi o defeito do GTL-46 e o mesmo defeito reapareceu na V1 deste runbook.

> Recomendacao transversal: o predicado errou duas vezes em runbooks diferentes. Deveria existir
> **uma** fonte reutilizavel, derivada da constraint, em vez de a lista ser reescrita a cada vez.

### G3 - nota de endpoint
`server/cmd/server/router.go:443-445` (literal):
```go
	r.Get("/health", health.liveHandler)
	r.Get("/readyz", health.readyHandler)
	r.Get("/healthz", health.readyHandler)
```
Os tres existem: `/health` e **liveness**, `/readyz` e `/healthz` sao **readiness**. A V1 usava
`/healthz` sem distinguir. Na V2, G3 usa `/health` para liveness e o passo pos-recreate usa
`/readyz`, que e o que prova dependencia pronta.

---

## 5. PROCEDIMENTO - EVENTO DISRUPTIVO UNICO

Todos os passos abaixo sao **propostos**. Os passos 2 a 4 sao classe **STOP-AND-WAIT** e exigem
autorizacao escrita do owner.

### Passo 0 - preparacao (sem mutacao)
1. Validar G1 a G6 e anexar as saidas.
2. Determinar, por G5, qual dos 4 config files define `JWT_SECRET`.
3. Preparar o comunicado de manutencao: usuarios deslogam, daemons re-emparelham, WebSocket reconecta.

### Passo 1 - criar o segredo novo no Secrets Manager (mutacao de AWS, autorizacao 1)
Usar geracao **server-side** de R4. O valor **nunca** aparece no cliente, no argv, no log nem em
contexto de agente. Comprimento >= 32 bytes por G4 e por `jwt.go:55-57`.

### Passo 2 - trocar a variavel na configuracao do backend (autorizacao 2)
Atualizar **apenas** a fonte identificada em G5, usando referencia dinamica:
```
JWT_SECRET={{resolve:secretsmanager:prod/multica/jwt-secret:SecretString:jwt_secret}}
```
Sem escrever o valor em arquivo em claro. Preservar o arquivo anterior como backup em `0600`.

### Passo 3 - RECRIAR o container do backend no ORQ1 (autorizacao 3)
`reload` **nao serve**: `jwtSecretOnce.Do` executa uma vez por processo, logo a chave nova so entra
em um **processo novo**.
```bash
# no ORQ1, projeto multica-dev-transition, servico backend
asm-exec -- docker compose \
  -f docker-compose.selfhost.yml \
  -f docker-compose.selfhost.build.yml \
  -f ~/.config/multica-transition/images.yml \
  -f ~/.config/multica-transition/backend-env.override.yml \
  up -d --force-recreate --no-deps backend
```
`--no-deps` para nao recriar Postgres nem frontend. **Nao** usar `docker restart`: restart reusa a
configuracao antiga de ambiente do container; recreate e o que aplica a variavel nova.

### Passo 4 - re-emparelhar daemons (autorizacao 4)
O daemon do ORQ2 perde auth pelo call site 4. Re-emparelhamento e passo explicito, nao consequencia
automatica, e envolve a unit `multica-daemon-orq2-credential.service`.

### Passo 5 - validacao (secao 6)

### Passo 6 - retirar o segredo antigo
Somente **apos** a validacao passar. `delete-secret` com janela de recuperacao de **7 dias**, nunca
`--force-delete-without-recovery`.

---

## 6. VALIDACAO - TOKEN NOVO ACEITO, TOKEN ANTIGO REJEITADO

Este e o gate que a V1 nao tinha, e ele e obrigatorio porque **reiniciar o processo nao prova que a
chave nova carregou**: `JWTSecret()` resolve **lazily na primeira chamada** (`jwt.go:31`), nao no
boot.

Pre-requisito: **antes** do Passo 3, capturar um token de sessao emitido com a chave **antiga** e
guarda-lo em arquivo `0600` fora de qualquer artefato de evidencia. Token de teste e credencial:
nunca colar em runbook, chat ou log.

```bash
# V1: liveness e readiness apos o recreate
curl -s -o /dev/null -w 'health=%{http_code}\n'  http://127.0.0.1:18080/health
curl -s -o /dev/null -w 'readyz=%{http_code}\n'  http://127.0.0.1:18080/readyz

# V2: TOKEN ANTIGO deve ser REJEITADO  -> esperado 401
curl -s -o /dev/null -w 'old_token=%{http_code}\n' \
  -H "Authorization: Bearer $(cat /run/secrets-tmp/old.jwt)" \
  http://127.0.0.1:18080/api/me

# V3: TOKEN NOVO deve ser ACEITO -> esperado 200
#   emitir novo login apos o recreate e usar o token resultante
curl -s -o /dev/null -w 'new_token=%{http_code}\n' \
  -H "Authorization: Bearer $(cat /run/secrets-tmp/new.jwt)" \
  http://127.0.0.1:18080/api/me
```
**Criterio de aceite, todos obrigatorios:**
1. `health = 200` e `readyz = 200`;
2. `old_token = 401` - se der `200`, a chave antiga **ainda esta em memoria** e a rotacao **nao
   ocorreu**; e falso sucesso e exige investigacao antes de qualquer passo seguinte;
3. `new_token = 200`;
4. daemon re-emparelhado com heartbeat atualizado em `agent_runtime`;
5. `active_tasks` ainda `0` ao final.

Higiene: apagar `/run/secrets-tmp/*.jwt` ao final; nunca anexar esses arquivos a evidencia. O
endpoint exato de verificacao (`/api/me` acima e ilustrativo) deve ser confirmado no router antes da
execucao - **eu nao verifiquei qual rota autenticada mais barata usar**.

---

## 7. ROLLBACK

Gatilho: qualquer criterio da secao 6 falhar, ou taxa de `401`/`403` acima do esperado depois da
janela anunciada.

1. Repor a referencia dinamica para o segredo **antigo** na fonte identificada em G5.
2. **Recriar** o container backend novamente (mesmo comando do Passo 3) - `reload` nao reverte,
   pelo mesmo motivo do `sync.Once`.
3. Re-emparelhar daemons de novo.
4. Repetir a secao 6 com os papeis invertidos: token **antigo** volta a `200`, token novo passa a `401`.

Nao existe rollback "sem downtime": como a chave e unica, tanto ir quanto voltar sao eventos
disruptivos. O segredo antigo **nao deve ser deletado** antes de a validacao passar (Passo 6),
exatamente para manter o rollback possivel.

Ressalva herdada e mantida: rollback **nunca** reaplica branch antiga nem copia bruta de diretorio de
credencial - a arvore AGY antiga com `cli.log` reintroduziria o AGY task-incapaz e antecede o
hardening `0600`/`O_NOFOLLOW`.

---

## 8. AUDITORIA

- CloudTrail: confirmar que **nenhum** agente ou humano chamou `GetSecretValue`; toda leitura deve
  ter vindo da role autorizada via `asm-exec`.
- Contagem de falhas de verificacao no log do backend, **sem** imprimir token:
  ```bash
  docker logs multica-dev-transition-backend-1 2>&1 | grep -c 'auth: invalid token'
  ```
  A string vem de `internal/middleware/auth.go:298-300`, que loga `path` e `error`, **nunca** o token.
- Evidencia a anexar: saidas de G1 a G6, os tres codigos HTTP da secao 6, e o heartbeat do daemon.
  **Nenhum valor de segredo, nenhum token.**

---

## 9. STATUS E CONDICOES DE PARADA

**STATUS: PROPOSTA, AGUARDANDO REVIEW INDEPENDENTE.** Nao me auto-aprovo, e nao repito o erro da V1
de declarar PASS no proprio artefato.

Paro e escalo, sem executar, se:
1. faltar autorizacao escrita do owner para qualquer um dos passos 1 a 4;
2. G1 nao fechar em `0` nos **quatro** estados;
3. G5 nao identificar com certeza qual config file define `JWT_SECRET`;
4. o segredo novo tiver menos de 32 bytes (`jwt.go:55-57` recusa fora de dev/test);
5. `old_token` continuar `200` apos o recreate - falso sucesso;
6. qualquer passo exigir imprimir segredo ou token.

---

## 10. NAO-AFIRMACOES
- READ-ONLY: **nenhum segredo lido**, zero `GetSecretValue`, zero `asm-exec` executado, zero restart,
  zero mutacao de AWS, codigo, banco, container ou unit. Nenhum comando deste runbook foi executado.
- Nao inspecionei `~/.config/multica-transition/backend-env.override.yml`: pode conter segredo, e por
  isso G5 existe como gate em vez de eu ter aberto o arquivo.
- Nao sei se o segredo `prod/multica/jwt-secret` existe; o nome e **proposto**. Nao consultei o
  Secrets Manager.
- Nao verifiquei qual rota autenticada e a mais adequada para a validacao da secao 6; `/api/me` e
  ilustrativo e precisa ser confirmado no router.
- Nao capturei nem gerei token algum.
- Nao validei o comando de `docker compose up -d --force-recreate` contra o host: montei a lista de
  `-f` a partir do label `com.docker.compose.project.config_files` do container em execucao, lido por
  `docker inspect`. A ordem e a completude precisam ser confirmadas por quem executar.
- Nao medi taxa de erro de auth em producao; o gatilho de rollback da secao 7 e qualitativo.
- Nao alterei o artefato V1 de Agy-P0-A8: esta V2 e arquivo novo, e o V1 permanece como entregue.
- Nao criei, atribui nem comentei issue alguma; o freeze de assignment e comentario permanece
  respeitado.
