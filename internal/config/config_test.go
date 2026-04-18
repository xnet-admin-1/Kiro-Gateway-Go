package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(*testing.T, *Config)
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Port != "8080" {
					t.Errorf("Port = %v, want 8080", cfg.Port)
				}
				if cfg.AWSRegion != "us-east-1" {
					t.Errorf("AWSRegion = %v, want us-east-1", cfg.AWSRegion)
				}
				if cfg.AWSProfile != "" {
					t.Errorf("AWSProfile = %v, want empty", cfg.AWSProfile)
				}
				if cfg.EnableSigV4 {
					t.Errorf("EnableSigV4 = %v, want false", cfg.EnableSigV4)
				}
				if cfg.OIDCStartURL != "" {
					t.Errorf("OIDCStartURL = %v, want empty", cfg.OIDCStartURL)
				}
				if cfg.OIDCRegion != "us-east-1" {
					t.Errorf("OIDCRegion = %v, want us-east-1", cfg.OIDCRegion)
				}
				if cfg.ProfileARN != "" {
					t.Errorf("ProfileARN = %v, want empty", cfg.ProfileARN)
				}
				if cfg.OptOutTelemetry {
					t.Errorf("OptOutTelemetry = %v, want false", cfg.OptOutTelemetry)
				}
				if cfg.MaxRetries != 3 {
					t.Errorf("MaxRetries = %v, want 3", cfg.MaxRetries)
				}
				if cfg.MaxBackoff != 30*time.Second {
					t.Errorf("MaxBackoff = %v, want 30s", cfg.MaxBackoff)
				}
				if cfg.ConnectTimeout != 10*time.Second {
					t.Errorf("ConnectTimeout = %v, want 10s", cfg.ConnectTimeout)
				}
				if cfg.ReadTimeout != 30*time.Second {
					t.Errorf("ReadTimeout = %v, want 30s", cfg.ReadTimeout)
				}
				if cfg.OperationTimeout != 60*time.Second {
					t.Errorf("OperationTimeout = %v, want 60s", cfg.OperationTimeout)
				}
				if cfg.StalledStreamGrace != 5*time.Minute {
					t.Errorf("StalledStreamGrace = %v, want 5m", cfg.StalledStreamGrace)
				}
			},
		},
		{
			name: "custom AWS configuration",
			envVars: map[string]string{
				"AWS_REGION":     "eu-west-1",
				"AWS_PROFILE":    "production",
				"AMAZON_Q_SIGV4": "true",
				"OIDC_START_URL": "https://my-sso.awsapps.com/start",
				"OIDC_REGION":    "eu-central-1",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.AWSRegion != "eu-west-1" {
					t.Errorf("AWSRegion = %v, want eu-west-1", cfg.AWSRegion)
				}
				if cfg.AWSProfile != "production" {
					t.Errorf("AWSProfile = %v, want production", cfg.AWSProfile)
				}
				if !cfg.EnableSigV4 {
					t.Errorf("EnableSigV4 = %v, want true", cfg.EnableSigV4)
				}
				if cfg.OIDCStartURL != "https://my-sso.awsapps.com/start" {
					t.Errorf("OIDCStartURL = %v, want https://my-sso.awsapps.com/start", cfg.OIDCStartURL)
				}
				if cfg.OIDCRegion != "eu-central-1" {
					t.Errorf("OIDCRegion = %v, want eu-central-1", cfg.OIDCRegion)
				}
			},
		},
		{
			name: "custom retry and timeout configuration",
			envVars: map[string]string{
				"MAX_RETRIES":          "5",
				"MAX_BACKOFF":          "1m",
				"CONNECT_TIMEOUT":      "5s",
				"READ_TIMEOUT":         "45s",
				"OPERATION_TIMEOUT":    "2m",
				"STALLED_STREAM_GRACE": "10m",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.MaxRetries != 5 {
					t.Errorf("MaxRetries = %v, want 5", cfg.MaxRetries)
				}
				if cfg.MaxBackoff != time.Minute {
					t.Errorf("MaxBackoff = %v, want 1m", cfg.MaxBackoff)
				}
				if cfg.ConnectTimeout != 5*time.Second {
					t.Errorf("ConnectTimeout = %v, want 5s", cfg.ConnectTimeout)
				}
				if cfg.ReadTimeout != 45*time.Second {
					t.Errorf("ReadTimeout = %v, want 45s", cfg.ReadTimeout)
				}
				if cfg.OperationTimeout != 2*time.Minute {
					t.Errorf("OperationTimeout = %v, want 2m", cfg.OperationTimeout)
				}
				if cfg.StalledStreamGrace != 10*time.Minute {
					t.Errorf("StalledStreamGrace = %v, want 10m", cfg.StalledStreamGrace)
				}
			},
		},
		{
			name: "opt-out telemetry enabled",
			envVars: map[string]string{
				"OPT_OUT_TELEMETRY": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.OptOutTelemetry {
					t.Errorf("OptOutTelemetry = %v, want true", cfg.OptOutTelemetry)
				}
			},
		},
		{
			name: "profile ARN configuration",
			envVars: map[string]string{
				"PROFILE_ARN": "arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.ProfileARN != "arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123" {
					t.Errorf("ProfileARN = %v, want arn:aws:codewhisperer:us-east-1:123456789012:profile/abc123", cfg.ProfileARN)
				}
			},
		},
		{
			name: "boolean parsing variations",
			envVars: map[string]string{
				"AMAZON_Q_SIGV4":    "false",
				"OPT_OUT_TELEMETRY": "false",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.EnableSigV4 {
					t.Errorf("EnableSigV4 = %v, want false", cfg.EnableSigV4)
				}
				if cfg.OptOutTelemetry {
					t.Errorf("OptOutTelemetry = %v, want false", cfg.OptOutTelemetry)
				}
			},
		},
		{
			name: "invalid values fallback to defaults",
			envVars: map[string]string{
				"MAX_RETRIES":          "invalid",
				"MAX_BACKOFF":          "invalid",
				"CONNECT_TIMEOUT":      "invalid",
				"READ_TIMEOUT":         "invalid",
				"OPERATION_TIMEOUT":    "invalid",
				"STALLED_STREAM_GRACE": "invalid",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.MaxRetries != 3 {
					t.Errorf("MaxRetries = %v, want 3 (default)", cfg.MaxRetries)
				}
				if cfg.MaxBackoff != 30*time.Second {
					t.Errorf("MaxBackoff = %v, want 30s (default)", cfg.MaxBackoff)
				}
				if cfg.ConnectTimeout != 10*time.Second {
					t.Errorf("ConnectTimeout = %v, want 10s (default)", cfg.ConnectTimeout)
				}
				if cfg.ReadTimeout != 30*time.Second {
					t.Errorf("ReadTimeout = %v, want 30s (default)", cfg.ReadTimeout)
				}
				if cfg.OperationTimeout != 60*time.Second {
					t.Errorf("OperationTimeout = %v, want 60s (default)", cfg.OperationTimeout)
				}
				if cfg.StalledStreamGrace != 5*time.Minute {
					t.Errorf("StalledStreamGrace = %v, want 5m (default)", cfg.StalledStreamGrace)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer clearEnv()

			// Load configuration
			cfg := Load()

			// Validate configuration
			tt.validate(t, cfg)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "environment variable not set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
		{
			name:         "empty environment variable",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv(tt.key)

			// Set environment variable if provided
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		want         bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			want:         false,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			want:         true,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			want:         true,
		},
		{
			name:         "unset uses default",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv(tt.key)

			// Set environment variable if provided
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getBoolEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getBoolEnv(%q, %v) = %v, want %v", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "42",
			want:         42,
		},
		{
			name:         "zero value",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "0",
			want:         0,
		},
		{
			name:         "negative integer",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "-5",
			want:         -5,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "invalid",
			want:         10,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv(tt.key)

			// Set environment variable if provided
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getIntEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getIntEnv(%q, %v) = %v, want %v", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestGetDurationEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		want         time.Duration
	}{
		{
			name:         "valid duration with seconds",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "30s",
			want:         30 * time.Second,
		},
		{
			name:         "valid duration with minutes",
			key:          "TEST_DURATION",
			defaultValue: 1 * time.Minute,
			envValue:     "5m",
			want:         5 * time.Minute,
		},
		{
			name:         "valid duration with hours",
			key:          "TEST_DURATION",
			defaultValue: 1 * time.Hour,
			envValue:     "2h",
			want:         2 * time.Hour,
		},
		{
			name:         "invalid duration uses default",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "invalid",
			want:         10 * time.Second,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "",
			want:         10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv(tt.key)

			// Set environment variable if provided
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getDurationEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getDurationEnv(%q, %v) = %v, want %v", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}

// clearEnv clears all environment variables used by the config
func clearEnv() {
	envVars := []string{
		"PORT",
		"AWS_REGION", "AWS_PROFILE", "AMAZON_Q_SIGV4",
		"OIDC_START_URL", "OIDC_REGION",
		"PROFILE_ARN",
		"OPT_OUT_TELEMETRY",
		"MAX_RETRIES", "MAX_BACKOFF",
		"CONNECT_TIMEOUT", "READ_TIMEOUT", "OPERATION_TIMEOUT",
		"STALLED_STREAM_GRACE",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
