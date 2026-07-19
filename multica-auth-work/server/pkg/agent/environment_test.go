package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestProcessEnvironmentExactDoesNotReinheritParent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "synthetic-parent-provider-key")
	t.Setenv("CODEX_HOME", "/synthetic/parent-codex-home")

	exact := []string{
		"PATH=/synthetic/bin",
		"HOME=/synthetic/task-home",
		"ANTHROPIC_BASE_URL=http://127.0.0.1:20128",
		"ANTHROPIC_AUTH_TOKEN=synthetic-omniroute-only",
	}
	got, err := processEnvironment(Config{
		Env: map[string]string{
			"OPENAI_API_KEY": "synthetic-map-provider-key",
			"CODEX_HOME":     "/synthetic/map-codex-home",
		},
		ExactEnv: exact,
	})
	if err != nil {
		t.Fatalf("processEnvironment: %v", err)
	}
	if !reflect.DeepEqual(got, exact) {
		t.Fatalf("exact process environment changed: got %v want %v", got, exact)
	}
	for _, entry := range got {
		upper := strings.ToUpper(entry)
		if strings.HasPrefix(upper, "OPENAI_API_KEY=") || strings.HasPrefix(upper, "CODEX_HOME=") {
			t.Fatalf("parent or map credential root was reintroduced: %s", entry)
		}
	}

	got[0] = "PATH=/mutated"
	if exact[0] == got[0] {
		t.Fatal("processEnvironment returned caller-owned exact slice")
	}
}

func TestProcessEnvironmentExactFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		exact []string
	}{
		{name: "empty", exact: []string{}},
		{name: "malformed", exact: []string{"PATH"}},
		{name: "invalid key", exact: []string{"1PATH=/synthetic"}},
		{name: "duplicate key", exact: []string{"PATH=/synthetic", "path=/shadow"}},
		{name: "nul value", exact: []string{"PATH=/synthetic\x00shadow"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := processEnvironment(Config{ExactEnv: test.exact})
			if !errors.Is(err, errExactProcessEnvironment) {
				t.Fatalf("error = %v, want exact-environment rejection", err)
			}
		})
	}
}

func TestProcessEnvironmentValueUsesAuthoritativeExactEnvironment(t *testing.T) {
	config := Config{
		Env:      map[string]string{"CODEX_HOME": "/synthetic/legacy-home"},
		ExactEnv: []string{"PATH=/synthetic/bin", "CODEX_HOME=/synthetic/controlled-home"},
	}
	if got := processEnvironmentValue(config, "codex_home"); got != "/synthetic/controlled-home" {
		t.Fatalf("CODEX_HOME = %q, want controlled exact value", got)
	}
	if got := processEnvironmentValue(Config{Env: config.Env}, "CODEX_HOME"); got != "/synthetic/legacy-home" {
		t.Fatalf("legacy CODEX_HOME = %q", got)
	}
}

func TestCredentiallessBackendsFailClosedOnEmptyExactEnvironment(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}

	for _, agentType := range []string{"claude", "codex"} {
		t.Run(agentType, func(t *testing.T) {
			backend, err := New(agentType, Config{
				ExecutablePath: executable,
				ExactEnv:       []string{},
			})
			if err != nil {
				t.Fatalf("New(%q): %v", agentType, err)
			}

			if _, err := backend.Execute(context.Background(), "synthetic prompt", ExecOptions{}); !errors.Is(err, errExactProcessEnvironment) {
				t.Fatalf("Execute error = %v, want exact-environment rejection", err)
			}
		})
	}
}

func TestExactEnvironmentExecutableResolutionIgnoresParentPath(t *testing.T) {
	parentDirectory := t.TempDir()
	exactDirectory := t.TempDir()
	t.Setenv("PATH", parentDirectory)

	for _, agentType := range []string{"claude", "codex"} {
		t.Run(agentType, func(t *testing.T) {
			executableName := agentType
			if runtime.GOOS == "windows" {
				executableName += ".exe"
			}
			parentExecutable := filepath.Join(parentDirectory, executableName)
			exactExecutable := filepath.Join(exactDirectory, executableName)
			writeSyntheticExecutable(t, parentExecutable)
			writeSyntheticExecutable(t, exactExecutable)

			config := Config{
				ExecutablePath: executableName,
				ExactEnv: []string{
					"PATH=" + exactDirectory,
					"HOME=" + filepath.Join(exactDirectory, "home"),
				},
			}
			environment, err := processEnvironment(config)
			if err != nil {
				t.Fatalf("processEnvironment: %v", err)
			}
			resolved, err := resolveProcessExecutable(config, agentType, environment)
			if err != nil {
				t.Fatalf("resolveProcessExecutable: %v", err)
			}
			if resolved != exactExecutable {
				t.Fatalf("resolved executable = %q, want exact-environment executable %q", resolved, exactExecutable)
			}
			if resolved == parentExecutable {
				t.Fatal("exact-environment resolution selected the daemon-parent PATH executable")
			}
		})
	}
}

func TestExactEnvironmentExecutableResolutionDoesNotFallbackToParentPath(t *testing.T) {
	parentDirectory := t.TempDir()
	exactDirectory := t.TempDir()
	executableName := "codex"
	if runtime.GOOS == "windows" {
		executableName += ".exe"
	}
	writeSyntheticExecutable(t, filepath.Join(parentDirectory, executableName))
	t.Setenv("PATH", parentDirectory)

	config := Config{
		ExecutablePath: executableName,
		ExactEnv:       []string{"PATH=" + exactDirectory, "HOME=" + filepath.Join(exactDirectory, "home")},
	}
	environment, err := processEnvironment(config)
	if err != nil {
		t.Fatalf("processEnvironment: %v", err)
	}
	if resolved, err := resolveProcessExecutable(config, "codex", environment); err == nil {
		t.Fatalf("resolved parent executable %q despite empty exact PATH", resolved)
	}
}

func TestExactEnvironmentExecutableResolutionRejectsUncontrolledPaths(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		path       string
	}{
		{name: "relative executable path", executable: filepath.Join("relative", "codex"), path: t.TempDir()},
		{name: "relative PATH entry", executable: "codex", path: filepath.Join("relative", "bin")},
		{name: "noncanonical PATH entry", executable: "codex", path: t.TempDir() + string(filepath.Separator) + ".." + string(filepath.Separator) + "bin"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := Config{ExecutablePath: test.executable, ExactEnv: []string{"PATH=" + test.path}}
			environment, err := processEnvironment(config)
			if err != nil {
				t.Fatalf("processEnvironment: %v", err)
			}
			if resolved, err := resolveProcessExecutable(config, "codex", environment); err == nil {
				t.Fatalf("resolved uncontrolled executable %q", resolved)
			}
		})
	}
}

func writeSyntheticExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("synthetic executable fixture"), 0o700); err != nil {
		t.Fatalf("write synthetic executable: %v", err)
	}
}
