package rotation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DefaultCodexCredentialStalenessThreshold is the offline expiry heuristic for
// Codex auth.json credentials. It is intentionally conservative and does not
// make a network call; callers that need a different policy can use
// InspectCodexCredentialWithThreshold.
const DefaultCodexCredentialStalenessThreshold = 7 * 24 * time.Hour

const (
	credentialReasonFresh              = "fresh"
	credentialReasonStale              = "stale"
	credentialReasonMissing            = "missing"
	credentialReasonUnreadable         = "unreadable"
	credentialReasonUnparseable        = "unparseable"
	credentialReasonLoggedIn           = "logged_in"
	credentialReasonNotLoggedIn        = "not_logged_in"
	credentialReasonStatusError        = "status_error"
	credentialReasonCheckerUnavailable = "checker_unavailable"
)

var errUnrecognizedCodexLoginStatus = errors.New("rotation: unrecognized codex login status")

type CredentialLiveness struct {
	Usable      bool
	Reason      string
	LastRefresh *time.Time
	AgeDays     int
}

// LoginStatusChecker is the injectable port for live login checks. Tests should
// provide a fake implementation instead of invoking the real Codex CLI.
type LoginStatusChecker interface {
	Status(ctx context.Context, homeDir string) (loggedIn bool, err error)
}

type codexAuthJSON struct {
	LastRefresh string `json:"last_refresh"`
}

// InspectCodexCredential inspects a Codex auth.json without logging credential
// material or contacting the network. The default stale-token heuristic is
// DefaultCodexCredentialStalenessThreshold.
func InspectCodexCredential(authJSONPath string, now time.Time) CredentialLiveness {
	return InspectCodexCredentialWithThreshold(authJSONPath, now, DefaultCodexCredentialStalenessThreshold)
}

// InspectCodexCredentialWithThreshold inspects a Codex auth.json using a caller
// supplied staleness threshold. Non-positive thresholds fall back to the default.
func InspectCodexCredentialWithThreshold(authJSONPath string, now time.Time, threshold time.Duration) CredentialLiveness {
	if threshold <= 0 {
		threshold = DefaultCodexCredentialStalenessThreshold
	}

	data, err := os.ReadFile(authJSONPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CredentialLiveness{Reason: credentialReasonMissing}
		}
		return CredentialLiveness{Reason: credentialReasonUnreadable}
	}

	var auth codexAuthJSON
	if err := json.Unmarshal(data, &auth); err != nil {
		return CredentialLiveness{Reason: credentialReasonUnparseable}
	}

	lastRefreshRaw := strings.TrimSpace(auth.LastRefresh)
	if lastRefreshRaw == "" {
		return CredentialLiveness{Reason: credentialReasonUnparseable}
	}

	lastRefresh, err := time.Parse(time.RFC3339, lastRefreshRaw)
	if err != nil {
		return CredentialLiveness{Reason: credentialReasonUnparseable}
	}

	age := now.Sub(lastRefresh)
	ageDays := wholeAgeDays(age)
	if age > threshold {
		return CredentialLiveness{
			Usable:      false,
			Reason:      credentialReasonStale,
			LastRefresh: &lastRefresh,
			AgeDays:     ageDays,
		}
	}

	return CredentialLiveness{
		Usable:      true,
		Reason:      credentialReasonFresh,
		LastRefresh: &lastRefresh,
		AgeDays:     ageDays,
	}
}

// VerifyCodexLogin verifies live Codex login status through an injected checker.
func VerifyCodexLogin(ctx context.Context, homeDir string, checker LoginStatusChecker) CredentialLiveness {
	if checker == nil {
		return CredentialLiveness{Reason: credentialReasonCheckerUnavailable}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	loggedIn, err := checker.Status(ctx, homeDir)
	if err != nil {
		return CredentialLiveness{Reason: credentialReasonStatusError}
	}
	if !loggedIn {
		return CredentialLiveness{Reason: credentialReasonNotLoggedIn}
	}
	return CredentialLiveness{Usable: true, Reason: credentialReasonLoggedIn}
}

// CodexCLIStatusChecker is the production adapter for "codex login status".
// It only reports coarse liveness and never includes command output in errors.
type CodexCLIStatusChecker struct {
	Binary string
}

func (c CodexCLIStatusChecker) Status(ctx context.Context, homeDir string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	binary := strings.TrimSpace(c.Binary)
	if binary == "" {
		binary = "codex"
	}

	cmd := exec.CommandContext(ctx, binary, "login", "status")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+homeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return parseCodexLoginStatus(stdout.String()+"\n"+stderr.String(), err)
}

func parseCodexLoginStatus(output string, runErr error) (bool, error) {
	switch {
	case strings.Contains(output, "Logged in using ChatGPT"):
		return true, nil
	case strings.Contains(output, "Not logged in"):
		return false, nil
	case runErr != nil:
		return false, runErr
	default:
		return false, errUnrecognizedCodexLoginStatus
	}
}

func wholeAgeDays(age time.Duration) int {
	if age <= 0 {
		return 0
	}
	return int(age / (24 * time.Hour))
}
