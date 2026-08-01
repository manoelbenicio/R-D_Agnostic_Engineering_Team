package runtimeconfig

import (
	"sort"
	"strings"
	"unicode"
)

type errorCollector struct {
	seen map[string]struct{}
	list ValidationErrors
}

func newErrorCollector() *errorCollector {
	return &errorCollector{seen: make(map[string]struct{})}
}

func (c *errorCollector) add(code ErrorCode, field Field) {
	key := string(code) + "\x00" + string(field)
	if _, ok := c.seen[key]; ok {
		return
	}
	c.seen[key] = struct{}{}
	c.list = append(c.list, ValidationError{Code: code, Field: field})
}

func (c *errorCollector) errors() ValidationErrors {
	sort.Slice(c.list, func(i, j int) bool {
		if c.list[i].Field != c.list[j].Field {
			return c.list[i].Field < c.list[j].Field
		}
		return c.list[i].Code < c.list[j].Code
	})
	return c.list
}

func validateConfig(config *Config, taskLayer bool, errs *errorCollector) {
	if config == nil {
		return
	}
	if config.Version != VersionV1 {
		errs.add(ErrInvalidVersion, Field("schema_version"))
	}
	validateValues(config.Values, false, errs)
	for field := range config.Delegability {
		if !knownField(field) {
			errs.add(ErrUnknownField, field)
		}
	}
	if taskLayer && len(config.Delegability) != 0 {
		errs.add(ErrPolicyNotAllowed, Field("delegability"))
	}
}

// validateValues checks one layer's values. requireCore is set only for a
// resolved effective configuration: individual layers may legally omit a
// required field, but the resolved result may not. The required set is read
// from the frozen registry so it cannot drift from the field contract.
func validateValues(values Values, requireCore bool, errs *errorCollector) {
	if requireCore {
		for _, field := range requiredFields() {
			if _, ok := fieldValue(values, field); !ok {
				errs.add(ErrRequiredField, field)
			}
		}
	}

	if values.TransportBinding != nil {
		switch *values.TransportBinding {
		case TransportOmniRoute, TransportNativeCredentialHome:
		default:
			errs.add(ErrInvalidField, FieldTransportBinding)
		}
	}

	validateIdentifier(values.CLIKind, FieldCLIKind, false, errs)
	validateIdentifier(values.Provider, FieldProvider, false, errs)
	validateIdentifier(values.Model, FieldModel, false, errs)
	validateIdentifier(values.ReasoningEffort, FieldReasoningEffort, false, errs)
	validatePositive(values.Limits.MaxInputTokens, FieldMaxInputTokens, errs)
	validatePositive(values.Limits.MaxOutputTokens, FieldMaxOutputTokens, errs)
	validatePositive(values.Limits.MaxToolCalls, FieldMaxToolCalls, errs)
}

func validateIdentifier[T ~string](value *T, field Field, required bool, errs *errorCollector) {
	if value == nil {
		if required {
			errs.add(ErrRequiredField, field)
		}
		return
	}
	raw := string(*value)
	if raw == "" || raw != strings.TrimSpace(raw) || len(raw) > 256 {
		errs.add(ErrInvalidField, field)
		return
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			errs.add(ErrInvalidField, field)
			return
		}
	}
}

func validatePositive(value *int64, field Field, errs *errorCollector) {
	if value != nil && *value <= 0 {
		errs.add(ErrInvalidField, field)
	}
}

func validateCapabilities(capabilities ProviderCapabilities, errs *errorCollector) {
	if capabilities.Version != VersionV1 {
		errs.add(ErrInvalidVersion, Field("capabilities_version"))
	}
	validateIdentifier(&capabilities.Provider, FieldProvider, true, errs)
	if len(capabilities.Models) == 0 {
		errs.add(ErrInvalidCapabilities, FieldModel)
		return
	}
	for model, capability := range capabilities.Models {
		modelCopy := model
		validateIdentifier(&modelCopy, FieldModel, true, errs)
		if capability.MaxInputTokens <= 0 || capability.MaxOutputTokens <= 0 || capability.ContextWindowTokens <= 0 || capability.MaxToolCalls < 0 {
			errs.add(ErrInvalidCapabilities, FieldModel)
		}
		if capability.MaxInputTokens > capability.ContextWindowTokens || capability.MaxOutputTokens > capability.ContextWindowTokens {
			errs.add(ErrInvalidCapabilities, FieldModel)
		}
		seenReasoning := make(map[ReasoningEffort]struct{}, len(capability.ReasoningEfforts))
		for _, effort := range capability.ReasoningEfforts {
			effortCopy := effort
			validateIdentifier(&effortCopy, FieldReasoningEffort, true, errs)
			if _, ok := seenReasoning[effort]; ok {
				errs.add(ErrInvalidCapabilities, FieldReasoningEffort)
			}
			seenReasoning[effort] = struct{}{}
		}
	}
}

func validateEffectiveCapabilities(values Values, capabilities ProviderCapabilities, errs *errorCollector) {
	if values.Provider == nil || values.Model == nil {
		return
	}
	if *values.Provider != capabilities.Provider {
		errs.add(ErrProviderMismatch, FieldProvider)
		return
	}
	modelCapabilities, ok := capabilities.Models[*values.Model]
	if !ok {
		errs.add(ErrUnsupportedModel, FieldModel)
		return
	}
	if values.ReasoningEffort != nil && !containsReasoning(modelCapabilities.ReasoningEfforts, *values.ReasoningEffort) {
		errs.add(ErrUnsupportedReasoning, FieldReasoningEffort)
	}
	validateMaximum(values.Limits.MaxInputTokens, modelCapabilities.MaxInputTokens, FieldMaxInputTokens, errs)
	validateMaximum(values.Limits.MaxOutputTokens, modelCapabilities.MaxOutputTokens, FieldMaxOutputTokens, errs)
	validateMaximum(values.Limits.MaxToolCalls, modelCapabilities.MaxToolCalls, FieldMaxToolCalls, errs)

	if values.Limits.MaxInputTokens != nil && values.Limits.MaxOutputTokens != nil {
		input := *values.Limits.MaxInputTokens
		output := *values.Limits.MaxOutputTokens
		if input > 0 && output > 0 && modelCapabilities.ContextWindowTokens > 0 &&
			(input > modelCapabilities.ContextWindowTokens || output > modelCapabilities.ContextWindowTokens-input) {
			errs.add(ErrContextWindowExceeded, Field("token_budget"))
		}
	}
}

func validateMaximum(value *int64, maximum int64, field Field, errs *errorCollector) {
	if value != nil && (*value > maximum || maximum == 0) {
		errs.add(ErrLimitExceeded, field)
	}
}

func containsReasoning(supported []ReasoningEffort, requested ReasoningEffort) bool {
	for _, effort := range supported {
		if effort == requested {
			return true
		}
	}
	return false
}

// knownField and the field accessors live in registry.go.
