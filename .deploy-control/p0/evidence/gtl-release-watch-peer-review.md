# Peer Review de Auditoria: Vendor Release Watch (OpenAI + Anthropic) — GTL-44

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T11:45Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-official-release-notes-watch.md` (autor: Codex56#A w7:p3)  
**modo**: READ-ONLY / PEER REVIEW — NENHUM upgrade, alteração de código ou configuração executada  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: PASS** ✅

**Resumo da Avaliação**:
O documento de release watch `gtl-official-release-notes-watch.md` é de **excelente qualidade técnica e integridade factual**. Utiliza estritamente as fontes oficiais citadas (OpenAI Codex changelog e Anthropic Claude Code CHANGELOG no GitHub) e as medições de versão locais no ORQ1 e ORQ2. Não inventa datas para o CHANGELOG do Claude Code, classifica corretamente os riscos de quebra (breaking candidates), mapeia a variação crítica de profundidade de subagentes aninhados e entrega recomendações bem fundamentadas (APLICAR / AGENDAR / IGNORAR).

---

## 2. Matriz de Validação dos Critérios Requisitados

| Critério de Auditoria | Avaliação no Documento Original | Evidência & Confirmação | Status |
|---|---|---|---|
| **1. Fontes Oficiais & Datas** | Usou exclusivamente `<https://developers.openai.com/codex/changelog>` e `<https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md>`. | Datas do Codex validadas (0.145.0 em 2026-07-21). Ausência de datas no Claude Code reportada honestamente sem invenção. | ✅ PASS |
| **2. Versões Locais Medidas** | Medições no ORQ1 e ORQ2 registradas. | ORQ2: Codex `0.145.0`, Claude `2.1.215`. ORQ1: Codex `0.144.6`, Claude `2.1.218`. Desalinhamento da frota confirmado. | ✅ PASS |
| **3. Breaking Candidates** | Identificados pontos críticos de regressão e quebra em ambas as ferramentas CLI. | **Codex**: Rejeição de fork de threads paginadas (#33109) e migração de exec-policy (#34271). **Claude**: Subagents depth default 1->3, skills background por default, rejeição de `:` em nomes de agentes. | ✅ PASS |
| **4. Nested Subagent Depth** | Identificada a alteração de maior blast radius no Claude Code 2.1.219 (depth 1 -> 3 por padrão). | Recomenda fixar `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1` antes de qualquer upgrade para conter spawn descontrolado de processos. | ✅ PASS |
| **5. Workspace Trust & Segurança** | Mapeadas correções de segurança em Claude 2.1.218. | Frontmatter hooks agora exigem workspace trust no diretório do arquivo do agente (ORQ2 em 2.1.215 está vulnerável sem essa correção). | ✅ PASS |
| **6. Gateway Spend Metering** | Mapeada a correção em Claude 2.1.218. | Correção da precificação de perfis de inferência e IDs de modelos mapeados em gateway/Bedrock. | ✅ PASS |
| **7. Recomendações (Aplicar/Agendar/Ignorar)** | Classificação clara de cada item das releases. | APLICAR (endurecimento de segurança/metering), AGENDAR (alinhamento de frota e validação de breaking), IGNORAR (Bedrock direto, áudio, ripgrep interno). | ✅ PASS |
| **8. Rejeição de Claims sem Fonte** | Rigor estrito contra inferências não fundadas. | Seção 5 de "Não-afirmações" explicita todos os limites e não assume disponibilidade em gerenciadores de pacotes não testados. | ✅ PASS |

---

## 3. Análise dos Principais Achados de Segurança e Arquitetura

### 3.1 Desalinhamento da Frota entre ORQ1 e ORQ2
- **Codex**: ORQ2 está na `0.145.0`, enquanto ORQ1 está na `0.144.6`.
- **Claude Code**: ORQ1 está na `2.1.218`, enquanto ORQ2 está na `2.1.215`.
- **Risco**: Runtimes com comportamentos divergentes entre hosts invalidam testes comparativos da frota. Recomenda-se alinhar ambas as instâncias para uma versão idêntica de forma coordenada.

### 3.2 Alerta de Blast Radius: Profundidade de Subagentes (Claude Code 2.1.219)
A alteração do padrão de profundidade de subagentes de `1` para `3` em Claude Code 2.1.219 pode provocar uma multiplicação exponencial de sub-processos em execuções paralelas. A recomendação de forçar `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1` nas variáveis de ambiente da frota antes de qualquer atualização é **estritamente necessária**.

### 3.3 Correção de Execução de Hooks e Workspace Trust (Claude Code 2.1.218)
A versão 2.1.218 corrigiu um vetor onde hooks de frontmatter em arquivos markdown podiam ser executados a partir de diretórios não confiáveis. Como o host ORQ2 está na versão `2.1.215`, ele atualmente **não possui** esse patch de segurança.

---

## 4. Conclusão

O relatório `gtl-official-release-notes-watch.md` está **TOTALMENTE APROVADO (PASS)**. Ele fornece um diagnóstico seguro, livre de alucinações e com altíssimo valor estratégico para a governança dos CLIs da frota.

*Auditoria 100% READ-ONLY. Nenhum pacote foi instalado, atualizado ou reiniciado.*
