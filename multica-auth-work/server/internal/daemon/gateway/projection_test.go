package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestProjectRouteModelsPreservesExactIDsSortsAndExcludesUnavailable(t *testing.T) {
	snapshot := syntheticProjectionSnapshot(
		projectionSpec("agy/z-route", brain.ProtocolAnthropicMessages, true),
		projectionSpec("agy/a-route", brain.ProtocolAnthropicMessages, true),
		projectionSpec("agy/unavailable", brain.ProtocolAnthropicMessages, false),
		projectionSpec("openai/responses-route", brain.ProtocolOpenAIResponses, true),
		projectionSpec("kimi/chat-route", brain.ProtocolOpenAIChat, true),
		projectionSpec("agy/native-route", brain.ProtocolAntigravity, true),
	)
	projection, err := ProjectSnapshotRouteModels(snapshot, brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
	if err != nil {
		t.Fatalf("ProjectSnapshotRouteModels: %v", err)
	}
	want := []brain.RouteModel{"agy/a-route", "agy/z-route"}
	if projection.RegistryVersion != snapshot.Version || len(projection.Models) != len(want) {
		t.Fatalf("projection=%+v", projection)
	}
	for index, model := range projection.Models {
		if model.ID != want[index] {
			t.Fatalf("model[%d].ID=%q, want exact %q", index, model.ID, want[index])
		}
		if model.DisplayNamespace != RouteDisplayClaudeCode || model.CLIKind != brain.CLIClaudeCode || model.Protocol != brain.ProtocolAnthropicMessages || model.TrustedProfile != ProfileAnthropicMessages {
			t.Fatalf("model[%d] metadata=%+v", index, model)
		}
	}
}

func TestProjectRouteModelsSeparatesDisplayNamespaceFromRoutePrefix(t *testing.T) {
	tests := []struct {
		name      string
		cli       brain.CLIKind
		spec      ModelSpec
		namespace RouteDisplayNamespace
		profile   ProfileID
	}{
		{
			name: "Kimi-prefixed route through accepted Claude protocol", cli: brain.CLIClaudeCode,
			spec:      projectionSpec("kimi/compatible-via-messages", brain.ProtocolAnthropicMessages, true),
			namespace: RouteDisplayClaudeCode, profile: ProfileAnthropicMessages,
		},
		{
			name: "NVIDIA-prefixed route through accepted Codex protocol", cli: brain.CLICodex,
			spec:      projectionSpec("nvidia/z-ai/glm-5.2", brain.ProtocolOpenAIResponses, true),
			namespace: RouteDisplayCodex, profile: ProfileOpenAIResponses,
		},
		{
			name: "Antigravity-prefixed route through accepted Claude protocol", cli: brain.CLIClaudeCode,
			spec:      projectionSpec("agy/claude-opus-4-6-thinking", brain.ProtocolAnthropicMessages, true),
			namespace: RouteDisplayClaudeCode, profile: ProfileAnthropicMessages,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			projection, err := ProjectSnapshotRouteModels(
				syntheticProjectionSnapshot(test.spec), test.cli, acceptedCredentiallessProjectionAdapter,
			)
			if err != nil {
				t.Fatalf("ProjectSnapshotRouteModels: %v", err)
			}
			if len(projection.Models) != 1 {
				t.Fatalf("models=%+v", projection.Models)
			}
			model := projection.Models[0]
			if model.ID != test.spec.Capability.RouteModel || model.DisplayNamespace != test.namespace || model.TrustedProfile != test.profile {
				t.Fatalf("projected model=%+v", model)
			}
		})
	}
}

func TestProjectRouteModelsRejectsUnsupportedNativeFrontends(t *testing.T) {
	snapshot := syntheticProjectionSnapshot(projectionSpec("synthetic/chat", brain.ProtocolOpenAIChat, true))
	for _, cli := range []brain.CLIKind{
		brain.CLIKimi,
		brain.CLIOpenAICompatible,
		brain.CLINIM,
		brain.CLIAntigravity,
	} {
		t.Run(string(cli), func(t *testing.T) {
			var filterCalls atomic.Int64
			projection, err := ProjectSnapshotRouteModels(snapshot, cli, func(brain.CLIKind, brain.ProtocolFamily) bool {
				filterCalls.Add(1)
				return true
			})
			if !IsErrorClass(err, ErrorCapability) || !isZeroProjection(projection) {
				t.Fatalf("projection=(%+v, %v)", projection, err)
			}
			if filterCalls.Load() != 0 {
				t.Fatalf("unsupported frontend reached adapter filter %d times", filterCalls.Load())
			}
		})
	}
}

func TestProjectRouteModelsRequiresTrustedProfileAndAcceptedAdapter(t *testing.T) {
	claudeSnapshot := syntheticProjectionSnapshot(projectionSpec("synthetic/claude", brain.ProtocolAnthropicMessages, true))
	if projection, err := ProjectSnapshotRouteModels(claudeSnapshot, brain.CLIClaudeCode, nil); !IsErrorClass(err, ErrorInvalidConfiguration) || !isZeroProjection(projection) {
		t.Fatalf("nil filter projection=(%+v, %v)", projection, err)
	}
	if projection, err := ProjectSnapshotRouteModels(claudeSnapshot, brain.CLIClaudeCode, func(brain.CLIKind, brain.ProtocolFamily) bool { return false }); !IsErrorClass(err, ErrorCapability) || !isZeroProjection(projection) {
		t.Fatalf("unaccepted adapter projection=(%+v, %v)", projection, err)
	}

	// Codex has no trusted Chat profile, so even an over-permissive filter
	// cannot expose the route.
	chatSnapshot := syntheticProjectionSnapshot(projectionSpec("synthetic/chat", brain.ProtocolOpenAIChat, true))
	var filterCalls atomic.Int64
	projection, err := ProjectSnapshotRouteModels(chatSnapshot, brain.CLICodex, func(brain.CLIKind, brain.ProtocolFamily) bool {
		filterCalls.Add(1)
		return true
	})
	if !IsErrorClass(err, ErrorCapability) || !isZeroProjection(projection) {
		t.Fatalf("untrusted profile projection=(%+v, %v)", projection, err)
	}
	if filterCalls.Load() != 0 {
		t.Fatalf("profile-incompatible protocol reached filter %d times", filterCalls.Load())
	}
}

func TestProjectRouteModelsRejectsNativeProtocolsForClaudeAndCodex(t *testing.T) {
	for _, test := range []struct {
		name     string
		cli      brain.CLIKind
		protocol brain.ProtocolFamily
	}{
		{name: "Claude direct Antigravity", cli: brain.CLIClaudeCode, protocol: brain.ProtocolAntigravity},
		{name: "Claude OpenAI Chat", cli: brain.CLIClaudeCode, protocol: brain.ProtocolOpenAIChat},
		{name: "Codex direct Antigravity", cli: brain.CLICodex, protocol: brain.ProtocolAntigravity},
		{name: "Codex OpenAI Chat", cli: brain.CLICodex, protocol: brain.ProtocolOpenAIChat},
	} {
		t.Run(test.name, func(t *testing.T) {
			projection, err := ProjectSnapshotRouteModels(
				syntheticProjectionSnapshot(projectionSpec("synthetic/route", test.protocol, true)),
				test.cli,
				func(brain.CLIKind, brain.ProtocolFamily) bool { return true },
			)
			if !IsErrorClass(err, ErrorCapability) || !isZeroProjection(projection) {
				t.Fatalf("projection=(%+v, %v)", projection, err)
			}
		})
	}
}

func TestProjectRouteModelsFailsClosedOnMalformedEmptyAndNoCompatibleSnapshots(t *testing.T) {
	validSpec := projectionSpec("synthetic/model", brain.ProtocolAnthropicMessages, true)
	tests := []struct {
		name      string
		snapshot  RegistrySnapshot
		wantClass ErrorClass
	}{
		{name: "empty snapshot", snapshot: RegistrySnapshot{}, wantClass: ErrorProtocol},
		{name: "empty models", snapshot: RegistrySnapshot{Version: "synthetic-v1", Models: map[brain.RouteModel]ModelSpec{}}, wantClass: ErrorProtocol},
		{name: "malformed version", snapshot: RegistrySnapshot{Version: " synthetic-v1 ", Models: map[brain.RouteModel]ModelSpec{validSpec.Capability.RouteModel: validSpec}}, wantClass: ErrorProtocol},
		{
			name: "route key mismatch",
			snapshot: RegistrySnapshot{Version: "synthetic-v1", Models: map[brain.RouteModel]ModelSpec{
				"synthetic/other": validSpec,
			}},
			wantClass: ErrorProtocol,
		},
		{
			name: "invalid context",
			snapshot: func() RegistrySnapshot {
				spec := validSpec
				spec.Capability.ContextLimit = 0
				return RegistrySnapshot{Version: "synthetic-v1", Models: map[brain.RouteModel]ModelSpec{spec.Capability.RouteModel: spec}}
			}(),
			wantClass: ErrorProtocol,
		},
		{
			name: "missing account pool",
			snapshot: func() RegistrySnapshot {
				spec := validSpec
				spec.AccountPool = ""
				return RegistrySnapshot{Version: "synthetic-v1", Models: map[brain.RouteModel]ModelSpec{spec.Capability.RouteModel: spec}}
			}(),
			wantClass: ErrorProtocol,
		},
		{
			name: "no available compatible models",
			snapshot: syntheticProjectionSnapshot(
				projectionSpec("synthetic/unavailable", brain.ProtocolAnthropicMessages, false),
				projectionSpec("synthetic/responses", brain.ProtocolOpenAIResponses, true),
			),
			wantClass: ErrorCapability,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			projection, err := ProjectSnapshotRouteModels(test.snapshot, brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
			if !IsErrorClass(err, test.wantClass) || !isZeroProjection(projection) {
				t.Fatalf("projection=(%+v, %v), want class %s", projection, err, test.wantClass)
			}
		})
	}
}

func TestRegistryProjectRouteModelsPreservesRegistryErrorsAndCancellation(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		projection, err := registry.ProjectRouteModels(context.Background(), brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
		if !IsErrorClass(err, ErrorInvalidConfiguration) || !isZeroProjection(projection) {
			t.Fatalf("projection=(%+v, %v)", projection, err)
		}
	})

	t.Run("registry error", func(t *testing.T) {
		expected := &GatewayError{Operation: operationModels, Class: ErrorAuthentication}
		var fetches atomic.Int64
		registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
			fetches.Add(1)
			return ModelsDocument{}, expected
		}), time.Minute)
		if err != nil {
			t.Fatalf("NewRegistry: %v", err)
		}
		projection, err := registry.ProjectRouteModels(context.Background(), brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
		if !errors.Is(err, expected) || !isZeroProjection(projection) {
			t.Fatalf("projection=(%+v, %v)", projection, err)
		}
		if fetches.Load() != 1 {
			t.Fatalf("registry fetched %d times", fetches.Load())
		}
	})

	t.Run("empty registry document", func(t *testing.T) {
		registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
			return ModelsDocument{RegistryVersion: "synthetic-empty-v1"}, nil
		}), time.Minute)
		if err != nil {
			t.Fatalf("NewRegistry: %v", err)
		}
		projection, err := registry.ProjectRouteModels(context.Background(), brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
		if !IsErrorClass(err, ErrorProtocol) || !isZeroProjection(projection) {
			t.Fatalf("projection=(%+v, %v)", projection, err)
		}
	})

	t.Run("no compatible registry model", func(t *testing.T) {
		registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
			return projectionModelsDocument("synthetic/chat", brain.ProtocolOpenAIChat, true), nil
		}), time.Minute)
		if err != nil {
			t.Fatalf("NewRegistry: %v", err)
		}
		projection, err := registry.ProjectRouteModels(context.Background(), brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
		if !IsErrorClass(err, ErrorCapability) || !isZeroProjection(projection) {
			t.Fatalf("projection=(%+v, %v)", projection, err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		registry, err := NewRegistry(ModelsFetchFunc(func(ctx context.Context) (ModelsDocument, error) {
			<-ctx.Done()
			return ModelsDocument{}, classifyContextError(operationModels, ctx)
		}), time.Minute)
		if err != nil {
			t.Fatalf("NewRegistry: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		projection, err := registry.ProjectRouteModels(ctx, brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter)
		if !IsErrorClass(err, ErrorCancelled) || !isZeroProjection(projection) {
			t.Fatalf("projection=(%+v, %v)", projection, err)
		}
	})
}

func TestRegistryProjectRouteModelsRejectsCancelledWarmCacheWithoutSideEffects(t *testing.T) {
	tests := []struct {
		name      string
		context   func() (context.Context, context.CancelFunc)
		wantClass ErrorClass
	}{
		{
			name: "cancelled",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantClass: ErrorCancelled,
		},
		{
			name: "deadline exceeded",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Unix(1, 0))
			},
			wantClass: ErrorDeadlineExceeded,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fetches atomic.Int64
			registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
				fetches.Add(1)
				return projectionModelsDocument("agy/claude-opus-4-6-thinking", brain.ProtocolAnthropicMessages, true), nil
			}), time.Minute)
			if err != nil {
				t.Fatalf("NewRegistry: %v", err)
			}
			if _, err := registry.ProjectRouteModels(context.Background(), brain.CLIClaudeCode, acceptedCredentiallessProjectionAdapter); err != nil {
				t.Fatalf("prime projection: %v", err)
			}
			if fetches.Load() != 1 {
				t.Fatalf("prime fetched %d times", fetches.Load())
			}

			ctx, cancel := test.context()
			defer cancel()
			var filterCalls atomic.Int64
			projection, err := registry.ProjectRouteModels(ctx, brain.CLIClaudeCode, func(brain.CLIKind, brain.ProtocolFamily) bool {
				filterCalls.Add(1)
				return true
			})
			if !IsErrorClass(err, test.wantClass) || !isZeroProjection(projection) {
				t.Fatalf("projection=(%+v, %v), want class %s", projection, err, test.wantClass)
			}
			if fetches.Load() != 1 {
				t.Fatalf("cancelled warm-cache call fetched %d times", fetches.Load())
			}
			if filterCalls.Load() != 0 {
				t.Fatalf("cancelled warm-cache call reached adapter filter %d times", filterCalls.Load())
			}
		})
	}
}

func TestRegistryProjectRouteModelsRejectsCancellationAfterSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var fetches atomic.Int64
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		fetches.Add(1)
		cancel()
		return projectionModelsDocument("agy/claude-opus-4-6-thinking", brain.ProtocolAnthropicMessages, true), nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	var filterCalls atomic.Int64
	projection, err := registry.ProjectRouteModels(ctx, brain.CLIClaudeCode, func(brain.CLIKind, brain.ProtocolFamily) bool {
		filterCalls.Add(1)
		return true
	})
	if !IsErrorClass(err, ErrorCancelled) || !isZeroProjection(projection) {
		t.Fatalf("projection=(%+v, %v)", projection, err)
	}
	if fetches.Load() != 1 {
		t.Fatalf("snapshot fetched %d times", fetches.Load())
	}
	if filterCalls.Load() != 0 {
		t.Fatalf("post-snapshot cancellation reached adapter filter %d times", filterCalls.Load())
	}
}

func TestProjectRouteModelsIsDeterministicAcrossMapInsertionOrder(t *testing.T) {
	specs := []ModelSpec{
		projectionSpec("synthetic/d", brain.ProtocolOpenAIResponses, true),
		projectionSpec("synthetic/b", brain.ProtocolOpenAIResponses, true),
		projectionSpec("synthetic/a", brain.ProtocolOpenAIResponses, true),
		projectionSpec("synthetic/c", brain.ProtocolOpenAIResponses, true),
	}
	orders := [][]int{
		{0, 1, 2, 3},
		{3, 2, 1, 0},
		{1, 3, 0, 2},
		{2, 0, 3, 1},
	}
	var baseline RouteModelProjection
	for index, order := range orders {
		snapshot := RegistrySnapshot{Version: "synthetic-projection-v1", Models: make(map[brain.RouteModel]ModelSpec, len(specs))}
		for _, position := range order {
			spec := specs[position]
			snapshot.Models[spec.Capability.RouteModel] = spec
		}
		var filterCalls atomic.Int64
		projection, err := ProjectSnapshotRouteModels(snapshot, brain.CLICodex, func(cli brain.CLIKind, protocol brain.ProtocolFamily) bool {
			filterCalls.Add(1)
			return acceptedCredentiallessProjectionAdapter(cli, protocol)
		})
		if err != nil {
			t.Fatalf("order %v: %v", order, err)
		}
		if filterCalls.Load() != 1 {
			t.Fatalf("order %v called filter %d times, want once per protocol", order, filterCalls.Load())
		}
		if index == 0 {
			baseline = projection
		} else if !reflect.DeepEqual(projection, baseline) {
			t.Fatalf("order %v projection=%+v, want %+v", order, projection, baseline)
		}
	}
}

func TestProjectedRouteModelJSONContainsNoRoutingAccountOrProviderOwnership(t *testing.T) {
	projection, err := ProjectSnapshotRouteModels(
		syntheticProjectionSnapshot(projectionSpec("agy/claude-opus-4-6-thinking", brain.ProtocolAnthropicMessages, true)),
		brain.CLIClaudeCode,
		acceptedCredentiallessProjectionAdapter,
	)
	if err != nil {
		t.Fatalf("ProjectSnapshotRouteModels: %v", err)
	}
	body, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	assertExactJSONKeys(t, document, "registry_version", "models")
	models, ok := document["models"].([]any)
	if !ok || len(models) != 1 {
		t.Fatalf("models JSON=%T %+v", document["models"], document["models"])
	}
	model, ok := models[0].(map[string]any)
	if !ok {
		t.Fatalf("model JSON=%T", models[0])
	}
	assertExactJSONKeys(t, model, "id", "display_namespace", "cli_kind", "protocol", "trusted_profile")
}

func syntheticProjectionSnapshot(specs ...ModelSpec) RegistrySnapshot {
	models := make(map[brain.RouteModel]ModelSpec, len(specs))
	for _, spec := range specs {
		models[spec.Capability.RouteModel] = spec
	}
	return RegistrySnapshot{Version: "synthetic-projection-v1", Models: models}
}

func projectionSpec(model brain.RouteModel, protocol brain.ProtocolFamily, available bool) ModelSpec {
	return ModelSpec{
		Capability: brain.ModelCapability{
			RouteModel:       model,
			Protocol:         protocol,
			Streaming:        true,
			Tools:            true,
			Reasoning:        true,
			StructuredOutput: false,
			ContextLimit:     200000,
		},
		AccountPool: "synthetic-pool",
		Rotation:    RotationStrictIndependentRequest,
		Affinity:    AffinityOriginAccount,
		Available:   available,
	}
}

func projectionModelsDocument(model brain.RouteModel, protocol brain.ProtocolFamily, available bool) ModelsDocument {
	return ModelsDocument{
		Object:          "list",
		RegistryVersion: "synthetic-projection-v1",
		Models: []ModelDocument{
			{
				ID:               string(model),
				Protocol:         string(protocol),
				Streaming:        projectionBoolPointer(true),
				Tools:            projectionBoolPointer(true),
				Reasoning:        projectionBoolPointer(true),
				StructuredOutput: projectionBoolPointer(false),
				ContextLimit:     200000,
				AccountPool:      "synthetic-pool",
				Rotation:         string(RotationStrictIndependentRequest),
				Affinity:         string(AffinityOriginAccount),
				Available:        projectionBoolPointer(available),
			},
		},
	}
}

func projectionBoolPointer(value bool) *bool {
	return &value
}

func acceptedCredentiallessProjectionAdapter(cli brain.CLIKind, protocol brain.ProtocolFamily) bool {
	return cli == brain.CLIClaudeCode && protocol == brain.ProtocolAnthropicMessages ||
		cli == brain.CLICodex && protocol == brain.ProtocolOpenAIResponses
}

func isZeroProjection(projection RouteModelProjection) bool {
	return reflect.DeepEqual(projection, RouteModelProjection{})
}

func assertExactJSONKeys(t *testing.T, value map[string]any, want ...string) {
	t.Helper()
	if len(value) != len(want) {
		t.Fatalf("JSON keys=%v, want %v", reflect.ValueOf(value).MapKeys(), want)
	}
	for _, key := range want {
		if _, ok := value[key]; !ok {
			t.Fatalf("missing JSON key %q in %v", key, reflect.ValueOf(value).MapKeys())
		}
	}
}
