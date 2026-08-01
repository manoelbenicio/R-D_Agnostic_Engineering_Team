package credentialcatalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TombstoneRecord represents one persistent tombstone entry for a deleted home.
type TombstoneRecord struct {
	HomeRef      string    `json:"home_ref"`
	NameRef      string    `json:"name_ref"`
	Provider     Provider  `json:"provider"`
	TombstonedAt time.Time `json:"tombstoned_at"`
}

// TombstoneStore defines the interface for durable tombstone persistence across
// catalog reconstructions and daemon restarts.
//
// DEPENDENCY NOTE FOR C2/K1 AUTHORIZATION RECONCILIATION:
// The production implementation of this interface in C2/K1 uses a SQL adapter
// backed by PostgreSQL. Until C2 registers the SQL adapter, Catalog uses
// FileTombstoneStore or MemoryTombstoneStore, fail-closing if Store is nil in
// production initialization mode.
type TombstoneStore interface {
	LoadTombstones(ctx context.Context, provider Provider) ([]TombstoneRecord, error)
	SaveTombstone(ctx context.Context, record TombstoneRecord) error
}

// MemoryTombstoneStore provides an in-memory TombstoneStore implementation for testing.
type MemoryTombstoneStore struct {
	mu      sync.RWMutex
	records []TombstoneRecord
}

func NewMemoryTombstoneStore() *MemoryTombstoneStore {
	return &MemoryTombstoneStore{
		records: make([]TombstoneRecord, 0),
	}
}

func (m *MemoryTombstoneStore) LoadTombstones(ctx context.Context, provider Provider) ([]TombstoneRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []TombstoneRecord
	for _, r := range m.records {
		if r.Provider == provider {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MemoryTombstoneStore) SaveTombstone(ctx context.Context, record TombstoneRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, record)
	return nil
}

// FileTombstoneStore provides a durable file-backed TombstoneStore for daemon persistence across restarts.
type FileTombstoneStore struct {
	mu   sync.RWMutex
	path string
}

func NewFileTombstoneStore(path string) (*FileTombstoneStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &FileTombstoneStore{path: path}, nil
}

func (f *FileTombstoneStore) LoadTombstones(ctx context.Context, provider Provider) ([]TombstoneRecord, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var all []TombstoneRecord
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	var result []TombstoneRecord
	for _, r := range all {
		if r.Provider == provider {
			result = append(result, r)
		}
	}
	return result, nil
}

func (f *FileTombstoneStore) SaveTombstone(ctx context.Context, record TombstoneRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var all []TombstoneRecord
	data, err := os.ReadFile(f.path)
	if err == nil {
		_ = json.Unmarshal(data, &all)
	}
	all = append(all, record)
	out, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, out, 0o600)
}
