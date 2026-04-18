package models

import "encoding/json"

// Q Developer API Conversation State Types
// Based on AWS Q Developer CLI source code

// ConversationStateRequest is the top-level request structure for Q Developer API
type ConversationStateRequest struct {
	ConversationState *ConversationState `json:"conversationState"`
	ProfileArn        string             `json:"profileArn,omitempty"` // Only for Identity Center mode
	Source            string             `json:"source,omitempty"`     // "CLI", "IDE", "AI_EDITOR" - top-level origin (separate from UserInputMessage.Origin)
	CustomizationArn  string             `json:"customizationArn,omitempty"`
	Tools             []ToolSpecWrapper  `json:"tools,omitempty"`
}

// ConversationState represents the conversation state sent to Q Developer API
type ConversationState struct {
	ConversationID  *string          `json:"conversationId"` // null for first message, UUID for subsequent (removed omitempty to send explicit null)
	CurrentMessage  ChatMessage      `json:"currentMessage"`
	ChatTriggerType string           `json:"chatTriggerType"` // "MANUAL", "INLINE_CHAT", "DIAGNOSTIC"
	History         []ChatMessage    `json:"history,omitempty"`
}

// ChatMessage represents a message in the conversation (user or assistant)
type ChatMessage struct {
	UserInputMessage        *UserInputMessage        `json:"userInputMessage,omitempty"`
	AssistantResponseMessage *AssistantResponseMessage `json:"assistantResponseMessage,omitempty"`
}

// UserInputMessage represents a message from the user
type UserInputMessage struct {
	Content                 string                   `json:"content"`
	UserInputMessageContext *UserInputMessageContext `json:"userInputMessageContext,omitempty"`
	UserIntent              *string                  `json:"userIntent,omitempty"`
	Images                  []ImageBlock             `json:"images,omitempty"`
	ToolResults             []ToolResult             `json:"toolResults,omitempty"`
	Origin                  string                   `json:"origin,omitempty"` // "CLI", "IDE", "AI_EDITOR", etc. - required for multimodal
	ModelID                 string                   `json:"modelId,omitempty"` // Model ID - required for multimodal (camelCase per Q CLI source)
}

// UserInputMessageContext provides context about the user's environment
type UserInputMessageContext struct {
	EditorState *EditorState `json:"editorState,omitempty"`
	EnvState    *EnvState    `json:"envState,omitempty"`
	GitState    *GitState    `json:"gitState,omitempty"`
	ToolResults []ToolResult `json:"toolResults,omitempty"`
	Tools       []ToolSpecWrapper `json:"tools,omitempty"`
}

// EnvState represents the state of the user's environment
type EnvState struct {
	OperatingSystem         *string              `json:"operatingSystem,omitempty"`
	CurrentWorkingDirectory *string              `json:"currentWorkingDirectory,omitempty"`
	EnvironmentVariables    []EnvironmentVariable `json:"environmentVariables,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GitState represents the state of the user's git repository
type GitState struct {
	Status string `json:"status"`
}

// EditorState represents the state of the user's editor
type EditorState struct {
	Document *Document `json:"document,omitempty"`
}

// Document represents a document in the editor
type Document struct {
	RelativeFilePath    string               `json:"relativeFilePath,omitempty"`
	ProgrammingLanguage *ProgrammingLanguage `json:"programmingLanguage,omitempty"`
	Text                string               `json:"text,omitempty"`
	CursorState         *CursorState         `json:"cursorState,omitempty"`
}

// ProgrammingLanguage represents the programming language of a document
type ProgrammingLanguage struct {
	LanguageName string `json:"languageName"`
}

// CursorState represents the cursor position in a document
type CursorState struct {
	Position *Position `json:"position,omitempty"`
}

// Position represents a position in a document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// AssistantResponseMessage represents a message from the assistant
type AssistantResponseMessage struct {
	Content   string     `json:"content"`
	MessageID *string    `json:"messageId,omitempty"` // Also called utterance_id
	ToolUses  []ToolUse  `json:"toolUses,omitempty"`
}

// ToolUse represents a tool use request from the assistant
type ToolUse struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
	Input     json.RawMessage `json:"input"` // JSON object
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolUseID string              `json:"toolUseId"`
	Content   []ToolResultContent `json:"content"`
	Status    string              `json:"status"` // "success", "error"
}

// ToolResultContent represents content in a tool result
type ToolResultContent struct {
	Text string      `json:"text,omitempty"`
	JSON interface{} `json:"json,omitempty"` // JSON content as object (AWS expects object, not string)
}

// ImageBlock represents an image in a message
// CRITICAL: Based on Q CLI serialization code (shape_image_block.rs):
// The structure is: {"format": "png", "source": {"bytes": "base64..."}}
// The ImageSource enum is serialized INSIDE the "source" object
type ImageBlock struct {
	Format string      `json:"format"` // "png", "jpeg", "gif", "webp"
	Source ImageSource `json:"source"` // Nested structure containing bytes
}

// ImageSource represents the source of an image
// In Q CLI this is an enum, but we represent it as a struct with the Bytes variant
type ImageSource struct {
	Bytes []byte `json:"bytes"` // Raw bytes, JSON encoder auto-base64-encodes
}

// ToolSpecWrapper wraps a tool specification
type ToolSpecWrapper struct {
	ToolSpecification *ToolSpecification `json:"toolSpecification"`
}

// ToolSpecification defines a tool that can be used by the assistant
type ToolSpecification struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema ToolInputSchema  `json:"inputSchema"`
}

// ToolInputSchema defines the input schema for a tool
type ToolInputSchema struct {
	JSON interface{} `json:"json"` // JSON schema as object or string (AWS expects object)
}

// Streaming Response Types

// ChatResponseStream represents a streaming response event
type ChatResponseStream struct {
	// Event types
	AssistantResponseEvent       *AssistantResponseEvent       `json:"assistantResponseEvent,omitempty"`
	CodeReferenceEvent           *CodeReferenceEvent           `json:"codeReferenceEvent,omitempty"`
	SupplementaryWebLinksEvent   *SupplementaryWebLinksEvent   `json:"supplementaryWebLinksEvent,omitempty"`
	FollowupPromptEvent          *FollowupPromptEvent          `json:"followupPromptEvent,omitempty"`
	MessageMetadataEvent         *MessageMetadataEvent         `json:"messageMetadataEvent,omitempty"`
	Error                        *ErrorEvent                   `json:"error,omitempty"`
	IntentsEvent                 *IntentsEvent                 `json:"intentsEvent,omitempty"`
	InvalidStateEvent            *InvalidStateEvent            `json:"invalidStateEvent,omitempty"`
	CodeEvent                    *CodeEvent                    `json:"codeEvent,omitempty"`
}

// AssistantResponseEvent contains assistant response content
type AssistantResponseEvent struct {
	Content string `json:"content"`
}

// CodeReferenceEvent contains code reference information
type CodeReferenceEvent struct {
	References []CodeReference `json:"references,omitempty"`
}

// CodeReference represents a reference to external code
type CodeReference struct {
	LicenseName  string `json:"licenseName,omitempty"`
	Repository   string `json:"repository,omitempty"`
	URL          string `json:"url,omitempty"`
	RecommendationContentSpan *RecommendationContentSpan `json:"recommendationContentSpan,omitempty"`
}

// RecommendationContentSpan represents the span of recommended content
type RecommendationContentSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// SupplementaryWebLinksEvent contains supplementary web links
type SupplementaryWebLinksEvent struct {
	SupplementaryWebLinks []SupplementaryWebLink `json:"supplementaryWebLinks,omitempty"`
}

// SupplementaryWebLink represents a supplementary web link
type SupplementaryWebLink struct {
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// FollowupPromptEvent contains follow-up prompt suggestions
type FollowupPromptEvent struct {
	FollowupPrompt *FollowupPrompt `json:"followupPrompt,omitempty"`
}

// FollowupPrompt represents a suggested follow-up prompt
type FollowupPrompt struct {
	Content string `json:"content"`
	UserIntent string `json:"userIntent,omitempty"`
}

// MessageMetadataEvent contains metadata about the message
type MessageMetadataEvent struct {
	ConversationID string `json:"conversationId,omitempty"`
	UtteranceID    string `json:"utteranceId,omitempty"` // Same as messageId
	FinalMessage   bool   `json:"finalMessage,omitempty"`
}

// ErrorEvent represents an error in the stream
type ErrorEvent struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// IntentsEvent contains detected user intents
type IntentsEvent struct {
	Intents []Intent `json:"intents,omitempty"`
}

// Intent represents a detected user intent
type Intent struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence,omitempty"`
}

// InvalidStateEvent indicates an invalid conversation state
type InvalidStateEvent struct {
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
}

// CodeEvent contains code-related events
type CodeEvent struct {
	Content string `json:"content"`
	Type    string `json:"type,omitempty"` // "code_block", "inline_code"
}

// AWS Profile Types (for ListAvailableProfiles API)

// ListAvailableProfilesResponse represents the response from listing available profiles
type ListAvailableProfilesResponse struct {
	Profiles  []AWSProfile `json:"profiles"`
	NextToken *string      `json:"nextToken,omitempty"`
}

// AWSProfile represents a profile from AWS Q Developer API
type AWSProfile struct {
	Arn         string `json:"arn"`
	ProfileName string `json:"profileName"`
}

// AWS Model Types (for ListAvailableModels API)

// ListAvailableModelsRequest represents the request to list available models
type ListAvailableModelsRequest struct {
	Origin        string  `json:"origin"`        // Required: "CLI", "IDE", etc.
	ProfileArn    *string `json:"profileArn,omitempty"`
	MaxResults    *int    `json:"maxResults,omitempty"`
	NextToken     *string `json:"nextToken,omitempty"`
	ModelProvider *string `json:"modelProvider,omitempty"`
}

// ListAvailableModelsResponse represents the response from listing available models
type ListAvailableModelsResponse struct {
	Models       []AWSModel `json:"models"`
	DefaultModel AWSModel   `json:"defaultModel"`
	NextToken    *string    `json:"nextToken,omitempty"`
}

// AWSModel represents a model from AWS Q Developer API
type AWSModel struct {
	ModelID              string        `json:"modelId"`
	ModelName            *string       `json:"modelName,omitempty"`
	Description          *string       `json:"description,omitempty"`
	RateMultiplier       *float64      `json:"rateMultiplier,omitempty"`
	RateUnit             *string       `json:"rateUnit,omitempty"`
	TokenLimits          *TokenLimits  `json:"tokenLimits,omitempty"`
	SupportedInputTypes  []string      `json:"supportedInputTypes,omitempty"` // "TEXT", "IMAGE"
	SupportsPromptCache  *bool         `json:"supportsPromptCache,omitempty"`
}

// TokenLimits represents token limits for a model
type TokenLimits struct {
	MaxInputTokens  *int `json:"maxInputTokens,omitempty"`
	MaxOutputTokens *int `json:"maxOutputTokens,omitempty"`
}
