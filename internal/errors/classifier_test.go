package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		body         []byte
		originalErr  error
		expectedKind ErrorKind
		expectedMsg  string
		expectedID   string
	}{
		{
			name:         "throttling error 429",
			statusCode:   429,
			body:         []byte(`{"error": "Too many requests", "requestId": "req-123"}`),
			expectedKind: ErrorKindThrottling,
			expectedMsg:  "Too many requests",
			expectedID:   "req-123",
		},
		{
			name:         "model overloaded 429",
			statusCode:   429,
			body:         []byte(`{"error": "INSUFFICIENT_MODEL_CAPACITY", "requestId": "req-456"}`),
			expectedKind: ErrorKindModelOverloaded,
			expectedMsg:  "Model is currently overloaded",
			expectedID:   "req-456",
		},
		{
			name:         "context window overflow 400",
			statusCode:   400,
			body:         []byte(`{"error": "Input is too long", "requestId": "req-789"}`),
			expectedKind: ErrorKindContextWindowOverflow,
			expectedMsg:  "Input is too long for the model context",
			expectedID:   "req-789",
		},
		{
			name:         "monthly limit reached",
			statusCode:   400,
			body:         []byte(`{"error": "MONTHLY_REQUEST_COUNT exceeded", "requestId": "req-abc"}`),
			expectedKind: ErrorKindMonthlyLimitReached,
			expectedMsg:  "Monthly usage limit reached",
			expectedID:   "req-abc",
		},
		{
			name:         "authentication failed 401",
			statusCode:   401,
			body:         []byte(`{"error": "Unauthorized", "requestId": "req-def"}`),
			expectedKind: ErrorKindAuthenticationFailed,
			expectedMsg:  "Authentication failed",
			expectedID:   "req-def",
		},
		{
			name:         "authentication failed 403",
			statusCode:   403,
			body:         []byte(`{"error": "Forbidden", "requestId": "req-ghi"}`),
			expectedKind: ErrorKindAuthenticationFailed,
			expectedMsg:  "Authentication failed",
			expectedID:   "req-ghi",
		},
		{
			name:         "invalid request 400",
			statusCode:   400,
			body:         []byte(`{"error": "Bad request", "requestId": "req-jkl"}`),
			expectedKind: ErrorKindInvalidRequest,
			expectedMsg:  "Invalid request",
			expectedID:   "req-jkl",
		},
		{
			name:         "internal server error 500",
			statusCode:   500,
			body:         []byte(`{"error": "Internal error", "requestId": "req-mno"}`),
			expectedKind: ErrorKindInternalServerError,
			expectedMsg:  "Internal server error",
			expectedID:   "req-mno",
		},
		{
			name:         "bad gateway 502",
			statusCode:   502,
			body:         []byte(`{"error": "Bad gateway", "requestId": "req-pqr"}`),
			expectedKind: ErrorKindInternalServerError,
			expectedMsg:  "Internal server error",
			expectedID:   "req-pqr",
		},
		{
			name:         "service unavailable 503",
			statusCode:   503,
			body:         []byte(`{"error": "Service unavailable", "requestId": "req-stu"}`),
			expectedKind: ErrorKindInternalServerError,
			expectedMsg:  "Internal server error",
			expectedID:   "req-stu",
		},
		{
			name:         "gateway timeout 504",
			statusCode:   504,
			body:         []byte(`{"error": "Gateway timeout", "requestId": "req-vwx"}`),
			expectedKind: ErrorKindInternalServerError,
			expectedMsg:  "Internal server error",
			expectedID:   "req-vwx",
		},
		{
			name:         "unknown error with original error",
			statusCode:   418,
			body:         []byte(`{"error": "I'm a teapot"}`),
			originalErr:  errors.New("network error"),
			expectedKind: ErrorKindUnknown,
			expectedMsg:  "network error",
			expectedID:   "",
		},
		{
			name:         "unknown error without original error",
			statusCode:   418,
			body:         []byte(`{"error": "I'm a teapot"}`),
			expectedKind: ErrorKindUnknown,
			expectedMsg:  "Unknown error",
			expectedID:   "",
		},
		{
			name:         "case insensitive pattern matching",
			statusCode:   200,
			body:         []byte(`{"error": "input is too long for processing", "requestId": "req-case"}`),
			expectedKind: ErrorKindContextWindowOverflow,
			expectedMsg:  "Input is too long for the model context",
			expectedID:   "req-case",
		},
		{
			name:         "case insensitive model capacity",
			statusCode:   200,
			body:         []byte(`{"error": "insufficient_model_capacity detected", "requestId": "req-capacity"}`),
			expectedKind: ErrorKindModelOverloaded,
			expectedMsg:  "Model is currently overloaded",
			expectedID:   "req-capacity",
		},
		{
			name:         "case insensitive monthly limit",
			statusCode:   200,
			body:         []byte(`{"error": "monthly_request_count limit exceeded", "requestId": "req-monthly"}`),
			expectedKind: ErrorKindMonthlyLimitReached,
			expectedMsg:  "Monthly usage limit reached",
			expectedID:   "req-monthly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.statusCode, tt.body, tt.originalErr)

			if result.Kind != tt.expectedKind {
				t.Errorf("ClassifyError() Kind = %v, want %v", result.Kind, tt.expectedKind)
			}

			if result.Message != tt.expectedMsg {
				t.Errorf("ClassifyError() Message = %v, want %v", result.Message, tt.expectedMsg)
			}

			if result.RequestID != tt.expectedID {
				t.Errorf("ClassifyError() RequestID = %v, want %v", result.RequestID, tt.expectedID)
			}

			if result.StatusCode != tt.statusCode {
				t.Errorf("ClassifyError() StatusCode = %v, want %v", result.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestExtractRequestID(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "valid request ID",
			body:     `{"error": "test", "requestId": "req-123"}`,
			expected: "req-123",
		},
		{
			name:     "request ID with spaces",
			body:     `{"error": "test", "requestId" : "req-456" }`,
			expected: "req-456",
		},
		{
			name:     "no request ID",
			body:     `{"error": "test"}`,
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name:     "malformed JSON",
			body:     `{"error": "test", "requestId": "req-789"`,
			expected: "req-789",
		},
		{
			name:     "multiple request IDs - first match",
			body:     `{"error": "test", "requestId": "req-first", "anotherRequestId": "req-second"}`,
			expected: "req-first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRequestID(tt.body)
			if result != tt.expected {
				t.Errorf("extractRequestID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "Input is too long",
			substr:   "Input is too long",
			expected: true,
		},
		{
			name:     "case insensitive match",
			s:        "INPUT IS TOO LONG",
			substr:   "input is too long",
			expected: true,
		},
		{
			name:     "partial match",
			s:        "The input is too long for processing",
			substr:   "input is too long",
			expected: true,
		},
		{
			name:     "no match",
			s:        "Everything is fine",
			substr:   "input is too long",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "test string",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("containsIgnoreCase() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyHTTPError(t *testing.T) {
	tests := []struct {
		name         string
		resp         *http.Response
		body         []byte
		originalErr  error
		expectedKind ErrorKind
	}{
		{
			name: "valid HTTP response",
			resp: &http.Response{
				StatusCode: 429,
			},
			body:         []byte(`{"error": "Too many requests", "requestId": "req-123"}`),
			expectedKind: ErrorKindThrottling,
		},
		{
			name:         "nil HTTP response",
			resp:         nil,
			body:         []byte(`{"error": "Network error"}`),
			originalErr:  errors.New("connection failed"),
			expectedKind: ErrorKindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyHTTPError(tt.resp, tt.body, tt.originalErr)
			if result.Kind != tt.expectedKind {
				t.Errorf("ClassifyHTTPError() Kind = %v, want %v", result.Kind, tt.expectedKind)
			}
		})
	}
}

func TestFormatErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		apiErr     *APIError
		statusCode int
		expected   map[string]interface{}
	}{
		{
			name: "error with request ID",
			apiErr: &APIError{
				Kind:      ErrorKindThrottling,
				Message:   "Too many requests",
				RequestID: "req-123",
			},
			statusCode: 429,
			expected: map[string]interface{}{
				"error": map[string]interface{}{
					"message":    "Too many requests have been sent recently. Please wait and try again later.",
					"type":       "Throttling",
					"code":       429,
					"request_id": "req-123",
				},
			},
		},
		{
			name: "error without request ID",
			apiErr: &APIError{
				Kind:    ErrorKindInvalidRequest,
				Message: "Bad request",
			},
			statusCode: 400,
			expected: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "The request is invalid. Please check your input and try again.",
					"type":    "InvalidRequest",
					"code":    400,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatErrorResponse(tt.apiErr, tt.statusCode)
			
			// Check top-level structure
			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected error object in response")
			}
			
			expectedError := tt.expected["error"].(map[string]interface{})
			
			// Check each field
			if errorObj["message"] != expectedError["message"] {
				t.Errorf("Expected message %v, got %v", expectedError["message"], errorObj["message"])
			}
			
			if errorObj["type"] != expectedError["type"] {
				t.Errorf("Expected type %v, got %v", expectedError["type"], errorObj["type"])
			}
			
			if errorObj["code"] != expectedError["code"] {
				t.Errorf("Expected code %v, got %v", expectedError["code"], errorObj["code"])
			}
			
			// Check request_id if expected
			if expectedRequestID, exists := expectedError["request_id"]; exists {
				if errorObj["request_id"] != expectedRequestID {
					t.Errorf("Expected request_id %v, got %v", expectedRequestID, errorObj["request_id"])
				}
			} else {
				if _, exists := errorObj["request_id"]; exists {
					t.Error("Did not expect request_id in response")
				}
			}
		})
	}
}

// Benchmark tests
// Additional edge case tests
func TestClassifyError_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		body         []byte
		originalErr  error
		expectedKind ErrorKind
	}{
		{
			name:         "empty body",
			statusCode:   500,
			body:         []byte{},
			expectedKind: ErrorKindInternalServerError,
		},
		{
			name:         "nil body",
			statusCode:   400,
			body:         nil,
			expectedKind: ErrorKindInvalidRequest,
		},
		{
			name:         "malformed JSON body",
			statusCode:   200,
			body:         []byte(`{"error": "INSUFFICIENT_MODEL_CAPACITY", "requestId"`),
			expectedKind: ErrorKindModelOverloaded,
		},
		{
			name:         "multiple patterns in body - first match wins",
			statusCode:   200,
			body:         []byte(`{"error": "MONTHLY_REQUEST_COUNT and Input is too long", "requestId": "req-multi"}`),
			expectedKind: ErrorKindMonthlyLimitReached,
		},
		{
			name:         "pattern in different JSON field",
			statusCode:   200,
			body:         []byte(`{"message": "Input is too long", "details": "context overflow", "requestId": "req-field"}`),
			expectedKind: ErrorKindContextWindowOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.statusCode, tt.body, tt.originalErr)
			if result.Kind != tt.expectedKind {
				t.Errorf("ClassifyError() Kind = %v, want %v", result.Kind, tt.expectedKind)
			}
		})
	}
}

func TestClassifyError_PriorityOrder(t *testing.T) {
	// Test that body patterns take precedence over status codes
	tests := []struct {
		name         string
		statusCode   int
		body         string
		expectedKind ErrorKind
	}{
		{
			name:         "monthly limit overrides 429 status",
			statusCode:   429,
			body:         `{"error": "MONTHLY_REQUEST_COUNT exceeded"}`,
			expectedKind: ErrorKindMonthlyLimitReached,
		},
		{
			name:         "context overflow overrides 400 status",
			statusCode:   400,
			body:         `{"error": "Input is too long for processing"}`,
			expectedKind: ErrorKindContextWindowOverflow,
		},
		{
			name:         "model capacity overrides 500 status",
			statusCode:   500,
			body:         `{"error": "INSUFFICIENT_MODEL_CAPACITY detected"}`,
			expectedKind: ErrorKindModelOverloaded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.statusCode, []byte(tt.body), nil)
			if result.Kind != tt.expectedKind {
				t.Errorf("ClassifyError() Kind = %v, want %v", result.Kind, tt.expectedKind)
			}
		})
	}
}

// Benchmark tests
func BenchmarkClassifyError(b *testing.B) {
	body := []byte(`{"error": "INSUFFICIENT_MODEL_CAPACITY", "requestId": "req-bench"}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyError(429, body, nil)
	}
}

func BenchmarkExtractRequestID(b *testing.B) {
	body := `{"error": "test error", "requestId": "req-benchmark-123", "timestamp": "2026-01-22T14:58:33Z"}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractRequestID(body)
	}
}
