[FROM Codex56#B w7:p4][ORQ-23 DEEP AUDIT][2026-07-27T11:24:04Z]
Worktree ORQ-23 NAO deve ser reaplicado: antigravity_home.go:39-49 ainda copia a arvore com cli.log symlink; kiro_home.go:63-101 antecede hardening 0600/O_NOFOLLOW.
Live green confirmado: daemon+tunnel active, NRestarts=0, binary 88ca4f..., assignments v1 0600 com 8 afinidades.
F2.3 segue aberto: AGY tem afinidades em slots 145/146, mas nao ha correlacao task->slot->resultado/custo.
Fase 3 ausente: TaskUsageEntry types.go:161-169 e payload handler:2053-2062 nao carregam account/tier; SQL conflita so task/provider/model.
AGY antigravity.go:156-165 envia Usage vazio; Kiro extrai ACP, mas pode bucket unknown.
Discovery daemon.go:2075-2079 para no primeiro HOME; 11 modelos nao prova paridade dos 4 slots elegiveis.
Gate 4.5: nenhum rollback T2 de um comando no ORQ2; backups pre-token-only reintroduzem AGY task-incapaz.
Evidencia: .deploy-control/p0/evidence/orq-9-issues-exit-criteria.md:399-515
Somente leitura; nenhum build, teste, task, restart, rollback ou estado live alterado.
