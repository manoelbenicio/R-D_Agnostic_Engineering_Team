# ORQ-42 - V9: Q-H e Q-I **implementadas**, com pins reais (PROPOSTA)

- **Status: PROPOSTA.** Nao aprovada, **nao executavel**. **Nao me auto-aprovo** - peco review.
- autor/executor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T20:14Z
- substitui: V8 (`orq42-jwt-rotation-runbook-v8-design.md`, sha256 `21d820de…`)
- skill carregada **integralmente** antes desta rodada: `.agents/skills/aws-secrets-manager/SKILL.md`
  v1 - as 3 regras MUST, sintaxe `{{resolve:…}}` com defaults `SecretString`/`AWSCURRENT`, `asm-exec`
  resolvendo **argumentos e variaveis de ambiente**, ordem SMA -> MCP SigV4, `re.sub` single-pass,
  ausencia de fallback para CLI local, Common Patterns, hook `PreToolUse`, troubleshooting.
- **Diferenca central em relacao a V8: as duas dependencias deixaram de ser texto.** Estao
  implementadas, compiladas e testadas.

---

## V9.1 - Worktree, escopo e commits (nada de produto tocado)

```text
worktree   /home/ec2-user/workspace/worktrees/orq42-secret-tools
branch     agent/opus48-a/orq42-secret-tools        (novo, base 0cb8aeb)
commits    355c57b  feat(orq42): editor + probe
           bbeb80b  fix(orq42): remove binarios commitados por acidente + .gitignore
```
**FILES_LOCKED (todos novos, modulos separados, apenas biblioteca padrao):**
```text
tools/orq42/envsecret-editor/main.go        8793 B  sha256 824f7fc6746c8703097877df1da14ddbffe07994ca048bddcdb72a041265b4ce
tools/orq42/envsecret-editor/main_test.go   7113 B  sha256 a20a21a8c6f813791faf5d1708d88e108d6f332b0fd8c5edcd179111841fdeae
tools/orq42/envsecret-editor/go.mod           67 B  sha256 7e30eca6984ade31fb1b0891cf29afabbb9ae4abd1766e541a7404a073dd55c5
tools/orq42/wsprobe/main.go                10290 B  sha256 4508d9fddb7a16ee1fa749200066f677b214f9b0d96132ca0c39f618f2800e10
tools/orq42/wsprobe/main_test.go            9299 B  sha256 8db06bef8fb3f69e444b2890d1fe434f8edc8a5a1ed204e72321d6fc2dfbb87e
tools/orq42/wsprobe/go.mod                    58 B  sha256 cc7a3ca1fe57284c86a7e99058a49a89de3320c3ce839c04bcabdf98f29b5d08
tools/orq42/.gitignore                       259 B  sha256 1f71df507f15d3718b947173c782f1af4aa0f19de804f4794117d451c243d783
```
Modulos proprios (`github.com/multica-ai/orq42-tools/…`) **de proposito**: o `go.mod`/`go.sum` do
`server` **nao** e alterado, e nenhum arquivo de produto entra no escopo.

## V9.2 - Q-H **FECHADA**: `envsecret-editor`

O que o codigo garante, ponto a ponto contra a exigencia da V8:

| exigencia | implementacao |
|---|---|
| `O_NOFOLLOW` real | `os.OpenFile(path, O_RDONLY\|syscall.O_NOFOLLOW, 0)`; `ELOOP` -> `STOP_SYMLINK` |
| arquivo regular | `info.Mode().IsRegular()` -> `STOP_NOT_REGULAR` |
| modo restrito | `Perm()&0o077 != 0` -> `STOP_PERM` |
| dono | `Stat_t.Uid != os.Getuid()` -> `STOP_OWNER` |
| **uma** atribuicao ativa | `assignmentSpan` conta; `0` -> `STOP_NONE`, `>1` -> `STOP_MULTI`. Linha comentada ou indentada **nao** e atribuicao |
| prefixo/sufixo byte-preserving | reconstroi `data[:valueStart] + valor + data[valueEnd:]`; comentarios, ordem, `CRLF`/`LF` e newline terminal sobrevivem **por construcao**; um `\r` final fica no sufixo |
| temp `O_EXCL` no mesmo diretorio | `os.CreateTemp(dir, ".envsecret-*.tmp")` + `Chmod` para o modo original |
| `fsync` + `rename` + `fsync` do dir | `tmp.Sync()` -> `os.Rename` -> `os.Open(dir).Sync()`; comentado que `rename` da **visibilidade** atomica, nao durabilidade |
| limpeza sempre | `defer` remove o temp em **todo** caminho de falha |
| valor so por stdin | `readValue`; rejeita vazio, `\r\n` embutido, `NUL`, literal `{{resolve:` e, com `-expect-hex64`, o que nao for 64 hex |
| rotulos fixos | 17 constantes; **o valor nunca e impresso**, e ele e zerado com `defer` |

**Testes: 10, todos passando sob `-race`** (`go test -race -count=1`), incluindo:
preservacao byte-a-byte de nao-alvo; `CRLF`, sem-newline-terminal e `CRLF` sem newline terminal;
recusa de zero, indentada e duplicada **sem escrever no arquivo**; recusa de symlink **sem escrever
atraves dele**; recusa de `0644`; cinco formas de valor invalido; `NUL` no arquivo; **nenhum temporario
deixado para tras** apos sucesso **e** apos falha; modo preservado; e recusa de caminho relativo e de
chave com `=`.

## V9.3 - Q-I **FECHADA**: `wsprobe`

Implementa os **dois** caminhos reais de `internal/realtime/hub.go`, so com biblioteca padrao:

- **`-mode cookie`** - o servidor valida **antes** do upgrade, entao o veredito e o **status HTTP**:
  `101` -> `WS_ACCEPTED`, `401/403` -> `WS_REJECTED`, `400` -> `WS_BAD_REQUEST` (`workspace_id` ou
  `workspace_slug` ausente). **Nao precisa de framing.**
- **`-mode first-frame`** - faz o upgrade, envia
  `{"type":"auth","payload":{"token":"…"}}` e **exige** `{"type":"auth_ack"}`; frame de erro ou `close`
  -> `WS_AUTH_DENIED`.
- handshake escrito a mao com `Sec-WebSocket-Key` aleatorio e **verificacao** do
  `Sec-WebSocket-Accept` (SHA-1 + GUID da RFC 6455); frames de cliente **mascarados**, como a RFC exige;
  recusa de frame de 64 bits e de frame truncado.
- **token so por stdin**, nunca em `argv`, nunca logado; a copia serializada e **zerada** apos ir ao fio;
  saida e **um** rotulo `WS_*`; codigo de saida `0` aceito, `3` rejeicao definitiva (nao e falha de
  ferramenta), `1` falha de ferramenta, `2` uso.

**Testes: 8, todos passando sob `-race`**, com um hub falso que reproduz os dois caminhos: aceita token
valido e rejeita o antigo em **ambos** os modos; `400` sem workspace; **um teste dedicado prova que o
token nunca aparece em stdout/stderr** e que toda linha emitida comeca com `WS_`; stdin vazio recusado
antes de qualquer conexao e newline final tolerado; usos invalidos; porta fechada -> `WS_STOP_DIAL`; e
round-trip de mascaramento provando que o payload **nao** vai ao fio em claro.

## V9.4 - Determinismo de build e pins de binario

```bash
GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -o <PRIVATE_0700>/<tool> .
```
Dois builds, com **caches diferentes**, produziram **hashes identicos**:
```text
envsecret-editor  sha256 0a191f4387343ecf358e8c7a703fbc331fdd37716bba5a3e40f14f79dafd5f73
wsprobe           sha256 305550b1a1cfb5a8ee87347b7eb09850fdb31b7fc23f4ce7f26c5cd1a5618453
go                go1.26.1 linux/amd64
```
Os binarios ficam **fora** do repositorio, em cache privado `0700`; o `.gitignore` impede que um
`go build ./...` dentro do pacote os reintroduza.

## V9.5 - Q-C: comando **somente metadado**, para o perfil `owner-p0`

**Nao executei nenhum destes.** Se o perfil nao existir ou nao tiver permissao, **nao executar** e
escalar - a identidade deste agente (`assumed-role/cw-agent-orquestradores`) ja levou `AccessDenied` em
`describe-secret` em `sa-east-1`.
```bash
# 0) o perfil precisa existir; sem ele, PARAR (nao improvisar credencial)
aws configure list-profiles | grep -qx owner-p0 || { echo E_NO_PROFILE; exit 1; }

# 1) identidade e existencia - METADADO, zero valor
aws --profile owner-p0 --region "$REGION" sts get-caller-identity --output json
aws --profile owner-p0 --region "$REGION" secretsmanager describe-secret \
  --secret-id "$SECRET_ARN" \
  --query '{Name:Name,Arn:ARN,KmsKeyId:KmsKeyId,RotationEnabled:RotationEnabled,LastChangedDate:LastChangedDate,DeletedDate:DeletedDate}' \
  --output json

# 2) estagios por versao - decide E3.1 (AWSPREVIOUS existe) vs E3.2 (.prev obrigatorio)
aws --profile owner-p0 --region "$REGION" secretsmanager list-secret-version-ids \
  --secret-id "$SECRET_ARN" --include-deprecated \
  --query 'Versions[].{VersionId:VersionId,Stages:VersionStages,Created:CreatedDate}' \
  --output json

# 3) ESTABILIDADE: fixar o VersionId de AWSCURRENT no inicio e reconferir no fim da janela
aws --profile owner-p0 --region "$REGION" secretsmanager list-secret-version-ids \
  --secret-id "$SECRET_ARN" \
  --query 'Versions[?contains(VersionStages, `AWSCURRENT`)].VersionId | [0]' --output text
# guardar como AWSCURRENT_VERSION_ID_START; se mudar ate o fim -> STOP (rotacao por fora)
```
Aceitacao: `DeletedDate` **ausente**; existe **exatamente uma** versao com `AWSCURRENT`; a presenca de
`AWSPREVIOUS` decide o caminho de rollback; e o `VersionId` do `AWSCURRENT` **nao muda** durante a
janela. Nenhum comando acima e `get-secret-value` ou `batch-get-secret-value`, portanto **nenhum viola a
regra 1** - e nenhum retorna `SecretString`.

## V9.6 - O que a V9 **nao** muda em relacao a V8

Permanecem, sem alteracao: allowlist absoluta com os **5** `config_files` na ordem; `env -u JWT_SECRET`
em **toda** invocacao (armadilha real: a precedencia de interpolacao do Compose e
**shell > `--env-file` > `.env`**); igualdade **criptografica booleana** `EQ`/`NEQ` com sal efemero em
lugar de qualquer leitura de `.Config.Env`; matriz de 6 consumidores com `/api/me` e `/ws` quebrando e o
caminho `mdt_` por hash **nao** quebrando; login do owner consumindo **stdin** com `--data-binary @-`,
`printf` builtin, `ulimit -c 0` e `mkdir -m 0700` sem `-p`; e as 10 autorizacoes.

Duas mudancas de estado: `Q-H` e `Q-I` deixam de ser bloqueadores, e o `sha256` do editor - que na V8
estava **vazio** - agora e real (`824f7fc6…` para a fonte; `0a191f43…` para o binario reproduzivel).

## V9.7 - Nao-afirmacoes

- **Nenhum push, PR, merge, quadro, AWS, segredo, Docker, SSH ao ORQ1 ou mutacao de runtime.** Os
  commits sao **locais**.
- **Defeito meu, corrigido e registrado**: o commit `355c57b` incluiu por acidente os **dois binarios**
  (3,1 MB e 6,2 MB), porque eu rodei `go build ./...` dentro dos diretorios de pacote. Removi no commit
  `bbeb80b` e adicionei `.gitignore`. **Os blobs continuam no historico** de `355c57b`; limpar exigiria
  reescrever dois commits locais, o que **nao** fiz por conta propria - depende de autorizacao explicita.
- **Os utilitarios nunca foram executados contra alvo real.** O `wsprobe` foi testado **so** contra um
  hub falso `httptest` local; ele **nao** foi apontado para o backend do ORQ1, e portanto **nao** afirmo
  que o contrato real de `/ws` se comporta como o falso - isso e exatamente o que o gate deve provar na
  janela.
- O `envsecret-editor` **nunca** editou o `dev.env` real; foi exercitado somente em fixtures de
  `t.TempDir()`.
- **Nao executei nenhum comando da V9.5**, nem verifiquei se o perfil `owner-p0` existe.
- O `wsprobe` implementa **o minimo** de RFC 6455 que os dois gates exigem: sem TLS (`wss`), sem
  fragmentacao, sem `ping/pong`, sem frames de 64 bits. Se o alvo exigir `wss` ou fragmentar a resposta,
  **falha** com `WS_STOP_*` em vez de dar veredito - fail-closed, mas e uma limitacao real.
- O gate `first-frame` depende de o servidor responder em **no maximo 4 frames**; acima disso devolve
  `WS_STOP_PROTOCOL`.
- Os fatos de `hub.go` usados no desenho vem de leituras minhas anteriores e de agora; **nao** remedi o
  `router.go` nesta rodada.
- Nao alterei assignee, nao postei comentario, nao criei card.

---

**PROPOSTA - requer review.** Ataques sugeridos: (a) se o `assignmentSpan` trata alguma grafia dotenv
legitima como "nao-atribuicao" e assim recusa um arquivo valido; (b) se o `wsprobe` deveria exigir `wss`
antes de ser autorizado contra o alvo; (c) se a igualdade `EQ`/`NEQ` mais a procedencia por label
fecham a cadeia ate a chave efetiva; (d) se os blobs no historico local exigem reescrita antes de
qualquer push.
