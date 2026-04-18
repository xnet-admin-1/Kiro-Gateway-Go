package concurrency

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// LoadShedder implements load shedding to protect the system under high load
type LoadShedder struct {
	// Configuration
	enabled                bool
	queueThreshold         float64 // 0.0-1.0
	memoryThreshold        float64 // 0.0-1.0
	cpuThreshold           float64 // 0.0-1.0
	workerThreshold        float64 // 0.0-1.0
	circuitBreakerThreshold bool
	
	// Components
	priorityQueue     *PriorityQueue
	workerPool        *WorkerPool
	circuitBreaker    *CircuitBreaker
	connectionMonitor *ConnectionPoolMonitor
	
	// Metrics
	metrics *LoadShedderMetrics
	
	// State
	started atomic.Bool
	stopped atomic.Bool
	shedding atomic.Bool
	
	// Context
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	mu sync.RWMutex
}

// LoadShedderMetrics tracks load shedding performance
type LoadShedderMetrics struct {
	// Shedding events
	SheddingEvents      atomic.Int64
	RequestsShed        atomic.Int64
	RequestsShedByPriority [3]atomic.Int64
	
	// Triggers
	QueueTriggers       atomic.Int64
	MemoryTriggers      atomic.Int64
	CPUTriggers         atomic.Int64
	WorkerTriggers      atomic.Int64
	CircuitTriggers     atomic.Int64
	
	// Recovery
	RecoveryEvents      atomic.Int64
	TimeInShedding      atomic.Int64 // nanoseconds
	LastSheddingStart   atomic.Int64 // Unix nano
	
	mu sync.RWMutex
}

// LoadShedderConfig holds load shedder configuration
type LoadShedderConfig struct {
	Enabled                 bool
	QueueThreshold          float64 // 0.8 = 80% queue depth
	MemoryThreshold         float64 // 0.8 = 80% memory usage
	CPUThreshold            float64 // 0.9 = 90% CPU usage
	WorkerThreshold         float64 // 0.9 = 90% worker utilization
	CircuitBreakerThreshold bool    // Shed when circuit breaker is open
	CheckInterval           time.Duration
	PriorityQueue           *PriorityQueue
	WorkerPool              *WorkerPool
	CircuitBreaker          *CircuitBreaker
	ConnectionMonitor       *ConnectionPoolMonitor
}

// NewLoadShedder creates a new load shedder
func NewLoadShedder(cfg LoadShedderConfig) *LoadShedder {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set defaults
	if cfg.QueueThreshold <= 0 {
		cfg.QueueThreshold = 0.8
	}
	if cfg.MemoryThreshold <= 0 {
		cfg.MemoryThreshold = 0.8
	}
	if cfg.CPUThreshold <= 0 {
		cfg.CPUThreshold = 0.9
	}
	if cfg.WorkerThreshold <= 0 {
		cfg.WorkerThreshold = 0.9
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 5 * time.Second
	}
	
	ls := &LoadShedder{
		enabled:                 cfg.Enabled,
		queueThreshold:          cfg.QueueThreshold,
		memoryThreshold:         cfg.MemoryThreshold,
		cpuThreshold:            cfg.CPUThreshold,
		workerThreshold:         cfg.WorkerThreshold,
		circuitBreakerThreshold: cfg.CircuitBreakerThreshold,
		priorityQueue:           cfg.PriorityQueue,
		workerPool:              cfg.WorkerPool,
		circuitBreaker:          cfg.CircuitBreaker,
		connectionMonitor:       cfg.ConnectionMonitor,
		metrics:                 &LoadShedderMetrics{},
		ctx:                     ctx,
		cancel:                  cancel,
	}
	
	log.Printf("Load shedder initialized (enabled: %v, queue: %.0f%%, memory: %.0f%%, cpu: %.0f%%, worker: %.0f%%)",
		cfg.Enabled, cfg.QueueThreshold*100, cfg.MemoryThreshold*100, 
		cfg.CPUThreshold*100, cfg.WorkerThreshold*100)
	
	return ls
}

// Start starts the load shedder
func (ls *LoadShedder) Start() error {
	if !ls.enabled {
		log.Println("Load shedder is disabled")
		return nil
	}
	
	if ls.started.Load() {
		return fmt.Errorf("load shedder already started")
	}
	
	ls.started.Store(true)
	
	// Start monitoring loop
	ls.wg.Add(1)
	go ls.monitorLoop()
	
	log.Println("Load shedder started")
	
	return nil
}

// Stop stops the load shedder
func (ls *LoadShedder) Stop(timeout time.Duration) error {
	if !ls.enabled {
		return nil
	}
	
	if ls.stopped.Load() {
		return fmt.Errorf("load shedder already stopped")
	}
	
	log.Println("Stopping load shedder...")
	
	ls.stopped.Store(true)
	ls.cancel()
	
	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		ls.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("Load shedder stopped gracefully")
		return nil
	case <-time.After(timeout):
		log.Println("Load shedder stopped with timeout")
		return fmt.Errorf("load shedder stop timeout after %v", timeout)
	}
}

// ShouldShed determines if a request should be shed based on priority
func (ls *LoadShedder) ShouldShed(priority int) (bool, string) {
	if !ls.enabled {
		return false, ""
	}
	
	if !ls.shedding.Load() {
		return false, ""
	}
	
	// Always allow high priority requests
	if priority == PriorityHigh {
		return false, ""
	}
	
	// Shed low priority requests first
	if priority == PriorityLow {
		ls.metrics.RequestsShed.Add(1)
		ls.metrics.RequestsShedByPriority[PriorityLow].Add(1)
		return true, "system under high load (low priority requests shed)"
	}
	
	// Shed normal priority requests if load is very high
	if ls.isVeryHighLoad() {
		ls.metrics.RequestsShed.Add(1)
		ls.metrics.RequestsShedByPriority[PriorityNormal].Add(1)
		return true, "system under very high load (normal priority requests shed)"
	}
	
	return false, ""
}

// monitorLoop monitors system load and triggers shedding
func (ls *LoadShedder) monitorLoop() {
	defer ls.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ls.ctx.Done():
			return
		case <-ticker.C:
			ls.checkLoad()
		}
	}
}

// checkLoad checks system load and updates shedding state
func (ls *LoadShedder) checkLoad() {
	wasShedding := ls.shedding.Load()
	shouldShed := false
	triggers := []string{}
	
	// Check queue depth
	if ls.priorityQueue != nil {
		queueDepth := ls.priorityQueue.GetTotalQueueDepth()
		totalCapacity := int32(ls.priorityQueue.queueSizes[0] + 
			ls.priorityQueue.queueSizes[1] + 
			ls.priorityQueue.queueSizes[2])
		queueUtil := float64(queueDepth) / float64(totalCapacity)
		
		if queueUtil > ls.queueThreshold {
			shouldShed = true
			triggers = append(triggers, fmt.Sprintf("queue %.0f%%", queueUtil*100))
			ls.metrics.QueueTriggers.Add(1)
		}
	}
	
	// Check worker pool utilization
	if ls.workerPool != nil {
		activeWorkers := ls.workerPool.metrics.ActiveWorkers.Load()
		totalWorkers := int32(ls.workerPool.workerCount)
		workerUtil := float64(activeWorkers) / float64(totalWorkers)
		
		if workerUtil > ls.workerThreshold {
			shouldShed = true
			triggers = append(triggers, fmt.Sprintf("workers %.0f%%", workerUtil*100))
			ls.metrics.WorkerTriggers.Add(1)
		}
	}
	
	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUtil := float64(m.Alloc) / float64(m.Sys)
	
	if memoryUtil > ls.memoryThreshold {
		shouldShed = true
		triggers = append(triggers, fmt.Sprintf("memory %.0f%%", memoryUtil*100))
		ls.metrics.MemoryTriggers.Add(1)
	}
	
	// Check circuit breaker
	if ls.circuitBreakerThreshold && ls.circuitBreaker != nil {
		if ls.circuitBreaker.IsOpen() {
			shouldShed = true
			triggers = append(triggers, "circuit breaker open")
			ls.metrics.CircuitTriggers.Add(1)
		}
	}
	
	// Update shedding state
	if shouldShed && !wasShedding {
		// Start shedding
		ls.shedding.Store(true)
		ls.metrics.SheddingEvents.Add(1)
		ls.metrics.LastSheddingStart.Store(time.Now().UnixNano())
		log.Printf("[WARNING] Load shedding STARTED (triggers: %v)", triggers)
	} else if !shouldShed && wasShedding {
		// Stop shedding
		ls.shedding.Store(false)
		ls.metrics.RecoveryEvents.Add(1)
		
		// Update time in shedding
		sheddingStart := time.Unix(0, ls.metrics.LastSheddingStart.Load())
		duration := time.Since(sheddingStart)
		ls.metrics.TimeInShedding.Add(duration.Nanoseconds())
		
		log.Printf("[SUCCESS] Load shedding STOPPED (duration: %v)", duration)
	} else if shouldShed && wasShedding {
		// Still shedding
		log.Printf("[WARNING] Load shedding ACTIVE (triggers: %v)", triggers)
	}
}

// isVeryHighLoad checks if the system is under very high load
func (ls *LoadShedder) isVeryHighLoad() bool {
	triggerCount := 0
	
	// Check queue depth
	if ls.priorityQueue != nil {
		queueDepth := ls.priorityQueue.GetTotalQueueDepth()
		totalCapacity := int32(ls.priorityQueue.queueSizes[0] + 
			ls.priorityQueue.queueSizes[1] + 
			ls.priorityQueue.queueSizes[2])
		queueUtil := float64(queueDepth) / float64(totalCapacity)
		
		if queueUtil > 0.95 { // >95% queue depth
			triggerCount++
		}
	}
	
	// Check worker pool utilization
	if ls.workerPool != nil {
		activeWorkers := ls.workerPool.metrics.ActiveWorkers.Load()
		totalWorkers := int32(ls.workerPool.workerCount)
		workerUtil := float64(activeWorkers) / float64(totalWorkers)
		
		if workerUtil > 0.95 { // >95% worker utilization
			triggerCount++
		}
	}
	
	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUtil := float64(m.Alloc) / float64(m.Sys)
	
	if memoryUtil > 0.9 { // >90% memory usage
		triggerCount++
	}
	
	// Very high load if 2+ triggers
	return triggerCount >= 2
}

// GetRetryAfter returns the suggested retry-after duration
func (ls *LoadShedder) GetRetryAfter() time.Duration {
	if !ls.shedding.Load() {
		return 0
	}
	
	// Base retry-after on queue depth
	if ls.priorityQueue != nil {
		queueDepth := ls.priorityQueue.GetTotalQueueDepth()
		
		// Estimate: 1 second per 10 queued jobs
		retryAfter := time.Duration(queueDepth/10) * time.Second
		
		// Cap at 60 seconds
		if retryAfter > 60*time.Second {
			retryAfter = 60 * time.Second
		}
		
		// Minimum 5 seconds
		if retryAfter < 5*time.Second {
			retryAfter = 5 * time.Second
		}
		
		return retryAfter
	}
	
	// Default: 10 seconds
	return 10 * time.Second
}

// GetStats returns human-readable statistics
func (ls *LoadShedder) GetStats() map[string]interface{} {
	isShedding := ls.shedding.Load()
	
	var currentSheddingDuration time.Duration
	if isShedding {
		sheddingStart := time.Unix(0, ls.metrics.LastSheddingStart.Load())
		currentSheddingDuration = time.Since(sheddingStart)
	}
	
	totalTimeInShedding := time.Duration(ls.metrics.TimeInShedding.Load())
	if isShedding {
		totalTimeInShedding += currentSheddingDuration
	}
	
	return map[string]interface{}{
		"enabled":     ls.enabled,
		"shedding":    isShedding,
		"configuration": map[string]interface{}{
			"queue_threshold":  fmt.Sprintf("%.0f%%", ls.queueThreshold*100),
			"memory_threshold": fmt.Sprintf("%.0f%%", ls.memoryThreshold*100),
			"cpu_threshold":    fmt.Sprintf("%.0f%%", ls.cpuThreshold*100),
			"worker_threshold": fmt.Sprintf("%.0f%%", ls.workerThreshold*100),
		},
		"events": map[string]interface{}{
			"shedding_events":  ls.metrics.SheddingEvents.Load(),
			"recovery_events":  ls.metrics.RecoveryEvents.Load(),
			"requests_shed":    ls.metrics.RequestsShed.Load(),
			"high_priority_shed":   ls.metrics.RequestsShedByPriority[PriorityHigh].Load(),
			"normal_priority_shed": ls.metrics.RequestsShedByPriority[PriorityNormal].Load(),
			"low_priority_shed":    ls.metrics.RequestsShedByPriority[PriorityLow].Load(),
		},
		"triggers": map[string]interface{}{
			"queue":          ls.metrics.QueueTriggers.Load(),
			"memory":         ls.metrics.MemoryTriggers.Load(),
			"cpu":            ls.metrics.CPUTriggers.Load(),
			"worker":         ls.metrics.WorkerTriggers.Load(),
			"circuit_breaker": ls.metrics.CircuitTriggers.Load(),
		},
		"timing": map[string]interface{}{
			"current_shedding_duration": currentSheddingDuration.String(),
			"total_time_in_shedding":    totalTimeInShedding.String(),
			"retry_after":               ls.GetRetryAfter().String(),
		},
	}
}

// IsHealthy returns true if the load shedder is healthy (not shedding)
func (ls *LoadShedder) IsHealthy() bool {
	if !ls.enabled {
		return true
	}
	
	return !ls.shedding.Load()
}

// IsShedding returns true if the system is currently shedding load
func (ls *LoadShedder) IsShedding() bool {
	return ls.shedding.Load()
}
