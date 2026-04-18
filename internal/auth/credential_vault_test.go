package auth

import (
	"testing"
)

func TestCredentialVault_StoreAndRetrieve(t *testing.T) {
	// Use a test-specific service name to avoid conflicts
	vault := NewCredentialVaultWithService("kiro-gateway-test")
	
	// Clean up before and after test
	defer vault.Delete()
	vault.Delete() // Clean up any existing test data
	
	// Create test credentials
	creds := &StoredCredentials{
		Username:  "test@example.com",
		Password:  "test-password",
		MFASecret: "JBSWY3DPEHPK3PXP",
		StartURL:  "https://test.awsapps.com/start",
		Region:    "us-east-1",
	}
	
	// Store credentials
	if err := vault.Store(creds); err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}
	
	// Retrieve credentials
	retrieved, err := vault.Retrieve()
	if err != nil {
		t.Fatalf("Failed to retrieve credentials: %v", err)
	}
	
	// Verify credentials
	if retrieved.Username != creds.Username {
		t.Errorf("Username mismatch: got %s, want %s", retrieved.Username, creds.Username)
	}
	
	if retrieved.Password != creds.Password {
		t.Errorf("Password mismatch: got %s, want %s", retrieved.Password, creds.Password)
	}
	
	if retrieved.MFASecret != creds.MFASecret {
		t.Errorf("MFASecret mismatch: got %s, want %s", retrieved.MFASecret, creds.MFASecret)
	}
	
	if retrieved.StartURL != creds.StartURL {
		t.Errorf("StartURL mismatch: got %s, want %s", retrieved.StartURL, creds.StartURL)
	}
	
	if retrieved.Region != creds.Region {
		t.Errorf("Region mismatch: got %s, want %s", retrieved.Region, creds.Region)
	}
}

func TestCredentialVault_Delete(t *testing.T) {
	vault := NewCredentialVaultWithService("kiro-gateway-test-delete")
	
	// Clean up before test
	vault.Delete()
	
	// Store credentials
	creds := &StoredCredentials{
		Username: "test@example.com",
		Password: "test-password",
		StartURL: "https://test.awsapps.com/start",
		Region:   "us-east-1",
	}
	
	if err := vault.Store(creds); err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}
	
	// Verify credentials exist
	if !vault.Exists() {
		t.Error("Credentials should exist after storing")
	}
	
	// Delete credentials
	if err := vault.Delete(); err != nil {
		t.Fatalf("Failed to delete credentials: %v", err)
	}
	
	// Verify credentials don't exist
	if vault.Exists() {
		t.Error("Credentials should not exist after deletion")
	}
	
	// Verify retrieval fails
	_, err := vault.Retrieve()
	if err == nil {
		t.Error("Expected error when retrieving deleted credentials")
	}
}

func TestCredentialVault_Exists(t *testing.T) {
	vault := NewCredentialVaultWithService("kiro-gateway-test-exists")
	
	// Clean up before test
	vault.Delete()
	
	// Should not exist initially
	if vault.Exists() {
		t.Error("Credentials should not exist initially")
	}
	
	// Store credentials
	creds := &StoredCredentials{
		Username: "test@example.com",
		Password: "test-password",
		StartURL: "https://test.awsapps.com/start",
		Region:   "us-east-1",
	}
	
	if err := vault.Store(creds); err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}
	
	// Should exist after storing
	if !vault.Exists() {
		t.Error("Credentials should exist after storing")
	}
	
	// Clean up
	vault.Delete()
}

func TestCredentialVault_Update(t *testing.T) {
	vault := NewCredentialVaultWithService("kiro-gateway-test-update")
	
	// Clean up before and after test
	defer vault.Delete()
	vault.Delete()
	
	// Store initial credentials
	creds1 := &StoredCredentials{
		Username: "test1@example.com",
		Password: "password1",
		StartURL: "https://test1.awsapps.com/start",
		Region:   "us-east-1",
	}
	
	if err := vault.Store(creds1); err != nil {
		t.Fatalf("Failed to store initial credentials: %v", err)
	}
	
	// Update credentials
	creds2 := &StoredCredentials{
		Username:  "test2@example.com",
		Password:  "password2",
		MFASecret: "NEWSECRET",
		StartURL:  "https://test2.awsapps.com/start",
		Region:    "us-west-2",
	}
	
	if err := vault.Update(creds2); err != nil {
		t.Fatalf("Failed to update credentials: %v", err)
	}
	
	// Retrieve and verify updated credentials
	retrieved, err := vault.Retrieve()
	if err != nil {
		t.Fatalf("Failed to retrieve updated credentials: %v", err)
	}
	
	if retrieved.Username != creds2.Username {
		t.Errorf("Username not updated: got %s, want %s", retrieved.Username, creds2.Username)
	}
	
	if retrieved.Password != creds2.Password {
		t.Errorf("Password not updated: got %s, want %s", retrieved.Password, creds2.Password)
	}
	
	if retrieved.MFASecret != creds2.MFASecret {
		t.Errorf("MFASecret not updated: got %s, want %s", retrieved.MFASecret, creds2.MFASecret)
	}
	
	if retrieved.StartURL != creds2.StartURL {
		t.Errorf("StartURL not updated: got %s, want %s", retrieved.StartURL, creds2.StartURL)
	}
	
	if retrieved.Region != creds2.Region {
		t.Errorf("Region not updated: got %s, want %s", retrieved.Region, creds2.Region)
	}
}

func TestCredentialVault_GetService(t *testing.T) {
	serviceName := "test-service"
	vault := NewCredentialVaultWithService(serviceName)
	
	if vault.GetService() != serviceName {
		t.Errorf("GetService() = %s, want %s", vault.GetService(), serviceName)
	}
}

func TestNewCredentialVault(t *testing.T) {
	vault := NewCredentialVault()
	
	expectedService := "kiro-gateway-identity-center"
	if vault.GetService() != expectedService {
		t.Errorf("Default service name = %s, want %s", vault.GetService(), expectedService)
	}
}

func TestStoredCredentials_WithoutMFA(t *testing.T) {
	vault := NewCredentialVaultWithService("kiro-gateway-test-no-mfa")
	
	// Clean up before and after test
	defer vault.Delete()
	vault.Delete()
	
	// Create credentials without MFA
	creds := &StoredCredentials{
		Username: "test@example.com",
		Password: "test-password",
		StartURL: "https://test.awsapps.com/start",
		Region:   "us-east-1",
		// MFASecret intentionally omitted
	}
	
	// Store credentials
	if err := vault.Store(creds); err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}
	
	// Retrieve credentials
	retrieved, err := vault.Retrieve()
	if err != nil {
		t.Fatalf("Failed to retrieve credentials: %v", err)
	}
	
	// Verify MFA secret is empty
	if retrieved.MFASecret != "" {
		t.Errorf("MFASecret should be empty, got %s", retrieved.MFASecret)
	}
}
