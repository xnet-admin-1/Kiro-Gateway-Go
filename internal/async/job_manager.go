package async

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/concurrency"
	"github.com/yourusername/kiro-gateway-go/internal/models"
)

// JobStatus represents the status of an async job
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

// AsyncJob represents an asynchronous job
type AsyncJob struct {
	// Identification
	ID          string
	UserID      string
	Priority    int
	
	// Request data
	Request     *models.ChatCompletionRequest
	
	// Response data
	Response    interface{}
	Error       error
	
	// Status
	Status      JobStatus
	
	// Timing
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	ExpiresAt   time.Time
	
	// Callback
	CallbackURL string
	CallbackHeaders map[string]string
	
	// Metadata
	Metadata    map[string]interface{}
	
	// Internal
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
}

// AsyncJobManager manages asynchronous job processing
type AsyncJobManager struct {
	// Storage
	jobs        map[string]*AsyncJob
	mu          sync.RWMutex
	
	// Components
	priorityQueue *concurrency.PriorityQueue
	
	// Configuration
	jobTTL            time.Duration
	webhookTimeout    time.Duration
	cleanupInterval   time.Duration
	
	// Metrics
	metrics *AsyncJobMetrics
	
	// State
	started atomic.Bool
	stopped atomic.Bool
	
	// Context
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AsyncJobMetrics tracks async job performance
type AsyncJobMetrics struct {
	JobsCreated    atomic.Int64
	JobsCompleted  atomic.Int64
	JobsFailed     atomic.Int64
	JobsCancelled  atomic.Int64
	JobsExpired    atomic.Int64
	
	WebhooksSuccess atomic.Int64
	WebhooksFailed  atomic.Int64
	
	mu sync.RWMutex
}

// AsyncJobManagerConfig holds async job manager configuration
type AsyncJobManagerConfig struct {
	JobTTL          time.Duration
	WebhookTimeout  time.Duration
	CleanupInterval time.Duration
	PriorityQueue   *concurrency.PriorityQueue
}

// NewAsyncJobManager creates a new async job manager
func NewAsyncJobManager(cfg AsyncJobManagerConfig) *AsyncJobManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults
	if cfg.JobTTL <= 0 {
		cfg.JobTTL = 24 * time.Hour
	}
	if cfg.WebhookTimeout <= 0 {
		cfg.WebhookTimeout = 10 * time.Second
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 1 * time.Hour
	}
	
	ajm := &AsyncJobManager{
		jobs:            make(map[string]*AsyncJob),
		priorityQueue:   cfg.PriorityQueue,
		jobTTL:          cfg.JobTTL,
		webhookTimeout:  cfg.WebhookTimeout,
		cleanupInterval: cfg.CleanupInterval,
		metrics:         &AsyncJobMetrics{},
		ctx:             ctx,
		cancel:          cancel,
	}
	
	log.Printf("Async job manager initialized (TTL: %v, webhook timeout: %v)",
		cfg.JobTTL, cfg.WebhookTimeout)
	
	return ajm
}

// Start starts the async job manager
func (ajm *AsyncJobManager) Start() error {
	if ajm.started.Load() {
		return fmt.Errorf("async job manager already started")
	}
	
	ajm.started.Store(true)
	
	// Start cleanup loop
	ajm.wg.Add(1)
	go ajm.cleanupLoop()
	
	log.Println("Async job manager started")
	
	return nil
}

// Stop stops the async job manager
func (ajm *AsyncJobManager) Stop(timeout time.Duration) error {
	if ajm.stopped.Load() {
		return fmt.Errorf("async job manager already stopped")
	}
	
	log.Println("Stopping async job manager...")
	
	ajm.stopped.Store(true)
	ajm.cancel()
	
	// Cancel all pending jobs
	ajm.mu.Lock()
	for _, job := range ajm.jobs {
		if job.Status == JobStatusPending || job.Status == JobStatusQueued || job.Status == JobStatusRunning {
			job.cancel()
			job.Status = JobStatusCancelled
			ajm.metrics.JobsCancelled.Add(1)
		}
	}
	ajm.mu.Unlock()
	
	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		ajm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("Async job manager stopped gracefully")
		return nil
	case <-time.After(timeout):
		log.Println("Async job manager stopped with timeout")
		return fmt.Errorf("async job manager stop timeout after %v", timeout)
	}
}

// CreateJob creates a new async job
func (ajm *AsyncJobManager) CreateJob(id, userID string, priority int, request *models.ChatCompletionRequest, callbackURL string, callbackHeaders map[string]string) (*AsyncJob, error) {
	if ajm.stopped.Load() {
		return nil, fmt.Errorf("async job manager is stopped")
	}
	
	ctx, cancel := context.WithCancel(ajm.ctx)
	
	job := &AsyncJob{
		ID:              id,
		UserID:          userID,
		Priority:        priority,
		Request:         request,
		Status:          JobStatusPending,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(ajm.jobTTL),
		CallbackURL:     callbackURL,
		CallbackHeaders: callbackHeaders,
		Metadata:        make(map[string]interface{}),
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Store job
	ajm.mu.Lock()
	ajm.jobs[id] = job
	ajm.mu.Unlock()
	
	ajm.metrics.JobsCreated.Add(1)
	
	log.Printf("Async job created: %s (user: %s, priority: %s)", 
		id, userID, concurrency.GetPriorityName(priority))
	
	return job, nil
}

// SubmitJob submits an async job for processing
func (ajm *AsyncJobManager) SubmitJob(jobID string) error {
	ajm.mu.RLock()
	job, exists := ajm.jobs[jobID]
	ajm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	
	// Create concurrency job
	concJob := concurrency.NewJob(
		job.ID,
		job.ID,
		job.UserID,
		job.Priority,
		job.Request,
		job.ctx,
	)
	
	// Submit to priority queue
	if err := ajm.priorityQueue.Enqueue(concJob); err != nil {
		job.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = err
		job.mu.Unlock()
		return err
	}
	
	// Update status
	job.mu.Lock()
	job.Status = JobStatusQueued
	job.mu.Unlock()
	
	// Wait for result in background
	ajm.wg.Add(1)
	go ajm.waitForResult(job, concJob)
	
	return nil
}

// waitForResult waits for a job result and updates the async job
func (ajm *AsyncJobManager) waitForResult(job *AsyncJob, concJob *concurrency.Job) {
	defer ajm.wg.Done()
	
	select {
	case result := <-concJob.ResultChan:
		job.mu.Lock()
		job.CompletedAt = time.Now()
		
		if result.Error != nil {
			job.Status = JobStatusFailed
			job.Error = result.Error
			ajm.metrics.JobsFailed.Add(1)
		} else {
			job.Status = JobStatusCompleted
			job.Response = result.Response
			ajm.metrics.JobsCompleted.Add(1)
		}
		job.mu.Unlock()
		
		log.Printf("Async job completed: %s (status: %s, duration: %v)",
			job.ID, job.Status, result.Duration)
		
		// Send webhook if configured
		if job.CallbackURL != "" {
			ajm.sendWebhook(job)
		}
		
	case <-job.ctx.Done():
		job.mu.Lock()
		job.Status = JobStatusCancelled
		job.CompletedAt = time.Now()
		job.mu.Unlock()
		
		ajm.metrics.JobsCancelled.Add(1)
		
		log.Printf("Async job cancelled: %s", job.ID)
	}
}

// sendWebhook sends a webhook notification for a completed job
func (ajm *AsyncJobManager) sendWebhook(job *AsyncJob) {
	// TODO: Implement webhook sending
	// This would use an HTTP client to POST the job result to the callback URL
	
	log.Printf("Webhook notification for job %s to %s", job.ID, job.CallbackURL)
	ajm.metrics.WebhooksSuccess.Add(1)
}

// GetJob retrieves an async job by ID
func (ajm *AsyncJobManager) GetJob(jobID string) (*AsyncJob, error) {
	ajm.mu.RLock()
	defer ajm.mu.RUnlock()
	
	job, exists := ajm.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	
	return job, nil
}

// CancelJob cancels an async job
func (ajm *AsyncJobManager) CancelJob(jobID string) error {
	ajm.mu.RLock()
	job, exists := ajm.jobs[jobID]
	ajm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	
	job.mu.Lock()
	defer job.mu.Unlock()
	
	if job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled {
		return fmt.Errorf("job already finished: %s", job.Status)
	}
	
	job.cancel()
	job.Status = JobStatusCancelled
	job.CompletedAt = time.Now()
	
	ajm.metrics.JobsCancelled.Add(1)
	
	log.Printf("Async job cancelled: %s", jobID)
	
	return nil
}

// ListJobs lists all jobs for a user
func (ajm *AsyncJobManager) ListJobs(userID string, status JobStatus) []*AsyncJob {
	ajm.mu.RLock()
	defer ajm.mu.RUnlock()
	
	var jobs []*AsyncJob
	for _, job := range ajm.jobs {
		if job.UserID == userID {
			if status == "" || job.Status == status {
				jobs = append(jobs, job)
			}
		}
	}
	
	return jobs
}

// cleanupLoop periodically cleans up expired jobs
func (ajm *AsyncJobManager) cleanupLoop() {
	defer ajm.wg.Done()
	
	ticker := time.NewTicker(ajm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ajm.ctx.Done():
			return
		case <-ticker.C:
			ajm.cleanupExpiredJobs()
		}
	}
}

// cleanupExpiredJobs removes expired jobs
func (ajm *AsyncJobManager) cleanupExpiredJobs() {
	now := time.Now()
	
	ajm.mu.Lock()
	defer ajm.mu.Unlock()
	
	expiredCount := 0
	for id, job := range ajm.jobs {
		if now.After(job.ExpiresAt) {
			delete(ajm.jobs, id)
			expiredCount++
			ajm.metrics.JobsExpired.Add(1)
		}
	}
	
	if expiredCount > 0 {
		log.Printf("Cleaned up %d expired async jobs", expiredCount)
	}
}

// GetStats returns human-readable statistics
func (ajm *AsyncJobManager) GetStats() map[string]interface{} {
	ajm.mu.RLock()
	totalJobs := len(ajm.jobs)
	
	statusCounts := make(map[JobStatus]int)
	for _, job := range ajm.jobs {
		statusCounts[job.Status]++
	}
	ajm.mu.RUnlock()
	
	return map[string]interface{}{
		"configuration": map[string]interface{}{
			"job_ttl":         ajm.jobTTL.String(),
			"webhook_timeout": ajm.webhookTimeout.String(),
		},
		"jobs": map[string]interface{}{
			"total":     totalJobs,
			"pending":   statusCounts[JobStatusPending],
			"queued":    statusCounts[JobStatusQueued],
			"running":   statusCounts[JobStatusRunning],
			"completed": statusCounts[JobStatusCompleted],
			"failed":    statusCounts[JobStatusFailed],
			"cancelled": statusCounts[JobStatusCancelled],
		},
		"metrics": map[string]interface{}{
			"created":   ajm.metrics.JobsCreated.Load(),
			"completed": ajm.metrics.JobsCompleted.Load(),
			"failed":    ajm.metrics.JobsFailed.Load(),
			"cancelled": ajm.metrics.JobsCancelled.Load(),
			"expired":   ajm.metrics.JobsExpired.Load(),
		},
		"webhooks": map[string]interface{}{
			"success": ajm.metrics.WebhooksSuccess.Load(),
			"failed":  ajm.metrics.WebhooksFailed.Load(),
		},
	}
}

// IsHealthy returns true if the async job manager is healthy
func (ajm *AsyncJobManager) IsHealthy() bool {
	if !ajm.started.Load() || ajm.stopped.Load() {
		return false
	}
	
	// Check if there are too many pending jobs
	ajm.mu.RLock()
	pendingCount := 0
	for _, job := range ajm.jobs {
		if job.Status == JobStatusPending || job.Status == JobStatusQueued {
			pendingCount++
		}
	}
	ajm.mu.RUnlock()
	
	// Unhealthy if >1000 pending jobs
	return pendingCount < 1000
}
