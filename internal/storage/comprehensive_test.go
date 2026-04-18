package storage

import (
    "bytes"
    "testing"
)

func TestKeychainStore_Operations(t *testing.T) {
    store := NewKeychainStore("test-service")
    
    testKey := "test-key"
    testValue := []byte("test-value-data")
    
    // Test Set
    err := store.Set(testKey, testValue)
    if err != nil {
        t.Errorf("Set() error = %v", err)
    }
    
    // Test Get
    retrieved, err := store.Get(testKey)
    if err != nil {
        t.Errorf("Get() error = %v", err)
    }
    
    if !bytes.Equal(retrieved, testValue) {
        t.Errorf("Get() = %v, want %v", retrieved, testValue)
    }
    
    // Test Delete
    err = store.Delete(testKey)
    if err != nil {
        t.Errorf("Delete() error = %v", err)
    }
}

func TestStore_Interface(t *testing.T) {
    var _ Store = (*KeychainStore)(nil)
    
    // Verify interface compliance
    store := NewKeychainStore("test")
    
    // Test with empty key
    _, err := store.Get("")
    if err == nil {
        t.Error("Expected error for empty key")
    }
    
    // Test with nil value
    err = store.Set("test", nil)
    if err != nil {
        t.Errorf("Set() with nil value error = %v", err)
    }
}

func TestStore_ErrorHandling(t *testing.T) {
    store := NewKeychainStore("test-service")
    
    // Test getting non-existent key
    _, err := store.Get("non-existent-key")
    if err == nil {
        t.Error("Expected error for non-existent key")
    }
    
    // Test deleting non-existent key
    err = store.Delete("non-existent-key")
    if err == nil {
        t.Error("Expected error for deleting non-existent key")
    }
}
