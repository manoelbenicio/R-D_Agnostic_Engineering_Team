package execenv

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const clineCredentialRelDir = ".cline"

// ClineHomeOptions carries optional inputs for prepareClineHome.
type ClineHomeOptions struct {
	// AccountHome, when non-empty, is the per-account Cline source directory.
	// Cline CLI 2.0 can use CLINE_DATA_DIR/--data-dir instead of ~/.cline;
	// enrollment may store either a home-like root containing .cline/ or the
	// data-dir root itself. The preparer accepts both layouts.
	AccountHome string
}

// prepareCredentiallessClineHome creates task-local directories without
// resolving account homes, inspecting provider auth, or copying configuration.
func prepareCredentiallessClineHome(clineDataDir string) error {
	if strings.TrimSpace(clineDataDir) == "" || !filepath.IsAbs(clineDataDir) || filepath.Clean(clineDataDir) == string(filepath.Separator) {
		return fmt.Errorf("credentialless cline data dir must be an absolute non-root path")
	}
	for _, dir := range []string{clineDataDir, filepath.Join(clineDataDir, "settings")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create credentialless cline data dir: %w", err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("restrict credentialless cline data dir: %w", err)
		}
	}
	return nil
}

// WriteCredentiallessClineConfig installs validated non-secret providers config.
func WriteCredentiallessClineConfig(clineDataDir string, raw []byte) error {
	if strings.TrimSpace(clineDataDir) == "" || !filepath.IsAbs(clineDataDir) || filepath.Clean(clineDataDir) == string(filepath.Separator) {
		return fmt.Errorf("credentialless cline data dir must be an absolute non-root path")
	}
	if len(raw) == 0 || len(raw) > 64<<10 {
		return fmt.Errorf("credentialless Cline configuration is empty or too large")
	}
	if err := prepareCredentiallessClineHome(clineDataDir); err != nil {
		return err
	}
	path := filepath.Join(clineDataDir, "settings", "providers.json")
	if err := os.WriteFile(path, append([]byte(nil), raw...), 0o600); err != nil {
		return fmt.Errorf("write credentialless Cline configuration: %w", err)
	}
	return os.Chmod(path, 0o600)
}

// prepareClineHome restores a per-account Cline data dir into a per-task
// CLINE_DATA_DIR. The caller wires CLINE_DATA_DIR, CLINE_SANDBOX and
// CLINE_SANDBOX_DATA_DIR into the child process env.
func prepareClineHome(dataDir string, opts ClineHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if dataDir == "" {
		return fmt.Errorf("cline data dir is empty")
	}

	src := resolveClineSourceDir(opts.AccountHome)
	if src == "" {
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("remove stale cline data dir %s: %w", dataDir, err)
		}
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return fmt.Errorf("create empty cline data dir: %w", err)
		}
		if err := os.Chmod(dataDir, 0o700); err != nil {
			return fmt.Errorf("chmod empty cline data dir: %w", err)
		}
		logCredentialDirState("execenv: cline data dir", dataDir, logger)
		return nil
	}

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat cline source %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cline source %s is not a directory", src)
	}
	if err := syncCredentialDir(src, dataDir); err != nil {
		return fmt.Errorf("seed per-account cline data dir: %w", err)
	}
	if err := os.Chmod(dataDir, 0o700); err != nil {
		return fmt.Errorf("chmod cline data dir: %w", err)
	}
	logCredentialDirState("execenv: cline data dir", dataDir, logger)
	return nil
}

func resolveClineSourceDir(accountHome string) string {
	for _, candidate := range []string{
		filepath.Join(accountHome, clineCredentialRelDir),
		filepath.Join(accountHome, "cline"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	for _, marker := range []string{
		filepath.Join(accountHome, "data", "settings", "providers.json"),
		filepath.Join(accountHome, "settings", "providers.json"),
	} {
		if info, err := os.Stat(marker); err == nil && info.Mode().IsRegular() {
			return accountHome
		}
	}
	return ""
}
