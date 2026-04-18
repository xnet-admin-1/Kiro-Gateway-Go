# Concurrency Architecture Design

## Current State Analysis

### What We Have ✅

1. **HTTP Connection Pooling** (`internal/client/client.go`)
   - `MaxIdleConns`: 100 connections
   - `MaxIdleConnsPerHost`: 10 connections per host
   - `IdleConnTimeout`: 90 seconds
   - Basic connection reuse

2. **Per-User Rate Limiting** (`internal/validation/ratelimiter.go`)
   - Token bucket algorithm
   - 10 RPS per user
   - 50 burst capacity
   - Per-user limiters with sync.RWMutex

3. **Monthly Quota Tracking** (`internal/validation/ratelimiter.go`)
   - 10K agentic requests/month
   - 10K inference calls/month
   - Auto-reset per month

4. **Request Retry Logic** (`internal/client/client.go`)
   - Exponential backoff
   - 3 retries max
   - Retry-After header support

5. **Graceful Shutdown** (`cmd/kiro-gateway/main.go`)
   - 30-second timeout
   - Signal handling (SIGINT, SIGTERM)

### What's Missing ❌

1. **Request Queue Management**
   - No queue for handling bursts
   - No priority queuing
   - No queue size limits

2. **Worker Pool Pattern**
   - All requests handled immediately
   - No worker pool for backend requests
   - No concurrency limits per endpoint

3. **Circuit Breaker**
   - No protection against cascading failures
   - No automatic recovery
   - No health checks

4. **Connection Pool Monitoring**
   - No metrics on pool usage
   - No connection health checks
   - No pool exhaustion handling

5. **Async Request Processing**
   - No background job processing
   - No webhook callbacks
   - No long-running task support

6. **Load Shedding**
   - No automatic request rejection under load
   - No priority-based shedding
   - No backpressure mechanism

## Proposed Architecture

### 1. Request Queue System

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Rate Limiter Middleware        │
│  (Per-user token bucket)            │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Priority Queue                 │
│  ┌─────────────────────────────┐   │
│  │ High Priority (Premium)     │   │
│  ├─────────────────────────────┤   │
│  │ Normal Priority (Standard)  │   │
│  ├─────────────────────────────┤   │
│  │ Low Priority (Batch)        │   │
│  └─────────────────────────────┘   │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Worker Pool                    │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐       │
│  │ W1 │ │ W2 │ │ W3 │ │ W4 │ ...   │
│  └────┘ └────┘ └────┘ └────┘       │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Circuit Breaker                │
│  (Protect AWS Q API)                │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      Connection Pool                │
│  (HTTP client with pooling)         │
└──────┬──────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────┐
│      AWS Q Developer API            │
└─────────────────────────────────────┘
```

### 2. Worker Pool Design

**Purpose**: Limit concurrent requests to AWS Q API

```go
type WorkerPool struct {
    workers      int           // Number of workers
    jobQueue     chan *Job     // Buffered channel for jobs
    resultQueue  chan *Result  // Results channel
    wg           sync.WaitGroup
    ctx          context.Context
    cancel       context.CancelFunc
}

type Job struct {
    ID          string
    Priority    int
    Request     *models.ChatCompletionRequest
    Context     context.Context
    ResultChan  chan *Result
}

type Result struct {
    JobID       string
    Response    *http.Response
    Error       error
    Duration    time.Duration
}
```

**Configuration**:
- Workers: 20 (configurable)
- Queue size: 1000 (configurable)
- Timeout: 60s per job

### 3. Priority Queue System

**Priority Levels**:
1. **High (0)**: Premium users, urgent requests
2. **Normal (1)**: Standard users
3. **Low (2)**: Batch processing, background tasks

**Implementation**:
```go
type PriorityQueue struct {
    queues   [3]chan *Job  // 3 priority levels
    mu       sync.RWMutex
    metrics  *QueueMetrics
}

type QueueMetrics struct {
    Enqueued    int64
    Dequeued    int64
    Dropped     int64
    QueueDepth  [3]int64
}
```

### 4. Circuit Breaker Pattern

**States**:
- **Closed**: Normal operation
- **Open**: Failing, reject requests
- **Half-Open**: Testing recovery

**Configuration**:
```go
type CircuitBreaker struct {
    maxFailures     int           // 5 failures
    timeout         time.Duration // 30s timeout
    halfOpenMax     int           // 3 test requests
    state           State
    failures        int
    lastFailTime    time.Time
    mu              sync.RWMutex
}
```

### 5. Connection Pool Enhancement

**Current**:
```go
MaxIdleConns:        100
MaxIdleConnsPerHost: 10
IdleConnTimeout:     90s
```

**Enhanced**:
```go
MaxIdleConns:          200  // Increased
MaxIdleConnsPerHost:   50   // Increased for AWS
MaxConnsPerHost:       100  // New: limit total connections
IdleConnTimeout:       120s // Increased
ResponseHeaderTimeout: 30s  // New: prevent slow headers
ExpectContinueTimeout: 1s   // New: 100-continue timeout
```

### 6. Load Shedding Strategy

**Triggers**:
1. Queue depth > 80% capacity
2. Worker pool saturation > 90%
3. Circuit breaker open
4. Memory usage > 80%

**Actions**:
1. Reject low-priority requests (503)
2. Return cached responses (if available)
3. Suggest retry-after time
4. Log shedding events

### 7. Async Request Processing

**Use Cases**:
- Long-running analysis (>60s)
- Batch processing
- Webhook callbacks

**Flow**:
```
Client → Submit Job → Get Job ID → Poll Status → Get Result
```

**Implementation**:
```go
type AsyncJobManager struct {
    jobs        map[string]*AsyncJob
    mu          sync.RWMutex
    storage     JobStorage
    workerPool  *WorkerPool
}

type AsyncJob struct {
    ID          string
    Status      JobStatus  // pending, running, completed, failed
    Request     *models.ChatCompletionRequest
    Result      *models.ChatCompletionResponse
    Error       error
    CreatedAt   time.Time
    CompletedAt time.Time
    CallbackURL string
}
```

## Implementation Plan

### Phase 1: Worker Pool (Priority: HIGH)
**Files to Create**:
- `internal/concurrency/worker_pool.go`
- `internal/concurrency/job.go`

**Features**:
- Configurable worker count
- Job queue with buffering
- Graceful shutdown
- Metrics collection

### Phase 2: Priority Queue (Priority: HIGH)
**Files to Create**:
- `internal/concurrency/priority_queue.go`
- `internal/concurrency/queue_metrics.go`

**Features**:
- 3-level priority system
- Per-priority queue depth limits
- Queue metrics and monitoring

### Phase 3: Circuit Breaker (Priority: MEDIUM)
**Files to Create**:
- `internal/concurrency/circuit_breaker.go`

**Features**:
- State machine (Closed/Open/Half-Open)
- Configurable thresholds
- Automatic recovery testing
- Metrics and logging

### Phase 4: Enhanced Connection Pool (Priority: MEDIUM)
**Files to Modify**:
- `internal/client/client.go`

**Features**:
- Increased limits
- Connection health checks
- Pool metrics
- Timeout configuration

### Phase 5: Load Shedding (Priority: LOW)
**Files to Create**:
- `internal/concurrency/load_shedder.go`

**Features**:
- Resource monitoring
- Priority-based rejection
- Retry-After headers
- Metrics and alerts

### Phase 6: Async Processing (Priority: LOW)
**Files to Create**:
- `internal/async/job_manager.go`
- `internal/async/job_storage.go`
- `internal/handlers/async.go`

**Features**:
- Job submission endpoint
- Status polling endpoint
- Webhook callbacks
- Job persistence

## Configuration

### Environment Variables

```bash
# Worker Pool
WORKER_POOL_SIZE=20              # Number of workers
WORKER_QUEUE_SIZE=1000           # Job queue buffer size
WORKER_TIMEOUT=60s               # Job timeout

# Priority Queue
PRIORITY_QUEUE_HIGH_SIZE=200     # High priority queue size
PRIORITY_QUEUE_NORMAL_SIZE=500   # Normal priority queue size
PRIORITY_QUEUE_LOW_SIZE=300      # Low priority queue size

# Circuit Breaker
CIRCUIT_BREAKER_MAX_FAILURES=5   # Failures before opening
CIRCUIT_BREAKER_TIMEOUT=30s      # Timeout before half-open
CIRCUIT_BREAKER_HALF_OPEN_MAX=3  # Test requests in half-open

# Connection Pool
MAX_IDLE_CONNS=200               # Total idle connections
MAX_IDLE_CONNS_PER_HOST=50       # Idle per host
MAX_CONNS_PER_HOST=100           # Total per host
IDLE_CONN_TIMEOUT=120s           # Idle timeout
RESPONSE_HEADER_TIMEOUT=30s      # Header timeout

# Load Shedding
LOAD_SHEDDING_ENABLED=true       # Enable load shedding
LOAD_SHEDDING_THRESHOLD=0.8      # Queue depth threshold
LOAD_SHEDDING_MEMORY_LIMIT=0.8   # Memory usage threshold

# Async Processing
ASYNC_ENABLED=false              # Enable async endpoints
ASYNC_JOB_TTL=24h                # Job retention time
ASYNC_WEBHOOK_TIMEOUT=10s        # Webhook timeout
```

## Metrics to Track

### Worker Pool Metrics
- Active workers
- Queue depth (per priority)
- Job processing time (p50, p95, p99)
- Job success/failure rate
- Queue wait time

### Circuit Breaker Metrics
- State (closed/open/half-open)
- Failure count
- Success count
- State transitions
- Time in each state

### Connection Pool Metrics
- Active connections
- Idle connections
- Connection wait time
- Connection errors
- Pool exhaustion events

### Load Shedding Metrics
- Requests shed (per priority)
- Shedding triggers
- Resource usage (CPU, memory, queue)
- Recovery time

## Testing Strategy

### Load Testing
```bash
# Test concurrent requests
ab -n 10000 -c 100 http://localhost:8080/v1/chat/completions

# Test with different priorities
# High priority: 30% of traffic
# Normal priority: 60% of traffic
# Low priority: 10% of traffic
```

### Stress Testing
```bash
# Gradually increase load until failure
# Measure: throughput, latency, error rate
# Target: Graceful degradation, no crashes
```

### Chaos Testing
```bash
# Simulate AWS API failures
# Simulate network issues
# Simulate high latency
# Verify: Circuit breaker works, recovery is automatic
```

## Benefits

### 1. Better Resource Utilization
- ✅ Controlled concurrency
- ✅ Connection reuse
- ✅ Memory efficiency

### 2. Improved Reliability
- ✅ Circuit breaker protection
- ✅ Graceful degradation
- ✅ Automatic recovery

### 3. Fair Resource Allocation
- ✅ Priority-based queuing
- ✅ Per-user rate limiting
- ✅ Load shedding

### 4. Better Observability
- ✅ Comprehensive metrics
- ✅ Queue depth monitoring
- ✅ Performance tracking

### 5. Scalability
- ✅ Horizontal scaling ready
- ✅ Configurable limits
- ✅ Async processing support

## Next Steps

1. **Implement Worker Pool** (Phase 1)
2. **Add Priority Queue** (Phase 2)
3. **Integrate Circuit Breaker** (Phase 3)
4. **Enhance Connection Pool** (Phase 4)
5. **Add Load Shedding** (Phase 5)
6. **Implement Async Processing** (Phase 6)

Each phase can be implemented and tested independently, allowing for incremental deployment and validation.
