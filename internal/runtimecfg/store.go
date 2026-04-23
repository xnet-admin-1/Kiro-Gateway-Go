package runtimecfg

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Entry is a single config key-value with metadata.
type Entry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// Store holds runtime configuration that can be modified via API and persists to disk.
type Store struct {
	mu   sync.RWMutex
	data map[string]*Entry
	path string
}

// New creates a Store backed by the given JSON file.
func New(path string) (*Store, error) {
	s := &Store{data: make(map[string]*Entry), path: path}
	if raw, err := os.ReadFile(path); err == nil {
		json.Unmarshal(raw, &s.data)
	}
	return s, nil
}

func (s *Store) Get(key string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	return e, ok
}

// GetString returns the string value for a key, or "" if not found.
func (s *Store) GetString(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok {
		return ""
	}
	if str, ok := e.Value.(string); ok {
		return str
	}
	return ""
}

func (s *Store) List() map[string]*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]*Entry, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}

func (s *Store) Set(key string, value interface{}) *Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := &Entry{Key: key, Value: value, UpdatedAt: time.Now()}
	s.data[key] = e
	s.flush()
	return e
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return false
	}
	delete(s.data, key)
	s.flush()
	return true
}

func (s *Store) flush() {
	raw, _ := json.MarshalIndent(s.data, "", "  ")
	os.MkdirAll(s.path[:max(0, len(s.path)-len("/runtime-config.json"))], 0700)
	os.WriteFile(s.path, raw, 0600)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
