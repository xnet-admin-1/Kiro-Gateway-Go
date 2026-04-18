package concurrency

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of workers that process jobs
type WorkerPool struct {
	// Configuration
	workerCount int
	queueSize   int
	jobTimeout  time.Duration
	
	// Channels
	jobQueue    chan *Job
	stopChan    chan struct{}
	
	// Context
	ctx    context.Context
	cancel context.CancelFunc
	
	// State
	wg      sync.WaitGroup
	started atomic.Bool
	stopped atomic.Bool
	
	// Metrics
	metrics *WorkerPoolMetrics
	
	// Job processor
	processor JobProcessor
}

// JobProcessor is a function that processes a job
type JobProcessor func(ctx context.Context, job *Job) (*Result, error)

// WorkerPoolMetrics tracks worker pool performance
type WorkerPoolMetrics struct {
	// Counters
	JobsEnqueued   atomic.Int64
	JobsProcessed  atomic.Int64
	JobsFailed     atomic.Int64
	JobsTimeout    atomic.Int64
	JobsDropped    atomic.Int64
	
	// Gauges
	ActiveWorkers  atomic.Int32
	QueueDepth     atomic.Int32
	
	// Timing
	TotalProcessTime atomic.Int64 // nanoseconds
	TotalQueueTime   atomic.Int64 // nanoseconds
	
	// Lock for complex operations
	mu sync.RWMutex
}

// WorkerPoolConfig holds worker pool configuration
type WorkerPoolConfig struct {
	WorkerCount int
	QueueSize   int
	JobTimeout  time.Duration
	Processor   JobProcessor
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(cfg WorkerPoolConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 20
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1000
	}
	if cfg.JobTimeout <= 0 {
		cfg.JobTimeout = 60 * time.Second
	}
	
	return &WorkerPool{
		workerCount: cfg.WorkerCount,
		queueSize:   cfg.QueueSize,
		jobTimeout:  cfg.JobTimeout,
		jobQueue:    make(chan *Job, cfg.QueueSize),
		stopChan:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &WorkerPoolMetrics{},
		processor:   cfg.Processor,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() error {
	if wp.started.Load() {
		return fmt.Errorf("worker pool already started")
	}
	
	if wp.processor == nil {
		return fmt.Errorf("job processor not set")
	}
	
	wp.started.Store(true)
	
	// Start workers
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	
	log.Printf("Worker pool started with %d workers (queue size: %d)", wp.workerCount, wp.queueSize)
	
	return nil
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop(timeout time.Duration) error {
	if wp.stopped.Load() {
		return fmt.Errorf("worker pool already stopped")
	}
	
	log.Println("Stopping worker pool...")
	
	wp.stopped.Store(true)
	close(wp.stopChan)
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("Worker pool stopped gracefully")
		return nil
	case <-time.After(timeout):
		wp.cancel() // Force cancel remaining jobs
		log.Println("Worker pool stopped with timeout")
		return fmt.Errorf("worker pool stop timeout after %v", timeout)
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job *Job) error {
	if wp.stopped.Load() {
		return fmt.Errorf("worker pool is stopped")
	}
	
	if !wp.started.Load() {
		return fmt.Errorf("worker pool not started")
	}
	
	// Try to enqueue job
	select {
	case wp.jobQueue <- job:
		wp.metrics.JobsEnqueued.Add(1)
		wp.metrics.QueueDepth.Add(1)
		return nil
	default:
		// Queue is full, drop job
		wp.metrics.JobsDropped.Add(1)
		return fmt.Errorf("job queue is full (size: %d)", wp.queueSize)
	}
}

// SubmitWithTimeout submits a job with a timeout
func (wp *WorkerPool) SubmitWithTimeout(job *Job, timeout time.Duration) error {
	if wp.stopped.Load() {
		return fmt.Errorf("worker pool is stopped")
	}
	
	if !wp.started.Load() {
		return fmt.Errorf("worker pool not started")
	}
	
	// Try to enqueue job with timeout
	select {
	case wp.jobQueue <- job:
		wp.metrics.JobsEnqueued.Add(1)
		wp.metrics.QueueDepth.Add(1)
		return nil
	case <-time.After(timeout):
		wp.metrics.JobsDropped.Add(1)
		return fmt.Errorf("job submission timeout after %v", timeout)
	}
}

// worker is the main worker loop
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	log.Printf("Worker %d started", id)
	
	for {
		select {
		case <-wp.stopChan:
			log.Printf("Worker %d stopping", id)
			return
			
		case <-wp.ctx.Done():
			log.Printf("Worker %d cancelled", id)
			return
			
		case job := <-wp.jobQueue:
			wp.metrics.QueueDepth.Add(-1)
			wp.processJob(id, job)
		}
	}
}

// processJob processes a single job
func (wp *WorkerPool) processJob(workerID int, job *Job) {
	wp.metrics.ActiveWorkers.Add(1)
	defer wp.metrics.ActiveWorkers.Add(-1)
	
	// Mark job as started
	job.StartedAt = time.Now()
	queueTime := job.StartedAt.Sub(job.CreatedAt)
	wp.metrics.TotalQueueTime.Add(queueTime.Nanoseconds())
	
	log.Printf("[Worker %d] Processing job %s (priority: %s, queue time: %v)", 
		workerID, job.ID, GetPriorityName(job.Priority), queueTime)
	
	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(job.Context, wp.jobTimeout)
	defer cancel()
	
	// Process job
	result, err := wp.processor(jobCtx, job)
	
	// Handle timeout
	if jobCtx.Err() == context.DeadlineExceeded {
		log.Printf("[Worker %d] Job %s timeout after %v", workerID, job.ID, wp.jobTimeout)
		wp.metrics.JobsTimeout.Add(1)
		
		result = &Result{
			JobID:     job.ID,
			RequestID: job.RequestID,
			Error:     fmt.Errorf("job timeout after %v", wp.jobTimeout),
			Status:    JobStatusTimeout,
			Duration:  wp.jobTimeout,
			QueueTime: queueTime,
		}
	} else if err != nil {
		log.Printf("[Worker %d] Job %s failed: %v", workerID, job.ID, err)
		wp.metrics.JobsFailed.Add(1)
		
		if result == nil {
			result = &Result{
				JobID:     job.ID,
				RequestID: job.RequestID,
				Error:     err,
				Status:    JobStatusFailed,
				Duration:  time.Since(job.StartedAt),
				QueueTime: queueTime,
			}
		}
	} else {
		wp.metrics.JobsProcessed.Add(1)
		
		if result == nil {
			result = &Result{
				JobID:     job.ID,
				RequestID: job.RequestID,
				Status:    JobStatusCompleted,
				Duration:  time.Since(job.StartedAt),
				QueueTime: queueTime,
			}
		}
	}
	
	// Update metrics
	job.CompletedAt = time.Now()
	processingTime := job.CompletedAt.Sub(job.StartedAt)
	wp.metrics.TotalProcessTime.Add(processingTime.Nanoseconds())
	
	log.Printf("[Worker %d] Job %s completed (status: %s, duration: %v)", 
		workerID, job.ID, result.Status, processingTime)
	
	// Send result to job's result channel
	select {
	case job.ResultChan <- result:
		// Result sent successfully
	default:
		// Result channel full or closed, log warning
		log.Printf("[Worker %d] Warning: Could not send result for job %s", workerID, job.ID)
	}
}

// GetMetrics returns a snapshot of worker pool metrics
func (wp *WorkerPool) GetMetrics() WorkerPoolMetrics {
	return WorkerPoolMetrics{
		JobsEnqueued:     atomic.Int64{},
		JobsProcessed:    atomic.Int64{},
		JobsFailed:       atomic.Int64{},
		JobsTimeout:      atomic.Int64{},
		JobsDropped:      atomic.Int64{},
		ActiveWorkers:    atomic.Int32{},
		QueueDepth:       atomic.Int32{},
		TotalProcessTime: atomic.Int64{},
		TotalQueueTime:   atomic.Int64{},
	}
}

// GetStats returns human-readable statistics
func (wp *WorkerPool) GetStats() map[string]interface{} {
	enqueued := wp.metrics.JobsEnqueued.Load()
	processed := wp.metrics.JobsProcessed.Load()
	failed := wp.metrics.JobsFailed.Load()
	timeout := wp.metrics.JobsTimeout.Load()
	dropped := wp.metrics.JobsDropped.Load()
	
	var avgProcessTime, avgQueueTime time.Duration
	if processed > 0 {
		avgProcessTime = time.Duration(wp.metrics.TotalProcessTime.Load() / processed)
		avgQueueTime = time.Duration(wp.metrics.TotalQueueTime.Load() / processed)
	}
	
	return map[string]interface{}{
		"worker_count":       wp.workerCount,
		"queue_size":         wp.queueSize,
		"active_workers":     wp.metrics.ActiveWorkers.Load(),
		"queue_depth":        wp.metrics.QueueDepth.Load(),
		"jobs_enqueued":      enqueued,
		"jobs_processed":     processed,
		"jobs_failed":        failed,
		"jobs_timeout":       timeout,
		"jobs_dropped":       dropped,
		"success_rate":       float64(processed) / float64(enqueued+1) * 100,
		"avg_process_time":   avgProcessTime.String(),
		"avg_queue_time":     avgQueueTime.String(),
		"queue_utilization":  float64(wp.metrics.QueueDepth.Load()) / float64(wp.queueSize) * 100,
	}
}

// IsHealthy returns true if the worker pool is healthy
func (wp *WorkerPool) IsHealthy() bool {
	if !wp.started.Load() || wp.stopped.Load() {
		return false
	}
	
	// Check queue utilization
	queueUtil := float64(wp.metrics.QueueDepth.Load()) / float64(wp.queueSize)
	if queueUtil > 0.9 {
		return false // Queue is >90% full
	}
	
	// Check if workers are active
	if wp.metrics.ActiveWorkers.Load() == 0 && wp.metrics.QueueDepth.Load() > 0 {
		return false // Jobs queued but no workers active
	}
	
	return true
}
