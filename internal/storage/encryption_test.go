package storage

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestNewEncryption tests encryption creation
func TestNewEncryption(t *testing.T) {
	enc, err := NewEncryption()
	if err != nil {
		t.Fatalf("NewEncryption() error = %v", err)
	}
	
	if enc == nil {
		t.Fatal("NewEncryption() returned nil")
	}
	
	if len(enc.key) != 32 {
		t.Errorf("key length = %d, want 32", len(enc.key))
	}
}

// TestNewEncryptionWithKey tests encryption creation with custom key
func TestNewEncryptionWithKey(t *testing.T) {
	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			key:     make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "invalid 16-byte key",
			key:     make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "invalid 64-byte key",
			key:     make([]byte, 64),
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     []byte{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncryptionWithKey(tt.key)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptionWithKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && enc == nil {
				t.Error("NewEncryptionWithKey() returned nil")
			}
		})
	}
}

// TestEncryptDecrypt tests encryption and decryption round-trip
func TestEncryptDecrypt(t *testing.T) {
	enc, err := NewEncryption()
	if err != nil {
		t.Fatalf("NewEncryption() error = %v", err)
	}
	
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "simple text",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "empty data",
			plaintext: []byte{},
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "large data",
			plaintext: bytes.Repeat([]byte("test"), 1000),
		},
		{
			name:      "unicode text",
			plaintext: []byte("Hello 世界 🌍"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			
			// Verify ciphertext is different from plaintext
			if len(tt.plaintext) > 0 && bytes.Equal(ciphertext, tt.plaintext) {
				t.Error("Encrypt() ciphertext equals plaintext")
			}
			
			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			
			// Verify decrypted equals original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

// TestEncryptDifferentCiphertexts tests that encryption produces different ciphertexts
func TestEncryptDifferentCiphertexts(t *testing.T) {
	enc, err := NewEncryption()
	if err != nil {
		t.Fatalf("NewEncryption() error = %v", err)
	}
	
	plaintext := []byte("test data")
	
	// Encrypt same plaintext multiple times
	ciphertext1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	ciphertext2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Verify ciphertexts are different (due to random nonce)
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Encrypt() produced identical ciphertexts for same plaintext")
	}
	
	// Verify both decrypt to same plaintext
	decrypted1, err := enc.Decrypt(ciphertext1)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	decrypted2, err := enc.Decrypt(ciphertext2)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Decrypt() failed to recover original plaintext")
	}
}

// TestDecryptInvalidCiphertext tests decryption with invalid data
func TestDecryptInvalidCiphertext(t *testing.T) {
	enc, err := NewEncryption()
	if err != nil {
		t.Fatalf("NewEncryption() error = %v", err)
	}
	
	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
		},
		{
			name:       "too short ciphertext",
			ciphertext: []byte{0x01, 0x02, 0x03},
		},
		{
			name:       "random data",
			ciphertext: []byte("not encrypted data"),
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: func() []byte {
				plaintext := []byte("test")
				ciphertext, _ := enc.Encrypt(plaintext)
				// Corrupt the ciphertext
				if len(ciphertext) > 20 {
					ciphertext[20] ^= 0xFF
				}
				return ciphertext
			}(),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Decrypt() expected error for invalid ciphertext")
			}
		})
	}
}

// TestDecryptWithDifferentKey tests that decryption fails with wrong key
func TestDecryptWithDifferentKey(t *testing.T) {
	// Create two encryption instances with different keys
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)
	
	enc1, err := NewEncryptionWithKey(key1)
	if err != nil {
		t.Fatalf("NewEncryptionWithKey() error = %v", err)
	}
	
	enc2, err := NewEncryptionWithKey(key2)
	if err != nil {
		t.Fatalf("NewEncryptionWithKey() error = %v", err)
	}
	
	plaintext := []byte("secret data")
	
	// Encrypt with first key
	ciphertext, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Try to decrypt with second key
	_, err = enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() expected error when using wrong key")
	}
}

// TestDeriveKey tests key derivation
func TestDeriveKey(t *testing.T) {
	// Derive key multiple times
	key1, err := deriveKey()
	if err != nil {
		t.Fatalf("deriveKey() error = %v", err)
	}
	
	key2, err := deriveKey()
	if err != nil {
		t.Fatalf("deriveKey() error = %v", err)
	}
	
	// Verify keys are consistent
	if !bytes.Equal(key1, key2) {
		t.Error("deriveKey() produced different keys")
	}
	
	// Verify key length
	if len(key1) != 32 {
		t.Errorf("deriveKey() key length = %d, want 32", len(key1))
	}
}

// TestEncryptionConcurrency tests concurrent encryption/decryption
func TestEncryptionConcurrency(t *testing.T) {
	enc, err := NewEncryption()
	if err != nil {
		t.Fatalf("NewEncryption() error = %v", err)
	}
	
	const numGoroutines = 10
	const numOperations = 100
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < numOperations; j++ {
				plaintext := []byte("test data")
				
				ciphertext, err := enc.Encrypt(plaintext)
				if err != nil {
					t.Errorf("Encrypt() error = %v", err)
					return
				}
				
				decrypted, err := enc.Decrypt(ciphertext)
				if err != nil {
					t.Errorf("Decrypt() error = %v", err)
					return
				}
				
				if !bytes.Equal(decrypted, plaintext) {
					t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
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

// BenchmarkEncrypt benchmarks encryption performance
func BenchmarkEncrypt(b *testing.B) {
	enc, err := NewEncryption()
	if err != nil {
		b.Fatalf("NewEncryption() error = %v", err)
	}
	
	plaintext := []byte("test data for benchmarking")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enc.Encrypt(plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecrypt benchmarks decryption performance
func BenchmarkDecrypt(b *testing.B) {
	enc, err := NewEncryption()
	if err != nil {
		b.Fatalf("NewEncryption() error = %v", err)
	}
	
	plaintext := []byte("test data for benchmarking")
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		b.Fatalf("Encrypt() error = %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enc.Decrypt(ciphertext)
		if err != nil {
			b.Fatal(err)
		}
	}
}
