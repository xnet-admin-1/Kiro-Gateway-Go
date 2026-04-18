package sigv4

import (
	"crypto/hmac"
	"crypto/sha256"
	"time"
)

// deriveSigningKey derives the signing key for SigV4
func deriveSigningKey(secretKey, region, service string, timestamp time.Time) []byte {
	date := timestamp.Format("20060102")
	
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	
	return kSigning
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
