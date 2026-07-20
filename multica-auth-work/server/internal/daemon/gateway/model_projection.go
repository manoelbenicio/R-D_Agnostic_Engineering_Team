package gateway

import (
	"os"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// envDevModelsCompat gates the DEV-only OmniRoute native /v1/models
// compatibility projection. It is OFF by default; the enriched-schema fetch
// path is unchanged unless OMNIROUTE_DEV_MODELS_COMPAT=1.
const envDevModelsCompat = "OMNIROUTE_DEV_MODELS_COMPAT"

// devModelsCompatEnabled reports whether the DEV compatibility projection is
// enabled for Client.FetchModels.
func devModelsCompatEnabled() bool {
	return strings.TrimSpace(os.Getenv(envDevModelsCompat)) == "1"
}

// DEV-only compatibility projection.
//
// OmniRoute's native GET /v1/models returns an OpenAI-basic shape
// ({object,data:[{id,object,owned_by}]}) that lacks the enriched fields the
// gateway model registry requires (protocol, streaming/tools/reasoning/
// structured_output, context_limit, account_pool, rotation, affinity,
// available). Because Registry.buildSnapshot validates EVERY row and
// fail-closed-rejects the ENTIRE snapshot as ErrorProtocol if any row is
// missing a required field, the raw OmniRoute catalog makes no route
// admissible.
//
// ProjectOmniRouteModels maps each native row into a schema-valid ModelDocument
// so the snapshot parses, while PRESERVING gateway-required fail-closed
// semantics: every projected row is available=false (present but not selectable
// — LookupModel returns ErrorUnknownModel) EXCEPT the single approved route
// claude_code_kimi_2.7_Code, which is enriched with frozen Anthropic/Claude
// profile values and available=true.
//
// Capabilities, account_pool, rotation and affinity are NEVER inferred from
// OmniRoute data; they are the frozen constants below. This layer is intended
// to be wired only in DEV compatibility mode (it transforms the raw /v1/models
// response before Registry.Snapshot parses it) and is fully reversible by
// removing the projection from the models-fetch path.
const (
	// approvedProjectionRouteModel is the sole route allowed to be selectable.
	approvedProjectionRouteModel = "claude_code_kimi_2.7_Code"

	// Frozen approved-row profile (Claude Code / Anthropic Messages).
	approvedProjectionContextLimit = 200000

	// projectionAccountPool is a frozen, non-inferred pool label.
	projectionAccountPool = "default"

	// projectionRegistryVersionDefault is used when the caller supplies no
	// version (OmniRoute's native /v1/models carries none).
	projectionRegistryVersionDefault = "omniroute-dev-compat-v1"
)

// Frozen enum wire values. The registry only accepts specific rotation /
// affinity / protocol strings; the reviewer's shorthand ("round-robin",
// "continuation") maps to these canonical values or buildSnapshot rejects them.
var (
	projectionRotation = string(RotationStrictIndependentRequest) // strict independent-request round-robin
	projectionAffinity = string(AffinityOriginAccount)            // continuation affinity (origin account)

	approvedProjectionProtocol = string(brain.ProtocolAnthropicMessages)
	// Non-approved rows carry a registry-valid placeholder protocol. It is
	// inert because the row is available=false, but must be a recognized family
	// so buildSnapshot accepts the row instead of rejecting the whole snapshot.
	unavailableProjectionProtocol = string(brain.ProtocolOpenAIChat)
)

// OmniRouteNativeModels is the OpenAI-basic response OmniRoute serves at
// GET /v1/models.
type OmniRouteNativeModels struct {
	Object string                 `json:"object"`
	Data   []OmniRouteNativeModel `json:"data"`
}

// OmniRouteNativeModel is one native model row. Only ID is consumed; Object and
// OwnedBy are accepted for completeness and deliberately NOT used to infer any
// capability.
type OmniRouteNativeModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// ProjectOmniRouteModels transforms a native OmniRoute /v1/models response into
// the enriched ModelsDocument the registry requires. registryVersion is the
// version to stamp (e.g. from the X-OmniRoute-Registry-Version header); a
// frozen default is used when empty.
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

func approvedProjectionRow(id string) ModelDocument {
	return ModelDocument{
		ID:               id,
		Protocol:         approvedProjectionProtocol,
		Streaming:        projectionBoolPtr(true),
		Tools:            projectionBoolPtr(true),
		Reasoning:        projectionBoolPtr(false),
		StructuredOutput: projectionBoolPtr(false),
		ContextLimit:     approvedProjectionContextLimit,
		AccountPool:      projectionAccountPool,
		Rotation:         projectionRotation,
		Affinity:         projectionAffinity,
		Available:        projectionBoolPtr(true),
	}
}

func unavailableProjectionRow(id string) ModelDocument {
	return ModelDocument{
		ID:               id,
		Protocol:         unavailableProjectionProtocol,
		Streaming:        projectionBoolPtr(false),
		Tools:            projectionBoolPtr(false),
		Reasoning:        projectionBoolPtr(false),
		StructuredOutput: projectionBoolPtr(false),
		ContextLimit:     1, // must be > 0 to satisfy buildSnapshot; inert while unavailable
		AccountPool:      projectionAccountPool,
		Rotation:         projectionRotation,
		Affinity:         projectionAffinity,
		Available:        projectionBoolPtr(false),
	}
}

func projectionBoolPtr(b bool) *bool { return &b }
