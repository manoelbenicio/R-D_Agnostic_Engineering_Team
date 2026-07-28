# GW discovery — pacote 4: allowlist versionada

## Veredito

`Registry.LookupModel` deve deixar de interpretar `snapshot.Models` como autorização. O snapshot do OmniRoute é somente disponibilidade/capacidade observada. A autorização é uma política local, imutável durante o processo e compilada no binário. Um modelo só pode ser exposto ou usado quando:

1. a revisão do registry do gateway é exatamente a revisão aprovada;
2. a entrada existe na allowlist e está em estado `approved`;
3. o mesmo ID existe no snapshot atual e `Available == true`;
4. toda rota de fallback referenciada também está aprovada, presente e disponível.

Qualquer falha retorna fechado. A política contém somente IDs de rota, estado e referência da decisão do owner; nunca contém segredo, identidade de conta, pool, afinidade ou credencial.

## Estado atual — trecho literal

Arquivo: `multica-auth-work/server/internal/daemon/gateway/registry.go`.

Linhas 68–87 atuais:

```go
type Registry struct {
	fetcher ModelsFetcher
	ttl     time.Duration
	now     func() time.Time

	mu         sync.Mutex
	snapshot   RegistrySnapshot
	expiresAt  time.Time
	refreshing chan struct{}
	refreshErr error
	retryAt    time.Time
	generation uint64
}

func NewRegistry(fetcher ModelsFetcher, ttl time.Duration) (*Registry, error) {
	if fetcher == nil || ttl < time.Second || ttl > 24*time.Hour {
		return nil, &GatewayError{Operation: "registry", Class: ErrorInvalidConfiguration}
	}
	return &Registry{fetcher: fetcher, ttl: ttl, now: time.Now}, nil
}
```

Linhas 176–190 atuais:

```go
func (r *Registry) LookupModel(ctx context.Context, model brain.RouteModel) (ModelSpec, error) {
	parsed, err := brain.ParseRouteModel(string(model))
	if err != nil {
		return ModelSpec{}, &GatewayError{Operation: "registry.lookup", Class: ErrorInvalidRequest}
	}
	snapshot, err := r.Snapshot(ctx)
	if err != nil {
		return ModelSpec{}, err
	}
	spec, ok := snapshot.Models[parsed]
	if !ok || !spec.Available {
		return ModelSpec{}, &GatewayError{Operation: "registry.lookup", Class: ErrorUnknownModel}
	}
	return cloneModelSpec(spec), nil
}
```

Problema literal: a única condição de autorização é `snapshot.Models[parsed]` com `Available == true`. Portanto qualquer um dos 327 IDs que o gateway marque disponível pode ser declarado pelo daemon.

## Persistência e versionamento propostos

Novo artefato versionado e revisado como código:

`multica-auth-work/server/internal/daemon/gateway/model_allowlist.v1.json`

Conteúdo inicial seguro, deliberadamente deny-all até decisão escrita do owner:

```json
{
  "schema": "multica.gateway-model-allowlist.v1",
  "policy_revision": "bootstrap-deny-all-v1",
  "gateway_registry_version": "unapproved",
  "entries": []
}
```

Formato após uma decisão real do owner:

```json
{
  "schema": "multica.gateway-model-allowlist.v1",
  "policy_revision": "owner-20260726.1",
  "gateway_registry_version": "<valor exato de X-OmniRoute-Registry-Version aprovado>",
  "entries": [
    {
      "id": "<route-id exato aprovado>",
      "state": "approved",
      "decision_ref": "<referência escrita do owner>"
    },
    {
      "id": "<route-id explicitamente revogado>",
      "state": "revoked",
      "decision_ref": "<referência escrita do owner>"
    }
  ]
}
```

`policy_revision` versiona a decisão Multica. `gateway_registry_version` fixa a revisão externa contra a qual a decisão foi tomada. O JSON entra no Git, passa por review e é embutido via `go:embed`; não existe endpoint de mutação, variável de ambiente ou arquivo operacional que permita ampliar a allowlist depois do build.

## Patch proposto — depois

Âncora: adicionar imports e tipos antes do `Registry` atual, substituir `Registry`/`NewRegistry` nas linhas atuais 68–87 e substituir `LookupModel` nas linhas atuais 176–190.

```go
import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const modelAllowlistSchema = "multica.gateway-model-allowlist.v1"

type ModelApprovalState string

const (
	ModelApprovalApproved ModelApprovalState = "approved"
	ModelApprovalRevoked  ModelApprovalState = "revoked"
)

type modelAllowlistEntryDocument struct {
	ID          string             `json:"id"`
	State       ModelApprovalState `json:"state"`
	DecisionRef string             `json:"decision_ref"`
}

type modelAllowlistDocument struct {
	Schema                 string                        `json:"schema"`
	PolicyRevision         string                        `json:"policy_revision"`
	GatewayRegistryVersion string                        `json:"gateway_registry_version"`
	Entries                []modelAllowlistEntryDocument `json:"entries"`
}

type modelAllowlist struct {
	policyRevision         string
	gatewayRegistryVersion string
	states                 map[brain.RouteModel]ModelApprovalState
}

//go:embed model_allowlist.v1.json
var embeddedModelAllowlist []byte

func validPolicyToken(value string) bool {
	return value != "" &&
		value == strings.TrimSpace(value) &&
		len(value) <= 128 &&
		!strings.ContainsAny(value, "\r\n\x00")
}

func parseModelAllowlist(raw []byte) (modelAllowlist, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document modelAllowlistDocument
	if err := decoder.Decode(&document); err != nil {
		return modelAllowlist{}, &GatewayError{
			Operation: "registry.allowlist", Class: ErrorInvalidConfiguration,
			Detail: "malformed_policy",
		}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return modelAllowlist{}, &GatewayError{
			Operation: "registry.allowlist", Class: ErrorInvalidConfiguration,
			Detail: "trailing_policy_data",
		}
	}
	if document.Schema != modelAllowlistSchema ||
		!validPolicyToken(document.PolicyRevision) ||
		!validPolicyToken(document.GatewayRegistryVersion) ||
		len(document.Entries) > MaxRegistryModels {
		return modelAllowlist{}, &GatewayError{
			Operation: "registry.allowlist", Class: ErrorInvalidConfiguration,
			Detail: "invalid_policy_header",
		}
	}
	states := make(map[brain.RouteModel]ModelApprovalState, len(document.Entries))
	for _, entry := range document.Entries {
		model, err := brain.ParseRouteModel(entry.ID)
		if err != nil || !validPolicyToken(entry.DecisionRef) {
			return modelAllowlist{}, &GatewayError{
				Operation: "registry.allowlist", Class: ErrorInvalidConfiguration,
				Detail: "invalid_policy_entry",
			}
		}
		if entry.State != ModelApprovalApproved && entry.State != ModelApprovalRevoked {
			return modelAllowlist{}, &GatewayError{
				Operation: "registry.allowlist", Class: ErrorInvalidConfiguration,
				Detail: "invalid_policy_state",
			}
		}
		if _, duplicate := states[model]; duplicate {
			return modelAllowlist{}, &GatewayError{
				Operation: "registry.allowlist", Class: ErrorInvalidConfiguration,
				Detail: "duplicate_policy_model",
			}
		}
		states[model] = entry.State
	}
	return modelAllowlist{
		policyRevision:         document.PolicyRevision,
		gatewayRegistryVersion: document.GatewayRegistryVersion,
		states:                 states,
	}, nil
}

type ApprovedRegistrySnapshot struct {
	PolicyRevision  string
	RegistryVersion string
	FetchedAt       time.Time
	Models          map[brain.RouteModel]ModelSpec
}

type Registry struct {
	fetcher   ModelsFetcher
	ttl       time.Duration
	now       func() time.Time
	allowlist modelAllowlist

	mu         sync.Mutex
	snapshot   RegistrySnapshot
	expiresAt  time.Time
	refreshing chan struct{}
	refreshErr error
	retryAt    time.Time
	generation uint64
}

func NewRegistry(fetcher ModelsFetcher, ttl time.Duration) (*Registry, error) {
	allowlist, err := parseModelAllowlist(embeddedModelAllowlist)
	if err != nil {
		return nil, err
	}
	return newRegistryWithAllowlist(fetcher, ttl, allowlist)
}

// Test-only seam. It is unexported so production composition cannot inject an
// ad-hoc allowlist or bypass the owner-approved embedded policy.
func newRegistryWithAllowlist(fetcher ModelsFetcher, ttl time.Duration, allowlist modelAllowlist) (*Registry, error) {
	if fetcher == nil || ttl < time.Second || ttl > 24*time.Hour ||
		!validPolicyToken(allowlist.policyRevision) ||
		!validPolicyToken(allowlist.gatewayRegistryVersion) ||
		allowlist.states == nil {
		return nil, &GatewayError{Operation: "registry", Class: ErrorInvalidConfiguration}
	}
	return &Registry{
		fetcher: fetcher, ttl: ttl, now: time.Now, allowlist: allowlist,
	}, nil
}

func (r *Registry) ApprovedSnapshot(ctx context.Context) (ApprovedRegistrySnapshot, error) {
	snapshot, err := r.Snapshot(ctx)
	if err != nil {
		return ApprovedRegistrySnapshot{}, err
	}
	if snapshot.Version != r.allowlist.gatewayRegistryVersion {
		return ApprovedRegistrySnapshot{}, &GatewayError{
			Operation: "registry.allowlist", Class: ErrorProtocol,
			Detail: "registry_revision_not_approved",
		}
	}

	approved := make(map[brain.RouteModel]ModelSpec)
	for model, state := range r.allowlist.states {
		if state != ModelApprovalApproved {
			continue
		}
		spec, present := snapshot.Models[model]
		if !present || !spec.Available {
			continue
		}
		approved[model] = cloneModelSpec(spec)
	}

	// A fallback edge is another executable route. It cannot bypass the same
	// owner approval, presence and availability checks as the primary route.
	for _, spec := range approved {
		for _, fallback := range spec.Fallback {
			if _, ok := approved[fallback]; !ok {
				return ApprovedRegistrySnapshot{}, &GatewayError{
					Operation: "registry.allowlist", Class: ErrorCapability,
					Detail: "fallback_not_approved",
				}
			}
		}
	}
	return ApprovedRegistrySnapshot{
		PolicyRevision:  r.allowlist.policyRevision,
		RegistryVersion: snapshot.Version,
		FetchedAt:       snapshot.FetchedAt,
		Models:          approved,
	}, nil
}

func (r *Registry) LookupModel(ctx context.Context, model brain.RouteModel) (ModelSpec, error) {
	parsed, err := brain.ParseRouteModel(string(model))
	if err != nil {
		return ModelSpec{}, &GatewayError{Operation: "registry.lookup", Class: ErrorInvalidRequest}
	}
	approved, err := r.ApprovedSnapshot(ctx)
	if err != nil {
		return ModelSpec{}, err
	}
	spec, ok := approved.Models[parsed]
	if !ok {
		// Missing, unavailable, absent from policy and explicitly revoked are
		// intentionally indistinguishable to callers.
		return ModelSpec{}, &GatewayError{
			Operation: "registry.lookup", Class: ErrorUnknownModel,
			Detail: "model_not_approved_or_available",
		}
	}
	return cloneModelSpec(spec), nil
}
```

O método de descoberta que alimentará a UI deve consumir exclusivamente `ApprovedSnapshot.Models`. Consumir `Registry.Snapshot().Models` diretamente continuaria contornando a política. O `ProjectRouteModels` atual usa o snapshot cru em `projection.go:65-72`; esse call site precisa ser trocado pelo owner do pacote de projeção no split, não por P4.

## Aprovação e revogação pelo owner

### Aprovar

1. Registrar decisão escrita com o route ID exato e a revisão exata do registry do OmniRoute.
2. Alterar somente `model_allowlist.v1.json`.
3. Incrementar `policy_revision`.
4. Fixar `gateway_registry_version` na revisão observada e aprovada.
5. Inserir/alterar a entrada para `state: "approved"` com `decision_ref`.
6. Rodar validações e review; somente um build posterior incorpora a decisão.

### Revogar

1. Registrar decisão escrita com o route ID exato.
2. Incrementar `policy_revision`.
3. Manter a entrada para auditoria, alterando `state` para `"revoked"` e atualizando `decision_ref`.
4. Após o build autorizado, `LookupModel` retorna `unknown_model` mesmo que o gateway continue anunciando `available=true`.

Não se propõe endpoint admin de mutação nesta fase: uma API runtime permitiria ampliar a superfície sem que a revisão ficasse presa ao binário aprovado. Git + decisão escrita + build autorizado são o controle de mudança.

## Modelo removido do gateway

- Se o gateway cumprir o contrato e incrementar `RegistryVersion`, `ApprovedSnapshot` rejeita a revisão inteira com `registry_revision_not_approved`. Nenhum modelo da revisão nova é exposto até o owner revalidar a política.
- Se o ID aprovado estiver ausente ou `available=false` no snapshot da mesma revisão, ele não entra na interseção e `LookupModel` retorna `unknown_model`.
- O registry de produção usa TTL de 1 segundo em `brain_integration.go:334-352`; portanto o snapshot expirado não é reutilizado e a remoção é observada no próximo refresh, no máximo após essa janela.
- Se um modelo removido reaparecer sob outra revisão, a divergência de `gateway_registry_version` mantém o fail-closed. Não existe “ressurreição” automática.
- Se o gateway alterar o catálogo sem incrementar a revisão, isso viola o contrato de imutabilidade. A ausência ainda é negada; o split deve adicionar um teste de conformance exigindo mudança de revisão para alteração de catálogo.

## Testes exigidos para assinatura do split

1. `approved + present + available + matching registry version` retorna o `ModelSpec`.
2. `present + available`, mas ausente da allowlist, retorna `ErrorUnknownModel`.
3. `revoked + present + available` retorna `ErrorUnknownModel`.
4. `approved`, mas removido ou `available=false`, retorna `ErrorUnknownModel` após TTL/Invalidate.
5. revisão do gateway diferente da revisão aprovada retorna `ErrorProtocol` e conjunto zero.
6. fallback não aprovado/presente/disponível falha fechado.
7. schema, revisão, estado, decision ref ou ID inválido; chave desconhecida; duplicata e dados após o JSON rejeitam a construção.
8. deny-all vazio é válido e expõe zero modelos.
9. `ApprovedSnapshot` devolve cópia imutável e somente a interseção aprovada.
10. o bridge da UI usa `ApprovedSnapshot`, nunca `Snapshot` cru.

## Justificativa

O registry do gateway responde “o que existe”; o owner responde “o que Multica autoriza”. Separar esses fatos impede que a adição de um 328º modelo no OmniRoute se torne automaticamente uma permissão no daemon. A política compilada não conhece conta nem credencial, preserva o gateway como único dono de seleção/rotação/custo e permite rollback junto com o próprio binário.

## Ações nesta rodada

Somente proposta e evidência. Nenhum arquivo de código editado, nenhum build, deploy, commit, push ou restart.
