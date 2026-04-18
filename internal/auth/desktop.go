package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Desktop auth implementation

type desktopAuthData struct {
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expires_at"`
	ProfileArn string    `json:"profile_arn"`
}

func (a *AuthManager) loadDesktopToken(ctx context.Context) error {
	dbPath := a.kiroDBPath
	if dbPath == "" {
		// Default path
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".kiro", "kiro.db")
	}
	
	// Expand ~ if present
	if strings.HasPrefix(dbPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[1:])
	}
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open Kiro Desktop database: %w", err)
	}
	defer db.Close()
	
	var authDataJSON string
	err = db.QueryRowContext(ctx, "SELECT auth_data FROM auth WHERE id = 1").Scan(&authDataJSON)
	if err != nil {
		return fmt.Errorf("failed to read auth data from database: %w", err)
	}
	
	var authData desktopAuthData
	if err := json.Unmarshal([]byte(authDataJSON), &authData); err != nil {
		return fmt.Errorf("failed to parse auth data: %w", err)
	}
	
	a.token = authData.Token
	a.tokenExp = authData.ExpiresAt
	
	// Only set profile ARN from database if not already set from environment
	if a.profileArn == "" {
		a.profileArn = authData.ProfileArn
	}
	
	return nil
}

func (a *AuthManager) refreshDesktopToken(ctx context.Context) error {
	// For desktop auth, we just reload from database
	// The Kiro Desktop app handles the actual refresh
	return a.loadDesktopToken(ctx)
}
