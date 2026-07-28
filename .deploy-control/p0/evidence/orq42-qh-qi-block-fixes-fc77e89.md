# ORQ-42 Q-H/Q-I - correcao do BLOCK de `cebc96a` (commit `fc77e89`)

- autor/executor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-28T13:40Z
- branch: `agent/opus48-a/orq42-secret-tools-clean`, **commit novo `fc77e89`** sobre `cebc96a`
- **sem amend, sem rewrite**; **branch original `agent/opus48-a/orq42-secret-tools` PRESERVADO** (H1:
  so sera descartado depois que o clean estiver integrado, e nunca por mim sem ordem)
- **PEDIDO: RE-REVIEW.** Nada de push, PR, merge, quadro, AWS, segredo, Docker ou alvo real.

## (1) `wsprobe` - loopback numerico obrigatorio, **antes** de ler stdin

`requireNumericLoopback` exige `net.ParseIP(host) != nil && ip.IsLoopback()`, e e chamado **antes** de
`readToken`. Portanto um token **nunca** e consumido para um destino recusado - o teste prova isso
passando **stdin vazio**: se a ordem fosse a inversa, o erro seria `WS_STOP_TOKEN` em vez de
`WS_STOP_TARGET`.

Recusados (exit 2, `WS_STOP_TARGET`): `example.com`, **`localhost`** (e um **nome**, nao um endereco -
pode resolver para qualquer lugar, agora ou depois), `10.0.0.5`, `100.118.244.61` (o IP real do ORQ1 na
tailnet) e `[2001:db8::1]`. Aceitos pelo guard: `127.0.0.1` e `[::1]`. Como o probe fala **HTTP em
texto claro**, so loopback e admissivel - sem TLS, qualquer outro destino poria o token na rede.

## (2) Contrato de saida estrito - `exit 3` so para rejeicao **de credencial**

| resultado | label | exit |
|---|---|---|
| cookie `101` | `WS_ACCEPTED` | **0** |
| first-frame `{"type":"auth_ack"}` | `WS_AUTH_ACK` | **0** |
| cookie `401` | `WS_REJECTED` | **3** |
| first-frame erro **exatamente** `invalid token` | `WS_REJECTED` | **3** |
| `400` | `WS_STOP_BAD_REQUEST` | 1 |
| `403` | `WS_STOP_FORBIDDEN` | 1 |
| outro status | `WS_STOP_STATUS` | 1 |
| frame `close` antes do veredito | `WS_STOP_CLOSED` | 1 |
| conexao encerrada | `WS_STOP_EOF` | 1 |
| frame ilegivel | `WS_STOP_MALFORMED` | 1 |
| erro **diferente** de `invalid token` | `WS_STOP_AUTH_ERROR` | 1 |
| teto de frames / framing | `WS_STOP_PROTOCOL` | 1 |
| deadline | `WS_STOP_TIMEOUT` | 1 |

O principio: **um gate nunca pode ler "nao consegui medir" como "credencial rejeitada"**. `403` e
membership, `400` e requisicao errada, e `auth timeout or read error` nao e veredito de credencial.

**`101` so e aceito como upgrade** quando `Sec-WebSocket-Accept` **e** `Upgrade: websocket` **e**
`Connection: upgrade` conferem; um `101` pelado da `WS_STOP_HANDSHAKE`, nao aceitacao. Ha teste com um
servidor que responde `101` sem cabecalho nenhum.

## (3) `editor` - chave restrita, variantes recusadas, preflight content-free

- **`-key`** deve casar `^[A-Za-z_][A-Za-z0-9_]*$`; recusa com `STOP_KEY` **antes** de abrir o arquivo.
  Testados e recusados: vazio, `BAD KEY`, `BAD=KEY`, `bad-key`, `bad.key`, `1BAD`, `K"Q`, `K\n`.
- **variantes semanticas agora PARAM** (`STOP_VARIANT`) em vez de serem ignoradas: `export KEY=`,
  `EXPORT` em qualquer caixa, espaco/tab antes do `=`, indentacao por espaco ou tab, e o caso **misto**
  (uma linha canonica **mais** uma variante). Em todos, o arquivo fica **intacto**.
- **`-check-only`**: preflight **content-free** que **nao escreve** e **nao le stdin** - o teste
  envenena a stdin com `THIS-MUST-NOT-BE-READ` e exige sucesso. Aceita **somente** a linha canonica
  `KEY=<64hex>` (hex maiusculo tambem); recusa com `STOP_NOT_CANONICAL` valor entre aspas duplas ou
  simples, comentario inline, espaco a direita, valor curto e nao-hex; e propaga `STOP_VARIANT`,
  `STOP_NONE` e `STOP_MULTI`.

## (4) Pin do `hub.go` documentado no cabecalho do probe

`multica-auth-work/server/internal/realtime/hub.go`, sha256
`d5e2dbc654b2316aa79435c4a833827039a9323fb48500817b0cbb2e2c5a7a87`:
```text
:770-775  cookie multica_auth verificado ANTES do upgrade  -> 401 nao chega ao WebSocket
:719-728  sem cookie, o primeiro frame deve ser {"type":"auth","payload":{"token"}}
:809-816  no sucesso, {"type":"auth_ack"} e o PRIMEIRO frame apos a autenticacao
:678,:682,:694  falha de verificacao emite exatamente {"error":"invalid token"}
:716      {"error":"auth timeout or read error"}   NAO e veredito de credencial
:726      {"error":"expected auth message as first frame"}  idem
```
E porque o `auth_ack` vem primeiro, o teto de 4 frames basta - e **estourar o teto, ou EOF, e `exit 1`**.

## (5) Verificacao executada

```
gofmt -l            -> vazio nos dois modulos
go vet ./...        -> OK nos dois
go build ./...      -> OK nos dois
go test -race -count=1 -> ok / ok
testes: editor 13 + probe 11 = 24   (eram 18)
determinismo: 2 builds, caches distintos, hashes IDENTICOS
  envsecret-editor 7c5b58989a442f76226275cb474b3b5f8c5c8d1a914eede46ceb181592e552f4
  wsprobe          18dadf56f778c6172460cb2b9862c5f1fddca464ed90eb2a22743bb0b2b13e19
git diff --check    -> limpo
```
Pins de fonte novos:
```
envsecret-editor/main.go       18310375fae6f158759031ffda72a22296ef5eefa72214bc9e0682717db4af83
envsecret-editor/main_test.go  e1e5ac95b4734f1b12e1de140584fee3939408188c6cb77865d99dbf3cd61a17
wsprobe/main.go                71a2ed7fe134db3f1171dfa754102743f7ea9b1e12bbd90fd4dd08477ecd8dfb
wsprobe/main_test.go           6e3eb16acd70a960fe4fb34e10295fd3092330992599cbaf068d1291999c4837
```

## (6) Duas expectativas antigas foram **atualizadas**, e digo por que

O contrato ficou mais estrito, entao dois testes meus anteriores afirmavam o comportamento **antigo**:
uma linha indentada passou de "ausente" (`STOP_NONE`) para **variante recusada** (`STOP_VARIANT`), e uma
chave invalida passou de `STOP_USAGE` para `STOP_KEY`. Nao foram testes "consertados para passar": a
regra mudou por exigencia da review, e as duas mudancas **endurecem**, nao afrouxam.

## (7) Nao-afirmacoes

- **Nada de push, PR, merge, quadro, AWS, segredo, Docker ou alvo real.** Tudo local.
- **Nao usei amend nem rewrite**; o branch original segue intacto, conforme H1.
- **Os utilitarios continuam sem execucao contra alvo real.** O `wsprobe` so viu hubs falsos
  `httptest`; o pin do `hub.go` prova o **contrato no codigo**, nao o comportamento em execucao. O
  editor so tocou fixtures de `t.TempDir()` - **nunca** o `dev.env`.
- **A premissa do `dev.env` real continua sem confirmacao**: se o arquivo usar `export`, aspas ou
  espaco, o editor agora **para** (o que e o comportamento desejado) - mas isso significa que o
  `-check-only` **precisa** rodar contra o arquivo real, sob autorizacao, antes de qualquer janela.
- O `wsprobe` **nao** faz TLS por desenho; com o guard de loopback isso deixou de ser lacuna e passou a
  ser invariante imposta.
- Nao executei nada da Q-C e nao sei se o perfil `owner-p0` existe.
- ORQ-26 segue em `HOLD_EXTERNAL_BILLING`; nao tentei contornar nem rodei `rerun`.
