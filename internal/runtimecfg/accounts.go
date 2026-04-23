package runtimecfg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Account represents a stored auth credential.
type Account struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	AuthType     string    `json:"auth_type"` // "sso_token", "refresh_token", "cli_db"
	Region       string    `json:"region,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
	ClientSecret string    `json:"client_secret,omitempty"`
	ExpiresAt    string    `json:"expires_at,omitempty"`
	ImportedAt   time.Time `json:"imported_at"`
	Active       bool      `json:"active"`
}

// AccountsHandler serves /api/accounts endpoints.
type AccountsHandler struct {
	store *Store
}

// NewAccountsHandler creates an accounts handler backed by the runtime config store.
func NewAccountsHandler(store *Store) *AccountsHandler {
	return &AccountsHandler{store: store}
}

// RegisterRoutes registers /api/accounts routes.
func (h *AccountsHandler) RegisterRoutes(mux *http.ServeMux, wrap func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/admin/api/accounts", wrap(h.handleList))
	mux.HandleFunc("/admin/api/accounts/import", wrap(h.handleImport))
	mux.HandleFunc("/admin/api/accounts/export", wrap(h.handleExport))
	mux.HandleFunc("/admin/api/accounts/import/kiro-cli", wrap(h.handleImportKiroCLI))
	mux.HandleFunc("/admin/api/accounts/import/sso-token", wrap(h.handleImportSSOToken))
}

func (h *AccountsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	accounts := h.getAccounts()
	writeJSON(w, http.StatusOK, accounts)
}

func (h *AccountsHandler) handleExport(w http.ResponseWriter, r *http.Request) {
	accounts := h.getAccounts()
	w.Header().Set("Content-Disposition", "attachment; filename=kiro-accounts.json")
	writeJSON(w, http.StatusOK, accounts)
}

func (h *AccountsHandler) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var accounts []Account
	if err := json.NewDecoder(r.Body).Decode(&accounts); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON array of accounts"})
		return
	}
	for i := range accounts {
		accounts[i].ImportedAt = time.Now()
		if accounts[i].ID == "" {
			accounts[i].ID = fmt.Sprintf("acc_%d", time.Now().UnixNano()+int64(i))
		}
		h.store.Set("account:"+accounts[i].ID, accounts[i])
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"imported": len(accounts)})
}

// handleImportSSOToken imports a raw SSO/refresh token JSON.
// Body: {"access_token":"...", "refresh_token":"...", "client_id":"...", "client_secret":"...", "expires_at":"..."}
func (h *AccountsHandler) handleImportSSOToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name         string `json:"name"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		ExpiresAt    string `json:"expires_at"`
		Region       string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "access_token required"})
		return
	}
	acc := Account{
		ID:           fmt.Sprintf("acc_%d", time.Now().UnixNano()),
		Name:         body.Name,
		AuthType:     "sso_token",
		Region:       body.Region,
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		ClientID:     body.ClientID,
		ClientSecret: body.ClientSecret,
		ExpiresAt:    body.ExpiresAt,
		ImportedAt:   time.Now(),
		Active:       true,
	}
	if acc.Name == "" {
		acc.Name = "SSO Import " + time.Now().Format("2006-01-02 15:04")
	}
	h.store.Set("account:"+acc.ID, acc)
	writeJSON(w, http.StatusOK, acc)
}

// handleImportKiroCLI scans the local Kiro CLI DB or SSO cache and imports tokens.
// Body (optional): {"db_path": "/path/to/cache.db"} or {"sso_cache_dir": "~/.aws/sso/cache"}
func (h *AccountsHandler) handleImportKiroCLI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DBPath      string `json:"db_path"`
		SSOCacheDir string `json:"sso_cache_dir"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Default: scan ~/.aws/sso/cache/ for JSON token files
	scanDir := body.SSOCacheDir
	if scanDir == "" {
		home, _ := os.UserHomeDir()
		scanDir = filepath.Join(home, ".aws", "sso", "cache")
	}
	scanDir = expandHome(scanDir)

	var imported []Account
	entries, err := os.ReadDir(scanDir)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"imported": 0, "error": "cannot read " + scanDir})
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(scanDir, e.Name()))
		if err != nil {
			continue
		}
		var tok struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresAt    string `json:"expiresAt"`
			ClientID     string `json:"clientId"`
			ClientSecret string `json:"clientSecret"`
		}
		if json.Unmarshal(data, &tok) != nil || tok.AccessToken == "" {
			continue
		}
		acc := Account{
			ID:           fmt.Sprintf("acc_%d", time.Now().UnixNano()),
			Name:         "Kiro CLI: " + e.Name(),
			AuthType:     "cli_db",
			AccessToken:  tok.AccessToken,
			RefreshToken: tok.RefreshToken,
			ClientID:     tok.ClientID,
			ClientSecret: tok.ClientSecret,
			ExpiresAt:    tok.ExpiresAt,
			ImportedAt:   time.Now(),
			Active:       true,
		}
		h.store.Set("account:"+acc.ID, acc)
		imported = append(imported, acc)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"imported": len(imported), "accounts": imported})
}

func (h *AccountsHandler) getAccounts() []Account {
	all := h.store.List()
	var accounts []Account
	for k, e := range all {
		if !strings.HasPrefix(k, "account:") {
			continue
		}
		raw, _ := json.Marshal(e.Value)
		var acc Account
		if json.Unmarshal(raw, &acc) == nil {
			accounts = append(accounts, acc)
		}
	}
	return accounts
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}
