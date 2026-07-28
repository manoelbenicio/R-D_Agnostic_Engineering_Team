# ORQ-42 - Runbook V6: correcao dos bloqueios B1-B9 (PROPOSTA, READ-ONLY)

- **Status: PROPOSTA.** Nao aprovada, **nao executavel**. **Nao me auto-aprovo** - peco nova revisao
  independente por agente que nao seja eu (autor da V5/V6), nem o autor da V4, nem o revisor da V4.
- autor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T16:55Z
- card: **ORQ-42**, UUID `64bfcae0-b867-4812-ad33-ae03ef7f25ae` (citado pelo revisor; **nao verifiquei
  por GET** e nao predigo numero)
- substitui: V5 (`orq42-jwt-rotation-runbook-v5-design.md`, sha256 `2504cc3b…`)
- fecha: `orq42-jwt-rotation-runbook-v5-independent-adversarial-review.md`, bloqueios **B1-B9**
- skill carregada **integralmente** nesta rodada: `.agents/skills/aws-secrets-manager/SKILL.md` v1 -
  overview, aviso *"best-effort defense, not a security boundary"*, as **3 regras MUST**, sintaxe
  `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}` com defaults
  `SecretString`/`AWSCURRENT`, `Using asm-exec` (**"resolves `{{resolve:...}}` references in command
  arguments and environment variables"**), `How It Works` (ordem SMA -> MCP SigV4, regiao por ARN ou
  `AWS_REGION`, `re.sub` single-pass, **sem** fallback para CLI local), SigV4, prerequisitos, Common
  Patterns incluindo *Configuration file templating*, hook `PreToolUse` do `aws-core` e troubleshooting.
- modo: **READ-ONLY**. Zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA, **zero `asm-exec`
  executado**, zero Docker/Compose, zero AWS, zero leitura ou escrita de `dev.env`, zero
  `.Config.Env`, zero token emitido ou testado, zero acao em daemon/unit/DB/provider, zero quadro.

---

## V6.0 - Aceito o BLOCK. E corrijo um erro **meu**, de fato, na V5

O revisor esta certo em **B8**, e minha V5 afirmou algo **factualmente falso**. Verifiquei eu mesmo, no
codigo, nesta rodada:

`internal/middleware/daemon_auth.go` trata os prefixos **nesta ordem**:
1. **`mdt_`** (`:107`) -> `auth.HashToken(tokenString)` (`:108`) -> cache (`:110`) ou
   `queries.GetDaemonTokenByHash` (`:122`); marca o contexto com
   `DaemonAuthPathDaemonToken` (`:113`, `:143`). **Nao chama `JWTSecret()` em nenhum ponto.**
2. **`mcn_`** (`:150`) -> verificador cloud;
3. **`mul_`** (`:190`) -> `auth.HashToken` + PAT;
4. **somente o fallback final** (`:229-234`, comentario literal `// Fallback: JWT tokens.`) chama
   `auth.JWTSecret()`.

Portanto **rotacionar o `JWT_SECRET` nao invalida um token `mdt_` real**. A afirmacao da V5 (secoes 4 e
8) de que "os tokens `mdt_` param de validar" estava **errada**, e a exigencia incondicional de
re-pareamento do ORQ2 estava mal fundamentada. Registro isso explicitamente em vez de reescrever em
silencio.

**Refino B8 com dois fatos que o parecer nao trouxe**, ambos verificados agora:

- **o WebSocket do daemon tambem esta no caminho hash.** `internal/daemon/wakeup.go:81-83` envia o
  **mesmo** token do cliente como `Authorization: Bearer`, e `wakeup.go:343` monta o destino
  `/api/daemon/ws`, que e registrado em `cmd/server/router.go:516` **dentro** do grupo
  `/api/daemon` (protegido por `DaemonAuth`). Logo o WS do daemon usa o mesmo caminho `mdt_`
  hash-based, e **nao** quebra com a rotacao.
- **o WebSocket de usuario quebra.** A rota de usuario e `/ws` (`router.go:468`) e a verificacao e
  `internal/realtime/hub.go:676-691 authenticateToken`, que trata **apenas** `mul_` e, para todo o
  resto, faz `jwt.Parse` com `auth.JWTSecret()`. Portanto **o realtime de usuario cai** com a rotacao,
  ainda que o do daemon nao caia.

## V6.1 - fecha **B1**: fonte de interpolacao explicita, via helper com hash fixado

Premissa mantida e confirmada pelo revisor: `services.backend.environment` vence
`services.backend.env_file`, e a interpolacao de `${JWT_SECRET}` vem de shell, `--env-file` de CLI ou
`.env` do projeto - **nunca** do `env_file` de servico. Meu erro na V5 foi nao passar a ponte no
comando.

**Caminho executavel unico (V6):** o helper ORQ-30, com **igualdade exata** de hash como gate, nao
apenas impressao:
```bash
EXP_R=fa1dae2043035152a3dd818353cc1f02aa7fd7e3aa43c6faad9684c267e576af
EXP_B=0e42c65e6d2f0ff250bc25925a37cfd3da7754359c1a0ea62c09e66f810b00a4
GOT_R=$(sha256sum "$HOME/.local/bin/multica-backend-recreate"     | cut -d' ' -f1)
GOT_B=$(sha256sum "$HOME/.local/bin/multica-backend-env-rollback" | cut -d' ' -f1)
[ "$GOT_R" = "$EXP_R" ] || { echo "STOP: recreate helper divergente"; exit 1; }
[ "$GOT_B" = "$EXP_B" ] || { echo "STOP: rollback helper divergente"; exit 1; }
```
E, **antes** da janela, uma leitura de revisao do **codigo-fonte** dos dois helpers provando que eles
(a) passam o arquivo duravel como **entrada de interpolacao** do Compose e (b) usam **somente**
`--force-recreate --no-deps backend`. Sem essa prova, **STOP**: hash igual prova imutabilidade, nao
comportamento.

**Se o owner superseder os helpers (Q-A)**, o comando direto passa a exigir, obrigatoriamente:
```bash
docker compose --project-directory "$APPROVED_PROJECT_DIR" -p "$PROJ" "$@" \
  --env-file "$ENVF" up -d --force-recreate --no-deps backend
```
com `--project-directory` **fixado** (nao inferido do primeiro `-f` nem do PWD) e `--env-file "$ENVF"`
explicito - e **essa** variante exige revisao independente propria antes de existir.

## V6.2 - fecha **B2**: derivar **e comparar** contra allowlist independente do ORQ1

A V5 dizia "derivar tudo, zero hardcode". Aceito a correcao: derivacao sozinha e **auto-autorizante**.
V6 passa a **derive-and-compare**, com allowlist aprovada fora do container:

```text
host       ip-172-31-18-217.sa-east-1.compute.internal
project    multica-dev-transition
service    backend
env file   /home/ec2-user/.config/multica-transition/dev.env
config 1   /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml
config 2   /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml
config 3   /home/ec2-user/.config/multica-transition/images.yml
config 4   /home/ec2-user/.config/multica-transition/backend-env.override.yml
```
Regras duras:
- o container selecionado deve casar `project` **e** `service` da allowlist; divergencia = **STOP**,
  nunca "o container manda";
- a lista de `config_files` deve ser **exatamente** esse conjunto ordenado. **Se a pilha viva tiver um
  quinto arquivo**, ele precisa ser **nomeado por caminho absoluto** e vir com evidencia aprovada pelo
  owner - **nao** se infere quinto arquivo do container. Rejeitar caminho relativo, duplicado, vazio,
  com espaco, symlink ou fora da allowlist;
- testes **numericos**, nao impressao: `[ "$(stat -c %a "$ENVF")" = "600" ]`,
  `[ "$(stat -c %U:%G "$ENVF")" = "ec2-user:ec2-user" ]`, `[ -f "$ENVF" ] && [ ! -L "$ENVF" ]`, e o
  mesmo para o diretorio pai com `700`;
- **CID fresco** apos o recreate: re-selecionar exatamente um container filtrando pelos **dois** labels
  aprovados e exigir `NEW_CID != OLD_CID`.

## V6.3 - fecha **B3**: editor byte-preserving, sem `grep || true`

Retiro o `grep -v "^JWT_SECRET=" ... || true`. O revisor esta certo: ele **mascara erro de leitura** e
poderia produzir um arquivo contendo **apenas** o JWT novo, substituindo o original. E reconhece uma
unica grafia, enquanto a gramatica dotenv do Compose admite espacos, `:`, aspas, comentario inline e
valor multilinha.

**Politica adotada (a variante estrita que o proprio parecer aceita como suficiente):**
1. exigir **exatamente uma** atribuicao ativa de `JWT_SECRET`, na forma canonica **sem aspas**
   `JWT_SECRET=<64hex>`; **zero ou duas ou mais**, ou qualquer grafia alternativa, = **STOP**;
2. abrir o arquivo **regular** sem seguir symlink, com lock exclusivo, ler os bytes **uma vez** e
   **falhar** em erro de leitura ou byte NUL - nunca mascarar;
3. substituir **somente o token de valor**, preservando todos os demais bytes, comentarios, fins de
   linha e a newline terminal;
4. preservar uid/gid/modo;
5. escrever temporario `O_EXCL` **no mesmo diretorio**, `fsync` no arquivo, `rename(2)`, `fsync` no
   **diretorio** - porque `rename` da visibilidade atomica, **nao** durabilidade de crash;
6. `trap` em **EXIT/INT/TERM** removendo o temporario;
7. emitir **apenas** status fixo (`OK_EDIT`, `STOP_MULTI`, `STOP_NONE`, `STOP_READ`, `STOP_NUL`);
8. prova pos-edicao: **todos os bytes que nao sao o valor** devem ser identicos - comparacao
   mecanica com o backup por `cmp` sobre as regioes nao-alvo, nunca por hash impresso.

O editor e um **helper fixado por hash**, revisado antes da janela - nao um one-liner improvisado.

## V6.4 - fecha **B4**: referencia pelo **ambiente**, nao pelo `argv`

A skill diz que o `asm-exec` resolve referencias **em argumentos de comando *e* em variaveis de
ambiente**. A V5 usava argumento, e por isso o valor resolvido aparecia em `/proc/<pid>/cmdline`. V6
passa a referencia pelo **ambiente de entrada**:
```bash
ROTATING_JWT='{{resolve:secretsmanager:<FULL-ARN>:SecretString:<JSON-KEY>:<STAGE>}}' \
ENVF_PATH="$ENVF" \
EDITOR_BIN="$APPROVED_EDITOR" \
asm-exec -- sh -eu -c '
  S=$ROTATING_JWT
  [ -n "$S" ]                        || { echo FAIL_EMPTY;      exit 1; }
  case "$S" in *"{{resolve:"*)          echo FAIL_UNRESOLVED;   exit 1 ;; esac
  case "$S" in *[!0-9a-fA-F]*)          echo FAIL_NOT_HEX;      exit 1 ;; esac
  [ "$(printf %s "$S" | wc -c)" -eq 64 ] || { echo FAIL_LEN;    exit 1; }
  printf %s "$S" | "$EDITOR_BIN" --replace-jwt-secret --file "$ENVF_PATH" --stdin-value
'
```
- o **pai** so conhece a **referencia literal** e caminhos - nunca valor; proibido
  `X=$(asm-exec ...)`, `export` de valor, `docker compose config`;
- o valor vai ao editor por **stdin**, nao por argumento - nem no filho;
- as guardas **G1-G4** que o parecer aprovou como corretas ficam inalteradas.

**Exposicao residual, declarada e nao escondida:** o valor passa a existir em
`/proc/<pid>/environ` do processo `asm-exec`/filho, legivel por processo do **mesmo UID** e por root.
Trocamos `cmdline` por `environ`; **nao** eliminamos a janela. Isso exige uma autorizacao **propria**
(**A5**, secao V6.9), e nao pode ser lido como implicitamente aceito por A1-A4. A eliminacao real
dependeria de um modo "template por stdin" no `asm-exec`, que **nao existe** hoje - e eu **nao**
proponho modificar o `asm-exec` nesta carta.

## V6.5 - fecha **B5**: `G5` com CID fresco e os **dois** status do pipeline

Defeito reconhecido: com CID velho, `docker inspect` falha, o `grep` nao ve nada, devolve 1 e o runbook
imprime o `rc=1` **esperado** - falso PASS.
```bash
NEW_CID=$(docker ps -q --filter "label=com.docker.compose.project=$PROJ" \
                        --filter 'label=com.docker.compose.service=backend')
[ "$(printf %s "$NEW_CID" | wc -w)" -eq 1 ] || { echo "STOP: backend nao unico"; exit 1; }
[ "$NEW_CID" != "$OLD_CID" ]                || { echo "STOP: container nao foi recriado"; exit 1; }

set -o pipefail 2>/dev/null || true
docker inspect -f '{{json .Config.Env}}' "$NEW_CID" | grep -qF 'resolve:secretsmanager'
ins=${PIPESTATUS[0]}; grp=${PIPESTATUS[1]}
[ "$ins" = "0" ] && [ "$grp" = "1" ] || { echo "STOP: G5 indeterminado ins=$ins grp=$grp"; exit 1; }
```
Aceitacao: `ins=0` **e** `grp=1`. Saida permitida: apenas esses dois inteiros. Proibido `echo` do
inspect, `tee`, redirecionar para arquivo ou `grep` sem `-q`. Este segue sendo o **unico** contato
autorizado com `.Config.Env`, e ainda assim exige aprovacao explicita do owner.

Por que `G5` nao e dispensavel: se o literal `{{resolve:...}}` fosse gravado, teria mais de 32 bytes,
passaria `ValidateJWTConfiguration` e o backend assinaria/verificaria **consistentemente** - logo
`antigo=401` e `novo=200` **passariam**. O defeito e invisivel por HTTP.

## V6.6 - fecha **B6**: prova do token **antes** da mutacao, e provas **preservadas** para o rollback

Erro reconhecido: a V5 nunca provava que o token antigo valia **antes** da troca (um token ja expirado
daria `401` depois e seria lido como "invalidacao provada"), e **apagava** os dois arquivos de config
logo depois do forward - justamente o que o `R4` precisa.

**Gates pre-mutacao (nesta ordem):**
```
anonimo  /api/me = 401     (bypass local desligado - efeito, sem ler env)
antigo   /api/me = 200     (o token antigo E valido AGORA)
```
**Gates pos-forward:** `antigo = 401` e **token recem-emitido = 200**.
**Rollback (`R4`):** `antigo = 200` e `novo = 401`.

Regras dos arquivos de credencial do `curl` (criados **pelo owner**, nunca por agente): dois caminhos
**distintos**, regulares, **nao** symlink, dono correto, modo `0600`, diretorio pai `0700` -
verificados **numericamente**. **Preservar ambos** ate a janela inteira ser aceita e o rollback se
tornar impossivel; a remocao e passo final **autorizado pelo owner**, nao efeito colateral do forward.

Detalhes que continuam valendo, verificados: cookie **`multica_auth`** (`internal/auth/cookie.go:20-21`),
precedencia `Authorization: Bearer` antes do cookie (`internal/middleware/auth.go:329-341`), endpoint
`GET /api/me` (`cmd/server/router.go:552`), codigo **exato** via `-o /dev/null -w '%{http_code}'` e
**`curl -f` proibido** (nao distingue `401` esperado de falha de transporte).

## V6.7 - fecha **B7**: backup e restore mecanicamente simetricos

```bash
# backup: nome aleatorio no mesmo diretorio, sem clobber
BK=$(umask 077 && mktemp "$(dirname "$ENVF")/.dev.env.bak.XXXXXX")
cat -- "$ENVF" > "$BK"        # nao imprime; cat para o arquivo, nunca para stdout do runbook
sync -f "$BK" 2>/dev/null || sync
cmp -s -- "$ENVF" "$BK" || { echo "STOP: backup divergente"; exit 1; }
[ "$(stat -c %a "$BK")" = "600" ] || { echo "STOP: modo do backup"; exit 1; }

# restore: mesmo editor/helper fixado, temporario aleatorio, cmp -s como gate
RT=$(umask 077 && mktemp "$(dirname "$ENVF")/.dev.env.restore.XXXXXX")
cat -- "$BK" > "$RT"; sync -f "$RT" 2>/dev/null || sync
cmp -s -- "$BK" "$RT" || { echo "STOP: restore divergente"; exit 1; }
mv -f -- "$RT" "$ENVF"; sync
cmp -s -- "$ENVF" "$BK" || { echo "STOP: dev.env nao restaurado"; exit 1; }
```
- `cmp -s` e o gate **mecanico** - hash **impresso** nao e gate;
- rejeitar symlink e dono/modo inesperados **antes** de tocar;
- recreate do rollback pelo **helper de rollback fixado por hash** (`EXP_B`);
- re-derivar o CID e aplicar o `G5` corrigido (V6.5);
- **manter** os dois configs de token ate o `R4`;
- **`--no-recreate` proibido nas duas direcoes** - foi o defeito que reprovei na V3 e ele torna o
  rollback inerte;
- qualquer gate de rollback falho = **STOP**, sem segunda mutacao.

## V6.8 - fecha **B8**: modelo de dependencia do daemon **condicional**, com evidencia segura do caminho

Ordem real do `daemon_auth.go`, verificada por mim (V6.0): `mdt_` hash -> `mcn_` cloud -> `mul_` PAT ->
**so entao** JWT. E o rotulo do caminho existe no codigo:
`DaemonAuthPathDaemonToken="daemon_token"`, `PAT="pat"`, `CloudPAT="cloud_pat"`, `JWT="jwt"`
(`daemon_auth.go:27-30`), lido por `DaemonAuthPathFromContext` (`:48`).

**Onde o rotulo aparece - e a limitacao honesta:** o unico consumidor fora de teste e
`internal/handler/daemon.go:747`, que passa `authPath` a `logHeartbeatEndpointSlow(...)`. Ou seja, o
rotulo **so** aparece em log de heartbeat **lento**, e **nao** e exposto em nenhuma resposta de API.
Portanto nao existe endpoint que responda "qual caminho eu uso".

**Evidencia segura do caminho ativo (duas opcoes, ambas sem token e sem valor):**
- **E-1 (log)**: procurar no log do backend o campo `auth_path` das linhas de heartbeat lento do
  runtime do ORQ2. Nao produz sinal se nao houver heartbeat lento -> **inconclusivo, nao negativo**;
- **E-2 (metadado de DB, preferida)**: `SELECT daemon_id, expires_at, created_at FROM daemon_token
  WHERE daemon_id = 'orq2-credential-runtime-v1'` - **selecionando explicitamente as colunas, nunca a
  coluna de hash** e nunca `SELECT *`. Linha nao expirada presente = caminho `mdt_`. Exige autorizacao
  de leitura de DB e **nao** foi executada por mim.

**Arvore de decisao (substitui a exigencia incondicional da V5):**
```
caminho = daemon_token (mdt_)  -> SEM re-pareamento. Exigir apenas verificacao pos-rotacao:
                                  heartbeat verde e runtimes online. Nenhuma mutacao no ORQ2.
caminho = pat (mul_) ou cloud_pat (mcn_) -> verificacao especifica do caminho; sem premissa de JWT.
caminho = jwt                  -> SO ENTAO exigir card proprio ja verificado, procedimento revisado,
                                  operador nomeado e autorizacao separada, na MESMA janela.
caminho indeterminado          -> STOP. Nao rotacionar as cegas.
```
**Independente do caminho do daemon**, permanece verdadeiro e deve constar do aviso ao owner: sessoes
humanas caem, e o **realtime de usuario** em `/ws` cai, porque `hub.go:676-691` so trata `mul_` e
verifica todo o resto com `JWTSecret()`.

## V6.9 - fecha **B9**: perguntas e autorizacoes reescritas

| # | pergunta (reescrita) | estado |
|---|---|---|
| **Q-A** | usar os helpers ORQ-30 **fixados por hash** (padrao) ou superseder? Supersedimento exige desenho e revisao novos, com `--env-file` e `--project-directory` explicitos | **ABERTA** |
| **Q-B** | **ARN completo** do segredo (nao id ambiguo em regiao), `json-key` e `version-stage`, com conta/regiao amarradas e **estagio congelado** durante a janela | **ABERTA** |
| **Q-C** | confirmacao de existencia **somente por metadado**, por identidade autorizada; nenhuma API de valor e **nenhuma mutacao** de segredo. Existencia da `json-key` permanece **atestacao do owner** | **ABERTA** |
| **Q-D** | atestacao de **64 hex** sem revelar valor - o parecer aprovou; mantida | **ABERTA (contrato ok)** |
| **Q-E** | procedimento **revisado, somente-owner**, de obtencao do token pos-rotacao (apos o recreate), sem token entrar em contexto de agente; mais o gate **pre-forward `antigo=200`** | **ABERTA** |
| **Q-F** | **condicional**: primeiro evidencia segura do caminho de auth (E-1/E-2); re-pareamento **so** se o caminho for `jwt` | **ABERTA** |
| **Q-G** (nova) | aceite explicito da **queda de sessoes humanas e do realtime de usuario** (`/ws` via `JWTSecret`), com janela e comunicacao aos usuarios | **ABERTA** |

| # | autorizacao (escopo corrigido) |
|---|---|
| **A1** | leitura AWS **somente metadado**. Criacao/atualizacao/rotacao de segredo esta **fora de escopo** e exige autorizacao separada |
| **A2** | backup + **uma** edicao duravel via editor fixado + recreate pelo helper fixado + gates do forward |
| **A3** | rollback simetrico **pre**-autorizado, incluindo **retencao** das provas de token |
| **A4** | ORQ2 **condicional**: se o caminho for `jwt`, re-pareamento com card proprio; caso contrario autoriza **apenas verificacao**, sem mutacao |
| **A5** (nova) | aceite explicito da exposicao residual em `/proc/<pid>/environ` (V6.4) |

As **sete** respostas e as **cinco** autorizacoes devem existir **antes** de qualquer resolucao de
referencia ou mutacao no host.

## V6.10 - sequencia consolidada

```
0  hashes dos helpers == EXP_R/EXP_B  +  revisao do FONTE dos helpers        [B1]
1  derive-and-compare contra a allowlist; testes numericos de modo/dono      [B2]
2  evidencia segura do caminho de auth do daemon (E-2 preferida)             [B8]
3  anonimo /api/me=401  e  antigo /api/me=200                               [B6]
4  gate de metadado do segredo (existencia, por identidade autorizada)       [Q-C]
5  backup com mktemp + cmp -s + fsync                                        [B7]
   -- fronteira: nada acima muta nada --
6  asm-exec com referencia no AMBIENTE; G1-G4; editor fixado por stdin       [B3,B4]
7  recreate pelo helper fixado (--force-recreate --no-deps backend)          [B1]
8  NEW_CID != OLD_CID; /health 200; /readyz 200                              [B2,B5]
9  antigo=401; token novo=200                                                [B6]
10 G5 com PIPESTATUS (ins=0, grp=1)                                          [B5]
11 verificacao do daemon conforme o caminho do passo 2                       [B8]
R  rollback simetrico; R4 com os configs preservados; G5 novamente           [B6,B7]
```

## V6.11 - nao-afirmacoes

- **Nao li valor de segredo**; zero `GetSecretValue`/`BatchGetSecretValue`, zero SMA, **zero
  `asm-exec` executado**, zero AWS, zero Docker/Compose, **nao li nem escrevi `dev.env`**, zero
  `.Config.Env`, zero token emitido ou testado, zero acao em daemon/unit/DB/provider/quadro.
- **Nao executei nenhum passo** de V6.1-V6.10; nao verifiquei hash de helper, nao li o **fonte** dos
  helpers (eles estao no ORQ1 e nao os abri nesta rodada), nao executei `E-2` nem consultei o DB.
- **Nao reli as labels do ORQ1** nesta rodada; a allowlist de V6.2 e **transcrita do parecer** e
  precisa de confirmacao do owner - eu **nao** a validei contra o host.
- O UUID `64bfcae0-…` vem do parecer; **nao** confirmei por `GET` e nao predigo numero de card.
- Verifiquei diretamente, nesta rodada: a ordem de prefixos e o uso de `JWTSecret` em
  `daemon_auth.go`; `wakeup.go:81-83` e `:343`; `router.go:468` e `:516`; `hub.go:676-691`;
  `daemon_auth.go:27-30` e `handler/daemon.go:747`. **Nao** verifiquei o corpo de
  `logHeartbeatEndpointSlow` nem o schema da tabela `daemon_token` (por isso `E-2` exige revisao).
- **Nao adivinho nome de segredo** e nao sei se o segredo existe (Q-B/Q-C).
- **Nao elimino** a exposicao residual de `/proc` - troco `cmdline` por `environ` e peco aceite (A5).
- Nao alterei assignee, nao postei comentario, nao criei card.

---

**PROPOSTA - requer revisao independente.** Ao revisor, ataque prioritariamente: (a) se passar a
referencia pelo **ambiente** realmente e resolvida pelo `asm-exec` como a skill afirma, e se `environ`
e melhor que `cmdline` no modelo de ameaca do owner; (b) se a politica canonica "uma linha
`JWT_SECRET=<64hex>` sem aspas, senao STOP" e aceitavel operacionalmente, ou se o editor precisa mesmo
de parser dotenv completo; (c) se `E-2` (SELECT de colunas nomeadas em `daemon_token`) e admissivel
como evidencia de caminho, ou se ate isso deve ficar com o owner; (d) se a queda do realtime de usuario
que eu identifiquei (`/ws` -> `JWTSecret`) tem outros consumidores que eu nao mapeei.
