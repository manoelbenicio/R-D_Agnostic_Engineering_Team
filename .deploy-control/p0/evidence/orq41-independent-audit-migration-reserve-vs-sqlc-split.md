# ORQ-41 — Auditoria independente: reserva de migrations × split SQLC (READ-ONLY)

- Card: **ORQ-41** · UUID `666f1ead-7fe9-4051-bab9-5d0a936c4701`
- Auditor: Codex56#A (`w7:p3`) · UTC 2026-07-27T14:20Z
- Documentos auditados: `orq41-migration-reservation-audit.md` (Kiro-Opus5, 13:45Z) e
  `orq13-sqlc-split-manifest.md` (Opus48#B, 13:50Z)
- Modo: READ-ONLY. Sem DB live, sem `sqlc`, sem build/teste, sem edição de código, sem mutação de
  board, sem `assignee`, sem comentário. Única escrita: este arquivo e o check-out.

## 0. Veredito

**PASS com 4 emendas.** Os dois documentos são factualmente sólidos — reverifiquei as afirmações
centrais de ambos e todas conferem. O que falta é **coerência entre eles**: a tabela de reserva não
tem lugar para o commit A (baseline SQLC), e o split não pode existir sob a regra "generated nunca é
mergeado" como está escrita. Nenhuma emenda invalida os documentos; todas mudam a **ordem segura**.

## 1. Reverificação independente (o que eu medi, não o que li)

| afirmação | fonte | meu resultado |
|---|---|---|
| 163 up / 163 down, min 001, max 126, sem gap | reserva §1 | **CONFIRMADO**: `up=163 down=163 min=001 max=126` |
| 26 prefixos numéricos duplicados já no tronco | reserva §1.1 | **CONFIRMADO**: `count_dup=26`, lista idêntica (`020 026 029 032 033 035 040 041 043 046 050 060 065 069 079 083 084 091 095 096 098 109 111 112 113 120`) |
| identidade da migration é o basename; ordem lexicográfica; advisory lock | reserva §1.1 | **CONFIRMADO** em `cmd/migrate/main.go` (`ExtractVersion`, `schema_migrations`, `pg_advisory_lock` em :206) |
| `105_project_squad_lead` é reserva inválida | reserva §3.2 | **CONFIRMADO**: `squad-lead-migration-design.md:19` e `:60` propõem literalmente `105_project_squad_lead.up.sql` |
| drift do generated: 6 structs ausentes | split §2.1 | **CONFIRMADO**: `Account`, `ApprovedAccount`, `Assignment`, `Credential`, `RotationEvent`, `UserPasswordCredential` → `grep -c "^type X struct"` = **0** no `models.go` canônico, embora `123_rotation`, `124_approved_accounts` e `125_user_password_credential` existam em disco |
| `task_message.sql.go` foi editado à mão (`maxSeq` vs alias `max_seq`) | split §2.2 | **CONFIRMADO**: `:56 AS max_seq` no SQL, `:63-65 maxSeq` no Go — o gerador não emitiria isso |
| sem `.golangci.yml` em nenhum nível | split §2.2 | **CONFIRMADO**: `find -maxdepth 3 -name ".golangci*"` vazio ⇒ defaults do golangci-lint v2 |
| workflows só existem sob `multica-auth-work/.github/workflows` | split §2.2 | **CONFIRMADO**: raiz **não** tem `.github/workflows`; o inner tem `ci.yml`, `desktop-smoke.yml`, `mobile-verify.yml` |
| `/tmp` do ORQ2 sem espaço | split §2.3 | **CONFIRMADO e ainda vigente**: `tmpfs 7.7G, 7.6G usados, 114M livres, 99%`; `TMPDIR`/`GOCACHE` seguem **unset** |

Nenhuma discrepância factual entre os dois documentos e o disco.

## 2. Colisões semânticas encontradas (o foco pedido)

### C-1 (alta) — A tabela de reserva não tem slot para o **commit A**, que precede tudo

A reserva (§5 do doc de reserva) começa em `127_task_usage_thinking_level` = "wave A — ORQ-13". Mas o
split é explícito: **commit A antes de B**, porque A é o baseline do gerador e, se B entrar primeiro,
"o próximo `sqlc generate` de qualquer pessoa arrasta o drift de A para dentro de um PR alheio".
O commit A **não tem migration**, logo não aparece na tabela — e por isso a tabela, lida
isoladamente, autoriza começar pela feature.

**Emenda:** inserir **wave 0 — `chore(db)` baseline SQLC, sem número de migration**, como
pré-condição de *todas* as waves (não só da 127). Sem isso, a wave B (ledger) ou a C (custo)
regeneram e arrastam o mesmo drift.

### C-2 (alta) — Regra "generated nunca é mergeado" × commit A, que é um commit de generated

Reserva §4.3 e regra 5: *"`generated/**` nunca é editado à mão e nunca é mergeado de dois lados —
apenas **regenerado após** a integração sequencial das queries"*. O commit A é exatamente um commit
de `generated/**` (2 arquivos, 7 hunks) **sem** mudança de query correspondente. As duas coisas não
podem ser verdadeiras ao mesmo tempo como escritas.

**Emenda:** reformular a regra para: *generated nunca é editado à mão; é permitido **um** commit de
generated por lane, cujo conteúdo seja exatamente a saída de `sqlc generate` no HEAD integrado,
provado por segunda execução com diff zero.* Assim o commit A é legítimo — deixa de ser "merge de
generated" e passa a ser "materialização do gerador". O critério de aceite do split (§2.3,
`sqlc generate` idempotente) já satisfaz essa formulação.

### C-3 (alta) — `models.go` compartilhado entre A e B torna o split **por hunk**, e isso quebra a
premissa de lane exclusiva por arquivo

O split §6 identifica corretamente que `models.go` é o **único** arquivo compartilhado (5 hunks em A,
1 em B) e que o split precisa ser por hunk. Já a reserva trata `generated/**` como lane exclusiva por
**arquivo**. Consequência prática: um `FILES_LOCKED` por arquivo faz A e B colidirem no lock, e o
mecanismo de lock não sabe expressar "hunk".

**Emenda:** para a wave 0 e a wave 127, declarar lock por **arquivo com owner único e sequência
temporal** (A e B são o *mesmo* owner, em ordem), e proibir qualquer outra lane de tocar
`generated/**` no intervalo. Não tentar dividir lock por hunk — o hunk é unidade de commit, não de
lock.

### C-4 (média) — O risco de lint do commit A é um **bloqueador de cadeia**, ausente da reserva

O split §2.2 mostra que A **reverte uma edição manual** (`maxSeq` → `max_seq`) e que `max_seq` é
exatamente o padrão que `ST1003` reprova; sem `.golangci.yml`, valem os defaults. A reserva não
menciona isso. Como A precede tudo (C-1), um lint vermelho em A **bloqueia 127, 128, 129 e 130**.

**Emenda:** o gate da wave 0 inclui `golangci-lint run` **antes** de A ser aceito, e a decisão de
excluir `pkg/db/generated/` do linter (mudança de outro dono, como o próprio split observa) passa a
ser **pré-requisito declarado** da wave 0, não um item paralelo. Mitigação alternativa registrada:
como o `ci.yml` vive em `multica-auth-work/.github/workflows` e o GitHub Actions só lê a raiz, esse
gate **hoje não roda automaticamente** — o que reduz o risco de CI vermelho, mas **não** o risco de
código gerado divergindo do padrão do repo.

## 3. Pontos onde os dois documentos concordam e eu endosso

1. **127 fica com o ORQ-13.** É o único materializado em disco (confirmei: 1 migration nova em 21
   worktrees), e a reserva do `gtl-cross-gate-execution-plan` (127 = ledger) está deslocada. Isto
   coincide com a emenda A4 do meu GTL-83, feita antes e de forma independente.
2. **Ledger 127→128, custo→129, squad 105→130, retenção condicional em 131, Kanban sem número.**
3. **`127_task_usage_reasoning_tier` é variante histórica** do mesmo pacote, não reserva própria.
4. **Migration antes do binário** (split §3.2): sem a coluna, o `UpsertTaskUsage` de B falha contra
   banco. Isso torna a ordem "wave 0 → migration 127 → binário 127" obrigatória, não preferencial.
5. **Prefixo de 3 dígitos, faixa 127–999** (reserva §1.2). Endosso: a ordenação é lexicográfica, logo
   `1000_` ou `99_` aplicariam fora de ordem.
6. **Commit B está limpo de `account_id`/pricing/rollup** (split §5.1) — as ocorrências são comentário
   ou código pré-existente. Isso é o que permite 127 e 129 serem waves separadas sem acoplamento.

## 4. Ordem segura consolidada (com as 4 emendas aplicadas)

```text
wave 0  chore(db) baseline SQLC              sem migration   gate: sqlc idempotente + build + vet + gofmt + LINT
        └─ pré-condição: decisão sobre exclusão de generated/ no linter
wave A  127_task_usage_thinking_level        migration ANTES do binário; commit B do split
wave B  128_task_ledger_summary              renumerar no design; regenerar generated após integrar
wave C  129_task_account_usage_dimensions    absorve 127_task_usage_account_id
wave D  130_project_squad_lead               substitui a reserva inválida 105_*
wave E  131_runtime_history_retention        condicional a ruling de retenção do owner
wave F  (sem número) monitor Kanban          só reservar se um redraft provar DDL
```

Invariantes por wave: base limpa e rebaseada; `generated/**` exclusivo de uma wave; `sqlc generate`
uma vez, com segunda execução diff-zero; nunca reutilizar número em disco; par `.up`/`.down`
obrigatório (163/163 hoje); e, por causa do `/tmp` em 99%, **todo gate roda com `TMPDIR`, `GOCACHE` e
`GOTMPDIR` fora de `/tmp`** — hoje ambos estão `unset`, então isso precisa ser explícito no comando,
não presumido.

## 5. O que continua não fechável sem execução (concordo com ambos)

`E-1` migration aplicar limpo · `E-2` `sqlc generate` reproduzir exatamente os `generated/**` do
ORQ-13 · `E-3` conflito real entre `queries/task_usage.sql` do ORQ-13 e a lane de custo (a segunda
não existe em disco) · `E-4` conteúdo de `schema_migrations` no ambiente vivo · **E-5 (meu)** se
`golangci-lint` v2 default reprova `max_seq` — decide a viabilidade da wave 0 e ninguém pode afirmar
sem rodar o linter.

## 6. Não-alegações

- Não rodei migration, `sqlc`, build, teste, linter, DB live nem li `schema_migrations`.
- Não editei código, migration, query, generated, board ou os dois documentos auditados.
- Não renomeei nem reservei número: a reserva permanece proposta até ruling do GTL.
- Não reexecutei os gates já reportados pelo split (build/vet/testes com A+B juntos); cito-os como
  relato do autor, não como prova minha.
- Não avaliei o mérito do design de ledger, custo, squad, retenção ou Kanban — apenas a numeração e a
  ordem.
