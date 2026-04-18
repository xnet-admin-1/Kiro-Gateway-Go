package errors

import (
	"net/http"
	"regexp"
	"strings"
)

var (
	requestIDRegex = regexp.MustCompile(`"requestId"\s*:\s*"([^"]+)"`)
)

// ClassifyError classifies an API error based on status code and response body
func ClassifyError(statusCode int, body []byte, originalErr error) *APIError {
	bodyStr := string(body)
	requestID := extractRequestID(bodyStr)
	
	// Check body content patterns first (they take precedence)
	if containsIgnoreCase(bodyStr, "MONTHLY_REQUEST_COUNT") {
		return &APIError{
			Kind:       ErrorKindMonthlyLimitReached,
			Message:    "Monthly usage limit reached",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	}
	
	if containsIgnoreCase(bodyStr, "Input is too long") {
		return &APIError{
			Kind:       ErrorKindContextWindowOverflow,
			Message:    "Input is too long for the model context",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	}
	
	if containsIgnoreCase(bodyStr, "INSUFFICIENT_MODEL_CAPACITY") {
		return &APIError{
			Kind:       ErrorKindModelOverloaded,
			Message:    "Model is currently overloaded",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	}
	
	// Check status code patterns
	switch statusCode {
	case http.StatusTooManyRequests: // 429
		return &APIError{
			Kind:       ErrorKindThrottling,
			Message:    "Too many requests",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	case http.StatusUnauthorized, http.StatusForbidden: // 401, 403
		return &APIError{
			Kind:       ErrorKindAuthenticationFailed,
			Message:    "Authentication failed",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	case http.StatusBadRequest: // 400
		return &APIError{
			Kind:       ErrorKindInvalidRequest,
			Message:    "Invalid request",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout: // 500, 502, 503, 504
		return &APIError{
			Kind:       ErrorKindInternalServerError,
			Message:    "Internal server error",
			RequestID:  requestID,
			StatusCode: statusCode,
		}
	}
	
	// Default to unknown error
	message := "Unknown error"
	if originalErr != nil {
		message = originalErr.Error()
	}
	
	return &APIError{
		Kind:       ErrorKindUnknown,
		Message:    message,
		RequestID:  requestID,
		StatusCode: statusCode,
	}
}

// extractRequestID extracts request ID from response body
func extractRequestID(body string) string {
	matches := requestIDRegex.FindStringSubmatch(body)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// containsIgnoreCase performs case-insensitive substring search
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ClassifyHTTPError is a convenience function for HTTP responses
func ClassifyHTTPError(resp *http.Response, body []byte, originalErr error) *APIError {
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return ClassifyError(statusCode, body, originalErr)
}

// FormatErrorResponse formats an APIError into an OpenAI-compatible error response
func FormatErrorResponse(apiErr *APIError, statusCode int) map[string]interface{} {
	errorObj := map[string]interface{}{
		"message": apiErr.GetUserMessage(),
		"type":    apiErr.Kind.String(),
		"code":    statusCode,
	}
	
	if apiErr.RequestID != "" {
		errorObj["request_id"] = apiErr.RequestID
	}
	
	return map[string]interface{}{
		"error": errorObj,
	}
}
