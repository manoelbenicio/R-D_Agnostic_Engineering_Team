Plano C (B7) — ciclo de vida dos cabeçalhos MCP do tooling nos hosts

Card: ORQ-37, escopo canônico definido pelo ruling do GTL de 2026-07-28T16:17Z
Executor: Codex56#A (w7:p3) · UTC 2026-07-28T16:34Z
Substitui, no escopo deste card, a leitura anterior de três planos:
  - "Gatekeeper" removido do vocabulário — não existe no produto;
  - nenhuma alegação de superfície MCP de produto é feita aqui;
  - Plano A (mdt_) pertence exclusivamente ao ORQ-43;
  - Plano B (autorização de leitura de agent.mcp_config) permanece comportamento existente e vira
    follow-up separado de teste de regressão (§7), não implementado aqui.

Modo: READ-ONLY e VALOR NUNCA LIDO. Registro explicitamente que, numa das minhas sondagens, tentei
extrair apenas nomes de cabeçalho e comprimento de valor; o filtro de redação encobriu também os nomes,
e eu não repeti a tentativa — parsear o arquivo é caminho de leitura de valor e o ruling proíbe. Toda a
caracterização abaixo usa metadado de arquivo, hash, ausência de referência e conhecimento já registrado
na Wave A, que classificou esses arquivos por NOME de chave (Authorization/Bearer) no momento da
contenção.

1. Inventário factual, medido agora

  /tmp/mh      mode=600  owner=ec2-user:ec2-user  size=1387  mtime=2026-07-24 12:11:26Z
  /tmp/mcp_h   mode=600  owner=ec2-user:ec2-user  size=1387  mtime=2026-07-24 12:03:43Z
  sha256(prefixo 16)  mh=a65e2f5e28b58806   mcp_h=5d9fd799237482f9   -> conteúdos DIFERENTES
  cópias/backups em /home/ec2-user e /tmp (profundidade 3): NENHUMA além dos dois originais
  host: ORQ1 ip-172-31-18-217 apenas; não existem equivalentes no ORQ2 nem no LOCAL

  Mesmo tamanho exato (1387 B) com hashes distintos e criação a 7m43s de distância indica duas variantes
  do mesmo conjunto de cabeçalhos, não duplicata.

2. Issuer: NÃO DETERMINÁVEL a partir do host — e isso é a lacuna central do card

  - nenhuma referência a /tmp/mh ou /tmp/mcp_h em código de produto, script, unit systemd, config em
    ~/.config, ~/.kiro, ~/.aws, nem no histórico de shell legível;
  - não existe arquivo mcp.json no host ORQ1 (a busca por mcp.json e .mcp.json em profundidade 4 sob
    /home/ec2-user retorna vazio); os únicos configs MCP do projeto estão no repositório, e o do repo
    (.kiro/settings/mcp.json, servidor aws-mcp) declara apenas command, args e timeout, sem bloco env e
    sem headers;
  - uvx e npx estão instalados, o que é compatível com servidores MCP lançados ad hoc por ferramenta,
    sem unit nem config persistente.

  Conclusão: os dois arquivos são artefatos manuais de sessão, criados em 2026-07-24, sem emissor
  identificável no host e sem nenhum consumidor automatizado. Quem emitiu o Bearer é informação que
  somente o owner possui.

3. Consumer: ZERO consumidores ativos ou declarados

  - lsof nos dois arquivos: nenhum processo com o arquivo aberto;
  - nenhuma referência textual em qualquer script, unit ou config do host;
  - portanto remover, mover ou invalidar esses arquivos não quebra nenhum fluxo automatizado conhecido.
    O risco de disrupção é limitado a um operador humano que os use manualmente em linha de comando.

4. Custódia atual e o que está errado nela

  correto hoje: modo 600 e dono ec2-user (contenção da Wave A, ainda íntegra — reverificado no ORQ-31).
  errado hoje:
    a) localização: /tmp é diretório compartilhado, 1777, sujeito a limpeza automática e a leitura por
       qualquer processo do mesmo UID; um segredo de longa duração não pertence a /tmp;
    b) causa raiz ativa: /etc/bashrc:75 define umask 002 e o shell atual reporta 0002, então qualquer
       novo artefato nasce mundo-legível e a contenção precisa ser reaplicada a cada criação;
    c) ausência de registro de proveniência: não há nota de qual serviço emitiu, quando expira, nem quem
       pode rotacionar.

5. Contrato executável do Plano C (nenhum passo exige ler valor)

  Pré-condições
  C1. owner declara o EMISSOR de cada um dos dois cabeçalhos e se são a mesma credencial em formatos
      diferentes ou credenciais distintas — decidir isso lendo metadado é impossível.
  C2. owner declara se existe consumidor humano ativo; se não existir, o card passa a ser remoção, não
      rotação.
  C3. janela sem uso manual concorrente, e nenhuma ação durante execução de tarefa que depende de MCP.

  Custódia
  C4. mover os dois arquivos para diretório privado do usuário com install -d -m 700, por exemplo
      /home/ec2-user/.config/mcp-headers, preservando modo 600, usando mv -n para nunca sobrescrever;
      nenhuma leitura de conteúdo é necessária para mover.
  C5. registrar, em nota separada e sem valor, a proveniência declarada em C1, a data de criação
      (2026-07-24) e o hash atual de cada arquivo, para detecção futura de troca.
  C6. estrutural: aplicar umask 077 por drop-in para o usuário de execução, de modo que novos artefatos
      não renasçam mundo-legíveis. Isto é Wave B estrutural e continua sem autorização.

  Rotação, se C1 indicar credencial ativa
  C7. rotação ocorre NA ORIGEM declarada em C1, não no host: o arquivo é cópia local de um cabeçalho.
  C8. o novo valor entra no arquivo pelo próprio owner, ou por asm-exec com referência dinâmica se e
      quando a credencial passar a viver no Secrets Manager; eu não manipulo valor em nenhuma hipótese.
  C9. gate de sucesso sem leitura de valor: hash do arquivo muda, modo permanece 600, dono permanece
      ec2-user, tamanho registrado, e a ferramenta que o consome — se existir — volta a autenticar num
      teste conduzido pelo owner.

  Rollback
  C10. antes de qualquer movimentação, snapshot de metadado (caminho, modo, dono, tamanho, mtime, hash)
       em arquivo 0600 sob diretório privado; nada de copiar conteúdo para backup adicional, para não
       multiplicar cópias do segredo.
  C11. rollback de custódia é mv -n de volta ao caminho original com o mesmo modo; verificação por hash
       igual ao snapshot.
  C12. rollback de rotação é responsabilidade da origem (C7): reverter valor local sem reverter na
       origem produz credencial inválida silenciosa, o que é pior que a falha.
  C13. se a decisão de C2 for remoção, a remoção deve ser precedida de rotação/revogação na origem;
       apagar o arquivo NÃO invalida a credencial — é exatamente o que aconteceu no host LOCAL, onde os
       16 artefatos da Wave A desapareceram e os segredos seguem válidos até rotação.

6. Readiness do Plano C: BLOCK, e o bloqueio é de INPUT do owner, não técnico

  Todo o resto está pronto: inventário medido, consumidor zero, custódia com defeito conhecido, contrato
  e rollback escritos e executáveis sem tocar valor. O que impede executar é C1 e C2, que nenhum agente
  pode responder por inspeção.

  Inputs exatos que eu preciso, e nada além destes três:
  I1. para /tmp/mh e para /tmp/mcp_h, qual serviço emitiu o cabeçalho e qual é a rota de rotação na
      origem (nome do serviço e do procedimento, sem valor).
  I2. existe consumidor humano ativo desses arquivos? Sim -> executar C4 a C9. Não -> executar C13 e
      encerrar o item como remoção após revogação na origem.
  I3. autorização, ou negativa explícita, para o drop-in de umask 077 do C6, que é a única correção que
      impede a recriação do problema.

7. Follow-up separado, por determinação do GTL (não implementado aqui)

  Teste de regressão do Plano B: garantir que ator do tipo agent, mesmo com PAT do host que satisfaria o
  papel owner/admin, NUNCA receba agent.mcp_config. O comportamento existe hoje em
  internal/handler/agent.go:571 e :601-606; o follow-up é apenas o teste que o congela. Card próprio,
  fora do ORQ-37.

8. Não-alegações

  - Não li o valor dos arquivos e não repeti a sondagem que poderia expor conteúdo.
  - Não afirmo qual serviço emitiu os cabeçalhos: é exatamente o input I1.
  - Não afirmo que as credenciais estão válidas ou inválidas; não há como saber por metadado.
  - Não movi, apaguei, rotacionei nem alterei modo de nenhum arquivo; nenhuma mutação em nenhum host.
  - Não toquei mdt_ (ORQ-43), não implementei nada do Plano B, não chamei AWS, não usei asm-exec, não
    acessei o SMA e não toquei board, runtime, container ou daemon.
  - A afirmação de que esses arquivos contêm cabeçalho Authorization/Bearer vem da classificação por
    NOME de chave feita na Wave A, não de leitura de valor agora.
