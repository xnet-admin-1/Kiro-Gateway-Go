package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/kiro-gateway-go/internal/errors"
)

func TestWriteAPIError(t *testing.T) {
	tests := []struct {
		name           string
		apiErr         *errors.APIError
		wantStatusCode int
		wantErrorType  string
		wantRequestID  string
	}{
		{
			name: "Throttling error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindThrottling,
				Message:    "Too many requests",
				RequestID:  "req-123",
				StatusCode: 429,
			},
			wantStatusCode: 429,
			wantErrorType:  "Throttling",
			wantRequestID:  "req-123",
		},
		{
			name: "Context window overflow error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindContextWindowOverflow,
				Message:    "Input is too long",
				RequestID:  "req-456",
				StatusCode: 400,
			},
			wantStatusCode: 400,
			wantErrorType:  "ContextWindowOverflow",
			wantRequestID:  "req-456",
		},
		{
			name: "Model overloaded error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindModelOverloaded,
				Message:    "Model is overloaded",
				RequestID:  "req-789",
				StatusCode: 429,
			},
			wantStatusCode: 429,
			wantErrorType:  "ModelOverloaded",
			wantRequestID:  "req-789",
		},
		{
			name: "Monthly limit reached error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindMonthlyLimitReached,
				Message:    "Monthly limit reached",
				RequestID:  "req-abc",
				StatusCode: 429,
			},
			wantStatusCode: 429,
			wantErrorType:  "MonthlyLimitReached",
			wantRequestID:  "req-abc",
		},
		{
			name: "Authentication failed error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindAuthenticationFailed,
				Message:    "Authentication failed",
				RequestID:  "req-def",
				StatusCode: 401,
			},
			wantStatusCode: 401,
			wantErrorType:  "AuthenticationFailed",
			wantRequestID:  "req-def",
		},
		{
			name: "Invalid request error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindInvalidRequest,
				Message:    "Invalid request",
				RequestID:  "req-ghi",
				StatusCode: 400,
			},
			wantStatusCode: 400,
			wantErrorType:  "InvalidRequest",
			wantRequestID:  "req-ghi",
		},
		{
			name: "Internal server error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindInternalServerError,
				Message:    "Internal server error",
				RequestID:  "req-jkl",
				StatusCode: 500,
			},
			wantStatusCode: 500,
			wantErrorType:  "InternalServerError",
			wantRequestID:  "req-jkl",
		},
		{
			name: "Unknown error",
			apiErr: &errors.APIError{
				Kind:       errors.ErrorKindUnknown,
				Message:    "Unknown error",
				RequestID:  "req-mno",
				StatusCode: 500,
			},
			wantStatusCode: 500,
			wantErrorType:  "Unknown",
			wantRequestID:  "req-mno",
		},
		{
			name: "Error without status code - throttling",
			apiErr: &errors.APIError{
				Kind:      errors.ErrorKindThrottling,
				Message:   "Too many requests",
				RequestID: "req-pqr",
			},
			wantStatusCode: 429,
			wantErrorType:  "Throttling",
			wantRequestID:  "req-pqr",
		},
		{
			name: "Error without status code - context overflow",
			apiErr: &errors.APIError{
				Kind:      errors.ErrorKindContextWindowOverflow,
				Message:   "Input too long",
				RequestID: "req-stu",
			},
			wantStatusCode: 400,
			wantErrorType:  "ContextWindowOverflow",
			wantRequestID:  "req-stu",
		},
		{
			name: "Error without status code - auth failed",
			apiErr: &errors.APIError{
				Kind:      errors.ErrorKindAuthenticationFailed,
				Message:   "Auth failed",
				RequestID: "req-vwx",
			},
			wantStatusCode: 401,
			wantErrorType:  "AuthenticationFailed",
			wantRequestID:  "req-vwx",
		},
		{
			name: "Error without status code - unknown",
			apiErr: &errors.APIError{
				Kind:      errors.ErrorKindUnknown,
				Message:   "Unknown",
				RequestID: "req-yz",
			},
			wantStatusCode: 500,
			wantErrorType:  "Unknown",
			wantRequestID:  "req-yz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response recorder
			w := httptest.NewRecorder()

			// Call writeAPIError
			writeAPIError(w, tt.apiErr)

			// Check status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("writeAPIError() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("writeAPIError() Content-Type = %v, want application/json", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			// Check error object exists
			errorObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Response does not contain error object")
			}

			// Check error type
			errorType, ok := errorObj["type"].(string)
			if !ok {
				t.Fatal("Error object does not contain type field")
			}
			if errorType != tt.wantErrorType {
				t.Errorf("writeAPIError() error type = %v, want %v", errorType, tt.wantErrorType)
			}

			// Check error code
			errorCode, ok := errorObj["code"].(float64)
			if !ok {
				t.Fatal("Error object does not contain code field")
			}
			if int(errorCode) != tt.wantStatusCode {
				t.Errorf("writeAPIError() error code = %v, want %v", int(errorCode), tt.wantStatusCode)
			}

			// Check error message exists and is not empty
			errorMessage, ok := errorObj["message"].(string)
			if !ok {
				t.Fatal("Error object does not contain message field")
			}
			if errorMessage == "" {
				t.Error("writeAPIError() error message is empty")
			}

			// Check request ID if present
			if tt.wantRequestID != "" {
				requestID, ok := errorObj["request_id"].(string)
				if !ok {
					t.Fatal("Error object does not contain request_id field")
				}
				if requestID != tt.wantRequestID {
					t.Errorf("writeAPIError() request_id = %v, want %v", requestID, tt.wantRequestID)
				}
			}
		})
	}
}

func TestWriteAPIError_UserFriendlyMessages(t *testing.T) {
	tests := []struct {
		name        string
		kind        errors.ErrorKind
		wantContain string
	}{
		{
			name:        "Throttling message",
			kind:        errors.ErrorKindThrottling,
			wantContain: "Too many requests",
		},
		{
			name:        "Context overflow message",
			kind:        errors.ErrorKindContextWindowOverflow,
			wantContain: "too long",
		},
		{
			name:        "Model overloaded message",
			kind:        errors.ErrorKindModelOverloaded,
			wantContain: "overloaded",
		},
		{
			name:        "Monthly limit message",
			kind:        errors.ErrorKindMonthlyLimitReached,
			wantContain: "monthly",
		},
		{
			name:        "Auth failed message",
			kind:        errors.ErrorKindAuthenticationFailed,
			wantContain: "Authentication",
		},
		{
			name:        "Invalid request message",
			kind:        errors.ErrorKindInvalidRequest,
			wantContain: "invalid",
		},
		{
			name:        "Internal server error message",
			kind:        errors.ErrorKindInternalServerError,
			wantContain: "server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			apiErr := &errors.APIError{
				Kind:       tt.kind,
				Message:    "Test error",
				StatusCode: 500,
			}

			writeAPIError(w, apiErr)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			errorObj := response["error"].(map[string]interface{})
			message := errorObj["message"].(string)

			// Check that message contains expected text (case insensitive)
			if !bytes.Contains([]byte(bytes.ToLower([]byte(message))), bytes.ToLower([]byte(tt.wantContain))) {
				t.Errorf("writeAPIError() message = %v, want to contain %v", message, tt.wantContain)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		message        string
		err            error
		wantStatusCode int
		wantMessage    string
	}{
		{
			name:           "Error with cause",
			statusCode:     400,
			message:        "Invalid request",
			err:            http.ErrBodyNotAllowed,
			wantStatusCode: 400,
			wantMessage:    "Invalid request: http: request method or response status code does not allow body",
		},
		{
			name:           "Error without cause",
			statusCode:     500,
			message:        "Internal error",
			err:            nil,
			wantStatusCode: 500,
			wantMessage:    "Internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			writeError(w, tt.statusCode, tt.message, tt.err)

			if w.Code != tt.wantStatusCode {
				t.Errorf("writeError() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			errorObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Response does not contain error object")
			}

			message, ok := errorObj["message"].(string)
			if !ok {
				t.Fatal("Error object does not contain message field")
			}

			if message != tt.wantMessage {
				t.Errorf("writeError() message = %v, want %v", message, tt.wantMessage)
			}
		})
	}
}

func TestErrorResponseFormat(t *testing.T) {
	// Test that error responses follow OpenAI format
	w := httptest.NewRecorder()
	apiErr := &errors.APIError{
		Kind:       errors.ErrorKindThrottling,
		Message:    "Rate limit exceeded",
		RequestID:  "req-test",
		StatusCode: 429,
	}

	writeAPIError(w, apiErr)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	// Check top-level structure
	if _, ok := response["error"]; !ok {
		t.Error("Response missing 'error' field")
	}

	errorObj := response["error"].(map[string]interface{})

	// Check required fields
	requiredFields := []string{"message", "type", "code"}
	for _, field := range requiredFields {
		if _, ok := errorObj[field]; !ok {
			t.Errorf("Error object missing required field: %s", field)
		}
	}

	// Check optional request_id field
	if _, ok := errorObj["request_id"]; !ok {
		t.Error("Error object missing request_id field")
	}
}

func TestErrorClassificationIntegration(t *testing.T) {
	// Test that error classification works end-to-end
	tests := []struct {
		name           string
		statusCode     int
		body           []byte
		wantKind       errors.ErrorKind
		wantStatusCode int
	}{
		{
			name:           "Throttling error from API",
			statusCode:     429,
			body:           []byte(`{"message": "Too many requests"}`),
			wantKind:       errors.ErrorKindThrottling,
			wantStatusCode: 429,
		},
		{
			name:           "Model overloaded from API",
			statusCode:     429,
			body:           []byte(`{"error": "INSUFFICIENT_MODEL_CAPACITY"}`),
			wantKind:       errors.ErrorKindModelOverloaded,
			wantStatusCode: 429,
		},
		{
			name:           "Context overflow from API",
			statusCode:     400,
			body:           []byte(`{"message": "Input is too long"}`),
			wantKind:       errors.ErrorKindContextWindowOverflow,
			wantStatusCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Classify the error
			apiErr := errors.ClassifyError(tt.statusCode, tt.body, nil)

			// Check classification
			if apiErr.Kind != tt.wantKind {
				t.Errorf("ClassifyError() Kind = %v, want %v", apiErr.Kind, tt.wantKind)
			}

			// Write the error response
			w := httptest.NewRecorder()
			writeAPIError(w, apiErr)

			// Check status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("writeAPIError() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			// Parse and verify response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			errorObj := response["error"].(map[string]interface{})
			if errorObj["type"] != tt.wantKind.String() {
				t.Errorf("Response error type = %v, want %v", errorObj["type"], tt.wantKind.String())
			}
		})
	}
}
