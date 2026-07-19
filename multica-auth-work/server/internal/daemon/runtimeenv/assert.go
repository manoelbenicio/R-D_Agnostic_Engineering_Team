package runtimeenv

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

var ErrPreLaunchPolicy = errors.New("pre-launch credentialless runtime assertion failed")

type LaunchPlan struct {
	Environment   ChildEnvironment
	CodexConfig   *CodexConfigContract
	TaskHome      []HomeEntry
	ExecutionRoot string
}

// AssertPreLaunch verifies provenance, key names, and controlled execution
// roots. It never compares, formats, hashes, or otherwise exposes the stable
// secret value.
func AssertPreLaunch(plan LaunchPlan) error {
	environment := plan.Environment
	if _, err := CredentiallessAdapterContract(environment.cli); err != nil {
		return ErrPreLaunchPolicy
	}
	if len(environment.entries) == 0 || environment.secretKey == "" || environment.gatewayRoot == "" {
		return ErrPreLaunchPolicy
	}
	if !launchRootsAreControlled(plan.ExecutionRoot, environment) {
		return ErrPreLaunchPolicy
	}
	for _, required := range []string{"PATH", "HOME"} {
		entry, ok := environment.entries[required]
		if !ok || strings.TrimSpace(entry.value) == "" {
			return ErrPreLaunchPolicy
		}
	}
	secretCount := 0
	for canonical, entry := range environment.entries {
		classification := ClassifyEnvironmentKey(entry.key)
		if classification.Denied && !trustedEntryAllowed(environment.cli, canonical, entry.origin) {
			return ErrPreLaunchPolicy
		}
		if entry.origin == originTrustedSecret {
			secretCount++
			if canonical != strings.ToUpper(environment.secretKey) || entry.value == "" {
				return ErrPreLaunchPolicy
			}
		}
	}
	if secretCount != 1 {
		return ErrPreLaunchPolicy
	}
	if err := ValidateTaskHomeManifest(environment.cli, plan.TaskHome); err != nil {
		return ErrPreLaunchPolicy
	}
	switch environment.cli {
	case brain.CLIClaudeCode:
		if plan.CodexConfig != nil {
			return ErrPreLaunchPolicy
		}
	case brain.CLICodex:
		if plan.CodexConfig == nil || plan.CodexConfig.Validate() != nil {
			return ErrPreLaunchPolicy
		}
	default:
		return ErrPreLaunchPolicy
	}
	return nil
}

func launchRootsAreControlled(executionRoot string, environment ChildEnvironment) bool {
	if ValidateExecutionRoot(executionRoot) != nil {
		return false
	}
	home, ok := environment.entries["HOME"]
	if !ok || !exactPathWithin(executionRoot, environment.taskHome, home.value) {
		return false
	}
	if environment.cli != brain.CLICodex {
		return environment.codexHome == ""
	}
	codexHome, ok := environment.entries["CODEX_HOME"]
	return ok && exactPathWithin(executionRoot, environment.codexHome, codexHome.value)
}

func exactPathWithin(root, expected, actual string) bool {
	if expected == "" || actual == "" || !filepath.IsAbs(expected) || !filepath.IsAbs(actual) {
		return false
	}
	if filepath.Clean(expected) != expected || filepath.Clean(actual) != actual || expected != actual {
		return false
	}
	relative, err := filepath.Rel(root, actual)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	return physicalDirectoryWithin(root, actual)
}

func physicalDirectoryWithin(root, path string) bool {
	if validatePhysicalControlledDirectory(root, "execution root") != nil ||
		validatePhysicalControlledDirectory(path, "controlled home") != nil {
		return false
	}
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedPath, pathErr := filepath.EvalSymlinks(path)
	if rootErr != nil || pathErr != nil {
		return false
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	return err == nil && !filepath.IsAbs(relative) && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func trustedEntryAllowed(cli brain.CLIKind, canonical string, origin envOrigin) bool {
	switch cli {
	case brain.CLIClaudeCode:
		switch canonical {
		case "HOME":
			return origin == originTrustedLocal
		case "ANTHROPIC_BASE_URL":
			return origin == originTrustedGateway
		case "ANTHROPIC_AUTH_TOKEN":
			return origin == originTrustedSecret
		}
	case brain.CLICodex:
		switch canonical {
		case "HOME", "CODEX_HOME":
			return origin == originTrustedLocal
		case CodexOmniRouteAPIKeyEnv:
			return origin == originTrustedSecret
		}
	}
	return false
}
