package sigv4

import (
	"encoding/hex"
)

// calculateSignature calculates the final signature for SigV4
func calculateSignature(signingKey []byte, stringToSign string) string {
	signature := hmacSHA256(signingKey, []byte(stringToSign))
	return hex.EncodeToString(signature)
}
