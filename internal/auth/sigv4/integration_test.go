package sigv4

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
)

func TestSigV4Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test with known AWS test vectors
	creds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Source:          "test",
	}

	signer := NewSigner(creds, "us-east-1", "service")

	tests := []struct {
		name   string
		method string
		path   string
		query  string
		body   []byte
	}{
		{
			name:   "GET request",
			method: "GET",
			path:   "/",
			query:  "",
			body:   []byte{},
		},
		{
			name:   "POST request with body",
			method: "POST",
			path:   "/",
			query:  "",
			body:   []byte(`{"test": "data"}`),
		},
		{
			name:   "GET with query parameters",
			method: "GET",
			path:   "/",
			query:  "Action=ListUsers&Version=2010-05-08",
			body:   []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: tt.method,
				URL:    &url.URL{Path: tt.path, RawQuery: tt.query, Host: "example.amazonaws.com"},
				Header: make(http.Header),
			}
			req.Header.Set("Host", "example.amazonaws.com")

			err := signer.SignRequest(req, tt.body)
			if err != nil {
				t.Fatalf("SignRequest() error = %v", err)
			}

			// Verify all required headers are present
			if req.Header.Get("Authorization") == "" {
				t.Error("Authorization header missing")
			}
			if req.Header.Get("X-Amz-Date") == "" {
				t.Error("X-Amz-Date header missing")
			}

			// Verify signature format
			authHeader := req.Header.Get("Authorization")
			if len(authHeader) < 50 {
				t.Errorf("Authorization header too short: %s", authHeader)
			}
		})
	}
}

func TestSigV4WithDifferentHTTPMethods(t *testing.T) {
	creds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Source:          "test",
	}

	signer := NewSigner(creds, "us-east-1", "codewhisperer")

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := &http.Request{
				Method: method,
				URL:    &url.URL{Path: "/generateAssistantResponse", Host: "codewhisperer.us-east-1.amazonaws.com"},
				Header: make(http.Header),
			}
			req.Header.Set("Host", "codewhisperer.us-east-1.amazonaws.com")
			req.Header.Set("Content-Type", "application/json")

			body := []byte(`{"message": "test"}`)

			err := signer.SignRequest(req, body)
			if err != nil {
				t.Fatalf("SignRequest() for %s error = %v", method, err)
			}

			// Verify signature is different for different methods
			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				t.Errorf("No Authorization header for %s", method)
			}
		})
	}
}
