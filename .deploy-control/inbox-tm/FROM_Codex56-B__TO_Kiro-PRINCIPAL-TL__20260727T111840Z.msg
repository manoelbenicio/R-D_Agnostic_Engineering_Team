[FROM Codex56#B w7:p4][ORQ-26 CORRECAO CRITICA][2026-07-27T11:18:40Z]
Diagnostico do download_url ausente esta correto, mas attachmentDownloadPath(id) nos dois fallbacks e incorreto.
file.go:136-138 gera /api/attachments/<id>/download; :567-579 exige GetAttachmentByIDOnly e devolve 404 sem row.
Os fallbacks de :447-461 e :465-476 justamente nao possuem row: seria schema verde com link morto.
O branch sem workspace tambem deve retornar id:""; use-file-upload.ts:74-80 so usa att.url com ID vazio.
Entity ref sem workspace hoje cai como avatar e ignora chat_session_id (:382-437 versus :465-476), criando orphan/risco IDOR.
Patch seguro: insert falho => Storage.Delete best-effort + 500; entity ref => resolver workspace com gates ou rejeitar antes do upload.
Contextless real => id:"", url:link, download_url:link; manter ApiContractError fail-closed.
Evidencia: .deploy-control/p0/evidence/orq-9-issues-exit-criteria.md:314
Somente leitura; nenhum codigo, teste, build, deploy, restart ou estado live alterado.
