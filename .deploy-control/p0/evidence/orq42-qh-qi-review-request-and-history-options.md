# ORQ-42 Q-H/Q-I - pedido de peer review independente e opcoes para o historico

- autor/executor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T21:05Z
- ORQ-26 em **HOLD_EXTERNAL_BILLING** (branch e SHA remotos confirmados, run 30294480915 sem nenhum
  step). **Nao** tentei contornar billing, **nao** rodei rerun, **nao** abri PR nem merge.
- worktree: `/home/ec2-user/workspace/worktrees/orq42-secret-tools`, limpo

## 1. O que existe agora - duas linhagens locais, nenhuma reescrita

```text
agent/opus48-a/orq42-secret-tools        355c57b  feat: editor + probe   (contem os binarios)
                                          bbeb80b  fix: remove binarios + .gitignore
agent/opus48-a/orq42-secret-tools-clean  cebc96a  feat: mesmas fontes, 1 commit, historico limpo
```
O branch `-clean` foi **criado a partir de `0cb8aeb`** e recebeu **apenas as fontes revisadas**.
Isso e **aditivo**: os dois commits originais permanecem **intactos**, nada foi reescrito, e nenhuma
autorizacao de rewrite foi usada. Ambos os branches produzem a **mesma arvore** em `tools/orq42`.

## 2. O problema do historico, medido - e o contexto que muda a recomendacao

Objetos exclusivos do branch original: **9 blobs, 9,01 MB**, dos quais os dois binarios que eu commitei
por acidente sao **3,0 MB** e **6,0 MB**.

Contexto que eu **nao** tinha registrado antes e que e relevante: este repositorio **ja** versiona
binarios muito maiores no proprio `0cb8aeb`:
```text
47.9 MB  multica-auth-work/server/multica-server
34.1 MB  bin/prodex
14.8 MB  multica-auth-work/server/multica-migrate
 4.8 MB  .deploy-control/p0/monitor.jsonl
```
Ou seja: os meus 9 MB sao um defeito real **meu**, mas **nao** sao excepcionais neste repo, e uma
reescrita de historico so por causa deles seria **desproporcional** ao ganho.

## 3. Tres opcoes para o owner decidir - eu **nao** executo nenhuma sem autorizacao

| # | opcao | efeito | risco | precisa de autorizacao? |
|---|---|---|---|---|
| **H1** | **empurrar o branch `-clean`** e descartar o original localmente | historico sem os blobs, sem reescrever nada | nenhum: e um branch novo | **nao** para criar (ja criei); **sim** para push |
| **H2** | manter o branch original como esta | 9 MB de blobs no historico do branch | proporcional ao que o repo ja carrega | nenhuma |
| **H3** | reescrever `355c57b`/`bbeb80b` (rebase interativo ou `filter-repo`) | historico limpo no **mesmo** branch | reescrita de commits; foi exatamente o tipo de operacao que **causou** um incidente meu nesta sessao (amend que reescreveu commit de outra lane) | **SIM, explicita** |

**Minha recomendacao: H1.** Ele entrega o resultado de H3 sem nenhum risco de reescrita. Se o owner
preferir H3, eu paro e espero autorizacao escrita - **nao** vou rebasear por conta propria.

## 4. Pedido de peer review independente

**Revisor deve ser agente que nao seja eu** (autor de V5-V9 e implementador). Alvo exato:

```text
branch          agent/opus48-a/orq42-secret-tools-clean   commit cebc96a
tools/orq42/envsecret-editor/main.go       sha256 824f7fc6746c8703097877df1da14ddbffe07994ca048bddcdb72a041265b4ce
tools/orq42/envsecret-editor/main_test.go  sha256 a20a21a8c6f813791faf5d1708d88e108d6f332b0fd8c5edcd179111841fdeae
tools/orq42/envsecret-editor/go.mod        sha256 7e30eca6984ade31fb1b0891cf29afabbb9ae4abd1766e541a7404a073dd55c5
tools/orq42/wsprobe/main.go                sha256 4508d9fddb7a16ee1fa749200066f677b214f9b0d96132ca0c39f618f2800e10
tools/orq42/wsprobe/main_test.go           sha256 8db06bef8fb3f69e444b2890d1fe434f8edc8a5a1ed204e72321d6fc2dfbb87e
tools/orq42/wsprobe/go.mod                 sha256 cc7a3ca1fe57284c86a7e99058a49a89de3320c3ce839c04bcabdf98f29b5d08
tools/orq42/.gitignore                     sha256 1f71df507f15d3718b947173c782f1af4aa0f19de804f4794117d451c243d783
binario editor (fora do repo)              sha256 0a191f4387343ecf358e8c7a703fbc331fdd37716bba5a3e40f14f79dafd5f73
binario probe  (fora do repo)              sha256 305550b1a1cfb5a8ee87347b7eb09850fdb31b7fc23f4ce7f26c5cd1a5618453
```
Reproduzir localmente:
```bash
cd /home/ec2-user/workspace/worktrees/orq42-secret-tools   # ou um worktree proprio do revisor
git switch agent/opus48-a/orq42-secret-tools-clean
export PATH=/home/ec2-user/goroot/go/bin:$PATH GOWORK=off GOFLAGS=-mod=readonly
export GOCACHE=<cache privado 0700> GOTMPDIR=<privado 0700> TMPDIR=$GOTMPDIR
for t in envsecret-editor wsprobe; do (cd tools/orq42/$t && gofmt -l . && go vet ./... && go test -race -count=1 ./...); done
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o <privado>/<tool> .   # comparar hash
```

### Perguntas que eu quero que o revisor ataque

1. **`assignmentSpan`**: alguma grafia dotenv **legitima** e tratada como nao-atribuicao e portanto faz
   o editor recusar um arquivo valido? Casos que eu **deliberadamente** trato como nao-atribuicao:
   linha indentada, linha comentada, e `export JWT_SECRET=…`. Isso e **correto** para o `dev.env` do
   ORQ-30 (que e `KEY=value` puro), mas seria errado num dotenv com `export`.
2. **`wsprobe` sem `wss`**: ele **nao** faz TLS. No alvo (`127.0.0.1:18080`, loopback) isso e
   adequado; se algum gate futuro apontar para FQDN com HTTPS, ele **falha** com `WS_STOP_*` em vez de
   dar veredito. Isso e aceitavel ou o probe deve exigir `wss` antes de ser autorizado?
3. **`first-frame` com teto de 4 frames**: o servidor pode legitimamente enviar frames antes do
   `auth_ack`? Se puder, o teto e baixo.
4. **fail-closed vs veredito**: os codigos de saida (`0` aceito, `3` rejeicao definitiva, `1` falha de
   ferramenta, `2` uso) distinguem bem "rejeitou" de "nao consegui medir"?
5. **historico**: H1, H2 ou H3?

## 5. Nao-afirmacoes

- **Nenhum push, PR, merge, quadro, AWS, segredo, Docker, SSH ou mutacao de runtime.** Tudo local.
- **Nao reescrevi nem apaguei commit algum.** O branch `-clean` e **adicional**; `355c57b` e
  `bbeb80b` continuam existindo, com os blobs.
- **Nao tentei contornar o billing** e **nao** executei `gh run rerun`.
- Os utilitarios **nunca** foram executados contra alvo real: o `wsprobe` so viu um hub falso
  `httptest` local, e o editor so tocou fixtures de `t.TempDir()`. **Nao** afirmo que o `/ws` real se
  comporta como o falso.
- Nao executei nenhum comando de metadado da Q-C, e nao verifiquei se o perfil `owner-p0` existe.
- Nao alterei assignee, nao postei comentario, nao criei card - a evidencia vai para
  `.deploy-control/p0/`, como sempre.
