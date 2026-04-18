package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	defaultGatewayURL = "http://localhost:8080"
	configFileName    = "totp-keys-config.json"
)

type Config struct {
	GatewayURL string `json:"gateway_url"`
	AdminKey   string `json:"admin_key"`
	APIKeys    []APIKeyInfo `json:"api_keys"`
}

type APIKeyInfo struct {
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	UserID      string    `json:"user_id"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

type TOTPResponse struct {
	Code      string    `json:"code"`
	ExpiresIn int       `json:"expires_in"`
	Timestamp time.Time `json:"timestamp"`
}

type CreateKeyRequest struct {
	Name        string   `json:"name"`
	UserID      string   `json:"user_id"`
	Permissions []string `json:"permissions"`
}

type CreateKeyResponse struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	UserID      string    `json:"user_id"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

type App struct {
	window       fyne.Window
	config       *Config
	configPath   string
	
	// UI elements
	gatewayURLEntry *widget.Entry
	adminKeyEntry   *widget.Entry
	totpCodeLabel   *widget.Label
	expiresLabel    *widget.Label
	apiKeysList     *widget.List
	statusLabel     *widget.Label
	
	// State
	currentTOTP     string
	totpExpiry      int
	refreshTicker   *time.Ticker
	stopRefresh     chan bool
}

func main() {
	a := app.NewWithID("com.kiro.totp-keys")
	w := a.NewWindow("Kiro Gateway TOTP Keys")
	w.Resize(fyne.NewSize(800, 600))
	
	application := &App{
		window:      w,
		stopRefresh: make(chan bool),
	}
	
	application.loadConfig()
	application.setupUI()
	
	w.ShowAndRun()
}

func (a *App) loadConfig() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		exePath = "."
	}
	exeDir := filepath.Dir(exePath)
	
	// Config goes in x-ai/.data directory
	configDir := filepath.Join(exeDir, ".data")
	a.configPath = filepath.Join(configDir, configFileName)
	
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		a.config = &Config{
			GatewayURL: defaultGatewayURL,
			APIKeys:    []APIKeyInfo{},
		}
		return
	}
	
	a.config = &Config{}
	if err := json.Unmarshal(data, a.config); err != nil {
		a.config = &Config{
			GatewayURL: defaultGatewayURL,
			APIKeys:    []APIKeyInfo{},
		}
	}
}

func (a *App) saveConfig() error {
	dir := filepath.Dir(a.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(a.configPath, data, 0600)
}

func (a *App) setupUI() {
	// Configuration section
	a.gatewayURLEntry = widget.NewEntry()
	a.gatewayURLEntry.SetPlaceHolder("http://localhost:8080")
	a.gatewayURLEntry.SetText(a.config.GatewayURL)
	
	a.adminKeyEntry = widget.NewPasswordEntry()
	a.adminKeyEntry.SetPlaceHolder("Admin API Key")
	a.adminKeyEntry.SetText(a.config.AdminKey)
	
	saveConfigBtn := widget.NewButton("Save Configuration", func() {
		a.config.GatewayURL = a.gatewayURLEntry.Text
		a.config.AdminKey = a.adminKeyEntry.Text
		
		if err := a.saveConfig(); err != nil {
			dialog.ShowError(err, a.window)
		} else {
			a.setStatus("Configuration saved", false)
		}
	})
	
	configForm := container.NewVBox(
		widget.NewLabel("Gateway Configuration"),
		widget.NewForm(
			widget.NewFormItem("Gateway URL", a.gatewayURLEntry),
			widget.NewFormItem("Admin API Key", a.adminKeyEntry),
		),
		saveConfigBtn,
	)
	
	// TOTP section
	a.totpCodeLabel = widget.NewLabel("------")
	a.totpCodeLabel.TextStyle = fyne.TextStyle{Bold: true}
	a.totpCodeLabel.Alignment = fyne.TextAlignCenter
	
	a.expiresLabel = widget.NewLabel("Not retrieved")
	a.expiresLabel.Alignment = fyne.TextAlignCenter
	
	getTOTPBtn := widget.NewButtonWithIcon("Get TOTP Code", theme.DownloadIcon(), func() {
		a.getTOTPCode()
	})
	
	copyTOTPBtn := widget.NewButtonWithIcon("Copy to Clipboard", theme.ContentCopyIcon(), func() {
		if a.currentTOTP != "" {
			a.window.Clipboard().SetContent(a.currentTOTP)
			a.setStatus("TOTP code copied to clipboard", false)
		}
	})
	
	autoRefreshCheck := widget.NewCheck("Auto-refresh", func(checked bool) {
		if checked {
			a.startAutoRefresh()
		} else {
			a.stopAutoRefresh()
		}
	})
	
	totpBox := container.NewVBox(
		widget.NewLabel("TOTP Code"),
		container.NewCenter(a.totpCodeLabel),
		container.NewCenter(a.expiresLabel),
		container.NewHBox(getTOTPBtn, copyTOTPBtn, layout.NewSpacer(), autoRefreshCheck),
	)
	
	// API Keys section
	a.apiKeysList = widget.NewList(
		func() int {
			return len(a.config.APIKeys)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Key Name"),
				layout.NewSpacer(),
				widget.NewButton("Copy", nil),
				widget.NewButton("Delete", nil),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			keyInfo := a.config.APIKeys[id]
			box := item.(*fyne.Container)
			
			label := box.Objects[0].(*widget.Label)
			label.SetText(fmt.Sprintf("%s (%s)", keyInfo.Name, keyInfo.UserID))
			
			copyBtn := box.Objects[2].(*widget.Button)
			copyBtn.OnTapped = func() {
				a.window.Clipboard().SetContent(keyInfo.Key)
				a.setStatus(fmt.Sprintf("API key '%s' copied to clipboard", keyInfo.Name), false)
			}
			
			deleteBtn := box.Objects[3].(*widget.Button)
			deleteBtn.OnTapped = func() {
				a.deleteAPIKey(id)
			}
		},
	)
	
	createKeyBtn := widget.NewButtonWithIcon("Create New API Key", theme.ContentAddIcon(), func() {
		a.showCreateKeyDialog()
	})
	
	apiKeysBox := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("API Keys"),
			createKeyBtn,
		),
		nil, nil, nil,
		a.apiKeysList,
	)
	
	// Status bar
	a.statusLabel = widget.NewLabel("")
	statusBar := container.NewHBox(a.statusLabel)
	
	// Main layout
	tabs := container.NewAppTabs(
		container.NewTabItem("TOTP", totpBox),
		container.NewTabItem("API Keys", apiKeysBox),
		container.NewTabItem("Configuration", configForm),
	)
	
	content := container.NewBorder(
		nil,
		statusBar,
		nil, nil,
		tabs,
	)
	
	a.window.SetContent(content)
}


func (a *App) getTOTPCode() {
	if a.config.GatewayURL == "" {
		a.setStatus("Gateway URL not configured", true)
		return
	}
	
	// Try with admin key first, then try stored API keys
	keys := []string{a.config.AdminKey}
	for _, keyInfo := range a.config.APIKeys {
		keys = append(keys, keyInfo.Key)
	}
	
	var lastErr error
	for _, key := range keys {
		if key == "" {
			continue
		}
		
		req, err := http.NewRequest("GET", a.config.GatewayURL+"/totp", nil)
		if err != nil {
			lastErr = err
			continue
		}
		
		req.Header.Set("Authorization", "Bearer "+key)
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			var totpResp TOTPResponse
			if err := json.NewDecoder(resp.Body).Decode(&totpResp); err != nil {
				lastErr = err
				continue
			}
			
			a.currentTOTP = totpResp.Code
			a.totpExpiry = totpResp.ExpiresIn
			a.totpCodeLabel.SetText(totpResp.Code)
			a.expiresLabel.SetText(fmt.Sprintf("Expires in %d seconds", totpResp.ExpiresIn))
			a.setStatus("TOTP code retrieved successfully", false)
			return
		}
		
		if resp.StatusCode == http.StatusUnauthorized {
			lastErr = fmt.Errorf("authentication failed")
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	if lastErr != nil {
		a.setStatus(fmt.Sprintf("Failed to get TOTP: %v", lastErr), true)
		dialog.ShowError(lastErr, a.window)
	} else {
		a.setStatus("No valid API key configured", true)
		dialog.ShowError(fmt.Errorf("no valid API key configured"), a.window)
	}
}

func (a *App) startAutoRefresh() {
	a.stopAutoRefresh()
	
	a.refreshTicker = time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-a.refreshTicker.C:
				a.getTOTPCode()
			case <-a.stopRefresh:
				return
			}
		}
	}()
}

func (a *App) stopAutoRefresh() {
	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
		a.stopRefresh <- true
	}
}

func (a *App) showCreateKeyDialog() {
	if a.config.AdminKey == "" {
		dialog.ShowError(fmt.Errorf("admin API key not configured"), a.window)
		return
	}
	
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Key name (e.g., 'totp-access')")
	
	userIDEntry := widget.NewEntry()
	userIDEntry.SetPlaceHolder("User ID (e.g., 'totp-user')")
	
	readCheck := widget.NewCheck("Read", nil)
	readCheck.SetChecked(true)
	writeCheck := widget.NewCheck("Write", nil)
	adminCheck := widget.NewCheck("Admin", nil)
	
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: nameEntry},
			{Text: "User ID", Widget: userIDEntry},
			{Text: "Permissions", Widget: container.NewVBox(readCheck, writeCheck, adminCheck)},
		},
	}
	
	d := dialog.NewCustomConfirm("Create API Key", "Create", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		
		if nameEntry.Text == "" || userIDEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("name and user ID are required"), a.window)
			return
		}
		
		var permissions []string
		if readCheck.Checked {
			permissions = append(permissions, "read")
		}
		if writeCheck.Checked {
			permissions = append(permissions, "write")
		}
		if adminCheck.Checked {
			permissions = append(permissions, "admin")
		}
		
		if len(permissions) == 0 {
			dialog.ShowError(fmt.Errorf("at least one permission is required"), a.window)
			return
		}
		
		a.createAPIKey(nameEntry.Text, userIDEntry.Text, permissions)
	}, a.window)
	
	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}

func (a *App) createAPIKey(name, userID string, permissions []string) {
	reqBody := CreateKeyRequest{
		Name:        name,
		UserID:      userID,
		Permissions: permissions,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	
	req, err := http.NewRequest("POST", a.config.GatewayURL+"/v1/api-keys", bytes.NewBuffer(jsonData))
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	
	req.Header.Set("Authorization", "Bearer "+a.config.AdminKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		dialog.ShowError(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)), a.window)
		return
	}
	
	var createResp CreateKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	
	// Add to config
	keyInfo := APIKeyInfo{
		Name:        createResp.Name,
		Key:         createResp.Key,
		UserID:      createResp.UserID,
		Permissions: createResp.Permissions,
		CreatedAt:   createResp.CreatedAt,
	}
	
	a.config.APIKeys = append(a.config.APIKeys, keyInfo)
	
	if err := a.saveConfig(); err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	
	a.apiKeysList.Refresh()
	a.setStatus(fmt.Sprintf("API key '%s' created successfully", name), false)
	
	// Show the key in a dialog
	keyLabel := widget.NewLabel(createResp.Key)
	keyLabel.Wrapping = fyne.TextWrapWord
	
	copyBtn := widget.NewButton("Copy to Clipboard", func() {
		a.window.Clipboard().SetContent(createResp.Key)
		a.setStatus("API key copied to clipboard", false)
	})
	
	content := container.NewVBox(
		widget.NewLabel("API Key Created Successfully"),
		widget.NewLabel("Save this key - it won't be shown again!"),
		widget.NewSeparator(),
		keyLabel,
		copyBtn,
	)
	
	dialog.ShowCustom("New API Key", "Close", content, a.window)
}

func (a *App) deleteAPIKey(index int) {
	if index < 0 || index >= len(a.config.APIKeys) {
		return
	}
	
	keyInfo := a.config.APIKeys[index]
	
	dialog.ShowConfirm("Delete API Key", 
		fmt.Sprintf("Are you sure you want to delete '%s'?", keyInfo.Name),
		func(ok bool) {
			if !ok {
				return
			}
			
			a.config.APIKeys = append(a.config.APIKeys[:index], a.config.APIKeys[index+1:]...)
			
			if err := a.saveConfig(); err != nil {
				dialog.ShowError(err, a.window)
				return
			}
			
			a.apiKeysList.Refresh()
			a.setStatus(fmt.Sprintf("API key '%s' deleted", keyInfo.Name), false)
		}, a.window)
}

func (a *App) setStatus(message string, isError bool) {
	if isError {
		a.statusLabel.SetText("[ERROR] " + message)
	} else {
		a.statusLabel.SetText("[SUCCESS] " + message)
	}
	
	// Clear status after 5 seconds
	go func() {
		time.Sleep(5 * time.Second)
		a.statusLabel.SetText("")
	}()
}
