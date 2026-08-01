package credentialcatalog

import (
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
	ErrInvalidRoot     = errors.New("credential catalog: invalid controlled root")
	ErrRootChanged     = errors.New("credential catalog: controlled root identity changed")
	ErrHomeNotHealthy  = errors.New("credential catalog: home is not healthy")
	ErrHomeTombstoned  = errors.New("credential catalog: home is tombstoned")
	ErrHomeNotFound    = errors.New("credential catalog: home ref not found")
	ErrRefUnderflow    = errors.New("credential catalog: active ref underflow")
	ErrRevalidateFailed = errors.New("credential catalog: launch-time revalidation failed")
)

// Config defines one private controlled root and one provider layout, along
// with optional lifecycle, retention, and watermark parameters.
type Config struct {
	Root            string
	Provider        Provider
	TTL             time.Duration
	RetentionPeriod time.Duration
	HighWatermark   int
	LowWatermark    int
}

// Catalog serializes complete scans, manages lifecycle transitions (healthy,
// draining, retired, tombstoned), tracks active reference counts, performs
// fast launch-time revalidations, and handles watcher overflow recovery.
type Catalog struct {
	mu               sync.RWMutex
	root             string
	rootIdentity     fileIdentity
	provider         Provider
	layout           Layout
	ttl              time.Duration
	retentionPeriod  time.Duration
	highWatermark    int
	lowWatermark     int
	generation       uint64
	current          Snapshot
	hasCurrent       bool
	activeRefs       map[string]int
	tombstones       map[string]time.Time
	draining         map[string]discoveredHome
	watcherOverflows uint64
	lastOverflowAt   time.Time
	lastHintAt       time.Time
}

// New validates the controlled root without reading any child artifact.
func New(config Config) (*Catalog, error) {
	layout, ok := LayoutFor(config.Provider)
	if !ok {
		return nil, fmt.Errorf("credential catalog: unsupported provider")
	}
	if err := validateLayout(layout); err != nil {
		return nil, err
	}
	root := filepath.Clean(config.Root)
	identity, err := inspectControlledRoot(root)
	if err != nil {
		return nil, err
	}

	ttl := config.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	retention := config.RetentionPeriod
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}
	highWM := config.HighWatermark
	if highWM <= 0 {
		highWM = 10000
	}
	lowWM := config.LowWatermark
	if lowWM <= 0 {
		lowWM = 8000
	}

	return &Catalog{
		root:            root,
		rootIdentity:    identity,
		provider:        config.Provider,
		layout:          layout,
		ttl:             ttl,
		retentionPeriod: retention,
		highWatermark:   highWM,
		lowWatermark:    lowWM,
		activeRefs:      make(map[string]int),
		tombstones:      make(map[string]time.Time),
		draining:        make(map[string]discoveredHome),
	}, nil
}

// Reconcile performs one full scan. Candidate-specific failures are published
// as metadata-only quarantine records. Root-level failures abort publication,
// preserving the previous complete generation.
func (c *Catalog) Reconcile() (Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reconcileLocked()
}

func (c *Catalog) reconcileLocked() (Snapshot, error) {
	rootIdentity, err := inspectControlledRoot(c.root)
	if err != nil {
		return Snapshot{}, err
	}
	if rootIdentity != c.rootIdentity {
		return Snapshot{}, ErrRootChanged
	}

	c.pruneTombstonesLocked(time.Now().UTC())

	snapshot, scannedHomes, err := c.scanLocked()
	if err != nil {
		return Snapshot{}, err
	}

	c.reconcileLifecycleLocked(scannedHomes)

	// Filter out healthy entries that are tombstoned
	healthyEntries := make([]Entry, 0, len(snapshot.entries))
	for _, entry := range snapshot.entries {
		if _, tomb := c.tombstones[entry.homeRef]; tomb {
			continue
		}
		entry.activeRefs = c.activeRefs[entry.homeRef]
		healthyEntries = append(healthyEntries, entry)
	}

	drainingEntries := make([]Entry, 0, len(c.draining))
	for ref, home := range c.draining {
		entry := home.entry
		entry.state = StateDraining
		entry.activeRefs = c.activeRefs[ref]
		drainingEntries = append(drainingEntries, entry)
	}

	tombstonedList := make([]string, 0, len(c.tombstones))
	for ref := range c.tombstones {
		tombstonedList = append(tombstonedList, ref)
	}

	snapshot.entries = healthyEntries
	snapshot.draining = drainingEntries
	snapshot.tombstoned = tombstonedList
	snapshot.generation = c.generation + 1
	snapshot.capturedAt = time.Now().UTC()
	sortSnapshot(&snapshot)

	c.enforceWatermarksLocked(&snapshot)

	c.generation = snapshot.generation
	c.current = cloneSnapshot(snapshot)
	c.hasCurrent = true
	return cloneSnapshot(snapshot), nil
}

// Current returns a copy of the last complete snapshot.
func (c *Catalog) Current() (Snapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.hasCurrent {
		return Snapshot{}, false
	}
	return cloneSnapshot(c.current), true
}

// AcquireRef increments the active reference count for a healthy or draining home.
// It returns a release function and an error if the home is tombstoned or unknown.
func (c *Catalog) AcquireRef(homeRef string) (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, tomb := c.tombstones[homeRef]; tomb {
		return nil, ErrHomeTombstoned
	}

	var found bool
	if c.hasCurrent {
		for _, entry := range c.current.entries {
			if entry.homeRef == homeRef {
				found = true
				break
			}
		}
		if !found {
			for _, entry := range c.current.draining {
				if entry.homeRef == homeRef {
					found = true
					break
				}
			}
		}
	}
	if !found {
		if _, drain := c.draining[homeRef]; drain {
			found = true
		}
	}

	if !found {
		return nil, ErrHomeNotFound
	}

	c.activeRefs[homeRef]++

	var once sync.Once
	release := func() {
		once.Do(func() {
			_ = c.ReleaseRef(homeRef)
		})
	}
	return release, nil
}

// ReleaseRef decrements the active reference count for a homeRef. If the home
// is in draining state and activeRefs reaches 0, it transitions to tombstoned.
func (c *Catalog) ReleaseRef(homeRef string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	count, exists := c.activeRefs[homeRef]
	if !exists || count <= 0 {
		return ErrRefUnderflow
	}
	count--
	if count == 0 {
		delete(c.activeRefs, homeRef)
		if _, draining := c.draining[homeRef]; draining {
			delete(c.draining, homeRef)
			c.tombstones[homeRef] = time.Now().UTC()
		}
	} else {
		c.activeRefs[homeRef] = count
	}
	return nil
}

// ActiveRefs returns the current active reference count for a homeRef.
func (c *Catalog) ActiveRefs(homeRef string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeRefs[homeRef]
}

// Revalidate performs a fast launch-time metadata inspection of a single home
// before task launch, confirming metadata safety without opening credential contents.
func (c *Catalog) Revalidate(homeRef string) (Entry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, tomb := c.tombstones[homeRef]; tomb {
		return Entry{}, ErrHomeTombstoned
	}

	var targetEntry Entry
	var found bool
	if c.hasCurrent {
		for _, e := range c.current.entries {
			if e.homeRef == homeRef {
				targetEntry = e
				found = true
				break
			}
		}
	}
	if !found {
		return Entry{}, ErrHomeNotFound
	}

	// Re-verify controlled root identity
	rootId, err := inspectControlledRoot(c.root)
	if err != nil || rootId != c.rootIdentity {
		return Entry{}, ErrRootChanged
	}

	// Inspect candidate home directory
	candidateDir := filepath.Dir(targetEntry.homePath)
	if filepath.Base(targetEntry.homePath) != c.layout.homeRelative {
		candidateDir = filepath.Dir(targetEntry.homePath)
	}
	candidateName := filepath.Base(candidateDir)
	discovered, quarantine := c.inspectCandidate(candidateDir, candidateName)
	if quarantine != nil || discovered == nil || discovered.entry.homeRef != homeRef {
		c.quarantineHomeLocked(homeRef, targetEntry)
		return Entry{}, ErrRevalidateFailed
	}

	targetEntry.activeRefs = c.activeRefs[homeRef]
	return targetEntry, nil
}

// NotifyHint signals a watcher event for a path.
func (c *Catalog) NotifyHint(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastHintAt = time.Now().UTC()
}

// NotifyWatcherOverflow signals an inotify/fsnotify event loss or buffer overflow,
// triggering an immediate full scan to restore full catalog consistency.
func (c *Catalog) NotifyWatcherOverflow() (Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watcherOverflows++
	c.lastOverflowAt = time.Now().UTC()
	return c.reconcileLocked()
}

// WatcherOverflowCount returns the total number of watcher overflows recovered.
func (c *Catalog) WatcherOverflowCount() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.watcherOverflows
}

func (c *Catalog) quarantineHomeLocked(homeRef string, entry Entry) {
	if count := c.activeRefs[homeRef]; count > 0 {
		c.draining[homeRef] = discoveredHome{entry: entry}
	} else {
		c.tombstones[homeRef] = time.Now().UTC()
	}
}

func (c *Catalog) reconcileLifecycleLocked(scannedHomes map[string]discoveredHome) {
	if !c.hasCurrent {
		return
	}
	// Check previously healthy entries
	for _, prevEntry := range c.current.entries {
		ref := prevEntry.homeRef
		if _, exists := scannedHomes[ref]; !exists {
			// Previously healthy home is now missing from scan
			if count := c.activeRefs[ref]; count > 0 {
				c.draining[ref] = discoveredHome{entry: prevEntry}
			} else {
				c.tombstones[ref] = time.Now().UTC()
			}
		}
	}
}

func (c *Catalog) pruneTombstonesLocked(now time.Time) {
	for ref, tombTime := range c.tombstones {
		if now.Sub(tombTime) > c.retentionPeriod && c.activeRefs[ref] == 0 {
			delete(c.tombstones, ref)
		}
	}
}

func (c *Catalog) enforceWatermarksLocked(snapshot *Snapshot) {
	if len(snapshot.entries) > c.highWatermark {
		// Retain up to lowWatermark entries deterministically
		snapshot.entries = snapshot.entries[:c.lowWatermark]
	}
}

type discoveredHome struct {
	entry      Entry
	homeID     fileIdentity
	artifactID fileIdentity
}

func (c *Catalog) scanLocked() (Snapshot, map[string]discoveredHome, error) {
	directoryEntries, err := os.ReadDir(c.root)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("credential catalog: controlled root scan failed")
	}
	sort.Slice(directoryEntries, func(i, j int) bool {
		return directoryEntries[i].Name() < directoryEntries[j].Name()
	})

	discovered := make([]discoveredHome, 0, len(directoryEntries))
	scannedMap := make(map[string]discoveredHome, len(directoryEntries))
	quarantined := make([]Quarantine, 0)
	for _, directoryEntry := range directoryEntries {
		candidatePath := filepath.Join(c.root, directoryEntry.Name())
		home, quarantine := c.inspectCandidate(candidatePath, directoryEntry.Name())
		if quarantine != nil {
			quarantined = append(quarantined, *quarantine)
			continue
		}
		discovered = append(discovered, *home)
		scannedMap[home.entry.homeRef] = *home
	}

	conflicts := conflictingIndexes(discovered)
	entries := make([]Entry, 0, len(discovered)-len(conflicts))
	for index, home := range discovered {
		if _, conflict := conflicts[index]; conflict {
			quarantined = append(quarantined, Quarantine{
				candidateRef: home.entry.homeRef,
				provider:     c.provider,
				reason:       ReasonFilesystemIdentityConflict,
			})
			delete(scannedMap, home.entry.homeRef)
			continue
		}
		entries = append(entries, home.entry)
	}

	return Snapshot{
		provider:    c.provider,
		entries:     entries,
		quarantined: quarantined,
	}, scannedMap, nil
}

func (c *Catalog) inspectCandidate(candidatePath, candidateName string) (*discoveredHome, *Quarantine) {
	candidateInfo, err := os.Lstat(candidatePath)
	if err != nil {
		return nil, c.quarantineForName(candidateName, ReasonMetadataUnavailable)
	}
	candidateID, identityErr := identityFromInfo(candidateInfo)
	if identityErr != nil {
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
			homeRef:      opaqueRef("home", c.rootIdentity, c.provider, homeID),
			provider:     c.provider,
			state:        StateHealthy,
			homePath:     homePath,
			artifactPath: artifactPath,
		},
		homeID:     homeID,
		artifactID: artifactID,
	}, nil
}

func (c *Catalog) quarantineForName(name string, reason QuarantineReason) *Quarantine {
	return &Quarantine{
		candidateRef: opaqueNameRef(c.rootIdentity, c.provider, name),
		provider:     c.provider,
		reason:       reason,
	}
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
		if relative == "" || clean == "." || filepath.IsAbs(clean) || clean == ".." ||
			strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("credential catalog: unsafe provider layout")
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != "." && relative != ".." &&
		!filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
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

func opaqueRef(prefix string, root fileIdentity, provider Provider, identities ...fileIdentity) string {
	hash := sha256.New()
	hash.Write([]byte("multica-credential-catalog-v1\x00"))
	hash.Write([]byte(provider))
	hash.Write([]byte{0})
	writeIdentity(hash.Write, root)
	for _, identity := range identities {
		writeIdentity(hash.Write, identity)
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func opaqueNameRef(root fileIdentity, provider Provider, name string) string {
	hash := sha256.New()
	hash.Write([]byte("multica-credential-catalog-name-v1\x00"))
	hash.Write([]byte(provider))
	hash.Write([]byte{0})
	writeIdentity(hash.Write, root)
	hash.Write([]byte(name))
	return "candidate_" + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}

func writeIdentity(write func([]byte) (int, error), identity fileIdentity) {
	var encoded [16]byte
	binary.BigEndian.PutUint64(encoded[:8], identity.device)
	binary.BigEndian.PutUint64(encoded[8:], identity.inode)
	_, _ = write(encoded[:])
}
