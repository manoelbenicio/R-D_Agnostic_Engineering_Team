# AGENTIC PLAN + PROMPT — W-PROACTIVE-INT (Fase 2, Wave 4)

**Orquestrador/SME/Arquiteto:** Opus 4.8 (planejamento, validação final independente).
**Executor (job mais pesado):** CODEX (5.5 high-thinking).
**Natureza:** SERIAL — hotspot `daemon.go`. Lock EXCLUSIVO. Nenhum outro stream
toca `daemon.go`/`daemon_test.go` enquanto este estiver IN_PROGRESS.

## Decisões de arquitetura (já tomadas pelo Opus — não reabrir)
1. **Reusar** `rotationService.OnExhaustion(..., ReasonQuotaProactive, ...)`. ZERO
   mudança em `contract.go` (o `ReasonQuotaProactive` já existe). NÃO editar contract.go.
2. **v1 = dois caminhos proativos que já têm código verde**, sem probe injetado:
   - **Codex (banner passivo):** inspecionar o texto que já passa por `MessageText`
     no loop do daemon com `WarningDetector`/`UsageDetector`. Sem injeção de comando.
   - **Ledger (todos vendors):** `ProactiveDetector.ShouldRotate` (proactive.go, já
     existe) dispara ENTRE tasks lendo o ledger. Sem probe.
   - Probe ativo `/usage` (Kiro/Antigravity) = **fora de escopo v1** (stream futuro).

---

## 1. PAPEL E MODO
Você é o CODEX, engenheiro Go sênior 5.5 high-thinking. Feature de CRITICIDADE
MÁXIMA (corporativa): rotação ANTECIPADA zero-interrupção. O parser já existe e
está verde (usage.go/warnbanner.go). A rotação REATIVA já está fiada no daemon
(`rotateTaskOnExhaustion`, ~linha 3845). Falta o gatilho PROATIVO. Rigor MÁXIMO.
Opus valida você no container assim que entregar.

## 2. REGRA DE OURO
- SOMENTE em /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/. Nunca o source original.
- NÃO FAÇA COMMIT (sem git add/commit/push/branch).
- NÃO edite: contract.go, detector.go, proactive.go, service.go, pool.go,
  store_pg.go, usage.go, warnbanner.go, auth_authenticator.go, execenv.go.
  Use os tipos/métodos existentes. NÃO altere contract.go.
- Lock EXCLUSIVO de daemon.go/daemon_test.go — você é o único a tocá-los.
- Preservar comportamento AS-IS: se rotação desligada (rotationService==nil) ou
  sem conta disponível, NADA muda no fluxo atual. Feature aditiva, não regressiva.
- Se faltar contrato/ponto de fiação, PARE e marque BLOCKED.

## 3. CHECK-IN (antes de editar) — LOCAL E FORMATO EXATOS
- Local: Automonous_Agentic/.deploy-control/ (BOARD PRINCIPAL na raiz).
- Nome: CODEX__W-PROACTIVE-INT__<START_UTC>.md  (START_UTC via: date -u +%Y%m%dT%H%M%SZ)
- Conteúdo:
  agent: CODEX
  stream: W-PROACTIVE-INT
  started_at: <UTC ISO 8601>
  finished_at:
  status: IN_PROGRESS
  files_locked:
    - server/internal/daemon/daemon.go
    - server/internal/daemon/daemon_test.go
  depends_on: [W-ROT-contract, W-DETECT, W-ROTATE, W-PGSTORE, W-USAGE, W-WARNBANNER]
  build_result:
  notes:

## 4. TAREFA — fiar rotação PROATIVA no daemon (aditiva, AS-IS preservado)
Contexto de código (leia antes): `daemon.go` já tem `rotationService`,
`rotationDetector`, `credentialMetrics`, e `rotateTaskOnExhaustion` (reativo).
O loop de sessão trata `case agent.MessageText:` (~4124) acumulando `pendingText`.

Adicione (novos campos + funções no daemon; sem tocar arquivos proibidos):
- Um `*rotation.WarningDetector` e `*rotation.UsageDetector` no struct Daemon,
  inicializados em `New()` (NewWarningDetector() / NewUsageDetector(0)).
- **Caminho A (banner passivo, Codex):** ao processar `MessageText`, além de
  acumular, chamar um novo método `d.maybeProactiveRotateOnText(ctx, task, provider, msg.Content, taskLog)`:
  * Só age se `d.rotationService != nil && d.warningDetector != nil`.
  * Roda WarningDetector.DetectWarning(provider, text) E/OU UsageDetector.Detect.
  * Se `approaching==true` (ou algum UsageSample.Approaching), dispara UMA vez por
    task (guardado por um flag atômico) `rotationService.OnExhaustion(..., ReasonQuotaProactive, now)`.
  * Mesma disciplina de erro/métrica que `rotateTaskOnExhaustion`:
    ErrNoAccountAvailable → preserva comportamento atual; erro → warn + segue;
    ok → métrica ObserveRotation(provider, ReasonQuotaProactive, "ok", secs).
  * NUNCA logar segredo/email/screen bruto (use truncateLog e não logue token).
- **Caminho B (ledger, entre tasks):** antes de iniciar uma task (ou logo após
  claim), se `rotationService != nil`, ler a conta atual via store, rodar
  `rotation.NewProactiveDetector(0).ShouldRotate(acc, now)`; se Exhausted →
  `OnExhaustion(..., ReasonQuotaProactive, now)` ANTES de rodar, para já começar
  na conta nova. Preserva AS-IS se sem conta/erro.
- Idempotência: no máximo uma rotação proativa por task; se já rotacionou reativo,
  não duplicar. Sem segredo em log.

## 5. TESTES (daemon_test.go) — nível corporativo, fakes (sem Postgres real)
Use fakes/stubs das interfaces do contrato (rotation.RotationService,
rotation.Store) — NÃO dependa de Postgres nestes testes de unidade.
- Banner de pré-aviso Codex em MessageText ("less than 10% of your 5h limit left")
  → dispara OnExhaustion com ReasonQuotaProactive EXATAMENTE uma vez.
- Texto normal → não dispara.
- rotationService==nil → nunca dispara (AS-IS preservado).
- ErrNoAccountAvailable → sem panic, fluxo segue como antes.
- Ledger: conta com TokensUsed>=95% de TokensPerWin antes da task → rotaciona
  antes de rodar; conta abaixo → não.
- Idempotência: banner repetido na mesma task → só 1 rotação.

## 6. VERIFICAÇÃO (antes de DONE)
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine \
  sh -c "go build ./... && go vet ./internal/daemon/... && go test ./internal/daemon/ -run 'Proactive|Rotat' -v"
DONE só com build (./... inteiro) + vet + test verdes. Cole o tail no build_result.
Rode também um teste de NÃO-REGRESSÃO do daemon existente (o pacote inteiro:
go test ./internal/daemon/) para provar que nada quebrou.

## 7. RESUMO DO QUE ESPERO
ANTES: check-in no board principal, started_at UTC, lock exclusivo de daemon.go.
DURANTE: só daemon.go + daemon_test.go; sem commit; sem tocar arquivos proibidos;
sem alterar contract.go; feature ADITIVA (AS-IS intacto quando rotação off/sem conta).
DEPOIS: verde no container (build ./... + vet + test + pacote daemon inteiro),
check-out com finished_at + DONE + build_result colado. Sem segredo/email em log.
BLOCKED se faltar ponto de fiação.

## Validação final (Opus, ao receber DONE)
1. Reler o diff de daemon.go (aditivo, AS-IS preservado, sem segredo em log).
2. Re-rodar no container: build ./... + vet + `go test ./internal/daemon/` inteiro
   (não confiar no tail). 3. Confirmar idempotência e o gate rotationService==nil.
4. Rodar o E2E de rotação (Postgres real) de novo para garantir não-regressão do
   caminho reativo. Só então marcar Wave 4 como aceita.
