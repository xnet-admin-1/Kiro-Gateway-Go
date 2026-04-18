package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/apikeys"
)

// handleAPIKeys handles API key management endpoints
func (h *Handler) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Extract action from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/api-keys")
	path = strings.TrimPrefix(path, "/")
	
	switch r.Method {
	case http.MethodGet:
		if path == "" {
			h.listAPIKeys(w, r)
		} else {
			h.getAPIKey(w, r, path)
		}
	case http.MethodPost:
		h.createAPIKey(w, r)
	case http.MethodDelete:
		h.deleteAPIKey(w, r, path)
	case http.MethodPatch:
		h.updateAPIKey(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createAPIKey handles POST /v1/api-keys
func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string            `json:"name"`
		UserID      string            `json:"user_id"`
		ExpiresIn   *string           `json:"expires_in,omitempty"` // e.g., "30d", "1y"
		Permissions []string          `json:"permissions,omitempty"`
		Metadata    map[string]string `json:"metadata,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	
	// Default user ID if not provided
	if req.UserID == "" {
		req.UserID = "default"
	}
	
	// Parse expiration
	var expiresIn *time.Duration
	if req.ExpiresIn != nil {
		duration, err := parseDuration(*req.ExpiresIn)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid expires_in: %v", err), http.StatusBadRequest)
			return
		}
		expiresIn = &duration
	}
	
	// Default permissions
	if req.Permissions == nil {
		req.Permissions = []string{"chat.completions"}
	}
	
	// Generate API key
	apiKey, err := h.apiKeyManager.GenerateKey(req.Name, req.UserID, expiresIn, req.Permissions)
	if err != nil {
		log.Printf("Failed to generate API key: %v", err)
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}
	
	// Update metadata if provided
	if req.Metadata != nil {
		apiKey.Metadata = req.Metadata
	}
	
	log.Printf("Created API key: %s (user: %s, name: %s)", apikeys.MaskKey(apiKey.Key), apiKey.UserID, apiKey.Name)
	
	// Return response
	response := map[string]interface{}{
		"key":         apiKey.Key, // Only time the full key is returned
		"name":        apiKey.Name,
		"user_id":     apiKey.UserID,
		"created_at":  apiKey.CreatedAt,
		"expires_at":  apiKey.ExpiresAt,
		"permissions": apiKey.Permissions,
		"metadata":    apiKey.Metadata,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listAPIKeys handles GET /v1/api-keys
func (h *Handler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Get user ID from query params (optional)
	userID := r.URL.Query().Get("user_id")
	
	// List keys
	keys := h.apiKeyManager.ListKeys(userID)
	
	// Format response (mask keys)
	var response []map[string]interface{}
	for _, key := range keys {
		response = append(response, map[string]interface{}{
			"key_preview":  apikeys.MaskKey(key.Key),
			"name":         key.Name,
			"user_id":      key.UserID,
			"created_at":   key.CreatedAt,
			"expires_at":   key.ExpiresAt,
			"last_used_at": key.LastUsedAt,
			"is_active":    key.IsActive,
			"usage_count":  key.UsageCount,
			"permissions":  key.Permissions,
			"metadata":     key.Metadata,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keys":  response,
		"count": len(response),
	})
}

// getAPIKey handles GET /v1/api-keys/{key}
func (h *Handler) getAPIKey(w http.ResponseWriter, r *http.Request, keyString string) {
	apiKey, err := h.apiKeyManager.GetKey(keyString)
	if err != nil {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}
	
	// Return response (mask key)
	response := map[string]interface{}{
		"key_preview":  apikeys.MaskKey(apiKey.Key),
		"name":         apiKey.Name,
		"user_id":      apiKey.UserID,
		"created_at":   apiKey.CreatedAt,
		"expires_at":   apiKey.ExpiresAt,
		"last_used_at": apiKey.LastUsedAt,
		"is_active":    apiKey.IsActive,
		"usage_count":  apiKey.UsageCount,
		"permissions":  apiKey.Permissions,
		"metadata":     apiKey.Metadata,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteAPIKey handles DELETE /v1/api-keys/{key}
func (h *Handler) deleteAPIKey(w http.ResponseWriter, r *http.Request, keyString string) {
	if err := h.apiKeyManager.DeleteKey(keyString); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete API key: %v", err), http.StatusBadRequest)
		return
	}
	
	log.Printf("Deleted API key: %s", apikeys.MaskKey(keyString))
	
	w.WriteHeader(http.StatusNoContent)
}

// updateAPIKey handles PATCH /v1/api-keys/{key}
func (h *Handler) updateAPIKey(w http.ResponseWriter, r *http.Request, keyString string) {
	var req struct {
		Name     string            `json:"name,omitempty"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	
	if err := h.apiKeyManager.UpdateKey(keyString, req.Name, req.Metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update API key: %v", err), http.StatusBadRequest)
		return
	}
	
	log.Printf("Updated API key: %s", apikeys.MaskKey(keyString))
	
	// Return updated key
	h.getAPIKey(w, r, keyString)
}

// handleRevokeAPIKey handles POST /v1/api-keys/{key}/revoke
func (h *Handler) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract key from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/api-keys/")
	keyString := strings.TrimSuffix(path, "/revoke")
	
	if err := h.apiKeyManager.RevokeKey(keyString); err != nil {
		http.Error(w, fmt.Sprintf("Failed to revoke API key: %v", err), http.StatusBadRequest)
		return
	}
	
	log.Printf("Revoked API key: %s", apikeys.MaskKey(keyString))
	
	w.WriteHeader(http.StatusNoContent)
}

// parseDuration parses duration strings like "30d", "1y", "24h"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}
	
	unit := s[len(s)-1:]
	value := s[:len(s)-1]
	
	var multiplier time.Duration
	switch unit {
	case "h":
		multiplier = time.Hour
	case "d":
		multiplier = 24 * time.Hour
	case "w":
		multiplier = 7 * 24 * time.Hour
	case "m":
		multiplier = 30 * 24 * time.Hour
	case "y":
		multiplier = 365 * 24 * time.Hour
	default:
		// Try parsing as standard duration
		return time.ParseDuration(s)
	}
	
	var count int
	if _, err := fmt.Sscanf(value, "%d", &count); err != nil {
		return 0, fmt.Errorf("invalid duration value: %v", err)
	}
	
	return time.Duration(count) * multiplier, nil
}
