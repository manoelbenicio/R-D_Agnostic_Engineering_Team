// Package runtimeenv builds credentialless child-process environments and
// controlled CLI configuration for gateway-required Agent Brain tasks.
//
// The package is deliberately disconnected from the active daemon path. It
// does not read secret files, inspect provider credential stores, copy auth
// state, or launch processes. The central integrator supplies an already
// loaded opaque OmniRoute secret and wires the resulting contracts later.
package runtimeenv

const (
	EvidenceMinimalInheritedEnv = "EV-G2C-01"
	EvidenceCustomEnvPolicy     = "EV-G2C-02"
	EvidenceCredentiallessHome  = "EV-G2C-03"
	EvidenceCodexContract       = "EV-G2C-04"
	EvidenceClaudeEnvironment   = "EV-G2C-05"
	EvidenceCompatibleStub      = "EV-G2C-06"
	EvidenceKimiStub            = "EV-G2C-07"
	EvidenceAntigravityStub     = "EV-G2C-08"
	EvidenceModelValidation     = "EV-G2C-09"
	EvidencePreLaunchAssertion  = "EV-G2C-10"
	EvidenceG4ProtocolPaths     = "EV-G4-02"
	EvidenceG4RuntimeIsolation  = "EV-G4-03"
	EvidenceG4Codex             = "EV-G4-COD"
	EvidenceG4Adapters          = "EV-G4-ADP"
	EvidenceG4NIM               = "EV-G4-NIM"
	EvidenceG4Antigravity       = "EV-G4-AGY"
)
