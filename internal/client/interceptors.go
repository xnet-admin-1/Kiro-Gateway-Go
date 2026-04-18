package client

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Interceptor intercepts HTTP requests and responses
type Interceptor interface {
	// BeforeRequest is called before sending the request
	BeforeRequest(req *http.Request) error
	
	// AfterResponse is called after receiving the response
	AfterResponse(resp *http.Response, err error) error
}

// InterceptorChain manages a chain of interceptors
type InterceptorChain struct {
	interceptors []Interceptor
}

// NewInterceptorChain creates a new interceptor chain
func NewInterceptorChain(interceptors ...Interceptor) *InterceptorChain {
	return &InterceptorChain{
		interceptors: interceptors,
	}
}

// Add adds an interceptor to the chain
func (c *InterceptorChain) Add(interceptor Interceptor) {
	c.interceptors = append(c.interceptors, interceptor)
}

// BeforeRequest calls BeforeRequest on all interceptors in order
func (c *InterceptorChain) BeforeRequest(req *http.Request) error {
	for _, interceptor := range c.interceptors {
		if err := interceptor.BeforeRequest(req); err != nil {
			return fmt.Errorf("interceptor BeforeRequest failed: %w", err)
		}
	}
	return nil
}

// AfterResponse calls AfterResponse on all interceptors in reverse order
func (c *InterceptorChain) AfterResponse(resp *http.Response, err error) error {
	// Call in reverse order (LIFO)
	for i := len(c.interceptors) - 1; i >= 0; i-- {
		if interceptorErr := c.interceptors[i].AfterResponse(resp, err); interceptorErr != nil {
			return fmt.Errorf("interceptor AfterResponse failed: %w", interceptorErr)
		}
	}
	return nil
}

// LoggingInterceptor logs HTTP requests and responses
type LoggingInterceptor struct {
	logger *log.Logger
}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor(logger *log.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{
		logger: logger,
	}
}

// BeforeRequest logs the outgoing request
func (i *LoggingInterceptor) BeforeRequest(req *http.Request) error {
	if i.logger != nil {
		i.logger.Printf("[HTTP] %s %s", req.Method, req.URL.String())
		
		// Log headers (redact sensitive ones)
		for name, values := range req.Header {
			if i.isSensitiveHeader(name) {
				i.logger.Printf("[HTTP] Header: %s: [REDACTED]", name)
			} else {
				i.logger.Printf("[HTTP] Header: %s: %v", name, values)
			}
		}
	}
	return nil
}

// isSensitiveHeader checks if a header contains sensitive information
func (i *LoggingInterceptor) isSensitiveHeader(name string) bool {
	name = strings.ToLower(name)
	sensitiveHeaders := []string{
		"authorization",
		"x-amz-security-token",
		"cookie",
		"set-cookie",
		"x-api-key",
		"x-auth-token",
	}
	
	for _, sensitive := range sensitiveHeaders {
		if name == sensitive {
			return true
		}
	}
	return false
}

// AfterResponse logs the response
func (i *LoggingInterceptor) AfterResponse(resp *http.Response, err error) error {
	if i.logger != nil {
		if err != nil {
			i.logger.Printf("[HTTP] Error: %v", err)
		} else if resp != nil {
			i.logger.Printf("[HTTP] Response: %d %s", resp.StatusCode, resp.Status)
		}
	}
	return nil
}

// TelemetryInterceptor collects telemetry data
type TelemetryInterceptor struct {
	appName    string
	appVersion string
	enabled    bool
}

// NewTelemetryInterceptor creates a new telemetry interceptor
func NewTelemetryInterceptor(appName, appVersion string, enabled bool) *TelemetryInterceptor {
	return &TelemetryInterceptor{
		appName:    appName,
		appVersion: appVersion,
		enabled:    enabled,
	}
}

// BeforeRequest records the start time for telemetry
func (i *TelemetryInterceptor) BeforeRequest(req *http.Request) error {
	if i.enabled {
		// Set start time in request header for later retrieval
		req.Header.Set("X-Request-Start-Time", fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	return nil
}

// AfterResponse records metrics
func (i *TelemetryInterceptor) AfterResponse(resp *http.Response, err error) error {
	if i.enabled && resp != nil && resp.Request != nil {
		startTimeStr := resp.Request.Header.Get("X-Request-Start-Time")
		if startTimeStr != "" {
			// Calculate duration and record metrics
			// In a real implementation, this would send metrics to a telemetry service
		}
	}
	return nil
}

// OptOutInterceptor adds opt-out header for telemetry
type OptOutInterceptor struct {
	optOut bool
}

// NewOptOutInterceptor creates a new opt-out interceptor
func NewOptOutInterceptor(optOut bool) *OptOutInterceptor {
	return &OptOutInterceptor{
		optOut: optOut,
	}
}

// BeforeRequest adds the opt-out header
func (i *OptOutInterceptor) BeforeRequest(req *http.Request) error {
	if i.optOut {
		req.Header.Set("x-amzn-codewhisperer-optout", "true")
	} else {
		req.Header.Set("x-amzn-codewhisperer-optout", "false")
	}
	return nil
}

// AfterResponse does nothing
func (i *OptOutInterceptor) AfterResponse(resp *http.Response, err error) error {
	return nil
}

// UserAgentInterceptor adds custom user agent header
type UserAgentInterceptor struct {
	appName     string
	appVersion  string
	fingerprint string
}

// NewUserAgentInterceptor creates a new user agent interceptor
func NewUserAgentInterceptor(appName, appVersion, fingerprint string) *UserAgentInterceptor {
	return &UserAgentInterceptor{
		appName:     appName,
		appVersion:  appVersion,
		fingerprint: fingerprint,
	}
}

// BeforeRequest adds the user agent header
func (i *UserAgentInterceptor) BeforeRequest(req *http.Request) error {
	// Format: AppName/Version (OS/Arch) Go/Version (fingerprint)
	userAgent := fmt.Sprintf("%s/%s (%s/%s) Go/%s (%s)",
		i.appName,
		i.appVersion,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
		i.fingerprint,
	)
	req.Header.Set("User-Agent", userAgent)
	return nil
}

// AfterResponse does nothing
func (i *UserAgentInterceptor) AfterResponse(resp *http.Response, err error) error {
	return nil
}

// RequestIDInterceptor extracts and logs request IDs
type RequestIDInterceptor struct {
	logger *log.Logger
}

// NewRequestIDInterceptor creates a new request ID interceptor
func NewRequestIDInterceptor(logger *log.Logger) *RequestIDInterceptor {
	return &RequestIDInterceptor{
		logger: logger,
	}
}

// BeforeRequest does nothing
func (i *RequestIDInterceptor) BeforeRequest(req *http.Request) error {
	return nil
}

// AfterResponse extracts the request ID from response headers
func (i *RequestIDInterceptor) AfterResponse(resp *http.Response, err error) error {
	if i.logger != nil && resp != nil {
		// Try common request ID headers
		requestID := resp.Header.Get("x-amzn-requestid")
		if requestID == "" {
			requestID = resp.Header.Get("x-amzn-request-id")
		}
		if requestID == "" {
			requestID = resp.Header.Get("x-request-id")
		}
		
		if requestID != "" {
			i.logger.Printf("[HTTP] Request ID: %s", requestID)
		}
	}
	return nil
}

// RetryHeaderInterceptor adds retry attempt information to requests
type RetryHeaderInterceptor struct {
	attempt int
}

// NewRetryHeaderInterceptor creates a new retry header interceptor
func NewRetryHeaderInterceptor() *RetryHeaderInterceptor {
	return &RetryHeaderInterceptor{
		attempt: 0,
	}
}

// SetAttempt sets the current retry attempt number
func (i *RetryHeaderInterceptor) SetAttempt(attempt int) {
	i.attempt = attempt
}

// BeforeRequest adds retry attempt header
func (i *RetryHeaderInterceptor) BeforeRequest(req *http.Request) error {
	if i.attempt > 0 {
		req.Header.Set("X-Retry-Attempt", fmt.Sprintf("%d", i.attempt))
	}
	return nil
}

// AfterResponse does nothing
func (i *RetryHeaderInterceptor) AfterResponse(resp *http.Response, err error) error {
	return nil
}

// RequestIDHeaderInterceptor adds request ID to outgoing requests
type RequestIDHeaderInterceptor struct {
	requestID string
}

// NewRequestIDHeaderInterceptor creates a new request ID header interceptor
func NewRequestIDHeaderInterceptor(requestID string) *RequestIDHeaderInterceptor {
	return &RequestIDHeaderInterceptor{
		requestID: requestID,
	}
}

// BeforeRequest adds the request ID header
func (i *RequestIDHeaderInterceptor) BeforeRequest(req *http.Request) error {
	if i.requestID != "" {
		req.Header.Set("X-Request-ID", i.requestID)
	}
	return nil
}

// AfterResponse does nothing
func (i *RequestIDHeaderInterceptor) AfterResponse(resp *http.Response, err error) error {
	return nil
}
