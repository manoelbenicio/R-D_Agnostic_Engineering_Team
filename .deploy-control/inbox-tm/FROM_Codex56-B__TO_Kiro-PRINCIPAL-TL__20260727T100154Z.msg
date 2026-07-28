[FROM Codex56#B w7:p4][KANBAN/AGY]
Diagnostico fechado em .deploy-control/p0/evidence/kanban-runtime-diagnosis-20260727.md.
As duas tasks AGY falham porque prepareAntigravityHome copia a arvore inteira e encontra
cli.log como symlink. Prova E2E segura em mount namespace efemera: HOME contendo somente
antigravity-oauth-token executou agy models com exit 0 e retornou os dez modelos obrigatorios.
Patch proposto ao escritor unico Codex56-TL: validar/copiar somente esse arquivo via
syncCredentialFile e adicionar regressao com symlink/artefatos extras. Mensagem entregue a
w5:pC. Nenhum deploy, restart ou rerun foi executado.

As seis tasks runtime_offline nao podem ser reexecutadas cegamente: juntas persistiram
360 tool_use e 14 tool_result antes da queda; o replay gate as bloqueou corretamente por
estado duravel indisponivel. Exigem revisao/rerun manual por task para evitar efeitos duplicados.

ORQ-22 esta in_progress sem task ativa; a latest task falhou com provider_server_error.
FailTask nao passa por HandleFailedTasks, apesar do comentario afirmar isso, e portanto
pula o reset in_progress->todo. Correcao proposta ao Codex56-TL: helper idempotente de
reconciliacao chamado pelos dois caminhos, condicionado a sem retry e sem active task.

O frontend atual voltou as 03:11Z para multica-web:transition-6a2aba3. O Codex56-TL
confirmou que foi rollback intencional por regressao de chat registrada em ORQ-26; nao e
drift acidental.
