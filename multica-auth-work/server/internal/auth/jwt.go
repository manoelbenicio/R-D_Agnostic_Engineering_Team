package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

const defaultJWTSecret = "multica-dev-secret-change-in-production"

var ErrInsecureJWTConfiguration = errors.New("JWT_SECRET must be set to a non-development value outside explicit development or test mode")

var (
	jwtSecret     []byte
	jwtSecretOnce sync.Once
)

func JWTSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = defaultJWTSecret
		}
		jwtSecret = []byte(secret)
	})

	return jwtSecret
}

// ValidateJWTConfiguration prevents a production-like server from starting
// with no signing secret or with the repository's known development secret.
// The insecure default remains available only when APP_ENV explicitly opts
// into a development or test mode.
func ValidateJWTConfiguration(appEnv, secret string) error {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "dev", "development", "test":
		return nil
	}
	if strings.TrimSpace(secret) == "" || secret == defaultJWTSecret {
		return ErrInsecureJWTConfiguration
	}
	return nil
}

// GeneratePATToken creates a new personal access token: "mul_" + 40 random hex chars.
func GeneratePATToken() (string, error) {
	b := make([]byte, 20) // 20 bytes = 40 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate PAT token: %w", err)
	}
	return "mul_" + hex.EncodeToString(b), nil
}

// GenerateDaemonToken creates a new daemon auth token: "mdt_" + 40 random hex chars.
func GenerateDaemonToken() (string, error) {
	b := make([]byte, 20) // 20 bytes = 40 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate daemon token: %w", err)
	}
	return "mdt_" + hex.EncodeToString(b), nil
}

// GenerateAgentTaskToken creates a new task-scoped agent auth token:
// "mat_" + 40 random hex chars. The token is single-purpose — bound to a
// specific (agent_id, task_id) pair on the server side — and is what the
// daemon injects into the agent process in place of its own owner PAT.
// See MUL-2600.
func GenerateAgentTaskToken() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate agent task token: %w", err)
	}
	return "mat_" + hex.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 hash of a token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
