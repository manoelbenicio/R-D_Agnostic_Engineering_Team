# CHECK-IN DONE — Codex-56-A — Task 1.2

- UTC: 2026-07-14T20:45:07Z
- Slot: A (`~/.codex-slotA`)
- Change OpenSpec: `native-runtimes-onboarding`
- Status: DONE

## Entrega

- NIM usa `accounts.home_dir/NVIDIA_API_KEY` como fonte de credencial por conta.
- A credencial e copiada para `nim-home/NVIDIA_API_KEY` como arquivo regular `0600`; nunca e exposta por symlink.
- Ausencia ou conteudo vazio falha fechado antes do backend HTTP iniciar.
- `CredentialEnv("nim")` le somente a copia da task e injeta `NVIDIA_API_KEY`; `custom_env` nao pode sobrescreve-la.
- Rotacao NIM reconhece o erro nativo `NIM API returned 429`, rate/quota/resource/credit exhaustion com retry/reset e ignora 503/high-traffic.
- Parser compartilhado aceita `Reset at` e `Resets at`.
- Checkbox OpenSpec 1.2 marcado concluido.
- Nenhum arquivo de frontend editado; nenhum segredo registrado nos testes/logs.

## Evidencia verde em container

Imagem: `golang:1.26-alpine`.

- `go test ./internal/daemon/execenv ./internal/rotation ./internal/daemon -count=1`
  - `ok .../internal/daemon/execenv 0.360s`
  - `ok .../internal/rotation 0.018s`
  - `ok .../internal/daemon 15.621s`
- `git diff --check`: PASS.
