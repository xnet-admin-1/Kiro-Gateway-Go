// Package errors provides error classification and handling for API responses.
// It defines error types and provides utilities for classifying and handling
// different kinds of errors from the Amazon Q Developer API.
package errors

import "fmt"

// ErrorKind represents the classification of an API error.
// This allows for appropriate handling of different error types.
type ErrorKind int

const (
    // ErrorKindUnknown represents an unclassified error
    ErrorKindUnknown ErrorKind = iota
    
    // ErrorKindThrottling represents rate limiting errors (429 status)
    ErrorKindThrottling
    
    // ErrorKindContextWindowOverflow represents input too large errors
    ErrorKindContextWindowOverflow
    
    // ErrorKindModelOverloaded represents model capacity errors
    ErrorKindModelOverloaded
    
    // ErrorKindMonthlyLimitReached represents quota exceeded errors
    ErrorKindMonthlyLimitReached
    
    // ErrorKindAuthenticationFailed represents authentication errors
    ErrorKindAuthenticationFailed
    
    // ErrorKindInvalidRequest represents client errors (4xx)
    ErrorKindInvalidRequest
    
    // ErrorKindInternalServerError represents server errors (5xx)
    ErrorKindInternalServerError
)

// APIError represents a classified API error with additional context.
// It provides structured error information for better error handling.
type APIError struct {
    // Kind is the classification of the error
    Kind       ErrorKind
    
    // Message is the human-readable error message
    Message    string
    
    // RequestID is the AWS request ID for debugging
    RequestID  string
    
    // StatusCode is the HTTP status code
    StatusCode int
    
    // Cause is the underlying error that caused this error
    Cause      error
}

// Error returns the string representation of the error.
// It includes the error kind, message, request ID, and status code.
func (e *APIError) Error() string {
    result := fmt.Sprintf("[%s] %s", e.Kind.String(), e.Message)
    
    if e.RequestID != "" {
        result += fmt.Sprintf(" (RequestID: %s", e.RequestID)
        if e.StatusCode != 0 {
            result += fmt.Sprintf(", Status: %d)", e.StatusCode)
        } else {
            result += ")"
        }
    } else if e.StatusCode != 0 {
        result += fmt.Sprintf(" (Status: %d)", e.StatusCode)
    }
    
    return result
}

// Unwrap returns the underlying error for error chain unwrapping.
func (e *APIError) Unwrap() error {
    return e.Cause
}

// IsRetryable returns true if the error is retryable.
// Retryable errors include throttling, model overloaded, and server errors.
func (e *APIError) IsRetryable() bool {
    switch e.Kind {
    case ErrorKindThrottling, ErrorKindModelOverloaded, ErrorKindInternalServerError:
        return true
    default:
        return false
    }
}

// GetUserMessage returns a user-friendly error message.
// This provides helpful guidance to users for different error types.
func (e *APIError) GetUserMessage() string {
    switch e.Kind {
    case ErrorKindThrottling:
        return "Too many requests have been sent recently. Please wait and try again later."
    case ErrorKindContextWindowOverflow:
        return "The input is too long to fit within the model's context window. Please reduce the input size."
    case ErrorKindModelOverloaded:
        return "The model is currently overloaded. Please try again in a few moments."
    case ErrorKindMonthlyLimitReached:
        return "The monthly usage limit has been reached. Please upgrade your plan or wait until next month."
    case ErrorKindAuthenticationFailed:
        return "Authentication failed. Please check your credentials and try again."
    case ErrorKindInvalidRequest:
        return "The request is invalid. Please check your input and try again."
    case ErrorKindInternalServerError:
        return "An internal server error occurred. Please try again later."
    default:
        if e.Message != "" {
            return e.Message
        }
        return "An unexpected error occurred. Please try again."
    }
}

// String returns the string representation of the ErrorKind.
func (k ErrorKind) String() string {
    switch k {
    case ErrorKindThrottling:
        return "Throttling"
    case ErrorKindContextWindowOverflow:
        return "ContextWindowOverflow"
    case ErrorKindModelOverloaded:
        return "ModelOverloaded"
    case ErrorKindMonthlyLimitReached:
        return "MonthlyLimitReached"
    case ErrorKindAuthenticationFailed:
        return "AuthenticationFailed"
    case ErrorKindInvalidRequest:
        return "InvalidRequest"
    case ErrorKindInternalServerError:
        return "InternalServerError"
    default:
        return "Unknown"
    }
}
