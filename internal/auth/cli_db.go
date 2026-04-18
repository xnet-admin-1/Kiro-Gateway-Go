package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// CLI DB auth implementation

type cliDBAuthData struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AWS SSO cache JSON format
type ssoTokenCache struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

func (a *AuthManager) loadCLIDBToken(ctx context.Context) error {
	dbPath := a.cliDBPath
	if dbPath == "" {
		// Default path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".aws", "amazonq", "cache.db")
	}
	
	// Expand ~ if present
	if strings.HasPrefix(dbPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[1:])
	}
	
	// Check if path is a JSON file (AWS SSO cache format)
	if strings.HasSuffix(dbPath, ".json") {
		return a.loadSSOTokenFromJSON(dbPath)
	}
	
	// Otherwise treat as SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open CLI database: %w", err)
	}
	defer db.Close()
	
	var authDataJSON string
	err = db.QueryRowContext(ctx, "SELECT value FROM cache WHERE key = 'auth_token'").Scan(&authDataJSON)
	if err != nil {
		return fmt.Errorf("failed to read auth token from database: %w", err)
	}
	
	var authData cliDBAuthData
	if err := json.Unmarshal([]byte(authDataJSON), &authData); err != nil {
		return fmt.Errorf("failed to parse auth data: %w", err)
	}
	
	a.token = authData.Token
	a.tokenExp = authData.ExpiresAt
	
	return nil
}

// loadSSOTokenFromJSON loads bearer token from AWS SSO cache JSON file
func (a *AuthManager) loadSSOTokenFromJSON(jsonPath string) error {
	log.Printf("[DEBUG] Loading SSO token from: %s", jsonPath)
	
	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read SSO token file: %w", err)
	}
	
	// Parse JSON
	var ssoToken ssoTokenCache
	if err := json.Unmarshal(data, &ssoToken); err != nil {
		return fmt.Errorf("failed to parse SSO token JSON: %w", err)
	}
	
	// Parse expiration time
	expiresAt, err := time.Parse(time.RFC3339, ssoToken.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to parse expiration time: %w", err)
	}
	
	log.Printf("[DEBUG] SSO token loaded - expires at: %s (in %v)", expiresAt.Format(time.RFC3339), time.Until(expiresAt))
	
	// Set token and expiration
	a.token = ssoToken.AccessToken
	a.tokenExp = expiresAt
	
	return nil
}

func (a *AuthManager) refreshCLIDBToken(ctx context.Context) error {
	log.Printf("[DEBUG] Refreshing CLI DB token by reloading from file")
	
	// For CLI DB auth, we just reload from database/file
	// The CLI handles the actual refresh
	err := a.loadCLIDBToken(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to reload CLI DB token: %v", err)
		return err
	}
	
	log.Printf("[DEBUG] CLI DB token reloaded successfully")
	return nil
}
