package optimization

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/yourusername/kiro-gateway-go/internal/auth/credentials"
	"github.com/yourusername/kiro-gateway-go/internal/auth/sigv4"
)

func TestOptimizer_AnalyzeCredentialChain(t *testing.T) {
	optimizer := NewOptimizer()
	
	// Create a mock credential chain
	chain := credentials.NewChain(
		&mockProvider{
			creds: &credentials.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Source:          "test",
			},
		},
	)
	
	result, err := optimizer.AnalyzeCredentialChain(context.Background(), chain)
	if err != nil {
		t.Fatalf("AnalyzeCredentialChain failed: %v", err)
	}
	
	if result.Component != "CredentialChain" {
		t.Errorf("Expected component 'CredentialChain', got %s", result.Component)
	}
	
	if result.Operation != "Retrieve" {
		t.Errorf("Expected operation 'Retrieve', got %s", result.Operation)
	}
	
	if result.Duration < 0 {
		t.Error("Expected non-negative duration")
	}
	
	// Check if bottleneck detection works
	if result.Duration > 100*time.Millisecond && !result.Bottleneck {
		t.Error("Expected bottleneck detection for slow credential retrieval")
	}
	
	t.Logf("Credential chain analysis: %+v", result)
}

func TestOptimizer_AnalyzeSigV4Signing(t *testing.T) {
	optimizer := NewOptimizer()
	
	// Create mock credentials
	mockCreds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "",
		Source:          "test",
	}
	
	signer := sigv4.NewSigner(mockCreds, "us-east-1", "codewhisperer")
	
	result, err := optimizer.AnalyzeSigV4Signing(context.Background(), signer)
	if err != nil {
		t.Fatalf("AnalyzeSigV4Signing failed: %v", err)
	}
	
	if result.Component != "SigV4Signer" {
		t.Errorf("Expected component 'SigV4Signer', got %s", result.Component)
	}
	
	if result.Operation != "SignRequest" {
		t.Errorf("Expected operation 'SignRequest', got %s", result.Operation)
	}
	
	if result.Duration < 0 {
		t.Error("Expected non-negative duration")
	}
	
	// Check if bottleneck detection works
	if result.Duration > 5*time.Millisecond && !result.Bottleneck {
		t.Error("Expected bottleneck detection for slow signing")
	}
	
	t.Logf("SigV4 signing analysis: %+v", result)
}

func TestOptimizer_AnalyzeCacheEffectiveness(t *testing.T) {
	optimizer := NewOptimizer()
	
	// Create a chain with cached credentials using mock provider
	mockProvider := &mockProvider{
		creds: &credentials.Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			Source:          "test",
		},
	}
	chain := credentials.NewChain(mockProvider)
	
	// Prime the cache
	_, _ = chain.Retrieve(context.Background())
	
	result, err := optimizer.AnalyzeCacheEffectiveness(context.Background(), chain, 10)
	if err != nil {
		t.Fatalf("AnalyzeCacheEffectiveness failed: %v", err)
	}
	
	if result.Component != "CredentialCache" {
		t.Errorf("Expected component 'CredentialCache', got %s", result.Component)
	}
	
	if result.Duration < 0 {
		t.Error("Expected non-negative duration")
	}
	
	t.Logf("Cache effectiveness analysis: %+v", result)
}

func TestOptimizer_IdentifyBottlenecks(t *testing.T) {
	optimizer := NewOptimizer()
	
	bottlenecks, err := optimizer.IdentifyBottlenecks(context.Background())
	if err != nil {
		t.Fatalf("IdentifyBottlenecks failed: %v", err)
	}
	
	// Should have analyzed multiple components
	results := optimizer.GetResults()
	if len(results) == 0 {
		t.Error("Expected at least one analysis result")
	}
	
	t.Logf("Found %d bottlenecks out of %d components", len(bottlenecks), len(results))
	
	for _, bottleneck := range bottlenecks {
		t.Logf("Bottleneck: %s.%s - %v", bottleneck.Component, bottleneck.Operation, bottleneck.Duration)
	}
}

func TestOptimizer_GenerateReport(t *testing.T) {
	optimizer := NewOptimizer()
	
	// Run some analysis first
	_, _ = optimizer.IdentifyBottlenecks(context.Background())
	
	report := optimizer.GenerateReport()
	if report == "" {
		t.Error("Expected non-empty report")
	}
	
	if len(report) < 100 {
		t.Error("Expected detailed report")
	}
	
	t.Logf("Generated report:\n%s", report)
}

func TestOptimizer_GetResults(t *testing.T) {
	optimizer := NewOptimizer()
	
	// Initially should be empty
	results := optimizer.GetResults()
	if len(results) != 0 {
		t.Error("Expected empty results initially")
	}
	
	// Run analysis
	_, _ = optimizer.IdentifyBottlenecks(context.Background())
	
	// Should have results now
	results = optimizer.GetResults()
	if len(results) == 0 {
		t.Error("Expected results after analysis")
	}
}

func TestCreateTestRequest(t *testing.T) {
	req, err := createTestRequest()
	if err != nil {
		t.Fatalf("createTestRequest failed: %v", err)
	}
	
	if req.Method != "POST" {
		t.Errorf("Expected POST method, got %s", req.Method)
	}
	
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header")
	}
	
	expectedURL := "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse"
	if req.URL.String() != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, req.URL.String())
	}
}

func BenchmarkCredentialChainRetrieve(b *testing.B) {
	chain := credentials.NewChain(
		&mockProvider{
			creds: &credentials.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Source:          "test",
			},
		},
	)
	
	ctx := context.Background()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := chain.Retrieve(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigV4Signing(b *testing.B) {
	mockCreds := &credentials.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Source:          "test",
	}
	
	signer := sigv4.NewSigner(mockCreds, "us-east-1", "codewhisperer")
	req, _ := http.NewRequest("POST", "https://example.com/api", nil)
	body := []byte(`{"test": "data"}`)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := signer.SignRequest(req, body)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// mockProvider implements the Provider interface for testing
type mockProvider struct {
	creds *credentials.Credentials
	err   error
}

func (m *mockProvider) Retrieve(ctx context.Context) (*credentials.Credentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.creds, nil
}

func (m *mockProvider) IsExpired() bool {
	return false
}
