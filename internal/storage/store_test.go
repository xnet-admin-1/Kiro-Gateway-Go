package storage

import (
	"errors"
	"testing"
)

// mockStore is a simple in-memory implementation for testing
type mockStore struct {
	data map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string][]byte),
	}
}

func (m *mockStore) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	
	value, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	
	return value, nil
}

func (m *mockStore) Set(key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	m.data[key] = value
	return nil
}

func (m *mockStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	if _, ok := m.data[key]; !ok {
		return ErrNotFound
	}
	
	delete(m.data, key)
	return nil
}

func (m *mockStore) List() ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// TestStoreInterface verifies the Store interface contract
func TestStoreInterface(t *testing.T) {
	var _ Store = (*mockStore)(nil)
}

// TestStoreBasicOperations tests basic store operations
func TestStoreBasicOperations(t *testing.T) {
	store := newMockStore()
	
	// Test Set
	err := store.Set("test-key", []byte("test-value"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Test Get
	value, err := store.Get("test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	
	if string(value) != "test-value" {
		t.Errorf("Get() = %s, want test-value", string(value))
	}
	
	// Test Delete
	err = store.Delete("test-key")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	
	// Verify deleted
	_, err = store.Get("test-key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

// TestStoreInvalidKey tests invalid key handling
func TestStoreInvalidKey(t *testing.T) {
	store := newMockStore()
	
	tests := []struct {
		name string
		op   func() error
	}{
		{
			name: "Get with empty key",
			op: func() error {
				_, err := store.Get("")
				return err
			},
		},
		{
			name: "Set with empty key",
			op: func() error {
				return store.Set("", []byte("value"))
			},
		},
		{
			name: "Delete with empty key",
			op: func() error {
				return store.Delete("")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if !errors.Is(err, ErrInvalidKey) {
				t.Errorf("error = %v, want ErrInvalidKey", err)
			}
		})
	}
}

// TestStoreNotFound tests not found error handling
func TestStoreNotFound(t *testing.T) {
	store := newMockStore()
	
	tests := []struct {
		name string
		op   func() error
	}{
		{
			name: "Get non-existent key",
			op: func() error {
				_, err := store.Get("non-existent")
				return err
			},
		},
		{
			name: "Delete non-existent key",
			op: func() error {
				return store.Delete("non-existent")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
		})
	}
}

// TestStoreList tests listing keys
func TestStoreList(t *testing.T) {
	store := newMockStore()
	
	// Empty store
	keys, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	
	if len(keys) != 0 {
		t.Errorf("List() on empty store = %d keys, want 0", len(keys))
	}
	
	// Add some keys
	testKeys := []string{"key1", "key2", "key3"}
	for _, key := range testKeys {
		if err := store.Set(key, []byte("value")); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}
	
	// List keys
	keys, err = store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	
	if len(keys) != len(testKeys) {
		t.Errorf("List() = %d keys, want %d", len(keys), len(testKeys))
	}
	
	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	
	for _, key := range testKeys {
		if !keyMap[key] {
			t.Errorf("List() missing key %s", key)
		}
	}
}

// TestStoreUpdate tests updating existing values
func TestStoreUpdate(t *testing.T) {
	store := newMockStore()
	
	// Set initial value
	err := store.Set("test-key", []byte("initial-value"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Update value
	err = store.Set("test-key", []byte("updated-value"))
	if err != nil {
		t.Fatalf("Set() update error = %v", err)
	}
	
	// Verify updated value
	value, err := store.Get("test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	
	if string(value) != "updated-value" {
		t.Errorf("Get() after update = %s, want updated-value", string(value))
	}
}

// TestStoreBinaryData tests storing binary data
func TestStoreBinaryData(t *testing.T) {
	store := newMockStore()
	
	// Test with binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	
	err := store.Set("binary-key", binaryData)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	value, err := store.Get("binary-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	
	if len(value) != len(binaryData) {
		t.Errorf("Get() length = %d, want %d", len(value), len(binaryData))
	}
	
	for i, b := range value {
		if b != binaryData[i] {
			t.Errorf("Get() byte[%d] = %x, want %x", i, b, binaryData[i])
		}
	}
}
