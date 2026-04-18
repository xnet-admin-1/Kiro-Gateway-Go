package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// Model cache with TTL
var (
	modelCache      *models.ListAvailableModelsResponse
	modelCacheMutex sync.RWMutex
	modelCacheTime  time.Time
	modelCacheTTL   = 5 * time.Minute // Cache for 5 minutes
)

// handleModels handles GET /v1/models
func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	log.Println("Request to /v1/models")
	
	// Check cache first
	modelCacheMutex.RLock()
	if modelCache != nil && time.Since(modelCacheTime) < modelCacheTTL {
		cached := modelCache
		modelCacheMutex.RUnlock()
		
		log.Println("Returning cached model list")
		h.sendModelListResponse(w, cached)
		return
	}
	modelCacheMutex.RUnlock()
	
	// Cache miss or expired - fetch from AWS
	log.Println("Cache miss, fetching models from AWS Q Developer API")
	
	// Get profile ARN from auth manager, auto-discover if empty
	profileArn := h.authManager.GetProfileArn()
	if profileArn == "" {
		log.Println("Profile ARN not set, attempting auto-discovery via ListAvailableProfiles...")
		discoverCtx, discoverCancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer discoverCancel()
		profiles, err := h.client.ListAvailableProfiles(discoverCtx)
		if err != nil {
			log.Printf("Failed to discover profiles: %v, falling back to hardcoded models", err)
			h.sendFallbackModels(w)
			return
		}
		if len(profiles.Profiles) == 0 {
			log.Println("No profiles found, falling back to hardcoded models")
			h.sendFallbackModels(w)
			return
		}
		profileArn = profiles.Profiles[0].Arn
		log.Printf("Auto-discovered profile: %s (%s)", profiles.Profiles[0].ProfileName, profileArn)
		h.authManager.SetProfileArn(profileArn)
	}
	
	// Fetch models from AWS with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	
	awsModels, err := h.client.ListAvailableModels(ctx, profileArn)
	if err != nil {
		log.Printf("Error fetching models from AWS: %v, falling back to hardcoded models", err)
		h.sendFallbackModels(w)
		return
	}
	
	// Update cache
	modelCacheMutex.Lock()
	modelCache = awsModels
	modelCacheTime = time.Now()
	modelCacheMutex.Unlock()
	
	log.Printf("Successfully fetched %d models from AWS", len(awsModels.Models))
	h.sendModelListResponse(w, awsModels)
}

// sendModelListResponse converts AWS models to OpenAI format and sends response
func (h *Handler) sendModelListResponse(w http.ResponseWriter, awsModels *models.ListAvailableModelsResponse) {
	// Convert AWS models to OpenAI format
	openAIModels := make([]models.Model, 0, len(awsModels.Models))
	created := time.Now().Unix()
	
	for _, awsModel := range awsModels.Models {
		// Determine owned_by based on model ID
		ownedBy := "anthropic"
		if awsModel.ModelName != nil && *awsModel.ModelName != "" {
			ownedBy = "aws"
		}
		
		// Build description
		description := ""
		if awsModel.Description != nil {
			description = *awsModel.Description
		} else if awsModel.ModelName != nil {
			description = *awsModel.ModelName
		}
		
		// Add vision support indicator if model supports IMAGE input
		supportsVision := false
		for _, inputType := range awsModel.SupportedInputTypes {
			if inputType == "IMAGE" {
				supportsVision = true
				break
			}
		}
		if supportsVision && description != "" {
			description += " (supports vision)"
		}
		
		openAIModels = append(openAIModels, models.Model{
			ID:          awsModel.ModelID,
			Object:      "model",
			Created:     created,
			OwnedBy:     ownedBy,
			Description: description,
		})
	}
	
	// Add hidden models if configured
	for _, hiddenModelID := range h.config.HiddenModels {
		openAIModels = append(openAIModels, models.Model{
			ID:          hiddenModelID,
			Object:      "model",
			Created:     created,
			OwnedBy:     "aws",
			Description: "Hidden model",
		})
	}
	
	modelList := models.ModelList{
		Object: "list",
		Data:   openAIModels,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelList)
}

// sendFallbackModels sends a hardcoded list of models as fallback
func (h *Handler) sendFallbackModels(w http.ResponseWriter) {
	// Fallback to validator's available models
	availableModels := h.validator.GetAvailableModels()
	
	// Add hidden models if configured
	availableModels = append(availableModels, h.config.HiddenModels...)
	
	modelList := models.NewModelList(availableModels)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelList)
}
