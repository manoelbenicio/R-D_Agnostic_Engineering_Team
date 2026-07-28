# Plano de Cutover e Canário — Descoberta Definitiva de Modelos (p8-cutover)

- **Pacote:** PACOTE 8 (`p8-cutover`)
- **Autor:** Agy-P0-A8 (wB:p2)
- **Modo:** PREPARAR (Sem alterações no código, binário ou daemon nesta etapa)
- **Data UTC:** 2026-07-26T22:45:09Z

---

## 1. Objetivo Operacional
Transicionar o mecanismo de descoberta de modelos do daemon Multica do estado provisório atual (credenciais copiadas no HOME global do ORQ1) para o estado definitivo via gateway OmniRoute (`http://100.118.244.61:20128/v1/models`), garantindo **zero janela de dropdown vazio**, isolamento total de segredos e plano de rollback determinístico.

---

## 2. Ordem Sequencial das Etapas, Critérios de Sucesso e Pontos de Rollback

### Etapa 1: Validação Pré-Flight do Gateway e Binário (Readiness)
- **Ação:** Compilar e validar o novo binário `multica-auth-gw` contendo a integração dos Pacotes 1–7 (OmniRoute client, cache e mapeamento dos 6 runtimes). Verificar conectividade com `/v1/models` via `/etc/agent-brain/secrets/omniroute-inference-key`.
- **Critério de Sucesso:** `go vet` e `go test` com exit 0; resposta HTTP 200 do OmniRoute com 327+ modelos.
- **Ponto de Rollback R1:** Se a API do gateway responder 40x/50x ou o teste do binário falhar, **ABORTAR** antes de alterar qualquer processo ou arquivo.

### Etapa 2: Subida Canário com Descoberta Dupla / Primary-Fallback (Sem Downtime no Dropdown)
- **Ação:** Iniciar o novo binário no ORQ1 operando com OmniRoute como fonte primária de descoberta e catálogo estático/fallback secundário.
- **Critério de Sucesso:** O endpoint `GET /api/models` do daemon do Multica popula o dropdown na UI para todos os 6 runtimes (*antigravity, claude, cline, codex, kiro, opencode*) via OmniRoute sem depender de CLIs locais.
- **Ponto de Rollback R2:** Se o dropdown de qualquer runtime ficar vazio ou falhar, alternar a flag de configuração do daemon para fallback estático sem reiniciar ou alterar arquivos de credencial.

### Etapa 3: Restauração de Backup e Limpeza do HOME Global (Zero-Trust)
- **Ação:** Reverter credenciais temporárias no HOME global do ORQ1 restaurando o backup de segurança `~/cred-bak-20260726` e removendo quaisquer arquivos residuais de segredo expostos em `~/.codex`, `~/.config/kiro` ou `~/.antigravity`.
- **Critério de Sucesso:** Auditoria no sistema de arquivos confirma ausência de segredos estáticos no HOME global; o daemon continua populando 100% dos modelos via OmniRoute.
- **Ponto de Rollback R3:** Se a remoção de resíduos afetar a operação de algum runtime, restaurar o snapshot de `~/cred-bak-20260726` imediatamente e reativar a ponte do canário.

### Etapa 4: Auditoria Final de Estabilidade e Encerramento
- **Ação:** Verificar os heartbeats dos 6 runtimes no backend do Multica, confirmar estabilidade dos 327 modelos no dropdown e emitir o registro de verificação.
- **Critério de Sucesso:** 6 runtimes online, 0 segredos estáticos em HOME global, 0 falhas no dropdown durante a transição.

---

## 3. Matriz de Reversão / Rollback Rápido

| Etapa | Gatilho de Falha | Ação Imparável de Rollback | Tempo Estimado |
| :--- | :--- | :--- | :--- |
| **R1 (Pré-Flight)** | Erro HTTP no Gateway ou falha em `go test` | Manter daemon PID 3244391 inalterado | 0s (Instantâneo) |
| **R2 (Canário)** | Dropdown de modelos fica vazio na UI | Reverter o binário do daemon para a versão com fallback estático | < 5s |
| **R3 (Limpeza)** | Runtime perde conectividade ao limpar HOME | Restaurar backup `~/cred-bak-20260726` via rsync/cp -a | < 10s |

---

## 4. Garantia da Regra de Não-Intervenção
Nenhum arquivo de código Go (`config.go`, `agent.go`), binário, arquivo de credencial ou processo do daemon foi modificado nesta rodada. A implementação desta especificação permanece congelada aguardando a assinatura prévia do Codex56-TL.
