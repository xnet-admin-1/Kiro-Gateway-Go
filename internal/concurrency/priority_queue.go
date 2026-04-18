package concurrency

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// PriorityQueue manages multiple queues with different priority levels
type PriorityQueue struct {
	// Queues for each priority level
	queues [3]chan *Job
	
	// Configuration
	queueSizes [3]int
	
	// State
	started atomic.Bool
	stopped atomic.Bool
	
	// Metrics
	metrics *PriorityQueueMetrics
	
	// Worker pool
	workerPool *WorkerPool
	
	// Context
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// PriorityQueueMetrics tracks priority queue performance
type PriorityQueueMetrics struct {
	// Per-priority counters
	Enqueued [3]atomic.Int64
	Dequeued [3]atomic.Int64
	Dropped  [3]atomic.Int64
	
	// Per-priority gauges
	QueueDepth [3]atomic.Int32
	
	// Timing
	TotalWaitTime [3]atomic.Int64 // nanoseconds
	
	mu sync.RWMutex
}

// PriorityQueueConfig holds priority queue configuration
type PriorityQueueConfig struct {
	HighPrioritySize   int
	NormalPrioritySize int
	LowPrioritySize    int
	WorkerPool         *WorkerPool
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(cfg PriorityQueueConfig) *PriorityQueue {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults
	if cfg.HighPrioritySize <= 0 {
		cfg.HighPrioritySize = 200
	}
	if cfg.NormalPrioritySize <= 0 {
		cfg.NormalPrioritySize = 500
	}
	if cfg.LowPrioritySize <= 0 {
		cfg.LowPrioritySize = 300
	}
	
	pq := &PriorityQueue{
		queueSizes: [3]int{
			cfg.HighPrioritySize,
			cfg.NormalPrioritySize,
			cfg.LowPrioritySize,
		},
		metrics:    &PriorityQueueMetrics{},
		workerPool: cfg.WorkerPool,
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Initialize queues
	pq.queues[PriorityHigh] = make(chan *Job, cfg.HighPrioritySize)
	pq.queues[PriorityNormal] = make(chan *Job, cfg.NormalPrioritySize)
	pq.queues[PriorityLow] = make(chan *Job, cfg.LowPrioritySize)
	
	return pq
}

// Start starts the priority queue dispatcher
func (pq *PriorityQueue) Start() error {
	if pq.started.Load() {
		return fmt.Errorf("priority queue already started")
	}
	
	if pq.workerPool == nil {
		return fmt.Errorf("worker pool not set")
	}
	
	pq.started.Store(true)
	
	// Start dispatcher
	pq.wg.Add(1)
	go pq.dispatcher()
	
	log.Printf("Priority queue started (high: %d, normal: %d, low: %d)",
		pq.queueSizes[PriorityHigh],
		pq.queueSizes[PriorityNormal],
		pq.queueSizes[PriorityLow])
	
	return nil
}

// Stop stops the priority queue gracefully
func (pq *PriorityQueue) Stop(timeout time.Duration) error {
	if pq.stopped.Load() {
		return fmt.Errorf("priority queue already stopped")
	}
	
	log.Println("Stopping priority queue...")
	
	pq.stopped.Store(true)
	pq.cancel()
	
	// Wait for dispatcher to finish with timeout
	done := make(chan struct{})
	go func() {
		pq.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("Priority queue stopped gracefully")
		return nil
	case <-time.After(timeout):
		log.Println("Priority queue stopped with timeout")
		return fmt.Errorf("priority queue stop timeout after %v", timeout)
	}
}

// Enqueue adds a job to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(job *Job) error {
	if pq.stopped.Load() {
		return fmt.Errorf("priority queue is stopped")
	}
	
	if !pq.started.Load() {
		return fmt.Errorf("priority queue not started")
	}
	
	// Validate priority
	if job.Priority < PriorityHigh || job.Priority > PriorityLow {
		return fmt.Errorf("invalid priority: %d", job.Priority)
	}
	
	// Try to enqueue job
	select {
	case pq.queues[job.Priority] <- job:
		pq.metrics.Enqueued[job.Priority].Add(1)
		pq.metrics.QueueDepth[job.Priority].Add(1)
		log.Printf("Job %s enqueued (priority: %s, depth: %d)",
			job.ID, GetPriorityName(job.Priority), pq.metrics.QueueDepth[job.Priority].Load())
		return nil
	default:
		// Queue is full, drop job
		pq.metrics.Dropped[job.Priority].Add(1)
		return fmt.Errorf("priority queue is full (priority: %s, size: %d)",
			GetPriorityName(job.Priority), pq.queueSizes[job.Priority])
	}
}

// EnqueueWithTimeout adds a job with a timeout
func (pq *PriorityQueue) EnqueueWithTimeout(job *Job, timeout time.Duration) error {
	if pq.stopped.Load() {
		return fmt.Errorf("priority queue is stopped")
	}
	
	if !pq.started.Load() {
		return fmt.Errorf("priority queue not started")
	}
	
	// Validate priority
	if job.Priority < PriorityHigh || job.Priority > PriorityLow {
		return fmt.Errorf("invalid priority: %d", job.Priority)
	}
	
	// Try to enqueue job with timeout
	select {
	case pq.queues[job.Priority] <- job:
		pq.metrics.Enqueued[job.Priority].Add(1)
		pq.metrics.QueueDepth[job.Priority].Add(1)
		return nil
	case <-time.After(timeout):
		pq.metrics.Dropped[job.Priority].Add(1)
		return fmt.Errorf("enqueue timeout after %v (priority: %s)",
			timeout, GetPriorityName(job.Priority))
	}
}

// dispatcher dispatches jobs from priority queues to worker pool
func (pq *PriorityQueue) dispatcher() {
	defer pq.wg.Done()
	
	log.Println("Priority queue dispatcher started")
	
	// Dispatch loop with priority-based selection
	for {
		select {
		case <-pq.ctx.Done():
			log.Println("Priority queue dispatcher stopping")
			return
			
		// High priority (check first)
		case job := <-pq.queues[PriorityHigh]:
			pq.dispatchJob(job, PriorityHigh)
			
		// Normal priority (check second)
		case job := <-pq.queues[PriorityNormal]:
			pq.dispatchJob(job, PriorityNormal)
			
		// Low priority (check last)
		case job := <-pq.queues[PriorityLow]:
			pq.dispatchJob(job, PriorityLow)
		}
	}
}

// dispatchJob dispatches a single job to the worker pool
func (pq *PriorityQueue) dispatchJob(job *Job, priority int) {
	pq.metrics.QueueDepth[priority].Add(-1)
	pq.metrics.Dequeued[priority].Add(1)
	
	// Calculate wait time
	waitTime := time.Since(job.CreatedAt)
	pq.metrics.TotalWaitTime[priority].Add(waitTime.Nanoseconds())
	
	log.Printf("Dispatching job %s (priority: %s, wait time: %v)",
		job.ID, GetPriorityName(priority), waitTime)
	
	// Submit to worker pool
	if err := pq.workerPool.Submit(job); err != nil {
		log.Printf("Failed to submit job %s to worker pool: %v", job.ID, err)
		
		// Send error result
		result := &Result{
			JobID:     job.ID,
			RequestID: job.RequestID,
			Error:     fmt.Errorf("failed to submit to worker pool: %w", err),
			Status:    JobStatusFailed,
			QueueTime: waitTime,
		}
		
		select {
		case job.ResultChan <- result:
		default:
			log.Printf("Warning: Could not send error result for job %s", job.ID)
		}
	}
}

// GetStats returns human-readable statistics
func (pq *PriorityQueue) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	for priority := PriorityHigh; priority <= PriorityLow; priority++ {
		priorityName := GetPriorityName(priority)
		
		enqueued := pq.metrics.Enqueued[priority].Load()
		dequeued := pq.metrics.Dequeued[priority].Load()
		dropped := pq.metrics.Dropped[priority].Load()
		depth := pq.metrics.QueueDepth[priority].Load()
		
		var avgWaitTime time.Duration
		if dequeued > 0 {
			avgWaitTime = time.Duration(pq.metrics.TotalWaitTime[priority].Load() / dequeued)
		}
		
		stats[priorityName] = map[string]interface{}{
			"queue_size":       pq.queueSizes[priority],
			"queue_depth":      depth,
			"enqueued":         enqueued,
			"dequeued":         dequeued,
			"dropped":          dropped,
			"avg_wait_time":    avgWaitTime.String(),
			"utilization":      float64(depth) / float64(pq.queueSizes[priority]) * 100,
		}
	}
	
	// Overall stats
	totalEnqueued := pq.metrics.Enqueued[0].Load() + pq.metrics.Enqueued[1].Load() + pq.metrics.Enqueued[2].Load()
	totalDequeued := pq.metrics.Dequeued[0].Load() + pq.metrics.Dequeued[1].Load() + pq.metrics.Dequeued[2].Load()
	totalDropped := pq.metrics.Dropped[0].Load() + pq.metrics.Dropped[1].Load() + pq.metrics.Dropped[2].Load()
	totalDepth := pq.metrics.QueueDepth[0].Load() + pq.metrics.QueueDepth[1].Load() + pq.metrics.QueueDepth[2].Load()
	
	stats["total"] = map[string]interface{}{
		"enqueued":    totalEnqueued,
		"dequeued":    totalDequeued,
		"dropped":     totalDropped,
		"queue_depth": totalDepth,
		"drop_rate":   float64(totalDropped) / float64(totalEnqueued+1) * 100,
	}
	
	return stats
}

// IsHealthy returns true if the priority queue is healthy
func (pq *PriorityQueue) IsHealthy() bool {
	if !pq.started.Load() || pq.stopped.Load() {
		return false
	}
	
	// Check if any queue is >90% full
	for priority := PriorityHigh; priority <= PriorityLow; priority++ {
		depth := pq.metrics.QueueDepth[priority].Load()
		size := pq.queueSizes[priority]
		utilization := float64(depth) / float64(size)
		
		if utilization > 0.9 {
			log.Printf("Warning: %s priority queue is %.1f%% full",
				GetPriorityName(priority), utilization*100)
			return false
		}
	}
	
	return true
}

// GetQueueDepth returns the current depth of a specific priority queue
func (pq *PriorityQueue) GetQueueDepth(priority int) int32 {
	if priority < PriorityHigh || priority > PriorityLow {
		return 0
	}
	return pq.metrics.QueueDepth[priority].Load()
}

// GetTotalQueueDepth returns the total depth across all priority queues
func (pq *PriorityQueue) GetTotalQueueDepth() int32 {
	return pq.metrics.QueueDepth[0].Load() +
		pq.metrics.QueueDepth[1].Load() +
		pq.metrics.QueueDepth[2].Load()
}
