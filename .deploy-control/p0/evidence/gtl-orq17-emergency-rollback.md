# Relatório de Rollback de Emergência: Reset de Tailscale Serve no ORQ1 (ORQ-17)

**autor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T15:12Z  
**governança**: Kanban Issue `ORQ-17`  
**autorização**: Rollback de emergência pré-autorizado executado a pedido prioritário  
**nó de destino**: ORQ1 (`100.118.244.61`, `orq1.tail96e2c0.ts.net`)  
**modo**: ROLLBACK DE EMERGÊNCIA E CONTENÇÃO — `tailscale serve reset` executado UMA ÚNICA VEZ. Zero alterações de env, container, ACL ou Funnel. Zero leitura de segredos.  

---

## 1. Veredito Terminal de Rollback

### **VEREDITO TERMINAL: CONTAINED / ROLLED_BACK** ✅

**Resumo Executivo**:
O procedimento de rollback de emergência para o Gate A no host ORQ1 foi executado e concluído com 100% de contenção. O comando `tailscale serve reset` foi rodado exatamente uma vez (código de saída 0). As verificações pós-rollback confirmam que o `tailscale serve status` retornou ao estado limpo (`No serve config`), a porta pública HTTPS 443 foi fechada (`000 Connection refused`), os serviços locais no loopback (`13100` e `18080`) continuam intactos e respondendo `HTTP 200 OK`, e o túnel ORQ2 permanece ativo e estável.

---

## 2. Registro do Comando de Rollback (Execution Log)

```bash
# Executado uma única vez no host ORQ1:
tailscale serve reset
# Exit Code: 0
```

---

## 3. Medições Factuais Pós-Rollback (Validation Matrix)

| Item de Validação | Comando / Método Probe | Resultado Medido | Estado de Contenção |
|---|---|---|---|
| **Exit Code do Reset** | `tailscale serve reset` | **`0` (Sucesso)** | ✅ **EXECUTADO UMA VEZ** |
| **Estado do Serve** | `tailscale serve status` | **`No serve config`** | ✅ **100% LIMPO** |
| **Porta 443 HTTPS FQDN** | `curl https://orq1.tail96e2c0.ts.net/` | **`000` (Connection Refused)** | ✅ **DESATIVADO / FECHADO** |
| **Frontend Next.js Loopback** | `curl http://127.0.0.1:13100/` | **`200 OK`** | ✅ **PRESERVADO INTATCO** |
| **Backend Go Loopback** | `curl http://127.0.0.1:18080/healthz` | **`200 OK`** | ✅ **PRESERVADO INTACTO** |
| **Túnel de Comunicação ORQ2** | SSH / Conectividade de Rede | **Conectado / Ativo** | ✅ **SEM INTERRUPÇÕES** |

---

## 4. Garantias de Integridade e Segurança

1. **Zero Mutações não Autorizadas**: Nenhuma alteração foi realizada em variáveis de ambiente, containers Docker, regras de ACL do Tailscale ou configurações de Funnel.
2. **Zero Restarts de Processos**: Os processos do Next.js e do Go Backend continuam rodando continuamente sem reinícios.
3. **Privacidade de Segredos**: Zero credenciais, tokens ou biscoitos foram lidos ou impressos.

---

## 5. Check-out Citing ORQ-17

- **Governança**: `ORQ-17`
- **Artefato Gerado**: `.deploy-control/p0/evidence/gtl-orq17-emergency-rollback.md`
- **Veredito Terminal**: **CONTAINED / ROLLED_BACK** ✅
- **Status de Mutação**: Rollback executado com sucesso e verificado.
