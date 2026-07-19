package runtimeenv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const (
	g4SyntheticProviderValue = "synthetic-provider-native-value"
	g4SyntheticCookieValue   = "synthetic-cookie-value"
	g4ProcessHelperKey       = "RUNTIMEENV_G4_HELPER"
	g4ProcessHelperModeKey   = "RUNTIMEENV_G4_HELPER_MODE"
)

var g4DirectProviderFragments = []string{
	"api.anthropic.",
	"api.openai.",
	"api.moonshot.",
	"integrate.api.nvidia.",
	"generativelanguage.googleapis.",
	"open.bigmodel.",
}

func TestG4ChildHomeProcessTreeAndLogIsolation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-tree environment inspection requires Linux procfs")
	}

	for _, cli := range []brain.CLIKind{brain.CLIClaudeCode, brain.CLICodex} {
		t.Run(string(cli), func(t *testing.T) {
			taskHome := t.TempDir()
			codexHome := filepath.Join(taskHome, "codex")
			if err := os.MkdirAll(codexHome, 0o700); err != nil {
				t.Fatal("controlled Codex directory creation failed")
			}
			secret, err := NewStableSecret(g4SyntheticGatewayValue)
			if err != nil {
				t.Fatal("synthetic stable value was rejected")
			}
			environment, report, err := BuildGatewayEnvironment(ComposeOptions{
				Inherited: []string{
					"PATH=/usr/bin",
					"OPENAI_API_KEY=" + g4SyntheticProviderValue,
					"NVIDIA_API_KEY=" + g4SyntheticProviderValue,
					"ANTHROPIC_BASE_URL=https://api.anthropic.invalid",
					"COOKIE=" + g4SyntheticCookieValue,
				},
				Local:  map[string]string{"TASK_RUNTIME_MODE": "synthetic-g4"},
				Custom: map[string]string{"SAFE_FIXTURE_MODE": "enabled"},
				Adapter: AdapterEnvironment{
					CLI: cli, GatewayRoot: "http://127.0.0.1:21999", TaskHome: taskHome,
					CodexHome: codexHome, StableSecret: secret,
				},
			})
			if err != nil {
				t.Fatal("credentialless synthetic environment construction failed")
			}

			var codexConfig *CodexConfigContract
			if cli == brain.CLICodex {
				model, err := brain.ParseRouteModel("openai/codex-g4-fixture")
				if err != nil {
					t.Fatal("synthetic Codex model was rejected")
				}
				config, err := NewCodexConfigContract("http://127.0.0.1:21999", model, testCorrelation())
				if err != nil {
					t.Fatal("controlled Codex configuration construction failed")
				}
				codexConfig = &config
				if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), config.Bytes(), 0o600); err != nil {
					t.Fatal("controlled Codex configuration write failed")
				}
			}
			if err := os.WriteFile(filepath.Join(taskHome, "session.json"), []byte(`{"fixture":"local-task-state"}`), 0o600); err != nil {
				t.Fatal("approved local task-state write failed")
			}

			manifest, err := g4SafeHomeManifest(taskHome)
			if err != nil {
				t.Fatal("controlled task-home inspection failed")
			}
			if err := AssertPreLaunch(LaunchPlan{
				Environment: environment, CodexConfig: codexConfig, TaskHome: manifest, ExecutionRoot: taskHome,
			}); err != nil {
				t.Fatal("pre-launch isolation assertion failed")
			}
			assertG4SafeHomeContents(t, taskHome)

			var diagnostic bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&diagnostic, nil))
			logger.Info("runtime_isolation", "environment", environment, "removals", report.Removed)
			assertG4SafeDiagnostic(t, diagnostic.Bytes())

			rootPID, leafPID, stderr, stop := startG4SyntheticProcessTree(t, environment)
			defer stop()
			for _, pid := range []int{rootPID, leafPID} {
				environ, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
				if err != nil {
					t.Fatal("synthetic process environment inspection failed")
				}
				assertG4SafeProcessEnvironment(t, cli, environ)
				cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
				if err != nil {
					t.Fatal("synthetic process command-line inspection failed")
				}
				assertG4SafeDiagnostic(t, cmdline)
			}
			if parent := g4ProcessParent(t, leafPID); parent != rootPID {
				t.Fatal("synthetic leaf process was not owned by the inspected root")
			}
			stop()
			assertG4SafeDiagnostic(t, stderr.Bytes())
		})
	}
}

func TestG4NativeCredentialBearingAdaptersStayFailClosed(t *testing.T) {
	tests := []struct {
		cli  brain.CLIKind
		gate AdapterGate
	}{
		{cli: brain.CLIOpenAICompatible, gate: GateOpenAICompatibleUnaccepted},
		{cli: brain.CLIKimi, gate: GateNativeKimiUnaccepted},
		{cli: brain.CLINIM, gate: GateNativeNIMUnaccepted},
		{cli: brain.CLIAntigravity, gate: GateNativeAntigravityUnaccepted},
	}

	for _, test := range tests {
		t.Run(string(test.cli), func(t *testing.T) {
			contract, err := CredentiallessAdapterContract(test.cli)
			if !errors.Is(err, ErrAdapterFailClosed) || contract.State != AdapterFailClosed || contract.Gate != test.gate {
				t.Fatal("unaccepted native adapter did not fail closed")
			}
			if strings.Contains(fmt.Sprint(err), g4SyntheticGatewayValue) {
				t.Fatal("fail-closed adapter error exposed synthetic value")
			}

			secret, err := NewStableSecret(g4SyntheticGatewayValue)
			if err != nil {
				t.Fatal("synthetic stable value was rejected")
			}
			environment, _, buildErr := BuildGatewayEnvironment(ComposeOptions{
				Inherited: []string{"PATH=/usr/bin", "OPENAI_API_KEY=" + g4SyntheticProviderValue},
				Adapter: AdapterEnvironment{
					CLI: test.cli, GatewayRoot: "http://127.0.0.1:21999", TaskHome: "/controlled/g4-native",
					CodexHome: "/controlled/g4-native/codex", StableSecret: secret,
				},
			})
			if !errors.Is(buildErr, ErrAdapterFailClosed) || len(environment.Keys()) != 0 {
				t.Fatal("unaccepted native adapter produced a child environment")
			}
			if strings.Contains(fmt.Sprint(buildErr), g4SyntheticGatewayValue) || strings.Contains(fmt.Sprint(buildErr), g4SyntheticProviderValue) {
				t.Fatal("fail-closed environment error exposed synthetic values")
			}
		})
	}
}

func TestG4IsolationProcessHelper(t *testing.T) {
	if os.Getenv(g4ProcessHelperKey) != "1" {
		return
	}
	if os.Getenv(g4ProcessHelperModeKey) == "leaf" {
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}

	leaf := exec.Command(os.Args[0], "-test.run=^TestG4IsolationProcessHelper$")
	leaf.Env = g4ReplaceEnvironment(os.Environ(), g4ProcessHelperModeKey, "leaf")
	leafInput, err := leaf.StdinPipe()
	if err != nil {
		os.Exit(2)
	}
	leaf.Stdout = io.Discard
	leaf.Stderr = io.Discard
	if err := leaf.Start(); err != nil {
		os.Exit(2)
	}
	_, _ = fmt.Fprintf(os.Stdout, "%d %d\n", os.Getpid(), leaf.Process.Pid)
	_, _ = io.Copy(io.Discard, os.Stdin)
	_ = leafInput.Close()
	_ = leaf.Wait()
	os.Exit(0)
}

func startG4SyntheticProcessTree(t *testing.T, environment ChildEnvironment) (int, int, *bytes.Buffer, func()) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestG4IsolationProcessHelper$")
	cmd.Env = append(environment.Exec(), g4ProcessHelperKey+"=1", g4ProcessHelperModeKey+"=root")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal("synthetic process-tree input creation failed")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("synthetic process-tree output creation failed")
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatal("synthetic process-tree launch failed")
	}
	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("synthetic process-tree identity read failed")
	}
	fields := strings.Fields(line)
	if len(fields) != 2 {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("synthetic process-tree identity was malformed")
	}
	rootPID, rootErr := strconv.Atoi(fields[0])
	leafPID, leafErr := strconv.Atoi(fields[1])
	if rootErr != nil || leafErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("synthetic process-tree identity was invalid")
	}

	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("synthetic process-tree shutdown failed")
		}
	}
	return rootPID, leafPID, stderr, stop
}

func g4ReplaceEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}

func assertG4SafeProcessEnvironment(t *testing.T, cli brain.CLIKind, raw []byte) {
	t.Helper()
	stableCount := 0
	for _, encoded := range bytes.Split(raw, []byte{0}) {
		if len(encoded) == 0 {
			continue
		}
		key, value, ok := strings.Cut(string(encoded), "=")
		if !ok {
			t.Fatal("synthetic process environment entry was malformed")
		}
		classification := ClassifyEnvironmentKey(key)
		if classification.Denied && !g4TrustedObservedKey(cli, strings.ToUpper(key)) {
			t.Fatal("synthetic process inherited a denied environment key")
		}
		stableEntry := key == "ANTHROPIC_AUTH_TOKEN" || key == CodexOmniRouteAPIKeyEnv
		if stableEntry {
			stableCount++
			if value != g4SyntheticGatewayValue {
				t.Fatal("synthetic process stable value did not match its controlled fixture")
			}
		}
		if !stableEntry {
			assertG4SafeDiagnostic(t, []byte(value))
		}
	}
	if stableCount != 1 {
		t.Fatal("synthetic process did not contain exactly one controlled gateway value")
	}
}

func g4TrustedObservedKey(cli brain.CLIKind, key string) bool {
	if key == "HOME" {
		return true
	}
	switch cli {
	case brain.CLIClaudeCode:
		return key == "ANTHROPIC_BASE_URL" || key == "ANTHROPIC_AUTH_TOKEN"
	case brain.CLICodex:
		return key == "CODEX_HOME" || key == CodexOmniRouteAPIKeyEnv
	default:
		return false
	}
}

func g4ProcessParent(t *testing.T, pid int) int {
	t.Helper()
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		t.Fatal("synthetic process parent inspection failed")
	}
	closeParen := bytes.LastIndexByte(raw, ')')
	if closeParen < 0 {
		t.Fatal("synthetic process stat was malformed")
	}
	fields := strings.Fields(string(raw[closeParen+1:]))
	if len(fields) < 2 {
		t.Fatal("synthetic process stat was incomplete")
	}
	parent, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatal("synthetic process parent was invalid")
	}
	return parent
}

func g4SafeHomeManifest(root string) ([]HomeEntry, error) {
	entries := []HomeEntry{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, HomeEntry{RelativePath: relative, Directory: entry.IsDir()})
		return nil
	})
	return entries, err
}

func assertG4SafeHomeContents(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		lower := strings.ToLower(filepath.Base(path))
		if lower == "auth.json" || strings.Contains(lower, "cookie") || strings.Contains(lower, "credential") {
			return errors.New("forbidden task-home entry")
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if g4ContainsForbiddenMaterial(raw) {
			return errors.New("forbidden task-home content")
		}
		return nil
	})
	if err != nil {
		t.Fatal("controlled task home contained forbidden material")
	}
}

func assertG4SafeDiagnostic(t *testing.T, raw []byte) {
	t.Helper()
	if g4ContainsForbiddenMaterial(raw) {
		t.Fatal("synthetic diagnostic contained forbidden material")
	}
}

func g4ContainsForbiddenMaterial(raw []byte) bool {
	lower := strings.ToLower(string(raw))
	for _, marker := range []string{
		strings.ToLower(g4SyntheticProviderValue),
		strings.ToLower(g4SyntheticCookieValue),
		strings.ToLower(g4SyntheticGatewayValue),
		"authorization: bearer ",
		"cookie:",
		"set-cookie:",
		"auth.json",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, endpoint := range g4DirectProviderFragments {
		if strings.Contains(lower, endpoint) {
			return true
		}
	}
	return false
}
