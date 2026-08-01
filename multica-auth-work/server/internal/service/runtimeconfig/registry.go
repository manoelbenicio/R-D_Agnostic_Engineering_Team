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
	{
		field:    FieldTransportBinding,
		reload:   ReloadRestartRequired,
		required: true,
		get: func(v Values) (any, bool) {
			if v.TransportBinding == nil {
				return nil, false
			}
			return *v.TransportBinding, true
		},
		set: func(v *Values, value any) {
			typed := value.(TransportBinding)
			v.TransportBinding = &typed
		},
	},
	{
		field:    FieldCLIKind,
		reload:   ReloadRestartRequired,
		required: true,
		get: func(v Values) (any, bool) {
			if v.CLIKind == nil {
				return nil, false
			}
			return *v.CLIKind, true
		},
		set: func(v *Values, value any) {
			typed := value.(CLIKind)
			v.CLIKind = &typed
		},
	},
	{
		field:    FieldProvider,
		reload:   ReloadRestartRequired,
		required: true,
		get: func(v Values) (any, bool) {
			if v.Provider == nil {
				return nil, false
			}
			return *v.Provider, true
		},
		set: func(v *Values, value any) {
			typed := value.(ProviderID)
			v.Provider = &typed
		},
	},
	{
		field:    FieldModel,
		reload:   ReloadHot,
		required: true,
		get: func(v Values) (any, bool) {
			if v.Model == nil {
				return nil, false
			}
			return *v.Model, true
		},
		set: func(v *Values, value any) {
			typed := value.(ModelID)
			v.Model = &typed
		},
	},
	{
		field:  FieldReasoningEffort,
		reload: ReloadHot,
		get: func(v Values) (any, bool) {
			if v.ReasoningEffort == nil {
				return nil, false
			}
			return *v.ReasoningEffort, true
		},
		set: func(v *Values, value any) {
			typed := value.(ReasoningEffort)
			v.ReasoningEffort = &typed
		},
	},
	{
		field:  FieldMaxInputTokens,
		reload: ReloadHot,
		get: func(v Values) (any, bool) {
			if v.Limits.MaxInputTokens == nil {
				return nil, false
			}
			return *v.Limits.MaxInputTokens, true
		},
		set: func(v *Values, value any) {
			typed := value.(int64)
			v.Limits.MaxInputTokens = &typed
		},
	},
	{
		field:  FieldMaxOutputTokens,
		reload: ReloadHot,
		get: func(v Values) (any, bool) {
			if v.Limits.MaxOutputTokens == nil {
				return nil, false
			}
			return *v.Limits.MaxOutputTokens, true
		},
		set: func(v *Values, value any) {
			typed := value.(int64)
			v.Limits.MaxOutputTokens = &typed
		},
	},
	{
		field:  FieldMaxToolCalls,
		reload: ReloadHot,
		get: func(v Values) (any, bool) {
			if v.Limits.MaxToolCalls == nil {
				return nil, false
			}
			return *v.Limits.MaxToolCalls, true
		},
		set: func(v *Values, value any) {
			typed := value.(int64)
			v.Limits.MaxToolCalls = &typed
		},
	},
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
