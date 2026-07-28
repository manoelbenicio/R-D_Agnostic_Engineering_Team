# Legacy daemon security remediation — ORQ-22

- Executor: Codex56-TL (`w5:pC`)
- Data UTC: 2026-07-27
- Host: ORQ1
- Escopo: metadados, nomes de chave e permissoes; nenhum valor sensivel foi
  impresso, registrado ou incluido nesta evidencia.

## Achado

`/tmp/daemon.env.bak` e `/tmp/daemon.cmd.bak` pertenciam a
`ec2-user:ec2-user` e estavam em modo `0664`. O primeiro continha nomes de
chave sensiveis. O daemon legado do ORQ1 nao tinha processo, unit de usuario,
unit de sistema nem referencia de autostart.

## Remediacao

1. Ambos os backups foram restringidos imediatamente para `0600`.
2. Nenhum arquivo estava em uso.
3. Os backups e `/tmp/multica-auth-fixed` foram movidos, sem delecao, para:

   `/home/ec2-user/.local/state/multica-quarantine/20260727Tsecurity-orq22/legacy-daemon`

4. A quarentena e subdiretorios estao em `0700`; backups em `0600`; binario
   legado em `0700`.
5. Sete candidatos adicionais identificados somente por nomes de chave foram
   restringidos e movidos para:

   `/home/ec2-user/.local/state/multica-quarantine/20260727Tsecurity-orq22/sensitive-tmp`

6. Todos os 74 arquivos `*.sh` e `*.json` restantes de `ec2-user` em `/tmp`
   foram normalizados para `0600`.
7. A varredura final encontrou zero arquivo com marcador sensivel ainda
   legivel por grupo ou outros.

## Decisao operacional

- O daemon legado do ORQ1 permanece permanentemente desativado.
- O runtime vigente e o daemon T2 do ORQ2, administrado por `systemd --user`.
- Os rollbacks operacionais vigentes sao os binarios duraveis
  `.pre-agy-fix`, `.previous` e o backup pre-token-only do Gate 2.
- O binario e o ambiente legados em quarentena nao sao necessarios para
  rollback operacional; ficam retidos apenas para analise forense ate decisao
  de descarte do owner. Nenhum item foi apagado.
- `ORQ-22` foi atualizada para `done`.

## Recomendacao de descarte

Autorizar a remocao da quarentena legada apos uma janela curta de retencao
forense (72 horas) e confirmacao de que nenhum incidente depende desses
artefatos. Manter indefinidamente aumenta a superficie de segredo sem fornecer
rollback util.
