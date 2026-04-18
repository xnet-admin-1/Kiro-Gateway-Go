package handlers

import (
	"context"
	"crypto/subtle"
	"math/rand"
	"net/http"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/apikeys"
	"github.com/yourusername/kiro-gateway-go/internal/async"
	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/cache"
	"github.com/yourusername/kiro-gateway-go/internal/concurrency"
	"github.com/yourusername/kiro-gateway-go/internal/config"
	"github.com/yourusername/kiro-gateway-go/internal/conversation"
	"github.com/yourusername/kiro-gateway-go/internal/models"
	"github.com/yourusername/kiro-gateway-go/internal/validation"
)

// SetupRoutes configures all HTTP routes with middleware
func SetupRoutes(mux *http.ServeMux, authManager *auth.AuthManager, cfg *config.Config, priorityQueue *concurrency.PriorityQueue, loadShedder *concurrency.LoadShedder, asyncJobManager *async.AsyncJobManager, apiKeyManager *apikeys.PersistentAPIKeyManager, contextManager *conversation.Manager, client interface {
	ListAvailableModels(ctx context.Context, profileArn string) (*models.ListAvailableModelsResponse, error)
	ListAvailableProfiles(ctx context.Context) (*models.ListAvailableProfilesResponse, error)
}, responseCache *cache.ResponseCache, requestDedup *cache.RequestDeduplicator) {
	// Create handler with dependencies
	h := &Handler{
		authManager:     authManager,
		config:          cfg,
		priorityQueue:   priorityQueue,
		loadShedder:     loadShedder,
		asyncJobManager: asyncJobManager,
		apiKeyManager:   apiKeyManager,
		contextManager:  contextManager,
		client:          client,
		responseCache:   responseCache,
		requestDedup:    requestDedup,
		validator:       validation.NewRequestValidator(
			true, // Enforce strict limits
			cfg.BetaFeatures.EnableExtendedContext,
			cfg.BetaFeatures.WarnOnBetaFeatures,
		),
		rateLimiter:  validation.NewRateLimiter(validation.DefaultRateLimitRPS, validation.DefaultRateLimitBurst),
		quotaTracker: validation.NewQuotaTracker(),
	}
	
	// Health check endpoints (with rate limiting for security)
	mux.HandleFunc("/", ChainMiddleware(
		h.handleRoot,
		h.RateLimitMiddleware,
	))
	mux.HandleFunc("/health", ChainMiddleware(
		h.handleHealth,
		h.RateLimitMiddleware,
	))
	mux.HandleFunc("/metrics", ChainMiddleware(
		h.handleMetrics,
		h.requireAuth, // Metrics require authentication
		h.RateLimitMiddleware,
	))
	
	// Cache metrics endpoint (admin only)
	mux.HandleFunc("/admin/cache/stats", ChainMiddleware(
		h.handleCacheStats,
		h.requireAdminAuth, // Admin authentication required
		h.RateLimitMiddleware,
	))
	
	// OpenAI API endpoints with full middleware chain
	mux.HandleFunc("/v1/models", ChainMiddleware(
		h.handleModels,
		h.RecoveryMiddleware,
		h.LoggingMiddleware,
		h.CORSMiddleware,
		h.RequestSizeLimitMiddleware(10 * 1024 * 1024), // 10MB for model list
		h.requireAuth,
	))
	
	mux.HandleFunc("/v1/chat/completions", ChainMiddleware(
		h.handleChatCompletions,
		h.RecoveryMiddleware,
		h.LoggingMiddleware,
		h.CORSMiddleware,
		h.RequestSizeLimitMiddleware(200 * 1024 * 1024), // 200MB for multimodal
		h.RateLimitMiddleware,
		h.QuotaMiddleware,
		h.requireAuth,
	))
	
	// Direct Q Developer API endpoint (native format)
	mux.HandleFunc("/api/chat", ChainMiddleware(
		h.handleDirectChat,
		h.RecoveryMiddleware,
		h.LoggingMiddleware,
		h.CORSMiddleware,
		h.RequestSizeLimitMiddleware(200 * 1024 * 1024), // 200MB for multimodal
		h.RateLimitMiddleware,
		h.QuotaMiddleware,
		h.requireAuth,
	))
	
	// Async job endpoints (if enabled)
	if asyncJobManager != nil {
		mux.HandleFunc("/v1/async/jobs", ChainMiddleware(
			h.handleAsyncJobs,
			h.RecoveryMiddleware,
			h.LoggingMiddleware,
			h.CORSMiddleware,
			h.requireAuth,
		))
		
		mux.HandleFunc("/v1/async/jobs/", ChainMiddleware(
			h.handleAsyncJobStatus,
			h.RecoveryMiddleware,
			h.LoggingMiddleware,
			h.CORSMiddleware,
			h.requireAuth,
		))
		
		// Async direct API endpoint
		mux.HandleFunc("/v1/async/direct", ChainMiddleware(
			h.handleAsyncDirectJobs,
			h.RecoveryMiddleware,
			h.LoggingMiddleware,
			h.CORSMiddleware,
			h.requireAuth,
		))
	}
	
	// API key management endpoints (admin only)
	if apiKeyManager != nil {
		mux.HandleFunc("/v1/api-keys", ChainMiddleware(
			h.handleAPIKeys,
			h.RecoveryMiddleware,
			h.LoggingMiddleware,
			h.CORSMiddleware,
			h.requireAdminAuth,
		))
		
		mux.HandleFunc("/v1/api-keys/", ChainMiddleware(
			h.handleAPIKeys,
			h.RecoveryMiddleware,
			h.LoggingMiddleware,
			h.CORSMiddleware,
			h.requireAdminAuth,
		))
	}
	
	// TOTP code generation endpoint (requires authentication)
	mux.HandleFunc("/totp", ChainMiddleware(
		TOTPHandler,
		h.RecoveryMiddleware,
		h.LoggingMiddleware,
		h.CORSMiddleware,
		h.RateLimitMiddleware,
		h.requireAuth,
	))
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	authManager     *auth.AuthManager
	config          *config.Config
	priorityQueue   *concurrency.PriorityQueue
	loadShedder     *concurrency.LoadShedder
	asyncJobManager *async.AsyncJobManager
	apiKeyManager   *apikeys.PersistentAPIKeyManager
	contextManager  *conversation.Manager               // Context manager for tracking conversation state
	responseCache   *cache.ResponseCache                // Response cache for API responses
	requestDedup    *cache.RequestDeduplicator          // Request deduplicator for concurrent identical requests
	validator       *validation.RequestValidator
	rateLimiter     *validation.RateLimiter
	quotaTracker    *validation.QuotaTracker
	client          interface {
		ListAvailableModels(ctx context.Context, profileArn string) (*models.ListAvailableModelsResponse, error)
	ListAvailableProfiles(ctx context.Context) (*models.ListAvailableProfilesResponse, error)
	}
}

// requireAuth is a middleware that checks API key
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		
		if authHeader == "" {
			// Log failed auth attempt
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "AUTH_FAILED",
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   false,
				Details:   map[string]interface{}{"reason": "missing_header"},
			})
			
			// Add random delay to prevent timing analysis
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			http.Error(w, `{"error":{"message":"Missing API Key","type":"auth_error","code":401}}`, http.StatusUnauthorized)
			return
		}
		
		// Extract token from "Bearer <token>"
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "AUTH_FAILED",
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   false,
				Details:   map[string]interface{}{"reason": "invalid_format"},
			})
			
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			http.Error(w, `{"error":{"message":"Invalid Authorization header format","type":"auth_error","code":401}}`, http.StatusUnauthorized)
			return
		}
		
		// If API key manager is available, use it for validation
		if h.apiKeyManager != nil {
			apiKey, err := h.apiKeyManager.ValidateKey(token)
			if err != nil {
				h.logSecurityEvent(SecurityEvent{
					Timestamp: time.Now(),
					EventType: "AUTH_FAILED",
					UserID:    "unknown",
					IPAddress: r.RemoteAddr,
					Endpoint:  r.URL.Path,
					Success:   false,
					Details:   map[string]interface{}{"reason": "invalid_key", "error": err.Error()},
				})
				
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				http.Error(w, `{"error":{"message":"Invalid or expired API Key","type":"auth_error","code":401}}`, http.StatusUnauthorized)
				return
			}
			
			// Log successful auth
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "AUTH_SUCCESS",
				UserID:    apiKey.UserID,
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   true,
				Details:   map[string]interface{}{"key_name": apiKey.Name},
			})
			
			// Store API key info in request context for later use
			r.Header.Set("X-API-Key-User-ID", apiKey.UserID)
			r.Header.Set("X-API-Key-Name", apiKey.Name)
		} else {
			// Fallback to single PROXY_API_KEY for backward compatibility
			// Use constant-time comparison to prevent timing attacks
			expectedKey := []byte(h.config.ProxyAPIKey)
			providedKey := []byte(token)
			
			if subtle.ConstantTimeCompare(expectedKey, providedKey) != 1 {
				h.logSecurityEvent(SecurityEvent{
					Timestamp: time.Now(),
					EventType: "AUTH_FAILED",
					IPAddress: r.RemoteAddr,
					Endpoint:  r.URL.Path,
					Success:   false,
					Details:   map[string]interface{}{"reason": "invalid_proxy_key"},
				})
				
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				http.Error(w, `{"error":{"message":"Invalid API Key","type":"auth_error","code":401}}`, http.StatusUnauthorized)
				return
			}
			
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "AUTH_SUCCESS",
				UserID:    "proxy",
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   true,
				Details:   map[string]interface{}{"method": "proxy_key"},
			})
		}
		
		next(w, r)
	}
}

// requireAdminAuth is a middleware that checks for admin API key
func (h *Handler) requireAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		
		if authHeader == "" {
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "ADMIN_AUTH_FAILED",
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   false,
				Details:   map[string]interface{}{"reason": "missing_header"},
			})
			
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			http.Error(w, `{"error":{"message":"Missing API Key","type":"auth_error","code":401}}`, http.StatusUnauthorized)
			return
		}
		
		// Extract token from "Bearer <token>"
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "ADMIN_AUTH_FAILED",
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   false,
				Details:   map[string]interface{}{"reason": "invalid_format"},
			})
			
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			http.Error(w, `{"error":{"message":"Invalid Authorization header format","type":"auth_error","code":401}}`, http.StatusUnauthorized)
			return
		}
		
		// Check if this is the admin API key (constant-time comparison)
		expectedKey := []byte(h.config.AdminAPIKey)
		providedKey := []byte(token)
		
		if subtle.ConstantTimeCompare(expectedKey, providedKey) == 1 {
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "ADMIN_AUTH_SUCCESS",
				UserID:    "admin",
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   true,
				Details:   map[string]interface{}{"method": "admin_key"},
			})
			next(w, r)
			return
		}
		
		// If API key manager is available, check for admin permissions
		if h.apiKeyManager != nil {
			apiKey, err := h.apiKeyManager.ValidateKey(token)
			if err != nil {
				h.logSecurityEvent(SecurityEvent{
					Timestamp: time.Now(),
					EventType: "ADMIN_AUTH_FAILED",
					IPAddress: r.RemoteAddr,
					Endpoint:  r.URL.Path,
					Success:   false,
					Details:   map[string]interface{}{"reason": "invalid_key"},
				})
				
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				http.Error(w, `{"error":{"message":"Invalid or expired API Key","type":"auth_error","code":401}}`, http.StatusUnauthorized)
				return
			}
			
			// Check if key has admin permissions
			hasAdminPerm := false
			for _, perm := range apiKey.Permissions {
				if perm == "admin" || perm == "*" {
					hasAdminPerm = true
					break
				}
			}
			
			if !hasAdminPerm {
				h.logSecurityEvent(SecurityEvent{
					Timestamp: time.Now(),
					EventType: "ADMIN_AUTH_FAILED",
					UserID:    apiKey.UserID,
					IPAddress: r.RemoteAddr,
					Endpoint:  r.URL.Path,
					Success:   false,
					Details:   map[string]interface{}{"reason": "insufficient_permissions"},
				})
				
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				http.Error(w, `{"error":{"message":"Insufficient permissions - admin access required","type":"auth_error","code":403}}`, http.StatusForbidden)
				return
			}
			
			h.logSecurityEvent(SecurityEvent{
				Timestamp: time.Now(),
				EventType: "ADMIN_AUTH_SUCCESS",
				UserID:    apiKey.UserID,
				IPAddress: r.RemoteAddr,
				Endpoint:  r.URL.Path,
				Success:   true,
				Details:   map[string]interface{}{"key_name": apiKey.Name},
			})
			
			next(w, r)
			return
		}
		
		// No API key manager and not admin key
		h.logSecurityEvent(SecurityEvent{
			Timestamp: time.Now(),
			EventType: "ADMIN_AUTH_FAILED",
			IPAddress: r.RemoteAddr,
			Endpoint:  r.URL.Path,
			Success:   false,
			Details:   map[string]interface{}{"reason": "no_valid_credentials"},
		})
		
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
		http.Error(w, `{"error":{"message":"Admin access required","type":"auth_error","code":403}}`, http.StatusForbidden)
	}
}
