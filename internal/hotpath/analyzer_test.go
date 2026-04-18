package hotpath

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHotPathAnalyzer_TrackCall(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	// Track a simple function call
	err := analyzer.TrackCall("test_function", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	
	if err != nil {
		t.Fatalf("TrackCall failed: %v", err)
	}
	
	stats := analyzer.GetStats("test_function")
	if stats == nil {
		t.Fatal("Expected stats for test_function")
	}
	
	if stats.CallCount != 1 {
		t.Errorf("Expected call count 1, got %d", stats.CallCount)
	}
	
	// Allow for timing overhead - sleep is not perfectly accurate
	minExpected := 8 * time.Millisecond // Allow 2ms under
	if stats.AvgTime < minExpected {
		t.Errorf("Expected avg time >= %v, got %v", minExpected, stats.AvgTime)
	}
}

func TestHotPathAnalyzer_TrackCallWithContext(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	ctx := context.Background()
	
	err := analyzer.TrackCallWithContext(ctx, "context_function", func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	
	if err != nil {
		t.Fatalf("TrackCallWithContext failed: %v", err)
	}
	
	stats := analyzer.GetStats("context_function")
	if stats == nil {
		t.Fatal("Expected stats for context_function")
	}
	
	if stats.CallCount != 1 {
		t.Errorf("Expected call count 1, got %d", stats.CallCount)
	}
}

func TestHotPathAnalyzer_MultipleCallsStats(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	// Make multiple calls with different durations
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		5 * time.Millisecond,
	}
	
	for _, duration := range durations {
		err := analyzer.TrackCall("multi_call", func() error {
			time.Sleep(duration)
			return nil
		})
		if err != nil {
			t.Fatalf("TrackCall failed: %v", err)
		}
	}
	
	stats := analyzer.GetStats("multi_call")
	if stats == nil {
		t.Fatal("Expected stats for multi_call")
	}
	
	if stats.CallCount != 3 {
		t.Errorf("Expected call count 3, got %d", stats.CallCount)
	}
	
	// Verify min time is reasonable (should be close to the shortest sleep)
	// Allow generous tolerance since sleep timing can vary significantly
	if stats.MinTime < 3*time.Millisecond {
		t.Errorf("Min time %v is unexpectedly low (< 3ms)", stats.MinTime)
	}
	if stats.MinTime > 15*time.Millisecond {
		t.Errorf("Min time %v is unexpectedly high (> 15ms)", stats.MinTime)
	}
	
	// Verify max time is reasonable (should be close to the longest sleep)
	if stats.MaxTime < 18*time.Millisecond {
		t.Errorf("Max time %v is unexpectedly low (< 18ms)", stats.MaxTime)
	}
	if stats.MaxTime > 30*time.Millisecond {
		t.Errorf("Max time %v is unexpectedly high (> 30ms)", stats.MaxTime)
	}
	
	// Verify min < max (basic sanity check)
	if stats.MinTime >= stats.MaxTime {
		t.Errorf("Min time %v should be less than max time %v", stats.MinTime, stats.MaxTime)
	}
	
	// Verify average is between min and max
	if stats.AvgTime < stats.MinTime || stats.AvgTime > stats.MaxTime {
		t.Errorf("Average time %v should be between min %v and max %v", 
			stats.AvgTime, stats.MinTime, stats.MaxTime)
	}
}

func TestHotPathAnalyzer_GetHotPaths(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	// Create functions with different call counts
	functions := map[string]int{
		"hot_function":    10,
		"warm_function":   5,
		"cold_function":   1,
	}
	
	for name, count := range functions {
		for i := 0; i < count; i++ {
			analyzer.TrackCall(name, func() error {
				return nil
			})
		}
	}
	
	hotPaths := analyzer.GetHotPaths(3)
	if len(hotPaths) != 3 {
		t.Errorf("Expected 3 hot paths, got %d", len(hotPaths))
	}
	
	// Should be sorted by call count (descending)
	if hotPaths[0].Name != "hot_function" || hotPaths[0].CallCount != 10 {
		t.Errorf("Expected hot_function with 10 calls first, got %s with %d calls", 
			hotPaths[0].Name, hotPaths[0].CallCount)
	}
	
	if hotPaths[1].Name != "warm_function" || hotPaths[1].CallCount != 5 {
		t.Errorf("Expected warm_function with 5 calls second, got %s with %d calls", 
			hotPaths[1].Name, hotPaths[1].CallCount)
	}
}

func TestHotPathAnalyzer_GetSlowestPaths(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	// Create functions with different execution times
	analyzer.TrackCall("slow_function", func() error {
		time.Sleep(30 * time.Millisecond)
		return nil
	})
	
	analyzer.TrackCall("medium_function", func() error {
		time.Sleep(15 * time.Millisecond)
		return nil
	})
	
	analyzer.TrackCall("fast_function", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	
	slowPaths := analyzer.GetSlowestPaths(3)
	if len(slowPaths) != 3 {
		t.Errorf("Expected 3 slow paths, got %d", len(slowPaths))
	}
	
	// Should be sorted by avg time (descending)
	// Verify the order is correct by checking relative timing
	if slowPaths[0].Name != "slow_function" {
		t.Errorf("Expected slow_function first, got %s (avg: %v)", slowPaths[0].Name, slowPaths[0].AvgTime)
	}
	
	if slowPaths[1].Name != "medium_function" {
		t.Errorf("Expected medium_function second, got %s (avg: %v)", slowPaths[1].Name, slowPaths[1].AvgTime)
	}
	
	if slowPaths[2].Name != "fast_function" {
		t.Errorf("Expected fast_function third, got %s (avg: %v)", slowPaths[2].Name, slowPaths[2].AvgTime)
	}
	
	// Verify the ordering is correct (slow > medium > fast)
	if slowPaths[0].AvgTime <= slowPaths[1].AvgTime {
		t.Errorf("Expected slow_function (%v) to be slower than medium_function (%v)", 
			slowPaths[0].AvgTime, slowPaths[1].AvgTime)
	}
	
	if slowPaths[1].AvgTime <= slowPaths[2].AvgTime {
		t.Errorf("Expected medium_function (%v) to be slower than fast_function (%v)", 
			slowPaths[1].AvgTime, slowPaths[2].AvgTime)
	}
}

func TestHotPathAnalyzer_ErrorHandling(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	testError := errors.New("test error")
	err := analyzer.TrackCall("error_function", func() error {
		return testError
	})
	
	if err != testError {
		t.Errorf("Expected test error, got %v", err)
	}
	
	// Should still track the call even if it errors
	stats := analyzer.GetStats("error_function")
	if stats == nil {
		t.Fatal("Expected stats for error_function")
	}
	
	if stats.CallCount != 1 {
		t.Errorf("Expected call count 1, got %d", stats.CallCount)
	}
}

func TestHotPathAnalyzer_Reset(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	analyzer.TrackCall("test_function", func() error {
		return nil
	})
	
	stats := analyzer.GetStats("test_function")
	if stats == nil {
		t.Fatal("Expected stats before reset")
	}
	
	analyzer.Reset()
	
	stats = analyzer.GetStats("test_function")
	if stats != nil {
		t.Error("Expected no stats after reset")
	}
}

func TestHotPathAnalyzer_GenerateReport(t *testing.T) {
	analyzer := NewHotPathAnalyzer()
	
	// Add some test data
	analyzer.TrackCall("function1", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	
	analyzer.TrackCall("function2", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	
	report := analyzer.GenerateReport()
	if report == "" {
		t.Error("Expected non-empty report")
	}
	
	if len(report) < 100 {
		t.Error("Expected detailed report")
	}
	
	t.Logf("Generated report:\n%s", report)
}

func TestMemoryAnalyzer_TakeSnapshot(t *testing.T) {
	analyzer := NewMemoryAnalyzer()
	
	analyzer.TakeSnapshot()
	
	snapshots := analyzer.GetSnapshots()
	if len(snapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snapshots))
	}
	
	snapshot := snapshots[0]
	if snapshot.Alloc == 0 {
		t.Error("Expected non-zero allocation")
	}
	
	if snapshot.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestMemoryAnalyzer_AnalyzeMemoryTrend(t *testing.T) {
	analyzer := NewMemoryAnalyzer()
	
	// Test with insufficient data
	trend := analyzer.AnalyzeMemoryTrend()
	if trend != "Insufficient data for trend analysis" {
		t.Errorf("Expected insufficient data message, got: %s", trend)
	}
	
	// Take multiple snapshots
	analyzer.TakeSnapshot()
	time.Sleep(10 * time.Millisecond)
	
	// Allocate some memory
	data := make([]byte, 1024*1024) // 1MB
	_ = data
	
	analyzer.TakeSnapshot()
	
	trend = analyzer.AnalyzeMemoryTrend()
	if trend == "" {
		t.Error("Expected non-empty trend analysis")
	}
	
	if len(trend) < 100 {
		t.Error("Expected detailed trend analysis")
	}
	
	t.Logf("Memory trend analysis:\n%s", trend)
}

func BenchmarkHotPathAnalyzer_TrackCall(b *testing.B) {
	analyzer := NewHotPathAnalyzer()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.TrackCall("benchmark_function", func() error {
			return nil
		})
	}
}

func BenchmarkMemoryAnalyzer_TakeSnapshot(b *testing.B) {
	analyzer := NewMemoryAnalyzer()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.TakeSnapshot()
	}
}
