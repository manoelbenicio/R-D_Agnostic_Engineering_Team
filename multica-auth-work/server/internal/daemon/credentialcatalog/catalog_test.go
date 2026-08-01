//go:build !windows

package credentialcatalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	nameRefPattern = regexp.MustCompile(`^name_[A-Za-z0-9_-]{43}$`)
)

func newPrivateRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "catalog")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeFakeHome(t *testing.T, root, child string, provider Provider) (string, string) {
	t.Helper()
	layout, ok := LayoutFor(provider)
	if !ok {
		t.Fatalf("missing layout for %q", provider)
	}
	home := filepath.Join(root, child, layout.HomeRelative())
	artifact := filepath.Join(home, layout.ArtifactRelative())
	if err := os.MkdirAll(filepath.Dir(artifact), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("synthetic fixture only"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, artifact
}

func testIdentity() CatalogIdentity {
	return CatalogIdentity{WorkspaceID: "workspace-test", DaemonID: "daemon-test", CatalogID: "catalog-test"}
}

func testConfig(root string, provider Provider, store LifecycleStore) Config {
	identity := testIdentity()
	return Config{
		Root: root, Provider: provider, Store: store,
		WorkspaceID: identity.WorkspaceID, DaemonID: identity.DaemonID, CatalogID: identity.CatalogID,
	}
}

func mustCatalog(t *testing.T, root string, provider Provider) (*Catalog, *MemoryLifecycleStore) {
	t.Helper()
	store := NewMemoryLifecycleStore()
	catalog, err := New(testConfig(root, provider, store))
	if err != nil {
		t.Fatal(err)
	}
	return catalog, store
}

func TestDependenciesAndCatalogIdentityFailClosed(t *testing.T) {
	root := newPrivateRoot(t)
	config := testConfig(root, ProviderCodex, nil)
	if _, err := New(config); !errors.Is(err, ErrMissingTombstoneStore) {
		t.Fatalf("nil store err = %v", err)
	}
	config.Store = NewMemoryLifecycleStore()
	config.WorkspaceID = ""
	if _, err := New(config); !errors.Is(err, ErrInvalidCatalogIdentity) {
		t.Fatalf("missing identity err = %v", err)
	}
	if _, err := NewProductionLifecycleStore(nil); !errors.Is(err, ErrMissingProductionAdapter) {
		t.Fatalf("nil production adapter err = %v", err)
	}
}

func TestProductionStoreForwardsInjectedAdapter(t *testing.T) {
	adapter := NewMemoryLifecycleStore()
	production, err := NewProductionLifecycleStore(adapter)
	if err != nil {
		t.Fatal(err)
	}
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderCodex)
	catalog, err := New(testConfig(root, ProviderCodex, production))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Reconcile(); err != nil {
		t.Fatal(err)
	}
	if got := len(adapter.Generations(testIdentity())); got != 1 {
		t.Fatalf("forwarded generations = %d", got)
	}
}

func TestAppendOnlyGenerationContractCarriesC2Metadata(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderCodex)
	catalog, store := mustCatalog(t, root, ProviderCodex)

	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	writeFakeHome(t, root, "two", ProviderCodex)
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if first.PreviousGeneration() != 0 || first.Generation() != 1 || first.ScanKind() != ScanStartup {
		t.Fatalf("first generation = prev:%d current:%d kind:%q", first.PreviousGeneration(), first.Generation(), first.ScanKind())
	}
	if second.PreviousGeneration() != 1 || second.Generation() != 2 || second.ScanKind() != ScanPeriodic {
		t.Fatalf("second generation = prev:%d current:%d kind:%q", second.PreviousGeneration(), second.Generation(), second.ScanKind())
	}
	if first.WorkspaceID() != testIdentity().WorkspaceID || first.DaemonID() != testIdentity().DaemonID || first.CatalogID() != testIdentity().CatalogID {
		t.Fatalf("snapshot identity = %q/%q/%q", first.WorkspaceID(), first.DaemonID(), first.CatalogID())
	}
	if !validRaw64Digest(first.Digest()) || !validRaw64Digest(second.Digest()) || first.Digest() == second.Digest() {
		t.Fatalf("digests = %q, %q", first.Digest(), second.Digest())
	}

	generations := store.Generations(testIdentity())
	if len(generations) != 2 || len(generations[0].Entries) != 1 || len(generations[1].Entries) != 2 {
		t.Fatalf("generation history lengths = %d/%d/%d", len(generations), len(generations[0].Entries), len(generations[1].Entries))
	}
	entry := generations[0].Entries[0]
	if entry.State != StateHealthy || !entry.Approved || entry.TTL <= 0 || !entry.RetentionDeadline.After(entry.FirstSeenAt) || entry.HealthWatermark == nil {
		t.Fatalf("incomplete persisted entry: %#v", entry)
	}
	if !uuidPattern.MatchString(entry.HomeRef) || !nameRefPattern.MatchString(entry.NameRef) || entry.HomeRef == entry.NameRef {
		t.Fatalf("canonical refs home=%q name=%q", entry.HomeRef, entry.NameRef)
	}
	if err := store.AppendGeneration(context.Background(), generations[0]); !errors.Is(err, ErrGenerationConflict) {
		t.Fatalf("reappend err = %v", err)
	}

	// Returned history is a deep copy; mutation cannot rewrite evidence.
	generations[0].Entries[0].State = StateRetired
	if got := store.Generations(testIdentity())[0].Entries[0].State; got != StateHealthy {
		t.Fatalf("stored immutable entry changed to %q", got)
	}
}

func TestRestartLoadsDurableTombstonesAndPreventsNameReuse(t *testing.T) {
	root := newPrivateRoot(t)
	store := NewMemoryLifecycleStore()
	config := testConfig(root, ProviderCodex, store)
	writeFakeHome(t, root, "restart-slot", ProviderCodex)
	firstCatalog, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	first, err := firstCatalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	oldRef := first.Entries()[0].HomeRef()
	if err := os.RemoveAll(filepath.Join(root, "restart-slot")); err != nil {
		t.Fatal(err)
	}
	missing, err := firstCatalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if missing.MissingCount() != 1 || missing.TombstonedCount() != 0 {
		t.Fatalf("first absence missing=%d tombstones=%d", missing.MissingCount(), missing.TombstonedCount())
	}
	retired, err := firstCatalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if retired.RetiredCount() != 1 || retired.TombstonedCount() != 1 || retired.Tombstones()[0].HomeRef() != oldRef {
		t.Fatalf("retired=%d tombstones=%#v", retired.RetiredCount(), retired.Tombstones())
	}

	// Recreate the same candidate name with a different filesystem identity and restart.
	writeFakeHome(t, root, "restart-slot", ProviderCodex)
	restarted, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.AcquireRef(oldRef); !errors.Is(err, ErrHomeNotHealthy) {
		t.Fatalf("restart acquire before startup scan err = %v", err)
	}
	if _, err := restarted.Revalidate(oldRef); !errors.Is(err, ErrHomeNotHealthy) {
		t.Fatalf("restart revalidate before startup scan err = %v", err)
	}
	afterRestart, err := restarted.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if afterRestart.Generation() != 4 || afterRestart.HealthyCount() != 0 || afterRestart.QuarantinedCount() != 1 {
		t.Fatalf("restart generation=%d healthy=%d quarantine=%d", afterRestart.Generation(), afterRestart.HealthyCount(), afterRestart.QuarantinedCount())
	}
	if afterRestart.Quarantined()[0].Reason() != ReasonTombstoned {
		t.Fatalf("restart quarantine reason = %q", afterRestart.Quarantined()[0].Reason())
	}

	latest := store.Generations(testIdentity())[3]
	var homeMarker, lifecycleMarker bool
	for _, entry := range latest.Entries {
		if entry.State == StateRetired && entry.ReasonCode == ReasonTombstoned && entry.HomeRef == oldRef && nameRefPattern.MatchString(entry.NameRef) {
			homeMarker = true
		}
		if entry.State == State("tombstoned") || strings.HasPrefix(entry.HomeRef, "name_") {
			t.Fatalf("invalid durable catalog entry: %#v", entry)
		}
	}
	for _, record := range latest.Lifecycle {
		if record.HomeRef == oldRef && record.ReasonCode == ReasonTombstoned && nameRefPattern.MatchString(record.NameRef) && record.Generation == latest.Generation {
			lifecycleMarker = true
		}
	}
	if !homeMarker || !lifecycleMarker {
		t.Fatalf("durable tombstone markers home=%v lifecycle=%v", homeMarker, lifecycleMarker)
	}
}

func TestActiveReferenceDrainsBeforeRetirement(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "active-slot", ProviderCodex)
	catalog, _ := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	ref := first.Entries()[0].HomeRef()
	release, err := catalog.AcquireRef(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "active-slot")); err != nil {
		t.Fatal(err)
	}
	draining, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if draining.DrainingCount() != 1 || draining.Draining()[0].ActiveRefs() != 1 || draining.TombstonedCount() != 0 {
		t.Fatalf("draining=%d refs=%d tombstones=%d", draining.DrainingCount(), draining.Draining()[0].ActiveRefs(), draining.TombstonedCount())
	}
	if _, err := catalog.AcquireRef(ref); !errors.Is(err, ErrHomeNotHealthy) {
		t.Fatalf("new ref on draining home err = %v", err)
	}
	release()
	retired, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if retired.DrainingCount() != 0 || retired.RetiredCount() != 1 || retired.TombstonedCount() != 1 {
		t.Fatalf("retired lifecycle draining=%d retired=%d tombstones=%d", retired.DrainingCount(), retired.RetiredCount(), retired.TombstonedCount())
	}
}

func TestReleaseRefPersistsImmediatelyAndRollsBackOnAppendFailure(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "release-slot", ProviderCodex)
	catalog, store := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	ref := first.Entries()[0].HomeRef()
	if _, err := catalog.AcquireRef(ref); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "release-slot")); err != nil {
		t.Fatal(err)
	}
	draining, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	store.SetInjectFailAppend(true)
	if err := catalog.ReleaseRef(ref); err == nil {
		t.Fatal("expected release append failure")
	}
	current, _ := catalog.Current()
	if current.Generation() != draining.Generation() || current.DrainingCount() != 1 || catalog.ActiveRefs(ref) != 1 || len(store.Generations(testIdentity())) != 2 {
		t.Fatalf("failed release mutated state generation=%d draining=%d refs=%d history=%d", current.Generation(), current.DrainingCount(), catalog.ActiveRefs(ref), len(store.Generations(testIdentity())))
	}
	store.SetInjectFailAppend(false)
	if err := catalog.ReleaseRef(ref); err != nil {
		t.Fatal(err)
	}
	current, _ = catalog.Current()
	generations := store.Generations(testIdentity())
	if current.Generation() != draining.Generation()+1 || current.RetiredCount() != 1 || current.TombstonedCount() != 1 || catalog.ActiveRefs(ref) != 0 || len(generations) != 3 {
		t.Fatalf("durable release generation=%d retired=%d tombstones=%d refs=%d history=%d", current.Generation(), current.RetiredCount(), current.TombstonedCount(), catalog.ActiveRefs(ref), len(generations))
	}
	latest := generations[2]
	if latest.ScanKind != ScanRequested || len(latest.Lifecycle) != 1 || latest.Lifecycle[0].ReasonCode != ReasonTombstoned {
		t.Fatalf("release generation lifecycle = %#v", latest)
	}
}

func TestFailedRevalidatePersistsImmediatelyAndRollsBackOnAppendFailure(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifact := writeFakeHome(t, root, "revalidate-slot", ProviderCodex)
	catalog, store := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	ref := first.Entries()[0].HomeRef()
	if err := os.Chmod(artifact, 0o666); err != nil {
		t.Fatal(err)
	}
	store.SetInjectFailAppend(true)
	if _, err := catalog.Revalidate(ref); !errors.Is(err, ErrRevalidateFailed) {
		t.Fatalf("failed persistence revalidation err = %v", err)
	}
	current, _ := catalog.Current()
	if current.Generation() != first.Generation() || current.HealthyCount() != 1 || current.TombstonedCount() != 0 || len(store.Generations(testIdentity())) != 1 {
		t.Fatalf("failed revalidation mutated state generation=%d healthy=%d tombstones=%d history=%d", current.Generation(), current.HealthyCount(), current.TombstonedCount(), len(store.Generations(testIdentity())))
	}
	store.SetInjectFailAppend(false)
	if _, err := catalog.Revalidate(ref); !errors.Is(err, ErrRevalidateFailed) {
		t.Fatalf("retry revalidation err = %v", err)
	}
	current, _ = catalog.Current()
	generations := store.Generations(testIdentity())
	if current.Generation() != first.Generation()+1 || current.HealthyCount() != 0 || current.TombstonedCount() != 1 || len(generations) != 2 {
		t.Fatalf("durable revalidation generation=%d healthy=%d tombstones=%d history=%d", current.Generation(), current.HealthyCount(), current.TombstonedCount(), len(generations))
	}
	if generations[1].ScanKind != ScanRequested || len(generations[1].Lifecycle) != 1 || generations[1].Lifecycle[0].ReasonCode != ReasonTombstoned {
		t.Fatalf("revalidation generation lifecycle = %#v", generations[1])
	}
}

func TestTombstoneMetadataSurvivesRetentionWithoutPhysicalCopies(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "ephemeral-slot", ProviderKiro)
	store := NewMemoryLifecycleStore()
	config := testConfig(root, ProviderKiro, store)
	config.RetentionPeriod = time.Millisecond
	catalog, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	ref, nameRef := first.Entries()[0].HomeRef(), first.Entries()[0].NameRef()
	if !nameRefPattern.MatchString(nameRef) {
		t.Fatalf("name ref = %q", nameRef)
	}
	if err := os.RemoveAll(filepath.Join(root, "ephemeral-slot")); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Reconcile(); err != nil {
		t.Fatal(err)
	}
	retired, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if retired.TombstonedCount() != 1 {
		t.Fatalf("initial tombstones = %d", retired.TombstonedCount())
	}
	time.Sleep(5 * time.Millisecond)
	afterDeadline, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if afterDeadline.TombstonedCount() != 1 || afterDeadline.Tombstones()[0].HomeRef() != ref {
		t.Fatalf("expired metadata removed: %#v", afterDeadline.Tombstones())
	}
	children, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Fatalf("catalog created historical physical folders: %#v", children)
	}
	writeFakeHome(t, root, "ephemeral-slot", ProviderKiro)
	reuse, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if reuse.HealthyCount() != 0 || reuse.QuarantinedCount() != 1 || reuse.Quarantined()[0].Reason() != ReasonTombstoned {
		t.Fatalf("name reuse admitted after deadline: healthy=%d quarantine=%#v", reuse.HealthyCount(), reuse.Quarantined())
	}
	latest := store.Generations(testIdentity())[len(store.Generations(testIdentity()))-1]
	if len(latest.Lifecycle) != 1 || latest.Lifecycle[0].NameRef != nameRef {
		t.Fatalf("retained lifecycle metadata = %#v", latest.Lifecycle)
	}
}

func TestPersistenceFailureDoesNotPublishOrMutateCandidateLifecycle(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderCodex)
	catalog, store := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "one")); err != nil {
		t.Fatal(err)
	}
	store.SetInjectFailAppend(true)
	if _, err := catalog.Reconcile(); err == nil {
		t.Fatal("expected append failure")
	}
	current, ok := catalog.Current()
	if !ok || current.Generation() != first.Generation() || current.HealthyCount() != 1 {
		t.Fatalf("published snapshot changed: ok=%v generation=%d healthy=%d", ok, current.Generation(), current.HealthyCount())
	}
	if got := len(store.Generations(testIdentity())); got != 1 {
		t.Fatalf("durable history length after failure = %d", got)
	}
	store.SetInjectFailAppend(false)
	retry, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if retry.Generation() != 2 || retry.PreviousGeneration() != 1 || retry.MissingCount() != 1 || retry.RetiredCount() != 0 {
		t.Fatalf("retry generation=%d previous=%d missing=%d retired=%d", retry.Generation(), retry.PreviousGeneration(), retry.MissingCount(), retry.RetiredCount())
	}
}

func TestOverflowAndHintLossAlwaysProduceCompleteScans(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderKiro)
	catalog, store := mustCatalog(t, root, ProviderKiro)
	if _, err := catalog.Reconcile(); err != nil {
		t.Fatal(err)
	}
	writeFakeHome(t, root, "two", ProviderKiro)
	overflow, err := catalog.NotifyWatcherOverflow()
	if err != nil {
		t.Fatal(err)
	}
	if overflow.ScanKind() != ScanOverflow || overflow.HealthyCount() != 2 || catalog.WatcherOverflowCount() != 1 {
		t.Fatalf("overflow kind=%q healthy=%d count=%d", overflow.ScanKind(), overflow.HealthyCount(), catalog.WatcherOverflowCount())
	}
	if err := os.RemoveAll(filepath.Join(root, "one")); err != nil {
		t.Fatal(err)
	}
	hintLoss, err := catalog.NotifyHintLoss()
	if err != nil {
		t.Fatal(err)
	}
	if hintLoss.ScanKind() != ScanHintLoss || hintLoss.HealthyCount() != 1 || hintLoss.MissingCount() != 1 {
		t.Fatalf("hint-loss kind=%q healthy=%d missing=%d", hintLoss.ScanKind(), hintLoss.HealthyCount(), hintLoss.MissingCount())
	}
	generations := store.Generations(testIdentity())
	if len(generations) != 3 || len(generations[1].Entries) != 2 || generations[2].ScanKind != ScanHintLoss {
		t.Fatalf("persisted full scans = %#v", generations)
	}
}

func TestLaunchRevalidationFailsClosedAndBlocksReuse(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifact := writeFakeHome(t, root, "launch-slot", ProviderAntigravity)
	catalog, _ := mustCatalog(t, root, ProviderAntigravity)
	snapshot, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	ref := snapshot.Entries()[0].HomeRef()
	if entry, err := catalog.Revalidate(ref); err != nil || entry.HomeRef() != ref {
		t.Fatalf("healthy revalidation entry=%#v err=%v", entry, err)
	}
	if err := os.Chmod(artifact, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Revalidate(ref); !errors.Is(err, ErrRevalidateFailed) {
		t.Fatalf("unsafe launch revalidation err = %v", err)
	}
	if _, err := catalog.AcquireRef(ref); !errors.Is(err, ErrHomeTombstoned) {
		t.Fatalf("launch after failed revalidation err = %v", err)
	}
	persisted, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.HealthyCount() != 0 || persisted.TombstonedCount() != 1 {
		t.Fatalf("post-revalidation healthy=%d tombstones=%d", persisted.HealthyCount(), persisted.TombstonedCount())
	}
}

func TestWatermarkVocabularyDoesNotTruncateDiscovery(t *testing.T) {
	root := newPrivateRoot(t)
	for i := 0; i < 7; i++ {
		writeFakeHome(t, root, fmt.Sprintf("slot-%02d", i), ProviderCodex)
	}
	store := NewMemoryLifecycleStore()
	config := testConfig(root, ProviderCodex, store)
	config.LowWatermark, config.HighWatermark, config.CriticalWatermark = 3, 5, 7
	catalog, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.HealthyCount() != 7 || snapshot.Watermark() != WatermarkCritical || !snapshot.HighWatermarkExceeded() {
		t.Fatalf("healthy=%d watermark=%q", snapshot.HealthyCount(), snapshot.Watermark())
	}
}

func TestDiscoveryUsesMetadataOnlyAndSchemaStorableUUIDRefs(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifact := writeFakeHome(t, root, "arbitrary.name", ProviderCodex)
	catalog, _ := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	ref := first.Entries()[0].HomeRef()
	if !uuidPattern.MatchString(ref) {
		t.Fatalf("home ref is not schema-storable UUID: %q", ref)
	}
	if err := os.Chmod(artifact, 0o000); err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if second.Entries()[0].HomeRef() != ref {
		t.Fatal("metadata-only identity changed when artifact became unreadable")
	}
}

func TestUnsafeCandidateMetadataIsQuarantined(t *testing.T) {
	root := newPrivateRoot(t)
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	catalog, _ := mustCatalog(t, root, ProviderCodex)
	snapshot, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.HealthyCount() != 0 || snapshot.QuarantinedCount() != 1 || snapshot.Quarantined()[0].Reason() != ReasonCandidateSymlink {
		t.Fatalf("unsafe candidate result = %#v", snapshot.Quarantined())
	}
	if !uuidPattern.MatchString(snapshot.Quarantined()[0].CandidateRef()) {
		t.Fatalf("candidate ref is not UUID: %q", snapshot.Quarantined()[0].CandidateRef())
	}
}

func TestDuplicateArtifactIdentityQuarantinesAllDeterministically(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifactA := writeFakeHome(t, root, "z-last", ProviderCodex)
	_, artifactB := writeFakeHome(t, root, "a-first", ProviderCodex)
	if err := os.Remove(artifactB); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(artifactA, artifactB); err != nil {
		t.Fatal(err)
	}
	catalog, _ := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	firstRefs, secondRefs := quarantineRefs(first), quarantineRefs(second)
	if first.HealthyCount() != 0 || len(firstRefs) != 2 || strings.Join(firstRefs, ",") != strings.Join(secondRefs, ",") || !sort.StringsAreSorted(firstRefs) {
		t.Fatalf("duplicate identity first=%v second=%v", firstRefs, secondRefs)
	}
}

func TestFailedFullScanDoesNotConsumeGeneration(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderCodex)
	catalog, store := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	away := root + ".away"
	if err := os.Rename(root, away); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Reconcile(); err == nil {
		t.Fatal("expected full scan failure")
	}
	if err := os.Rename(away, root); err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if first.Generation() != 1 || second.Generation() != 2 || len(store.Generations(testIdentity())) != 2 {
		t.Fatalf("failed scan consumed generation: %d -> %d", first.Generation(), second.Generation())
	}
}

func TestSnapshotAndStoreCopiesAreImmutable(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderKiro)
	catalog, _ := mustCatalog(t, root, ProviderKiro)
	snapshot, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	want := snapshot.Entries()[0].HomeRef()
	entries := snapshot.Entries()
	entries[0] = Entry{}
	current, ok := catalog.Current()
	if !ok || current.Entries()[0].HomeRef() != want || snapshot.Entries()[0].HomeRef() != want {
		t.Fatal("published snapshot mutated through accessor")
	}
}

func TestConcurrentAcquireReleaseRevalidateAndReconcile(t *testing.T) {
	root := newPrivateRoot(t)
	for i := 0; i < 4; i++ {
		writeFakeHome(t, root, fmt.Sprintf("race-%d", i), ProviderCodex)
	}
	catalog, _ := mustCatalog(t, root, ProviderCodex)
	snapshot, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, entry := range snapshot.Entries() {
		ref := entry.HomeRef()
		for worker := 0; worker < 3; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 12; i++ {
					release, acquireErr := catalog.AcquireRef(ref)
					if acquireErr == nil {
						_, _ = catalog.Revalidate(ref)
						release()
					}
					_, _ = catalog.NotifyHint("ignored-path")
				}
			}()
		}
	}
	wg.Wait()
	final, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if final.HealthyCount() != 4 {
		t.Fatalf("final healthy count = %d", final.HealthyCount())
	}
}

func TestLayoutAndOwnershipContracts(t *testing.T) {
	for _, provider := range []Provider{ProviderAntigravity, ProviderCodex, ProviderKiro} {
		layout, ok := LayoutFor(provider)
		if !ok || validateLayout(layout) != nil {
			t.Fatalf("invalid layout for %q", provider)
		}
	}
	root := newPrivateRoot(t)
	info, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := ownedByEffectiveUser(info)
	if err != nil || !owned {
		t.Fatalf("owned=%v err=%v", owned, err)
	}
	stat := info.Sys().(*syscall.Stat_t)
	if stat.Uid != uint32(os.Geteuid()) {
		t.Fatalf("fixture uid=%d effective uid=%d", stat.Uid, os.Geteuid())
	}
}

func quarantineRefs(snapshot Snapshot) []string {
	refs := make([]string, 0, snapshot.QuarantinedCount())
	for _, quarantine := range snapshot.Quarantined() {
		refs = append(refs, quarantine.CandidateRef())
	}
	return refs
}
