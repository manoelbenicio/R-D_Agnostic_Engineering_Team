package rotation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	errAuthenticatorAccountID = errors.New("rotation: account id is required")
	errAuthenticatorHomeDir   = errors.New("rotation: account home dir is required")
	errCredentialUnavailable  = errors.New("rotation: account credential is unavailable")
)

type CredentialPath struct {
	SourceRoot string
	HomeRoot   string
	RelPath    string
	Dir        bool
}

type CredentialPathSelector func(Account) []CredentialPath

type CredentialAuthenticatorOption func(*CredentialAuthenticator)

type CredentialAuthenticator struct {
	selectPaths CredentialPathSelector

	mu       sync.Mutex
	sessions map[string]Account
}

var _ AccountAuthenticator = (*CredentialAuthenticator)(nil)

func NewCredentialAuthenticator(opts ...CredentialAuthenticatorOption) *CredentialAuthenticator {
	a := &CredentialAuthenticator{
		selectPaths: defaultCredentialPaths,
		sessions:    map[string]Account{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	if a.selectPaths == nil {
		a.selectPaths = defaultCredentialPaths
	}
	return a
}

func WithCredentialPathSelector(selector CredentialPathSelector) CredentialAuthenticatorOption {
	return func(a *CredentialAuthenticator) {
		a.selectPaths = selector
	}
}

func (a *CredentialAuthenticator) Login(ctx context.Context, acc Account) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(acc.AccountID) == "" {
		return "", errAuthenticatorAccountID
	}
	if strings.TrimSpace(acc.HomeDir) == "" {
		return "", errAuthenticatorHomeDir
	}

	restored := false
	for _, p := range a.selectPaths(acc) {
		if err := restoreCredentialPath(p); err != nil {
			return "", err
		}
		if credentialPathPresent(filepath.Join(p.HomeRoot, p.RelPath), p.Dir) {
			restored = true
		}
	}
	if !restored {
		return "", errCredentialUnavailable
	}

	sessionID := "account:" + acc.AccountID
	a.mu.Lock()
	a.sessions[sessionID] = acc
	a.mu.Unlock()
	return sessionID, nil
}

func (a *CredentialAuthenticator) Logout(ctx context.Context, acc Account) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, p := range a.selectPaths(acc) {
		if p.HomeRoot == "" || p.RelPath == "" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(p.HomeRoot, p.RelPath)); err != nil {
			return fmt.Errorf("remove credential path: %w", err)
		}
	}
	return nil
}

func (a *CredentialAuthenticator) WaitAuthenticated(ctx context.Context, sessionID string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	if timeout <= 0 {
		deadline = time.Now()
	}
	for {
		acc, ok := a.session(sessionID)
		if !ok {
			return false, nil
		}
		for _, p := range a.selectPaths(acc) {
			if credentialPathPresent(filepath.Join(p.HomeRoot, p.RelPath), p.Dir) {
				return true, nil
			}
		}
		if !time.Now().Before(deadline) {
			return false, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func (a *CredentialAuthenticator) session(sessionID string) (Account, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	acc, ok := a.sessions[sessionID]
	return acc, ok
}

func defaultCredentialPaths(acc Account) []CredentialPath {
	sourceRoot := strings.TrimSpace(acc.ConfigDir)
	if sourceRoot == "" {
		sourceRoot = acc.HomeDir
	}
	base := CredentialPath{
		SourceRoot: sourceRoot,
		HomeRoot:   acc.HomeDir,
	}
	switch strings.ToLower(strings.TrimSpace(acc.Vendor)) {
	case "codex":
		base.RelPath = "auth.json"
	case "kiro", "opus":
		base.RelPath = "kiro-cli/data.sqlite3"
	case "antigravity":
		base.RelPath = ".gemini/antigravity-cli"
		base.Dir = true
	default:
		base.RelPath = ".credential"
	}
	return []CredentialPath{base}
}

func restoreCredentialPath(p CredentialPath) error {
	if p.SourceRoot == "" || p.HomeRoot == "" || p.RelPath == "" {
		return nil
	}
	src := filepath.Join(p.SourceRoot, p.RelPath)
	dst := filepath.Join(p.HomeRoot, p.RelPath)
	if filepath.Clean(src) == filepath.Clean(dst) {
		return nil
	}
	if p.Dir {
		return copyCredentialDirTree(src, dst)
	}
	return copyCredentialFilePath(src, dst)
}

func credentialPathPresent(path string, wantDir bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if wantDir {
		return info.IsDir()
	}
	return info.Mode().IsRegular()
}

func copyCredentialFilePath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat credential source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("credential source is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return fmt.Errorf("create credential parent: %w", err)
	}
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("replace credential destination: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open credential source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create credential destination: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy credential: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close credential destination: %w", err)
	}
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return fmt.Errorf("restore credential mtime: %w", err)
	}
	return nil
}

func copyCredentialDirTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat credential dir source: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("credential source is not a directory")
	}
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("replace credential dir destination: %w", err)
	}
	return copyDirRecursive(src, dst, info.Mode().Perm())
}

func copyDirRecursive(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(dst, mode); err != nil {
		return fmt.Errorf("create credential dir destination: %w", err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read credential dir: %w", err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat credential child: %w", err)
		}
		switch {
		case info.IsDir():
			if err := copyDirRecursive(srcPath, dstPath, info.Mode().Perm()); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if err := copyCredentialFilePath(srcPath, dstPath); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported credential path type")
		}
	}
	return nil
}
