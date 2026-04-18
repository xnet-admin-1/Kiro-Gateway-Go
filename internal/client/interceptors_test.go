package client

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInterceptorChain_BeforeRequest(t *testing.T) {
	tests := []struct {
		name          string
		interceptors  []Interceptor
		expectError   bool
		errorContains string
	}{
		{
			name:         "empty chain",
			interceptors: []Interceptor{},
			expectError:  false,
		},
		{
			name: "single interceptor success",
			interceptors: []Interceptor{
				&mockInterceptor{beforeRequestErr: nil},
			},
			expectError: false,
		},
		{
			name: "single interceptor error",
			interceptors: []Interceptor{
				&mockInterceptor{beforeRequestErr: errors.New("test error")},
			},
			expectError:   true,
			errorContains: "interceptor BeforeRequest failed",
		},
		{
			name: "multiple interceptors success",
			interceptors: []Interceptor{
				&mockInterceptor{beforeRequestErr: nil},
				&mockInterceptor{beforeRequestErr: nil},
			},
			expectError: false,
		},
		{
			name: "first interceptor fails",
			interceptors: []Interceptor{
				&mockInterceptor{beforeRequestErr: errors.New("first error")},
				&mockInterceptor{beforeRequestErr: nil},
			},
			expectError:   true,
			errorContains: "first error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewInterceptorChain(tt.interceptors...)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := chain.BeforeRequest(req)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInterceptorChain_AfterResponse(t *testing.T) {
	tests := []struct {
		name          string
		interceptors  []Interceptor
		expectError   bool
		errorContains string
	}{
		{
			name:         "empty chain",
			interceptors: []Interceptor{},
			expectError:  false,
		},
		{
			name: "single interceptor success",
			interceptors: []Interceptor{
				&mockInterceptor{afterResponseErr: nil},
			},
			expectError: false,
		},
		{
			name: "single interceptor error",
			interceptors: []Interceptor{
				&mockInterceptor{afterResponseErr: errors.New("test error")},
			},
			expectError:   true,
			errorContains: "interceptor AfterResponse failed",
		},
		{
			name: "multiple interceptors success",
			interceptors: []Interceptor{
				&mockInterceptor{afterResponseErr: nil},
				&mockInterceptor{afterResponseErr: nil},
			},
			expectError: false,
		},
		{
			name: "last interceptor fails (reverse order)",
			interceptors: []Interceptor{
				&mockInterceptor{afterResponseErr: nil},
				&mockInterceptor{afterResponseErr: errors.New("last error")},
			},
			expectError:   true,
			errorContains: "last error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewInterceptorChain(tt.interceptors...)
			resp := &http.Response{StatusCode: 200}

			err := chain.AfterResponse(resp, nil)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInterceptorChain_NewInterceptorChain(t *testing.T) {
	interceptor1 := &mockInterceptor{}
	interceptor2 := &mockInterceptor{}
	
	chain := NewInterceptorChain(interceptor1, interceptor2)
	
	if len(chain.interceptors) != 2 {
		t.Errorf("Expected 2 interceptors, got %d", len(chain.interceptors))
	}
}

func TestInterceptorChain_Add(t *testing.T) {
	chain := NewInterceptorChain()
	interceptor := &mockInterceptor{}

	if len(chain.interceptors) != 0 {
		t.Errorf("expected 0 interceptors, got %d", len(chain.interceptors))
	}

	chain.Add(interceptor)

	if len(chain.interceptors) != 1 {
		t.Errorf("expected 1 interceptor, got %d", len(chain.interceptors))
	}
}

func TestLoggingInterceptor_BeforeRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		headers        map[string]string
		expectInLog    []string
		expectNotInLog []string
	}{
		{
			name:   "basic request",
			method: "GET",
			url:    "http://example.com/api",
			expectInLog: []string{
				"GET http://example.com/api",
			},
		},
		{
			name:   "request with headers",
			method: "POST",
			url:    "http://example.com/api",
			headers: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "value",
			},
			expectInLog: []string{
				"POST http://example.com/api",
				"Content-Type: [application/json]",
				"X-Custom: [value]",
			},
		},
		{
			name:   "request with sensitive headers",
			method: "POST",
			url:    "http://example.com/api",
			headers: map[string]string{
				"Authorization":        "Bearer token123",
				"X-Amz-Security-Token": "session123",
				"Cookie":               "session=abc123",
				"Content-Type":         "application/json",
			},
			expectInLog: []string{
				"POST http://example.com/api",
				"Authorization: [REDACTED]",
				"X-Amz-Security-Token: [REDACTED]",
				"Cookie: [REDACTED]",
				"Content-Type: [application/json]",
			},
			expectNotInLog: []string{
				"token123",
				"session123",
				"abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := log.New(&buf, "", 0)
			interceptor := NewLoggingInterceptor(logger)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			err := interceptor.BeforeRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			logOutput := buf.String()

			for _, expected := range tt.expectInLog {
				if !strings.Contains(logOutput, expected) {
					t.Errorf("expected log to contain %q, got: %s", expected, logOutput)
				}
			}

			for _, notExpected := range tt.expectNotInLog {
				if strings.Contains(logOutput, notExpected) {
					t.Errorf("expected log NOT to contain %q, got: %s", notExpected, logOutput)
				}
			}
		})
	}
}

func TestLoggingInterceptor_AfterResponse(t *testing.T) {
	tests := []struct {
		name        string
		resp        *http.Response
		err         error
		expectInLog []string
	}{
		{
			name: "successful response",
			resp: &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
			},
			expectInLog: []string{
				"Response: 200 200 OK",
			},
		},
		{
			name: "error response",
			err:  errors.New("connection failed"),
			expectInLog: []string{
				"Error: connection failed",
			},
		},
		{
			name: "nil response and error",
			expectInLog: []string{
				// Should not log anything
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := log.New(&buf, "", 0)
			interceptor := NewLoggingInterceptor(logger)

			err := interceptor.AfterResponse(tt.resp, tt.err)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			logOutput := buf.String()

			for _, expected := range tt.expectInLog {
				if expected != "" && !strings.Contains(logOutput, expected) {
					t.Errorf("expected log to contain %q, got: %s", expected, logOutput)
				}
			}
		})
	}
}

func TestLoggingInterceptor_isSensitiveHeader(t *testing.T) {
	interceptor := NewLoggingInterceptor(nil)

	tests := []struct {
		header    string
		sensitive bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"X-Amz-Security-Token", true},
		{"x-amz-security-token", true},
		{"Cookie", true},
		{"Set-Cookie", true},
		{"X-API-Key", true},
		{"X-Auth-Token", true},
		{"Content-Type", false},
		{"User-Agent", false},
		{"X-Custom-Header", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := interceptor.isSensitiveHeader(tt.header)
			if result != tt.sensitive {
				t.Errorf("isSensitiveHeader(%q) = %v, want %v", tt.header, result, tt.sensitive)
			}
		})
	}
}

func TestLoggingInterceptor_NilLogger(t *testing.T) {
	interceptor := NewLoggingInterceptor(nil)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	// Should not panic with nil logger
	err := interceptor.BeforeRequest(req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	err = interceptor.AfterResponse(nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTelemetryInterceptor_BeforeRequest(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:    "disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewTelemetryInterceptor("test-app", "1.0.0", tt.enabled)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := interceptor.BeforeRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			startTime := req.Header.Get("X-Request-Start-Time")
			if tt.enabled {
				if startTime == "" {
					t.Error("expected start time header to be set when enabled")
				}
			} else {
				if startTime != "" {
					t.Error("expected start time header NOT to be set when disabled")
				}
			}
		})
	}
}

func TestTelemetryInterceptor_AfterResponse(t *testing.T) {
	interceptor := NewTelemetryInterceptor("test-app", "1.0.0", true)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Request-Start-Time", fmt.Sprintf("%d", time.Now().UnixNano()))

	resp := &http.Response{
		StatusCode: 200,
		Request:    req,
	}

	err := interceptor.AfterResponse(resp, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptOutInterceptor_BeforeRequest(t *testing.T) {
	tests := []struct {
		name           string
		optOut         bool
		expectedHeader string
	}{
		{
			name:           "opt out enabled",
			optOut:         true,
			expectedHeader: "true",
		},
		{
			name:           "opt out disabled",
			optOut:         false,
			expectedHeader: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewOptOutInterceptor(tt.optOut)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := interceptor.BeforeRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			header := req.Header.Get("x-amzn-codewhisperer-optout")
			if header != tt.expectedHeader {
				t.Errorf("expected header %q, got %q", tt.expectedHeader, header)
			}
		})
	}
}

func TestOptOutInterceptor_AfterResponse(t *testing.T) {
	interceptor := NewOptOutInterceptor(true)
	err := interceptor.AfterResponse(nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUserAgentInterceptor_BeforeRequest(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		appVersion  string
		fingerprint string
	}{
		{
			name:        "basic user agent",
			appName:     "kiro-gateway",
			appVersion:  "1.0.0",
			fingerprint: "abc123",
		},
		{
			name:        "empty fingerprint",
			appName:     "test-app",
			appVersion:  "2.0.0",
			fingerprint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewUserAgentInterceptor(tt.appName, tt.appVersion, tt.fingerprint)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := interceptor.BeforeRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			userAgent := req.Header.Get("User-Agent")
			if userAgent == "" {
				t.Error("expected User-Agent header to be set")
			}

			expectedParts := []string{
				tt.appName,
				tt.appVersion,
				runtime.GOOS,
				runtime.GOARCH,
				runtime.Version(),
			}

			for _, part := range expectedParts {
				if !strings.Contains(userAgent, part) {
					t.Errorf("expected User-Agent %q to contain %q", userAgent, part)
				}
			}

			if tt.fingerprint != "" && !strings.Contains(userAgent, tt.fingerprint) {
				t.Errorf("expected User-Agent %q to contain fingerprint %q", userAgent, tt.fingerprint)
			}
		})
	}
}

func TestUserAgentInterceptor_AfterResponse(t *testing.T) {
	interceptor := NewUserAgentInterceptor("test", "1.0", "abc")
	err := interceptor.AfterResponse(nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRequestIDInterceptor_AfterResponse(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectInLog    string
		expectNotInLog bool
	}{
		{
			name: "x-amzn-requestid header",
			headers: map[string]string{
				"x-amzn-requestid": "req-123",
			},
			expectInLog: "Request ID: req-123",
		},
		{
			name: "x-amzn-request-id header",
			headers: map[string]string{
				"x-amzn-request-id": "req-456",
			},
			expectInLog: "Request ID: req-456",
		},
		{
			name: "x-request-id header",
			headers: map[string]string{
				"x-request-id": "req-789",
			},
			expectInLog: "Request ID: req-789",
		},
		{
			name: "multiple headers - first wins",
			headers: map[string]string{
				"x-amzn-requestid": "req-first",
				"x-request-id":     "req-second",
			},
			expectInLog: "Request ID: req-first",
		},
		{
			name:           "no request id headers",
			headers:        map[string]string{},
			expectNotInLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := log.New(&buf, "", 0)
			interceptor := NewRequestIDInterceptor(logger)

			resp := &http.Response{
				Header: make(http.Header),
			}
			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			err := interceptor.AfterResponse(resp, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			logOutput := buf.String()

			if tt.expectNotInLog {
				if strings.Contains(logOutput, "Request ID:") {
					t.Errorf("expected no request ID in log, got: %s", logOutput)
				}
			} else {
				if !strings.Contains(logOutput, tt.expectInLog) {
					t.Errorf("expected log to contain %q, got: %s", tt.expectInLog, logOutput)
				}
			}
		})
	}
}

func TestRequestIDInterceptor_BeforeRequest(t *testing.T) {
	interceptor := NewRequestIDInterceptor(nil)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := interceptor.BeforeRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRequestIDInterceptor_NilLogger(t *testing.T) {
	interceptor := NewRequestIDInterceptor(nil)
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("x-amzn-requestid", "test-id")
	
	// Should not panic with nil logger
	err := interceptor.AfterResponse(resp, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRetryHeaderInterceptor_BeforeRequest(t *testing.T) {
	tests := []struct {
		name            string
		attempt         int
		expectHeader    bool
		expectedValue   string
	}{
		{
			name:         "no retry attempt",
			attempt:      0,
			expectHeader: false,
		},
		{
			name:          "first retry",
			attempt:       1,
			expectHeader:  true,
			expectedValue: "1",
		},
		{
			name:          "third retry",
			attempt:       3,
			expectHeader:  true,
			expectedValue: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewRetryHeaderInterceptor()
			interceptor.SetAttempt(tt.attempt)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := interceptor.BeforeRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			header := req.Header.Get("X-Retry-Attempt")
			if tt.expectHeader {
				if header != tt.expectedValue {
					t.Errorf("expected header %q, got %q", tt.expectedValue, header)
				}
			} else {
				if header != "" {
					t.Errorf("expected no header, got %q", header)
				}
			}
		})
	}
}

func TestRetryHeaderInterceptor_AfterResponse(t *testing.T) {
	interceptor := NewRetryHeaderInterceptor()
	err := interceptor.AfterResponse(nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRetryHeaderInterceptor_SetAttempt(t *testing.T) {
	interceptor := NewRetryHeaderInterceptor()
	
	if interceptor.attempt != 0 {
		t.Errorf("expected initial attempt to be 0, got %d", interceptor.attempt)
	}

	interceptor.SetAttempt(5)
	
	if interceptor.attempt != 5 {
		t.Errorf("expected attempt to be 5, got %d", interceptor.attempt)
	}
}

// mockInterceptor is a test helper that implements the Interceptor interface
type mockInterceptor struct {
	beforeRequestErr  error
	afterResponseErr  error
	beforeRequestCalls int
	afterResponseCalls int
}

func (m *mockInterceptor) BeforeRequest(req *http.Request) error {
	m.beforeRequestCalls++
	return m.beforeRequestErr
}

func (m *mockInterceptor) AfterResponse(resp *http.Response, err error) error {
	m.afterResponseCalls++
	return m.afterResponseErr
}

// Test interceptor chain execution order
func TestInterceptorChain_ExecutionOrder(t *testing.T) {
	var executionOrder []string
	
	interceptor1 := &orderTrackingInterceptor{name: "first", order: &executionOrder}
	interceptor2 := &orderTrackingInterceptor{name: "second", order: &executionOrder}
	interceptor3 := &orderTrackingInterceptor{name: "third", order: &executionOrder}
	
	chain := NewInterceptorChain(interceptor1, interceptor2, interceptor3)
	req := httptest.NewRequest("GET", "http://example.com", nil)
	resp := &http.Response{StatusCode: 200}
	
	// Test BeforeRequest order (should be forward)
	err := chain.BeforeRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	expectedBeforeOrder := []string{"first-before", "second-before", "third-before"}
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 before calls, got %d", len(executionOrder))
	}
	for i, expected := range expectedBeforeOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("expected order[%d] = %q, got %q", i, expected, executionOrder[i])
		}
	}
	
	// Reset and test AfterResponse order (should be reverse)
	executionOrder = []string{}
	err = chain.AfterResponse(resp, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	expectedAfterOrder := []string{"third-after", "second-after", "first-after"}
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 after calls, got %d", len(executionOrder))
	}
	for i, expected := range expectedAfterOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("expected order[%d] = %q, got %q", i, expected, executionOrder[i])
		}
	}
}

// orderTrackingInterceptor tracks the order of execution for testing
type orderTrackingInterceptor struct {
	name  string
	order *[]string
}

func (o *orderTrackingInterceptor) BeforeRequest(req *http.Request) error {
	*o.order = append(*o.order, o.name+"-before")
	return nil
}

func (o *orderTrackingInterceptor) AfterResponse(resp *http.Response, err error) error {
	*o.order = append(*o.order, o.name+"-after")
	return nil
}
