package runtimeconfig

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func ptr[T any](value T) *T { return &value }

func testCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Version:  VersionV1,
		Provider: ProviderID("provider-a"),
		Models: map[ModelID]ModelCapabilities{
			ModelID("model-a"): {
				ReasoningEfforts:    []ReasoningEffort{"low", "medium", "high"},
				MaxInputTokens:      8_000,
				MaxOutputTokens:     4_000,
				ContextWindowTokens: 10_000,
				MaxToolCalls:        20,
			},
			ModelID("model-b"): {
				ReasoningEfforts:    []ReasoningEffort{"low", "high"},
				MaxInputTokens:      16_000,
				MaxOutputTokens:     8_000,
				ContextWindowTokens: 20_000,
				MaxToolCalls:        40,
			},
		},
	}
}

func fullValues() Values {
	return Values{
		TransportBinding: ptr(TransportOmniRoute),
		CLIKind:          ptr(CLIKind("codex")),
		Provider:         ptr(ProviderID("provider-a")),
		Model:            ptr(ModelID("model-a")),
		ReasoningEffort:  ptr(ReasoningEffort("medium")),
		Limits: Limits{
			MaxInputTokens:  ptr(int64(6_000)),
			MaxOutputTokens: ptr(int64(3_000)),
			MaxToolCalls:    ptr(int64(12)),
		},
	}
}

func fullInput() ResolveInput {
	return ResolveInput{
		Platform: &Config{
			Version: VersionV1,
			Values:  fullValues(),
			Delegability: map[Field]bool{
				FieldModel:           true,
				FieldReasoningEffort: true,
				FieldMaxInputTokens:  true,
				FieldMaxOutputTokens: true,
				FieldMaxToolCalls:    true,
			},
		},
		Capabilities: testCapabilities(),
	}
}

func TestResolvePrecedenceAndExplicitTaskDelegation(t *testing.T) {
	input := ResolveInput{
		Platform: &Config{
			Version: VersionV1,
			Values: Values{
				TransportBinding: ptr(TransportOmniRoute),
				CLIKind:          ptr(CLIKind("codex")),
				Provider:         ptr(ProviderID("provider-a")),
			},
			Delegability: map[Field]bool{
				FieldMaxToolCalls:    true,
				FieldReasoningEffort: true,
			},
		},
		Standard: &Config{
			Version: VersionV1,
			Values: Values{
				Model:  ptr(ModelID("model-a")),
				Limits: Limits{MaxInputTokens: ptr(int64(6_000))},
			},
		},
		Runtime: &Config{
			Version: VersionV1,
			Values: Values{
				Model:           ptr(ModelID("model-b")),
				ReasoningEffort: ptr(ReasoningEffort("medium")),
				Limits:          Limits{MaxOutputTokens: ptr(int64(3_000))},
			},
		},
		Task: &Config{
			Version: VersionV1,
			Values: Values{
				ReasoningEffort: ptr(ReasoningEffort("high")),
				Limits:          Limits{MaxToolCalls: ptr(int64(7))},
			},
		},
		Capabilities: testCapabilities(),
	}

	effective, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := *effective.Values.Model; got != "model-a" {
		t.Fatalf("model = %q, want standard model-a", got)
	}
	if got := effective.Origins[FieldModel]; got != SourceStandard {
		t.Fatalf("model origin = %q, want standard", got)
	}
	if got := *effective.Values.ReasoningEffort; got != "medium" {
		t.Fatalf("reasoning = %q, want runtime medium", got)
	}
	if got := effective.Origins[FieldReasoningEffort]; got != SourceRuntime {
		t.Fatalf("reasoning origin = %q, want runtime", got)
	}
	if got := *effective.Values.Limits.MaxToolCalls; got != 7 {
		t.Fatalf("max tool calls = %d, want delegated task value 7", got)
	}
	if got := effective.Origins[FieldMaxToolCalls]; got != SourceTask {
		t.Fatalf("max tool calls origin = %q, want task", got)
	}
}

func TestResolveDelegabilityUsesHighestExplicitPolicy(t *testing.T) {
	input := fullInput()
	input.Platform.Delegability[FieldMaxToolCalls] = false
	input.Standard = &Config{
		Version:      VersionV1,
		Delegability: map[Field]bool{FieldMaxToolCalls: true},
	}
	input.Task = &Config{
		Version: VersionV1,
		Values:  Values{Limits: Limits{MaxToolCalls: ptr(int64(5))}},
	}

	effective, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve succeeded for platform-denied task field")
	}
	if effective.Digest != "" {
		t.Fatal("failure returned a partial effective configuration")
	}
	assertErrorCode(t, err, ErrTaskFieldNotDelegable, FieldMaxToolCalls)
}

func TestResolveRejectsNonDelegableTaskFieldWithoutLeakingValue(t *testing.T) {
	input := fullInput()
	input.Platform.Delegability[FieldModel] = false
	input.Task = &Config{
		Version: VersionV1,
		Values:  Values{Model: ptr(ModelID("sensitive-looking-model-value"))},
	}

	_, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve succeeded for non-delegable model")
	}
	assertErrorCode(t, err, ErrTaskFieldNotDelegable, FieldModel)
	if strings.Contains(err.Error(), "sensitive-looking-model-value") {
		t.Fatal("validation error leaked rejected field value")
	}
}

func TestResolveRejectsTaskDelegabilityPolicy(t *testing.T) {
	input := fullInput()
	input.Task = &Config{
		Version:      VersionV1,
		Delegability: map[Field]bool{FieldModel: true},
	}
	_, err := Resolve(input)
	assertErrorCode(t, err, ErrPolicyNotAllowed, Field("delegability"))
}

func TestResolveSubagentInheritsAndOverridesOnlyDelegatedFields(t *testing.T) {
	input := fullInput()
	parent, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve parent: %v", err)
	}
	child, err := ResolveSubagent(parent, &Config{
		Version: VersionV1,
		Values: Values{
			Model:           ptr(ModelID("model-b")),
			ReasoningEffort: ptr(ReasoningEffort("high")),
		},
	}, testCapabilities())
	if err != nil {
		t.Fatalf("ResolveSubagent: %v", err)
	}
	if got := *child.Values.Model; got != "model-b" {
		t.Fatalf("child model = %q, want model-b", got)
	}
	if child.Origins[FieldModel] != SourceTask {
		t.Fatalf("child model origin = %q, want task", child.Origins[FieldModel])
	}
	if child.Origins[FieldProvider] != SourceParent {
		t.Fatalf("child provider origin = %q, want parent", child.Origins[FieldProvider])
	}
	if *child.Values.Provider != *parent.Values.Provider || *child.Values.CLIKind != *parent.Values.CLIKind {
		t.Fatal("child did not inherit parent provider and CLI")
	}
}

func TestResolveSubagentFailsClosedForNonDelegableOrTamperedParent(t *testing.T) {
	parent, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve parent: %v", err)
	}
	_, err = ResolveSubagent(parent, &Config{
		Version: VersionV1,
		Values:  Values{Provider: ptr(ProviderID("provider-b"))},
	}, testCapabilities())
	assertErrorCode(t, err, ErrTaskFieldNotDelegable, FieldProvider)

	parent.Values.Model = ptr(ModelID("model-b"))
	_, err = ResolveSubagent(parent, nil, testCapabilities())
	assertErrorCode(t, err, ErrInvalidParentDigest, Field("digest"))
}

func TestResolveProviderCapabilityValidation(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*ResolveInput)
		code  ErrorCode
		field Field
	}{
		{
			name: "provider mismatch",
			alter: func(input *ResolveInput) {
				input.Capabilities.Provider = "provider-b"
			},
			code: ErrProviderMismatch, field: FieldProvider,
		},
		{
			name: "unsupported model",
			alter: func(input *ResolveInput) {
				input.Platform.Values.Model = ptr(ModelID("model-unknown"))
			},
			code: ErrUnsupportedModel, field: FieldModel,
		},
		{
			name: "unsupported reasoning",
			alter: func(input *ResolveInput) {
				input.Platform.Values.ReasoningEffort = ptr(ReasoningEffort("extreme"))
			},
			code: ErrUnsupportedReasoning, field: FieldReasoningEffort,
		},
		{
			name: "input limit",
			alter: func(input *ResolveInput) {
				input.Platform.Values.Limits.MaxInputTokens = ptr(int64(8_001))
			},
			code: ErrLimitExceeded, field: FieldMaxInputTokens,
		},
		{
			name: "output limit",
			alter: func(input *ResolveInput) {
				input.Platform.Values.Limits.MaxOutputTokens = ptr(int64(4_001))
			},
			code: ErrLimitExceeded, field: FieldMaxOutputTokens,
		},
		{
			name: "tool limit",
			alter: func(input *ResolveInput) {
				input.Platform.Values.Limits.MaxToolCalls = ptr(int64(21))
			},
			code: ErrLimitExceeded, field: FieldMaxToolCalls,
		},
		{
			name: "combined context window",
			alter: func(input *ResolveInput) {
				input.Platform.Values.Limits.MaxInputTokens = ptr(int64(7_000))
				input.Platform.Values.Limits.MaxOutputTokens = ptr(int64(3_001))
			},
			code: ErrContextWindowExceeded, field: Field("token_budget"),
		},
		{
			name: "malformed capabilities",
			alter: func(input *ResolveInput) {
				capability := input.Capabilities.Models["model-a"]
				capability.ContextWindowTokens = 0
				input.Capabilities.Models["model-a"] = capability
			},
			code: ErrInvalidCapabilities, field: FieldModel,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := fullInput()
			test.alter(&input)
			effective, err := Resolve(input)
			if err == nil {
				t.Fatal("Resolve succeeded, want capability failure")
			}
			if effective.Digest != "" {
				t.Fatal("capability failure returned partial configuration")
			}
			assertErrorCode(t, err, test.code, test.field)
		})
	}
}

func TestResolveFailClosedValidation(t *testing.T) {
	input := fullInput()
	input.Platform.Version = Version("v999")
	input.Platform.Values.CLIKind = nil
	input.Platform.Values.Limits.MaxToolCalls = ptr(int64(0))
	input.Platform.Delegability[Field("unknown")] = true

	effective, err := Resolve(input)
	if err == nil {
		t.Fatal("Resolve succeeded for invalid input")
	}
	if effective.Version != "" || effective.Digest != "" || effective.Values.Model != nil || effective.Origins != nil || effective.Delegable != nil {
		t.Fatalf("failure returned partial effective config: %#v", effective)
	}
	assertErrorCode(t, err, ErrInvalidVersion, Field("schema_version"))
	assertErrorCode(t, err, ErrRequiredField, FieldCLIKind)
	assertErrorCode(t, err, ErrInvalidField, FieldMaxToolCalls)
	assertErrorCode(t, err, ErrUnknownField, Field("unknown"))
}

func TestCanonicalRedactedDigestIsDeterministicAndOriginIndependent(t *testing.T) {
	firstInput := fullInput()
	first, err := Resolve(firstInput)
	if err != nil {
		t.Fatalf("Resolve first: %v", err)
	}

	secondInput := fullInput()
	secondInput.Standard = &Config{Version: VersionV1, Values: cloneValues(secondInput.Platform.Values)}
	secondInput.Platform.Values = Values{}
	second, err := Resolve(secondInput)
	if err != nil {
		t.Fatalf("Resolve second: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("equivalent config digests differ:\n%s\n%s", first.Digest, second.Digest)
	}
	if first.Origins[FieldModel] == second.Origins[FieldModel] {
		t.Fatal("test precondition failed: origins should differ")
	}

	canonical, err := first.CanonicalRedacted()
	if err != nil {
		t.Fatalf("CanonicalRedacted: %v", err)
	}
	text := string(canonical)
	for _, forbidden := range []string{"origin", "platform", "credential", "token\"", "cookie", "prompt", "path"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("canonical redacted form contains forbidden %q: %s", forbidden, text)
		}
	}
	if !ValidDigest(first.Digest) {
		t.Fatalf("digest is not raw lowercase 64-hex: %q", first.Digest)
	}
	if len(first.Digest) != DigestLength {
		t.Fatalf("digest length = %d, want %d", len(first.Digest), DigestLength)
	}
	if strings.ContainsAny(first.Digest, ":ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("digest carries a prefix or uppercase hex: %q", first.Digest)
	}

	changedInput := fullInput()
	changedInput.Platform.Delegability[FieldCLIKind] = true
	changed, err := Resolve(changedInput)
	if err != nil {
		t.Fatalf("Resolve changed policy: %v", err)
	}
	if first.Digest == changed.Digest {
		t.Fatal("delegability policy change did not change digest")
	}
}

func TestClassifyChangesHotAndRestartRequired(t *testing.T) {
	current, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve current: %v", err)
	}
	nextInput := fullInput()
	nextInput.Platform.Values.Provider = ptr(ProviderID("provider-b"))
	nextInput.Platform.Values.Model = ptr(ModelID("model-b"))
	nextInput.Platform.Values.ReasoningEffort = ptr(ReasoningEffort("high"))
	nextInput.Platform.Values.Limits.MaxToolCalls = ptr(int64(13))
	nextInput.Capabilities.Provider = "provider-b"
	next, err := Resolve(nextInput)
	if err != nil {
		t.Fatalf("Resolve next: %v", err)
	}

	plan, err := ClassifyChanges(current, next)
	if err != nil {
		t.Fatalf("ClassifyChanges: %v", err)
	}
	assertFields(t, plan.RestartRequired, []Field{FieldProvider})
	assertFields(t, plan.Hot, []Field{FieldModel, FieldReasoningEffort, FieldMaxToolCalls})

	// A well-formed digest that belongs to a different configuration must be
	// rejected, and so must the legacy prefixed form. Both are the same
	// fail-closed outcome, so neither shape can smuggle an unvalidated config.
	next.Digest = digestFor(current.Values, map[Field]bool{})
	_, err = ClassifyChanges(current, next)
	assertErrorCode(t, err, ErrInvalidParentDigest, Field("digest"))

	const legacyPrefixedDigest = "sha256:" + "0000000000000000000000000000000000000000000000000000000000000000"
	next.Digest = legacyPrefixedDigest
	_, err = ClassifyChanges(current, next)
	assertErrorCode(t, err, ErrInvalidParentDigest, Field("digest"))
}

func TestClassifyFieldCoversEveryField(t *testing.T) {
	for _, field := range allFields {
		if _, ok := ClassifyField(field); !ok {
			t.Fatalf("field %q has no reload classification", field)
		}
	}
	if _, ok := ClassifyField(Field("unknown")); ok {
		t.Fatal("unknown field was classified")
	}
}

func TestResolveIsPureUnderConcurrency(t *testing.T) {
	input := fullInput()
	want, err := Resolve(input)
	if err != nil {
		t.Fatalf("Resolve baseline: %v", err)
	}

	const workers = 64
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			got, resolveErr := Resolve(input)
			if resolveErr != nil {
				errCh <- resolveErr
				return
			}
			if got.Digest != want.Digest {
				errCh <- errors.New("concurrent resolve returned a different digest")
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

// --- frozen field registry -------------------------------------------------

func TestFrozenFieldRegistryContract(t *testing.T) {
	// The ordered set is frozen for v1. Changing this literal is a contract
	// version bump, not an edit: the order is observable in ChangePlan and in
	// the canonical digest input.
	want := []Field{
		FieldTransportBinding,
		FieldCLIKind,
		FieldProvider,
		FieldModel,
		FieldReasoningEffort,
		FieldMaxInputTokens,
		FieldMaxOutputTokens,
		FieldMaxToolCalls,
	}
	assertFields(t, Fields(), want)
	assertFields(t, allFields, want)

	// Fields must hand out a copy, never the registry order itself.
	mutated := Fields()
	mutated[0] = Field("clobbered")
	assertFields(t, Fields(), want)

	if len(fieldRegistry) != len(want) {
		t.Fatalf("registry length = %d, want %d", len(fieldRegistry), len(want))
	}
	for _, spec := range fieldRegistry {
		if spec.reload != ReloadHot && spec.reload != ReloadRestartRequired {
			t.Fatalf("field %q has reload class %q, want hot or restart", spec.field, spec.reload)
		}
		if spec.get == nil || spec.set == nil {
			t.Fatalf("field %q is missing an accessor", spec.field)
		}
		if _, ok := ClassifyField(spec.field); !ok {
			t.Fatalf("field %q is not classifiable", spec.field)
		}
		if !knownField(spec.field) {
			t.Fatalf("field %q is not recognised as known", spec.field)
		}
	}

	assertFields(t, requiredFields(), []Field{
		FieldTransportBinding, FieldCLIKind, FieldProvider, FieldModel,
	})

	if knownField(Field("unknown")) {
		t.Fatal("unknown field was accepted as known")
	}
	if _, ok := fieldValue(fullValues(), Field("unknown")); ok {
		t.Fatal("unknown field returned a value")
	}
	var sink Values
	setField(&sink, Field("unknown"), int64(1))
	if len(setFields(sink)) != 0 {
		t.Fatal("unknown field was written into Values")
	}
}

func TestRegistryAccessorsRoundTripEveryField(t *testing.T) {
	source := fullValues()
	for _, field := range Fields() {
		value, ok := fieldValue(source, field)
		if !ok {
			t.Fatalf("field %q unset in fullValues", field)
		}
		var target Values
		setField(&target, field, value)
		roundTripped, ok := fieldValue(target, field)
		if !ok {
			t.Fatalf("field %q did not round trip", field)
		}
		if roundTripped != value {
			t.Fatalf("field %q round tripped to %v, want %v", field, roundTripped, value)
		}
		// Writing one field must not populate any other field, which is what
		// catches an accessor wired to the wrong struct member.
		if written := setFields(target); len(written) != 1 || written[0] != field {
			t.Fatalf("writing %q also set %#v", field, written)
		}
	}
}

func TestRequiredFieldsAreEnforcedOnlyOnEffectiveConfig(t *testing.T) {
	// A layer may omit a required field; the resolved result may not.
	input := fullInput()
	input.Platform.Values.Provider = nil
	_, err := Resolve(input)
	assertErrorCode(t, err, ErrRequiredField, FieldProvider)

	input = fullInput()
	input.Standard = &Config{Version: VersionV1, Values: Values{Model: ptr(ModelID("model-b"))}}
	if _, err := Resolve(input); err != nil {
		t.Fatalf("partial layer rejected: %v", err)
	}
}

// --- digest normalization (P0) ---------------------------------------------

func TestDigestIsRawLowercaseHexEverywhere(t *testing.T) {
	parent, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	child, err := ResolveSubagent(parent, &Config{
		Version: VersionV1,
		Values:  Values{Model: ptr(ModelID("model-b"))},
	}, capabilitiesModelBAcceptsMedium())
	if err != nil {
		t.Fatalf("ResolveSubagent: %v", err)
	}
	activation, _, err := InitialActivation(parent)
	if err != nil {
		t.Fatalf("InitialActivation: %v", err)
	}

	for name, digest := range map[string]string{
		"resolve":  parent.Digest,
		"subagent": child.Digest,
		"active":   activation.Active.Digest,
	} {
		if !ValidDigest(digest) {
			t.Fatalf("%s digest %q is not raw lowercase 64-hex", name, digest)
		}
		if strings.Contains(digest, ":") || digest != strings.ToLower(digest) {
			t.Fatalf("%s digest %q is prefixed or uppercase", name, digest)
		}
	}
}

func TestValidDigestRejectsEveryNonCanonicalForm(t *testing.T) {
	valid := digestFor(fullValues(), map[Field]bool{})
	if !ValidDigest(valid) {
		t.Fatalf("canonical digest rejected: %q", valid)
	}
	rejected := map[string]string{
		"empty":           "",
		"prefixed":        "sha256:" + valid,
		"uppercase":       strings.ToUpper(valid),
		"truncated":       valid[:DigestLength-1],
		"overlong":        valid + "0",
		"non hex":         strings.Repeat("g", DigestLength),
		"leading space":   " " + valid[1:],
		"embedded hyphen": valid[:DigestLength-1] + "-",
	}
	for name, digest := range rejected {
		if ValidDigest(digest) {
			t.Fatalf("%s digest was accepted: %q", name, digest)
		}
	}
}

// --- activation and rollback ----------------------------------------------

func TestInitialActivationClassifiesEverySetField(t *testing.T) {
	config, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	activation, plan, err := InitialActivation(config)
	if err != nil {
		t.Fatalf("InitialActivation: %v", err)
	}
	if activation.Generation != 1 {
		t.Fatalf("generation = %d, want 1", activation.Generation)
	}
	if activation.Previous != nil {
		t.Fatal("first activation has a rollback target")
	}
	assertFields(t, plan.RestartRequired, []Field{FieldTransportBinding, FieldCLIKind, FieldProvider})
	assertFields(t, plan.Hot, []Field{
		FieldModel, FieldReasoningEffort, FieldMaxInputTokens, FieldMaxOutputTokens, FieldMaxToolCalls,
	})
	if plan.DelegabilityChanged {
		t.Fatal("first activation reported a delegability change")
	}
	if !plan.RequiresRestart() || plan.IsEmpty() {
		t.Fatal("first activation plan should require a restart and be non-empty")
	}

	// Activate with a nil current is the same transition.
	viaActivate, activatePlan, err := Activate(nil, config)
	if err != nil {
		t.Fatalf("Activate(nil): %v", err)
	}
	if viaActivate.Generation != 1 || viaActivate.Active.Digest != activation.Active.Digest {
		t.Fatal("Activate(nil) diverged from InitialActivation")
	}
	assertFields(t, activatePlan.Hot, plan.Hot)
}

func TestActivateIsIdempotentByDigest(t *testing.T) {
	first, second := twoGenerations(t)
	activation, _, err := Activate(nil, first)
	if err != nil {
		t.Fatalf("Activate first: %v", err)
	}
	advanced, _, err := Activate(&activation, second)
	if err != nil {
		t.Fatalf("Activate second: %v", err)
	}
	if advanced.Generation != 2 || advanced.Previous == nil {
		t.Fatalf("generation = %d, previous set = %t", advanced.Generation, advanced.Previous != nil)
	}

	repeat, plan, err := Activate(&advanced, second)
	if err != nil {
		t.Fatalf("Activate repeat: %v", err)
	}
	if repeat.Generation != advanced.Generation {
		t.Fatalf("idempotent activation advanced generation to %d", repeat.Generation)
	}
	if !plan.IsEmpty() || plan.RequiresRestart() {
		t.Fatalf("idempotent activation produced a plan: %#v", plan)
	}
	if repeat.Previous == nil || repeat.Previous.Digest != first.Digest {
		t.Fatal("idempotent activation destroyed the rollback target")
	}
}

func TestActivateThenRollbackSwapsGenerations(t *testing.T) {
	first, second := twoGenerations(t)
	activation, _, err := Activate(nil, first)
	if err != nil {
		t.Fatalf("Activate first: %v", err)
	}
	activation, plan, err := Activate(&activation, second)
	if err != nil {
		t.Fatalf("Activate second: %v", err)
	}
	assertFields(t, plan.RestartRequired, []Field{FieldProvider})
	assertFields(t, plan.Hot, []Field{FieldModel})

	rolled, rollbackPlan, err := Rollback(activation)
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if rolled.Active.Digest != first.Digest {
		t.Fatal("rollback did not restore the previous generation")
	}
	if rolled.Generation != 3 {
		t.Fatalf("generation = %d, want 3 (rollback is an applied transition)", rolled.Generation)
	}
	if rolled.Previous == nil || rolled.Previous.Digest != second.Digest {
		t.Fatal("rollback did not retain the rolled-back-from configuration")
	}
	assertFields(t, rollbackPlan.RestartRequired, []Field{FieldProvider})
	assertFields(t, rollbackPlan.Hot, []Field{FieldModel})

	// The swap makes a mistaken rollback itself reversible.
	redone, _, err := Rollback(rolled)
	if err != nil {
		t.Fatalf("Rollback twice: %v", err)
	}
	if redone.Active.Digest != second.Digest || redone.Generation != 4 {
		t.Fatalf("redo digest = %q generation = %d", redone.Active.Digest, redone.Generation)
	}
}

func TestRollbackWithoutTargetFailsClosed(t *testing.T) {
	config, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	activation, _, err := InitialActivation(config)
	if err != nil {
		t.Fatalf("InitialActivation: %v", err)
	}
	rolled, plan, err := Rollback(activation)
	assertErrorCode(t, err, ErrNoRollbackTarget, Field("previous"))
	if rolled.Generation != 0 || rolled.Active.Digest != "" || !plan.IsEmpty() {
		t.Fatal("failed rollback returned partial state")
	}
}

func TestActivationFailsClosedOnTamperedState(t *testing.T) {
	first, second := twoGenerations(t)
	activation, _, err := Activate(nil, first)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	activation, _, err = Activate(&activation, second)
	if err != nil {
		t.Fatalf("Activate second: %v", err)
	}

	tamperedCandidate := cloneEffective(second)
	tamperedCandidate.Values.Model = ptr(ModelID("model-a"))
	_, _, err = Activate(&activation, tamperedCandidate)
	assertErrorCode(t, err, ErrInvalidParentDigest, Field("digest"))

	tamperedActive := cloneActivation(activation)
	tamperedActive.Active.Delegable[FieldCLIKind] = true
	if _, _, err := Activate(&tamperedActive, second); err == nil {
		t.Fatal("Activate accepted a tampered active generation")
	}
	if _, _, err := Rollback(tamperedActive); err == nil {
		t.Fatal("Rollback accepted a tampered active generation")
	}

	tamperedPrevious := cloneActivation(activation)
	tamperedPrevious.Previous.Values.Limits.MaxToolCalls = ptr(int64(1))
	_, _, err = Rollback(tamperedPrevious)
	assertErrorCode(t, err, ErrInvalidParentDigest, Field("digest"))

	unversioned := cloneActivation(activation)
	unversioned.Active.Version = Version("v999")
	_, _, err = Activate(&unversioned, second)
	assertErrorCode(t, err, ErrInvalidVersion, Field("schema_version"))
}

func TestActivationSharesNoMutableStateWithCaller(t *testing.T) {
	first, second := twoGenerations(t)
	activation, _, err := Activate(nil, first)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	activation, plan, err := Activate(&activation, second)
	if err != nil {
		t.Fatalf("Activate second: %v", err)
	}

	if origin := activation.Active.Origins[FieldModel]; origin != SourcePlatform {
		t.Fatalf("test precondition failed: model origin = %q, want platform", origin)
	}
	second.Delegable[FieldCLIKind] = true
	second.Origins[FieldModel] = SourceTask
	if activation.Active.Delegable[FieldCLIKind] {
		t.Fatal("activation shares the candidate delegability map")
	}
	if activation.Active.Origins[FieldModel] != SourcePlatform {
		t.Fatal("activation shares the candidate origins map")
	}

	plan.Hot[0] = Field("clobbered")
	plan.RestartRequired[0] = Field("clobbered")
	assertFields(t, activation.LastPlan.Hot, []Field{FieldModel})
	assertFields(t, activation.LastPlan.RestartRequired, []Field{FieldProvider})

	if err := ValidateEffective(activation.Active); err != nil {
		t.Fatalf("activation state stopped validating: %v", err)
	}
}

func TestDelegabilityOnlyChangeIsHotAndStillANewGeneration(t *testing.T) {
	current, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve current: %v", err)
	}
	changedInput := fullInput()
	changedInput.Platform.Delegability[FieldCLIKind] = true
	next, err := Resolve(changedInput)
	if err != nil {
		t.Fatalf("Resolve next: %v", err)
	}
	if current.Digest == next.Digest {
		t.Fatal("test precondition failed: policy change must move the digest")
	}

	plan, err := ClassifyChanges(current, next)
	if err != nil {
		t.Fatalf("ClassifyChanges: %v", err)
	}
	if len(plan.Hot) != 0 || len(plan.RestartRequired) != 0 {
		t.Fatalf("policy-only change moved field values: %#v", plan)
	}
	if !plan.DelegabilityChanged {
		t.Fatal("policy-only change was not reported")
	}
	if plan.RequiresRestart() {
		t.Fatal("policy-only change demanded a restart")
	}
	if plan.IsEmpty() {
		t.Fatal("policy-only change reported an empty plan")
	}

	activation, _, err := Activate(nil, current)
	if err != nil {
		t.Fatalf("Activate current: %v", err)
	}
	advanced, activatePlan, err := Activate(&activation, next)
	if err != nil {
		t.Fatalf("Activate next: %v", err)
	}
	if advanced.Generation != 2 {
		t.Fatalf("generation = %d, want 2", advanced.Generation)
	}
	if !activatePlan.DelegabilityChanged || activatePlan.RequiresRestart() {
		t.Fatalf("policy-only activation plan = %#v", activatePlan)
	}
}

// capabilitiesModelBAcceptsMedium lets model-b accept the same reasoning
// effort as model-a, so a fixture can change only the fields under test
// instead of dragging reasoning_effort into every change plan.
func capabilitiesModelBAcceptsMedium() ProviderCapabilities {
	capabilities := testCapabilities()
	model := capabilities.Models[ModelID("model-b")]
	model.ReasoningEfforts = []ReasoningEffort{"low", "medium", "high"}
	capabilities.Models[ModelID("model-b")] = model
	return capabilities
}

// twoGenerations returns two valid effective configurations differing in one
// restart-required field (provider) and one hot field (model).
func twoGenerations(t *testing.T) (Effective, Effective) {
	t.Helper()
	first, err := Resolve(fullInput())
	if err != nil {
		t.Fatalf("Resolve first: %v", err)
	}
	nextInput := fullInput()
	nextInput.Platform.Values.Provider = ptr(ProviderID("provider-b"))
	nextInput.Platform.Values.Model = ptr(ModelID("model-b"))
	nextInput.Capabilities = capabilitiesModelBAcceptsMedium()
	nextInput.Capabilities.Provider = "provider-b"
	second, err := Resolve(nextInput)
	if err != nil {
		t.Fatalf("Resolve second: %v", err)
	}
	return first, second
}

func assertErrorCode(t *testing.T, err error, code ErrorCode, field Field) {
	t.Helper()
	if err == nil {
		t.Fatalf("error is nil, want %s on %s", code, field)
	}
	var failures ValidationErrors
	if !errors.As(err, &failures) {
		t.Fatalf("error type = %T, want ValidationErrors: %v", err, err)
	}
	for _, failure := range failures {
		if failure.Code == code && failure.Field == field {
			return
		}
	}
	t.Fatalf("missing error %s on %s in %#v", code, field, failures)
}

func assertFields(t *testing.T, got, want []Field) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("fields = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fields = %#v, want %#v", got, want)
		}
	}
}

// --- cross-contract conformance --------------------------------------------
//
// These tests pin this package against the two accepted downstream contracts
// rather than against its own prior behaviour. The literals below are copied
// from those contracts on purpose: storage constrains apply_class to
// ('hot', 'restart') in migrations 130 and 134, and the accepted client
// contract declares the same two-member union. If either contract is ever
// reopened, these literals are the intended failure point.

// acceptedApplyClasses is the exact apply_class domain of the accepted
// storage and client contracts.
func acceptedApplyClasses() map[string]struct{} {
	return map[string]struct{}{"hot": {}, "restart": {}}
}

func TestReloadClassMatchesAcceptedApplyClassVocabulary(t *testing.T) {
	accepted := acceptedApplyClasses()

	if string(ReloadHot) != "hot" {
		t.Fatalf("ReloadHot = %q, want %q", ReloadHot, "hot")
	}
	if string(ReloadRestartRequired) != "restart" {
		t.Fatalf("ReloadRestartRequired = %q, want %q", ReloadRestartRequired, "restart")
	}

	// The restart class must not carry the acknowledgement-status spelling.
	// That value is a legal member of a different enum in migration 134, so a
	// regression here would be silently accepted by that column and rejected
	// by the apply_class column.
	if string(ReloadRestartRequired) == "restart_required" {
		t.Fatal("restart apply class must not reuse the acknowledgement status spelling")
	}

	// Every classification this package can emit must be storable.
	for _, spec := range fieldRegistry {
		if _, ok := accepted[string(spec.reload)]; !ok {
			t.Fatalf("field %q classifies as %q, outside the accepted apply_class domain", spec.field, spec.reload)
		}
	}
	for _, field := range allFields {
		class, ok := ClassifyField(field)
		if !ok {
			t.Fatalf("field %q has no reload classification", field)
		}
		if _, ok := accepted[string(class)]; !ok {
			t.Fatalf("field %q classifies as %q, outside the accepted apply_class domain", field, class)
		}
	}
}

func TestApplyClassCollapsesPlanToOneAcceptedValue(t *testing.T) {
	cases := []struct {
		name string
		plan ChangePlan
		want ReloadClass
	}{
		{"empty plan", ChangePlan{}, ReloadHot},
		{"hot only", ChangePlan{Hot: []Field{FieldModel}}, ReloadHot},
		{"delegability only", ChangePlan{DelegabilityChanged: true}, ReloadHot},
		{"restart only", ChangePlan{RestartRequired: []Field{FieldProvider}}, ReloadRestartRequired},
		{
			"mixed is restart as a whole",
			ChangePlan{Hot: []Field{FieldModel}, RestartRequired: []Field{FieldProvider}},
			ReloadRestartRequired,
		},
		{
			"restart with delegability change",
			ChangePlan{RestartRequired: []Field{FieldCLIKind}, DelegabilityChanged: true},
			ReloadRestartRequired,
		},
	}

	accepted := acceptedApplyClasses()
	for _, testCase := range cases {
		got := testCase.plan.ApplyClass()
		if got != testCase.want {
			t.Fatalf("%s: ApplyClass() = %q, want %q", testCase.name, got, testCase.want)
		}
		if _, ok := accepted[string(got)]; !ok {
			t.Fatalf("%s: ApplyClass() = %q, outside the accepted apply_class domain", testCase.name, got)
		}
	}
}

func TestApplyClassOfResolvedTransitionIsStorable(t *testing.T) {
	current, next := twoGenerations(t)

	plan, err := ClassifyChanges(current, next)
	if err != nil {
		t.Fatalf("ClassifyChanges: %v", err)
	}
	if !plan.RequiresRestart() {
		t.Fatal("a provider change must require a restart")
	}
	if got := plan.ApplyClass(); got != ReloadRestartRequired {
		t.Fatalf("ApplyClass() = %q, want %q", got, ReloadRestartRequired)
	}
	if _, ok := acceptedApplyClasses()[string(plan.ApplyClass())]; !ok {
		t.Fatalf("ApplyClass() = %q is not storable", plan.ApplyClass())
	}

	// The reverse transition is equally restart-bearing, so a rollback records
	// the same storable apply class rather than a hot no-op.
	reverse, err := ClassifyChanges(next, current)
	if err != nil {
		t.Fatalf("ClassifyChanges reverse: %v", err)
	}
	if got := reverse.ApplyClass(); got != ReloadRestartRequired {
		t.Fatalf("reverse ApplyClass() = %q, want %q", got, ReloadRestartRequired)
	}
}

func TestCapabilityDigestIsRawHexAndOrderIndependent(t *testing.T) {
	digest, err := CapabilityDigest(testCapabilities())
	if err != nil {
		t.Fatalf("CapabilityDigest: %v", err)
	}

	// Same encoding as a configuration digest, so one check covers both and
	// the value satisfies the 64-hex constraint on every storage column that
	// records it.
	if !ValidDigest(digest) {
		t.Fatalf("capability digest is not raw lowercase hex: %q", digest)
	}
	if len(digest) != DigestLength {
		t.Fatalf("capability digest length = %d, want %d", len(digest), DigestLength)
	}
	if strings.Contains(digest, ":") {
		t.Fatalf("capability digest carries an algorithm prefix: %q", digest)
	}

	repeat, err := CapabilityDigest(testCapabilities())
	if err != nil {
		t.Fatalf("CapabilityDigest repeat: %v", err)
	}
	if repeat != digest {
		t.Fatalf("capability digest is unstable: %q then %q", digest, repeat)
	}

	// An equal declaration built in a different order must hash identically,
	// otherwise the digest would depend on how a caller assembled the maps
	// and slices rather than on the declaration itself.
	reordered := testCapabilities()
	modelA := reordered.Models[ModelID("model-a")]
	modelA.ReasoningEfforts = []ReasoningEffort{"high", "low", "medium"}
	reordered.Models = map[ModelID]ModelCapabilities{
		ModelID("model-b"): reordered.Models[ModelID("model-b")],
		ModelID("model-a"): modelA,
	}
	reorderedDigest, err := CapabilityDigest(reordered)
	if err != nil {
		t.Fatalf("CapabilityDigest reordered: %v", err)
	}
	if reorderedDigest != digest {
		t.Fatalf("capability digest depends on input ordering: %q vs %q", digest, reorderedDigest)
	}
}

func TestCapabilityDigestChangesOnMaterialDifference(t *testing.T) {
	baseline, err := CapabilityDigest(testCapabilities())
	if err != nil {
		t.Fatalf("CapabilityDigest baseline: %v", err)
	}

	mutations := map[string]func(*ProviderCapabilities){
		"provider": func(c *ProviderCapabilities) { c.Provider = ProviderID("provider-z") },
		"model removed": func(c *ProviderCapabilities) {
			delete(c.Models, ModelID("model-b"))
		},
		"reasoning effort removed": func(c *ProviderCapabilities) {
			model := c.Models[ModelID("model-a")]
			model.ReasoningEfforts = []ReasoningEffort{"low", "medium"}
			c.Models[ModelID("model-a")] = model
		},
		"max input tokens": func(c *ProviderCapabilities) {
			model := c.Models[ModelID("model-a")]
			model.MaxInputTokens = 7_000
			c.Models[ModelID("model-a")] = model
		},
		"context window": func(c *ProviderCapabilities) {
			model := c.Models[ModelID("model-a")]
			model.ContextWindowTokens = 9_000
			c.Models[ModelID("model-a")] = model
		},
		"max tool calls": func(c *ProviderCapabilities) {
			model := c.Models[ModelID("model-a")]
			model.MaxToolCalls = 21
			c.Models[ModelID("model-a")] = model
		},
	}

	for name, mutate := range mutations {
		capabilities := testCapabilities()
		mutate(&capabilities)
		digest, err := CapabilityDigest(capabilities)
		if err != nil {
			t.Fatalf("%s: CapabilityDigest: %v", name, err)
		}
		if digest == baseline {
			t.Fatalf("%s: capability digest did not change", name)
		}
	}
}

func TestCapabilityDigestFailsClosed(t *testing.T) {
	cases := []struct {
		name         string
		capabilities func() ProviderCapabilities
		code         ErrorCode
		field        Field
	}{
		{
			name: "unversioned declaration",
			capabilities: func() ProviderCapabilities {
				capabilities := testCapabilities()
				capabilities.Version = Version("")
				return capabilities
			},
			code:  ErrInvalidVersion,
			field: Field("capabilities_version"),
		},
		{
			name: "no models",
			capabilities: func() ProviderCapabilities {
				capabilities := testCapabilities()
				capabilities.Models = nil
				return capabilities
			},
			code:  ErrInvalidCapabilities,
			field: FieldModel,
		},
		{
			name: "model bounds exceed the context window",
			capabilities: func() ProviderCapabilities {
				capabilities := testCapabilities()
				model := capabilities.Models[ModelID("model-a")]
				model.MaxInputTokens = model.ContextWindowTokens + 1
				capabilities.Models[ModelID("model-a")] = model
				return capabilities
			},
			code:  ErrInvalidCapabilities,
			field: FieldModel,
		},
	}

	for _, testCase := range cases {
		digest, err := CapabilityDigest(testCase.capabilities())
		if err == nil {
			t.Fatalf("%s: invalid capabilities produced digest %q", testCase.name, digest)
		}
		if digest != "" {
			t.Fatalf("%s: failed call returned digest %q", testCase.name, digest)
		}
		assertErrorCode(t, err, testCase.code, testCase.field)
	}
}
