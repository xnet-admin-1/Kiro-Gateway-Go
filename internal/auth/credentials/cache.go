// Package credentials provides AWS credential provider implementations.
package credentials

import (
	"context"
	"sync"
	"time"
)

// Cache wraps a credential provider with caching functionality.
// It caches credentials until they expire, reducing the number of calls
// to the underlying provider.
//
// The cache is thread-safe and can be used concurrently.
type Cache struct {
	// provider is the underlying credential provider
	provider Provider

	// cached stores the cached credentials
	cached *Credentials

	// mu protects concurrent access to cached credentials
	mu sync.RWMutex

	// expiryWindow is the time before expiration to refresh credentials
	// This provides a safety margin to avoid using credentials that are
	// about to expire. Default is 1 minute.
	expiryWindow time.Duration
}

// NewCache creates a new credential cache wrapping the given provider.
// The default expiry window is 1 minute.
func NewCache(provider Provider) *Cache {
	return &Cache{
		provider:     provider,
		expiryWindow: time.Minute,
	}
}

// NewCacheWithExpiryWindow creates a new credential cache with a custom expiry window.
func NewCacheWithExpiryWindow(provider Provider, expiryWindow time.Duration) *Cache {
	return &Cache{
		provider:     provider,
		expiryWindow: expiryWindow,
	}
}

// Retrieve returns cached credentials if they are still valid, otherwise
// retrieves fresh credentials from the underlying provider.
func (c *Cache) Retrieve(ctx context.Context) (*Credentials, error) {
	// Check if we have valid cached credentials
	c.mu.RLock()
	if c.cached != nil && !c.isExpired(c.cached) {
		cached := c.cached.Copy()
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	// Retrieve fresh credentials
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have refreshed)
	if c.cached != nil && !c.isExpired(c.cached) {
		return c.cached.Copy(), nil
	}

	// Retrieve from underlying provider
	creds, err := c.provider.Retrieve(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the credentials
	c.cached = creds.Copy()
	return creds, nil
}

// IsExpired checks if the cached credentials are expired.
func (c *Cache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cached == nil {
		return true
	}
	return c.isExpired(c.cached)
}

// isExpired checks if credentials are expired, considering the expiry window.
// This method assumes the caller holds at least a read lock.
func (c *Cache) isExpired(creds *Credentials) bool {
	if creds == nil {
		return true
	}

	if !creds.CanExpire {
		return false
	}

	// Consider credentials expired if they expire within the expiry window
	expiryTime := creds.Expires.Add(-c.expiryWindow)
	return time.Now().After(expiryTime)
}

// Invalidate clears the cached credentials, forcing the next Retrieve call
// to fetch fresh credentials from the underlying provider.
func (c *Cache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = nil
}

// SetExpiryWindow sets the expiry window for the cache.
// Credentials are considered expired if they expire within this window.
func (c *Cache) SetExpiryWindow(window time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expiryWindow = window
}
