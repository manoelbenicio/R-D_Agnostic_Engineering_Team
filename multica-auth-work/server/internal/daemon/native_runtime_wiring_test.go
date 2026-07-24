package daemon

import (
	"context"
	"testing"
)

func TestNIMUsesNativeHTTPRuntimeVersion(t *testing.T) {
	got, err := runtimeVersion(context.Background(), "nim", "")
	if err != nil {
		t.Fatalf("runtimeVersion(nim): %v", err)
	}
	if got != "native-http" {
		t.Fatalf("runtimeVersion(nim) = %q, want native-http", got)
	}
}

// TestRequiresCredentialIsolationIncludesNIM was removed (REC-DAEMON-TEST,
// Principal adjudication 2026-07-22). The daemon-level requiresCredentialIsolation
// gate was retired in the OmniRoute credentialless cleanup. Provider-secret
// isolation for NIM/NVIDIA — fail-closed exclusion of NVIDIA_API_KEY / NIM_*
// keys per AB tasks 2.4/2.5 — is now enforced by the runtimeenv environment
// deny-list (internal/daemon/runtimeenv/policy.go) and covered by
// runtimeenv/env_test.go (TestBuildMinimalInheritedRemovesCredentialAndRoutingSurface)
// and runtimeenv/isolation_g4_test.go. No daemon-level helper or test remains,
// and credential/account rotation is intentionally NOT reintroduced.
