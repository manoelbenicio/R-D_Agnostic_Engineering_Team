package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/l2runtime"
)

func loadProdexLaunchConfig() (ProdexConfig, AgentEntry, error) {
	enabled := envBool("MULTICA_PRODEX_ENABLED")
	if !enabled {
		return ProdexConfig{}, AgentEntry{}, nil
	}

	path := strings.TrimSpace(os.Getenv("MULTICA_PRODEX_PATH"))
	if path == "" {
		path = "prodex"
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return ProdexConfig{}, AgentEntry{}, fmt.Errorf("prodex enabled but executable %q was not found: %w", path, err)
	}

	version := strings.TrimSpace(os.Getenv("MULTICA_PRODEX_VERSION"))
	commit := strings.TrimSpace(os.Getenv("MULTICA_PRODEX_COMMIT"))
	if version == "" || commit == "" {
		return ProdexConfig{}, AgentEntry{}, fmt.Errorf("prodex enabled but MULTICA_PRODEX_VERSION and MULTICA_PRODEX_COMMIT are both required")
	}

	cfg := ProdexConfig{
		Enabled:             true,
		Path:                resolved,
		Version:             version,
		Commit:              commit,
		SmartContextShadow:  envBoolDefault("MULTICA_PRODEX_SMART_CONTEXT_SHADOW", true),
		SmartContextCanary:  strings.TrimSpace(os.Getenv("MULTICA_PRODEX_SMART_CONTEXT_CANARY_PERCENT")),
		KillSwitchDefaultOn: envBoolDefault("MULTICA_PRODEX_KILL_SWITCH_DEFAULT_ON", true),
	}
	if cfg.SmartContextCanary == "" {
		cfg.SmartContextCanary = "0"
	}

	return cfg, AgentEntry{
		Path:  resolved,
		Model: strings.TrimSpace(os.Getenv("MULTICA_CODEX_MODEL")),
	}, nil
}

func loadL2RuntimeConfig() (L2RuntimeConfig, error) {
	enabled := envBool("MULTICA_L2_ENABLED")
	if !enabled {
		return L2RuntimeConfig{}, nil
	}

	timeout, err := durationFromEnv("MULTICA_L2_TIMEOUT", 5*time.Second)
	if err != nil {
		return L2RuntimeConfig{}, err
	}
	baseURL := strings.TrimSpace(os.Getenv("MULTICA_L2_BASE_URL"))
	token := strings.TrimSpace(os.Getenv("MULTICA_L2_BEARER_TOKEN"))
	if _, err := l2runtime.NewClient(baseURL, token, timeout); err != nil {
		return L2RuntimeConfig{}, fmt.Errorf("l2 runtime enabled but config is invalid: %w", err)
	}
	policyID := strings.TrimSpace(os.Getenv("MULTICA_L2_POLICY_ID"))
	if policyID == "" {
		policyID = "default"
	}
	return L2RuntimeConfig{
		Enabled:     true,
		BaseURL:     baseURL,
		BearerToken: token,
		Timeout:     timeout,
		PolicyID:    policyID,
	}, nil
}

func (d *Daemon) applyProdexEnv(provider string, envRoot string, agentEnv map[string]string) {
	if provider != "codex" || !d.cfg.Prodex.Enabled {
		return
	}
	prodexHome := strings.TrimSpace(os.Getenv("PRODEX_HOME"))
	if prodexHome == "" && envRoot != "" {
		prodexHome = filepath.Join(envRoot, "prodex")
	}
	if prodexHome != "" {
		agentEnv["PRODEX_HOME"] = prodexHome
	}
	agentEnv["MULTICA_PRODEX_ENABLED"] = "1"
	agentEnv["MULTICA_PRODEX_VERSION"] = d.cfg.Prodex.Version
	agentEnv["MULTICA_PRODEX_COMMIT"] = d.cfg.Prodex.Commit
	if d.cfg.Prodex.SmartContextShadow {
		agentEnv["PRODEX_SMART_CONTEXT_SHADOW"] = "1"
	}
	if d.cfg.Prodex.SmartContextCanary != "" {
		agentEnv["PRODEX_SMART_CONTEXT_CANARY_PERCENT"] = d.cfg.Prodex.SmartContextCanary
	}
	if d.cfg.Prodex.KillSwitchDefaultOn {
		agentEnv["PRODEX_KILL_SWITCH_DEFAULT_ON"] = "1"
	}
}

func prodexSidecarEnv() []string {
	env := envMap(os.Environ())
	env["PRODEX_ALLOW_UNSAFE_CHILD_ENV"] = "off"
	env["NO_PROXY"] = appendNoProxyLoopback(env["NO_PROXY"])
	env["no_proxy"] = appendNoProxyLoopback(env["no_proxy"])
	return flattenEnvMap(env)
}

func envMap(values []string) map[string]string {
	out := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func flattenEnvMap(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func appendNoProxyLoopback(value string) string {
	required := []string{"127.0.0.1", "localhost", "::1"}
	seen := make(map[string]struct{}, len(required))
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts)+len(required))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		seen[strings.ToLower(part)] = struct{}{}
		out = append(out, part)
	}
	for _, part := range required {
		if _, ok := seen[strings.ToLower(part)]; ok {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, ",")
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envBoolDefault(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
