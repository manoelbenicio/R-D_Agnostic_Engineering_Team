# ORQ-21 R3 — re-review adversarial independente de `ee87f7b` (READ-ONLY)

- Revisor: Codex56#A (`w7:p3`) · UTC 2026-07-28T14:46Z
- Alvo: `ee87f7b` — *"fix(orq21): enforce exclusive credential assignment rollout"*, sobre `182b7b3`
  (imutável), base `0cb8aeb`; worktree do autor `gtl-orq21-r2`, branch `agent/codex56-b/orq21-r3`
- Escopo: **10 arquivos, +674/-49**. `git show ee87f7b --name-only | grep -c '^\.deploy-control'` = **0**
  → **code-only confirmado**, minha amenda A3 do R2 está fechada.
- `git show ee87f7b --check` → exit 0.
- Modo: READ-ONLY. Não editei arquivo do autor, não commitei, não fiz push, não toquei board nem
  produção. Reproduzi gates em **banco efêmero meu**, em porta e nome distintos do gate do owner.

## VEREDITO: **BLOCK único e estreito** — todos os defeitos anteriores fechados; um ramo de contenção passou a ser alcançável

Os três BLOCKs e as três amendas do R2 estão resolvidos, e eu os verifiquei um por um, inclusive
reproduzindo os gates de banco. Resta **um** defeito, introduzido justamente pela mudança que faz o
per-account home funcionar.

## 1. BLOCK — o ramo `envRoot` existente deixou de recusar e passou a apagar

`daemon.go` agora envia `CredentiallessGateway: credentialAccountHome == ""` (antes era `true` fixo).
Isso é **necessário e correto** para o Codex consumir `AccountHome`: com `true`, o execenv chama
`prepareCredentiallessCodexHome` e ignora a autenticação da conta. Não questiono a mudança em si.

O efeito colateral está em `execenv/execenv.go:234-242`:

```go
if _, err := os.Stat(envRoot); err == nil {
    if params.CredentiallessGateway {
        return nil, fmt.Errorf("execenv: credentialless gateway task root already exists; refusing to inspect or rewrite it")
    }
    if err := os.RemoveAll(envRoot); err != nil { ... }
}
```

Antes do R3, tarefas de providers cobertos sempre chegavam com `CredentiallessGateway=true`, então o
ramo de recusa era o único alcançável e o `RemoveAll` era código morto nesse caminho. Depois do R3,
**toda** tarefa com assignment aprovado chega com `false` e passa a executar `os.RemoveAll` sobre um
root pré-existente, em silêncio.

Agravante medido: `envRoot = WorkspacesRoot/WorkspaceID/shortID(TaskID)` e
`shortID` **trunca em 8 caracteres hex** (`execenv/git.go:187-193`). O espaço é 2^32, logo colisão de
prefixo entre tarefas do mesmo workspace é um problema de aniversário real (≈50% em ~77 mil tarefas, e
já mensurável em milhares). Numa colisão, o R3 apaga o root da tarefa anterior sem inspecionar, e é
exatamente no caminho que agora carrega credencial por conta.

Isto contradiz a propriedade que o próprio card defende: contenção verificada antes de usar. A guarda
existia para nunca "inspecionar ou reescrever" um root alheio; agora ela vale só para o modo sem
credencial.

**Fix exato — escolher A ou B, e A é o que eu recomendo:**

- **A (fail-closed simétrico)**: remover a condicional e recusar sempre que o root existir, com
  mensagem que não vaze caminho de credencial. O retry legítimo já cria **nova** linha de task e,
  portanto, novo `shortID`, então a recusa não bloqueia recuperação — isso está provado pelo próprio
  teste de recuperabilidade do R3.
- **B (se a remoção for realmente desejada)**: antes de `RemoveAll`, exigir que `envRoot` seja
  diretório **não-symlink**, de dono `os.Geteuid()` e com `Perm()&0o077 == 0` — a mesma validação que o
  R3 já aplica a root/slot/home em `validatePrivateOwnedPath` — e registrar a remoção em log.
  Adicionalmente, alargar `shortID` para esse caminho, ou usar o UUID completo, para reduzir colisão.

Escopo: `execenv/execenv.go` **não** está no conjunto de 10 arquivos do R3, então a correção implica ou
estender o FILES_LOCKED do card, ou tratar como handoff imediato ao dono de `execenv`. Enquanto não for
decidido, considero o R3 **não integrável**, porque o R3 é o que torna o ramo alcançável.

## 2. Todos os itens anteriores: fechados e verificados

| item | estado | evidência |
|---|---|---|
| exclusividade por conta (meu BLOCK-1 do R2) | **fechado** | resolver acrescenta `EXISTS(SELECT 1 FROM assignments other WHERE other.account_id = ass.account_id AND other.agent_id <> ass.agent_id)` → `ErrAccountAlreadyUsed`; seed acrescenta `account_assignment_conflict` → `E_ACCOUNT_ALREADY_ASSIGNED` sob `pg_advisory_xact_lock` |
| revogado nunca reativado | **fechado** | seed detecta `allowed = false` → `E_ACCOUNT_REVOKED`; o teste de banco reverte e ainda **assere** `if allowed { t.Fatal("seed rerun reactivated a revoked approval") }` |
| `GENERAL` como único vocabulário | **fechado, com honestidade** | resolver exige `*WorktypeScope == "GENERAL"`; seed rejeita com `E_WORKTYPE_SCOPE_UNSUPPORTED`; o comentário diz explicitamente que outros valores ficam armazenados para política futura e **não** devem ser apresentados como escopo aplicado — era exatamente a minha amenda A1 |
| `leased` só com posse exclusiva | **fechado** | `resolver_db_test` prova: conta `leased` **exclusivamente** atribuída resolve; a mesma conta compartilhada por dois agentes dá `ErrAccountAlreadyUsed`; `cooldown` dá `ErrAccountUnavailable` |
| mixed-version fail-closed, servidor-primeiro (meu BLOCK-2) | **fechado** | `MULTICA_CREDENTIAL_ASSIGNMENT_ENFORCED`: com o gate ligado, provider coberto e payload dizendo `required=false`, o daemon recusa; o payload só **aperta**, nunca afrouxa |
| `agy` → `antigravity` antes do execenv | **fechado** | `daemon.go` canonicaliza `claimedProvider` antes da busca de config e antes do execenv, com comentário explicando que o alias não pode contornar o HOME isolado do Antigravity |
| contenção: root/slot/home privados, do dono, sem symlink | **fechado** | `validatePrivateOwnedPath` exige `Lstat` sem `ModeSymlink`, `Perm()&0o077 == 0` e `Uid == os.Geteuid()`; aplicado a `root`, `Dir(clean)` e `clean`; mais `EvalSymlinks` no root controlado e `pathIsControlledSlotHome` |
| layouts por provider | **fechado** | `codex` → `auth.json`; `kiro` → `kiro-cli/data.sqlite3`; `antigravity` → `.gemini/antigravity-cli` com checagem de modo e symlink |
| retry recuperável (minha amenda A2) | **fechado** | há teste dedicado de recuperabilidade; o retry cria **nova linha** de task, então o cancelamento na falha de assignment não sequestra o trabalho |
| `AccountID` preservado para o snapshot do ORQ-12 (meu BLOCK-3) | **parcialmente fechado, e é o correto** | `slog.Info("task claim: approved credential assignment resolved", …, "account_id", …)` server-side; o payload continua com **apenas** home e booleano. A persistência durável segue dependendo da coluna da migration 128 (ORQ-12/LANE-DB) — fronteira respeitada |
| `\quit 1` fail-open no seed | **fechado** | todas as guardas passaram a `DO $$ BEGIN RAISE EXCEPTION 'E_…'; END $$;`, que aborta a transação de verdade em vez de sair do cliente com o `BEGIN` pendente |
| evidência misturada no commit | **fechado** | 0 arquivos `.deploy-control` no commit |

## 3. Gates que eu reproduzi de forma independente

Banco efêmero **meu**: container `codex56a-orq21r3-review-20260728T144029Z`, ORQ1 loopback `15551`,
túnel local `25551` — deliberadamente distinto do gate do owner em `15546`/`25546`, que eu não toquei
(confirmei ao final que `orq21-r3-pg-20260728` continua no ar).

```text
go run ./cmd/migrate up                                        -> Done, até 126
go vet ./internal/credentialregistry ./internal/daemon ./internal/handler -> exit 0
go build ./cmd/server                                          -> exit 0
go test -tags orq21db ./internal/credentialregistry -count=1    -> 5/5 PASS, ok 0.315s, zero skip
  TestResolverDBApprovedAssignmentContract              PASS
  TestSeedApprovedAssignmentIdempotencyRevocationAndExclusivity  PASS
  TestCanonicalProvider / TestRequiresApprovedAssignment / TestValidAbsoluteMetadataPath  PASS
go test ./internal/handler -run '^TestORQ21' -count=1 -v        -> 2/2 PASS, zero skip
  TestORQ21ClaimIncludesOnlyApprovedAssignmentMetadata   PASS
  TestORQ21ClaimCancelsWithoutApprovedAssignment         PASS
go test ./internal/daemon -run Credential -count=1             -> ok 17.699s
go test ./internal/daemon -run Credential -race -count=1       -> ok 17.833s
go test -tags orq21db ./internal/credentialregistry -race      -> ok 1.300s
```

**Prova incidental de anti-falso-verde**: na minha primeira execução os dois testes de banco
**falharam** com `ORQ21_TEST_DATABASE_URL is required for the orq21db gate`. Ou seja, sem a variável o
gate **falha**, não faz skip. Foi um erro meu de configuração e virou a melhor evidência de que o
pacote não aceita verde por ausência de ambiente.

Teardown completo: removi os dois containers meus (incluindo o que falhou o bind em `15546`), encerrei
o túnel `25551`, e `docker ps -a | grep -c codex56a` = **0**. Nenhum recurso do owner foi alterado.

## 4. Não-alegações

- Não editei, commitei, pushei nem toquei board; nenhum arquivo do autor foi alterado.
- Não reproduzi o diferencial completo do pacote `internal/handler` (base × candidato) alegado pelo
  autor: rodei apenas os testes ORQ-21 e os focados. As 3 falhas pré-existentes eu já havia medido de
  forma independente em outra task, e não contesto o número de skips relatado — apenas **não o
  verifiquei**.
- Não executei o seed manualmente com fixtures próprias: minha tentativa de montar workspace/agent por
  SQL cru falhou por colunas obrigatórias, e em vez de forçar preferi apoiar-me no teste de banco do
  autor, que **assere** revogação, idempotência e exclusividade e que eu executei.
- Não avaliei ainda o comportamento com `MULTICA_CREDENTIAL_ASSIGNMENT_ENFORCED=1` num daemon real: a
  verificação é de código e de teste unitário, não de execução em runtime.
- Não li segredo algum; as credenciais do banco efêmero são literais descartáveis em loopback.
- A colisão de `shortID` é uma propriedade combinatória do código (8 hex), não um incidente observado.
