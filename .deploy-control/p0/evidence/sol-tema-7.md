# Relatório de Medição Factual por Conta e Matriz CLI — Tema 7

## 1. Evidência do Bug de Ignoração de Slot no Código
- **\`execenv.go\` (linhas 300-349)**: Os adaptadores de isolamento por conta JÁ EXISTEM para todos os CLIs:
  * L301-303: \`kiro\` -> \`prepareKiroHome(kiroDataHome, KiroHomeOptions{AccountHome: params.CredentialAccountHome})\`
  * L312-314: \`antigravity\` -> \`prepareAntigravityHome(agyHome, AntigravityHomeOptions{AccountHome: params.CredentialAccountHome})\`
  * L322-325: \`cline\` -> \`prepareClineHome(clineDataDir, ClineHomeOptions{AccountHome: params.CredentialAccountHome})\`
  * L341-345: \`opencode\` / \`glm\` -> \`prepareOpenCodeHome(opencodeDataHome, opencodeConfigHome, OpenCodeHomeOptions{AccountHome: params.CredentialAccountHome})\`
  * L300: Comentário literal: \`// Empty CredentialAccountHome = shared/global behavior (no isolated home).\`
- **\`daemon.go\` (linhas 3448, 3475, 3491)**:
  * L3448: \`credentialAccountHome := ""\` (Hardcoded para string vazia).
  * L3475 & L3491: Passa a string vazia para \`execenv.Reuse\` e \`execenv.PrepareParams\`, forçando o retorno ao comportamento global compartilhado.

## 2. Matriz de Medição Factual por Slot e CLI (Leitura em Disco)

| Slot ID | AGY (\`antigravity\`) | OPENCODE | KIRO-CLI | CODEX | Status da Credencial |
|---|---|---|---|---|---|
| \`slot-141\` | **11 modelos OK** | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | **CONTA ATIVA 1 (a2a7860c)** |
| \`slot-145\` | **11 modelos OK** | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | **CONTA ATIVA 2 (965277db)** |
| \`slot-146\` | **11 modelos OK** | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | **CONTA ATIVA 3 (aebdffce)** |
| \`slot-150\` | **11 modelos OK** | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | **CONTA ATIVA 4 (199009bc)** |
| \`slot-139\` | Requer Login | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | OpenCode Ativo / Kiro Logado |
| \`slot-140\` | Requer Login | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | OpenCode Ativo / Kiro Logado |
| \`slot-142\` | Requer Login | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | OpenCode Ativo |
| \`slot-143\` | Requer Login | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | OpenCode Ativo / Kiro Logado |
| \`slot-149\` | Requer Login | **7 modelos OK** | Presente (\`.kiro\`) | Presente (\`.codex\`) | OpenCode Ativo / Kiro Logado |
| \`slot-152\` | Requer Login | **7 modelos OK** | Ausente | Presente (\`.codex\`) | OpenCode Ativo |

## 3. Conclusão da Medição
- **Contas AGY**: Exatamente 4 das 4 contas do OmniRoute (\`slot-141\`, \`slot-145\`, \`slot-146\`, \`slot-150\`) respondem com 100% de sucesso retornando os 11 modelos completos (incluindo \`claude-opus-4-6-thinking\`, \`gemini-3.1-pro-high\`, etc.).
- **OPENCODE**: Funciona e retorna 7 modelos em 100% dos slots ativos.
- **KIRO**: Estruturas de SSO/credencial logada existem nos slots 139, 140, 142, 143, 149.
- **Solução no Patch**: Basta passar o caminho absoluto do slot (ex: \`~/.agent-cred-homes/slots/slot-NNN/home\`) em \`credentialAccountHome\` na linha 3448 do \`daemon.go\`.
