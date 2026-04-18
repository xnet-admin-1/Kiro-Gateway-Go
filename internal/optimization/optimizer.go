// Package optimization provides performance optimization utilities and bottleneck identification.
package optimization

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
	"github.com/yourusername/kiro-gateway-go/internal/auth/sigv4"
	"github.com/yourusername/kiro-gateway-go/internal/profiling"
)

// Optimizer identifies and measures performance bottlenecks.
type Optimizer struct {
	profiler *profiling.Profiler
	results  []OptimizationResult
	mu       sync.RWMutex
}

// OptimizationResult contains performance analysis results.
type OptimizationResult struct {
	Component    string
	Operation    string
	Duration     time.Duration
	MemoryDelta  uint64
	Bottleneck   bool
	Suggestion   string
	Timestamp    time.Time
}

// NewOptimizer creates a new performance optimizer.
func NewOptimizer() *Optimizer {
	return &Optimizer{
		profiler: profiling.NewProfiler("."),
		results:  make([]OptimizationResult, 0),
	}
}

// AnalyzeCredentialChain analyzes credential chain performance.
func (o *Optimizer) AnalyzeCredentialChain(ctx context.Context, chain *credentials.Chain) (*OptimizationResult, error) {
	start := time.Now()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Simulate credential retrieval
	_, err := chain.Retrieve(ctx)
	
	duration := time.Since(start)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	result := &OptimizationResult{
		Component:   "CredentialChain",
		Operation:   "Retrieve",
		Duration:    duration,
		MemoryDelta: memAfter.Alloc - memBefore.Alloc,
		Bottleneck:  duration > 100*time.Millisecond, // Requirement: <100ms
		Timestamp:   time.Now(),
	}
	
	if result.Bottleneck {
		result.Suggestion = "Credential resolution exceeds 100ms threshold. Consider caching or optimizing provider chain order."
	} else {
		result.Suggestion = "Credential resolution performance is acceptable."
	}
	
	o.addResult(*result)
	return result, err
}

// AnalyzeSigV4Signing analyzes SigV4 signing performance.
func (o *Optimizer) AnalyzeSigV4Signing(ctx context.Context, signer *sigv4.Signer) (*OptimizationResult, error) {
	start := time.Now()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Create a test request for signing
	req, _ := createTestRequest()
	body := []byte(`{"test": "data"}`)
	
	err := signer.SignRequest(req, body)
	
	duration := time.Since(start)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	result := &OptimizationResult{
		Component:   "SigV4Signer",
		Operation:   "SignRequest",
		Duration:    duration,
		MemoryDelta: memAfter.Alloc - memBefore.Alloc,
		Bottleneck:  duration > 5*time.Millisecond, // Requirement: <5ms
		Timestamp:   time.Now(),
	}
	
	if result.Bottleneck {
		result.Suggestion = "SigV4 signing exceeds 5ms threshold. Consider optimizing signature calculation or caching signing keys."
	} else {
		result.Suggestion = "SigV4 signing performance is acceptable."
	}
	
	o.addResult(*result)
	return result, err
}

// AnalyzeCacheEffectiveness analyzes cache hit rates and effectiveness.
func (o *Optimizer) AnalyzeCacheEffectiveness(ctx context.Context, chain *credentials.Chain, iterations int) (*OptimizationResult, error) {
	start := time.Now()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	
	cacheHits := 0
	for i := 0; i < iterations; i++ {
		retrieveStart := time.Now()
		_, err := chain.Retrieve(ctx)
		if err != nil {
			continue
		}
		
		// If retrieval is very fast, it's likely a cache hit
		if time.Since(retrieveStart) < 1*time.Millisecond {
			cacheHits++
		}
	}
	
	duration := time.Since(start)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	cacheHitRate := float64(cacheHits) / float64(iterations) * 100
	
	result := &OptimizationResult{
		Component:   "CredentialCache",
		Operation:   fmt.Sprintf("CacheAnalysis_%d_iterations", iterations),
		Duration:    duration,
		MemoryDelta: memAfter.Alloc - memBefore.Alloc,
		Bottleneck:  cacheHitRate < 80.0, // Expect >80% cache hit rate
		Timestamp:   time.Now(),
	}
	
	if result.Bottleneck {
		result.Suggestion = fmt.Sprintf("Cache hit rate is %.1f%%, below 80%% threshold. Consider adjusting cache TTL or invalidation strategy.", cacheHitRate)
	} else {
		result.Suggestion = fmt.Sprintf("Cache effectiveness is good with %.1f%% hit rate.", cacheHitRate)
	}
	
	o.addResult(*result)
	return result, nil
}

// IdentifyBottlenecks runs comprehensive performance analysis.
func (o *Optimizer) IdentifyBottlenecks(ctx context.Context) ([]OptimizationResult, error) {
	var bottlenecks []OptimizationResult
	
	// Analyze credential chain with mock provider to avoid network calls
	mockProvider := &mockCredentialProvider{
		creds: &credentials.Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			SessionToken:    "",
			Source:          "test",
		},
	}
	chain := credentials.NewChain(mockProvider)
	credResult, err := o.AnalyzeCredentialChain(ctx, chain)
	if err == nil && credResult.Bottleneck {
		bottlenecks = append(bottlenecks, *credResult)
	}
	
	// Analyze SigV4 signing (with mock credentials)
	mockCreds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "",
		Source:          "test",
	}
	signer := sigv4.NewSigner(mockCreds, "us-east-1", "codewhisperer")
	sigResult, err := o.AnalyzeSigV4Signing(ctx, signer)
	if err == nil && sigResult.Bottleneck {
		bottlenecks = append(bottlenecks, *sigResult)
	}
	
	// Analyze cache effectiveness with smaller iteration count
	cacheResult, err := o.AnalyzeCacheEffectiveness(ctx, chain, 10)
	if err == nil && cacheResult.Bottleneck {
		bottlenecks = append(bottlenecks, *cacheResult)
	}
	
	return bottlenecks, nil
}

// mockCredentialProvider is a mock provider for testing
type mockCredentialProvider struct {
	creds *credentials.Credentials
	err   error
}

func (m *mockCredentialProvider) Retrieve(ctx context.Context) (*credentials.Credentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.creds, nil
}

func (m *mockCredentialProvider) IsExpired() bool {
	return false
}

// GetResults returns all optimization results.
func (o *Optimizer) GetResults() []OptimizationResult {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	results := make([]OptimizationResult, len(o.results))
	copy(results, o.results)
	return results
}

// GenerateReport generates a performance optimization report.
func (o *Optimizer) GenerateReport() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	report := "Performance Optimization Report\n"
	report += "================================\n\n"
	
	bottleneckCount := 0
	for _, result := range o.results {
		if result.Bottleneck {
			bottleneckCount++
		}
		
		status := "✓ PASS"
		if result.Bottleneck {
			status = "✗ BOTTLENECK"
		}
		
		report += fmt.Sprintf("%s %s.%s\n", status, result.Component, result.Operation)
		report += fmt.Sprintf("  Duration: %v\n", result.Duration)
		report += fmt.Sprintf("  Memory: %d bytes\n", result.MemoryDelta)
		report += fmt.Sprintf("  Suggestion: %s\n\n", result.Suggestion)
	}
	
	report += fmt.Sprintf("Summary: %d bottlenecks found out of %d components analyzed.\n", bottleneckCount, len(o.results))
	
	return report
}

// addResult adds a result to the optimizer (thread-safe).
func (o *Optimizer) addResult(result OptimizationResult) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.results = append(o.results, result)
}

// createTestRequest creates a test HTTP request for performance testing.
func createTestRequest() (*http.Request, error) {
	req, err := http.NewRequest("POST", "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
