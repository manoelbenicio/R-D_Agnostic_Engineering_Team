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
		errs.add(ErrInvalidVersion, Field("version"))
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
	validateIdentifier(values.SubscriptionRef, FieldSubscriptionRef, false, errs)
	validateIdentifier(values.ProviderCatalogVersion, FieldProviderCatalog, false, errs)
	if values.CapabilityDigest != nil && !ValidDigest(*values.CapabilityDigest) {
		errs.add(ErrInvalidField, FieldCapabilityDigest)
	}
	validateIdentifier(values.Model, FieldModel, false, errs)
	validateIdentifier(values.ReasoningMode, FieldReasoningMode, false, errs)
	validateIdentifier(values.ReasoningEffort, FieldReasoningEffort, false, errs)
	validatePositive(values.ReasoningBudget, FieldReasoningBudget, errs)
	validatePositive(values.Limits.MaxContextTokens, FieldMaxContextTokens, errs)
	validatePositive(values.Limits.MaxInputTokens, FieldMaxInputTokens, errs)
	validatePositive(values.Limits.MaxOutputTokens, FieldMaxOutputTokens, errs)
	validatePositive(values.Limits.MaxTotalTokens, FieldMaxTotalTokens, errs)
	validatePositive(values.Limits.MaxToolCalls, FieldMaxToolCalls, errs)
	validatePositive(values.Limits.WallTimeoutMS, FieldWallTimeoutMS, errs)
	validatePositive(values.Limits.IdleTimeoutMS, FieldIdleTimeoutMS, errs)
	validateExpandedPolicies(values, errs)
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

func validateExpandedPolicies(values Values, errs *errorCollector) {
	if p := values.Concurrency; p != nil {
		for _, value := range []*int64{p.MaxSessions, p.MaxTasks, p.MaxSubagents, p.MaxQueue} {
			validatePositive(value, FieldConcurrency, errs)
		}
	}
	if p := values.Retry; p != nil {
		for _, value := range []*int64{p.MaxAttempts, p.BackoffMS, p.MaxBackoffMS, p.DeadlineBudgetMS} {
			validatePositive(value, FieldRetry, errs)
		}
		if p.JitterPercent != nil && (*p.JitterPercent < 0 || *p.JitterPercent > 100) {
			errs.add(ErrInvalidField, FieldRetry)
		}
		validateStringList(p.RetryableClasses, FieldRetry, errs)
	}
	if p := values.Flags; p != nil {
		for _, flag := range p.Ordered {
			lower := strings.ToLower(flag)
			if flag == "" || flag != strings.TrimSpace(flag) || len(flag) > 512 || strings.ContainsAny(flag, "\r\n\x00") ||
				strings.Contains(lower, "/home/") || strings.Contains(lower, "credential") || strings.Contains(lower, "password") || strings.Contains(lower, "token=") {
				errs.add(ErrInvalidField, FieldFlags)
			}
		}
	}
	if p := values.Environment; p != nil {
		seen := make(map[string]struct{}, len(p.Entries))
		for _, entry := range p.Entries {
			lower := strings.ToLower(entry.Key)
			_, duplicate := seen[entry.Key]
			seen[entry.Key] = struct{}{}
			if !validEnvironmentKey(entry.Key) || duplicate || (entry.Value == nil) == (entry.Reference == nil) ||
				strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "credential") {
				errs.add(ErrInvalidField, FieldEnvironment)
			}
			if entry.Value != nil && (len(*entry.Value) > 1024 || strings.ContainsAny(*entry.Value, "\x00\r\n")) {
				errs.add(ErrInvalidField, FieldEnvironment)
			}
			if entry.Reference != nil {
				validateIdentifier(entry.Reference, FieldEnvironment, true, errs)
			}
		}
	}
	if p := values.Skills; p != nil {
		for _, ref := range p.Allowed {
			validateVersionedRef(ref, FieldSkills, errs)
		}
	}
	if p := values.MCPTools; p != nil {
		for _, ref := range p.Allowed {
			validateIdentifier(&ref.ServerID, FieldMCPTools, true, errs)
			validateIdentifier(&ref.ToolID, FieldMCPTools, true, errs)
			validateIdentifier(&ref.Scope, FieldMCPTools, true, errs)
			validateIdentifier(&ref.Version, FieldMCPTools, true, errs)
			validatePositive(ref.TimeoutMS, FieldMCPTools, errs)
		}
	}
	if p := values.Permissions; p != nil {
		validateStringList(p.FilesystemPolicies, FieldPermissions, errs)
		validateStringList(p.NetworkPolicies, FieldPermissions, errs)
		validateStringList(p.ProcessGrants, FieldPermissions, errs)
		validateStringList(p.ToolGrants, FieldPermissions, errs)
	}
	if p := values.Eligibility; p != nil {
		validateStringList(p.Providers, FieldEligibility, errs)
		validateStringList(p.RuntimeKinds, FieldEligibility, errs)
		validateStringList(p.DaemonClasses, FieldEligibility, errs)
		validateStringList(p.WorkspaceClasses, FieldEligibility, errs)
		validateStringList(p.RequiredCapabilities, FieldEligibility, errs)
	}
	if p := values.Health; p != nil {
		validatePositive(p.FreshnessTTLMS, FieldHealth, errs)
		if p.ReadinessPercent != nil && (*p.ReadinessPercent < 0 || *p.ReadinessPercent > 100) {
			errs.add(ErrInvalidField, FieldHealth)
		}
		if p.CircuitState != "" {
			validateIdentifier(&p.CircuitState, FieldHealth, false, errs)
		}
		if p.ProbeClass != "" {
			validateIdentifier(&p.ProbeClass, FieldHealth, false, errs)
		}
	}
	if p := values.Fallback; p != nil {
		for _, route := range p.Routes {
			validateIdentifier(&route.RouteID, FieldFallback, true, errs)
			validateStringList(route.FailureClasses, FieldFallback, errs)
			validatePositive(route.MaxAttempts, FieldFallback, errs)
		}
	}
}

func validateVersionedRef(ref VersionedRef, field Field, errs *errorCollector) {
	validateIdentifier(&ref.ID, field, true, errs)
	validateIdentifier(&ref.Version, field, true, errs)
	if !ValidDigest(ref.Digest) {
		errs.add(ErrInvalidField, field)
	}
}

func validateStringList(values []string, field Field, errs *errorCollector) {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		copy := value
		_, duplicate := seen[value]
		seen[value] = struct{}{}
		validateIdentifier(&copy, field, true, errs)
		if duplicate || strings.Contains(value, "/") || strings.Contains(value, "\\") {
			errs.add(ErrInvalidField, field)
		}
	}
}

func validEnvironmentKey(key string) bool {
	if key == "" || len(key) > 128 {
		return false
	}
	for i, r := range key {
		if r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
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
	if values.ProviderCatalogVersion != nil && *values.ProviderCatalogVersion != string(capabilities.Version) {
		errs.add(ErrInvalidCapabilities, FieldProviderCatalog)
	}
	if values.CapabilityDigest != nil {
		digest, err := CapabilityDigest(capabilities)
		if err != nil || *values.CapabilityDigest != digest {
			errs.add(ErrInvalidCapabilities, FieldCapabilityDigest)
		}
	}
	validateMaximum(values.Limits.MaxContextTokens, modelCapabilities.ContextWindowTokens, FieldMaxContextTokens, errs)
	validateMaximum(values.Limits.MaxInputTokens, modelCapabilities.MaxInputTokens, FieldMaxInputTokens, errs)
	validateMaximum(values.Limits.MaxOutputTokens, modelCapabilities.MaxOutputTokens, FieldMaxOutputTokens, errs)
	validateMaximum(values.Limits.MaxTotalTokens, modelCapabilities.ContextWindowTokens, FieldMaxTotalTokens, errs)
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
