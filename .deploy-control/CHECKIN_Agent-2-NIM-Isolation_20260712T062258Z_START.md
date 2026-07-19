---
agent: Agent-2-NIM-Isolation
model: glm-5.2 (High)
stream: W1-T1.2
started_at: 2026-07-12T06:22:58Z
finished_at:
status: IN_PROGRESS
authority: openspec/changes/native-runtimes-onboarding (task 1.2) + specs/agent-runtimes/spec.md
files_locked:
  - multica-auth-work/server/internal/daemon/execenv/nim_home.go
  - multica-auth-work/server/internal/daemon/execenv/nim_home_test.go
  - multica-auth-work/server/internal/daemon/rotation_detector_nim.go
  - multica-auth-work/server/internal/daemon/rotation_detector_nim_test.go
  - multica-auth-work/server/internal/rotation/detector_nim.go
  - multica-auth-work/server/internal/rotation/detector_nim_test.go
depends_on: []
build_result:
notes: |
  Task 1.2: isolamento de credencial + rotacao do NIM. NIM = NVIDIA NIM,
  runtime nativo OpenAI-compatible (https://integrate.api.nvidia.com/v1), auth =
  NVIDIA API key (Bearer / NVIDIA_API_KEY). Espelha os padroes codex/opencode/cline
  existentes (execenv/<vendor>_home.go, rotation/detector_<vendor>.go,
  daemon/rotation_detector_<vendor>.go + tests).
  Abordagem anti-colisao (Code of Conduct item 5): NAO edito arquivos compartilhados
  (config.go, agent.go, requiresCredentialIsolation em daemon.go). Meus 3 arquivos +
  3 testes sao STANDALONE e ficam verdes sem depender do wiring dos dispatchers
  (detector.go matchesVendorExhaustion switch, execenv.go Prepare/CredentialEnv).
  Todo o wiring e entregue como patch no DONE — Kiro aplica na Wave 2.
---

# CHECKIN START — Agent-2 (NIM-Isolation), Task 1.2

## Escopo
Isolamento de credencial + deteccao de exaustao/rotacao do runtime nativo NIM:
- `execenv/nim_home.go`: preparador que isola o dir de credencial NIM
  (`<accountHome>/.nim/`) num per-task `nimDataDir` (0700), espelhando
  `cline_home.go`/`antigravity_home.go`. Nao inspeciona conteudo (anti-vazamento).
- `rotation/detector_nim.go`: matcher `matchesNimExhaustion` (banner
  OpenAI-compatible 429/rate-limit/quota + reset/retry), espelhando
  `detector_cline.go`/`detector_opencode.go`.
- `daemon/rotation_detector_nim.go`: struct `NimExhaustionDetector` implementando
  `rotation.ExhaustionDetector` via o helper compartilhado `detectExhaustion`,
  espelhando `rotation_detector_cline.go`.

## Arquivos (disjuntos com Agent-1/3/4/5/6)
- server/internal/daemon/execenv/nim_home.go (+test)
- server/internal/rotation/detector_nim.go (+test)
- server/internal/daemon/rotation_detector_nim.go (+test)
Nenhum arquivo compartilhado tocado.

## Dependencias
- Nenhuma bloqueante. O backend HTTP NIM (server/pkg/agent/nim.go) e task 1.1
  (Agent-1), disjunto. O wiring do dispatcher (detector.go/execenv.go/daemon.go
  requiresCredentialIsolation) e do Kiro (Wave 2).

## Riscos
1. Padrao de banner NIM: NIM e OpenAI-compatible, entao banners sao
   OpenAI/NVIDIA-style (429/rate_limit_exceeded/quota). Padroes sao best-effort
   (doc 36 §2.1: "confirmar contra a tela real no deploy"). Sem credencial em log.
2. Layout da credencial NIM: assume `.nim/credentials.json` (ou `api_key`).
   Preparador copia o dir inteiro AS-IS (nao parseia) — compativel com qualquer
   layout que o Agent-1 escolher para nim.go.
3. Tests standalone: nao chamam `detector.Detect("nim",...)` (requer wiring do
   Kiro); cobrem via `NimExhaustionDetector.Detect` (struct, usa detectExhaustion
   compartilhado) + matcher direto. Verde sem wiring.

## Plano
1. Escrever os 3 arquivos + 3 testes espelhando cline/opencode/antigravity.
2. go build ./... + go test nos pacotes tocados (execenv, rotation, daemon).
3. Verde-em-container antes de DONE.
4. Entregar patch de wiring (requiresCredentialIsolation+nim, detector.go case nim,
   execenv.go Prepare+CredencialEnv nim) no DONE — Kiro aplica W2.

## Boot confirmado
- HERDR_ENV=1 exportado.
- git log verificado: repo vivo em main, commits recentes (610c847 docs(prompts)
  check-in per task; d59c95e plan(openspec) native runtimes).
- Prompt Agent-2 + Codigo de Conduta + spec agent-runtimes lidos.
- Toolchain: go1.26.4 em /home/dataops-lab/go-sdk/bin/go.
