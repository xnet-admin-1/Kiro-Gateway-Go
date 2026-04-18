package auth

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

// CredentialVault securely stores Identity Center credentials in OS keychain
type CredentialVault struct {
	service string
}

// NewCredentialVault creates a new credential vault
func NewCredentialVault() *CredentialVault {
	return &CredentialVault{
		service: "kiro-gateway-identity-center",
	}
}

// NewCredentialVaultWithService creates a new credential vault with custom service name
func NewCredentialVaultWithService(service string) *CredentialVault {
	return &CredentialVault{
		service: service,
	}
}

// StoredCredentials represents stored Identity Center credentials
type StoredCredentials struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	MFASecret string `json:"mfa_secret,omitempty"`
	StartURL  string `json:"start_url"`
	Region    string `json:"region"`
}

// Store stores credentials securely in OS keychain
// On Windows: Uses Windows Credential Manager
// On macOS: Uses Keychain
// On Linux: Uses Secret Service API (gnome-keyring, kwallet)
func (v *CredentialVault) Store(creds *StoredCredentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := keyring.Set(v.service, "credentials", string(data)); err != nil {
		return fmt.Errorf("failed to store credentials in keychain: %w", err)
	}

	return nil
}

// Retrieve retrieves credentials from OS keychain
func (v *CredentialVault) Retrieve() (*StoredCredentials, error) {
	data, err := keyring.Get(v.service, "credentials")
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from keychain: %w", err)
	}

	var creds StoredCredentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// Delete removes credentials from OS keychain
func (v *CredentialVault) Delete() error {
	if err := keyring.Delete(v.service, "credentials"); err != nil {
		return fmt.Errorf("failed to delete credentials from keychain: %w", err)
	}

	return nil
}

// Exists checks if credentials exist in OS keychain
func (v *CredentialVault) Exists() bool {
	_, err := keyring.Get(v.service, "credentials")
	return err == nil
}

// Update updates existing credentials in OS keychain
func (v *CredentialVault) Update(creds *StoredCredentials) error {
	// Delete existing credentials
	_ = v.Delete() // Ignore error if credentials don't exist

	// Store new credentials
	return v.Store(creds)
}

// GetService returns the service name used for keychain storage
func (v *CredentialVault) GetService() string {
	return v.service
}
