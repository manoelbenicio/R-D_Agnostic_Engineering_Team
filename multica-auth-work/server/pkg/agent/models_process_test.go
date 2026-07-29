//go:build !windows

package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestDiscoverACPModelsClosesStdinAndWaitsGracefully(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "stdin-closed")
	fake := filepath.Join(dir, "fake-acp")
	writeTestExecutable(t, fake, []byte(`#!/bin/sh
marker=$1
IFS= read -r initialize
IFS= read -r session_new
printf '%s\n' '{"jsonrpc":"2.0","id":2,"result":{"models":{"availableModels":[{"modelId":"synthetic/model","name":"Synthetic"}],"currentModelId":"synthetic/model"}}}'
while IFS= read -r ignored; do :; done
printf '%s\n' closed > "$marker"
`))

	models, err := discoverACPModels(context.Background(), fake, acpDiscoveryProvider{
		clientName:   "synthetic-test",
		acpArgs:      []string{marker},
		tmpdirPrefix: "synthetic-acp-grace-",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "synthetic/model" {
		t.Fatalf("models = %+v", models)
	}
	if got, err := os.ReadFile(marker); err != nil || strings.TrimSpace(string(got)) != "closed" {
		t.Fatalf("ACP child did not observe stdin EOF before cleanup: marker=%q err=%v", got, err)
	}
}

func TestDiscoverACPModelsReapsOrphanChild(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child-pid")
	fake := filepath.Join(dir, "fake-acp")
	writeTestExecutable(t, fake, []byte(`#!/bin/sh
pid_file=$1
sleep 30 &
printf '%s\n' "$!" > "$pid_file"
IFS= read -r initialize
IFS= read -r session_new
printf '%s\n' '{"jsonrpc":"2.0","id":2,"result":{"models":{"availableModels":[{"modelId":"synthetic/model","name":"Synthetic"}]}}}'
exit 0
`))

	models, err := discoverACPModels(context.Background(), fake, acpDiscoveryProvider{
		clientName:   "synthetic-test",
		acpArgs:      []string{pidFile},
		tmpdirPrefix: "synthetic-acp-orphan-",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("models = %+v", models)
	}
	assertSyntheticProcessGone(t, readSyntheticPID(t, pidFile))
}

func TestDiscoverACPModelsTimeoutReapsProcessTree(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child-pid")
	fake := filepath.Join(dir, "fake-acp")
	writeTestExecutable(t, fake, []byte(`#!/bin/sh
pid_file=$1
sleep 30 &
printf '%s\n' "$!" > "$pid_file"
while :; do sleep 1; done
`))

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	models, err := discoverACPModels(ctx, fake, acpDiscoveryProvider{
		clientName:   "synthetic-test",
		acpArgs:      []string{pidFile},
		tmpdirPrefix: "synthetic-acp-timeout-",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 0 {
		t.Fatalf("timeout models = %+v, want empty", models)
	}
	assertSyntheticProcessGone(t, readSyntheticPID(t, pidFile))
}

func readSyntheticPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if convErr != nil || pid <= 0 {
				t.Fatalf("invalid synthetic pid %q: %v", data, convErr)
			}
			return pid
		}
		if time.Now().After(deadline) {
			t.Fatalf("synthetic pid file was not created: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertSyntheticProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if err != nil {
			t.Fatalf("checking synthetic child %d: %v", pid, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("synthetic child %d survived ACP cleanup", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
