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

// Layout describes paths relative to one immediate child of the controlled root.
type Layout struct {
	homeRelative     string
	artifactRelative string
}

func (l Layout) HomeRelative() string     { return l.homeRelative }
func (l Layout) ArtifactRelative() string { return l.artifactRelative }

// LayoutFor returns the provider-specific home and credential artifact paths.
func LayoutFor(provider Provider) (Layout, bool) {
	switch provider {
	case ProviderAntigravity:
		return Layout{homeRelative: "home", artifactRelative: ".gemini/antigravity-cli/antigravity-oauth-token"}, true
	case ProviderCodex:
		return Layout{homeRelative: "codex", artifactRelative: "auth.json"}, true
	case ProviderKiro:
		return Layout{homeRelative: "xdg-data", artifactRelative: "kiro-cli/data.sqlite3"}, true
	default:
		return Layout{}, false
	}
}

// State uses the credential_home_catalog_entry vocabulary reserved by C2.
type State string

const (
	StateCandidate   State = "candidate"
	StateHealthy     State = "healthy"
	StateDegraded    State = "degraded"
	StateQuarantined State = "quarantined"
	StateMissing     State = "missing"
	StateDraining    State = "draining"
	StateRetired     State = "retired"
)

func validState(state State) bool {
	switch state {
	case StateCandidate, StateHealthy, StateDegraded, StateQuarantined, StateMissing, StateDraining, StateRetired:
		return true
	default:
		return false
	}
}

// Watermark uses the credential_home_catalog vocabulary reserved by C2.
type Watermark string

const (
	WatermarkNormal   Watermark = "normal"
	WatermarkLow      Watermark = "low"
	WatermarkHigh     Watermark = "high"
	WatermarkCritical Watermark = "critical"
)

func validWatermark(watermark Watermark) bool {
	switch watermark {
	case WatermarkNormal, WatermarkLow, WatermarkHigh, WatermarkCritical:
		return true
	default:
		return false
	}
}

// ScanKind records why a complete generation was produced.
type ScanKind string

const (
	ScanStartup        ScanKind = "startup"
	ScanPeriodic       ScanKind = "periodic"
	ScanHintLoss       ScanKind = "hint_loss"
	ScanOverflow       ScanKind = "overflow"
	ScanWatcherRestart ScanKind = "watcher_restart"
	ScanRequested      ScanKind = "requested"
)

func validScanKind(kind ScanKind) bool {
	switch kind {
	case ScanStartup, ScanPeriodic, ScanHintLoss, ScanOverflow, ScanWatcherRestart, ScanRequested:
		return true
	default:
		return false
	}
}

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
	ReasonNameTombstone              QuarantineReason = "name_tombstone"
	ReasonRevalidationFailed         QuarantineReason = "revalidation_failed"
)

// Entry is immutable catalog metadata. Paths remain daemon-local and are never
// included in durable generation records.
type Entry struct {
	homeRef           string
	nameRef           string
	provider          Provider
	state             State
	reason            QuarantineReason
	homePath          string
	artifactPath      string
	approved          bool
	activeRefs        int
	discoveredAt      time.Time
	firstSeenAt       time.Time
	lastSeenAt        time.Time
	lastFullScanAt    time.Time
	healthWatermark   *time.Time
	missingWatermark  *time.Time
	ttl               time.Duration
	retentionDeadline time.Time
}

func (e Entry) HomeRef() string                     { return e.homeRef }
func (e Entry) NameRef() string                     { return e.nameRef }
func (e Entry) Provider() Provider                  { return e.provider }
func (e Entry) State() State                        { return e.state }
func (e Entry) Reason() QuarantineReason            { return e.reason }
func (e Entry) HomePath() string                    { return e.homePath }
func (e Entry) ArtifactPath() string                { return e.artifactPath }
func (e Entry) Approved() bool                      { return e.approved }
func (e Entry) ActiveRefs() int                     { return e.activeRefs }
func (e Entry) DiscoveredAt() time.Time             { return e.discoveredAt }
func (e Entry) FirstSeenAt() time.Time              { return e.firstSeenAt }
func (e Entry) LastSeenAt() time.Time               { return e.lastSeenAt }
func (e Entry) LastFullScanAt() time.Time           { return e.lastFullScanAt }
func (e Entry) TTL() time.Duration                  { return e.ttl }
func (e Entry) RetentionDeadline() time.Time        { return e.retentionDeadline }
func (e Entry) HealthWatermark() (time.Time, bool)  { return copiedTime(e.healthWatermark) }
func (e Entry) MissingWatermark() (time.Time, bool) { return copiedTime(e.missingWatermark) }

// Quarantine contains only an opaque reference and a reason code.
type Quarantine struct {
	candidateRef string
	provider     Provider
	reason       QuarantineReason
}

func (q Quarantine) CandidateRef() string     { return q.candidateRef }
func (q Quarantine) Provider() Provider       { return q.provider }
func (q Quarantine) State() State             { return StateQuarantined }
func (q Quarantine) Reason() QuarantineReason { return q.reason }

// Tombstone is the public view of a durable retired-entry marker. The store
// encodes it with StateRetired and a tombstone reason, which the C2 schema can
// persist without introducing a non-schema lifecycle state.
type Tombstone struct {
	homeRef           string
	retiredAt         time.Time
	retentionDeadline time.Time
}

func (t Tombstone) HomeRef() string              { return t.homeRef }
func (t Tombstone) State() State                 { return StateRetired }
func (t Tombstone) Reason() QuarantineReason     { return ReasonTombstoned }
func (t Tombstone) RetiredAt() time.Time         { return t.retiredAt }
func (t Tombstone) RetentionDeadline() time.Time { return t.retentionDeadline }

// Snapshot is a complete immutable full-scan result.
type Snapshot struct {
	workspaceID        string
	daemonID           string
	catalogID          string
	previousGeneration uint64
	generation         uint64
	scanKind           ScanKind
	digest             string
	startedAt          time.Time
	capturedAt         time.Time
	publishedAt        time.Time
	ttl                time.Duration
	provider           Provider
	watermark          Watermark
	entries            []Entry
	quarantined        []Quarantine
	draining           []Entry
	missing            []Entry
	retired            []Entry
	tombstones         []Tombstone
}

func (s Snapshot) WorkspaceID() string        { return s.workspaceID }
func (s Snapshot) DaemonID() string           { return s.daemonID }
func (s Snapshot) CatalogID() string          { return s.catalogID }
func (s Snapshot) PreviousGeneration() uint64 { return s.previousGeneration }
func (s Snapshot) Generation() uint64         { return s.generation }
func (s Snapshot) ScanKind() ScanKind         { return s.scanKind }
func (s Snapshot) Digest() string             { return s.digest }
func (s Snapshot) StartedAt() time.Time       { return s.startedAt }
func (s Snapshot) CapturedAt() time.Time      { return s.capturedAt }
func (s Snapshot) PublishedAt() time.Time     { return s.publishedAt }
func (s Snapshot) TTL() time.Duration         { return s.ttl }
func (s Snapshot) Provider() Provider         { return s.provider }
func (s Snapshot) Watermark() Watermark       { return s.watermark }
func (s Snapshot) HighWatermarkExceeded() bool {
	return s.watermark == WatermarkHigh || s.watermark == WatermarkCritical
}

func (s Snapshot) IsExpired(now time.Time) bool {
	return s.ttl > 0 && now.After(s.capturedAt.Add(s.ttl))
}

func (s Snapshot) Entries() []Entry          { return append([]Entry(nil), s.entries...) }
func (s Snapshot) Quarantined() []Quarantine { return append([]Quarantine(nil), s.quarantined...) }
func (s Snapshot) Draining() []Entry         { return append([]Entry(nil), s.draining...) }
func (s Snapshot) Missing() []Entry          { return append([]Entry(nil), s.missing...) }
func (s Snapshot) Retired() []Entry          { return append([]Entry(nil), s.retired...) }
func (s Snapshot) Tombstones() []Tombstone   { return append([]Tombstone(nil), s.tombstones...) }
func (s Snapshot) Tombstoned() []string {
	refs := make([]string, 0, len(s.tombstones))
	for _, tombstone := range s.tombstones {
		refs = append(refs, tombstone.homeRef)
	}
	return refs
}

func (s Snapshot) HealthyCount() int     { return len(s.entries) }
func (s Snapshot) QuarantinedCount() int { return len(s.quarantined) }
func (s Snapshot) DrainingCount() int    { return len(s.draining) }
func (s Snapshot) MissingCount() int     { return len(s.missing) }
func (s Snapshot) RetiredCount() int     { return len(s.retired) }
func (s Snapshot) TombstonedCount() int  { return len(s.tombstones) }

func cloneSnapshot(source Snapshot) Snapshot {
	clone := source
	clone.entries = source.Entries()
	clone.quarantined = source.Quarantined()
	clone.draining = source.Draining()
	clone.missing = source.Missing()
	clone.retired = source.Retired()
	clone.tombstones = source.Tombstones()
	return clone
}

func sortSnapshot(snapshot *Snapshot) {
	sort.Slice(snapshot.entries, func(i, j int) bool { return snapshot.entries[i].homeRef < snapshot.entries[j].homeRef })
	sort.Slice(snapshot.quarantined, func(i, j int) bool {
		if snapshot.quarantined[i].candidateRef != snapshot.quarantined[j].candidateRef {
			return snapshot.quarantined[i].candidateRef < snapshot.quarantined[j].candidateRef
		}
		return snapshot.quarantined[i].reason < snapshot.quarantined[j].reason
	})
	sort.Slice(snapshot.draining, func(i, j int) bool { return snapshot.draining[i].homeRef < snapshot.draining[j].homeRef })
	sort.Slice(snapshot.missing, func(i, j int) bool { return snapshot.missing[i].homeRef < snapshot.missing[j].homeRef })
	sort.Slice(snapshot.retired, func(i, j int) bool { return snapshot.retired[i].homeRef < snapshot.retired[j].homeRef })
	sort.Slice(snapshot.tombstones, func(i, j int) bool { return snapshot.tombstones[i].homeRef < snapshot.tombstones[j].homeRef })
}

func copiedTime(value *time.Time) (time.Time, bool) {
	if value == nil {
		return time.Time{}, false
	}
	return *value, true
}

func timePointer(value time.Time) *time.Time {
	copy := value
	return &copy
}
