// Package brain defines the frozen, brand-neutral control-plane contracts for
// Agent Brain. It intentionally has no dependency on the active daemon path.
// Runtime wiring is a later, sole-integrator change.
package brain

// ContractVersion is the first frozen Agent Brain contract version.
const ContractVersion = "agent-brain.v1"
