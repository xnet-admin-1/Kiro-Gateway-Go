// Package hotpath provides hot path analysis for performance optimization.
package hotpath

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
)

// HotPathAnalyzer tracks function call frequencies and execution times.
type HotPathAnalyzer struct {
	calls map[string]*CallStats
	mu    sync.RWMutex
}

// CallStats tracks statistics for a function call.
type CallStats struct {
	Name         string
	CallCount    int64
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	AvgTime      time.Duration
	LastCalled   time.Time
}

// NewHotPathAnalyzer creates a new hot path analyzer.
func NewHotPathAnalyzer() *HotPathAnalyzer {
	return &HotPathAnalyzer{
		calls: make(map[string]*CallStats),
	}
}

// TrackCall tracks a function call execution.
func (h *HotPathAnalyzer) TrackCall(name string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	h.recordCall(name, duration)
	return err
}

// TrackCallWithContext tracks a function call with context.
func (h *HotPathAnalyzer) TrackCallWithContext(ctx context.Context, name string, fn func(context.Context) error) error {
	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)
	
	h.recordCall(name, duration)
	return err
}

// recordCall records call statistics.
func (h *HotPathAnalyzer) recordCall(name string, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	stats, exists := h.calls[name]
	if !exists {
		stats = &CallStats{
			Name:       name,
			MinTime:    duration,
			MaxTime:    duration,
		}
		h.calls[name] = stats
	}
	
	stats.CallCount++
	stats.TotalTime += duration
	stats.LastCalled = time.Now()
	
	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}
	
	stats.AvgTime = time.Duration(int64(stats.TotalTime) / stats.CallCount)
}

// GetHotPaths returns the most frequently called functions.
func (h *HotPathAnalyzer) GetHotPaths(limit int) []*CallStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	var stats []*CallStats
	for _, stat := range h.calls {
		stats = append(stats, stat)
	}
	
	// Sort by call count (descending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].CallCount > stats[j].CallCount
	})
	
	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}
	
	return stats
}

// GetSlowestPaths returns the slowest function calls.
func (h *HotPathAnalyzer) GetSlowestPaths(limit int) []*CallStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	var stats []*CallStats
	for _, stat := range h.calls {
		stats = append(stats, stat)
	}
	
	// Sort by average time (descending)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].AvgTime > stats[j].AvgTime
	})
	
	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}
	
	return stats
}

// GetStats returns statistics for a specific function.
func (h *HotPathAnalyzer) GetStats(name string) *CallStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if stats, exists := h.calls[name]; exists {
		// Return a copy to avoid race conditions
		return &CallStats{
			Name:       stats.Name,
			CallCount:  stats.CallCount,
			TotalTime:  stats.TotalTime,
			MinTime:    stats.MinTime,
			MaxTime:    stats.MaxTime,
			AvgTime:    stats.AvgTime,
			LastCalled: stats.LastCalled,
		}
	}
	return nil
}

// Reset clears all tracked statistics.
func (h *HotPathAnalyzer) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = make(map[string]*CallStats)
}

// GenerateReport generates a performance analysis report.
func (h *HotPathAnalyzer) GenerateReport() string {
	report := "Hot Path Analysis Report\n"
	report += "========================\n\n"
	
	// Hot paths (most called)
	hotPaths := h.GetHotPaths(10)
	report += "Top 10 Most Called Functions:\n"
	report += "-----------------------------\n"
	for i, stats := range hotPaths {
		report += fmt.Sprintf("%d. %s\n", i+1, stats.Name)
		report += fmt.Sprintf("   Calls: %d\n", stats.CallCount)
		report += fmt.Sprintf("   Avg Time: %v\n", stats.AvgTime)
		report += fmt.Sprintf("   Total Time: %v\n", stats.TotalTime)
		report += "\n"
	}
	
	// Slowest paths
	slowPaths := h.GetSlowestPaths(10)
	report += "Top 10 Slowest Functions:\n"
	report += "-------------------------\n"
	for i, stats := range slowPaths {
		report += fmt.Sprintf("%d. %s\n", i+1, stats.Name)
		report += fmt.Sprintf("   Avg Time: %v\n", stats.AvgTime)
		report += fmt.Sprintf("   Min Time: %v\n", stats.MinTime)
		report += fmt.Sprintf("   Max Time: %v\n", stats.MaxTime)
		report += fmt.Sprintf("   Calls: %d\n", stats.CallCount)
		report += "\n"
	}
	
	return report
}

// MemoryAnalyzer tracks memory usage patterns.
type MemoryAnalyzer struct {
	snapshots []MemorySnapshot
	mu        sync.RWMutex
}

// MemorySnapshot represents a memory usage snapshot.
type MemorySnapshot struct {
	Timestamp    time.Time
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	NumGC        uint32
	GCCPUFraction float64
}

// NewMemoryAnalyzer creates a new memory analyzer.
func NewMemoryAnalyzer() *MemoryAnalyzer {
	return &MemoryAnalyzer{
		snapshots: make([]MemorySnapshot, 0),
	}
}

// TakeSnapshot takes a memory usage snapshot.
func (m *MemoryAnalyzer) TakeSnapshot() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	
	snapshot := MemorySnapshot{
		Timestamp:     time.Now(),
		Alloc:         stats.Alloc,
		TotalAlloc:    stats.TotalAlloc,
		Sys:           stats.Sys,
		NumGC:         stats.NumGC,
		GCCPUFraction: stats.GCCPUFraction,
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots = append(m.snapshots, snapshot)
}

// GetSnapshots returns all memory snapshots.
func (m *MemoryAnalyzer) GetSnapshots() []MemorySnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	snapshots := make([]MemorySnapshot, len(m.snapshots))
	copy(snapshots, m.snapshots)
	return snapshots
}

// AnalyzeMemoryTrend analyzes memory usage trends.
func (m *MemoryAnalyzer) AnalyzeMemoryTrend() string {
	snapshots := m.GetSnapshots()
	if len(snapshots) < 2 {
		return "Insufficient data for trend analysis"
	}
	
	first := snapshots[0]
	last := snapshots[len(snapshots)-1]
	
	allocDelta := int64(last.Alloc) - int64(first.Alloc)
	sysDelta := int64(last.Sys) - int64(first.Sys)
	gcDelta := last.NumGC - first.NumGC
	
	report := "Memory Trend Analysis\n"
	report += "====================\n\n"
	report += fmt.Sprintf("Time Period: %v to %v\n", first.Timestamp.Format("15:04:05"), last.Timestamp.Format("15:04:05"))
	report += fmt.Sprintf("Alloc Delta: %+d bytes\n", allocDelta)
	report += fmt.Sprintf("Sys Delta: %+d bytes\n", sysDelta)
	report += fmt.Sprintf("GC Cycles: %d\n", gcDelta)
	report += fmt.Sprintf("Current GC CPU Fraction: %.2f%%\n", last.GCCPUFraction*100)
	
	if allocDelta > 1024*1024 { // 1MB
		report += "\n[WARNING] Significant memory growth detected (>1MB)\n"
	}
	
	if last.GCCPUFraction > 0.1 { // 10%
		report += "\n[WARNING] High GC CPU usage (>10%)\n"
	}
	
	return report
}
