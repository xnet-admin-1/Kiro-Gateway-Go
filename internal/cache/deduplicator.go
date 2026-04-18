package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// RequestDeduplicator prevents duplicate concurrent requests
type RequestDeduplicator struct {
	enabled bool
	mu      sync.RWMutex
	pending map[string]*DedupResult
}

// DedupResult holds the result of a deduplicated request
type DedupResult struct {
	Done     chan struct{}
	Response []byte
	Error    error
}

// RequestDeduplicatorConfig holds deduplicator configuration
type RequestDeduplicatorConfig struct {
	Enabled bool
}

// NewRequestDeduplicator creates a new request deduplicator
func NewRequestDeduplicator(cfg RequestDeduplicatorConfig) *RequestDeduplicator {
	return &RequestDeduplicator{
		enabled: cfg.Enabled,
		pending: make(map[string]*DedupResult),
	}
}

// CheckDuplicate checks if a request is a duplicate
// Returns (result, isDuplicate)
func (d *RequestDeduplicator) CheckDuplicate(key string) (*DedupResult, bool) {
	if !d.enabled {
		return nil, false
	}
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if result, exists := d.pending[key]; exists {
		return result, true
	}
	
	// Register new request
	result := &DedupResult{
		Done: make(chan struct{}),
	}
	d.pending[key] = result
	
	return result, false
}

// Complete marks a request as complete
func (d *RequestDeduplicator) Complete(key string, response []byte, err error) {
	if !d.enabled {
		return
	}
	
	d.mu.Lock()
	result, exists := d.pending[key]
	if !exists {
		d.mu.Unlock()
		return
	}
	
	result.Response = response
	result.Error = err
	close(result.Done)
	
	// Clean up after a delay
	go func() {
		time.Sleep(30 * time.Second)
		d.mu.Lock()
		delete(d.pending, key)
		d.mu.Unlock()
	}()
	
	d.mu.Unlock()
}

// IsEnabled returns true if deduplication is enabled
func (d *RequestDeduplicator) IsEnabled() bool {
	return d.enabled
}

// CreateDedupKey creates a deduplication key from request body
func CreateDedupKey(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}
