package validation

import (
	"fmt"
)

// API Limits based on AWS Q Developer specifications
const (
	// Context Window Limits
	MaxContextWindowTokens        = 200_000 // Standard context window
	MaxContextWindowTokensExtended = 1_000_000 // Beta extended context (Sonnet 4.5 only)
	MaxOutputTokens               = 32_000
	
	// Character to Token Ratio (English average)
	CharsPerToken = 4.7
	
	// Message Size Limits (from Q CLI source)
	MaxUserMessageChars          = 400_000 // Service limit: 600,000
	MaxToolResponseChars         = 400_000 // Service limit: 800,000
	MaxConversationHistoryLength = 10_000  // Max messages in history
	MaxCurrentWorkingDirLength   = 256     // Max path length
	
	// Image Limits (from Q CLI source)
	MaxImagesPerRequest      = 10        // Q CLI limit
	MaxImagesPerRequestAPI   = 20        // API limit (standard resolution)
	MaxImagesPerRequestBulk  = 100       // API limit (reduced resolution)
	MaxImageSizeBytes        = 10 * 1024 * 1024 // 10 MB
	MaxImageResolutionStd    = 8000      // 8000x8000 px for ≤20 images
	MaxImageResolutionBulk   = 2000      // 2000x2000 px for >20 images
	
	// Request Size Limits
	MaxTotalPayloadSize = 40 * 1024 * 1024 // 40 MB (with images)
	MaxTextOnlyPayload  = 400 * 1024       // 400 KB (text only)
	
	// Rate Limits (conservative estimates)
	DefaultRateLimitRPS   = 10  // Requests per second
	DefaultRateLimitBurst = 50  // Burst capacity
	
	// Quota Limits (per user per month)
	MaxAgenticRequestsPerMonth = 10_000 // ~1,000 user inputs
	MaxInferenceCallsPerMonth  = 10_000
)

// Model-specific limits
var ModelLimits = map[string]ModelLimit{
	// Claude Sonnet 4.5 - Balanced performance with extended context (beta)
	"anthropic.claude-sonnet-4-5-20250929-v1:0": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000, // Beta feature
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000, // Reduced for extended context
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true, // Beta flag
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-sonnet-4-5": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000,
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-sonnet-4-5-20250929": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000,
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-sonnet-4": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000, // Beta feature
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000, // Reduced for extended context
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true, // Beta flag
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-sonnet-4-20250514": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000,
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-sonnet-4-20250929": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000,
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	
	// Claude Sonnet 4 - Latest
	"anthropic.claude-sonnet-4-20250514-v1:0": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    1_000_000,
		MaxOutput:                32_000,
		MaxOutputExtended:        8_000,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  true,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	
	// Claude Opus 4.5 - Most intelligent, highest cost
	"anthropic.claude-opus-4-5-20251101-v1:0": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0, // Not supported
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  false,
		Speed:                    "slow",
		Intelligence:             "highest",
		CostTier:                 "high",
	},
	"claude-opus-4-5": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0, // Not supported
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  false,
		Speed:                    "slow",
		Intelligence:             "highest",
		CostTier:                 "high",
	},
	"claude-opus-4-5-20251101": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  false,
		Speed:                    "slow",
		Intelligence:             "highest",
		CostTier:                 "high",
	},
	"claude-opus-4-20251124": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: true,
		SupportsExtendedContext:  false,
		Speed:                    "slow",
		Intelligence:             "highest",
		CostTier:                 "high",
	},
	
	// Claude Haiku 4.5 - Fastest, lowest cost
	"anthropic.claude-haiku-4-5-20251001-v1:0": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0, // Not supported
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "fast",
		Intelligence:             "good",
		CostTier:                 "low",
	},
	"claude-haiku-4-5": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0, // Not supported
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "fast",
		Intelligence:             "good",
		CostTier:                 "low",
	},
	"claude-haiku-4-20251015": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "fast",
		Intelligence:             "good",
		CostTier:                 "low",
	},
	
	// Claude 3.7 Sonnet - Legacy model
	"claude-3-7-sonnet-20250219": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "good",
		CostTier:                 "medium",
	},
	"claude-3.7-sonnet": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "good",
		CostTier:                 "medium",
	},
	"claude-3-7-sonnet-20250224": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                32_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "good",
		CostTier:                 "medium",
	},
	
	// Claude 3.5 Sonnet - Previous generation
	"claude-3-5-sonnet-20241022": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                8_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-3-5-sonnet-20241022-v2": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                8_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	// Alias for Claude 3.5 Sonnet v2
	"claude-sonnet-3-5-v2": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                8_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	"claude-3-5-sonnet-20240620": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                8_000,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "high",
		CostTier:                 "medium",
	},
	
	// Claude 3 Opus
	"claude-3-opus-20240229": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                4_096,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "slow",
		Intelligence:             "highest",
		CostTier:                 "high",
	},
	
	// Claude 3 Sonnet
	"claude-3-sonnet-20240229": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                4_096,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "medium",
		Intelligence:             "good",
		CostTier:                 "medium",
	},
	
	// Claude 3 Haiku
	"claude-3-haiku-20240307": {
		ContextWindow:            200_000,
		ExtendedContextWindow:    0,
		MaxOutput:                4_096,
		MaxOutputExtended:        0,
		SupportsMultimodal:       true,
		SupportsExtendedThinking: false,
		SupportsExtendedContext:  false,
		Speed:                    "fast",
		Intelligence:             "good",
		CostTier:                 "low",
	},

	// Auto (routes to best available)
	"auto": {
		ContextWindow: 200_000, ExtendedContextWindow: 1_000_000,
		MaxOutput: 32_000, MaxOutputExtended: 8_000,
		SupportsMultimodal: true, SupportsExtendedThinking: true, SupportsExtendedContext: true,
		Speed: "medium", Intelligence: "high", CostTier: "medium",
	},
	"claude-opus-4.6": {
		ContextWindow: 200_000, ExtendedContextWindow: 1_000_000,
		MaxOutput: 32_000, MaxOutputExtended: 8_000,
		SupportsMultimodal: true, SupportsExtendedThinking: true, SupportsExtendedContext: true,
		Speed: "slow", Intelligence: "highest", CostTier: "high",
	},
	"claude-sonnet-4.6": {
		ContextWindow: 200_000, ExtendedContextWindow: 1_000_000,
		MaxOutput: 32_000, MaxOutputExtended: 8_000,
		SupportsMultimodal: true, SupportsExtendedThinking: true, SupportsExtendedContext: true,
		Speed: "medium", Intelligence: "high", CostTier: "medium",
	},
	"claude-opus-4.5": {
		ContextWindow: 200_000, ExtendedContextWindow: 1_000_000,
		MaxOutput: 32_000, MaxOutputExtended: 8_000,
		SupportsMultimodal: true, SupportsExtendedThinking: true, SupportsExtendedContext: true,
		Speed: "slow", Intelligence: "highest", CostTier: "high",
	},
	"claude-sonnet-4.5": {
		ContextWindow: 200_000, ExtendedContextWindow: 1_000_000,
		MaxOutput: 32_000, MaxOutputExtended: 8_000,
		SupportsMultimodal: true, SupportsExtendedThinking: true, SupportsExtendedContext: true,
		Speed: "medium", Intelligence: "high", CostTier: "medium",
	},
	"claude-haiku-4.5": {
		ContextWindow: 200_000, ExtendedContextWindow: 0,
		MaxOutput: 8_192, MaxOutputExtended: 0,
		SupportsMultimodal: true, SupportsExtendedThinking: false, SupportsExtendedContext: false,
		Speed: "fast", Intelligence: "good", CostTier: "low",
	},
	"deepseek-3.2": {
		ContextWindow: 128_000, ExtendedContextWindow: 0,
		MaxOutput: 16_000, MaxOutputExtended: 0,
		SupportsMultimodal: false, SupportsExtendedThinking: true, SupportsExtendedContext: false,
		Speed: "medium", Intelligence: "high", CostTier: "low",
	},
	"minimax-m2.5": {
		ContextWindow: 200_000, ExtendedContextWindow: 0,
		MaxOutput: 16_000, MaxOutputExtended: 0,
		SupportsMultimodal: true, SupportsExtendedThinking: false, SupportsExtendedContext: false,
		Speed: "fast", Intelligence: "good", CostTier: "low",
	},
	"minimax-m2.1": {
		ContextWindow: 200_000, ExtendedContextWindow: 0,
		MaxOutput: 16_000, MaxOutputExtended: 0,
		SupportsMultimodal: true, SupportsExtendedThinking: false, SupportsExtendedContext: false,
		Speed: "fast", Intelligence: "good", CostTier: "low",
	},
	"glm-5": {
		ContextWindow: 128_000, ExtendedContextWindow: 0,
		MaxOutput: 16_000, MaxOutputExtended: 0,
		SupportsMultimodal: true, SupportsExtendedThinking: false, SupportsExtendedContext: false,
		Speed: "fast", Intelligence: "good", CostTier: "low",
	},
	"qwen3-coder-next": {
		ContextWindow: 128_000, ExtendedContextWindow: 0,
		MaxOutput: 16_000, MaxOutputExtended: 0,
		SupportsMultimodal: false, SupportsExtendedThinking: true, SupportsExtendedContext: false,
		Speed: "medium", Intelligence: "high", CostTier: "low",
	},
}

// ModelLimit defines limits for a specific model
type ModelLimit struct {
	// Context window limits
	ContextWindow         int  // Standard context window
	ExtendedContextWindow int  // Extended context (beta), 0 if not supported
	MaxOutput             int  // Max output tokens (standard context)
	MaxOutputExtended     int  // Max output tokens (extended context), 0 if not supported
	
	// Feature support
	SupportsMultimodal       bool // Images support
	SupportsExtendedThinking bool // Multi-hour extended thinking
	SupportsExtendedContext  bool // Beta extended context window
	
	// Model characteristics
	Speed        string // "fast", "medium", "slow"
	Intelligence string // "good", "high", "highest"
	CostTier     string // "low", "medium", "high"
}

// GetModelLimit returns the limits for a given model
func GetModelLimit(modelID string) (ModelLimit, error) {
	limit, ok := ModelLimits[modelID]
	if !ok {
		// Default to Sonnet 4 limits for unknown models
		return ModelLimits["claude-sonnet-4"], fmt.Errorf("unknown model %s, using default limits", modelID)
	}
	return limit, nil
}

// GetEffectiveContextWindow returns the context window based on beta feature flag
func (m ModelLimit) GetEffectiveContextWindow(enableExtendedContext bool) int {
	if enableExtendedContext && m.SupportsExtendedContext && m.ExtendedContextWindow > 0 {
		return m.ExtendedContextWindow
	}
	return m.ContextWindow
}

// GetEffectiveMaxOutput returns the max output tokens based on context window
func (m ModelLimit) GetEffectiveMaxOutput(enableExtendedContext bool) int {
	if enableExtendedContext && m.SupportsExtendedContext && m.MaxOutputExtended > 0 {
		return m.MaxOutputExtended
	}
	return m.MaxOutput
}

// ValidateFeatureSupport checks if a feature is supported by the model
func (m ModelLimit) ValidateFeatureSupport(feature string) error {
	switch feature {
	case "multimodal":
		if !m.SupportsMultimodal {
			return fmt.Errorf("model does not support multimodal (images)")
		}
	case "extended_thinking":
		if !m.SupportsExtendedThinking {
			return fmt.Errorf("model does not support extended thinking")
		}
	case "extended_context":
		if !m.SupportsExtendedContext {
			return fmt.Errorf("model does not support extended context window (beta)")
		}
	default:
		return fmt.Errorf("unknown feature: %s", feature)
	}
	return nil
}

// Supported image formats
var SupportedImageFormats = map[string]bool{
	"png":  true,
	"jpeg": true,
	"jpg":  true,
	"webp": true,
	"gif":  true,
}

// IsSupportedImageFormat checks if an image format is supported
func IsSupportedImageFormat(format string) bool {
	return SupportedImageFormats[format]
}

// EstimateTokenCount estimates token count from character count
func EstimateTokenCount(chars int) int {
	return int(float64(chars) / CharsPerToken)
}

// EstimateCharCount estimates character count from token count
func EstimateCharCount(tokens int) int {
	return int(float64(tokens) * CharsPerToken)
}
