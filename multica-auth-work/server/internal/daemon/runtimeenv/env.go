package runtimeenv

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
			return MinimalEnvironment{}, SanitizationReport{}, fmt.Errorf("inherited environment entry %d is malformed", index)
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
	StableSecret StableSecret
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
	entries     map[string]environmentEntry
	cli         brain.CLIKind
	gatewayRoot string
	secretKey   string
	taskHome    string
	codexHome   string
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
		return entries, root, "ANTHROPIC_AUTH_TOKEN", nil
	case brain.CLICodex:
		if err := validatePhysicalControlledDirectory(profile.CodexHome, "Codex home"); err != nil {
			return nil, "", "", err
		}
		entries["CODEX_HOME"] = environmentEntry{key: "CODEX_HOME", value: profile.CodexHome, origin: originTrustedLocal}
		entries[CodexOmniRouteAPIKeyEnv] = environmentEntry{key: CodexOmniRouteAPIKeyEnv, value: profile.StableSecret.value, origin: originTrustedSecret}
		return entries, root, CodexOmniRouteAPIKeyEnv, nil
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
