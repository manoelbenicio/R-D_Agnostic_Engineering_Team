# ORQ-42 - Runbook V7: fechamento de B1-B5 (PROPOSTA, SOMENTE DOCUMENTO)

- **Status: PROPOSTA.** Nao aprovada, **nao executavel**. **Nao me auto-aprovo** - peco peer review
  independente por agente que nao seja eu (autor de V5/V6/V7).
- autor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T18:55Z
- substitui: V6 (`orq42-jwt-rotation-runbook-v6-design.md`)
- fecha: `orq42-jwt-rotation-runbook-v6-independent-review.md`, bloqueadores **B1-B5**
- skill carregada **integralmente** nesta rodada: `.agents/skills/aws-secrets-manager/SKILL.md` v1 -
  overview, aviso *"best-effort defense, not a security boundary"*, as **3 regras MUST**, sintaxe
  `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}` com defaults
  `SecretString`/`AWSCURRENT`, `Using asm-exec` (**resolve em argumentos *e* em variaveis de
  ambiente**), `How It Works` (SMA -> MCP SigV4, regiao por ARN ou `AWS_REGION`, `re.sub` single-pass,
  **sem** fallback para CLI local), SigV4, prerequisitos, Common Patterns incluindo *Configuration file
  templating*, hook `PreToolUse` do `aws-core` e troubleshooting.
- modo: **SOMENTE DOCUMENTO**. Nesta rodada **nao executei nada** - zero
  `GetSecretValue`/`BatchGetSecretValue`, zero SMA, zero `asm-exec`, zero AWS, zero Docker/Compose,
  zero SSH ao ORQ1, zero leitura de `dev.env`, zero `.Config.Env`, zero token, zero DB, zero quadro.

---

## V7.0 - B1: prova e pino de helpers, fonte e **interpolacao** do `dev.env`

O revisor esta certo: a V6 **pinou hashes que ela mesma nao leu** e **afirmou** que `dev.env` e a fonte
de interpolacao sem prova. Corrijo separando o que **e** provado por leitura local do repositorio do que
**ainda nao** foi capturado no ORQ1.

### O que eu **provei** localmente (repositorio, nao-segredo)
`multica-auth-work/docker-compose.selfhost.yml:58`, literal:
```yaml
      JWT_SECRET: ${JWT_SECRET:?JWT_SECRET must be set to a generated value of at least 32 bytes}
```
Isso estabelece **dois** fatos: (a) o valor efetivo do container vem de **interpolacao** de
`${JWT_SECRET}`, e `environment` vence `env_file`; (b) se a interpolacao nao tiver fonte, o Compose
**falha ruidosamente** pelo `:?`. Portanto qualquer recreate bem-sucedido **prova**, por si, que havia
fonte de interpolacao - mas **nao prova qual**.

### O que **falta** e como capturar - `P0` (metadado, sem valor, sem `Config.Env`)
Nenhum comando abaixo le valor de segredo. **Eu nao os executei.**
```bash
# P0.1 identidade do container e as 4 labels canonicas (nao-segredo)
CID=$(docker ps -q --filter 'label=com.docker.compose.project=multica-dev-transition' \
                   --filter 'label=com.docker.compose.service=backend')
test "$(printf %s "$CID" | wc -w)" -eq 1
docker inspect -f 'project={{index .Config.Labels "com.docker.compose.project"}}
service={{index .Config.Labels "com.docker.compose.service"}}
workdir={{index .Config.Labels "com.docker.compose.project.working_dir"}}
configs={{index .Config.Labels "com.docker.compose.project.config_files"}}
envfile={{index .Config.Labels "com.multica.backend.env_file"}}
created={{.Created}}' "$CID"

# P0.2 identidade e modo do arquivo duravel - METADADO, nunca conteudo
stat -c '%n modo=%a dono=%U:%G tam=%s mtime=%Y' "$ENVF"
stat -c '%n modo=%a dono=%U:%G' "$(dirname "$ENVF")"
test -f "$ENVF" && test ! -L "$ENVF"

# P0.3 a chave existe no arquivo, KEY-ONLY, sem imprimir valor
grep -c '^JWT_SECRET=' "$ENVF"            # aceitacao: exatamente 1
grep -cE '^JWT_SECRET=[0-9a-fA-F]{64}$' "$ENVF"   # aceitacao: exatamente 1

# P0.4 hashes dos helpers, comparados por IGUALDADE
sha256sum "$HOME/.local/bin/multica-backend-recreate" "$HOME/.local/bin/multica-backend-env-rollback"

# P0.5 revisao de FONTE dos helpers - o que precisa ser provado por leitura, nao por hash
grep -n -- '--env-file\|--force-recreate\|--no-deps\|-p \|--project-name\|--project-directory' \
  "$HOME/.local/bin/multica-backend-recreate"
```
**Aceitacao de B1** (todas obrigatorias, e nenhuma pode ser presumida):
1. as 4 labels casam a allowlist aprovada e `configs` e o conjunto ordenado exato;
2. `envfile` da label **aponta** para o `dev.env` da allowlist - isso liga container e arquivo **sem**
   ler ambiente;
3. `P0.3` devolve **1** e **1** - a chave existe **uma** vez e ja esta na forma canonica hex-64, que e
   a mesma forma que o editor da secao V7.4 exige;
4. `P0.4` casa por **igualdade** com os hashes registrados no ORQ-30
   (`fa1dae2043035152a3dd818353cc1f02aa7fd7e3aa43c6faad9684c267e576af` e
   `0e42c65e6d2f0ff250bc25925a37cfd3da7754359c1a0ea62c09e66f810b00a4`);
5. `P0.5` **prova por fonte** que o helper passa `--env-file <dev.env>` e recria **so** `backend`.
   Hash igual prova imutabilidade, **nao** comportamento - essa distincao e o nucleo de B1;
6. qualquer item ausente ou divergente = **STOP**. Sem `P0.5` verde, a V7 **nao** autoriza usar o
   helper, e sem `P0.2`/`P0.3` nao autoriza editar o arquivo.

**Pino declarado, a preencher pela captura autorizada** (deixo os campos vazios de proposito, em vez de
inventar valor):
```text
helper recreate  sha256 esperado = fa1dae20…  observado = <P0.4>
helper rollback  sha256 esperado = 0e42c65e…  observado = <P0.4>
dev.env          modo esperado 600 dono ec2-user:ec2-user  observado = <P0.2>
config_files     esperado = allowlist ordenada             observado = <P0.1>
env_file label   esperado = <dev.env da allowlist>          observado = <P0.1>
```

## V7.1 - B2: `G5` deixa de serializar `.Config.Env`

Aceito o BLOCK: propor `docker inspect -f '{{json .Config.Env}}' | grep` contradizia a propria
proibicao da V6 e exporia **variaveis alheias** ao operador e ao log. Substituo por tres verificacoes,
nenhuma serializando ambiente:

- **`G5a` (arquivo, key-only, sem valor)** - o literal nunca pode ter sido gravado na fonte duravel:
  ```bash
  grep -c '^JWT_SECRET={{resolve:' "$ENVF"                  # aceitacao: 0
  grep -cE '^JWT_SECRET=[0-9a-fA-F]{64}$' "$ENVF"           # aceitacao: 1
  ```
  Sai **apenas contagem** de um padrao **nao-secreto**; nenhum valor e impresso, e `grep` sem `-q` aqui
  e seguro porque `-c` emite so o numero.
- **`G5b` (metadado, sem ambiente)** - o container em uso foi criado **a partir** daquele arquivo e
  **depois** da edicao:
  ```bash
  docker inspect -f '{{index .Config.Labels "com.multica.backend.env_file"}}{{"\n"}}{{.Created}}' "$NEW_CID"
  stat -c '%Y' "$ENVF"
  # aceitacao: label == ENVF  E  .Created > mtime do arquivo  E  NEW_CID != OLD_CID
  ```
  Isso substitui a inspecao de ambiente por **ordenacao temporal + procedencia declarada**.
- **`G5c` (opcional, autorizacao propria) - assercao booleana dentro de um container efemero**, jamais
  no container servindo trafego, e emitindo **so** status:
  ```bash
  docker compose -p "$PROJ" "$@" run --rm --no-deps -T --entrypoint /bin/sh backend -c '
    case "${JWT_SECRET-}" in
      "") echo JWT_EMPTY; exit 1 ;;
      *"{{resolve:"*) echo JWT_LITERAL; exit 1 ;;
    esac
    case "$JWT_SECRET" in *[!0-9a-fA-F]*) echo JWT_NOT_HEX; exit 1 ;; esac
    [ "$(printf %s "$JWT_SECRET" | wc -c)" -eq 64 ] || { echo JWT_LEN; exit 1; }
    echo JWT_SHAPE_OK'
  # unica saida permitida: uma das palavras acima. Proibido echo do valor, redirecionar para arquivo,
  # ou usar `env`/`set`/`printenv`.
  ```
  `G5c` le a variavel **dentro do filho** e devolve **um rotulo**; e a versao "child-side boolean" que o
  revisor pediu. Ainda assim exige **autorizacao separada**, porque cria um container efemero.

Registro por que a checagem anti-literal continua necessaria: o literal `{{resolve:...}}` tem mais de
32 bytes, passaria `ValidateJWTConfiguration` (`internal/auth/jwt.go:46-58`) e o backend
assinaria/verificaria **consistentemente** - logo `antigo=401` e `novo=200` **passariam** e o defeito
seria invisivel por HTTP.

## V7.2 - B3: matriz completa de consumidores de `JWTSecret()`

Os 5 consumidores medidos: `internal/handler/auth.go:213,226` (assinam),
`internal/middleware/auth.go:295`, `internal/middleware/daemon_auth.go:234` e
`internal/realtime/hub.go:691` (verificam).

| # | consumidor | caminho de auth | impacto da rotacao | gate de aceitacao |
|---|---|---|---|---|
| **C1** | HTTP de usuario, `GET /api/me` (`cmd/server/router.go:552`), cookie `multica_auth` (`internal/auth/cookie.go:20-21`), precedencia `Bearer` antes do cookie (`middleware/auth.go:329-341`) | JWT | **quebra**: toda sessao humana cai | `anonimo=401` (pre), `antigo=200` (pre), `antigo=401` **e** `novo=200` (pos) - codigos **exatos**, `curl -f` proibido |
| **C2** | **realtime de usuario**, `GET /ws` (`router.go:468`) via `hub.go:676-691 authenticateToken`, que trata **so** `mul_` e verifica **todo o resto** com `JWTSecret()` | JWT | **quebra**: WebSocket de usuario cai | tentativa de upgrade com credencial em arquivo `--config` `0600`: **antigo** nao pode devolver `101`; **novo** deve devolver `101`. Codigo HTTP do handshake e suficiente como gate |
| **C3** | REST de daemon, `/api/daemon/*` com `mdt_` -> `auth.HashToken` + `GetDaemonTokenByHash` (`daemon_auth.go:107-147`) | **hash em DB**, **nao** JWT | **nao quebra** | verificacao pos-rotacao: heartbeat verde e runtimes online. **Nenhuma** mutacao no ORQ2 |
| **C4** | **WebSocket de daemon**, `/api/daemon/ws` (`wakeup.go:343`, registrado em `router.go:516` **dentro** do grupo `DaemonAuth`), com o **mesmo** token do cliente (`wakeup.go:81-83`) | **hash em DB** | **nao quebra** | coberto por C3 |
| **C5** | PAT `mul_` e cloud `mcn_` (`daemon_auth.go:190`, `:150`) | hash / verificador cloud | **nao quebra** | verificacao especifica do caminho, sem premissa de JWT |
| **C6** | fallback JWT de daemon (`daemon_auth.go:229-234`) | JWT | **quebra** *se* algum daemon usar esse caminho | **so** aqui o re-pareamento e exigido - e **so** se medido |

**Re-pareamento continua condicional e medido**, nunca inferido da existencia do daemon. Evidencia do
caminho ativo, sem token e sem valor:
- **E-1** log: o rotulo existe (`daemon_auth.go:27-30`: `daemon_token`, `pat`, `cloud_pat`, `jwt`) mas o
  **unico** consumidor fora de teste e `internal/handler/daemon.go:747`, que o passa a
  `logHeartbeatEndpointSlow` - ou seja aparece **so** em heartbeat **lento** e **nunca** em resposta de
  API. Ausencia de linha = **inconclusivo**, nao negativo;
- **E-2** (preferida) metadado de DB: `SELECT daemon_id, expires_at, created_at FROM daemon_token
  WHERE daemon_id = '<id>'` - **colunas nomeadas**, **nunca** a coluna de hash, **nunca** `SELECT *`.
  Linha nao expirada = caminho `mdt_`.
- indeterminado = **STOP**.

**Janela de comunicacao (nova exigencia, autorizacao propria):** como C1 e C2 quebram, o owner precisa
autorizar **e** comunicar a janela aos usuarios antes do L2. Isso e a **Q-G**, e nao pode ser suprida
por agente.

## V7.3 - B4/Q-E: custodia e obtencao de token **sem** agente ver valor

Resolvo `Q-E` nomeando o mecanismo, os caminhos, os modos, o dono da custodia e o recibo - e **reuso um
padrao ja revisado** em vez de inventar: o do ORQ-17 Stage3B, em que o segredo vai por **stdin** ate o
`curl` e **nunca** a `argv` nem a arquivo.

| item | definicao |
|---|---|
| dono da custodia | **owner humano**; nenhum agente participa |
| aquisicao do token | o **owner** executa o login e grava o cookie em jar privado: `curl --cookie-jar "$TOKDIR/old.jar" --config "$TOKDIR/login.cfg" http://127.0.0.1:18080/auth/login`, com o corpo JSON entregue por **stdin** (padrao ORQ-17), nunca em `argv` |
| caminhos nomeados | `TOKDIR=/home/ec2-user/.local/state/orq42-tokens` (`0700`); `old.jar`, `new.jar`, `old.cfg`, `new.cfg` - todos `0600`, arquivos regulares, **nao** symlink, dono `ec2-user` |
| verificacao pelo agente | **apenas** `stat -c '%a %U'` e **codigo de saida** do `curl`. O agente **nao** le jar nem cfg |
| retencao | ambos preservados **ate** a janela ser aceita e o rollback impossivel - `R4` precisa deles |
| limpeza | passo **final autorizado pelo owner**: `rm -f` dos 4 arquivos + `rmdir`, com **recibo** declarando os 4 caminhos e o `sha256` do **recibo** (nao dos tokens) |
| recusa de `shred` | mantida e fundamentada: CoW e wear-leveling tornam `shred` uma falsa garantia; a protecao real e `0600` em diretorio `0700` |
| prova pre-mutacao | `anonimo=401` **e** `antigo=200`. Sem os dois, **nao** se rotaciona - um token ja expirado daria `401` depois e seria lido como "invalidacao provada" |

## V7.4 - B5: referencia dinamica fail-closed, sem valor

**Identificacao pinada, a preencher pelo owner** (nao invento nome de segredo):
```text
ARN completo    arn:aws:secretsmanager:<REGIAO>:<CONTA>:secret:<NOME>-<SUFIXO>   (Q-B)
regiao          <REGIAO>, amarrada a conta                                        (Q-B)
json-key        <JSON-KEY>                                                        (Q-B)
version-stage   AWSCURRENT, CONGELADO durante a janela                            (Q-B)
```
Checagem de existencia/identidade **somente metadado**, por identidade autorizada:
`describe-secret` e `list-secret-version-ids` inspecionando `VersionIdsToStages`. Nenhuma das duas e
`get-secret-value`/`batch-get-secret-value`, logo nao violam a **regra 1**; a saida deve ser reduzida a
**booleanos** no runbook, sem colar ARNs de versao. **Eu nao as executei**, e a identidade desta sessao
(`arn:aws:sts::809809509961:assumed-role/cw-agent-orquestradores/…`) ja recebeu **AccessDenied** em
`describe-secret` em `sa-east-1`, o que torna `Q-C` bloqueante.

Injecao pelo **ambiente**, nunca por `argv` (a skill declara que o `asm-exec` resolve **argumentos e
variaveis de ambiente**):
```bash
ROTATING_JWT='{{resolve:secretsmanager:<ARN>:SecretString:<JSON-KEY>:AWSCURRENT}}' \
ENVF_PATH="$ENVF" EDITOR_BIN="$APPROVED_EDITOR" \
asm-exec -- sh -eu -c '
  S=$ROTATING_JWT
  [ -n "$S" ]                            || { echo FAIL_EMPTY;      exit 1; }
  case "$S" in *"{{resolve:"*)              echo FAIL_UNRESOLVED;   exit 1 ;; esac
  case "$S" in *[!0-9a-fA-F]*)              echo FAIL_NOT_HEX;      exit 1 ;; esac
  [ "$(printf %s "$S" | wc -c)" -eq 64 ] || { echo FAIL_LEN;        exit 1; }
  printf %s "$S" | "$EDITOR_BIN" --replace-jwt-secret --file "$ENVF_PATH" --stdin-value'
```
**Prova de que os bytes nao-alvo nao mudaram** (exigencia explicita de B5), pelo editor fixado por hash:
1. copia de backup por `mktemp` no mesmo diretorio, `cmp -s` contra o original;
2. exigir **exatamente uma** linha `^JWT_SECRET=<64hex>$`; zero ou multiplas = **STOP**;
3. gravar temporario `O_EXCL` no mesmo diretorio, `fsync` do arquivo **e** do diretorio, `rename(2)`;
4. **pos-edicao**: `diff` entre backup e novo tem de conter **exatamente uma** hunk de **uma** linha,
   cuja unica diferenca e a linha `JWT_SECRET=` - verificado por
   `diff <(grep -v '^JWT_SECRET=' bak) <(grep -v '^JWT_SECRET=' novo)` devolvendo **vazio**, o que prova
   igualdade byte-a-byte de **todo o resto** sem imprimir o valor;
5. `trap` em `EXIT/INT/TERM` limpando temporarios; saida so em rotulos fixos.

## V7.5 - sequencia consolidada

```
P0    B1: labels, identidade/modo do dev.env, key-only, hashes E FONTE dos helpers   [STOP se algo faltar]
E-2   caminho de auth do daemon por metadado                                          [indeterminado = STOP]
Q-C   existencia do segredo por metadado, identidade autorizada
T     anonimo /api/me = 401  e  antigo /api/me = 200                                  [B4]
T-ws  antigo /ws NAO devolve 101                                                       [C2, baseline]
BK    backup por mktemp + cmp -s + fsync
      -- fronteira: nada acima muta nada --
L2    asm-exec com referencia no AMBIENTE; G1-G4; editor fixado; prova de bytes nao-alvo [B5]
REC   recreate pelo helper com hash E fonte provados (--env-file dev.env, --no-deps backend)
V     NEW_CID != OLD_CID; /health 200; /readyz 200
G5a   grep key-only: 0 literais, 1 linha hex-64                                        [B2]
G5b   label env_file == ENVF  E  .Created > mtime                                      [B2]
G5c   (opcional, autorizacao propria) rotulo booleano em container efemero             [B2]
A     antigo /api/me = 401  e  novo /api/me = 200                                      [C1]
A-ws  antigo /ws nao 101  e  novo /ws = 101                                            [C2]
D     daemon conforme o caminho medido em E-2 (sem mutacao se for mdt_)                [C3-C6]
R     rollback simetrico; R4 com os tokens preservados; repetir G5a/G5b e inverter A/A-ws
```

## V7.6 - perguntas e autorizacoes

| # | pergunta | estado |
|---|---|---|
| **Q-A** | usar os helpers ORQ-30 (padrao) - agora condicionada a `P0.5`, prova de **fonte**, nao so de hash | **ABERTA** |
| **Q-B** | **ARN completo**, regiao, `json-key`, estagio congelado | **ABERTA** |
| **Q-C** | existencia por **metadado**, por identidade autorizada (esta sessao levou `AccessDenied`) | **ABERTA/BLOQUEANTE** |
| **Q-D** | atestacao de **64 hex** sem revelar valor | **ABERTA (contrato ok)** |
| **Q-E** | **RESOLVIDA NA V7.3** como proposta: mecanismo, caminhos, modos, dono e recibo nomeados; falta a **confirmacao** do owner | **PROPOSTA, aguarda confirmacao** |
| **Q-F** | re-pareamento **condicional** ao caminho medido (`E-2`) | **ABERTA** |
| **Q-G** | aceite **e comunicacao** da queda de sessoes humanas e do realtime `/ws` | **ABERTA** |

| # | autorizacao |
|---|---|
| **A1** | leitura AWS **somente metadado**; mutacao de segredo **fora de escopo** |
| **A2** | captura `P0` no ORQ1 (metadado + leitura de **fonte** dos helpers) |
| **A3** | edicao do `dev.env` + recreate `backend` + gates |
| **A4** | rollback **pre**-autorizado, com retencao dos tokens |
| **A5** | aceite da exposicao residual em `/proc/<pid>/environ` |
| **A6** | ORQ2 **condicional**: so se `E-2` indicar caminho `jwt` |
| **A7** | **nova**: `G5c`, container efemero de assercao |

## V7.7 - nao-afirmacoes

- **Nesta rodada nao executei nada.** Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA, zero
  `asm-exec`, zero AWS, zero Docker/Compose, **zero SSH ao ORQ1**, zero leitura de `dev.env`, zero
  `.Config.Env`, zero token, zero DB, zero quadro.
- **Nao li o fonte dos helpers** e **nao** verifiquei seus hashes: `P0.4`/`P0.5` sao **gates propostos**,
  e por isso os campos "observado" do pino estao **vazios**. Nao repito o erro da V6 de pinar o que nao
  li.
- **Nao reli as labels do ORQ1** nesta rodada; a allowlist vem da evidencia do Stage 2 do ORQ-17 e
  precisa ser reconfirmada por `P0.1`.
- O unico fato que **provei** aqui e local e nao-secreto: `docker-compose.selfhost.yml:58` declara
  `JWT_SECRET` por interpolacao com `:?`.
- Os fatos de codigo da matriz V7.2 vem de leituras minhas anteriores (`daemon_auth.go`, `hub.go`,
  `wakeup.go`, `router.go`, `cookie.go`, `jwt.go`); **nao** foram remedidos agora.
- **Nao adivinho nome de segredo**; `Q-B` fica com placeholders.
- **Nao elimino** a exposicao residual de `/proc/environ` - troco `argv` por `environ` e peco `A5`.
- `G5c` **le** `JWT_SECRET` dentro do filho; e menos invasivo que serializar `.Config.Env`, mas **nao**
  e zero-contato, e por isso e opcional e tem autorizacao propria (`A7`).
- O gate de `/ws` usa o **codigo do handshake**; `curl` nao completa a sessao WebSocket, e eu **nao** o
  testei.
- Nao alterei assignee, nao postei comentario, nao criei card.

---

**PROPOSTA - requer peer review independente.** Ataques prioritarios sugeridos: (a) se `G5a`+`G5b`
realmente substituem a garantia que `G5` dava, ou se so `G5c` a da; (b) se o `diff` de bytes nao-alvo
proposto em V7.4 e suficiente sem imprimir nada; (c) se o gate de `/ws` por codigo de handshake e
aceitavel; (d) se a custodia de token da V7.3 mantem o agente fora do valor em **todos** os passos.
