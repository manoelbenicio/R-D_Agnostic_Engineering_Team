# ORQ-32 - Revisao independente do runbook de rotacao do handshake token

- revisor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T14:22Z
- auditados: `orq32-handshake-token-rotation-runbook.md` (Antigravity, veredito PASS) e
  `orq32-handshake-token-rotation-runbook-peer-review.md` (Agy-P0-A8, veredito BLOCK)
- card: **ORQ-32**, UUID `b01925fe-e914-422a-812e-f63cada274dc`, number 32, status `todo`,
  **sem assignee**, titulo "Security Wave B: Handshake Token Rotation & Lifecycle" - par UUID+numero
  **verificado** por `GET /api/issues?workspace_slug=orq2-dev` (o UUID citado no peer review confere)
- modo: READ-ONLY. **Nenhum segredo lido, resolvido, impresso ou tocado. Nenhuma rotacao executada.**
  Zero chamada a `get-secret-value` ou `batch-get-secret-value`. Nenhuma mutacao de codigo, AWS,
  board ou config.

---

## VEREDITO: **BLOCK mantido, e agravado**

O BLOCK do peer review esta **correto na direcao**, mas **subestima o defeito 1** e **corrige pela
metade o defeito 3**. Encontrei tambem **dois defeitos de sintaxe de segredo que nenhum dos dois
documentos viu**, sendo um deles uma **violacao da propria regra de ouro do runbook**.

---

## 1. Correcao BLOCK 1 - "falta suporte a dual-token": **subestimada**

O peer review afirma que falta `HANDSHAKE_TOKEN_SECONDARY`, o que **implica** que o primario existe.
**Nao existe nenhum dos dois.** Medido:

```
grep -rniE "handshake_token|HANDSHAKE_TOKEN|MULTICA_HANDSHAKE|CLOUD_FLEET_HANDSHAKE" \
  --include=*.go server/  (sem testes)   ->  ZERO ocorrencias
```
E `grep -rni "handshake" --include=*.go` no server retorna **apenas** usos de TLS/WebSocket, sem
relacao com autenticacao de token:
```
cmd/server/dbstats.go:33          "pods don't pay handshake cost on first traffic"
cmd/server/router.go:343          "for the QR-scan handshake"
internal/cli/client.go:94         TLS handshake
internal/cli/errors.go:31         KindNetworkTLS  // x509 / tls handshake failures
internal/daemon/gateway/client.go:127   transport.TLSHandshakeTimeout
internal/daemon/wakeup.go:95            websocket.Dialer{HandshakeTimeout: ...}
internal/integrations/lark/ws_connector.go:73,550   HandshakeTimeout
```
**Consequencia**: a tabela de consumidores da secao 2 do runbook - `HANDSHAKE_TOKEN_PRIMARY`,
`HANDSHAKE_TOKEN_SECONDARY`, `MULTICA_HANDSHAKE_TOKEN`, `CLOUD_FLEET_HANDSHAKE_TOKEN` - **nao
corresponde a nenhuma variavel lida por este codigo**. Nao e "falta a versao secundaria": e um
runbook para rotacionar um segredo que **este software nao consome**.

O mecanismo real de autenticacao de daemon e outro: `internal/auth/jwt.go:70-71`
`GenerateDaemonToken` produz `"mdt_" + 40 hex`, e o daemon carrega `MULTICA_TOKEN` no ambiente
(observado no unit do daemon do ORQ2). Ou seja, se o objetivo do ORQ-32 e rotacionar o segredo que
autentica daemon contra servidor, o alvo correto e o **daemon token `mdt_`**, nao um
`handshake_token`.

**Correcao exigida, mais forte que a do peer review**: antes de qualquer patch de dual-token, o
runbook precisa **identificar o segredo real** que pretende rotacionar e provar, com arquivo:linha,
qual variavel o codigo le. Se o alvo for o `mdt_`, o runbook inteiro - consumidores, fases e rollback
- precisa ser reescrito sobre esse mecanismo. Escrever a PR de dual-token para
`HANDSHAKE_TOKEN_SECONDARY` seria implementar suporte a um nome que ninguem usa.

## 2. Correcao BLOCK 2 - sequenciamento: **correta, e insuficiente pelo motivo acima**

Exigir deploy do Go com dual-token **antes** de mexer em estagio do Secrets Manager e correto e eu
mantenho. Mas com o item 1, o sequenciamento correto ganha um passo **zero**: identificar o segredo e
o consumidor reais. Sem isso, a ordem certa aplicada ao alvo errado nao produz seguranca.

## 3. Correcao BLOCK 3 - gate de fila: **corrigida pela metade**

- Runbook: `status = running`. Insuficiente, e o peer review acertou em apontar.
- Peer review: `status IN ('queued','running')`. **Ainda incompleto.**
- Correto: os **quatro** estados ativos, conforme
  `migrations/109_agent_task_waiting_local_directory.up.sql:15`, cujo CHECK e
  `('queued','dispatched','running','waiting_local_directory','completed','failed','cancelled')`:

```sql
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```
Faltam `dispatched` e `waiting_local_directory`. Uma task `dispatched` **ja foi entregue ao daemon** e
esta a um passo de rodar; uma `waiting_local_directory` esta viva aguardando path. Reiniciar daemon
com qualquer uma delas presente interrompe trabalho pago. E o mesmo predicado incompleto que eu ja
apontei no GTL-32 e no GTL-43 - e a terceira vez que ele aparece, o que sugere padronizar uma unica
constante/consulta em vez de reescrever a lista em cada runbook.

Observacao adicional sobre o Gate 1: ele exige fila zero **antes do reinicio do daemon**, mas nao
diz **como impedir que uma nova task entre** entre a verificacao e o reinicio. Sem pausar a admissao,
o gate e uma leitura instantanea sem garantia. Precisa de um mecanismo de drain/pausa, ou aceitar
explicitamente a janela de corrida.

## 4. DEFEITO NOVO A - `{{resolve:...:VersionId:...}}` nao existe

Fase 4 do runbook:
```bash
--move-to-version-id {{resolve:secretsmanager:prod/handshake-token:VersionId:AWSPENDING}}
--remove-from-version-id {{resolve:secretsmanager:prod/handshake-token:VersionId:AWSCURRENT}}
```
A sintaxe suportada, conforme `.agents/skills/aws-secrets-manager/SKILL.md:41-49`, e
`{{resolve:secretsmanager:<secret-id>:<field-type>:<json-key>:<version-stage>}}`, e a tabela lista
`field-type` com **um** valor possivel, `SecretString` (default). **`VersionId` nao e um
`field-type`**, e `asm-exec` resolve **valores de segredo**, nao metadado de versao.

Efeito pratico: o `asm-exec` nao resolve a referencia, e o `aws secretsmanager
update-secret-version-stage` recebe a string literal `{{resolve:...}}` como `--move-to-version-id`.
Melhor caso, erro de validacao; pior caso, o operador "corrige" colando um VersionId a mao e
transforma a Fase 4 num passo manual sem trilha. O ID de versao e **metadado, nao segredo** - deve ser
obtido por `aws secretsmanager list-secret-version-ids` (que **nao** retorna valor) e passado
normalmente, sem `asm-exec`.

## 5. DEFEITO NOVO B - a Fase 1 viola a regra de ouro do proprio runbook

Fase 1:
```bash
asm-exec -- aws secretsmanager put-secret-value --secret-id prod/handshake-token \
  --secret-string "{"token":"{{generate:token}}"}" --version-stages AWSPENDING
```
Tres problemas num unico comando:
1. **`{{generate:token}}` nao existe.** Nao ha diretiva de geracao na skill - so `resolve`. O
   placeholder ficaria literal, gravando o **texto** `{{generate:token}}` como token novo. Isso nao
   falha ruidosamente: cria um segredo previsivel e publico.
2. **Valor de segredo em `argv`.** A secao 1.1, regra 2, do proprio runbook proibe: *"Valores de
   segredos nunca devem ser ... passados em `argv` de processos"*. `--secret-string` com o valor
   inline e exatamente isso, e apareceria em `ps`, em history e em log de auditoria de comando.
3. **JSON malformado.** `"{"token":"..."}"` tem aspas internas nao escapadas; o shell entrega
   `{token:...}` quebrado.

Correcao: gerar o novo valor **dentro** do processo filho, sem passar por argv, por exemplo
`aws secretsmanager put-secret-value --secret-string fileb:///dev/stdin` alimentado por um gerador
local, ou usar a rotacao gerenciada do proprio Secrets Manager (funcao de rotacao Lambda), que nunca
expoe o valor a quem dispara. Qual das duas e decisao de arquitetura, nao minha.

## 6. O que o runbook acertou, e eu confirmo

- **Aderencia a skill de segredos no eixo de leitura**: nao ha `get-secret-value` nem
  `batch-get-secret-value` em nenhum comando; o uso de `asm-exec --` e de
  `{{resolve:secretsmanager:...}}` para **consumo** esta correto, e o `version-stage`
  (`AWSCURRENT`/`AWSPENDING`/`AWSPREVIOUS`) e suportado pela skill (linha 49). O peer review acertou
  ao dar PASS nesse eixo - com a ressalva da secao 5, que e eixo de **escrita**, nao de leitura.
- **`asm-exec` existe de fato**: `/home/ec2-user/.local/bin/asm-exec`.
- **Dual-token como estrategia** de corte sem downtime e o desenho certo, e o rollback via
  `AWSPREVIOUS` com servidor ainda em modo dual e coerente.
- **Gate 3** (`/readyz` 200) e valido: a rota existe (`cmd/server/router.go:444`).
- **Gate 2** (IAM) e pertinente, e observo que ele deve validar a permissao da **role que executa o
  `asm-exec`**, nao a do agente - e nao verifiquei nenhuma das duas, porque IAM falha fechado e testar
  permissao exigiria chamada que eu nao faco.

## 7. Correcoes exigidas para virar PASS

1. **C1** identificar o segredo e o consumidor **reais**, com arquivo:linha. `handshake_token` nao e
   lido por este codigo; o candidato provavel e o daemon token `mdt_` (`internal/auth/jwt.go:70-71`).
   Sem isso, todo o resto e sobre um alvo inexistente.
2. **C2** declarar a dependencia de PR em Go para dual-token **do segredo correto**, e exigir deploy
   antes de qualquer mudanca de estagio (mantem a correcao 1 e 2 do peer review, reancorada).
3. **C3** Gate 1 com os **quatro** estados ativos, e definir como a admissao e pausada durante a
   janela entre verificacao e reinicio.
4. **C4** remover `{{resolve:...:VersionId:...}}`; obter VersionId por
   `list-secret-version-ids` (metadado, sem `asm-exec`).
5. **C5** reescrever a Fase 1 sem valor de segredo em `argv` e sem `{{generate:token}}`; preferir
   rotacao gerenciada ou geracao dentro do processo filho.
6. **C6** confirmar que o segredo `prod/handshake-token` **existe** e em qual conta/regiao - eu **nao
   verifiquei**, porque exigiria chamada AWS que esta fora do meu escopo read-only autorizado, e
   porque o nome provavelmente muda por causa de C1.

## 8. Perguntas em aberto (declaradas, nao afirmadas)

- se existe algum consumidor de `handshake_token` **fora** de `multica-auth-work/server` (outro repo,
  imagem do gateway, OmniRoute): varri o server, nao a frota inteira.
- se o `prod/handshake-token` existe no Secrets Manager: nao consultei.
- se a role que roda `asm-exec` tem a permissao do Gate 2: nao testei.
- se o "Multica Fleet Gateway" da secao 2 e um componente real deste produto: nao encontrei
  `CLOUD_FLEET_HANDSHAKE_TOKEN` no codigo.

## 9. Nada mutado

Nenhum segredo lido, resolvido, impresso, salvo ou passado em comando. Zero
`get-secret-value`/`batch-get-secret-value`. Nenhuma rotacao, promocao de estagio, mutacao de AWS,
codigo, board ou config. Nao alterei assignee nem postei comentario (freeze vigente). Somente leitura
de dois documentos, de fonte Go, da skill `aws-secrets-manager` e um `GET` de issue para verificar o
par UUID+numero do ORQ-32.
