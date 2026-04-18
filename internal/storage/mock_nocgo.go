// +build nocgo

package storage

import (
	"errors"
)

// MockStore provides a simple in-memory store for CGO-disabled builds
type MockStore struct {
	data map[string][]byte
}

// NewMockStore creates a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string][]byte),
	}
}

// Get retrieves a value from the mock store
func (s *MockStore) Get(key string) ([]byte, error) {
	if value, exists := s.data[key]; exists {
		return value, nil
	}
	return nil, errors.New("key not found")
}

// Set stores a value in the mock store
func (s *MockStore) Set(key string, value []byte) error {
	s.data[key] = value
	return nil
}

// Delete removes a value from the mock store
func (s *MockStore) Delete(key string) error {
	delete(s.data, key)
	return nil
}
