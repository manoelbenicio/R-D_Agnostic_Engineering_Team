package auth

import (
	"errors"
	"testing"
)

// configuredProductionSecret is a representative deployment-owned JWT secret:
// it is not a known-insecure placeholder and is at least
// minimumProductionJWTSecretBytes (32) bytes, so it satisfies the real
// production rule enforced by ValidateJWTConfiguration. It is NOT a test-only
// shortcut around validation — a shorter or placeholder secret still fails
// closed (see the FailsClosed cases below).
const configuredProductionSecret = "deployment-owned-secret-0123456789-abcdefgh" // 43 bytes

// isolateAuthEnv makes the JWT configuration tests hermetic: they must not
// depend on the host/CI environment. ValidateJWTConfiguration is a pure
// function of its arguments today, but neutralizing the auth-domain variables
// keeps these tests independent of ambient JWT_SECRET / APP_ENV and any future
// env-reading path, so 5.3 runs identically on every host.
func isolateAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "")
	t.Setenv("APP_ENV", "")
}

func TestValidateJWTConfigurationFailsClosedOutsideExplicitDevelopment(t *testing.T) {
	isolateAuthEnv(t)
	for _, tc := range []struct {
		name   string
		env    string
		secret string
	}{
		{name: "unset mode and secret"},
		{name: "production missing", env: "production"},
		{name: "production known default", env: "production", secret: defaultJWTSecret},
		{name: "production too-short non-default", env: "production", secret: "short-owned-secret"},
		{name: "staging missing", env: "staging"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateJWTConfiguration(tc.env, tc.secret); !errors.Is(err, ErrInsecureJWTConfiguration) {
				t.Fatalf("error = %v, want ErrInsecureJWTConfiguration", err)
			}
		})
	}
}

func TestValidateJWTConfigurationAllowsExplicitDevelopmentAndConfiguredProduction(t *testing.T) {
	isolateAuthEnv(t)
	for _, tc := range []struct {
		name   string
		env    string
		secret string
	}{
		{name: "development default", env: "development", secret: defaultJWTSecret},
		{name: "dev missing", env: "dev"},
		{name: "test missing", env: "test"},
		{name: "production configured", env: "production", secret: configuredProductionSecret},
		{name: "staging configured", env: "staging", secret: configuredProductionSecret},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateJWTConfiguration(tc.env, tc.secret); err != nil {
				t.Fatal(err)
			}
		})
	}
}
