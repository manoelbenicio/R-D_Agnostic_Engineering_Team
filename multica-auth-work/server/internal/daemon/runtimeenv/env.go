package runtimeenv

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const CodexOmniRouteAPIKeyEnv = brain.ChildEnvOmniRouteAPIKey

type envOrigin uint8

const (
	originInherited envOrigin = iota
	originLocal
	originCustom
	originTrustedLocal
	originTrustedGateway
	originTrustedSecret
)

type environmentEntry struct {
	key    string
	value  string
	origin envOrigin
}

// StableSecret is an opaque, redacting value supplied by the service layer.
// runtimeenv never reads the configured secret file. Formatting the value is
// always redacted; it is revealed only while constructing exec.Cmd.Env output.
type StableSecret struct {
	value string
}

func NewStableSecret(value string) (StableSecret, error) {
	if value == "" {
		return StableSecret{}, fmt.Errorf("stable OmniRoute secret is required")
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return StableSecret{}, fmt.Errorf("stable OmniRoute secret contains an invalid control character")
	}
	return StableSecret{value: value}, nil
}

func (s StableSecret) IsSet() bool { return s.value != "" }

func (s StableSecret) String() string   { return "[REDACTED]" }
func (s StableSecret) GoString() string { return "[REDACTED]" }
func (s StableSecret) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, "[REDACTED]")
}

type Removal struct {
	Key    string
	Reason DenyReason
}

type SanitizationReport struct {
	Removed []Removal
}

// MinimalEnvironment is an intermediate allowlisted inherited environment.
// Values remain private so diagnostics naturally operate on key names only.
type MinimalEnvironment struct {
	entries map[string]environmentEntry
}

func BuildMinimalInherited(inherited []string) (MinimalEnvironment, SanitizationReport, error) {
	entries := make(map[string]environmentEntry)
	report := SanitizationReport{}
	for index, raw := range inherited {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || !validEnvironmentKey(key) {
			// Skip entries with no '=' separator or invalid key names (e.g.
			// exported bash functions like BASH_FUNC_which%% whose key
			// contains characters not accepted by validEnvironmentKey).
			report.Removed = append(report.Removed, Removal{Key: fmt.Sprintf("_malformed_%d", index), Reason: DenyGatewayOverride})
			continue
		}
		canonical := strings.ToUpper(key)
		classification := ClassifyEnvironmentKey(key)
		if classification.Denied {
			report.Removed = append(report.Removed, Removal{Key: key, Reason: classification.Reason})
			continue
		}
		if !isSafeInheritedKey(key) {
			report.Removed = append(report.Removed, Removal{Key: key, Reason: DenyGatewayOverride})
			continue
		}
		entries[canonical] = environmentEntry{key: key, value: value, origin: originInherited}
	}
	sort.Slice(report.Removed, func(i, j int) bool { return report.Removed[i].Key < report.Removed[j].Key })
	return MinimalEnvironment{entries: entries}, report, nil
}

func (e MinimalEnvironment) Keys() []string {
	return sortedEntryKeys(e.entries)
}

type AdapterEnvironment struct {
	CLI          brain.CLIKind
	GatewayRoot  string
	TaskHome     string
	CodexHome    string
	ClineDataDir string
	StableSecret StableSecret
	// TelemetryOTLPLogsEndpoint, when non-empty, enables the official Claude
	// Code OTLP-logs telemetry as TRUSTED child env pointing at the fixed
	// loopback receiver (e.g. http://127.0.0.1:<port>/v1/logs). TelemetryTaskID
	// and TelemetryRequestID are the canonical correlation carried in
	// OTEL_RESOURCE_ATTRIBUTES. All content/prompt/response/tool logging is
	// forced off; user/local/inherited values cannot override these.
	TelemetryOTLPLogsEndpoint string
	TelemetryTaskID           string
	TelemetryRequestID        string
}

// trustedTelemetryEnv returns the EXACT official Claude Code OTLP-logs trusted
// environment for the fixed loopback receiver, or (nil, nil) when telemetry is
// entirely absent (endpoint empty). It is FAIL-CLOSED: a present-but-malformed
// endpoint or unsafe/empty correlation IDs return an error and never enable
// telemetry. The endpoint must be exactly http://127.0.0.1:<port>/v1/logs with
// an explicit port and no userinfo/query/fragment; task/request IDs must match
// the safe bounded correlation charset (which makes raw comma/equals — and thus
// resource-attribute ambiguity — impossible). Content logging is forced off.
func trustedTelemetryEnv(p AdapterEnvironment) (map[string]string, error) {
	ep := strings.TrimSpace(p.TelemetryOTLPLogsEndpoint)
	if ep == "" {
		// Off ONLY when telemetry is entirely absent. IDs present without an
		// endpoint is a partial/misconfigured state and must fail closed.
		if strings.TrimSpace(p.TelemetryTaskID) != "" || strings.TrimSpace(p.TelemetryRequestID) != "" {
			return nil, fmt.Errorf("telemetry partial config: correlation ids set without an endpoint")
		}
		return nil, nil
	}
	u, err := url.Parse(ep)
	if err != nil {
		return nil, fmt.Errorf("telemetry endpoint is not a valid URL")
	}
	if u.Scheme != "http" {
		return nil, fmt.Errorf("telemetry endpoint must use the http scheme")
	}
	if u.User != nil {
		return nil, fmt.Errorf("telemetry endpoint must not contain userinfo")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("telemetry endpoint must not contain a query or fragment")
	}
	if u.Hostname() != "127.0.0.1" {
		return nil, fmt.Errorf("telemetry endpoint host must be the loopback IP 127.0.0.1")
	}
	port := u.Port()
	if port == "" {
		return nil, fmt.Errorf("telemetry endpoint must specify an explicit port")
	}
	if n, perr := strconv.Atoi(port); perr != nil || n < 1 || n > 65535 {
		return nil, fmt.Errorf("telemetry endpoint port is invalid")
	}
	if u.Path != "/v1/logs" {
		return nil, fmt.Errorf("telemetry endpoint path must be exactly /v1/logs")
	}
	if !safeCorrelationValue(p.TelemetryTaskID) || !safeCorrelationValue(p.TelemetryRequestID) {
		return nil, fmt.Errorf("telemetry correlation ids must be non-empty and safe")
	}
	return map[string]string{
		"CLAUDE_CODE_ENABLE_TELEMETRY":     "1",
		"OTEL_LOGS_EXPORTER":               "otlp",
		"OTEL_METRICS_EXPORTER":            "none",
		"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL": "http/json",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": ep,
		"OTEL_RESOURCE_ATTRIBUTES":         "agent_brain.task_id=" + p.TelemetryTaskID + ",agent_brain.request_id=" + p.TelemetryRequestID,
		"OTEL_LOG_USER_PROMPTS":            "0",
		"OTEL_LOG_ASSISTANT_RESPONSES":     "0",
		"OTEL_LOG_TOOL_DETAILS":            "0",
		"OTEL_LOG_TOOL_CONTENT":            "0",
	}, nil
}

// safeCorrelationValue accepts only a bounded, unambiguous correlation charset
// ([A-Za-z0-9._-], 1..128). It deliberately excludes comma, equals, and
// whitespace so OTEL_RESOURCE_ATTRIBUTES cannot be spoofed or made ambiguous.
func safeCorrelationValue(v string) bool {
	if len(v) == 0 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// isTrustedTelemetryKey reports whether canonical is one of the exact OTLP
// telemetry keys the daemon injects trusted-last (allowed only as trusted).
func isTrustedTelemetryKey(canonical string) bool {
	switch canonical {
	case "CLAUDE_CODE_ENABLE_TELEMETRY", "OTEL_LOGS_EXPORTER", "OTEL_METRICS_EXPORTER",
		"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_RESOURCE_ATTRIBUTES", "OTEL_LOG_USER_PROMPTS", "OTEL_LOG_ASSISTANT_RESPONSES",
		"OTEL_LOG_TOOL_DETAILS", "OTEL_LOG_TOOL_CONTENT":
		return true
	}
	return false
}

type ComposeOptions struct {
	Inherited []string
	Local     map[string]string
	Custom    map[string]string
	Adapter   AdapterEnvironment
}

// ChildEnvironment is safe to format: its formatter returns keys only. Exec
// is the sole value-bearing projection and should be assigned directly to
// exec.Cmd.Env, never logged.
type ChildEnvironment struct {
	entries      map[string]environmentEntry
	cli          brain.CLIKind
	gatewayRoot  string
	secretKey    string
	taskHome     string
	codexHome    string
	clineDataDir string
}

func (e ChildEnvironment) Keys() []string { return sortedEntryKeys(e.entries) }

func (e ChildEnvironment) Exec() []string {
	keys := make([]string, 0, len(e.entries))
	for canonical := range e.entries {
		keys = append(keys, canonical)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, canonical := range keys {
		entry := e.entries[canonical]
		out = append(out, entry.key+"="+entry.value)
	}
	return out
}

func (e ChildEnvironment) String() string {
	return "ChildEnvironment{" + strings.Join(e.Keys(), ",") + "}"
}
func (e ChildEnvironment) GoString() string { return e.String() }
func (e ChildEnvironment) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.String())
}

func BuildGatewayEnvironment(opts ComposeOptions) (ChildEnvironment, SanitizationReport, error) {
	if _, err := CredentiallessAdapterContract(opts.Adapter.CLI); err != nil {
		return ChildEnvironment{}, SanitizationReport{}, err
	}
	minimal, report, err := BuildMinimalInherited(opts.Inherited)
	if err != nil {
		return ChildEnvironment{}, SanitizationReport{}, err
	}
	if err := ValidateCustomEnvironment(opts.Local); err != nil {
		return ChildEnvironment{}, SanitizationReport{}, fmt.Errorf("approved local environment: %w", err)
	}
	if err := ValidateCustomEnvironment(opts.Custom); err != nil {
		return ChildEnvironment{}, SanitizationReport{}, fmt.Errorf("custom environment: %w", err)
	}
	entries := cloneEntries(minimal.entries)
	mergeEnvironment(entries, opts.Local, originLocal)
	mergeEnvironment(entries, opts.Custom, originCustom)

	trusted, gatewayRoot, secretKey, err := trustedAdapterEntries(opts.Adapter)
	if err != nil {
		return ChildEnvironment{}, SanitizationReport{}, err
	}
	// Trusted entries are merged last by contract and therefore cannot be
	// shadowed by inherited, local, or custom values.
	for canonical, entry := range trusted {
		entries[canonical] = entry
	}
	child := ChildEnvironment{
		entries: entries, cli: opts.Adapter.CLI, gatewayRoot: gatewayRoot, secretKey: secretKey,
		taskHome: opts.Adapter.TaskHome,
	}
	if opts.Adapter.CLI == brain.CLICodex {
		child.codexHome = opts.Adapter.CodexHome
	}
	if opts.Adapter.CLI == brain.CLIOpenAICompatible {
		child.clineDataDir = opts.Adapter.ClineDataDir
	}
	return child, report, nil
}

func trustedAdapterEntries(profile AdapterEnvironment) (map[string]environmentEntry, string, string, error) {
	if !profile.StableSecret.IsSet() {
		return nil, "", "", fmt.Errorf("stable OmniRoute secret is required")
	}
	root, err := normalizeGatewayRoot(profile.GatewayRoot)
	if err != nil {
		return nil, "", "", err
	}
	if err := validatePhysicalControlledDirectory(profile.TaskHome, "task home"); err != nil {
		return nil, "", "", err
	}
	entries := map[string]environmentEntry{
		"HOME": {key: "HOME", value: profile.TaskHome, origin: originTrustedLocal},
	}
	switch profile.CLI {
	case brain.CLIClaudeCode:
		entries["ANTHROPIC_BASE_URL"] = environmentEntry{key: "ANTHROPIC_BASE_URL", value: root, origin: originTrustedGateway}
		entries["ANTHROPIC_AUTH_TOKEN"] = environmentEntry{key: "ANTHROPIC_AUTH_TOKEN", value: profile.StableSecret.value, origin: originTrustedSecret}
		telemetry, err := trustedTelemetryEnv(profile)
		if err != nil {
			return nil, "", "", err
		}
		for k, v := range telemetry {
			entries[k] = environmentEntry{key: k, value: v, origin: originTrustedLocal}
		}
		return entries, root, "ANTHROPIC_AUTH_TOKEN", nil
	case brain.CLICodex:
		if err := validatePhysicalControlledDirectory(profile.CodexHome, "Codex home"); err != nil {
			return nil, "", "", err
		}
		entries["CODEX_HOME"] = environmentEntry{key: "CODEX_HOME", value: profile.CodexHome, origin: originTrustedLocal}
		entries[CodexOmniRouteAPIKeyEnv] = environmentEntry{key: CodexOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
		return entries, root, CodexOmniRouteAPIKeyEnv, nil
	case brain.CLIOpenAICompatible:
		// Accepted OmniRoute OpenAI-compatible (Cline) route. The controlled
		// per-task Cline data dir carries providers.json; the stable OmniRoute
		// secret is injected trusted-last as CLINE_OMNIROUTE_API_KEY. Both keys
		// are rejected by the deny-list for inherited/local/custom origins and
		// are legal only as trusted entries merged after validation.
		if err := validatePhysicalControlledDirectory(profile.ClineDataDir, "Cline data dir"); err != nil {
			return nil, "", "", err
		}
		entries["CLINE_DATA_DIR"] = environmentEntry{key: "CLINE_DATA_DIR", value: profile.ClineDataDir, origin: originTrustedLocal}
		entries[ClineOmniRouteAPIKeyEnv] = environmentEntry{key: ClineOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
		return entries, root, ClineOmniRouteAPIKeyEnv, nil
	default:
		return nil, "", "", ErrAdapterFailClosed
	}
}

func normalizeGatewayRoot(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("gateway root URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("gateway root URL scheme must be http or https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("gateway root URL contains forbidden components")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("gateway root URL must not include an API path")
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

func validateControlledDirectory(path, label string) error {
	if path == "" || strings.TrimSpace(path) != path || !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path", label)
	}
	clean := filepath.Clean(path)
	if clean != path {
		return fmt.Errorf("%s must be canonical", label)
	}
	volumeRoot := filepath.VolumeName(clean) + string(filepath.Separator)
	if clean == volumeRoot || clean == "." {
		return fmt.Errorf("%s must not be a filesystem root", label)
	}
	return nil
}

// ValidateExecutionRoot proves that a launch root is canonical, exists as a
// directory, and has no symlinked path component. It reads metadata only.
func ValidateExecutionRoot(path string) error {
	return validatePhysicalControlledDirectory(path, "execution root")
}

func validatePhysicalControlledDirectory(path, label string) error {
	if err := validateControlledDirectory(path, label); err != nil {
		return err
	}
	anchor := filepath.VolumeName(path) + string(filepath.Separator)
	relative, err := filepath.Rel(anchor, path)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s physical path cannot be verified", label)
	}
	current := anchor
	components := []string{}
	if relative != "." {
		components = strings.Split(relative, string(filepath.Separator))
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return fmt.Errorf("%s physical path cannot be verified", label)
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%s must be an existing directory without symlink components", label)
		}
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !sameCanonicalPath(path, resolved) {
		return fmt.Errorf("%s physical path cannot be verified", label)
	}
	return nil
}

func sameCanonicalPath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func mergeEnvironment(entries map[string]environmentEntry, values map[string]string, origin envOrigin) {
	for key, value := range values {
		entries[strings.ToUpper(key)] = environmentEntry{key: key, value: value, origin: origin}
	}
}

func cloneEntries(source map[string]environmentEntry) map[string]environmentEntry {
	out := make(map[string]environmentEntry, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func sortedEntryKeys(entries map[string]environmentEntry) []string {
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.key)
	}
	sort.Strings(keys)
	return keys
}
