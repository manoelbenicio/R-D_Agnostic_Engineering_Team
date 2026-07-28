# Peer Review READ-ONLY GTL-R02 — Consumidores e Contrato ORQ-26 (file.go)

## 1. Identificação do Reviewer e Escopo
- **Reviewer**: Antigravity w8:p2
- **Documento / Worktree Revisado**: `/home/ec2-user/workspace/worktrees/gtl-orq26` (`server/internal/handler/file.go`)
- **Foco da Revisão**: Consumidores (Web / Mobile / CLI \`uploadFile\`), segurança contra \`ApiContractError\`, vazamentos S3 e retrocompatibilidade.
- **Modo**: READ-ONLY. Zero alterações de código, banco de dados ou ambiente de produção.

## 2. Análise Técnica dos Três Ajustes sob a Ótica do Consumidor

### A. Correção P3 — Entity Refs sem Workspace ID (\`file.go:391-400\`)
- **Código**:
  \`\`\`go
  if workspaceID == "" {
      for _, field := range []string{"issue_id", "comment_id", "chat_session_id"} {
          if r.FormValue(field) != "" {
              writeError(w, http.StatusBadRequest, "workspace context is required to link an attachment")
              return
          }
      }
  }
  \`\`\`
- **Impacto no Consumidor**: Evita a falha onde o cliente envia \`issue_id\` sem o cabeçalho \`X-Workspace-ID\`. Anteriormente o backend ignorava o vinculo, subia o arquivo sem criar a linha de attachment e devolvia 200 com ID vazio. Agora rejeita sumariamente com **HTTP 400 Bad Request ANTES** de gravar no S3.

### B. Correção P1 — Fail-Closed em Falha de Persistência no Banco (\`file.go:468-473\`)
- **Código**:
  \`\`\`go
  if err != nil {
      slog.Error("failed to create attachment record", "error", err)
      h.deleteS3Object(r.Context(), link)
      writeError(w, http.StatusInternalServerError, "failed to persist attachment")
      return
  }
  \`\`\`
- **Impacto no Consumidor**: Substitui a degradação silenciosa (que retornava 200 com ID vazio e gerava \`ApiContractError\` no cliente Zod/TypeScript) por **HTTP 500 Internal Server Error** explícito com limpeza best-effort do objeto no S3, prevenindo arquivos órfãos.

### C. Correção P2 — Formato de Resposta Contextless / Avatar (\`file.go:490-505\`)
- **Código**:
  \`\`\`go
  writeJSON(w, http.StatusOK, map[string]string{
      "id":           "",
      "url":          link,
      "download_url": link,
      "markdown_url": link,
      "filename":     header.Filename,
  })
  \`\`\`
- **Impacto no Consumidor**:
  * Passa no schema do cliente (\`schemas.ts\`) que exige \`id\`, \`url\`, \`download_url\` e \`filename\`, eliminando o \`ApiContractError\` no throw da linha 56.
  * \`"id": ""\` força o cliente web/mobile a usar o link direto (\`url\`/\`download_url\`) em vez de tentar chamar \`/api/attachments/{id}/download\` (que daria 404 por falta de linha no banco).

## 3. Matriz de Cobertura de Testes Recomendada (C1-C3 / U1-U9)
- **C1 (Upload Contextless / Avatar)**: Testar POST sem workspace_id -> verificar \`id == ""\`, \`download_url == link\`, \`markdown_url == link\` e HTTP 200.
- **C2 (Entity Refs Sem Workspace)**: Testar POST com \`issue_id\` mas sem \`X-Workspace-ID\` -> verificar HTTP 400 e zero chamadas ao S3.
- **C3 (DB Error Cleanup)**: Simular erro no \`CreateAttachment\` -> verificar exclusão do objeto S3 e HTTP 500.
- **U1-U9 (Testes de Manipulação e Header)**: Validar integridade dos parsers em \`file_test.go\`.

## 4. Veredito Final
- **STATUS: PASS (APROVADO PARA MERGE / REBASE POR ESCRITOR ÚNICO)**
- **Linhas de Evidência Citadas**: \`file.go:391-400\` (P3 / 400 pre-flight), \`file.go:468-473\` (P1 / S3 cleanup & 500), \`file.go:490-505\` (P2 / schema fix).
