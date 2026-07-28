# ORQ-26 - root cause do ApiContractError no uploadFile do chat

Agente: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T11:06Z
Modo: ANALISE READ-ONLY. Nao editei codigo, nao editei arquivo do repo alem deste, nao compilei,
nao reiniciei nada, nao toquei em `/tmp/daemon.env.bak` nem em `.env`. O patch e do Codex56-TL.

## VEREDITO EM UMA LINHA

O campo divergente e **`download_url`**. O cliente exige `download_url: z.string()` obrigatorio;
o servidor **omite `download_url`** em DUAS rotas de fallback do `UploadFile`, que devolvem um
`map[string]string` com apenas `id`, `url` e `filename`. Antes de `d10d09e0` isso degradava em
silencio; depois de `d10d09e0` o `parseWithFallback` **lanca**, e a falha aparece como
`ApiContractError` em `schema.ts:56`.

## 1. O QUE MUDOU EM d10d09e0 (por que so agora quebra)

`d10d09e0` = `autosave(pre-shutdown): ... 2026-07-21T21:33:16Z`, 40 linhas alteradas em
`packages/core/api/schema.ts`. Diff literal (linhas removidas/adicionadas):
```diff
-  fallback: T,
- * returning the parsed value on success or `fallback` on failure.
- * but never throw — the UI layer must keep rendering. This is the boundary
-  return fallback;
+export class ApiContractError extends Error {
+    this.name = "ApiContractError";
+  throw new ApiContractError(opts.endpoint, result.error.issues);
```
Estado atual, `packages/core/api/schema.ts:39-56` (literal):
```ts
export function parseWithFallback<T>(
  data: unknown,
  schema: ZodType,
  _legacyFallback: T,
  opts: ParseOptions,
): T {
  const result = schema.safeParse(data);
  if (result.success) return result.data as T;
  schemaLogger.warn(
    `API response failed schema validation: ${opts.endpoint}`,
    {
      endpoint: opts.endpoint,
      issues: result.error.issues,
    },
  );
  throw new ApiContractError(opts.endpoint, result.error.issues);
}
```
`schema.ts:56` e exatamente o `throw new ApiContractError(...)`. O terceiro parametro passou a ser
`_legacyFallback`, deliberadamente ignorado (comentario em `schema.ts:31-37`). Ou seja: o contrato
nao mudou, o TRATAMENTO mudou de degradar para falhar fechado. A divergencia de campo ja existia.

## 2. LADO CLIENTE - o contrato exigido

Chamada, `packages/core/api/client.ts:1693-1697` (literal):
```ts
    const raw = (await res.json()) as unknown;
    return parseWithFallback(raw, AttachmentResponseSchema, EMPTY_ATTACHMENT, {
      endpoint: "POST /api/upload-file",
    });
```
Schema, `packages/core/api/schemas.ts:100-108` (literal):
```ts
export const AttachmentResponseSchema = z.object({
  id: z.string(),
  url: z.string(),
  download_url: z.string(),
  markdown_url: z.string().optional().default(""),
  filename: z.string(),
  chat_session_id: z.string().nullable().optional(),
  chat_message_id: z.string().nullable().optional(),
}).loose();
```
Obrigatorios: `id`, `url`, `download_url`, `filename`. `markdown_url` e lenient com default `""`
(comentario em `schemas.ts:93-99` explica o predecessor MUL-3192). `.loose()` aceita campos extras,
portanto o problema **nunca** e campo a mais: e campo obrigatorio a menos.

Envio, `packages/core/api/client.ts:1665-1685` (literal, recortado):
```ts
  async uploadFile(
    file: File,
    opts?: { issueId?: string; commentId?: string; chatSessionId?: string },
  ): Promise<Attachment> {
    const formData = new FormData();
    formData.append("file", file);
    if (opts?.issueId) formData.append("issue_id", opts.issueId);
    if (opts?.commentId) formData.append("comment_id", opts.commentId);
    if (opts?.chatSessionId) formData.append("chat_session_id", opts.chatSessionId);
    ...
    const res = await fetch(`${this.baseUrl}/api/upload-file`, {
      method: "POST",
      headers: this.authHeaders(),
      body: formData,
      credentials: "include",
    });
```
Chamador do chat: `packages/core/hooks/use-file-upload.ts:97` -> `await api.uploadFile(file, {...})`.

## 3. LADO SERVIDOR - as tres saidas de `POST /api/upload-file`

Rota: `server/cmd/server/router.go:567` -> `r.Post("/api/upload-file", h.UploadFile)`.
Handler: `server/internal/handler/file.go:315` -> `func (h *Handler) UploadFile(...)`.

### 3.1 Saida CORRETA (unica que cumpre o contrato) - `file.go:453`
```go
		} else {
			writeJSON(w, http.StatusOK, h.attachmentToResponse(att))
			return
		}
```
`AttachmentResponse` declara o campo, `file.go:56-67` (literal, recortado):
```go
type AttachmentResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspace_id"`
	...
	Filename      string  `json:"filename"`
	URL           string  `json:"url"`
	DownloadURL   string  `json:"download_url"`
```
e o preenche em `file.go:99-115` (literal, recortado):
```go
func (h *Handler) attachmentToResponse(a db.Attachment) AttachmentResponse {
	id := uuidToString(a.ID)
	resp := AttachmentResponse{
		ID:           id,
		...
		URL:          a.Url,
		DownloadURL:  attachmentDownloadPath(id),
		MarkdownURL:  h.buildMarkdownURL(a, id),
```

### 3.2 SAIDA QUEBRADA A - insert no banco falhou, `file.go:446-461` (literal)
```go
		att, err := h.Queries.CreateAttachment(r.Context(), params)
		if err != nil {
			slog.Error("failed to create attachment record", "error", err)
			// S3 upload succeeded but DB record failed — still return the link
			// so the file is usable. Log the error for investigation.
		} else {
			writeJSON(w, http.StatusOK, h.attachmentToResponse(att))
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"id":       "",
			"url":      link,
			"filename": header.Filename,
		})
		return
	}
```
Viola o contrato em DOIS pontos: **`download_url` ausente** (obrigatorio) e `id` devolvido como
string vazia. O comentario do proprio codigo declara a intencao de degradar ("still return the link
so the file is usable"), intencao que `d10d09e0` tornou impossivel no cliente.

### 3.3 SAIDA QUEBRADA B - sem contexto de workspace, `file.go:465-476` (literal)
```go
	// No workspace context (e.g. avatar upload) — upload directly.
	link, err := h.Storage.Upload(r.Context(), key, data, contentType, header.Filename)
	if err != nil {
		slog.Error("file upload failed", "error", err)
		writeError(w, http.StatusInternalServerError, "upload failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"id":       id.String(),
		"url":      link,
		"filename": header.Filename,
	})
}
```
Tambem **sem `download_url`**. O gatilho e `workspaceID := h.resolveWorkspaceID(r)` vazio
(`file.go` no inicio do handler; `handler.go:445-447` delega para
`middleware.ResolveWorkspaceIDFromRequest`). O header vem de
`packages/core/api/client.ts:300-301` (literal):
```ts
    const slug = getCurrentSlug();
    if (slug) headers["X-Workspace-Slug"] = slug;
```
Se `getCurrentSlug()` devolver vazio no momento do upload do chat, o servidor cai em 3.3 e o cliente
lanca `ApiContractError`.

## 4. RESUMO DA DIVERGENCIA

| campo | cliente `schemas.ts:100-108` | servidor 3.1 (ok) | servidor 3.2 | servidor 3.3 |
|---|---|---|---|---|
| `id` | `z.string()` obrigatorio | `att.ID` | **`""`** | `id.String()` |
| `url` | `z.string()` obrigatorio | `a.Url` | `link` | `link` |
| **`download_url`** | **`z.string()` obrigatorio** | `attachmentDownloadPath(id)` | **AUSENTE** | **AUSENTE** |
| `markdown_url` | opcional, default `""` | `buildMarkdownURL` | ausente (ok) | ausente (ok) |
| `filename` | `z.string()` obrigatorio | `a.Filename` | `header.Filename` | `header.Filename` |

O erro do Zod nas duas rotas quebradas e `invalid_type` em `download_url`, `expected string,
received undefined`. Como `parseWithFallback` agora lanca (`schema.ts:56`), o resultado e
`ApiContractError` com `endpoint: "POST /api/upload-file"`.

## 5. POR QUE ISSO EXPLICA "86 de 88 em core api/chat"

O contrato do caminho feliz (3.1) e correto e passa; as duas rotas de fallback (3.2 e 3.3) sao as
unicas que divergem. Isso e coerente com uma falha isolada e nao com quebra generalizada de schema.

## 6. RECOMENDACAO PARA O ESCRITOR UNICO (Codex56-TL) - NAO APLIQUEI

Ordem de preferencia, aditiva primeiro:
1. **Fazer as duas rotas de fallback devolverem o contrato completo**, incluindo `download_url`
   (`attachmentDownloadPath(id)` ja existe e nao depende do registro no banco). Corrige 3.2 e 3.3
   sem afrouxar o contrato do cliente. Em 3.2 tambem parar de devolver `id: ""`.
2. Alternativa inferior: tornar `download_url` opcional no schema. Rejeito: `schemas.ts:88-91`
   documenta que os dois campos abertos em nova aba precisam ser string, senao o app faria
   `window.open(undefined)`.
3. Investigar separadamente por que 3.2 ou 3.3 e alcancado no chat (insert falhando, ou
   `getCurrentSlug()` vazio). Isso muda a frequencia, nao o defeito de contrato.

## 7. NAO-AFIRMACOES
- Nao editei nenhum arquivo de codigo; gravei apenas este arquivo de evidencia.
- Nao compilei, nao rodei teste, nao reiniciei nada, nao toquei `/tmp/daemon.env.bak` nem `.env`.
- Nao reproduzi a falha em runtime: a atribuicao de causa vem da leitura dos dois lados do contrato,
  nao de um request capturado. Nao sei qual das duas rotas (3.2 ou 3.3) o bundle live cf8017e3
  esta atingindo; ambas violam o contrato do mesmo modo.
- Nao inspecionei o bundle `cf8017e3` nem o estado do rollback: fora da minha tarefa.
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.
