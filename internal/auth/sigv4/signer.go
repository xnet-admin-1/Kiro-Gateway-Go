// Package sigv4 provides AWS Signature Version 4 request signing.
// It implements the AWS SigV4 signing algorithm for authenticating requests
// to AWS services using IAM credentials.
package sigv4

import (
    "fmt"
    "net/http"
    "time"
    
    "github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
)

// Signer signs HTTP requests using AWS Signature Version 4.
// It uses AWS credentials to create cryptographic signatures that
// authenticate requests to AWS services.
type Signer struct {
    // credentials are the AWS credentials used for signing
    credentials *credentials.Credentials
    
    // region is the AWS region for the service endpoint
    region      string
    
    // service is the AWS service name (e.g., "codewhisperer")
    service     string
}

// NewSigner creates a new SigV4 signer with the provided credentials and configuration.
// The signer will use these credentials to sign all requests.
func NewSigner(creds *credentials.Credentials, region, service string) *Signer {
    return &Signer{
        credentials: creds,
        region:      region,
        service:     service,
    }
}

// SignRequest signs an HTTP request using AWS Signature Version 4.
// It adds the required headers (Authorization, X-Amz-Date, X-Amz-Security-Token)
// and calculates the cryptographic signature.
func (s *Signer) SignRequest(req *http.Request, body []byte) error {
    // Debug: Log signing details
    fmt.Printf("[DEBUG] SigV4 Signing - Service: %s, Region: %s, URL: %s\n", 
        s.service, s.region, req.URL.String())
    
    // Add timestamp header (required for SigV4)
    timestamp := time.Now().UTC()
    req.Header.Set("X-Amz-Date", timestamp.Format("20060102T150405Z"))
    
    // Add Host header if not present (required for canonical request)
    if req.Header.Get("Host") == "" {
        req.Header.Set("Host", req.URL.Host)
    }
    
    // Add security token header if using temporary credentials
    if s.credentials.SessionToken != "" {
        req.Header.Set("X-Amz-Security-Token", s.credentials.SessionToken)
    }
    
    // Step 1: Build canonical request
    canonicalRequest := buildCanonicalRequest(req, body)
    
    // Step 2: Build string to sign
    stringToSign := buildStringToSign(timestamp, s.region, s.service, canonicalRequest)
    
    // Step 3: Derive signing key
    signingKey := deriveSigningKey(s.credentials.SecretAccessKey, s.region, s.service, timestamp)
    
    // Step 4: Calculate signature
    signature := calculateSignature(signingKey, stringToSign)
    
    // Step 5: Build and add authorization header
    authHeader := s.buildAuthorizationHeader(timestamp, signature, req)
    req.Header.Set("Authorization", authHeader)
    
    return nil
}

// buildAuthorizationHeader constructs the Authorization header value.
// The header includes the credential scope, signed headers, and signature.
func (s *Signer) buildAuthorizationHeader(timestamp time.Time, signature string, req *http.Request) string {
    date := timestamp.Format("20060102")
    credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, s.region, s.service)
    credential := fmt.Sprintf("%s/%s", s.credentials.AccessKeyID, credentialScope)
    
    // Get signed headers from canonical headers
    _, signedHeaders := getCanonicalHeaders(req)
    
    return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s, SignedHeaders=%s, Signature=%s",
        credential, signedHeaders, signature)
}
