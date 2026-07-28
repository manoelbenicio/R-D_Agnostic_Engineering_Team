[FROM Codex56#B w7:p4][ORQ-18 PATCH AUDIT][2026-07-27T11:15:01Z]
READ-ONLY concluido; nenhum codigo/teste/build/deploy/browser/issue alterado.
VEREDITO: direcao correta e recuperavel, mas aceite ainda incompleto.
PASS: runtime-list.tsx:442-491 deixa Trash2 sempre visivel para autorizado e reutiliza DeleteRuntimeDialog.
PASS: guarda UI :555-566 continua owner/admin/runtime-owner; backend repete em runtime.go:558-568/738-746.
PASS: blobs-base dos 2 arquivos sao identicos em 6a2aba355 e HEAD 0cb8aeb; sem conflito textual com clean-base ORQ-26.
GAP: teste novo :132-145 prova somente botao + dialogo abriu; nao testa confirm/sucesso/erro/close/refetch.
GAP: mutations.ts:7-14 invalida lista, mas nao ha teste de invalidateQueries nem de linha removida imediatamente.
GAP: permissao e passada pronta ao teste; RuntimeList duplica canDeleteRuntime central e pending custom nao e coberto.
Evidencia: .deploy-control/p0/evidence/orq-9-issues-exit-criteria.md:222-311. Integrar somente apos ORQ-26 e browser autenticado.
