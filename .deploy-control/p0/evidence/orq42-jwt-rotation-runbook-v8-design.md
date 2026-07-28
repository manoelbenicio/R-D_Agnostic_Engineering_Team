# ORQ-42 - Runbook V8: fechamento exato de B1-B5 (PROPOSTA, SOMENTE DOCUMENTO)

- **Status: PROPOSTA.** Nao aprovada, **nao executavel**. **Nao me auto-aprovo** - peco re-review
  independente por agente que nao seja eu (autor de V5/V6/V7/V8).
- autor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T19:20Z
- substitui: V7 (`orq42-jwt-rotation-runbook-v7-design.md`, sha256 `b3b54413…`)
- skill carregada **integralmente** nesta rodada: `.agents/skills/aws-secrets-manager/SKILL.md` v1 -
  overview, aviso *"best-effort defense, not a security boundary"*, as **3 regras MUST**, sintaxe
  `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}` com defaults
  `SecretString`/`AWSCURRENT`, `Using asm-exec` (**resolve em argumentos *e* em variaveis de
  ambiente**), `How It Works` (SMA -> MCP SigV4, regiao por ARN ou `AWS_REGION`, `re.sub` single-pass,
  **sem** fallback para CLI local), SigV4, prerequisitos, Common Patterns incl. *Configuration file
  templating*, hook `PreToolUse` do `aws-core`, troubleshooting.
- modo: **SOMENTE DOCUMENTO.** Nesta rodada executei apenas leituras **locais e nao-secretas** do
  repositorio e do `PATH` (secao V8.9). Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA, zero
  `asm-exec`, zero AWS, zero Docker/Compose, zero SSH ao ORQ1, zero leitura de `dev.env`, zero
  `.Config.Env`, zero token, zero DB, zero quadro.

---

## B1 - allowlist **absoluta**, pino de helper **e** editor com **fonte integral**, e `env -u JWT_SECRET`

### B1.1 Allowlist absoluta (nada derivado, nada inferido)
```text
host                ip-172-31-18-217.sa-east-1.compute.internal   (100.118.244.61)
compose project     multica-dev-transition
compose service     backend
arquivo duravel     /home/ec2-user/.config/multica-transition/dev.env            modo 0600 ec2-user:ec2-user
diretorio pai       /home/ec2-user/.config/multica-transition                    modo 0700 ec2-user:ec2-user
config_files (5, NESTA ORDEM):
  1 /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml
  2 /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml
  3 /home/ec2-user/.config/multica-transition/images.yml
  4 /home/ec2-user/.config/multica-transition/backend-env.override.yml
  5 /home/ec2-user/.config/multica-transition/orq17-auth-cutover.override.yml
helper recreate     /home/ec2-user/.local/bin/multica-backend-recreate       0700  sha256 fa1dae2043035152a3dd818353cc1f02aa7fd7e3aa43c6faad9684c267e576af
helper rollback     /home/ec2-user/.local/bin/multica-backend-env-rollback   0700  sha256 0e42c65e6d2f0ff250bc25925a37cfd3da7754359c1a0ea62c09e66f810b00a4
editor              <APPROVED_EDITOR>                                        0700  sha256 <A PREENCHER: nao existe ainda>
```
Regras: **igualdade exata** de conjunto e **ordem** para os 5 `config_files`; qualquer caminho relativo,
duplicado, vazio, symlink ou fora da lista = **STOP**; label
`com.multica.backend.env_file` deve apontar ao `dev.env` da lista; `stat` numerico obrigatorio para
modo/dono do arquivo **e** do pai; **nunca** ler conteudo.

### B1.2 Fonte **integral**, nao `grep`
A V7 propunha `grep` nos helpers. Insuficiente: `grep` prova presenca de um argumento, nao **ausencia**
de outro. Exigencia da V8: **leitura integral** dos dois helpers e do editor, por revisor independente,
com estas assercoes por leitura de codigo, cada uma marcada SIM/NAO no parecer:
1. usa `docker compose` (plugin), nunca `docker-compose`;
2. passa **exatamente** os 5 `-f` da B1.1, na ordem;
3. passa `--env-file <dev.env absoluto>`;
4. passa `-p multica-dev-transition` (ou `--project-name`) **e** `--project-directory` fixado;
5. usa `up -d --force-recreate --no-deps backend` e **nada mais** - **sem** `--no-recreate`, sem
   `--build`, sem `down`, sem `restart`, sem outro servico;
6. **nao** escreve, imprime, copia nem faz `diff` de `dev.env`;
7. **nao** exporta `JWT_SECRET` nem qualquer variavel resolvida;
8. `set -eu` (ou equivalente) e nenhuma continuacao silenciosa em erro.
Hash igual sem essas 8 respostas = **STOP**. Hash prova imutabilidade, **nao** comportamento.

O editor **nao existe ainda**: e um artefato a construir, revisar e pinar **antes** da janela. A V8
**nao** autoriza edicao com ferramenta ad-hoc.

### B1.3 `env -u JWT_SECRET` em **toda** invocacao - armadilha real
A precedencia de interpolacao do Compose e **shell > `--env-file` > `.env`**. Logo, se a shell do
operador tiver `JWT_SECRET` exportado - de uma sessao anterior, de um `.bashrc`, de outro runbook -, o
Compose usa o **valor da shell** e ignora o `dev.env`: a rotacao pareceria funcionar e instalaria o
valor errado, sem erro. Obrigatorio, nas **duas** direcoes e tambem no `G5c`:
```bash
env -u JWT_SECRET -u ROTATING_JWT docker compose --project-directory <DIR> -p multica-dev-transition \
  -f <1> -f <2> -f <3> -f <4> -f <5> --env-file /home/ec2-user/.config/multica-transition/dev.env \
  up -d --force-recreate --no-deps backend
```
E uma pre-checagem barata, antes de qualquer coisa:
```bash
test -z "${JWT_SECRET+x}" || { echo E_JWT_IN_SHELL; exit 1; }
```

## B2 - **igualdade criptografica booleana** entre `AWSCURRENT` e a chave efetiva, sem `.Config.Env`

Substitui de vez o `docker inspect '{{json .Config.Env}}'`. A prova e feita **num unico filho**
`asm-exec`, comparando **digests HMAC com sal efemero**, e emitindo **uma palavra**:
```bash
SALT=$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')   # sal descartavel, so nesta janela
ROTATING_JWT='{{resolve:secretsmanager:<ARN>:SecretString:<JSON-KEY>:AWSCURRENT}}' \
ENVF_PATH=/home/ec2-user/.config/multica-transition/dev.env SALT="$SALT" \
asm-exec -- sh -eu -c '
  umask 077
  S=$ROTATING_JWT
  [ -n "$S" ]                            || { echo E_EMPTY;        exit 1; }
  case "$S" in *"{{resolve:"*)              echo E_UNRESOLVED;     exit 1 ;; esac
  case "$S" in *[!0-9a-fA-F]*)              echo E_NOT_HEX;        exit 1 ;; esac
  [ "$(printf %s "$S" | wc -c)" -eq 64 ] || { echo E_LEN;          exit 1; }
  # valor efetivo lido do arquivo duravel, KEY-ONLY, nunca impresso
  F=$(sed -n "s/^JWT_SECRET=\(.*\)$/\1/p" "$ENVF_PATH")
  [ "$(printf %s "$F" | wc -l)" -eq 0 ] || { echo E_MULTILINE;     exit 1; }
  A=$(printf %s "$S" | openssl dgst -sha256 -hmac "$SALT" -r | cut -d" " -f1)
  B=$(printf %s "$F" | openssl dgst -sha256 -hmac "$SALT" -r | cut -d" " -f1)
  [ "$A" = "$B" ] && echo EQ || echo NEQ'
```
- saida permitida: **`EQ`**, **`NEQ`** ou um `E_*`. Nunca valor, nunca digest, nunca comprimento de valor
  valido - o digest fica **dentro** do filho e o sal e descartado;
- **antes** do swap espera-se **`NEQ`** (o arquivo ainda tem a chave antiga); **depois** do swap,
  **`EQ`** prova que a fonte duravel e **exatamente** o `AWSCURRENT`. Essa inversao e o gate;
- ligacao com a chave **efetiva do container**, sem tocar ambiente: `G5b` de procedencia -
  label `com.multica.backend.env_file` == `dev.env`, `.Created` do container **posterior** ao `mtime` do
  arquivo, e `NEW_CID != OLD_CID`. Interpolacao vem do `--env-file` (B1.3), portanto arquivo + procedencia
  + `env -u` fecham a cadeia;
- `G5c` (opcional, `A7`) permanece como rotulo booleano em container **efemero**, tambem sob
  `env -u JWT_SECRET`;
- exige `openssl` presente no ambiente do filho - **a verificar** na captura `P0`, e **STOP** se ausente.

## B3 - gates **concretos** por consumidor

Fatos medidos de `internal/realtime/hub.go` que tornam o gate de WS concreto:
`/ws` exige `workspace_id` **ou** `workspace_slug` (senao `400`); se o cookie `multica_auth` estiver
presente, `authenticateToken` roda **antes** do upgrade e devolve **`401`** quando invalido (`:770-775`);
sem cookie, o upgrade ocorre e o **primeiro frame** deve ser
`{"type":"auth","payload":{"token":"…"}}` (`:719-728`), respondido com `{"type":"auth_ack"}` (`:813`).

| # | consumidor | caminho | gate **concreto** |
|---|---|---|---|
| **C1** | `GET /api/me` (`router.go:552`), cookie `multica_auth` (`cookie.go:20-21`), `Bearer` antes do cookie (`middleware/auth.go:329-341`) | JWT | pre: anonimo **`401`**, antigo **`200`**; pos: antigo **`401`**, novo **`200`**. Codigo **exato** via `-o /dev/null -w '%{http_code}'`; **`curl -f` proibido**; tokens em `--config` `0600` |
| **C2a** | `/ws` **por cookie** | JWT, pre-upgrade | `GET /ws?workspace_id=<ID>` com `--config`: pre antigo = **`101`**; pos antigo = **`401`**, novo = **`101`**. Nao exige cliente WebSocket - o `401` e emitido **antes** do upgrade |
| **C2b** | `/ws` **por primeiro frame**, com `auth_ack` | JWT, pos-upgrade | enviar `{"type":"auth","payload":{"token":…}}` e exigir `{"type":"auth_ack"}` com o token novo, e **erro/close** com o antigo. **Nao ha cliente WebSocket disponivel** (secao V8.9) -> **owner-run**, no navegador ou com cliente pinado sob autorizacao |
| **C3** | REST de daemon `/api/daemon/*`, `mdt_` -> `HashToken` + `GetDaemonTokenByHash` (`daemon_auth.go:107-147`) | hash em DB | **nao quebra**. Gate: heartbeat verde e runtimes online pos-swap. **Nenhuma** mutacao no ORQ2 |
| **C4** | WS de daemon `/api/daemon/ws` (`wakeup.go:343`, `router.go:516`, dentro de `DaemonAuth`), mesmo token do cliente (`wakeup.go:81-83`) | hash em DB | **nao quebra**; coberto por C3 |
| **C5** | PAT `mul_` (`daemon_auth.go:190`) e cloud `mcn_` (`:150`); no `/ws`, `mul_` e resolvido por `PATResolver` **antes** do JWT (`hub.go:676-684`) | hash / cloud | **nao quebram**. Gate: uma chamada autenticada por PAT devolvendo `200` pos-swap, com o PAT em `--config` `0600` |
| **C6** | fallback JWT de daemon (`daemon_auth.go:229-234`) | JWT | quebra **se** medido. **So aqui** o re-pareamento e exigido |

Caminho do daemon **medido**, nunca inferido: `E-1` log de heartbeat **lento** (o rotulo
`daemon_token|pat|cloud_pat|jwt` de `daemon_auth.go:27-30` so aparece via
`handler/daemon.go:747` -> `logHeartbeatEndpointSlow`; ausencia = **inconclusivo**) ou `E-2`, preferida,
`SELECT daemon_id, expires_at, created_at FROM daemon_token WHERE daemon_id='<id>'` - **colunas
nomeadas**, nunca a de hash, nunca `SELECT *`. Indeterminado = **STOP**.

## B4 - o login **consome stdin de verdade**, com `umask`/`O_NOFOLLOW`/`0700`/`0600`

A V7 dizia "por stdin" sem provar. Contrato da V8, **executado pelo owner**:
```bash
umask 077
TOKDIR=/home/ec2-user/.local/state/orq42-tokens
mkdir -m 0700 "$TOKDIR"                      # sem -p: falha se existir (nao reusa diretorio alheio)
test "$(stat -c %a "$TOKDIR")" = 700 && test "$(stat -c %U "$TOKDIR")" = ec2-user
# o corpo do login vai por STDIN, jamais em argv, jamais em arquivo:
printf '{"email":"%s","password":"%s"}' "$OWNER_EMAIL" "$OWNER_PASSWORD" |
  env -u OWNER_EMAIL -u OWNER_PASSWORD curl --silent --show-error --output /dev/null \
    --header 'Content-Type: application/json' --data-binary @- \
    --cookie-jar "$TOKDIR/new.jar" --write-out '%{http_code}' \
    http://127.0.0.1:18080/auth/login
test "$(stat -c %a "$TOKDIR/new.jar")" = 600
```
- `--data-binary @-` **e** a prova de consumo de stdin: sem `@-` o corpo iria para `argv`;
- `printf` deve ser **builtin** (`case "$(command -V printf)" in *builtin*)`), senao a senha vai ao
  `argv` do `/usr/bin/printf` - mesma trava exigida e aceita no ORQ-17 Stage3B;
- `ulimit -c 0` antes, para nao materializar segredo em core dump;
- **`O_NOFOLLOW`**: `curl` nao expoe a flag, portanto a protecao equivalente e **rejeitar symlink e
  reusar diretorio**: `mkdir -m 0700` sem `-p` (falha se existir) e, para cada arquivo,
  `test -f X && test ! -L X` antes de usar. Registro explicitamente que isso e **equivalencia
  operacional**, nao a syscall - nao afirmo `O_NOFOLLOW` onde nao ha;
- **o editor**, que e nosso codigo, **deve** abrir o `dev.env` com `O_NOFOLLOW` de verdade, e isso entra
  nas 8 assercoes de fonte da B1.2;
- os 4 arquivos (`old.jar`, `new.jar`, `old.cfg`, `new.cfg`) sao `0600`, regulares, nao-symlink, dono
  `ec2-user`; o **agente** verifica **apenas** `stat -c '%a %U'` e **codigo de saida**, e **nunca** le
  o conteudo;
- retencao obrigatoria **ate** o rollback ser impossivel (o `R4` precisa dos dois); limpeza como passo
  final autorizado, com recibo declarando os 4 caminhos e o `sha256` **do recibo**;
- `shred` recusado com fundamento: CoW e wear-leveling o tornam falsa garantia.

## B5 - editor byte-preserving por **prefixo/sufixo** + `cmp -s`, rollback com `fsync`/`rename`, e pinos

### B5.1 Editor (a construir, revisar e pinar)
```
1  open(dev.env, O_RDONLY|O_NOFOLLOW) em arquivo REGULAR; falha em symlink, erro de leitura ou byte NUL
2  localizar EXATAMENTE UMA linha ^JWT_SECRET=<64hex>$   (zero ou >1 = STOP_MULTI/STOP_NONE)
3  fatiar em PREFIXO (bytes antes da linha) e SUFIXO (bytes depois), guardando ambos INTACTOS
4  novo conteudo = PREFIXO + "JWT_SECRET=" + <valor de stdin> + SUFIXO
   -> comentarios, ordem, fins de linha (CRLF/LF) e newline terminal preservados por construcao
5  mktemp O_EXCL no MESMO diretorio, modo 0600, mesmo uid/gid; write; fsync(arquivo)
6  rename(2) sobre o destino; fsync(diretorio)   [rename da visibilidade atomica, NAO durabilidade]
7  trap EXIT/INT/TERM removendo o temporario
8  saida so em rotulos fixos: OK_EDIT | STOP_MULTI | STOP_NONE | STOP_READ | STOP_NUL | STOP_SYMLINK
```
### B5.2 Prova de que **so** a linha alvo mudou - `cmp -s`, sem imprimir nada
```bash
# antes: backup por mktemp no mesmo diretorio + igualdade mecanica
BK=$(umask 077 && mktemp "$(dirname "$ENVF")/.dev.env.bak.XXXXXX"); cat -- "$ENVF" >"$BK"; sync
cmp -s -- "$ENVF" "$BK" || { echo E_BACKUP; exit 1; }
# depois: prefixo e sufixo identicos, byte a byte, sem exibir conteudo
cmp -s <(grep -v '^JWT_SECRET=' -- "$BK") <(grep -v '^JWT_SECRET=' -- "$ENVF") \
  || { echo E_NON_TARGET_BYTES_CHANGED; exit 1; }
grep -c '^JWT_SECRET=' -- "$ENVF"                        # aceitacao: 1
grep -cE '^JWT_SECRET=[0-9a-fA-F]{64}$' -- "$ENVF"       # aceitacao: 1
grep -c '^JWT_SECRET={{resolve:' -- "$ENVF"              # aceitacao: 0
```
`cmp -s` e o gate mecanico: sem stdout, so codigo de saida. Nada de hash impresso, nada de `diff` com
conteudo.

### B5.3 Rollback simetrico
```bash
RT=$(umask 077 && mktemp "$(dirname "$ENVF")/.dev.env.restore.XXXXXX")
cat -- "$BK" >"$RT"; sync                                   # fsync do arquivo
cmp -s -- "$BK" "$RT" || { echo E_RESTORE_COPY; exit 1; }
mv -f -- "$RT" "$ENVF"; sync                                # rename(2) + fsync do diretorio
cmp -s -- "$ENVF" "$BK" || { echo E_RESTORE; exit 1; }
env -u JWT_SECRET "$HOME/.local/bin/multica-backend-env-rollback"   # helper pinado por hash E fonte
```
Depois: repetir `G5b`, inverter `C1` (antigo `200`, novo `401`), inverter `C2a` (antigo `101`, novo
`401`), e o `B2` volta a **`NEQ`**. `--no-recreate` **proibido** nas duas direcoes.

### B5.4 Pinos de identidade do segredo, incluindo **metadado de versao**
```text
ARN completo    arn:aws:secretsmanager:<REGIAO>:<CONTA>:secret:<NOME>-<SUFIXO>   (Q-B)
regiao          <REGIAO>                                                        (Q-B)
json-key        <JSON-KEY>                                                      (Q-B)
version-stage   AWSCURRENT  -- CONGELADO durante toda a janela                   (Q-B)
version-id      <VersionId de AWSCURRENT no inicio da janela>                    (Q-C, metadado)
metadado        VersionIdsToStages contendo AWSCURRENT e, se houver, AWSPREVIOUS  (Q-C)
```
Captura **somente metadado**, por identidade autorizada: `describe-secret` e
`list-secret-version-ids` (nenhuma e `get-secret-value`/`batch-get-secret-value`, logo **nao** violam a
**regra 1**), com saida reduzida a **booleanos e ao `VersionId`**. Se o `VersionId` do `AWSCURRENT`
**mudar** entre a captura e o fim da janela, **STOP**: alguem rotacionou por fora. Esta sessao ja levou
**`AccessDenied`** em `describe-secret` em `sa-east-1`, portanto **`Q-C` e bloqueante** e **eu nao
executei** nenhuma das duas.

## V8.6 - sequencia consolidada

```
0   test -z "${JWT_SECRET+x}"                                       [B1.3 - STOP se a shell tiver]
P0  allowlist absoluta + stat numerico + label env_file + key-only + hashes + openssl presente   [B1.1]
P0f leitura INTEGRAL de helpers e editor, 8 assercoes SIM/NAO                                    [B1.2]
Q-C metadado do segredo: ARN, regiao, key, stage congelado, VersionId                            [B5.4]
E-2 caminho de auth do daemon por metadado de DB                                                 [B3]
T   anonimo /api/me=401; antigo /api/me=200; antigo /ws?workspace_id=…=101                       [C1,C2a]
B2- igualdade criptografica: esperado NEQ (arquivo ainda tem a chave antiga)                      [B2]
BK  backup mktemp + cmp -s + sync                                                                [B5.2]
    -- fronteira: nada acima muta nada --
L2  asm-exec (env) + G1-G4 + editor pinado (prefixo/sufixo, O_NOFOLLOW, fsync, rename)            [B5.1]
V   cmp -s de prefixo/sufixo; key-only 1/1/0                                                     [B5.2]
B2+ igualdade criptografica: agora EQ                                                            [B2]
REC env -u JWT_SECRET + helper pinado (5 -f, --env-file, --no-deps backend)                       [B1.3]
P   NEW_CID != OLD_CID; label env_file == ENVF; .Created > mtime; /health 200; /readyz 200        [G5b]
A   antigo /api/me=401 e novo=200; antigo /ws=401 e novo=101                                     [C1,C2a]
Ack (owner-run) auth_ack no /ws por primeiro frame, com o token novo                             [C2b]
D   daemon conforme E-2; PAT 200; sem mutacao no ORQ2 se for mdt_                                [C3-C6]
R   rollback B5.3 com tokens retidos; B2 volta a NEQ                                             [B5.3]
```

## V8.7 - perguntas

| # | pergunta | estado |
|---|---|---|
| **Q-A** | helpers ORQ-30 como via unica, agora condicionada a **fonte integral** (B1.2) | ABERTA |
| **Q-B** | ARN completo, regiao, `json-key`, estagio congelado | ABERTA |
| **Q-C** | metadado por identidade autorizada + `VersionId` inicial (esta sessao levou `AccessDenied`) | **ABERTA/BLOQUEANTE** |
| **Q-D** | atestacao de **64 hex** sem revelar valor | ABERTA (contrato ok) |
| **Q-E** | custodia e obtencao de token - contrato completo em **B4**; falta confirmacao | PROPOSTA |
| **Q-F** | re-pareamento **condicional** a `E-2` | ABERTA |
| **Q-G** | aceite **e comunicacao** da queda de sessoes e do realtime `/ws` | ABERTA |
| **Q-H** | **nova**: quem constroi, revisa e pina o **editor**, que ainda **nao existe**? | **ABERTA/BLOQUEANTE** |
| **Q-I** | **nova**: quem executa o `C2b` (`auth_ack`), dado que **nao ha cliente WebSocket** no ORQ2? | ABERTA |

## V8.8 - autorizacoes **explicitas** (9)

| # | autorizacao | escopo exato |
|---|---|---|
| **A1** | AWS **somente metadado** | `describe-secret`, `list-secret-version-ids`. Mutacao de segredo **fora de escopo** |
| **A2** | captura `P0` no ORQ1 | leituras de metadado + **leitura integral de fonte** dos helpers/editor |
| **A3** | **`asm-exec`** | uma unica resolucao por janela, referencia **no ambiente**, nunca em `argv` |
| **A4** | **login do owner** | `POST /auth/login` por **stdin**, criacao de `TOKDIR` e dos 4 arquivos `0600` |
| **A5** | residual `/proc/<pid>/environ` | aceite explicito do owner |
| **A6** | edicao do `dev.env` + `force-recreate` do `backend` | sob `env -u JWT_SECRET` |
| **A7** | rollback **pre**-autorizado | com **retencao** dos tokens ate a aceitacao |
| **A8** | **leitura de DB** para `E-2` | `SELECT` de **colunas nomeadas** em `daemon_token`; nunca a de hash, nunca `SELECT *` |
| **A9** | **comunicacao** da janela aos usuarios | C1 e C2 caem; sem isso, nao rotacionar |
| **A10** | `G5c` opcional | container efemero de assercao booleana |

## V8.9 - nao-afirmacoes

- **Nesta rodada nao mutei nada e nao toquei o ORQ1.** As unicas execucoes foram leituras locais
  nao-secretas: `internal/realtime/hub.go` (para os fatos de `/ws` e `auth_ack`), `cmd/server/router.go`
  e uma checagem de `PATH`.
- **Medi que nao existe cliente WebSocket no ORQ2**: `websocat`, `wscat` e `wsdump.py` ausentes, e o
  Python 3.9.25 nao tem `websockets` nem `websocket` (`ModuleNotFoundError`). Logo o `C2b`/`auth_ack`
  **nao** e executavel aqui - dai a `Q-I`. **Nao** proponho instalar nada.
- **Nao verifiquei se `openssl` existe** no host/container onde o `B2` rodaria; virou item de `P0` com
  **STOP** se ausente.
- **Nao li o fonte dos helpers, nao verifiquei seus hashes, e o editor nao existe.** `P0`/`P0f` sao
  gates propostos; o `sha256` do editor esta **vazio** de proposito.
- **Nao reli as labels nem os 5 `config_files` no ORQ1** nesta rodada: a allowlist vem da evidencia
  preservada do Stage 2 do ORQ-17 e **precisa** ser reconfirmada por `P0`.
- Sobre `O_NOFOLLOW`: **nao afirmo** que `curl`/`mkdir` fazem `O_NOFOLLOW`. Descrevi **equivalencia
  operacional** (rejeitar symlink, nao reusar diretorio) e reservei a syscall real para o **editor**,
  que e nosso codigo.
- `B2` prova igualdade entre `AWSCURRENT` e o **arquivo duravel**; a ligacao com a chave **efetiva do
  container** vem de `G5b` (procedencia + ordem temporal) mais `--env-file` mais `env -u`, **nao** de
  leitura de ambiente. Se o revisor considerar isso insuficiente, o unico caminho restante e o `G5c`.
- Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA, zero `asm-exec` executado, zero AWS, zero
  Docker/Compose, zero leitura de `dev.env`, zero `.Config.Env`, zero token, zero DB, zero quadro.
- **Nao adivinho nome de segredo**; `Q-B` fica com placeholders.
- Nao alterei assignee, nao postei comentario, nao criei card.

---

**PROPOSTA - requer re-review.** Ataques sugeridos: (a) se `B2` (`EQ` pos-swap, `NEQ` pre-swap) mais
`G5b` fecham a cadeia ate a chave **efetiva** sem `Config.Env`; (b) se as 8 assercoes de fonte da `B1.2`
sao suficientes para autorizar o helper; (c) se o `C2a` por cookie realmente distingue `401` pre-upgrade
de falha de transporte; (d) se o contrato de `O_NOFOLLOW` por equivalencia e aceitavel fora do editor.
