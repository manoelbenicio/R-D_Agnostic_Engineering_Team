# ORQ-13 + squad — Integration readiness final

> ## RETOMADO — coordenacao ativa (2026-07-28T14:20Z)
>
> Historico de governanca deste cabecalho, para o registro ficar honesto:
> **14:16Z** o owner emitiu **STOP-AND-WAIT** sobre a sequencia ORQ-21/13/12 e eu congelei;
> **14:20Z** o owner esclareceu que **nao ha pausa** e determinou retomada da coordenacao.
> Nada foi executado durante a janela de congelamento, e nada foi desfeito na retomada.
>
> **Regra que permanece igual nos dois estados:** o alvo
> `integration/dev-transition-candidate-20260719` fica **intocado** ate os gates passarem, e o merge
> exige revisao independente (nao o autor) mais autorizacao do owner. Coordeno e preparo; **nao
> mergeio** e **nao edito arquivo de outro owner**.
>
> Estado agora: alvo = **`0cb8aeb`**, intocado; ORQ-13 = `c0e93a2` e squad = `67e9a4c`, **nao
> integrados**; script preservado, **nao executado**, sem bit de execucao; nada meu em execucao —
> nenhum container, tunel, banco ou worktree temporario ativo.
>
> As tres constatacoes abaixo motivam a revisao do plano e **nao** caducaram com a retomada.
>
> ### F1. Overlap ORQ-21 x ORQ-13 — **prospectivo**, ainda nao no git
> A revisao do ORQ-21 aponta sobreposicao em `internal/daemon/daemon.go`,
> `internal/daemon/types.go` e `internal/handler/daemon.go`. Os tres **estao** no meu diff
> (`0cb8aeb..c0e93a2`) — confirmei um a um — logo o risco e **real**.
> Medido agora, porem: `agent/codex-b/orq-21` esta em **`0cb8aeb`, ZERO commits**, e seu worktree tem
> apenas **2 caminhos nao rastreados**: `server/internal/credentialregistry/` (pacote novo) e
> `scripts/staging/seed_approved_assignment.sql`. **Nenhum dos meus tres arquivos foi tocado ainda.**
> Conclusao honesta: a sobreposicao e do **ORQ-21 corrigido que ainda vem**, nao do estado atual — nao e
> conflito hoje, e conflito **anunciado**.
>
> ### F2. Correcao de uma afirmacao minha — "ZERO sobreposicao" era medicao, nao previsao
> A secao 2 afirma "zero sobreposicao de arquivos" com as outras lanes. Era verdade **quando medi**,
> contra o estado **commitado**. **Nao era previsao.** Com ORQ-21 tocando aqueles tres arquivos, a
> afirmacao **caduca no instante em que ORQ-21 commitar**. Registro para que ninguem cite a minha
> varredura como prova de ausencia de conflito com ORQ-21.
>
> ### F3. Regra de invalidacao — "auto-merge anterior nunca e final"
> Toda a evidencia de arvore combinada (Adendo 1) foi medida contra o alvo em **`0cb8aeb`**. **Se o alvo
> mudar — e ele muda quando ORQ-21 entrar — o cherry-pick limpo e os numeros de teste EXPIRAM.**
> Sob autorizacao futura sera necessario: rebase/re-merge do ORQ-13 sobre o alvo **atualizado**, **nova**
> corrida dos **7 testes criticos** e re-execucao dos gates G3 a G7. **Nao trato** o resultado de 13:35Z
> como definitivo, e peco que ninguem o trate.
> **Implementado no script (v2):** gate **G-1 de frescor** aborta automaticamente se o alvo divergir do
> alvo em que a evidencia foi medida, e o gate **G8** re-valida os tres arquivos de overlap depois que
> ORQ-21 entrar.

> **REVISAO 2026-07-28T13:40Z — GTM R2.** Este cabecalho **supersede** os parametros das secoes 5 e 7.

> **REVISAO 2026-07-28T13:40Z — GTM R2.** Este cabecalho **supersede** os parametros das secoes 5 e 7.
>
> - **Veredito: `CODE_PASS` / `EVIDENCE_PASS`.** O BLOCK evidence-only foi atendido pelo gate da arvore
>   combinada real (Adendo 1): package `pass`, `fail 0`, `race 0`, 7/7 testes obrigatorios em `pass`,
>   lint com **zero issues novas**, schema **ate 127**.
> - **Reserva R1:** migration **127**, validade estendida para **2026-07-29T13:21:29Z** (medido em
>   2026-07-28T13:36Z: **23h45m** restantes). Supersede a janela de 15:37:12Z citada na secao 5.
> - **Sequencia canonica:** **ORQ-21 -> ORQ-13 (127) -> ORQ-12 (128)**.
> - **NAO MERGEAR ate ORQ-21 ter PASS.** ORQ-13 deixa de ser o primeiro da fila; passa a ser o segundo.
> - Script de integracao com gates: `.deploy-control/p0/evidence/orq13-integration-script.sh`,
>   sha256 `5f6e235164f196155bdb87b71b98984c8cc00847eb153b1b1b6f0cd3cf5910f4`. **Nao executado**, sem
>   bit de execucao, validado apenas com `bash -n`.

## 0. Estado da sequencia canonica — medido em 2026-07-28T13:36Z

| lane | branch | HEAD | commits sobre a base | situacao |
|---|---|---|---|---|
| **ORQ-21** | `agent/codex-b/orq-21` | **`0cb8aeb`** | **0** | **sem output.** Worktree existe (`…/e39199ed/workdir/repo`), tambem em `0cb8aeb`. Ha check-out anterior de fase de desenho: `CHECKOUT__Agy-P0-A8__gtl-orq21-bridge-redesign__20260727T112804Z.json` + `gtl-orq21-bridge-redesign.md` |
| **ORQ-13** | `agent/opus48-b/orq-13-thinking-level` (+ squad) | `c0e93a2` / `67e9a4c` | 2 + 2 | **pronto**, EVIDENCE_PASS |
| **ORQ-12** | `agent/opus48-a/orq-12-task-usage-account-id` | **`0cb8aeb`** | **0** | **sem output.** Worktree em `…/d4f7f744/workdir/repo`, tambem em `0cb8aeb`. Check-out anterior de auditoria: `CHECKOUT__Codex56-B__ORQ-12-ACCOUNT-SNAPSHOT-AUDIT__20260727T112753Z.json` |

**Consequencia direta:** nao existe output de ORQ-21 para revisar neste momento — a branch nao tem
commit algum. O gate G0 do script para exatamente aqui. Migration **128 nao existe** em nenhuma
branch; a varredura de hoje confirma que **127 e a unica** reivindicada, e e minha.

### 0.1 Colisao anunciada — ORQ-12 vai conflitar com ORQ-13 nos arquivos gerados
Isto e a informacao mais util que tenho para a sequencia, e vale registrar **antes** de o ORQ-12
comecar a escrever codigo.

ORQ-12 e `task_usage.account_id`; ORQ-13 e `task_usage.thinking_level`. **A mesma tabela.** Ao rodar
`sqlc generate`, o ORQ-12 vai reescrever exatamente os arquivos que o ORQ-13 toca:

```
pkg/db/generated/models.go
pkg/db/generated/task_usage.sql.go
pkg/db/queries/task_usage.sql
```

Conflito textual e **garantido** se o ORQ-12 for desenvolvido sobre `0cb8aeb` e integrado depois.
Recomendacao operacional, que **nao** e ordem minha:
1. ORQ-12 **rebaseia** sobre o alvo **ja com ORQ-13 dentro**, e so entao roda `sqlc generate`;
2. resolucao de conflito em `models.go`/`task_usage.sql.go` deve ser feita **regerando com o mesmo
   `sqlc` v1.31.1**, nunca a mao — foi essa a licao do split A/B do ORQ-13;
3. `128` deve ser reservado formalmente antes de materializar o arquivo. Eu **nao** sou o registrador e
   nao emito reserva.

Se a ordem for invertida (ORQ-12 antes de ORQ-13), o mesmo conflito acontece do outro lado e a reserva
127/128 fica fora de ordem no historico de migrations.

---

- **Revisor desta reconciliacao:** Opus48#B (ORQ2, w6:p2) — **READ-ONLY**, sem merge, sem push
- **Data:** 2026-07-28T13:15Z
- **Alvo de integracao medido:** `integration/dev-transition-candidate-20260719`, hoje em **`0cb8aeb`**
- **Veredito: PASS para integracao**, condicionado a **uma** janela de tempo (secao 5)

Aviso de papel: **eu escrevi os quatro commits.** Este documento e reconciliacao de evidencia e
prontidao, **nao** substitui revisao independente nem autorizacao de merge. Nao autorizo merge.

## 1. Estado verificado agora (nao herdado de relatorio anterior)

| item | medido |
|---|---|
| `integration/dev-transition-candidate-20260719` | `0cb8aeb` — **identico a base** dos quatro commits, **zero drift** |
| `agent/opus48-b/orq-13-thinking-level` | `c0e93a2` sobre `b1f08e3`, arvore **limpa** (0 linhas) |
| `agent/opus48-b/squad-default-leader` | `67e9a4c` sobre `11ef715`, arvore **limpa** (0 linhas) |
| base comum das duas branches | `git merge-base` = **`0cb8aebb5aff`**, a mesma |
| sobreposicao de arquivos entre as duas | **NENHUMA** (`comm -12` vazio) |
| `merge-tree` da integracao + ORQ-13 | **LIMPO** |
| `merge-tree` da integracao + squad | **LIMPO** |
| stash de salvaguarda `ORQ13-SAFEGUARD-V31` | **presente** (ref global, visivel dos dois worktrees) |

## 2. Colisao e ownership — varredura de **todas** as branches

- **Migration 127 e exclusivamente minha.** Varri as 20+ branches `agent/*` e `ci/*`: a unica que
  adiciona `127_*` e `agent/opus48-b/orq-13-thinking-level`, com
  `127_task_usage_thinking_level`. **Sem colisao de numero.**
- **Zero sobreposicao de arquivos** com `p1-reasoning-thinking-gateway`, `p0-agent-name-unique`,
  `orq38-get-contract`, `orq43-mdt-enablement`, `orq42-secret-tools-clean`, `ci/orq26-db-gate` e
  `orq41-w4-autopilot`.
- **`agent/codex56-a/p1-reasoning-thinking-gateway` e COMPLEMENTAR, nao duplicada** — e a checagem que
  mais importava, porque o nome sugere duplicidade. Ela mexe em `internal/daemon/brain_integration.go`
  e corrige a **admissao** de `thinking_level` (antes, qualquer valor persistido nao-vazio era
  rejeitado e so `NULL` era despachavel). A minha **persiste** o campo para custo. Arquivos disjuntos,
  minha lane nao toca `brain_integration.go`. **Consequencia util:** o `thinking_level` fim-a-fim
  precisa das **duas**; nenhuma sozinha entrega o resultado de negocio.

## 3. Reconciliacao de evidencia

| commit | conteudo | evidencia |
|---|---|---|
| `b1f08e3` | drift puro do `sqlc` v1.31.1 (2 arquivos gerados, inclui revert `maxSeq`->`max_seq`) | gofmt/build/vet OK; determinismo provado por 3 vias, diff dos 7 rastreados = `b87bc5ca…4707be` |
| `c0e93a2` | `thinking_level` fim-a-fim (6 modificados + 2 migrations + 2 testes) | `./internal/daemon` **4 PASS nominais**, 0 skip, 0 race; handler **4/4 PASS nominais** em DB efemero |
| `11ef715` | producao: `CreateWorkspace` deixa de criar squad sem leader | 2 alvos **PASS nominais**; pacote 3 FAIL -> 1 |
| `67e9a4c` | teste: 3 correcoes em `TestCreateChatSession_Routing` | pacote **VERDE** |

**Corrida canonica final**, `go test -race -count=1 -json ./internal/handler` em `67e9a4c`:
`exit 0` · **package `pass`** · `fail = 0` · **`DATA RACE = 0`** · `pass = 1338` · `skip = 37`.
Log `package-json.log` sha256 `9027b7896a5015c01460628133c356ebf5a184ccf59966eefe365988f86d974d`.
DB: PostgreSQL 17.10 efemero, digest `sha256:d2ef61f4…3bad0`, loopback-only, migration maxima lida de
`schema_migrations` = `126_runtime_profile_protocol_family_native_runtimes`.

**Lint:** `golangci-lint` v2.12.0, checksum oficial conferido: **119 issues no baseline, 119 no HEAD,
zero novas**, relatorios byte a byte identicos. O risco `max_seq` esta **fechado como negativo**.

### 3.1 Duas correcoes minhas que ficam no registro
1. Eu afirmei que `leader_id` explicava os **3** FAIL. Explicava **2**. O terceiro era defeito de teste
   independente (contexto vs header). Corrigido em A3.1–A3.3.
2. Eu propus um fix de **uma linha** para o chat. Medi: **nao bastava** — resolvia o 400 e expunha um
   segundo defeito atras dele. Foram **tres** correcoes. Prova preservada em
   `one-line-insufficient.log`.

## 4. Ordem exata de integracao — **squad antes de ORQ-13**

```
1)  11ef715   fix(workspace): stop creating a default squad with no leader
2)  67e9a4c   test(chat): fix TestCreateChatSession_Routing setup
3)  b1f08e3   chore(sqlc): regenerate baseline output with sqlc v1.31.1
4)  c0e93a2   feat(cost): record thinking_level on task usage end to end
```

**Por que squad primeiro, e nao ORQ-13:** o alvo `0cb8aeb` esta **vermelho** hoje — `internal/handler`
tem 3 FAIL. A branch do ORQ-13 **nao** conserta isso (medi: `c0e93a2` ainda apresenta os 3 FAIL,
pre-existentes). Integrar ORQ-13 primeiro deixaria a branch de integracao vermelha e tornaria
qualquer falha subsequente ambigua. Com 1) e 2) primeiro, o pacote fica **verde antes** de entrar
codigo novo, e a partir dai qualquer regressao tem culpado obvio.

**Restricoes de ordem que sao rigidas:**
- `b1f08e3` **antes** de `c0e93a2`, sempre. `c0e93a2` contem o estado final de `models.go`, que
  pressupoe o drift de A. Aplicar B sem A perde a separacao e reintroduz drift no commit de feature.
- `11ef715` **antes** de `67e9a4c`. O teste corrigido em 4) exercita o comportamento de 3); invertido,
  o teste falha.
- Entre os **pares** (`11ef715`+`67e9a4c`) e (`b1f08e3`+`c0e93a2`) nao ha dependencia tecnica —
  arquivos disjuntos, `merge-tree` limpo nos dois sentidos. A ordem recomendada e por **higiene de
  bissecao**, nao por conflito.

## 5. ~~A UNICA condicao do PASS — janela da reserva~~ — **SUPERSEDED pelo cabecalho GTM R2**

> **SUPERSEDED.** A janela citada aqui (`2026-07-28T15:37:12Z`) foi **estendida**: a reserva **R1** da
> migration 127 agora vale ate **2026-07-29T13:21:29Z**. E a condicao dura **deixou de ser apenas
> temporal**: passou a haver uma **dependencia de sequencia** — `ORQ-21` deve ter **PASS** antes de
> ORQ-13 entrar. Ver secao 0 e o gate G0 do script.
>
> Continua valido: **eu nao posso renovar nem emitir reserva**; e se a janela for perdida, renovar o
> numero 127 e menos arriscado que renumerar, porque renumerar invalida os hashes registrados.

`RES-ORQ13-001` (migration **127**) expira em ~~**2026-07-28T15:37:12Z**~~ -> **2026-07-29T13:21:29Z**.

- Integrar dentro da janela: **PASS, sem acao adicional.**
- Passar da janela: **BLOCK administrativo**, nao tecnico. O codigo continua valido; o que caduca e a
  reserva do numero. Seria preciso **renovar** a reserva ou **renumerar** a migration — e renumerar
  invalida os hashes que registrei (`up 0ea3005d…`, `down 74354ae2…`) e exige re-verificacao.
- **Eu nao posso renovar nem emitir reserva** — nao sou o registrador.

Se a janela for perdida, o caminho de menor risco e **renovar o numero 127**, nao renumerar: a
varredura da secao 2 prova que ninguem mais reivindicou 127.

## 6. Divida separada — os 37 skips externos (**nao** bloqueiam)

`skip = 37`, **todos** top-level, zero subtestes, **nenhum** tocando os quatro commits. Taxonomia:

| grupo | qtd | guard | zeravel? |
|---|---|---|---|
| `TestRedis*`, `TestRequireDaemonWorkspaceAccess_*`, `TestMembershipCache_*` | **35** | `newRedisTestClient`: `t.Skip` se `REDIS_TEST_URL` vazio; `t.Skipf` se `Ping` falhar | sim, com Redis efemero — **imagem inexistente no ORQ1**, exigiria pull |
| `TestFetchFromSkillsSh_AnthropicPptxIntegration` | 1 | `MULTICA_RUN_SKILLS_SH_INTEGRATION` vazio | so com rede viva ao GitHub |
| `TestWebhookHandler_DBErrorOnTokenLookupReturns500` | 1 | **`t.Skip` incondicional** — marcador de regressao deliberado (`autopilot_webhook_handler_test.go:807-819`) | **NAO, nunca** |

Portanto **`skip = 0` e estruturalmente impossivel** neste pacote, e concordo em nao exigi-lo.
Criterio de aceite que sustento: `package pass` + `fail = 0` + `race = 0` + **zero skip entre os testes
afetados** — satisfeito.

**Divida registrada, para cartoes proprios, nao meus para criar:**
- **D1** — cobertura Redis ausente por falta de `REDIS_TEST_URL` no ambiente de teste (35 testes).
- **D2** — `t.Skip` incondicional do webhook impede qualquer metrica de `skip` zerada; decidir entre
  reescrever com injecao ou remover o teste-marcador.
- **D3** — `TestMain` de `internal/handler` faz `os.Exit(0)` sem banco, produzindo **falso-verde**.
  Transversal; ja me custou uma retratacao.
- **D4** — **119 issues de lint pre-existentes**: o job de lint falhara ao ser ligado, por divida
  anterior, nao por estes commits.
- **D5** — os 4 workflows vivem em `multica-auth-work/.github/workflows/`, **fora da raiz**, logo
  **inertes**; "diferir para o CI" so vale depois de ligar o CI.

## 7. Veredito

**PASS para integracao**, na ordem da secao 4, condicionado a:
1. janela de `RES-ORQ13-001` (secao 5) — a unica condicao dura;
2. revisao independente que **nao** seja eu, ja pedida e ainda em aberto;
3. autorizacao de merge do owner, que **eu nao dou**.

**Nao ha BLOCK tecnico.** Os 37 skips sao divida externa documentada, nao impedimento.

Lacuna que declaro pela quarta vez, porque continua aberta: verifiquei os consumidores em **Go**;
**nao** varri o frontend Next.js para confirmar que nada depende do squad default existir imediatamente
apos criar o workspace. `11ef715` **muda comportamento de produto**. Se algo no frontend assumir o
squad, o sintoma aparece la, nao nos testes de Go.

## 8. Nota pronta para o Kanban — **v2, GTM R2 (2026-07-28T13:40Z)**

A v1 desta secao esta **substituida**; mudou o veredito de evidencia, a validade da reserva e a
posicao de ORQ-13 na fila.

> **Titulo:** ORQ-13 fase 1 + fix squad default leader — EVIDENCE_PASS, aguardando ORQ-21 (4 commits)
>
> **Estado:** **`CODE_PASS` / `EVIDENCE_PASS`**. Sem BLOCK tecnico. **Bloqueado por sequencia**, nao por
> qualidade: nao merge­ar antes de `ORQ-21` ter PASS.
>
> **Sequencia canonica:** `ORQ-21` -> **`ORQ-13` (migration 127)** -> `ORQ-12` (migration 128).
> Situacao medida em 13:36Z: **ORQ-21 e ORQ-12 estao ambas em `0cb8aeb`, com zero commits.** Nao ha
> output de ORQ-21 para revisar ainda.
>
> **Ordem interna dos 4 commits (rigida):** `11ef715` -> `67e9a4c` -> `b1f08e3` -> `c0e93a2`, sobre
> `integration/dev-transition-candidate-20260719` (@`0cb8aeb`, zero drift).
>
> **Evidencia da arvore combinada REAL** (cherry-pick dos 4 em worktree temporario, sem merge no alvo):
> `go test -race -count=1 -json ./internal/handler` -> package **pass**, `fail=0`, `race=0`,
> `pass=1347`, `skip=37` externos; `"Skipping tests:"=0`; log sha256 `abf8c39f…4df8d`. DB efemero
> PostgreSQL 17.10 com digest pinado, schema **ate 127**. Lint v2.12.0 baseline vs combinada:
> **119 = 119, zero novas**, relatorios byte a byte identicos.
>
> **Reserva:** **R1 da migration 127 vale ate 2026-07-29T13:21:29Z.** Fora da janela e BLOCK
> administrativo, nao tecnico: renovar o 127 (recomendado) ou renumerar, o que invalida os hashes
> registrados.
>
> **Script de integracao com 8 gates:** `.deploy-control/p0/evidence/orq13-integration-script.sh`
> (sha256 `5f6e2351…10f4`). **Nao executado**, sem bit de execucao, validado com `bash -n`. G0 exige
> veredito PASS de ORQ-21 **e** que ORQ-21 ja esteja contido no alvo; o script **para** no ponto do
> merge, que continua exigindo revisao independente e autorizacao do owner.
>
> **Alerta para o ORQ-12, antes de ele escrever codigo:** `account_id` e `thinking_level` sao a **mesma
> tabela** `task_usage`. O `sqlc generate` do ORQ-12 vai reescrever `models.go`,
> `task_usage.sql.go` e `queries/task_usage.sql` — os mesmos arquivos do ORQ-13. Conflito **garantido**
> se desenvolvido sobre `0cb8aeb`. ORQ-12 deve **rebasear sobre o alvo ja com ORQ-13** e resolver
> **regerando com sqlc v1.31.1**, nunca a mao. E reservar o **128** formalmente antes de materializar o
> arquivo.
>
> **Dependencia de negocio:** `thinking_level` fim-a-fim tambem precisa de
> `agent/codex56-a/p1-reasoning-thinking-gateway`, que corrige a **admissao**. Lanes complementares,
> arquivos disjuntos.
>
> **Divida a abrir em separado:** D1 Redis sem `REDIS_TEST_URL` (35 skips); D2 `t.Skip` incondicional do
> webhook; D3 falso-verde do `TestMain`; D4 119 issues de lint pre-existentes; D5 workflows fora da raiz
> e inertes; **D6 44 arquivos nao formatados no modulo** (nenhum dos 4 commits).
>
> **Pendente:** ORQ-21 PASS; revisor independente (nao o autor); autorizacao de merge do owner. Lacuna
> conhecida: frontend Next.js nao varrido quanto a dependencia do squad default imediato.

### 8.1 ~~Nota v1~~ — **SUPERSEDED pela v2 acima. NAO COPIAR.**

> Mantida so para rastreio. Ela diz "pronto para integracao" sem a dependencia de ORQ-21, e cita a
> janela de reserva antiga (15:37:12Z de 28/07) em vez da R1 (13:21:29Z de 29/07). **Usar a v2.**

> **Titulo:** ORQ-13 fase 1 + fix squad default leader — pronto para integracao (4 commits)
>
> **Estado:** CODE PASS, EVIDENCE PASS. Sem BLOCK tecnico. Read-only review concluida.
>
> **Ordem de integracao:** `11ef715` -> `67e9a4c` -> `b1f08e3` -> `c0e93a2`, sobre
> `integration/dev-transition-candidate-20260719` (@`0cb8aeb`, zero drift). `merge-tree` limpo, sem
> sobreposicao de arquivos entre as branches nem com outras lanes. Migration **127** exclusiva.
>
> **O que entra:** (a) `CreateWorkspace` deixa de retornar 500 sempre — o endpoint estava
> **inoperante**, criando squad sem leader contra o `NOT NULL` da migration 084; (b)
> `TestCreateChatSession_Routing` corrigido (3 defeitos de teste empilhados); (c) split do drift de
> `sqlc` separado da feature; (d) `thinking_level` persistido no uso, para Opus thinking parar de ser
> cobrado como Opus base.
>
> **Evidencia:** `go test -race -count=1 -json ./internal/handler` -> package **pass**, `fail=0`,
> `race=0`, `pass=1338`, `skip=37` externos e documentados; DB PostgreSQL 17.10 efemero com digest
> pinado; log sha256 `9027b789…d974d`. Lint v2.12.0: **zero issues novas** (119 = 119).
>
> **Prazo duro:** `RES-ORQ13-001` (migration 127) expira **2026-07-28T15:37:12Z**. Depois disso e
> BLOCK administrativo: renovar o numero 127 (recomendado) ou renumerar, o que invalida os hashes
> registrados.
>
> **Dependencia de negocio:** `thinking_level` fim-a-fim tambem precisa de
> `agent/codex56-a/p1-reasoning-thinking-gateway`, que corrige a **admissao**. Lanes complementares,
> arquivos disjuntos.
>
> **Divida a abrir em separado:** D1 Redis sem `REDIS_TEST_URL`; D2 `t.Skip` incondicional do webhook;
> D3 falso-verde do `TestMain`; D4 119 issues de lint pre-existentes; D5 workflows fora da raiz,
> inertes.
>
> **Pendente:** revisor independente (nao o autor) e autorizacao de merge do owner. Lacuna conhecida:
> frontend Next.js nao varrido quanto a dependencia do squad default imediato.

---

# ADENDO 1 — gate da ARVORE COMBINADA REAL (2026-07-28T13:3xZ)

Pedido pela revisao final, que manteve **BLOCK evidence-only**: nao bastava evidencia por branch, era
preciso medir a **arvore combinada de fato**. Feito. **Sem merge no alvo, sem push.**

## A1.1 Construcao da arvore combinada

Worktree **temporario** e privado (`0700`), `--detach` em `0cb8aeb`, com `cherry-pick -x` na ordem
recomendada. Nenhum conflito em nenhum dos quatro:

| passo | commit original | commit na arvore combinada |
|---|---|---|
| 1 | `11ef715` fix(workspace) | `7d7b692` |
| 2 | `67e9a4c` test(chat) | `c5f298b` |
| 3 | `b1f08e3` chore(sqlc) | `cfcfb0b` |
| 4 | `c0e93a2` feat(cost) | `6757249` |

**HEAD combinado: `6757249006983c0f1c929f558082b3c9a069f308`**, arvore limpa, `13 files changed,
400 insertions(+), 50 deletions(-)` contra `0cb8aeb`.

Migrations conferidas **na arvore combinada**, batendo com RES-ORQ13-001:
`up` `0ea3005da0ee257618062552cf8792f9c2e6ed478dce3ba174bf08692486cac1`,
`down` `74354ae28dee526c7dbc6bc6733471a59c2f3dabfe5a7fe609fe20d747e61113`.

## A1.2 Banco efemero — schema **ate 127**

| campo | valor |
|---|---|
| container | `orq13-combined-pg`, id `30db43410b9c303b91261bf259af8cdcfe186da424c2ea5db14dacc39d3f4ef2` |
| imagem | `pgvector/pgvector@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` (local, sem pull) |
| binding | `{"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"47477"}]}` — **loopback-only** |
| tunel | SSH privado `127.0.0.1:38807` -> ORQ1 `127.0.0.1:47477` |
| `DATABASE_URL` | arquivo modo **600**, senha aleatoria gerada no ORQ1, nunca em argv |
| migrations | `up 127_task_usage_thinking_level` -> `Done.` — **schema ate 127**, so no banco descartavel |

## A1.3 Resultado — `go test -race -count=1 -json ./internal/handler`

```
exit = 0
package verdict: github.com/multica-ai/multica/server/internal/handler = "pass"
Action counts: cont 16 · output 2938 · pass 1347 · pause 16 · run 1383 · skip 37 · start 1
fail = 0 · skip = 37 (todos top-level, 0 subtestes) · DATA RACE = 0
"Skipping tests:" = 0  (o falso-verde do TestMain NAO ocorreu: o pacote rodou de fato)
```
Log `combined-json.log` sha256 `abf8c39f90547d4aa6335b191d042d1de37295ed2e00f09539cd75ae28a4df8d`.

**Os 7 testes que importam, verificados um a um no JSON da arvore combinada:**

| teste | Action |
|---|---|
| `TestCreateChatSession_Routing` | **pass** |
| `TestCreateWorkspaceUsesRequestedSlug` | **pass** |
| `TestCreateWorkspace_DoesNotMarkOnboarded` | **pass** |
| `TestThinkingLevelText` | **pass** |
| `TestThinkingLevelTextNeverStoresEmptyString` | **pass** |
| `TestTaskUsagePayloadDecodesThinkingLevel` | **pass** |
| `TestTaskUsagePayloadLegacyDaemonYieldsNull` | **pass** |

Comparacao com a corrida por-branch (`skip 37`, `pass 1338`): a combinada tem **`pass 1347`**, nove a
mais, coerente com os testes que antes nem chegavam a rodar. `skip` permanece **37**, os mesmos
externos, e `fail`/`race` seguem em **0**.

## A1.4 Qualidade estatica na arvore combinada

| verificacao | resultado |
|---|---|
| `gofmt -l` nos **10** arquivos Go alterados pelos 4 commits | **vazio** = limpo |
| `go build ./...` | **OK** |
| `go vet ./internal/handler ./internal/daemon ./pkg/db/generated` | **OK** |

Observacao honesta que encontrei ao medir: `gofmt -l` no **modulo inteiro** acusa **44** arquivos nao
formatados. **Nenhum e meu** — os 10 alterados estao limpos. E divida pre-existente e vale um cartao,
somando-se a D4.

## A1.5 Lint v2.12.0 — baseline vs **arvore combinada**

Binario rebaixado da release oficial e verificado: `sha256sum -c` **OK** contra
`6b89d77b6396f81decee882f20473486bda5ef28b160f9a13b08f24848d9003c`, o **mesmo** hash da rodada
anterior; binario reporta `2.12.0 built with go1.26.2 from 7761527a`. Instalado em diretorio
**privado** `0700`, nunca global, **removido no teardown**. Caches de lint **isolados por execucao**,
para nao repetir a contaminacao que eu mesmo detectei antes.

| | issues | errcheck | ineffassign | staticcheck | unused |
|---|---|---|---|---|---|
| baseline `0cb8aeb` | **119** | 50 | 1 | 39 | 29 |
| **arvore combinada** `6757249` | **119** | 50 | 1 | 39 | 29 |

**NOVAS: 0. RESOLVIDAS: 0.** Os dois relatorios sao **byte a byte identicos**, sha256
`e5e3b46ba3d23a3400ba3a7c155d803598ad5dbe93df7a8d1a0751794fd122d8` nos dois.
`exit 1` nos dois lados, pelas 119 issues **pre-existentes** (D4), nao pelos commits.

## A1.6 Teardown verificado
- worktrees temporarios `wt` e `wt-base` **removidos** via `git worktree remove`; restam **30** entradas
- container `orq13-combined-pg` removido; `docker ps -a | grep -c orq13-combined` = **0**
- tunel encerrado; **nenhum** `ssh -L 127.0.0.1` residual
- senha `/home/ec2-user/.orq13-combined-pw` removida
- cache `orq13-combined` removido **inteiro**, incluindo o binario `golangci-lint` e os 2 caches de lint
- `docker prune` **nao** executado
- **Produto intacto:** backend `Up 21 hours`, frontend `Up 34 hours`, postgres `Up 6 days (healthy)`,
  omniroute `Up 3 days (healthy)`; ORQ1 com 4.3G livres

## A1.7 Governanca preservada
Nada foi contornado. Verificado **depois** do gate:
`agent/opus48-b/orq-13-thinking-level` = `c0e93a2`, `agent/opus48-b/squad-default-leader` = `67e9a4c`,
`integration/dev-transition-candidate-20260719` = **`0cb8aeb`** — o alvo **nao** foi tocado. Nao houve
merge, push, rebase ou alteracao de branch de terceiro; toda escrita ocorreu dentro dos **meus** dois
worktrees temporarios, ambos removidos. As issues preservadas, incluindo **ORQ-21** e **ORQ-12**, nao
foram lidas para escrita, reexecutadas nem alteradas.

## A1.8 Efeito sobre o veredito
A secao 7 dizia PASS com evidencia **por branch**. Agora ha evidencia da **arvore combinada real**:
package `pass`, `fail 0`, `race 0`, 7/7 testes relevantes em `pass`, lint com zero issues novas,
schema ate 127. **O BLOCK evidence-only esta atendido.**

**Condicoes atualizadas em 2026-07-28T14:15Z** (substituem o texto anterior desta secao, que citava a
janela vencida de 15:37:12Z e tratava a evidencia como definitiva):
1. **Reserva R1** da migration 127 vale ate **2026-07-29T13:21:29Z**.
2. **Sequencia:** ORQ-21 corrigido entra **primeiro**; ORQ-13 depois.
3. **Esta evidencia e PROVISORIA.** Ela foi medida contra o alvo em `0cb8aeb`. Quando ORQ-21 entrar, o
   alvo muda e **este gate expira** — ver o bloco de congelamento no topo do documento. Nao tratar o
   cherry-pick limpo de 13:35Z como veredito final.

Continuam pendentes, e **nao** sao meus: revisao independente (nao o autor) e autorizacao de merge do
owner. **Nao autorizo merge.**

## 9. Nao-afirmacoes
**Atualizadas pelo Adendo 1.** A secao 7 e as duas primeiras nao-afirmacoes descreviam uma revisao
**read-only sem execucao**; o Adendo 1 **executou** o gate da arvore combinada sob autorizacao
posterior. As duas estao marcadas.

- ~~**READ-ONLY**: nenhum merge, push, rebase, amend, commit, tag ou mutacao de branch nesta revisao.~~
  **ATUALIZADO:** continua verdadeiro que **nao houve merge, push, rebase, amend nem alteracao de
  branch existente**. Mas o Adendo 1 **criou** dois worktrees temporarios e **quatro commits de
  cherry-pick** dentro deles (`7d7b692`, `c5f298b`, `cfcfb0b`, `6757249`), todos **descartados** com os
  worktrees. O alvo permaneceu em `0cb8aeb`, verificado depois.
- ~~Nenhum banco tocado nesta revisao.~~ **ATUALIZADO:** o Adendo 1 subiu container efemero, abriu
  tunel privado, aplicou migrations **ate 127 em banco descartavel** e rodou os testes. Nenhum banco de
  **produto** foi tocado; `multica-dev-transition-postgres-1` segue `Up 6 days (healthy)`.
- **Nao autorizo merge** e **nao me autoaprovo**: escrevi os quatro commits, e nem esta reconciliacao
  nem o gate da arvore combinada substituem revisao independente.
- Nao criei, atribui nem comentei cartao algum: a secao 8 e **texto pronto**, nao um cartao criado.
- **Nao posso renovar nem emitir** a reserva de migration; nao sou o registrador.
- **Nao varri o frontend Next.js** (secao 7). Segue sendo a lacuna consciente.
- Nao verifiquei se as outras lanes passam nos proprios testes: avaliei apenas **colisao** com as
  minhas, por arquivo e por numero de migration.
- Nao removi cache de terceiros nem rodei `docker prune`; removi so o que criei, incluindo o binario de
  lint.
- **Nao contornei governanca**: nao toquei ORQ-21, ORQ-12 nem qualquer branch de terceiro (A1.7).
- Nao li nem usei segredo, token ou credencial.
- **Nao afirmo `skip = 0`**: foram 37, os mesmos externos ja documentados, e o criterio e
  estruturalmente impossivel (secao 6).
