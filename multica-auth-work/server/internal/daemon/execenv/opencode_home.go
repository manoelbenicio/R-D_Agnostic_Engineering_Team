package execenv

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	opencodeDefaultDataRelDir   = ".local/share/opencode"
	opencodeDefaultConfigRelDir = ".config/opencode"
	opencodeXDGRelDir           = "opencode"
)

// OpenCodeHomeOptions carries optional inputs for prepareOpenCodeHome.
type OpenCodeHomeOptions struct {
	// AccountHome, when non-empty, is the per-account source root. OpenCode
	// stores auth under XDG_DATA_HOME/opencode/auth.json and config under
	// XDG_CONFIG_HOME/opencode/. Enrollment may store a home-like root
	// containing .local/share/opencode + .config/opencode, or XDG roots.
	AccountHome string
}

// prepareOpenCodeHome restores per-account OpenCode credential/config state
// into per-task XDG_DATA_HOME and XDG_CONFIG_HOME directories. GLM fleet agents
// that run through the OpenCode-compatible runtime use the same preparer.
func prepareOpenCodeHome(dataHome, configHome string, opts OpenCodeHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if dataHome == "" {
		return fmt.Errorf("opencode data home is empty")
	}
	if configHome == "" {
		return fmt.Errorf("opencode config home is empty")
	}

	if err := os.MkdirAll(dataHome, 0o700); err != nil {
		return fmt.Errorf("create opencode data home: %w", err)
	}
	if err := os.Chmod(dataHome, 0o700); err != nil {
		return fmt.Errorf("chmod opencode data home: %w", err)
	}
	if err := os.MkdirAll(configHome, 0o700); err != nil {
		return fmt.Errorf("create opencode config home: %w", err)
	}
	if err := os.Chmod(configHome, 0o700); err != nil {
		return fmt.Errorf("chmod opencode config home: %w", err)
	}

	dataSrc := firstExistingDir(
		filepath.Join(opts.AccountHome, opencodeDefaultDataRelDir),
		filepath.Join(opts.AccountHome, opencodeXDGRelDir),
	)
	if dataSrc == "" {
		if err := syncCredentialDir(filepath.Join(opts.AccountHome, opencodeDefaultDataRelDir), filepath.Join(dataHome, opencodeXDGRelDir)); err != nil {
			return fmt.Errorf("clear stale opencode data dir: %w", err)
		}
	} else if err := syncCredentialDir(dataSrc, filepath.Join(dataHome, opencodeXDGRelDir)); err != nil {
		return fmt.Errorf("seed per-account opencode data dir: %w", err)
	}

	configSrc := firstExistingDir(
		filepath.Join(opts.AccountHome, opencodeDefaultConfigRelDir),
		filepath.Join(opts.AccountHome, "config", opencodeXDGRelDir),
		filepath.Join(opts.AccountHome, "opencode-config"),
	)
	if configSrc == "" {
		if err := syncCredentialDir(filepath.Join(opts.AccountHome, opencodeDefaultConfigRelDir), filepath.Join(configHome, opencodeXDGRelDir)); err != nil {
			return fmt.Errorf("clear stale opencode config dir: %w", err)
		}
	} else if err := syncCredentialDir(configSrc, filepath.Join(configHome, opencodeXDGRelDir)); err != nil {
		return fmt.Errorf("seed per-account opencode config dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(dataHome, opencodeXDGRelDir), 0o700); err != nil {
		return fmt.Errorf("create empty opencode data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(configHome, opencodeXDGRelDir), 0o700); err != nil {
		return fmt.Errorf("create empty opencode config dir: %w", err)
	}
	logCredentialDirState("execenv: opencode data dir", filepath.Join(dataHome, opencodeXDGRelDir), logger)
	logCredentialDirState("execenv: opencode config dir", filepath.Join(configHome, opencodeXDGRelDir), logger)
	return nil
}

func firstExistingDir(candidates ...string) string {
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}
