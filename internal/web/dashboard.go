package web

import (
	"context"
	"crypto/tls"
	"embed"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/apikeys"
	"github.com/yourusername/kiro-gateway-go/internal/auth"
	"github.com/yourusername/kiro-gateway-go/internal/config"
)

//go:embed static/*
var staticFiles embed.FS

type StatsProvider interface {
	GetMetrics() map[string]interface{}
}

// LogEntry is a captured log line
type LogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

// RequestRecord tracks an API request
type RequestRecord struct {
	Time      string `json:"time"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Model     string `json:"model"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	UserID    string `json:"user_id"`
}

// AuditEntry records key management actions
type AuditEntry struct {
	Time    string `json:"time"`
	Action  string `json:"action"`
	KeyName string `json:"key_name"`
	KeyID   string `json:"key_id"`
	Actor   string `json:"actor"`
	Detail  string `json:"detail,omitempty"`
}

// LogCapture captures log output into a ring buffer and broadcasts via SSE
type LogCapture struct {
	mu      sync.Mutex
	buf     []LogEntry
	maxSize int
	clients map[chan LogEntry]struct{}
}

func newLogCapture(size int) *LogCapture {
	return &LogCapture{buf: make([]LogEntry, 0, size), maxSize: size, clients: make(map[chan LogEntry]struct{})}
}

func (lc *LogCapture) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}
	entry := LogEntry{Time: time.Now().Format("15:04:05"), Message: msg}
	lc.mu.Lock()
	if len(lc.buf) >= lc.maxSize {
		lc.buf = lc.buf[1:]
	}
	lc.buf = append(lc.buf, entry)
	// Broadcast to SSE clients
	for ch := range lc.clients {
		select {
		case ch <- entry:
		default:
		}
	}
	lc.mu.Unlock()
	return len(p), nil
}

func (lc *LogCapture) Recent(n int) []LogEntry {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	start := len(lc.buf) - n
	if start < 0 {
		start = 0
	}
	out := make([]LogEntry, len(lc.buf[start:]))
	copy(out, lc.buf[start:])
	return out
}

func (lc *LogCapture) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 32)
	lc.mu.Lock()
	lc.clients[ch] = struct{}{}
	lc.mu.Unlock()
	return ch
}

func (lc *LogCapture) Unsubscribe(ch chan LogEntry) {
	lc.mu.Lock()
	delete(lc.clients, ch)
	lc.mu.Unlock()
}

// Dashboard serves the admin web UI
// RuntimeCfgStore provides access to runtime config values.
type RuntimeCfgStore interface {
	GetString(key string) string
}

type Dashboard struct {
	authManager *auth.AuthManager
	config      *config.Config
	stats       StatsProvider
	runtimeCfg  RuntimeCfgStore
	startTime   time.Time
	mu          sync.RWMutex
	logCapture  *LogCapture
	sseClients  map[chan []byte]struct{}
	sseMu       sync.Mutex
	ssoClients  map[chan []byte]struct{}
	ssoMu       sync.Mutex
	requests    []RequestRecord
	reqMu       sync.Mutex
	maxRequests    int
	sessionToken   string
	keyManager     *apikeys.PersistentAPIKeyManager
	auditLog       []AuditEntry
	auditMu        sync.Mutex
}

func NewDashboard(am *auth.AuthManager, cfg *config.Config, stats StatsProvider, km *apikeys.PersistentAPIKeyManager, rtCfg RuntimeCfgStore) *Dashboard {
	lc := newLogCapture(500)
	// Tee log output to both stderr and our capture buffer
	log.SetOutput(io.MultiWriter(os.Stderr, lc))

	tok := make([]byte, 32)
	rand.Read(tok)
	sessionToken := hex.EncodeToString(tok)

	d := &Dashboard{
		authManager: am,
		config:      cfg,
		stats:       stats,
		runtimeCfg:  rtCfg,
		startTime:   time.Now(),
		logCapture:  lc,
		sseClients:  make(map[chan []byte]struct{}),
		ssoClients:  make(map[chan []byte]struct{}),
		maxRequests:  200,
		keyManager:   km,
		sessionToken: sessionToken,
	}
	go d.broadcastLoop()

	// Wire up device flow callback to broadcast SSO updates
	if ha := am.GetHeadlessAuth(); ha != nil {
		ha.SetDeviceFlowCallback(func() {
			state := ha.DeviceFlowState()
			state["authenticated"] = ha.IsAuthenticated()
			d.BroadcastSSOUpdate(state)
			if ha.IsAuthenticated() {
				d.addAudit("login", "device-flow", "", "Authentication successful")
			}
		})
	}

	return d
}

// RecordRequest records an API request for the history view
func (d *Dashboard) RecordRequest(method, path, model, userID string, status int, latency time.Duration) {
	d.reqMu.Lock()
	defer d.reqMu.Unlock()
	rec := RequestRecord{
		Time:      time.Now().Format("15:04:05"),
		Method:    method,
		Path:      path,
		Model:     model,
		Status:    status,
		LatencyMs: latency.Milliseconds(),
		UserID:    userID,
	}
	if len(d.requests) >= d.maxRequests {
		d.requests = d.requests[1:]
	}
	d.requests = append(d.requests, rec)
}

// Middleware wraps an http.Handler to record API requests for the dashboard
func (d *Dashboard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only track API endpoints, not admin/static
		p := r.URL.Path
		if p == "/v1/chat/completions" || p == "/v1/models" || p == "/v1/messages" || p == "/api/chat" {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			d.RecordRequest(r.Method, p, "", "", sw.status, time.Since(start))
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	written bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.written {
		sw.status = code
		sw.written = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (d *Dashboard) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin/", d.servePage)
	mux.HandleFunc("/manifest.json", d.serveStatic("static/manifest.json", "application/json"))
	mux.HandleFunc("/sw.js", d.serveStatic("static/sw.js", "application/javascript"))
	mux.HandleFunc("/icon-192.png", d.serveStatic("static/icon-192.png", "image/png"))
	mux.HandleFunc("/icon-512.png", d.serveStatic("static/icon-512.png", "image/png"))
	mux.HandleFunc("/screenshot-wide.png", d.serveStatic("static/screenshot-wide.png", "image/png"))
	mux.HandleFunc("/screenshot-narrow.png", d.serveStatic("static/screenshot-narrow.png", "image/png"))
	mux.HandleFunc("/admin/api/login", d.handleLogin)
	mux.HandleFunc("/admin/api/stats", d.handleStatsSSE)
	mux.HandleFunc("/admin/api/settings", d.handleSettings)
	mux.HandleFunc("/admin/api/terminal", d.handleTerminal)
	mux.HandleFunc("/admin/api/sso", d.handleSSOFlow)
	mux.HandleFunc("/admin/api/device-auth", d.handleDeviceAuth)
	mux.HandleFunc("/admin/api/logs", d.handleLogsSSE)
	mux.HandleFunc("/admin/api/models", d.handleModels)
	mux.HandleFunc("/admin/api/requests", d.handleRequests)
	mux.HandleFunc("/admin/api/keys/issue", d.handleKeyIssue)
	mux.HandleFunc("/admin/api/keys/rotate", d.handleKeyRotate)
	mux.HandleFunc("/admin/api/keys/revoke", d.handleKeyRevoke)
	mux.HandleFunc("/admin/api/keys", d.handleKeysList)
	mux.HandleFunc("/admin/api/audit", d.handleAuditLog)
	mux.HandleFunc("/admin/api/routes", d.handleRoutes)
}

func (d *Dashboard) AuthWrap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.checkAuth(r) {
			http.Error(w, "Unauthorized", 401)
			return
		}
		h(w, r)
	}
}

func (d *Dashboard) handleRoutes(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	routes := map[string]interface{}{
		"public": []string{
			"GET /health",
			"GET /v1/models",
			"POST /v1/chat/completions",
		},
		"admin": []string{
			"GET  /admin/",
			"POST /admin/api/login",
			"GET  /admin/api/stats",
			"GET  /admin/api/logs",
			"GET  /admin/api/sso",
			"POST /admin/api/device-auth",
			"GET  /admin/api/settings",
			"POST /admin/api/settings",
			"GET  /admin/api/models",
			"GET  /admin/api/keys",
			"POST /admin/api/keys/issue",
			"POST /admin/api/keys/rotate",
			"POST /admin/api/keys/revoke",
			"GET  /admin/api/audit",
			"GET  /admin/api/routes",
			"GET  /admin/api/config",
			"POST /admin/api/config",
			"GET  /admin/api/config/{key}",
			"PUT  /admin/api/config/{key}",
			"DELETE /admin/api/config/{key}",
			"GET  /admin/api/accounts",
			"GET  /admin/api/accounts/export",
			"POST /admin/api/accounts/import",
			"POST /admin/api/accounts/import/sso-token",
			"POST /admin/api/accounts/import/kiro-cli",
		},
	}
	writeJSON(w, 200, routes)
}

func (d *Dashboard) checkAuth(r *http.Request) bool {
	c, err := r.Cookie("kiro_admin")
	return err == nil && c.Value == d.sessionToken
}

func (d *Dashboard) servePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if d.checkAuth(r) {
		data, _ := staticFiles.ReadFile("static/index.html")
		w.Write(data)
	} else {
		data, _ := staticFiles.ReadFile("static/login.html")
		w.Write(data)
	}
}

func (d *Dashboard) serveStatic(path, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, _ := staticFiles.ReadFile(path)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(data)
	}
}

func (d *Dashboard) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct{ Key string `json:"key"` }
	json.NewDecoder(r.Body).Decode(&req)
	valid := req.Key == d.config.AdminAPIKey
	if !valid && d.keyManager != nil {
		if k, err := d.keyManager.ValidateKey(req.Key); err == nil && k.IsActive {
			valid = true
		}
	}
	if !valid {
		writeJSON(w, 401, map[string]string{"error": "invalid key"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "kiro_admin", Value: d.sessionToken,
		Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400,
	})
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// handleLogsSSE streams live logs via SSE
func (d *Dashboard) handleLogsSSE(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send recent logs first
	for _, entry := range d.logCapture.Recent(50) {
		data, _ := json.Marshal(entry)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	flusher.Flush()

	// Stream new logs
	ch := d.logCapture.Subscribe()
	defer d.logCapture.Unsubscribe(ch)

	for {
		select {
		case entry := <-ch:
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleModels returns the current model list
func (d *Dashboard) handleModels(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	// Proxy to the gateway's own /v1/models endpoint using the first valid API key
	key := d.getProxyKey()
	scheme := "http"
	if os.Getenv("TLS_CERT") != "" {
		scheme = "https"
	}
	req, _ := http.NewRequest("GET", scheme+"://127.0.0.1:"+d.config.Port+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func (d *Dashboard) getProxyKey() string {
	if d.config.ProxyAPIKey != "" {
		return d.config.ProxyAPIKey
	}
	if d.keyManager != nil {
		for _, k := range d.keyManager.ListKeys("") {
			if k.IsActive {
				return k.Key
			}
		}
	}
	return d.config.AdminAPIKey
}

// handleRequests returns recent request history
func (d *Dashboard) handleRequests(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	d.reqMu.Lock()
	// Return in reverse order (newest first)
	out := make([]RequestRecord, len(d.requests))
	for i, r := range d.requests {
		out[len(d.requests)-1-i] = r
	}
	d.reqMu.Unlock()
	writeJSON(w, 200, out)
}

// handleKeysList returns all API keys
func (d *Dashboard) handleKeysList(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) { http.Error(w, "Unauthorized", 401); return }
	if d.keyManager == nil { writeJSON(w, 200, map[string]interface{}{"keys": []interface{}{}}); return }
	keys := d.keyManager.ListKeys("")
	out := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]interface{}{
			"name": k.Name, "key_preview": apikeys.MaskKey(k.Key),
			"created_at": k.CreatedAt.Format(time.RFC3339),
			"is_active": k.IsActive, "usage_count": k.UsageCount,
			"user_id": k.UserID,
		})
	}
	writeJSON(w, 200, map[string]interface{}{"keys": out})
}

// handleKeyIssue creates a new 128-bit kiro-* key
func (d *Dashboard) handleKeyIssue(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) { http.Error(w, "Unauthorized", 401); return }
	if r.Method != http.MethodPost { http.Error(w, "Method not allowed", 405); return }
	if d.keyManager == nil { writeJSON(w, 500, map[string]string{"error": "key manager not available"}); return }
	var req struct{ Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" { writeJSON(w, 400, map[string]string{"error": "name required"}); return }
	key, err := d.keyManager.GenerateKey(req.Name, "admin", nil, []string{"*"})
	if err != nil { writeJSON(w, 500, map[string]string{"error": err.Error()}); return }
	d.addAudit("issue", req.Name, apikeys.MaskKey(key.Key), "New key issued")
	log.Printf("[ADMIN] Issued API key: %s (%s)", req.Name, apikeys.MaskKey(key.Key))
	writeJSON(w, 200, map[string]interface{}{"key": key.Key, "name": key.Name, "key_preview": apikeys.MaskKey(key.Key)})
}

// handleKeyRotate revokes old key and issues new one with same name
func (d *Dashboard) handleKeyRotate(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) { http.Error(w, "Unauthorized", 401); return }
	if r.Method != http.MethodPost { http.Error(w, "Method not allowed", 405); return }
	if d.keyManager == nil { writeJSON(w, 500, map[string]string{"error": "key manager not available"}); return }
	var req struct{ Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" { writeJSON(w, 400, map[string]string{"error": "name required"}); return }
	// Find and revoke existing key with this name
	for _, k := range d.keyManager.ListKeys("") {
		if k.Name == req.Name && k.IsActive {
			d.keyManager.RevokeKey(k.Key)
			d.addAudit("revoke", k.Name, apikeys.MaskKey(k.Key), "Rotated out")
		}
	}
	// Issue new key
	key, err := d.keyManager.GenerateKey(req.Name, "admin", nil, []string{"*"})
	if err != nil { writeJSON(w, 500, map[string]string{"error": err.Error()}); return }
	d.addAudit("rotate", req.Name, apikeys.MaskKey(key.Key), "New key after rotation")
	log.Printf("[ADMIN] Rotated API key: %s -> %s", req.Name, apikeys.MaskKey(key.Key))
	writeJSON(w, 200, map[string]interface{}{"key": key.Key, "name": key.Name, "key_preview": apikeys.MaskKey(key.Key)})
}

// handleKeyRevoke revokes a key by name
func (d *Dashboard) handleKeyRevoke(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) { http.Error(w, "Unauthorized", 401); return }
	if r.Method != http.MethodPost { http.Error(w, "Method not allowed", 405); return }
	if d.keyManager == nil { writeJSON(w, 500, map[string]string{"error": "key manager not available"}); return }
	var req struct{ Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" { writeJSON(w, 400, map[string]string{"error": "name required"}); return }
	revoked := 0
	for _, k := range d.keyManager.ListKeys("") {
		if k.Name == req.Name && k.IsActive {
			d.keyManager.RevokeKey(k.Key)
			d.addAudit("revoke", k.Name, apikeys.MaskKey(k.Key), "Manual revoke")
			revoked++
		}
	}
	if revoked == 0 { writeJSON(w, 404, map[string]string{"error": "no active key with that name"}); return }
	log.Printf("[ADMIN] Revoked %d key(s): %s", revoked, req.Name)
	writeJSON(w, 200, map[string]string{"status": "ok", "revoked": fmt.Sprintf("%d", revoked)})
}

// handleAuditLog returns the audit log
func (d *Dashboard) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) { http.Error(w, "Unauthorized", 401); return }
	d.auditMu.Lock()
	out := make([]AuditEntry, len(d.auditLog))
	for i, e := range d.auditLog {
		out[len(d.auditLog)-1-i] = e
	}
	d.auditMu.Unlock()
	writeJSON(w, 200, out)
}

func (d *Dashboard) addAudit(action, keyName, keyID, detail string) {
	d.auditMu.Lock()
	defer d.auditMu.Unlock()
	d.auditLog = append(d.auditLog, AuditEntry{
		Time: time.Now().Format("2006-01-02 15:04:05"), Action: action,
		KeyName: keyName, KeyID: keyID, Actor: "admin", Detail: detail,
	})
	if len(d.auditLog) > 500 { d.auditLog = d.auditLog[1:] }
}

func (d *Dashboard) handleStatsSSE(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 8)
	d.sseMu.Lock()
	d.sseClients[ch] = struct{}{}
	d.sseMu.Unlock()
	defer func() {
		d.sseMu.Lock()
		delete(d.sseClients, ch)
		d.sseMu.Unlock()
	}()

	if data, err := json.Marshal(d.collectStats()); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (d *Dashboard) broadcastLoop() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for range t.C {
		data, _ := json.Marshal(d.collectStats())
		d.sseMu.Lock()
		for ch := range d.sseClients {
			select {
			case ch <- data:
			default:
			}
		}
		d.sseMu.Unlock()
	}
}

func getAuthModeLabel(mode auth.AuthMode) string {
	if mode == auth.AuthModeSigV4 {
		return "SigV4"
	}
	return "Bearer"
}

func (d *Dashboard) collectStats() map[string]interface{} {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	d.reqMu.Lock()
	reqCount := len(d.requests)
	d.reqMu.Unlock()
	s := map[string]interface{}{
		"uptime_secs":  int(time.Since(d.startTime).Seconds()),
		"uptime":       time.Since(d.startTime).Round(time.Second).String(),
		"goroutines":   runtime.NumGoroutine(),
		"mem_alloc_mb": fmt.Sprintf("%.1f", float64(mem.Alloc)/1024/1024),
		"mem_sys_mb":   fmt.Sprintf("%.1f", float64(mem.Sys)/1024/1024),
		"gc_cycles":    mem.NumGC,
		"auth_type":    string(d.authManager.GetAuthType()),
		"auth_mode":    getAuthModeLabel(d.authManager.GetAuthMode()),
		"region":       d.authManager.GetRegion(),
		"profile_arn":  d.authManager.GetProfileArn(),
		"port":         d.config.Port,
		"total_requests": reqCount,
	}
	if d.stats != nil {
		for k, v := range d.stats.GetMetrics() {
			s[k] = v
		}
	}
	return s
}

func (d *Dashboard) handleSettings(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, map[string]interface{}{
			"hidden_models":         d.config.HiddenModels,
			"max_connections":       d.config.MaxConnections,
			"first_token_timeout_s": int(d.config.FirstTokenTimeout.Seconds()),
			"multimodal_timeout_s":  int(d.config.MultimodalFirstTokenTimeout.Seconds()),
		})
	case http.MethodPost:
		var u map[string]interface{}
		json.NewDecoder(r.Body).Decode(&u)
		d.applySettings(u)
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

func (d *Dashboard) applySettings(u map[string]interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v, ok := u["hidden_models"]; ok {
		if arr, ok := v.([]interface{}); ok {
			m := make([]string, 0, len(arr))
			for _, x := range arr {
				if s, ok := x.(string); ok {
					m = append(m, s)
				}
			}
			d.config.HiddenModels = m
			log.Printf("[ADMIN] Updated hidden_models: %v", m)
		}
	}
	if v, ok := u["max_connections"].(float64); ok {
		d.config.MaxConnections = int(v)
		log.Printf("[ADMIN] Updated max_connections: %d", int(v))
	}
	if v, ok := u["first_token_timeout_s"].(float64); ok {
		d.config.FirstTokenTimeout = time.Duration(v) * time.Second
	}
	if v, ok := u["multimodal_timeout_s"].(float64); ok {
		d.config.MultimodalFirstTokenTimeout = time.Duration(v) * time.Second
	}
	if v, ok := u["vpn_proxy_url"].(string); ok {
		d.config.VPNProxyURL = v
	}
}

func (d *Dashboard) handleTerminal(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct{ Command string `json:"command"` }
	json.NewDecoder(r.Body).Decode(&req)
	writeJSON(w, 200, map[string]string{"output": d.execCmd(req.Command)})
}

func (d *Dashboard) execCmd(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "help":
		return "Commands: login, logout, status, models, profile, env, uptime, cache, date, df, free, whoami, uname"
	case "login":
		ha := d.authManager.GetHeadlessAuth()
		if ha == nil {
			return "Error: not in headless mode"
		}
		if ha.IsAuthenticated() {
			return "Already authenticated. Use 'logout' to re-authenticate."
		}
		url, code, err := ha.StartLogin(context.Background())
		if err != nil {
			return "Login failed: " + err.Error()
		}
		return fmt.Sprintf("Open this URL and approve:\n\n  %s\n\n  Code: %s", url, code)
	case "logout":
		ha := d.authManager.GetHeadlessAuth()
		if ha == nil {
			return "Error: not in headless mode"
		}
		ha.ClearCredentials()
		// Broadcast update
		state := ha.DeviceFlowState()
		state["authenticated"] = false
		d.BroadcastSSOUpdate(state)
		d.addAudit("logout", "admin", "", "Manual logout")
		return "Token cleared. Run 'login' to re-authenticate."
	case "status":
		b, _ := json.MarshalIndent(d.collectStats(), "", "  ")
		return string(b)
	case "models":
		return fmt.Sprintf("Profile: %s\nHit GET /v1/models for current list", d.authManager.GetProfileArn())
	case "profile":
		if arn := d.authManager.GetProfileArn(); arn != "" {
			return arn
		}
		return "(not set)"
	case "env":
		var lines []string
		for _, k := range []string{"AWS_REGION", "HEADLESS_MODE", "AUTOMATE_AUTH", "Q_USE_SENDMESSAGE", "ENABLE_SIGV4"} {
			v := os.Getenv(k)
			if v == "" {
				v = "(not set)"
			}
			lines = append(lines, k+"="+v)
		}
		return strings.Join(lines, "\n")
	case "uptime":
		return time.Since(d.startTime).Round(time.Second).String()
	case "cache":
		if d.stats != nil {
			if c, ok := d.stats.GetMetrics()["response_cache"]; ok {
				b, _ := json.MarshalIndent(c, "", "  ")
				return string(b)
			}
		}
		return "No cache stats"
	default:
		safe := map[string]bool{"date": true, "whoami": true, "df": true, "free": true, "uname": true}
		if safe[parts[0]] {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, parts[0], parts[1:]...).CombinedOutput()
			if err != nil {
				return string(out) + "\n" + err.Error()
			}
			return string(out)
		}
		return "Unknown command: " + parts[0] + " (type 'help')"
	}
}

func (d *Dashboard) handleSSOFlow(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	data, _ := json.Marshal(d.getSSOState())
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	ch := make(chan []byte, 4)
	d.ssoMu.Lock()
	d.ssoClients[ch] = struct{}{}
	d.ssoMu.Unlock()
	defer func() {
		d.ssoMu.Lock()
		delete(d.ssoClients, ch)
		d.ssoMu.Unlock()
	}()
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (d *Dashboard) handleDeviceAuth(w http.ResponseWriter, r *http.Request) {
	if !d.checkAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	ha := d.authManager.GetHeadlessAuth()
	if ha == nil {
		writeJSON(w, 400, map[string]string{"error": "not in headless mode"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		// Start a new device flow login
		if ha.IsAuthenticated() {
			writeJSON(w, 200, map[string]string{"status": "already authenticated"})
			return
		}
		// Apply runtime config overrides before login
		if d.runtimeCfg != nil {
			ha.UpdateConfig(
				d.runtimeCfg.GetString("sso_start_url"),
				d.runtimeCfg.GetString("sso_region"),
				d.runtimeCfg.GetString("sso_account_id"),
				d.runtimeCfg.GetString("sso_role_name"),
			)
		}
		url, code, err := ha.StartLogin(context.Background())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		// Broadcast to SSE clients
		state := map[string]interface{}{"pending": true, "authenticated": false, "verification_url": url, "user_code": code}
		d.BroadcastSSOUpdate(state)
		d.addAudit("login", "device-flow", code, "Device flow started")
		writeJSON(w, 200, map[string]interface{}{"status": "pending", "verification_url": url, "user_code": code})
		return
	}
	state := ha.DeviceFlowState()
	state["authenticated"] = ha.IsAuthenticated()
	json.NewEncoder(w).Encode(state)
}

func (d *Dashboard) getSSOState() map[string]interface{} {
	ha := d.authManager.GetHeadlessAuth()
	state := map[string]interface{}{
		"auth_type": string(d.authManager.GetAuthType()),
		"region":    d.authManager.GetRegion(),
	}
	if ha != nil {
		isAuth := ha.IsAuthenticated()
		// Also consider authenticated if we have a working profile (server can make API calls)
		if !isAuth && d.authManager.GetProfileArn() != "" {
			isAuth = true
		}
		if isAuth {
			state["authenticated"] = true
		} else {
			df := ha.DeviceFlowState()
			if df["active"].(bool) {
				state["pending"] = true
				state["authenticated"] = false
				state["user_code"] = df["user_code"]
				state["verification_url"] = df["verify_url"]
			} else {
				state["authenticated"] = false
			}
		}
	} else {
		state["authenticated"] = d.authManager.GetProfileArn() != ""
	}
	return state
}

func (d *Dashboard) BroadcastSSOUpdate(state map[string]interface{}) {
	data, _ := json.Marshal(state)
	d.ssoMu.Lock()
	for ch := range d.ssoClients {
		select {
		case ch <- data:
		default:
		}
	}
	d.ssoMu.Unlock()
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
