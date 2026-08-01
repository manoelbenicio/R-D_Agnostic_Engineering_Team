package credentialcatalog

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrMissingProductionAdapter = errors.New("credential catalog: production generation adapter is required")
	ErrGenerationConflict       = errors.New("credential catalog: generation append conflict")
	ErrInvalidGeneration        = errors.New("credential catalog: invalid generation")
)

// CatalogIdentity is the durable tenant and producer key for one catalog.
type CatalogIdentity struct {
	WorkspaceID string
	DaemonID    string
	CatalogID   string
}

func (i CatalogIdentity) valid() bool {
	return strings.TrimSpace(i.WorkspaceID) != "" && strings.TrimSpace(i.DaemonID) != "" && strings.TrimSpace(i.CatalogID) != ""
}

func (i CatalogIdentity) key() string {
	return i.WorkspaceID + "\x00" + i.DaemonID + "\x00" + i.CatalogID
}

// CatalogEntryRecord mirrors the append-only C2 generation-entry projection.
// Tombstones are explicit retired entries with ReasonTombstoned or
// ReasonNameTombstone, never an out-of-vocabulary lifecycle state.
type CatalogEntryRecord struct {
	HomeRef           string
	NameRef           string
	Provider          Provider
	Approved          bool
	State             State
	ReasonCode        QuarantineReason
	ActiveRefs        int
	FirstSeenAt       time.Time
	LastSeenAt        time.Time
	LastFullScanAt    time.Time
	HealthWatermark   *time.Time
	MissingWatermark  *time.Time
	TTL               time.Duration
	RetentionDeadline time.Time
}

func (e CatalogEntryRecord) IsTombstone() bool {
	return e.State == StateRetired && (e.ReasonCode == ReasonTombstoned || e.ReasonCode == ReasonNameTombstone)
}

// LifecycleRecord is the durable current-state projection written atomically
// with its immutable catalog generation. HomeRef is always the canonical UUID
// used by catalog entries and task snapshots. NameRef is a separate pathless
// name_<sha256> marker used only to prevent child-name reuse.
type LifecycleRecord struct {
	HomeRef           string
	NameRef           string
	Provider          Provider
	State             State
	ReasonCode        QuarantineReason
	ActiveRefs        int
	Generation        uint64
	UpdatedAt         time.Time
	RetentionDeadline time.Time
}

// CatalogGeneration is one complete immutable catalog generation.
type CatalogGeneration struct {
	CatalogIdentity
	PreviousGeneration uint64
	Generation         uint64
	ScanKind           ScanKind
	Watermark          Watermark
	CatalogDigest      string
	StartedAt          time.Time
	PublishedAt        time.Time
	Entries            []CatalogEntryRecord
	Lifecycle          []LifecycleRecord
}

// LifecycleStore is deliberately generation-oriented. Implementations must
// atomically append the generation and every entry, then make that generation
// current. Previously appended generations and entries are immutable.
type LifecycleStore interface {
	LoadLatestGeneration(context.Context, CatalogIdentity) (*CatalogGeneration, error)
	AppendGeneration(context.Context, CatalogGeneration) error
}

// CatalogGenerationAdapter is the dependency implemented by production C2 SQL
// wiring. Keeping it injectable lets this package remain independent of sqlc.
type CatalogGenerationAdapter interface {
	LoadLatestGeneration(context.Context, CatalogIdentity) (*CatalogGeneration, error)
	AppendGeneration(context.Context, CatalogGeneration) error
}

// ProductionLifecycleStore is a fail-closed forwarding adapter. There is no
// file-backed production fallback.
type ProductionLifecycleStore struct {
	adapter CatalogGenerationAdapter
}

func NewProductionLifecycleStore(adapter CatalogGenerationAdapter) (*ProductionLifecycleStore, error) {
	if adapter == nil {
		return nil, ErrMissingProductionAdapter
	}
	return &ProductionLifecycleStore{adapter: adapter}, nil
}

func (s *ProductionLifecycleStore) LoadLatestGeneration(ctx context.Context, identity CatalogIdentity) (*CatalogGeneration, error) {
	if s == nil || s.adapter == nil {
		return nil, ErrMissingProductionAdapter
	}
	return s.adapter.LoadLatestGeneration(ctx, identity)
}

func (s *ProductionLifecycleStore) AppendGeneration(ctx context.Context, generation CatalogGeneration) error {
	if s == nil || s.adapter == nil {
		return ErrMissingProductionAdapter
	}
	return s.adapter.AppendGeneration(ctx, generation)
}

// MemoryLifecycleStore is an append-only test adapter.
type MemoryLifecycleStore struct {
	mu               sync.RWMutex
	generations      map[string][]CatalogGeneration
	injectFailAppend bool
	injectFailLoad   bool
}

func NewMemoryLifecycleStore() *MemoryLifecycleStore {
	return &MemoryLifecycleStore{generations: make(map[string][]CatalogGeneration)}
}

func (m *MemoryLifecycleStore) SetInjectFailAppend(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.injectFailAppend = fail
}

// SetInjectFailSave is retained as test terminology for persist-before-publish checks.
func (m *MemoryLifecycleStore) SetInjectFailSave(fail bool) { m.SetInjectFailAppend(fail) }

func (m *MemoryLifecycleStore) SetInjectFailLoad(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.injectFailLoad = fail
}

func (m *MemoryLifecycleStore) LoadLatestGeneration(ctx context.Context, identity CatalogIdentity) (*CatalogGeneration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.injectFailLoad {
		return nil, fmt.Errorf("injected store load failure")
	}
	generations := m.generations[identity.key()]
	if len(generations) == 0 {
		return nil, nil
	}
	clone := cloneGeneration(generations[len(generations)-1])
	return &clone, nil
}

func (m *MemoryLifecycleStore) AppendGeneration(ctx context.Context, generation CatalogGeneration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateGeneration(generation); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.injectFailAppend {
		return fmt.Errorf("injected store append failure")
	}
	key := generation.CatalogIdentity.key()
	existing := m.generations[key]
	var latest uint64
	if len(existing) > 0 {
		latest = existing[len(existing)-1].Generation
	}
	if generation.PreviousGeneration != latest || generation.Generation != latest+1 {
		return ErrGenerationConflict
	}
	m.generations[key] = append(existing, cloneGeneration(generation))
	return nil
}

// Generations returns immutable copies for append-only contract tests.
func (m *MemoryLifecycleStore) Generations(identity CatalogIdentity) []CatalogGeneration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	existing := m.generations[identity.key()]
	result := make([]CatalogGeneration, len(existing))
	for i := range existing {
		result[i] = cloneGeneration(existing[i])
	}
	return result
}

func validateGeneration(generation CatalogGeneration) error {
	if !generation.CatalogIdentity.valid() || generation.Generation == 0 ||
		generation.Generation != generation.PreviousGeneration+1 ||
		!validScanKind(generation.ScanKind) || !validWatermark(generation.Watermark) ||
		generation.StartedAt.IsZero() || generation.PublishedAt.Before(generation.StartedAt) ||
		!validRaw64Digest(generation.CatalogDigest) {
		return ErrInvalidGeneration
	}
	if ComputeCatalogDigest(generation) != generation.CatalogDigest {
		return fmt.Errorf("%w: digest mismatch", ErrInvalidGeneration)
	}
	seen := make(map[string]struct{}, len(generation.Entries))
	for _, entry := range generation.Entries {
		if !validUUIDRef(entry.HomeRef) {
			return fmt.Errorf("%w: invalid entry home ref %q", ErrInvalidGeneration, entry.HomeRef)
		}
		if entry.NameRef != "" && !validNameRef(entry.NameRef) {
			return fmt.Errorf("%w: invalid entry name ref %q", ErrInvalidGeneration, entry.NameRef)
		}
		if strings.TrimSpace(string(entry.Provider)) == "" || entry.ActiveRefs < 0 ||
			!validState(entry.State) || entry.TTL <= 0 || entry.FirstSeenAt.IsZero() ||
			entry.LastSeenAt.Before(entry.FirstSeenAt) || entry.LastFullScanAt.Before(entry.LastSeenAt) ||
			!entry.RetentionDeadline.After(entry.FirstSeenAt) {
			return fmt.Errorf("%w: invalid entry metadata for %q", ErrInvalidGeneration, entry.HomeRef)
		}
		if _, exists := seen[entry.HomeRef]; exists {
			return fmt.Errorf("%w: duplicate home ref", ErrInvalidGeneration)
		}
		seen[entry.HomeRef] = struct{}{}
		if (entry.State == StateMissing || entry.State == StateDraining || entry.State == StateRetired) && entry.MissingWatermark == nil {
			return fmt.Errorf("%w: missing lifecycle watermark", ErrInvalidGeneration)
		}
		if entry.State != StateHealthy && entry.State != StateDraining && entry.ActiveRefs != 0 {
			return fmt.Errorf("%w: active refs on terminal entry", ErrInvalidGeneration)
		}
	}
	lifecycleSeen := make(map[string]struct{}, len(generation.Lifecycle))
	for _, record := range generation.Lifecycle {
		if !validUUIDRef(record.HomeRef) {
			return fmt.Errorf("%w: invalid lifecycle home ref %q", ErrInvalidGeneration, record.HomeRef)
		}
		if record.NameRef != "" && !validNameRef(record.NameRef) {
			return fmt.Errorf("%w: invalid lifecycle name ref %q", ErrInvalidGeneration, record.NameRef)
		}
		if record.Provider == "" || record.ActiveRefs < 0 || record.Generation != generation.Generation ||
			record.UpdatedAt.IsZero() || !record.RetentionDeadline.After(record.UpdatedAt) {
			return fmt.Errorf("%w: invalid lifecycle metadata for %q", ErrInvalidGeneration, record.HomeRef)
		}
		switch record.State {
		case StateMissing, StateDraining, StateRetired:
		default:
			return ErrInvalidGeneration
		}
		if record.State != StateDraining && record.ActiveRefs != 0 {
			return fmt.Errorf("%w: active refs on terminal lifecycle record", ErrInvalidGeneration)
		}
		if record.ReasonCode == ReasonTombstoned && record.NameRef == "" {
			return fmt.Errorf("%w: tombstone missing name ref", ErrInvalidGeneration)
		}
		if _, exists := lifecycleSeen[record.HomeRef]; exists {
			return fmt.Errorf("%w: duplicate lifecycle home ref", ErrInvalidGeneration)
		}
		lifecycleSeen[record.HomeRef] = struct{}{}
	}
	return nil
}

func validUUIDRef(value string) bool {
	if len(value) != 36 || value != strings.ToLower(value) || value[8] != '-' || value[13] != '-' ||
		value[18] != '-' || value[23] != '-' || value[14] != '5' {
		return false
	}
	compact := strings.ReplaceAll(value, "-", "")
	decoded, err := hex.DecodeString(compact)
	return err == nil && len(decoded) == 16 && decoded[8]&0xc0 == 0x80
}

func validNameRef(value string) bool {
	if len(value) != len("name_")+43 || !strings.HasPrefix(value, "name_") {
		return false
	}
	for _, character := range value[len("name_"):] {
		if (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func validRaw64Digest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// ComputeCatalogDigest returns a deterministic lowercase raw 64-hex SHA-256
// over pathless generation metadata and a home-ref-sorted complete entry set.
func ComputeCatalogDigest(generation CatalogGeneration) string {
	hash := sha256.New()
	writeDigestString := func(value string) {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(value))
	}
	writeDigestTime := func(value time.Time) { writeDigestString(value.UTC().Format(time.RFC3339Nano)) }
	writeDigestUint := func(value uint64) {
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], value)
		_, _ = hash.Write(encoded[:])
	}

	writeDigestString(generation.WorkspaceID)
	writeDigestString(generation.DaemonID)
	writeDigestString(generation.CatalogID)
	writeDigestUint(generation.PreviousGeneration)
	writeDigestUint(generation.Generation)
	writeDigestString(string(generation.ScanKind))
	writeDigestString(string(generation.Watermark))
	writeDigestTime(generation.StartedAt)
	writeDigestTime(generation.PublishedAt)

	entries := append([]CatalogEntryRecord(nil), generation.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].HomeRef < entries[j].HomeRef })
	for _, entry := range entries {
		writeDigestString(entry.HomeRef)
		writeDigestString(entry.NameRef)
		writeDigestString(string(entry.Provider))
		writeDigestString(fmt.Sprintf("%t", entry.Approved))
		writeDigestString(string(entry.State))
		writeDigestString(string(entry.ReasonCode))
		writeDigestUint(uint64(entry.ActiveRefs))
		writeDigestTime(entry.FirstSeenAt)
		writeDigestTime(entry.LastSeenAt)
		writeDigestTime(entry.LastFullScanAt)
		if entry.HealthWatermark != nil {
			writeDigestTime(*entry.HealthWatermark)
		} else {
			writeDigestString("")
		}
		if entry.MissingWatermark != nil {
			writeDigestTime(*entry.MissingWatermark)
		} else {
			writeDigestString("")
		}
		writeDigestString(entry.TTL.String())
		writeDigestTime(entry.RetentionDeadline)
	}

	lifecycle := append([]LifecycleRecord(nil), generation.Lifecycle...)
	sort.Slice(lifecycle, func(i, j int) bool { return lifecycle[i].HomeRef < lifecycle[j].HomeRef })
	for _, record := range lifecycle {
		writeDigestString(record.HomeRef)
		writeDigestString(record.NameRef)
		writeDigestString(string(record.Provider))
		writeDigestString(string(record.State))
		writeDigestString(string(record.ReasonCode))
		writeDigestUint(uint64(record.ActiveRefs))
		writeDigestUint(record.Generation)
		writeDigestTime(record.UpdatedAt)
		writeDigestTime(record.RetentionDeadline)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func cloneGeneration(source CatalogGeneration) CatalogGeneration {
	clone := source
	clone.Entries = make([]CatalogEntryRecord, len(source.Entries))
	for i, entry := range source.Entries {
		clone.Entries[i] = entry
		if entry.HealthWatermark != nil {
			clone.Entries[i].HealthWatermark = timePointer(*entry.HealthWatermark)
		}
		if entry.MissingWatermark != nil {
			clone.Entries[i].MissingWatermark = timePointer(*entry.MissingWatermark)
		}
	}
	clone.Lifecycle = append([]LifecycleRecord(nil), source.Lifecycle...)
	return clone
}
