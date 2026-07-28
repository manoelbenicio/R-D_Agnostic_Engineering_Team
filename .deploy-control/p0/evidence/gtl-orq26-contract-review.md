# GTL-01 / ORQ-26 — Auditoria READ-ONLY do contrato de upload e chat

- Auditor: Kiro-Opus5 (leitura e recomendação; sem poder de decisão, AA-001 §0.0)
- Dispatch: GENERAL-TL GTL-01, 2026-07-27T11:27Z
- Histórico de instrução durante a execução: uma ordem de encerramento chegou e foi em
  seguida **revogada pelo owner**; a tarefa GTL-01 seguiu no escopo original. O owner
  também comunicou a promoção de Codex56-TL (w5:pC) a GENERAL-TECH-LEAD, com autoridade
  final permanecendo exclusiva do owner humano.
- Ações executadas: somente leitura (`git show`/`git log`, grep, leitura de arquivos).
  Nenhuma edição de código, build, deploy, restart, commit, push, board, credencial,
  permissão ou rerun. Única escrita: este arquivo.

## Resultado

**BLOCK.** Os três itens do contrato final estão violados no estado atual de
`server/internal/handler/file.go`. As três correções cabem em 3 hunks de um único arquivo
server-side, sem tocar em `packages/core/api/schema.ts` e sem reverter `d10d09e0`.

## Correção de referência (importante)

`d10d09e0` (autosave `2026-07-21T21:33:16Z`) **não altera**
`server/internal/handler/file.go`. O conteúdo dele relevante para upload/chat está em
`packages/core/api/schema.ts`: introduz `ApiContractError` (linhas 19-30) e faz
`parseWithFallback` **falhar fechado**, com `throw new ApiContractError(...)` em
`packages/core/api/schema.ts:56` em vez de devolver o fallback. `handler/file.go` foi
alterado por último em `aa62401`. Portanto "não reverter `d10d09e0` inteiro" equivale a
"preservar a validação fail-closed do cliente" — e é exatamente essa validação que
transforma as duas respostas fora do caminho felizes do servidor em erro visível de UI.

## Item 1 — contextless sem entity refs deve `id=""` + `url`/`download_url` = storage link

Estado atual: `file.go:465-476`.

```go
// file.go:465-476
// No workspace context (e.g. avatar upload) — upload directly.
link, err := h.Storage.Upload(...)
...
writeJSON(w, http.StatusOK, map[string]string{
    "id":       id.String(),   // file.go:473
    "url":      link,          // file.go:474
    "filename": header.Filename,
})
```

Violações:

1. `file.go:473` devolve um UUIDv7 **não vazio** para o qual não existe linha em
   `attachments`. O cliente então persiste um link morto:
   `packages/core/hooks/use-file-upload.ts:79` faz
   `if (att.id) return attachmentDownloadPath(att.id)`, e
   `/api/attachments/<id>/download` resolve por `GetAttachmentByIDOnly`
   (`file.go:583-590`), que responde 404.
2. Falta `download_url`. `AttachmentResponseSchema`
   (`packages/core/api/schemas.ts:100-108`) exige `id`, `url`, `download_url` e
   `filename`. Com o fail-closed de `d10d09e0`, `safeParse` falha,
   `packages/core/api/schema.ts:56` lança `ApiContractError` e
   `packages/core/api/client.ts:1695-1697` rejeita a promise de `uploadFile`. Ou seja:
   todo upload sem workspace resolvido falha no cliente **mesmo com o objeto já gravado no
   storage**. Este é o sintoma ORQ-26.

Contrato exigido: `id: ""`, `url` e `download_url` iguais ao link de storage. Com `id: ""`
o `pickMarkdownLink` cai em `att.url` (`use-file-upload.ts:80`), que é o comportamento
correto para a superfície sem linha de attachment.

## Item 2 — entity refs sem workspace devem resolver+gates ou 4xx pré-upload

Estado atual: o handler ramifica somente por presença de workspace, em `file.go:376` e
`file.go:383`. Quando `workspaceID == ""`, os campos de formulário `issue_id`
(`file.go:401`), `comment_id` (`file.go:416`) e `chat_session_id` (`file.go:428`) **nunca
são lidos**, e o fluxo cai em `file.go:465-476`.

Consequências:

- `getWorkspaceMember` (`file.go:384`) e `gateChatSessionForUser` (`file.go:432`) são
  contornados por completo — não por elevação de leitura (nenhuma linha é criada, então
  `/download` não serve nada), mas o gate documentado não roda.
- Perda silenciosa de dado com HTTP 200: o objeto vai para `users/<userID>/...`
  (`file.go:376-381`), nunca é vinculado ao chat/issue/comment e fica órfão.
- É alcançável na prática: o header de workspace só é enviado quando há slug corrente —
  `packages/core/api/client.ts:300-301` e `apps/mobile/data/api.ts:1216-1217`
  (`if (slug) headers["X-Workspace-Slug"] = slug`). Resolução server-side aceita contexto,
  header de slug, `?workspace_slug`, header de ID e `?workspace_id`
  (`internal/middleware/workspace.go:68-95`); nenhum deles existe fora de uma rota de
  workspace.

Observação a favor do código atual: no ramo com workspace, membership e todos os gates de
entidade rodam **antes** de `h.Storage.Upload` (`file.go:439`), então o requisito
"4xx pré-upload" já vale ali.

## Item 3 — falha de insert deve cleanup best-effort + 500

Estado atual: `file.go:447-463`.

```go
att, err := h.Queries.CreateAttachment(r.Context(), params)   // file.go:447
if err != nil {
    slog.Error("failed to create attachment record", "error", err)
    // S3 upload succeeded but DB record failed — still return the link
    // so the file is usable. Log the error for investigation.   // file.go:449-451
} else {
    writeJSON(w, http.StatusOK, h.attachmentToResponse(att))   // file.go:453
    return
}
writeJSON(w, http.StatusOK, map[string]string{                 // file.go:457
    "id":       "",                                            // file.go:458
    "url":      link,
    "filename": header.Filename,
})
```

Violações: nenhum cleanup do objeto já gravado (o helper best-effort já existe,
`h.deleteS3Object`, usado em `file.go:938`), resposta 200 em vez de 500, e ausência de
`download_url` — que cai no mesmo `ApiContractError` do Item 1. A intenção registrada no
comentário `file.go:449-451` ("still return the link so the file is usable") foi
invalidada por `d10d09e0`: hoje o cliente não degrada, ele lança.

## Patch mínimo proposto (3 hunks, somente `server/internal/handler/file.go`)

**P1 — Item 3, substituir `file.go:447-463`.** Em erro de `CreateAttachment`:
`h.deleteS3Object(r.Context(), link)` e `writeError(w, http.StatusInternalServerError,
"failed to persist attachment")`, com `return`; remover o bloco de fall-through 200. O
`else` deixa de ser necessário.

**P2 — Item 1, substituir o corpo de `file.go:472-476`.** Responder
`{"id": "", "url": link, "download_url": link, "filename": header.Filename}`; opcional e
barato: `"markdown_url": link`. Satisfaz o trio obrigatório do schema e força
`pickMarkdownLink` para `att.url`.

**P3 — Item 2, inserir antes de `file.go:376`** (depois do parse do multipart, antes do
cálculo de `key` e de qualquer `Upload`):

```go
if workspaceID == "" {
    for _, field := range []string{"issue_id", "comment_id", "chat_session_id"} {
        if r.FormValue(field) != "" {
            writeError(w, http.StatusBadRequest,
                "workspace context is required to link an attachment")
            return
        }
    }
}
```

Falha fechada, pré-upload, sem query nova. Alternativa B (maior, não recomendada agora):
resolver o workspace a partir da linha da entidade e então rodar o caminho com gates —
exige queries by-id-only para issue, comment e chat session; hoje só existe
`GetAttachmentByIDOnly`. Só vale se algum cliente legítimo precisar enviar refs sem slug.

Nada em `packages/core` precisa mudar; `d10d09e0` permanece intacto.

## Matriz de testes

Servidor, novos em `server/internal/handler/file_test.go`:

| # | Caso | Asserção |
|---|---|---|
| S1 | contextless, sem refs | 200; `id==""`; `url==download_url==`link; nenhuma linha em `attachments` |
| S2 | contextless + `issue_id` | 400; mock de storage com **zero** chamadas de `Upload` |
| S3 | contextless + `comment_id` | 400; zero `Upload` |
| S4 | contextless + `chat_session_id` | 400; zero `Upload`; `gateChatSessionForUser` não alcançado |
| S5 | `CreateAttachment` retorna erro | 500; `Delete` chamado 1x com a key enviada; corpo não-200 |
| S6 | `CreateAttachment` erro + `Delete` erro | ainda 500 (best-effort) |
| S7 | guarda de forma de resposta | tabela sobre as 2 formas de sucesso: chaves `id`, `url`, `download_url`, `filename` sempre presentes |

Regressões que devem permanecer verdes (pinam o que P1..P3 não podem quebrar):
`TestUploadFileForeignWorkspace` (`file_test.go:138`),
`TestUploadFileResolvesWorkspaceViaSlugHeader` (`:173`),
`TestUploadFileResolvesWorkspaceViaIDHeaderStill` (`:241`),
`TestUploadFile_AttachesToChatSession` (`:280`),
`TestUploadFile_RejectsForeignChatSession` (`:353`), mais os dois uploads dentro de
`chat_test.go:60` e `chat_test.go:149`.

Cliente, unitários:

| # | Arquivo | Asserção |
|---|---|---|
| C1 | `packages/core/hooks/use-file-upload.test.ts` | `att.id===""` e `markdown_url===""` ⇒ `markdownLink===att.url` |
| C2 | `packages/core/api/client.test.ts` | resposta sem `download_url` lança `ApiContractError` (mantém o fail-closed de `d10d09e0` sob teste) |
| C3 | `packages/core/api/client.test.ts` | forma contextless `{id:"",url,download_url,filename}` parseia sem lançar |

Superfícies (todos os botões e o chat), uma asserção cada, pelo mesmo hook:

| # | Superfície | Forma esperada |
|---|---|---|
| U1 | `packages/views/chat/components/chat-input.tsx` (clipe, sessão ativa) | workspace + `chat_session_id`, link vinculado |
| U2 | `packages/views/chat/components/chat-window.tsx` (drag and drop) | idem U1 |
| U3 | `packages/views/editor/content-editor.tsx` + `editor/extensions/file-upload.ts` (paste e drop) | workspace, sem refs |
| U4 | `comment-input.tsx`, `reply-input.tsx`, `comment-card.tsx` | workspace + `issue_id`/`comment_id` |
| U5 | `modals/create-issue.tsx`, `modals/quick-create-issue.tsx` | workspace sem `issue_id` (issue ainda não existe); deve seguir o ramo com workspace |
| U6 | `modals/feedback.tsx` | contextless; é o botão que hoje lança `ApiContractError` |
| U7 | avatar/perfil web e `apps/mobile/app/(app)/[workspace]/more/settings/profile.tsx` | contextless; `url` utilizável |
| U8 | mobile `components/editor/use-file-attach.ts`, `composer/message-composer.tsx`, `chat/chat-composer.tsx`, `issue/composer-attachment-row.tsx` via `apps/mobile/data/api.ts:1200-1228` | mesmas formas |
| U9 | `e2e/chat-attachments.spec.ts` | um cenário por forma: chat vinculado, contextless e falha de insert (500 em toast, sem card fantasma) |

## Itens que exigem decisão escrita do owner (AA-001 §0.1)

| # | Item | Estado |
|---|---|---|
| G1 | Aplicar P1, P2, P3 em `handler/file.go` (refactor) | pendente |
| G2 | Adicionar S1..S7, C1..C3, U1..U9 (escrita de teste) | pendente |
| G3 | Rodar gates Go e front (não há toolchain Go neste host; instalar é §0.1) | pendente |
| G4 | Alternativa B do Item 2 (queries by-id-only + resolução de workspace) | pendente |

## Check-out

- Escopo entregue: localização exata das três violações, patch mínimo por hunk, matriz de
  testes cobrindo todos os botões e o chat, e a correção da referência `d10d09e0`.
- Não entregue por bloqueio: execução de gates (sem toolchain Go no host) e qualquer
  aplicação de patch (requer decisão escrita do owner, AA-001 §0.1).
- Estado: análise concluída, evidência gravada, reportada ao General-Tech-Lead em w5:pC.
  Aguardando redistribuição. Nenhuma colisão de escopo: nenhum arquivo de produto foi
  tocado, e o único arquivo escrito é esta evidência.
