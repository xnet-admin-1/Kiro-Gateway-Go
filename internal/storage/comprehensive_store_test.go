package storage

import (
	"errors"
	"testing"
)

// TestStoreOperations_Comprehensive provides comprehensive table-driven tests for store operations
func TestStoreOperations_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		operations  []func(*mockStore) error
		wantErr     bool
		description string
	}{
		{
			name: "set and get basic operation",
			operations: []func(*mockStore) error{
				func(s *mockStore) error {
					return s.Set("key1", []byte("value1"))
				},
				func(s *mockStore) error {
					val, err := s.Get("key1")
					if err != nil {
						return err
					}
					if string(val) != "value1" {
						return errors.New("value mismatch")
					}
					return nil
				},
			},
			wantErr:     false,
			description: "Basic set and get should work",
		},
		{
			name: "set, update, and get",
			operations: []func(*mockStore) error{
				func(s *mockStore) error {
					return s.Set("key1", []byte("initial"))
				},
				func(s *mockStore) error {
					return s.Set("key1", []byte("updated"))
				},
				func(s *mockStore) error {
					val, err := s.Get("key1")
					if err != nil {
						return err
					}
					if string(val) != "updated" {
						return errors.New("value not updated")
					}
					return nil
				},
			},
			wantErr:     false,
			description: "Update operation should overwrite existing value",
		},
		{
			name: "set, delete, and get",
			operations: []func(*mockStore) error{
				func(s *mockStore) error {
					return s.Set("key1", []byte("value1"))
				},
				func(s *mockStore) error {
					return s.Delete("key1")
				},
				func(s *mockStore) error {
					_, err := s.Get("key1")
					if !errors.Is(err, ErrNotFound) {
						return errors.New("expected ErrNotFound")
					}
					return nil
				},
			},
			wantErr:     false,
			description: "Delete should remove key and subsequent get should fail",
		},
		{
			name: "multiple keys operations",
			operations: []func(*mockStore) error{
				func(s *mockStore) error {
					return s.Set("key1", []byte("value1"))
				},
				func(s *mockStore) error {
					return s.Set("key2", []byte("value2"))
				},
				func(s *mockStore) error {
					return s.Set("key3", []byte("value3"))
				},
				func(s *mockStore) error {
					keys, err := s.List()
					if err != nil {
						return err
					}
					if len(keys) != 3 {
						return errors.New("expected 3 keys")
					}
					return nil
				},
			},
			wantErr:     false,
			description: "Multiple keys should be stored independently",
		},
		{
			name: "empty key operations",
			operations: []func(*mockStore) error{
				func(s *mockStore) error {
					err := s.Set("", []byte("value"))
					if !errors.Is(err, ErrInvalidKey) {
						return errors.New("expected ErrInvalidKey for empty key set")
					}
					return nil
				},
				func(s *mockStore) error {
					_, err := s.Get("")
					if !errors.Is(err, ErrInvalidKey) {
						return errors.New("expected ErrInvalidKey for empty key get")
					}
					return nil
				},
				func(s *mockStore) error {
					err := s.Delete("")
					if !errors.Is(err, ErrInvalidKey) {
						return errors.New("expected ErrInvalidKey for empty key delete")
					}
					return nil
				},
			},
			wantErr:     false,
			description: "Empty keys should return ErrInvalidKey",
		},
		{
			name: "non-existent key operations",
			operations: []func(*mockStore) error{
				func(s *mockStore) error {
					_, err := s.Get("nonexistent")
					if !errors.Is(err, ErrNotFound) {
						return errors.New("expected ErrNotFound for nonexistent key")
					}
					return nil
				},
				func(s *mockStore) error {
					err := s.Delete("nonexistent")
					if !errors.Is(err, ErrNotFound) {
						return errors.New("expected ErrNotFound for nonexistent key delete")
					}
					return nil
				},
			},
			wantErr:     false,
			description: "Non-existent keys should return ErrNotFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			
			for i, op := range tt.operations {
				err := op(store)
				if (err != nil) != tt.wantErr {
					t.Errorf("operation %d error = %v, wantErr %v (%s)", 
						i, err, tt.wantErr, tt.description)
					return
				}
			}
		})
	}
}

// TestStoreBinaryData_Comprehensive provides comprehensive tests for binary data handling
func TestStoreBinaryData_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		data        []byte
		description string
	}{
		{
			name:        "empty binary data",
			key:         "empty",
			data:        []byte{},
			description: "Empty byte slice should be stored and retrieved",
		},
		{
			name:        "single byte",
			key:         "single",
			data:        []byte{0xFF},
			description: "Single byte should be preserved",
		},
		{
			name:        "null bytes",
			key:         "nulls",
			data:        []byte{0x00, 0x00, 0x00},
			description: "Null bytes should be preserved",
		},
		{
			name:        "mixed binary data",
			key:         "mixed",
			data:        []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			description: "Mixed binary data should be preserved exactly",
		},
		{
			name:        "large binary data",
			key:         "large",
			data:        make([]byte, 1024),
			description: "Large binary data should be handled",
		},
		{
			name:        "utf8 encoded text as binary",
			key:         "utf8",
			data:        []byte("Hello, 世界! 🌍"),
			description: "UTF-8 text stored as binary should be preserved",
		},
		{
			name:        "json data as binary",
			key:         "json",
			data:        []byte(`{"key": "value", "number": 42, "array": [1,2,3]}`),
			description: "JSON data stored as binary should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			
			// Set the data
			err := store.Set(tt.key, tt.data)
			if err != nil {
				t.Fatalf("Set() error = %v (%s)", err, tt.description)
			}
			
			// Get the data back
			retrieved, err := store.Get(tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v (%s)", err, tt.description)
			}
			
			// Compare lengths
			if len(retrieved) != len(tt.data) {
				t.Errorf("Retrieved data length = %d, want %d (%s)", 
					len(retrieved), len(tt.data), tt.description)
				return
			}
			
			// Compare byte by byte
			for i, b := range retrieved {
				if b != tt.data[i] {
					t.Errorf("Retrieved data byte[%d] = 0x%02X, want 0x%02X (%s)", 
						i, b, tt.data[i], tt.description)
				}
			}
		})
	}
}

// TestStoreKeyValidation_Comprehensive provides comprehensive tests for key validation
func TestStoreKeyValidation_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantErr     error
		description string
	}{
		{
			name:        "empty key",
			key:         "",
			wantErr:     ErrInvalidKey,
			description: "Empty key should be invalid",
		},
		{
			name:        "normal key",
			key:         "normal-key",
			wantErr:     nil,
			description: "Normal key should be valid",
		},
		{
			name:        "key with spaces",
			key:         "key with spaces",
			wantErr:     nil,
			description: "Key with spaces should be valid",
		},
		{
			name:        "key with special characters",
			key:         "key-with_special.chars@123",
			wantErr:     nil,
			description: "Key with special characters should be valid",
		},
		{
			name:        "unicode key",
			key:         "键名-clé-キー",
			wantErr:     nil,
			description: "Unicode key should be valid",
		},
		{
			name:        "very long key",
			key:         string(make([]byte, 1000)),
			wantErr:     nil,
			description: "Very long key should be valid",
		},
		{
			name:        "key with newlines",
			key:         "key\nwith\nnewlines",
			wantErr:     nil,
			description: "Key with newlines should be valid",
		},
		{
			name:        "key with tabs",
			key:         "key\twith\ttabs",
			wantErr:     nil,
			description: "Key with tabs should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			
			// Test Set operation
			err := store.Set(tt.key, []byte("test-value"))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Set() error = %v, wantErr %v (%s)", 
					err, tt.wantErr, tt.description)
				return
			}
			
			// If Set succeeded, test Get and Delete
			if err == nil {
				// Test Get operation
				_, err = store.Get(tt.key)
				if err != nil {
					t.Errorf("Get() error = %v, want nil (%s)", err, tt.description)
				}
				
				// Test Delete operation
				err = store.Delete(tt.key)
				if err != nil {
					t.Errorf("Delete() error = %v, want nil (%s)", err, tt.description)
				}
			} else {
				// Test Get operation with invalid key
				_, err = store.Get(tt.key)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Get() error = %v, wantErr %v (%s)", 
						err, tt.wantErr, tt.description)
				}
				
				// Test Delete operation with invalid key
				err = store.Delete(tt.key)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Delete() error = %v, wantErr %v (%s)", 
						err, tt.wantErr, tt.description)
				}
			}
		})
	}
}

// TestStoreList_Comprehensive provides comprehensive tests for listing keys
func TestStoreList_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*mockStore) error
		wantCount   int
		wantKeys    []string
		description string
	}{
		{
			name: "empty store",
			setup: func(s *mockStore) error {
				return nil // No setup needed
			},
			wantCount:   0,
			wantKeys:    []string{},
			description: "Empty store should return empty list",
		},
		{
			name: "single key",
			setup: func(s *mockStore) error {
				return s.Set("single-key", []byte("value"))
			},
			wantCount:   1,
			wantKeys:    []string{"single-key"},
			description: "Store with single key should return one key",
		},
		{
			name: "multiple keys",
			setup: func(s *mockStore) error {
				keys := []string{"key1", "key2", "key3"}
				for _, key := range keys {
					if err := s.Set(key, []byte("value")); err != nil {
						return err
					}
				}
				return nil
			},
			wantCount:   3,
			wantKeys:    []string{"key1", "key2", "key3"},
			description: "Store with multiple keys should return all keys",
		},
		{
			name: "keys with special characters",
			setup: func(s *mockStore) error {
				keys := []string{"key-with-hyphens", "key_with_underscores", "key.with.dots"}
				for _, key := range keys {
					if err := s.Set(key, []byte("value")); err != nil {
						return err
					}
				}
				return nil
			},
			wantCount:   3,
			wantKeys:    []string{"key-with-hyphens", "key_with_underscores", "key.with.dots"},
			description: "Keys with special characters should be listed",
		},
		{
			name: "after deletion",
			setup: func(s *mockStore) error {
				// Add keys
				keys := []string{"key1", "key2", "key3"}
				for _, key := range keys {
					if err := s.Set(key, []byte("value")); err != nil {
						return err
					}
				}
				// Delete one key
				return s.Delete("key2")
			},
			wantCount:   2,
			wantKeys:    []string{"key1", "key3"},
			description: "After deletion, remaining keys should be listed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			
			// Setup the store
			err := tt.setup(store)
			if err != nil {
				t.Fatalf("Setup error = %v (%s)", err, tt.description)
			}
			
			// List keys
			keys, err := store.List()
			if err != nil {
				t.Fatalf("List() error = %v (%s)", err, tt.description)
			}
			
			// Check count
			if len(keys) != tt.wantCount {
				t.Errorf("List() returned %d keys, want %d (%s)", 
					len(keys), tt.wantCount, tt.description)
				return
			}
			
			// Check that all expected keys are present
			keyMap := make(map[string]bool)
			for _, key := range keys {
				keyMap[key] = true
			}
			
			for _, expectedKey := range tt.wantKeys {
				if !keyMap[expectedKey] {
					t.Errorf("List() missing expected key %q (%s)", 
						expectedKey, tt.description)
				}
			}
		})
	}
}
