# Inventário de Falso-Verde e Sondas de Testes DB/Env (ORQ-13 Reviewer Feed)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Escopo:** `multica-auth-work/server` (*_test.go)
- **Modo:** READ-ONLY (Zero execução de testes, banco, API, docker ou escritas no board)
- **Data UTC:** 2026-07-27T16:17:00Z
- **Finalidade:** Fornecer inventário exato de `TestMain`, `os.Exit(0)`, skips por `DATABASE_URL` e saídas precoces sem `m.Run()` para o revisor final da ORQ-13.

---

## 1. Tabela TSV de Inventário (`package\tfile\tline\tpattern\trisk`)

```tsv
package	file	line	pattern	risk
main	cmd/server/integration_test.go	55	TestMain -> DB error -> os.Exit(0) sem m.Run()	HIGH
main	cmd/server/integration_test.go	60	TestMain -> DB ping fail -> os.Exit(0) sem m.Run()	HIGH
handler	internal/handler/handler_test.go	48	TestMain -> DB error -> os.Exit(0) sem m.Run()	HIGH
handler	internal/handler/handler_test.go	53	TestMain -> DB ping fail -> os.Exit(0) sem m.Run()	HIGH
passwordtest	internal/handler/passwordtest/provision_test.go	4	Sub-pacote isolado sem TestMain para evitar os.Exit(0)	MEDIUM
main	cmd/backfill_codex_usage_cache/integration_test.go	28	t.Skip("integration test requires Postgres at DATABASE_URL")	LOW
main	cmd/migrate/migrate_concurrent_test.go	85	t.Skipf("could not connect to %s: %v", dbURL, err)	LOW
main	cmd/migrate/migrate_concurrent_test.go	89	t.Skipf("database not reachable at %s: %v", dbURL, err)	LOW
main	cmd/migrate/migrate_concurrent_test.go	421	t.Skipf("parse DATABASE_URL: %v", err)	LOW
main	cmd/migrate/migrate_concurrent_test.go	435	t.Skipf("could not open small pool: %v", err)	LOW
main	cmd/migrate/migrate_concurrent_test.go	439	t.Skipf("small pool not reachable: %v", err)	LOW
metrics	internal/metrics/business_sampler_pgsleep_test.go	39	t.Skip("DATABASE_URL not set...")	LOW
scheduler	internal/scheduler/stale_steal_test.go	15	Helper integrationPool(t) -> t.Skip("no database connection")	LOW
taskusagebackfill	internal/taskusagebackfill/hook_test.go	47	t.Skip("integration test requires Postgres at DATABASE_URL")	LOW
taskusagebackfill	internal/taskusagebackfill/hook_test.go	166	t.Skip("integration test requires Postgres at DATABASE_URL")	LOW
middleware	internal/middleware/auth_test.go	28	t.Skip("REDIS_TEST_URL not set")	LOW
agent	pkg/agent/claude_deadlock_test.go	18	TestMain re-entry de processo subprocesso simulado	INFO
main	cmd/multica/cmd_auth_test.go	12	TestMain padrão com os.Exit(m.Run())	INFO
```

---

## 2. Análise Detalhada dos Padrões de Risco

### A. Risco ALTO (Falso-Verde em Nível de Pacote)
1. **`cmd/server/integration_test.go` (Linhas 45-60)**:
   - Se `DATABASE_URL` estiver ausente ou o Postgres inacessível, `TestMain` imprime mensagem no stdout e chama `os.Exit(0)` **antes** de executar `m.Run()`.
   - **Consequência:** O test runner do Go interpreta `os.Exit(0)` como aprovação (PASS), e 100% dos testes de integração do pacote `cmd/server` são silenciosamente ignorados sem figurar como `SKIP` nos relatórios de teste.

2. **`internal/handler/handler_test.go` (Linhas 38-54)**:
   - Se `DATABASE_URL` estiver ausente ou o Postgres inacessível, `TestMain` imprime mensagem no stdout e chama `os.Exit(0)` **antes** de executar `m.Run()`.
   - **Consequência:** O test runner do Go interpreta `os.Exit(0)` como aprovação (PASS), e todo o pacote `handler` (que contém quase 4000 linhas de testes de API) é marcado como `ok` (PASS) de forma falsa.

### B. Risco MÉDIO (Sub-pacotes Isolados para Desvio de TestMain)
1. **`internal/handler/passwordtest/provision_test.go` (Linhas 4-7)**:
   - Mantido como um pacote Go separado (`package passwordtest`) para não herdar o `TestMain` do pacote `handler`, que encerraria com `os.Exit(0)` quando o banco não estivesse disponível.

### C. Risco BAIXO / ACEITÁVEL (Uso de `t.Skip` / `t.Skipf`)
- Pacotes como `backfill_codex_usage_cache`, `migrate`, `metrics`, `scheduler`, `taskusagebackfill` e `middleware` usam `t.Skip()` / `t.Skipf()` dentro dos métodos de teste individuais.
- **Diferencial:** O `t.Skip()` é registrado corretamente na telemetria do runner Go como `SKIP`, sem gerar falso-positivo de aprovação (`PASS`).

---

## 3. Recomendações para o Revisor da ORQ-13
1. Substituir `os.Exit(0)` em `TestMain` por falha explícita (`os.Exit(1)`) ou descontinuação em CI de produção quando o banco de testes obrigatório não responder.
2. Garantir que pipelines de CI forneçam `DATABASE_URL` explícita ou validem o número de testes executados (`> 0`) para evitar aprovação de suítes vazias.
