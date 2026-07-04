# P0 — Diligência: Fundação (runtime prodex + ambiente)

> **BLOQUEIA TUDO.** Sem P0 verde, nenhuma outra fase avança. Corrige o furo raiz do plano anterior
> (que assumia o binário instalado).

## Objetivo
Produzir e verificar o **binário do prodex** pinado e deixar o ambiente dev/deploy pronto, de modo que
o Multica consiga resolver e lançar o runtime.

## REQ-IDs
REQ-01 (provisionar binário) · REQ-02 (ambiente) · REQ-03 (migrations reversíveis).

## Pré-requisitos (verificados)
- Source prodex presente: `/tmp/prodex-audit-7750da9` no commit `7750da9b` ✅
- docker v29 ✅ · Postgres :5432 ✅ · Redis :6379 ✅
- **Faltando:** toolchain Rust/cargo; binário buildado; source em local estável (está em `/tmp`, efêmero).

## Passos

### 0.1 — Estabilizar o source
- Mover `/tmp/prodex-audit-7750da9` → local persistente (ex.: `~/runtime/prodex-src`).
- Confirmar: `git -C <dst> rev-parse HEAD` = `7750da9b6a5c...`.

### 0.2 — Instalar Rust
- Instalar toolchain compatível com o workspace (via `rustup` ou imagem docker `rust:*`).
- Alternativa reprodutível (sem poluir host): build em container `rust:<ver>-alpine/bookworm`.

### 0.3 — Build release
```
cargo build --release            # no diretório do source prodex
# saída esperada: target/release/prodex
```

### 0.4 — Verificar pin + integridade
- Confirmar versão `v0.246.0` e commit `7750da9b`.
- Registrar `sha256sum target/release/prodex` (attestation) em evidência.

### 0.5 — Wire no Multica (env)
- `MULTICA_PRODEX_ENABLED=1`
- `MULTICA_PRODEX_PATH=<abs>/target/release/prodex`
- `MULTICA_PRODEX_VERSION=v0.246.0`  ·  `MULTICA_PRODEX_COMMIT=7750da9b`
- `PRODEX_HOME=<home isolado do runtime>`
- (nota: `prodex.go` exige VERSION e COMMIT juntos, senão falha closed — comportamento correto)

### 0.6 — Resolver executável
- Rodar o path do `prodex.go` (`exec.LookPath`) e confirmar que resolve o binário.

### 0.7 — Datastores
- Confirmar o server (container) alcança Postgres :5432 e Redis :6379.

### 0.8 — Toolchain de build do server
- Validar o gate de container (golang:1.26-alpine) buildando `./...`.

## Verificação / evidência esperada
- `prodex --version` responde do binário pinado (log em `.deploy-control/evidence/`).
- Hash do binário registrado.
- `go build ./...` verde em container.
- Conexão Postgres/Redis OK.

## Critério de GATE (DONE)
✅ Binário pinado produzido + verificado (hash) · ✅ Multica resolve o executável · ✅ Postgres/Redis alcançáveis · ✅ build do server verde em container.

## Riscos / notas
- prodex bus-factor 1; build pode exigir deps de sistema (documentar no runbook).
- Não deixar o source em `/tmp` (efêmero) — perda em reboot.
- Sem `--auto-redeem` nesta fase (reset-claim é P9).
