# MASTER — PROD READINESS PLAN (Multica rotação + auth + observabilidade)
> Orquestrador/SME: Opus 4.8. Fonte única de verdade do caminho até PROD.
> Regra: nada de código pelo Opus; Opus prepara ambiente + coordena agentes + valida.
> Cada item tem STATUS, DONO, EVIDÊNCIA/ENTREGA e GATE. Atualizar aqui a cada avanço.
> Última atualização: 2026-07-02.

## 0. O QUE JÁ ESTÁ PROVADO (verificado pelo Opus, não confiado em tail)
- Rotação: state machine, detector reativo, proativo (banner+ledger) — unit + E2E
  Postgres real + realtime daemon vivo. Evidência: rotation_events ...001→...002
  quota_forecast_proactive (docs/project/realtime-rotation-evidence.md).
- Isolamento por conta: CODEX_HOME / XDG_DATA_HOME / HOME (execenv) — testado.
- Switch de credencial: CredentialAuthenticator é REAL (restore de arquivo por vendor:
  codex auth.json / kiro data.sqlite3 / antigravity .gemini/antigravity-cli). NÃO é
  OAuth/device-login — é restore de credencial pré-provisionada.
- Stack self-host de pé (backend+postgres) buildada da nossa cópia; /metrics ligado.
- Observability stack AUTORADA (deploy/observability): prometheus+grafana+alertmanager+
  postgres-exporter, 4 dashboards (rotation/accounts-quota/credential-health/platform-health),
  secrets populados. AINDA NÃO SUBIDO; scrape target é placeholder host.docker.internal:8081.

## 1. BLOQUEADORES DE PROD (têm de resolver antes do GA)
| # | Item | Por quê | Dono | Gate |
|---|------|---------|------|------|
| B1 | **Ciclo de vida do token** | Switch só faz RESTORE de arquivo; token EXPIRADO não é renovado → rotacionar p/ conta stale = conta morta. Prior art (subswap) tem keepalive. | Opus decide arquitetura → Codex implementa | Rotação p/ conta com token expirado é detectada e/ou renovada; teste prova. |
| B2 | **Daemon sem DATABASE_URL = rotação SILENCIOSAMENTE off** | rotationStore nil, sem erro. Achado do Codex#d. | Codex | Startup emite WARN/erro alto quando rotação configurada mas DATABASE_URL ausente. |
| B3 | **Falso-positivo do detector reativo** | "usage limit reached" dispara mesmo com cota (teto mensal ChatGPT-seat). openai/codex#23994. | Codex | Detector cruza com /status ("5h limit: N% left") antes de tratar como esgotado; teste. |
| B4 | **Provisionamento real de N contas/vendor** | staging usou auth.json clonado. PROD precisa enroll real, isolado, documentado. | Opus (runbook) + Codex (script) | Runbook + script enrolam >=2 contas reais por vendor com credencial isolada. |
| B5 | **Teste de AUTH real (pendente)** | Nunca exercitamos login/switch com CONTA REAL de vendor ponta a ponta. | Opus (ambiente) + Codex (harness) | Login/switch real de 1 vendor prova credencial válida troca e task roda na conta nova. |

## 2. HARDENING (should-have antes de PROD sério)
| # | Item | Dono | Entrega |
|---|------|------|---------|
| H1 | **Observability operativa** (prometheus+grafana up, scrape real, dashboards vivos) | Codex/infra | Stack up; Prometheus scrapeando /metrics real; dashboard rotation com dados. |
| H2 | **Expor métricas do DAEMON** (gap: rotation_total do daemon não é exposto) | Codex | Daemon sobe metrics server (METRICS_ADDR próprio) OU push; Prometheus scrapeia; rotation_total incrementa live. |
| H3 | **Cooldown-return** (conta exausta volta a available após reset da janela) | Codex | Teste prova: conta em cooldown/exhausted retorna selecionável após window reset. |
| H4 | **Concorrência do pool** (N agentes rotacionando o mesmo pool) | Codex | Stress test: lease/ref-count sem corrida; 2 contas, K agentes. |
| H5 | **Robustez (padrões subswap)**: swap manual independente de quota; snapshot/rollback atômico; cache de quota com stale fallback | Codex | Cada padrão implementado + teste. |
| H6 | **Alertas** (alertmanager) validados com dados reais (all_accounts_exhausted, sem contas) | Codex/infra | Alerta dispara em cenário de pool esgotado. |

## 3. OPERAÇÃO / DOCS (runbooks)
| # | Item | Dono | Entrega |
|---|------|------|---------|
| O1 | Runbook de deploy PROD (stack + daemon + DATABASE_URL + METRICS_ADDR) | Opus | docs/project/prod-deploy-runbook.md |
| O2 | Runbook de enrollment de contas (por vendor, credencial isolada, segredo seguro) | Opus | docs/project/account-enrollment-runbook.md |
| O3 | Runbook de observabilidade (subir stack, dashboards, alertas, o que olhar) | Opus | docs/project/observability-runbook.md |
| O4 | Segredos em repouso (arquivo 600 vs keyring/KMS) — decisão + doc | Opus | decisão registrada; subswap usa keyring. |

## 4. ESCOPO FUTURO (não bloqueia GA dos 3 vendors)
| # | Item | Estado |
|---|------|--------|
| F1 | Fase 3 vendors: Kimchi / OpenCode / Cline | Planejado (SPRINT-NEXT-vendors.md); espera layout real do Kimchi |
| F2 | Detector: banner "usage limit reached / wait until HH:MM" + janela semanal | Pesquisa feita (BACKLOG-detection.md); implementar junto de B3 |

## 5. SEQUÊNCIA DE EXECUÇÃO (waves) — ORDEM RECOMENDADA
- **WAVE A — Ambiente operativo completo (Opus + 1 infra agent):**
  A1 subir observability stack (prometheus/grafana/alertmanager/pg-exporter).
  A2 apontar scrape do Prometheus ao /metrics REAL (backend + daemon quando H2).
  A3 confirmar dashboards vivos + pg-exporter lendo o Postgres do produto.
  Gate: Grafana mostra dados; Prometheus targets UP.
- **WAVE B — Auth real + provisionamento (B4, B5, O2):** enrolar contas reais de 1 vendor
  (começar Codex), rodar login/switch real ponta a ponta, task na conta nova.
  Gate: switch real provado + runbook de enrollment.
- **WAVE C — Bloqueadores de robustez (B1, B2, B3):** token lifecycle, guard de
  DATABASE_URL, cross-check do detector. Gate: 3 testes verdes + evidência.
- **WAVE D — Hardening (H2, H3, H4, H5, H6):** métricas do daemon, cooldown-return,
  concorrência, padrões subswap, alertas. Gate: testes + dashboards refletindo.
- **WAVE E — Runbooks finais (O1, O3, O4) + aceite PROD.**
- **WAVE F — Fase 3 vendors (quando priorizado).**

## 6. DISCIPLINA (inegociável)
1. Check-in/out em .deploy-control/ antes de editar; files_locked declarado.
2. Hotspots (daemon.go, execenv.go, contract.go) = dono único; demais em arquivos novos.
3. Verde no container ANTES de DONE; Opus re-roda/valida — não confia no tail.
4. Nada inventado: string/endpoint/flag de vendor só com fonte primária (vendor/GH/issues).
5. Sem segredo em log/commit; credencial por referência; tokens mascarados.
6. Pesquisa de comportamento de vendor SEMPRE na fonte (somos early adopters).

## 7. RASTREIO (marcar aqui a cada avanço)
- [x] WAVE A (A1 A2 A3) — observability OPERATIVA (Opus, 2026-07-02). Evidência:
      prometheus :9090 (targets credential-service/postgres/prometheus = UP; prometheus
      conectado à rede multica_default, scrape backend:9090 corrigido do placeholder),
      grafana :3000 health 200 (4 dashboards provisionados), alertmanager :9093,
      postgres-exporter :9187 UP. Métricas de rotação registradas (0 séries até uma
      rotação passar pelo processo do BACKEND — vide H2, gap do daemon).
- [ ] WAVE B (B4 B5 O2)  — auth real + provisionamento
- [ ] WAVE C (B1 B2 B3)  — bloqueadores robustez
- [ ] WAVE D (H1 H2 H3 H4 H5 H6) — hardening
- [ ] WAVE E (O1 O3 O4) — runbooks + aceite
- [ ] WAVE F (F1 F2) — fase 3