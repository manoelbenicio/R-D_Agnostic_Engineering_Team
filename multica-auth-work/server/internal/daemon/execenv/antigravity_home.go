package execenv

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	antigravityCredentialRelDir  = ".gemini/antigravity-cli"
	antigravityCredentialRelPath = antigravityCredentialRelDir + "/antigravity-oauth-token"
)

// AntigravityHomeOptions carries optional inputs for prepareAntigravityHome.
type AntigravityHomeOptions struct {
	// AccountHome, when non-empty, is the per-account HOME source directory.
	// The Antigravity CLI reads token files under
	// HOME/.gemini/antigravity-cli/, so isolation is driven by HOME rather than
	// a vendor-specific environment variable.
	AccountHome string
}

// prepareAntigravityHome restores only the per-account Antigravity OAuth token
// into the isolated HOME for a task. Volatile sibling artifacts such as logs,
// caches, and databases remain outside the task HOME.
func prepareAntigravityHome(home string, opts AntigravityHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if home == "" {
		return fmt.Errorf("antigravity home is empty")
	}

	for _, dir := range []string{
		home,
		filepath.Join(home, ".gemini"),
		filepath.Join(home, antigravityCredentialRelDir),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create antigravity credential directory %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("chmod antigravity credential directory %s: %w", dir, err)
		}
	}

	src := filepath.Join(opts.AccountHome, antigravityCredentialRelPath)
	dst := filepath.Join(home, antigravityCredentialRelPath)
	if info, err := os.Lstat(src); err != nil {
		return fmt.Errorf("required per-account antigravity OAuth token is unavailable: %w", err)
	} else if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("required per-account antigravity OAuth token must be a physical regular file")
	}
	if err := syncCredentialFile(src, dst); err != nil {
		return fmt.Errorf("seed per-account antigravity OAuth token: %w", err)
	}
	// syncCredentialFile already creates credentials as 0600. Keep the
	// explicit chmod as a postcondition in case that shared helper changes.
	if err := os.Chmod(dst, 0o600); err != nil {
		return fmt.Errorf("restrict per-account antigravity OAuth token: %w", err)
	}
	logCredentialFileState("execenv: antigravity OAuth token", dst, logger)
	return nil
}

func syncCredentialDir(src, dst string) error {
	srcInfo, srcErr := os.Stat(src)
	srcMissing := os.IsNotExist(srcErr)
	if srcErr != nil && !srcMissing {
		return fmt.Errorf("stat src %s: %w", src, srcErr)
	}
	if !srcMissing && !srcInfo.IsDir() {
		return fmt.Errorf("src %s is not a directory", src)
	}

	if _, err := os.Lstat(dst); err == nil {
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("remove stale dst %s: %w", dst, err)
		}
	}

	if srcMissing {
		return nil
	}
	return copyCredentialDir(src, dst, srcInfo)
}

func copyCredentialDir(src, dst string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(dst, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("create dst dir %s: %w", dst, err)
	}
	if err := os.Chmod(dst, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("chmod dst dir %s: %w", dst, err)
	}
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("restore mtime %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", src, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", srcPath, err)
		}
		switch {
		case info.IsDir():
			if err := copyCredentialDir(srcPath, dstPath, info); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if err := copyCredentialFile(srcPath, dstPath, info); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported credential path type %s", srcPath)
		}
	}
	return nil
}

func logCredentialDirState(label, path string, logger *slog.Logger) {
	if logger == nil {
		return
	}
	fi, err := os.Lstat(path)
	if err != nil {
		logger.Info(label+" absent", "path", path, "error", err)
		return
	}
	logger.Info(label+" present",
		"path", path,
		"type", fi.Mode().Type().String(),
		"mtime", fi.ModTime().UTC(),
	)
}
