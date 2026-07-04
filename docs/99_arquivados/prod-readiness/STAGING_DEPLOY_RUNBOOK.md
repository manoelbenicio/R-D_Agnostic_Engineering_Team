# STAGING DEPLOY + REALTIME TEST — Runbook & Agent Orchestration (Fase 2)

> Orquestrador/SME: Opus 4.8. Objetivo: subir a stack Multica NESTE lab de staging,
> a partir da NOSSA cópia (multica-auth-work, com as mudanças de auth/rotação embutidas),
> e provar rotação antecipada em tempo real. O Opus NÃO escreve código — prepara o
> ambiente e coordena os 3 agentes. Fonte da verdade de progresso: este arquivo + STATUS.md.

## FATO DE AMBIENTE (confirmado neste lab, 2026-07-01)
- Docker 29.4.3, daemon UP. Imagem golang:1.26-alpine presente. Gates A/B verdes.
- Postgres 17 de teste (multica-staging-pg :55432) subiu e o E2E de rotação PASSOU
  contra DB real. (Será RETIRADO em favor do Postgres da stack self-host — ver abaixo.)
- Artefatos de deploy presentes na nossa cópia:
  * docker-compose.selfhost.yml  → postgres(pgvector/pgvector:pg17)+backend+frontend
  * docker-compose.selfhost.build.yml → BUILD backend/web a partir DESTE checkout
  * Dockerfile (backend, inclui daemon+rotação+auth), Dockerfile.web, .env.example
- A OUTRA stack Multica está no OUTRO servidor; aqui subimos a NOSSA para staging real.

## TOPOLOGIA-ALVO (staging local)
- postgres (pgvector:pg17) — DB real, migrations aplicadas pelo backend no boot.
- backend multica-backend:dev — BUILDADO do nosso Dockerfile → contém daemon + rotação.
- frontend multica-web:dev — opcional para o teste de rotação (só p/ UI/login).
- daemon — roda dentro/junto do backend e é quem executa a CLI (Codex) e dispara rotação.

## PRÉ-REQUISITOS QUE O OPUS GARANTE ANTES DE DISPARAR AGENTES (readiness)
1. `.env` de staging derivado de `.env.example` com:
   - APP_ENV != production (staging), JWT_SECRET trocado, MULTICA_DEV_VERIFICATION_CODE
     setado (login sem e-mail real), portas livres (5432/8080/3000 ou remapear).
   - Sem segredo real commitado; `.env` fica só no lab.
2. Build das imagens dev: `docker compose -f docker-compose.selfhost.yml -f docker-compose.selfhost.build.yml build`.
3. Stack up: `... up -d`; provar `postgres healthy`, backend `/health` 200, migrations
   aplicadas (incl. 123_rotation → accounts/credentials/assignments/rotation_events).
4. Retirar o multica-staging-pg ad-hoc (evitar confusão de 2 Postgres).
> Estes 4 itens são do Opus (ambiente). Só depois disso os agentes entram.

## SEQUÊNCIA (waves)
- WAVE 0 (Opus, ambiente): pré-requisitos 1–4 acima. Gate: stack de pé + migrations ok.
- WAVE 1 (Agente A, sozinho): SEED do pool no Postgres da stack.
- WAVE 2 (Agentes B + C, paralelos): B dirige a rotação realtime; C observa métricas/log.

---

## AGENTE A — SEED DO POOL (arquivo novo; sem tocar Go de produção)
Objetivo: inserir ≥2 contas Codex no schema real da migration 123 para o pool ter o que
rotacionar. Entregar script idempotente + query de verificação.
- Escreve: scripts/staging/seed_rotation_pool.sql (novo) + README curto de uso.
- Usa o schema REAL (ler server/migrations/123_rotation.up.sql antes; NÃO inventar coluna).
- Popular: 2 rows em `accounts` (vendor=codex, prioridades distintas, status=available,
  tokens_per_win/ tokens_used realistas) + `credentials` por conta (por REFERÊNCIA, nunca
  segredo em claro) apontando para 2 dirs de credencial isolados de simulação.
- Verificação (colar no check-out): SELECT que prove 2 contas selecionáveis para
  SelectNext(codex). Rodar via: docker exec -i <pg> psql -U multica -d multica < seed.sql
- Locks: apenas scripts/staging/*. NÃO editar rotation/*, daemon/*, execenv/*, migrations/*.
- Regra de ouro: se faltar coluna/constraint no schema real → PARAR e marcar BLOCKED.

## AGENTE B — HARNESS DE ROTAÇÃO REALTIME (teste de staging; arquivo novo)
Objetivo: provar, contra Postgres real + pool seedado, que o sinal proativo dispara a
rotação fim-a-fim (reusa maybeProactiveRotateOnText → rotateTaskWithReason(ReasonQuotaProactive)).
- Escreve: server/internal/daemon/staging_rotation_smoke_test.go (novo, build tag
  `//go:build staging` p/ não entrar no gate normal) OU um harness em scripts/staging/.
- Cenário: injeta o texto real de banner Codex ("less than 10% of your 5h limit left")
  no caminho do daemon com rotationService real (store Postgres) e pool de 2 contas;
  asserta: rotaciona da conta#1→#2 UMA vez, task segue sem interrupção, rotation_events
  registra a linha com reason=quota_forecast_proactive.
- Verificação: rodar o teste taggeado contra DATABASE_URL da stack; colar tail verde.
- Locks: apenas o arquivo de teste novo. NÃO editar daemon.go/rotation/* de produção.
- Depende de: WAVE 0 (stack) + Agente A (pool). Se pool vazio → BLOCKED.

## AGENTE C — VERIFICAÇÃO DE OBSERVABILIDADE (read-only + doc)
Objetivo: confirmar que a rotação é observável em tempo real (o que o operador vê).
- Verifica, durante/depois do run do B: métrica ObserveRotation(provider,reason,result,secs),
  SetAllAccountsExhausted, e a linha de log "rotation: proactive quota signal detected".
- Entrega: docs/project/observability-rotation-staging.md com os nomes/labels EXATOS das
  métricas e as queries/greps para acompanhar rotações no dashboard (Prometheus METRICS_ADDR).
- Locks: somente o doc novo + leitura de código/métricas. NÃO editar código de produção.
- Depende de: run do Agente B (para observar sinais reais).

## DEFINIÇÃO DE PRONTO (staging realtime)
- Stack self-host de pé a partir da nossa cópia; migrations (incl. 123) aplicadas.
- Pool com ≥2 contas Codex (Agente A) verificado por query.
- Rotação proativa disparada e observada em Postgres real (Agente B), rotation_events com
  a linha proativa; task sem interrupção.
- Observabilidade documentada (Agente C): operador consegue ver rotações em tempo real.
- E2E de rotação original re-rodado sem regressão.

## RISCOS / GUARDA (SME)
- 2 Postgres no lab = confusão → retirar o ad-hoc antes da WAVE 1.
- Portas 5432/8080/3000: se ocupadas, remapear no .env (não forçar bind 0.0.0.0).
- Credencial de simulação: dirs isolados fake; NUNCA segredo real no seed/log.
- Rotação realtime aqui é STAGING (contas de simulação), sem blast radius de PROD.
- Nenhum agente toca daemon.go/rotation/*/execenv/* de produção — só arquivos novos.