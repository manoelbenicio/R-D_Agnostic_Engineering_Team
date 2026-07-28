# ORQ-41 — Auditoria de colisão e reserva de migrations (READ-ONLY)

- Card: **ORQ-41** · UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`
- Auditor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Data UTC: 2026-07-27T13:45Z
- Modo READ-ONLY. Nenhuma edição de código, migration, query, arquivo gerado, board, build,
  teste ou segredo. Nenhum `assignee` definido, nenhum comentário postado (freeze de
  atribuição/comentário respeitado). Única escrita: este arquivo.

## VEREDITO: **PASS** na auditoria — com **1 colisão ativa** que exige decisão do GTL antes de qualquer merge

A sequência canônica está sã. A colisão não é no disco canônico: é **entre quatro designs que
todos reservaram o número 127**, sendo que um deles já materializou os arquivos num worktree.
Nenhuma ação corretiva é possível sem decisão, por isso o item fica como decisão pendente e não
como defeito de execução.

---

## 1. Sequência canônica medida

```console
$ cd multica-auth-work/server/migrations
163 *.up.sql   ·   163 *.down.sql   ·   nenhum up sem down
min=001   max=126   nenhum gap numérico
```

Últimos aplicáveis, em ordem: `120_autopilot_subscriber`, `120_comment_source_task_id`,
`120_github_pending_installation`, `120_runtime_profile`,
`121_agent_runtime_provider_partial_unique`, `122_lark_chat_session_binding_thread_reply`,
`123_rotation`, `124_approved_accounts`, `125_user_password_credential`,
`126_runtime_profile_protocol_family_native_runtimes`.

### 1.1 Fato que muda o enquadramento do risco: número repetido **já é normal** aqui

26 prefixos numéricos são compartilhados por mais de um arquivo no tronco canônico:
`020, 026, 029, 032, 033, 035, 040, 041, 043, 046, 050, 060, 065, 069, 079, 083, 084, 091,
095, 096, 098, 109, 111, 112, 113, 120`.

Isso é possível porque a identidade da migration é o **basename completo**, não o número:

- `cmd/migrate/main.go:236` → `version := migrations.ExtractVersion(file)`;
- `schema_migrations(version TEXT PRIMARY KEY)` (`main.go:221-227`);
- ordem de aplicação = `sort.Strings(files)` (`internal/migrations/migrations.go:67`), com
  `sort.Reverse` para `down` (`:65`);
- já aplicada ⇒ `skip` (`main.go:243-246`);
- toda a execução sob `pg_advisory_lock` (`main.go:215-218`), logo duas execuções concorrentes
  serializam;
- `migrations.AllVersions()` verifica **todas** as versões em disco contra `schema_migrations`,
  e o comentário em `migrations.go:72-76` diz explicitamente que checar só a última lexicográfica
  perderia uma migration fora de ordem.

**Consequência:** um número duplicado **não quebra o runner** e não quebra o readiness. O que ele
quebra é (a) a intenção de ordem, porque o desempate passa a ser lexicográfico sobre o resto do
nome; (b) o merge, porque duas lanes editam a mesma faixa; (c) a revisão, porque "127" deixa de
identificar unicamente um pacote.

### 1.2 Invariante de formato que precisa entrar na reserva

Todos os prefixos são de **3 dígitos com zero à esquerda**. Como a ordenação é lexicográfica,
isso só é seguro até `999`. Um prefixo de 4 dígitos (`1000_`) ou sem padding (`99_`) ordenaria
**antes** de `126_`, aplicando fora de ordem. Regra a fixar na reserva: exatamente 3 dígitos,
`127`–`999`.

---

## 2. Estado por worktree (21 árvores inspecionadas)

| worktree | up | max | extra vs canônico |
|---|---|---|---|
| `R-D_Agnostic_Engineering_Team` (canônico) | 163 | 126 | — |
| `worktrees/gtl-i03-orq13-phase1` | **164** | **127** | **`127_task_usage_thinking_level.{up,down}.sql`** |
| `worktrees/{gtl-orq26, gtl-orq26-consumer-tests, gtl-orq17-auth-tests, ci-orq26-db-gate, ci-orq39-browser-qa, p0-w1-central, p0-w2-gateway, p0-w3-runtime}` | 163 | 126 | — |
| `multica_workspaces/.../{91e70c79, d4f7f744, e39199ed, ff121b28}` | 163 | 126 | — |
| `/mnt/shared/agent-worktrees/{agent-brain-p0-integration, p0-acceptance, p0-failures, p0-lifecycle, p0-retry, p0-routes, p0-w4-ops}` | 163 | 126 | — |

**Uma única migration nova existe em qualquer lugar**, e ela está **untracked**:

```console
$ cd worktrees/gtl-i03-orq13-phase1 && git status --porcelain
 M multica-auth-work/server/pkg/db/generated/models.go
 M multica-auth-work/server/pkg/db/generated/task_message.sql.go
 M multica-auth-work/server/pkg/db/generated/task_usage.sql.go
 M multica-auth-work/server/pkg/db/queries/task_usage.sql
?? multica-auth-work/server/migrations/127_task_usage_thinking_level.down.sql
?? multica-auth-work/server/migrations/127_task_usage_thinking_level.up.sql
$ git log -1 -- .../127_task_usage_thinking_level.up.sql   → (vazio: não versionada)
```

---

## 3. Colisão ativa: quatro designs reservaram **127**

| # | migration proposta | origem | estado |
|---|---|---|---|
| 1 | `127_task_usage_thinking_level` | `gtl-i03-orq13-phase1-implementation.md`, revisado em `gtl-orq13-phase1-peer-review.md` | **materializada em disco** (untracked, ORQ-13 Fase 1) |
| 2 | `127_task_usage_account_id` | `cost-accounting-design.md:63,66,85,155,349` (lane de custo/ORQ-12) | só design |
| 3 | `127_task_usage_reasoning_tier` | `gtl-orq13-salvage-map.md`, `gtl-orq13-salvage-peer-review.md` | só design; parece variante anterior do item 1 |
| 4 | `127_task_ledger_summary` | `gtl-ledger-v2-corrected-design.md`, `gtl-ledger-v2-peer-review.md`, e a reserva do `gtl-cross-gate-execution-plan.md:105` | só design |

O `gtl-cross-gate-execution-plan.md:97-109` já havia diagnosticado parte disso — "GTL-02 reserva
migration 127 para ledger; GTL-03 também assume migration 127" — e propôs a serial
`127 ledger → 128 account/tier → 129 squad → 130 retention → 131 kanban`. O
`gtl-z01-master-integration-control.md` cita simultaneamente `127_task_ledger_summary`,
`127_task_usage_account_id` e `127_task_usage_thinking_level`, o que confirma que o conflito é
real e não hipotético.

### 3.1 Divergência entre a reserva do plano cross-gate e o disco

O plano reservou `127` para o **ledger**, mas quem **materializou** `127` foi o **ORQ-13**
(`thinking_level`). Se ambos forem integrados sem renumerar, o resultado é dois arquivos `127_*`
aplicando em ordem lexicográfica `127_task_ledger_summary` **antes** de
`127_task_usage_thinking_level` — ordem que ninguém escolheu.

### 3.2 Reserva insegura herdada do GTL-08

`squad-lead-migration-design.md:19,60` propõe **`105_project_squad_lead`**. O tronco já está em
126 e `105_*` já existe como faixa aplicada; um `105_` novo ordenaria **antes** de tudo o que já
foi aplicado. O runner o aplicaria (é um basename novo, não presente em `schema_migrations`), mas
depois de 21 migrations posteriores — exatamente o cenário "out-of-order" que o comentário em
`migrations.go:72-76` descreve. **Reserva a rejeitar.**

### 3.3 Placeholders que talvez não precisem de número

- `130_runtime_history_retention`: o plano cross-gate já o condiciona a "somente se o owner
  escolher schema". `gtl-runtime-retention-safe-lifecycle-v2.md` só referencia
  `004_agent_runtime_loop` e `120_runtime_profile`, ou seja **nenhuma migration nova** aparece no
  design v2. Manter como reserva condicional, não alocada.
- `131_kanban_integrity_monitor`: `gtl-kanban-integrity-monitor-v2.md` descreve **runner
  background** (`server/cmd/server/kanban_integrity_monitor.go`) e um teste; a busca por
  `CREATE TABLE`/`ALTER TABLE`/`migration` nesse design **não retorna nada**. Recomendo
  **não reservar** número até que um redraft prove necessidade de schema.

---

## 4. Riscos exatos de colisão, por camada

### 4.1 `server/migrations/**`

| risco | severidade | evidência |
|---|---|---|
| Dois arquivos `127_*` de lanes diferentes | **alta** | §3; ordem passa a ser desempate lexicográfico, não intenção |
| `105_project_squad_lead` aplicando fora de ordem | **alta** | §3.2 |
| Conflito de merge git na mesma faixa numérica | média | duas lanes criando arquivos vizinhos; git não conflita em arquivos distintos, então o conflito é **semântico**, não textual — pior, porque passa pelo merge sem alarme |
| Prefixo fora de 3 dígitos | baixa hoje, alta se ocorrer | §1.2 |

### 4.2 `server/pkg/db/queries/**`

Colisão textual real quando duas lanes editam o **mesmo arquivo** de queries. Medido: o ORQ-13
já modifica `queries/task_usage.sql`, e a lane de custo (`127_task_usage_account_id`) mexe na
**mesma tabela `task_usage`**, logo quase certamente no mesmo arquivo. Probabilidade de conflito
textual: alta. Precedente de coordenação já existe: o meu worktree ORQ-26 tocou
`queries/issue.sql` e o `pkg/db/generated/issue.sql.go` numa lane exclusiva.

### 4.3 `server/pkg/db/generated/**` (sqlc)

Este é o ponto mais frágil e o plano cross-gate já o declarou **lane exclusiva**, com razão.
Medido no ORQ-13: `generated/models.go`, `generated/task_message.sql.go`,
`generated/task_usage.sql.go` estão modificados. `models.go` é **um único arquivo para todo o
schema**, então **qualquer** duas lanes que adicionem coluna/tabela colidem textualmente nele.
Regeneração (`make sqlc`) sobre um merge parcial produz diff que parece correto e não é.

Regra derivada: `generated/**` nunca é editado à mão e nunca é mergeado de dois lados —
apenas **regenerado após** a integração sequencial das queries.

---

## 5. Tabela de reserva proposta (ordenada, durável)

Proposta, não decidida. Baseia-se em preservar o que já está materializado, o que é o critério
menos destrutivo.

| ordem | versão reservada | pacote | dono / wave | pré-condição |
|---|---|---|---|---|
| 1 | **`127_task_usage_thinking_level`** | ORQ-13 Fase 1 (thinking level em `task_usage`) | wave A — ORQ-13 | **já materializada em disco**; mantém-se por ser a única existente |
| 2 | **`128_task_ledger_summary`** | ledger durável | wave B — ledger (GTL-02) | renumerar de 127→128 no design v2 |
| 3 | **`129_task_account_usage_dimensions`** | account/tier em `task_usage` | wave C — custo (GTL-03 / ORQ-12) | absorve `127_task_usage_account_id`; renumerar |
| 4 | **`130_project_squad_lead`** | squad lead em `project` | wave D — squad (GTL-08) | **substitui** `105_project_squad_lead`, que é reserva inválida |
| 5 | `131_runtime_history_retention` | retenção de histórico de runtime | wave E — retenção (ORQ-19) | **condicional**: só se o owner escolher a opção com schema |
| 6 | *(não reservar)* | monitor de integridade do Kanban | wave F — GTL-14 | só reservar se um redraft provar necessidade de DDL |

Regras que acompanham a tabela:

1. Prefixo com exatamente **3 dígitos**, faixa `127`–`999`; nunca reutilizar número já em disco.
2. Nome do arquivo em `snake_case`, `<nnn>_<escopo>_<objeto>`, com par `.up.sql`/`.down.sql`
   obrigatório — o tronco hoje tem 163/163, invariante a preservar.
3. `127_task_usage_reasoning_tier` fica **retirada**: é variante histórica do item 1, não um
   pacote independente. Registrar como superseded, não como reserva.
4. Uma wave só começa depois de a anterior estar integrada **e** o worktree seguinte rebaseado —
   é a regra que o `gtl-cross-gate-execution-plan.md:111-113` já propôs e que eu endosso.
5. `pkg/db/generated/**` é lane exclusiva de uma wave por vez; regeneração sempre pós-merge.
6. A reserva é registrada **neste arquivo de evidência**, e a fonte de verdade do que existe
   continua sendo `ls server/migrations/*.up.sql` no tronco canônico, reconferida no branch cut.

---

## 6. Reconciliação com o mapa GTL-83

`gtl-z01-master-integration-control.md` e `gtl-z01-master-integration-peer-review.md` (GTL-83 /
Z01) citam `126_runtime_profile_protocol_family_native_runtimes` como último aplicado — **confere
com o disco** — e listam os três candidatos a 127 sem escolher. Portanto o mapa GTL-83 não está
errado; está **incompleto** quanto ao desempate, e esta auditoria fornece o desempate proposto
(§5). A reserva do `gtl-cross-gate-execution-plan.md` está **um passo deslocada** frente ao disco
e precisa do renumber de 127→128 no ledger.

---

## 7. Riscos que eu NÃO posso fechar sem execução

| # | item | por quê |
|---|---|---|
| E-1 | Se `127_task_usage_thinking_level` aplica limpo sobre o schema atual | exigiria banco e `go run ./cmd/migrate up`, proibido aqui |
| E-2 | Se `make sqlc` reproduz exatamente os `generated/**` já modificados no ORQ-13 | exigiria rodar o gerador |
| E-3 | Conteúdo real de conflito entre `queries/task_usage.sql` do ORQ-13 e a lane de custo | a segunda ainda não existe em disco |
| E-4 | Se o `schema_migrations` do ambiente vivo já contém algo além de `126_*` | leitura de banco live, fora de escopo desta tarefa |

---

## Check-out — ORQ-41 · `666f1ead-7fe9-4051-bab9-5d0a936c4701`

- Veredito: **PASS** na auditoria; **decisão pendente** do GTL sobre a colisão de 127 e sobre a
  tabela de reserva de §5. Nada bloqueia a produção da reserva; o que bloqueia é integrar duas
  lanes na mesma faixa.
- Medido: 163 up/163 down, min 001, max 126, 26 números duplicados já existentes no tronco, 21
  worktrees inspecionados, uma única migration nova (`127_task_usage_thinking_level`, untracked,
  ORQ-13), quatro designs reivindicando 127, uma reserva inválida (`105_project_squad_lead`), e
  semântica do runner confirmada em `cmd/migrate/main.go:215-290` e
  `internal/migrations/migrations.go:65-79`.
- Correções propostas: renumerar ledger 127→128, custo→129, squad 105→130, retenção condicional
  em 131, Kanban sem número, `reasoning_tier` marcada superseded.
- Não executei: nenhuma migration, build, teste, `sqlc`, DDL ou leitura de banco live. Nenhuma
  mutação de card, nenhum `assignee`, nenhum comentário. ORQ-39 e ORQ-40 não foram reexecutados.
- Frentes congeladas intocadas: worktree ORQ-26 (`48553c6c…`/`815b7d1c…`) e ORQ-17.
