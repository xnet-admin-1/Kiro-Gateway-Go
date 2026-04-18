package adapters

import (
	"github.com/yourusername/kiro-gateway-go/internal/auth"
)

// DetermineAPIEndpoint determines the correct API endpoint based on auth mode and Q Developer usage
// This ensures consistent endpoint selection across all adapters and handlers
func DetermineAPIEndpoint(authMode auth.AuthMode, useQDeveloper bool) string {
	if useQDeveloper {
		// Q Developer mode
		if authMode == auth.AuthModeSigV4 {
			// SigV4 authentication uses "/" with X-Amz-Target header (JSON-RPC style)
			return "/"
		}
		// Bearer token mode uses /generateAssistantResponse even for Q Developer
		return "/generateAssistantResponse"
	}
	
	// CodeWhisperer endpoint (text-only, legacy)
	return "/generateAssistantResponse"
}

// GetAPIEndpointDescription returns a human-readable description of the endpoint selection
func GetAPIEndpointDescription(authMode auth.AuthMode, useQDeveloper bool) string {
	endpoint := DetermineAPIEndpoint(authMode, useQDeveloper)
	
	switch {
	case endpoint == "/" && authMode == auth.AuthModeSigV4:
		return "Q Developer (SigV4) - JSON-RPC style with X-Amz-Target header"
	case endpoint == "/generateAssistantResponse" && useQDeveloper:
		return "Q Developer (Bearer Token) - generateAssistantResponse endpoint"
	case endpoint == "/generateAssistantResponse" && !useQDeveloper:
		return "CodeWhisperer - generateAssistantResponse endpoint"
	default:
		return "Unknown endpoint configuration"
	}
}
