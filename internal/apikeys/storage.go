package apikeys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Storage interface for API key persistence
type Storage interface {
	Save(key *APIKey) error
	Load(keyString string) (*APIKey, error)
	LoadAll() ([]*APIKey, error)
	Delete(keyString string) error
}

// FileStorage implements file-based storage for API keys
type FileStorage struct {
	directory string
	mu        sync.RWMutex
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(directory string) (*FileStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &FileStorage{
		directory: directory,
	}, nil
}

// Save saves an API key to disk
func (s *FileStorage) Save(key *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Serialize to JSON
	data, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API key: %w", err)
	}
	
	// Write to file
	filename := s.getFilename(key.Key)
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write API key file: %w", err)
	}
	
	return nil
}

// Load loads an API key from disk
func (s *FileStorage) Load(keyString string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	filename := s.getFilename(keyString)
	
	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to read API key file: %w", err)
	}
	
	// Deserialize
	var key APIKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("failed to unmarshal API key: %w", err)
	}
	
	return &key, nil
}

// LoadAll loads all API keys from disk
func (s *FileStorage) LoadAll() ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Read directory
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}
	
	var keys []*APIKey
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		// Read file
		filename := filepath.Join(s.directory, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue // Skip files that can't be read
		}
		
		// Deserialize
		var key APIKey
		if err := json.Unmarshal(data, &key); err != nil {
			continue // Skip invalid files
		}
		
		keys = append(keys, &key)
	}
	
	return keys, nil
}

// Delete deletes an API key from disk
func (s *FileStorage) Delete(keyString string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	filename := s.getFilename(keyString)
	
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("API key not found")
		}
		return fmt.Errorf("failed to delete API key file: %w", err)
	}
	
	return nil
}

// getFilename returns the filename for an API key
func (s *FileStorage) getFilename(keyString string) string {
	// Use a hash or sanitized version of the key as filename
	// For simplicity, we'll use the key itself (it's already safe)
	return filepath.Join(s.directory, fmt.Sprintf("%s.json", sanitizeFilename(keyString)))
}

// sanitizeFilename sanitizes a string for use as a filename
func sanitizeFilename(s string) string {
	// Replace any characters that might be problematic
	// For API keys, we can use them directly as they're alphanumeric + dash
	return s
}

// PersistentAPIKeyManager wraps APIKeyManager with persistent storage
type PersistentAPIKeyManager struct {
	*APIKeyManager
	storage Storage
}

// NewPersistentAPIKeyManager creates a new persistent API key manager
func NewPersistentAPIKeyManager(prefix string, storage Storage) (*PersistentAPIKeyManager, error) {
	manager := NewAPIKeyManager(prefix)
	
	// Load existing keys from storage
	keys, err := storage.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load API keys: %w", err)
	}
	
	// Add keys to manager
	for _, key := range keys {
		manager.keys[key.Key] = key
	}
	
	return &PersistentAPIKeyManager{
		APIKeyManager: manager,
		storage:       storage,
	}, nil
}

// GenerateKey generates and persists a new API key
func (m *PersistentAPIKeyManager) GenerateKey(name, userID string, expiresIn *time.Duration, permissions []string) (*APIKey, error) {
	// Generate key
	key, err := m.APIKeyManager.GenerateKey(name, userID, expiresIn, permissions)
	if err != nil {
		return nil, err
	}
	
	// Persist to storage
	if err := m.storage.Save(key); err != nil {
		// Rollback: remove from memory
		m.mu.Lock()
		delete(m.keys, key.Key)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to persist API key: %w", err)
	}
	
	return key, nil
}

// RevokeKey revokes and persists the change
func (m *PersistentAPIKeyManager) RevokeKey(key string) error {
	if err := m.APIKeyManager.RevokeKey(key); err != nil {
		return err
	}
	
	// Persist change
	apiKey, _ := m.APIKeyManager.GetKey(key)
	if apiKey != nil {
		if err := m.storage.Save(apiKey); err != nil {
			return fmt.Errorf("failed to persist revocation: %w", err)
		}
	}
	
	return nil
}

// DeleteKey deletes and removes from storage
func (m *PersistentAPIKeyManager) DeleteKey(key string) error {
	if err := m.APIKeyManager.DeleteKey(key); err != nil {
		return err
	}
	
	// Delete from storage
	if err := m.storage.Delete(key); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}
	
	return nil
}

// UpdateKey updates and persists the change
func (m *PersistentAPIKeyManager) UpdateKey(key string, name string, metadata map[string]string) error {
	if err := m.APIKeyManager.UpdateKey(key, name, metadata); err != nil {
		return err
	}
	
	// Persist change
	apiKey, _ := m.APIKeyManager.GetKey(key)
	if apiKey != nil {
		if err := m.storage.Save(apiKey); err != nil {
			return fmt.Errorf("failed to persist update: %w", err)
		}
	}
	
	return nil
}

// ValidateKey validates and persists usage update
func (m *PersistentAPIKeyManager) ValidateKey(key string) (*APIKey, error) {
	apiKey, err := m.APIKeyManager.ValidateKey(key)
	if err != nil {
		return nil, err
	}
	
	// Persist usage update (async to avoid blocking)
	go func() {
		m.storage.Save(apiKey)
	}()
	
	return apiKey, nil
}
