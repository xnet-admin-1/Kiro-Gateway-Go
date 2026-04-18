package main

import (
	"context"
	"fmt"
	"os"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
)

// Example demonstrating AWS credential provider chain usage
func main() {
	fmt.Println("AWS Credential Chain Example")
	fmt.Println("=============================")

	// Example 1: Create and test individual providers
	fmt.Println("\n1. Testing individual credential providers...")

	// Environment provider
	fmt.Println("\n   a) Environment Provider:")
	envProvider := credentials.NewEnvProvider()
	envCreds, err := envProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("      ❌ Environment credentials not available: %v\n", err)
	} else {
		fmt.Printf("      ✅ Environment credentials found (Source: %s)\n", envCreds.Source)
		if len(envCreds.AccessKeyID) >= 10 {
			fmt.Printf("      Access Key: %s...\n", envCreds.AccessKeyID[:10])
		} else {
			fmt.Printf("      Access Key: %s\n", envCreds.AccessKeyID)
		}
	}

	// Profile provider
	fmt.Println("\n   b) Profile Provider:")
	profileProvider := credentials.NewProfileProvider()
	profileCreds, err := profileProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("      ❌ Profile credentials not available: %v\n", err)
	} else {
		fmt.Printf("      ✅ Profile credentials found (Source: %s)\n", profileCreds.Source)
		if len(profileCreds.AccessKeyID) >= 10 {
			fmt.Printf("      Access Key: %s...\n", profileCreds.AccessKeyID[:10])
		} else {
			fmt.Printf("      Access Key: %s\n", profileCreds.AccessKeyID)
		}
	}

	// Web Identity provider
	fmt.Println("\n   c) Web Identity Provider:")
	webIdentityProvider := credentials.NewWebIdentityProvider()
	webCreds, err := webIdentityProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("      ❌ Web Identity credentials not available: %v\n", err)
	} else {
		fmt.Printf("      ✅ Web Identity credentials found (Source: %s)\n", webCreds.Source)
		if len(webCreds.AccessKeyID) >= 10 {
			fmt.Printf("      Access Key: %s...\n", webCreds.AccessKeyID[:10])
		} else {
			fmt.Printf("      Access Key: %s\n", webCreds.AccessKeyID)
		}
	}

	// ECS provider
	fmt.Println("\n   d) ECS Provider:")
	ecsProvider := credentials.NewECSProvider()
	ecsCreds, err := ecsProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("      ❌ ECS credentials not available: %v\n", err)
	} else {
		fmt.Printf("      ✅ ECS credentials found (Source: %s)\n", ecsCreds.Source)
		if len(ecsCreds.AccessKeyID) >= 10 {
			fmt.Printf("      Access Key: %s...\n", ecsCreds.AccessKeyID[:10])
		} else {
			fmt.Printf("      Access Key: %s\n", ecsCreds.AccessKeyID)
		}
	}

	// IMDS provider
	fmt.Println("\n   e) IMDS Provider:")
	imdsProvider := credentials.NewIMDSProvider()
	imdsCreds, err := imdsProvider.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("      ❌ IMDS credentials not available: %v\n", err)
	} else {
		fmt.Printf("      ✅ IMDS credentials found (Source: %s)\n", imdsCreds.Source)
		if len(imdsCreds.AccessKeyID) >= 10 {
			fmt.Printf("      Access Key: %s...\n", imdsCreds.AccessKeyID[:10])
		} else {
			fmt.Printf("      Access Key: %s\n", imdsCreds.AccessKeyID)
		}
	}

	// Example 2: Use credential chain (standard AWS order)
	fmt.Println("\n2. Using AWS credential chain...")
	chain := credentials.NewChain()

	creds, err := chain.Retrieve(context.Background())
	if err != nil {
		fmt.Printf("❌ No credentials found in chain: %v\n", err)
		fmt.Println("\nTo test with real credentials, try:")
		fmt.Println("   export AWS_ACCESS_KEY_ID=your-key")
		fmt.Println("   export AWS_SECRET_ACCESS_KEY=your-secret")
		fmt.Println("   or configure AWS profile with 'aws configure'")
	} else {
		fmt.Printf("✅ Credentials found via: %s\n", creds.Source)
		if len(creds.AccessKeyID) >= 10 {
			fmt.Printf("   Access Key: %s...\n", creds.AccessKeyID[:10])
		} else {
			fmt.Printf("   Access Key: %s\n", creds.AccessKeyID)
		}
		fmt.Printf("   Has Session Token: %t\n", creds.SessionToken != "")
		fmt.Printf("   Can Expire: %t\n", creds.CanExpire)
		if creds.CanExpire {
			fmt.Printf("   Expires At: %s\n", creds.Expires.Format("2006-01-02 15:04:05"))
		}
	}

	// Example 3: Demonstrate credential caching
	fmt.Println("\n3. Testing credential caching...")
	
	// First retrieval
	creds1, err1 := chain.Retrieve(context.Background())
	
	// Second retrieval (should use cache)
	creds2, err2 := chain.Retrieve(context.Background())
	
	if err1 != nil || err2 != nil {
		fmt.Println("   Caching test skipped (no credentials available)")
	} else {
		fmt.Printf("   First retrieval: %s (Source: %s)\n", creds1.AccessKeyID[:10], creds1.Source)
		fmt.Printf("   Second retrieval: %s (Source: %s)\n", creds2.AccessKeyID[:10], creds2.Source)
		fmt.Printf("   Same instance: %t (indicates caching)\n", creds1 == creds2)
	}

	// Example 4: Environment variable demonstration
	fmt.Println("\n4. Environment variable configuration examples...")
	fmt.Println("   Set these environment variables to test:")
	fmt.Println()
	fmt.Println("   # Basic credentials:")
	fmt.Println("   export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
	fmt.Println("   export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	fmt.Println()
	fmt.Println("   # With session token (temporary credentials):")
	fmt.Println("   export AWS_SESSION_TOKEN=your-session-token")
	fmt.Println()
	fmt.Println("   # Profile selection:")
	fmt.Println("   export AWS_PROFILE=your-profile-name")
	fmt.Println()
	fmt.Println("   # Web Identity Token (for Kubernetes IRSA):")
	fmt.Println("   export AWS_WEB_IDENTITY_TOKEN_FILE=/var/run/secrets/eks.amazonaws.com/serviceaccount/token")
	fmt.Println("   export AWS_ROLE_ARN=arn:aws:iam::123456789012:role/your-role")
	fmt.Println()
	fmt.Println("   # ECS Task Role:")
	fmt.Println("   export AWS_CONTAINER_CREDENTIALS_RELATIVE_URI=/v2/credentials/your-task-id")

	// Example 5: Show current environment
	fmt.Println("\n5. Current environment variables:")
	envVars := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY", 
		"AWS_SESSION_TOKEN",
		"AWS_PROFILE",
		"AWS_WEB_IDENTITY_TOKEN_FILE",
		"AWS_ROLE_ARN",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI",
	}

	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value != "" {
			if envVar == "AWS_SECRET_ACCESS_KEY" || envVar == "AWS_SESSION_TOKEN" {
				if len(value) >= 10 {
					fmt.Printf("   %s: %s... (redacted)\n", envVar, value[:10])
				} else {
					fmt.Printf("   %s: %s (redacted)\n", envVar, value)
				}
			} else {
				fmt.Printf("   %s: %s\n", envVar, value)
			}
		} else {
			fmt.Printf("   %s: (not set)\n", envVar)
		}
	}

	fmt.Println("\nCredential chain example completed!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- Individual credential provider testing")
	fmt.Println("- Standard AWS credential chain order")
	fmt.Println("- Credential caching mechanism")
	fmt.Println("- Environment variable configuration")
	fmt.Println("- Multiple authentication methods")
}
