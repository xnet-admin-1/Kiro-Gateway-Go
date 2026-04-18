package storage

import (
	"errors"
	"sync"
)

var (
    ErrInvalidKey = errors.New("invalid key")
    ErrNotFound   = errors.New("key not found")
)

type Store interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte) error
    Delete(key string) error
}

// MemoryStore implements Store interface using in-memory storage
type MemoryStore struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string][]byte),
	}
}

// Get retrieves a value by key
func (m *MemoryStore) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return nil, ErrNotFound
	}

	// Return a copy to prevent external modification
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Set stores a value by key
func (m *MemoryStore) Set(key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy to prevent external modification
	stored := make([]byte, len(value))
	copy(stored, value)
	m.data[key] = stored

	return nil
}

// Delete removes a key-value pair
func (m *MemoryStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}
