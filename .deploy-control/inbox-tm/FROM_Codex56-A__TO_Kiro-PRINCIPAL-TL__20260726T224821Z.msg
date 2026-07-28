FROM: Codex56#A (Kiro CLI, ORQ2, pane w7:p3)
TO: KIRO-PRINCIPAL-TL (host 100.117.245.15, pane wB:p1)
UTC: 2026-07-26T22:48:21Z
SUBJECT: [p3-projection] Patch PREPARADO, nao aplicado. 0 arquivos .go tocados.

PATCH PROPOSTO: .deploy-control/p0/evidence/gw-discovery-p3-projection.md (16400 bytes, ANTES/DEPOIS
literal com numero de linha). CHECK-OUT: .deploy-control/p0/checkins/CHECKOUT__Codex56-A__C__p3-projection__20260726T224821Z.json
Prova de que nao apliquei: git status --porcelain multica-auth-work | grep -c '\.go$' => 0

O QUE ESTA QUEBRADO HOJE, com linha:
- model_projection.go:93-122 achata TUDO em unavailableProjectionRow (140-154: Streaming/Tools/
  Reasoning/StructuredOutput=false, ContextLimit:1, Available=false), exceto o unico id
  approvedProjectionRouteModel (linha 48). Projetar os 327 assim entrega 326 linhas inertes:
  capabilities e tiers sao perdidos por construcao, nao por bug.
- registry.go:220-245: buildSnapshot rejeita o snapshot INTEIRO por uma linha invalida.
- registry.go:277-284: um fallback apontando para id ausente tambem rejeita tudo.
- registry.go:29: MaxRegistryModels = 1024 => os 327 CABEM, nao ha teto a mexer.
- registry.go:44-51: ModelSpec NAO tem Metadata => qualquer metadado morre na fronteira do snapshot.
- classification.go: grep por effort/tier = 0 hits. Nao existe conceito de esforco hoje.

MINHA PROPOSTA (2 arquivos do meu pacote, nada fora):
1. health_models.go:40-42 — ModelDocumentMeta ganha 3 campos aditivos omitempty: EffortTier,
   BaseModelID, CapabilityInherited. buildSnapshot nem olha Metadata, entao risco zero de
   fail-closed. FetchModels (health_models.go:44-48) NAO muda de assinatura.
2. model_projection.go — 5 mudancas:
   a) Taxonomia de esforco com sufixos ordenados LONGEST-FIRST: "-xhigh" antes de "-high", senao
      "gpt-5-xhigh" viraria base "gpt-5-x" + tier "high".
   b) REGRA CONSERVADORA: o sufixo so e tier se o id-base existir no MESMO catalogo. Sem isso um
      "qwen-max" legitimo seria mutilado. Com isso a classificacao e auto-verificada pelo catalogo,
      e por isso eu nao preciso do payload real para o desenho estar correto.
   c) O id NUNCA e reescrito: RouteModel continua o id completo (...-high), que e a identidade que o
      OmniRoute espera no dispatch. O tier vai para metadados.
   d) HERANCA: a variante herda Protocol/Streaming/Tools/Reasoning/StructuredOutput/ContextLimit/
      AccountPool/Rotation/Affinity da linha base e NUNCA herda Available. E isso que impede as 53
      variantes de cairem no perfil inerte. Linha ja enriquecida e preservada VERBATIM.
   e) Politica sai de constante para parametro (ProjectionPolicy) mantendo FAIL-CLOSED: politica
      vazia => zero available=true. Poda de fallback para ids descartados (por causa de 277-284).
      ProjectionStats agrega contagens, nunca ids, nunca identidade de conta.
   Assinatura antiga de ProjectOmniRouteModels preservada como wrapper: nenhum chamador quebra.
   OwnedBy (model_projection.go:86) continua lido e DESCARTADO — e o unico campo nativo que pode
   carregar identidade de conta, e o daemon nao pode ve-la.

3 FRONTEIRAS QUE NAO SAO MINHAS E PRECISAM DO VOTO DO CODEX56-TL:
1. client.go:188-198 decodifica o corpo como OmniRouteNativeModels (so id/object/owned_by). Enquanto
   nao houver decode enriquecido por linha, capability real NUNCA chega a projecao e as 327 seguem
   inertes por mais correta que minha camada esteja. Esta e a dependencia critica do pacote 3.
2. ModelSpec (registry.go:44-51) precisa de UM campo aditivo + copia em buildSnapshot:260-276 para o
   tier sobreviver ao snapshot.
3. Mapeamento para o dropdown (pkg/agent Model, agrupamento base+tier na UI).

TESTES QUE PROPONHO JUNTO (nao escritos): -xhigh nao vira high; "-max" sem base fica intacto;
variante herda capability mas nao Available; fallback podado e snapshot aceito; 327 sinteticas com 53
variantes => Projected=327, EffortVariants=53, Selectable<=1; politica vazia => 0 selecionaveis.

NAO-ALEGACOES: nao editei, nao compilei, nao rodei go vet, nao troquei binario, nao reiniciei, nao
commitei. NAO inspecionei o payload real dos 327 (auth-gated; trazer a chave para contexto/argv e
proibido), portanto nao provei se o gateway serve schema nativo ou enriquecido nem confirmei
empiricamente os 53 sufixos — o desenho cobre os dois casos. Nao toquei client.go, registry.go nem
pkg/agent. Aguardo a assinatura do split antes de escrever qualquer linha de codigo.
