package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// SQLiteStore implements Store using encrypted SQLite database
type SQLiteStore struct {
	db         *sql.DB
	encryption *Encryption
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	encryption, err := NewEncryption()
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption: %w", err)
	}
	
	store := &SQLiteStore{
		db:         db,
		encryption: encryption,
	}
	
	// Create table if not exists
	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	
	return store, nil
}

// NewSQLiteStoreWithEncryption creates a new SQLite store with custom encryption
func NewSQLiteStoreWithEncryption(db *sql.DB, encryption *Encryption) (*SQLiteStore, error) {
	store := &SQLiteStore{
		db:         db,
		encryption: encryption,
	}
	
	// Create table if not exists
	if err := store.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	
	return store, nil
}

// createTable creates the secrets table if it doesn't exist
func (s *SQLiteStore) createTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS secrets (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`
	
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create secrets table: %w", err)
	}
	
	return nil
}

// Get retrieves a value from the database
func (s *SQLiteStore) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	
	var encrypted []byte
	err := s.db.QueryRow(
		"SELECT value FROM secrets WHERE key = ?",
		key,
	).Scan(&encrypted)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get from database: %w", err)
	}
	
	// Decrypt value
	decrypted, err := s.encryption.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt value: %w", err)
	}
	
	return decrypted, nil
}

// Set stores a value in the database
func (s *SQLiteStore) Set(key string, value []byte) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	// Encrypt value
	encrypted, err := s.encryption.Encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}
	
	now := time.Now().Unix()
	
	// Insert or replace
	_, err = s.db.Exec(
		`INSERT INTO secrets (key, value, created_at, updated_at) 
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET 
		   value = excluded.value,
		   updated_at = excluded.updated_at`,
		key, encrypted, now, now,
	)
	
	if err != nil {
		return fmt.Errorf("failed to set in database: %w", err)
	}
	
	return nil
}

// Delete removes a value from the database
func (s *SQLiteStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	result, err := s.db.Exec(
		"DELETE FROM secrets WHERE key = ?",
		key,
	)
	
	if err != nil {
		return fmt.Errorf("failed to delete from database: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return ErrNotFound
	}
	
	return nil
}

// List returns all keys in the database
func (s *SQLiteStore) List() ([]string, error) {
	rows, err := s.db.Query("SELECT key FROM secrets ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()
	
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return keys, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
