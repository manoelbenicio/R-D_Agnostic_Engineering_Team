package agent

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var errExactProcessEnvironment = errors.New("exact process environment is invalid")

// processEnvironment returns the environment that must be assigned directly
// to exec.Cmd.Env. ExactEnv is intentionally distinguished from an empty Env
// map: an explicitly selected exact environment fails closed instead of
// falling back to parent-process inheritance.
func processEnvironment(config Config) ([]string, error) {
	if config.ExactEnv == nil {
		return buildEnv(config.Env), nil
	}
	if len(config.ExactEnv) == 0 {
		return nil, errExactProcessEnvironment
	}

	result := make([]string, 0, len(config.ExactEnv))
	seen := make(map[string]struct{}, len(config.ExactEnv))
	for index, entry := range config.ExactEnv {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || !validProcessEnvironmentKey(key) || strings.ContainsRune(value, '\x00') {
			return nil, fmt.Errorf("%w: malformed entry %d", errExactProcessEnvironment, index)
		}
		canonical := strings.ToUpper(key)
		if _, duplicate := seen[canonical]; duplicate {
			return nil, fmt.Errorf("%w: duplicate key %s", errExactProcessEnvironment, key)
		}
		seen[canonical] = struct{}{}
		result = append(result, entry)
	}
	return result, nil
}

func processEnvironmentValue(config Config, key string) string {
	if config.ExactEnv == nil {
		return config.Env[key]
	}
	return environmentSliceValue(config.ExactEnv, key)
}

func environmentSliceValue(environment []string, key string) string {
	for _, entry := range environment {
		candidate, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(candidate, key) {
			return value
		}
	}
	return ""
}

// resolveProcessExecutable preserves the historical parent-PATH behavior for
// legacy launches. In exact-environment mode it returns an absolute executable
// resolved exclusively from the already-validated ExactEnv PATH, so os/exec
// can never perform a second lookup against the daemon parent's PATH.
func resolveProcessExecutable(config Config, defaultExecutable string, validatedEnvironment []string) (string, error) {
	candidate := config.ExecutablePath
	if candidate == "" {
		candidate = defaultExecutable
	}

	if config.ExactEnv == nil {
		if _, err := exec.LookPath(candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}

	if filepath.IsAbs(candidate) {
		if filepath.Clean(candidate) != candidate {
			return "", fmt.Errorf("exact-environment absolute executable must be canonical")
		}
		resolved, err := exec.LookPath(candidate)
		if err != nil {
			return "", err
		}
		return resolved, nil
	}
	if candidate == "." || candidate == ".." || filepath.Base(candidate) != candidate || filepath.VolumeName(candidate) != "" {
		return "", fmt.Errorf("exact-environment executable must be bare or absolute")
	}

	exactPath := environmentSliceValue(validatedEnvironment, "PATH")
	if exactPath == "" {
		return "", fmt.Errorf("exact environment PATH is required for bare executable %q", candidate)
	}
	for index, directory := range filepath.SplitList(exactPath) {
		if directory == "" || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
			return "", fmt.Errorf("exact environment PATH entry %d must be canonical and absolute", index)
		}
		resolved, err := exec.LookPath(filepath.Join(directory, candidate))
		if err == nil {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("executable %q not found in exact environment PATH", candidate)
}

func validProcessEnvironmentKey(key string) bool {
	if key == "" {
		return false
	}
	for index, character := range key {
		switch {
		case character == '_':
		case character >= 'A' && character <= 'Z':
		case character >= 'a' && character <= 'z':
		case index > 0 && character >= '0' && character <= '9':
		default:
			return false
		}
	}
	return true
}
