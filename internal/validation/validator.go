package validation

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Limit   interface{}
	Actual  interface{}
}

func (e *ValidationError) Error() string {
	if e.Limit != nil && e.Actual != nil {
		return fmt.Sprintf("%s: %s (limit: %v, actual: %v)", e.Field, e.Message, e.Limit, e.Actual)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// RequestValidator validates requests against AWS Q Developer limits
type RequestValidator struct {
	enforceStrictLimits   bool
	enableExtendedContext bool
	warnOnBetaFeatures    bool
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(enforceStrictLimits, enableExtendedContext, warnOnBetaFeatures bool) *RequestValidator {
	return &RequestValidator{
		enforceStrictLimits:   enforceStrictLimits,
		enableExtendedContext: enableExtendedContext,
		warnOnBetaFeatures:    warnOnBetaFeatures,
	}
}

// GetAvailableModels returns a list of all supported model IDs
func (v *RequestValidator) GetAvailableModels() []string {
	models := make([]string, 0, len(ModelLimits))
	for modelID := range ModelLimits {
		models = append(models, modelID)
	}
	return models
}

// ValidateRequest validates a chat completion request
func (v *RequestValidator) ValidateRequest(req *models.ChatCompletionRequest) error {
	// Validate model
	modelLimit, err := v.validateModel(req.Model)
	if err != nil {
		return err
	}
	
	// Validate messages
	if err := v.validateMessages(req.Messages); err != nil {
		return err
	}
	
	// Validate message content size (model-specific)
	if err := v.validateMessageSize(req.Messages, modelLimit); err != nil {
		return err
	}
	
	// Validate images if present (check model support)
	if err := v.validateImages(req.Messages, modelLimit); err != nil {
		return err
	}
	
	// Validate tools if present
	if err := v.validateTools(req.Tools); err != nil {
		return err
	}
	
	// Validate max_tokens if specified (model-specific)
	if err := v.validateMaxTokens(req.MaxTokens, modelLimit); err != nil {
		return err
	}
	
	return nil
}

// validateModel validates the model ID and returns model limits
func (v *RequestValidator) validateModel(modelID string) (ModelLimit, error) {
	if modelID == "" {
		return ModelLimit{}, &ValidationError{
			Field:   "model",
			Message: "model is required",
		}
	}
	
	// Get model limits (allow unknown models through with defaults)
	modelLimit, err := GetModelLimit(modelID)
	if err != nil {
		log.Printf("[WARN] Unknown model %s, using default limits", modelID)
	}
	
	// Warn if using extended context (beta feature)
	if v.enableExtendedContext && modelLimit.SupportsExtendedContext && v.warnOnBetaFeatures {
		// Note: This is just validation, actual warning is logged in handler
	}
	
	return modelLimit, nil
}

// validateMessages validates the messages array
func (v *RequestValidator) validateMessages(messages []models.Message) error {
	if len(messages) == 0 {
		return &ValidationError{
			Field:   "messages",
			Message: "at least one message is required",
		}
	}
	
	if len(messages) > MaxConversationHistoryLength {
		return &ValidationError{
			Field:   "messages",
			Message: "too many messages in conversation history",
			Limit:   MaxConversationHistoryLength,
			Actual:  len(messages),
		}
	}
	
	// Validate last message is from user
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" && lastMsg.Role != "tool" {
		return &ValidationError{
			Field:   "messages",
			Message: "last message must be from user",
		}
	}
	
	return nil
}

// validateMessageSize validates the total size of messages (model-specific)
func (v *RequestValidator) validateMessageSize(messages []models.Message, modelLimit ModelLimit) error {
	totalChars := 0
	
	for i, msg := range messages {
		msgChars := 0
		
		// Count text content
		switch content := msg.Content.(type) {
		case string:
			msgChars = len(content)
		case []interface{}:
			for _, item := range content {
				if part, ok := item.(map[string]interface{}); ok {
					if part["type"] == "text" {
						if text, ok := part["text"].(string); ok {
							msgChars += len(text)
						}
					}
				}
			}
		}
		
		// Check individual message size for user messages
		if msg.Role == "user" && msgChars > MaxUserMessageChars {
			return &ValidationError{
				Field:   fmt.Sprintf("messages[%d]", i),
				Message: "user message exceeds maximum size",
				Limit:   MaxUserMessageChars,
				Actual:  msgChars,
			}
		}
		
		// Check tool response size
		if msg.Role == "tool" && msgChars > MaxToolResponseChars {
			return &ValidationError{
				Field:   fmt.Sprintf("messages[%d]", i),
				Message: "tool response exceeds maximum size",
				Limit:   MaxToolResponseChars,
				Actual:  msgChars,
			}
		}
		
		totalChars += msgChars
	}
	
	// Estimate token count
	estimatedTokens := EstimateTokenCount(totalChars)
	
	// Get effective context window and max output based on beta features
	effectiveContextWindow := modelLimit.GetEffectiveContextWindow(v.enableExtendedContext)
	effectiveMaxOutput := modelLimit.GetEffectiveMaxOutput(v.enableExtendedContext)
	
	// Check against context window (leaving room for response)
	maxInputTokens := effectiveContextWindow - effectiveMaxOutput
	if estimatedTokens > maxInputTokens {
		return &ValidationError{
			Field:   "messages",
			Message: fmt.Sprintf("total message size exceeds context window (using %s context)", 
				map[bool]string{true: "extended", false: "standard"}[v.enableExtendedContext && modelLimit.SupportsExtendedContext]),
			Limit:   maxInputTokens,
			Actual:  estimatedTokens,
		}
	}
	
	return nil
}

// validateImages validates images in messages (check model support)
func (v *RequestValidator) validateImages(messages []models.Message, modelLimit ModelLimit) error {
	totalImages := 0
	
	for i, msg := range messages {
		images := extractImages(msg.Content)
		
		if len(images) > 0 {
			// Check if model supports multimodal
			if err := modelLimit.ValidateFeatureSupport("multimodal"); err != nil {
				return &ValidationError{
					Field:   fmt.Sprintf("messages[%d]", i),
					Message: fmt.Sprintf("model does not support images: %v", err),
				}
			}
		}
		
		for j, img := range images {
			totalImages++
			
			// Check image count
			if totalImages > MaxImagesPerRequest {
				return &ValidationError{
					Field:   fmt.Sprintf("messages[%d].content[%d]", i, j),
					Message: "too many images in request",
					Limit:   MaxImagesPerRequest,
					Actual:  totalImages,
				}
			}
			
			// Validate image format
			format := detectImageFormat(img)
			if !IsSupportedImageFormat(format) {
				return &ValidationError{
					Field:   fmt.Sprintf("messages[%d].content[%d]", i, j),
					Message: fmt.Sprintf("unsupported image format: %s", format),
				}
			}
			
			// Validate image size
			imageSize := estimateImageSize(img)
			if imageSize > MaxImageSizeBytes {
				return &ValidationError{
					Field:   fmt.Sprintf("messages[%d].content[%d]", i, j),
					Message: "image exceeds maximum size",
					Limit:   fmt.Sprintf("%d MB", MaxImageSizeBytes/(1024*1024)),
					Actual:  fmt.Sprintf("%.2f MB", float64(imageSize)/(1024*1024)),
				}
			}
		}
	}
	
	return nil
}

// validateTools validates the tools array
func (v *RequestValidator) validateTools(tools []models.Tool) error {
	if len(tools) == 0 {
		return nil
	}
	
	// Check for duplicate tool names
	seen := make(map[string]bool)
	for i, tool := range tools {
		if seen[tool.Function.Name] {
			return &ValidationError{
				Field:   fmt.Sprintf("tools[%d]", i),
				Message: fmt.Sprintf("duplicate tool name: %s", tool.Function.Name),
			}
		}
		seen[tool.Function.Name] = true
		
		// Validate tool name
		if tool.Function.Name == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("tools[%d]", i),
				Message: "tool name is required",
			}
		}
		
		// Validate tool description
		if tool.Function.Description == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("tools[%d]", i),
				Message: "tool description is required",
			}
		}
	}
	
	return nil
}

// validateMaxTokens validates the max_tokens parameter (model-specific)
func (v *RequestValidator) validateMaxTokens(maxTokens int, modelLimit ModelLimit) error {
	if maxTokens == 0 {
		return nil // Not specified
	}
	
	if maxTokens < 0 {
		return &ValidationError{
			Field:   "max_tokens",
			Message: "max_tokens must be positive",
			Actual:  maxTokens,
		}
	}
	
	// Get effective max output based on beta features
	effectiveMaxOutput := modelLimit.GetEffectiveMaxOutput(v.enableExtendedContext)
	
	if maxTokens > effectiveMaxOutput {
		return &ValidationError{
			Field:   "max_tokens",
			Message: fmt.Sprintf("max_tokens exceeds maximum output tokens for this model (using %s context)",
				map[bool]string{true: "extended", false: "standard"}[v.enableExtendedContext && modelLimit.SupportsExtendedContext]),
			Limit:   effectiveMaxOutput,
			Actual:  maxTokens,
		}
	}
	
	return nil
}

// Helper functions

// extractImages extracts image URLs from message content
func extractImages(content interface{}) []string {
	var images []string
	
	if parts, ok := content.([]interface{}); ok {
		for _, item := range parts {
			if part, ok := item.(map[string]interface{}); ok {
				if part["type"] == "image_url" {
					if imageURL, ok := part["image_url"].(map[string]interface{}); ok {
						if url, ok := imageURL["url"].(string); ok {
							images = append(images, url)
						}
					}
				}
			}
		}
	}
	
	return images
}

// detectImageFormat detects image format from data URL
func detectImageFormat(dataURL string) string {
	if strings.HasPrefix(dataURL, "data:image/") {
		parts := strings.SplitN(dataURL, ";", 2)
		if len(parts) > 0 {
			format := strings.TrimPrefix(parts[0], "data:image/")
			return format
		}
	}
	return "unknown"
}

// estimateImageSize estimates image size from base64 data URL
func estimateImageSize(dataURL string) int {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return 0
	}
	
	// Base64 encoded size
	encodedSize := len(parts[1])
	
	// Actual size is approximately 3/4 of base64 size
	actualSize := (encodedSize * 3) / 4
	
	return actualSize
}

// decodeBase64Image decodes a base64 image and returns its size
func decodeBase64Image(dataURL string) ([]byte, error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URL format")
	}
	
	return base64.StdEncoding.DecodeString(parts[1])
}
