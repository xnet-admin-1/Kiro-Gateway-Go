package sigv4

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestDeriveSigningKey(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		region    string
		service   string
		timestamp time.Time
	}{
		{
			name:      "basic signing key derivation",
			secretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
			region:    "us-east-1",
			service:   "service",
			timestamp: time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveSigningKey(tt.secretKey, tt.region, tt.service, tt.timestamp)
			
			// Check that we get a 32-byte key (256 bits)
			if len(result) != 32 {
				t.Errorf("deriveSigningKey() should return 32 bytes, got %d", len(result))
			}
			
			// Check that the key is not all zeros
			allZeros := true
			for _, b := range result {
				if b != 0 {
					allZeros = false
					break
				}
			}
			if allZeros {
				t.Error("deriveSigningKey() should not return all zeros")
			}
		})
	}
}

func TestHmacSHA256(t *testing.T) {
	key := []byte("key")
	data := []byte("The quick brown fox jumps over the lazy dog")
	expected := "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8"
	
	result := hmacSHA256(key, data)
	resultHex := hex.EncodeToString(result)
	
	if resultHex != expected {
		t.Errorf("hmacSHA256() = %s, want %s", resultHex, expected)
	}
}
