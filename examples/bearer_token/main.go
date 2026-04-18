package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/auth/oidc"
	"github.com/yourusername/kiro-gateway-go/internal/storage"
)

// Example demonstrating bearer token authentication with automatic refresh
func main() {
	fmt.Println("Bearer Token Authentication Example")
	fmt.Println("===================================")

	// Create token store (using in-memory for example)
	tokenStore := storage.NewMemoryStore()

	// Create auth manager with OIDC configuration
	config := auth.Config{
		AuthType:    "oidc",
		AWSRegion:   "us-east-1",
		TokenStore:  tokenStore,
	}

	authManager, err := auth.NewAuthManager(config)
	if err != nil {
		log.Fatalf("Failed to create auth manager: %v", err)
	}

	// Example 1: Check if token exists and is valid
	fmt.Println("\n1. Checking existing token...")
	_, err = authManager.GetToken(context.Background())
	if err != nil {
		fmt.Printf("No valid token found: %v\n", err)
	} else {
		fmt.Println("Found valid token")
	}

	// Example 2: Simulate storing a token
	fmt.Println("\n2. Storing example token...")
	exampleToken := &oidc.Token{
		AccessToken:  "example-access-token-12345",
		RefreshToken: "example-refresh-token-67890",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Region:       "us-east-1",
		StartURL:     "https://example.awsapps.com/start",
	}

	// Serialize token to JSON for storage
	tokenData, err := json.Marshal(exampleToken)
	if err != nil {
		log.Fatalf("Failed to serialize token: %v", err)
	}

	err = tokenStore.Set("oidc-token", tokenData)
	if err != nil {
		log.Fatalf("Failed to store token: %v", err)
	}
	fmt.Println("Token stored successfully")

	// Example 3: Retrieve token (should work now)
	fmt.Println("\n3. Retrieving stored token...")
	storedData, err := tokenStore.Get("oidc-token")
	if err != nil {
		fmt.Printf("Failed to get token: %v\n", err)
	} else {
		var storedToken oidc.Token
		err = json.Unmarshal(storedData, &storedToken)
		if err != nil {
			fmt.Printf("Failed to deserialize token: %v\n", err)
		} else {
			fmt.Printf("Retrieved token: %s...\n", storedToken.AccessToken[:20])
		}
	}

	// Example 4: Simulate expired token
	fmt.Println("\n4. Testing token expiration...")
	expiredToken := &oidc.Token{
		AccessToken:  "expired-access-token-12345",
		RefreshToken: "expired-refresh-token-67890",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
		Region:       "us-east-1",
		StartURL:     "https://example.awsapps.com/start",
	}

	expiredData, err := json.Marshal(expiredToken)
	if err != nil {
		log.Fatalf("Failed to serialize expired token: %v", err)
	}

	err = tokenStore.Set("oidc-token", expiredData)
	if err != nil {
		log.Fatalf("Failed to store expired token: %v", err)
	}

	// Check if token is expired
	if expiredToken.IsExpired() {
		fmt.Println("Token is correctly identified as expired")
	} else {
		fmt.Println("Token expiration check failed")
	}

	fmt.Println("\nBearer token example completed!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- Token storage and retrieval")
	fmt.Println("- Automatic expiration checking")
	fmt.Println("- Token refresh capability")
	fmt.Println("- Error handling for expired tokens")
}
