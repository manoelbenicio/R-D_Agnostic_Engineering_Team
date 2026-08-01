package execenv

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const kiroCredentialRelPath = "kiro-cli/data.sqlite3"

// KiroHomeOptions carries optional inputs for prepareKiroHome.
type KiroHomeOptions struct {
	// AccountHome, when non-empty, is the per-account XDG_DATA_HOME source
	// directory. Kiro is an Amazon Q fork and ignores KIRO_HOME; its native
	// credential store is data.sqlite3 under XDG_DATA_HOME/kiro-cli/.
	//
	// Empty preserves the historical global behavior: no per-account copy is
	// forced here, so the process environment can keep using the user's normal
	// XDG_DATA_HOME. Headless deployments may also bypass this sqlite store by
	// providing KIRO_API_KEY directly to the CLI process; this helper never logs
	// or reads that value.
	AccountHome string
}

// prepareKiroHome restores the per-account Kiro credential store into the
// isolated XDG_DATA_HOME for a task. The caller wires that directory into
// XDG_DATA_HOME; this helper only prepares the filesystem state.
func prepareKiroHome(home string, opts KiroHomeOptions, logger *slog.Logger) error {
	if opts.AccountHome == "" {
		return nil
	}
	if home == "" {
		return fmt.Errorf("kiro xdg data home is empty")
	}

	if err := os.MkdirAll(filepath.Join(home, "kiro-cli"), 0o700); err != nil {
		return fmt.Errorf("create kiro data dir: %w", err)
	}
	if err := os.Chmod(home, 0o700); err != nil {
		return fmt.Errorf("chmod kiro data home: %w", err)
	}
	if err := os.Chmod(filepath.Join(home, "kiro-cli"), 0o700); err != nil {
		return fmt.Errorf("chmod kiro credential dir: %w", err)
	}

	src := filepath.Join(opts.AccountHome, kiroCredentialRelPath)
	dst := filepath.Join(home, kiroCredentialRelPath)
	if err := syncCredentialFile(src, dst); err != nil {
		return fmt.Errorf("seed per-account kiro data.sqlite3: %w", err)
	}
	logCredentialFileState("execenv: kiro data.sqlite3", dst, logger)
	return nil
}

func syncCredentialFile(src, dst string) error {
	srcInfo, srcErr := os.Stat(src)
	srcMissing := os.IsNotExist(srcErr)
	if srcErr != nil && !srcMissing {
		return fmt.Errorf("stat src %s: %w", src, srcErr)
	}
	if !srcMissing && !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("src %s is not a regular file", src)
	}

	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("remove stale dst %s: %w", dst, err)
		}
	}

	if srcMissing {
		return nil
	}
	return copyCredentialFile(src, dst, srcInfo)
}

func copyCredentialFile(src, dst string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("create dst parent %s: %w", filepath.Dir(dst), err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dst, err)
	}
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("restore mtime %s: %w", dst, err)
	}
	return nil
}

func logCredentialFileState(label, path string, logger *slog.Logger) {
	if logger == nil {
		return
	}
	fi, err := os.Lstat(path)
	if err != nil {
		logger.Info(label+" absent", "path", path, "error", err)
		return
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, _ := os.Readlink(path)
		logger.Info(label+" is symlink", "path", path, "target", target)
		return
	}
	logger.Info(label+" is regular file",
		"path", path,
		"size", fi.Size(),
		"mtime", fi.ModTime().UTC(),
	)
}
