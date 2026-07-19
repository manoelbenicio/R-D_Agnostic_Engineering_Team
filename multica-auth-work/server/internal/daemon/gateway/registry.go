package gateway

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type RotationMode string

const (
	RotationStrictIndependentRequest RotationMode = "strict-independent-request-round-robin"
	RotationFailureOnly              RotationMode = "failure-only"

	// RegistryRefreshFailureBackoff bounds failed model-registry refreshes to
	// at most one upstream attempt every two seconds per Registry. That is slow
	// enough to prevent a readiness/Snapshot loop from hammering a failed local
	// gateway while keeping development recovery prompt and deterministic.
	RegistryRefreshFailureBackoff = 2 * time.Second

	// MaxRegistryModels is deliberately well above the frozen development
	// route set while bounding per-refresh map and validation work before any
	// model rows are copied into the cached snapshot.
	MaxRegistryModels = 1024
	// MaxRegistryFallbackDepth bounds the longest ordered fallback path in
	// edges. Approved routes use short chains; thirty-two preserves ample
	// compatibility while keeping recursive graph validation shallow.
	MaxRegistryFallbackDepth = 32
)

type AffinityMode string

const (
	AffinityNone                 AffinityMode = "none"
	AffinityOriginAccount        AffinityMode = "origin-account"
	AffinityStatelessMaterialize AffinityMode = "stateless-materialize"
)

type ModelSpec struct {
	Capability  brain.ModelCapability
	AccountPool string
	Rotation    RotationMode
	Affinity    AffinityMode
	Fallback    []brain.RouteModel
	Available   bool
}

type RegistrySnapshot struct {
	Version   string
	FetchedAt time.Time
	Models    map[brain.RouteModel]ModelSpec

	fallbackCycleProof FallbackCycleProof
}

// FallbackCycleProof returns the separate cycle-free proof produced while
// building this immutable registry snapshot. A zero snapshot returns a proof
// that is not accepted.
func (s RegistrySnapshot) FallbackCycleProof() FallbackCycleProof {
	return s.fallbackCycleProof
}

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

func (r *Registry) Snapshot(ctx context.Context) (RegistrySnapshot, error) {
	for {
		now := r.now()
		r.mu.Lock()
		if r.snapshot.Version != "" && now.Before(r.expiresAt) {
			snapshot := cloneSnapshot(r.snapshot)
			r.mu.Unlock()
			return snapshot, nil
		}
		if r.refreshErr != nil && now.Before(r.retryAt) {
			err := r.refreshErr
			r.mu.Unlock()
			return RegistrySnapshot{}, err
		}
		if r.refreshing != nil {
			wait := r.refreshing
			r.mu.Unlock()
			select {
			case <-ctx.Done():
				return RegistrySnapshot{}, classifyContextError("registry", ctx)
			case <-wait:
			}
			continue
		}
		wait := make(chan struct{})
		r.refreshing = wait
		generation := r.generation
		r.mu.Unlock()

		document, err := r.fetcher.FetchModels(ctx)
		var snapshot RegistrySnapshot
		if err == nil {
			snapshot, err = buildSnapshot(document, now)
		}
		failedAt := r.now()

		r.mu.Lock()
		stale := generation != r.generation
		if !stale {
			if err == nil {
				r.snapshot = snapshot
				r.expiresAt = now.Add(r.ttl)
				r.refreshErr = nil
				r.retryAt = time.Time{}
			} else if ctx.Err() == nil {
				// Cache only gateway/registry failures. A caller-local cancellation
				// must not suppress a healthy caller's immediate refresh attempt.
				r.refreshErr = err
				r.retryAt = failedAt.Add(RegistryRefreshFailureBackoff)
			}
		}
		close(wait)
		r.refreshing = nil
		r.mu.Unlock()
		if stale {
			// Invalidate is a generation fence. A completion from an older
			// generation wakes waiters but cannot populate either the positive
			// cache or the negative-cache backoff window.
			if ctx.Err() != nil {
				return RegistrySnapshot{}, classifyContextError("registry", ctx)
			}
			continue
		}
		if err != nil {
			return RegistrySnapshot{}, err
		}
		return cloneSnapshot(snapshot), nil
	}
}

func (r *Registry) Invalidate() {
	r.mu.Lock()
	r.generation++
	r.expiresAt = time.Time{}
	r.refreshErr = nil
	r.retryAt = time.Time{}
	r.mu.Unlock()
}

func (r *Registry) LookupModelCapability(ctx context.Context, model brain.RouteModel) (brain.ModelCapability, error) {
	spec, err := r.LookupModel(ctx, model)
	if err != nil {
		return brain.ModelCapability{}, err
	}
	return spec.Capability, nil
}

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

type CapabilityRequirement struct {
	Protocol         brain.ProtocolFamily
	Streaming        bool
	Tools            bool
	Reasoning        bool
	StructuredOutput bool
	MinimumContext   int
}

func (r *Registry) ValidateCapability(ctx context.Context, model brain.RouteModel, requirement CapabilityRequirement) error {
	spec, err := r.LookupModel(ctx, model)
	if err != nil {
		return err
	}
	capability := spec.Capability
	if requirement.Protocol != "" && capability.Protocol != requirement.Protocol {
		return &GatewayError{Operation: "registry.validate", Class: ErrorCapability}
	}
	if requirement.Streaming && !capability.Streaming ||
		requirement.Tools && !capability.Tools ||
		requirement.Reasoning && !capability.Reasoning ||
		requirement.StructuredOutput && !capability.StructuredOutput ||
		requirement.MinimumContext > capability.ContextLimit {
		return &GatewayError{Operation: "registry.validate", Class: ErrorCapability}
	}
	return nil
}

func buildSnapshot(document ModelsDocument, observedAt time.Time) (RegistrySnapshot, error) {
	version := strings.TrimSpace(document.RegistryVersion)
	if version == "" || len(version) > 128 || strings.ContainsAny(version, "\r\n\x00") || len(document.Models) == 0 || len(document.Models) > MaxRegistryModels {
		return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
	}
	models := make(map[brain.RouteModel]ModelSpec, len(document.Models))
	for _, row := range document.Models {
		model, err := brain.ParseRouteModel(row.ID)
		if err != nil {
			return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
		}
		if _, exists := models[model]; exists {
			return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
		}
		protocol, err := protocolFromWire(row.Protocol)
		if err != nil || row.Streaming == nil || row.Tools == nil || row.Reasoning == nil || row.StructuredOutput == nil || row.Available == nil {
			return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
		}
		if row.ContextLimit <= 0 || strings.TrimSpace(row.AccountPool) == "" || len(row.AccountPool) > 128 || strings.ContainsAny(row.AccountPool, "\r\n\x00") {
			return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
		}
		rotation, err := parseRotation(row.Rotation)
		if err != nil {
			return RegistrySnapshot{}, err
		}
		affinity, err := parseAffinity(row.Affinity)
		if err != nil {
			return RegistrySnapshot{}, err
		}
		if len(row.Fallback) > MaxRegistryModels {
			return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
		}
		fallback := make([]brain.RouteModel, 0, len(row.Fallback))
		for _, fallbackID := range row.Fallback {
			fallbackModel, parseErr := brain.ParseRouteModel(fallbackID)
			if parseErr != nil {
				return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
			}
			fallback = append(fallback, fallbackModel)
		}
		models[model] = ModelSpec{
			Capability: brain.ModelCapability{
				RouteModel:       model,
				Protocol:         protocol,
				Streaming:        *row.Streaming,
				Tools:            *row.Tools,
				Reasoning:        *row.Reasoning,
				StructuredOutput: *row.StructuredOutput,
				ContextLimit:     row.ContextLimit,
				ObservedAt:       observedAt,
			},
			AccountPool: strings.TrimSpace(row.AccountPool),
			Rotation:    rotation,
			Affinity:    affinity,
			Fallback:    fallback,
			Available:   *row.Available,
		}
	}
	for _, spec := range models {
		for _, fallback := range spec.Fallback {
			if _, exists := models[fallback]; !exists {
				return RegistrySnapshot{}, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
			}
		}
	}
	if err := validateFallbackGraph(models); err != nil {
		return RegistrySnapshot{}, err
	}
	return RegistrySnapshot{
		Version:            version,
		FetchedAt:          observedAt,
		Models:             models,
		fallbackCycleProof: acceptedFallbackCycleProof(),
	}, nil
}

func validateFallbackGraph(models map[brain.RouteModel]ModelSpec) error {
	type visitState uint8
	const (
		visitUnseen visitState = iota
		visitActive
		visitComplete
	)

	keys := make([]string, 0, len(models))
	for model := range models {
		keys = append(keys, string(model))
	}
	sort.Strings(keys)
	states := make(map[brain.RouteModel]visitState, len(models))
	depths := make(map[brain.RouteModel]int, len(models))
	var visit func(brain.RouteModel) (int, error)
	visit = func(model brain.RouteModel) (int, error) {
		switch states[model] {
		case visitActive:
			return 0, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
		case visitComplete:
			return depths[model], nil
		}
		states[model] = visitActive
		longest := 0
		fallbacks := append([]brain.RouteModel(nil), models[model].Fallback...)
		sort.Slice(fallbacks, func(left, right int) bool {
			return fallbacks[left] < fallbacks[right]
		})
		for _, fallback := range fallbacks {
			childDepth, err := visit(fallback)
			if err != nil {
				return 0, err
			}
			candidate := childDepth + 1
			if candidate > MaxRegistryFallbackDepth {
				return 0, &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
			}
			longest = max(longest, candidate)
		}
		states[model] = visitComplete
		depths[model] = longest
		return longest, nil
	}
	for _, key := range keys {
		if _, err := visit(brain.RouteModel(key)); err != nil {
			return err
		}
	}
	return nil
}

func parseRotation(value string) (RotationMode, error) {
	mode := RotationMode(strings.TrimSpace(value))
	switch mode {
	case RotationStrictIndependentRequest, RotationFailureOnly:
		return mode, nil
	default:
		return "", &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
	}
}

func parseAffinity(value string) (AffinityMode, error) {
	mode := AffinityMode(strings.TrimSpace(value))
	switch mode {
	case AffinityNone, AffinityOriginAccount, AffinityStatelessMaterialize:
		return mode, nil
	default:
		return "", &GatewayError{Operation: "registry.refresh", Class: ErrorProtocol}
	}
}

func cloneSnapshot(source RegistrySnapshot) RegistrySnapshot {
	models := make(map[brain.RouteModel]ModelSpec, len(source.Models))
	keys := make([]string, 0, len(source.Models))
	for model := range source.Models {
		keys = append(keys, string(model))
	}
	sort.Strings(keys)
	for _, key := range keys {
		model := brain.RouteModel(key)
		models[model] = cloneModelSpec(source.Models[model])
	}
	return RegistrySnapshot{
		Version:            source.Version,
		FetchedAt:          source.FetchedAt,
		Models:             models,
		fallbackCycleProof: source.fallbackCycleProof,
	}
}

func cloneModelSpec(source ModelSpec) ModelSpec {
	result := source
	result.Fallback = append([]brain.RouteModel(nil), source.Fallback...)
	return result
}

func classifyContextError(operation string, ctx context.Context) error {
	if ctx.Err() == context.Canceled {
		return &GatewayError{Operation: operation, Class: ErrorCancelled}
	}
	return &GatewayError{Operation: operation, Class: ErrorTimeout, Retryable: true}
}

func (s ModelSpec) String() string {
	return fmt.Sprintf("gateway.ModelSpec{model:%q, protocol:%q, account_pool:[redacted]}", s.Capability.RouteModel, s.Capability.Protocol)
}
