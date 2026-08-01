package runtimeconfig

// ClassifyField returns the reload behavior for a known field. The class is
// read from the frozen registry, so a newly registered field cannot ship
// without a reload class.
func ClassifyField(field Field) (ReloadClass, bool) {
	spec, ok := fieldSpecs[field]
	if !ok {
		return "", false
	}
	return spec.reload, true
}

// ValidateEffective checks that an effective configuration is internally
// consistent and digest-anchored. It is the single integrity gate used by
// change classification and by activation, so every entry point applies the
// same fail-closed rules: frozen schema version, whole value set, in-contract
// delegability keys, and a raw lowercase hex digest that matches the canonical
// form. Provider capabilities are intentionally not re-checked here because an
// Effective does not carry them; capability admission happens once, in Resolve
// or ResolveSubagent, and the digest binds that decision.
func ValidateEffective(effective Effective) error {
	errs := newErrorCollector()
	validateEffectiveIntegrity(effective, errs)
	if failures := errs.errors(); len(failures) != 0 {
		return failures
	}
	return nil
}

// ClassifyChanges compares two validated effective configurations and returns
// changed fields in frozen registry order. Tampered or unsupported effective
// values fail closed. A delegability-only difference yields an empty field
// partition with DelegabilityChanged set: the digest moves, so it is a real
// transition, but no field value has to be pushed to a running process.
func ClassifyChanges(current, next Effective) (ChangePlan, error) {
	errs := newErrorCollector()
	validateEffectiveIntegrity(current, errs)
	validateEffectiveIntegrity(next, errs)
	if failures := errs.errors(); len(failures) != 0 {
		return ChangePlan{}, failures
	}

	var plan ChangePlan
	for _, field := range allFields {
		left, leftSet := fieldValue(current.Values, field)
		right, rightSet := fieldValue(next.Values, field)
		if leftSet == rightSet && (!leftSet || left == right) {
			continue
		}
		class, _ := ClassifyField(field)
		if class == ReloadRestartRequired {
			plan.RestartRequired = append(plan.RestartRequired, field)
		} else {
			plan.Hot = append(plan.Hot, field)
		}
	}
	plan.DelegabilityChanged = policyDiffers(current.Delegable, next.Delegable)
	return plan, nil
}

// policyDiffers compares effective delegability over the frozen field set. An
// absent key and an explicit false are the same effective policy, matching how
// the digest canonicalises policy.
func policyDiffers(current, next map[Field]bool) bool {
	for _, field := range allFields {
		if current[field] != next[field] {
			return true
		}
	}
	return false
}

func validateEffectiveIntegrity(effective Effective, errs *errorCollector) {
	if effective.Version != VersionV1 {
		errs.add(ErrInvalidVersion, Field("schema_version"))
	}
	validateValues(effective.Values, true, errs)
	for field := range effective.Delegable {
		if !knownField(field) {
			errs.add(ErrUnknownField, field)
		}
	}
	if !digestMatches(effective.Digest, effective.Values, effective.Delegable) {
		errs.add(ErrInvalidParentDigest, Field("digest"))
	}
}
