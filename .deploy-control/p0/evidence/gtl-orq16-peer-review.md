# Peer Review READ-ONLY GTL-23 — Design de Elegibilidade Dinâmica de Slots (ORQ-16)

## 1. Identificação do Reviewer e Escopo
- **Reviewer**: Antigravity w8:p2
- **Documento Revisado**: `.deploy-control/p0/evidence/gtl-orq16-agy-slot-health.md` (Seção `CORREÇÃO GTL-16R`)
- **Modo**: READ-ONLY. Zero leitura de segredo em memória, zero alterações de código/DB/ambiente.

## 2. Validação Factual contra o Código-Fonte do Server/Daemon

### A. Validação de Subdiretórios e Paths Reais de Credenciais
- **Kiro**: Validado subdiretório `<slot_path>/xdg-data` e arquivo de validação `kiro-cli/data.sqlite3` (`kiro_home.go` L11). Corrigida a premissa incorreta de `tokens.json`.
- **AGY (Antigravity)**: Validado subdiretório `<slot_path>/home` e arquivo de validação `.gemini/antigravity-cli/antigravity-oauth-token` (`antigravity_home.go` L12).
- **Codex**: Validado subdiretório `<slot_path>/codex` (`codex_home.go`).
- **Cline**: Validado subdiretório `<slot_path>/cline` (`cline_home.go`).

### B. Garantia de Zero Leitura de Segredos
- Confirmado que a verificação de saúde **NÃO faz parse nem carrega o conteúdo dos tokens para a memória do processo** (L23-25).
- A decisão de elegibilidade é 100% baseada nos metadados físicos do inode no filesystem.

### C. Proteção contra Symlinks (`O_NOFOLLOW` / `os.Lstat`)
- Validado o algoritmo de verificação (L28-64):
  * Utiliza `os.Lstat` (que não segue symlinks) e rejeita explicitamente `ModeSymlink` (L48).
  * Exige arquivo regular (`info.Mode().IsRegular()`, L54), descartando sockets, FIFOs e diretórios.
  * Rejeita arquivos zerados (`info.Size() <= 0`, L60).

### D. Distinção entre `Auth Present` vs `Session Alive`
- Validada a separação conceitual (L67-76): `Auth Present` afere a existência física da credencial no slot; `Session Alive` afere a vivacidade do processo/daemon via heartbeat.

### E. Político de Caching, Revogação e Fail-Closed
- Cache síncrono com TTL de 60s protegido por `sync.RWMutex` (L78-89).
- Revogação ou exclusão física do arquivo resulta em `AuthPresent = false` no próximo tick.
- Falha fechada: slots inválidos ou inseguros são automaticamente omitidos do mapa de dispatch sem pânico no daemon.

## 3. Desenho da Suíte de Testes Unitários
- Testes unitários especificados (L91-96):
  1. `TestKiroDataSqlite3Presence`: Valida presença de `data.sqlite3` com `size > 0`.
  2. `TestRejectSymlinks`: Garante que symlinks para credenciais falham fechados.
  3. `TestRejectZeroBytePlaceholder`: Garante que arquivos de 0 bytes são rejeitados.
  4. `TestTTLCacheExpiration`: Garante expiração correta do cache em 60s.

## 4. Veredito Final
- **STATUS: PASS (APROVADO SEM RESSALVAS)**
- **Linhas de Evidência Citadas**: `gtl-orq16-agy-slot-health.md` L12-22 (paths reais), L28-64 (`Lstat`/`O_NOFOLLOW`), L67-76 (`Auth Present` vs `Session Alive`) e L78-89 (Cache/Fail-Closed).
