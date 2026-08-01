package runtimeconfig

// Resolve validates and resolves platform, standard, runtime, and task layers.
// It returns no partial configuration on any validation failure.
func Resolve(input ResolveInput) (Effective, error) {
	errs := newErrorCollector()
	validateConfig(input.Platform, false, errs)
	validateConfig(input.Standard, false, errs)
	validateConfig(input.Runtime, false, errs)
	validateConfig(input.Task, true, errs)
	validateCapabilities(input.Capabilities, errs)

	policy := resolveDelegability(input.Platform, input.Standard, input.Runtime)
	if input.Task != nil {
		for _, field := range setFields(input.Task.Values) {
			if !policy[field] {
				errs.add(ErrTaskFieldNotDelegable, field)
			}
		}
	}

	values, origins := resolveValues(
		layer{SourcePlatform, input.Platform},
		layer{SourceStandard, input.Standard},
		layer{SourceRuntime, input.Runtime},
		layer{SourceTask, input.Task},
	)
	validateValues(values, true, errs)
	validateEffectiveCapabilities(values, input.Capabilities, errs)
	if failures := errs.errors(); len(failures) != 0 {
		return Effective{}, failures
	}
	return makeEffective(values, origins, policy), nil
}

// ResolveSubagent inherits every parent effective field and policy, then
// applies only explicitly delegated child task fields. Unlike normal layer
// precedence, an explicitly delegated child field may replace its inherited
// parent value. Parent integrity and provider capabilities are revalidated.
func ResolveSubagent(parent Effective, task *Config, capabilities ProviderCapabilities) (Effective, error) {
	errs := newErrorCollector()
	if parent.Version != VersionV1 {
		errs.add(ErrInvalidVersion, Field("schema_version"))
	}
	validateValues(parent.Values, true, errs)
	for field := range parent.Delegable {
		if !knownField(field) {
			errs.add(ErrUnknownField, field)
		}
	}
	if !digestMatches(parent.Digest, parent.Values, parent.Delegable) {
		errs.add(ErrInvalidParentDigest, Field("digest"))
	}
	validateConfig(task, true, errs)
	validateCapabilities(capabilities, errs)

	values := cloneValues(parent.Values)
	origins := make(map[Field]Source)
	for _, field := range setFields(values) {
		origins[field] = SourceParent
	}
	if task != nil {
		for _, field := range setFields(task.Values) {
			if !parent.Delegable[field] {
				errs.add(ErrTaskFieldNotDelegable, field)
				continue
			}
			value, _ := fieldValue(task.Values, field)
			setField(&values, field, value)
			origins[field] = SourceTask
		}
	}
	validateValues(values, true, errs)
	validateEffectiveCapabilities(values, capabilities, errs)
	if failures := errs.errors(); len(failures) != 0 {
		return Effective{}, failures
	}
	return makeEffective(values, origins, clonePolicy(parent.Delegable)), nil
}

type layer struct {
	source Source
	config *Config
}

func resolveDelegability(configs ...*Config) map[Field]bool {
	policy := make(map[Field]bool, len(allFields))
	for _, field := range allFields {
		for _, config := range configs {
			if config == nil {
				continue
			}
			if allowed, ok := config.Delegability[field]; ok {
				policy[field] = allowed
				break
			}
		}
	}
	return policy
}

func resolveValues(layers ...layer) (Values, map[Field]Source) {
	var values Values
	origins := make(map[Field]Source)
	for _, field := range allFields {
		for _, current := range layers {
			if current.config == nil {
				continue
			}
			value, ok := fieldValue(current.config.Values, field)
			if !ok {
				continue
			}
			setField(&values, field, value)
			origins[field] = current.source
			break
		}
	}
	return values, origins
}

func makeEffective(values Values, origins map[Field]Source, policy map[Field]bool) Effective {
	effective := Effective{
		Version:   VersionV1,
		Values:    cloneValues(values),
		Origins:   cloneOrigins(origins),
		Delegable: clonePolicy(policy),
	}
	effective.Digest = digestFor(effective.Values, effective.Delegable)
	return effective
}

func setFields(values Values) []Field {
	fields := make([]Field, 0, len(allFields))
	for _, field := range allFields {
		if _, ok := fieldValue(values, field); ok {
			fields = append(fields, field)
		}
	}
	return fields
}

func cloneValues(values Values) Values {
	var clone Values
	for _, field := range setFields(values) {
		value, _ := fieldValue(values, field)
		setField(&clone, field, value)
	}
	return clone
}

func clonePolicy(policy map[Field]bool) map[Field]bool {
	clone := make(map[Field]bool, len(policy))
	for field, allowed := range policy {
		clone[field] = allowed
	}
	return clone
}

func cloneOrigins(origins map[Field]Source) map[Field]Source {
	clone := make(map[Field]Source, len(origins))
	for field, source := range origins {
		clone[field] = source
	}
	return clone
}

// cloneEffective deep-copies an effective configuration so activation state
// never shares a mutable map with a caller.
func cloneEffective(effective Effective) Effective {
	return Effective{
		Version:   effective.Version,
		Values:    cloneValues(effective.Values),
		Origins:   cloneOrigins(effective.Origins),
		Delegable: clonePolicy(effective.Delegable),
		Digest:    effective.Digest,
	}
}
