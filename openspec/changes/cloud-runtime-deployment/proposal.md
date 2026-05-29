## Problema

AgentVerse v1 roda como SPA local conectada ao CAO em `127.0.0.1:9889`. Esse modelo atende desenvolvimento local, mas nao entrega runtime compartilhado, autenticado, isolado por tenant, nem operavel em producao cloud.

## Solucao proposta

Projetar o runtime cloud do AgentVerse com CAO em Cloud Run ou GKE, autenticacao via Firebase Auth e isolamento por tenant. Cada tenant deve ter CAO containerizado, configuracao isolada e limites operacionais claros para sessoes, terminais, providers e storage.

## Escopo

- Definir arquitetura Cloud Run vs. GKE para CAO.
- Integrar Firebase Auth no acesso ao runtime cloud.
- Criar modelo de isolamento per-tenant para CAO containers.
- Definir provisionamento, lifecycle, networking e secrets.
- Mapear custos e limites operacionais por tenant.
- Atualizar SPA para selecionar runtime local ou cloud.
- Criar smoke tests e runbooks de deploy.

## Dependencias de v1

- SPA estavel e CAO client encapsulado.
- Settings de `caoBaseUrl`.
- Health page e first-run checks.
- API key management BYOK local como base para migracao de secrets cloud.
- Canvas deploy/run end-to-end funcionando localmente.
