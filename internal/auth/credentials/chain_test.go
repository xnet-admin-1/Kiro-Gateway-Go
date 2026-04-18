package credentials

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockProvider implements Provider interface for testing
type mockProvider struct {
	creds       *Credentials
	err         error
	isExpired   bool
	callCount   int
	retrieveFunc func(ctx context.Context) (*Credentials, error)
}

func (m *mockProvider) Retrieve(ctx context.Context) (*Credentials, error) {
	m.callCount++
	if m.retrieveFunc != nil {
		return m.retrieveFunc(ctx)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.creds, nil
}

func (m *mockProvider) IsExpired() bool {
	return m.isExpired
}

func TestChain_Retrieve(t *testing.T) {
	tests := []struct {
		name        string
		providers   []Provider
		wantSource  string
		wantErr     bool
	}{
		{
			name: "first provider succeeds",
			providers: []Provider{
				&mockProvider{creds: NewCredentials("key1", "secret1", "", "provider1")},
				&mockProvider{err: errors.New("should not be called")},
			},
			wantSource: "provider1",
			wantErr:    false,
		},
		{
			name: "second provider succeeds",
			providers: []Provider{
				&mockProvider{err: errors.New("first provider fails")},
				&mockProvider{creds: NewCredentials("key2", "secret2", "", "provider2")},
			},
			wantSource: "provider2",
			wantErr:    false,
		},
		{
			name: "all providers fail",
			providers: []Provider{
				&mockProvider{err: errors.New("provider1 fails")},
				&mockProvider{err: errors.New("provider2 fails")},
			},
			wantErr: true,
		},
		{
			name: "provider returns invalid credentials",
			providers: []Provider{
				&mockProvider{creds: NewCredentials("", "", "", "invalid")}, // Missing access key
				&mockProvider{creds: NewCredentials("key2", "secret2", "", "provider2")},
			},
			wantSource: "provider2",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewChain(tt.providers...)
			creds, err := chain.Retrieve(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Chain.Retrieve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if creds == nil {
					t.Error("Chain.Retrieve() returned nil credentials")
					return
				}

				if creds.Source != tt.wantSource {
					t.Errorf("Chain.Retrieve() source = %v, want %v", creds.Source, tt.wantSource)
				}
			}
		})
	}
}

func TestChain_Caching(t *testing.T) {
	callCount := 0
	
	provider := &mockProvider{
		retrieveFunc: func(ctx context.Context) (*Credentials, error) {
			callCount++
			return NewCredentials("key", "secret", "", "test"), nil
		},
	}

	chain := NewChain(provider)

	// First call should hit the provider
	creds1, err := chain.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("First retrieve failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 provider call, got %d", callCount)
	}

	// Second call should use cache
	creds2, err := chain.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Second retrieve failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 provider call (cached), got %d", callCount)
	}

	// Credentials should be the same
	if creds1.AccessKeyID != creds2.AccessKeyID {
		t.Error("Cached credentials differ from original")
	}
}

func TestChain_ExpiredCredentials(t *testing.T) {
	callCount := 0
	provider := &mockProvider{
		retrieveFunc: func(ctx context.Context) (*Credentials, error) {
			callCount++
			// Always return fresh credentials for this test
			return NewCredentials("key", "secret", "", "test"), nil
		},
	}

	chain := NewChain(provider)

	// First call gets credentials
	creds1, err := chain.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("First retrieve failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 provider call, got %d", callCount)
	}

	// Manually set expired credentials in cache
	chain.cached = NewTemporaryCredentials("oldkey", "oldsecret", "token", "test", time.Now().Add(-time.Hour))

	// Second call should get fresh credentials (expired ones not used)
	creds2, err := chain.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Second retrieve failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 provider calls (expired not cached), got %d", callCount)
	}

	if creds2.AccessKeyID != "key" {
		t.Error("Should have gotten fresh credentials")
	}

	// Verify we didn't get the expired credentials
	if creds1.AccessKeyID == creds2.AccessKeyID && creds1.AccessKeyID == "oldkey" {
		t.Error("Should not have returned expired credentials")
	}
}

func TestChain_ThreadSafety(t *testing.T) {
	provider := &mockProvider{
		creds: NewCredentials("key", "secret", "", "test"),
	}

	chain := NewChain(provider)

	// Run multiple goroutines concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := chain.Retrieve(context.Background())
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent retrieve failed: %v", err)
	}
}

func TestChain_IsExpired(t *testing.T) {
	chain := NewChain()

	// Empty chain should be expired
	if !chain.IsExpired() {
		t.Error("Empty chain should be expired")
	}

	// Add credentials to cache
	chain.cached = NewCredentials("key", "secret", "", "test")
	if chain.IsExpired() {
		t.Error("Chain with valid credentials should not be expired")
	}

	// Add expired credentials
	chain.cached = NewTemporaryCredentials("key", "secret", "token", "test", time.Now().Add(-time.Hour))
	if !chain.IsExpired() {
		t.Error("Chain with expired credentials should be expired")
	}
}

func TestChain_Invalidate(t *testing.T) {
	provider := &mockProvider{
		creds: NewCredentials("key", "secret", "", "test"),
	}

	chain := NewChain(provider)

	// Get credentials (should be cached)
	_, err := chain.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if chain.cached == nil {
		t.Error("Credentials should be cached")
	}

	// Invalidate cache
	chain.Invalidate()

	if chain.cached != nil {
		t.Error("Cache should be cleared after invalidate")
	}
}

func TestNewDefaultChain(t *testing.T) {
	chain := NewDefaultChain()

	if len(chain.providers) != 5 {
		t.Errorf("Expected 5 providers in default chain, got %d", len(chain.providers))
	}

	// Verify provider types (basic check)
	expectedTypes := []string{"*credentials.EnvProvider", "*credentials.ProfileProvider", "*credentials.WebIdentityProvider", "*credentials.ECSProvider", "*credentials.IMDSProvider"}
	
	for i, provider := range chain.providers {
		providerType := fmt.Sprintf("%T", provider)
		if providerType != expectedTypes[i] {
			t.Errorf("Provider %d: expected %s, got %s", i, expectedTypes[i], providerType)
		}
	}
}
func TestChain_AddProvider(t *testing.T) {
	chain := NewChain()
	
	mockProvider := &mockProvider{
		creds: NewCredentials("test-key", "test-secret", "", "test"),
	}
	
	chain.AddProvider(mockProvider)
	
	ctx := context.Background()
	creds, err := chain.Retrieve(ctx)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	
	if creds.AccessKeyID != "test-key" {
		t.Errorf("AccessKeyID = %v, want test-key", creds.AccessKeyID)
	}
	
	if creds.Source != "test" {
		t.Errorf("Source = %v, want test", creds.Source)
	}
}
