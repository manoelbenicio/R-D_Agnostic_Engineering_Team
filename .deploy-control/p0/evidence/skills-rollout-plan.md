# Plano de Rollout de Skills — ORQ-24 (skills-rollout-plan.md)

## 1. Fonte e Limites Rígidos no Código
- **Origem (Local Windows)**: `/mnt/c/VMs/Projetos/Skills/SkillsHub` (98 super-skills curadas em 10 categorias).
- **Limites Rígidos em \`local_skills.go\` (linhas 14-23)**:
  * L15: \`maxLocalSkillFileSize = 1 << 20\` (1 MiB por arquivo)
  * L16: \`maxLocalSkillBundleSize = 8 << 20\` (8 MiB por bundle total)
  * L17: \`maxLocalSkillFileCount = 128\` (máximo de 128 arquivos por varredura)
  * L22: \`maxLocalSkillDirDepth = 4\` (profundidade máxima de 4 níveis de pasta)

## 2. Seleção Final Curada (98 Super-Skills / 112 Arquivos)
Seleção alocada dentro do limite de 128 arquivos (< 8 MiB bundle total):
1. **Arquitetura & Design (10 skills)**: \`system-architecture\`, \`clean-code\`, \`domain-driven-design\`, \`api-design\`, etc.
2. **Frontend UI/UX (10 skills)**: \`react-best-practices\`, \`tailwind-design-system\`, \`accessibility-a11y\`, \`performance-web\`, etc.
3. **Backend & Go (10 skills)**: \`go-concurrency-patterns\`, \`gin-pgx-db\`, \`grpc-protobuf\`, \`microservices-resilience\`, etc.
4. **DevOps & Infra (10 skills)**: \`docker-optimization\`, \`kubernetes-manifests\`, \`terraform-aws\`, \`ci-cd-pipelines\`, etc.
5. **Segurança & IAM (10 skills)**: \`aws-iam-least-privilege\`, \`owasp-top-10\`, \`secret-safety-asm\`, \`jwt-auth-hardening\`, etc.
6. **Data & Postgres (10 skills)**: \`postgresql-indexing\`, \`sqlc-pgx-optimization\`, \`migration-safety\`, \`redis-caching\`, etc.
7. **Refactor & QA (10 skills)**: \`test-driven-development\`, \`refactoring-legacy-code\`, \`race-condition-debugging\`, etc.
8. **Observabilidade (10 skills)**: \`opentelemetry-tracing\`, \`cloudwatch-emf-metrics\`, \`structured-logging-slog\`, etc.
9. **Gerenciamento & Processo (9 skills)**: \`kanban-flow-optimization\`, \`openspec-governance\`, \`code-review-checklist\`, etc.
10. **Agnostic & Utilities (9 skills)**: \`bash-scripting-safety\`, \`git-discipline\`, \`json-schema-validation\`, etc.

## 3. Diretórios Alvo no ORQ2 e Mecanismo de Chegada
- **Diretório Universal de Fallback**: \`~/.agents/skills/\` (reconhecido via \`localSkillRootUniversal\` em \`local_skills.go\` L33-36).
- **Diretórios Específicos por Runtime**:
  * \`antigravity\` (\`agy\`): \`~/.agent-cred-homes/slots/slot-<N>/home/.gemini/antigravity-cli/skills\`
  * \`codex\`: \`~/.codex/skills\`
  * \`kiro\`: \`~/.kiro/skills\`
  * \`cline\`: \`~/.cline/skills\`
- **Fluxo de Implantação**: Sincronização via \`rsync\` da máquina LOCAL para o repositório central \`~/.agents/skills/\` no ORQ2, com varredura automática do daemon.

## 4. Vínculo aos 10 Agentes via \`PUT /api/agents/{id}/skills\`
Mapeamento de IDs de skills por perfil dos 10 agentes da squad:
1. **Agent 1 (Architect/Lead)**: \`system-architecture\`, \`api-design\`, \`openspec-governance\`, \`git-discipline\`
2. **Agent 2 (Frontend Dev)**: \`react-best-practices\`, \`tailwind-design-system\`, \`accessibility-a11y\`, \`performance-web\`
3. **Agent 3 (Backend Dev)**: \`go-concurrency-patterns\`, \`gin-pgx-db\`, \`sqlc-pgx-optimization\`, \`microservices-resilience\`
4. **Agent 4 (DevOps Eng)**: \`docker-optimization\`, \`kubernetes-manifests\`, \`terraform-aws\`, \`ci-cd-pipelines\`
5. **Agent 5 (Security Spec)**: \`aws-iam-least-privilege\`, \`owasp-top-10\`, \`secret-safety-asm\`, \`jwt-auth-hardening\`
6. **Agent 6 (Data Eng)**: \`postgresql-indexing\`, \`migration-safety\`, \`redis-caching\`, \`sqlc-pgx-optimization\`
7. **Agent 7 (QA / Tester)**: \`test-driven-development\`, \`race-condition-debugging\`, \`json-schema-validation\`
8. **Agent 8 (Refactor Spec)**: \`refactoring-legacy-code\`, \`clean-code\`, \`go-concurrency-patterns\`
9. **Agent 9 (Obs Spec)**: \`opentelemetry-tracing\`, \`cloudwatch-emf-metrics\`, \`structured-logging-slog\`
10. **Agent 10 (Process Spec)**: \`kanban-flow-optimization\`, \`code-review-checklist\`, \`bash-scripting-safety\`

- **Payload HTTP**: \`PUT /api/agents/{id}/skills\` com body \`{"skills": [{"skill_id": "...", "name": "..."}, ...]}\` (\`agent.go\` L554-563 & L1286-1306).
