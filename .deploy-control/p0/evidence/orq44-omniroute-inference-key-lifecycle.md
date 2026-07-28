# ORQ-44 - ciclo de vida e rotacao da chave de inferencia do gateway OmniRoute

- **Autor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T15:35Z
- **Cartao:** `ORQ-44`
- **Skill carregada integralmente antes do trabalho:** `.agents/skills/aws-secrets-manager/SKILL.md` (v1)
- **Modo:** READ-ONLY / DESENHO. **Nenhum segredo lido**, zero `GetSecretValue`/`BatchGetSecretValue`,
  **zero execucao de `asm-exec`**, zero mutacao de codigo, board, config, container, provider ou AWS.
- **Escritor/executor:** Codex56-TL (GENERAL-TECH-LEAD). Autoridade final: **owner humano**.

---

## 1. REGRAS DA SKILL APLICADAS A ESTE DESENHO

Da skill, o que rege este cartao:
- **R1** MUST NOT chamar `get-secret-value` nem `batch-get-secret-value` por CLI, SDK, MCP, curl ou
  qualquer mecanismo. Nao chamei.
- **R2** MUST NOT ler o SMA em `localhost:2773` nem variante de loopback. Nao li.
- **R3** MUST usar `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}`,
  resolvido em runtime por `asm-exec`, com o valor existindo **apenas no processo filho**.

Dois pontos da skill que este desenho explora e que a maioria dos runbooks ignora:
1. **`version-stage` e parametro da propria sintaxe**, com default `AWSCURRENT` e exemplo `AWSPENDING`.
   Isso e o que permite preparar a chave nova **sem** promove-la (secao 5).
2. A skill declara-se **"best-effort defense, not a security boundary"** e manda combinar com IAM
   least-privilege, CloudTrail e VPC endpoint policies. Portanto o desenho abaixo nao trata `asm-exec`
   como controle suficiente; o controle real de exposicao aqui e o modo `0600` mais dono do arquivo,
   validado pelo consumidor (secao 2).

---

## 2. CONSUMIDOR E COMPORTAMENTO FAIL-CLOSED (medido)

Alvo: o arquivo referenciado por `AGENT_BRAIN_GATEWAY_SECRET_FILE`
(`internal/daemon/brain/config.go:16`), na topologia atual
`/etc/agent-brain/secrets/omniroute-inference-key`.

### 2.1 Referencia e validacao de configuracao
`brain/config.go:84-92` exige caminho **absoluto** e nao vazio (`NewSecretFileRef`).
`brain/config.go:163-181` (`GatewayConfig.Validate`) exige, em modo `Required`:
```go
	if c.Required && c.SecretFile.Path == "" {
		return fmt.Errorf("gateway-required mode requires a secret file reference")
	}
	if c.Required && (!c.Readiness.FailClosed || c.Readiness.Name != ReadinessStrict) {
		return fmt.Errorf("gateway-required mode requires strict fail-closed readiness")
	}
```
Ou seja: sem referencia de secret, o modo gateway-required **nao valida** - falha na configuracao,
antes de qualquer inferencia.

### 2.2 Leitura por chamada, nunca em cache - a diferenca decisiva
`internal/daemon/credential_file_source.go:26-39` (comentario literal do codigo):
```go
// FileCredentialSource is the production gateway.CredentialSource.
// It reads a restricted secret file at call time with fail-closed
// metadata and content-shape validation. The value is never cached,
// logged, or returned outside the callback scope.
```
E o fluxo em `:45-108`:
1. `openCredentialFile(ref.Path)` com **`O_NOFOLLOW`** - symlink e rejeitado no `open`, eliminando o
   TOCTOU entre `Lstat` e `ReadFile` (comentario `:32-34`);
2. `f.Stat()` no **mesmo descritor** aberto, nao re-stat de caminho (`:61-62`);
3. rejeita nao-regular (`:66-68`);
4. `info.Mode().Perm() & 0o077 != 0` -> **rejeita qualquer bit de grupo ou mundo**, aceitando `0600`
   ou mais restrito como `0400` (`:21-23`, `:69-72`);
5. `checkCredentialOwner(info)` -> dono deve ser o **UID do processo** (`:73-75`);
6. tamanho maximo **4096 bytes** (`:17-18`, `:77-79`, `:87-89`);
7. conteudo lido do **mesmo descritor** (`:81-83`);
8. forma do valor: minimo **8** caracteres apos trim (`:19-20`, `:93-95`), UTF-8 valido (`:96-98`),
   **sem** espaco/tab/CR/LF (`:99-101`), **sem** caractere de controle (`:102-106`);
9. `return use(value)` (`:108`) - o valor vive somente no escopo do callback.

Erros sao `credentialFileError` com **razao sem conteudo** (`:111-119`), por exemplo
`secret_file_permissions_too_open`, `secret_file_content_too_short`. Nunca vaza valor nem erro de OS.

**Consequencia central para o ciclo de vida:** como a leitura e **por chamada** e nao cacheada,
substituir o arquivo passa a valer na **proxima requisicao ao gateway**, sem reiniciar processo.
Isso e o oposto do `JWT_SECRET`, que fica preso em `sync.Once` (secao 7).

### 2.3 Validacao no cliente HTTP
`internal/daemon/gateway/client.go` valida o valor antes de usar (`validCredential`): nao vazio,
`<= 4096`, sem espaco nas bordas, sem `\r`, `\n` ou `\x00`; e injeta como
`Authorization: Bearer <credential>` **dentro** do callback, removendo o header em seguida.
Credencial invalida -> `GatewayError{Class: ErrorAuthentication}`.

---

## 3. FEASIBILIDADE DE SOBREPOSICAO NO LADO DO PROVIDER

Aqui o desenho depende de um fato que **eu nao posso verificar**, e digo isso antes de propor:
a sobreposicao de duas chaves de inferencia e **capacidade do OmniRoute**, nao do daemon.

O que **e** verificavel do lado do consumidor:
- o daemon envia **uma** credencial por requisicao, lida do arquivo naquele instante;
- nao existe conceito de chave primaria/secundaria no daemon: `SecretFileRef` e **um** caminho
  (`brain/config.go:80-92`);
- portanto o daemon **suporta** naturalmente uma janela de sobreposicao **se e somente se** o
  OmniRoute aceitar as duas chaves simultaneamente. Trocar o arquivo passa a usar a chave nova na
  requisicao seguinte; se o provider ainda aceita a antiga, nada quebra; se nao aceita, quem estiver
  a meio de uma requisicao com a antiga recebe `401`.

O que **nao** e verificavel por mim e precisa de resposta do operador do OmniRoute, **antes** de
executar:
1. o OmniRoute aceita **duas** chaves ativas ao mesmo tempo? Se sim, por quanto tempo e como se
   revoga a antiga?
2. a chave e por **conta** ou global? O gateway tem 4 contas Antigravity, 4 ClinePass e 2 OpenAI
   Codex documentadas; rotacionar a chave de inferencia afeta todas ou apenas o par de credencial do
   cliente?
3. existe endpoint de introspeccao que diga "esta chave e valida" **sem** consumir cota?

Se a resposta a (1) for **nao**, a rotacao vira evento disruptivo curto: a janela de risco e apenas o
tempo de requisicoes em voo, nao a vida de tokens de sessao. Isso ainda e muito melhor que o
`JWT_SECRET`, porque nao ha estado assinado de longa duracao.

**Nao afirmo** qual das duas hipoteses vale. O desenho abaixo funciona nas duas, com o passo de
canario ajustando a expectativa.

---

## 4. TROCA ATOMICA DO ARQUIVO EM `0600` SEM LER O VALOR

Requisito: substituir o conteudo **sem** que o valor passe pelo contexto do agente e **sem** janela
em que o consumidor leia arquivo parcial.

Por que `rename(2)` e obrigatorio e nao opcional: a leitura e por chamada (2.2). Escrever "no lugar"
com `>` truncaria o arquivo e uma requisicao concorrente leria conteudo vazio ou parcial, o que o
consumidor rejeitaria com `secret_file_content_too_short` -> `401` espurio. `rename` no **mesmo
filesystem** e atomico: quem abriu o inode antigo continua lendo o antigo; quem abrir depois ve o novo.

Forma proposta, com o valor existindo **somente** no processo filho de `asm-exec`:
```bash
# PROPOSTA — NAO EXECUTADA
umask 077
D=/etc/agent-brain/secrets
asm-exec -- sh -c 'printf %s "{{resolve:secretsmanager:prod/multica/omniroute-inference-key:SecretString:api_key:AWSPENDING}}" > '"$D"'/.omniroute-inference-key.new'
chmod 0600 "$D/.omniroute-inference-key.new"
chown "$(id -u):$(id -g)" "$D/.omniroute-inference-key.new"
# validacao SEM ler valor: apenas metadados e forma
test "$(stat -c %a "$D/.omniroute-inference-key.new")" = 600
test "$(stat -c %u "$D/.omniroute-inference-key.new")" = "$(id -u)"
S=$(stat -c %s "$D/.omniroute-inference-key.new"); test "$S" -ge 8 -a "$S" -le 4096
LC_ALL=C grep -qP '[\s\x00-\x1f]' "$D/.omniroute-inference-key.new" && { echo "conteudo invalido"; exit 1; }
# swap atomico
mv -f "$D/.omniroute-inference-key.new" "$D/omniroute-inference-key"
```
Notas de projeto, cada uma amarrada a uma regra do consumidor:
- `umask 077` **antes** de criar, para o arquivo nunca existir com bit de grupo/mundo, nem por um
  instante - o consumidor rejeita `& 0o077 != 0` (`:69-72`).
- o temporario fica **no mesmo diretorio**, senao `mv` cruza filesystem e deixa de ser atomico.
- `.omniroute-inference-key.new` com ponto inicial, e **nunca** com o nome final ate estar pronto.
- as quatro validacoes espelham exatamente o que o consumidor exige, e **nenhuma delas imprime o
  valor**: `stat` le metadado, e o `grep -q` devolve apenas codigo de saida.
- `mv -f` faz `rename(2)`; **nao** usar `cp` sobre o destino, que trunca.
- **nao** usar `ln -s`: `O_NOFOLLOW` rejeita symlink (`:32-34`, `:53-54`).
- o arquivo antigo pode ser preservado com `cp -a` para `omniroute-inference-key.prev` em `0600`
  **antes** do swap, para viabilizar rollback (secao 8). Isso duplica o segredo em disco: e trade-off
  deliberado, e o `.prev` deve ser removido ao fim da janela.

`version-stage` `AWSPENDING` na referencia e o que permite preparar sem promover, conforme a tabela
de sintaxe da skill.

---

## 5. CICLO DE VIDA PROPOSTO (5 estagios)

| estagio | acao | mutacao? | autorizacao |
|---|---|---|---|
| L0 | criar/atualizar o segredo em `AWSPENDING` com geracao **server-side** | AWS | **1** |
| L1 | canario: uma requisicao de leitura ao gateway usando a chave nova por `asm-exec`, sem tocar o arquivo | nenhuma no host | **2** |
| L2 | swap atomico do arquivo (secao 4) | filesystem | **3** |
| L3 | validacao pos-swap (secao 6) | nenhuma | - |
| L4 | promover `AWSPENDING` -> `AWSCURRENT` e revogar a chave antiga no provider | AWS + provider | **4** |

Ordem importa: **L1 antes de L2.** Testar a chave nova **antes** de instalar evita instalar credencial
invalida e transformar o daemon em `401` continuo. Como a leitura e por chamada, L2 nao precisa de
reload nem restart - ver secao 7.

Nao existe "fase de drenagem" a esperar, porque nao ha cache de credencial no consumidor. O unico
tempo a respeitar e o das requisicoes em voo, que e da ordem do `requestTimeout` do cliente.

---

## 6. GATES DE FILA (4 ESTADOS) E VALIDACAO

### 6.1 Gate de fila, com os quatro estados ativos
Derivado da constraint em `server/migrations/109_agent_task_waiting_local_directory.up.sql:13-15`,
que lista 7 status dos quais 3 sao terminais:
```sql
SELECT count(*) AS active_tasks
FROM agent_task_queue
WHERE status IN ('queued', 'dispatched', 'running', 'waiting_local_directory');
```
Aceite: `active_tasks = 0`, com saida real anexada. `dispatched` e `waiting_local_directory` **nao
podem** ser omitidos - foi o defeito do GTL-46 e reapareceu no runbook V1 do ORQ-33.

Justificativa especifica para ORQ-44: uma task `running` esta com processo de CLI vivo que pode fazer
requisicao ao gateway a qualquer instante. O swap atomico protege contra leitura parcial, mas **nao**
contra o provider rejeitar a chave antiga em L4.

### 6.2 Validacao pos-swap, sem ler o valor
```
V1 metadados: stat -c '%a %u %s' do arquivo final  -> esperado 600, UID do processo, 8..4096
V2 daemon saudavel:  curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:19514/health -> 200
V3 readiness do gateway inclui protocolo/modelo selecionados (fail-closed estrito)
V4 uma task de smoke nao-inferencia conclui sem 401
V5 log do daemon sem novas linhas de classe authentication apos o swap
```
Criterio negativo obrigatorio: **nenhuma** ocorrencia nova de `ErrorAuthentication` ou
`ErrorAuthorization` do lado do gateway. Um `401` apos o swap significa chave nova invalida ou
provider ja revogou a antiga - parar e ir para rollback.

---

## 7. RELOAD OU RESTART - NAO E NECESSARIO, E ISSO E MEDIDO

`FileCredentialSource` le **por chamada** e o comentario do codigo e explicito: "reads a restricted
secret file at call time ... The value is never cached". Nao existe `sync.Once`, nao existe cache em
memoria, nao existe TTL.

Portanto:
- **nao** e preciso `systemctl --user restart multica-daemon-orq2-credential.service`;
- **nao** e preciso recriar container;
- a chave nova entra em vigor na **proxima** chamada ao gateway.

Contraste com o ORQ-33 (`JWT_SECRET`): la o segredo e capturado em `jwtSecretOnce.Do`
(`internal/auth/jwt.go:31`), logo **exige** processo novo. Sao mecanismos opostos, e confundi-los
levaria a um restart desnecessario do daemon - que, pelo T3, custou 6 tasks `runtime_offline` e 1
`runtime_recovery` na ultima troca.

Ressalva honesta: se a **referencia** (`AGENT_BRAIN_GATEWAY_SECRET_FILE`) mudar de caminho, isso e
mudanca de ambiente da unit e **exige** restart. Trocar o **conteudo** nao exige; trocar o **caminho**
exige. O desenho acima troca conteudo.

---

## 8. ROLLBACK

Como nao ha cache, o rollback e simetrico ao swap e igualmente barato:
1. `mv -f $D/omniroute-inference-key.prev $D/omniroute-inference-key` - `rename(2)` atomico de volta;
2. reexecutar V1 a V5 da secao 6.2 esperando os mesmos criterios;
3. **nao** promover `AWSPENDING` (L4 nao aconteceu ainda);
4. se L4 **ja** aconteceu e o provider revogou a antiga, o rollback de arquivo **nao resolve**: e
   preciso reemitir chave no provider. Por isso L4 e o **ultimo** estagio e tem autorizacao propria.

Regra dura: **nunca deletar o segredo antigo no Secrets Manager antes de V1..V5 passarem**. Usar
`delete-secret` com janela de recuperacao de 7 dias, nunca `--force-delete-without-recovery`.

Ressalva herdada e mantida: rollback **nunca** reaplica branch antiga nem copia bruta de diretorio de
credencial - a arvore AGY antiga com `cli.log` reintroduziria o AGY task-incapaz e antecede o
hardening `0600`/`O_NOFOLLOW`.

---

## 9. AUDITORIA

- **CloudTrail:** confirmar que nenhuma identidade de agente chamou `GetSecretValue` diretamente e que
  toda resolucao veio da role autorizada via `asm-exec`. A skill lembra que ela e defesa best-effort,
  nao fronteira: CloudTrail e o controle que fecha isso.
- **Host:** `stat` do arquivo antes e depois (modo, dono, tamanho, mtime). Nunca `cat`, nunca `head`,
  nunca `xxd`.
- **Daemon:** contagem de erros de classe authentication antes e depois, sem imprimir credencial. Os
  `credentialFileError` sao content-free por construcao (`:111-119`), logo o log e seguro de anexar.
- **Evidencia a anexar:** saida do gate de fila, os cinco resultados de V1..V5 e os dois `stat`.
  **Nenhum valor de segredo, nenhum trecho de arquivo.**

---

## 10. DISTINCAO DE ORQ-37 (MCP) E DO TOKEN `mdt_`

Tres segredos distintos que **nao** devem ser rotacionados juntos nem confundidos:

| # | segredo | consumidor | mecanismo | rotacao |
|---|---|---|---|---|
| 1 | **chave de inferencia OmniRoute** (este cartao) | daemon -> gateway, via `FileCredentialSource` | arquivo `0600`, lido **por chamada**, `O_NOFOLLOW` | swap atomico de arquivo, **sem restart** |
| 2 | **autorizacao MCP** (`ORQ-37`, UUID `36d18727-f516-4147-9b0c-1cb7c2b91e83`) | caminho de MCP/Authorization, escopo do proprio ORQ-37 | fora deste cartao | **nao** tratada aqui |
| 3 | **daemon token `mdt_`** | `middleware/daemon_auth.go` | `"mdt_" + 40 hex` de `crypto/rand` (`internal/auth/jwt.go:70-76`), validado **contra o banco** | reemissao de token, nao rotacao de chave |

Pontos que evitam erro operacional:
- o `mdt_` **nao** e assinado com `JWT_SECRET` nem com a chave do gateway: e valor aleatorio validado
  no banco. Rotacionar a chave do gateway **nao** invalida `mdt_`; rotacionar `JWT_SECRET` tambem
  nao invalida `mdt_`, mas quebra o **JWT** do mesmo middleware.
- inversamente, rotacionar a chave do gateway **nao** desloga usuario nem derruba WebSocket - nada em
  `middleware/`, `handler/auth.go` ou `realtime/hub.go` consome `AGENT_BRAIN_GATEWAY_SECRET_FILE`.
  O raio de ORQ-44 e **somente** a admissao de inferencia.
- ORQ-37 tem UUID proprio e cartao proprio; qualquer passo que toque MCP sai do escopo de ORQ-44 e
  deve ser escalado, nao absorvido.

---

## 11. CONDICOES DE PARADA

Paro e escalo, sem executar, se:
1. faltar autorizacao escrita do owner para L0, L1, L2 ou L4;
2. o operador do OmniRoute nao responder as tres perguntas da secao 3, sobretudo se ha sobreposicao;
3. o gate de fila nao fechar em `0` nos **quatro** estados;
4. qualquer validacao de metadado de V1 falhar;
5. aparecer `401`/`403` novo apos o swap;
6. qualquer passo exigir imprimir o valor do segredo;
7. o caminho da referencia precisar mudar - isso muda a unit e vira outro cartao.

---

## 12. NAO-AFIRMACOES
- READ-ONLY: **nenhum segredo lido**, zero `GetSecretValue`/`BatchGetSecretValue`, **zero execucao de
  `asm-exec`**, zero mutacao de codigo, board, config, container, provider ou AWS. Nenhum comando
  deste documento foi executado.
- Nao inspecionei o arquivo `/etc/agent-brain/secrets/omniroute-inference-key`, nem por `stat`: nao
  sei seu modo, dono, tamanho ou existencia atual no host. As exigencias citadas vem do **codigo do
  consumidor**, nao de medicao do arquivo.
- Nao sei se o segredo `prod/multica/omniroute-inference-key` existe no Secrets Manager: o nome e
  **proposto**, e nao consultei o Secrets Manager.
- **Nao sei se o OmniRoute suporta sobreposicao de duas chaves.** As tres perguntas da secao 3 sao
  bloqueantes e nao foram respondidas.
- Nao verifiquei `checkCredentialOwner` nem `openCredentialFile` nos arquivos com build tag de
  plataforma; li o contrato pelos comentarios e pelas chamadas em `credential_file_source.go`.
- Nao verifiquei o comportamento do provider diante de chave revogada em requisicao em voo.
- Nao executei o gate de fila nem nenhuma validacao V1..V5.
- Nao toquei em ORQ-37 nem em qualquer artefato de MCP.
- Nao criei, atribui nem comentei issue alguma; o freeze de assignment e comentario permanece
  respeitado.
- Nao me auto-aprovo: este documento e **proposta** e pede review independente.

---
---

# EMENDA OBRIGATORIA 1 (ORQ-44) - fecha os dois achados de peer review

- **Autor:** Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T15:50Z
- **Cartao:** `ORQ-44` · emenda determinada pelo **owner**
- **Skill recarregada integralmente antes desta emenda:** `.agents/skills/aws-secrets-manager/SKILL.md`
  (v1) - R1 sem `get-secret-value`/`batch-get-secret-value`, R2 sem SMA em `localhost:2773`, R3 apenas
  `{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}` via `asm-exec`, com
  o valor existindo **somente no processo filho**; e a advertencia da propria skill de que ela e
  **"best-effort defense, not a security boundary"**, a combinar com IAM least-privilege, CloudTrail e
  VPC endpoint policies.
- **STATUS: PROPOSTA, aguardando re-review independente.** Nao me auto-aprovo.
- **Modo:** READ-ONLY. Nesta emenda **nao** li segredo, **nao** fiz `stat` do arquivo de segredo alvo,
  **nao** chamei provider nem AWS, **nao** troquei arquivo, **nao** reiniciei nada, **nao** mutei
  board, **nao** compilei e **nao** implementei.

## E1. ACHADO 1 - porta de health: **nunca** fixar 19514

O corpo original usava `http://127.0.0.1:19514/health` em `V2` da secao 6.2. **Isso esta errado e e
retirado.** Medido:

`internal/daemon/config.go:58` e `:88` (literal):
```go
	DefaultHealthPort              = 19514
	HealthPort                     int                   // local HTTP port for health checks (default: 19514)
```
`internal/daemon/config.go:571-573`:
```go
	healthPort := DefaultHealthPort
	if overrides.HealthPort > 0 {
		healthPort = overrides.HealthPort
	}
```
`internal/daemon/health.go:63`:
```go
	addr := fmt.Sprintf("127.0.0.1:%d", d.cfg.HealthPort)
```
Ou seja **19514 e apenas o default**, sobreponivel por `overrides.HealthPort`. Em host com mais de um
perfil de daemon, cada instancia pode escutar em porta diferente, e checar 19514 as cegas pode:
1. dar `200` de **outro** daemon e produzir falso PASS; ou
2. dar recusa de conexao com o daemon-alvo saudavel em outra porta, produzindo falso BLOCK.

### E1.1 Regra nova (substitui a linha `V2` da secao 6.2)
A porta **tem de ser derivada do perfil alvo, ou exigida como parametro**, e a **identidade do daemon
tem de ser provada** antes de o `200` valer como evidencia.

Derivacao, em ordem de preferencia, sem `stat` do segredo e sem ler valor:
```
D1. porta declarada explicitamente pelo operador para o perfil alvo (parametro obrigatorio); ou
D2. derivada da unit do perfil: a linha ExecStart e o Environment da unit systemd do daemon alvo,
    lidos por `systemctl --user cat <unit>` (metadado de unit, nao segredo); ou
D3. derivada do processo: a porta em LISTEN pertencente ao PID do daemon alvo, por
    `ss -ltnp` correlacionado ao PID, ou pelo inode do socket.
Se nenhuma das tres resolver -> PARAR. Nao presumir 19514.
```

### E1.2 Prova de identidade do daemon (obrigatoria, dois caminhos aceitos)
```
I1. por daemon_id: o daemon alvo declara --daemon-id; confirmar que o daemon_id do perfil alvo e o
    esperado (na topologia atual, `orq2-credential-runtime-v1`), lendo o ExecStart da unit e/ou o
    /proc/<pid>/cmdline. O `200` so conta se vier da instancia com esse daemon_id.
I2. por PID do socket: obter o PID que possui o socket em LISTEN daquela porta e confirmar que e o
    MainPID da unit alvo (`systemctl --user show <unit> -p MainPID`). Se o PID do socket difere do
    MainPID, o `200` e de outro processo e NAO vale.
```
Aceite do gate de saude passa a ser: **`200` na porta derivada E identidade confirmada por I1 ou I2.**
Um sem o outro nao e aceito.

Nenhum desses passos toca o arquivo de segredo, nem le valor, nem chama provider.

## E2. ACHADO 2 - UNIX-only, com parada em plataforma nao-Unix

Medido, `internal/daemon/credential_file_source_windows.go:9-18` (literal):
```go
// TOCTOU-safe equivalent of O_NOFOLLOW without Win32 API calls, and the
// gateway-required daemon is not deployed to Windows targets.
func openCredentialFile(_ string) (*os.File, error) {
	return nil, &credentialFileError{reason: "platform_unsupported"}
}

// checkCredentialOwner on Windows fails closed unconditionally.
func checkCredentialOwner(_ os.FileInfo) error {
	return &credentialFileError{reason: "platform_unsupported"}
}
```
Arquivos com build tag: `credential_file_source_unix.go` e `credential_file_source_windows.go`.

### E2.1 Declaracao normativa
**Este runbook e UNIX-only.** Em Windows, `openCredentialFile` e `checkCredentialOwner` **falham
incondicionalmente** com `platform_unsupported`, logo nao existe rotacao possivel por arquivo: o
consumidor recusa a credencial em qualquer estado do arquivo.

Gate de entrada, primeiro passo de tudo:
```
P0. Confirmar que o host alvo e Unix (Linux, na topologia atual: ORQ1/ORQ2 sao EC2 Amazon Linux).
    Se o host nao for Unix -> PARAR e escalar. Nao adaptar, nao improvisar equivalente de
    O_NOFOLLOW, nao relaxar a checagem de dono.
```
Consequencia declarada: qualquer plano futuro de rodar o daemon gateway-required em Windows exige
**mudanca de codigo** nos dois helpers, e isso e cartao proprio, fora do ORQ-44.

## E3. `.prev` - limpeza explicita, e preferencia por rollback **sem** copia em disco

O corpo original permitia `cp -a` para `omniroute-inference-key.prev`. Isso **duplica o segredo em
disco** e cria um segundo objeto a proteger e a apagar. Reordeno as opcoes:

### E3.1 PREFERIDA - rollback por *handle*, com `AWSPREVIOUS`, sem `.prev`
A propria sintaxe da skill aceita `version-stage`, cujo default e `AWSCURRENT` e que admite estagios
como `AWSPENDING`. O rollback passa a **reescrever o arquivo a partir do estagio anterior**, sem
nenhuma copia do valor antigo em disco:
```bash
# PROPOSTA — NAO EXECUTADA
set -euo pipefail
umask 077
D=/etc/agent-brain/secrets
asm-exec -- sh -c 'printf %s "{{resolve:secretsmanager:prod/multica/omniroute-inference-key:SecretString:api_key:AWSPREVIOUS}}" > '"$D"'/.omniroute-inference-key.rollback'
chmod 0600 "$D/.omniroute-inference-key.rollback"
# validacao apenas por metadado e codigo de saida (ver corpo, secao 4)
mv -f "$D/.omniroute-inference-key.rollback" "$D/omniroute-inference-key"
```
Vantagens: **zero** segredo extra em disco, e a fonte de verdade continua sendo o Secrets Manager.
Pre-condicao: o estagio `AWSPREVIOUS` tem de existir - o que **depende** de a promocao ter sido feita
por rotacao de estagio, e nao por sobrescrita. Se a promocao apagar o valor anterior, `AWSPREVIOUS`
nao existe e essa via **nao funciona**.

### E3.2 FALLBACK - `.prev`, somente se `AWSPREVIOUS` nao existir, com limpeza EXPLICITA
Se e somente se E3.1 for inviavel:
```bash
# criacao, antes do swap
install -m 0600 /dev/null "$D/omniroute-inference-key.prev"
cp -a "$D/omniroute-inference-key" "$D/omniroute-inference-key.prev"
test "$(stat -c %a "$D/omniroute-inference-key.prev")" = 600

# LIMPEZA OBRIGATORIA, no fim da janela, em trap para nao ser esquecida
trap 'rm -f "$D/omniroute-inference-key.prev" "$D/.omniroute-inference-key.new" "$D/.omniroute-inference-key.rollback"' EXIT INT TERM
```
Regras duras do fallback:
1. `.prev` existe **somente** durante a janela de rotacao; `trap ... EXIT` garante remocao mesmo em
   falha ou interrupcao;
2. `.prev` **nunca** entra em backup, artefato de evidencia, tarball ou repositorio;
3. ao fim, confirmar ausencia: `test ! -e "$D/omniroute-inference-key.prev"`;
4. `rm -f` e suficiente aqui; nao proponho `shred`, porque em filesystem com copy-on-write ou SSD
   `shred` nao garante apagamento e daria falsa seguranca.

## E4. CUSTO DO CANARIO NO PROVIDER - declarado

O estagio `L1` do corpo faz um canario com a chave nova. **Isso pode custar dinheiro** e a emenda
declara o que se sabe e o que nao se sabe:

- O que se sabe: o gateway responde `GET /v1/models` com `200` e catalogo de 327 modelos, e esse e um
  endpoint de **listagem**, nao de inferencia. Uma listagem autenticada e o canario mais barato
  possivel e, em qualquer provider razoavel, **nao** consome cota de tokens.
- O que **nao** se sabe e e pergunta bloqueante (ver E6, pergunta 3): se o OmniRoute contabiliza
  chamada de listagem em alguma cota, e se existe endpoint de introspeccao de credencial que nao
  consuma nada.
- Custo **proibido**: o canario **nao** deve emitir prompt nem completar tokens. Nenhuma chamada de
  inferencia. Se a unica forma de validar a chave for uma inferencia real, isso muda o custo do plano
  e exige decisao explicita do owner, com valor estimado - **nao** e decisao de agente.
- Teto proposto: **uma** requisicao de listagem, uma vez, com timeout curto. Se falhar, parar; nao
  repetir em loop, para nao gerar custo nem disparar rate limit.

## E5. SMOKE TASK SUBSTITUIDO POR PROBE DE READINESS (freeze de atribuicao)

O corpo original tinha, na secao 6.2, `V4 uma task de smoke nao-inferencia conclui sem 401`. **Isso
conflita com o freeze vigente**: atribuir agente a uma issue **enfileira task paga**, e comentario em
issue atribuida tambem pode enfileirar. Criar ou atribuir task de smoke seria, portanto, uma acao de
execucao disfarcada de validacao.

**Substituicao normativa de `V4`:**
```
V4 (novo) — PROBE DE READINESS, sem criar nem atribuir task:
  a) health do daemon na porta DERIVADA (E1.1) com identidade confirmada (E1.2) -> 200
  b) readiness do gateway reportando protocolo e modelo selecionados sob politica estrita
     fail-closed, conforme GatewayConfig.Validate exige em modo Required
  c) ausencia de NOVAS ocorrencias de classe authentication/authorization no log do daemon
     apos o swap
Nenhuma issue criada, nenhuma issue atribuida, nenhum comentario postado, nenhuma task enfileirada.
```
Isso preserva o valor da validacao — prova que o daemon aceita a credencial nova — sem gastar
orcamento e sem violar o freeze.

## E6. AS QUATRO PERGUNTAS AO PROVIDER/OWNER - **BLOQUEANTES E NAO RESOLVIDAS**

Reafirmadas explicitamente, e **nenhuma** foi respondida:

1. **Sobreposicao:** o OmniRoute aceita **duas** chaves de inferencia ativas ao mesmo tempo? Se sim,
   por quanto tempo e como se revoga a antiga? **NAO RESOLVIDA.**
2. **Escopo da chave:** a chave e por **conta** ou global? O gateway tem 4 contas Antigravity,
   4 ClinePass e 2 OpenAI Codex documentadas - a rotacao afeta todas ou so o par do cliente?
   **NAO RESOLVIDA.**
3. **Introspeccao e custo:** existe endpoint que valide a chave **sem consumir cota**, e a chamada de
   listagem usada no canario conta em alguma cota? **NAO RESOLVIDA.**
4. **Estagio anterior:** a promocao no Secrets Manager preserva `AWSPREVIOUS`, viabilizando o rollback
   por handle de E3.1, ou sobrescreve o valor? **NAO RESOLVIDA.**

Enquanto as quatro estiverem sem resposta, **este runbook nao pode ser executado**, nem parcialmente.
A pergunta 4 e nova nesta emenda e e o que decide entre E3.1 e E3.2.

## E7. LOGGING SEM CONTEUDO - preservado e reafirmado

Mantido do corpo, e reafirmado como invariante da emenda:
- `credentialFileError` e **content-free** por construcao
  (`internal/daemon/credential_file_source.go:111-119`): expoe apenas um token de razao, como
  `secret_file_permissions_too_open` ou `secret_file_content_too_short`, e **nunca** o valor, o
  conteudo do arquivo ou a mensagem detalhada do OS.
- Toda validacao proposta usa **metadado** (`stat -c %a/%u/%s`) ou **codigo de saida** (`grep -q`).
  Nenhum passo imprime, loga ou concatena o valor.
- `GatewayError.Detail` e token sanitizado por contrato: nunca corpo, URL ou credencial.
- Evidencia a anexar continua sendo: gate de fila, resultados de V1..V5, e `stat` antes/depois.
  **Nenhum valor de segredo, nenhum trecho de arquivo, nenhum token.**
- A emenda **nao** introduz nenhum log novo, e proibe explicitamente `cat`, `head`, `tail`, `xxd` ou
  `od` sobre o arquivo alvo.

## E8. NAO-AFIRMACOES DESTA EMENDA
- Nesta emenda **nao** li segredo, **nao** fiz `stat` do arquivo de segredo alvo, **nao** chamei
  provider nem AWS, **nao** troquei arquivo, **nao** reiniciei nada, **nao** mutei board, **nao**
  compilei e **nao** implementei.
- Nao derivei a porta real de nenhum perfil: E1.1 e **procedimento**, e eu nao executei D1, D2 nem D3.
- Nao confirmei o `daemon_id` nem o PID de socket de nenhuma instancia nesta rodada.
- Nao verifiquei se `AWSPREVIOUS` existe para o segredo proposto - e a pergunta bloqueante 4.
- Nao medi custo de nenhuma chamada ao provider; a analise de E4 e sobre a **natureza** do endpoint
  (listagem versus inferencia), nao sobre fatura observada.
- Nao verifiquei o conteudo de `credential_file_source_unix.go`: a conclusao UNIX-only vem do arquivo
  **windows**, que falha incondicionalmente, e da existencia dos dois arquivos com build tag.
- Nao removi nem criei nenhum `.prev`; E3 e desenho.
- Nao criei, atribui nem comentei issue alguma, e nao enfileirei task.
- Nao me auto-aprovo: **PROPOSTA aguardando re-review independente.**
