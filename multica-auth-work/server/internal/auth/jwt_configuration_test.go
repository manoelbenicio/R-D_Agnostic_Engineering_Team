package auth

import (
	"errors"
	"testing"
)

func TestValidateJWTConfigurationFailsClosedOutsideExplicitDevelopment(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    string
		secret string
	}{
		{name: "unset mode and secret"},
		{name: "production missing", env: "production"},
		{name: "production known default", env: "production", secret: defaultJWTSecret},
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
	for _, tc := range []struct {
		name   string
		env    string
		secret string
	}{
		{name: "development default", env: "development", secret: defaultJWTSecret},
		{name: "dev missing", env: "dev"},
		{name: "test missing", env: "test"},
		{name: "production configured", env: "production", secret: "deployment-owned-secret"},
		{name: "staging configured", env: "staging", secret: "deployment-owned-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateJWTConfiguration(tc.env, tc.secret); err != nil {
				t.Fatal(err)
			}
		})
	}
}
