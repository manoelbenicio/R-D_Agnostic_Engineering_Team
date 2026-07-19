package observability

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDevelopment20ProfileIsDeterministicAndBounded(t *testing.T) {
	first := RunDevelopment20Profile()
	second := RunDevelopment20Profile()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("development profile is not deterministic")
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("development profile invalid: %v", err)
	}
	if first.Counts.Completed != 17 || first.Counts.Failed != 1 || first.Counts.Cancelled != 2 {
		t.Fatalf("unexpected terminal counts: %+v", first.Counts)
	}
	if first.Resources.PeakActive != 4 || first.Resources.Source != "deterministic-model-not-host-sampled" {
		t.Fatalf("unexpected modeled resources: %+v", first.Resources)
	}
	t.Logf("run=%s counts=%+v peak_queue=%d selection=%+v queue=%+v ttft=%+v request=%+v retries=%d fallbacks=%d resources=%+v",
		first.RunID, first.Counts, first.PeakQueue, first.SelectionLatency, first.QueueLatency,
		first.FirstOutputLatency, first.RequestLatency, first.Retries, first.Fallbacks, first.Resources)
}

func TestDevelopment20LifecycleAndFairnessReconcile(t *testing.T) {
	result := RunDevelopment20Profile()
	if result.IndependentCount != 16 || result.FairnessDeviationPct != 0 {
		t.Fatalf("unexpected independent-request fairness: %d/%d", result.IndependentCount, result.FairnessDeviationPct)
	}
	for _, slot := range result.SlotDistribution {
		if slot.IndependentRequests != 4 || slot.ExpectedRequests != 4 || slot.AbsoluteDeviation != 0 {
			t.Fatalf("unexpected slot distribution: %+v", slot)
		}
	}
	if result.Counts.Started != 19 || result.Counts.Admitted != result.Counts.Completed+result.Counts.Failed+result.Counts.Cancelled {
		t.Fatalf("lifecycle counters do not reconcile: %+v", result.Counts)
	}
}

func TestSyntheticEvidenceCannotPromoteAcceptance(t *testing.T) {
	record := EvidenceRecord{
		SchemaVersion: EvidenceSchemaVersion, EvidenceID: "EV-G4-CAP", Disposition: DispositionSupported,
		Scope: "development-synthetic", SyntheticOnly: true, AcceptanceClaim: true, ManifestID: "manifest-g4-phase1",
		Blockers: []string{"host measurements absent"},
	}
	if err := record.Validate(); err == nil || !strings.Contains(err.Error(), "cannot claim") {
		t.Fatalf("synthetic Supported evidence was not rejected: %v", err)
	}
}

func TestG4ResultSchemaAndCanonicalMarshalling(t *testing.T) {
	schema := DefaultG4ResultSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("result schema invalid: %v", err)
	}
	first, err := MarshalDevelopmentRunResult(RunDevelopment20Profile())
	if err != nil {
		t.Fatalf("marshal development result: %v", err)
	}
	second, err := MarshalDevelopmentRunResult(RunDevelopment20Profile())
	if err != nil {
		t.Fatalf("marshal repeated development result: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("canonical development result bytes differ")
	}
	digest, err := DevelopmentRunResultDigest(RunDevelopment20Profile())
	if err != nil {
		t.Fatalf("digest development result: %v", err)
	}
	if len(digest) != 64 {
		t.Fatalf("unexpected development result digest: %q", digest)
	}
	t.Logf("canonical_result_sha256=%s bytes=%d", digest, len(first))
}

func TestConsolidationRequiresBothIndependentArtifacts(t *testing.T) {
	gate := DefaultConsolidationGate()
	if err := gate.ValidateInputs(); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("missing independent artifacts were not rejected: %v", err)
	}
	if len(gate.RequiredAC) != 116 || len(gate.RequiredParity) != 44 {
		t.Fatalf("unexpected frozen catalog coverage: AC=%d parity=%d", len(gate.RequiredAC), len(gate.RequiredParity))
	}
}

func TestConsolidationAcceptsBothProvenancedCatalogInputs(t *testing.T) {
	gate := DefaultConsolidationGate()
	gate.GatewayArtifact.Present = true
	gate.GatewayArtifact.SyntheticOnly = true
	gate.GatewayArtifact.SHA256 = strings.Repeat("a", 64)
	gate.RuntimeArtifact.Present = true
	gate.RuntimeArtifact.SyntheticOnly = true
	gate.RuntimeArtifact.SHA256 = strings.Repeat("b", 64)
	if err := gate.ValidateInputs(); err != nil {
		t.Fatalf("complete provenanced consolidation inputs rejected: %v", err)
	}
}

func TestG4AcceptanceBlocksOnG3SecurityCorrectionsAndIndependentReview(t *testing.T) {
	gate := DefaultSecurityCorrectionGate()
	if err := gate.ValidateForG4Acceptance(); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("absent G3 correction artifacts did not block G4 acceptance: %v", err)
	}
	gate.CoreArtifactPresent = true
	gate.CoreArtifactSHA256 = strings.Repeat("a", 64)
	gate.AdapterArtifactPresent = true
	gate.AdapterArtifactSHA256 = strings.Repeat("b", 64)
	if err := gate.ValidateForG4Acceptance(); err == nil || !strings.Contains(err.Error(), "re-review") {
		t.Fatalf("missing independent re-review did not block G4 acceptance: %v", err)
	}
	gate.IndependentReviewAccepted = true
	if err := gate.ValidateForG4Acceptance(); err != nil {
		t.Fatalf("complete G3 correction gate rejected: %v", err)
	}
}

func TestSyntheticProvenanceIsContentOffAndDoesNotReadPaths(t *testing.T) {
	input := NewSyntheticArtifactProvenance("g4/development-result.json", []byte("safe aggregate"), "g4-development-runner")
	manifest := ProvenanceManifest{
		SchemaVersion:   EvidenceSchemaVersion,
		ManifestID:      "manifest-g4-phase1",
		GeneratedAt:     time.Date(2026, 7, 18, 3, 30, 0, 0, time.UTC),
		Generator:       "g4-evidence-automation",
		GeneratorDigest: strings.Repeat("a", 64),
		SyntheticOnly:   true,
		Inputs:          []ArtifactProvenance{input},
		Constraints:     []string{"content-off", "no live endpoint"},
		MissingInputs: []string{
			".planning/agent-brain-v3/evidence/g4-gateway-tests.md",
			".planning/agent-brain-v3/evidence/g4-runtime-isolation.md",
		},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid synthetic manifest rejected: %v", err)
	}
	if _, err := MarshalProvenanceManifest(manifest); err != nil {
		t.Fatalf("marshal valid synthetic manifest: %v", err)
	}
	if input.Bytes != int64(len("safe aggregate")) || !input.SyntheticOnly {
		t.Fatalf("unexpected provenance descriptor: %+v", input)
	}
}
