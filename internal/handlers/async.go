package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/yourusername/kiro-gateway-go/internal/concurrency"
	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// handleAsyncJobs handles POST /v1/async/jobs (create async job)
func (h *Handler) handleAsyncJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.asyncJobManager == nil {
		http.Error(w, "Async jobs not enabled", http.StatusNotImplemented)
		return
	}
	
	// Generate request ID
	requestID := generateRequestID()
	
	log.Printf("[%s] Creating async job", requestID)
	
	// Parse request
	var req models.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Invalid request body", err, requestID)
		return
	}
	
	// Validate request
	if err := h.validator.ValidateRequest(&req); err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Validation failed", err, requestID)
		return
	}
	
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		userID = "anonymous"
	}
	
	// Determine priority
	priority := concurrency.GetPriorityFromUser(userID, nil)
	
	// Get callback URL from headers
	callbackURL := r.Header.Get("X-Callback-URL")
	
	// Get callback headers
	callbackHeaders := make(map[string]string)
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-Callback-Header-") {
			headerName := strings.TrimPrefix(key, "X-Callback-Header-")
			if len(values) > 0 {
				callbackHeaders[headerName] = values[0]
			}
		}
	}
	
	// Create async job
	job, err := h.asyncJobManager.CreateJob(requestID, userID, priority, &req, callbackURL, callbackHeaders)
	if err != nil {
		h.writeErrorWithRequestID(w, http.StatusInternalServerError, "Failed to create job", err, requestID)
		return
	}
	
	// Submit job for processing
	if err := h.asyncJobManager.SubmitJob(job.ID); err != nil {
		h.writeErrorWithRequestID(w, http.StatusServiceUnavailable, "Failed to submit job", err, requestID)
		return
	}
	
	// Return job info
	response := map[string]interface{}{
		"id":          job.ID,
		"status":      job.Status,
		"created_at":  job.CreatedAt,
		"expires_at":  job.ExpiresAt,
		"status_url":  fmt.Sprintf("/v1/async/jobs/%s", job.ID),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
	
	log.Printf("[%s] Async job created: %s", requestID, job.ID)
}

// handleAsyncJobStatus handles GET /v1/async/jobs/{id} (get job status)
func (h *Handler) handleAsyncJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.asyncJobManager == nil {
		http.Error(w, "Async jobs not enabled", http.StatusNotImplemented)
		return
	}
	
	// Extract job ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/async/jobs/")
	jobID := strings.Split(path, "/")[0]
	
	if jobID == "" {
		http.Error(w, "Job ID required", http.StatusBadRequest)
		return
	}
	
	// Get job
	job, err := h.asyncJobManager.GetJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	
	// Handle DELETE (cancel job)
	if r.Method == http.MethodDelete {
		if err := h.asyncJobManager.CancelJob(jobID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to cancel job: %v", err), http.StatusBadRequest)
			return
		}
		
		w.WriteHeader(http.StatusNoContent)
		log.Printf("Async job cancelled: %s", jobID)
		return
	}
	
	// Handle GET (get status)
	response := map[string]interface{}{
		"id":         job.ID,
		"status":     job.Status,
		"created_at": job.CreatedAt,
		"expires_at": job.ExpiresAt,
	}
	
	if !job.StartedAt.IsZero() {
		response["started_at"] = job.StartedAt
	}
	
	if !job.CompletedAt.IsZero() {
		response["completed_at"] = job.CompletedAt
	}
	
	if job.Response != nil {
		response["response"] = job.Response
	}
	
	if job.Error != nil {
		response["error"] = job.Error.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAsyncDirectJobs handles POST /v1/async/direct (create async job for direct API)
func (h *Handler) handleAsyncDirectJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.asyncJobManager == nil {
		http.Error(w, "Async jobs not enabled", http.StatusNotImplemented)
		return
	}
	
	requestID := generateRequestID()
	log.Printf("[%s] Creating async direct API job", requestID)
	
	// Parse Q Developer native request
	var req models.ConversationStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorWithRequestID(w, http.StatusBadRequest, "Invalid request body", err, requestID)
		return
	}
	
	// Extract model ID from header
	modelID := r.Header.Get("X-Model-Id")
	if modelID == "" {
		modelID = "claude-sonnet-4-5"
	}
	
	// TODO: Validation removed for pure passthrough - async endpoint may need refactoring
	// For now, skip validation since we're not unmarshaling the request
	
	// Get user ID and priority
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		userID = "anonymous"
	}
	priority := concurrency.GetPriorityFromUser(userID, nil)
	
	// Get callback info
	callbackURL := r.Header.Get("X-Callback-URL")
	callbackHeaders := make(map[string]string)
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-Callback-Header-") {
			headerName := strings.TrimPrefix(key, "X-Callback-Header-")
			if len(values) > 0 {
				callbackHeaders[headerName] = values[0]
			}
		}
	}
	
	// Create async job
	// TODO: Pass raw request instead of chatReq for pure passthrough
	job, err := h.asyncJobManager.CreateJob(requestID, userID, priority, nil, callbackURL, callbackHeaders)
	if err != nil {
		h.writeErrorWithRequestID(w, http.StatusInternalServerError, "Failed to create job", err, requestID)
		return
	}
	
	// Submit job
	if err := h.asyncJobManager.SubmitJob(job.ID); err != nil {
		h.writeErrorWithRequestID(w, http.StatusServiceUnavailable, "Failed to submit job", err, requestID)
		return
	}
	
	// Return job info
	response := map[string]interface{}{
		"id":         job.ID,
		"status":     job.Status,
		"created_at": job.CreatedAt,
		"expires_at": job.ExpiresAt,
		"status_url": fmt.Sprintf("/v1/async/jobs/%s", job.ID),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
	
	log.Printf("[%s] Async direct API job created: %s", requestID, job.ID)
}
