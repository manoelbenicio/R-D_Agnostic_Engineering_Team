# DECISIONS — Agent Brain v3 (assinável; re-abrir só com nova decisão do dono)

> Decisões irreversíveis/gates. Decisões herdados do v2.1 preservados com status.
> Padrão: `decisão · porquê · como aplicar · status`.

## Decisões v2.1 ABSORVIDAS como governança permanente

### D-007 — Isolamento de credenciais por agente (ABSORVIDA, FECHADA)
- Isolamento total de homes/credential stores entre agents; nunca copiar/importar creds
  de um agent p/ outro.
- **Why:** fronteira de segurança; menor blast radius;EVITAR clobber/overwrite.
- **How to apply:** no target Agent Brain, isto evolui para **credentialless**: cada task
  recebe **uma** chave OmniRoute e **nenhuma** credencial provider (AB-REQ-16). O isolamento
  de credenciais entre accounts é agora responsabilidade exclusiva do OmniRoute (P01).
- STATUS: absorvida em AB-REQ-16/17; não reabrir.

### D-008 — TL é delegation-only (ABSORVIDA, FECHADA)
- TL planeja/atribui/valida; nunca roda comandos de produto nem escreve código; só commita
  plano (não código).
- **Why:** causa-raiz da evidência fake 2026-07-06 (TL rodou prodex e gerou evidência "as agente").
- **How to apply:** Kiro/Opus-4.8 owns planning/adjudication; Codex#56#A owns transport,
  independent verification and execution control; Codex workers implement under disjoint locks.
- STATUS: absorvida; vigente. Prorrogação: TL também **não edita produção**.

### EVIDENCE_CONTRACT (legado) — ABSORVIDO como EVIDENCE_CONTRACT.md v3
- Proveniência exata, rejeição de fabricação (localhost/fake-upstream/identical numbers/
  sign-off forjado), logs scrubbed, kill-switch real (não descrição), check-in antes.
- STATUS: absorvido; ver EVIDENCE_CONTRACT.md deste GSD.

## Decisões v3 (este milestone)

### D-V3-01 — OmniRoute é o único hot router/credential owner
- Brain envia intent+correlação; OmniRoute possui contas, credenciais, rotation, quota,
  fallback, Smart Context, telemetria hot.
- Never: dual router, provider keys no Brain.
- STATUS: APROVADO (handover §8.1; design §3). Não reabrir.

### D-V3-02 — Brain guarda **uma** chave OmniRoute estável e limitada
- Scoped a rotas aprovadas; rotate/revoca independente de provider accounts.
- STATUS: APROVADO. Restrição dura: nunca ler/imprimir/copiar/registrar seu valor.

### D-V3-03 — Nome provisório: `Agent Brain`
- Definitivo só em G8 (debrand). Não quebrar tie(nome) com critical path.
- STATUS: APROVADO (architect response 7.2).

### D-V3-04 — Primeiro tier = 20 tarefas
- 50/100 exigem relatório de capacidade aprovado e decisão de estado.
- STATUS: APROVADO (architect response 7.3). Não ativar 50/100.

### D-V3-05 — Prodex removal e gateway-required default NÃO autorizados no escopo inicial
- Só após paridade assinada (G5) + cutover gate (G6) + zero-use.
- **End-state atualizado por D-V3-16 (2026-07-19):** Prodex NÃO é deletado. Após os gates, é
  *quiesced* para um **cold platform recovery mode default-OFF, mutuamente exclusivo e
  operator-gated** na última raia do Kanban — nunca per-request, nunca fallback automático,
  nunca hot simultâneo com OmniRoute.
- STATUS: APROVADO (handover §8.10); end-state emendado por D-V3-16.

### D-V3-06 — Rotation unit = request lógico independente
- Não avança por SSE/retry/tool/limite. Strict round-robin ≠ limite global 1-at-a-time.
- STATUS: APROVADO (architect response 2.2).

### D-V3-07 — Continuation affinity é explícita e separada
- Affinity sobrepõe rotation só p/ continuação dependente (previous_response_id, cache, tool turn).
- STATUS: APROVADO; comprovar `preserve` no G4 (evidence pendente).

### D-V3-08 — Estado single-node SQLite aceito APENAS para validação tier 20
- Arquiteto: global strict RR garantido em instância única (P23/Q18). Multi-node exige
  estado compartilhado (Redis/Postgres) + reverificação.
- Why: entrega pragmática da validação de desenvolvimento tier 20 sem re-arquitetar OmniRoute.
- How to apply: tier 50+ exige decisão single-node vs compartilhado (gate G7).
- STATUS: APROVADO p/ dev-validation 20 (com caveat); REABRE em G7 para tiers>=50.

### D-V3-09 — Claude/GLM-5.2 substitui Kiro como planning/orchestration owner deste milestone
- Decisão do dono 2026-07-17. Histórico v2.1 preservado; regra "só Kiro autora .planning/"
  fica SUPERSEDED para este milestone (documentada aqui, não apagada).
- Codex agents **não** sobrescrevem GSD v3 silenciosamente; todo plano on-disk tem task-ID.
- STATUS: HISTÓRICA / SUPERSEDED em 2026-07-18 por D-V3-13. Preservada para auditoria.

### D-V3-10 — Strangler extraction, não rewrite global
- Neutral interfaces em torno do daemon provado; rename binário/module/storage é G8.
- STATUS: APROVADO (design §1).

### D-V3-11 — Retry/fallback = pre-commit only
- Sem replay após output parcial ou tool action não-idempotente. Same-model fallback primeiro.
- STATUS: APROVADO (design §7).

### D-V3-12 — Secret source: Linux restricted secret, nunca Windows world-readable injetado
- Derivado operacionalmente da origem Windows existente **sem** expor/commitar/copiar valor.
- STATUS: APROVADO (design §5; CLE spec). Codex 4 executa via worker (TL não toca secrets).

### D-V3-13 — Kiro/Opus-4.8 + Codex#56#A assumem liderança dividida e não-concorrente
- Decisão explícita do dono em 2026-07-18 após encerramento do pane Claude `w3:p5`.
- Kiro/Opus-4.8 (`w3:p3`) = planning/adjudication/architecture priority.
- Codex#56#A (`w3:p1`) = operational co-lead, Herdr transport, independent verification,
  authoritative state/docs and execution control.
- Apenas uma decisão arquitetural por vez; workers recebem prompts depois de reconciliação.
- STATUS: APROVADO / ATIVO.

### D-V3-14 — Sem production canary/soak; manter acceptance de desenvolvimento
- O sistema ainda não está em produção. O dono removeu production canary/soak como gate imediato.
- Não remover testes: G3/G4 mantêm integração, credential-isolation, protocol/failure, rollback,
  no-dual-router e bounded-capacity validation em ambiente de desenvolvimento.
- Nenhum resultado de desenvolvimento autoriza produção, cutover, Prodex removal ou tiers 50/100.
- STATUS: APROVADO / ATIVO.

### D-V3-15 — Kanban existente permanece parked até o boundary credentialless
- MUL-2..MUL-25 já existem; não criar duplicatas. MUL-11/12/15 conflitam com OmniRoute como
  owner exclusivo de credentials/accounts/rotation e devem ser reescopados ou superseded.
- O daemon Multica atual não deve despachar Codex antes de G3 credentialless wiring + isolation smoke.
- STATUS: APROVADO / ATIVO.

### D-V3-16 — Prodex retido como cold platform recovery mode (default-OFF, mutuamente exclusivo)
- Decisão final do dono 2026-07-19. Agent Brain + OmniRoute são o caminho primário e terminam
  primeiro. Prodex NÃO é deletado e NÃO é o target router: é **retido** apenas como um
  **cold platform recovery mode default-OFF, mutuamente exclusivo, operator-gated**, na última
  raia do Kanban. Nunca per-request, nunca fallback automático, nunca hot simultâneo com OmniRoute.
- **How to apply:** emenda OpenSpec task **10.4** (retain-as-recovery, não delete) e **AB-REQ-37**
  (removal→retention); re-escopo do change `persist-prodex-runtime-integration` para
  cold-recovery-only (`MULTICA_PRODEX_REQUIRED` default `0`); wire à máquina de estados de
  recovery (AB-REQ-41) no ponto único de runtime-authority select (`health.go:177-184`);
  `REMOVAL_REGISTER` muda Prodex/L2 de RETIRE para RETAIN-AS-RECOVERY.
- **Resolve** o `persist-prodex-vs-omniroute-reconciliation-audit.md` como variante da Opção C
  (continuidade mínima reversível) e levanta o PROGRAM HOLD, mantendo 0/16 e sem mudar produto.
- STATUS: APROVADO / ATIVO. Não reabrir.

### D-V3-17 — G4-OBS: stop-gate de observabilidade E2E antes de capacidade/cutover
- Decisão final do dono 2026-07-19. Observabilidade metadata-only end-to-end é **gate
  bloqueante** antes de qualquer tier de capacidade (§9) ou cutover (§10).
- Escopo: 8 hops (ingress API → DB queue → daemon → CLI → OmniRoute/provider → terminal
  persistence → WS/UI delivery → trace assembly), correlação metadata-only, redação estrutural,
  trace sintético contínuo, aceitação leak-clean, dashboards/alerts.
- **How to apply:** nova capability OpenSpec `end-to-end-observability`; seção de tasks
  **8-OBS OBS-1..OBS-11** com evidence `EV-OBS-01..11`; gate G4-OBS no ROADMAP entre G4 e G5;
  novos requisitos AB-REQ-39/40; `EVIDENCE_CONTRACT`/`EVIDENCE_INDEX` definem a evidência OBS.
- STATUS: APROVADO / ATIVO.

### D-V3-18 — Topologia de 8 lanes com prova de zero-overlap
- Decisão final do dono 2026-07-19. Expande a topologia de 4 para **8 lanes** (W1–W8) com
  ownership de arquivos disjunto par-a-par, integrador único dos hotspots (W1) e biblioteca de
  correlação (W5) chamada — nunca co-editada.
- **How to apply:** `design.md §11` e `FILE_OWNERSHIP.md` expandidos; `DISPATCH_QUEUE.md` recebe
  as entradas da Wave B; Codex#56#A roda a checagem de interseção de globs e registra
  `EV-ZERO-OVERLAP`; novo requisito AB-REQ-41 (recovery-mode) e risco R29 (contenção de merge).
- STATUS: APROVADO / ATIVO.

### D-V3-20 — Testes funcionais do Main Brain podem RODAR antes do G4-OBS; D-V3-17 permanece stop-gate de ACEITAÇÃO
- Direção do dono 2026-07-19 (owner + Codex56-Principal-TL recomendação; Kiro-TL endossa). Esclarece o
  ESCOPO de D-V3-17 — não o enfraquece. D-V3-17 sempre bloqueou a **validade de alegações de
  capacidade/cutover**, nunca a **execução de testes de correção funcional**.
- **EXECUÇÃO — permitida agora, em paralelo, ANTES do G4-OBS:** testes de Main Brain
  **funcional / unit / integração / protocolo / falha / retry / afinidade / cancelamento / paridade**
  podem rodar imediatamente e concorrentemente com as lanes OBS. Rodá-los NÃO consome, satisfaz ou
  contorna o gate G4-OBS.
- **ACEITAÇÃO — permanece bloqueada por D-V3-17 (OBS-1..OBS-11 aceitos):** nenhuma **certificação de
  capacidade tier-20 / task 9.1**, **cutover**, **produção** ou **alegação de readiness** pode ser
  afirmada até o G4-OBS ser aceito. D-V3-17 permanece inalterada como gate de ACEITAÇÃO.
- **Firewall de evidência (guarda obrigatória):** execuções de teste funcional não-observadas são
  **evidência de correção apenas**. NÃO podem ser rotuladas, agregadas ou promovidas a evidência de
  **capacidade / cutover / readiness**. Verdes funcionais não substituem exporters de observabilidade
  aceitos + sinal E2E ao vivo e não podem ser citados como tal.
- **Escopo:** D-V3-20 governa a trilha de teste funcional do Main Brain; NÃO altera o gating de dispatch
  das lanes OBS (ainda pendente de aceitação EV-ZERO-OVERLAP / council) e NÃO toca D-V3-19.
- **ETA (registrada, com ressalva):** excluindo observabilidade, **24–48h nominal, 72h conservador**
  (estimativa do Principal) para COMPLETAR a trilha funcional. NÃO é ETA para aceitação G4-OBS,
  certificação de capacidade, cutover ou readiness — esses permanecem gated e sem cronograma.
- STATUS: **ACEITA** (owner-dirigida + Principal-recomendada + Kiro-TL-endossada; quorum de council para
  decisão CRÍTICA). Holds preservados: 9.1/capacidade/PD-08/keys/Prodex/cutover/produção/tier 50/100.

## Decisões resolvidas pelo dono

- **PD-01 — RESOLVIDA (2026-07-17):** preservar e incorporar a worktree existente de
  `persist-prodex-runtime-integration` como baseline auditável de segurança de credenciais.
  É proibido resetar, reverter, shelvar ou descartar esse trabalho. Codex1 recebe ownership
  exclusivo dos hotspots durante o freeze G1; antes de qualquer integração, deve auditar o
  diff, executar os testes do change e reconciliar cada uma das 16 tasks. O change deixa de
  ser classificado como `SUPERSEDED` enquanto suas garantias contra overwrite/fallback não
  estiverem implementadas e verificadas.

## Decisões PENDENTES do dono (registradas para gates posteriores)

- **PD-02** — Digest da imagem OmniRoute fixado (substituir `:latest`) — pré-cutover total.
- **PD-03** — Waiver de produto+segurança para SC01–SC10 caso OmniRoute não prove (ou plan de impl).
- **PD-04** — Decisão de estado single-node vs compartilhado (gate G7, pré-tier 50/100).
- **PD-05** — Nome definitivo do produto (gate G8).
- **PD-06** — Dono do produto assina §7.5 (e arquiteto, segurança se houver waivers).
- **PD-07** — Prefixo parcial de chave anteriormente exposto no architect response (§0) foi
  redigido em 2026-07-18; rotação continua pendente do dono. Treat historical exposure as real.

## Adjudicação de monitor (2026-07-17) — isolamento de credenciais + STOP

- **Monitor autoritativo (dono):** as 4 reauth Linux geraram 4 homes privadas distintas;
  global Linux `~/.codex/auth.json` AUSENTE; slots inalterados; arquivos auth mode 0600;
  sem inode compartilhado; sem duplicata cross-login. Guarda de isolamento válida p/ Linux.
- **Legacy Windows credential:** `C:\Users\dataops-lab\.codex\auth.json` é uma credencial
  **legada preexistente** (v9fs), criada/escrita em **2026-07-15T17:12:59Z**, **inalterada**
  durante a reauth corrente. É uma **exposição real** (Windows ACL concede `CodexSandboxUsers`
  ReadAndExecute) — **não** é overwrite event, **não** é falso-positivo de modo MSYS.
- **Decisões vigentes (dono):**
  1. **Manter STOP** em mutações de credencial / qualquer dispatch que toque auth.
  2. **TL NÃO lê, move, apaga, nem reescreve** o arquivo Windows legado.
  3. Tasks de **documentação/contrato** G1 em andamento podem **terminar** — são no-secret/read-only
     w.r.t. auth. (Confirmado: brain/** + artifacts produzidos sem tocar auth.)
  4. **Exige autorização do dono** para restringir/quarentena/remover o arquivo Windows legado
     e rotacionar a conta associada → vira **PD-08** (abaixo).
- **PD-08 (NOVO, PENDENTE do dono)** — Restringir/quarentena/remover a credencial Windows legada
  `C:\Users\dataops-lab\.codex\auth.json` (exposição ACL `CodexSandboxUsers` ReadAndExecute) e
  rotacionar a conta associada. Gate de segurança; requer autorização explícita do dono
  (ação de segredo, fora do mandato delegation-only do TL). Blocker minor p/ hardening; não
  bloqueia freeze G1 (que é no-secret).
