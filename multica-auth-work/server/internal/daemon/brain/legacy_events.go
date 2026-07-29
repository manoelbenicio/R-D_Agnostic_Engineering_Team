package brain

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type LegacyUseOutcome string

const (
	LegacyUseTranslated LegacyUseOutcome = "translated"
	LegacyUseShadowed   LegacyUseOutcome = "shadowed"
	LegacyUseRejected   LegacyUseOutcome = "rejected"
)

type LegacyUseMeasurement struct {
	Surface CompatibilitySurface
	Alias   string
	Outcome LegacyUseOutcome
	Count   uint64
}

type legacyUseKey struct {
	surface CompatibilitySurface
	alias   string
	outcome LegacyUseOutcome
}

// MemoryLegacyUseRecorder provides bounded-cardinality measurements for the
// supported legacy aliases. It stores names and counts only, never values.
type MemoryLegacyUseRecorder struct {
	mu     sync.RWMutex
	counts map[legacyUseKey]uint64
}

func NewMemoryLegacyUseRecorder() *MemoryLegacyUseRecorder {
	return &MemoryLegacyUseRecorder{counts: make(map[legacyUseKey]uint64)}
}

func (r *MemoryLegacyUseRecorder) RecordLegacyUse(ctx context.Context, event LegacyUseEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("legacy use recorder is nil")
	}
	if !knownLegacyMeasurement(event) {
		return fmt.Errorf("legacy measurement is not part of the frozen compatibility surface")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.counts == nil {
		r.counts = make(map[legacyUseKey]uint64)
	}
	r.counts[legacyUseKey{surface: event.Surface, alias: event.Alias, outcome: event.Outcome}]++
	return nil
}

func (r *MemoryLegacyUseRecorder) Snapshot() []LegacyUseMeasurement {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LegacyUseMeasurement, 0, len(r.counts))
	for key, count := range r.counts {
		out = append(out, LegacyUseMeasurement{Surface: key.surface, Alias: key.alias, Outcome: key.outcome, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Surface != out[j].Surface {
			return out[i].Surface < out[j].Surface
		}
		if out[i].Alias != out[j].Alias {
			return out[i].Alias < out[j].Alias
		}
		return out[i].Outcome < out[j].Outcome
	})
	return out
}

type CompatibilityTranslator struct {
	Recorder LegacyUseRecorder
}

func NewCompatibilityTranslator(recorder LegacyUseRecorder) (*CompatibilityTranslator, error) {
	if recorder == nil {
		return nil, fmt.Errorf("legacy use recorder is required")
	}
	return &CompatibilityTranslator{Recorder: recorder}, nil
}

type NeutralTaskTranslation struct {
	Task  Task
	Token TaskToken
}

func (t *CompatibilityTranslator) TranslateTask(ctx context.Context, input LegacyTaskInput, correlation Correlation, policy ApprovedRoutePolicy, lifecycle LifecycleBindings, gatewayRequired bool) (NeutralTaskTranslation, error) {
	translated, err := TranslateLegacyTask(input, correlation, policy.ID, gatewayRequired)
	outcome := LegacyUseTranslated
	if err != nil {
		outcome = LegacyUseRejected
	}
	events := []LegacyUseEvent{
		{Surface: SurfaceDaemonAPI, Alias: "provider/model", Outcome: outcome},
		{Surface: SurfaceTaskToken, Alias: "auth_token", Outcome: outcome},
	}
	if input.RuntimeRouterOwner != "" {
		events = append(events, LegacyUseEvent{Surface: SurfaceRouterOwner, Alias: "runtime_router_owner", Outcome: outcome})
	}
	if recordErr := t.record(ctx, events...); recordErr != nil {
		return NeutralTaskTranslation{}, recordErr
	}
	if err != nil {
		return NeutralTaskTranslation{}, err
	}
	task := Task{Request: translated.Request, RoutePolicy: policy, Lifecycle: lifecycle}
	if err := task.Validate(); err != nil {
		return NeutralTaskTranslation{}, err
	}
	return NeutralTaskTranslation{Task: task, Token: translated.Token}, nil
}

func (t *CompatibilityTranslator) ResolveConfig(ctx context.Context, candidates ...ConfigCandidate) (ResolvedConfigValue, error) {
	resolved, resolveErr := ResolveConfigValue(candidates...)
	for _, candidate := range candidates {
		if !candidate.Set || !isLegacySource(candidate.Source) {
			continue
		}
		surface, err := legacySourceSurface(candidate.Source)
		if err != nil {
			return ResolvedConfigValue{}, err
		}
		compatible, known := legacyAliasCompatibility(candidate.Name)
		if !known {
			return ResolvedConfigValue{}, fmt.Errorf("legacy config alias %q is not frozen", candidate.Name)
		}
		outcome := LegacyUseShadowed
		if resolveErr != nil || (resolved.Name == candidate.Name && !compatible) {
			outcome = LegacyUseRejected
		} else if resolved.Name == candidate.Name {
			outcome = LegacyUseTranslated
		}
		if err := t.record(ctx, LegacyUseEvent{Surface: surface, Alias: candidate.Name, Outcome: outcome}); err != nil {
			return ResolvedConfigValue{}, err
		}
		if resolved.Name == candidate.Name && !compatible {
			return ResolvedConfigValue{}, fmt.Errorf("legacy config alias %q is not semantically compatible", candidate.Name)
		}
	}
	return resolved, resolveErr
}

func (t *CompatibilityTranslator) record(ctx context.Context, events ...LegacyUseEvent) error {
	if t == nil || t.Recorder == nil {
		return fmt.Errorf("compatibility translator is not configured")
	}
	for _, event := range events {
		if err := t.Recorder.RecordLegacyUse(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func isLegacySource(source ValueSource) bool {
	return source == SourceLegacyCLI || source == SourceLegacyEnv || source == SourceLegacyStored
}

func legacySourceSurface(source ValueSource) (CompatibilitySurface, error) {
	switch source {
	case SourceLegacyCLI:
		return SurfaceCLICommand, nil
	case SourceLegacyEnv:
		return SurfaceEnvironment, nil
	case SourceLegacyStored:
		return SurfaceStoredConfig, nil
	default:
		return "", fmt.Errorf("source %q is not legacy", source)
	}
}

func legacyAliasCompatibility(name string) (bool, bool) {
	for _, definition := range FrozenConfigAliases() {
		for _, legacy := range definition.Legacy {
			if legacy == name {
				return definition.SemanticCompatibility, true
			}
		}
	}
	return false, false
}

func knownLegacyMeasurement(event LegacyUseEvent) bool {
	switch event.Outcome {
	case LegacyUseTranslated, LegacyUseShadowed, LegacyUseRejected:
	default:
		return false
	}
	switch event.Surface {
	case SurfaceDaemonAPI:
		return event.Alias == "provider/model"
	case SurfaceTaskToken:
		return event.Alias == "auth_token"
	case SurfaceRouterOwner:
		return event.Alias == "runtime_router_owner"
	case SurfaceEnvironment, SurfaceStoredConfig, SurfaceCLICommand:
		_, known := legacyAliasCompatibility(event.Alias)
		return known
	default:
		return false
	}
}
