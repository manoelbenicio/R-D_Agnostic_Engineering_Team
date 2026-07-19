package runtimeenv

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestCredentiallessHomeContractPreservesLocalStateWithoutAuthCopy(t *testing.T) {
	contract := CredentiallessHomeContract(brain.CLICodex)
	if contract.ProviderAuthCopyAllowed || !contract.ControlledHomeRequired {
		t.Fatalf("credentialless home contract = %+v", contract)
	}
	want := map[PreservedState]bool{
		PreserveSandboxConfig: true, PreserveSkills: true, PreserveSessions: true, PreserveWorkspace: true,
	}
	for _, preserved := range contract.Preserve {
		delete(want, preserved)
	}
	if len(want) != 0 {
		t.Fatalf("credentialless home contract omitted preserved state: %v", want)
	}
}

func TestAssertPreLaunchAcceptsControlledCodexPlan(t *testing.T) {
	executionRoot, taskHome, codexHome := controlledTestDirectories(t)
	secret, _ := NewStableSecret(syntheticSecret)
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Inherited: []string{"PATH=/usr/bin"},
		Local:     map[string]string{"TASK_RUNTIME_MODE": "controlled"},
		Adapter: AdapterEnvironment{
			CLI: brain.CLICodex, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, CodexHome: codexHome, StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	config, err := NewCodexConfigContract("http://127.0.0.1:20128", brain.RouteModel("approved/codex-model"), testCorrelation())
	if err != nil {
		t.Fatalf("NewCodexConfigContract returned error: %v", err)
	}
	plan := LaunchPlan{
		Environment: environment, CodexConfig: &config, ExecutionRoot: executionRoot,
		TaskHome: []HomeEntry{{RelativePath: "codex-home/config.toml"}, {RelativePath: "codex-home/skills/review/SKILL.md"}},
	}
	if err := AssertPreLaunch(plan); err != nil {
		t.Fatalf("controlled Codex plan rejected: %v", err)
	}
}

func TestAssertPreLaunchAcceptsControlledClaudePlan(t *testing.T) {
	executionRoot, taskHome, _ := controlledTestDirectories(t)
	secret, _ := NewStableSecret(syntheticSecret)
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Inherited: []string{"PATH=/usr/bin"},
		Adapter: AdapterEnvironment{
			CLI: brain.CLIClaudeCode, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	if err := AssertPreLaunch(LaunchPlan{
		Environment: environment, ExecutionRoot: executionRoot,
		TaskHome: []HomeEntry{{RelativePath: "skills/review/SKILL.md"}},
	}); err != nil {
		t.Fatalf("controlled Claude plan rejected: %v", err)
	}
}

func TestAssertPreLaunchRejectsMissingRequiredProcessBasics(t *testing.T) {
	executionRoot, taskHome, _ := controlledTestDirectories(t)
	secret, _ := NewStableSecret(syntheticSecret)
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Inherited: nil,
		Adapter: AdapterEnvironment{
			CLI: brain.CLIClaudeCode, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	if err := AssertPreLaunch(LaunchPlan{Environment: environment, ExecutionRoot: executionRoot}); !errors.Is(err, ErrPreLaunchPolicy) {
		t.Fatalf("missing PATH assertion error = %v", err)
	}
}

func TestAssertPreLaunchRejectsProviderCredentialAndAuthPath(t *testing.T) {
	executionRoot, taskHome, _ := controlledTestDirectories(t)
	secret, _ := NewStableSecret(syntheticSecret)
	environment, _, err := BuildGatewayEnvironment(ComposeOptions{
		Adapter: AdapterEnvironment{
			CLI: brain.CLIClaudeCode, GatewayRoot: "http://127.0.0.1:20128",
			TaskHome: taskHome, StableSecret: secret,
		},
	})
	if err != nil {
		t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
	}
	environment.entries["OPENAI_API_KEY"] = environmentEntry{key: "OPENAI_API_KEY", value: "synthetic", origin: originCustom}
	if err := AssertPreLaunch(LaunchPlan{Environment: environment, ExecutionRoot: executionRoot}); !errors.Is(err, ErrPreLaunchPolicy) {
		t.Fatalf("provider credential assertion error = %v", err)
	}
	delete(environment.entries, "OPENAI_API_KEY")
	if err := AssertPreLaunch(LaunchPlan{
		Environment: environment, ExecutionRoot: executionRoot,
		TaskHome: []HomeEntry{{RelativePath: "codex-home/auth.json"}},
	}); !errors.Is(err, ErrPreLaunchPolicy) {
		t.Fatalf("auth path assertion error = %v", err)
	}
}

func TestAssertPreLaunchRejectsHomesOutsideOrEscapingExecutionRoot(t *testing.T) {
	secret, _ := NewStableSecret(syntheticSecret)
	tests := []struct {
		name         string
		cli          brain.CLIKind
		outsideTask  bool
		outsideCodex bool
		mutateKey    string
	}{
		{name: "task home outside root", cli: brain.CLIClaudeCode, outsideTask: true},
		{name: "task home traversal", cli: brain.CLIClaudeCode, mutateKey: "HOME"},
		{name: "Codex home outside root", cli: brain.CLICodex, outsideCodex: true},
		{name: "Codex home traversal", cli: brain.CLICodex, mutateKey: "CODEX_HOME"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executionRoot, taskHome, codexHome := controlledTestDirectories(t)
			if test.outsideTask {
				_, taskHome, _ = controlledTestDirectories(t)
			}
			if test.outsideCodex {
				_, _, codexHome = controlledTestDirectories(t)
			}
			environment, _, err := BuildGatewayEnvironment(ComposeOptions{
				Inherited: []string{"PATH=/usr/bin"},
				Adapter: AdapterEnvironment{
					CLI: test.cli, GatewayRoot: "http://127.0.0.1:20128",
					TaskHome: taskHome, CodexHome: codexHome, StableSecret: secret,
				},
			})
			if err != nil {
				t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
			}
			if test.mutateKey != "" {
				entry := environment.entries[test.mutateKey]
				entry.value += string(filepath.Separator) + ".." + string(filepath.Separator) + "escape"
				environment.entries[test.mutateKey] = entry
			}
			var codexConfig *CodexConfigContract
			if test.cli == brain.CLICodex {
				config, configErr := NewCodexConfigContract(
					"http://127.0.0.1:20128", brain.RouteModel("approved/codex-model"), testCorrelation(),
				)
				if configErr != nil {
					t.Fatalf("NewCodexConfigContract returned error: %v", configErr)
				}
				codexConfig = &config
			}
			if err := AssertPreLaunch(LaunchPlan{
				Environment: environment, CodexConfig: codexConfig, ExecutionRoot: executionRoot,
			}); !errors.Is(err, ErrPreLaunchPolicy) {
				t.Fatalf("AssertPreLaunch error = %v, want root-policy rejection", err)
			}
		})
	}
}

func TestAssertPreLaunchRejectsPhysicalHomeSubstitution(t *testing.T) {
	secret, _ := NewStableSecret(syntheticSecret)
	for _, cli := range []brain.CLIKind{brain.CLIClaudeCode, brain.CLICodex} {
		t.Run(string(cli), func(t *testing.T) {
			executionRoot, taskHome, codexHome := controlledTestDirectories(t)
			environment, _, err := BuildGatewayEnvironment(ComposeOptions{
				Inherited: []string{"PATH=/usr/bin"},
				Adapter: AdapterEnvironment{
					CLI: cli, GatewayRoot: "http://127.0.0.1:20128",
					TaskHome: taskHome, CodexHome: codexHome, StableSecret: secret,
				},
			})
			if err != nil {
				t.Fatalf("BuildGatewayEnvironment returned error: %v", err)
			}
			target := taskHome
			if cli == brain.CLICodex {
				target = codexHome
			}
			if err := os.Remove(target); err != nil {
				t.Fatalf("remove controlled synthetic directory: %v", err)
			}
			createTestDirectorySymlink(t, t.TempDir(), target)

			var codexConfig *CodexConfigContract
			if cli == brain.CLICodex {
				config, configErr := NewCodexConfigContract(
					"http://127.0.0.1:20128", brain.RouteModel("approved/codex-model"), testCorrelation(),
				)
				if configErr != nil {
					t.Fatalf("NewCodexConfigContract returned error: %v", configErr)
				}
				codexConfig = &config
			}
			if err := AssertPreLaunch(LaunchPlan{
				Environment: environment, CodexConfig: codexConfig, ExecutionRoot: executionRoot,
			}); !errors.Is(err, ErrPreLaunchPolicy) {
				t.Fatalf("AssertPreLaunch error = %v, want physical-path rejection", err)
			}
		})
	}
}
