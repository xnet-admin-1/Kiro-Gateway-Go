package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth/oidc"
	"github.com/yourusername/kiro-gateway-go/internal/storage"
)

// Example demonstrating OIDC device code flow for browser-less authentication
func main() {
	fmt.Println("OIDC Device Code Flow Example")
	fmt.Println("==============================")

	// Example 1: Create OIDC client
	fmt.Println("\n1. Creating OIDC client...")
	client := oidc.NewClient("us-east-1")
	fmt.Println("OIDC client created successfully")

	// Example 2: Device registration (simulated)
	fmt.Println("\n2. Device registration...")
	
	// In a real scenario, this would call AWS SSO-OIDC RegisterClient API
	// For this example, we'll simulate the registration
	registration := &oidc.DeviceRegistration{
		ClientID:              "example-client-id-12345",
		ClientSecret:          "example-client-secret-67890",
		ClientSecretExpiresAt: time.Now().Add(24 * time.Hour),
		Region:                "us-east-1",
	}

	fmt.Printf("Device registered with client ID: %s...\n", registration.ClientID[:20])

	// Example 3: Start device authorization (simulated)
	fmt.Println("\n3. Starting device authorization...")
	
	// In a real scenario, this would call StartDeviceAuthorization API
	authorization := &oidc.DeviceAuthorization{
		DeviceCode:              "example-device-code-12345",
		UserCode:                "ABCD-EFGH",
		VerificationURI:         "https://device.sso.us-east-1.amazonaws.com/",
		VerificationURIComplete: "https://device.sso.us-east-1.amazonaws.com/?user_code=ABCD-EFGH",
		ExpiresIn:               900, // 15 minutes
		Interval:                5,   // Poll every 5 seconds
	}

	fmt.Printf("Device authorization started:\n")
	fmt.Printf("  User Code: %s\n", authorization.UserCode)
	fmt.Printf("  Verification URL: %s\n", authorization.VerificationURI)
	fmt.Printf("  Complete URL: %s\n", authorization.VerificationURIComplete)
	fmt.Printf("  Expires in: %d seconds\n", authorization.ExpiresIn)

	// Example 4: Simulate user instruction
	fmt.Println("\n4. User instructions:")
	fmt.Println("In a real scenario, the user would:")
	fmt.Printf("1. Open: %s\n", authorization.VerificationURI)
	fmt.Printf("2. Enter code: %s\n", authorization.UserCode)
	fmt.Println("3. Complete authorization in browser")
	fmt.Println("4. Application would poll for token...")

	// Example 5: Simulate token polling result
	fmt.Println("\n5. Simulating successful token retrieval...")
	
	// In a real scenario, this would be the result of polling CreateToken API
	token := &oidc.Token{
		AccessToken:  "example-access-token-from-device-flow",
		RefreshToken: "example-refresh-token-from-device-flow",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Region:       "us-east-1",
		StartURL:     "https://example.awsapps.com/start",
	}

	fmt.Printf("Token retrieved successfully:\n")
	fmt.Printf("  Access Token: %s...\n", token.AccessToken[:30])
	fmt.Printf("  Expires At: %s\n", token.ExpiresAt.Format(time.RFC3339))

	// Example 6: Store token securely
	fmt.Println("\n6. Storing token securely...")
	tokenStore := storage.NewMemoryStore() // In production, use keychain
	
	tokenData, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Failed to serialize token: %v", err)
	}

	err = tokenStore.Set("oidc-device-token", tokenData)
	if err != nil {
		log.Fatalf("Failed to store token: %v", err)
	}
	fmt.Println("Token stored securely")

	// Example 7: Retrieve and validate stored token
	fmt.Println("\n7. Retrieving stored token...")
	storedData, err := tokenStore.Get("oidc-device-token")
	if err != nil {
		log.Fatalf("Failed to retrieve token: %v", err)
	}

	var storedToken oidc.Token
	err = json.Unmarshal(storedData, &storedToken)
	if err != nil {
		log.Fatalf("Failed to deserialize token: %v", err)
	}

	if storedToken.IsExpired() {
		fmt.Println("Token is expired and needs refresh")
	} else {
		fmt.Printf("Token is valid until: %s\n", storedToken.ExpiresAt.Format(time.RFC3339))
	}

	// Example 8: Demonstrate token refresh
	fmt.Println("\n8. Token refresh capability...")
	fmt.Println("In a real scenario, when token expires:")
	fmt.Println("- Check if refresh token exists")
	fmt.Println("- Call CreateToken API with refresh_token grant")
	fmt.Println("- Update stored token with new values")
	fmt.Println("- Handle refresh token rotation")

	fmt.Println("\nOIDC device code flow example completed!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- Device registration with AWS SSO-OIDC")
	fmt.Println("- Device authorization flow")
	fmt.Println("- User code display and instructions")
	fmt.Println("- Token polling simulation")
	fmt.Println("- Secure token storage")
	fmt.Println("- Token expiration checking")
	fmt.Println("- Refresh token capability")

	// Suppress unused variable warnings
	_ = client
}
