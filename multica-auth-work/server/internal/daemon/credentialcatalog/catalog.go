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
	ErrInvalidRoot = errors.New("credential catalog: invalid controlled root")
	ErrRootChanged = errors.New("credential catalog: controlled root identity changed")
)

// Config defines one private controlled root and one provider layout. A
// separate Catalog should be used for each provider/root authority boundary.
type Config struct {
	Root     string
	Provider Provider
}

// Catalog serializes complete scans and publishes a generation only after the
// entire immediate-child scan and deterministic deduplication succeed.
type Catalog struct {
	mu           sync.RWMutex
	root         string
	rootIdentity fileIdentity
	provider     Provider
	layout       Layout
	generation   uint64
	current      Snapshot
	hasCurrent   bool
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
	return &Catalog{
		root:         root,
		rootIdentity: identity,
		provider:     config.Provider,
		layout:       layout,
	}, nil
}

// Reconcile performs one full scan. Candidate-specific failures are published
// as metadata-only quarantine records. Root-level failures abort publication,
// preserving the previous complete generation.
func (c *Catalog) Reconcile() (Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rootIdentity, err := inspectControlledRoot(c.root)
	if err != nil {
		return Snapshot{}, err
	}
	if rootIdentity != c.rootIdentity {
		return Snapshot{}, ErrRootChanged
	}

	snapshot, err := c.scanLocked()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.generation = c.generation + 1
	snapshot.capturedAt = time.Now().UTC()
	sortSnapshot(&snapshot)

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

type discoveredHome struct {
	entry      Entry
	homeID     fileIdentity
	artifactID fileIdentity
}

func (c *Catalog) scanLocked() (Snapshot, error) {
	directoryEntries, err := os.ReadDir(c.root)
	if err != nil {
		return Snapshot{}, fmt.Errorf("credential catalog: controlled root scan failed")
	}
	// os.ReadDir currently sorts names, but sorting explicitly makes this
	// contract independent of filesystem and implementation ordering.
	sort.Slice(directoryEntries, func(i, j int) bool {
		return directoryEntries[i].Name() < directoryEntries[j].Name()
	})

	discovered := make([]discoveredHome, 0, len(directoryEntries))
	quarantined := make([]Quarantine, 0)
	for _, directoryEntry := range directoryEntries {
		candidatePath := filepath.Join(c.root, directoryEntry.Name())
		home, quarantine := c.inspectCandidate(candidatePath, directoryEntry.Name())
		if quarantine != nil {
			quarantined = append(quarantined, *quarantine)
			continue
		}
		discovered = append(discovered, *home)
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
			continue
		}
		entries = append(entries, home.entry)
	}

	return Snapshot{
		provider:    c.provider,
		entries:     entries,
		quarantined: quarantined,
	}, nil
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
