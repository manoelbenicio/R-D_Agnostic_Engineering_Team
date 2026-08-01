package credentialcatalog

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Provider identifies a native credential-home layout supported by the catalog.
type Provider string

const (
	ProviderAntigravity Provider = "antigravity"
	ProviderCodex       Provider = "codex"
	ProviderKiro        Provider = "kiro"
)

// ParseProvider canonicalizes a supported provider name.
func ParseProvider(value string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "agy", "antigravity":
		return ProviderAntigravity, nil
	case "codex":
		return ProviderCodex, nil
	case "kiro":
		return ProviderKiro, nil
	default:
		return "", fmt.Errorf("credential catalog: unsupported provider")
	}
}

// Layout describes paths relative to one immediate child of the controlled
// root. Its fields are private so callers cannot alter a validated layout.
type Layout struct {
	homeRelative     string
	artifactRelative string
}

func (l Layout) HomeRelative() string     { return l.homeRelative }
func (l Layout) ArtifactRelative() string { return l.artifactRelative }

// LayoutFor returns the provider-specific home and credential artifact paths.
// These paths identify artifacts by metadata only; the catalog never opens
// their contents.
func LayoutFor(provider Provider) (Layout, bool) {
	switch provider {
	case ProviderAntigravity:
		return Layout{
			homeRelative:     "home",
			artifactRelative: ".gemini/antigravity-cli/antigravity-oauth-token",
		}, true
	case ProviderCodex:
		return Layout{homeRelative: "codex", artifactRelative: "auth.json"}, true
	case ProviderKiro:
		return Layout{
			homeRelative:     "xdg-data",
			artifactRelative: "kiro-cli/data.sqlite3",
		}, true
	default:
		return Layout{}, false
	}
}

// State is the metadata-only catalog state of a discovered home.
type State string

const (
	StateHealthy     State = "healthy"
	StateQuarantined State = "quarantined"
	StateMissing     State = "missing"
	StateDraining    State = "draining"
	StateRetired     State = "retired"
	StateTombstoned  State = "tombstoned"
)

// AdmissionStatus represents the watermark and capacity admission status.
type AdmissionStatus string

const (
	AdmissionNormal   AdmissionStatus = "normal"
	AdmissionDegraded AdmissionStatus = "degraded"
)

// QuarantineReason is a stable, pathless reason code.
type QuarantineReason string

const (
	ReasonMetadataUnavailable        QuarantineReason = "metadata_unavailable"
	ReasonCandidateSymlink           QuarantineReason = "candidate_symlink"
	ReasonCandidateNotDirectory      QuarantineReason = "candidate_not_directory"
	ReasonLayoutMissing              QuarantineReason = "layout_missing"
	ReasonLayoutSymlink              QuarantineReason = "layout_symlink"
	ReasonLayoutInvalidType          QuarantineReason = "layout_invalid_type"
	ReasonPathEscape                 QuarantineReason = "path_escape"
	ReasonWrongOwner                 QuarantineReason = "wrong_owner"
	ReasonPermissionsTooOpen         QuarantineReason = "permissions_too_open"
	ReasonFilesystemIdentityConflict QuarantineReason = "filesystem_identity_conflict"
	ReasonTombstoned                 QuarantineReason = "tombstoned"
	ReasonRevalidationFailed         QuarantineReason = "revalidation_failed"
	ReasonWatermarkExceeded          QuarantineReason = "watermark_exceeded"
)

// Entry is an immutable, daemon-local catalog entry. Paths are intentionally
// available only through methods on this internal package type and must never
// be projected to product APIs, events, logs, or evidence.
type Entry struct {
	homeRef      string
	provider     Provider
	state        State
	homePath     string
	artifactPath string
	activeRefs   int
	discoveredAt time.Time
}

func (e Entry) HomeRef() string        { return e.homeRef }
func (e Entry) Provider() Provider     { return e.provider }
func (e Entry) State() State           { return e.state }
func (e Entry) HomePath() string       { return e.homePath }
func (e Entry) ArtifactPath() string   { return e.artifactPath }
func (e Entry) ActiveRefs() int        { return e.activeRefs }
func (e Entry) DiscoveredAt() time.Time { return e.discoveredAt }

// Quarantine records only an opaque candidate reference and a reason code.
// It deliberately contains no child name, raw path, account identity, or
// credential filename.
type Quarantine struct {
	candidateRef string
	provider     Provider
	reason       QuarantineReason
}

func (q Quarantine) CandidateRef() string     { return q.candidateRef }
func (q Quarantine) Provider() Provider       { return q.provider }
func (q Quarantine) State() State             { return StateQuarantined }
func (q Quarantine) Reason() QuarantineReason { return q.reason }

// Snapshot is a complete, immutable full-scan result. Slice fields are private
// and every accessor returns a copy, so a published generation cannot be
// changed by callers.
type Snapshot struct {
	generation            uint64
	capturedAt            time.Time
	ttl                   time.Duration
	provider              Provider
	admissionStatus       AdmissionStatus
	highWatermarkExceeded bool
	entries               []Entry
	quarantined           []Quarantine
	draining              []Entry
	tombstoned            []string
}

func (s Snapshot) Generation() uint64            { return s.generation }
func (s Snapshot) CapturedAt() time.Time         { return s.capturedAt }
func (s Snapshot) TTL() time.Duration            { return s.ttl }
func (s Snapshot) Provider() Provider            { return s.provider }
func (s Snapshot) AdmissionStatus() AdmissionStatus { return s.admissionStatus }
func (s Snapshot) HighWatermarkExceeded() bool   { return s.highWatermarkExceeded }

func (s Snapshot) IsExpired(now time.Time) bool {
	if s.ttl <= 0 {
		return false
	}
	return now.Sub(s.capturedAt) > s.ttl
}

func (s Snapshot) Entries() []Entry {
	return append([]Entry(nil), s.entries...)
}

func (s Snapshot) Quarantined() []Quarantine {
	return append([]Quarantine(nil), s.quarantined...)
}

func (s Snapshot) Draining() []Entry {
	return append([]Entry(nil), s.draining...)
}

func (s Snapshot) Tombstoned() []string {
	return append([]string(nil), s.tombstoned...)
}

func (s Snapshot) HealthyCount() int     { return len(s.entries) }
func (s Snapshot) QuarantinedCount() int { return len(s.quarantined) }
func (s Snapshot) DrainingCount() int    { return len(s.draining) }
func (s Snapshot) TombstonedCount() int  { return len(s.tombstoned) }

func cloneSnapshot(source Snapshot) Snapshot {
	clone := source
	clone.entries = source.Entries()
	clone.quarantined = source.Quarantined()
	clone.draining = source.Draining()
	clone.tombstoned = source.Tombstoned()
	return clone
}

func sortSnapshot(snapshot *Snapshot) {
	sort.Slice(snapshot.entries, func(i, j int) bool {
		return snapshot.entries[i].homeRef < snapshot.entries[j].homeRef
	})
	sort.Slice(snapshot.quarantined, func(i, j int) bool {
		if snapshot.quarantined[i].candidateRef != snapshot.quarantined[j].candidateRef {
			return snapshot.quarantined[i].candidateRef < snapshot.quarantined[j].candidateRef
		}
		return snapshot.quarantined[i].reason < snapshot.quarantined[j].reason
	})
	sort.Slice(snapshot.draining, func(i, j int) bool {
		return snapshot.draining[i].homeRef < snapshot.draining[j].homeRef
	})
	sort.Strings(snapshot.tombstoned)
}
