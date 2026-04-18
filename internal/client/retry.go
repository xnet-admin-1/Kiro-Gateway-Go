// Package client provides HTTP client functionality with retry logic and error classification.
// It implements intelligent retry strategies for different types of errors and provides
// utilities for classifying API responses.
package client

import (
    "context"
    "fmt"
    "net/http"
    "strings"
)

// RetryAction represents the action to take for a failed request.
// This determines whether and how a request should be retried.
type RetryAction int

const (
    // NoRetry indicates the request should not be retried
    NoRetry RetryAction = iota
    
    // Retry indicates the request should be retried immediately
    Retry
    
    // RetryWithBackoff indicates the request should be retried with exponential backoff
    RetryWithBackoff
)

// String returns string representation of RetryAction for debugging.
func (r RetryAction) String() string {
    switch r {
    case NoRetry:
        return "NoRetry"
    case Retry:
        return "Retry"
    case RetryWithBackoff:
        return "RetryWithBackoff"
    default:
        return fmt.Sprintf("Unknown(%d)", int(r))
    }
}

// RetryClassifier classifies errors to determine retry strategy.
// It analyzes HTTP responses and errors to make intelligent retry decisions.
type RetryClassifier struct{}

// NewRetryClassifier creates a new RetryClassifier instance.
func NewRetryClassifier() *RetryClassifier {
    return &RetryClassifier{}
}

// ClassifyError classifies an error for retry based on the error and HTTP response.
// It returns the appropriate retry action based on error type and status code.
func (c *RetryClassifier) ClassifyError(err error, resp *http.Response) RetryAction {
    // Handle network errors (no response received)
    if err != nil && resp == nil {
        // Don't retry context cancellation
        if err == context.Canceled || err == context.DeadlineExceeded {
            return NoRetry
        }
        
        // Retry common network errors with backoff
        errStr := err.Error()
        if strings.Contains(errStr, "connection refused") ||
           strings.Contains(errStr, "connection reset") ||
           strings.Contains(errStr, "broken pipe") ||
           strings.Contains(errStr, "no such host") ||
           strings.Contains(errStr, "timeout") {
            return RetryWithBackoff
        }
        
        return NoRetry
    }
    
    if resp == nil {
        return NoRetry
    }
    
    // Classify based on HTTP status code
    switch resp.StatusCode {
    case 429, 500, 502, 503, 504, 529:
        // Retryable server errors and rate limiting
        return RetryWithBackoff
    default:
        // Client errors (4xx) and success (2xx) are not retryable
        return NoRetry
    }
}

// ClassifyErrorWithBody classifies an error with access to the response body.
// This allows for more sophisticated error classification based on error messages.
func (c *RetryClassifier) ClassifyErrorWithBody(err error, resp *http.Response, body []byte) RetryAction {
    // First check network errors
    if err != nil && resp == nil {
        return c.ClassifyError(err, resp)
    }
    
    if resp == nil {
        return NoRetry
    }
    
    bodyStr := string(body)
    
    // Check for specific error patterns in response body
    if strings.Contains(bodyStr, "INSUFFICIENT_MODEL_CAPACITY") ||
       strings.Contains(bodyStr, "ThrottlingException") ||
       strings.Contains(bodyStr, "TooManyRequestsException") {
        return RetryWithBackoff
    }
    
    // Non-retryable errors (client errors)
    if strings.Contains(bodyStr, "Input is too long") ||
       strings.Contains(bodyStr, "MONTHLY_REQUEST_COUNT") ||
       strings.Contains(bodyStr, "ValidationException") {
        return NoRetry
    }
    
    // Fall back to status code classification
    return c.ClassifyError(err, resp)
}

// ShouldRetry returns true if the action indicates the request should be retried.
func (c *RetryClassifier) ShouldRetry(action RetryAction) bool {
    return action == Retry || action == RetryWithBackoff
}
