package storage

import (
	"errors"
	"testing"
)

// TestNewKeychainStore tests keychain store creation
func TestNewKeychainStore(t *testing.T) {
	tests := []struct {
		name        string
		service     string
		wantService string
	}{
		{
			name:        "with custom service",
			service:     "test-service",
			wantService: "test-service",
		},
		{
			name:        "with empty service",
			service:     "",
			wantService: "kiro-gateway",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewKeychainStore(tt.service)
			
			if store == nil {
				t.Fatal("NewKeychainStore() returned nil")
			}
			
			if store.service != tt.wantService {
				t.Errorf("service = %s, want %s", store.service, tt.wantService)
			}
		})
	}
}

// TestKeychainStoreInterface verifies KeychainStore implements Store
func TestKeychainStoreInterface(t *testing.T) {
	var _ Store = (*KeychainStore)(nil)
}

// TestKeychainStoreInvalidKey tests invalid key handling
func TestKeychainStoreInvalidKey(t *testing.T) {
	store := NewKeychainStore("test-service")
	
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

// Note: The following tests require actual OS keychain access and may fail
// in CI/CD environments without proper keychain setup. They are marked as
// integration tests and can be skipped with -short flag.

// TestKeychainStoreSetGet tests basic set/get operations
func TestKeychainStoreSetGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keychain integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-test")
	testKey := "test-key-set-get"
	testValue := []byte("test-value")
	
	// Clean up before and after test
	defer store.Delete(testKey)
	store.Delete(testKey) // Clean up any previous test data
	
	// Test Set
	err := store.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Test Get
	value, err := store.Get(testKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	
	if string(value) != string(testValue) {
		t.Errorf("Get() = %s, want %s", string(value), string(testValue))
	}
}

// TestKeychainStoreDelete tests delete operation
func TestKeychainStoreDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keychain integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-test")
	testKey := "test-key-delete"
	testValue := []byte("test-value")
	
	// Clean up before test
	store.Delete(testKey)
	
	// Set a value
	err := store.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Delete the value
	err = store.Delete(testKey)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	
	// Verify deleted
	_, err = store.Get(testKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

// TestKeychainStoreNotFound tests not found error
func TestKeychainStoreNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keychain integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-test")
	testKey := "non-existent-key"
	
	// Ensure key doesn't exist
	store.Delete(testKey)
	
	// Test Get
	_, err := store.Get(testKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
	
	// Test Delete
	err = store.Delete(testKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

// TestKeychainStoreUpdate tests updating existing values
func TestKeychainStoreUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keychain integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-test")
	testKey := "test-key-update"
	
	// Clean up before and after test
	defer store.Delete(testKey)
	store.Delete(testKey)
	
	// Set initial value
	err := store.Set(testKey, []byte("initial-value"))
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Update value
	err = store.Set(testKey, []byte("updated-value"))
	if err != nil {
		t.Fatalf("Set() update error = %v", err)
	}
	
	// Verify updated value
	value, err := store.Get(testKey)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	
	if string(value) != "updated-value" {
		t.Errorf("Get() after update = %s, want updated-value", string(value))
	}
}

// TestKeychainStoreBinaryData tests storing binary data
func TestKeychainStoreBinaryData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keychain integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-test")
	testKey := "test-key-binary"
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	
	// Clean up before and after test
	defer store.Delete(testKey)
	store.Delete(testKey)
	
	// Set binary data
	err := store.Set(testKey, binaryData)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Get binary data
	value, err := store.Get(testKey)
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

// TestKeychainStoreList tests listing keys
func TestKeychainStoreList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keychain integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-test")
	
	// Clean up test keys
	testKeys := []string{"list-key-1", "list-key-2", "list-key-3"}
	for _, key := range testKeys {
		store.Delete(key)
	}
	defer func() {
		for _, key := range testKeys {
			store.Delete(key)
		}
		store.Delete("__index__")
	}()
	
	// Add test keys
	for _, key := range testKeys {
		if err := store.Set(key, []byte("value")); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if err := store.addToIndex(key); err != nil {
			t.Fatalf("addToIndex() error = %v", err)
		}
	}
	
	// List keys
	keys, err := store.List()
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
