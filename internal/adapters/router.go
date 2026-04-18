package adapters

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/yourusername/kiro-gateway-go/internal/apikeys"
	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/client"
	"github.com/yourusername/kiro-gateway-go/internal/validation"
)

// Router handles routing between OpenAI and Anthropic adapters
type Router struct {
	openaiAdapter     *OpenAIAdapter
	anthropicAdapter  *AnthropicAdapter
	authMiddleware    func(http.Handler) http.Handler
	apiKeyManager     *apikeys.PersistentAPIKeyManager
	proxyAPIKey       string
}

// NewRouter creates a new adapter router
func NewRouter(authManager *auth.AuthManager, client *client.Client, validator *validation.RequestValidator, config *AdapterConfig, apiKeyManager *apikeys.PersistentAPIKeyManager, proxyAPIKey string) *Router {
	return &Router{
		openaiAdapter:    NewOpenAIAdapter(authManager, client, validator, config),
		anthropicAdapter: NewAnthropicAdapter(authManager, client, validator, config),
		apiKeyManager:    apiKeyManager,
		proxyAPIKey:      proxyAPIKey,
		authMiddleware:   createAuthMiddleware(apiKeyManager, proxyAPIKey),
	}
}

// RegisterRoutes registers all adapter routes
func (r *Router) RegisterRoutes(mux *http.ServeMux) {
	log.Println("Registering adapter routes...")
	
	// Note: OpenAI-compatible endpoints (/v1/chat/completions, /v1/models) 
	// are already registered in internal/handlers/routes.go
	// We only register the Anthropic-specific endpoint here
	
	// Anthropic-compatible endpoints
	mux.HandleFunc("/v1/messages", r.authMiddleware(http.HandlerFunc(r.anthropicAdapter.HandleMessages)).ServeHTTP)
	
	log.Println("[SUCCESS] Adapter routes registered:")
	log.Println("  - POST /v1/messages (Anthropic)")
	log.Println("")
	log.Println("Note: OpenAI endpoints already registered by handlers:")
	log.Println("  - POST /v1/chat/completions (OpenAI)")
	log.Println("  - GET  /v1/models (OpenAI)")
}

// createAuthMiddleware creates authentication middleware that validates API keys
func createAuthMiddleware(apiKeyManager *apikeys.PersistentAPIKeyManager, proxyAPIKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from headers
			// OpenAI uses: Authorization: Bearer <key>
			// Anthropic uses: x-api-key: <key>
			
			var token string
			
			// Try Authorization header first (OpenAI style)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// Extract Bearer token
				if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
					token = authHeader[7:]
				}
			}
			
			// Try x-api-key header (Anthropic style)
			if token == "" {
				token = r.Header.Get("x-api-key")
			}
			
			// Validate API key
			if token == "" {
				writeAuthError(w, "Missing API key")
				return
			}
			
			// If API key manager is available, use it for validation
			if apiKeyManager != nil {
				apiKey, err := apiKeyManager.ValidateKey(token)
				if err != nil {
					writeAuthError(w, "Invalid or expired API Key")
					return
				}
				
				// Store API key info in request headers for later use
				r.Header.Set("X-API-Key-User-ID", apiKey.UserID)
				r.Header.Set("X-API-Key-Name", apiKey.Name)
			} else {
				// Fallback to single PROXY_API_KEY for backward compatibility
				if token != proxyAPIKey {
					writeAuthError(w, "Invalid API Key")
					return
				}
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// writeAuthError writes an authentication error response
func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "auth_error",
			"code":    401,
		},
	}
	
	json.NewEncoder(w).Encode(errorResp)
}

// handleHealth handles GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// handleHealthDetailed handles GET /v1/health
func handleHealthDetailed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{
		"status": "ok",
		"version": "1.0.0",
		"adapters": {
			"openai": "enabled",
			"anthropic": "enabled"
		},
		"endpoints": {
			"openai": ["/v1/chat/completions", "/v1/models"],
			"anthropic": ["/v1/messages"]
		}
	}`))
}
