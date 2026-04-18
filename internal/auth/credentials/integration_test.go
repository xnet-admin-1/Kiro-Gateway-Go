package credentials

import (
	"context"
	"os"
	"testing"
)

func TestCredentialChain_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name       string
		setupEnv   func()
		cleanupEnv func()
		wantSource string
		wantErr    bool
	}{
		{
			name: "environment credentials",
			setupEnv: func() {
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
				os.Setenv("AWS_SESSION_TOKEN", "session-token")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				os.Unsetenv("AWS_SESSION_TOKEN")
			},
			wantSource: "environment",
			wantErr:    false,
		},
		{
			name: "no credentials available",
			setupEnv: func() {
				// Ensure no environment variables are set
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				os.Unsetenv("AWS_SESSION_TOKEN")
				os.Unsetenv("AWS_PROFILE")
				os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
				os.Unsetenv("AWS_ROLE_ARN")
				os.Unsetenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
			},
			cleanupEnv: func() {
				// Nothing to clean up
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			chain := NewDefaultChain()
			creds, err := chain.Retrieve(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Chain.Retrieve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if creds == nil {
					t.Error("Chain.Retrieve() returned nil credentials")
					return
				}

				if creds.Source != tt.wantSource {
					t.Errorf("Chain.Retrieve() source = %v, want %v", creds.Source, tt.wantSource)
				}

				if creds.AccessKeyID == "" {
					t.Error("Chain.Retrieve() returned empty AccessKeyID")
				}

				if creds.SecretAccessKey == "" {
					t.Error("Chain.Retrieve() returned empty SecretAccessKey")
				}

				// Test caching
				creds2, err := chain.Retrieve(context.Background())
				if err != nil {
					t.Errorf("Second Chain.Retrieve() failed: %v", err)
				}

				if creds2.AccessKeyID != creds.AccessKeyID {
					t.Error("Cached credentials differ from original")
				}
			}
		})
	}
}

func TestCredentialChain_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test")
	}

	// Set up environment credentials for fast retrieval
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	chain := NewDefaultChain()

	// First call to populate cache
	_, err := chain.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Initial retrieve failed: %v", err)
	}

	// Measure cached retrieval performance
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		_, err := chain.Retrieve(context.Background())
		if err != nil {
			t.Fatalf("Retrieve %d failed: %v", i, err)
		}
	}

	// This test mainly ensures no panics or errors during repeated calls
	// Performance measurement would require benchmarking
}
