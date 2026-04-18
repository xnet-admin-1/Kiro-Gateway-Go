package sigv4

import (
	"testing"
)

func TestCalculateSignature(t *testing.T) {
	tests := []struct {
		name         string
		signingKey   []byte
		stringToSign string
	}{
		{
			name:         "basic signature calculation",
			signingKey:   []byte("test-signing-key-32-bytes-long!!"),
			stringToSign: "AWS4-HMAC-SHA256\n20150830T123600Z\n20150830/us-east-1/service/aws4_request\n816cd5b414d056048ba4f7c5386d6e0533120fb1fcfa93762cf0fc39e2cf19e0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSignature(tt.signingKey, tt.stringToSign)
			
			// Check that we get a hex string of the right length (64 chars for SHA256)
			if len(result) != 64 {
				t.Errorf("calculateSignature() should return 64-char hex string, got %d chars", len(result))
			}
			
			// Check that it's valid hex
			for _, c := range result {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("calculateSignature() should return valid hex, got char %c", c)
				}
			}
		})
	}
}
