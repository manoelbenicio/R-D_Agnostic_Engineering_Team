# GTL-83 — Peer review externo do Z01 Master Integration Control — VEREDITO: **BLOCK** (amendável)

Auditor: Codex56#A (`w7:p3`, ORQ2) · UTC 2026-07-27T12:26Z · READ-ONLY
Auditado: `.deploy-control/p0/evidence/gtl-z01-master-integration-control.md`
`sha256 = 14b988c1306aaa8cc51484f41aeb6c65335ef91921c850da4491263cf15f042b` (46018 bytes, 562 linhas, mtime 12:19)
Destinatários: Codex56-TL (`w5:pC`) e Codex56#Z (`w8:p4`).
Não editei o master, o repo, worktree, board, banco ou config. Escrita única: este relatório + meu check-out.

## 0. Veredito

**BLOCK** — não do conteúdo em bloco, mas do documento **como controle autoritativo**: ele foi escrito
às 12:19 com snapshot 12:16:22Z e **cinco rulings mais novos já existiam ou surgiram**, três deles
contradizendo diretamente seções operacionais (§0, §3.1, §4, §5, §7.3). Um master control stale é pior
que ausência de master, porque congela BLOCKs já superados e libera número de migration errado.
O veredito global do master ("BLOCK para cutover único agora") **eu confirmo**. O que reprova é a
precedência e o mapa de migrations.

Sete amendments exatos abaixo; com eles aplicados, o documento vira PASS como controle.

## 1. Verificações que eu reproduzi (o master acerta)

| Alegação do master | Meu comando | Resultado |
|---|---|---|
| base termina em `126_runtime_profile_protocol_family_native_runtimes` | `ls server/migrations \| grep ^12` | CONFIRMADO; `grep -c ^127` = **0** na base |
| único `127*` materializado é da ORQ-13 | varredura dos 18 worktrees por `^12[789]` | CONFIRMADO: só `/workspace/worktrees/gtl-i03-orq13-phase1` → `127_task_usage_thinking_level.{up,down}.sql` |
| 18 worktrees (root + 7 recentes + 10 P0) | `git worktree list \| wc -l` | CONFIRMADO = **18**; 1+7+10=18 fecha |
| known-bad: green `88ca4f39...`, `.new`/`pre-token-only` `5f9ec49e...`, `.pre-agy-fix` `f3463086...`, `.previous` `e0510d7d...` | `gtl-orq23-durable-cutover-plan.md:16-21` | CONFIRMADO linha a linha, incluindo o par `gate1` com o mesmo hash green |
| nove issues preservadas ORQ-12/13/15/16/17/18/21/22/23, zero rerun | contagem + coerência com o comunicado do TL | CONFIRMADO (9 itens; nenhuma proposta de rerun no documento) |
| segurança `/tmp`: marcador sensível em `backend-recover.sh`, sem leitura de valor | `gtl-tmp-umask-remediation-plan.md:13` e `:127` | CONFIRMADO e **citação correta**: meu doc classifica `JWT_SECRET=` com valor por contagem, e o master preserva o "não foi lido valor" |
| Skills BLOCK/PENDING PROOF por falta de manifest+SHA LOCAL, sem teto de 48 | `gtl-skills-rollout-second-peer-review.md` citado | CONFIRMADO como leitura; mantém-se PENDING PROOF |
| decisões owner × técnicas (§8) | leitura | CORRETO: owner decide billing/retenção/exposição/replay/custodiante; engenharia decide número de migration, SQLC, locks, trigger. Coerente com o critério #1 do GTL-68, que pede ruling **da liderança técnica**, não do owner |

## 2. Amendments obrigatórios (exatos)

### A1 — ORQ-26 CI: BLOCK do master está STALE; existe ruling oficial PASS

Master, §0 e §5 e §7.3: "proposta atual … permanece **BLOCK** aguardando security review GTL-69";
"A alternativa V2 `push`-only existe, mas contradiz o steering atual e **não está autorizada**".

Fatos mais novos:

```text
gtl-orq26-root-ci-manifest-v2-peer-review.md:1   "# Parecer de Peer Review Adversarial: Manifesto CI V2 para a ORQ-26 (READ-ONLY GTL-69R)"
gtl-orq26-root-ci-manifest-v2-peer-review.md:5   "Ruling Oficial: Technical Ruling — `workflow_dispatch` impossível fora de default branch (`main`);
                                                  `push` restrito à branch efêmera `ci/orq26-db-gate` é o mecanismo correto."
gtl-orq26-root-ci-manifest-v2-peer-review.md:7-8 "Data UTC: 2026-07-27T12:14:53Z" · "Veredito: **PASS**
                                                  (Manifesto CI V2 Totalmente Aprovado para Autorizações B1-B4)"
gtl-orq26-ci-commit-manifest.md:364              "# V2 — Correção do manifesto após GTL-73 (2026-07-27T12:11Z)"
```

O master (12:19) é **posterior** ao GTL-69R (12:14:53Z) e ao GTL-73 (12:11Z) e ainda assim mantém o
BLOCK e chama a V2 de não autorizada. **Amendment:** trocar, em §0, §4 (linha ORQ-26 server), §5
(linha GTL-65) e §7.3, para: *"`workflow_dispatch` fora da default branch é tecnicamente impossível
(ruling oficial). O mecanismo aprovado é `push` em branch efêmera `ci/orq26-db-gate`. Manifesto V2 =
**PASS** (GTL-69R, 12:14:53Z) após correção GTL-73. Pendência remanescente não é security review, e sim
as **autorizações B1–B4 do owner** (push de branch efêmera é STOP-AND-WAIT §8.2)."* GTL-69 deixa de ser
pendência: foi emitido como 69R.

### A2 — Kanban: V7 não é "sem peer review", e V8 já existe

Master §0 e §4 e §5: "O artefato mais novo é V7/GTL-75, **DESIGN-ONLY ainda sem peer review**";
"131 — reservado para Kanban integrity somente se **V7** passar peer review".

Fatos mais novos:

```text
gtl-kanban-v7-peer-review.md:1,3,8      "# GTL-77 — Peer Review Adversarial do Kanban Integrity Monitor V7"
                                        · auditor Codex56#B (w7:p4) · "Veredito: **BLOCK**"
gtl-kanban-v7-peer-review.md:21-23      upsert monotônico = BLOCK (sem SQL); clear = "PASS de intenção /
                                        BLOCK de atomicidade"; transação/lock = BLOCK (sem BeginTx)
gtl-kanban-integrity-monitor-v8.md:1,4,6 "# ... V8 — Reconciliação Transacional Durável (READ-ONLY GTL-79)"
                                        · 2026-07-27T12:18Z · "incorporando integralmente GTL-67 e GTL-77
                                        (... - PASS V7 Revogado)"
```

**Amendment:** linha Kanban de §4/§5 passa a *"V4/V5/V6 revogados; **V7 BLOCK (GTL-77)**; **V8 (GTL-79,
12:18Z) é o artefato mais novo e está pendente de peer review GTL-80**"*, e a reserva de número Kanban
deixa de estar condicionada ao V7 (ver §3).

### A3 — Combinar ORQ-21 + ORQ-12 + ORQ-13 em um único 128 está **tecnicamente errado**: separar

Master §3.1 item 2 e §6.4: "**128 — `task_account_usage_dimensions`**: única evolução expand-first
reunindo ORQ-21/12/13"; "**R128** é um único pacote técnico, não três branches".

Três razões factuais, na ordem de peso:

1. **O ruling que o master emite contradiz o parecer que o master deveria consumir.** GTL-68
   (`gtl-orq13-phase1-peer-review.md:8`) é *"PASS DE CÓDIGO E TESTES UNITÁRIOS **COM SPLIT PLAN**,
   CONDICIONAL A 3 CRITÉRIOS"*, e `:55-61` define o Commit 2 **exclusivo** da ORQ-13
   (`127_task_usage_thinking_level.{up,down}.sql`, `queries/task_usage.sql`, `generated/task_usage.sql.go`,
   `daemon.go`, `daemon/types.go`, `handler/daemon.go`, testes). O critério `:69` é literalmente
   *"Ruling Z01: definição e renumeração formal da migration (127 vs 128) pela liderança técnica"*.
   Fundir em 128 anula o split aprovado.
2. **Atomicidade não é requerida.** Pelo próprio §3.2 do master, `credential_account_id` e
   `thinking_level` são **snapshots nullable aditivos** e `UNIQUE(task_id,provider,model)`
   **permanece inalterado** (`032_task_usage.up.sql:1-14`). Não há constraint compartilhada, backfill
   comum nem invariante cruzada que exija uma única transação DDL. O acoplamento real é de **lane**
   (mesmo `queries/task_usage.sql` e `generated/**`), que se resolve por serialização de lock, não por
   fusão de migration.
3. **Fundir torna um item pronto hostage de dois não prontos.** Estado por §4 do próprio master:
   ORQ-13 = patch existente + PASS condicional; ORQ-12 = **design, patch ausente**; ORQ-21 = patch
   **REJECT** (zero callers, seed reativa). Um 128 atômico faz a única peça com peer review PASS
   esperar a peça rejeitada — e amplia o blast radius da migration financeira em vez de reduzi-lo.

**Amendment:** substituir "128 único reunindo ORQ-21/12/13" por três pacotes seriais na lane SQL, com
ORQ-13 primeiro por ser o único materializado e revisado (mapa em §3).

### A4 — Contradição interna de precedência: o número mais baixo foi dado ao artefato sem patch

Master §1.2 fixa a ordem canônica: *"1. schema/fonte/diff/status medidos no corte; … 4. design
revisado; 5. plano/relato do autor"*. Mas §3.1 atribui **127 ao ledger**, que é *"DESIGN PASS
condicionado a P0 secret; **sem patch**"*, e renumera o **único 127 materializado** (ORQ-13,
verificado por mim nos 18 worktrees). Isso inverte a própria regra de precedência do documento e
impõe custo de renomeação a quem tem evidência medida, em favor de quem tem design.

**Amendment:** aplicar §1.2 ao próprio mapa: material medido fica com o número que já ocupa.

### A5 — ORQ-18: o master ignora evidência anterior a ele e subclassifica o estado

Master §2.1 e §4: *"Direção recuperável; integração de produção retida por política de histórico e
testes faltantes"*; §4 lista o gate como *"incompleto: success/error/cancel/pending/refetch/permissão"*.
`grep -c "orq18-independent-test-audit" gtl-z01-master-integration-control.md` = **0**, embora
`.deploy-control/p0/evidence/gtl-orq18-independent-test-audit.md` exista desde **12:12** (antes do
master). Aquele audit prova, por execução no harness sancionado:

- base `runtime-row-menu.test.tsx` = exit 0, `Tests 7 passed (7)` em 111 ms;
- worktree ORQ-18 = **exit 124, não termina, zero linha de resultado** (reproduzido 2×);
- rodar da raiz `multica-auth-work` (sem `packages/views/vitest.config.ts:7-9`, que traz
  `setupFiles ./test/setup.ts`) dá **16 failed / 1 passed idêntico nas duas árvores** — logo qualquer
  PASS/FAIL de frontend obtido da raiz é inválido.

**Amendment:** (a) linha ORQ-18 de §2.1/§4 passa a *"PATCH BLOCK: o arquivo de teste alterado não
conclui sob o harness sancionado (GTL-42); produção segue HOLD por retenção"*; (b) §7.1 ganha dois
itens: *"12. testes de frontend executam no harness do pacote (`packages/views`), nunca da raiz"* e
*"13. término do processo é critério: exit code registrado; timeout = FAIL, não 'inconclusivo'"*.

### A6 — Retention: spec já tem PASS duplo, o master ainda a trata só como pendência de owner

Master §3.1 item 4 e §4: *"130 — `runtime_history_retention`, somente após decisão de retenção do
owner"*. Fatos mais novos: `gtl-runtime-retention-safe-lifecycle-v2.md:1,4` = *"GTL-82 — Specification
V2 … 12:20:34Z"* com `:147-148` *"STATUS: PASS (ESPECIFICAÇÃO TÉCNICA V2 APROVADA PARA SUBMISSÃO)"*, e
`gtl-runtime-retention-safe-lifecycle-v2-peer-review.md:8` = *"Veredito: **PASS** (Especificação
Técnica V2 Totalmente Aprovada)"* (12:22, portanto **posterior** ao master).

**Amendment:** *"retention: **spec V2 PASS (GTL-82) + peer review PASS**; pendência é apenas o ruling
de prazo/exclusão do owner (§8.1.2) e o número da migration"*. Isso muda a ordem prática: retention
deixa de ser o elo mais fraco da fila.

### A7 — A reserva não é durável e o próprio master admite

§3.1: *"**Ainda não é reserva operacional:** este relatório está untracked"*. Confirmo: o arquivo é
untracked no root e o root é o checkout misto que o próprio §2.2 proíbe como staging. Enquanto isso,
qualquer writer pode materializar um segundo `127`.

**Amendment:** o General-TL registra o mapa canônico no controle autoritativo com fingerprint próprio
**antes** de liberar qualquer L-RED-*; até lá, escrita de migration é STOP.

## 3. Mapa canônico proposto (substitui §3.1 do master)

| Nº | Conteúdo | Por quê nesta posição | Pré-condições |
|---|---|---|---|
| **127** | `task_usage_thinking_level` (**ORQ-13**, já materializado) | é o único `127` que existe em disco; PASS de código/testes (GTL-68); §1.2 do master manda material medido prevalecer; renomear custa churn sem ganho | critério GTL-68 #2 (commit de baseline sqlc separado) e #3 (prova em Postgres real); ruling Z01 = este mapa |
| **128** | `task_ledger_summary` (ledger V2) | design PASS sem patch: numerar depois não custa nada, e a lane fecha replay durável antes das dimensões financeiras | P0 do segredo HMAC resolvido por injeção sem valor em contexto |
| **129** | `task_usage_account_id` (**ORQ-12** snapshot no claim) | depende de autoridade de conta no backend, que hoje é a ORQ-21 **REJECT**; separar impede que o account bloqueie o tier | ORQ-21 reescrita (backend authority), sem o seed atual |
| **130** | `project_squad_lead` | independente da lane financeira; entra quando o lock SQLC liberar | — |
| **131** | `runtime_history_retention` | spec V2 PASS (GTL-82 + peer review); libera ORQ-18 produção | ruling de retenção do owner (§8.1.2) |
| **132** | `kanban_integrity` (se houver schema) | V7 está **BLOCK** e V8 é o artefato vivo; amarrar Kanban ao 131 assumia V7 | **GTL-80 PASS do V8**; não criar antes |

Regras que permanecem válidas do master, sem alteração: lane vermelha serial em `daemon.go`,
`task.go`, `handler/daemon.go`, `client.go`; lock global em `generated/**`; `sqlc generate` uma vez por
lane com segunda geração diff-zero; expand-first sem down destrutivo em migration financeira.

## 4. Itens do dispatch que fecham sem amendment

- **(3) worktrees/owners/overlap/waves:** verifiquei 18 worktrees e a sobreposição root × ORQ-23 nos
  dois helpers `execenv`; a regra "root não é staging" e o DAG de seis frentes da Wave 1 são
  consistentes com os locks declarados. Sem objeção.
- **(4) known-bad:** hashes conferem com a fonte citada; a distinção "cópia `gate1` é baseline válido
  por conteúdo mas só vira rollback após wrapper+drill" está correta.
- **(5) nove issues:** lista correta e nenhuma proposta de rerun; a regra `tool_use` prova início e não
  conclusão está preservada.
- **(6) owner × técnico:** correto, e reforçado pelo GTL-68, que endereça a renumeração à liderança
  técnica, não ao owner.
- **(7) segurança/skills:** citações corretas; Skills segue PENDING PROOF por falta de
  `skills-manifest.json` + SHA-256; `/tmp`/umask segue com contenção pendente e sem leitura de valor.

## 5. Não-alegações

- Não editei o master nem o repo; não rodei build, teste, migration, SQL, deploy, restart, chmod nem
  qualquer endpoint mutante nesta auditoria. Comandos: `sha256sum`, `wc`, `ls`, `nl`, `sed`, `grep`,
  `git worktree list`.
- Não reexecutei os gates de ORQ-26, Kanban, ledger, retention ou CLI: avaliei **precedência
  documental contra fingerprint/timestamp**, mais os fatos de schema/worktree que medi.
- Não afirmo que o V8 esteja correto: afirmo que ele existe, é posterior ao V7 e está sem peer review.
- Não afirmo que a CI possa rodar: afirmo que o mecanismo `push` em branch efêmera tem ruling oficial e
  PASS de manifesto, e que **push permanece STOP-AND-WAIT do owner** (B1–B4).
- Não li valor de segredo algum; a discussão de `arch.txt`/`backend-recover.sh` continua por marcador
  e modo, como no master.
- Este relatório não autoriza execução, commit, push, merge, migration, deploy ou restart.
