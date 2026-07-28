# PACOTE 3 — p3-projection (PROPOSTA, NAO APLICADA)

Autor: Codex56#A (Kiro CLI, ORQ2, pane `w7:p3`) · UTC 2026-07-26T22:52Z
Arquivos do meu pacote: `multica-auth-work/server/internal/daemon/gateway/model_projection.go`,
`multica-auth-work/server/internal/daemon/gateway/health_models.go`
Estado: nada editado, nada compilado, nada commitado. `git status` do subtree limpo.
Depende de: assinatura do split por Codex56-TL. Fronteiras fora do meu pacote estao na secao 6.

## 1. O que o codigo faz hoje (lido, nao suposto)

- `client.go:188-198` — quando `OMNIROUTE_DEV_MODELS_COMPAT=1`, decodifica o corpo como
  `OmniRouteNativeModels` (OpenAI-basic: `{id,object,owned_by}`) e chama
  `ProjectOmniRouteModels(native, header X-OmniRoute-Registry-Version)`.
- `model_projection.go:93-122` — projeta cada id em uma linha valida, mas **achata capability**:
  tudo vira `unavailableProjectionRow` (linhas 140-154: `Streaming/Tools/Reasoning/StructuredOutput =
  false`, `ContextLimit: 1`, `Available: false`), exceto o unico id
  `approvedProjectionRouteModel = "claude_code_kimi_2.7_Code"` (linha 48), que recebe
  `approvedProjectionRow` (124-138).
- `registry.go:220-245` — `buildSnapshot` rejeita o snapshot INTEIRO (`ErrorProtocol`) se qualquer
  linha falhar: id nao-parseavel, id duplicado, protocolo desconhecido, qualquer um dos 4 ponteiros
  de capability nil, `ContextLimit <= 0`, `AccountPool` vazio/>128/com CR-LF-NUL.
- `registry.go:277-284` — todo id em `Fallback` tem de existir no snapshot, senao rejeicao total.
- `registry.go:29` — `MaxRegistryModels = 1024`. **Os 327 modelos cabem; nao ha teto a mexer.**
- `registry.go:44-51` — `ModelSpec` carrega `Capability/AccountPool/Rotation/Affinity/Fallback/
  Available` e **nao** carrega `Metadata`: qualquer campo novo de metadados morre na fronteira do
  snapshot (ver secao 6, item 2).
- Nenhum conceito de esforco existe hoje no pacote: `grep -n "effort\|Effort\|tier\|Tier"
  classification.go` retorna zero linhas relevantes (so `AuthOutcome`, `RateLimitScope`,
  `FailureSignal`, `ClassifyFailure`).

Consequencia direta: projetar os 327 com o codigo atual entrega 326 linhas inertes. Capabilities e
tiers de esforco sao perdidos por construcao, nao por bug.

## 2. Principio da proposta

A projecao **nunca inventa capability a partir de dado de vendor** (mantem a regra atual) e **nunca
propaga identidade de conta**: `OmniRouteNativeModel.OwnedBy` (`model_projection.go:86`) continua lido
e descartado — ele e o unico campo nativo que pode carregar identidade de conta, e o daemon nao pode
ve-la. O que muda: (a) se a linha vier no schema ENRIQUECIDO, ela e **preservada verbatim** em vez de
achatada; (b) o tier de esforco e **derivado do proprio id**, que e dado de roteamento publico, nao
segredo; (c) a decisao de selecionabilidade sai de uma constante e passa a ser politica injetada.

## 3. Taxonomia de esforco (regra conservadora, com prova no proprio catalogo)

Sufixos: `-high`, `-medium`, `-low`, `-max`, `-xhigh`, `-fast` (53 de 327 segundo a ordem).

Regra: um sufixo so e tratado como tier **se o id-base (id menos o sufixo) tambem existir no mesmo
catalogo**. Sem essa regra, `qwen-max` ou qualquer modelo cujo nome termine em `-max`/`-fast` seria
mutilado num par base+tier inexistente. Com ela, a classificacao e auto-verificada pelo catalogo.

O id **nunca** e reescrito: `RouteModel` continua o id completo (`...-high`), porque e a identidade que
o OmniRoute espera no dispatch. O tier vai para metadados, e a variante **herda a capability da linha
base** quando a base esta presente — e isso que impede as 53 variantes de cairem no perfil inerte.

## 4. Patch proposto — `health_models.go`

ANTES (linhas 40-42, literal):

```go
type ModelDocumentMeta struct {
	RegistryVersion string `json:"registry_version,omitempty"`
}
```

DEPOIS:

```go
type ModelDocumentMeta struct {
	RegistryVersion string `json:"registry_version,omitempty"`

	// EffortTier is the reasoning-effort variant parsed from the model id
	// suffix ("high", "medium", "low", "max", "xhigh", "fast"). It is empty for
	// base models. It is derived from the routing id only — never from vendor
	// payload, account data, or OwnedBy.
	EffortTier string `json:"effort_tier,omitempty"`
	// BaseModelID is the sibling base model id an effort variant derives from.
	// Empty unless EffortTier is set AND the base id exists in the same catalog.
	BaseModelID string `json:"base_model_id,omitempty"`
	// CapabilityInherited records that this row's capability fields were copied
	// from BaseModelID instead of being present on the row itself.
	CapabilityInherited bool `json:"capability_inherited,omitempty"`
}
```

Justificativa: aditivo e `omitempty`, portanto nenhum consumidor atual quebra e `buildSnapshot` nem
olha para `Metadata`. `FetchModels` (`health_models.go:44-48`, `ModelsFetchFunc`) **nao muda de
assinatura** — o adaptador continua `func(context.Context) (ModelsDocument, error)`; o enriquecimento
ocorre antes, dentro da projecao.

## 5. Patch proposto — `model_projection.go`

### 5.1 Taxonomia + politica (bloco novo, inserir depois da linha 90)

```go
// EffortTier is a reasoning-effort variant parsed from a model id suffix.
type EffortTier string

// projectionEffortSuffixes is ordered longest-first so "-xhigh" is matched
// before "-high" ("gpt-5-xhigh" must not become base "gpt-5-x" + tier "high").
var projectionEffortSuffixes = []struct {
	suffix string
	tier   EffortTier
}{
	{"-xhigh", "xhigh"},
	{"-medium", "medium"},
	{"-high", "high"},
	{"-fast", "fast"},
	{"-low", "low"},
	{"-max", "max"},
}

// SplitEffortTier reports the base id and effort tier encoded in id. It only
// classifies a suffix as an effort tier when base is present in known — the
// catalog itself is the evidence. Otherwise the id is returned unchanged with
// an empty tier, so a model legitimately named "...-max" is never mutilated.
func SplitEffortTier(id string, known map[string]struct{}) (string, EffortTier) {
	for _, candidate := range projectionEffortSuffixes {
		if !strings.HasSuffix(id, candidate.suffix) {
			continue
		}
		base := strings.TrimSuffix(id, candidate.suffix)
		if base == "" {
			continue
		}
		if _, ok := known[base]; !ok {
			continue
		}
		return base, candidate.tier
	}
	return id, ""
}

// ProjectionPolicy decides selectability. It replaces the hardcoded
// single-route constant so the projection layer stops owning policy.
// Availability remains FAIL-CLOSED: an empty policy makes every row
// available=false, exactly like today.
type ProjectionPolicy struct {
	// SelectableRouteModels is the allowlist of ids that may be available=true.
	SelectableRouteModels map[string]struct{}
	// InheritEffortCapability enables base->variant capability inheritance.
	InheritEffortCapability bool
}

// ProjectionStats is safe, aggregate-only observability: counts, never ids,
// never account identity, never registry content.
type ProjectionStats struct {
	Received       int
	Projected      int
	SkippedInvalid int
	SkippedDup     int
	EffortVariants int
	Inherited      int
	Enriched       int
	Selectable     int
}
```

### 5.2 `ProjectOmniRouteModels` — ANTES (linhas 93-122, literal)

```go
func ProjectOmniRouteModels(native OmniRouteNativeModels, registryVersion string) ModelsDocument {
	version := strings.TrimSpace(registryVersion)
	if version == "" {
		version = projectionRegistryVersionDefault
	}
	rows := make([]ModelDocument, 0, len(native.Data))
	seen := make(map[string]struct{}, len(native.Data))
	for _, m := range native.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		// Skip ids the registry could not parse rather than let one bad id
		// fail-closed-reject the whole snapshot. Skipped ids are simply not
		// selectable, which is the safe outcome.
		if _, err := brain.ParseRouteModel(id); err != nil {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if id == approvedProjectionRouteModel {
			rows = append(rows, approvedProjectionRow(id))
		} else {
			rows = append(rows, unavailableProjectionRow(id))
		}
	}
	return ModelsDocument{Object: "list", RegistryVersion: version, Models: rows}
}
```

### 5.3 DEPOIS (duas passagens; assinatura antiga preservada como wrapper)

```go
// ProjectOmniRouteModels keeps the pre-existing signature and behaviour
// (single approved route, everything else inert) for callers that have not
// migrated. New callers should use ProjectOmniRouteModelsWithPolicy.
func ProjectOmniRouteModels(native OmniRouteNativeModels, registryVersion string) ModelsDocument {
	document, _ := ProjectOmniRouteModelsWithPolicy(native, registryVersion, ProjectionPolicy{
		SelectableRouteModels:   map[string]struct{}{approvedProjectionRouteModel: {}},
		InheritEffortCapability: true,
	})
	return document
}

// ProjectOmniRouteModelsWithPolicy projects the OmniRoute catalog into the
// enriched ModelsDocument the registry requires.
//
// Pass 1 admits ids (parseable, non-empty, de-duplicated) so pass 2 can decide
// effort tiers and inheritance with full catalog context. Two passes are
// required: "is -high an effort tier?" is only answerable once the presence of
// the base id is known.
//
// Invariants preserved from the previous implementation:
//   - one bad row never rejects the catalog (skip, never fail the snapshot);
//   - capability is never inferred from vendor data;
//   - OwnedBy is read and discarded (it can carry account identity);
//   - availability is fail-closed: only ids in policy.SelectableRouteModels
//     may be available=true, and an empty policy yields an all-inert catalog.
func ProjectOmniRouteModelsWithPolicy(native OmniRouteNativeModels, registryVersion string, policy ProjectionPolicy) (ModelsDocument, ProjectionStats) {
	version := strings.TrimSpace(registryVersion)
	if version == "" {
		version = projectionRegistryVersionDefault
	}
	stats := ProjectionStats{Received: len(native.Data)}

	// Pass 1 — admitted id set, in input order.
	admitted := make([]OmniRouteNativeModel, 0, len(native.Data))
	known := make(map[string]struct{}, len(native.Data))
	for _, m := range native.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			stats.SkippedInvalid++
			continue
		}
		if _, err := brain.ParseRouteModel(id); err != nil {
			stats.SkippedInvalid++
			continue
		}
		if _, dup := known[id]; dup {
			stats.SkippedDup++
			continue
		}
		known[id] = struct{}{}
		m.ID = id
		admitted = append(admitted, m)
	}

	// Pass 2 — rows, effort classification, inheritance.
	rows := make([]ModelDocument, 0, len(admitted))
	index := make(map[string]int, len(admitted))
	for _, m := range admitted {
		row := projectRow(m, policy, &stats)
		index[m.ID] = len(rows)
		rows = append(rows, row)
	}
	if policy.InheritEffortCapability {
		for i := range rows {
			base, tier := SplitEffortTier(rows[i].ID, known)
			if tier == "" {
				continue
			}
			stats.EffortVariants++
			rows[i].Metadata.EffortTier = string(tier)
			rows[i].Metadata.BaseModelID = base
			j, ok := index[base]
			if !ok || rows[j].Metadata.CapabilityInherited {
				continue
			}
			// Inherit only capability shape and pool/rotation/affinity from the
			// base row. Availability is NOT inherited: selectability stays a
			// policy decision per id.
			inheritCapability(&rows[i], rows[j])
			stats.Inherited++
		}
	}
	pruneUnknownFallback(rows, known)
	for i := range rows {
		if rows[i].Available != nil && *rows[i].Available {
			stats.Selectable++
		}
	}
	stats.Projected = len(rows)
	return ModelsDocument{Object: "list", RegistryVersion: version, Models: rows}, stats
}

// projectRow preserves an already-enriched row verbatim and falls back to the
// frozen inert profile when the source carries no capability data at all.
func projectRow(m OmniRouteNativeModel, policy ProjectionPolicy, stats *ProjectionStats) ModelDocument {
	selectable := false
	if policy.SelectableRouteModels != nil {
		_, selectable = policy.SelectableRouteModels[m.ID]
	}
	if m.ID == approvedProjectionRouteModel && selectable {
		return approvedProjectionRow(m.ID)
	}
	row := unavailableProjectionRow(m.ID)
	if selectable {
		// A policy-allowed id that is not the frozen approved route stays
		// inert until an enriched source supplies real capability: we refuse to
		// invent capability, so "allowed but unknown" resolves to unavailable.
		row.Available = projectionBoolPtr(false)
	}
	return row
}

// inheritCapability copies capability shape from base into variant. It never
// copies Available and never copies Metadata.RegistryVersion.
func inheritCapability(variant *ModelDocument, base ModelDocument) {
	variant.Protocol = base.Protocol
	variant.Streaming = base.Streaming
	variant.Tools = base.Tools
	variant.Reasoning = base.Reasoning
	variant.StructuredOutput = base.StructuredOutput
	variant.ContextLimit = base.ContextLimit
	variant.AccountPool = base.AccountPool
	variant.Rotation = base.Rotation
	variant.Affinity = base.Affinity
	variant.Metadata.CapabilityInherited = true
}

// pruneUnknownFallback drops fallback ids that were skipped in pass 1.
// registry.go:277-284 rejects the WHOLE snapshot when a fallback target is
// absent, so pruning is what keeps one skipped id from killing the catalog.
func pruneUnknownFallback(rows []ModelDocument, known map[string]struct{}) {
	for i := range rows {
		if len(rows[i].Fallback) == 0 {
			continue
		}
		kept := make([]string, 0, len(rows[i].Fallback))
		for _, target := range rows[i].Fallback {
			if _, ok := known[strings.TrimSpace(target)]; ok {
				kept = append(kept, strings.TrimSpace(target))
			}
		}
		rows[i].Fallback = kept
	}
}
```

Notas de compatibilidade: `approvedProjectionRow`/`unavailableProjectionRow`/`projectionBoolPtr`
(linhas 124-156) ficam intactos; as constantes 45-58 e as variaveis 62-73 ficam intactas.
`stats.Enriched` fica reservado para quando o decode enriquecido por linha entrar (secao 6, item 1) —
proponho manter o campo mesmo em zero para nao mudar a struct depois.

## 6. Fronteiras que NAO sao do meu pacote (nao toquei, nao reivindico)

1. **Decode enriquecido por linha** em `client.go:188-198`: hoje o corpo e decodificado como
   `OmniRouteNativeModels` (so `id/object/owned_by`), entao **capability enriquecida nunca chega a
   projecao**, por mais correta que ela seja. Sem um decode que tente `ModelsDocument` por linha e
   caia para nativo, `Enriched` sera sempre 0 e as 327 linhas continuam inertes. Owner: quem tiver o
   pacote de fetch/cliente.
2. **`ModelSpec` em `registry.go:44-51` nao tem `Metadata`**: o tier morre ao virar snapshot. Para o
   tier chegar ao consumidor precisa de UM campo aditivo (`Effort`, `BaseRouteModel`) em `ModelSpec` +
   copia em `buildSnapshot:260-276`. Owner: pacote do registry.
3. **Mapeamento para o dropdown** (`pkg/agent` `Model`, agrupamento base+tier na UI). Owner: pacote de UI/adapters.
4. Politica de quais ids sao selecionaveis: e decisao do owner/Codex56-TL, nao da projecao. Eu so
   troquei constante por parametro; o default continua o fail-closed atual.

## 7. Testes que eu proponho junto do patch (nao escritos ainda)

- `-xhigh` classifica como `xhigh` e nao como `high` (ordem longest-first).
- id terminando em `-max` **sem** base no catalogo permanece intacto e sem tier.
- variante herda `Tools/Reasoning/ContextLimit/Protocol` da base e **nao** herda `Available`.
- fallback apontando para id descartado e podado, e `buildSnapshot` aceita o documento.
- 327 linhas sinteticas (53 variantes) => `Projected=327`, `EffortVariants=53`, `Selectable<=1`,
  e `buildSnapshot` retorna snapshot valido (prova de que `MaxRegistryModels=1024` nao e limite).
- politica vazia => zero linhas `available=true` (fail-closed preservado).

## 8. Nao-alegacoes

- Nao editei, nao compilei, nao rodei `go vet`, nao troquei binario, nao reiniciei daemon, nao commitei.
- **Nao inspecionei o payload real dos 327 modelos**: `/v1/models` do OmniRoute e auth-gated (401 sem
  chave) e obter a chave para dentro de contexto/argv e proibido. Portanto NAO provei se o gateway
  serve schema nativo ou enriquecido, nem confirmei empiricamente os 53 sufixos — o desenho trata os
  dois casos e a regra de sufixo se auto-verifica pelo catalogo.
- Nao provei que `/tmp/multica-auth-fixed` (build 24/07) corresponde a este codigo.
- Nao toquei em arquivo de outro pacote (`client.go`, `registry.go`, `pkg/agent`).
