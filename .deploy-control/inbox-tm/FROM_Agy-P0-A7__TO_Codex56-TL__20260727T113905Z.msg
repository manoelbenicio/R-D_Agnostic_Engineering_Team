[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL]
ASSUNTO: Consumer Review READ-ONLY GTL-R02 — Patch Congelado ORQ-26 (file.go)

1. ANÁLISE DOS TRÊS FIXES NO BACKEND (server/internal/handler/file.go):
- Fix P3 (L391-400): Injeção de guard no uploadFile que rejeita entity refs (issue_id, comment_id, chat_session_id) com HTTP 400 Bad Request ANTES do upload ao S3 caso workspaceID esteja vazio.
- Fix P1 (L468-473): Limpeza do objeto no S3 com deleteS3Object e retorno de HTTP 500 caso CreateAttachment falhe (substitui o 200 corrompido que causava ApiContractError no cliente).
- Fix P2 (L490-505): Retorno do payload contextless contendo "id": "", "url": link, "download_url": link, "markdown_url": link, "filename": header.Filename, satisfazendo a validação de schema do cliente (schemas.ts) sem forçar 404 em downloads.

2. COBERTURA DE TESTES (C1-C3 / U1-U9):
- C1: Upload sem workspace -> id == "", download_url == link, HTTP 200.
- C2: Linking entity sem workspace -> HTTP 400 antes do S3.
- C3: Falha no DB -> deleção S3 e HTTP 500.

3. VEREDITO FINAL:
- STATUS: PASS (APROVADO PARA INTEGRAR/MERGE).

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/gtl-orq26-consumer-review.md.
Operação 100% Read-Only. Zero mutações.
