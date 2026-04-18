# Complete Concurrency System - Implementation Guide

## Overview

This document provides a comprehensive guide to the fully implemented concurrency system for the Kiro Gateway, covering all 6 phases of implementation.

## ✅ All Phases Complete

### Phase 1: Worker Pool ✅
### Phase 2: Priority Queue ✅
### Phase 3: Circuit Breaker ✅
### Phase 4: Connection Pool Monitoring ✅
### Phase 5: Load Shedding ✅
### Phase 6: Async Request Processing ✅

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Rate Limiter (Per-User)                      │
│                   Token Bucket: 10 RPS, 50 Burst                │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Load Shedder                             │
│  Monitors: Queue (80%), Memory (80%), CPU (90%), Workers (90%)  │
│  Actions: Shed Low Priority → Shed Normal Priority              │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Priority Queue (3 Levels)                  │
│  ┌──────────────┬──────────────┬──────────────┐                │
│  │ High (200)   │ Normal (500) │ Low (300)    │                │
│  │ Premium      │ Standard     │ Batch        │                │
│  └──────────────┴──────────────┴──────────────┘                │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Worker Pool (20 Workers)                   │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ... ┌────┐               │
│  │ W1 │ │ W2 │ │ W3 │ │ W4 │ │ W5 │     │W20 │               │
│  └────┘ └────┘ └────┘ └────┘ └────┘     └────┘               │
│  Queue: 1000 jobs, Timeout: 60s                                │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Circuit Breaker                            │
│  States: Closed → Open (5 failures) → Half-Open (3 tests)      │
│  Timeout: 30s, Auto-recovery testing                            │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Connection Pool Monitor                       │
│  Max Idle: 200, Per Host: 50, Max Per Host: 100                │
│  Health Checks: 30s interval, Metrics tracking                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      AWS Q Developer API                        │
└─────────────────────────────────────────────────────────────────┘

                    ┌──────────────────────┐
                    │  Async Job Manager   │
                    │  (Background Jobs)   │
                    │  TTL: 24h            │
                    │  Webhooks: 10s       │
                    └──────────────────────┘
```

## Component Details

### 1. Worker Pool

**File**: `internal/concurrency/worker_pool.go`

**Purpose**: Limit concurrent requests to AWS Q API

**Configuration**:
```go
WorkerPoolConfig{
    WorkerCount: 20,        // Number of concurrent workers
    QueueSize:   1000,      // Job queue buffer size
    JobTimeout:  60s,       // Per-job timeout
    Processor:   jobFunc,   // Job processing function
}
```

**Metrics**:
- Jobs enqueued/processed/failed/timeout/dropped
- Active workers, queue depth
- Average processing time, queue time
- Queue utilization, success rate

**Health Checks**:
- Queue utilization < 90%
- Workers active when jobs queued

### 2. Priority Queue

**File**: `internal/concurrency/priority_queue.go`

**Purpose**: Fair resource allocation based on user priority

**Priority Levels**:
- **High (0)**: Premium users, urgent requests (200 capacity)
- **Normal (1)**: Standard users (500 capacity)
- **Low (2)**: Batch processing, background tasks (300 capacity)

**Configuration**:
```go
PriorityQueueConfig{
    HighPrioritySize:   200,
    NormalPrioritySize: 500,
    LowPrioritySize:    300,
    WorkerPool:         workerPool,
}
```

**Metrics**:
- Per-priority: enqueued, dequeued, dropped, wait time
- Queue depth, utilization per priority
- Total drop rate

**Health Checks**:
- No queue >90% full

### 3. Circuit Breaker

**File**: `internal/concurrency/circuit_breaker.go`

**Purpose**: Protect AWS Q API from cascading failures

**States**:
- **Closed**: Normal operation
- **Open**: Failing, reject requests (after 5 failures)
- **Half-Open**: Testing recovery (3 test requests)

**Configuration**:
```go
CircuitBreakerConfig{
    MaxFailures: 5,         // Failures before opening
    Timeout:     30s,       // Time before half-open
    HalfOpenMax: 3,         // Test requests in half-open
}
```

**Metrics**:
- Total/successful/failed/rejected requests
- State transitions, time in each state
- Success rate

**Health Checks**:
- State is Closed or Half-Open

### 4. Connection Pool Monitor

**File**: `internal/concurrency/connection_pool.go`

**Purpose**: Monitor HTTP connection pool health and performance

**Configuration**:
```go
ConnectionPoolConfig{
    MaxIdleConns:          200,
    MaxIdleConnsPerHost:   50,
    MaxConnsPerHost:       100,
    IdleConnTimeout:       120s,
    ResponseHeaderTimeout: 30s,
    ExpectContinueTimeout: 1s,
    DialTimeout:           10s,
    KeepAlive:             30s,
    HealthCheckInterval:   30s,
}
```

**Metrics**:
- Active/idle/total connections
- Connections created/closed/reused/failed
- Pool exhaustion count, wait time
- Dial/idle/response timeouts
- Health checks passed/failed
- Connection reuse rate

**Health Checks**:
- Pool utilization < 90%
- No recent exhaustion events

### 5. Load Shedder

**File**: `internal/concurrency/load_shedder.go`

**Purpose**: Protect system under high load with priority-based shedding

**Triggers**:
- Queue depth > 80%
- Memory usage > 80%
- CPU usage > 90%
- Worker utilization > 90%
- Circuit breaker open

**Shedding Strategy**:
1. **Low Load**: Allow all requests
2. **High Load**: Shed low-priority requests
3. **Very High Load**: Shed low + normal priority requests
4. **Always Allow**: High-priority requests

**Configuration**:
```go
LoadShedderConfig{
    Enabled:                 true,
    QueueThreshold:          0.8,   // 80%
    MemoryThreshold:         0.8,   // 80%
    CPUThreshold:            0.9,   // 90%
    WorkerThreshold:         0.9,   // 90%
    CircuitBreakerThreshold: true,
    CheckInterval:           5s,
}
```

**Metrics**:
- Shedding events, recovery events
- Requests shed (total and per priority)
- Trigger counts (queue, memory, CPU, worker, circuit)
- Time in shedding, retry-after duration

**Health Checks**:
- Not currently shedding

### 6. Async Job Manager

**File**: `internal/async/job_manager.go`

**Purpose**: Handle long-running requests asynchronously with webhooks

**Features**:
- Job submission and status polling
- Webhook callbacks on completion
- Job TTL and automatic cleanup
- Priority-based processing

**Configuration**:
```go
AsyncJobManagerConfig{
    JobTTL:          24h,   // Job retention time
    WebhookTimeout:  10s,   // Webhook timeout
    CleanupInterval: 1h,    // Cleanup frequency
    PriorityQueue:   priorityQueue,
}
```

**Job Lifecycle**:
```
Pending → Queued → Running → Completed/Failed/Cancelled
                              ↓
                         Webhook (if configured)
                              ↓
                         Expired (after TTL)
```

**Metrics**:
- Jobs created/completed/failed/cancelled/expired
- Webhooks success/failed
- Jobs by status (pending, queued, running, completed, failed)

**Health Checks**:
- Pending jobs < 1000

## Environment Variables

```bash
# Worker Pool
WORKER_POOL_SIZE=20
WORKER_QUEUE_SIZE=1000
WORKER_TIMEOUT=60s

# Priority Queue
PRIORITY_QUEUE_HIGH_SIZE=200
PRIORITY_QUEUE_NORMAL_SIZE=500
PRIORITY_QUEUE_LOW_SIZE=300

# Circuit Breaker
CIRCUIT_BREAKER_MAX_FAILURES=5
CIRCUIT_BREAKER_TIMEOUT=30s
CIRCUIT_BREAKER_HALF_OPEN_MAX=3

# Connection Pool
MAX_IDLE_CONNS=200
MAX_IDLE_CONNS_PER_HOST=50
MAX_CONNS_PER_HOST=100
IDLE_CONN_TIMEOUT=120s
RESPONSE_HEADER_TIMEOUT=30s
DIAL_TIMEOUT=10s
HEALTH_CHECK_INTERVAL=30s

# Load Shedding
LOAD_SHEDDING_ENABLED=true
LOAD_SHEDDING_QUEUE_THRESHOLD=0.8
LOAD_SHEDDING_MEMORY_THRESHOLD=0.8
LOAD_SHEDDING_CPU_THRESHOLD=0.9
LOAD_SHEDDING_WORKER_THRESHOLD=0.9
LOAD_SHEDDING_CHECK_INTERVAL=5s

# Async Processing
ASYNC_ENABLED=true
ASYNC_JOB_TTL=24h
ASYNC_WEBHOOK_TIMEOUT=10s
ASYNC_CLEANUP_INTERVAL=1h
```

## Integration Example

### main.go

```go
func main() {
    // Load configuration
    cfg := config.Load()
    
    // 1. Create circuit breaker
    circuitBreaker := concurrency.NewCircuitBreaker(concurrency.CircuitBreakerConfig{
        MaxFailures: getIntEnv("CIRCUIT_BREAKER_MAX_FAILURES", 5),
        Timeout:     getDurationEnv("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),
        HalfOpenMax: getIntEnv("CIRCUIT_BREAKER_HALF_OPEN_MAX", 3),
    })
    
    // 2. Create connection pool monitor
    connMonitor := concurrency.NewConnectionPoolMonitor(concurrency.ConnectionPoolConfig{
        MaxIdleConns:          getIntEnv("MAX_IDLE_CONNS", 200),
        MaxIdleConnsPerHost:   getIntEnv("MAX_IDLE_CONNS_PER_HOST", 50),
        MaxConnsPerHost:       getIntEnv("MAX_CONNS_PER_HOST", 100),
        IdleConnTimeout:       getDurationEnv("IDLE_CONN_TIMEOUT", 120*time.Second),
        ResponseHeaderTimeout: getDurationEnv("RESPONSE_HEADER_TIMEOUT", 30*time.Second),
        HealthCheckInterval:   getDurationEnv("HEALTH_CHECK_INTERVAL", 30*time.Second),
    })
    
    // 3. Create worker pool
    workerPool := concurrency.NewWorkerPool(concurrency.WorkerPoolConfig{
        WorkerCount: getIntEnv("WORKER_POOL_SIZE", 20),
        QueueSize:   getIntEnv("WORKER_QUEUE_SIZE", 1000),
        JobTimeout:  getDurationEnv("WORKER_TIMEOUT", 60*time.Second),
        Processor:   createJobProcessor(authManager, cfg, circuitBreaker, connMonitor),
    })
    
    // 4. Create priority queue
    priorityQueue := concurrency.NewPriorityQueue(concurrency.PriorityQueueConfig{
        HighPrioritySize:   getIntEnv("PRIORITY_QUEUE_HIGH_SIZE", 200),
        NormalPrioritySize: getIntEnv("PRIORITY_QUEUE_NORMAL_SIZE", 500),
        LowPrioritySize:    getIntEnv("PRIORITY_QUEUE_LOW_SIZE", 300),
        WorkerPool:         workerPool,
    })
    
    // 5. Create load shedder
    loadShedder := concurrency.NewLoadShedder(concurrency.LoadShedderConfig{
        Enabled:                 getBoolEnv("LOAD_SHEDDING_ENABLED", true),
        QueueThreshold:          getFloatEnv("LOAD_SHEDDING_QUEUE_THRESHOLD", 0.8),
        MemoryThreshold:         getFloatEnv("LOAD_SHEDDING_MEMORY_THRESHOLD", 0.8),
        CPUThreshold:            getFloatEnv("LOAD_SHEDDING_CPU_THRESHOLD", 0.9),
        WorkerThreshold:         getFloatEnv("LOAD_SHEDDING_WORKER_THRESHOLD", 0.9),
        CircuitBreakerThreshold: true,
        PriorityQueue:           priorityQueue,
        WorkerPool:              workerPool,
        CircuitBreaker:          circuitBreaker,
        ConnectionMonitor:       connMonitor,
    })
    
    // 6. Create async job manager (optional)
    var asyncJobManager *async.AsyncJobManager
    if getBoolEnv("ASYNC_ENABLED", false) {
        asyncJobManager = async.NewAsyncJobManager(async.AsyncJobManagerConfig{
            JobTTL:          getDurationEnv("ASYNC_JOB_TTL", 24*time.Hour),
            WebhookTimeout:  getDurationEnv("ASYNC_WEBHOOK_TIMEOUT", 10*time.Second),
            CleanupInterval: getDurationEnv("ASYNC_CLEANUP_INTERVAL", 1*time.Hour),
            PriorityQueue:   priorityQueue,
        })
    }
    
    // Start all components
    connMonitor.Start()
    workerPool.Start()
    priorityQueue.Start()
    loadShedder.Start()
    if asyncJobManager != nil {
        asyncJobManager.Start()
    }
    
    // Setup HTTP routes with components
    handlers.SetupRoutes(mux, authManager, cfg, priorityQueue, loadShedder, asyncJobManager)
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down...")
    
    // Stop components in reverse order
    if asyncJobManager != nil {
        asyncJobManager.Stop(10 * time.Second)
    }
    loadShedder.Stop(10 * time.Second)
    priorityQueue.Stop(10 * time.Second)
    workerPool.Stop(30 * time.Second)
    connMonitor.Stop(10 * time.Second)
}
```

### handlers/chat.go

```go
func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
    // ... existing validation ...
    
    // Check load shedding
    priority := concurrency.GetPriorityFromUser(userID, metadata)
    if shouldShed, reason := h.loadShedder.ShouldShed(priority); shouldShed {
        retryAfter := h.loadShedder.GetRetryAfter()
        w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
        h.writeErrorWithRequestID(w, http.StatusServiceUnavailable, reason, nil, requestID)
        return
    }
    
    // Create job
    job := concurrency.NewJob(requestID, requestID, userID, priority, &req, r.Context())
    
    // Submit to priority queue
    if err := h.priorityQueue.Enqueue(job); err != nil {
        h.writeErrorWithRequestID(w, http.StatusServiceUnavailable, 
            "Service temporarily unavailable", err, requestID)
        return
    }
    
    // Wait for result
    select {
    case result := <-job.ResultChan:
        if result.Error != nil {
            h.writeErrorWithRequestID(w, http.StatusInternalServerError,
                "Request processing failed", result.Error, requestID)
            return
        }
        
        // Handle response
        resp := result.Response.(*http.Response)
        if req.Stream {
            h.handleStreamingWithRequestID(w, r, resp, modelID, &req, reqCtx)
        } else {
            h.handleNonStreamingWithRequestID(w, r, resp, modelID, &req, reqCtx)
        }
        
    case <-r.Context().Done():
        h.writeErrorWithRequestID(w, http.StatusRequestTimeout,
            "Request cancelled", r.Context().Err(), requestID)
        return
    }
}
```

### handlers/metrics.go

```go
func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
    metrics := map[string]interface{}{
        "worker_pool":        h.workerPool.GetStats(),
        "priority_queue":     h.priorityQueue.GetStats(),
        "circuit_breaker":    h.circuitBreaker.GetStats(),
        "connection_pool":    h.connectionMonitor.GetStats(),
        "load_shedder":       h.loadShedder.GetStats(),
    }
    
    if h.asyncJobManager != nil {
        metrics["async_jobs"] = h.asyncJobManager.GetStats()
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(metrics)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "worker_pool":        h.workerPool.IsHealthy(),
        "priority_queue":     h.priorityQueue.IsHealthy(),
        "circuit_breaker":    h.circuitBreaker.IsHealthy(),
        "connection_pool":    h.connectionMonitor.IsHealthy(),
        "load_shedder":       h.loadShedder.IsHealthy(),
    }
    
    if h.asyncJobManager != nil {
        health["async_jobs"] = h.asyncJobManager.IsHealthy()
    }
    
    // Overall health
    allHealthy := true
    for _, healthy := range health {
        if !healthy.(bool) {
            allHealthy = false
            break
        }
    }
    
    status := http.StatusOK
    if !allHealthy {
        status = http.StatusServiceUnavailable
    }
    
    health["status"] = allHealthy
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(health)
}
```

## Testing

### Load Testing

```bash
# Test with increasing concurrency
for i in {10..100..10}; do
    echo "Testing with $i concurrent requests"
    ab -n 1000 -c $i http://localhost:8080/v1/chat/completions
    sleep 5
done
```

### Stress Testing

```bash
# Monitor metrics during stress test
watch -n 1 'curl -s http://localhost:8080/metrics | jq .'

# Run stress test
ab -n 10000 -c 200 http://localhost:8080/v1/chat/completions
```

### Health Monitoring

```bash
# Check health endpoint
curl http://localhost:8080/health | jq .

# Monitor specific components
curl http://localhost:8080/metrics | jq '.worker_pool'
curl http://localhost:8080/metrics | jq '.priority_queue'
curl http://localhost:8080/metrics | jq '.circuit_breaker'
curl http://localhost:8080/metrics | jq '.connection_pool'
curl http://localhost:8080/metrics | jq '.load_shedder'
```

## Benefits Summary

### 1. Controlled Concurrency ✅
- Worker pool limits concurrent AWS API calls
- Prevents API rate limit violations
- Configurable worker count

### 2. Fair Resource Allocation ✅
- Priority-based queuing
- Premium users get faster processing
- Batch jobs don't block interactive requests

### 3. Fault Tolerance ✅
- Circuit breaker protects against cascading failures
- Automatic recovery testing
- Graceful degradation

### 4. Connection Management ✅
- Enhanced connection pooling
- Health monitoring and metrics
- Pool exhaustion detection

### 5. Load Protection ✅
- Automatic load shedding under high load
- Priority-based request rejection
- Retry-After headers for clients

### 6. Async Processing ✅
- Long-running task support
- Webhook callbacks
- Job TTL and cleanup

### 7. Observability ✅
- Comprehensive metrics for all components
- Health checks
- Performance tracking

## Conclusion

The complete concurrency system is now implemented with all 6 phases:

1. ✅ Worker Pool - Controlled concurrency
2. ✅ Priority Queue - Fair resource allocation
3. ✅ Circuit Breaker - Fault tolerance
4. ✅ Connection Pool Monitoring - Connection management
5. ✅ Load Shedding - Load protection
6. ✅ Async Request Processing - Long-running tasks

The system is production-ready and can handle:
- Multiple concurrent connections
- Multiple users with different priorities
- Async and long-running requests
- Request queuing and prioritization
- Automatic load shedding
- Fault tolerance and recovery
- Comprehensive monitoring and health checks

All components work together to provide a robust, scalable, and observable gateway for AWS Q Developer API.
