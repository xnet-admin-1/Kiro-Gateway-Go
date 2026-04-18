package cache

import (
	"context"
	"testing"
	"time"
)

func TestCacheKey_Hash(t *testing.T) {
	key1 := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	key2 := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	key3 := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon EC2?",
	}
	
	// Same keys should produce same hash
	if key1.Hash() != key2.Hash() {
		t.Errorf("Expected same hash for identical keys")
	}
	
	// Different keys should produce different hash
	if key1.Hash() == key3.Hash() {
		t.Errorf("Expected different hash for different keys")
	}
}

func TestCacheKey_IsCacheable(t *testing.T) {
	tests := []struct {
		name      string
		key       CacheKey
		cacheable bool
	}{
		{
			name: "valid key",
			key: CacheKey{
				ModelID: "claude-sonnet-4-5",
				Prompt:  "What is S3?",
			},
			cacheable: true,
		},
		{
			name: "empty prompt",
			key: CacheKey{
				ModelID: "claude-sonnet-4-5",
				Prompt:  "",
			},
			cacheable: false,
		},
		{
			name: "empty model",
			key: CacheKey{
				ModelID: "",
				Prompt:  "What is S3?",
			},
			cacheable: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.IsCacheable(); got != tt.cacheable {
				t.Errorf("IsCacheable() = %v, want %v", got, tt.cacheable)
			}
		})
	}
}

func TestResponseCache_GetSet(t *testing.T) {
	cache := NewResponseCache(ResponseCacheConfig{
		MaxSize: 10,
		TTL:     1 * time.Minute,
		Enabled: true,
	})
	
	key := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	// Cache miss
	_, found := cache.Get(key)
	if found {
		t.Errorf("Expected cache miss")
	}
	
	// Set value
	response := "Amazon S3 is an object storage service..."
	cache.Set(key, response)
	
	// Cache hit
	entry, found := cache.Get(key)
	if !found {
		t.Errorf("Expected cache hit")
	}
	
	if entry.Response != response {
		t.Errorf("Expected response %q, got %q", response, entry.Response)
	}
}

func TestResponseCache_Expiration(t *testing.T) {
	cache := NewResponseCache(ResponseCacheConfig{
		MaxSize: 10,
		TTL:     100 * time.Millisecond,
		Enabled: true,
	})
	
	key := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	// Set value
	cache.Set(key, "response")
	
	// Should be cached
	_, found := cache.Get(key)
	if !found {
		t.Errorf("Expected cache hit")
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Should be expired
	_, found = cache.Get(key)
	if found {
		t.Errorf("Expected cache miss after expiration")
	}
}

func TestResponseCache_LRUEviction(t *testing.T) {
	cache := NewResponseCache(ResponseCacheConfig{
		MaxSize: 3,
		TTL:     1 * time.Minute,
		Enabled: true,
	})
	
	// Fill cache
	for i := 1; i <= 3; i++ {
		key := CacheKey{
			ModelID: "claude-sonnet-4-5",
			Prompt:  string(rune('A' + i - 1)),
		}
		cache.Set(key, "response")
	}
	
	// Cache should be full
	if cache.GetSize() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.GetSize())
	}
	
	// Add one more (should evict oldest)
	key4 := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "D",
	}
	cache.Set(key4, "response")
	
	// Cache should still be 3
	if cache.GetSize() != 3 {
		t.Errorf("Expected cache size 3 after eviction, got %d", cache.GetSize())
	}
	
	// First key should be evicted
	key1 := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "A",
	}
	_, found := cache.Get(key1)
	if found {
		t.Errorf("Expected first key to be evicted")
	}
}

func TestRequestDeduplicator_Execute(t *testing.T) {
	dedup := NewRequestDeduplicator(RequestDeduplicatorConfig{
		Enabled: true,
	})
	
	key := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	executed := 0
	fn := func() (string, error) {
		executed++
		time.Sleep(100 * time.Millisecond)
		return "response", nil
	}
	
	// Execute 5 concurrent requests
	results := make(chan string, 5)
	for i := 0; i < 5; i++ {
		go func() {
			response, _, _ := dedup.Execute(context.Background(), key, fn)
			results <- response
		}()
	}
	
	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-results
	}
	
	// Function should only be executed once
	if executed != 1 {
		t.Errorf("Expected function to be executed once, got %d", executed)
	}
	
	// Check metrics
	stats := dedup.GetStats()
	deduplicated := stats["deduplicated"].(int64)
	if deduplicated != 4 {
		t.Errorf("Expected 4 deduplicated requests, got %d", deduplicated)
	}
}

func TestResponseCache_Disabled(t *testing.T) {
	cache := NewResponseCache(ResponseCacheConfig{
		MaxSize: 10,
		TTL:     1 * time.Minute,
		Enabled: false,
	})
	
	key := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	// Set should be no-op
	cache.Set(key, "response")
	
	// Get should always miss
	_, found := cache.Get(key)
	if found {
		t.Errorf("Expected cache miss when disabled")
	}
}

func TestRequestDeduplicator_Disabled(t *testing.T) {
	dedup := NewRequestDeduplicator(RequestDeduplicatorConfig{
		Enabled: false,
	})
	
	key := CacheKey{
		ModelID: "claude-sonnet-4-5",
		Prompt:  "What is Amazon S3?",
	}
	
	executed := 0
	fn := func() (string, error) {
		executed++
		return "response", nil
	}
	
	// Execute 5 concurrent requests
	for i := 0; i < 5; i++ {
		dedup.Execute(context.Background(), key, fn)
	}
	
	// Function should be executed 5 times (no deduplication)
	if executed != 5 {
		t.Errorf("Expected function to be executed 5 times, got %d", executed)
	}
}
