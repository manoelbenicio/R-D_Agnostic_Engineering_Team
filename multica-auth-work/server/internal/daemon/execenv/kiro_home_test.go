package execenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareKiroHomePerAccountIsolatesDataStore(t *testing.T) {
	accountA := filepath.Join(t.TempDir(), "accountA")
	accountB := filepath.Join(t.TempDir(), "accountB")
	writeTestCredential(t, filepath.Join(accountA, kiroCredentialRelPath), "kiro-account-A")
	writeTestCredential(t, filepath.Join(accountB, kiroCredentialRelPath), "kiro-account-B")

	homeA := filepath.Join(t.TempDir(), "xdg-data-A")
	homeB := filepath.Join(t.TempDir(), "xdg-data-B")

	if err := prepareKiroHome(homeA, KiroHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("prepare kiro home A: %v", err)
	}
	if err := prepareKiroHome(homeB, KiroHomeOptions{AccountHome: accountB}, testLogger()); err != nil {
		t.Fatalf("prepare kiro home B: %v", err)
	}

	storeA := filepath.Join(homeA, kiroCredentialRelPath)
	storeB := filepath.Join(homeB, kiroCredentialRelPath)
	assertFileContent(t, storeA, "kiro-account-A")
	assertFileContent(t, storeB, "kiro-account-B")
	assertNotSymlink(t, storeA)
	assertNotSymlink(t, storeB)

	if err := os.WriteFile(storeA, []byte("kiro-account-A-refreshed"), 0o600); err != nil {
		t.Fatalf("simulate kiro refresh on A: %v", err)
	}
	assertFileContent(t, storeB, "kiro-account-B")
	assertFileContent(t, filepath.Join(accountA, kiroCredentialRelPath), "kiro-account-A")

	writeTestCredential(t, filepath.Join(accountA, kiroCredentialRelPath), "kiro-account-A-source-refresh")
	if err := prepareKiroHome(homeA, KiroHomeOptions{AccountHome: accountA}, testLogger()); err != nil {
		t.Fatalf("reuse kiro home A: %v", err)
	}
	assertFileContent(t, storeA, "kiro-account-A-source-refresh")
	assertFileContent(t, storeB, "kiro-account-B")
}

func TestPrepareKiroHomeFallbackWhenNoAccount(t *testing.T) {
	home := filepath.Join(t.TempDir(), "xdg-data")
	if err := prepareKiroHome(home, KiroHomeOptions{}, testLogger()); err != nil {
		t.Fatalf("prepare kiro fallback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, kiroCredentialRelPath)); !os.IsNotExist(err) {
		t.Fatalf("kiro fallback created credential copy, stat err = %v", err)
	}
}
