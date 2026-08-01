package credentialcatalog

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const privateModeMask = os.FileMode(0o077)

var (
	ErrInvalidRoot            = errors.New("credential catalog: invalid controlled root")
	ErrRootChanged            = errors.New("credential catalog: controlled root identity changed")
	ErrHomeNotHealthy         = errors.New("credential catalog: home is not healthy")
	ErrHomeTombstoned         = errors.New("credential catalog: home is tombstoned")
	ErrHomeNotFound           = errors.New("credential catalog: home ref not found")
	ErrRefUnderflow           = errors.New("credential catalog: active ref underflow")
	ErrRevalidateFailed       = errors.New("credential catalog: launch-time revalidation failed")
	ErrMissingTombstoneStore  = errors.New("credential catalog: durable lifecycle store is required")
	ErrInvalidCatalogIdentity = errors.New("credential catalog: workspace, daemon, and catalog identifiers are required")
	ErrInvalidWatermarks      = errors.New("credential catalog: invalid watermark thresholds")
)

// Config defines one private controlled root and its durable catalog identity.
type Config struct {
	Root              string
	Provider          Provider
	WorkspaceID       string
	DaemonID          string
	CatalogID         string
	TTL               time.Duration
	RetentionPeriod   time.Duration
	LowWatermark      int
	HighWatermark     int
	CriticalWatermark int
	Store             LifecycleStore
}

type tombstoneState struct {
	retiredAt         time.Time
	retentionDeadline time.Time
}

type lifecycleState struct {
	tombstones          map[string]tombstoneState
	tombstonedNames     map[string]tombstoneState
	tombstoneHomeToName map[string]string
	draining            map[string]discoveredHome
	missing             map[string]discoveredHome
	retired             map[string]discoveredHome
}

// Catalog serializes complete scans and publishes only generations that have
// first been durably appended by the injected store.
type Catalog struct {
	mu                sync.RWMutex
	root              string
	rootIdentity      fileIdentity
	identity          CatalogIdentity
	provider          Provider
	layout            Layout
	ttl               time.Duration
	retentionPeriod   time.Duration
	lowWatermark      int
	highWatermark     int
	criticalWatermark int
	store             LifecycleStore
	generation        uint64
	current           Snapshot
	hasCurrent        bool
	startupPending    bool
	activeRefs        map[string]int
	lifecycleState
	watcherOverflows uint64
	lastOverflowAt   time.Time
	lastHintAt       time.Time
}

// New validates dependencies and loads the latest immutable generation. No
// memory or file fallback is selected when production persistence is absent.
func New(config Config) (*Catalog, error) {
	if config.Store == nil {
		return nil, ErrMissingTombstoneStore
	}
	identity := CatalogIdentity{WorkspaceID: config.WorkspaceID, DaemonID: config.DaemonID, CatalogID: config.CatalogID}
	if !identity.valid() {
		return nil, ErrInvalidCatalogIdentity
	}
	layout, ok := LayoutFor(config.Provider)
	if !ok {
		return nil, fmt.Errorf("credential catalog: unsupported provider")
	}
	if err := validateLayout(layout); err != nil {
		return nil, err
	}
	root := filepath.Clean(config.Root)
	rootIdentity, err := inspectControlledRoot(root)
	if err != nil {
		return nil, err
	}

	ttl := config.TTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	retention := config.RetentionPeriod
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}
	low, high, critical := config.LowWatermark, config.HighWatermark, config.CriticalWatermark
	if low == 0 {
		low = 8000
	}
	if high == 0 {
		high = 10000
	}
	if critical == 0 {
		critical = 12000
	}
	if low < 0 || high <= low || critical <= high {
		return nil, ErrInvalidWatermarks
	}

	catalog := &Catalog{
		root:              root,
		rootIdentity:      rootIdentity,
		identity:          identity,
		provider:          config.Provider,
		layout:            layout,
		ttl:               ttl,
		retentionPeriod:   retention,
		lowWatermark:      low,
		highWatermark:     high,
		criticalWatermark: critical,
		store:             config.Store,
		startupPending:    true,
		activeRefs:        make(map[string]int),
		lifecycleState:    newLifecycleState(),
	}
	if err := catalog.loadLatestGeneration(context.Background()); err != nil {
		return nil, fmt.Errorf("credential catalog: failed to load durable generation: %w", err)
	}
	return catalog, nil
}

func newLifecycleState() lifecycleState {
	return lifecycleState{
		tombstones:          make(map[string]tombstoneState),
		tombstonedNames:     make(map[string]tombstoneState),
		tombstoneHomeToName: make(map[string]string),
		draining:            make(map[string]discoveredHome),
		missing:             make(map[string]discoveredHome),
		retired:             make(map[string]discoveredHome),
	}
}

func (c *Catalog) loadLatestGeneration(ctx context.Context) error {
	generation, err := c.store.LoadLatestGeneration(ctx, c.identity)
	if err != nil {
		return err
	}
	if generation == nil {
		return nil
	}
	if generation.CatalogIdentity != c.identity {
		return ErrInvalidGeneration
	}
	if err := validateGeneration(*generation); err != nil {
		return err
	}

	c.generation = generation.Generation
	homes := make(map[string]discoveredHome, len(generation.Entries))
	snapshot := Snapshot{
		workspaceID: c.identity.WorkspaceID, daemonID: c.identity.DaemonID, catalogID: c.identity.CatalogID,
		previousGeneration: generation.PreviousGeneration, generation: generation.Generation,
		scanKind: generation.ScanKind, digest: generation.CatalogDigest, startedAt: generation.StartedAt,
		capturedAt: generation.PublishedAt, publishedAt: generation.PublishedAt, ttl: c.ttl,
		provider: c.provider, watermark: generation.Watermark,
	}
	for _, record := range generation.Entries {
		if record.Provider != c.provider {
			return ErrInvalidGeneration
		}
		home := discoveredHome{entry: entryFromRecord(record)}
		homes[record.HomeRef] = home
		if record.ActiveRefs > 0 {
			c.activeRefs[record.HomeRef] = record.ActiveRefs
		}
		switch record.State {
		case StateHealthy:
			snapshot.entries = append(snapshot.entries, home.entry)
		case StateQuarantined:
			snapshot.quarantined = append(snapshot.quarantined, Quarantine{candidateRef: record.HomeRef, provider: record.Provider, reason: record.ReasonCode})
		case StateDraining:
			c.draining[record.HomeRef] = home
		case StateMissing:
			c.missing[record.HomeRef] = home
		case StateRetired:
			c.retired[record.HomeRef] = home
		}
	}
	for _, record := range generation.Lifecycle {
		if record.Provider != c.provider || record.Generation != generation.Generation {
			return ErrInvalidGeneration
		}
		home, ok := homes[record.HomeRef]
		if !ok {
			home = discoveredHome{entry: Entry{homeRef: record.HomeRef, nameRef: record.NameRef, provider: record.Provider}}
		}
		home.entry.nameRef = record.NameRef
		home.entry.state = record.State
		home.entry.reason = record.ReasonCode
		home.entry.activeRefs = record.ActiveRefs
		home.entry.retentionDeadline = record.RetentionDeadline
		if record.ActiveRefs > 0 {
			c.activeRefs[record.HomeRef] = record.ActiveRefs
		}
		switch record.State {
		case StateDraining:
			c.draining[record.HomeRef] = home
		case StateMissing:
			c.missing[record.HomeRef] = home
		case StateRetired:
			c.retired[record.HomeRef] = home
		}
		if record.ReasonCode == ReasonTombstoned {
			tombstone := tombstoneState{retiredAt: record.UpdatedAt, retentionDeadline: record.RetentionDeadline}
			c.tombstones[record.HomeRef] = tombstone
			if record.NameRef != "" {
				c.tombstonedNames[record.NameRef] = tombstone
				c.tombstoneHomeToName[record.HomeRef] = record.NameRef
			}
		}
	}
	snapshot.draining, snapshot.missing, snapshot.retired = nil, nil, nil
	c.populateLifecycleSnapshot(&snapshot, c.lifecycleState, c.activeRefs, generation.PublishedAt, false)
	sortSnapshot(&snapshot)
	c.current = cloneSnapshot(snapshot)
	c.hasCurrent = true
	return nil
}

// Reconcile performs a complete startup or periodic scan.
func (c *Catalog) Reconcile() (Snapshot, error) {
	return c.ReconcileContext(context.Background())
}

func (c *Catalog) ReconcileContext(ctx context.Context) (Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	kind := ScanPeriodic
	if c.startupPending {
		kind = ScanStartup
	}
	return c.reconcileLocked(ctx, kind)
}

// ReconcileWithKind performs a complete scan with an explicit C2 scan reason.
func (c *Catalog) ReconcileWithKind(ctx context.Context, kind ScanKind) (Snapshot, error) {
	if !validScanKind(kind) {
		return Snapshot{}, ErrInvalidGeneration
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reconcileLocked(ctx, kind)
}

func (c *Catalog) reconcileLocked(ctx context.Context, kind ScanKind) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	startedAt := time.Now().UTC()
	rootIdentity, err := inspectControlledRoot(c.root)
	if err != nil {
		return Snapshot{}, err
	}
	if rootIdentity != c.rootIdentity {
		return Snapshot{}, ErrRootChanged
	}

	candidate := cloneLifecycleState(c.lifecycleState)
	preserveTombstoneMetadata(&candidate, startedAt)
	snapshot, scannedHomes, err := c.scanLocked(candidate)
	if err != nil {
		return Snapshot{}, err
	}
	c.reconcileLifecycleLocked(&candidate, scannedHomes, startedAt)
	c.applyEntryMetadataAndNonReuse(&snapshot, &candidate, scannedHomes, startedAt)

	snapshot.workspaceID = c.identity.WorkspaceID
	snapshot.daemonID = c.identity.DaemonID
	snapshot.catalogID = c.identity.CatalogID
	snapshot.previousGeneration = c.generation
	snapshot.generation = c.generation + 1
	snapshot.scanKind = kind
	snapshot.startedAt = startedAt
	snapshot.capturedAt = startedAt
	snapshot.publishedAt = startedAt
	snapshot.ttl = c.ttl
	snapshot.provider = c.provider
	c.populateLifecycleSnapshot(&snapshot, candidate, c.activeRefs, startedAt, true)
	snapshot.watermark = c.watermarkFor(len(snapshot.entries))
	sortSnapshot(&snapshot)

	generation := c.generationRecord(snapshot, candidate, startedAt)
	generation.CatalogDigest = ComputeCatalogDigest(generation)
	snapshot.digest = generation.CatalogDigest

	// PERSIST-BEFORE-PUBLISH: candidate lifecycle maps, generation, and current
	// snapshot are committed only after the adapter atomically appends the full
	// generation and all entries.
	if err := c.store.AppendGeneration(ctx, generation); err != nil {
		return Snapshot{}, fmt.Errorf("credential catalog: persistence failed prior to publication: %w", err)
	}
	c.lifecycleState = candidate
	c.generation = snapshot.generation
	c.current = cloneSnapshot(snapshot)
	c.hasCurrent = true
	c.startupPending = false
	return cloneSnapshot(snapshot), nil
}

func (c *Catalog) applyEntryMetadataAndNonReuse(snapshot *Snapshot, state *lifecycleState, scanned map[string]discoveredHome, now time.Time) {
	healthy := make([]Entry, 0, len(snapshot.entries))
	for _, entry := range snapshot.entries {
		nameRef := entry.nameRef
		if nameRef == "" {
			nameRef = opaqueNameRef(c.rootIdentity, c.provider, filepath.Base(filepath.Dir(entry.homePath)))
		}
		entry.nameRef = nameRef
		if _, blocked := state.tombstonedNames[nameRef]; blocked {
			snapshot.quarantined = append(snapshot.quarantined, Quarantine{candidateRef: entry.homeRef, provider: c.provider, reason: ReasonTombstoned})
			delete(scanned, entry.homeRef)
			continue
		}
		if _, blocked := state.tombstones[entry.homeRef]; blocked {
			snapshot.quarantined = append(snapshot.quarantined, Quarantine{candidateRef: entry.homeRef, provider: c.provider, reason: ReasonTombstoned})
			delete(scanned, entry.homeRef)
			continue
		}
		if previous, ok := c.entryFromCurrent(entry.homeRef); ok && !previous.firstSeenAt.IsZero() {
			entry.firstSeenAt = previous.firstSeenAt
		} else {
			entry.firstSeenAt = now
		}
		entry.lastSeenAt = now
		entry.lastFullScanAt = now
		entry.healthWatermark = timePointer(now)
		entry.missingWatermark = nil
		entry.ttl = c.ttl
		entry.retentionDeadline = now.Add(c.retentionPeriod)
		entry.approved = true
		entry.activeRefs = c.activeRefs[entry.homeRef]
		delete(state.missing, entry.homeRef)
		delete(state.draining, entry.homeRef)
		delete(state.retired, entry.homeRef)
		healthy = append(healthy, entry)
	}
	snapshot.entries = healthy
}

func (c *Catalog) reconcileLifecycleLocked(state *lifecycleState, scanned map[string]discoveredHome, now time.Time) {
	for ref, home := range c.missing {
		if _, present := scanned[ref]; present {
			delete(state.missing, ref)
			continue
		}
		delete(state.missing, ref)
		c.retireHome(state, ref, home, now)
	}
	for ref, home := range c.draining {
		if _, present := scanned[ref]; present {
			delete(state.draining, ref)
			continue
		}
		if c.activeRefs[ref] == 0 {
			delete(state.draining, ref)
			c.retireHome(state, ref, home, now)
		}
	}
	if !c.hasCurrent {
		return
	}
	for _, previous := range c.current.entries {
		if _, present := scanned[previous.homeRef]; present {
			continue
		}
		home := discoveredHome{entry: previous}
		missingAt := now
		home.entry.lastFullScanAt = now
		home.entry.missingWatermark = &missingAt
		if c.activeRefs[previous.homeRef] > 0 {
			home.entry.state = StateDraining
			state.draining[previous.homeRef] = home
		} else {
			home.entry.state = StateMissing
			state.missing[previous.homeRef] = home
		}
	}
}

func (c *Catalog) retireHome(state *lifecycleState, ref string, home discoveredHome, now time.Time) {
	home.entry.state = StateRetired
	home.entry.reason = ReasonTombstoned
	home.entry.lastFullScanAt = now
	if home.entry.missingWatermark == nil {
		home.entry.missingWatermark = timePointer(now)
	}
	nameRef := home.entry.nameRef
	if nameRef == "" && home.entry.homePath != "" {
		nameRef = opaqueNameRef(c.rootIdentity, c.provider, filepath.Base(filepath.Dir(home.entry.homePath)))
		home.entry.nameRef = nameRef
	}
	state.retired[ref] = home
	recordTombstone(state, ref, nameRef, now, c.retentionPeriod)
}

func (c *Catalog) populateLifecycleSnapshot(snapshot *Snapshot, state lifecycleState, activeRefs map[string]int, now time.Time, fullScan bool) {
	for ref, home := range state.draining {
		snapshot.draining = append(snapshot.draining, lifecycleEntry(home.entry, StateDraining, activeRefs[ref], c.ttl, c.retentionPeriod, now, fullScan))
	}
	for _, home := range state.missing {
		snapshot.missing = append(snapshot.missing, lifecycleEntry(home.entry, StateMissing, 0, c.ttl, c.retentionPeriod, now, fullScan))
	}
	for ref, home := range state.retired {
		entry := lifecycleEntry(home.entry, StateRetired, 0, c.ttl, c.retentionPeriod, now, fullScan)
		if _, tombstoned := state.tombstones[ref]; tombstoned {
			entry.reason = ReasonTombstoned
		}
		snapshot.retired = append(snapshot.retired, entry)
	}
	for ref, tombstone := range state.tombstones {
		snapshot.tombstones = append(snapshot.tombstones, Tombstone{homeRef: ref, retiredAt: tombstone.retiredAt, retentionDeadline: tombstone.retentionDeadline})
	}
}

func lifecycleEntry(entry Entry, state State, activeRefs int, ttl, retention time.Duration, now time.Time, fullScan bool) Entry {
	entry.state, entry.activeRefs, entry.approved, entry.ttl = state, activeRefs, false, ttl
	if fullScan {
		entry.lastFullScanAt = now
	}
	if entry.firstSeenAt.IsZero() {
		entry.firstSeenAt = now
	}
	if entry.lastSeenAt.IsZero() || entry.lastSeenAt.Before(entry.firstSeenAt) {
		entry.lastSeenAt = entry.firstSeenAt
	}
	if entry.lastFullScanAt.IsZero() || entry.lastFullScanAt.Before(entry.lastSeenAt) {
		entry.lastFullScanAt = entry.lastSeenAt
	}
	if entry.missingWatermark == nil {
		entry.missingWatermark = timePointer(now)
	}
	if !entry.retentionDeadline.After(entry.firstSeenAt) {
		entry.retentionDeadline = now.Add(retention)
	}
	return entry
}

func (c *Catalog) generationRecord(snapshot Snapshot, state lifecycleState, now time.Time) CatalogGeneration {
	records := make(map[string]CatalogEntryRecord)
	addEntry := func(entry Entry) { records[entry.homeRef] = recordFromEntry(entry) }
	for _, entry := range snapshot.entries {
		addEntry(entry)
	}
	for _, quarantine := range snapshot.quarantined {
		records[quarantine.candidateRef] = CatalogEntryRecord{
			HomeRef: quarantine.candidateRef, Provider: quarantine.provider, State: StateQuarantined,
			ReasonCode: quarantine.reason, FirstSeenAt: now, LastSeenAt: now, LastFullScanAt: now,
			TTL: c.ttl, RetentionDeadline: now.Add(c.retentionPeriod),
		}
	}
	for _, entry := range snapshot.draining {
		addEntry(entry)
	}
	for _, entry := range snapshot.missing {
		addEntry(entry)
	}
	for _, entry := range snapshot.retired {
		addEntry(entry)
	}
	for ref, tombstone := range state.tombstones {
		if _, exists := records[ref]; exists {
			continue
		}
		records[ref] = CatalogEntryRecord{
			HomeRef: ref, NameRef: state.tombstoneHomeToName[ref], Provider: c.provider, State: StateRetired, ReasonCode: ReasonTombstoned,
			FirstSeenAt: tombstone.retiredAt, LastSeenAt: tombstone.retiredAt, LastFullScanAt: tombstone.retiredAt,
			MissingWatermark: timePointer(tombstone.retiredAt), TTL: c.ttl, RetentionDeadline: tombstone.retentionDeadline,
		}
	}
	entries := make([]CatalogEntryRecord, 0, len(records))
	for _, record := range records {
		entries = append(entries, record)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].HomeRef < entries[j].HomeRef })

	lifecycleByRef := make(map[string]LifecycleRecord)
	addLifecycle := func(ref string, home discoveredHome, stateValue State, active int) {
		entry, updatedAt := home.entry, now
		if entry.missingWatermark != nil {
			updatedAt = *entry.missingWatermark
		}
		reason, nameRef, deadline := entry.reason, entry.nameRef, entry.retentionDeadline
		if tombstone, ok := state.tombstones[ref]; ok {
			reason, updatedAt, deadline = ReasonTombstoned, tombstone.retiredAt, tombstone.retentionDeadline
			if stored := state.tombstoneHomeToName[ref]; stored != "" {
				nameRef = stored
			}
		}
		if !deadline.After(updatedAt) {
			deadline = updatedAt.Add(c.retentionPeriod)
		}
		lifecycleByRef[ref] = LifecycleRecord{
			HomeRef: ref, NameRef: nameRef, Provider: c.provider, State: stateValue, ReasonCode: reason,
			ActiveRefs: active, Generation: snapshot.generation, UpdatedAt: updatedAt, RetentionDeadline: deadline,
		}
	}
	for ref, home := range state.draining {
		addLifecycle(ref, home, StateDraining, activeRefsValue(snapshot, ref))
	}
	for ref, home := range state.missing {
		addLifecycle(ref, home, StateMissing, 0)
	}
	for ref, home := range state.retired {
		addLifecycle(ref, home, StateRetired, 0)
	}
	lifecycle := make([]LifecycleRecord, 0, len(lifecycleByRef))
	for _, record := range lifecycleByRef {
		lifecycle = append(lifecycle, record)
	}
	sort.Slice(lifecycle, func(i, j int) bool { return lifecycle[i].HomeRef < lifecycle[j].HomeRef })
	return CatalogGeneration{
		CatalogIdentity: snapshotIdentity(snapshot), PreviousGeneration: snapshot.previousGeneration,
		Generation: snapshot.generation, ScanKind: snapshot.scanKind, Watermark: snapshot.watermark,
		StartedAt: snapshot.startedAt, PublishedAt: snapshot.publishedAt, Entries: entries, Lifecycle: lifecycle,
	}
}

func activeRefsValue(snapshot Snapshot, ref string) int {
	for _, entry := range snapshot.draining {
		if entry.homeRef == ref {
			return entry.activeRefs
		}
	}
	return 0
}

func snapshotIdentity(snapshot Snapshot) CatalogIdentity {
	return CatalogIdentity{WorkspaceID: snapshot.workspaceID, DaemonID: snapshot.daemonID, CatalogID: snapshot.catalogID}
}

func recordFromEntry(entry Entry) CatalogEntryRecord {
	return CatalogEntryRecord{
		HomeRef: entry.homeRef, NameRef: entry.nameRef, Provider: entry.provider, Approved: entry.approved, State: entry.state,
		ReasonCode: entry.reason, ActiveRefs: entry.activeRefs, FirstSeenAt: entry.firstSeenAt, LastSeenAt: entry.lastSeenAt,
		LastFullScanAt: entry.lastFullScanAt, HealthWatermark: entry.healthWatermark,
		MissingWatermark: entry.missingWatermark, TTL: entry.ttl, RetentionDeadline: entry.retentionDeadline,
	}
}

func entryFromRecord(record CatalogEntryRecord) Entry {
	return Entry{
		homeRef: record.HomeRef, nameRef: record.NameRef, provider: record.Provider, state: record.State, reason: record.ReasonCode,
		approved: record.Approved, activeRefs: record.ActiveRefs, firstSeenAt: record.FirstSeenAt, lastSeenAt: record.LastSeenAt,
		lastFullScanAt: record.LastFullScanAt, healthWatermark: record.HealthWatermark,
		missingWatermark: record.MissingWatermark, ttl: record.TTL, retentionDeadline: record.RetentionDeadline,
	}
}

func (c *Catalog) watermarkFor(count int) Watermark {
	switch {
	case count >= c.criticalWatermark:
		return WatermarkCritical
	case count >= c.highWatermark:
		return WatermarkHigh
	case count >= c.lowWatermark:
		return WatermarkLow
	default:
		return WatermarkNormal
	}
}

// Current returns the last complete locally published snapshot.
func (c *Catalog) Current() (Snapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hasCurrent {
		return Snapshot{}, false
	}
	if c.current.IsExpired(time.Now().UTC()) {
		kind := ScanPeriodic
		if c.startupPending {
			kind = ScanStartup
		}
		if snapshot, err := c.reconcileLocked(context.Background(), kind); err == nil {
			return snapshot, true
		}
	}
	return cloneSnapshot(c.current), true
}

// AcquireRef reserves a currently healthy home. Draining homes reject new work.
func (c *Catalog) AcquireRef(homeRef string) (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.startupPending {
		return nil, ErrHomeNotHealthy
	}
	if _, tombstoned := c.tombstones[homeRef]; tombstoned {
		return nil, ErrHomeTombstoned
	}
	found := false
	if c.hasCurrent {
		for _, entry := range c.current.entries {
			if entry.homeRef == homeRef {
				found = true
				break
			}
		}
	}
	if !found {
		if _, exists := c.draining[homeRef]; exists {
			return nil, ErrHomeNotHealthy
		}
		if _, exists := c.retired[homeRef]; exists {
			return nil, ErrHomeNotHealthy
		}
		return nil, ErrHomeNotFound
	}
	c.activeRefs[homeRef]++
	var releaseMu sync.Mutex
	released := false
	return func() {
		releaseMu.Lock()
		defer releaseMu.Unlock()
		if !released && c.ReleaseRef(homeRef) == nil {
			released = true
		}
	}, nil
}

// ReleaseRef decrements a reference and retires a fully drained home locally;
// the next generation durably records that transition before publication.
func (c *Catalog) ReleaseRef(homeRef string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := c.activeRefs[homeRef]
	if count <= 0 {
		return ErrRefUnderflow
	}
	if count > 1 {
		c.activeRefs[homeRef] = count - 1
		return nil
	}
	if home, draining := c.draining[homeRef]; draining {
		candidate, refs := cloneLifecycleState(c.lifecycleState), cloneActiveRefs(c.activeRefs)
		delete(refs, homeRef)
		delete(candidate.draining, homeRef)
		c.retireHome(&candidate, homeRef, home, time.Now().UTC())
		if err := c.persistLifecycleTransitionLocked(context.Background(), candidate, refs, nil); err != nil {
			return err
		}
		return nil
	}
	delete(c.activeRefs, homeRef)
	return nil
}

func (c *Catalog) ActiveRefs(homeRef string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeRefs[homeRef]
}

// Revalidate performs launch-time metadata validation without opening the
// credential artifact. Failure blocks launch and establishes non-reuse state.
func (c *Catalog) Revalidate(homeRef string) (Entry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.startupPending {
		return Entry{}, ErrHomeNotHealthy
	}
	if _, tombstoned := c.tombstones[homeRef]; tombstoned {
		return Entry{}, ErrHomeTombstoned
	}
	var target Entry
	found := false
	if c.hasCurrent {
		for _, entry := range c.current.entries {
			if entry.homeRef == homeRef {
				target, found = entry, true
				break
			}
		}
	}
	if !found {
		return Entry{}, ErrHomeNotFound
	}
	rootIdentity, err := inspectControlledRoot(c.root)
	if err != nil || rootIdentity != c.rootIdentity {
		return Entry{}, ErrRootChanged
	}
	candidatePath, candidateName := filepath.Dir(target.homePath), filepath.Base(filepath.Dir(target.homePath))
	discovered, quarantine := c.inspectCandidate(candidatePath, candidateName)
	if quarantine != nil || discovered == nil || discovered.entry.homeRef != homeRef {
		candidate, refs := cloneLifecycleState(c.lifecycleState), cloneActiveRefs(c.activeRefs)
		c.quarantineHome(&candidate, refs, homeRef, target)
		reason := Quarantine{candidateRef: homeRef, provider: c.provider, reason: ReasonRevalidationFailed}
		if persistErr := c.persistLifecycleTransitionLocked(context.Background(), candidate, refs, &reason); persistErr != nil {
			return Entry{}, errors.Join(ErrRevalidateFailed, persistErr)
		}
		return Entry{}, ErrRevalidateFailed
	}
	target.activeRefs = c.activeRefs[homeRef]
	return target, nil
}

func (c *Catalog) quarantineHome(state *lifecycleState, activeRefs map[string]int, homeRef string, entry Entry) {
	now := time.Now().UTC()
	recordTombstone(state, homeRef, entry.nameRef, now, c.retentionPeriod)
	home := discoveredHome{entry: entry}
	if activeRefs[homeRef] > 0 {
		home.entry.state, home.entry.missingWatermark = StateDraining, timePointer(now)
		state.draining[homeRef] = home
		return
	}
	home.entry.state, home.entry.reason, home.entry.missingWatermark = StateRetired, ReasonTombstoned, timePointer(now)
	state.retired[homeRef] = home
}

func (c *Catalog) persistLifecycleTransitionLocked(ctx context.Context, candidate lifecycleState, refs map[string]int, quarantine *Quarantine) error {
	if !c.hasCurrent || c.startupPending {
		return ErrInvalidGeneration
	}
	now := time.Now().UTC()
	snapshot := cloneSnapshot(c.current)
	healthy := make([]Entry, 0, len(snapshot.entries))
	for _, entry := range snapshot.entries {
		_, tombstoned := candidate.tombstones[entry.homeRef]
		_, draining := candidate.draining[entry.homeRef]
		_, missing := candidate.missing[entry.homeRef]
		_, retired := candidate.retired[entry.homeRef]
		if tombstoned || draining || missing || retired {
			continue
		}
		entry.activeRefs = refs[entry.homeRef]
		healthy = append(healthy, entry)
	}
	snapshot.entries = healthy
	if quarantine != nil {
		found := false
		for _, existing := range snapshot.quarantined {
			if existing.candidateRef == quarantine.candidateRef && existing.reason == quarantine.reason {
				found = true
				break
			}
		}
		if !found {
			snapshot.quarantined = append(snapshot.quarantined, *quarantine)
		}
	}
	snapshot.draining, snapshot.missing, snapshot.retired, snapshot.tombstones = nil, nil, nil, nil
	snapshot.previousGeneration, snapshot.generation, snapshot.scanKind = c.generation, c.generation+1, ScanRequested
	snapshot.startedAt, snapshot.capturedAt, snapshot.publishedAt = now, now, now
	snapshot.watermark = c.watermarkFor(len(snapshot.entries))
	c.populateLifecycleSnapshot(&snapshot, candidate, refs, now, false)
	sortSnapshot(&snapshot)
	generation := c.generationRecord(snapshot, candidate, now)
	generation.CatalogDigest = ComputeCatalogDigest(generation)
	snapshot.digest = generation.CatalogDigest
	if err := c.store.AppendGeneration(ctx, generation); err != nil {
		return fmt.Errorf("credential catalog: lifecycle transition persistence failed: %w", err)
	}
	c.lifecycleState, c.activeRefs, c.generation, c.current = candidate, refs, snapshot.generation, cloneSnapshot(snapshot)
	return nil
}

func cloneActiveRefs(source map[string]int) map[string]int {
	clone := make(map[string]int, len(source))
	for ref, count := range source {
		clone[ref] = count
	}
	return clone
}

// NotifyHint performs a requested complete scan; hints are never trusted as a
// partial source of truth.
func (c *Catalog) NotifyHint(_ string) (Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastHintAt = time.Now().UTC()
	return c.reconcileLocked(context.Background(), ScanRequested)
}

func (c *Catalog) NotifyHintLoss() (Snapshot, error) {
	return c.ReconcileWithKind(context.Background(), ScanHintLoss)
}

func (c *Catalog) NotifyWatcherRestart() (Snapshot, error) {
	return c.ReconcileWithKind(context.Background(), ScanWatcherRestart)
}

// NotifyWatcherOverflow records event loss and performs an immediate complete scan.
func (c *Catalog) NotifyWatcherOverflow() (Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watcherOverflows++
	c.lastOverflowAt = time.Now().UTC()
	return c.reconcileLocked(context.Background(), ScanOverflow)
}

func (c *Catalog) WatcherOverflowCount() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.watcherOverflows
}

func (c *Catalog) entryFromCurrent(ref string) (Entry, bool) {
	if !c.hasCurrent {
		return Entry{}, false
	}
	for _, group := range [][]Entry{c.current.entries, c.current.draining, c.current.missing, c.current.retired} {
		for _, entry := range group {
			if entry.homeRef == ref {
				return entry, true
			}
		}
	}
	return Entry{}, false
}

func recordTombstone(state *lifecycleState, homeRef, nameRef string, now time.Time, retention time.Duration) {
	deadline := now.Add(retention)
	state.tombstones[homeRef] = tombstoneState{retiredAt: now, retentionDeadline: deadline}
	if nameRef == "" {
		return
	}
	state.tombstonedNames[nameRef] = tombstoneState{retiredAt: now, retentionDeadline: deadline}
	state.tombstoneHomeToName[homeRef] = nameRef
}

// Tombstone retention deadlines are evidence metadata, not deletion timers.
// Non-reuse markers intentionally survive their deadlines. This catalog never
// copies or deletes source credential folders; physical cleanup belongs to a
// separately authorized task-local lifecycle owner.
func preserveTombstoneMetadata(_ *lifecycleState, _ time.Time) {}

func cloneLifecycleState(source lifecycleState) lifecycleState {
	clone := newLifecycleState()
	for key, value := range source.tombstones {
		clone.tombstones[key] = value
	}
	for key, value := range source.tombstonedNames {
		clone.tombstonedNames[key] = value
	}
	for key, value := range source.tombstoneHomeToName {
		clone.tombstoneHomeToName[key] = value
	}
	for key, value := range source.draining {
		clone.draining[key] = value
	}
	for key, value := range source.missing {
		clone.missing[key] = value
	}
	for key, value := range source.retired {
		clone.retired[key] = value
	}
	return clone
}

type discoveredHome struct {
	entry      Entry
	homeID     fileIdentity
	artifactID fileIdentity
}

func (c *Catalog) scanLocked(state lifecycleState) (Snapshot, map[string]discoveredHome, error) {
	directoryEntries, err := os.ReadDir(c.root)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("credential catalog: controlled root scan failed")
	}
	sort.Slice(directoryEntries, func(i, j int) bool { return directoryEntries[i].Name() < directoryEntries[j].Name() })
	discovered := make([]discoveredHome, 0, len(directoryEntries))
	scanned := make(map[string]discoveredHome, len(directoryEntries))
	quarantined := make([]Quarantine, 0)
	for _, directoryEntry := range directoryEntries {
		nameRef := opaqueNameRef(c.rootIdentity, c.provider, directoryEntry.Name())
		if _, blocked := state.tombstonedNames[nameRef]; blocked {
			quarantined = append(quarantined, *c.quarantineForName(directoryEntry.Name(), ReasonTombstoned))
			continue
		}
		candidatePath := filepath.Join(c.root, directoryEntry.Name())
		home, quarantine := c.inspectCandidate(candidatePath, directoryEntry.Name())
		if quarantine != nil {
			quarantined = append(quarantined, *quarantine)
			continue
		}
		if _, blocked := state.tombstones[home.entry.homeRef]; blocked {
			quarantined = append(quarantined, Quarantine{candidateRef: home.entry.homeRef, provider: c.provider, reason: ReasonTombstoned})
			continue
		}
		discovered = append(discovered, *home)
		scanned[home.entry.homeRef] = *home
	}
	conflicts := conflictingIndexes(discovered)
	entries := make([]Entry, 0, len(discovered)-len(conflicts))
	for index, home := range discovered {
		if _, conflict := conflicts[index]; conflict {
			quarantined = append(quarantined, Quarantine{candidateRef: home.entry.homeRef, provider: c.provider, reason: ReasonFilesystemIdentityConflict})
			delete(scanned, home.entry.homeRef)
			continue
		}
		entries = append(entries, home.entry)
	}
	return Snapshot{provider: c.provider, entries: entries, quarantined: quarantined}, scanned, nil
}

func (c *Catalog) inspectCandidate(candidatePath, candidateName string) (*discoveredHome, *Quarantine) {
	candidateInfo, err := os.Lstat(candidatePath)
	if err != nil {
		return nil, c.quarantineForName(candidateName, ReasonMetadataUnavailable)
	}
	candidateID, err := identityFromInfo(candidateInfo)
	if err != nil {
		return nil, c.quarantineForName(candidateName, ReasonMetadataUnavailable)
	}
	candidateRef := opaqueRef("candidate", c.rootIdentity, c.provider, candidateID)
	quarantine := func(reason QuarantineReason) (*discoveredHome, *Quarantine) {
		return nil, &Quarantine{candidateRef: candidateRef, provider: c.provider, reason: reason}
	}
	if candidateInfo.Mode()&os.ModeSymlink != 0 {
		return quarantine(ReasonCandidateSymlink)
	}
	if !candidateInfo.IsDir() {
		return quarantine(ReasonCandidateNotDirectory)
	}
	if reason := privateMetadataReason(candidateInfo); reason != "" {
		return quarantine(reason)
	}
	if !pathWithin(c.root, candidatePath) {
		return quarantine(ReasonPathEscape)
	}

	homePath := filepath.Join(candidatePath, c.layout.homeRelative)
	homeID, reason := inspectDirectoryChain(c.root, candidatePath, c.layout.homeRelative)
	if reason != "" {
		return quarantine(reason)
	}
	artifactPath := filepath.Join(homePath, c.layout.artifactRelative)
	artifactID, reason := inspectArtifact(c.root, homePath, c.layout.artifactRelative)
	if reason != "" {
		return quarantine(reason)
	}
	return &discoveredHome{
		entry: Entry{
			homeRef: opaqueRef("home", c.rootIdentity, c.provider, homeID), nameRef: opaqueNameRef(c.rootIdentity, c.provider, candidateName),
			provider: c.provider, state: StateHealthy, homePath: homePath, artifactPath: artifactPath,
			discoveredAt: candidateInfo.ModTime().UTC(),
		},
		homeID: homeID, artifactID: artifactID,
	}, nil
}

func (c *Catalog) quarantineForName(name string, reason QuarantineReason) *Quarantine {
	return &Quarantine{candidateRef: opaqueCandidateNameRef(c.rootIdentity, c.provider, name), provider: c.provider, reason: reason}
}

func inspectControlledRoot(root string) (fileIdentity, error) {
	if root == "" || !filepath.IsAbs(root) || root == filepath.Clean(string(filepath.Separator)) {
		return fileIdentity{}, ErrInvalidRoot
	}
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fileIdentity{}, ErrInvalidRoot
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || filepath.Clean(resolved) != root {
		return fileIdentity{}, ErrInvalidRoot
	}
	if privateMetadataReason(info) != "" {
		return fileIdentity{}, ErrInvalidRoot
	}
	identity, err := identityFromInfo(info)
	if err != nil {
		return fileIdentity{}, ErrInvalidRoot
	}
	return identity, nil
}

func inspectDirectoryChain(root, base, relative string) (fileIdentity, QuarantineReason) {
	current := base
	var identity fileIdentity
	for _, component := range strings.Split(filepath.Clean(relative), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		if !pathWithin(root, current) {
			return fileIdentity{}, ReasonPathEscape
		}
		info, err := os.Lstat(current)
		if err != nil {
			return fileIdentity{}, ReasonLayoutMissing
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fileIdentity{}, ReasonLayoutSymlink
		}
		if !info.IsDir() {
			return fileIdentity{}, ReasonLayoutInvalidType
		}
		if reason := privateMetadataReason(info); reason != "" {
			return fileIdentity{}, reason
		}
		identity, err = identityFromInfo(info)
		if err != nil {
			return fileIdentity{}, ReasonMetadataUnavailable
		}
	}
	return identity, ""
}

func inspectArtifact(root, home, relative string) (fileIdentity, QuarantineReason) {
	parentRelative := filepath.Dir(relative)
	if parentRelative != "." {
		if _, reason := inspectDirectoryChain(root, home, parentRelative); reason != "" {
			return fileIdentity{}, reason
		}
	}
	artifactPath := filepath.Join(home, relative)
	if !pathWithin(root, artifactPath) {
		return fileIdentity{}, ReasonPathEscape
	}
	info, err := os.Lstat(artifactPath)
	if err != nil {
		return fileIdentity{}, ReasonLayoutMissing
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fileIdentity{}, ReasonLayoutSymlink
	}
	if !info.Mode().IsRegular() {
		return fileIdentity{}, ReasonLayoutInvalidType
	}
	if reason := privateMetadataReason(info); reason != "" {
		return fileIdentity{}, reason
	}
	identity, err := identityFromInfo(info)
	if err != nil {
		return fileIdentity{}, ReasonMetadataUnavailable
	}
	return identity, ""
}

func privateMetadataReason(info os.FileInfo) QuarantineReason {
	owned, err := ownedByEffectiveUser(info)
	if err != nil || !owned {
		return ReasonWrongOwner
	}
	if info.Mode().Perm()&privateModeMask != 0 {
		return ReasonPermissionsTooOpen
	}
	return ""
}

func validateLayout(layout Layout) error {
	for _, relative := range []string{layout.homeRelative, layout.artifactRelative} {
		clean := filepath.Clean(relative)
		if relative == "" || clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("credential catalog: unsafe provider layout")
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != "." && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func conflictingIndexes(homes []discoveredHome) map[int]struct{} {
	byHome := make(map[fileIdentity][]int, len(homes))
	byArtifact := make(map[fileIdentity][]int, len(homes))
	for index, home := range homes {
		byHome[home.homeID] = append(byHome[home.homeID], index)
		byArtifact[home.artifactID] = append(byArtifact[home.artifactID], index)
	}
	conflicts := make(map[int]struct{})
	for _, groups := range []map[fileIdentity][]int{byHome, byArtifact} {
		for _, indexes := range groups {
			if len(indexes) < 2 {
				continue
			}
			for _, index := range indexes {
				conflicts[index] = struct{}{}
			}
		}
	}
	return conflicts
}

// opaqueRef returns a deterministic RFC-4122-form UUID suitable for C2's UUID
// home_ref column. The domain prefix remains part of the hash input.
func opaqueRef(prefix string, root fileIdentity, provider Provider, identities ...fileIdentity) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("multica-credential-catalog-v2\x00"))
	_, _ = hash.Write([]byte(prefix))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(provider))
	_, _ = hash.Write([]byte{0})
	writeIdentity(hash.Write, root)
	for _, identity := range identities {
		writeIdentity(hash.Write, identity)
	}
	sum := hash.Sum(nil)
	value := append([]byte(nil), sum[:16]...)
	value[6] = (value[6] & 0x0f) | 0x50
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}

func opaqueNameRef(root fileIdentity, provider Provider, name string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("multica-credential-catalog-name-v2\x00"))
	_, _ = hash.Write([]byte(provider))
	_, _ = hash.Write([]byte{0})
	writeIdentity(hash.Write, root)
	_, _ = hash.Write([]byte(name))
	return "name_" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func opaqueCandidateNameRef(root fileIdentity, provider Provider, name string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("multica-credential-catalog-candidate-name-v2\x00"))
	_, _ = hash.Write([]byte(provider))
	_, _ = hash.Write([]byte{0})
	writeIdentity(hash.Write, root)
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)
	value := append([]byte(nil), sum[:16]...)
	value[6] = (value[6] & 0x0f) | 0x50
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}

func writeIdentity(write func([]byte) (int, error), identity fileIdentity) {
	var encoded [16]byte
	binary.BigEndian.PutUint64(encoded[:8], identity.device)
	binary.BigEndian.PutUint64(encoded[8:], identity.inode)
	_, _ = write(encoded[:])
}
