package credentials

import (
	"context"
	"os"
	"testing"
)

func TestEnvProvider_Retrieve(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		wantSource  string
	}{
		{
			name: "successful credential retrieval",
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
			wantErr:    false,
			wantSource: "environment",
		},
		{
			name: "successful without session token",
			setupEnv: func() {
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			},
			wantErr:    false,
			wantSource: "environment",
		},
		{
			name: "missing access key",
			setupEnv: func() {
				os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			},
			wantErr: true,
		},
		{
			name: "missing secret key",
			setupEnv: func() {
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
			},
			wantErr: true,
		},
		{
			name: "missing both keys",
			setupEnv: func() {
				// No environment variables set
			},
			cleanupEnv: func() {
				// Nothing to clean up
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupEnv()
			defer tt.cleanupEnv()

			provider := NewEnvProvider()
			creds, err := provider.Retrieve(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("EnvProvider.Retrieve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if creds == nil {
					t.Error("EnvProvider.Retrieve() returned nil credentials")
					return
				}

				if creds.Source != tt.wantSource {
					t.Errorf("EnvProvider.Retrieve() source = %v, want %v", creds.Source, tt.wantSource)
				}

				if creds.AccessKeyID == "" {
					t.Error("EnvProvider.Retrieve() returned empty AccessKeyID")
				}

				if creds.SecretAccessKey == "" {
					t.Error("EnvProvider.Retrieve() returned empty SecretAccessKey")
				}

				if creds.CanExpire {
					t.Error("EnvProvider.Retrieve() credentials should not expire")
				}
			}
		})
	}
}

func TestEnvProvider_IsExpired(t *testing.T) {
	provider := NewEnvProvider()
	if provider.IsExpired() {
		t.Error("EnvProvider.IsExpired() should always return false")
	}
}
