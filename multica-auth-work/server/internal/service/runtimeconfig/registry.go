package runtimeconfig

// This file is the single frozen source of truth for the v1 field contract.
//
// Every other concern — field enumeration, membership, reload classification,
// typed read/write access, and required-field validation — is derived from
// fieldRegistry rather than restated in a parallel switch. Before this
// registry existed the same eight fields were listed in five places
// (allFields, knownField, ClassifyField, fieldValue, setField), so adding a
// field could compile while silently missing a reload class or an accessor.
//
// Freezing rules for v1:
//
//   - The ordered set of fields is frozen. Adding, removing, renaming, or
//     reordering an entry is a new contract version, not an edit, because the
//     order is observable in ChangePlan and in the canonical digest input.
//   - Every entry must carry a reload class and both accessors. A zero value
//     in either is a programming error, asserted by the registry test.
//   - required marks the fields an *effective* configuration must carry.
//     Individual layers may omit them; only the resolved result must be whole.

// fieldSpec is one frozen registry entry.
type fieldSpec struct {
	field    Field
	reload   ReloadClass
	required bool
	get      func(Values) (any, bool)
	set      func(*Values, any)
}

// fieldRegistry is the frozen, ordered v1 field contract. The order defines
// field enumeration, ChangePlan ordering, and canonical digest input order.
var fieldRegistry = []fieldSpec{
	pointerFieldSpec(FieldTransportBinding, ReloadRestartRequired, true,
		func(v Values) *TransportBinding { return v.TransportBinding },
		func(v *Values, value *TransportBinding) { v.TransportBinding = value }),
	pointerFieldSpec(FieldCLIKind, ReloadRestartRequired, true,
		func(v Values) *CLIKind { return v.CLIKind },
		func(v *Values, value *CLIKind) { v.CLIKind = value }),
	pointerFieldSpec(FieldProvider, ReloadRestartRequired, true,
		func(v Values) *ProviderID { return v.Provider },
		func(v *Values, value *ProviderID) { v.Provider = value }),
	pointerFieldSpec(FieldSubscriptionRef, ReloadRestartRequired, false,
		func(v Values) *string { return v.SubscriptionRef },
		func(v *Values, value *string) { v.SubscriptionRef = value }),
	pointerFieldSpec(FieldProviderCatalog, ReloadRestartRequired, false,
		func(v Values) *string { return v.ProviderCatalogVersion },
		func(v *Values, value *string) { v.ProviderCatalogVersion = value }),
	pointerFieldSpec(FieldCapabilityDigest, ReloadRestartRequired, false,
		func(v Values) *string { return v.CapabilityDigest },
		func(v *Values, value *string) { v.CapabilityDigest = value }),
	pointerFieldSpec(FieldModel, ReloadHot, true,
		func(v Values) *ModelID { return v.Model },
		func(v *Values, value *ModelID) { v.Model = value }),
	pointerFieldSpec(FieldReasoningMode, ReloadHot, false,
		func(v Values) *ReasoningMode { return v.ReasoningMode },
		func(v *Values, value *ReasoningMode) { v.ReasoningMode = value }),
	pointerFieldSpec(FieldReasoningEffort, ReloadHot, false,
		func(v Values) *ReasoningEffort { return v.ReasoningEffort },
		func(v *Values, value *ReasoningEffort) { v.ReasoningEffort = value }),
	pointerFieldSpec(FieldReasoningBudget, ReloadHot, false,
		func(v Values) *int64 { return v.ReasoningBudget },
		func(v *Values, value *int64) { v.ReasoningBudget = value }),
	limitFieldSpec(FieldMaxContextTokens, ReloadHot,
		func(v Values) *int64 { return v.Limits.MaxContextTokens },
		func(v *Values, value *int64) { v.Limits.MaxContextTokens = value }),
	limitFieldSpec(FieldMaxInputTokens, ReloadHot,
		func(v Values) *int64 { return v.Limits.MaxInputTokens },
		func(v *Values, value *int64) { v.Limits.MaxInputTokens = value }),
	limitFieldSpec(FieldMaxOutputTokens, ReloadHot,
		func(v Values) *int64 { return v.Limits.MaxOutputTokens },
		func(v *Values, value *int64) { v.Limits.MaxOutputTokens = value }),
	limitFieldSpec(FieldMaxTotalTokens, ReloadHot,
		func(v Values) *int64 { return v.Limits.MaxTotalTokens },
		func(v *Values, value *int64) { v.Limits.MaxTotalTokens = value }),
	limitFieldSpec(FieldMaxToolCalls, ReloadHot,
		func(v Values) *int64 { return v.Limits.MaxToolCalls },
		func(v *Values, value *int64) { v.Limits.MaxToolCalls = value }),
	limitFieldSpec(FieldWallTimeoutMS, ReloadHot,
		func(v Values) *int64 { return v.Limits.WallTimeoutMS },
		func(v *Values, value *int64) { v.Limits.WallTimeoutMS = value }),
	limitFieldSpec(FieldIdleTimeoutMS, ReloadHot,
		func(v Values) *int64 { return v.Limits.IdleTimeoutMS },
		func(v *Values, value *int64) { v.Limits.IdleTimeoutMS = value }),
	pointerFieldSpec(FieldConcurrency, ReloadHot, false,
		func(v Values) *ConcurrencyPolicy { return v.Concurrency },
		func(v *Values, value *ConcurrencyPolicy) { v.Concurrency = value }),
	pointerFieldSpec(FieldRetry, ReloadHot, false,
		func(v Values) *RetryPolicy { return v.Retry },
		func(v *Values, value *RetryPolicy) { v.Retry = value }),
	pointerFieldSpec(FieldFlags, ReloadRestartRequired, false,
		func(v Values) *FlagPolicy { return v.Flags },
		func(v *Values, value *FlagPolicy) { v.Flags = value }),
	pointerFieldSpec(FieldEnvironment, ReloadRestartRequired, false,
		func(v Values) *EnvironmentPolicy { return v.Environment },
		func(v *Values, value *EnvironmentPolicy) { v.Environment = value }),
	pointerFieldSpec(FieldSkills, ReloadRestartRequired, false,
		func(v Values) *SkillPolicy { return v.Skills },
		func(v *Values, value *SkillPolicy) { v.Skills = value }),
	pointerFieldSpec(FieldMCPTools, ReloadRestartRequired, false,
		func(v Values) *MCPToolPolicy { return v.MCPTools },
		func(v *Values, value *MCPToolPolicy) { v.MCPTools = value }),
	pointerFieldSpec(FieldPermissions, ReloadRestartRequired, false,
		func(v Values) *PermissionPolicy { return v.Permissions },
		func(v *Values, value *PermissionPolicy) { v.Permissions = value }),
	pointerFieldSpec(FieldEligibility, ReloadRestartRequired, false,
		func(v Values) *EligibilityPolicy { return v.Eligibility },
		func(v *Values, value *EligibilityPolicy) { v.Eligibility = value }),
	pointerFieldSpec(FieldHealth, ReloadHot, false,
		func(v Values) *HealthPolicy { return v.Health },
		func(v *Values, value *HealthPolicy) { v.Health = value }),
	pointerFieldSpec(FieldFallback, ReloadRestartRequired, false,
		func(v Values) *FallbackPolicy { return v.Fallback },
		func(v *Values, value *FallbackPolicy) { v.Fallback = value }),
}

func pointerFieldSpec[T any](field Field, reload ReloadClass, required bool, get func(Values) *T, set func(*Values, *T)) fieldSpec {
	return fieldSpec{
		field: field, reload: reload, required: required,
		get: func(v Values) (any, bool) {
			value := get(v)
			if value == nil {
				return nil, false
			}
			return *value, true
		},
		set: func(v *Values, value any) {
			typed := value.(T)
			set(v, &typed)
		},
	}
}

func limitFieldSpec(field Field, reload ReloadClass, get func(Values) *int64, set func(*Values, *int64)) fieldSpec {
	return pointerFieldSpec(field, reload, false, get, set)
}

// allFields is the frozen field order, derived from the registry.
var allFields = derivedAllFields()

// fieldSpecs indexes the registry by field for O(1) membership and lookup.
var fieldSpecs = derivedFieldSpecs()

func derivedAllFields() []Field {
	fields := make([]Field, 0, len(fieldRegistry))
	for _, spec := range fieldRegistry {
		fields = append(fields, spec.field)
	}
	return fields
}

func derivedFieldSpecs() map[Field]fieldSpec {
	specs := make(map[Field]fieldSpec, len(fieldRegistry))
	for _, spec := range fieldRegistry {
		specs[spec.field] = spec
	}
	return specs
}

// Fields returns the frozen v1 field contract in canonical order. The returned
// slice is a copy, so callers cannot mutate the registry.
func Fields() []Field {
	fields := make([]Field, len(allFields))
	copy(fields, allFields)
	return fields
}

// knownField reports whether field belongs to the frozen v1 contract.
func knownField(field Field) bool {
	_, ok := fieldSpecs[field]
	return ok
}

// requiredFields returns the fields an effective configuration must carry.
func requiredFields() []Field {
	fields := make([]Field, 0, len(fieldRegistry))
	for _, spec := range fieldRegistry {
		if spec.required {
			fields = append(fields, spec.field)
		}
	}
	return fields
}

// fieldValue reads one typed field. The bool distinguishes unset from a valid
// zero value.
func fieldValue(values Values, field Field) (any, bool) {
	spec, ok := fieldSpecs[field]
	if !ok {
		return nil, false
	}
	return spec.get(values)
}

// setField writes one typed field. Unknown fields are ignored so a caller
// cannot smuggle an out-of-contract key into Values.
func setField(values *Values, field Field, value any) {
	spec, ok := fieldSpecs[field]
	if !ok {
		return
	}
	spec.set(values, value)
}
