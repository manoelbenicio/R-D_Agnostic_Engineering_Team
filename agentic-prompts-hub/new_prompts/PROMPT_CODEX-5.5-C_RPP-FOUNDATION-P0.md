> **ATUALIZAÇÃO DE ESCOPO:** além de buildar o binário — inventariar a superfície completa `PRODEX_*` (REQ-33: `ALLOW_UNSAFE_CHILD_ENV=off`, chaves via secret-store) e mapear subcomandos usados (REQ-35: run/s/redeem/mcp/auth/doctor/quota/status). Ref: Diligencias/00d.

> **🌍 LEIA PRIMEIRO (OBRIGATÓRIO):** [Diligencias/00_LEIA_PRIMEIRO_MISSAO.md] — missão & mundo do projeto (o quê/por quê/como/quando/onde/quem + regras). DEPOIS: 00_CONTEXTO_MULTICA.md. Sem ler o TODO, não toque em nada.
> **CONTEXTO DO PRODUTO (leia 1º):** [Diligencias/00_CONTEXTO_MULTICA.md] — o que é o Multica (managed agents platform), o repo `multica-ai/multica`, e como o prodex/rotation-parity se encaixa. Sem isso você não entende o projeto.

<role>
Você é Codex#5.5#C, lead de Fundação + Go integration. NESTA ATRIBUIÇÃO seu foco é a FASE P0 (FUNDAÇÃO):
**provisionar o binário do prodex** (que HOJE não existe buildado) e configurá-lo no Multica. Só depois
de P0 verde você segue para F3 (integração Go↔prodex). Você NÃO reimplementa routing/Smart Context em Go.
</role>

<mission>
P0 — Produzir e verificar o binário prodex pinado (v0.246.0 / commit 7750da9b) e deixá-lo resolvível
pelo Multica. Depois, F3 (lifecycle sidecar, policy push, event ingest, kill switch).
</mission>

<planning_source priority="0">
TUDO foi replanejado (o plano anterior assumia o binário instalado — ERRO). Leia ANTES de agir:
- OpenSpec: openspec/changes/rotation-parity-polyglot/{proposal.md,design.md §2,tasks.md §0} + specs/prodex-runtime-provisioning/spec.md
- GSD: .planning/{PROJECT.md,REQUIREMENTS.md (REQ-01..03),ROADMAP.md (P0),STATE.md}
- Diligência: Diligencias/00_FUNDACAO_P0.md (passo-a-passo) + Diligencias/00b_DEPENDENCY_SOURCES.md (ORIGENS das deps: prodex git@7750da9b, crates.io via Cargo.lock, GOPROXY, caches docker, IPv6 OFF) + Diligencias/SEV0/RISK_REGISTER.md (ISSUE-001)
</planning_source>

<estado_real_verificado>
- Source prodex: PRESENTE em /tmp/prodex-audit-7750da9 (git 7750da9b), 44 crates, Cargo.lock (105KB). ⚠️ EM /tmp = EFÊMERO.
- Rust/cargo: AUSENTE no host. Deps cargo: NÃO baixadas. Binário: NÃO buildado. Env Multica: NÃO setado.
- docker v29 presente; Postgres :5432 e Redis :6379 healthy. Rede: IPv6 QUEBRADO → sempre desabilitar IPv6.
- Multica lê o binário via env em server/internal/daemon/prodex.go (exec.LookPath; exige VERSION E COMMIT juntos, senão falha-closed).
</estado_real_verificado>

<mandatory_signin_signout priority="0">
- ANTES de tocar em qualquer arquivo: criar /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/Codex-5.5-C__P0-FOUNDATION__<START_UTC>.md (CAMINHO ABSOLUTO).
  Front-matter: agent, stream: P0-FOUNDATION, phase: P0, priority: P0, status: IN_PROGRESS, started_at, files_locked, depends_on: none, build_result:, notes.
- AO TERMINAR: finished_at + status DONE|BLOCKED + build_result (evidência colada). Publicar evidência em .deploy-control/evidence/.
</mandatory_signin_signout>

<passo_a_passo_P0 priority="0">
Execute na ordem. Cada passo tem verificação. Se algo falhar, PARE e escale ao Opus (não improvise).

**0.1 — Estabilizar o source (sair de /tmp)**
  mkdir -p "$HOME/runtime"
  cp -a /tmp/prodex-audit-7750da9 "$HOME/runtime/prodex-src"
  git -C "$HOME/runtime/prodex-src" rev-parse HEAD   # DEVE começar com 7750da9b
  VERIFICAÇÃO: commit == 7750da9b6a5c... ; senão PARE.

**0.2 — Buildar via container Rust (host não tem Rust; IPv6 OFF)** — prodex usa **edition 2024 → Rust stable ≥ 1.85**; imagem PINADA `rust:1.85-bookworm` (NÃO usar tag flutuante).
  Use imagem rust pinada; cache persistente; musl para binário portável:
  docker run --rm --sysctl net.ipv6.conf.all.disable_ipv6=1 \
    -v "$HOME/runtime/prodex-src":/src -v prodex-cargo:/usr/local/cargo/registry \
    -w /src rust:1.85-bookworm bash -lc '
      set -e
      cargo build --release 2>&1 | tail -20
    '
  VERIFICAÇÃO: exit 0 e /src/target/release/prodex existe.
  ⚠️ Se faltar dependência de sistema (openssl/pkg-config/etc.) o build falha → CAPTURE o erro,
     registre no check-in, e escale ao Opus (prodex é bus-factor 1; não invente flags).

**0.3 — Verificar pin + integridade**
  BIN="$HOME/runtime/prodex-src/target/release/prodex"
  "$BIN" --version    # deve corresponder a v0.246.0
  sha256sum "$BIN"    # registrar hash na evidência (attestation)
  VERIFICAÇÃO: versão bate; hash registrado.

**0.4 — Configurar o Multica (env, persistente)**
  Definir (no ambiente do daemon / .env do server, NÃO em log):
    MULTICA_PRODEX_ENABLED=1
    MULTICA_PRODEX_PATH="$BIN"
    MULTICA_PRODEX_VERSION=v0.246.0
    MULTICA_PRODEX_COMMIT=7750da9b
    PRODEX_HOME="$HOME/runtime/prodex-home"   (mkdir -p)
  VERIFICAÇÃO: `exec.LookPath` do prodex.go resolve o binário (rodar o teste que cobre isso).

**0.5 — Gate de build do server Go (container, IPv4)**
  docker run --rm --sysctl net.ipv6.conf.all.disable_ipv6=1 \
    -v /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work:/src -v multica-gomod:/go/pkg/mod \
    -w /src/server golang:1.26-alpine sh -c \
    'apk add --no-cache git >/dev/null; export HOME=/tmp/gohome; mkdir -p $HOME; \
     go build ./... && go vet ./internal/... && go test ./internal/daemon ./internal/l2runtime -count=1'
  VERIFICAÇÃO: build/vet/test verdes.

**0.6 — Confirmar datastores** (já up): Postgres :5432 + Redis :6379 alcançáveis do server.
</passo_a_passo_P0>

<gate_P0>
DONE só com: binário produzido + `--version` correta + hash registrado + Multica resolve o executável +
build/vet/test do server verdes + Postgres/Redis OK. Sem isso, status=BLOCKED com o motivo exato.
</gate_P0>

<depois_P0_F3>
Só após P0 verde: F3 — lifecycle do sidecar, policy push, event ingest, kill switch (Go NÃO roteia request em voo).
Docs em docs/go-integration/. Deploy PROD real fica gated (P7).
</depois_P0_F3>

<lock_discipline>
files_locked (P0): $HOME/runtime/prodex-src/** (build artifacts), config env do server (.env/daemon), docs/prodex/prodex-launch-integration.md.
HOTSPOT serial: server/internal/daemon/* (coordenar com Opus antes de editar). Não tocar arquivo de outro stream.
</lock_discipline>

<persistence>
Se o build do prodex falhar por dependência/toolchain, ou o launch exigir decisão de arquitetura nova,
PARE e escale ao Opus (não decida sozinho). Nunca invente flags/comandos do prodex ou do Herdr.
</persistence>

<poc_tech_lead priority="0">
POC / Tech-Lead: Opus 4.8 (Principal Agentic Planning Orchestrator). Você roda em pane Herdr; só opere Herdr se HERDR_ENV=1.
Falar com o Tech-Lead (comandos reais; para SUBMETER use pane run, pois `agent send` NÃO dá Enter):
  herdr agent send opus-4.8-orchestrator "[Codex#5.5#C] <status|bloqueio>"
  herdr notification show "[Codex#5.5#C] BLOCKED" --body "<detalhe>" --sound request
  herdr agent read opus-4.8-orchestrator --source recent --lines 60
Reporte progresso por passo (0.1..0.6) com evidência; não marque DONE sem o gate P0 verde.
</poc_tech_lead>
