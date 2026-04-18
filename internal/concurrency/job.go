package concurrency

import (
	"context"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// Priority levels for jobs
const (
	PriorityHigh   = 0 // Premium users, urgent requests
	PriorityNormal = 1 // Standard users
	PriorityLow    = 2 // Batch processing, background tasks
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusTimeout   JobStatus = "timeout"
)

// Job represents a request to be processed by the worker pool
type Job struct {
	// Identification
	ID        string
	RequestID string
	UserID    string
	
	// Priority and timing
	Priority   int
	CreatedAt  time.Time
	StartedAt  time.Time
	CompletedAt time.Time
	
	// Request data
	Request    *models.ChatCompletionRequest
	Context    context.Context
	
	// Response handling
	ResultChan chan *Result
	
	// Metadata
	Metadata   map[string]interface{}
}

// Result represents the result of a processed job
type Result struct {
	// Job identification
	JobID     string
	RequestID string
	
	// Response data
	Response  interface{}
	Error     error
	
	// Timing
	Duration  time.Duration
	QueueTime time.Duration
	
	// Status
	Status    JobStatus
	
	// Metadata
	Metadata  map[string]interface{}
}

// NewJob creates a new job
func NewJob(id, requestID, userID string, priority int, req *models.ChatCompletionRequest, ctx context.Context) *Job {
	return &Job{
		ID:         id,
		RequestID:  requestID,
		UserID:     userID,
		Priority:   priority,
		CreatedAt:  time.Now(),
		Request:    req,
		Context:    ctx,
		ResultChan: make(chan *Result, 1),
		Metadata:   make(map[string]interface{}),
	}
}

// NewResult creates a new result
func NewResult(jobID, requestID string, response interface{}, err error, duration, queueTime time.Duration) *Result {
	status := JobStatusCompleted
	if err != nil {
		status = JobStatusFailed
	}
	
	return &Result{
		JobID:     jobID,
		RequestID: requestID,
		Response:  response,
		Error:     err,
		Duration:  duration,
		QueueTime: queueTime,
		Status:    status,
		Metadata:  make(map[string]interface{}),
	}
}

// GetPriorityName returns the name of a priority level
func GetPriorityName(priority int) string {
	switch priority {
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

// GetPriorityFromUser determines priority based on user tier
func GetPriorityFromUser(userID string, metadata map[string]interface{}) int {
	// Check metadata for explicit priority
	if priority, ok := metadata["priority"].(int); ok {
		return priority
	}
	
	// Check for premium user flag
	if isPremium, ok := metadata["is_premium"].(bool); ok && isPremium {
		return PriorityHigh
	}
	
	// Check for batch flag
	if isBatch, ok := metadata["is_batch"].(bool); ok && isBatch {
		return PriorityLow
	}
	
	// Default to normal priority
	return PriorityNormal
}
