# Regularização e Mapeamento Oficial Kanban GTL-K02: Segurança e Legados

- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-27T12:35:31Z
- **Workspace ID:** `20fce817-895d-447b-965a-49f5e279314a` (`orq2-dev`)
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Modo:** API / KANBAN REGULARIZATION — Zero mutações em código, testes ou infraestrutura viva.

---

## 1. Resumo da Regularização de Cards Kanban

Conforme diretiva do Owner e a governança GTL, foi feita a busca por duplicatas no Kanban antes da criação de cards. A Wave A de Segurança e os legados GTL-42, GTL-83 e GTL-18 foram devidamente mapeados para seus respectivos cards `ORQ-N`. Para os itens sem representação anterior no Kanban (Wave A retrospectivo e as 6 classes de credenciais da Wave B), foram criadas issues únicas dedicadas via API do backend.

---

## 2. Mapeamento Consolidado de Cards `ORQ-N`

| Task GTL | Card ID | Identificador | Título / Descrição Resumida | Status Kanban |
|---|---|---|---|---|
| **GTL-42** | `b3cec211-a6d8-4e32-a538-e857635578d2` | **ORQ-18** | Adicionar botão de exclusão de runtime na UI | `todo` |
| **GTL-83 / Backend JWT** | `4369b017-4740-4ea4-bd7e-d9a8a86d7c03` | **ORQ-30** | Persist backend restart environment and remediate JWT rotation incident | `in_review` |
| **GTL-18 / GTL-53** | `80698dbb-a746-4f0e-8ac7-d3998110c8a3` | **ORQ-24** | Implantar skills curadas nos 10 agentes | `done` |
| **Security Wave A (Permissions & Quarantine)** | `f7e13350-c7f2-4335-8a04-01b98527d034` | **ORQ-31** | Security Wave A Containment (Permissions & Quarantine) | `in_review` |
| **Security Wave B (Handshake Token)** | `b01925fe-e914-422a-812e-f63cada274dc` | **ORQ-32** | Security Wave B: Handshake Token Rotation & Lifecycle | `todo` |
| **Security Wave B (Rev Token)** | `dfeabbdc-33e1-4ab8-9460-27b43df227db` | **ORQ-33** | Security Wave B: Rev Token Rotation & Validation | `todo` |
| **Security Wave B (OPENAI_API_KEY)** | `685524e4-eec7-4e7e-9e10-a4a51346fc14` | **ORQ-34** | Security Wave B: OPENAI_API_KEY Secret Management & Rotation | `todo` |
| **Security Wave B (PostgreSQL / DATABASE_URL)** | `3f73ff90-55a1-4c2f-a52d-d3735580ce7e` | **ORQ-35** | Security Wave B: PostgreSQL / DATABASE_URL Credential Hardening | `todo` |
| **Security Wave B (MULTICA_TOKEN)** | `41645aaf-83ff-4f50-9322-177e51ed99c3` | **ORQ-36** | Security Wave B: MULTICA_TOKEN Authentication & Secret Governance | `todo` |
| **Security Wave B (MCP Authorization)** | `36d18727-f516-4147-9b0c-1cb7c2b91e83` | **ORQ-37** | Security Wave B: MCP Authorization & Gatekeeper Token Lifecycle | `todo` |

---

## 3. Diretivas de Segurança e Proteção de Segredos
- **Zero Segredos nos Cards**: Nenhum valor de credencial, senha, token, chave de API ou caminho privado foi incluído nos cards criados/atualizados.
- **Bloqueio de Rotações**: Todas as rotações de credenciais da Wave B permanecem com status `todo` e bloqueadas até a execução e revisão dos gates específicos por card.
