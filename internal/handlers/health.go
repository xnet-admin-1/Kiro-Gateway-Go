package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

const version = "1.0.0"

// handleRoot handles GET /
func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	response := map[string]interface{}{
		"status":  "ok",
		"message": "Kiro Gateway is running",
		"version": version,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles GET /health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	health := make(map[string]interface{})
	allHealthy := true
	
	// Check priority queue health
	if h.priorityQueue != nil {
		healthy := h.priorityQueue.IsHealthy()
		health["priority_queue"] = healthy
		if !healthy {
			allHealthy = false
		}
	}
	
	// Check load shedder health
	if h.loadShedder != nil {
		healthy := h.loadShedder.IsHealthy()
		health["load_shedder"] = healthy
		if !healthy {
			allHealthy = false
		}
	}
	
	// Check async job manager health
	if h.asyncJobManager != nil {
		healthy := h.asyncJobManager.IsHealthy()
		health["async_jobs"] = healthy
		if !healthy {
			allHealthy = false
		}
	}
	
	// Overall status
	health["status"] = allHealthy
	health["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	health["version"] = version
	
	if allHealthy {
		health["message"] = "All systems operational"
	} else {
		health["message"] = "Some systems degraded"
	}
	
	// Set status code
	statusCode := http.StatusOK
	if !allHealthy {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}
