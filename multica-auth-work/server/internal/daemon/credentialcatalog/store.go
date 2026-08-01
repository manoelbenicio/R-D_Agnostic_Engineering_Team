package credentialcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LifecycleRecord represents one persistent lifecycle state entry (missing, draining, retired, tombstoned).
type LifecycleRecord struct {
	HomeRef    string    `json:"home_ref"`
	NameRef    string    `json:"name_ref"`
	Provider   Provider  `json:"provider"`
	State      State     `json:"state"`
	ActiveRefs int       `json:"active_refs"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// LifecycleStore defines the interface for durable lifecycle state persistence across
// catalog reconstructions and daemon restarts.
//
// DEPENDENCY NOTE FOR C2/K1 AUTHORIZATION RECONCILIATION:
// The production implementation of this interface in C2/K1 uses a SQL adapter
// backed by PostgreSQL. Until C2 registers the SQL adapter, Catalog uses
// FileLifecycleStore or MemoryLifecycleStore. If Config.Store is nil, New() fails
// closed with ErrMissingTombstoneStore.
type LifecycleStore interface {
	LoadLifecycle(ctx context.Context, provider Provider) ([]LifecycleRecord, error)
	SaveLifecycle(ctx context.Context, records []LifecycleRecord) error
}

// MemoryLifecycleStore provides an in-memory LifecycleStore implementation for testing.
type MemoryLifecycleStore struct {
	mu             sync.RWMutex
	records        map[string]LifecycleRecord
	injectFailSave bool
	injectFailLoad bool
}

func NewMemoryLifecycleStore() *MemoryLifecycleStore {
	return &MemoryLifecycleStore{
		records: make(map[string]LifecycleRecord),
	}
}

func (m *MemoryLifecycleStore) SetInjectFailSave(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.injectFailSave = fail
}

func (m *MemoryLifecycleStore) SetInjectFailLoad(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.injectFailLoad = fail
}

func (m *MemoryLifecycleStore) LoadLifecycle(ctx context.Context, provider Provider) ([]LifecycleRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.injectFailLoad {
		return nil, fmt.Errorf("injected store load failure")
	}
	var result []LifecycleRecord
	for _, r := range m.records {
		if r.Provider == provider {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MemoryLifecycleStore) SaveLifecycle(ctx context.Context, records []LifecycleRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.injectFailSave {
		return fmt.Errorf("injected store save failure")
	}
	for _, r := range records {
		m.records[r.HomeRef] = r
	}
	return nil
}

// FileLifecycleStore provides an atomic, crash-safe file-backed LifecycleStore for testing and non-prod persistence.
type FileLifecycleStore struct {
	mu   sync.RWMutex
	path string
}

func NewFileLifecycleStore(path, controlledRoot string) (*FileLifecycleStore, error) {
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("invalid store path: %w", err)
	}
	if controlledRoot != "" {
		absRoot, err := filepath.Abs(filepath.Clean(controlledRoot))
		if err == nil && pathWithin(absRoot, absPath) {
			return nil, fmt.Errorf("store path cannot be inside controlled root")
		}
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &FileLifecycleStore{path: absPath}, nil
}

func (f *FileLifecycleStore) LoadLifecycle(ctx context.Context, provider Provider) ([]LifecycleRecord, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store read failed: %w", err)
	}
	var all []LifecycleRecord
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("store unmarshal failed: %w", err)
	}
	var result []LifecycleRecord
	for _, r := range all {
		if r.Provider == provider {
			result = append(result, r)
		}
	}
	return result, nil
}

func (f *FileLifecycleStore) SaveLifecycle(ctx context.Context, records []LifecycleRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var existingMap = make(map[string]LifecycleRecord)
	data, err := os.ReadFile(f.path)
	if err == nil {
		var existing []LifecycleRecord
		if err := json.Unmarshal(data, &existing); err == nil {
			for _, r := range existing {
				existingMap[r.HomeRef] = r
			}
		}
	}

	for _, r := range records {
		existingMap[r.HomeRef] = r
	}

	allList := make([]LifecycleRecord, 0, len(existingMap))
	for _, r := range existingMap {
		allList = append(allList, r)
	}

	out, err := json.MarshalIndent(allList, "", "  ")
	if err != nil {
		return err
	}

	// Atomic crash-safe write with temp file + Sync + Rename + Dir Sync
	dir := filepath.Dir(f.path)
	tmpFile, err := os.CreateTemp(dir, "lifecycle-store-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if err := tmpFile.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmpFile.Write(out); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, f.path); err != nil {
		return err
	}

	if dirF, err := os.Open(dir); err == nil {
		_ = dirF.Sync()
		_ = dirF.Close()
	}
	return nil
}
