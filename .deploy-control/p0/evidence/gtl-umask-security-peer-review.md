# Parecer de Peer Review de Segurança: Remediação de /tmp e Umask (READ-ONLY GTL-24)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Documento Auditado:** `.deploy-control/p0/evidence/gtl-tmp-umask-remediation-plan.md`
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T11:35:15Z
- **Veredito:** **PASS** (Aprovado com recomendação de sequência segura)

---

## 1. Avaliação do Inventário & Diagnóstico da Causa Raiz

### 1.1 Inventário (Sem Leitura de Segredos)
- O inventário apresentado no documento foi verificado sem leitura de valores em memória (`stat`, `find` e `grep -c`).
- Confirma-se a persistência de 4 arquivos críticos com exposição de credenciais em `/tmp` mundo-legível (`umask 0002` / `0022`):
  - `/tmp/arch.txt` (`OPENAI_API_KEY`)
  - `/tmp/backend-recover.sh` (`JWT_SECRET`, `POSTGRES_PASSWORD`)
  - `/tmp/handshake-token.txt` (token cru)
  - `/tmp/rev-token.txt` (token cru)

### 1.2 Causa Raiz (`/etc/bashrc`)
- Confirmada: `/etc/bashrc` (linhas 70-77) no Amazon Linux força `umask 002` em shells interativos non-login para `UID > 199`.
- **Conclusão:** `login.defs` e `pam_umask.so` são sobrepostos pelo `bashrc`. Apenas a Camada A (TMPDIR privado `0700`) e a Camada B (drop-in systemd `UMask=0077`) garantem isolamento efetivo.

---

## 2. Auditoria da Classificação: Rotação vs Contenção

| Categoria | Arquivos Afetados | Razão Técnica | Ação Obrigatória |
| :--- | :--- | :--- | :--- |
| **ROTAÇÃO OBRIGATÓRIA** | `/tmp/arch.txt`, `/tmp/backend-recover.sh`, `/tmp/handshake-token.txt`, `/tmp/rev-token.txt` | Expostos em `/tmp` (1777) mundo-legível por >21 horas. `chmod 600` é apenas contenção temporária. | **Rotacionar a credencial no provedor primeiro, depois deletar o arquivo.** |
| **CONTENÇÃO (`chmod 600` / Quarentena)** | `sec.txt`, `o30.txt`, `ph.txt`, `dec.txt`, `f2-fulltest.log`, `mcp.json.bak`, etc. | Logs de diagnóstico e scripts sem credenciais ativas. | Aplicar `chmod 0600` ou mover para `$HOME/.private-tmp/quarantine/`. |
| **ESCALAR / NÃO TOCAR** | `/tmp/test_login.html` (dono `root`), `/tmp/daemon.environ.*.bak` (já `0600`). | Pertencem ao root ou já são backups seguros de rollback. | Não alterar permissões. |

---

## 3. Auditoria de Segurança dos Comandos Propostos

- **Comandos Destrutivos:** **NENHUM** comando destrutivo desancorado ou `rm -rf` em massa foi encontrado no plano.
- **Comandos de Contenção (`find` + `chmod`):** O comando `find /tmp -maxdepth 1 -type f -user "$(id -un)" -perm /077 -exec chmod 600 {} +` é seguro e restrito estritamente aos arquivos do usuário logado.
- **Segurança de CLI Argv:** Nenhuma credencial é repassada em argumentos de linha de comando (`argv`), variáveis de ambiente inline ou flags HTTP (`-H`). O plano especifica o uso exclusivo de `asm-exec` com referências dinâmicas e arquivos `.curlrc` com permissão `0600`.

---

## 4. Ordem Segura de Execução Recomendada (para Codex56-TL)

1. **Passo 1 (Camada A):** Criar `$HOME/.private-tmp` com permissão `0700` e configurar `TMPDIR`, `GOCACHE` e `GOTMPDIR`.
2. **Passo 2 (Contenção Imediata):** Executar a contenção restrita `chmod 0600` nos arquivos `/tmp` do usuário logado.
3. **Passo 3 (Rotação de Credenciais):** Rotacionar as 4 credenciais de alta prioridade (`OPENAI_API_KEY`, `JWT_SECRET`, `POSTGRES_PASSWORD`, tokens de handshake) antes de remover os arquivos temporários.
4. **Passo 4 (Camada B):** Aplicar o drop-in systemd `UMask=0077` sob autorização do owner (GATE D3).

---

## 5. Veredito Final: PASS

O plano em `gtl-tmp-umask-remediation-plan.md` está **APROVADO (PASS)** para execução orientada pelo `GENERAL-TECH-LEAD` sob autorização de gates do owner.
