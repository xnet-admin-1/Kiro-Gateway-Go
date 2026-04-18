package credentials

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileProvider_Retrieve(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "aws-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .aws directory
	awsDir := filepath.Join(tempDir, ".aws")
	if err := os.MkdirAll(awsDir, 0755); err != nil {
		t.Fatalf("Failed to create .aws dir: %v", err)
	}

	// Create credentials file
	credentialsFile := filepath.Join(awsDir, "credentials")
	credentialsContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[test-profile]
aws_access_key_id = AKIATEST123456789
aws_secret_access_key = testSecretKey123456789
aws_session_token = testSessionToken
`
	if err := os.WriteFile(credentialsFile, []byte(credentialsContent), 0644); err != nil {
		t.Fatalf("Failed to write credentials file: %v", err)
	}

	// Create config file
	configFile := filepath.Join(awsDir, "config")
	configContent := `[default]
region = us-east-1

[profile config-profile]
aws_access_key_id = AKIACONFIG123456
aws_secret_access_key = configSecretKey123456
region = us-west-2
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Mock home directory
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
			os.Setenv("USERPROFILE", originalHome)
		}
	}()
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	tests := []struct {
		name        string
		profile     string
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		wantSource  string
		wantKeyID   string
	}{
		{
			name:       "default profile from credentials",
			profile:    "default",
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    false,
			wantSource: "profile",
			wantKeyID:  "AKIAIOSFODNN7EXAMPLE",
		},
		{
			name:       "named profile from credentials",
			profile:    "test-profile",
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    false,
			wantSource: "profile",
			wantKeyID:  "AKIATEST123456789",
		},
		{
			name:       "profile from config file",
			profile:    "config-profile",
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    false,
			wantSource: "profile",
			wantKeyID:  "AKIACONFIG123456",
		},
		{
			name:       "AWS_PROFILE environment variable",
			profile:    "", // Will use NewProfileProvider() which reads AWS_PROFILE
			setupEnv:   func() { os.Setenv("AWS_PROFILE", "test-profile") },
			cleanupEnv: func() { os.Unsetenv("AWS_PROFILE") },
			wantErr:    false,
			wantSource: "profile",
			wantKeyID:  "AKIATEST123456789",
		},
		{
			name:       "nonexistent profile",
			profile:    "nonexistent",
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			var provider *ProfileProvider
			if tt.profile == "" {
				provider = NewProfileProvider()
			} else {
				provider = NewProfileProviderWithProfile(tt.profile)
			}

			creds, err := provider.Retrieve(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ProfileProvider.Retrieve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if creds == nil {
					t.Error("ProfileProvider.Retrieve() returned nil credentials")
					return
				}

				if creds.Source != tt.wantSource {
					t.Errorf("ProfileProvider.Retrieve() source = %v, want %v", creds.Source, tt.wantSource)
				}

				if creds.AccessKeyID != tt.wantKeyID {
					t.Errorf("ProfileProvider.Retrieve() AccessKeyID = %v, want %v", creds.AccessKeyID, tt.wantKeyID)
				}

				if creds.SecretAccessKey == "" {
					t.Error("ProfileProvider.Retrieve() returned empty SecretAccessKey")
				}

				if creds.CanExpire {
					t.Error("ProfileProvider.Retrieve() credentials should not expire")
				}
			}
		})
	}
}

func TestProfileProvider_IsExpired(t *testing.T) {
	provider := NewProfileProvider()
	if provider.IsExpired() {
		t.Error("ProfileProvider.IsExpired() should always return false")
	}
}
