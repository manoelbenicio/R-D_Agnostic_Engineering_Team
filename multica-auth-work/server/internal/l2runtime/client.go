package l2runtime

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const ContractVersion = "rpp.l2.v1"

var (
	ErrNonLoopbackEndpoint   = errors.New("l2 runtime endpoint must be loopback")
	ErrInvalidControlRequest = errors.New("l2 runtime control request failed validation")
	ErrInvalidEvent          = errors.New("l2 runtime event failed validation")
	ErrSecretEvent           = errors.New("l2 runtime event declares secrets_present")
)

var accountProfileAuthModes = map[string]struct{}{
	"oauth_profile":    {},
	"api_key":          {},
	"cloud_iam":        {},
	"cli_native_store": {},
	"google_signin":    {},
}

var killSwitchFeatures = map[string]struct{}{
	"runtime_proxy":   {},
	"gateway":         {},
	"smart_context":   {},
	"auto_redeem":     {},
	"provider_bridge": {},
}

var killSwitchStates = map[string]struct{}{
	"disabled": {},
	"enabled":  {},
}

var killSwitchEffectiveAt = map[string]struct{}{
	"immediate":                {},
	"next_request":             {},
	"session_restart_required": {},
}

var runtimeEventTypes = map[string]struct{}{
	"sidecar_started":     {},
	"sidecar_ready":       {},
	"policy_applied":      {},
	"account_registered":  {},
	"session_started":     {},
	"session_stopped":     {},
	"selection":           {},
	"affinity":            {},
	"fallback":            {},
	"redeem_attempt":      {},
	"redeem_result":       {},
	"rewrite_decision":    {},
	"spend_savings":       {},
	"guardrail":           {},
	"quota_snapshot":      {},
	"gateway_request":     {},
	"gateway_response":    {},
	"kill_switch_applied": {},
	"error":               {},
}

var runtimeEventSeverity = map[string]struct{}{
	"debug":    {},
	"info":     {},
	"warn":     {},
	"error":    {},
	"critical": {},
}

var runtimeEventProducerComponents = map[string]struct{}{
	"sidecar":       {},
	"runtime_proxy": {},
	"gateway":       {},
	"smart_context": {},
	"redeem":        {},
	"policy":        {},
	"event_stream":  {},
}

var runtimeEventTopLevelFields = map[string]struct{}{
	"contract_version":   {},
	"event_id":           {},
	"event_type":         {},
	"occurred_at":        {},
	"severity":           {},
	"producer":           {},
	"tenant_id":          {},
	"workspace_id":       {},
	"task_id":            {},
	"session_id":         {},
	"runtime_session_id": {},
	"runtime_request_id": {},
	"policy_id":          {},
	"provider":           {},
	"profile_id":         {},
	"model":              {},
	"message":            {},
	"selection":          {},
	"affinity":           {},
	"fallback":           {},
	"redeem":             {},
	"rewrite_decision":   {},
	"spend_savings":      {},
	"guardrail":          {},
	"quota_snapshot":     {},
	"error":              {},
	"payload_ref":        {},
	"redaction":          {},
}

var runtimeEventRequiredByType = map[string][]string{
	"session_started":  {"tenant_id", "session_id"},
	"session_stopped":  {"tenant_id", "session_id"},
	"selection":        {"tenant_id", "session_id", "runtime_request_id", "selection"},
	"affinity":         {"tenant_id", "session_id", "runtime_request_id", "affinity"},
	"fallback":         {"tenant_id", "session_id", "runtime_request_id", "fallback"},
	"redeem_attempt":   {"tenant_id", "session_id", "profile_id", "redeem"},
	"redeem_result":    {"tenant_id", "session_id", "profile_id", "redeem"},
	"rewrite_decision": {"tenant_id", "session_id", "runtime_request_id", "rewrite_decision"},
	"spend_savings":    {"tenant_id", "session_id", "runtime_request_id", "spend_savings"},
	"guardrail":        {"tenant_id", "session_id", "guardrail"},
	"quota_snapshot":   {"tenant_id", "session_id", "profile_id", "quota_snapshot"},
	"gateway_request":  {"tenant_id", "session_id"},
	"gateway_response": {"tenant_id", "session_id"},
	"error":            {"error"},
}

var runtimeEventNestedRequiredByType = map[string]map[string][]string{
	"selection": {
		"selection": {"decision_phase", "selected_profile_id", "selected_provider", "reason", "committed"},
	},
	"affinity": {
		"affinity": {"binding_type", "binding_profile_id", "binding_source", "overrode_fresh_selection"},
	},
	"fallback": {
		"fallback": {"phase", "result", "from_profile_id", "reason", "committed"},
	},
	"redeem_attempt": {
		"redeem": {"action", "profile_id", "guard_state", "result"},
	},
	"redeem_result": {
		"redeem": {"action", "profile_id", "guard_state", "result"},
	},
	"rewrite_decision": {
		"rewrite_decision": {"mode", "decision", "fallback_exact", "validation_result"},
	},
	"guardrail": {
		"guardrail": {"guardrail_type", "action"},
	},
	"error": {
		"error": {"code"},
	},
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" {
		return nil, errors.New("l2 runtime base URL is required")
	}
	if token == "" {
		return nil, errors.New("l2 runtime bearer token is required")
	}
	if err := validateLoopbackURL(baseURL); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func GenerateBearerToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate l2 bearer token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

type HealthResponse struct {
	ContractVersion string       `json:"contract_version"`
	Status          string       `json:"status"`
	Sidecar         SidecarBuild `json:"sidecar"`
}

type SidecarBuild struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

type ReadyResponse struct {
	ContractVersion string       `json:"contract_version"`
	Status          string       `json:"status"`
	Checks          []ReadyCheck `json:"checks"`
}

type ReadyCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var out HealthResponse
	if err := c.do(ctx, http.MethodGet, "/healthz", nil, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != ContractVersion || out.Status != "alive" {
		return nil, fmt.Errorf("l2 health failed closed: contract=%q status=%q", out.ContractVersion, out.Status)
	}
	return &out, nil
}

func (c *Client) Ready(ctx context.Context) (*ReadyResponse, error) {
	var out ReadyResponse
	if err := c.do(ctx, http.MethodGet, "/readyz", nil, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != ContractVersion || out.Status != "ready" {
		return nil, fmt.Errorf("l2 readiness failed closed: contract=%q status=%q", out.ContractVersion, out.Status)
	}
	for _, check := range out.Checks {
		if check.Status != "pass" {
			return nil, fmt.Errorf("l2 readiness check %q failed closed with status %q", check.Name, check.Status)
		}
	}
	return &out, nil
}

type ControlEnvelope struct {
	ContractVersion string `json:"contract_version"`
	RequestID       string `json:"request_id"`
	TenantID        string `json:"tenant_id"`
}

type Policy struct {
	ControlEnvelope
	PolicyID             string               `json:"policy_id"`
	Revision             int64                `json:"revision"`
	AllowedProviders     []string             `json:"allowed_providers"`
	AllowedProfiles      []string             `json:"allowed_profiles"`
	Budgets              map[string]int64     `json:"budgets,omitempty"`
	SmartContext         map[string]any       `json:"smart_context,omitempty"`
	AutoRedeem           map[string]any       `json:"auto_redeem,omitempty"`
	Gateway              map[string]any       `json:"gateway,omitempty"`
	ProviderCapabilities []ProviderCapability `json:"provider_capabilities,omitempty"`
	KillSwitches         []KillSwitch         `json:"kill_switches,omitempty"`
}

type ProviderCapability struct {
	Provider         string `json:"provider"`
	LaunchMode       string `json:"launch_mode"`
	AuthMode         string `json:"auth_mode"`
	QuotaMode        string `json:"quota_mode"`
	RotationMode     string `json:"rotation_mode"`
	ContinuationMode string `json:"continuation_mode"`
	SmartContextMode string `json:"smart_context_mode"`
	ResetClaimMode   string `json:"reset_claim_mode"`
	ValidationStatus string `json:"validation_status"`
}

type ApplyPolicyResponse struct {
	ContractVersion string `json:"contract_version"`
	RequestID       string `json:"request_id"`
	PolicyID        string `json:"policy_id"`
	Revision        int64  `json:"revision"`
	Applied         bool   `json:"applied"`
}

func (c *Client) ApplyPolicy(ctx context.Context, policy Policy) (*ApplyPolicyResponse, error) {
	policy.ContractVersion = ContractVersion
	var out ApplyPolicyResponse
	if err := c.do(ctx, http.MethodPost, "/v1/policy/apply", policy, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != ContractVersion || !out.Applied {
		return nil, fmt.Errorf("l2 policy apply failed closed: contract=%q applied=%t", out.ContractVersion, out.Applied)
	}
	return &out, nil
}

type AccountRegistration struct {
	ControlEnvelope
	Profiles []AccountProfile `json:"profiles"`
}

type AccountProfile struct {
	ProfileID     string `json:"profile_id"`
	Provider      string `json:"provider"`
	ProfileHome   string `json:"profile_home"`
	AuthMode      string `json:"auth_mode"`
	Status        string `json:"status"`
	CapabilityRef string `json:"capability_ref"`
}

type RegisterAccountsResponse struct {
	ContractVersion        string   `json:"contract_version"`
	RequestID              string   `json:"request_id"`
	RegisteredProfileCount int      `json:"registered_profile_count"`
	RejectedProfiles       []string `json:"rejected_profiles"`
}

func (c *Client) RegisterAccounts(ctx context.Context, req AccountRegistration) (*RegisterAccountsResponse, error) {
	req.ContractVersion = ContractVersion
	if err := validateAccountRegistration(req); err != nil {
		return nil, err
	}
	var out RegisterAccountsResponse
	if err := c.do(ctx, http.MethodPost, "/v1/accounts/register", req, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != ContractVersion || len(out.RejectedProfiles) > 0 {
		return nil, fmt.Errorf("l2 account registration failed closed: contract=%q rejected=%d", out.ContractVersion, len(out.RejectedProfiles))
	}
	return &out, nil
}

type StartSessionRequest struct {
	ControlEnvelope
	WorkspaceID       string            `json:"workspace_id"`
	TaskID            string            `json:"task_id"`
	SessionID         string            `json:"session_id"`
	PolicyID          string            `json:"policy_id"`
	RequestedProvider string            `json:"requested_provider"`
	RequestedModel    string            `json:"requested_model,omitempty"`
	WorkingDirectory  string            `json:"working_directory"`
	ProfilePool       []string          `json:"profile_pool"`
	Continuation      map[string]string `json:"continuation,omitempty"`
}

type StartSessionResponse struct {
	ContractVersion  string `json:"contract_version"`
	RequestID        string `json:"request_id"`
	RuntimeSessionID string `json:"runtime_session_id"`
	RouterOwner      string `json:"router_owner"`
	EventStreamURL   string `json:"event_stream_url"`
	RuntimeEndpoint  string `json:"runtime_endpoint"`
	RuntimeLogRef    string `json:"runtime_log_ref"`
}

func (c *Client) StartSession(ctx context.Context, req StartSessionRequest) (*StartSessionResponse, error) {
	req.ContractVersion = ContractVersion
	var out StartSessionResponse
	if err := c.do(ctx, http.MethodPost, "/v1/session/start", req, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != ContractVersion || out.RouterOwner != "rust_l2" {
		return nil, fmt.Errorf("l2 start session failed closed: contract=%q router_owner=%q", out.ContractVersion, out.RouterOwner)
	}
	return &out, nil
}

type StopSessionRequest struct {
	ControlEnvelope
	SessionID        string `json:"session_id"`
	RuntimeSessionID string `json:"runtime_session_id"`
	Reason           string `json:"reason"`
}

func (c *Client) StopSession(ctx context.Context, req StopSessionRequest) error {
	req.ContractVersion = ContractVersion
	return c.do(ctx, http.MethodPost, "/v1/session/stop", req, nil)
}

type KillSwitch struct {
	ControlEnvelope
	Scope       KillSwitchScope `json:"scope"`
	Feature     string          `json:"feature"`
	State       string          `json:"state"`
	Reason      string          `json:"reason"`
	EffectiveAt string          `json:"effective_at"`
}

type KillSwitchScope struct {
	Provider  string `json:"provider,omitempty"`
	ProfileID string `json:"profile_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type KillSwitchResponse struct {
	ContractVersion string `json:"contract_version"`
	RequestID       string `json:"request_id"`
	Applied         bool   `json:"applied"`
	EffectiveAt     string `json:"effective_at"`
}

func (c *Client) ApplyKillSwitch(ctx context.Context, req KillSwitch) (*KillSwitchResponse, error) {
	req.ContractVersion = ContractVersion
	if err := validateKillSwitch(req); err != nil {
		return nil, err
	}
	var out KillSwitchResponse
	if err := c.do(ctx, http.MethodPost, "/v1/killswitch/apply", req, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != ContractVersion || !out.Applied {
		return nil, fmt.Errorf("l2 kill switch failed closed: contract=%q applied=%t", out.ContractVersion, out.Applied)
	}
	if _, ok := killSwitchEffectiveAt[out.EffectiveAt]; !ok {
		return nil, fmt.Errorf("l2 kill switch failed closed: effective_at=%q", out.EffectiveAt)
	}
	return &out, nil
}

func validateAccountRegistration(req AccountRegistration) error {
	if req.ContractVersion != ContractVersion {
		return fmt.Errorf("%w: contract_version %q", ErrInvalidControlRequest, req.ContractVersion)
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return fmt.Errorf("%w: request_id is required", ErrInvalidControlRequest)
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return fmt.Errorf("%w: tenant_id is required", ErrInvalidControlRequest)
	}
	if len(req.Profiles) == 0 {
		return fmt.Errorf("%w: at least one profile is required", ErrInvalidControlRequest)
	}
	for i, profile := range req.Profiles {
		if err := validateAccountProfile(profile); err != nil {
			return fmt.Errorf("%w: profile[%d]: %v", ErrInvalidControlRequest, i, err)
		}
	}
	return nil
}

func validateAccountProfile(profile AccountProfile) error {
	if strings.TrimSpace(profile.ProfileID) == "" {
		return errors.New("profile_id is required")
	}
	if strings.TrimSpace(profile.Provider) == "" {
		return errors.New("provider is required")
	}
	if strings.TrimSpace(profile.ProfileHome) == "" {
		return errors.New("profile_home is required")
	}
	authMode := strings.TrimSpace(profile.AuthMode)
	if _, ok := accountProfileAuthModes[authMode]; !ok {
		return fmt.Errorf("auth_mode %q is not accepted", profile.AuthMode)
	}
	if strings.TrimSpace(profile.Status) != "approved" {
		return fmt.Errorf("status %q is not approved", profile.Status)
	}
	if strings.TrimSpace(profile.CapabilityRef) == "" {
		return errors.New("capability_ref is required")
	}
	return nil
}

func validateKillSwitch(req KillSwitch) error {
	if req.ContractVersion != ContractVersion {
		return fmt.Errorf("%w: contract_version %q", ErrInvalidControlRequest, req.ContractVersion)
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return fmt.Errorf("%w: request_id is required", ErrInvalidControlRequest)
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return fmt.Errorf("%w: tenant_id is required", ErrInvalidControlRequest)
	}
	if _, ok := killSwitchFeatures[strings.TrimSpace(req.Feature)]; !ok {
		return fmt.Errorf("%w: feature %q is not accepted", ErrInvalidControlRequest, req.Feature)
	}
	if _, ok := killSwitchStates[strings.TrimSpace(req.State)]; !ok {
		return fmt.Errorf("%w: state %q is not accepted", ErrInvalidControlRequest, req.State)
	}
	if strings.TrimSpace(req.Reason) == "" {
		return fmt.Errorf("%w: reason is required", ErrInvalidControlRequest)
	}
	if _, ok := killSwitchEffectiveAt[strings.TrimSpace(req.EffectiveAt)]; !ok {
		return fmt.Errorf("%w: effective_at %q is not accepted", ErrInvalidControlRequest, req.EffectiveAt)
	}
	return nil
}

type RuntimeEvent struct {
	ContractVersion  string          `json:"contract_version"`
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	OccurredAt       time.Time       `json:"occurred_at"`
	Severity         string          `json:"severity"`
	Producer         json.RawMessage `json:"producer"`
	TenantID         string          `json:"tenant_id,omitempty"`
	WorkspaceID      string          `json:"workspace_id,omitempty"`
	TaskID           string          `json:"task_id,omitempty"`
	SessionID        string          `json:"session_id,omitempty"`
	RuntimeSessionID string          `json:"runtime_session_id,omitempty"`
	RuntimeRequestID string          `json:"runtime_request_id,omitempty"`
	PolicyID         string          `json:"policy_id,omitempty"`
	Provider         string          `json:"provider,omitempty"`
	ProfileID        string          `json:"profile_id,omitempty"`
	Model            string          `json:"model,omitempty"`
	Message          string          `json:"message,omitempty"`
	Redaction        EventRedaction  `json:"redaction"`
	Raw              json.RawMessage `json:"-"`
}

type EventRedaction struct {
	SecretsPresent  bool   `json:"secrets_present"`
	ScrubberVersion string `json:"scrubber_version"`
}

type EventHandler func(context.Context, RuntimeEvent) error

func (c *Client) StreamEvents(ctx context.Context, streamURL string, handle EventHandler) error {
	if handle == nil {
		return errors.New("l2 event handler is required")
	}
	if strings.TrimSpace(streamURL) == "" {
		streamURL = c.baseURL + "/v1/events/stream"
	}
	if err := validateLoopbackURL(streamURL); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("l2 event stream status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var event RuntimeEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return fmt.Errorf("decode l2 runtime event: %w", err)
		}
		event.Raw = append(event.Raw[:0], line...)
		if err := validateRuntimeEvent(line, event); err != nil {
			return err
		}
		if err := handle(ctx, event); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read l2 event stream: %w", err)
	}
	return nil
}

func validateRuntimeEvent(raw []byte, event RuntimeEvent) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("%w: decode object: %w", ErrInvalidEvent, err)
	}
	for field := range fields {
		if _, ok := runtimeEventTopLevelFields[field]; !ok {
			return fmt.Errorf("%w: unknown field %q", ErrInvalidEvent, field)
		}
	}
	for _, field := range []string{"contract_version", "event_id", "event_type", "occurred_at", "severity", "producer", "redaction"} {
		if err := requireRuntimeEventField(fields, field); err != nil {
			return err
		}
	}
	if event.ContractVersion != ContractVersion {
		return fmt.Errorf("%w: contract_version %q", ErrInvalidEvent, event.ContractVersion)
	}
	if _, ok := runtimeEventTypes[event.EventType]; !ok {
		return fmt.Errorf("%w: unknown event_type %q", ErrInvalidEvent, event.EventType)
	}
	if event.EventID = strings.TrimSpace(event.EventID); len(event.EventID) < 8 || len(event.EventID) > 128 {
		return fmt.Errorf("%w: event_id length out of range", ErrInvalidEvent)
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at is required", ErrInvalidEvent)
	}
	if _, ok := runtimeEventSeverity[event.Severity]; !ok {
		return fmt.Errorf("%w: severity %q", ErrInvalidEvent, event.Severity)
	}
	if err := validateRuntimeEventProducer(fields["producer"]); err != nil {
		return err
	}
	if err := validateRuntimeEventRedaction(fields["redaction"]); err != nil {
		return err
	}
	for _, field := range runtimeEventRequiredByType[event.EventType] {
		if err := requireRuntimeEventField(fields, field); err != nil {
			return fmt.Errorf("%w: event_type %q requires %s", err, event.EventType, field)
		}
	}
	for objectName, requiredFields := range runtimeEventNestedRequiredByType[event.EventType] {
		if err := requireRuntimeObjectFields(fields[objectName], objectName, requiredFields...); err != nil {
			return err
		}
	}
	return nil
}

func requireRuntimeEventField(fields map[string]json.RawMessage, field string) error {
	raw, ok := fields[field]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%w: missing %s", ErrInvalidEvent, field)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && strings.TrimSpace(s) == "" {
		return fmt.Errorf("%w: empty %s", ErrInvalidEvent, field)
	}
	return nil
}

func requireRuntimeObjectFields(raw json.RawMessage, objectName string, requiredFields ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("%w: invalid %s: %w", ErrInvalidEvent, objectName, err)
	}
	for _, field := range requiredFields {
		if err := requireRuntimeEventField(fields, field); err != nil {
			return fmt.Errorf("%w: %s.%s", err, objectName, field)
		}
	}
	return nil
}

func validateRuntimeEventProducer(raw json.RawMessage) error {
	if err := requireRuntimeObjectFields(raw, "producer", "plane", "component"); err != nil {
		return err
	}
	var producer struct {
		Plane     string `json:"plane"`
		Component string `json:"component"`
	}
	if err := json.Unmarshal(raw, &producer); err != nil {
		return fmt.Errorf("%w: invalid producer: %w", ErrInvalidEvent, err)
	}
	if producer.Plane != "rust_l2" {
		return fmt.Errorf("%w: producer.plane %q", ErrInvalidEvent, producer.Plane)
	}
	if _, ok := runtimeEventProducerComponents[producer.Component]; !ok {
		return fmt.Errorf("%w: producer.component %q", ErrInvalidEvent, producer.Component)
	}
	return nil
}

func validateRuntimeEventRedaction(raw json.RawMessage) error {
	if err := requireRuntimeObjectFields(raw, "redaction", "secrets_present", "scrubber_version"); err != nil {
		return err
	}
	var redaction EventRedaction
	if err := json.Unmarshal(raw, &redaction); err != nil {
		return fmt.Errorf("%w: invalid redaction: %w", ErrInvalidEvent, err)
	}
	if redaction.SecretsPresent {
		return ErrSecretEvent
	}
	if strings.TrimSpace(redaction.ScrubberVersion) == "" {
		return fmt.Errorf("%w: redaction.scrubber_version is required", ErrInvalidEvent)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	c.authorize(req)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("l2 runtime status %d for %s %s", resp.StatusCode, method, path)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func (c *Client) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
}

func validateLoopbackURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("l2 runtime endpoint scheme %q unsupported", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return ErrNonLoopbackEndpoint
	}
	ip := net.ParseIP(host)
	if ip == nil {
		addrs, err := net.LookupIP(host)
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			if !addr.IsLoopback() {
				return ErrNonLoopbackEndpoint
			}
		}
		return nil
	}
	if !ip.IsLoopback() {
		return ErrNonLoopbackEndpoint
	}
	return nil
}
