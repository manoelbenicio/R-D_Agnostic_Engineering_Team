//go:build !windows

package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestFileCredentialSource_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	if err := os.WriteFile(path, []byte("  sk-test-omniroute-key-value-1234  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ref, err := brain.NewSecretFileRef(path)
	if err != nil {
		t.Fatal(err)
	}

	var received string
	src := FileCredentialSource{}
	if err := src.WithCredential(context.Background(), ref, func(value string) error {
		received = value
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != "sk-test-omniroute-key-value-1234" {
		t.Fatalf("expected trimmed key, got %q", received)
	}
}

func TestFileCredentialSource_SymlinkTOCTOU(t *testing.T) {
	// O_NOFOLLOW atomically rejects symlinks at open time.
	dir := t.TempDir()
	real := filepath.Join(dir, "real-key")
	os.WriteFile(real, []byte("validkey12345678"), 0o600)
	link := filepath.Join(dir, "symlink-key")
	os.Symlink(real, link)

	ref, _ := brain.NewSecretFileRef(link)
	src := FileCredentialSource{}
	err := src.WithCredential(context.Background(), ref, func(value string) error {
		t.Fatal("callback must never be reached via symlink")
		return nil
	})
	if err == nil {
		t.Fatal("expected symlink rejection")
	}
	credErr, ok := err.(*credentialFileError)
	if !ok {
		t.Fatalf("expected *credentialFileError, got %T", err)
	}
	if credErr.reason != "secret_file_is_symlink" {
		t.Fatalf("expected secret_file_is_symlink, got %q", credErr.reason)
	}
}

func TestFileCredentialSource_ModeRejection(t *testing.T) {
	dir := t.TempDir()
	// Write then chmod — avoids umask stripping bits during WriteFile.
	writeAndChmod := func(name string, mode os.FileMode) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte("validkey12345678"), 0o600)
		os.Chmod(p, mode)
		return p
	}

	modes := []struct {
		mode   os.FileMode
		reject bool
	}{
		{0o600, false}, // owner rw — accepted
		{0o400, false}, // owner ro — accepted
		{0o640, true},  // group read
		{0o620, true},  // group write
		{0o610, true},  // group execute
		{0o604, true},  // world read
		{0o602, true},  // world write
		{0o601, true},  // world execute
	}
	for _, m := range modes {
		name := filepath.Join(dir, "mode-"+m.mode.String())
		p := writeAndChmod(filepath.Base(name), m.mode)
		ref, _ := brain.NewSecretFileRef(p)
		src := FileCredentialSource{}
		err := src.WithCredential(context.Background(), ref, func(string) error { return nil })
		if m.reject && err == nil {
			t.Errorf("mode %04o should be rejected", m.mode)
		}
		if !m.reject && err != nil {
			t.Errorf("mode %04o should be accepted, got: %v", m.mode, err)
		}
		if m.reject && err != nil {
			if ce, ok := err.(*credentialFileError); ok {
				if ce.reason != "secret_file_permissions_too_open" {
					t.Errorf("mode %04o: expected permissions_too_open, got %q", m.mode, ce.reason)
				}
			}
		}
	}
}

func TestFileCredentialSource_OwnerEnforcement(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test owner enforcement as root — uid 0 owns all files")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "own-key")
	os.WriteFile(path, []byte("validkey12345678"), 0o600)
	ref, _ := brain.NewSecretFileRef(path)
	src := FileCredentialSource{}
	// File created by current user must pass owner check.
	if err := src.WithCredential(context.Background(), ref, func(string) error { return nil }); err != nil {
		t.Fatalf("current-owner file should pass: %v", err)
	}
}

func TestFileCredentialSource_FailClosed(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name   string
		setup  func(t *testing.T) string
		reason string
	}{
		{
			name:   "empty_ref",
			setup:  func(t *testing.T) string { return "" },
			reason: "secret_file_ref_empty",
		},
		{
			name: "file_not_found",
			setup: func(t *testing.T) string {
				return filepath.Join(dir, "nonexistent")
			},
			reason: "secret_file_not_found",
		},
		{
			name: "symlink_rejected",
			setup: func(t *testing.T) string {
				real := filepath.Join(dir, "real")
				os.WriteFile(real, []byte("validkey12345678"), 0o600)
				link := filepath.Join(dir, "link")
				os.Symlink(real, link)
				return link
			},
			reason: "secret_file_is_symlink",
		},
		{
			name: "world_readable_rejected",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "world-r")
				os.WriteFile(p, []byte("validkey12345678"), 0o600)
				os.Chmod(p, 0o604)
				return p
			},
			reason: "secret_file_permissions_too_open",
		},
		{
			name: "group_readable_rejected",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "group-r")
				os.WriteFile(p, []byte("validkey12345678"), 0o600)
				os.Chmod(p, 0o640)
				return p
			},
			reason: "secret_file_permissions_too_open",
		},
		{
			name: "content_too_short",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "short")
				os.WriteFile(p, []byte("abc"), 0o600)
				return p
			},
			reason: "secret_file_content_too_short",
		},
		{
			name: "content_has_whitespace",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "spaces")
				os.WriteFile(p, []byte("key with spaces inside"), 0o600)
				return p
			},
			reason: "secret_file_content_has_whitespace",
		},
		{
			name: "content_has_control_char",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "ctrl")
				os.WriteFile(p, []byte("validkey\x00hidden"), 0o600)
				return p
			},
			reason: "secret_file_content_has_control_char",
		},
		{
			name: "file_too_large",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "large")
				data := strings.Repeat("x", credentialFileMaxBytes+1)
				os.WriteFile(p, []byte(data), 0o600)
				return p
			},
			reason: "secret_file_too_large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			var ref brain.SecretFileRef
			if path != "" {
				var err error
				ref, err = brain.NewSecretFileRef(path)
				if err != nil {
					t.Fatalf("NewSecretFileRef: %v", err)
				}
			}
			src := FileCredentialSource{}
			err := src.WithCredential(context.Background(), ref, func(value string) error {
				t.Fatal("callback must not be reached on failure")
				return nil
			})
			if err == nil {
				t.Fatal("expected error")
			}
			credErr, ok := err.(*credentialFileError)
			if !ok {
				t.Fatalf("expected *credentialFileError, got %T: %v", err, err)
			}
			if credErr.reason != tt.reason {
				t.Fatalf("expected reason %q, got %q", tt.reason, credErr.reason)
			}
			msg := err.Error()
			if path != "" && strings.Contains(msg, path) {
				t.Fatalf("error message leaks path: %s", msg)
			}
		})
	}
}

func TestFileCredentialSource_CancelledContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	os.WriteFile(path, []byte("validkey12345678"), 0o600)
	ref, _ := brain.NewSecretFileRef(path)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	src := FileCredentialSource{}
	err := src.WithCredential(ctx, ref, func(value string) error {
		t.Fatal("callback must not be reached on cancelled context")
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFileCredentialSource_CallbackErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	os.WriteFile(path, []byte("validkey12345678"), 0o600)
	ref, _ := brain.NewSecretFileRef(path)

	sentinel := &credentialFileError{reason: "callback_test"}
	src := FileCredentialSource{}
	err := src.WithCredential(context.Background(), ref, func(value string) error {
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
