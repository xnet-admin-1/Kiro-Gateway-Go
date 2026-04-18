package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// mockAuthManager implements the AuthManager interface for testing
type mockAuthManager struct {
	region string
	token  string
	err    error
}

func (m *mockAuthManager) GetRegion() string {
	return m.region
}

func (m *mockAuthManager) SignRequest(ctx context.Context, req *http.Request, body []byte) error {
	if m.err != nil {
		return m.err
	}
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}
	return nil
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "default config",
			config: Config{
				MaxConnections:       10,
				KeepAliveConnections: 5,
				ConnectionTimeout:    30 * time.Second,
				MaxRetries:           3,
			},
		},
		{
			name: "with logging and telemetry",
			config: Config{
				MaxConnections:       10,
				KeepAliveConnections: 5,
				ConnectionTimeout:    30 * time.Second,
				MaxRetries:           3,
				AppName:              "test-app",
				AppVersion:           "1.0.0",
				Fingerprint:          "test-fingerprint",
				OptOutTelemetry:      false,
				Logger:               log.New(os.Stdout, "", 0),
			},
		},
		{
			name: "with stalled stream protection",
			config: Config{
				MaxConnections:     10,
				ConnectionTimeout:  30 * time.Second,
				MaxRetries:         3,
				StalledStreamGrace: 2 * time.Minute,
				MinStreamSpeed:     100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authManager := &mockAuthManager{region: "us-east-1", token: "test-token"}
			client := NewClient(authManager, tt.config)

			if client == nil {
				t.Fatal("NewClient returned nil")
			}

			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}

			if client.authManager == nil {
				t.Error("authManager is nil")
			}

			if client.retryClassifier == nil {
				t.Error("retryClassifier is nil")
			}

			if client.interceptorChain == nil {
				t.Error("interceptorChain is nil")
			}

			if client.stalledStreamProtection == nil {
				t.Error("stalledStreamProtection is nil")
			}

			if client.maxRetries != tt.config.MaxRetries {
				t.Errorf("maxRetries = %d, want %d", client.maxRetries, tt.config.MaxRetries)
			}
		})
	}
}

// testTransport redirects all requests to the test server
type testTransport struct {
	server *httptest.Server
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect the request to our test server
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestClient_RetryClassifierIntegration(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(callCount *int) http.HandlerFunc
		maxRetries     int
		wantCallCount  int
		wantErr        bool
	}{
		{
			name: "retry classifier with 529 status",
			serverResponse: func(callCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*callCount++
					if *callCount < 2 {
						w.WriteHeader(529) // Service overloaded
						w.Write([]byte(`{"error": "service overloaded"}`))
					} else {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"success": true}`))
					}
				}
			},
			maxRetries:    3,
			wantCallCount: 2,
			wantErr:       false,
		},
		{
			name: "no retry on validation error",
			serverResponse: func(callCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*callCount++
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error": "ValidationException"}`))
				}
			},
			maxRetries:    3,
			wantCallCount: 1,
			wantErr:       false,
		},
		{
			name: "retry with ThrottlingException",
			serverResponse: func(callCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*callCount++
					if *callCount < 2 {
						w.WriteHeader(http.StatusTooManyRequests)
						w.Write([]byte(`{"error": "ThrottlingException"}`))
					} else {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"success": true}`))
					}
				}
			},
			maxRetries:    3,
			wantCallCount: 2,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(tt.serverResponse(&callCount))
			defer server.Close()

			authManager := &mockAuthManager{region: "us-east-1", token: "test-token"}
			config := Config{
				MaxRetries:        tt.maxRetries,
				ConnectionTimeout: 5 * time.Second,
			}
			client := NewClient(authManager, config)

			// Override the HTTP client to use our test server
			client.httpClient = &http.Client{
				Transport: &testTransport{server: server},
				Timeout:   5 * time.Second,
			}

			endpoint := "/test"
			ctx := context.Background()
			resp, err := client.Post(ctx, endpoint, map[string]string{"test": "data"})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if callCount != tt.wantCallCount {
				t.Errorf("Call count = %d, want %d", callCount, tt.wantCallCount)
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

func TestClient_InterceptorChainIntegration(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantHeaders    map[string]string
		checkUserAgent bool
	}{
		{
			name: "all interceptors enabled",
			config: Config{
				MaxRetries:      1,
				AppName:         "test-app",
				AppVersion:      "1.0.0",
				Fingerprint:     "test-fingerprint",
				OptOutTelemetry: false,
				Logger:          log.New(os.Stdout, "", 0),
				RequestID:       "test-request-123",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("x-amzn-requestid", "aws-request-456")
				w.WriteHeader(http.StatusOK)
			},
			wantHeaders: map[string]string{
				"x-amzn-codewhisperer-optout": "false",
				"X-Request-ID":                "test-request-123",
			},
			checkUserAgent: true,
		},
		{
			name: "opt-out enabled",
			config: Config{
				MaxRetries:      1,
				OptOutTelemetry: true,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantHeaders: map[string]string{
				"x-amzn-codewhisperer-optout": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check expected headers
				for key, expectedValue := range tt.wantHeaders {
					if got := r.Header.Get(key); got != expectedValue {
						t.Errorf("Header %s = %s, want %s", key, got, expectedValue)
					}
				}

				// Check user agent if enabled
				if tt.checkUserAgent {
					userAgent := r.Header.Get("User-Agent")
					if !strings.Contains(userAgent, "test-app/1.0.0") {
						t.Errorf("User-Agent should contain app name and version, got: %s", userAgent)
					}
					if !strings.Contains(userAgent, "test-fingerprint") {
						t.Errorf("User-Agent should contain fingerprint, got: %s", userAgent)
					}
				}

				tt.serverResponse(w, r)
			}))
			defer server.Close()

			authManager := &mockAuthManager{region: "us-east-1", token: "test-token"}
			client := NewClient(authManager, tt.config)

			// Override the HTTP client to use our test server
			client.httpClient = &http.Client{
				Transport: &testTransport{server: server},
				Timeout:   5 * time.Second,
			}

			endpoint := "/test"
			ctx := context.Background()
			resp, err := client.Post(ctx, endpoint, map[string]string{"test": "data"})

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			resp.Body.Close()
		})
	}
}

func TestClient_StalledStreamProtectionIntegration(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		serverResponse func(w http.ResponseWriter, r *http.Request)
		readTimeout    time.Duration
		wantErr        bool
		errContains    string
	}{
		{
			name: "normal stream with data",
			config: Config{
				MaxRetries:         1,
				StalledStreamGrace: 1 * time.Second,
				MinStreamSpeed:     1,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("data: normal stream data\n\n"))
			},
			readTimeout: 500 * time.Millisecond,
			wantErr:     false,
		},
		{
			name: "error response not wrapped",
			config: Config{
				MaxRetries:         1,
				StalledStreamGrace: 1 * time.Second,
				MinStreamSpeed:     1,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "bad request"}`))
			},
			readTimeout: 500 * time.Millisecond,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			authManager := &mockAuthManager{region: "us-east-1", token: "test-token"}
			client := NewClient(authManager, tt.config)

			// Override the HTTP client to use our test server
			client.httpClient = &http.Client{
				Transport: &testTransport{server: server},
				Timeout:   5 * time.Second,
			}

			endpoint := "/test-stream"
			ctx := context.Background()
			resp, err := client.PostStream(ctx, endpoint, map[string]string{"test": "data"})

			if err != nil {
				t.Fatalf("Unexpected error creating stream: %v", err)
			}

			// For successful streaming responses, try to read from the wrapped reader
			if resp.StatusCode < 400 {
				buf := make([]byte, 1024)
				
				// Set a timeout for the read operation
				done := make(chan error, 1)
				go func() {
					_, readErr := resp.Body.Read(buf)
					done <- readErr
				}()

				select {
				case readErr := <-done:
					if tt.wantErr {
						if readErr == nil {
							t.Error("Expected read error, got nil")
						} else if tt.errContains != "" && !strings.Contains(readErr.Error(), tt.errContains) {
							t.Errorf("Expected error containing %q, got: %v", tt.errContains, readErr)
						}
					} else if readErr != nil && readErr != io.EOF {
						t.Errorf("Unexpected read error: %v", readErr)
					}
				case <-time.After(tt.readTimeout):
					if !tt.wantErr {
						t.Error("Read operation timed out unexpectedly")
					}
				}
			}

			resp.Body.Close()
		})
	}
}

func TestClient_ErrorHandlingIntegration(t *testing.T) {
	tests := []struct {
		name           string
		authManager    *mockAuthManager
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:        "auth manager sign request error",
			authManager: &mockAuthManager{region: "us-east-1", err: errors.New("signing failed")},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:     true,
			errContains: "failed to sign request",
		},
		{
			name:        "successful request after auth",
			authManager: &mockAuthManager{region: "us-east-1", token: "valid-token"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer valid-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true}`))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			config := Config{
				MaxRetries:        1,
				ConnectionTimeout: 5 * time.Second,
			}
			client := NewClient(tt.authManager, config)

			// Override the HTTP client to use our test server
			client.httpClient = &http.Client{
				Transport: &testTransport{server: server},
				Timeout:   5 * time.Second,
			}

			endpoint := "/test"
			ctx := context.Background()
			resp, err := client.Post(ctx, endpoint, map[string]string{"test": "data"})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

func TestClient_ConfigurationValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		check  func(*Client) error
	}{
		{
			name: "default values applied",
			config: Config{
				MaxRetries: 3,
			},
			check: func(c *Client) error {
				if c.stalledStreamProtection.gracePeriod != 5*time.Minute {
					return fmt.Errorf("expected default grace period 5m, got %v", c.stalledStreamProtection.gracePeriod)
				}
				if c.stalledStreamProtection.minSpeed != 1 {
					return fmt.Errorf("expected default min speed 1, got %d", c.stalledStreamProtection.minSpeed)
				}
				return nil
			},
		},
		{
			name: "custom values preserved",
			config: Config{
				MaxRetries:         2,
				StalledStreamGrace: 2 * time.Minute,
				MinStreamSpeed:     100,
			},
			check: func(c *Client) error {
				if c.maxRetries != 2 {
					return fmt.Errorf("expected max retries 2, got %d", c.maxRetries)
				}
				if c.stalledStreamProtection.gracePeriod != 2*time.Minute {
					return fmt.Errorf("expected grace period 2m, got %v", c.stalledStreamProtection.gracePeriod)
				}
				if c.stalledStreamProtection.minSpeed != 100 {
					return fmt.Errorf("expected min speed 100, got %d", c.stalledStreamProtection.minSpeed)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authManager := &mockAuthManager{region: "us-east-1", token: "test-token"}
			client := NewClient(authManager, tt.config)

			if err := tt.check(client); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRetryClient(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(callCount *int) http.HandlerFunc
		maxRetries     int
		wantCallCount  int
		wantErr        bool
	}{
		{
			name: "successful request",
			serverResponse: func(callCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*callCount++
					w.WriteHeader(http.StatusOK)
				}
			},
			maxRetries:    3,
			wantCallCount: 1,
			wantErr:       false,
		},
		{
			name: "retry on 500",
			serverResponse: func(callCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*callCount++
					if *callCount < 3 {
						w.WriteHeader(http.StatusInternalServerError)
					} else {
						w.WriteHeader(http.StatusOK)
					}
				}
			},
			maxRetries:    3,
			wantCallCount: 3,
			wantErr:       false,
		},
		{
			name: "no retry on 400",
			serverResponse: func(callCount *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*callCount++
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			maxRetries:    3,
			wantCallCount: 1,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(tt.serverResponse(&callCount))
			defer server.Close()

			retryClient := &RetryClient{
				HTTPClient: &http.Client{Timeout: 5 * time.Second},
				MaxRetries: tt.maxRetries,
				BaseDelay:  10 * time.Millisecond, // Short delay for testing
			}

			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := retryClient.Do(req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if callCount != tt.wantCallCount {
				t.Errorf("Call count = %d, want %d", callCount, tt.wantCallCount)
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}
