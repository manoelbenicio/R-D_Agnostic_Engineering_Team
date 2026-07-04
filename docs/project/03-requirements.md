# 03 — Requisitos

**Data:** 2026-07-01 · Convenção: MUST/SHOULD/MAY (RFC 2119).

---

## 1. Requisitos Funcionais (RF)

### Isolamento (Fase 1)
- **RF-01** O sistema MUST armazenar a credencial de cada conta de vendor em um
  local isolado, sem compartilhar `auth.json`/token entre contas.
- **RF-02** O sistema MUST apontar cada agente para a conta atribuída via a env var
  nativa do vendor (Codex `CODEX_HOME`; Kiro `XDG_DATA_HOME`/`KIRO_API_KEY`;
  Antigravity `HOME`).
- **RF-03** O sistema MUST restaurar a credencial no formato bruto do vendor (AS-IS),
  preservando o `refresh_token`.
- **RF-04** O sistema MUST NOT persistir credencial já usada/expirada.
- **RF-05** O sistema MUST permitir múltiplas contas por vendor.
- **RF-06** O sistema MUST permitir uso concorrente de uma credencial válida por
  N agentes durante seu ciclo de vida.
- **RF-07** Sem atribuição configurada, o sistema MUST preservar o comportamento atual
  (fallback global) — compatibilidade retroativa.

### Seleção/atribuição
- **RF-08** O sistema MUST resolver qual conta um agente usa (mapa de atribuição).
- **RF-09** A seleção da próxima conta SHOULD seguir prioridade por expertise
  (ordem configurável).

### Rotação (Fase 2)
- **RF-10** O sistema MUST detectar esgotamento de cota (padrão na tela do vendor
  e/ou HTTP 429 e/ou ledger de cota).
- **RF-11** Ao esgotar, o sistema MUST trocar para a próxima conta disponível do
  mesmo vendor e retomar a tarefa, sem intervenção manual.
- **RF-12** Quando todas as contas estão esgotadas, o sistema MUST sinalizar e
  agendar retomada para o menor `cooldown_until`.
- **RF-13** O sistema SHOULD distinguir 503/"high traffic" de 429/quota (não
  rotacionar por erro transitório de servidor).

## 2. Requisitos Não-Funcionais (RNF)

### Segurança
- **RNF-01** O sistema MUST NOT registrar conteúdo de credencial em log (apenas
  metadados: caminho, tipo, mtime).
- **RNF-02** Segredos MUST NOT trafegar para o frontend em texto claro (masking).
- **RNF-03** Permissões de diretório de credencial MUST ser restritivas (0700).

### Persistência
- **RNF-04** Toda persistência introduzida MUST usar Postgres (com pool de conexões).
- **RNF-05** O sistema MUST NOT introduzir SQLite como backend próprio (o SQLite
  nativo de um vendor, ex. Kiro, é tolerado apenas como store daquele CLI).

### Confiabilidade
- **RNF-06** A mudança MUST NOT causar regressão em orquestração/canvas/dispatch/UI.
- **RNF-07** Cada mudança MUST compilar via container `golang:1.26-alpine`.
- **RNF-08** As suítes de teste dos pacotes tocados MUST permanecer verdes.

### Manutenibilidade / evolução
- **RNF-09** Arquivos tocados SHOULD ser higienizados de branding "Multica" sem
  alterar comportamento observável.
- **RNF-10** A superfície de mudança MUST ser mínima e localizada no caminho de auth.
- **RNF-11** O desenvolvimento MUST ocorrer na cópia local `multica-auth-work/`; o
  source original MUST NOT ser alterado.

### Observabilidade
- **RNF-12** Todo componente do fluxo de credencial/cota/rotação MUST expor métricas
  (Prometheus) e MUST ser visualizável em dashboards (Grafana) — ver doc 05.
- **RNF-13** O stack de observabilidade MUST ter deploy fácil (docker-compose).

## 3. Restrições

- **C-01** Vendors em escopo: Codex, Kiro (Opus via Kiro), Antigravity. (Sem Claude
  direto — não há integração.)
- **C-02** Login OAuth idêntico AS-IS ao sistema de auth já existente.
- **C-03** Sem dependência de runtime de sistema de outro departamento (integração
  só via padrão store/restore, não acoplamento a API externa em produção).

## 4. Rastreabilidade (requisito → lacuna AS-IS)

| Lacuna (doc 01) | Requisitos que a fecham |
|-----------------|-------------------------|
| G1 sem isolamento | RF-01, RF-02, RF-03 |
| G2 sem seleção | RF-08, RF-09 |
| G3 sem rotação | RF-10..RF-13 |
| G4 sem observabilidade | RNF-12, RNF-13, doc 05 |