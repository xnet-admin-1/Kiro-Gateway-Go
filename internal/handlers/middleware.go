package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
	"github.com/yourusername/kiro-gateway-go/internal/validation"
)

// RateLimitMiddleware applies rate limiting to requests
func (h *Handler) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user identifier (API key, user ID, etc.)
		userID := extractUserID(r)
		
		// Check rate limit
		if !h.rateLimiter.Allow(userID) {
			// Log rate limit violation
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "RATE_LIMIT_EXCEEDED",
				UserID:    userID,
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   false,
				Details:   map[string]interface{}{"user_id": userID},
			})
			
			h.writeValidationError(w, &validation.ValidationError{
				Field:   "rate_limit",
				Message: "Rate limit exceeded. Please wait and try again.",
				Limit:   validation.DefaultRateLimitRPS,
			}, http.StatusTooManyRequests)
			return
		}
		
		next(w, r)
	}
}

// QuotaMiddleware checks monthly quotas
func (h *Handler) QuotaMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user identifier
		userID := extractUserID(r)
		
		// Check quota
		if err := h.quotaTracker.CheckQuota(userID); err != nil {
			h.writeValidationError(w, &validation.ValidationError{
				Field:   "quota",
				Message: err.Error(),
			}, http.StatusPaymentRequired) // 402
			return
		}
		
		// Add user ID to context for later quota increment
		ctx := context.WithValue(r.Context(), "userID", userID)
		next(w, r.WithContext(ctx))
	}
}

// ValidationMiddleware validates requests before processing
func (h *Handler) ValidationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only validate POST requests to chat completions
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			next(w, r)
			return
		}
		
		// Parse request body
		var req models.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeValidationError(w, &validation.ValidationError{
				Field:   "body",
				Message: "Invalid request body",
			}, http.StatusBadRequest)
			return
		}
		
		// Validate request
		if err := h.validator.ValidateRequest(&req); err != nil {
			if valErr, ok := err.(*validation.ValidationError); ok {
				h.writeValidationError(w, valErr, http.StatusBadRequest)
			} else {
				h.writeValidationError(w, &validation.ValidationError{
					Field:   "request",
					Message: err.Error(),
				}, http.StatusBadRequest)
			}
			return
		}
		
		// Store validated request in context
		ctx := context.WithValue(r.Context(), "validatedRequest", &req)
		next(w, r.WithContext(ctx))
	}
}

// extractUserID extracts user identifier from request
func extractUserID(r *http.Request) string {
	// Try to get from Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" {
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	}
	
	// Try to get from X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}
	
	// Fallback to IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	
	return ip
}

// LoggingMiddleware logs request details
func (h *Handler) LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s from %s", r.Method, r.URL.Path, r.Proto, r.RemoteAddr)
		next(w, r)
	}
}

// CORSMiddleware adds CORS headers
func (h *Handler) CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

// RecoveryMiddleware recovers from panics
func (h *Handler) RecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				
				// Log security event
				h.logSecurityEvent(SecurityEvent{
					Timestamp: time.Now(),
					EventType: "PANIC_RECOVERED",
					IPAddress: r.RemoteAddr,
					Endpoint:  r.URL.Path,
					Success:   false,
					Details:   map[string]interface{}{"error": err},
				})
				
				response := map[string]interface{}{
					"error": map[string]interface{}{
						"message": "Internal server error",
						"type":    "internal_error",
						"code":    http.StatusInternalServerError,
					},
				}
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(response)
			}
		}()
		
		next(w, r)
	}
}

// RequestSizeLimitMiddleware limits request body size to prevent DoS
func (h *Handler) RequestSizeLimitMiddleware(maxBytes int64) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size (default 200MB for multimodal)
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next(w, r)
		}
	}
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Timestamp time.Time
	EventType string
	UserID    string
	IPAddress string
	Endpoint  string
	Success   bool
	Details   map[string]interface{}
}

// logSecurityEvent logs security events for monitoring and audit
func (h *Handler) logSecurityEvent(event SecurityEvent) {
	log.Printf("[SECURITY] %s | User: %s | IP: %s | Endpoint: %s | Success: %v | Details: %v",
		event.EventType, event.UserID, event.IPAddress, event.Endpoint,
		event.Success, event.Details)
	
	// TODO: Send to SIEM/monitoring system
	// h.securityLogger.Log(event)
}

// ChainMiddleware chains multiple middleware functions
func ChainMiddleware(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
