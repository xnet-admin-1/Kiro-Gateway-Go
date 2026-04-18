package storage

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
)

// FileBackedMemoryStore keeps data in memory, persists to a JSON file on writes, loads on startup.
type FileBackedMemoryStore struct {
	data map[string]string // base64-encoded values
	mu   sync.RWMutex
	path string
}

// NewFileBackedMemoryStore creates a store backed by the given JSON file path.
func NewFileBackedMemoryStore(path string) (*FileBackedMemoryStore, error) {
	s := &FileBackedMemoryStore{
		data: make(map[string]string),
		path: path,
	}
	// Load existing data if file exists
	if raw, err := os.ReadFile(path); err == nil {
		json.Unmarshal(raw, &s.data) // ignore errors on corrupt file, start fresh
	}
	return s, nil
}

func (s *FileBackedMemoryStore) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	return base64.StdEncoding.DecodeString(v)
}

func (s *FileBackedMemoryStore) Set(key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = base64.StdEncoding.EncodeToString(value)
	return s.flush()
}

func (s *FileBackedMemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return s.flush()
}

func (s *FileBackedMemoryStore) flush() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0600)
}
