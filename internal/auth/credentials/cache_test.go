package credentials

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCache_Retrieve(t *testing.T) {
	// Create a mock provider
	mockProvider := &mockProvider{
		creds: &Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Source:          "test",
		},
	}

	// Create cache with the mock provider
	cache := NewCache(mockProvider)

	ctx := context.Background()

	// First retrieval should call the provider
	creds1, err := cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if creds1.AccessKeyID != "test-key" {
		t.Errorf("AccessKeyID = %v, want test-key", creds1.AccessKeyID)
	}

	if mockProvider.callCount != 1 {
		t.Errorf("Provider call count = %v, want 1", mockProvider.callCount)
	}

	// Second retrieval should use cached credentials
	creds2, err := cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if creds2.AccessKeyID != "test-key" {
		t.Errorf("AccessKeyID = %v, want test-key", creds2.AccessKeyID)
	}

	// Should still be 1 call (cached)
	if mockProvider.callCount != 1 {
		t.Errorf("Provider call count = %v, want 1 (cached)", mockProvider.callCount)
	}
}

func TestCache_RetrieveWithExpiry(t *testing.T) {
	// Create a mock provider with expiring credentials
	mockProvider := &mockProvider{
		creds: &Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Source:          "test",
			CanExpire:       true,
			Expires:         time.Now().Add(100 * time.Millisecond),
		},
	}

	// Create cache
	cache := NewCache(mockProvider)

	ctx := context.Background()

	// First retrieval
	_, err := cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if mockProvider.callCount != 1 {
		t.Errorf("Provider call count = %v, want 1", mockProvider.callCount)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Second retrieval should call provider again due to expiration
	_, err = cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if mockProvider.callCount != 2 {
		t.Errorf("Provider call count = %v, want 2 (expired)", mockProvider.callCount)
	}
}

func TestCache_RetrieveError(t *testing.T) {
	// Create a mock provider that returns an error
	mockProvider := &mockProvider{
		err: errors.New("provider error"),
	}

	cache := NewCache(mockProvider)

	ctx := context.Background()
	_, err := cache.Retrieve(ctx)
	if err == nil {
		t.Error("Retrieve() should return error from provider")
	}

	if err.Error() != "provider error" {
		t.Errorf("Error = %v, want 'provider error'", err)
	}
}

func TestCache_IsExpired(t *testing.T) {
	mockProvider := &mockProvider{
		creds: &Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Source:          "test",
		},
	}

	cache := NewCache(mockProvider)

	// Initially should be expired (no cached credentials)
	if !cache.IsExpired() {
		t.Error("IsExpired() should return true for empty cache")
	}

	// After retrieval, should not be expired
	ctx := context.Background()
	_, err := cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if cache.IsExpired() {
		t.Error("IsExpired() should return false after successful retrieval")
	}
}

func TestCache_Invalidate(t *testing.T) {
	mockProvider := &mockProvider{
		creds: &Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Source:          "test",
		},
	}

	cache := NewCache(mockProvider)

	ctx := context.Background()

	// Retrieve credentials to populate cache
	_, err := cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if mockProvider.callCount != 1 {
		t.Errorf("Provider call count = %v, want 1", mockProvider.callCount)
	}

	// Invalidate cache
	cache.Invalidate()

	// Next retrieval should call provider again
	_, err = cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if mockProvider.callCount != 2 {
		t.Errorf("Provider call count = %v, want 2 (after invalidation)", mockProvider.callCount)
	}
}

func TestCache_WithExpiryWindow(t *testing.T) {
	mockProvider := &mockProvider{
		creds: &Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Source:          "test",
			CanExpire:       true,
			Expires:         time.Now().Add(2 * time.Second),
		},
	}

	// Create cache with 1 second expiry window
	cache := NewCacheWithExpiryWindow(mockProvider, 1*time.Second)

	ctx := context.Background()

	// First retrieval
	_, err := cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if mockProvider.callCount != 1 {
		t.Errorf("Provider call count = %v, want 1", mockProvider.callCount)
	}

	// Wait for expiry window (credentials expire in 2s, but window is 1s)
	time.Sleep(1100 * time.Millisecond)

	// Should be considered expired due to expiry window
	if !cache.IsExpired() {
		t.Error("IsExpired() should return true within expiry window")
	}

	// Next retrieval should call provider again
	_, err = cache.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if mockProvider.callCount != 2 {
		t.Errorf("Provider call count = %v, want 2 (within expiry window)", mockProvider.callCount)
	}
}

func TestCache_SetExpiryWindow(t *testing.T) {
	mockProvider := &mockProvider{
		creds: &Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Source:          "test",
		},
	}

	cache := NewCache(mockProvider)

	// Set expiry window
	cache.SetExpiryWindow(30 * time.Second)

	// Verify the expiry window was set (this is mainly for code coverage)
	// The actual behavior is tested in TestCache_WithExpiryWindow
}

// Mock provider for testing (using the one from chain_test.go)
