# ORQ-26 - revisao READ-ONLY do diff exato `d2447183` (handler/file.go + workflow)

- revisor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T17:36Z
- commit: `d2447183c5fb7d93c07ca4afdd86d4ab9ac374ee`, branch `ci/orq26-db-gate`, worktree
  `/home/ec2-user/workspace/worktrees/ci-orq26-db-gate` (HEAD = `d244718`, **0 linhas sujas**)
- escopo GATE 0: revisao de codigo do diff exato, com enfase em `handler/file.go` e em
  **seguranca/falso-verde** do workflow. Worktree principal sujo **fora de escopo**, nao o toquei.
- modo: **READ-ONLY**. Zero push, zero PR, zero `gh auth`, zero mutacao. Nao compilei, nao rodei teste.

---

## VEREDITO: **PASS para o push do gate** / **BLOCK para merge** ate aplicar F1 e F2

O workflow e o harness anti-falso-verde mais forte que revi nesta frota, e nao expoe segredo. O que
reprova nao e o gate: sao **duas** falhas no caminho de limpeza introduzido em `file.go`, uma delas
**destrutiva**. Como a acao autorizada e apenas rodar o gate num branch descartavel - sem merge, sem
deploy - o push pode prosseguir; o merge nao.

## 1. Seguranca do workflow - **PASS**

Verificado linha a linha:
- `permissions: contents: read` no workflow **e** no job; `persist-credentials: false` no checkout;
- **nenhum** `secrets.*` referenciado, **nenhum** upload de artefato, **nenhum** `pull_request_target`;
- `workflow_dispatch`/`schedule` deliberadamente ausentes, com a razao documentada (só valem no branch
  default, e o gate nao deve tocar `main`);
- acoes **pinadas por SHA** (`checkout@d23441a4` v6, `setup-go@40f1582b` v5) e Postgres **pinado por
  digest** (`pgvector/pgvector@sha256:d2ef61f4…`);
- `POSTGRES_PASSWORD: orq26_gate` e literal descartavel de service container alcancavel apenas pelo
  runner efemero, espelhando o `multica:multica` que ja existe no `ci.yml` - **nenhum segredo de
  repositorio criado ou consumido**;
- o DSN nunca vai a `argv`: `psql`/`pg_isready` usam as variaveis `PG*`; e o log do `migrate` e
  **retido** em falha justamente porque poderia ecoar o DSN;
- diretorios de gate `0700` com `umask 077`, `TMPDIR`/`GOCACHE`/`GOTMPDIR` fora do repo.

## 2. Falso-verde - **PASS**, e acima do padrao

O gate nao depende de heuristica de string: ele exige **evidencia positiva por folha**. Para cada um
dos **8** nomes esperados, o passo exige presenca em `run-names` **e** em `pass-names` **e** ausencia
em `skip-names`, mais o evento `pass` de **pacote** (`.Test==null`). Se o `TestMain` sair antes do
`m.Run`, nao existe evento nenhum e os `grep -Fxq` falham - **fail-closed**. Somam-se:
`-race -count=1`, identidade do banco provada por `select current_database()`, conjunto de migrations
conferido contra o `find migrations` real, guarda de arquivos alterados, `gofmt -l` vazio,
`git diff --check`, e fingerprint de `HEAD` **antes e depois**.

**Confirmei que os hashes congelados casam com os arquivos reais** - portanto o gate esta pinado
exatamente no conteudo que eu revi:
```
48553c6c48d4423ebfba7d5a05366c0a23ec77d27a463ae4934eda200b557161  internal/handler/file.go   MATCH
815b7d1cf12ed9ca18340f9e87813994eebb7e5153aee281125f9e5b3db9e44d  internal/handler/file_test.go MATCH
```

Limitacao registrada (nao bloqueante): a guarda de arquivos usa `git log -1 --name-only`, isto e
**apenas o ultimo commit**, e o `paths:` do trigger faz o workflow **nao rodar** se um push futuro
tocar outro arquivo. Logo "o gate passou" nao equivale a "o branch contem so estes tres arquivos".
Hoje isso e inofensivo - `d2447183` e o unico commit e `git branch --contains` o mostra so neste
branch - mas nao sustenta essa afirmacao no futuro.

## 3. 🔴 F1 - a limpeza do P1 pode apagar o **objeto errado** (destrutivo)

`file.go` novo, no caminho de falha do `CreateAttachment`:
```go
h.deleteS3Object(r.Context(), link)
```
e `deleteS3Object` (`file.go:1004-1009`) faz `h.Storage.Delete(ctx, h.Storage.KeyFromURL(url))`.
O problema esta no **fallback final** de `KeyFromURL` (`internal/storage/s3.go`):
```go
// Fallback: take everything after the last "/".
if i := strings.LastIndex(rawURL, "/"); i >= 0 { return rawURL[i+1:] }
```
Se nenhum prefixo conhecido casar, a chave derivada e **so o ultimo segmento** - por exemplo
`photo.png` em vez de `users/<uuid>/photo.png` - e o `Delete` passa a mirar um objeto na **raiz do
bucket**. Caso concreto e alcancavel: com `cdnDomain == ""`, `endpointURL == ""` **e** `region == ""`,
`uploadedURL` produz `https://<bucket>.s3..amazonaws.com/<key>` enquanto `KeyFromURL` so registra o
prefixo `https://<bucket>/` - nenhum casa, cai no fallback, e a limpeza apaga **outro** objeto.

Esse fallback e **pre-existente**; o que este commit faz e **passar a rotear um DELETE por ele**, num
caminho novo de falha. Nao e aceitavel introduzir delecao por URL quando a chave exata esta em escopo.

**Correcao exata (uma linha):** usar a chave, nao a URL.
```go
// no lugar de: h.deleteS3Object(r.Context(), link)
h.Storage.Delete(cleanupCtx, key)   // `key` ja esta em escopo, calculado antes do Upload
```
(`h.Storage != nil` ja e garantido pelo fluxo que chamou `Upload`; manter guarda se preferirem.)

## 4. 🔴 F2 - a limpeza morre exatamente no modo de falha mais provavel

A limpeza usa `r.Context()`. Se o `CreateAttachment` falhou **porque** o contexto foi cancelado - o
cliente desconectou, o caso correlacionado mais comum -, o contexto ja esta `Done` e o `Delete` falha
de imediato: o objeto orfao **permanece**, que e precisamente o que o P1 existe para evitar.

**Correcao exata:**
```go
cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
defer cancel()
h.Storage.Delete(cleanupCtx, key)
```
`context.WithoutCancel` esta disponivel (Go 1.21+; o gate fixa Go 1.26.1).

Nota: F2 nao produz falso sucesso - o `500` continua sendo emitido, e o teste
`TestUploadFile_InsertFailureStillFailsWhenCleanupIsNoop` mostra que a autoria considerou a limpeza
no-op. O dano e higiene de dados, nao resposta enganosa.

## 5. O que **passa** em `file.go`

- **P3 esta no lugar certo**: a rejeicao de `issue_id`/`comment_id`/`chat_session_id` sem workspace
  ocorre **antes** de qualquer `Storage.Upload`, nos dois ramos, e devolve `400`. Fecha o buraco real
  de 200-com-objeto-orfao-sem-linha. E `r.ParseMultipartForm(maxUploadSize)` ja rodou em `:330`,
  portanto o `r.FormValue` do guard **nao** dispara re-parse com o default de 32 MiB, e usa o mesmo
  acessor dos ramos `:420/:435/:447` - consistente.
- **P2 nao vaza assinatura.** Testei a hipotese e ela e **falsa**: `uploadedURL` compoe URL estatica
  (CDN > endpoint > host regional) e **nunca** assina, logo `download_url`/`markdown_url` nao carregam
  query de assinatura. Residual sem acao: em deployment de bucket privado sem CDN, o link e
  `unauth-deny` e agora tambem entra em `markdown_url`, podendo ser **persistido em conteudo**. Isso
  troca um `404` por um `403`, e a propria docstring do arquivo define um predicado de 3 condicoes
  para quando uma URL crua pode ser persistida - vale aplicar o mesmo predicado a `markdown_url`.
- P1, quanto a **resposta**, esta correto: deixar de devolver `200` com linha inexistente e o conserto
  certo, dado que o cliente agora valida fail-closed (`ApiContractError`).

Observacao menor: `r.FormValue` tambem consulta a **query string**, portanto um upload sem workspace
com `?issue_id=x` passa a receber `400`. E rejeicao a mais, nao a menos - registro so como mudanca de
comportamento visivel ao cliente.

## 6. Consequencia operacional das correcoes

F1/F2 mudam `file.go`, logo **invalidam** `LOCKED_FILE_SHA256` (e, se os testes cobrirem a nova
chamada, `LOCKED_TEST_SHA256`). O ciclo correto e: (a) rodar o gate **como esta**, para provar que os
testes executam contra Postgres real; (b) aplicar F1/F2 em commit novo; (c) atualizar os dois hashes
congelados; (d) rodar o gate de novo; (e) so entao PR/merge.

## 7. Nao-afirmacoes

- Zero push, zero PR, zero `gh auth`, zero mutacao. **Nao compilei e nao executei nenhum teste**
  (`go build`/`go vet`/`go test` nao foram rodados nesta revisao).
- Nao revisei `file_test.go` linha a linha; avaliei os **nomes exigidos** pelo workflow e a mecanica de
  asseveracao, nao o corpo dos testes - portanto **nao** afirmo que os 8 testes provam o que dizem.
- Nao verifiquei se `region` pode de fato ser vazio em producao; descrevi o caso como **alcancavel pela
  leitura do codigo**, nao medido.
- Nao consegui contar os commits a frente da base (o `ls-remote` depende da credencial ausente);
  confirmei apenas que `d2447183` esta contido **so** em `ci/orq26-db-gate` e que HEAD = `d244718`.
- Nao avaliei o worktree principal sujo, por instrucao.
- Nao alterei assignee, nao postei comentario, nao criei card.
