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

func TestRequiresCredentialIsolationIncludesNIM(t *testing.T) {
	for _, provider := range []string{"nim", " NIM "} {
		if !requiresCredentialIsolation(provider) {
			t.Errorf("requiresCredentialIsolation(%q) = false, want true", provider)
		}
	}
}
