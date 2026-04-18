package storage

import (
	"encoding/json"
	"fmt"
	
	"github.com/zalando/go-keyring"
)

// KeychainStore implements Store using OS keychain
// - Windows: Credential Manager
// - macOS: Keychain
// - Linux: Secret Service
type KeychainStore struct {
	service string
}

// NewKeychainStore creates a new keychain store
func NewKeychainStore(service string) *KeychainStore {
	if service == "" {
		service = "kiro-gateway"
	}
	return &KeychainStore{
		service: service,
	}
}

// Get retrieves a value from the keychain
func (k *KeychainStore) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	
	value, err := keyring.Get(k.service, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get from keychain: %w", err)
	}
	
	return []byte(value), nil
}

// Set stores a value in the keychain
func (k *KeychainStore) Set(key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	err := keyring.Set(k.service, key, string(value))
	if err != nil {
		return fmt.Errorf("failed to set in keychain: %w", err)
	}
	
	return nil
}

// Delete removes a value from the keychain
func (k *KeychainStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	err := keyring.Delete(k.service, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete from keychain: %w", err)
	}
	
	return nil
}

// List returns all keys in the keychain
// Note: go-keyring doesn't support listing, so we maintain a separate index
func (k *KeychainStore) List() ([]string, error) {
	// Try to get the index
	indexData, err := k.Get("__index__")
	if err != nil {
		if err == ErrNotFound {
			return []string{}, nil
		}
		return nil, err
	}
	
	var keys []string
	if err := json.Unmarshal(indexData, &keys); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}
	
	return keys, nil
}

// updateIndex updates the key index
func (k *KeychainStore) updateIndex(keys []string) error {
	indexData, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}
	
	return k.Set("__index__", indexData)
}

// addToIndex adds a key to the index
func (k *KeychainStore) addToIndex(key string) error {
	keys, err := k.List()
	if err != nil {
		return err
	}
	
	// Check if key already exists
	for _, k := range keys {
		if k == key {
			return nil
		}
	}
	
	keys = append(keys, key)
	return k.updateIndex(keys)
}

// removeFromIndex removes a key from the index
func (k *KeychainStore) removeFromIndex(key string) error {
	keys, err := k.List()
	if err != nil {
		return err
	}
	
	// Remove key from list
	newKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}
	
	return k.updateIndex(newKeys)
}
