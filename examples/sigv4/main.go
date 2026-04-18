package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
	"github.com/yourusername/kiro-gateway-go/internal/auth/sigv4"
)

// Example demonstrating SigV4 authentication with AWS credentials
func main() {
	fmt.Println("SigV4 Authentication Example")
	fmt.Println("=============================")

	// Example 1: Create credential chain
	fmt.Println("\n1. Setting up AWS credential chain...")
	chain := credentials.NewChain()

	// Retrieve credentials
	creds, err := chain.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("Failed to retrieve credentials: %v\n", err)
		fmt.Println("Note: This is expected if no AWS credentials are configured")
		
		// Use example credentials for demonstration
		creds = &credentials.Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			SessionToken:    "",
			Source:          "example",
		}
		fmt.Println("Using example credentials for demonstration")
	} else {
		fmt.Printf("Retrieved credentials from: %s\n", creds.Source)
	}

	// Example 2: Create SigV4 signer
	fmt.Println("\n2. Creating SigV4 signer...")
	signer := sigv4.NewSigner(creds, "us-east-1", "codewhisperer")
	fmt.Println("SigV4 signer created successfully")

	// Example 3: Sign a GET request
	fmt.Println("\n3. Signing GET request...")
	req, err := http.NewRequest("GET", "https://codewhisperer.us-east-1.amazonaws.com/ListAvailableModels", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Add required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "kiro-gateway/1.0.0")

	err = signer.SignRequest(req, nil)
	if err != nil {
		log.Fatalf("Failed to sign request: %v", err)
	}

	fmt.Println("GET request signed successfully")
	authHeader := req.Header.Get("Authorization")
	if len(authHeader) > 50 {
		fmt.Printf("Authorization header: %s...\n", authHeader[:50])
	} else {
		fmt.Printf("Authorization header: %s\n", authHeader)
	}

	// Example 4: Sign a POST request with body
	fmt.Println("\n4. Signing POST request with body...")
	requestBody := `{"message": "Hello, Q Developer!"}`
	postReq, err := http.NewRequest("POST", "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse", strings.NewReader(requestBody))
	if err != nil {
		log.Fatalf("Failed to create POST request: %v", err)
	}

	// Add required headers
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("User-Agent", "kiro-gateway/1.0.0")

	err = signer.SignRequest(postReq, []byte(requestBody))
	if err != nil {
		log.Fatalf("Failed to sign POST request: %v", err)
	}

	fmt.Println("POST request signed successfully")
	postAuthHeader := postReq.Header.Get("Authorization")
	if len(postAuthHeader) > 50 {
		fmt.Printf("Authorization header: %s...\n", postAuthHeader[:50])
	} else {
		fmt.Printf("Authorization header: %s\n", postAuthHeader)
	}

	// Example 5: Demonstrate different credential sources
	fmt.Println("\n5. Testing different credential sources...")
	
	// Environment provider
	envProvider := credentials.NewEnvProvider()
	envCreds, err := envProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("Environment credentials: Not available (%v)\n", err)
	} else {
		fmt.Printf("Environment credentials: Available from %s\n", envCreds.Source)
	}

	// Profile provider
	profileProvider := credentials.NewProfileProvider()
	profileCreds, err := profileProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("Profile credentials: Not available (%v)\n", err)
	} else {
		fmt.Printf("Profile credentials: Available from %s\n", profileCreds.Source)
	}

	fmt.Println("\nSigV4 example completed!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- AWS credential chain resolution")
	fmt.Println("- SigV4 request signing")
	fmt.Println("- GET and POST request handling")
	fmt.Println("- Multiple credential sources")
	fmt.Println("- Proper header management")
}
