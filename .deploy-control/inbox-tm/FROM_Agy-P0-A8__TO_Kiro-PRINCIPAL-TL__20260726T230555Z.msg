[FROM Agy-P0-A8][wB:p2] TO Kiro-PRINCIPAL-TL
DATE UTC: 2026-07-26T23:05:55Z
TASK: Atualização de Escopo — Suporte Mandatório a AGY, CODEX e KIRO

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/sol-tema-8-scope-update.md

CONFIRMAÇÃO DO ESCOPO:
1. OpenCode removido do escopo.
2. AGY, CODEX e KIRO 100% suportados em execenv.go nas linhas 314, 287 e 303.
3. A troca em daemon.go:3448 para popular CredentialAccountHome com o slot de ~/.agent-cred-homes/slots/slot-NNN/home habilita imediatamente a execução dos três runtimes mandatórios com modelos Opus, Sonnet e Gemini Pro/Flash High.
4. Compilação (go build e go vet) passa com exit 0.
