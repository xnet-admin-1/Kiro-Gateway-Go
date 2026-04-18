package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestRetryAction_String(t *testing.T) {
	tests := []struct {
		name   string
		action RetryAction
		want   string
	}{
		{
			name:   "NoRetry",
			action: NoRetry,
			want:   "NoRetry",
		},
		{
			name:   "Retry",
			action: Retry,
			want:   "Retry",
		},
		{
			name:   "RetryWithBackoff",
			action: RetryWithBackoff,
			want:   "RetryWithBackoff",
		},
		{
			name:   "Unknown",
			action: RetryAction(99),
			want:   "Unknown(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("RetryAction.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryClassifier_ClassifyStatusCode(t *testing.T) {
	classifier := NewRetryClassifier()

	tests := []struct {
		name       string
		statusCode int
		want       RetryAction
	}{
		// Retryable status codes
		{
			name:       "429 Too Many Requests",
			statusCode: http.StatusTooManyRequests,
			want:       RetryWithBackoff,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			want:       RetryWithBackoff,
		},
		{
			name:       "502 Bad Gateway",
			statusCode: http.StatusBadGateway,
			want:       RetryWithBackoff,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			want:       RetryWithBackoff,
		},
		{
			name:       "504 Gateway Timeout",
			statusCode: http.StatusGatewayTimeout,
			want:       RetryWithBackoff,
		},
		{
			name:       "529 AWS Overloaded",
			statusCode: 529,
			want:       RetryWithBackoff,
		},
		// Non-retryable status codes
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			want:       NoRetry,
		},
		{
			name:       "201 Created",
			statusCode: http.StatusCreated,
			want:       NoRetry,
		},
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			want:       NoRetry,
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			want:       NoRetry,
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			want:       NoRetry,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			want:       NoRetry,
		},
		{
			name:       "409 Conflict",
			statusCode: http.StatusConflict,
			want:       NoRetry,
		},
		{
			name:       "422 Unprocessable Entity",
			statusCode: http.StatusUnprocessableEntity,
			want:       NoRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode}
			got := classifier.ClassifyError(nil, resp)
			if got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryClassifier_ClassifyNetworkError(t *testing.T) {
	classifier := NewRetryClassifier()

	tests := []struct {
		name string
		err  error
		want RetryAction
	}{
		{
			name: "nil error",
			err:  nil,
			want: NoRetry,
		},
		{
			name: "connection refused",
			err:  errors.New("connection refused"),
			want: RetryWithBackoff,
		},
		{
			name: "connection reset",
			err:  errors.New("connection reset by peer"),
			want: RetryWithBackoff,
		},
		{
			name: "broken pipe",
			err:  errors.New("broken pipe"),
			want: RetryWithBackoff,
		},
		{
			name: "no such host",
			err:  errors.New("no such host"),
			want: RetryWithBackoff,
		},
		{
			name: "timeout",
			err:  errors.New("i/o timeout"),
			want: RetryWithBackoff,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: NoRetry,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: NoRetry,
		},
		{
			name: "unknown error",
			err:  errors.New("some random error"),
			want: NoRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyError(tt.err, nil)
			if got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryClassifier_ClassifyErrorWithBody(t *testing.T) {
	classifier := NewRetryClassifier()

	tests := []struct {
		name       string
		statusCode int
		body       string
		want       RetryAction
	}{
		{
			name:       "INSUFFICIENT_MODEL_CAPACITY",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error": "INSUFFICIENT_MODEL_CAPACITY"}`,
			want:       RetryWithBackoff,
		},
		{
			name:       "ThrottlingException",
			statusCode: http.StatusBadRequest,
			body:       `{"__type": "ThrottlingException"}`,
			want:       RetryWithBackoff,
		},
		{
			name:       "TooManyRequestsException",
			statusCode: http.StatusBadRequest,
			body:       `{"__type": "TooManyRequestsException"}`,
			want:       RetryWithBackoff,
		},
		{
			name:       "Input is too long",
			statusCode: http.StatusBadRequest,
			body:       `{"error": "Input is too long for the model"}`,
			want:       NoRetry,
		},
		{
			name:       "MONTHLY_REQUEST_COUNT",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error": "MONTHLY_REQUEST_COUNT exceeded"}`,
			want:       NoRetry,
		},
		{
			name:       "ValidationException",
			statusCode: http.StatusBadRequest,
			body:       `{"__type": "ValidationException"}`,
			want:       NoRetry,
		},
		{
			name:       "Generic 400 with no special body",
			statusCode: http.StatusBadRequest,
			body:       `{"error": "Invalid request"}`,
			want:       NoRetry,
		},
		{
			name:       "500 with no body",
			statusCode: http.StatusInternalServerError,
			body:       "",
			want:       RetryWithBackoff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode}
			got := classifier.ClassifyErrorWithBody(nil, resp, []byte(tt.body))
			if got != tt.want {
				t.Errorf("ClassifyErrorWithBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryClassifier_ShouldRetry(t *testing.T) {
	classifier := NewRetryClassifier()

	tests := []struct {
		name   string
		action RetryAction
		want   bool
	}{
		{
			name:   "NoRetry",
			action: NoRetry,
			want:   false,
		},
		{
			name:   "Retry",
			action: Retry,
			want:   true,
		},
		{
			name:   "RetryWithBackoff",
			action: RetryWithBackoff,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ShouldRetry(tt.action)
			if got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryClassifier_EdgeCases(t *testing.T) {
	classifier := NewRetryClassifier()

	t.Run("nil response and nil error", func(t *testing.T) {
		got := classifier.ClassifyError(nil, nil)
		if got != NoRetry {
			t.Errorf("ClassifyError(nil, nil) = %v, want NoRetry", got)
		}
	})

	t.Run("response with unusual status code", func(t *testing.T) {
		resp := &http.Response{StatusCode: 999}
		got := classifier.ClassifyError(nil, resp)
		if got != NoRetry {
			t.Errorf("ClassifyError() with status 999 = %v, want NoRetry", got)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		resp := &http.Response{StatusCode: http.StatusBadRequest}
		got := classifier.ClassifyErrorWithBody(nil, resp, []byte{})
		if got != NoRetry {
			t.Errorf("ClassifyErrorWithBody() with empty body = %v, want NoRetry", got)
		}
	})
}
