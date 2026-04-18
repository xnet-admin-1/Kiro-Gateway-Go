package sigv4

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// buildStringToSign creates the string to sign for SigV4
func buildStringToSign(timestamp time.Time, region, service, canonicalRequest string) string {
	date := timestamp.Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, region, service)
	hashedCanonicalRequest := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))

	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		timestamp.Format("20060102T150405Z"),
		credentialScope,
		hashedCanonicalRequest,
	)
}
