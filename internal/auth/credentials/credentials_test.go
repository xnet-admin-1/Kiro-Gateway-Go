package credentials

import (
	"testing"
	"time"
)

func TestCredentials_String(t *testing.T) {
	creds := &Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "session-token-123",
		Source:          "test",
	}

	str := creds.String()
	
	// Should not contain actual credentials
	if containsCredential(str, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("String() should not expose AccessKeyID")
	}
	if containsCredential(str, "wJalrXUtnFEMI") {
		t.Error("String() should not expose SecretAccessKey")
	}
	if containsCredential(str, "session-token-123") {
		t.Error("String() should not expose SessionToken")
	}
	
	// Should contain source
	if !containsCredential(str, "test") {
		t.Error("String() should contain source")
	}
}

func TestCredentials_Copy(t *testing.T) {
	original := &Credentials{
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		SessionToken:    "test-token",
		Source:          "test",
		CanExpire:       true,
		Expires:         time.Now().Add(time.Hour),
	}

	copy := original.Copy()

	// Verify all fields are copied
	if copy.AccessKeyID != original.AccessKeyID {
		t.Errorf("Copy AccessKeyID = %v, want %v", copy.AccessKeyID, original.AccessKeyID)
	}
	if copy.SecretAccessKey != original.SecretAccessKey {
		t.Errorf("Copy SecretAccessKey = %v, want %v", copy.SecretAccessKey, original.SecretAccessKey)
	}
	if copy.SessionToken != original.SessionToken {
		t.Errorf("Copy SessionToken = %v, want %v", copy.SessionToken, original.SessionToken)
	}
	if copy.Source != original.Source {
		t.Errorf("Copy Source = %v, want %v", copy.Source, original.Source)
	}
	if copy.CanExpire != original.CanExpire {
		t.Errorf("Copy CanExpire = %v, want %v", copy.CanExpire, original.CanExpire)
	}
	if !copy.Expires.Equal(original.Expires) {
		t.Errorf("Copy Expires = %v, want %v", copy.Expires, original.Expires)
	}

	// Verify it's a different instance
	if copy == original {
		t.Error("Copy should return a different instance")
	}

	// Verify modifying copy doesn't affect original
	copy.AccessKeyID = "modified"
	if original.AccessKeyID == "modified" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		creds     *Credentials
		wantExpired bool
	}{
		{
			name: "non-expiring credentials",
			creds: &Credentials{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Source:          "test",
				CanExpire:       false,
			},
			wantExpired: false,
		},
		{
			name: "future expiration",
			creds: &Credentials{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Source:          "test",
				CanExpire:       true,
				Expires:         time.Now().Add(time.Hour),
			},
			wantExpired: false,
		},
		{
			name: "past expiration",
			creds: &Credentials{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Source:          "test",
				CanExpire:       true,
				Expires:         time.Now().Add(-time.Hour),
			},
			wantExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.creds.IsExpired(); got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestNewCredentials(t *testing.T) {
	accessKey := "test-key"
	secretKey := "test-secret"
	source := "test"

	creds := NewCredentials(accessKey, secretKey, "", source)

	if creds.AccessKeyID != accessKey {
		t.Errorf("AccessKeyID = %v, want %v", creds.AccessKeyID, accessKey)
	}
	if creds.SecretAccessKey != secretKey {
		t.Errorf("SecretAccessKey = %v, want %v", creds.SecretAccessKey, secretKey)
	}
	if creds.Source != source {
		t.Errorf("Source = %v, want %v", creds.Source, source)
	}
	if creds.CanExpire {
		t.Error("CanExpire should be false for static credentials")
	}
}

func TestNewTemporaryCredentials(t *testing.T) {
	accessKey := "test-key"
	secretKey := "test-secret"
	sessionToken := "test-token"
	source := "test"
	expires := time.Now().Add(time.Hour)

	creds := NewTemporaryCredentials(accessKey, secretKey, sessionToken, source, expires)

	if creds.AccessKeyID != accessKey {
		t.Errorf("AccessKeyID = %v, want %v", creds.AccessKeyID, accessKey)
	}
	if creds.SecretAccessKey != secretKey {
		t.Errorf("SecretAccessKey = %v, want %v", creds.SecretAccessKey, secretKey)
	}
	if creds.SessionToken != sessionToken {
		t.Errorf("SessionToken = %v, want %v", creds.SessionToken, sessionToken)
	}
	if creds.Source != source {
		t.Errorf("Source = %v, want %v", creds.Source, source)
	}
	if !creds.CanExpire {
		t.Error("CanExpire should be true for temporary credentials")
	}
	if !creds.Expires.Equal(expires) {
		t.Errorf("Expires = %v, want %v", creds.Expires, expires)
	}
}

// Helper function to check if string contains credential (case-insensitive)
func containsCredential(s, credential string) bool {
	return len(s) >= len(credential) && findSubstring(s, credential)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
