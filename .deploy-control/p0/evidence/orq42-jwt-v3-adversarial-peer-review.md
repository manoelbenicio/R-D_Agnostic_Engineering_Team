# ORQ-42 - Peer review adversarial independente da Emenda V3 do ORQ-33

- revisor: **Opus48#A** - ORQ2 - pane w6:p1 - 2026-07-27T15:44Z
- auditado: `.deploy-control/p0/evidence/orq33-jwt-rotation-remediation-v3-design.md` (Antigravity,
  veredito proprio **PASS**)
- skill carregada **integralmente** antes da revisao: `.agents/skills/aws-secrets-manager/SKILL.md`
  (frontmatter, 3 regras, sintaxe, `asm-exec`, How It Works, SigV4, prerequisitos, Common Patterns,
  hook de enforcement e troubleshooting)
- modo: READ-ONLY. **Nenhum segredo lido**, zero `get-secret-value`/`batch-get-secret-value`, zero
  acesso a SMA em `localhost:2773`, **zero `asm-exec` executado**, zero mutacao de AWS, Docker,
  rotacao, login ou board.

---

## VEREDITO: **BLOCK**

A V3 acerta o **entendimento conceitual** que motivou o BLOCK da V2 - e o Ajuste 1 usa exatamente o
padrao sancionado pela skill. Mas **todos os fatos operacionais do Ajuste 3 estao errados**, o comando
central invoca um binario **que nao existe no host**, o guard do Ajuste 2 **captura o segredo em
texto claro no shell** (violando a propria skill que a V3 diz seguir), e o rollback do Ajuste 7 usa
uma flag que **impede o rollback de acontecer**. Quatro desses eu provei medindo o host.

---

## 1. ✅ CREDITO - o Ajuste 1 entendeu a semantica env/file corretamente

A V3 declara: *"`asm-exec` substitui referencias exclusivamente em argumentos de linha de comando
(`argv`) e variaveis de ambiente (`env`), NAO inspecionando nem modificando o conteudo de arquivos de
configuracao no disco"*. **Correto**, e confere com a skill:
- secao "Using `asm-exec`": *"resolves `{{resolve:...}}` references in command arguments and
  environment variables, then `exec`s the target command"*;
- "How It Works", passo 1: *"Scans all command arguments for `{{resolve:...}}` patterns"*.

E a solucao escolhida - `asm-exec -- sh -c '... echo "JWT_SECRET={{resolve:...}}" > "$TMP_ENV" ...'` -
e **literalmente o padrao "Configuration file templating" da skill**:
```bash
asm-exec -- sh -c 'echo "password={{resolve:secretsmanager:app/db:SecretString:password}}" > /tmp/app.conf'
```
O placeholder esta **dentro do argumento** do `sh -c`, logo e resolvido **antes** do `exec`, e o
arquivo recebe o valor real. Isso fecha o Bloqueador 1 da V2 pelo caminho correto. Credito integral.

Tambem correto e nao trivial: o `trap ... EXIT INT TERM`, o `chmod 0600`, a exigencia de `0700` no
diretorio, o Ajuste 4 fixando `GET /api/me` com `old=401`/`new=200`, e o Ajuste 8 tirar o re-pair do
ORQ2 do escopo automatico. A skill tambem respalda uma preocupacao implicita: *"Best-effort defense,
not a security boundary"* - por isso os gates de guarda existirem e certo.

---

## 2. 🔴 BLOQUEADOR 1 - `docker-compose` **nao existe** no ORQ1

O comando central do Ajuste 1 (e o do Ajuste 7) invoca `docker-compose` (v1, com hifen). Medido no
ORQ1:
```
$ docker compose version   -> Docker Compose version v5.3.1     (plugin v2, EXISTE)
$ command -v docker-compose -> docker-compose NAO EXISTE
```
O script falha com *command not found* na primeira execucao. Corrigir para `docker compose`
(subcomando, sem hifen).

## 3. 🔴 BLOQUEADOR 2 - os "4 caminhos absolutos" estao **todos errados**, e faltam os dois que importam

Medido, `docker compose ls` no ORQ1, saida literal:
```
NAME                     STATUS        CONFIG FILES
multica-dev-transition   running(3)    /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml,
                                       /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml,
                                       /home/ec2-user/.config/multica-transition/images.yml,
                                       /home/ec2-user/.config/multica-transition/backend-env.override.yml
```
Comparando com o Ajuste 3, item por item:

| # | caminho afirmado pela V3 | existe no ORQ1? | pertence a stack? |
|---|---|---|---|
| 1 | `/home/ec2-user/workspace/R-D_.../docker-compose.selfhost.yml` | **NAO-EXISTE** | nao |
| 2 | `/home/ec2-user/workspace/R-D_.../docker-compose.selfhost.build.yml` | **NAO-EXISTE** | nao |
| 3 | `/home/ec2-user/workspace/R-D_.../docker-compose.yml` | **NAO-EXISTE** | nao |
| 4 | `/home/ec2-user/workspace/worktrees/gtl-orq26/multica-auth-work/docker-compose.selfhost.yml` | **NAO-EXISTE** | nao - e worktree de **outro cartao** |

Ou seja **4 de 4 nao existem no host onde o comando roda**. O prefixo `/home/ec2-user/workspace/...`
e o caminho do **ORQ2**, onde a frota trabalha; no ORQ1 a arvore vive sob `$HOME` direto (verifiquei
que `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml` e
`.../docker-compose.yml` **existem**). Este e exatamente o erro que meu parecer da V2 apontou, e a V3
o **repetiu com novos caminhos** em vez de copiar o label.

Pior que os caminhos errados: a V3 **omite** `images.yml` e `backend-env.override.yml`. O override e
justamente onde o `JWT_SECRET` efetivo pode estar definido (gate G5 da V2, nunca resolvido). Rodar
`up -d` sem esses dois `-f` recria o backend com **configuracao diferente da atual** - perda de
imagem pinada e de overrides de ambiente, muito alem da troca de segredo.

E incluir o compose do worktree `gtl-orq26` e pior ainda: misturaria o estado de um cartao de UI
dentro da rotacao de segredo.

## 4. 🔴 BLOQUEADOR 3 - `-p multica` e o projeto errado

Medido: `docker inspect ... com.docker.compose.project` = **`multica-dev-transition`**, e
`docker compose ls` lista **um** stack, com esse nome.

`-p multica` nao recria o backend existente: ele cria/afeta um **projeto novo e vazio**. Efeito real
do Ajuste 1 como escrito: sobe um segundo conjunto de containers `multica-*` (ou falha por porta em
uso), **enquanto o backend real continua rodando com o segredo antigo**. E depois os gates de
validacao podem passar contra o container errado. Corrigir para `-p multica-dev-transition`, e
preferencialmente **omitir `-p`** quando os `-f` corretos forem usados, porque o nome do projeto ja
vem do diretorio/label.

## 5. 🔴 BLOQUEADOR 4 - o guard do Ajuste 2 **captura o segredo em texto claro**, contra a propria skill

```bash
RESOLVED_SECRET=$(asm-exec -- env | grep ^JWT_SECRET= | cut -d= -f2-)
```
Tres problemas, o primeiro grave:
1. **o valor entra em variavel do shell do operador**, via substituicao de comando. A skill existe para
   que *"the secret value exists only in the child process -- never in the agent's context"*. Aqui o
   valor sai do processo filho e vira `$RESOLVED_SECRET` no processo pai: aparece com `set -x`, em
   core dump, em `/proc/<pid>/environ` se exportado, e em qualquer log do script. E o mesmo tipo de
   vazamento que a regra 1 da skill proibe, apenas por outro caminho. O Ajuste 6 **agrava**, reusando
   `$RESOLVED_SECRET` em `echo -n "$RESOLVED_SECRET" | wc -c`.
2. **nao funciona como escrito**: `asm-exec -- env` so imprimiria `JWT_SECRET=` se essa variavel
   estivesse **no ambiente do `asm-exec`** com um placeholder. O comando nao define nada, entao o
   `grep` volta vazio, `-z` dispara e o gate **aborta sempre** - falso bloqueio.
3. o `case` embutido em `if ... || case ...; then` funciona em POSIX, mas e obscuro; e testa apenas o
   prefixo `{{`, nao `{{resolve:`.

**Correcao**: nunca capturar o valor. Verificar **efeito**, nao conteudo, e no lugar certo - o
container:
```bash
# 1. o placeholder literal chegou ao container?  criterio: 0
docker inspect multica-dev-transition-backend-1 \
  --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -c '{{resolve:'

# 2. comprimento sem expor valor, DENTRO do filho
asm-exec -- sh -c 'printf %s "{{resolve:secretsmanager:<id>:SecretString:<key>}}" | wc -c'
```
O item 2 mantem o valor no filho e imprime **apenas o numero**. E `printf %s` em vez de `echo -n`,
porque `echo -n` nao e portavel em `sh`/dash - pode imprimir literalmente `-n` e inflar a contagem em
2 bytes, mascarando um segredo de 30 bytes como 32.

## 6. 🔴 BLOQUEADOR 5 - o rollback do Ajuste 7 usa `--no-recreate`, que **impede o rollback**

```
docker-compose -p multica up -d --no-recreate
```
`--no-recreate` instrui o compose a **nao** recriar containers existentes. Como a V2 estabeleceu
corretamente - e a V3 nao contesta - `JWTSecret()` resolve **uma vez por processo** via `sync.Once`
(`internal/auth/jwt.go:31`), logo **so um processo novo carrega a chave**. Com `--no-recreate`, o
container permanece com a chave **nova**, e o rollback nao acontece; os probes voltariam a passar com
o segredo errado e o operador declararia rollback bem-sucedido.

**Correcao**: `--force-recreate --no-deps backend`, os mesmos flags do avanco. E `--no-deps` para nao
recriar Postgres nem frontend - o Ajuste 1 tambem omite `--no-deps` e `backend`, entao `up -d` atinge
**todos** os servicos, incluindo o Postgres.

---

## 7. Achados adicionais (nao bloqueantes isolados, bloqueantes em conjunto)

- **A1 - `$HOME/.private-tmp` nao existe no ORQ1.** `ls -ld` retorna *No such file or directory*.
  O `mktemp "$HOME/.private-tmp/jwt-env.XXXXXX"` falha. O Ajuste 5 **exige** `0700` mas nenhum passo
  **cria** o diretorio. Adicionar `mkdir -p -m 0700 "$HOME/.private-tmp"` **e** `chmod 0700` (o `-m`
  nao corrige diretorio pre-existente), e verificar com `stat -c%a`.
- **A2 - `umask 077` e citado no Ajuste 5 mas nao aparece no script.** Sem ele, e sem o `chmod`
  acontecer antes do `echo`, existe uma janela em que o arquivo pode nascer legivel. Ordem correta:
  `umask 077` -> `mktemp` -> `chmod 0600` -> `echo`.
- **A3 - `trap "rm -f $TMP_ENV"` com aspas duplas** expande no momento da definicao (funciona), mas o
  caminho fica **sem quotes** no corpo do trap. Usar `trap 'rm -f "$TMP_ENV"' EXIT INT TERM`.
- **A4 - o `--env-file` so funciona porque o compose interpola.** Nao esta declarado no desenho:
  `--env-file` alimenta **substituicao de variaveis do arquivo compose**, nao injeta env no container
  por si. Funciona porque `docker-compose.selfhost.yml:58` tem
  `JWT_SECRET: ${JWT_SECRET:?...}`. Se o `-f` correto nao for usado (Bloqueador 2), essa cadeia
  quebra silenciosamente. Precisa estar escrito como dependencia.
- **A5 - nome do segredo mudou sem justificativa e sem verificacao.** V2 propunha
  `prod/multica/jwt-secret` com json-key `jwt_secret`; V3 usa `prod/jwt-secret` com key `secret`.
  Nenhum dos dois foi confirmado. Pela skill, *"Resolution produces empty string: the JSON key may
  not exist"* - json-key errado resolve para **vazio**, e o compose com `:?` abortaria (bom), mas o
  desenho nao pode depender de sorte. Fixar id e key, e confirmar existencia por metadado
  (`describe-secret`, que **nao** retorna valor).
- **A6 - `FILES_LOCKED` incoerente.** Declara `internal/auth/jwt.go` como LOCKED, mas **nenhuma
  mudanca de codigo esta prevista** na V3 - lock espurio que bloqueia outras frentes. Declara
  `docker-compose.selfhost.yml` como LOCKED, porem o plano usa `--env-file` e **nao edita** o compose.
  E **nao** trava o que de fato sera tocado: `~/.config/multica-transition/backend-env.override.yml`.
  Alem disso o caminho declarado e relativo ao repo do ORQ2, nao ao arquivo do ORQ1.
- **A7 - Ajuste 8 transforma um efeito colateral certo em cartao futuro.** Correto separar
  autorizacao; **errado** tratar como fora de escopo. A rotacao **quebra** a auth de daemon por
  `internal/middleware/daemon_auth.go:234`, e o daemon do ORQ2 e o executor da frota. Sem o re-pair na
  **mesma janela**, a rotacao produz outage do executor por tempo indeterminado. O re-pair deve ser
  **pre-condicao com autorizacao propria**, agendada na mesma janela - nao um `ORQ-N` depois.
- **A8 - probes inconsistentes.** O Ajuste 7 cita `/healthz` e `/readyz`; a V2 estabeleceu que
  `/health` e liveness e `/readyz`/`/healthz` sao readiness (`cmd/server/router.go:443-445`). Fixar os
  tres explicitamente para nao repetir o erro da V1.
- **A9 - veredito proprio PASS.** A secao 4 declara PASS no proprio artefato, ao mesmo tempo que o
  Ajuste 9 pede peer review independente. As duas coisas nao coexistem. Status correto: *proposta
  aguardando review*.

---

## 8. Correcoes exatas para virar PASS

| # | correcao | onde |
|---|---|---|
| **F1** | `docker-compose` -> `docker compose` | Ajustes 1 e 7 |
| **F2** | usar os **quatro caminhos reais** do label: `/home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml`, `.../docker-compose.selfhost.build.yml`, `/home/ec2-user/.config/multica-transition/images.yml`, `/home/ec2-user/.config/multica-transition/backend-env.override.yml`; remover o compose do worktree `gtl-orq26` e o `docker-compose.yml`; adicionar gate que **releia** `docker compose ls`/label antes de executar | Ajuste 3 |
| **F3** | `-p multica-dev-transition` (ou omitir `-p`) | Ajustes 1, 3, 7 |
| **F4** | nunca capturar o segredo em variavel do shell; guard por `docker inspect ... \| grep -c '{{resolve:'` com criterio **0**, e comprimento via `asm-exec -- sh -c 'printf %s "{{resolve:...}}" \| wc -c'` | Ajustes 2 e 6 |
| **F5** | rollback com `--force-recreate --no-deps backend`; nunca `--no-recreate` | Ajuste 7 |
| **F6** | `up -d --force-recreate --no-deps backend` no avanco (hoje `up -d` atinge todos os servicos, inclusive Postgres) | Ajuste 1 |
| **F7** | criar e verificar `$HOME/.private-tmp` (`mkdir -p -m 0700` + `chmod 0700` + `stat -c%a`); aplicar `umask 077` antes do `mktemp`; `trap 'rm -f "$TMP_ENV"'` com quotes | Ajustes 1 e 5 |
| **F8** | declarar a dependencia de interpolacao (`JWT_SECRET: ${JWT_SECRET:?...}` em `docker-compose.selfhost.yml:58`) que faz o `--env-file` funcionar | Ajuste 1 |
| **F9** | fixar `secret-id` e `json-key` e confirmar existencia por `describe-secret` (metadado, sem valor) | Ajustes 1, 2, 6 |
| **F10** | corrigir `FILES_LOCKED`: remover `jwt.go`, remover `docker-compose.selfhost.yml`, incluir `~/.config/multica-transition/backend-env.override.yml` se for editado | secao 3 |
| **F11** | re-pair do daemon ORQ2 como **pre-condicao com autorizacao propria na mesma janela**, nao cartao futuro | Ajuste 8 |
| **F12** | fixar `/health` (liveness) + `/readyz` + `GET /api/me` como o trio de validacao | Ajustes 4 e 7 |
| **F13** | trocar o PASS proprio por "proposta aguardando review" | secao 4 |

O desenho conceitual esta certo; o que reprova e que **nenhum comando da V3 rodaria como escrito**, e
os dois que rodariam parcialmente agiriam no projeto errado ou impediriam o proprio rollback.

## 9. Nao-afirmacoes desta revisao

- Carreguei a skill **inteira** antes de revisar, e nao chamei `get-secret-value`,
  `batch-get-secret-value`, nem acessei SMA em `localhost:2773`. **Nao executei `asm-exec`.**
- Nao li nenhum segredo, nao gerei nem usei token, nao fiz login, nao rotacionei nada.
- Nao mutei AWS, Docker, container, unit, codigo ou board. Nao alterei assignee nem postei comentario.
- Leituras no ORQ1: `docker compose version`, `command -v docker-compose`, `docker compose ls`,
  `docker inspect` **apenas de labels** (nunca `.Config.Env`), `ls -e`/`ls -ld` de caminhos. Nenhuma
  delas expoe valor de segredo.
- Nao inspecionei `~/.config/multica-transition/backend-env.override.yml`: pode conter segredo.
  Portanto **nao sei** onde `JWT_SECRET` esta definido hoje, e o gate G5 da V2 continua aberto - a V3
  nao o resolveu nem o mencionou.
- Nao consultei o Secrets Manager: nao sei se `prod/jwt-secret` ou `prod/multica/jwt-secret` existem.
- Nao revalidei o UUID do ORQ-33 nem o do ORQ-42 por `GET`.
