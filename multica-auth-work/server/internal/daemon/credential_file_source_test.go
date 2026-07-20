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
	// Realistic key shape: single token, no whitespace, >= 8 chars.
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
				real := filepath.Join(dir, "real-symlink-test")
				os.WriteFile(real, []byte("validkey12345678"), 0o600)
				link := filepath.Join(dir, "link")
				os.Symlink(real, link)
				return link
			},
			reason: "secret_file_not_regular",
		},
		{
			name: "world_readable",
			setup: func(t *testing.T) string {
				p := filepath.Join(dir, "world-readable")
				os.WriteFile(p, []byte("validkey12345678"), 0o644)
				return p
			},
			reason: "secret_file_world_readable",
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
				p := filepath.Join(dir, "control")
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
			// For empty ref, we pass a zero SecretFileRef directly.
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
			// Verify error message never contains path or content.
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
	if err == nil || err != context.Canceled {
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
