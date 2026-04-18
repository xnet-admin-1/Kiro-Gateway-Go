package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleMetrics returns metrics from all concurrency components
func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	metrics := make(map[string]interface{})
	
	// Add priority queue metrics
	if h.priorityQueue != nil {
		metrics["priority_queue"] = h.priorityQueue.GetStats()
	}
	
	// Add load shedder metrics
	if h.loadShedder != nil {
		metrics["load_shedder"] = h.loadShedder.GetStats()
	}
	
	// Add async job manager metrics
	if h.asyncJobManager != nil {
		metrics["async_jobs"] = h.asyncJobManager.GetStats()
	}
	
	// Add cache metrics
	if h.responseCache != nil {
		metrics["response_cache"] = h.responseCache.GetStats()
	}
	
	// Add request deduplication status
	if h.requestDedup != nil {
		metrics["request_dedup"] = map[string]interface{}{
			"enabled": h.requestDedup.IsEnabled(),
		}
	}
	
	// Note: Rate limiter and quota tracker don't have GetStats methods yet
	// They can be added later if needed
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Error encoding metrics: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleCacheStats returns detailed cache statistics (admin only)
func (h *Handler) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats := make(map[string]interface{})
	
	// Response cache stats
	if h.responseCache != nil {
		stats["response_cache"] = h.responseCache.GetStats()
	} else {
		stats["response_cache"] = map[string]interface{}{
			"enabled": false,
		}
	}
	
	// Request deduplication stats
	if h.requestDedup != nil {
		stats["request_dedup"] = map[string]interface{}{
			"enabled": h.requestDedup.IsEnabled(),
		}
	} else {
		stats["request_dedup"] = map[string]interface{}{
			"enabled": false,
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding cache stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
