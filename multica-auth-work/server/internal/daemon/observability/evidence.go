package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

const EvidenceSchemaVersion = "agent-brain.g4-evidence.v1"

type EvidenceDisposition string

const (
	DispositionSupported    EvidenceDisposition = "Supported"
	DispositionPartial      EvidenceDisposition = "Partial"
	DispositionNotSupported EvidenceDisposition = "Not-supported"
)

type ArtifactProvenance struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Bytes         int64  `json:"bytes"`
	SyntheticOnly bool   `json:"synthetic_only"`
	Producer      string `json:"producer"`
}

// NewSyntheticArtifactProvenance hashes only bytes already produced by the
// synthetic evidence pipeline. It never opens a path and is deliberately not
// a general-purpose filesystem collector.
func NewSyntheticArtifactProvenance(path string, data []byte, producer string) ArtifactProvenance {
	digest := sha256.Sum256(data)
	return ArtifactProvenance{
		Path:          path,
		SHA256:        hex.EncodeToString(digest[:]),
		Bytes:         int64(len(data)),
		SyntheticOnly: true,
		Producer:      producer,
	}
}

type ProvenanceManifest struct {
	SchemaVersion   string               `json:"schema_version"`
	ManifestID      string               `json:"manifest_id"`
	GeneratedAt     time.Time            `json:"generated_at"`
	Generator       string               `json:"generator"`
	GeneratorDigest string               `json:"generator_digest"`
	SyntheticOnly   bool                 `json:"synthetic_only"`
	ContentCapture  bool                 `json:"content_capture"`
	Inputs          []ArtifactProvenance `json:"inputs"`
	Constraints     []string             `json:"constraints"`
	MissingInputs   []string             `json:"missing_inputs,omitempty"`
}

func (m ProvenanceManifest) Validate() error {
	if m.SchemaVersion != EvidenceSchemaVersion || !safeID(m.ManifestID, 128) || m.GeneratedAt.IsZero() {
		return fmt.Errorf("invalid evidence manifest identity")
	}
	if !safeID(m.Generator, 128) || !validDigest(m.GeneratorDigest) {
		return fmt.Errorf("generator identity and digest are required")
	}
	if !m.SyntheticOnly || m.ContentCapture || len(m.Constraints) == 0 {
		return fmt.Errorf("phase-one provenance must be synthetic-only and content-off")
	}
	seen := map[string]bool{}
	for _, input := range m.Inputs {
		if input.Path == "" || strings.HasPrefix(input.Path, "/") || strings.Contains(input.Path, "..") ||
			!validDigest(input.SHA256) || input.Bytes < 0 || !input.SyntheticOnly || !safeID(input.Producer, 128) || seen[input.Path] {
			return fmt.Errorf("invalid or duplicate synthetic provenance input")
		}
		seen[input.Path] = true
	}
	for _, path := range m.MissingInputs {
		if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "..") || seen[path] {
			return fmt.Errorf("invalid missing-input reference")
		}
	}
	return nil
}

func MarshalProvenanceManifest(manifest ProvenanceManifest) ([]byte, error) {
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(manifest, "", "  ")
}

type ResultSchema struct {
	SchemaVersion        string   `json:"schema_version"`
	ResultType           string   `json:"result_type"`
	RequiredFields       []string `json:"required_fields"`
	ProhibitedFields     []string `json:"prohibited_fields"`
	ResourceSemantics    string   `json:"resource_semantics"`
	DispositionSemantics string   `json:"disposition_semantics"`
}

func DefaultG4ResultSchema() ResultSchema {
	return ResultSchema{
		SchemaVersion: EvidenceSchemaVersion,
		ResultType:    "g4-development-capacity-result",
		RequiredFields: []string{
			"run_id", "synthetic_only", "live_endpoint_used", "capacity_tier_enabled", "acceptance_claim",
			"counts", "latency_percentiles", "peak_queue", "slot_distribution", "fairness_deviation_percent",
			"modeled_resources", "retries", "fallbacks", "failure_cases_covered", "blockers",
		},
		ProhibitedFields: []string{
			"authorization", "credential", "secret", "token_value", "cookie", "account_identity",
			"prompt", "completion", "message_content", "tool_payload", "repository_content", "reasoning_content",
		},
		ResourceSemantics:    "deterministic-modeled-not-host-sampled",
		DispositionSemantics: "synthetic-results-cannot-produce-Supported-or-enable-a-capacity-tier",
	}
}

func (s ResultSchema) Validate() error {
	if s.SchemaVersion != EvidenceSchemaVersion || !safeID(s.ResultType, 128) || len(s.RequiredFields) == 0 || len(s.ProhibitedFields) == 0 {
		return fmt.Errorf("incomplete G4 result schema")
	}
	if s.ResourceSemantics != "deterministic-modeled-not-host-sampled" ||
		s.DispositionSemantics != "synthetic-results-cannot-produce-Supported-or-enable-a-capacity-tier" {
		return fmt.Errorf("unsafe G4 result semantics")
	}
	for _, fields := range [][]string{s.RequiredFields, s.ProhibitedFields} {
		seen := map[string]bool{}
		for _, field := range fields {
			if !safeID(field, 96) || seen[field] {
				return fmt.Errorf("invalid or duplicate result-schema field")
			}
			seen[field] = true
		}
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type EvidenceRecord struct {
	SchemaVersion   string              `json:"schema_version"`
	EvidenceID      string              `json:"evidence_id"`
	Disposition     EvidenceDisposition `json:"disposition"`
	Scope           string              `json:"scope"`
	SyntheticOnly   bool                `json:"synthetic_only"`
	AcceptanceClaim bool                `json:"acceptance_claim"`
	ManifestID      string              `json:"manifest_id"`
	ChecklistIDs    []string            `json:"checklist_ids,omitempty"`
	ParityIDs       []string            `json:"parity_ids,omitempty"`
	Measurements    []string            `json:"measurements,omitempty"`
	Blockers        []string            `json:"blockers,omitempty"`
	Limitations     []string            `json:"limitations,omitempty"`
}

func (r EvidenceRecord) Validate() error {
	if r.SchemaVersion != EvidenceSchemaVersion || !validG4EvidenceID(r.EvidenceID) ||
		r.Scope == "" || !safeID(r.ManifestID, 128) {
		return fmt.Errorf("invalid G4 evidence record identity")
	}
	if r.Disposition != DispositionSupported && r.Disposition != DispositionPartial && r.Disposition != DispositionNotSupported {
		return fmt.Errorf("invalid evidence disposition")
	}
	if r.SyntheticOnly && r.AcceptanceClaim {
		return fmt.Errorf("synthetic evidence cannot claim protocol, provider, or capacity acceptance")
	}
	if r.Disposition == DispositionSupported && (!r.AcceptanceClaim || r.SyntheticOnly) {
		return fmt.Errorf("Supported requires accepted non-synthetic evidence")
	}
	if !r.AcceptanceClaim && len(r.Blockers) == 0 {
		return fmt.Errorf("non-acceptance evidence must retain explicit blockers")
	}
	if err := validateCoverageIDs(r.ChecklistIDs, "AC-"); err != nil {
		return err
	}
	for _, id := range r.ParityIDs {
		if !validParityID(id) {
			return fmt.Errorf("invalid parity ID %q", id)
		}
	}
	return nil
}

func validG4EvidenceID(value string) bool {
	switch value {
	case "EV-G4-01", "EV-G4-02", "EV-G4-03", "EV-G4-04", "EV-G4-05", "EV-G4-06", "EV-G4-07", "EV-G4-08", "EV-G4-CAP":
		return true
	default:
		return false
	}
}

func validateCoverageIDs(ids []string, prefix string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if !strings.HasPrefix(id, prefix) || !safeID(id, 32) || seen[id] {
			return fmt.Errorf("invalid or duplicate checklist ID %q", id)
		}
		seen[id] = true
	}
	return nil
}

func validParityID(value string) bool {
	if len(value) != 3 && len(value) != 4 {
		return false
	}
	prefixLen := 1
	if strings.HasPrefix(value, "SC") {
		prefixLen = 2
	} else if !strings.HasPrefix(value, "P") {
		return false
	}
	digits := value[prefixLen:]
	if len(digits) != 2 || digits[0] < '0' || digits[0] > '9' || digits[1] < '0' || digits[1] > '9' {
		return false
	}
	number, err := strconv.Atoi(digits)
	if err != nil {
		return false
	}
	if prefixLen == 2 {
		return number >= 1 && number <= 10
	}
	return number >= 1 && number <= 34
}

type ConsolidationInput struct {
	Path          string
	Present       bool
	SyntheticOnly bool
	SHA256        string
}

type ConsolidationGate struct {
	GatewayArtifact ConsolidationInput
	RuntimeArtifact ConsolidationInput
	RequiredAC      []string
	RequiredParity  []string
}

type SecurityCorrectionGate struct {
	CoreArtifactPath          string
	CoreArtifactPresent       bool
	CoreArtifactSHA256        string
	AdapterArtifactPath       string
	AdapterArtifactPresent    bool
	AdapterArtifactSHA256     string
	IndependentReviewAccepted bool
}

func DefaultSecurityCorrectionGate() SecurityCorrectionGate {
	return SecurityCorrectionGate{
		CoreArtifactPath:    ".planning/agent-brain-v3/evidence/g3-security-corrections.md",
		AdapterArtifactPath: ".planning/agent-brain-v3/evidence/g3-security-corrections-adapters.md",
	}
}

func (g SecurityCorrectionGate) ValidateForG4Acceptance() error {
	inputs := []struct {
		path    string
		present bool
		digest  string
	}{
		{g.CoreArtifactPath, g.CoreArtifactPresent, g.CoreArtifactSHA256},
		{g.AdapterArtifactPath, g.AdapterArtifactPresent, g.AdapterArtifactSHA256},
	}
	for _, input := range inputs {
		if input.path == "" || !input.present {
			return fmt.Errorf("G3 security correction artifact is absent: %s", input.path)
		}
		if !validDigest(input.digest) {
			return fmt.Errorf("G3 security correction artifact lacks SHA-256 provenance: %s", input.path)
		}
	}
	if !g.IndependentReviewAccepted {
		return fmt.Errorf("G3 security corrections lack independent pB re-review acceptance")
	}
	return nil
}

func DefaultConsolidationGate() ConsolidationGate {
	return ConsolidationGate{
		GatewayArtifact: ConsolidationInput{Path: ".planning/agent-brain-v3/evidence/g4-gateway-tests.md"},
		RuntimeArtifact: ConsolidationInput{Path: ".planning/agent-brain-v3/evidence/g4-runtime-isolation.md"},
		RequiredAC:      requiredChecklistIDs(),
		RequiredParity:  requiredParityIDs(),
	}
}

func (g ConsolidationGate) ValidateInputs() error {
	for _, input := range []ConsolidationInput{g.GatewayArtifact, g.RuntimeArtifact} {
		if input.Path == "" || !input.Present {
			return fmt.Errorf("required consolidation input is absent: %s", input.Path)
		}
		if !input.SyntheticOnly {
			return fmt.Errorf("phase-one consolidation input is not classified synthetic-only: %s", input.Path)
		}
		if !validDigest(input.SHA256) {
			return fmt.Errorf("required consolidation input lacks SHA-256 provenance: %s", input.Path)
		}
	}
	if !slices.Equal(g.RequiredAC, requiredChecklistIDs()) || !slices.Equal(g.RequiredParity, requiredParityIDs()) {
		return fmt.Errorf("consolidation coverage does not match the frozen G1 catalogs")
	}
	return nil
}

func requiredChecklistIDs() []string {
	sections := []struct {
		prefix string
		count  int
	}{
		{"AC-1.", 5}, {"AC-2.1.", 4}, {"AC-2.2.", 8}, {"AC-2.3.", 8}, {"AC-2.4.", 4}, {"AC-2.5.", 5},
		{"AC-3.", 10}, {"AC-4.", 11}, {"AC-5.", 11}, {"AC-6.", 6}, {"AC-7.", 8}, {"AC-8.", 7},
		{"AC-9.", 8}, {"AC-10.", 12}, {"AC-12.", 9},
	}
	var result []string
	for _, section := range sections {
		for index := 1; index <= section.count; index++ {
			result = append(result, fmt.Sprintf("%s%d", section.prefix, index))
		}
	}
	return result
}

func requiredParityIDs() []string {
	result := make([]string, 0, 44)
	for index := 1; index <= 34; index++ {
		result = append(result, fmt.Sprintf("P%02d", index))
	}
	for index := 1; index <= 10; index++ {
		result = append(result, fmt.Sprintf("SC%02d", index))
	}
	return result
}
