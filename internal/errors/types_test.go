package errors

import (
	"errors"
	"testing"
)

func TestErrorKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind ErrorKind
		want string
	}{
		{
			name: "Unknown",
			kind: ErrorKindUnknown,
			want: "Unknown",
		},
		{
			name: "Throttling",
			kind: ErrorKindThrottling,
			want: "Throttling",
		},
		{
			name: "ContextWindowOverflow",
			kind: ErrorKindContextWindowOverflow,
			want: "ContextWindowOverflow",
		},
		{
			name: "ModelOverloaded",
			kind: ErrorKindModelOverloaded,
			want: "ModelOverloaded",
		},
		{
			name: "MonthlyLimitReached",
			kind: ErrorKindMonthlyLimitReached,
			want: "MonthlyLimitReached",
		},
		{
			name: "AuthenticationFailed",
			kind: ErrorKindAuthenticationFailed,
			want: "AuthenticationFailed",
		},
		{
			name: "InvalidRequest",
			kind: ErrorKindInvalidRequest,
			want: "InvalidRequest",
		},
		{
			name: "InternalServerError",
			kind: ErrorKindInternalServerError,
			want: "InternalServerError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("ErrorKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name    string
		apiErr  *APIError
		want    string
	}{
		{
			name: "Error with RequestID",
			apiErr: &APIError{
				Kind:       ErrorKindThrottling,
				Message:    "Rate limit exceeded",
				RequestID:  "req-123",
				StatusCode: 429,
			},
			want: "[Throttling] Rate limit exceeded (RequestID: req-123, Status: 429)",
		},
		{
			name: "Error without RequestID",
			apiErr: &APIError{
				Kind:       ErrorKindInvalidRequest,
				Message:    "Invalid input",
				StatusCode: 400,
			},
			want: "[InvalidRequest] Invalid input (Status: 400)",
		},
		{
			name: "Error with cause",
			apiErr: &APIError{
				Kind:       ErrorKindInternalServerError,
				Message:    "Server error",
				StatusCode: 500,
				Cause:      errors.New("underlying error"),
			},
			want: "[InternalServerError] Server error (Status: 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiErr.Error(); got != tt.want {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	apiErr := &APIError{
		Kind:       ErrorKindInternalServerError,
		Message:    "Server error",
		StatusCode: 500,
		Cause:      underlyingErr,
	}

	if got := apiErr.Unwrap(); got != underlyingErr {
		t.Errorf("APIError.Unwrap() = %v, want %v", got, underlyingErr)
	}
}

func TestAPIError_Unwrap_Nil(t *testing.T) {
	apiErr := &APIError{
		Kind:       ErrorKindInvalidRequest,
		Message:    "Invalid input",
		StatusCode: 400,
	}

	if got := apiErr.Unwrap(); got != nil {
		t.Errorf("APIError.Unwrap() = %v, want nil", got)
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		name string
		kind ErrorKind
		want bool
	}{
		{
			name: "Throttling is retryable",
			kind: ErrorKindThrottling,
			want: true,
		},
		{
			name: "ModelOverloaded is retryable",
			kind: ErrorKindModelOverloaded,
			want: true,
		},
		{
			name: "InternalServerError is retryable",
			kind: ErrorKindInternalServerError,
			want: true,
		},
		{
			name: "ContextWindowOverflow is not retryable",
			kind: ErrorKindContextWindowOverflow,
			want: false,
		},
		{
			name: "MonthlyLimitReached is not retryable",
			kind: ErrorKindMonthlyLimitReached,
			want: false,
		},
		{
			name: "AuthenticationFailed is not retryable",
			kind: ErrorKindAuthenticationFailed,
			want: false,
		},
		{
			name: "InvalidRequest is not retryable",
			kind: ErrorKindInvalidRequest,
			want: false,
		},
		{
			name: "Unknown is not retryable",
			kind: ErrorKindUnknown,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &APIError{
				Kind:       tt.kind,
				Message:    "Test error",
				StatusCode: 500,
			}
			if got := apiErr.IsRetryable(); got != tt.want {
				t.Errorf("APIError.IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_GetUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		apiErr  *APIError
		want    string
	}{
		{
			name: "Throttling message",
			apiErr: &APIError{
				Kind:       ErrorKindThrottling,
				StatusCode: 429,
			},
			want: "Too many requests have been sent recently. Please wait and try again later.",
		},
		{
			name: "ContextWindowOverflow message",
			apiErr: &APIError{
				Kind:       ErrorKindContextWindowOverflow,
				StatusCode: 400,
			},
			want: "The input is too long to fit within the model's context window. Please reduce the input size.",
		},
		{
			name: "ModelOverloaded message",
			apiErr: &APIError{
				Kind:       ErrorKindModelOverloaded,
				StatusCode: 429,
			},
			want: "The model is currently overloaded. Please try again in a few moments.",
		},
		{
			name: "MonthlyLimitReached message",
			apiErr: &APIError{
				Kind:       ErrorKindMonthlyLimitReached,
				StatusCode: 429,
			},
			want: "The monthly usage limit has been reached. Please upgrade your plan or wait until next month.",
		},
		{
			name: "AuthenticationFailed message",
			apiErr: &APIError{
				Kind:       ErrorKindAuthenticationFailed,
				StatusCode: 401,
			},
			want: "Authentication failed. Please check your credentials and try again.",
		},
		{
			name: "InvalidRequest message",
			apiErr: &APIError{
				Kind:       ErrorKindInvalidRequest,
				StatusCode: 400,
			},
			want: "The request is invalid. Please check your input and try again.",
		},
		{
			name: "InternalServerError message",
			apiErr: &APIError{
				Kind:       ErrorKindInternalServerError,
				StatusCode: 500,
			},
			want: "An internal server error occurred. Please try again later.",
		},
		{
			name: "Unknown with custom message",
			apiErr: &APIError{
				Kind:       ErrorKindUnknown,
				Message:    "Custom error message",
				StatusCode: 500,
			},
			want: "Custom error message",
		},
		{
			name: "Unknown without custom message",
			apiErr: &APIError{
				Kind:       ErrorKindUnknown,
				StatusCode: 500,
			},
			want: "An unexpected error occurred. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiErr.GetUserMessage(); got != tt.want {
				t.Errorf("APIError.GetUserMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
