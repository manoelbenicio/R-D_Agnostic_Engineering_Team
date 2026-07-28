# ORQ-33 - revisao INDEPENDENTE do runbook de rev-token e do BLOCK GTL-87

Revisor: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T14:25Z
Cartao: **ORQ-33** · UUID `dfeabbdc-33e1-4ab8-9460-27b43df227db` (conforme citado nos dois artefatos)
Documentos revisados:
- `.deploy-control/p0/evidence/orq33-rev-token-rotation-runbook.md` (Agy-P0-A8, veredito proprio PASS)
- `.deploy-control/p0/evidence/gtl-orq33-rev-token-rotation-peer-review.md` (Antigravity w8:p2, GTL-87, **BLOCK**)
Modo: READ-ONLY. **Nenhum segredo lido**, nenhum `GetSecretValue`, nenhum restart, nenhuma mutacao de
codigo, banco ou infra. Nao criei, atribui nem comentei issue.

---

# VEREDITO: **BLOCK MANTIDO**, e ampliado

Confirmo os 3 achados do GTL-87 por medicao propria e independente. Acrescento **2 achados novos** que
o GTL-87 nao levantou, sendo o primeiro deles material para o raio de impacto da rotacao. Corrijo
tambem **2 imprecisoes** de referencia presentes nos dois documentos.

| # | achado | origem | status |
|---|---|---|---|
| 1 | chave JWT unica, sem dual-key | GTL-87 | **CONFIRMADO** |
| 2 | `sync.Once` impede recarga sem restart de processo | GTL-87 | **CONFIRMADO** |
| 3 | Gate 1 omite `dispatched` e `waiting_local_directory` | GTL-87 | **CONFIRMADO** |
| 4 | o runbook lista 3 de **5** consumidores; faltam daemon auth e WebSocket | **NOVO (meu)** | BLOQUEANTE |
| 5 | nao existe subsistema de "rev token" no codigo; o objeto real e `JWT_SECRET` | **NOVO (meu)** | BLOQUEANTE de nomenclatura |
| 6 | caminho errado de `pat_cache.go` / `membership_cache.go` | correcao (ambos os docs) | precisao |
| 7 | `systemctl reload multica-backend` nao corresponde a topologia medida | correcao (runbook) | precisao |

---

## 1. CONFIRMACAO DO ACHADO 1 - chave unica

`server/internal/auth/jwt.go:25-40` (literal):
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
Uma unica variavel, um unico `os.Getenv`. `grep -rniE 'kid|secondary|previous.*secret|keys\['` em
`internal/auth/*.go` retorna **apenas** a linha 23 (`ErrInsecureJWTConfiguration`) e a 32
(`os.Getenv`). **Nao existe `kid`, nao existe conjunto de chaves, nao existe chave secundaria.**

A verificacao reforca: `internal/middleware/auth.go:291-296` (literal)
```go
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return auth.JWTSecret(), nil
			})
```
O `keyfunc` devolve **uma** chave, e nao inspeciona `token.Header["kid"]`. Assinatura em
`internal/handler/auth.go:206` e `:218` usa `jwt.SigningMethodHS256`, ou seja **HMAC simetrico**: a
mesma chave assina e verifica. Portanto a janela de sobreposicao da secao 3 do runbook e
tecnicamente impossivel no codigo atual - **confirmo a premissa falsa.**

## 2. CONFIRMACAO DO ACHADO 2 - `sync.Once`

`jwtSecretOnce.Do` executa **uma vez por processo**. Mudar a variavel de ambiente e mandar
`reload` nao reexecuta o closure; o `[]byte` em memoria permanece. So um **novo processo** relê
`JWT_SECRET`. Confirmo.

Precisao que o GTL-87 nao explicitou e que muda o teste de aceite: como `JWTSecret()` e resolvido
**lazily na primeira chamada**, e nao no boot, o segredo efetivo e o que estava no ambiente **no
momento do primeiro pedido autenticado**, nao necessariamente o do `ExecStart`. Em pratica isso e
irrelevante se o env nao muda em voo, mas significa que "reiniciei o processo" **nao prova** que a
chave nova foi carregada - so um pedido autenticado bem-sucedido depois do restart prova.

## 3. CONFIRMACAO DO ACHADO 3 - Gate 1 incompleto

`server/migrations/109_agent_task_waiting_local_directory.up.sql:13-15` (literal):
```sql
ALTER TABLE agent_task_queue DROP CONSTRAINT IF EXISTS agent_task_queue_status_check;
ALTER TABLE agent_task_queue ADD CONSTRAINT agent_task_queue_status_check
CHECK (status IN ('queued', 'dispatched', 'running', 'waiting_local_directory', 'completed', 'failed', 'cancelled'));
```
7 status, 3 terminais (`completed`, `failed`, `cancelled`), logo **4 ativos**. O Gate 1 do runbook
filtra apenas 2 e declararia fila zero com tarefa em voo em `dispatched` ou
`waiting_local_directory`. Confirmo, e observo que este e o **mesmo defeito** que eu ja havia
apontado no GTL-37 contra o parecer GTL-46: e reincidente, o que sugere que o predicado correto
deveria virar um snippet unico e reutilizavel em vez de ser reescrito em cada runbook.

## 4. ACHADO NOVO - o runbook lista 3 de 5 consumidores da chave

`grep -rn 'JWTSecret()' --include='*.go'` (excluindo testes) devolve **5 call sites** alem da
definicao:

| arquivo:linha | papel |
|---|---|
| `internal/handler/auth.go:213` | **assina** (`token.SignedString`) |
| `internal/handler/auth.go:226` | **assina** (`token.SignedString`) |
| `internal/middleware/auth.go:295` | verifica - API HTTP |
| `internal/middleware/daemon_auth.go:234` | verifica - **auth de daemon** |
| `internal/realtime/hub.go:691` | verifica - **WebSocket / realtime** |

A secao 2.2 do runbook lista como consumidores apenas `middleware/auth.go`, os caches e "Daemons
Efemeros e SDKs". **Nao menciona `internal/realtime/hub.go`.** Consequencia concreta e nao
declarada: com chave unica e HS256, trocar `JWT_SECRET` invalida tambem **todas as conexoes
WebSocket autenticadas** e o **auth de daemon** no mesmo instante. O runbook descreve o evento como
rotacao sem downtime; medido, ele e um evento disruptivo simultaneo em **tres** superficies de
verificacao, nao uma. O plano de comunicacao e de rollback precisa cobrir realtime e daemons
explicitamente.

## 5. ACHADO NOVO - "rev token" nao existe no codigo

`grep -rniE 'rev_token|revToken|REV_TOKEN' --include='*.go'` (sem testes): **zero ocorrencias**.
`grep -nE 'JWT_SECRET|REV_TOKEN|APP_ENV' docker-compose.selfhost.yml`:
```
58:      JWT_SECRET: ${JWT_SECRET:?JWT_SECRET must be set to a generated value of at least 32 bytes}
84:      APP_ENV: ${APP_ENV:-production}
```
Existe `JWT_SECRET` (obrigatorio, com guarda de 32 bytes) e `APP_ENV`; **nao existe `REV_TOKEN`**.
As unicas ocorrencias de "revocation/revoke" estao em `internal/auth/cloud_pat.go`, e ali a
revogacao e por **TTL de PAT de Cloud**, nao por token de revogacao assinado
(`cloud_pat.go:43` diz literalmente que "the TTL itself IS the revocation").

Ou seja: o runbook inteiro nomeia e desenha a rotacao de um artefato - `prod/multica/rev-token/primary`
e `/secondary`, com "emissores que assinam eventos de revogacao" - que **nao tem contraparte no
codigo**. O GTL-87 acertou ao mapear o problema para `JWT_SECRET`, mas nao registrou que a
**nomenclatura do runbook e ficcional**. Isso importa por dois motivos operacionais:
1. os nomes de segredo propostos nao correspondem a nenhuma variavel que o backend le, logo criar
   `prod/multica/rev-token/*` no Secrets Manager produziria segredo orfao;
2. um operador seguindo o runbook literalmente giraria um segredo que ninguem consome, concluiria
   "rotacao concluida" e deixaria `JWT_SECRET` intacto - **falso sucesso**, a pior classe de falha.

## 6. CORRECAO DE PRECISAO - caminho dos caches

Os dois documentos referenciam `pat_cache.go` e `membership_cache.go` como se estivessem em
`internal/handler/`. Medido, estao em **`internal/auth/`**:
```
internal/auth/pat_cache.go:19        const AuthCacheTTL = 10 * time.Minute
internal/auth/membership_cache.go:19 const MembershipCacheTTL = 5 * time.Minute
```
Os **valores** citados pelo GTL-87 (10 min e 5 min) estao corretos; apenas o caminho esta errado.
E a ressalva do GTL-87 procede: com chave unica, a "janela de drenagem" de 15 minutos do Passo 4 do
runbook **nao drena nada** - a troca invalida tudo instantaneamente. O TTL de cache governa PAT e
membership, nao a validade de JWT.

## 7. CORRECAO DE PRECISAO - `systemctl reload multica-backend`

O Passo 2 do runbook propoe `asm-exec -- systemctl reload multica-backend`. Duas objecoes:
1. `reload` nao reexecuta `sync.Once` (achado 2), logo e inoperante para o objetivo;
2. a topologia medida nao tem essa unit. O backend roda como **container Docker no ORQ1**
   (`multica-dev-transition-backend-1`, `127.0.0.1:18080->8080`), e as units systemd que existem sao
   do **ORQ2** (`multica-daemon-orq2-credential.service` e `multica-orq1-backend-tunnel.service`).
   Nao existe `multica-backend.service`.
A correcao do GTL-87 ("substituir por `docker restart`/`systemctl restart`") esta na direcao certa,
mas precisa nomear o alvo real: **recriar o container do backend no ORQ1**, e isso e acao de classe
STOP-AND-WAIT que exige autorizacao do owner.

Nota tambem no Gate 3 do runbook: ele usa `http://localhost:18080/healthz`. O endpoint que eu medi
respondendo `200` e `/health` em `127.0.0.1:18080`; nao verifiquei `/healthz`. Precisa ser confirmado
antes de virar gate.

---

## 8. CORRECAO MINIMA PROPOSTA (o que foi pedido)

Tres correcoes, na ordem de dependencia. Nenhuma implementada por mim.

### 8.1 Chave JWT unica - correcao minima de RUNBOOK, nao de codigo

A correcao minima **nao** e implementar dual-key. E declarar a verdade e ajustar o procedimento:

> A rotacao de `JWT_SECRET` no codigo atual e um **evento disruptivo unico**, nao uma rotacao com
> sobreposicao. Ela invalida, no mesmo instante e em tres superficies: (a) tokens de API via
> `middleware/auth.go:295`; (b) auth de daemon via `middleware/daemon_auth.go:234`; (c) sessoes
> WebSocket via `realtime/hub.go:691`. Nao existe chave secundaria (`jwt.go:25-40`, HS256 simetrico,
> `keyfunc` sem `kid`). As fases 1 a 4 da secao 3 e o Passo 4 de drenagem de 15 minutos devem ser
> **removidos**, porque descrevem capacidade inexistente.

Substituir o desenho de 4 fases por um procedimento honesto de 3 passos: janela de manutencao
anunciada, recriacao do container do backend com o novo `JWT_SECRET`, e re-emparelhamento de daemons.

Se dual-key for realmente desejado, a correcao **minima de codigo** seria aditiva e cabe em uma
funcao: manter `JWTSecret()` intacta para assinatura e introduzir um verificador que tente as chaves
em ordem, algo como
```go
// PROPOSTA, NAO IMPLEMENTADA
func JWTVerificationKeys() [][]byte   // [primary, secondary...] a partir de JWT_SECRET e JWT_SECRET_PREVIOUS
```
consumida pelos **tres** `keyfunc`. Isso e RFC propria, com cartao proprio, e **nao** pertence ao
ORQ-33.

### 8.2 `sync.Once` - correcao minima

Nao remover o `sync.Once`: ele existe para evitar releitura em hot path e e correto. A correcao
minima e **de procedimento**, mais um teste de aceite que hoje nao existe:
1. trocar `systemctl reload multica-backend` por **recriacao do container do backend no ORQ1**,
   nomeando o container real, e classificar o passo como STOP-AND-WAIT;
2. adicionar gate pos-restart que **prova** o carregamento: um pedido autenticado com token novo
   deve retornar `200` **e** um pedido com token antigo deve retornar `401`. Isso e necessario porque
   `JWTSecret()` resolve na **primeira chamada**, nao no boot (secao 2), logo "o processo reiniciou"
   nao prova nada por si.

Se, em vez disso, o objetivo for permitir recarga sem restart, a mudanca minima de codigo seria
trocar `sync.Once` por `atomic.Pointer[[]byte]` com um gatilho explicito de recarga - novamente RFC
propria, fora do ORQ-33.

### 8.3 `dispatched` / `waiting_local_directory` - correcao minima

Trocar o predicado do Gate 1 por:
```sql
SELECT count(*) AS active_tasks
FROM agent_task_queue
WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
```
Aceite: `active_tasks = 0`, com a saida real anexada.

Recomendacao adicional, porque o defeito e reincidente (GTL-46, e agora GTL-87): definir **uma**
fonte para esse predicado, derivada da constraint em `migrations/109:15`, e referencia-la em todo
runbook em vez de reescrever a lista. Um runbook que enumera status a mao vai divergir de novo na
proxima migration que adicionar estado.

---

## 9. O QUE O RUNBOOK ACERTA (registro explicito)

Concordo integralmente com o GTL-87 nesta parte, e reconfirmo:
1. **Disciplina de segredo exemplar.** Proibicao de `GetSecretValue`/`BatchGetSecretValue`, proibicao
   de ler o SMA em `localhost:2773`, uso de `asm-exec` e de `{{resolve:secretsmanager:...}}`. Nenhum
   valor de segredo aparece em nenhum dos dois artefatos.
2. **Estrutura de gates antes de mutacao** e plano de rollback existirem e serem explicitos.
3. **Auditoria via CloudTrail** para provar que nenhum agente chamou `GetSecretValue` e um controle
   correto e verificavel.
Uma ressalva no Passo 1: `aws secretsmanager create-secret --secret-string '{"token":"..."}'` coloca
o valor em **argumento de linha de comando**, visivel em `ps` e em historico de shell. Mesmo com
placeholder, o padrao ensinado esta errado; o correto e `--secret-string fileb://` ou geracao
server-side. Isso contradiz a propria Regra 3 do runbook.

---

## 10. CONDICOES PARA VIRAR PASS

1. Reescrever as secoes 2.2, 3 e o Passo 4 declarando evento disruptivo unico, sem janela de
   sobreposicao, e listando os **5** call sites de `JWTSecret()`.
2. Renomear o objeto de rotacao de `rev-token` para `JWT_SECRET`, ou justificar a existencia de um
   artefato separado com referencia a codigo. Sem isso o runbook gira segredo que ninguem consome.
3. Corrigir o Gate 1 para os 4 status ativos, com saida anexada.
4. Substituir `systemctl reload multica-backend` pela recriacao do container real do backend no
   ORQ1, marcada como STOP-AND-WAIT.
5. Adicionar o gate pos-restart de prova de carga da chave (200 com token novo, 401 com token antigo).
6. Corrigir o caminho dos caches para `internal/auth/` e remover a alegacao de drenagem por TTL.
7. Confirmar se o endpoint de saude e `/health` ou `/healthz`.
8. Corrigir o Passo 1 para nao passar valor de segredo em argumento de comando.

---

## 11. NAO-AFIRMACOES
- READ-ONLY: **nenhum segredo lido**, nenhum `GetSecretValue`, nenhum `asm-exec` executado, nenhum
  restart, nenhuma mutacao de codigo, banco, container ou unit.
- Nao implementei nenhuma das correcoes: 8.1, 8.2 e 8.3 sao propostas.
- Nao executei o Gate 1 corrigido nem qualquer comando do runbook.
- Nao verifiquei se o endpoint `/healthz` existe; medi `/health` respondendo `200` em outra auditoria.
- Nao inspecionei o Secrets Manager: nao sei se `prod/multica/rev-token/primary` existe. Afirmo
  apenas que **nenhuma variavel de ambiente do backend le esse nome**.
- Nao verifiquei se `internal/realtime/hub.go:691` esta em caminho de codigo ativo hoje; ele existe e
  chama `JWTSecret()`, e por isso entra na lista de superficies afetadas.
- Nao avaliei o restante de `cloud_pat.go`: a revogacao por TTL de PAT de Cloud e mecanismo distinto
  e fora do escopo desta revisao.
- Nao contestei o UUID `dfeabbdc-33e1-4ab8-9460-27b43df227db`: aceitei como citado nos dois
  artefatos, sem consultar o board, que permanece sob restricao.
- Nao criei, atribui nem comentei issue alguma.
