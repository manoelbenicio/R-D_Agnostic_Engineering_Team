[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T11:37:27Z
TASK: GTL-29 Descoberta READ-ONLY de Ambiente & Preflight ORQ-26 (file_test.go / gtl-orq26)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-orq26-gate-preflight.md

RESUMO DA DESCOBERTA PREFLIGHT:
1. Versão Go: go1.26.1 linux/amd64 em /home/ec2-user/goroot/go/bin/go. Zero pacotes necessários.
2. Access ao Worktree: /home/ec2-user/workspace/worktrees/gtl-orq26 acessível com permissão drwxrwxr-x ec2-user:ec2-user.
3. Dependência DB: file_test.go possui guard 'if testPool == nil { t.Skip("test database not available") }' na L623. Pula DB se ausente; mocks em memória (mockStorage) rodam offline.
4. Isolamento: TMPDIR/GOCACHE/GOTMPDIR configurados em $HOME/.private-tmp (0700). Zero poluição de /tmp.
5. Comandos Alvo Preflight: (1) go vet ./internal/handler/... (2) go test -v -run "TestUploadFile|TestDownloadAttachment" ./internal/handler/...

MODO SOMENTE LEITURA MANTIDO. ZERO EXECUÇÃO DE BUILD/TEST E ZERO CÓPIA DE FONTES/SEGREDO.
