[FROM Codex56#B w7:p4][ORQ-12 ACCOUNT SNAPSHOT AUDIT][2026-07-27T11:27:53Z]
BLOQUEANTE: GetTaskAccountID por join da assignment atual corrompe historia se houver rotacao entre start e usage.
accounts/assignments migration123:36-42 guarda so estado atual por agent; account usado deve ser snapshot imutavel em agent_task_queue no claim atomico.
ClaimAgentTask agent.sql:268-304 e o ponto correto: validar tenant/vendor/approved/status, gravar credential_account_id e enviar so UUID pseudonimo ao daemon.
Daemon mapeia UUID a slot allowlisted local; backend nunca envia path/identidade/segredo; usage herda snapshot server-side.
Unificar migration 127 de ORQ12+13: task queue + raw/hourly/dirty com account_id e thinking_level; UNIQUE NULLS NOT DISTINCT; rollups e queries carregam ambos.
ON DELETE RESTRICT; historico fica account NULL; app rollback preserva schema, sem down destrutivo.
Gates incluem rotacao pos-start sem reatribuicao, mismatch daemon rejeitado, cross-tenant fail-closed e duas tasks/contas distintas.
Evidencia: .deploy-control/p0/evidence/orq-9-issues-exit-criteria.md:517-651
Somente leitura; nenhum schema, codigo, teste, task ou live alterado.
