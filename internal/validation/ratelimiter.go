package validation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      int
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps, burst int) *RateLimiter {
	if rps <= 0 {
		rps = DefaultRateLimitRPS
	}
	if burst <= 0 {
		burst = DefaultRateLimitBurst
	}
	
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}
}

// GetLimiter returns a rate limiter for a specific key (e.g., user ID, API key)
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
		rl.limiters[key] = limiter
	}
	
	return limiter
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	limiter := rl.GetLimiter(key)
	return limiter.Allow()
}

// Wait waits until a request is allowed for the given key
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
	limiter := rl.GetLimiter(key)
	return limiter.Wait(ctx)
}

// Reserve reserves a token for the given key and returns a reservation
func (rl *RateLimiter) Reserve(key string) *rate.Reservation {
	limiter := rl.GetLimiter(key)
	return limiter.Reserve()
}

// CleanupOldLimiters removes limiters that haven't been used recently
func (rl *RateLimiter) CleanupOldLimiters(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Note: This is a simple implementation. In production, you'd want to track
	// last access time for each limiter and remove old ones.
	// For now, we'll keep all limiters in memory.
}

// QuotaTracker tracks monthly quotas for users
type QuotaTracker struct {
	quotas map[string]*UserQuota
	mu     sync.RWMutex
}

// UserQuota tracks quota usage for a user
type UserQuota struct {
	AgenticRequests int
	InferenceCalls  int
	LastReset       time.Time
	mu              sync.Mutex
}

// NewQuotaTracker creates a new quota tracker
func NewQuotaTracker() *QuotaTracker {
	return &QuotaTracker{
		quotas: make(map[string]*UserQuota),
	}
}

// GetQuota returns the quota for a user
func (qt *QuotaTracker) GetQuota(userID string) *UserQuota {
	qt.mu.Lock()
	defer qt.mu.Unlock()
	
	quota, exists := qt.quotas[userID]
	if !exists {
		quota = &UserQuota{
			LastReset: time.Now(),
		}
		qt.quotas[userID] = quota
	}
	
	// Reset quota if it's a new month
	quota.mu.Lock()
	defer quota.mu.Unlock()
	
	now := time.Now()
	if now.Month() != quota.LastReset.Month() || now.Year() != quota.LastReset.Year() {
		quota.AgenticRequests = 0
		quota.InferenceCalls = 0
		quota.LastReset = now
	}
	
	return quota
}

// CheckQuota checks if a user has quota available
func (qt *QuotaTracker) CheckQuota(userID string) error {
	quota := qt.GetQuota(userID)
	
	quota.mu.Lock()
	defer quota.mu.Unlock()
	
	if quota.AgenticRequests >= MaxAgenticRequestsPerMonth {
		return fmt.Errorf("monthly agentic request quota exceeded (%d/%d)", 
			quota.AgenticRequests, MaxAgenticRequestsPerMonth)
	}
	
	if quota.InferenceCalls >= MaxInferenceCallsPerMonth {
		return fmt.Errorf("monthly inference call quota exceeded (%d/%d)", 
			quota.InferenceCalls, MaxInferenceCallsPerMonth)
	}
	
	return nil
}

// IncrementQuota increments the quota usage for a user
func (qt *QuotaTracker) IncrementQuota(userID string, agenticRequests, inferenceCalls int) {
	quota := qt.GetQuota(userID)
	
	quota.mu.Lock()
	defer quota.mu.Unlock()
	
	quota.AgenticRequests += agenticRequests
	quota.InferenceCalls += inferenceCalls
}

// GetQuotaUsage returns the current quota usage for a user
func (qt *QuotaTracker) GetQuotaUsage(userID string) (agenticRequests, inferenceCalls int) {
	quota := qt.GetQuota(userID)
	
	quota.mu.Lock()
	defer quota.mu.Unlock()
	
	return quota.AgenticRequests, quota.InferenceCalls
}
