package execenv

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const nimCredentialRelDir = ".nim"

// NimHomeOptions carries optional inputs for prepareNimHome.
type NimHomeOptions struct {
	// AccountHome, when non-empty, is the per-account NIM source directory.
	// NVIDIA NIM is an OpenAI-compatible inference API
	// (https://integrate.api.nvidia.com/v1) authenticated with an NVIDIA API key.
	// Enrollment stores the per-account NIM credential under
	// <AccountHome>/.nim/ (e.g. .nim/credentials.json, .nim/api_key). The
	// preparer restores that credential dir into a per-task nimDataDir so the
	// native nim runtime (server/pkg/agent/nim.go) reads the account's own key
	// instead of a shared/global one. Empty AccountHome = shared/global
	// behavior (no isolated home).
	AccountHome string
}

// prepareNimHome restores a per-account NIM credential directory into a per-task
// nimDataDir. The caller wires NIM_HOME (and NVIDIA_API_KEY, resolved from the
// credential file) into the child process env via CredentialEnv.
//
// Only the filesystem state is prepared here; the credential dir is copied AS-IS
// and never inspected, so no secret can leak into logs (mirror of
// prepareAntigravityHome / prepareClineHome). The directory is mode 0700 so the
// per-task key never leaks across tasks or to other users.
func prepareNimHome(nimDataDir string, opts NimHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if nimDataDir == "" {
		return fmt.Errorf("nim data dir is empty")
	}

	src := resolveNimSourceDir(opts.AccountHome)
	if src == "" {
		// No per-account NIM credential dir: create an empty isolated dir so the
		// runtime fails closed at read time rather than silently falling back to a
		// shared/global key (mirror of the cline empty-dir path).
		if err := os.RemoveAll(nimDataDir); err != nil {
			return fmt.Errorf("remove stale nim data dir %s: %w", nimDataDir, err)
		}
		if err := os.MkdirAll(nimDataDir, 0o700); err != nil {
			return fmt.Errorf("create empty nim data dir: %w", err)
		}
		if err := os.Chmod(nimDataDir, 0o700); err != nil {
			return fmt.Errorf("chmod empty nim data dir: %w", err)
		}
		logCredentialDirState("execenv: nim credential dir", nimDataDir, logger)
		return nil
	}

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat nim source %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("nim source %s is not a directory", src)
	}
	if err := syncCredentialDir(src, nimDataDir); err != nil {
		return fmt.Errorf("seed per-account nim credential dir: %w", err)
	}
	if err := os.Chmod(nimDataDir, 0o700); err != nil {
		return fmt.Errorf("chmod nim data dir: %w", err)
	}
	logCredentialDirState("execenv: nim credential dir", nimDataDir, logger)
	return nil
}

// resolveNimSourceDir resolves the per-account NIM credential directory. It
// accepts a home-like root containing .nim/ (or a bare nim/ subdir), and a
// data-dir root that already holds the NIM credential markers directly.
func resolveNimSourceDir(accountHome string) string {
	for _, candidate := range []string{
		filepath.Join(accountHome, nimCredentialRelDir),
		filepath.Join(accountHome, "nim"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	// A data-dir root passed as AccountHome: detect NIM credential markers directly.
	for _, marker := range []string{
		filepath.Join(accountHome, "credentials.json"),
		filepath.Join(accountHome, "api_key"),
		filepath.Join(accountHome, "config.json"),
	} {
		if info, err := os.Stat(marker); err == nil && info.Mode().IsRegular() {
			return accountHome
		}
	}
	return ""
}
