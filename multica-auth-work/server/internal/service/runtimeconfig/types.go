// Package runtimeconfig resolves typed runtime configuration without I/O.
package runtimeconfig

// Version identifies the runtime configuration contract.
type Version string

const (
	// VersionV1 is the first runtime configuration contract.
	VersionV1 Version = "v1"
)

// Source identifies the layer that supplied an effective field.
type Source string

const (
	SourcePlatform Source = "platform"
	SourceStandard Source = "standard"
	SourceRuntime  Source = "runtime"
	SourceTask     Source = "task"
	SourceParent   Source = "parent"
)

// Field is a stable configuration field identifier.
type Field string

const (
	FieldTransportBinding Field = "transport_binding"
	FieldCLIKind          Field = "cli_kind"
	FieldProvider         Field = "provider"
	FieldModel            Field = "model"
	FieldReasoningEffort  Field = "reasoning_effort"
	FieldMaxInputTokens   Field = "max_input_tokens"
	FieldMaxOutputTokens  Field = "max_output_tokens"
	FieldMaxToolCalls     Field = "max_tool_calls"
)

// The ordered field set, membership, reload class, and typed accessors all
// live in the frozen registry in registry.go. Nothing in this package may
// restate that list.

// TransportBinding is the exclusive transport selected for a runtime.
type TransportBinding string

const (
	TransportOmniRoute            TransportBinding = "omniroute"
	TransportNativeCredentialHome TransportBinding = "native_credential_home"
)

// CLIKind identifies an approved CLI frontend.
type CLIKind string

// ProviderID identifies the capability authority used to validate a model.
type ProviderID string

// ModelID identifies a model in a provider capability declaration.
type ModelID string

// ReasoningEffort is a provider-declared reasoning level.
type ReasoningEffort string

// Limits contains optional task execution limits. Pointers distinguish unset
// fields from explicit values, including invalid zero values.
type Limits struct {
	MaxInputTokens  *int64 `json:"max_input_tokens,omitempty"`
	MaxOutputTokens *int64 `json:"max_output_tokens,omitempty"`
	MaxToolCalls    *int64 `json:"max_tool_calls,omitempty"`
}

// Values is the typed runtime configuration payload. This contract
// intentionally has no credential, token, cookie, endpoint, path, prompt, or
// arbitrary map field.
type Values struct {
	TransportBinding *TransportBinding `json:"transport_binding,omitempty"`
	CLIKind          *CLIKind          `json:"cli_kind,omitempty"`
	Provider         *ProviderID       `json:"provider,omitempty"`
	Model            *ModelID          `json:"model,omitempty"`
	ReasoningEffort  *ReasoningEffort  `json:"reasoning_effort,omitempty"`
	Limits           Limits            `json:"limits"`
}

// Config is one versioned configuration layer. Delegability is a tri-state
// field policy: an absent key inherits lower policy, true explicitly allows a
// task/subagent value, and false explicitly denies it. Task layers cannot
// declare policy.
type Config struct {
	Version      Version
	Values       Values
	Delegability map[Field]bool
}

// ResolveInput contains the four precedence layers. Nil layers are absent.
// Precedence is always platform, standard, runtime, then task. Task values are
// admitted only where a higher layer explicitly delegates that field.
type ResolveInput struct {
	Platform     *Config
	Standard     *Config
	Runtime      *Config
	Task         *Config
	Capabilities ProviderCapabilities
}

// ModelCapabilities describes the accepted values and hard maxima for one
// model. ContextWindowTokens bounds configured input plus output tokens.
type ModelCapabilities struct {
	ReasoningEfforts    []ReasoningEffort
	MaxInputTokens      int64
	MaxOutputTokens     int64
	ContextWindowTokens int64
	MaxToolCalls        int64
}

// ProviderCapabilities is a versioned provider/model capability declaration.
type ProviderCapabilities struct {
	Version  Version
	Provider ProviderID
	Models   map[ModelID]ModelCapabilities
}

// Effective is a resolved configuration. Digest covers only the canonical
// secret-free effective values and effective delegability policy; origins are
// diagnostic metadata and do not affect identity.
type Effective struct {
	Version   Version
	Values    Values
	Origins   map[Field]Source
	Delegable map[Field]bool
	Digest    string
}

// ReloadClass states whether a changed field may be applied live. The two
// values are the frozen cross-contract apply-class vocabulary and are not
// free-form: storage constrains apply_class to exactly ('hot', 'restart') in
// migrations 130 and 134, and the accepted client contract declares the same
// two-member union. A third value, or a different spelling of either member,
// cannot be persisted or rendered, so the vocabulary is fixed here at the
// single point where it is defined.
//
// The acknowledgement status set in migration 134 separately contains
// 'restart_required'. That is a different column describing what a daemon
// reported after applying a configuration, not the apply class of the
// configuration itself. The two vocabularies must not be conflated.
type ReloadClass string

const (
	ReloadHot             ReloadClass = "hot"
	ReloadRestartRequired ReloadClass = "restart"
)

// ChangePlan is a deterministic partition of changed fields. Hot and
// RestartRequired are reported in frozen registry order. DelegabilityChanged
// records a delegability-policy-only difference: it changes the digest and so
// forms a new activation generation, but it moves no field value and therefore
// never requires a restart.
type ChangePlan struct {
	Hot                 []Field
	RestartRequired     []Field
	DelegabilityChanged bool
}

// RequiresRestart reports whether applying this plan needs a process restart.
func (p ChangePlan) RequiresRestart() bool { return len(p.RestartRequired) != 0 }

// ApplyClass collapses the plan to the single apply class that storage and the
// client contract record per configuration version. Both accepted contracts
// carry exactly one apply_class value, never a per-field partition, so the
// reduction has to happen somewhere; doing it here keeps every caller from
// re-deriving it and prevents a caller from inventing a third value.
//
// A plan that requires a restart for any field is restart as a whole: a
// partially applied configuration is not a state this contract admits. A plan
// that moves no value at all is hot, because a delegability-only transition
// changes the digest and forms a new generation without touching a running
// process.
func (p ChangePlan) ApplyClass() ReloadClass {
	if p.RequiresRestart() {
		return ReloadRestartRequired
	}
	return ReloadHot
}

// IsEmpty reports whether the plan carries no observable change at all.
func (p ChangePlan) IsEmpty() bool {
	return len(p.Hot) == 0 && len(p.RestartRequired) == 0 && !p.DelegabilityChanged
}

// Activation is the activation state of one runtime's configuration. It keeps
// exactly one previous generation so a bad activation can be reverted
// immediately without consulting external storage. Generation increases on
// every accepted transition, including a rollback, so it is a monotonic
// audit counter rather than a version of the configuration itself.
type Activation struct {
	Generation uint64
	Active     Effective
	Previous   *Effective
	LastPlan   ChangePlan
}

// ErrorCode is a stable, bounded validation failure classification.
type ErrorCode string

const (
	ErrInvalidVersion        ErrorCode = "invalid_version"
	ErrRequiredField         ErrorCode = "required_field"
	ErrInvalidField          ErrorCode = "invalid_field"
	ErrUnknownField          ErrorCode = "unknown_field"
	ErrPolicyNotAllowed      ErrorCode = "policy_not_allowed"
	ErrTaskFieldNotDelegable ErrorCode = "task_field_not_delegable"
	ErrInvalidCapabilities   ErrorCode = "invalid_capabilities"
	ErrProviderMismatch      ErrorCode = "provider_mismatch"
	ErrUnsupportedModel      ErrorCode = "unsupported_model"
	ErrUnsupportedReasoning  ErrorCode = "unsupported_reasoning"
	ErrLimitExceeded         ErrorCode = "limit_exceeded"
	ErrContextWindowExceeded ErrorCode = "context_window_exceeded"
	ErrInvalidParentDigest   ErrorCode = "invalid_parent_digest"
	ErrNoRollbackTarget      ErrorCode = "no_rollback_target"
)

// ValidationError omits rejected values so errors are safe for logs and API
// boundaries. Callers get a stable code and field only.
type ValidationError struct {
	Code  ErrorCode
	Field Field
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return "runtime config validation failed: " + string(e.Code)
	}
	return "runtime config validation failed: " + string(e.Code) + " (" + string(e.Field) + ")"
}

// ValidationErrors is a deterministic collection of fail-closed errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "runtime config validation failed"
	}
	return e[0].Error()
}
