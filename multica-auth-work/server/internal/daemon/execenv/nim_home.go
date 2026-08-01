package execenv

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const nimCredentialFileName = "NVIDIA_API_KEY"

// NimHomeOptions carries the per-account credential source for NIM.
type NimHomeOptions struct {
	// AccountHome is the assigned account directory. Its NVIDIA_API_KEY file
	// contains only the raw API key (no KEY= prefix).
	AccountHome string
}

// prepareNimHome copies the assigned account's NVIDIA_API_KEY into a per-task
// directory. The returned path always names a regular 0600 file; the child
// process receives the file's trimmed value through NVIDIA_API_KEY.
func prepareNimHome(home string, opts NimHomeOptions, logger *slog.Logger) (string, error) {
	if opts.AccountHome == "" {
		return "", nil
	}
	if home == "" {
		return "", fmt.Errorf("nim home is empty")
	}

	src := filepath.Join(opts.AccountHome, nimCredentialFileName)
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("stat NIM credential: %w", err)
	}
	if !srcInfo.Mode().IsRegular() {
		return "", fmt.Errorf("NIM credential %s is not a regular file", src)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", fmt.Errorf("create nim home: %w", err)
	}
	if err := os.Chmod(home, 0o700); err != nil {
		return "", fmt.Errorf("chmod nim home: %w", err)
	}

	dst := filepath.Join(home, nimCredentialFileName)
	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return "", fmt.Errorf("remove stale NIM credential: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("lstat NIM credential destination: %w", err)
	}
	if err := copyCredentialFile(src, dst, srcInfo); err != nil {
		return "", fmt.Errorf("copy per-account NIM credential: %w", err)
	}
	if err := os.Chmod(dst, 0o600); err != nil {
		return "", fmt.Errorf("chmod NIM credential: %w", err)
	}
	if _, err := readNIMAPIKey(dst); err != nil {
		_ = os.Remove(dst)
		return "", err
	}
	logCredentialFileState("execenv: NIM credential", dst, logger)
	return dst, nil
}

func readNIMAPIKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read NIM credential: %w", err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("NIM credential is empty")
	}
	return key, nil
}
