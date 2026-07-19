//go:build windows

package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const windowsDiscoveryHelperMarkerEnv = "MULTICA_SYNTHETIC_DISCOVERY_START_MARKER"

func TestWindowsACPDiscoveryFailsClosedBeforeStart(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	t.Setenv(windowsDiscoveryHelperMarkerEnv, marker)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	models, err := discoverACPModels(context.Background(), executable, acpDiscoveryProvider{
		clientName:   "synthetic-windows-containment-test",
		acpArgs:      []string{"-test.run=TestWindowsDiscoveryStartHelper"},
		tmpdirPrefix: "synthetic-windows-containment-",
	})
	if !errors.Is(err, errDiscoveryProcessContainmentUnavailable) {
		t.Fatalf("ACP discovery error = %v, want containment failure", err)
	}
	if models != nil {
		t.Fatalf("ACP discovery models = %+v, want nil", models)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("Windows discovery process started before containment refusal: %v", statErr)
	}
}

func TestWindowsDynamicModelDiscoveryFailsClosed(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	models, err := ListModels(context.Background(), "cursor", executable)
	if !errors.Is(err, errDiscoveryProcessContainmentUnavailable) {
		t.Fatalf("Cursor discovery error = %v, want containment failure", err)
	}
	if models != nil {
		t.Fatalf("Cursor discovery models = %+v, want nil", models)
	}
}

func TestWindowsDiscoveryStartHelper(t *testing.T) {
	marker := os.Getenv(windowsDiscoveryHelperMarkerEnv)
	if marker == "" {
		return
	}
	if err := os.WriteFile(marker, []byte("started"), 0o600); err != nil {
		t.Fatal(err)
	}
}
