# Protocolo de Check-in/Check-out — Deploy paralelo (auth isolation)

Controle em disco para múltiplos agentes trabalharem em paralelo **sem colisão**.
Diretório: `Automonous_Agentic/.deploy-control/`.

## Regra de ouro
Um arquivo-fonte só pode estar "checked-out" por **um** agente por vez. Antes de
editar QUALQUER arquivo, o agente:
1. Lê todos os `*.md` ativos em `.deploy-control/` (status=IN_PROGRESS).
2. Confere se algum arquivo que ele vai tocar já está em `files_locked` de outro.
3. Se houver conflito → **NÃO edita**; espera ou pega outra stream.
4. Se livre → cria seu arquivo de check-in ANTES de editar.

## Nome do arquivo de controle
```
<AGENTE>__<STREAM>__<START_UTC>.md
```
Ex.: `CODEX-1__W-KIRO__20260701T190500Z.md`
- `<AGENTE>`: nome do agente (CODEX-1, CODEX-2, GEMINI-31-PRO, GLM-52, KIRO-ORq).
- `<START_UTC>`: timestamp UTC de início (ISO 8601 compacto).

## Conteúdo (front-matter)
```
agent: CODEX-1
stream: W-KIRO
started_at: 2026-07-01T19:05:00Z
finished_at:            # preenchido no check-out
status: IN_PROGRESS     # IN_PROGRESS | DONE | BLOCKED
files_locked:
  - server/internal/daemon/execenv/kiro_home.go
  - server/internal/daemon/execenv/kiro_home_test.go
depends_on: [W-INT-contract]
build_result:           # preenchido no check-out (green/red + resumo)
notes:
```

## Check-out (ao terminar)
O mesmo arquivo é atualizado: `finished_at`, `status: DONE`, `build_result` (verde
no container `golang:1.26-alpine`), e lista de arquivos alterados. Se `BLOCKED`,
descrever o bloqueio em `notes` para o orquestrador redistribuir.

## Verificação obrigatória por agente (antes do check-out DONE)
```
docker run --rm -v "$PWD":/src -w /src/server golang:1.26-alpine \
  sh -c "go build ./... && go test ./internal/daemon/execenv/..."
```
Só marca DONE com build+test verdes nos pacotes que tocou.

## Contrato compartilhado (freeze antes do fan-out)
Arquivos-hotspot (`execenv/execenv.go`, `daemon/daemon.go`) têm **dono único**
(stream W-INT). Ninguém mais os edita. Os demais agentes programam contra o
**contrato** publicado por W-INT (assinaturas de helper + nome do campo em
`Environment`), em arquivos NOVOS por vendor.
