# ORQ-42 - Runbook V5 de rotacao do segredo JWT no ORQ1 (PROPOSTA, READ-ONLY)

- **Status: PROPOSTA.** Nao aprovado, **nao executavel** sem as autorizacoes da secao 9.
  **Nao me auto-aprovo**: peco revisao independente por agente que nao seja eu (Opus48#A) nem o autor
  da V4 (Antigravity) nem o revisor da V4.
- autor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T16:20Z
- substitui: V3 (`orq33-jwt-rotation-remediation-v3-design.md`) e V4
  (`orq42-jwt-rotation-runbook-v4-design.md`)
- reconcilia: meu parecer `orq42-jwt-v3-adversarial-peer-review.md` (BLOCK) e o parecer
  `orq42-jwt-rotation-runbook-v4-adversarial-peer-review.md` (BLOCK, F1-F13)
- skill carregada **integralmente** antes de escrever: `.agents/skills/aws-secrets-manager/SKILL.md`
  v1 - overview, aviso *"best-effort defense, not a security boundary"*, as **3 regras MUST**, sintaxe
  `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}` com defaults
  `SecretString`/`AWSCURRENT`, `Using asm-exec`, `How It Works` (scan de **argumentos**, ordem
  SMA -> MCP SigV4, regiao por ARN ou `AWS_REGION`, `re.sub` single-pass anti re-scan, sem fallback
  para CLI local), SigV4, prerequisitos, **Common Patterns** incluindo *Configuration file
  templating*, hook `PreToolUse` do `aws-core` e troubleshooting.
- modo desta redacao: **READ-ONLY**. Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA em
  `localhost:2773`, **zero `asm-exec` executado**, zero dump de `.Config.Env`, zero valor de segredo,
  zero mutacao em AWS/Docker/container/unit, zero alteracao de quadro, zero login, zero restart.

---

## 0. A correcao central da V5: a V4 rotacionaria de forma **nao duravel**

A V4 propoe passar um `--env-file` **temporario** ao Compose. Reconciliando com o estado pos-ORQ-30
(F13 do parecer da V4), isso e defeituoso de forma demonstravel:

1. O ORQ-30 estabeleceu, por decisao escrita do owner (opcao 1, arquivo no host), a fonte **duravel**
   `/home/ec2-user/.config/multica-transition/dev.env` (`0600`, `ec2-user:ec2-user`, pai `0700`), mais
   `backend-env.override.yml` (`0600`) que declara `services.backend.env_file` **sem** literais de
   segredo, mais dois helpers `0700` com SHA-256 registrado
   (`multica-backend-recreate fa1dae20...`, `multica-backend-env-rollback 0e42c65e...`), e a label
   `com.multica.backend.env_file` apontando ao arquivo.
2. O compose base declara o segredo em `services.backend.environment` como
   `JWT_SECRET: ${JWT_SECRET:?JWT_SECRET must be set to a generated value of at least 32 bytes}`
   (`multica-auth-work/docker-compose.selfhost.yml:58`). Em Compose, `environment` tem **precedencia
   sobre** `env_file`, e o valor vem da **interpolacao** `${JWT_SECRET}`, cuja fonte e o `--env-file`
   (ou `.env`), nao o `env_file:` do servico.
3. Logo, com o `--env-file` temporario da V4 o container **passaria** a ter o segredo novo, mas o
   `dev.env` duravel continuaria com o **antigo**. Na proxima execucao do helper
   `multica-backend-recreate` - que e o caminho autorizado e existente - o backend **reverteria
   silenciosamente** ao segredo antigo, invalidando todas as sessoes emitidas depois da rotacao, sem
   nenhum sinal de erro.

**V5 rotaciona a fonte duravel, nao um arquivo efemero.** O unico artefato que muda e o `dev.env`,
substituido **atomicamente**, com backup para rollback simetrico. Isso tambem elimina a objecao F13
(duas fontes concorrentes de `JWT_SECRET`): passa a existir **uma** fonte.

> **Pergunta obrigatoria ao owner (Q-A)**: confirmar que a rotacao deve usar os artefatos ORQ-30
> existentes (helpers + `dev.env`) e **nao** supersede-los. Se o owner preferir supersedimento, a V5
> precisa ser reescrita: os helpers tem hash registrado e nao podem ser silenciosamente contornados.

---

## 1. Premissas que **nao** estao provadas e bloqueiam a execucao

| # | pergunta aberta | por que bloqueia | quem responde |
|---|---|---|---|
| **Q-A** | usar os artefatos ORQ-30 ou supersede-los? | define todo o mecanismo (secao 0) | owner |
| **Q-B** | qual o **secret-id exato**, `json-key` e `version-stage` da referencia dinamica? | a V4 chuta `prod/jwt-secret`; eu **nao** vou adivinhar nome de segredo | owner |
| **Q-C** | presenca do segredo confirmada por identidade autorizada, **somente metadados** | `DescribeSecret` em `sa-east-1` retornou `AccessDenied` para esta identidade (registrado no parecer da V4). Sem confirmacao, falhar fechado | owner / identidade AWS autorizada |
| **Q-D** | o valor novo e **hex de 64 caracteres**? | e o contrato de caracteres que torna a escrita em `env_file` segura (secao 5.3) e mantem paridade com o comprimento **64** medido no ORQ-30 | owner (atestacao **sem** revelar valor) |
| **Q-E** | como o owner obtem o **token novo** pos-rotacao sem expo-lo a agente? | o gate `novo=200` depende disso (secao 7.3) | owner |
| **Q-F** | autorizacao e card proprio para o **re-pareamento do daemon no ORQ2** | pre-condicao dura da mesma janela (secao 8) | owner |

`DescribeSecret` **nao** prova que uma `json-key` existe. Isso exige atestacao do owner, sem divulgar
valor. Eu **nao** executei nenhuma dessas chamadas.

## 2. Preflight - derivar tudo das labels **no momento da execucao** (P1-P8)

Nada de caminho ou nome de projeto hardcoded. Tudo sai do container em execucao. Todos os comandos
abaixo sao **leitura**; nenhum toca `.Config.Env` exceto o G3 da secao 7.4, que emite so codigo de
saida.

```bash
# P1 - plugin Compose obrigatorio; a forma hifenizada nao existe no ORQ1
docker compose version || { echo "STOP: plugin docker compose ausente"; exit 1; }
command -v docker-compose >/dev/null && echo "AVISO: docker-compose v1 presente; NAO usar"

# P2 - identificar UNICO container do servico backend, sem assumir projeto
CID=$(docker ps -q --filter 'label=com.docker.compose.service=backend')
test "$(printf %s "$CID" | wc -w)" -eq 1 || { echo "STOP: nao ha exatamente 1 backend"; exit 1; }

# P3 - derivar projeto, working_dir e config_files SO das labels
PROJ=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project"}}' "$CID")
WDIR=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project.working_dir"}}' "$CID")
CFGS=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.project.config_files"}}' "$CID")
echo "projeto=$PROJ"; echo "working_dir=$WDIR"; echo "config_files=$CFGS"

# P4 - montar os -f na ORDEM EXATA da label, sem inventar nem omitir arquivo
set --
OLDIFS=$IFS; IFS=,
for f in $CFGS; do
  test -f "$f" || { echo "STOP: config_file inexistente: $f"; exit 1; }
  set -- "$@" -f "$f"
done
IFS=$OLDIFS
echo "argumentos -f derivados: $*"

# P5 - derivar o arquivo env duravel da label do ORQ-30 (nunca hardcode, nunca ler conteudo)
ENVF=$(docker inspect -f '{{index .Config.Labels "com.multica.backend.env_file"}}' "$CID")
test -n "$ENVF" || { echo "STOP: label com.multica.backend.env_file ausente"; exit 1; }
test -f "$ENVF" && test ! -L "$ENVF" || { echo "STOP: env_file ausente ou e symlink"; exit 1; }
stat -c '%n modo=%a dono=%U:%G' "$ENVF"                 # exigir 600 e ec2-user:ec2-user
stat -c '%n modo=%a dono=%U:%G' "$(dirname "$ENVF")"    # exigir 700 e ec2-user:ec2-user

# P6 - EFEITO: o bypass de auth local NAO pode estar ativo, senao o gate 401 e invalido
code=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 3 --max-time 10 \
  http://127.0.0.1:18080/api/me)
test "$code" = "401" || { echo "STOP: /api/me sem credencial devolveu $code, esperado 401"; exit 1; }

# P7 - estado saudavel ANTES de mexer em nada
docker inspect -f 'status={{.State.Status}} restarts={{.RestartCount}} imagem={{.Image}}' "$CID"
for p in /health /readyz; do
  echo "$p -> $(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 3 --max-time 10 \
    "http://127.0.0.1:18080$p")"
done   # ambos DEVEM ser exatamente 200

# P8 - helpers ORQ-30 intactos (hash registrado)
sha256sum "$HOME/.local/bin/multica-backend-recreate" "$HOME/.local/bin/multica-backend-env-rollback"
```

Aceitacao do preflight: **P1..P8 todos verdes**. `P6` e a guarda que nenhuma versao anterior tinha:
`internal/middleware/auth.go:38-55` desativa o bypass de auth local automaticamente quando
`FRONTEND_ORIGIN` nao e loopback - mas se o bypass **estiver** ativo, `/api/me` responde `200` sem
token e o gate `antigo=401` da secao 7.3 seria **falso positivo garantido**. `P6` prova o contrario
por **efeito**, sem ler nenhuma variavel de ambiente.

## 3. Por que `force-recreate` e nao `restart`

`internal/auth/jwt.go:25-40` guarda o segredo atras de `jwtSecretOnce sync.Once`: o valor e lido uma
vez por processo. Um `docker restart` cria processo novo, mas **nao** reprocessa o ambiente do
container - o `Config.Env` e imutavel apos a criacao. Portanto **so** `up -d --force-recreate` aplica
o segredo novo. E `--no-deps backend` garante que Postgres, web e omniroute nao sao tocados.

`internal/auth/jwt.go:46-58 ValidateJWTConfiguration` exige >= 32 bytes fora de
`APP_ENV in {dev,development,test}`. A guarda de comprimento da secao 5.3 e mais estrita (64 hex).

## 4. Consumidores afetados - raio de impacto declarado

Os 5 consumidores de `JWTSecret()` sao: `handler/auth.go:213,226` (assinam), `middleware/auth.go:295`,
`middleware/daemon_auth.go:234` e `realtime/hub.go:691` (verificam). Consequencias inevitaveis da
rotacao, que o owner precisa aceitar **antes**:

- **todas** as sessoes humanas caem (`401`) e exigem novo login;
- **todos** os WebSocket/realtime caem na proxima verificacao;
- **os tokens `mdt_` do daemon param de validar** -> daemon do ORQ2 desautenticado. Por isso o
  re-pareamento e pre-condicao da **mesma janela** (secao 8). O ORQ1 **nao** tem daemon; a unit e
  `multica-daemon-orq2-credential.service`, no ORQ2.

## 5. Passo forward - um filho `asm-exec` curto que so troca a fonte duravel

### 5.1 Fronteira de processo, explicita
O pai **jamais** captura valor. O pai so conhece: a **referencia literal**, caminhos e palavras de
status. **Proibido** em qualquer circunstancia: `X=$(asm-exec ...)`, `export`, `echo` do valor,
`docker compose config` (que **imprime o segredo interpolado**), `--no-interpolate` como se fosse
seguro, e qualquer redirecionamento de `.Config.Env` para arquivo ou stdout.

### 5.2 Backup para rollback simetrico (sem `asm-exec`, sem ler conteudo)
```bash
BK="$ENVF.bak.$(date -u +%Y%m%dT%H%M%SZ)"
(umask 077 && cp -p -- "$ENVF" "$BK")
stat -c '%n modo=%a dono=%U:%G' "$BK"     # exigir 600
sha256sum "$ENVF" "$BK"                    # hashes iguais; hash NAO revela valor
```
`cp -p` nao imprime conteudo. O backup fica no **mesmo diretorio `0700`**, portanto nao amplia a
classe de exposicao - o segredo antigo ja residia ali. **Nao** proponho `shred`: em sistema de
arquivos com CoW e SSD com wear-leveling ele daria falsa garantia; a protecao real e o `0600` no
diretorio `0700` e a remocao apos a janela por decisao do owner.

### 5.3 O filho unico (curto, com todas as guardas dentro)
Substituir `<SECRET-ID>`, `<JSON-KEY>` e `<VERSION-STAGE>` pelos valores que o owner declarar em
**Q-B**. O `sh -c` recebe o script como **argumento**, e e nele que o `asm-exec` faz a substituicao
(padrao *Configuration file templating* da skill).

```bash
asm-exec -- sh -c '
set -eu
umask 077
S="{{resolve:secretsmanager:<SECRET-ID>:SecretString:<JSON-KEY>:<VERSION-STAGE>}}"

# G1 nao vazio
[ -n "$S" ] || { echo "FAIL_EMPTY"; exit 1; }

# G2 placeholder literal em QUALQUER posicao (nao apenas prefixo)
case "$S" in *"{{resolve:"*) echo "FAIL_UNRESOLVED"; exit 1 ;; esac

# G3 contrato de caracteres: hex puro -> seguro para env_file e para interpolacao
case "$S" in *[!0-9a-fA-F]*) echo "FAIL_NOT_HEX"; exit 1 ;; esac

# G4 comprimento exato, sem contar quebra de linha
L=$(printf %s "$S" | wc -c)
[ "$L" -eq 64 ] || { echo "FAIL_LEN=$L"; exit 1; }

# escrita atomica no MESMO diretorio, preservando modo, com trap
D=$(dirname "$ENVF_PATH")
T=$(mktemp "$D/.dev.env.new.XXXXXX")
trap "rm -f \"$T\"; exit 1" INT TERM
chmod 0600 "$T"
grep -v "^JWT_SECRET=" "$ENVF_PATH" > "$T" || true
printf "JWT_SECRET=%s\n" "$S" >> "$T"
mv -f -- "$T" "$ENVF_PATH"
trap - INT TERM
echo "OK_ROTATED"
' 
```
Notas de execucao, todas obrigatorias:
- `ENVF_PATH` e exportado pelo **pai** com o caminho derivado em `P5` - e caminho, nao segredo;
- a unica saida possivel do filho e uma palavra de status (`OK_ROTATED`, `FAIL_EMPTY`,
  `FAIL_UNRESOLVED`, `FAIL_NOT_HEX`, `FAIL_LEN=<n>`). **O valor nunca e impresso**, nem o comprimento
  de um valor valido;
- `grep -v "^JWT_SECRET="` preserva as demais chaves do `dev.env`. Ele **nao** imprime nada em stdout,
  escreve so no temporario `0600`;
- `mv` no mesmo diretorio e `rename(2)`: leitor concorrente ve o arquivo antigo **ou** o novo, nunca
  truncado;
- qualquer `FAIL_*` = **STOP**, e como o `mv` ainda nao ocorreu, o `dev.env` segue intacto: a falha e
  atomica e nao precisa de rollback.

### 5.4 Recriar o backend - **sem segredo em argv**
Porque a fonte e duravel, o comando de recriacao nao carrega nenhum valor:
```bash
docker compose -p "$PROJ" "$@" up -d --force-recreate --no-deps backend
```
com `"$@"` sendo exatamente os `-f` derivados em `P4`, na ordem da label. Se `${JWT_SECRET}` nao
estiver disponivel para interpolacao, o `:?` de `docker-compose.selfhost.yml:58` **falha ruidosamente**
- isso e desejado e e a razao de o `--env-file`/`.env` efetivo precisar continuar sendo o `dev.env`
derivado, conforme o helper ORQ-30 ja faz.

> Se o owner responder em **Q-A** que os helpers ORQ-30 sao a via autorizada, substituir 5.4 por
> `"$HOME/.local/bin/multica-backend-recreate"`, **apos** conferir o SHA-256 em `P8`. Prefiro essa
> variante: reusa caminho ja auditado em vez de abrir um segundo.

## 6. Gates pos-forward (G1-G5) - codigo HTTP exato, nunca `curl -f`

`curl -f` **nao** distingue `401` esperado de falha de transporte, e por isso esta proibido nos gates.

```bash
# G1 liveness e readiness sao rotas DIFERENTES (router.go:443-445)
for p in /health /readyz; do
  printf '%s -> %s\n' "$p" "$(curl -s -o /dev/null -w '%{http_code}' \
    --connect-timeout 3 --max-time 10 --retry 0 "http://127.0.0.1:18080$p")"
done   # exigir exatamente 200 em ambos, com poll limitado (ate 10 tentativas, 3s)

# G2 container recriado e estavel
docker inspect -f 'status={{.State.Status}} restarts={{.RestartCount}} criado={{.Created}}' \
  "$(docker ps -q --filter 'label=com.docker.compose.service=backend')"
```

### 6.1 Gate de autenticacao - cookie correto e token fora de argv
O nome do cookie e **`multica_auth`** (`internal/auth/cookie.go:20-21`), **nao** `session` como a V4
escreveu. A precedencia e `Authorization: Bearer` **antes** do cookie
(`internal/middleware/auth.go:329-341`). O endpoint e `GET /api/me`
(`cmd/server/router.go:552`, dentro do grupo autenticado).

Para nao colocar token em `argv` (visivel em `/proc/<pid>/cmdline`), usar arquivo de config do curl
`0600` criado pelo **owner**, nunca por agente:
```bash
# arquivo criado pelo owner, modo 0600, em diretorio 0700; agente nao le nem cria
# conteudo: header = "Cookie: multica_auth=<TOKEN>"
old=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 3 --max-time 10 \
  --config "$CFG_OLD" http://127.0.0.1:18080/api/me)
new=$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 3 --max-time 10 \
  --config "$CFG_NEW" http://127.0.0.1:18080/api/me)
test "$old" = "401" || { echo "STOP: token antigo devolveu $old, esperado 401"; exit 1; }
test "$new" = "200" || { echo "STOP: token novo devolveu $new, esperado 200"; exit 1; }
rm -f -- "$CFG_OLD" "$CFG_NEW"
```
- **G3 = `antigo` exatamente `401`** (prova de invalidacao);
- **G4 = `novo` exatamente `200`** (prova de emissao com a chave nova). Depende de **Q-E**;
- `P6` ja provou que um `/api/me` sem credencial devolve `401`, logo o `401` de G3 nao pode ser
  confundido com bypass desligado devolvendo 401 para tudo... e por isso G4 e **obrigatorio**: sem
  ele, `401`+`401` seria indistinguivel de backend quebrado.

### 6.2 G5 - guarda de placeholder por efeito, minimizada
Por que ela e necessaria e **nao** redundante com G1-G4: se o literal `{{resolve:...}}` tivesse sido
gravado, ele teria mais de 32 bytes, passaria `ValidateJWTConfiguration`, e o backend assinaria e
verificaria tokens **consistentemente** - `antigo=401` e `novo=200` **passariam**. Logo o defeito e
invisivel por HTTP e precisa de checagem propria.
```bash
docker inspect -f '{{json .Config.Env}}' "$CID" | grep -qF 'resolve:secretsmanager'; echo "rc=$?"
# aceitacao: rc=1 (nao encontrado)
```
Restricoes duras: a saida **unica** permitida e `rc=`; e **proibido** `echo` do inspect, `tee`,
redirecionar para arquivo, ou usar `grep` sem `-q`. Este e o **unico** contato autorizado com
`.Config.Env` em todo o runbook, ele nao imprime entrada nenhuma, e ainda assim **requer aprovacao
explicita do owner** por tocar o ambiente do container. Eu **nao** o executei.

## 7. Rollback simetrico (R1-R5)

Mesmos `-f` derivados, mesmo `-p`, mesmas flags, mesmos gates. Nenhuma acao improvisada.
```bash
# R1 restaurar a fonte duravel, atomicamente
(umask 077 && cp -p -- "$BK" "$ENVF.restore.tmp") && mv -f -- "$ENVF.restore.tmp" "$ENVF"
stat -c '%n modo=%a dono=%U:%G' "$ENVF"; sha256sum "$ENVF" "$BK"   # hashes DEVEM coincidir

# R2 recriar SO o backend, mesmos argumentos do forward
docker compose -p "$PROJ" "$@" up -d --force-recreate --no-deps backend
#   ou "$HOME/.local/bin/multica-backend-env-rollback" se Q-A escolher os helpers ORQ-30

# R3 repetir G1 e G2 (liveness 200, readiness 200, container estavel)
# R4 inverter o gate de auth: o token ANTIGO volta a 200; o token novo passa a 401
# R5 repetir G5 (rc=1)
```
Proibido `--no-recreate` em qualquer direcao - foi exatamente o defeito que reprovei na V3: ele
**impede** que o container assuma o ambiente restaurado, tornando o rollback inerte.

Qualquer divergencia em R1-R5 e **STOP** com escalonamento ao owner, nunca uma segunda tentativa
improvisada.

## 8. Pre-condicao dura: re-pareamento do daemon no ORQ2, na mesma janela

Como `middleware/daemon_auth.go:234` verifica o token do daemon com o **mesmo** segredo JWT, a rotacao
desautentica o daemon do ORQ2. Exigencias, **antes** de tocar o segredo:

1. card `ORQ-N` **proprio, ja criado e verificado** (POST 201 devolvendo UUID+numero e GET pelo UUID
   confirmando o par) - **nunca** um `ORQ-N` previsto. Eu **nao** sou o registrador e **nao** crio card;
2. autorizacao escrita do owner, separada da autorizacao do ORQ-42;
3. operador nomeado, procedimento de re-pareamento **pronto e revisado** antes da janela;
4. criterio de aceitacao: daemon autenticado, heartbeat verde e os runtimes obrigatorios online;
5. rollback do re-pareamento e condicao de STOP escritos.

Autorizacao do ORQ-42 **nao** implica autorizacao de acao no ORQ2. Se o item 3 nao estiver pronto, a
rotacao **nao comeca**.

## 9. Autorizacoes necessarias (4, separadas)

| # | escopo | sem ela |
|---|---|---|
| A1 | mutacao/leitura de metadados do segredo na AWS por identidade autorizada (**Q-C**) | falhar fechado |
| A2 | escrita no `dev.env` do ORQ1 e `force-recreate` do backend | nao executar 5.3/5.4 |
| A3 | rollback (secao 7) pre-autorizado, para nao precisar de nova decisao sob pressao | STOP sem saida |
| A4 | re-pareamento do daemon no ORQ2, com card proprio (secao 8) | rotacao nao comeca |

## 10. FILES_LOCKED e escopo

- `/home/ec2-user/.config/multica-transition/dev.env` (**unico arquivo mutado**, atomicamente)
- `/home/ec2-user/.config/multica-transition/backend-env.override.yml` - **LOCKED, nao tocar**
- `$HOME/.local/bin/multica-backend-recreate` e `multica-backend-env-rollback` - **LOCKED**, so
  execucao apos conferencia de hash
- os arquivos de `config_files` derivados da label - **LOCKED, somente leitura**
- `multica-auth-work/server/internal/auth/jwt.go` - **LOCKED**, nenhuma mudanca de codigo nesta carta
- nenhuma migration, nenhum arquivo gerado, nenhuma alteracao de quadro

## 11. Mapa de rastreabilidade dos 13 achados da V4

| achado | tratamento na V5 |
|---|---|
| F1 plugin | `P1` exige `docker compose version` e alerta se `docker-compose` v1 existir |
| F2 quatro caminhos | **superado**: `P3`/`P4` derivam `config_files` da label no momento da execucao, na ordem exata, sem hardcode; nao leio `backend-env.override.yml` |
| F3 projeto/servico | `P2`/`P3` derivam projeto e servico das labels; `--no-deps backend` mantido |
| F4 forward vs rollback | secoes 5 e 7, simetricas, mesmos `-f`/`-p`/flags, gates invertidos, `--no-recreate` proibido |
| F5 sem texto claro no pai | 5.1 proibe explicitamente substituicao de comando, `export`, `docker compose config`; o filho e **um** e curto, e a recriacao **nao** precisa do valor |
| F6 comprimento no filho | `G4` no filho, `printf %s`, emite so status; exige 64 |
| F7 anti-literal por efeito | `G2` no filho (qualquer posicao) **mais** `G5` por efeito, com justificativa de por que HTTP nao detecta |
| F8 temp privado/trap | 5.2 e 5.3: `umask 077`, `0600`, rejeicao de symlink em `P5`, trap que **sai**, `mv` atomico, e contrato de caracteres **hex** resolvendo o problema de bytes arbitrarios |
| F9 health/readiness | `G1` com `/health` **e** `/readyz`, `127.0.0.1:18080`, codigo exato, timeouts, poll limitado, sem `-f` |
| F10 `/api/me` | cookie `multica_auth`, precedencia Bearer documentada, `--config` `0600` fora de argv, `antigo=401`/`novo=200` exatos, mais o novo `P6` |
| F11 presenca do segredo | **Q-B**/**Q-C** abertas; falha fechada; `DescribeSecret` nao prova `json-key` |
| F12 re-pareamento ORQ2 | secao 8, pre-condicao **dura** com card ja verificado, nao futuro |
| F13 reconciliacao ORQ-30 | secao 0 - e a correcao central; **uma** fonte duravel, helpers com hash, `Q-A` ao owner |

## 12. Nao-afirmacoes

- **Nao li nenhum valor de segredo.** Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA em
  `localhost:2773`, **zero `asm-exec` executado**, zero `docker compose config`, zero dump de
  `.Config.Env`.
- **Nenhuma mutacao**: nao escrevi em `dev.env`, nao recriei container, nao reiniciei unit, nao logei,
  nao rotacionei, nao alterei quadro, nao criei card.
- **Nao executei o preflight**: `P1`-`P8` sao propostas. Nao reli as labels do ORQ1 nesta sessao - a
  V5 foi escrita justamente para **nao** depender de snapshot, derivando tudo na execucao.
- **Exposicao residual que assumo e declaro**: o `asm-exec` substitui a referencia nos **argumentos**
  do filho, portanto o valor resolvido existe brevemente no `argv` do `sh` filho, legivel em
  `/proc/<pid>/cmdline` por processo do mesmo usuario. Isso e inerente ao desenho do `asm-exec` e a
  propria skill se declara *best-effort defense, not a security boundary*. Mitigacao adotada: o filho
  e **curto** e a recriacao do Compose fica **fora** dele. Nao ha, com as ferramentas disponiveis,
  forma de eliminar essa janela - o owner precisa aceita-la explicitamente.
- `P6` deriva a inatividade do bypass por **efeito**; nao inspecionei `FRONTEND_ORIGIN` nem qualquer
  variavel.
- O comprimento **64** e a paridade com `dev.env` vem da evidencia ORQ-30 preservada, nao de medicao
  minha nesta sessao.
- Nao verifiquei se a `json-key` existe, nem o nome do segredo. **Nao adivinho nome de segredo.**
- Nao confirmei o UUID do cartao ORQ-42 e **nao** o predigo.
- Nao alterei assignee nem postei comentario em issue.

---

**PROPOSTA - requer revisao independente.** Nao aprovo meu proprio desenho. O revisor deve atacar
prioritariamente: (a) a afirmacao de precedencia `environment` > `env_file` da secao 0, que e o eixo
da V5; (b) se `grep -v` + `printf` preserva fielmente o resto do `dev.env`; (c) se `G5` e aceitavel
sob a proibicao de dump de `.Config.Env`; (d) se a janela de `argv` do filho e tolerada pelo owner.
