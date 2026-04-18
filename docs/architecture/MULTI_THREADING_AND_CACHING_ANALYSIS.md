# Multi-Threading and Caching Analysis

**Date**: January 26, 2026  
**Status**: Analysis Complete

## Executive Summary

The gateway has **proper multi-threading** but **NO response caching** or **request deduplication**. CPU utilization relies on Go's default behavior (uses all available cores automatically).

---

## 1. Multi-Threading Analysis

### Current State: ✅ GOOD

#### HTTP Server Concurrency
- **Go's http.Server spawns a goroutine per connection** - unlimited concurrent connections
- Each request runs in its own goroutine automatically
- No explicit GOMAXPROCS configuration (Go defaults to `runtime.NumCPU()`)

#### Application-Level Concurrency Control (6 Layers)

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: HTTP Server (Unlimited Goroutines)                │
│  - ReadTimeout: 30s                                         │
│  - WriteTimeout: 5m (for streaming)                         │
│  - IdleTimeout: 120s                                        │
│  - MaxHeaderBytes: 1 MB                                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Priority Queue (1000 capacity)                    │
│  - High Priority: 200 slots                                 │
│  - Normal Priority: 500 slots                               │
│  - Low Priority: 300 slots                                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Worker Pool (20 workers)                          │
│  - Processes jobs from priority queue                       │
│  - Job timeout: 60s                                         │
│  - Queue size: 1000                                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 4: Load Shedder (80% threshold)                      │
│  - Rejects requests when queue >80% full                    │
│  - Monitors: Queue, Memory, CPU, Circuit Breaker            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 5: Rate Limiter (10 req/s per user)                  │
│  - Per-user rate limiting                                   │
│  - Token bucket algorithm                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 6: Connection Pool (to AWS)                          │
│  - Max idle: 200 connections                                │
│  - Max per host: 100 connections                            │
│  - Idle timeout: 120s                                       │
└─────────────────────────────────────────────────────────────┘
```

### CPU Utilization

**Current**: Go's default behavior
- `GOMAXPROCS` defaults to `runtime.NumCPU()` (number of logical CPUs)
- On a 4-core CPU: 4 OS threads can run Go code simultaneously
- On an 8-core CPU: 8 OS threads can run Go code simultaneously
- **This is optimal for most workloads** - no changes needed

**Why Go's Default is Good**:
- Go's scheduler efficiently distributes goroutines across available cores
- Setting GOMAXPROCS higher than CPU count doesn't help (more context switching)
- Setting it lower wastes CPU resources
- Only override if you have specific CPU affinity requirements

### Verdict: ✅ Multi-threading is properly configured

---

## 2. Caching Analysis

### Current State: ❌ NO RESPONSE CACHING

#### What Exists
1. **Credential Cache** (`internal/auth/credentials/cache.go`)
   - Caches AWS credentials (access keys, session tokens)
   - 1-minute expiry window before refresh
   - Thread-safe with mutex
   - **Purpose**: Avoid repeated credential fetches

2. **SSO Token Cache** (`internal/auth/cli_db.go`)
   - Caches SSO access/refresh tokens
   - Stored in SQLite database
   - **Purpose**: Persist authentication across restarts

#### What's Missing
1. **Response Caching** - ❌ NOT IMPLEMENTED
   - No caching of Q Developer API responses
   - Every request hits AWS, even for identical prompts
   - No TTL-based cache for repeated questions

2. **Request Deduplication** - ❌ NOT IMPLEMENTED
   - Multiple concurrent identical requests all execute
   - No in-flight request tracking
   - Wastes AWS quota and increases latency

### Impact of Missing Caching

#### Scenario: 5 users ask "What is Amazon S3?" simultaneously

**Current Behavior** (No Caching):
```
User 1 → Gateway → AWS Q Developer → Response (2s)
User 2 → Gateway → AWS Q Developer → Response (2s)
User 3 → Gateway → AWS Q Developer → Response (2s)
User 4 → Gateway → AWS Q Developer → Response (2s)
User 5 → Gateway → AWS Q Developer → Response (2s)

Total: 5 AWS API calls, 10s total latency
```

**With Response Caching**:
```
User 1 → Gateway → AWS Q Developer → Response (2s) → Cache
User 2 → Gateway → Cache → Response (10ms)
User 3 → Gateway → Cache → Response (10ms)
User 4 → Gateway → Cache → Response (10ms)
User 5 → Gateway → Cache → Response (10ms)

Total: 1 AWS API call, 2.04s total latency
Savings: 80% fewer API calls, 80% faster for cached requests
```

**With Request Deduplication**:
```
User 1 → Gateway → AWS Q Developer → Response (2s) → All 5 users
User 2 ↗
User 3 ↗
User 4 ↗
User 5 ↗

Total: 1 AWS API call, 2s latency for all
Savings: 80% fewer API calls, 60% faster average response
```

### Verdict: ❌ No response caching or request deduplication

---

## 3. Recommendations

### Priority 1: Add Response Caching (HIGH IMPACT)

**Benefits**:
- Reduce AWS API calls by 50-80% for repeated questions
- Faster responses (10ms vs 2s for cached responses)
- Lower AWS costs
- Better user experience

**Implementation**:
```go
// internal/cache/response_cache.go
type ResponseCache struct {
    cache *lru.Cache // LRU eviction
    ttl   time.Duration
    mu    sync.RWMutex
}

type CacheKey struct {
    ModelID string
    Prompt  string
    // Don't include images in cache key (multimodal requests not cached)
}

type CacheEntry struct {
    Response  string
    CreatedAt time.Time
    ExpiresAt time.Time
}
```

**Configuration**:
- Default TTL: 5 minutes (configurable via `RESPONSE_CACHE_TTL`)
- Max size: 1000 entries (configurable via `RESPONSE_CACHE_SIZE`)
- LRU eviction when full
- Skip caching for:
  - Multimodal requests (images change)
  - Streaming responses (partial data)
  - Errors

### Priority 2: Add Request Deduplication (MEDIUM IMPACT)

**Benefits**:
- Prevent duplicate concurrent requests
- Save AWS quota
- Reduce load on AWS API

**Implementation**:
```go
// internal/cache/request_dedup.go
type RequestDeduplicator struct {
    inflight map[string]*InflightRequest
    mu       sync.RWMutex
}

type InflightRequest struct {
    ResultChan chan *Result
    Waiters    []chan *Result
}
```

**How it works**:
1. Hash incoming request (model + prompt)
2. Check if identical request is in-flight
3. If yes: Wait for existing request to complete, share result
4. If no: Execute request, notify all waiters when done

### Priority 3: Explicit GOMAXPROCS (LOW PRIORITY)

**Only needed if**:
- Running in containerized environment with CPU limits
- Need CPU affinity for specific cores
- Debugging CPU utilization issues

**Implementation**:
```go
// cmd/kiro-gateway/main.go
import "runtime"

func main() {
    // Optional: Set GOMAXPROCS explicitly
    if maxProcs := os.Getenv("GOMAXPROCS"); maxProcs != "" {
        if n, err := strconv.Atoi(maxProcs); err == nil {
            runtime.GOMAXPROCS(n)
            log.Printf("Set GOMAXPROCS to %d", n)
        }
    }
    // Otherwise use default (runtime.NumCPU())
}
```

---

## 4. Implementation Plan

### Phase 1: Response Caching (2-3 hours)

1. **Create cache package** (`internal/cache/`)
   - `response_cache.go` - LRU cache with TTL
   - `cache_key.go` - Cache key generation
   - `cache_test.go` - Unit tests

2. **Integrate into handlers**
   - Check cache before AWS API call
   - Store response in cache after successful call
   - Skip caching for multimodal/streaming/errors

3. **Add configuration**
   - `RESPONSE_CACHE_ENABLED` (default: true)
   - `RESPONSE_CACHE_TTL` (default: 5m)
   - `RESPONSE_CACHE_SIZE` (default: 1000)

4. **Add metrics**
   - Cache hits/misses
   - Cache size
   - Eviction count

### Phase 2: Request Deduplication (1-2 hours)

1. **Create deduplicator** (`internal/cache/request_dedup.go`)
   - Track in-flight requests
   - Share results with waiters

2. **Integrate into handlers**
   - Check for in-flight request before execution
   - Register waiter if duplicate found
   - Broadcast result to all waiters

3. **Add metrics**
   - Deduplicated requests
   - Waiter count

### Phase 3: GOMAXPROCS Configuration (15 minutes)

1. **Add environment variable support**
   - `GOMAXPROCS` env var
   - Log current value on startup

2. **Document in README**
   - When to set GOMAXPROCS
   - Recommended values

---

## 5. Testing Plan

### Load Testing
```powershell
# Test concurrent requests
$jobs = 1..10 | ForEach-Object {
    Start-Job -ScriptBlock {
        Invoke-RestMethod -Uri "http://localhost:8080/v1/chat/completions" `
            -Method POST `
            -Headers @{"Authorization"="Bearer $env:API_KEY"} `
            -Body (@{
                model = "claude-sonnet-4-5"
                messages = @(@{role="user"; content="What is Amazon S3?"})
            } | ConvertTo-Json)
    }
}

# Wait for all jobs
$jobs | Wait-Job | Receive-Job
```

### Cache Testing
```powershell
# First request (cache miss)
Measure-Command {
    Invoke-RestMethod -Uri "http://localhost:8080/v1/chat/completions" ...
}
# Expected: ~2s

# Second request (cache hit)
Measure-Command {
    Invoke-RestMethod -Uri "http://localhost:8080/v1/chat/completions" ...
}
# Expected: ~10ms
```

---

## 6. Summary

| Feature | Status | Priority | Impact |
|---------|--------|----------|--------|
| Multi-threading | ✅ Implemented | - | High |
| HTTP Timeouts | ✅ Implemented | - | High |
| Concurrency Control | ✅ Implemented | - | High |
| CPU Utilization | ✅ Default (Good) | Low | Medium |
| Response Caching | ❌ Missing | **HIGH** | **High** |
| Request Deduplication | ❌ Missing | Medium | Medium |
| GOMAXPROCS Config | ❌ Missing | Low | Low |

**Recommendation**: Implement response caching first (highest impact), then request deduplication, then GOMAXPROCS configuration if needed.

---

## 7. Next Steps

1. ✅ Document current state (this file)
2. ⏭️ Implement response caching
3. ⏭️ Implement request deduplication
4. ⏭️ Add GOMAXPROCS configuration
5. ⏭️ Load test with caching enabled
6. ⏭️ Update documentation

**Estimated Total Time**: 4-6 hours for full implementation
