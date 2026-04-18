package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	
	_ "github.com/mattn/go-sqlite3"
)

// TestKeychainToSQLiteFallback tests fallback from keychain to SQLite
func TestKeychainToSQLiteFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Create keychain store
	keychainStore := NewKeychainStore("kiro-gateway-integration-test")
	
	// Create SQLite store
	tmpDir, err := os.MkdirTemp("", "storage-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	
	sqliteStore, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "integration-test-key"
	testValue := []byte("integration-test-value")
	
	// Clean up before test
	keychainStore.Delete(testKey)
	defer keychainStore.Delete(testKey)
	
	// Try keychain first
	err = keychainStore.Set(testKey, testValue)
	if err != nil {
		t.Logf("Keychain not available, testing SQLite fallback: %v", err)
		
		// Fallback to SQLite
		err = sqliteStore.Set(testKey, testValue)
		if err != nil {
			t.Fatalf("SQLite fallback Set() error = %v", err)
		}
		
		// Verify SQLite storage
		value, err := sqliteStore.Get(testKey)
		if err != nil {
			t.Fatalf("SQLite fallback Get() error = %v", err)
		}
		
		if string(value) != string(testValue) {
			t.Errorf("SQLite fallback Get() = %s, want %s", string(value), string(testValue))
		}
		
		return
	}
	
	// Keychain available - verify storage
	value, err := keychainStore.Get(testKey)
	if err != nil {
		t.Fatalf("Keychain Get() error = %v", err)
	}
	
	if string(value) != string(testValue) {
		t.Errorf("Keychain Get() = %s, want %s", string(value), string(testValue))
	}
}

// TestRealKeychainStorage tests actual OS keychain storage
func TestRealKeychainStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	store := NewKeychainStore("kiro-gateway-integration-test")
	
	testCases := []struct {
		name  string
		key   string
		value []byte
	}{
		{
			name:  "simple text",
			key:   "integration-text",
			value: []byte("Hello, World!"),
		},
		{
			name:  "binary data",
			key:   "integration-binary",
			value: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:  "large data",
			key:   "integration-large",
			value: make([]byte, 10000),
		},
		{
			name:  "unicode text",
			key:   "integration-unicode",
			value: []byte("Hello 世界 🌍"),
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up before and after test
			defer store.Delete(tc.key)
			store.Delete(tc.key)
			
			// Set value
			err := store.Set(tc.key, tc.value)
			if err != nil {
				t.Skipf("Keychain not available: %v", err)
			}
			
			// Get value
			value, err := store.Get(tc.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			
			// Verify value
			if len(value) != len(tc.value) {
				t.Errorf("Get() length = %d, want %d", len(value), len(tc.value))
			}
			
			for i, b := range value {
				if b != tc.value[i] {
					t.Errorf("Get() byte[%d] = %x, want %x", i, b, tc.value[i])
					break
				}
			}
		})
	}
}

// TestRealSQLiteStorage tests actual SQLite storage with encryption
func TestRealSQLiteStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "storage-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testCases := []struct {
		name  string
		key   string
		value []byte
	}{
		{
			name:  "simple text",
			key:   "integration-text",
			value: []byte("Hello, World!"),
		},
		{
			name:  "binary data",
			key:   "integration-binary",
			value: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:  "large data",
			key:   "integration-large",
			value: make([]byte, 100000),
		},
		{
			name:  "unicode text",
			key:   "integration-unicode",
			value: []byte("Hello 世界 🌍"),
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set value
			err := store.Set(tc.key, tc.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			
			// Get value
			value, err := store.Get(tc.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			
			// Verify value
			if len(value) != len(tc.value) {
				t.Errorf("Get() length = %d, want %d", len(value), len(tc.value))
			}
			
			for i, b := range value {
				if b != tc.value[i] {
					t.Errorf("Get() byte[%d] = %x, want %x", i, b, tc.value[i])
					break
				}
			}
			
			// Verify encryption at rest
			var storedValue []byte
			err = db.QueryRow("SELECT value FROM secrets WHERE key = ?", tc.key).Scan(&storedValue)
			if err != nil {
				t.Fatalf("QueryRow() error = %v", err)
			}
			
			// Verify stored value is encrypted (different from plaintext)
			if len(tc.value) > 0 && string(storedValue) == string(tc.value) {
				t.Error("value is stored in plaintext, expected encrypted")
			}
		})
	}
}

// TestStorePersistence tests that data persists across store instances
func TestStorePersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "storage-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	testKey := "persistence-key"
	testValue := []byte("persistence-value")
	
	// Create first store and set value
	{
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		
		store, err := NewSQLiteStore(db)
		if err != nil {
			t.Fatalf("NewSQLiteStore() error = %v", err)
		}
		
		err = store.Set(testKey, testValue)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		
		db.Close()
	}
	
	// Create second store and verify value persists
	{
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db.Close()
		
		store, err := NewSQLiteStore(db)
		if err != nil {
			t.Fatalf("NewSQLiteStore() error = %v", err)
		}
		
		value, err := store.Get(testKey)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		
		if string(value) != string(testValue) {
			t.Errorf("Get() = %s, want %s", string(value), string(testValue))
		}
	}
}

// TestEncryptionKeyConsistency tests that encryption key is consistent
func TestEncryptionKeyConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "storage-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	testKey := "encryption-key"
	testValue := []byte("encryption-value")
	
	// Create first store and set value
	{
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		
		store, err := NewSQLiteStore(db)
		if err != nil {
			t.Fatalf("NewSQLiteStore() error = %v", err)
		}
		
		err = store.Set(testKey, testValue)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		
		db.Close()
	}
	
	// Create second store with new encryption instance
	// Should be able to decrypt because key derivation is consistent
	{
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db.Close()
		
		// Create new encryption instance (should derive same key)
		encryption, err := NewEncryption()
		if err != nil {
			t.Fatalf("NewEncryption() error = %v", err)
		}
		
		store, err := NewSQLiteStoreWithEncryption(db, encryption)
		if err != nil {
			t.Fatalf("NewSQLiteStoreWithEncryption() error = %v", err)
		}
		
		value, err := store.Get(testKey)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		
		if string(value) != string(testValue) {
			t.Errorf("Get() = %s, want %s", string(value), string(testValue))
		}
	}
}

// TestMultipleStoresIndependence tests that different stores are independent
func TestMultipleStoresIndependence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Create two temp databases
	tmpDir1, err := os.MkdirTemp("", "storage-integration-1-*")
	if err != nil {
		t.Fatalf("failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tmpDir1)
	
	tmpDir2, err := os.MkdirTemp("", "storage-integration-2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tmpDir2)
	
	dbPath1 := filepath.Join(tmpDir1, "test1.db")
	dbPath2 := filepath.Join(tmpDir2, "test2.db")
	
	// Create first store
	db1, err := sql.Open("sqlite3", dbPath1)
	if err != nil {
		t.Fatalf("failed to open database 1: %v", err)
	}
	defer db1.Close()
	
	store1, err := NewSQLiteStore(db1)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Create second store
	db2, err := sql.Open("sqlite3", dbPath2)
	if err != nil {
		t.Fatalf("failed to open database 2: %v", err)
	}
	defer db2.Close()
	
	store2, err := NewSQLiteStore(db2)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	testKey := "test-key"
	
	// Set different values in each store
	err = store1.Set(testKey, []byte("value1"))
	if err != nil {
		t.Fatalf("store1.Set() error = %v", err)
	}
	
	err = store2.Set(testKey, []byte("value2"))
	if err != nil {
		t.Fatalf("store2.Set() error = %v", err)
	}
	
	// Verify values are independent
	value1, err := store1.Get(testKey)
	if err != nil {
		t.Fatalf("store1.Get() error = %v", err)
	}
	
	value2, err := store2.Get(testKey)
	if err != nil {
		t.Fatalf("store2.Get() error = %v", err)
	}
	
	if string(value1) != "value1" {
		t.Errorf("store1.Get() = %s, want value1", string(value1))
	}
	
	if string(value2) != "value2" {
		t.Errorf("store2.Get() = %s, want value2", string(value2))
	}
}

// TestStoreErrorRecovery tests error recovery scenarios
func TestStoreErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "storage-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	
	store, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	
	// Test recovery after failed operation
	testKey := "recovery-key"
	testValue := []byte("recovery-value")
	
	// Set value
	err = store.Set(testKey, testValue)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	
	// Try to get non-existent key (should fail gracefully)
	_, err = store.Get("non-existent")
	if err == nil {
		t.Error("Get() expected error for non-existent key")
	}
	
	// Verify store still works after error
	value, err := store.Get(testKey)
	if err != nil {
		t.Fatalf("Get() after error recovery error = %v", err)
	}
	
	if string(value) != string(testValue) {
		t.Errorf("Get() after error recovery = %s, want %s", string(value), string(testValue))
	}
}
