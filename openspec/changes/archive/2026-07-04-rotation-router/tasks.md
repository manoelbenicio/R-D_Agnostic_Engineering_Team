# Tasks — Rotation Router (execução: 2 Codex + 2 GLM-5.2)

> Base: proposal.md + design.md (metodologia fundamentada em Requesty, self-hosted).
> Orquestrador/validador: Opus 4.8 (não escreve código; valida cada DONE no container).
> **LATÊNCIA = FASE 2 (sem prioridade).** Claim-de-reset headless = gated (confirmar binário).

## REGRAS (inegociáveis — valem para TODOS os streams)
0. **SIGN-IN/OUT OBRIGATÓRIO (gate duro):** ANTES de editar, criar o check-in em
   `/mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/<AGENT>__<STREAM>__<START_UTC>.md`
   (CAMINHO ABSOLUTO — nunca `.deploy-control/` relativo, que cai em multica-auth-work).
   START_UTC=`date -u +%Y%m%dT%H%M%SZ`. Front-matter: agent, stream, started_at, finished_at:,
   status: IN_PROGRESS, files_locked, depends_on, build_result:, notes. AO TERMINAR: mesmo
   arquivo com finished_at + agent + status: DONE|BLOCKED + build_result colado.
1. Hotspots (contract.go, service.go, pool.go, daemon.go) = **dono único, SERIAL**. Resto = arquivo NOVO.
2. Verde no container ANTES de DONE; Opus re-roda e valida (não confia no tail).
3. Nada inventado; sem segredo em log. Prompts seguem template best-practice (XML + example + persistence).
4. Trabalho SOMENTE em multica-auth-work/. Sem commit.

---

## WAVE 1 — 4 agentes em paralelo (arquivos NOVOS disjuntos, ZERO colisão)

### [ ] RR-POLICY  → CODEX#1   (server/internal/rotation/policy.go + policy_test.go — NOVO)
- Definir tipos próprios (NÃO tocar contract.go): `RotationPolicy{name,type,workType,items[]}`,
  `PolicyItem{vendor,accountRef,retries,weight,credentialSrc}`, enums
  `PolicyType{FALLBACK,LOAD_BALANCING,LATENCY}` e `WorkType{GENERAL,HEAVY,CHEAP,REVIEW}`.
- `func ResolvePolicy(name string) (RotationPolicy, error)` + validação (type∈enum;
  items ordenados; retries 0–10 default 1; weight só p/ LOAD_BALANCING; 0<retries<=10).
- `func (p RotationPolicy) Ordered() []PolicyItem` (ordem = prioridade do fallback).
- Determinístico, tabela-driven. Ref: design.md §2, §4.
- Teste: parse/validação de cada type, workType→policy, retries fora de range → erro, ordenação.
- Verif: `go build ./internal/rotation/... && go vet ./internal/rotation/... && go test ./internal/rotation/ -run Policy -v`

### [ ] RR-FALLBACK → CODEX#2  (server/internal/rotation/fallback.go + fallback_test.go — NOVO)
- Mecânica de retry/backoff (design.md §3): `func NextBackoff(attempt int) time.Duration`
  (500ms→1s→2s→4s, cap) + `func Jitter(d time.Duration) time.Duration` (±10%).
- `func ClassifyError(err error, httpStatus int) RetryDecision` → {RETRY, FAILOVER_NOW}:
  429/timeout/503 = RETRY; 401/400/auth = FAILOVER_NOW. REUSA detector.go read-only p/ sinais.
- `type RetryPlan{maxRetries int}` + helper que decide "retry vs próximo item vs esgotou".
- NÃO tocar service.go/pool.go — só a lógica pura, testável isolada.
- Teste: curva de backoff, jitter dentro de ±10%, classificação por status, esgotar retries.
- Verif: `... go test ./internal/rotation/ -run Fallback -v`

### [ ] RR-REGISTRY → GLM-5.2#1  (server/migrations/124_approved_accounts.up/down.sql + scripts NOVOS)
- Migration NOVA (número livre — confirmar próximo após 123): tabela `approved_accounts`
  (tenant_id, account_id, allowed bool, worktype_scope, created_at) — governança per-tenant.
  Ler 123_rotation.up.sql antes; usar mesmas convenções (uuid, gen_random_uuid). NÃO inventar coluna.
- Script `scripts/staging/registry_query.sql`: view/consulta de "contas aprovadas por tenant + estado".
- Estender (arquivo NOVO, não editar enroll existente) um helper de registry se necessário.
- Verif: aplicar migration up no multica-postgres-1, provar tabela criada + down reverte;
  `docker exec -i multica-postgres-1 psql -U multica -d multica -c "\dt approved_accounts"`.

### [ ] RR-OBSERV → GLM-5.2#2  (scripts/observability/* + deploy/observability/* — NOVO/edição isolada)
- Estender o gen_dashboards.py (NOVO componente YAML) OU novo dashboard: adicionar dimensões
  task/workspace/repo/agent-version e o painel **SAVINGS** ($ economizado vs metered estimado).
  Savings = Σ(tokens × preço-metered-hipotético do modelo equivalente) — a fórmula fica
  documentada; usar SOMENTE métricas reais do catálogo (credential_metrics.go). Ref: design.md §8.
- Doc `docs/project/savings-kpi.md`: como o Savings é calculado + o que prova (o moat em número).
- Verif: gerar dashboard, JSON válido, Grafana carrega (curl /api/search, senha mascarada).
- NÃO tocar Go de produto. Só observability config/scripts/docs.

> COLISÃO: RR-POLICY (rotation/policy.go), RR-FALLBACK (rotation/fallback.go),
> RR-REGISTRY (migrations/scripts), RR-OBSERV (observability) = 4 áreas disjuntas. OK paralelo.

---

## WAVE 2 — depois da Wave 1 (dependências)

### [ ] RR-LOADBALANCE → CODEX#1  (server/internal/rotation/loadbalance.go + test — NOVO)
- Depende de RR-POLICY (usa RotationPolicy/PolicyItem). Arquivo NOVO.
- Seleção ponderada + **por saúde de janela** (design.md §5): escolhe conta pra equilibrar
  janelas 5h (esgotar juntas = throughput agregado). Consistência: `func PickConsistent(
  policy, traceID string) PolicyItem` via hashing determinístico (xxhash) sobre traceID/agentID.
- Teste: distribuição converge aos pesos; mesmo traceID→mesma conta; saúde-de-janela equilibra.
- Verif: `... go test ./internal/rotation/ -run LoadBalance -v`

### [ ] RR-INTEGRATE → CODEX#2  (SERIAL — lock EXCLUSIVO service.go/pool.go)
- Depende de RR-POLICY + RR-FALLBACK (+ RR-LOADBALANCE se pronto). Fiar a seleção
  policy-driven no `SelectNext`/rotação: usar ResolvePolicy(workType) → ordenar/escolher via
  fallback (retry/backoff/classify) e, se LOAD_BALANCING, via PickConsistent.
  Aditivo: sem policy configurada → comportamento priority-drain atual (AS-IS preservado).
- LOCK EXCLUSIVO service.go/pool.go. Nenhum outro stream toca esses no mesmo período.
- Teste: policy fallback rotaciona na ordem; retryable respeitado; sem-policy = AS-IS.
- Verif: gate canônico do pacote rotation + não-regressão (E2E Postgres se tocar store).

### [ ] RR-PROACTIVE-RESET → GLM-5.2#1 (após RR-REGISTRY)  (arquivo NOVO; claim = GATED)
- Integrar leitura proativa (reusa proactive.go/warnbanner.go/usage.go/probe_codex.go) à
  decisão de policy: se conta ativa aproxima do limite → rotacionar proativo ANTES da falha.
- **CLAIM-DE-RESET = BLOCKED até confirmar mecanismo headless** (`/usage` é TUI-only). NÃO
  inventar comando. Deixar interface `ResetClaimer` com impl no-op + nota "CONFIRMAR CONTRA
  BINÁRIO / app-server RPC". Documentar em notes.
- Verif: teste com fake; proativo dispara antes do esgotamento; claim = no-op documentado.

---

## FASE 2 (DEPOIS — sem prioridade agora)
### [ ] RR-LATENCY  — seleção multi-vendor por TTFT (design.md §5, §12). Exige instrumentar
  dispatch (execenv/daemon) p/ medir TTFT por vendor. NÃO iniciar agora.
### [ ] RR-CLAIM-RESET-REAL — implementar claim real quando o mecanismo headless for confirmado.

---

## MATRIZ DE DESPACHO (Wave 1 — todos já livres)
| Agente   | Stream        | Arquivo(s) locked (NOVO)                    |
|----------|---------------|---------------------------------------------|
| CODEX#1  | RR-POLICY     | rotation/policy.go (+test)                  |
| CODEX#2  | RR-FALLBACK   | rotation/fallback.go (+test)                |
| GLM-5.2#1| RR-REGISTRY   | migrations/124_* + scripts/staging/registry_query.sql |
| GLM-5.2#2| RR-OBSERV     | scripts/observability/* + docs/savings-kpi.md |

Wave 2: CODEX#1→RR-LOADBALANCE, CODEX#2→RR-INTEGRATE(serial), GLM-5.2#1→RR-PROACTIVE-RESET.

## VALIDAÇÃO (Opus, a cada DONE)
1. Sign-out completo (agent+started+finished). 2. Só arquivos locked tocados. 3. Re-rodar a
   verificação no container. 4. Confirmar AS-IS preservado onde aplicável. Só então marcar [x].
