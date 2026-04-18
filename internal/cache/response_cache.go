package cache

import (
	"container/list"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ResponseCache implements an LRU cache with TTL for API responses
type ResponseCache struct {
	// Configuration
	maxSize int
	ttl     time.Duration
	enabled bool
	
	// Cache storage
	items map[string]*cacheItem
	lru   *list.List
	
	// Synchronization
	mu sync.RWMutex
	
	// Metrics
	metrics *CacheMetrics
}

// cacheItem represents a cached response
type cacheItem struct {
	key       string
	value     *CacheEntry
	element   *list.Element
	expiresAt time.Time
}

// CacheEntry represents a cached API response
type CacheEntry struct {
	Response  string
	ModelID   string
	CreatedAt time.Time
	ExpiresAt time.Time
	Size      int
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits      atomic.Int64
	Misses    atomic.Int64
	Evictions atomic.Int64
	Expirations atomic.Int64
	Size      atomic.Int32
}

// ResponseCacheConfig holds cache configuration
type ResponseCacheConfig struct {
	MaxSize int
	TTL     time.Duration
	Enabled bool
}

// NewResponseCache creates a new response cache
func NewResponseCache(cfg ResponseCacheConfig) *ResponseCache {
	// Set defaults
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 1000
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}
	
	cache := &ResponseCache{
		maxSize: cfg.MaxSize,
		ttl:     cfg.TTL,
		enabled: cfg.Enabled,
		items:   make(map[string]*cacheItem),
		lru:     list.New(),
		metrics: &CacheMetrics{},
	}
	
	// Start cleanup goroutine
	if cfg.Enabled {
		go cache.cleanupExpired()
	}
	
	log.Printf("Response cache initialized (enabled: %v, max size: %d, TTL: %v)",
		cfg.Enabled, cfg.MaxSize, cfg.TTL)
	
	return cache
}

// Get retrieves a cached response
func (c *ResponseCache) Get(key CacheKey) (*CacheEntry, bool) {
	if !c.enabled {
		return nil, false
	}
	
	hash := key.Hash()
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, exists := c.items[hash]
	if !exists {
		c.metrics.Misses.Add(1)
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(item.expiresAt) {
		c.removeItem(hash)
		c.metrics.Misses.Add(1)
		c.metrics.Expirations.Add(1)
		return nil, false
	}
	
	// Move to front (most recently used)
	c.lru.MoveToFront(item.element)
	
	c.metrics.Hits.Add(1)
	log.Printf("Cache HIT: %s (age: %v)", key.String(), time.Since(item.value.CreatedAt))
	
	return item.value, true
}

// Set stores a response in the cache
func (c *ResponseCache) Set(key CacheKey, response string) {
	if !c.enabled {
		return
	}
	
	if !key.IsCacheable() {
		return
	}
	
	hash := key.Hash()
	now := time.Now()
	
	entry := &CacheEntry{
		Response:  response,
		ModelID:   key.ModelID,
		CreatedAt: now,
		ExpiresAt: now.Add(c.ttl),
		Size:      len(response),
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if already exists
	if item, exists := c.items[hash]; exists {
		// Update existing item
		item.value = entry
		item.expiresAt = entry.ExpiresAt
		c.lru.MoveToFront(item.element)
		log.Printf("Cache UPDATE: %s", key.String())
		return
	}
	
	// Evict if at capacity
	if c.lru.Len() >= c.maxSize {
		c.evictOldest()
	}
	
	// Add new item
	element := c.lru.PushFront(hash)
	c.items[hash] = &cacheItem{
		key:       hash,
		value:     entry,
		element:   element,
		expiresAt: entry.ExpiresAt,
	}
	
	c.metrics.Size.Store(int32(len(c.items)))
	log.Printf("Cache SET: %s (size: %d bytes, TTL: %v)", key.String(), entry.Size, c.ttl)
}

// evictOldest removes the least recently used item
func (c *ResponseCache) evictOldest() {
	element := c.lru.Back()
	if element == nil {
		return
	}
	
	hash := element.Value.(string)
	c.removeItem(hash)
	c.metrics.Evictions.Add(1)
	
	log.Printf("Cache EVICT: LRU item removed (size: %d/%d)", len(c.items), c.maxSize)
}

// removeItem removes an item from the cache
func (c *ResponseCache) removeItem(hash string) {
	item, exists := c.items[hash]
	if !exists {
		return
	}
	
	c.lru.Remove(item.element)
	delete(c.items, hash)
	c.metrics.Size.Store(int32(len(c.items)))
}

// cleanupExpired periodically removes expired items
func (c *ResponseCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		
		now := time.Now()
		expired := 0
		
		// Find and remove expired items
		for hash, item := range c.items {
			if now.After(item.expiresAt) {
				c.removeItem(hash)
				expired++
			}
		}
		
		c.mu.Unlock()
		
		if expired > 0 {
			c.metrics.Expirations.Add(int64(expired))
			log.Printf("Cache cleanup: removed %d expired item(s)", expired)
		}
	}
}

// Clear removes all items from the cache
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items = make(map[string]*cacheItem)
	c.lru = list.New()
	c.metrics.Size.Store(0)
	
	log.Println("Cache cleared")
}

// GetStats returns cache statistics
func (c *ResponseCache) GetStats() map[string]interface{} {
	hits := c.metrics.Hits.Load()
	misses := c.metrics.Misses.Load()
	total := hits + misses
	
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	
	return map[string]interface{}{
		"enabled":     c.enabled,
		"max_size":    c.maxSize,
		"ttl":         c.ttl.String(),
		"size":        c.metrics.Size.Load(),
		"hits":        hits,
		"misses":      misses,
		"hit_rate":    hitRate,
		"evictions":   c.metrics.Evictions.Load(),
		"expirations": c.metrics.Expirations.Load(),
	}
}

// IsEnabled returns true if caching is enabled
func (c *ResponseCache) IsEnabled() bool {
	return c.enabled
}

// GetSize returns the current cache size
func (c *ResponseCache) GetSize() int {
	return int(c.metrics.Size.Load())
}

// GetHitRate returns the cache hit rate as a percentage
func (c *ResponseCache) GetHitRate() float64 {
	hits := c.metrics.Hits.Load()
	misses := c.metrics.Misses.Load()
	total := hits + misses
	
	if total == 0 {
		return 0
	}
	
	return float64(hits) / float64(total) * 100
}
