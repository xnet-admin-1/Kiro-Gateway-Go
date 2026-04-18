package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// Encryption provides AES-256-GCM encryption/decryption
type Encryption struct {
	key []byte
}

// NewEncryption creates a new encryption instance
// The key is derived from machine-specific data
func NewEncryption() (*Encryption, error) {
	key, err := deriveKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	
	return &Encryption{
		key: key,
	}, nil
}

// NewEncryptionWithKey creates a new encryption instance with a specific key
func NewEncryptionWithKey(key []byte) (*Encryption, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}
	
	return &Encryption{
		key: key,
	}, nil
}

// Encrypt encrypts data using AES-256-GCM
func (e *Encryption) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func (e *Encryption) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return plaintext, nil
}

// deriveKey derives an encryption key from machine-specific data
func deriveKey() ([]byte, error) {
	// Use hostname as part of the key derivation
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "default-host"
	}
	
	// Use username as part of the key derivation
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}
	if username == "" {
		username = "default-user"
	}
	
	// Combine hostname and username
	data := fmt.Sprintf("%s-%s-kiro-gateway", hostname, username)
	
	// Hash to create 32-byte key
	hash := sha256.Sum256([]byte(data))
	
	return hash[:], nil
}
