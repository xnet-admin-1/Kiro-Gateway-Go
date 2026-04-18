package storage

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "sqlite-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	
	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open database: %v", err)
	}
	
	// Cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	
	return db, cleanup
}

// TestNewSQLiteStore tests SQLite store creation
func TestNewSQLiteStore(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	if store == nil {
		t.Fatal("NewSQLiteStore() returned nil")
	}
	
	if store.db == nil {
		t.Error("store.db is nil")
	}
	
	if store.encryption == nil {
		t.Error("store.encryption is nil")
	}
}

// TestNewSQLiteStoreWithEncryption tests SQLite store creation with custom encryption
func TestNewSQLiteStoreWithEncryption(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// Create custom encryption
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	
	encryption, err := NewEncryptionWithKey(key)
	if err != nil {
		t.Fatalf("NewEncryptionWithKey() error = %v", err)
	}
	
	store, err := NewSQLiteStoreWithEncryption(db, encryption)
	if err != nil {
		t.Fatalf("NewSQLiteStoreWithEncryption() error = %v", err)
	}
	
	if store == nil {
		t.Fatal("NewSQLiteStoreWithEncryption() returned nil")
	}
}

// TestSQLiteStoreInterface verifies SQLiteStore implements Store
func TestSQLiteStoreInterface(t *testing.T) {
	var _ Store = (*SQLiteStore)(nil)
}

// TestSQLiteStoreSetGet tests basic set/get operations
func TestSQLiteStoreSetGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "test-key"
	testValue := []byte("test-value")
	
	// Test Set
	err = store.Set(testKey, testValue)
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

// TestSQLiteStoreDelete tests delete operation
func TestSQLiteStoreDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "test-key"
	testValue := []byte("test-value")
	
	// Set a value
	err = store.Set(testKey, testValue)
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

// TestSQLiteStoreInvalidKey tests invalid key handling
func TestSQLiteStoreInvalidKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
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

// TestSQLiteStoreNotFound tests not found error
func TestSQLiteStoreNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Test Get
	_, err = store.Get("non-existent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
	
	// Test Delete
	err = store.Delete("non-existent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

// TestSQLiteStoreUpdate tests updating existing values
func TestSQLiteStoreUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "test-key"
	
	// Set initial value
	err = store.Set(testKey, []byte("initial-value"))
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

// TestSQLiteStoreBinaryData tests storing binary data
func TestSQLiteStoreBinaryData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "test-key"
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	
	// Set binary data
	err = store.Set(testKey, binaryData)
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

// TestSQLiteStoreList tests listing keys
func TestSQLiteStoreList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
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

// TestSQLiteStoreEncryptionAtRest tests that data is encrypted in database
func TestSQLiteStoreEncryptionAtRest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "test-key"
	testValue := []byte("secret-value")
	
	// Set value
	err = store.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Query database directly to verify encryption
	var storedValue []byte
	err = db.QueryRow("SELECT value FROM secrets WHERE key = ?", testKey).Scan(&storedValue)
	if err != nil {
		t.Fatalf("QueryRow() error = %v", err)
	}
	
	// Verify stored value is NOT the plaintext
	if string(storedValue) == string(testValue) {
		t.Error("value is stored in plaintext, expected encrypted")
	}
	
	// Verify stored value is different from plaintext
	if len(storedValue) == len(testValue) {
		t.Error("encrypted value has same length as plaintext, suspicious")
	}
}

// TestSQLiteStoreMultipleKeys tests storing multiple keys
func TestSQLiteStoreMultipleKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Store multiple key-value pairs
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	for key, value := range testData {
		err := store.Set(key, []byte(value))
		if err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
	}
	
	// Verify all values
	for key, expectedValue := range testData {
		value, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get(%s) error = %v", key, err)
		}
		
		if string(value) != expectedValue {
			t.Errorf("Get(%s) = %s, want %s", key, string(value), expectedValue)
		}
	}
}

// TestSQLiteStoreConcurrency tests concurrent access
func TestSQLiteStoreConcurrency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	const numGoroutines = 10
	const numOperations = 50
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < numOperations; j++ {
				key := "concurrent-key"
				value := []byte("test-value")
				
				// Set
				err := store.Set(key, value)
				if err != nil {
					t.Errorf("Set() error = %v", err)
					return
				}
				
				// Get
				_, err = store.Get(key)
				if err != nil {
					t.Errorf("Get() error = %v", err)
					return
				}
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestSQLiteStoreTableCreation tests that table is created automatically
func TestSQLiteStoreTableCreation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// Create store (should create table)
	_, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='secrets'").Scan(&tableName)
	if err != nil {
		t.Fatalf("table 'secrets' not found: %v", err)
	}
	
	if tableName != "secrets" {
		t.Errorf("table name = %s, want secrets", tableName)
	}
}

// TestSQLiteStoreClose tests closing the store
func TestSQLiteStoreClose(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Close store
	err = store.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
	
	// Verify database is closed (operations should fail)
	err = store.Set("key", []byte("value"))
	if err == nil {
		t.Error("Set() after Close() should fail")
	}
}

// BenchmarkSQLiteStoreSet benchmarks Set operation
func BenchmarkSQLiteStoreSet(b *testing.B) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		b.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	value := []byte("test value for benchmarking")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := store.Set("bench-key", value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSQLiteStoreGet benchmarks Get operation
func BenchmarkSQLiteStoreGet(b *testing.B) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		b.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Set initial value
	value := []byte("test value for benchmarking")
	err = store.Set("bench-key", value)
	if err != nil {
		b.Fatalf("Set() error = %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Get("bench-key")
		if err != nil {
			b.Fatal(err)
		}
	}
}
