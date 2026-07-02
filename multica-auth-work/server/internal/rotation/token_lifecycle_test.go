package rotation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInspectCodexCredentialStale(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	authPath := writeCodexAuthFixture(t, now.Add(-21*24*time.Hour))

	got := InspectCodexCredential(authPath, now)

	if got.Usable {
		t.Fatal("Usable = true, want false")
	}
	if got.Reason != credentialReasonStale {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonStale)
	}
	if got.AgeDays != 21 {
		t.Fatalf("AgeDays = %d, want 21", got.AgeDays)
	}
	if got.LastRefresh == nil || !got.LastRefresh.Equal(now.Add(-21*24*time.Hour)) {
		t.Fatalf("LastRefresh = %v, want fixture time", got.LastRefresh)
	}
}

func TestInspectCodexCredentialFresh(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	authPath := writeCodexAuthFixture(t, now.Add(-time.Hour))

	got := InspectCodexCredential(authPath, now)

	if !got.Usable {
		t.Fatalf("Usable = false, Reason = %q, want usable", got.Reason)
	}
	if got.Reason != credentialReasonFresh {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonFresh)
	}
	if got.AgeDays != 0 {
		t.Fatalf("AgeDays = %d, want 0", got.AgeDays)
	}
}

func TestInspectCodexCredentialMissing(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	got := InspectCodexCredential(filepath.Join(t.TempDir(), "auth.json"), now)

	if got.Usable {
		t.Fatal("Usable = true, want false")
	}
	if got.Reason != credentialReasonMissing {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonMissing)
	}
}

func TestInspectCodexCredentialUnparseable(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"last_refresh":"not-rfc3339"}`), 0o600); err != nil {
		t.Fatalf("write auth fixture: %v", err)
	}

	got := InspectCodexCredential(authPath, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC))

	if got.Usable {
		t.Fatal("Usable = true, want false")
	}
	if got.Reason != credentialReasonUnparseable {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonUnparseable)
	}
}

func TestInspectCodexCredentialCustomThreshold(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	authPath := writeCodexAuthFixture(t, now.Add(-36*time.Hour))

	got := InspectCodexCredentialWithThreshold(authPath, now, 24*time.Hour)

	if got.Usable {
		t.Fatal("Usable = true, want false")
	}
	if got.Reason != credentialReasonStale {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonStale)
	}
	if got.AgeDays != 1 {
		t.Fatalf("AgeDays = %d, want 1", got.AgeDays)
	}
}

func TestVerifyCodexLoginUsesInjectedChecker(t *testing.T) {
	checker := &fakeLoginStatusChecker{loggedIn: true}

	got := VerifyCodexLogin(context.Background(), "/tmp/codex-home", checker)

	if !got.Usable {
		t.Fatalf("Usable = false, Reason = %q, want usable", got.Reason)
	}
	if got.Reason != credentialReasonLoggedIn {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonLoggedIn)
	}
	if checker.seenHomeDir != "/tmp/codex-home" {
		t.Fatalf("checker homeDir = %q, want /tmp/codex-home", checker.seenHomeDir)
	}
}

func TestVerifyCodexLoginNotLoggedIn(t *testing.T) {
	got := VerifyCodexLogin(context.Background(), "/tmp/codex-home", &fakeLoginStatusChecker{loggedIn: false})

	if got.Usable {
		t.Fatal("Usable = true, want false")
	}
	if got.Reason != credentialReasonNotLoggedIn {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonNotLoggedIn)
	}
}

func TestVerifyCodexLoginStatusError(t *testing.T) {
	got := VerifyCodexLogin(context.Background(), "/tmp/codex-home", &fakeLoginStatusChecker{err: errors.New("boom")})

	if got.Usable {
		t.Fatal("Usable = true, want false")
	}
	if got.Reason != credentialReasonStatusError {
		t.Fatalf("Reason = %q, want %q", got.Reason, credentialReasonStatusError)
	}
}

func TestParseCodexLoginStatus(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		runErr  error
		want    bool
		wantErr bool
	}{
		{
			name:   "logged in",
			output: "Logged in using ChatGPT",
			want:   true,
		},
		{
			name:   "not logged in",
			output: "Not logged in",
			want:   false,
		},
		{
			name:    "unrecognized output",
			output:  "unknown status",
			wantErr: true,
		},
		{
			name:    "command error",
			runErr:  errors.New("exit"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCodexLoginStatus(tt.output, tt.runErr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("loggedIn = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeLoginStatusChecker struct {
	loggedIn    bool
	err         error
	seenHomeDir string
}

func (f *fakeLoginStatusChecker) Status(_ context.Context, homeDir string) (bool, error) {
	f.seenHomeDir = homeDir
	return f.loggedIn, f.err
}

func writeCodexAuthFixture(t *testing.T, lastRefresh time.Time) string {
	t.Helper()

	authPath := filepath.Join(t.TempDir(), "auth.json")
	data := []byte(`{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{},"last_refresh":"` + lastRefresh.Format(time.RFC3339) + `"}`)
	if err := os.WriteFile(authPath, data, 0o600); err != nil {
		t.Fatalf("write auth fixture: %v", err)
	}
	return authPath
}
