//go:build !windows

package credentialcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
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

func mustCatalog(t *testing.T, root string, provider Provider) *Catalog {
	t.Helper()
	catalog, err := New(Config{Root: root, Provider: provider})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestLayoutForMatchesDaemonProviderContracts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		provider Provider
		home     string
		artifact string
	}{
		{ProviderAntigravity, "home", ".gemini/antigravity-cli/antigravity-oauth-token"},
		{ProviderCodex, "codex", "auth.json"},
		{ProviderKiro, "xdg-data", "kiro-cli/data.sqlite3"},
	}
	for _, test := range tests {
		layout, ok := LayoutFor(test.provider)
		if !ok || layout.HomeRelative() != test.home || layout.ArtifactRelative() != test.artifact {
			t.Fatalf("layout %q = (%q, %q, %v)", test.provider, layout.HomeRelative(), layout.ArtifactRelative(), ok)
		}
		if err := validateLayout(layout); err != nil {
			t.Fatalf("layout %q is unsafe: %v", test.provider, err)
		}
	}
}

func TestReconcileDiscoversArbitraryImmediateChildren(t *testing.T) {
	root := newPrivateRoot(t)
	children := []string{"account-blue", "customer.with.dots", "slot-999999", "z"}
	for _, child := range children {
		writeFakeHome(t, root, child, ProviderCodex)
	}

	snapshot, err := mustCatalog(t, root, ProviderCodex).Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation() != 1 || snapshot.HealthyCount() != len(children) || snapshot.QuarantinedCount() != 0 {
		t.Fatalf("snapshot generation=%d healthy=%d quarantined=%d", snapshot.Generation(), snapshot.HealthyCount(), snapshot.QuarantinedCount())
	}
	for _, entry := range snapshot.Entries() {
		if !strings.HasPrefix(entry.HomeRef(), "home_") || entry.Provider() != ProviderCodex || entry.State() != StateHealthy {
			t.Fatalf("unexpected entry: ref=%q provider=%q state=%q", entry.HomeRef(), entry.Provider(), entry.State())
		}
		if filepath.Base(entry.ArtifactPath()) != "auth.json" {
			t.Fatalf("artifact = %q", entry.ArtifactPath())
		}
	}
}

func TestReconcileDerivesEachProviderArtifact(t *testing.T) {
	for _, provider := range []Provider{ProviderAntigravity, ProviderCodex, ProviderKiro} {
		t.Run(string(provider), func(t *testing.T) {
			root := newPrivateRoot(t)
			home, artifact := writeFakeHome(t, root, "any-child-name", provider)
			snapshot, err := mustCatalog(t, root, provider).Reconcile()
			if err != nil {
				t.Fatal(err)
			}
			entries := snapshot.Entries()
			if len(entries) != 1 || entries[0].HomePath() != home || entries[0].ArtifactPath() != artifact {
				t.Fatalf("entries = %#v, want home=%q artifact=%q", entries, home, artifact)
			}
		})
	}
}

func TestReconcileRejectsUnsafeCandidateMetadata(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, root string)
		reason QuarantineReason
	}{
		{
			name: "child_symlink",
			setup: func(t *testing.T, root string) {
				outside := filepath.Join(t.TempDir(), "outside")
				if err := os.Mkdir(outside, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(root, "alias")); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonCandidateSymlink,
		},
		{
			name: "child_not_directory",
			setup: func(t *testing.T, root string) {
				if err := os.WriteFile(filepath.Join(root, "plain"), []byte("fixture"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonCandidateNotDirectory,
		},
		{
			name: "child_permissions",
			setup: func(t *testing.T, root string) {
				writeFakeHome(t, root, "open", ProviderCodex)
				if err := os.Chmod(filepath.Join(root, "open"), 0o750); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonPermissionsTooOpen,
		},
		{
			name: "missing_artifact",
			setup: func(t *testing.T, root string) {
				if err := os.MkdirAll(filepath.Join(root, "missing", "codex"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonLayoutMissing,
		},
		{
			name: "artifact_symlink",
			setup: func(t *testing.T, root string) {
				_, artifact := writeFakeHome(t, root, "linked", ProviderCodex)
				if err := os.Remove(artifact); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(t.TempDir(), "target")
				if err := os.WriteFile(target, []byte("fixture"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, artifact); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonLayoutSymlink,
		},
		{
			name: "artifact_nonregular",
			setup: func(t *testing.T, root string) {
				_, artifact := writeFakeHome(t, root, "fifo", ProviderCodex)
				if err := os.Remove(artifact); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(artifact, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonLayoutInvalidType,
		},
		{
			name: "artifact_permissions",
			setup: func(t *testing.T, root string) {
				_, artifact := writeFakeHome(t, root, "open-file", ProviderCodex)
				if err := os.Chmod(artifact, 0o640); err != nil {
					t.Fatal(err)
				}
			},
			reason: ReasonPermissionsTooOpen,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newPrivateRoot(t)
			test.setup(t, root)
			snapshot, err := mustCatalog(t, root, ProviderCodex).Reconcile()
			if err != nil {
				t.Fatal(err)
			}
			quarantined := snapshot.Quarantined()
			if snapshot.HealthyCount() != 0 || len(quarantined) != 1 || quarantined[0].Reason() != test.reason {
				t.Fatalf("healthy=%d quarantine=%#v", snapshot.HealthyCount(), quarantined)
			}
			if !strings.HasPrefix(quarantined[0].CandidateRef(), "candidate_") {
				t.Fatalf("candidate ref = %q", quarantined[0].CandidateRef())
			}
		})
	}
}

func TestNewRejectsUncontrolledRoots(t *testing.T) {
	root := newPrivateRoot(t)
	if err := os.Chmod(root, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{Root: root, Provider: ProviderCodex}); err == nil {
		t.Fatal("expected permissive root rejection")
	}

	physical := newPrivateRoot(t)
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(physical, link); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{Root: link, Provider: ProviderCodex}); err == nil {
		t.Fatal("expected symlink root rejection")
	}

	if _, err := New(Config{Root: "relative", Provider: ProviderCodex}); err == nil {
		t.Fatal("expected relative root rejection")
	}
}

func TestOwnerCheckUsesEffectiveUID(t *testing.T) {
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

func TestOpaqueRefUsesIdentityNotNameOrContents(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifact := writeFakeHome(t, root, "first-name", ProviderCodex)
	catalog := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	firstRef := first.Entries()[0].HomeRef()

	// Mode 000 makes the synthetic artifact unreadable to this process. A
	// metadata-only scan still succeeds because it never opens the artifact.
	if err := os.Chmod(artifact, 0o000); err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if second.Entries()[0].HomeRef() != firstRef {
		t.Fatal("reference changed when artifact mode changed")
	}
	if err := os.Chmod(artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("different synthetic fixture content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "first-name"), filepath.Join(root, "renamed-arbitrarily")); err != nil {
		t.Fatal(err)
	}
	third, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if third.Entries()[0].HomeRef() != firstRef {
		t.Fatal("reference changed after content update or child rename")
	}
	if first.Generation() != 1 || second.Generation() != 2 || third.Generation() != 3 {
		t.Fatalf("generations = %d, %d, %d", first.Generation(), second.Generation(), third.Generation())
	}
}

func TestDuplicateArtifactIdentityQuarantinesEveryCandidateDeterministically(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifactA := writeFakeHome(t, root, "z-last", ProviderCodex)
	_, artifactB := writeFakeHome(t, root, "a-first", ProviderCodex)
	if err := os.Remove(artifactB); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(artifactA, artifactB); err != nil {
		t.Fatal(err)
	}
	catalog := mustCatalog(t, root, ProviderCodex)

	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if first.HealthyCount() != 0 || first.QuarantinedCount() != 2 {
		t.Fatalf("healthy=%d quarantined=%d", first.HealthyCount(), first.QuarantinedCount())
	}
	firstRefs := quarantineRefs(first)
	secondRefs := quarantineRefs(second)
	if !reflect.DeepEqual(firstRefs, secondRefs) || !sort.StringsAreSorted(firstRefs) {
		t.Fatalf("non-deterministic refs: first=%v second=%v", firstRefs, secondRefs)
	}
	for _, quarantine := range first.Quarantined() {
		if quarantine.Reason() != ReasonFilesystemIdentityConflict || quarantine.State() != StateQuarantined {
			t.Fatalf("quarantine = %#v", quarantine)
		}
	}
}

func TestActiveRefCountingAndDrainingLifecycle(t *testing.T) {
	root := newPrivateRoot(t)
	_, _ = writeFakeHome(t, root, "active-slot", ProviderCodex)
	catalog := mustCatalog(t, root, ProviderCodex)

	snap1, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snap1.HealthyCount() != 1 {
		t.Fatalf("healthy count = %d", snap1.HealthyCount())
	}
	homeRef := snap1.Entries()[0].HomeRef()

	// Acquire active ref
	release, err := catalog.AcquireRef(homeRef)
	if err != nil {
		t.Fatalf("AcquireRef failed: %v", err)
	}
	if catalog.ActiveRefs(homeRef) != 1 {
		t.Fatalf("active refs = %d, want 1", catalog.ActiveRefs(homeRef))
	}

	// Remove physical home directory while active ref is held -> transitions to Draining
	if err := os.RemoveAll(filepath.Join(root, "active-slot")); err != nil {
		t.Fatal(err)
	}

	snap2, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snap2.HealthyCount() != 0 || snap2.DrainingCount() != 1 {
		t.Fatalf("snap2 healthy=%d draining=%d", snap2.HealthyCount(), snap2.DrainingCount())
	}
	if snap2.Draining()[0].HomeRef() != homeRef {
		t.Fatalf("draining ref = %q, want %q", snap2.Draining()[0].HomeRef(), homeRef)
	}

	// Release active ref -> transitions to Tombstoned
	release()
	if catalog.ActiveRefs(homeRef) != 0 {
		t.Fatalf("active refs after release = %d, want 0", catalog.ActiveRefs(homeRef))
	}

	snap3, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snap3.HealthyCount() != 0 || snap3.DrainingCount() != 0 || snap3.TombstonedCount() != 1 {
		t.Fatalf("snap3 healthy=%d draining=%d tombstoned=%d", snap3.HealthyCount(), snap3.DrainingCount(), snap3.TombstonedCount())
	}

	// Attempting AcquireRef on tombstoned ref must fail
	if _, err := catalog.AcquireRef(homeRef); err != ErrHomeTombstoned {
		t.Fatalf("acquire tombstoned err = %v, want ErrHomeTombstoned", err)
	}
}

func TestDurableTombstoneNameNonReuse(t *testing.T) {
	root := newPrivateRoot(t)
	_, _ = writeFakeHome(t, root, "reused-slot", ProviderCodex)
	catalog := mustCatalog(t, root, ProviderCodex)

	snap1, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snap1.HealthyCount() != 1 {
		t.Fatalf("snap1 healthy count = %d, want 1", snap1.HealthyCount())
	}

	// Remove physical candidate directory
	if err := os.RemoveAll(filepath.Join(root, "reused-slot")); err != nil {
		t.Fatal(err)
	}

	snap2, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if snap2.HealthyCount() != 0 || snap2.TombstonedCount() != 1 {
		t.Fatalf("snap2 healthy=%d tombstoned=%d", snap2.HealthyCount(), snap2.TombstonedCount())
	}

	// Re-create a directory with the exact same name "reused-slot"
	_, _ = writeFakeHome(t, root, "reused-slot", ProviderCodex)

	snap3, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}

	// Re-created directory MUST be quarantined as Tombstoned, NEVER healthy!
	if snap3.HealthyCount() != 0 || snap3.QuarantinedCount() != 1 {
		t.Fatalf("snap3 healthy=%d quarantined=%d", snap3.HealthyCount(), snap3.QuarantinedCount())
	}
	if snap3.Quarantined()[0].Reason() != ReasonTombstoned {
		t.Fatalf("quarantine reason = %q, want ReasonTombstoned", snap3.Quarantined()[0].Reason())
	}
}

func TestFastLaunchRevalidation(t *testing.T) {
	root := newPrivateRoot(t)
	_, artifact := writeFakeHome(t, root, "revalidate-slot", ProviderAntigravity)
	catalog := mustCatalog(t, root, ProviderAntigravity)

	snap, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	homeRef := snap.Entries()[0].HomeRef()

	// Launch-time revalidation on healthy home passes fast
	entry, err := catalog.Revalidate(homeRef)
	if err != nil {
		t.Fatalf("Revalidate healthy failed: %v", err)
	}
	if entry.HomeRef() != homeRef || entry.State() != StateHealthy {
		t.Fatalf("entry = %#v", entry)
	}

	// Corrupt permissions on artifact -> launch revalidation fails and quarantines
	if err := os.Chmod(artifact, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Revalidate(homeRef); err != ErrRevalidateFailed {
		t.Fatalf("revalidate corrupted err = %v, want ErrRevalidateFailed", err)
	}
}

func TestNotifyHintTriggersReconciliation(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "slot-initial", ProviderKiro)
	catalog := mustCatalog(t, root, ProviderKiro)

	if _, err := catalog.Reconcile(); err != nil {
		t.Fatal(err)
	}

	// Add new directory physically
	writeFakeHome(t, root, "slot-hinted", ProviderKiro)

	// NotifyHint must trigger reconciliation immediately
	snap, err := catalog.NotifyHint("slot-hinted")
	if err != nil {
		t.Fatal(err)
	}
	if snap.HealthyCount() != 2 {
		t.Fatalf("healthy count after hint = %d, want 2", snap.HealthyCount())
	}
}

func TestWatcherOverflowRecovery(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "slot-1", ProviderKiro)
	catalog := mustCatalog(t, root, ProviderKiro)

	if _, err := catalog.Reconcile(); err != nil {
		t.Fatal(err)
	}

	// Add new home physical dir out-of-band
	writeFakeHome(t, root, "slot-2", ProviderKiro)

	// Trigger watcher overflow recovery
	snap, err := catalog.NotifyWatcherOverflow()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.WatcherOverflowCount() != 1 {
		t.Fatalf("overflow count = %d, want 1", catalog.WatcherOverflowCount())
	}
	if snap.HealthyCount() != 2 {
		t.Fatalf("healthy count after overflow recovery = %d, want 2", snap.HealthyCount())
	}
}

func TestWatermarkEnforcementQuarantinesExcessNoSilentDrop(t *testing.T) {
	root := newPrivateRoot(t)
	for i := 0; i < 25; i++ {
		writeFakeHome(t, root, fmt.Sprintf("child-%03d", i), ProviderCodex)
	}

	// Config with HighWatermark=20, LowWatermark=15
	catalog, err := New(Config{
		Root:          root,
		Provider:      ProviderCodex,
		HighWatermark: 20,
		LowWatermark:  15,
	})
	if err != nil {
		t.Fatal(err)
	}

	snap, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}

	if snap.HealthyCount() != 15 {
		t.Fatalf("healthy count after watermark enforcement = %d, want 15", snap.HealthyCount())
	}

	// Excess 10 entries MUST be explicitly recorded in Quarantined with ReasonWatermarkExceeded
	if snap.QuarantinedCount() != 10 {
		t.Fatalf("quarantined count = %d, want 10", snap.QuarantinedCount())
	}
	for _, q := range snap.Quarantined() {
		if q.Reason() != ReasonWatermarkExceeded {
			t.Fatalf("quarantine reason = %q, want ReasonWatermarkExceeded", q.Reason())
		}
	}
}

func TestConcurrentAcquireReleaseRevalidateUnderRace(t *testing.T) {
	root := newPrivateRoot(t)
	for i := 0; i < 5; i++ {
		writeFakeHome(t, root, fmt.Sprintf("race-slot-%d", i), ProviderCodex)
	}

	catalog := mustCatalog(t, root, ProviderCodex)
	snap, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, entry := range snap.Entries() {
		ref := entry.HomeRef()
		for g := 0; g < 5; g++ {
			wg.Add(1)
			go func(r string) {
				defer wg.Done()
				for i := 0; i < 20; i++ {
					rel, err := catalog.AcquireRef(r)
					if err == nil {
						_, _ = catalog.Revalidate(r)
						time.Sleep(100 * time.Microsecond)
						rel()
					}
					_, _ = catalog.NotifyHint("synthetic")
				}
			}(ref)
		}
	}
	wg.Wait()

	finalSnap, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if finalSnap.HealthyCount() != 5 {
		t.Fatalf("final healthy count = %d, want 5", finalSnap.HealthyCount())
	}
}

func quarantineRefs(snapshot Snapshot) []string {
	refs := make([]string, 0, snapshot.QuarantinedCount())
	for _, quarantine := range snapshot.Quarantined() {
		refs = append(refs, quarantine.CandidateRef())
	}
	return refs
}

func TestSnapshotAccessorsCannotMutatePublishedGeneration(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderKiro)
	catalog := mustCatalog(t, root, ProviderKiro)
	snapshot, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	wantRef := snapshot.Entries()[0].HomeRef()

	entries := snapshot.Entries()
	entries[0] = Entry{}
	current, ok := catalog.Current()
	if !ok || current.Entries()[0].HomeRef() != wantRef || snapshot.Entries()[0].HomeRef() != wantRef {
		t.Fatal("published snapshot was mutated through accessor result")
	}
}

func TestFailedFullScanDoesNotConsumeGeneration(t *testing.T) {
	root := newPrivateRoot(t)
	writeFakeHome(t, root, "one", ProviderCodex)
	catalog := mustCatalog(t, root, ProviderCodex)
	first, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}

	away := root + ".away"
	if err := os.Rename(root, away); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Reconcile(); err == nil {
		t.Fatal("expected missing controlled root to abort scan")
	}
	if err := os.Rename(away, root); err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if first.Generation() != 1 || second.Generation() != 2 {
		t.Fatalf("failed scan consumed generation: first=%d second=%d", first.Generation(), second.Generation())
	}
}

type syntheticOwnerInfo struct {
	os.FileInfo
	stat *syscall.Stat_t
}

func (info syntheticOwnerInfo) Sys() any { return info.stat }

func TestPrivateMetadataRejectsWrongOwner(t *testing.T) {
	wrongUID := uint32(os.Geteuid()) + 1
	info := syntheticOwnerInfo{stat: &syscall.Stat_t{Uid: wrongUID}}
	if reason := privateMetadataReason(info); reason != ReasonWrongOwner {
		t.Fatalf("reason = %q, want %q", reason, ReasonWrongOwner)
	}
}
