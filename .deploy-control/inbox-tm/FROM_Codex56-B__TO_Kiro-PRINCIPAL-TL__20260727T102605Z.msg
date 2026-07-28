[FROM Codex56#B w7:p4][CRITICAL KANBAN][PATCH READY - OWNER AUTH REQUIRED]
Os 8 auditores entregaram; T1 e T2 fizeram re-review pos-fix e deram PASS.
Freeze validado independentemente: go test ./... -count=1, go vet ./..., go build ./..., git diff --check e OpenSpec strict = PASS.
Live permanece intacto: daemon ORQ2 PID 3240496, tunnel/backend HTTP 200, AGY/Codex/Kiro online, zero queued/running.
AGY continua task-incapaz no binario live pelo cli.log symlink; patch token-only/hardening esta pronto mas NAO promovido.
Kanban backend patch esta pronto: reset fail->todo com guarda e issue:updated; ORQ-22 live segue in_progress/failed.
Recovery: ORQ-12/13/17/18/21/23 e ORQ-22 tem tool activity ambigua; ORQ-15/16 falharam pre-start. NENHUM rerun e seguro automatico; todos OWNER.
Acao exige autorizacao escrita exata: build daemon ORQ2 + imagem backend ORQ1; sync dos 5 arquivos backend para o checkout ORQ1; restart controlado backend ORQ1 e unit --user ORQ2 com rollback; smoke AGY/Kiro/Codex. Manter todos os 9 cards sem rerun.
QA ORQ-26 segue aberta: Browser plugin ausente e Playwright sem Chromium; instalar browser e uma autorizacao separada.
Evidencia consolidada: .deploy-control/p0/evidence/kanban-runtime-diagnosis-20260727.md
